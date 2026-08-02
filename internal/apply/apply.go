package apply

import (
	"fmt"

	"repoctl/internal/config"
	"repoctl/internal/executor"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/status"
)

const (
	OutputKind    = "repora.apply"
	OutputVersion = 1
)

type Output struct {
	Kind    string   `json:"kind"`
	Version int      `json:"version"`
	Results []Result `json:"results"`
}

type Result struct {
	ID      string       `json:"id"`
	UID     string       `json:"uid"`
	State   status.State `json:"state"`
	Applied bool         `json:"applied"`
	DryRun  bool         `json:"dry_run"`
	Actions []Action     `json:"actions"`
	Error   string       `json:"error,omitempty"`
}

type Action struct {
	Type              string `json:"type"`
	Source            string `json:"source"`
	Target            string `json:"target"`
	Force             bool   `json:"force"`
	ExpectedOldTarget string `json:"expected_old_target,omitempty"`
}

type Git interface {
	executor.Git
	ResolveRemoteHeadBranch(repoPath, remote string) (string, error)
}

// IsUnsafe is retained for callers that need to identify states requiring a
// mirror-head observation. The planner owns that classification.
func IsUnsafe(result status.Result) bool {
	return plan.RequiresMirrorHeadObservation(result)
}

// BuildArtifact is the single observation-to-plan boundary used by plan,
// dry-run, and convenience apply. Planning describes required destructive
// intent; execution separately authorizes it. The allowForce argument is
// retained at this internal call boundary for compatibility but does not alter
// the generated artifact.
func BuildArtifact(repo config.Repo, st status.Result, git Git, _ bool) (planartifact.Artifact, error) {
	planned, err := buildPlan(repo, st, git)
	if err != nil {
		return planartifact.Artifact{}, err
	}
	artifact := planartifact.FromPlans(planned)
	if err := artifact.Validate(); err != nil {
		return planartifact.Artifact{}, fmt.Errorf("validate plan artifact for repo %q: %w", repo.ID, err)
	}
	return artifact, nil
}

// Execute is the convenience observe-plan-execute path. It delegates to the
// same artifact boundary exposed to operators rather than executing an
// in-memory plan through a separate path.
func Execute(repo config.Repo, st status.Result, git Git, force bool, dryRun bool) (Result, error) {
	artifact, err := BuildArtifact(repo, st, git, force)
	if err != nil {
		return newResult(repo, st, dryRun), err
	}
	return ExecuteArtifact(repo, st, artifact, git, force, dryRun)
}

// ExecuteArtifact consumes an already-built artifact without recomputing
// reconciliation intent. It validates repository topology, projects the exact
// actions for output, then checks explicit force authorization before stale-ref
// preflight and mutation.
func ExecuteArtifact(repo config.Repo, st status.Result, artifact planartifact.Artifact, git Git, allowForce bool, dryRun bool) (Result, error) {
	result := newResult(repo, st, dryRun)
	planned, err := planForRepository(repo, artifact)
	if err != nil {
		return result, err
	}
	result.Actions = compatibilityActions(planned)
	if planRequiresForce(planned) && !allowForce {
		return result, fmt.Errorf("repo %q plan contains a forced action; rerun with --force", repo.ID)
	}
	if dryRun || len(planned.Actions) == 0 {
		return result, nil
	}

	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return result, err
	}
	executed, err := executor.Execute(path, artifact, git)
	if err != nil {
		return result, fmt.Errorf("execute plan artifact for repo %q: %w", repo.ID, err)
	}
	result.Applied = executed.AllApplied()
	return result, nil
}

func ArtifactRequiresForce(artifact planartifact.Artifact) (bool, error) {
	plans, err := artifact.Plans()
	if err != nil {
		return false, err
	}
	for _, planned := range plans {
		if planRequiresForce(planned) {
			return true, nil
		}
	}
	return false, nil
}

func newResult(repo config.Repo, st status.Result, dryRun bool) Result {
	return Result{
		ID:      repo.ID,
		UID:     repo.DurableID(),
		State:   st.State,
		DryRun:  dryRun,
		Actions: []Action{},
	}
}

func buildPlan(repo config.Repo, st status.Result, git Git) (plan.ReconciliationPlan, error) {
	path, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return plan.ReconciliationPlan{}, err
	}

	srcBranch, err := git.ResolveRemoteHeadBranch(path, "canonical")
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err)
	}
	dstBranch, err := git.ResolveRemoteHeadBranch(path, "mirror")
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("resolve mirror HEAD for repo %q: %w", repo.ID, err)
	}
	if dstBranch == "" {
		dstBranch = srcBranch
	}

	observation := plan.Observation{CanonicalBranch: srcBranch, MirrorBranch: dstBranch}
	if plan.RequiresRefObservation(st) {
		srcRef := "refs/remotes/canonical/" + srcBranch
		sourceOID, err := git.ResolveRevision(path, srcRef)
		if err != nil {
			return plan.ReconciliationPlan{}, fmt.Errorf("resolve canonical branch for repo %q: %w", repo.ID, err)
		}
		dstRef := "refs/remotes/mirror/" + dstBranch
		targetOID, err := git.ResolveRevision(path, dstRef)
		if err != nil {
			return plan.ReconciliationPlan{}, fmt.Errorf("resolve mirror branch for repo %q: %w", repo.ID, err)
		}
		observation.CanonicalHeadOID = sourceOID
		observation.MirrorHeadOID = targetOID
	}

	// Planning must describe destructive intent even before execution is
	// authorized. ExecuteArtifact owns the explicit --force gate.
	return plan.Reconcile(repo, st, observation, true)
}

func planForRepository(repo config.Repo, artifact planartifact.Artifact) (plan.ReconciliationPlan, error) {
	plans, err := artifact.Plans()
	if err != nil {
		return plan.ReconciliationPlan{}, fmt.Errorf("validate plan artifact: %w", err)
	}
	if len(plans) != 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("plan artifact requires exactly one repository, got %d", len(plans))
	}
	planned := plans[0]
	if planned.UID != repo.DurableID() {
		return plan.ReconciliationPlan{}, fmt.Errorf("plan repository uid %q does not match configured uid %q", planned.UID, repo.DurableID())
	}
	if len(repo.Mirrors) != 1 {
		return plan.ReconciliationPlan{}, fmt.Errorf("repo %q requires exactly one configured mirror, got %d", repo.ID, len(repo.Mirrors))
	}
	for i, action := range planned.Actions {
		if action.Source.Provider != repo.Canonical.Provider || action.Source.Name != "canonical" {
			return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d source does not match configured canonical repository", i)
		}
		if action.Target.Provider != repo.Mirrors[0].Provider || action.Target.Name != "mirror" {
			return plan.ReconciliationPlan{}, fmt.Errorf("plan action %d target does not match configured mirror repository", i)
		}
	}
	return planned, nil
}

func planRequiresForce(planned plan.ReconciliationPlan) bool {
	for _, action := range planned.Actions {
		if action.Force {
			return true
		}
	}
	return false
}

func compatibilityActions(planned plan.ReconciliationPlan) []Action {
	actions := make([]Action, 0, len(planned.Actions))
	for _, plannedAction := range planned.Actions {
		action := Action{
			Type:   string(plannedAction.Type),
			Source: plannedAction.Source.Name + "/" + plannedAction.Source.Branch,
			Target: plannedAction.Target.Provider + "/" + plannedAction.Target.Branch,
			Force:  plannedAction.Force,
		}
		if plannedAction.Force {
			action.ExpectedOldTarget = plannedAction.ExpectedOldTarget
		}
		actions = append(actions, action)
	}
	return actions
}
