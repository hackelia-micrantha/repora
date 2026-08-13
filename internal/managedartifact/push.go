package managedartifact

import (
	"fmt"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
)

const managedREADMECanonicalRemote = "canonical"

type guardedPushGit interface {
	preparedCommitReadGit
	ResolveRevision(repoPath, rev string) (string, error)
	ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error
}

type guardedPushMirrorPath func(string) (string, error)

// PushResult records one remote mutation attempt. A false Pushed value may be
// returned with an error when a lease or transport operation rejects the push.
type PushResult struct {
	UID       string
	ID        string
	Branch    string
	BaseOID   string
	CommitOID string
	Pushed    bool
}

// Pusher performs the only managed-README remote mutation: an exact-base leased
// canonical branch push of a previously prepared and re-verified commit.
type Pusher struct {
	git        guardedPushGit
	mirrorPath guardedPushMirrorPath
}

func NewPusher() *Pusher {
	return newPusher(gitwrap.Client{}, gitwrap.MirrorPath)
}

func newPusher(git guardedPushGit, mirrorPath guardedPushMirrorPath) *Pusher {
	return &Pusher{git: git, mirrorPath: mirrorPath}
}

// Push validates all prepared candidates locally, performs one fresh exact-plan
// preflight across all repositories, then pushes sequentially using exact
// reviewed-base leases. Multi-repository remote mutation is not atomic; results
// identify successful pushes before a later failure.
func (p *Pusher) Push(spec config.Spec, plan Plan, prepared []PreparedCommit, observer READMEObserver) ([]PushResult, error) {
	if p == nil || p.git == nil || p.mirrorPath == nil {
		return nil, fmt.Errorf("managed README pusher is not fully configured")
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("validate managed artifact plan: %w", err)
	}
	if len(prepared) != len(plan.Repositories) {
		return nil, fmt.Errorf("prepared commit count %d does not match planned repository count %d", len(prepared), len(plan.Repositories))
	}

	cachePaths := make([]string, len(plan.Repositories))
	for i, planned := range plan.Repositories {
		candidate := prepared[i]
		if candidate.UID != planned.UID || candidate.ID != planned.ID || candidate.BaseOID != planned.BaseOID {
			return nil, fmt.Errorf("prepared commit %d does not match planned repository %q", i, planned.ID)
		}
		if !planOIDPattern.MatchString(candidate.TreeOID) || !planOIDPattern.MatchString(candidate.CommitOID) {
			return nil, fmt.Errorf("prepared commit %d for repo %q has invalid tree or commit object ID", i, planned.ID)
		}
		cachePath, err := p.mirrorPath(planned.UID)
		if err != nil {
			return nil, fmt.Errorf("repo %q: resolve push cache path: %w", planned.ID, err)
		}
		parent, err := p.git.ResolveRevision(cachePath, candidate.CommitOID+"^")
		if err != nil {
			return nil, fmt.Errorf("repo %q: verify candidate parent: %w", planned.ID, err)
		}
		if parent != planned.BaseOID {
			return nil, fmt.Errorf("repo %q: prepared commit parent %s does not match reviewed base %s", planned.ID, parent, planned.BaseOID)
		}
		tree, err := p.git.ResolveRevision(cachePath, candidate.CommitOID+"^{tree}")
		if err != nil {
			return nil, fmt.Errorf("repo %q: verify candidate tree: %w", planned.ID, err)
		}
		if tree != candidate.TreeOID {
			return nil, fmt.Errorf("repo %q: prepared tree %s does not match commit tree %s", planned.ID, candidate.TreeOID, tree)
		}
		if err := verifyPreparedCommit(p.git, cachePath, planned, candidate.CommitOID, []byte(*planned.Actions[0].Desired.Content)); err != nil {
			return nil, err
		}
		cachePaths[i] = cachePath
	}

	// Re-observe every canonical repository immediately before the first push.
	// The per-branch lease below closes the remaining race between this preflight
	// and each individual network mutation.
	if err := PreflightPlan(spec, plan, observer); err != nil {
		return nil, err
	}

	results := make([]PushResult, 0, len(plan.Repositories))
	for i, planned := range plan.Repositories {
		candidate := prepared[i]
		result := PushResult{
			UID:       planned.UID,
			ID:        planned.ID,
			Branch:    planned.Target.Branch,
			BaseOID:   planned.BaseOID,
			CommitOID: candidate.CommitOID,
		}
		if err := p.git.ForcePushBranchWithLease(cachePaths[i], managedREADMECanonicalRemote, candidate.CommitOID, planned.Target.Branch, planned.BaseOID); err != nil {
			results = append(results, result)
			return results, fmt.Errorf("repo %q: push managed README commit with exact base lease: %w", planned.ID, err)
		}
		result.Pushed = true
		results = append(results, result)
	}
	return results, nil
}
