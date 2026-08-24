package posturepolicy

import (
	"fmt"
	"strings"

	"repoctl/internal/posture"
)

// ArtifactSet contains the versioned posture collector artifacts to converge.
// Empty fields are ignored. MirrorRepoUID is required when Mirrors is supplied.
type ArtifactSet struct {
	Inventory     []byte
	Documentation []byte
	Hooks         []byte
	Commits       []byte
	Mirrors       []byte
	MirrorRepoUID string
}

// ConvergeArtifacts validates and atomically converges collector artifacts into
// the normalized posture policy input contract.
func ConvergeArtifacts(artifacts ArtifactSet) (Inputs, error) {
	if len(artifacts.Inventory) == 0 && len(artifacts.Documentation) == 0 && len(artifacts.Hooks) == 0 && len(artifacts.Commits) == 0 && len(artifacts.Mirrors) == 0 {
		return Inputs{}, fmt.Errorf("at least one posture artifact is required")
	}
	if len(artifacts.Mirrors) == 0 && strings.TrimSpace(artifacts.MirrorRepoUID) != "" {
		return Inputs{}, fmt.Errorf("mirror repository uid requires a mirror posture artifact")
	}
	if len(artifacts.Mirrors) != 0 && strings.TrimSpace(artifacts.MirrorRepoUID) == "" {
		return Inputs{}, fmt.Errorf("mirror repository uid is required with a mirror posture artifact")
	}

	var inputs Inputs

	if len(artifacts.Inventory) != 0 {
		var inventory posture.Inventory
		if err := decodeStrict(artifacts.Inventory, &inventory); err != nil {
			return Inputs{}, fmt.Errorf("parse posture inventory: %w", err)
		}
		if err := AddInventory(&inputs, inventory); err != nil {
			return Inputs{}, fmt.Errorf("converge posture inventory: %w", err)
		}
	}

	if len(artifacts.Documentation) != 0 {
		var inventory posture.DocumentationInventory
		if err := decodeStrict(artifacts.Documentation, &inventory); err != nil {
			return Inputs{}, fmt.Errorf("parse documentation posture inventory: %w", err)
		}
		if err := AddDocumentation(&inputs, inventory); err != nil {
			return Inputs{}, fmt.Errorf("converge documentation posture inventory: %w", err)
		}
	}

	if len(artifacts.Hooks) != 0 {
		var inventory posture.HooksInventory
		if err := decodeStrict(artifacts.Hooks, &inventory); err != nil {
			return Inputs{}, fmt.Errorf("parse hooks posture inventory: %w", err)
		}
		if err := AddHooks(&inputs, inventory); err != nil {
			return Inputs{}, fmt.Errorf("converge hooks posture inventory: %w", err)
		}
	}

	if len(artifacts.Commits) != 0 {
		var inventory posture.CommitInventory
		if err := decodeStrict(artifacts.Commits, &inventory); err != nil {
			return Inputs{}, fmt.Errorf("parse commit posture inventory: %w", err)
		}
		if err := AddCommits(&inputs, inventory); err != nil {
			return Inputs{}, fmt.Errorf("converge commit posture inventory: %w", err)
		}
	}

	if len(artifacts.Mirrors) != 0 {
		var inventory posture.MirrorInventory
		if err := decodeStrict(artifacts.Mirrors, &inventory); err != nil {
			return Inputs{}, fmt.Errorf("parse mirror posture inventory: %w", err)
		}
		if err := inventory.Validate(); err != nil {
			return Inputs{}, err
		}
		repository, err := mirrorGitHubRepository(inventory, artifacts.MirrorRepoUID)
		if err != nil {
			return Inputs{}, err
		}
		if inputs.Repository == "" {
			inputs = NewInputs(repository)
		} else if !strings.EqualFold(inputs.Repository, repository) {
			return Inputs{}, fmt.Errorf("mirror posture repository %q does not match policy inputs repository %q", repository, inputs.Repository)
		}
		if err := AddMirrors(&inputs, inventory, artifacts.MirrorRepoUID); err != nil {
			return Inputs{}, fmt.Errorf("converge mirror posture inventory: %w", err)
		}
	}

	if err := inputs.Validate(); err != nil {
		return Inputs{}, err
	}
	return inputs, nil
}

func mirrorGitHubRepository(inventory posture.MirrorInventory, repoUID string) (string, error) {
	var selected *posture.MirrorRepositoryFacts
	for idx := range inventory.Repos {
		if inventory.Repos[idx].UID != repoUID {
			continue
		}
		if selected != nil {
			return "", fmt.Errorf("mirror posture contains duplicate repository uid %q", repoUID)
		}
		selected = &inventory.Repos[idx]
	}
	if selected == nil {
		return "", fmt.Errorf("mirror posture repository uid %q was not found", repoUID)
	}

	paths := []string{}
	if strings.EqualFold(selected.Canonical.Identity.Provider, "github") {
		paths = append(paths, selected.Canonical.Identity.Path)
	}
	for _, mirror := range selected.Mirrors {
		if strings.EqualFold(mirror.Identity.Provider, "github") {
			paths = append(paths, mirror.Identity.Path)
		}
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("mirror posture repository uid %q has no GitHub endpoint to correlate with repository posture", repoUID)
	}
	for _, path := range paths[1:] {
		if !strings.EqualFold(path, paths[0]) {
			return "", fmt.Errorf("mirror posture repository uid %q has ambiguous GitHub endpoints", repoUID)
		}
	}
	return paths[0], nil
}
