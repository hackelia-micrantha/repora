package posture

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	StorageInventoryKind = "repora.posture-storage"
	StorageInventoryVersion = 1
	StorageLocalScope = "local_object_database"
)

// StorageObjectFacts describe bytes and object counts in this checkout's local
// Git object database, not the size of a canonical provider repository.
type StorageObjectFacts struct {
	LooseCount Fact[int64] `json:"loose_count"`
	LooseBytes Fact[int64] `json:"loose_bytes"`
	PackedCount Fact[int64] `json:"packed_count"`
	PackedBytes Fact[int64] `json:"packed_bytes"`
	PackCount Fact[int64] `json:"pack_count"`
}

// StorageGitState describes observed local checkout properties. In particular,
// shallow=false and promisor_configured=false do NOT prove complete remote history.
type StorageGitState struct {
	Shallow Fact[bool] `json:"shallow"`
	PromisorConfigured Fact[bool] `json:"promisor_configured"`
}

type StorageInventory struct {
	Kind string `json:"kind"`
	Version int `json:"version"`
	Repository RepositoryIdentity `json:"repository"`
	Scope string `json:"scope"`
	Objects StorageObjectFacts `json:"objects"`
	GitState StorageGitState `json:"git_state"`
	Evidence []Evidence `json:"evidence"`
}

func NewStorageInventory(fullName string) StorageInventory {
	return StorageInventory{
		Kind: StorageInventoryKind,
		Version: StorageInventoryVersion,
		Repository: RepositoryIdentity{Provider: "github", FullName: fullName},
		Scope: StorageLocalScope,
		Evidence: []Evidence{},
	}
}

func (i StorageInventory) Validate() error {
	if i.Kind != StorageInventoryKind || i.Version != StorageInventoryVersion {
		return fmt.Errorf("unsupported storage posture contract: kind=%q version=%d", i.Kind, i.Version)
	}
	if i.Repository.Provider != "github" {
		return fmt.Errorf("storage posture provider must be github")
	}
	if _, _, err := splitGitHubFullName(i.Repository.FullName); err != nil {
		return err
	}
	if i.Scope != StorageLocalScope {
		return fmt.Errorf("unsupported storage observation scope %q", i.Scope)
	}
	if i.Evidence == nil {
		return fmt.Errorf("storage evidence array is required")
	}
	checks := []struct {
		name string
		value Fact[int64]
	}{
		{"loose_count", i.Objects.LooseCount},
		{"loose_bytes", i.Objects.LooseBytes},
		{"packed_count", i.Objects.PackedCount},
		{"packed_bytes", i.Objects.PackedBytes},
		{"pack_count", i.Objects.PackCount},
	}
	for _, check := range checks {
		if err := validateFact(check.name, check.value); err != nil {
			return err
		}
		if check.value.State == StateObserved && *check.value.Value < 0 {
			return fmt.Errorf("storage fact %s cannot be negative", check.name)
		}
	}
	if err := validateFact("shallow", i.GitState.Shallow); err != nil {
		return err
	}
	if err := validateFact("promisor_configured", i.GitState.PromisorConfigured); err != nil {
		return err
	}
	for _, evidence := range i.Evidence {
		if strings.TrimSpace(evidence.Source) == "" || strings.TrimSpace(evidence.Reference) == "" {
			return fmt.Errorf("storage evidence requires source and reference")
		}
	}
	return nil
}

func (i StorageInventory) Marshal() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode storage posture inventory: %w", err)
	}
	return append(data, '\n'), nil
}
