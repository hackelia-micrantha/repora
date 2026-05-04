# ADR-0005: Authentication Model

Status: Draft

## Decision

Repora v0.1 shall delegate authentication to system Git, including SSH
configuration, credential helpers, and existing local Git authentication
mechanisms.

## Rationale

Git authentication is already environment-specific and frequently integrated
with SSH agents, credential managers, enterprise proxies, and platform-specific
helpers. Re-implementing credential handling in Repora would increase security
and portability risk.

## Consequences

- Repora does not store credentials in `repora.yaml`
- Operators are responsible for configuring Git authentication outside Repora
- Future integrations may document or support token-based authentication, Vault,
  SOPS, or other secret-management mechanisms, but these are out of scope for
  v0.1
