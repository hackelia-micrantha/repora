package apply

import (
	"fmt"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/plan"
	"repoctl/internal/planartifact"
	"repoctl/internal/refpolicy"
	"repoctl/internal/status"
)

// BuildRepositoryArtifact creates one exact path-bound artifact for every
// required mirror action in configuration order. Observations are matched by
// stable provider/path target identity, never by array position.
func BuildRepositoryArtifact(repo config.Repo, observed status.RepositoryResult, git Git) (planartifact.Artifact, error) {
	planned, err := buildRepositoryPlan(repo, observed, git)
	if err != nil {
		return planartifact.Artifact{}, err
	}
	artifact, err := planartifact.FromCurrentPlans(planned)
	if err != nil {
		return planartifact.Artifact{}, fmt.Errorf("validate multi-mirror plan artifact for repo %q: %w", repo.ID, err)
	}
	return artifact, nil
}

func buildRepositoryPlan(repo config.Repo, observed status.RepositoryResult, git Git) (plan.ReconciliationPlan, error) {
	planned := plan.ReconciliationPlan{ID: repo.ID, UID: repo.DurableID(), Actions: []plan.PlannedAction{}}
	if len(repo.Mirrors) == 0 {
		return planned, fmt.Errorf("repo %q requires at least one mirror", repo.ID)
	}
	if observed.ID != "" && observed.ID != repo.ID {
		return planned, fmt.Errorf("status repository id %q does not match configured id %q", observed.ID, repo.ID)
	}
	if observed.UID != "" && observed.UID != repo.DurableID() {
		return planned, fmt.Errorf("status repository uid %q does not match configured uid %q", observed.UID, repo.DurableID())
	}
	if strings.TrimSpace(observed.Error) != "" {
		return planned, fmt.Errorf("repo %q status is incomplete: %s", repo.ID, observed.Error)
	}
	if len(observed.Mirrors) != len(repo.Mirrors) {
		return planned, fmt.Errorf("repo %q status contains %d mirrors, want %d", repo.ID, len(observed.Mirrors), len(repo.Mirrors))
	}

	policy, err := repo.EffectiveRefPolicy()
	if err != nil {
		return planned, fmt.Errorf("invalid ref policy for repo %q: %w", repo.ID, err)
	}
	canonicalPath, err := repo.Canonical.RepositoryPath()
	if err != nil {
		return planned, fmt.Errorf("resolve canonical identity for repo %q: %w", repo.ID, err)
	}

	byTarget := make(map[string]status.MirrorResult, len(observed.Mirrors))
	for i, mirror := range observed.Mirrors {
		target := strings.TrimSpace(mirror.Target)
		if target == "" {
			return planned, fmt.Errorf("repo %q status mirror %d has no stable target identity", repo.ID, i)
		}
		if _, exists := byTarget[target]; exists {
			return planned, fmt.Errorf("repo %q status duplicates mirror target %q", repo.ID, target)
		}
		byTarget[target] = mirror
	}

	repoPath, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return planned, err
	}
	canonicalBranch, err := git.ResolveRemoteHeadBranch(repoPath, "canonical")
	if err != nil {
		return planned, fmt.Errorf("resolve canonical HEAD for repo %q: %w", repo.ID, err)
	}
	canonicalBranch = strings.TrimSpace(canonicalBranch)
	if canonicalBranch == "" {
		return planned, fmt.Errorf("repo %q canonical default branch is empty", repo.ID)
	}

	var sourceOID string
	for i, endpoint := range repo.Mirrors {
		targetID, err := endpoint.TargetID()
		if err != nil {
			return planned, fmt.Errorf("resolve mirror %d identity for repo %q: %w", i, repo.ID, err)
		}
		mirror, ok := byTarget[targetID]
		if !ok {
			return planned, fmt.Errorf("repo %q status is missing configured mirror %q", repo.ID, targetID)
		}
		if mirror.State == status.StateError || strings.TrimSpace(mirror.Error) != "" {
			return planned, fmt.Errorf("repo %q mirror %q status is incomplete: %s", repo.ID, targetID, mirror.Error)
		}

		decision, err := policy.Decide(refpolicy.Relationship(mirror.State))
		if err != nil {
			return planned, fmt.Errorf("repo %q mirror %q has unsupported state %q: %w", repo.ID, targetID, mirror.State, err)
		}
		if !decision.Action {
			continue
		}

		remoteName := mirrorRemoteName(len(repo.Mirrors), i)
		targetBranch, err := git.ResolveRemoteHeadBranch(repoPath, remoteName)
		if err != nil {
			return planned, fmt.Errorf("resolve mirror %q HEAD for repo %q: %w", targetID, repo.ID, err)
		}
		targetBranch = strings.TrimSpace(targetBranch)
		if targetBranch == "" {
			targetBranch = canonicalBranch
		}
		if err := policy.ValidateDefaultBranches(canonicalBranch, targetBranch, canonicalBranch, targetBranch); err != nil {
			return planned, fmt.Errorf("repo %q mirror %q violates ref policy: %w", repo.ID, targetID, err)
		}

		if sourceOID == "" {
			sourceOID, err = git.ResolveRevision(repoPath, "refs/remotes/canonical/"+canonicalBranch)
			if err != nil {
				return planned, fmt.Errorf("resolve canonical branch for repo %q: %w", repo.ID, err)
			}
		}
		targetOID, err := git.ResolveRevision(repoPath, "refs/remotes/"+remoteName+"/"+targetBranch)
		if err != nil {
			return planned, fmt.Errorf("resolve mirror %q branch for repo %q: %w", targetID, repo.ID, err)
		}
		targetPath, err := endpoint.RepositoryPath()
		if err != nil {
			return planned, fmt.Errorf("resolve mirror %q path for repo %q: %w", targetID, repo.ID, err)
		}

		planned.Actions = append(planned.Actions, plan.PlannedAction{
			Type: plan.ActionPushBranch,
			Source: plan.Remote{
				Provider: repo.Canonical.Provider,
				Path:     canonicalPath,
				Name:     "canonical",
				Branch:   canonicalBranch,
			},
			Target: plan.Remote{
				Provider: endpoint.Provider,
				Path:     targetPath,
				Name:     remoteName,
				Branch:   targetBranch,
			},
			ExpectedSource:    strings.TrimSpace(sourceOID),
			ExpectedOldTarget: strings.TrimSpace(targetOID),
			Force:             decision.Force,
			Reason:            decision.Reason,
		})
	}
	return planned, nil
}

func mirrorRemoteName(count, index int) string {
	if count == 1 {
		return "mirror"
	}
	return fmt.Sprintf("mirror-%d", index)
}
