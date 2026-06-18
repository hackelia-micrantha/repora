# Repora v0.1 Implementation Plan and Checklist

**Date**: May 3, 2026  
**Target Release**: Q3 2026  
**Total Effort**: 12-16 weeks  

This document outlines the phased implementation plan for Repora v0.1, based on RFC-0001. It includes detailed checklists for each phase, with tasks, subtasks, dependencies, and success criteria.

## Overview

Repora v0.1 focuses on:
- CLI-first operation via `repoctl`
- Status checking for GitLab repositories
- Declarative config via `repora.yaml`
- Safety-preserving divergence detection
- Unidirectional canonical-to-mirror synchronization

## v0.1 Scope and Boundaries

**In Scope**:
- Existing repositories only (no creation)
- GitLab provider support
- Mirror mode only
- Basic auth via tokens
- Local cache storage

**Out of Scope (Deferred)**:
- Global config file
- LFS support
- Multi-provider CI/CD
- Artifact registries
- Advanced templating integration
- Non-GitLab providers

## Dependencies

- Go 1.22+
- Git CLI 2.0+
- YAML library (gopkg.in/yaml.v3)
- GitLab access for testing

## Phase 1: Complete Single-Repository Status Command (Weeks 1-4)

**Goal**: Fully implement `repoctl status` for one repository with human and JSON output.  
**Dependencies**: Existing codebase.  
**Success Criteria**: `repoctl status -f repora.yaml` works for one repo, outputs correct state.

### Week 1: Fix Config Parsing
- [x] Replace custom parser in `config.go` with `gopkg.in/yaml.v3`
- [x] Add YAML dependency: `go get gopkg.in/yaml.v3`
- [x] Run `go mod tidy` to update go.mod and go.sum
- [x] Update `parse()` function to use YAML unmarshaling
- [x] Update `validate()` function for basic schema checks
- [x] Test parsing with `testdata/repora.yaml`
- [x] Update config_test.go with YAML tests

### Week 2-3: Complete Status Logic
- [x] Finish `Check()` in `status.go`: Implement fetch operations
- [x] Add divergence detection using `RevListLeftRightCount`
- [x] Handle ahead/behind counts accurately
- [ ] Add error handling for auth failures and invalid repos
- [x] Implement human-readable output in `main.go` (non-JSON mode)
- [x] Update status_test.go with unit tests for Check()

### Week 4: Add Timeouts and Error Handling
- [x] Add 30s timeout to Git operations in `git.go` (use context.WithTimeout)
- [x] Implement recovery for corrupted caches (detect and re-clone)
- [x] Update error model per SPEC-0006 (clear error messages)
- [x] Add logging for debug info
- [x] Test timeout behavior and error scenarios

## Phase 2: Multi-Repository Support and Validation (Weeks 5-7)

**Goal**: Extend status to handle multiple repos, add full schema validation.  
**Dependencies**: Phase 1 complete.  
**Success Criteria**: Status works for full config file with multiple repos.

### Week 5: Multi-Repo Status
- [x] Modify `main.go` to iterate over `spec.Repos`
- [x] Add concurrency with goroutines (limit to 5 concurrent per ADR-0007)
- [x] Use sync.WaitGroup for aggregation
- [x] Update JSON output to include all repos
- [x] Handle partial failures gracefully

### Week 6: Schema Validation
- [x] Enhance `validate()` in `config.go` for required fields (id, canonical, mirrors)
- [x] Add provider validation (only "gitlab" supported in v0.1)
- [x] Add mode validation (default to "mirror")
- [x] Implement explicit defaults for optional fields
- [x] Add validation tests in config_test.go

### Week 7: Ref Resolution and Divergence
- [x] Implement remote HEAD resolution using `origin/HEAD`
- [x] Refine divergence classification (handle edge cases)
- [ ] Add support for branch-specific checks (future-proof)
- [x] Update status tests for multi-repo scenarios
- [ ] Performance testing with mock repos

## Phase 3: Plan Command (Weeks 8-10)

