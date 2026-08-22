package posture

import (
	"context"
	"fmt"
	"net/url"
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
	observation := MirrorLocalObservation{
		Status:         result,
		MirrorBranches: make([]MirrorBranchObservation, len(repo.Mirrors)),
	}
	observation.CanonicalBranch = observeRemoteHead(client, cachePath, repo.DurableID(), "canonical")
	for i := range repo.Mirrors {
		if i < len(result.Mirrors) && result.Mirrors[i].State == status.StateError {
			observation.MirrorBranches[i] = MirrorBranchObservation{Evidence: Evidence{
				Source:    "git.remote_head",
				Reference: repo.DurableID() + ":mirror-" + strconv.Itoa(i) + "/HEAD",
				Detail:    "mirror reconciliation failed; cached remote HEAD is not current evidence",
			}}
			continue
		}
		observation.MirrorBranches[i] = observeRemoteHead(client, cachePath, repo.DurableID(), "mirror-"+strconv.Itoa(i))
	}
	return observation, nil
}

func observeRemoteHead(client gitwrap.Client, cachePath, uid, remote string) MirrorBranchObservation {
	branch, err := client.ResolveRemoteHeadBranch(cachePath, remote)
	evidence := Evidence{
		Source:    "git.remote_head",
		Reference: uid + ":" + remote + "/HEAD",
	}
	if err != nil || strings.TrimSpace(branch) == "" {
		evidence.Detail = "remote HEAD unavailable"
		return MirrorBranchObservation{Evidence: evidence}
	}
	return MirrorBranchObservation{
		Name:      strings.TrimSpace(branch),
		Available: true,
		Evidence:  evidence,
	}
}

type MirrorProviderRepository struct {
	DefaultBranch  string
	Visibility     string
	PushPermission *bool
}

type MirrorProviderReader interface {
	Repository(context.Context, config.Endpoint) (MirrorProviderRepository, ReadObservation, error)
}

// DefaultMirrorProviderReader reuses the structurally read-only GitHub reader.
// GitLab provider metadata remains unavailable until a posture adapter exists.
type DefaultMirrorProviderReader struct {
	GitHub GitHubReader
}

func (r DefaultMirrorProviderReader) Repository(ctx context.Context, endpoint config.Endpoint) (MirrorProviderRepository, ReadObservation, error) {
	identity, err := mirrorEndpointIdentity(endpoint)
	if err != nil {
		return MirrorProviderRepository{}, ReadObservation{}, err
	}
	if identity.Provider != "github" {
		return MirrorProviderRepository{}, ReadObservation{
			Available: false,
			Evidence: Evidence{
				Source:    "provider.unsupported",
				Reference: identity.Provider + ":" + identity.Path,
				Detail:    "provider metadata adapter is not implemented for mirror posture v1",
			},
		}, nil
	}
	if r.GitHub == nil {
		return MirrorProviderRepository{}, ReadObservation{}, fmt.Errorf("GitHub mirror provider reader is required")
	}
	repository, observation, err := r.GitHub.Repository(ctx, identity.Path)
	if err != nil || !observation.Available {
		return MirrorProviderRepository{}, observation, err
	}
	return MirrorProviderRepository{DefaultBranch: repository.DefaultBranch}, observation, nil
}

func mirrorEndpointIdentity(endpoint config.Endpoint) (MirrorEndpointIdentity, error) {
	provider := strings.TrimSpace(endpoint.Provider)
	pathValue := strings.Trim(strings.TrimSpace(endpoint.Path), "/")
	if pathValue == "" {
		legacy, err := mirrorLegacyPath(endpoint.URL)
		if err != nil {
			return MirrorEndpointIdentity{}, err
		}
		pathValue = legacy
	}
	if provider == "" || pathValue == "" {
		return MirrorEndpointIdentity{}, fmt.Errorf("provider and path are required")
	}
	return MirrorEndpointIdentity{Provider: provider, Path: pathValue}, nil
}

func mirrorLegacyPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	var pathValue string
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse legacy URL")
		}
		pathValue = parsed.Path
	} else if at := strings.LastIndex(raw, "@"); at >= 0 {
		if colon := strings.Index(raw[at+1:], ":"); colon >= 0 {
			pathValue = raw[at+1+colon+1:]
		}
	}
	pathValue = strings.TrimSuffix(strings.Trim(strings.TrimSpace(pathValue), "/"), ".git")
	parts := strings.Split(pathValue, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("legacy URL does not contain a safe repository path")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) {
			return "", fmt.Errorf("legacy URL contains an unsafe repository path")
		}
	}
	return pathValue, nil
}
