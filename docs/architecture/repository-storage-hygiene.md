# Repository storage hygiene and efficient checkout architecture

Status: Proposed

Issue: #166

## Purpose

Define the architecture boundary for repository size/storage posture, efficient Repora-managed checkout acquisition, safe local Git maintenance, and future destructive history cleanup.

This document is intentionally design-only. It does not describe implemented behavior and grants no new repository mutation authority.

## Core thesis

Repora should obtain the **smallest Git material required by an operation**, while separately observing repository storage health and performing only explicitly authorized maintenance against Repora-managed local state.

These are four distinct domains:

```text
operation requirements
        |
        v
checkout acquisition ---------> local Git material
                                   |
                                   v
storage observation ----------> normalized posture facts
                                   |
                                   v
local maintenance -----------> managed cache/workspace optimization

canonical history cleanup ---> separate destructive authority boundary
```

The architecture must not collapse shallow/sparse checkout, garbage collection, pruning, LFS migration, and history rewriting into one generic "cleanup" command. They affect different state and carry materially different recovery risks.

## Domain boundaries

### 1. Checkout acquisition

Checkout acquisition decides how much Git history, tree data, and blob content a Repora operation needs locally.

It may use mechanisms such as:

- bounded shallow history;
- sparse checkout;
- partial clone/blob filtering;
- ordinary full clone/fetch when required or when optimization is unsupported.

Checkout optimization changes local acquisition mechanics. It does **not** change durable repository identity, canonical/mirror topology, ref policy, or provider authority.

### 2. Storage observation

Storage observation produces evidence-backed facts about repository/object-store health.

It is read-first and should reuse the existing posture evidence semantics rather than create a parallel health-reporting framework.

### 3. Local maintenance

Local maintenance may compact or prune explicitly managed local Git state under a reviewed policy.

Examples include Git's supported maintenance/repack/gc mechanisms. Local maintenance is not equivalent to canonical repository history reduction.

### 4. Canonical history cleanup

Removing reachable historical objects requires rewriting refs/history or changing repository content representation. That is a destructive repository mutation domain and is deferred from the initial implementation.

Any future implementation must use a separate exact reviewed plan and coordinate canonical/mirror/recovery effects explicitly.

## Checkout requirement model

Operations should declare semantic requirements rather than clone flags.

A conceptual requirement contains at least four axes:

```text
History
  none       no commit traversal required
  head       current selected ref only
  bounded    an explicit history window is required
  full       all relevant reachable history is required

Tree
  none       refs/object metadata only
  selected   explicit path set/subtrees
  full       complete working tree/tree inventory

Blobs
  none       blob payloads unnecessary
  selected   only content required by selected paths/objects
  lazy       blob content may be fetched on demand
  full       complete reachable blob content required

LFS
  pointer-only
  selected-content
  full-content
```

Exact serialized names are an implementation decision. The important property is that callers state **what evidence/material they require**, while the Git layer chooses the mechanism.

Conceptually:

```go
type CheckoutRequirement struct {
    History HistoryRequirement
    Tree    TreeRequirement
    Blobs   BlobRequirement
    LFS     LFSRequirement
}
```

The requirement is an internal execution contract, not durable repository identity.

## Example operation requirements

| Operation | History | Tree | Blobs | Notes |
| --- | --- | --- | --- | --- |
| README/document inspection | head | selected | selected | candidate sparse/partial checkout |
| CI/config inspection | head | selected | selected | workflow/config paths only |
| workspace bootstrap | head/bounded | full or configured sparse set | lazy/selected | depends on workspace contract |
| mirror reconciliation | ref-specific | none/minimal | as required by existing Git mechanics | must preserve current reconciliation semantics |
| bounded commit posture | bounded | selected metadata | selected | completeness must reflect history window |
| complete large-object analysis | full | none/minimal | object metadata/full as required | cannot claim completeness from shallow state |
| history rewrite | full | full as required | full | destructive domain; deferred |

This table is illustrative. Existing implemented operations remain unchanged until explicitly migrated.

## Strategy selection

The Git layer may map a semantic requirement to an acquisition strategy.