**Goal**: Implement `repoctl plan` to show proposed changes.  
**Dependencies**: Phase 2 complete.  
**Success Criteria**: `repoctl plan` shows safe changes without applying.

### Week 8: Plan Logic
- [x] Add `plan` subcommand parsing in `main.go`
- [x] Create `internal/plan` package
- [x] Implement diff generation per ADR-0010 (unified diff model)
- [x] Output human-readable plan (e.g., "Mirror will receive X commits")
- [x] Add plan-specific Result struct

### Week 9: Dry-Run Mode
- [x] Add `--dry-run` flag to status and plan commands
- [x] Simulate sync operations without actual changes
- [x] Ensure dry-run matches real plan output
- [x] Add tests for dry-run behavior

### Week 10: Partial Success Handling
- [x] For multi-repo, report per-repo status in JSON
- [x] Aggregate overall success/failure
- [x] Add `--continue-on-error` flag
- [x] Update error handling for partial failures

## Phase 4: Apply Command and Safety (Weeks 11-13)

**Goal**: Implement `repoctl apply` with safety checks.  
**Dependencies**: Phase 3 complete.  
**Success Criteria**: Full sync workflow works safely.

### Week 11: Apply Logic
- [x] Add `apply` subcommand in `main.go`
- [x] Create `internal/apply` package
- [x] Implement push to mirrors (unidirectional per ADR-0002)
- [x] Only proceed if status is safe (no divergence)
- [x] Add apply-specific Result struct

### Week 12: Safety Features
- [x] Add `--force` flag for risky operations
- [x] Implement auth model (delegate to system Git per ADR-0005)
- [x] Secure token handling (no in-process token handling in v0.1)
- [x] Add progress indicators (e.g., for long fetches)
- [x] Implement concurrency limits for apply

### Week 13: Templating Prototype
- [ ] Isolated README templating using Go templates
- [ ] Create sample templates
- [ ] Test template rendering
- [ ] Defer integration into apply (per RFC next steps)

## Phase 5: Testing, Documentation, and Release (Weeks 14-16)

**Goal**: Comprehensive testing and v0.1 release.  
**Dependencies**: All phases complete.  
**Success Criteria**: Full test suite passes, v0.1 released.

### Week 14: Comprehensive Tests
- [ ] Add all test cases: auth failure, invalid config, network errors
- [ ] Implement performance benchmarks (memory, time)
- [ ] Add integration tests with real Git repos (if possible)
- [ ] CI/CD setup for GitLab (per resolved gaps)

### Week 15: Documentation and Examples
- [ ] Update project README with usage instructions
- [ ] Add examples for repora.yaml configurations
- [x] Validate against specs (e.g., SPEC-0002 JSON format)
- [ ] Create troubleshooting guide
- [x] Add code formatting and linting (gofmt, golint)

### Week 16: Release Prep
- [ ] Build cross-platform binaries (Windows, Linux, macOS)
- [ ] Tag v0.1 in Git
- [ ] Create changelog and release notes
- [ ] Final manual validation on real repos
- [ ] Publish release notes

## Risks and Mitigations

- **Git Timeouts**: Configurable timeouts, retry logic.
- **Auth Issues**: Clear error messages, env var documentation.
- **Performance**: Limit concurrency, monitor resources.
- **Data Loss**: Strict checks, no force by default.

## Dependencies

- Go 1.22+
- Git CLI
- YAML library (gopkg.in/yaml.v3)

## Identified Gaps and Future Considerations

- **Disk Usage Optimization (ADR-0008)**: No cache size limits or cleanup in v0.1; monitor and add in v0.2.
- **Memory Monitoring**: Basic concurrency limits, but no profiling; add benchmarks.
- **CI/CD for Project**: No GitLab CI setup for building repoctl itself; consider adding.
- **Integration Testing**: Limited real-world testing; use mocks for GitLab API if needed.
- **User Testing**: No beta testing phase; consider internal validation.
- **Security Audit**: Basic auth, but no formal review; ensure tokens are not exposed.

## Tracking

- Use GitHub issues for task tracking.
- Weekly check-ins to review progress.
- Automated tests run on each commit.
