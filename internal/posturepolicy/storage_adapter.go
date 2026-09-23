package posturepolicy

import "repoctl/internal/posture"

// AddStorage converges only facts about the materialized local object database.
// Its repository identity is asserted by the caller and is not a verified remote
// identity. Policies must not mistake these values for canonical repository size.
func AddStorage(inputs *Inputs, inventory posture.StorageInventory) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	if err := requireRepository(inputs, inventory.Repository.FullName); err != nil {
		return err
	}
	entries := map[string]FactInput{
		"storage.scope": observedInput(inventory.Scope, inventory.Evidence),
	}
	addConverted(entries, "storage.local.loose_count", inventory.Objects.LooseCount)
	addConverted(entries, "storage.local.loose_bytes", inventory.Objects.LooseBytes)
	addConverted(entries, "storage.local.packed_count", inventory.Objects.PackedCount)
	addConverted(entries, "storage.local.packed_bytes", inventory.Objects.PackedBytes)
	addConverted(entries, "storage.local.pack_count", inventory.Objects.PackCount)
	addConverted(entries, "storage.checkout.shallow", inventory.GitState.Shallow)
	addConverted(entries, "storage.checkout.promisor_configured", inventory.GitState.PromisorConfigured)
	return addEntries(inputs, entries)
}