Example decision flow:

```text
needs full history?
  yes -> full history acquisition
  no  -> bounded/shallow acquisition where supported

needs full tree?
  yes -> ordinary/full checkout tree
  no  -> sparse checkout where supported

needs all blob payloads?
  yes -> ordinary object acquisition
  no  -> partial clone/blob filter where supported
```

Optimizations are best-effort only when the operation's evidence requirements remain satisfied.

### Safe fallback

If a server/provider/Git version cannot satisfy an optimization, Repora may fall back to a broader acquisition only when:

- the broader acquisition does not widen mutation authority;
- credential scope remains unchanged;
- the operation still receives correct evidence semantics;
- the fallback is visible when it materially changes cost or completeness.

An optimization failure must never be represented as a successful complete observation if required material was not obtained.

## Sparse checkout semantics

Sparse checkout reduces working-tree material, not necessarily repository object history or object-store size.

Repora must therefore avoid claims such as:

```text
sparse checkout => small repository
```

Sparse checkout is useful for operations that need a bounded path set, especially workspace/bootstrap and file-oriented analysis, but it is orthogonal to shallow/partial history and object acquisition.

Path sets must use the same containment and normalization principles as the workspace contract under #147. Repository-controlled sparse specifications must not create filesystem escape or arbitrary local-path authority.

## Shallow history semantics

A shallow clone reduces transferred history and local reachable-history material, but it also limits what can be concluded from history.

Any collector consuming shallow state must preserve evidence scope explicitly.

For example:

```text
history:
  completeness: bounded
  depth: 1
```

must not be adapted into a fact that implies:

```text
no large historical blobs exist
```

or:

```text
repository growth has always been stable
```

when the evidence only covers the selected shallow boundary.

## Partial clone semantics

Partial clone can avoid transferring blob payloads until required. It can substantially reduce local cache cost for metadata/ref-oriented operations.

Important boundaries:

- promisor/partial-clone state must be detectable;
- missing-on-purpose blob content is not corruption;
- an operation requiring complete blob evidence must explicitly materialize it or fail with incomplete/unavailable evidence;
- invoking an operation that causes lazy blob fetches must not silently broaden the original scope without being expected by the operation contract.

## Git LFS boundary

Git LFS is a repository content/storage model, not merely a checkout optimization.

Initial workspace/bootstrap behavior should remain pointer/content conservative:

- do not implicitly fetch all LFS content;
- fetch selected LFS content only when an explicit operation requires it;
- represent LFS migration as remediation guidance, not automatic mutation.

Future LFS migration affects canonical repository content/history and therefore requires a different authority boundary from local clone optimization.

## Storage posture facts

The first storage capability should be evidence collection, not cleanup.

Candidate normalized facts include:

- Git object database size;
- packed object count/size;
- loose object count/size;
- pack count;
- prune-packable/garbage signals where Git exposes them safely;
- working-tree size for an explicitly selected local checkout;
- shallow repository state;
- partial/promisor repository state;
- largest reachable objects/blobs within the declared observation scope;
- historical largest objects when complete/bounded history is intentionally traversed;
- Git LFS pointer/content signals;
- generated/binary artifact candidates;
- unreachable-object estimates where safely observable;
- growth/change between comparable prior observations;
- observation completeness/truncation.

The exact v1 fact set should prefer facts that are deterministic, portable across supported Git versions, and cheap enough to collect deliberately.

## Fact completeness

Storage facts need one additional concern beyond the existing observed/unknown/unavailable model: **scope/completeness**.

A fact can be observed and still be bounded.

Examples:

```text
largest_blob = 14 MiB
state = observed
scope = selected_head
```

and:

```text
largest_blob = 612 MiB
state = observed
scope = all_reachable_history
```

are not equivalent claims.

The implementation should reuse existing fact states while binding facts or the containing artifact to an explicit observation scope. `truncated`, `shallow`, `partial`, and selected-path/history limits must never disappear during convergence into policy inputs.

## Growth semantics

Absolute repository size is insufficient as the primary health signal.

Repora should support comparison of compatible observations so policy can express conditions such as:

