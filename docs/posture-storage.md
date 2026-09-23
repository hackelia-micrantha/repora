# Local Git storage posture v1

Status: Current when the linked implementation is merged. Tracking: #170.

`repoctl posture storage --repository OWNER/REPO --path LOCAL_GIT_REPOSITORY` emits `repora.posture-storage` v1 JSON for an existing local checkout or bare repository. The identity is **operator asserted**; it is not verified against a remote. Run it on a repository you have independently identified, then use the same identity when converging artifacts.

The output deliberately uses `scope: local_object_database`. Counts and bytes reflect **materialized local Git objects**, not the complete object population or size of a canonical hosted repository. `shallow: false` is not proof that every remote ref or historical object is present. `promisor_configured: false` means relevant local configuration was not detected; it is not proof that no promised objects are missing.

## Example

```sh
repoctl posture storage --repository hackelia-micrantha/repora --path ./repora > storage.json
repoctl posture converge --storage storage.json > facts.json
repoctl posture report --profile policy.json --facts facts.json --as-of 2026-09-23 --format markdown
```

`--repository` must be an OWNER/REPO identity; `--path` is a pre-existing local Git repository. This command never clones or fetches. Future repository-size thresholds for canonical history must not be applied to `storage.local.*` facts as if they were authoritative hosted-repository measurements.

## Artifact and facts

The v1 contract is [`posture-storage-v1.schema.json`](../schemas/posture-storage-v1.schema.json). It records local loose object count/bytes, packed object count/bytes, and pack count using `git count-objects -v`. It also records `git rev-parse --is-shallow-repository` and observed local promisor/partial-clone configuration.

The collector uses `GIT_NO_LAZY_FETCH=1`, `GIT_OPTIONAL_LOCKS=0`, noninteractive Git, and a bounded environment to avoid implicit network materialization, optional locks, and inherited Git target overrides. It neither inspects historical blob contents nor performs Git maintenance, garbage collection, pruning, LFS fetches, or provider mutation. Git stderr and local paths are not persisted as evidence.

The offline convergence adapter adds `storage.scope`, `storage.local.loose_count`, `storage.local.loose_bytes`, `storage.local.packed_count`, `storage.local.packed_bytes`, `storage.local.pack_count`, `storage.checkout.shallow`, and `storage.checkout.promisor_configured` through the existing posture policy input contract. It does **not** produce canonical-size, complete-history, historical-large-blob, growth, or reclaim facts. Unknown/unavailable facts retain their original state.

This is the first bounded measurement slice of #170. Complete historical-object analysis, provable partial-clone completeness, full fixture coverage, and any policy for canonical repository size require subsequent implementation and review. Checkout optimization is tracked by #171, local maintenance by #172, and advisory remediation by #173.
