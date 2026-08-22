package posture

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"repoctl/internal/config"
	gitwrap "repoctl/internal/git"
	"repoctl/internal/status"
)

type MirrorBranchObservation struct {
	Name      string
	Available bool
	Evidence  Evidence
}
type MirrorLocalObservation struct {
	Status          status.RepositoryResult
	CanonicalBranch MirrorBranchObservation
	MirrorBranches  []MirrorBranchObservation
}
type MirrorLocalObserver interface {
	Observe(config.Repo) (MirrorLocalObservation, error)
}
type GitMirrorLocalObserver struct{}

func (GitMirrorLocalObserver) Observe(repo config.Repo) (MirrorLocalObservation, error) {
	client := gitwrap.Client{}
	result, err := status.CheckAll(repo, client)
	if err != nil && result.Error != "" {
		return MirrorLocalObservation{}, err
	}
	cachePath, err := gitwrap.MirrorPath(repo.DurableID())
	if err != nil {
		return MirrorLocalObservation{}, err
	}
	out := MirrorLocalObservation{Status: result, MirrorBranches: make([]MirrorBranchObservation, len(repo.Mirrors))}
	out.CanonicalBranch = observeRemoteHead(client, cachePath, repo.DurableID(), "canonical")
	for i := range repo.Mirrors {
		if i < len(result.Mirrors) && result.Mirrors[i].State == status.StateError {
			out.MirrorBranches[i] = MirrorBranchObservation{Evidence: Evidence{Source: "git.remote_head", Reference: repo.DurableID() + ":mirror-" + strconv.Itoa(i) + "/HEAD", Detail: "mirror reconciliation failed; cached remote HEAD is not current evidence"}}
			continue
		}
		out.MirrorBranches[i] = observeRemoteHead(client, cachePath, repo.DurableID(), "mirror-"+strconv.Itoa(i))
	}
	return out, nil
}

func observeRemoteHead(client gitwrap.Client, cachePath, uid, remote string) MirrorBranchObservation {
	branch, err := client.ResolveRemoteHeadBranch(cachePath, remote)
	evidence := Evidence{Source: "git.remote_head", Reference: uid + ":" + remote + "/HEAD"}
	if err != nil || strings.TrimSpace(branch) == "" {
		evidence.Detail = "remote HEAD unavailable"
		return MirrorBranchObservation{Evidence: evidence}
	}
	return MirrorBranchObservation{Name: strings.TrimSpace(branch), Available: true, Evidence: evidence}
}

type MirrorProviderRepository struct {
	DefaultBranch  string
	Visibility     string
	PushPermission *bool
}
type MirrorProviderReader interface {
	Repository(context.Context, config.Endpoint) (MirrorProviderRepository, ReadObservation, error)
}

// DefaultMirrorProviderReader reuses the read-only GitHub repository reader.
// The current shared reader exposes default-branch identity only; visibility and
// current-actor push permission therefore remain unknown for GitHub and
// unavailable for providers without a posture adapter.
type DefaultMirrorProviderReader struct{ GitHub GitHubReader }

func (r DefaultMirrorProviderReader) Repository(ctx context.Context, endpoint config.Endpoint) (MirrorProviderRepository, ReadObservation, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorProviderRepository{}, ReadObservation{}, err
	}
	if identity.Provider != "github" {
		return MirrorProviderRepository{}, ReadObservation{Available: false, Evidence: Evidence{Source: "provider.unsupported", Reference: identity.Provider + ":" + identity.Path, Detail: "provider metadata adapter is not implemented for mirror posture v1"}}, nil
	}
	if r.GitHub == nil {
		return MirrorProviderRepository{}, ReadObservation{}, fmt.Errorf("GitHub mirror provider reader is required")
	}
	repository, obs, err := r.GitHub.Repository(ctx, identity.Path)
	if err != nil || !obs.Available {
		return MirrorProviderRepository{}, obs, err
	}
	return MirrorProviderRepository{DefaultBranch: repository.DefaultBranch}, obs, nil
}