```text
repository A: 900 MiB, +14 MiB/year
repository B: 240 MiB, +170 MiB/month
```

The second repository may deserve more attention despite being smaller.

Growth evidence must bind compatible repository identity and comparable observation scope. A full-history observation and a shallow/filtered observation must not be compared as if they measured the same thing.

## Measurement implementation direction

Prefer Git-native plumbing over ad-hoc filesystem inference where it gives stable semantics.

Candidate mechanisms include:

- `git count-objects -v` / `-vH` for object-store summary;
- `git rev-list --objects` for reachable-object enumeration under an explicit ref/history scope;
- `git cat-file --batch-check` for object type/size metadata;
- repository configuration inspection for shallow/partial/promisor/LFS state;
- filesystem measurement only for explicitly local concerns such as working-tree footprint.

Exact command selection belongs in implementation and tests. Parser contracts should not depend on localized human output when machine-stable alternatives exist.

## Posture integration

Storage posture should reuse the completed #124 model:

```text
storage observation
  -> versioned storage artifact
  -> typed convergence adapter
  -> existing posture-policy inputs
  -> existing report engine
```

Do not create a second severity, exception, remediation, or Markdown report engine.

Repository-owned observation configuration may select measurement scope, but as with existing posture profiles it must not assign severity, suppress external policy, or authorize cleanup.

## Local maintenance model

The first mutating storage slice should target only state Repora can prove it owns/manages, such as a Repora cache or an explicitly managed workspace maintenance target.

Candidate operations:

- `git maintenance run`;
- repack/commit-graph/multi-pack-index maintenance selected through supported Git behavior;
- `git gc` under explicit policy;
- pruning only after an explicit recovery/retention horizon.

### Invariant: GC is not history cleanup

```text
git gc / maintenance
  -> repack reachable objects
  -> compact object storage
  -> optionally expire/prune unreachable objects under policy

it does not
  -> remove reachable historical blobs
  -> rewrite commits
  -> reduce canonical history merely because a file is old/large
```

Repora documentation and output must preserve this distinction.

## Ownership and target safety

Maintenance must fail closed unless the target is unambiguously authorized as a managed local repository/cache.

Initial rules should include:

- no recursive discovery-and-clean of arbitrary repositories;
- no deleting a user repository merely because it appears stale;
- no maintenance against a path that fails ownership/identity checks;
- no concurrent maintenance while another Repora operation actively uses the same managed Git state;
- no implicit LFS purge;
- no repository hook execution as a maintenance side effect;
- no credential persistence in plan/evidence.

## Concurrency

Storage maintenance and active clone/fetch/status/apply operations can contend over object stores and lock files.

Repora therefore needs one cache/repository-level exclusion mechanism shared by storage maintenance and existing operations before maintenance becomes automatic.

The architecture should prefer correctness over aggressive background cleanup. If safe coordination cannot be established, maintenance should be explicit/manual rather than asynchronous.

## Pruning and recovery horizon

Pruning unreachable objects changes local recovery options even though it does not rewrite reachable history.

Therefore pruning is a higher-risk local maintenance class than repack/commit-graph work.

A prune plan should state at least:

- target repository/cache identity;
- retention/recovery horizon;
- estimated unreachable storage where available;
- whether reflog expiry is involved;
- whether an independent archive/recovery representation exists when policy requires one.

Issue #156's archive/recovery semantics are relevant to future destructive cleanup but do not automatically authorize pruning.

## Remediation guidance

Storage posture may recommend actions without executing them.

Examples:

- stop tracking generated output prospectively;
- add appropriate ignore rules;
- migrate suitable file classes to Git LFS;
- investigate a specific historical large blob;
- run safe local maintenance;
- take a fresh clone when local repair is less useful than replacement;
- consider history rewrite only when reachable canonical history dominates storage.

Reports must distinguish:

```text
local reclaim estimate
```

from:

```text
canonical/history reclaim estimate
```

because GC may materially reduce the former while doing nothing to the latter.

## Deferred destructive history rewrite

Automatic history rewriting is outside the initial capability.

A future design must model at least:

