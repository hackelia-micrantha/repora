# RFC-0001: Repora Documentation Set (v0.1 Draft)

Status: Draft

Repora is an experimental repository control-plane tool for declaratively describing, observing, and reconciling Git repository topology across local caches and remote hosting systems. Version 0.1 focuses on existing repositories, GitLab-oriented workflows, full-ref mirroring, CLI-first operation via `repoctl`, and safe divergence detection.

## Problem Statement

Managing Git repository topology across multiple remote hosting systems (e.g., GitLab) is error-prone and manual. Teams struggle with keeping mirrors synchronized, detecting divergences, and ensuring consistency without risking data loss. Repora aims to provide a declarative, safe way to observe and reconcile repository states.

## Scope (v0.1)

- Existing repositories only (no creation)
- GitLab-oriented canonical workflows
- Full-ref mirroring
- CLI-first via `repoctl`
- Safety-preserving divergence detection

## Target Audience

- DevOps engineers managing GitLab repositories
- Teams needing automated repository synchronization
- Prerequisites: GitLab access, Go environment

## Requirements

Requirements for this RFC are maintained as separate records:

- [REQ-0001: Core System Requirements](0001-repora-documentation-set-v0.1/requirements/0001-core-system-requirements.md)
- [REQ-0002: CLI Requirements](0001-repora-documentation-set-v0.1/requirements/0002-cli-requirements.md)
- [REQ-0003: Repository Content and Workflow Control (Future)](0001-repora-documentation-set-v0.1/requirements/0003-repository-content-and-workflow-control-future.md)

## Architecture Decisions

Architecture decisions for this RFC are maintained as separate records:

- [ADR-0001: Git as the Primary State Authority](0001-repora-documentation-set-v0.1/decisions/0001-git-as-primary-state-authority.md)
- [ADR-0002: Unidirectional Canonical-to-Mirror Synchronization](0001-repora-documentation-set-v0.1/decisions/0002-unidirectional-canonical-to-mirror-synchronization.md)
- [ADR-0003: Divergence Handling](0001-repora-documentation-set-v0.1/decisions/0003-divergence-handling.md)
- [ADR-0004: Existing Repositories Only in v0.1](0001-repora-documentation-set-v0.1/decisions/0004-existing-repositories-only-v0.1.md)
- [ADR-0005: Authentication Model](0001-repora-documentation-set-v0.1/decisions/0005-authentication-model.md)
- [ADR-0006: Storage Model](0001-repora-documentation-set-v0.1/decisions/0006-storage-model.md)
- [ADR-0007: Concurrency Model](0001-repora-documentation-set-v0.1/decisions/0007-concurrency-model.md)
- [ADR-0008: Disk Usage Optimization](0001-repora-documentation-set-v0.1/decisions/0008-disk-usage-optimization.md)
- [ADR-0009: Scope Boundary (v0.1 vs Future)](0001-repora-documentation-set-v0.1/decisions/0009-scope-boundary-v0.1-vs-future.md)
- [ADR-0010: Unified Diff Model](0001-repora-documentation-set-v0.1/decisions/0010-unified-diff-model.md)

## Specifications

Specifications for this RFC are maintained as separate records:

- [SPEC-0001: `repora.yaml` Schema (v0.1)](0001-repora-documentation-set-v0.1/specs/0001-repora-yaml-schema-v0.1.md)
- [SPEC-0002: JSON Output](0001-repora-documentation-set-v0.1/specs/0002-json-output.md)
- [SPEC-0003: Sample Configuration](0001-repora-documentation-set-v0.1/specs/0003-sample-configuration.md)
- [SPEC-0004: Architecture (v0.1)](0001-repora-documentation-set-v0.1/specs/0004-architecture-v0.1.md)
- [SPEC-0005: Sync Algorithm (v0.1)](0001-repora-documentation-set-v0.1/specs/0005-sync-algorithm-v0.1.md)
- [SPEC-0006: Error Model](0001-repora-documentation-set-v0.1/specs/0006-error-model.md)
- [SPEC-0007: Performance and Memory Considerations](0001-repora-documentation-set-v0.1/specs/0007-performance-and-memory-considerations.md)

## Resolved Gaps

1. Global configuration file: Deferred to v0.2; use per-command config in v0.1.
2. Timeout behavior for Git operations: Default 30 seconds, configurable via repora.yaml.
3. Remote HEAD resolution: Uses Git's default remote HEAD (typically main/master branch).
4. LFS support: Explicitly unsupported in v0.1; deferred to future versions.
5. Recovery for corrupted local caches: Manual intervention required; cache recreation via repoctl clean.
6. repoctl apply --json partial success: Reports partial success with detailed per-repository status.
7. Templating engine: Go templates for v0.1.
8. CI/CD abstraction: Scoped to GitLab only in v0.1.
9. Artifact registry: Scoped to containers in v0.1.

## Next Steps

### Getting Started

1. Clone the repository and run `go mod tidy`.
2. Create a sample repora.yaml (see SPEC-0003).
3. Run `repoctl status --config repora.yaml` for a single repository.

### Implementation Priorities

1. Implement `repoctl status` for a single repository (1-2 weeks).
2. Validate schema parsing and explicit defaults (1 week).
3. Validate remote fetch, ref resolution, and divergence classification (2 weeks).
4. Implement human and JSON output for status (1 week).
5. Add tests for equal, behind, ahead, diverged, auth failure, and invalid configuration (2 weeks).
6. Introduce plan/apply only after status behavior is correct and stable (3 weeks).
7. Prototype README templating as an isolated feature before integrating into core reconciliation (2 weeks).

## Timeline

Target v0.1 release: Q3 2026

## Risks and Mitigations

- Data loss from sync failures: Strict divergence checks prevent overwrites; manual approval required for risky operations.
- Performance issues: Limit to small repository sets in v0.1; monitor memory usage.
- Authentication failures: Clear error messages and fallback to manual auth setup.

## Feedback

This is a draft RFC. Provide feedback via GitHub issues in the project repository or by contacting the maintainers.
