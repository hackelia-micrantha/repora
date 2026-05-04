# ADR-0004: Existing Repositories Only in v0.1

Status: Draft

## Decision

Repora v0.1 shall not create repositories through GitLab, GitHub, Bitbucket, or
other provider APIs. All canonical and mirror repositories must already exist.

## Rationale

Repository creation introduces provider-specific authorization, ownership,
namespace, visibility, and policy semantics. Excluding creation keeps v0.1
focused on Git topology correctness rather than provider administration.

## Consequences

- Operators must pre-provision repositories
- v0.1 can avoid provider SDKs and provider-specific API drift
- Future provider integration can be introduced behind a separate ADR and schema
  extension