```text
exact offender/object/ref inventory
  -> exact rewrite proposal
  -> affected canonical/mirror endpoint inventory
  -> recovery/archive preconditions
  -> review/authorization
  -> canonical ref rewrite
  -> mirror coordination
  -> stale-clone/reintroduction handling
  -> delayed prune/retention handling
  -> execution evidence
```

It must account for:

- commit/tag identity changes;
- signed commit/tag invalidation;
- release/provenance references;
- branch/ruleset/force protections;
- open pull requests;
- developer clones that can reintroduce old history;
- canonical/mirror ordering and partial failure;
- archive/recovery requirements;
- exact force authorization.

No existing mirror force authority should be interpreted as generic permission to rewrite repository history.

## Package ownership direction

The eventual implementation should keep responsibilities separated:

| Surface | Candidate responsibility | Must not own |
| --- | --- | --- |
| `internal/git` | Git command mechanics, capability detection, checkout/materialization primitives, object measurement primitives | posture severity/policy or destructive authority |
| checkout/workspace layer | operation requirement -> Git acquisition strategy | repository posture findings |
| `internal/posture` | versioned storage facts and read-only collector orchestration | GC/history rewrite execution |
| `internal/posturepolicy` | evaluate normalized storage facts using existing policy/report semantics | Git/provider access |
| future storage-maintenance package | exact local maintenance planning/apply/evidence | canonical history rewrite unless separately designed |
| `cmd/repoctl` | command routing/output | duplicated storage policy/mechanics |

Exact package names should follow the smallest implementation slice and need not introduce a package merely to mirror this table.

## Initial CLI direction

Names remain illustrative:

```text
repoctl posture storage ...
```

is preferred for read-only storage facts if it composes cleanly with existing posture commands.

Local maintenance may use a separate authority-visible command family, for example:

```text
repoctl storage plan-maintenance ...
repoctl storage maintain --plan-file ...
```

Read-only posture and mutation should not share a command whose behavior changes from observation to cleanup based on an incidental flag.

Workspace sparse/shallow configuration should remain under the workspace/bootstrap surface when #147 is implemented, while reusing the same internal checkout-requirement abstraction.

## Failure semantics

Initial storage observation should distinguish:

- complete observation;
- bounded/shallow/partial observation;
- unsupported Git/provider optimization;
- unavailable local/object evidence;
- malformed/inconsistent repository state;
- operational Git failure.

Maintenance should additionally distinguish:

- stale plan/target identity;
- target not Repora-managed;
- target currently in use/locked;
- maintenance accepted/succeeded;
- partial maintenance outcome;
- prune blocked by retention/recovery policy.

No cleanup failure should be hidden by deleting/recloning state automatically unless replacement is the exact reviewed operation.

## Security considerations

Repository content, Git configuration, attributes, filters, submodules, LFS settings, and hooks are untrusted inputs.

Implementation must preserve existing protections around:

- hook execution;
- credential helpers and environment secrets;
- external filters/commands;
- path/symlink containment;
- runtime transport resolution;
- command argument construction;
- durable evidence redaction.

Partial clone and LFS introduce implicit network-fetch opportunities. Operations must not accidentally turn a bounded read into unrestricted content retrieval through transparent filters or lazy fetches.

## Initial acceptance boundary

The first implementation slice following this proposal should prove only:

1. a versioned read-only storage observation contract;
2. explicit completeness/scope semantics;
3. deterministic Git-native measurement over a controlled local fixture;
4. convergence through the existing posture policy/report path;
5. no cleanup/history rewrite authority.

Efficient workspace checkout should follow as a separate low-risk slice under #147 compatibility. Local maintenance should follow only after managed-target ownership and locking are explicit.

## Non-goals

- Treating repository size as a universal quality score.
- Automatically cleaning every repository Repora can discover.
- Running GC as a substitute for historical blob cleanup.
- Automatically fetching all LFS objects.
- Automatically migrating files to LFS.
- Automatic canonical history rewriting.
- Cross-repository transactional cleanup.
- Provider artifact/package retention management.
- Background maintenance without safe target ownership and concurrency coordination.
