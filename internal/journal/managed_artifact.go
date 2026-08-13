package journal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"repoctl/internal/managedartifact"
)

const (
	ManagedArtifactVersion = 1
	ManagedArtifactKind    = "repora.io/managed-artifact-execution-record"
)

type ManagedArtifactRecord struct {
	Version      int                         `json:"version"`
	Kind         string                      `json:"kind"`
	ExecutionID  string                      `json:"execution_id"`
	Phase        Phase                       `json:"phase"`
	Mode         Mode                        `json:"mode"`
	Plan         ManagedArtifactPlanRef      `json:"plan"`
	Outcome      Outcome                     `json:"outcome"`
	FailureStage string                      `json:"failure_stage,omitempty"`
	Repositories []ManagedArtifactRepository `json:"repositories"`
}

type ManagedArtifactPlanRef struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	SHA256  string `json:"sha256"`
}

type ManagedArtifactRepository struct {
	UID            string  `json:"uid"`
	ID             string  `json:"id"`
	Provider       string  `json:"provider"`
	Path           string  `json:"path"`
	Branch         string  `json:"branch"`
	BaseOID        string  `json:"base_oid"`
	DesiredMode    string  `json:"desired_mode"`
	DesiredSHA256  string  `json:"desired_sha256"`
	PreparedCommit string  `json:"prepared_commit,omitempty"`
	Pushed         bool    `json:"pushed"`
	Outcome        Outcome `json:"outcome"`
}

func ManagedArtifactIntent(executionID string, plan managedartifact.Plan) (ManagedArtifactRecord, error) {
	ref, err := managedArtifactPlanRef(plan)
	if err != nil {
		return ManagedArtifactRecord{}, err
	}
	record := ManagedArtifactRecord{
		Version:      ManagedArtifactVersion,
		Kind:         ManagedArtifactKind,
		ExecutionID:  executionID,
		Phase:        PhaseIntent,
		Mode:         ModeApply,
		Plan:         ref,
		Outcome:      OutcomePlanned,
		Repositories: managedArtifactRepositories(plan),
	}
	for i := range record.Repositories {
		record.Repositories[i].Outcome = OutcomePlanned
	}
	if err := record.Validate(); err != nil {
		return ManagedArtifactRecord{}, err
	}
	return record, nil
}

func ManagedArtifactResult(executionID string, plan managedartifact.Plan, prepared []managedartifact.PreparedCommit, pushes []managedartifact.PushResult, executionErr error) (ManagedArtifactRecord, error) {
	ref, err := managedArtifactPlanRef(plan)
	if err != nil {
		return ManagedArtifactRecord{}, err
	}
	if err := validateManagedArtifactEvidence(plan, prepared, pushes, executionErr); err != nil {
		return ManagedArtifactRecord{}, err
	}
	record := ManagedArtifactRecord{
		Version:      ManagedArtifactVersion,
		Kind:         ManagedArtifactKind,
		ExecutionID:  executionID,
		Phase:        PhaseResult,
		Mode:         ModeApply,
		Plan:         ref,
		Repositories: managedArtifactRepositories(plan),
	}

	preparedByUID := make(map[string]managedartifact.PreparedCommit, len(prepared))
	for _, candidate := range prepared {
		preparedByUID[candidate.UID] = candidate
	}
	pushByUID := make(map[string]managedartifact.PushResult, len(pushes))
	for _, pushed := range pushes {
		pushByUID[pushed.UID] = pushed
	}

	if executionErr == nil {
		record.Outcome = OutcomeApplied
	} else if errors.Is(executionErr, managedartifact.ErrStale) {
		record.Outcome = OutcomeStale
		record.FailureStage = "STALE"
	} else if len(prepared) == 0 {
		record.Outcome = OutcomeFailed
		record.FailureStage = "PREPARE"
	} else {
		record.Outcome = OutcomeFailed
		record.FailureStage = "PUSH"
	}

	for i := range record.Repositories {
		repo := &record.Repositories[i]
		candidate, wasPrepared := preparedByUID[repo.UID]
		if wasPrepared {
			repo.PreparedCommit = candidate.CommitOID
		}
		push, wasPushed := pushByUID[repo.UID]
		switch {
		case wasPushed && push.Pushed:
			repo.Pushed = true
			repo.Outcome = OutcomeApplied
		case wasPushed:
			repo.Outcome = OutcomeFailed
		case executionErr == nil:
			return ManagedArtifactRecord{}, fmt.Errorf("managed artifact result is missing push result for repo %q", repo.ID)
		case record.Outcome == OutcomeStale:
			repo.Outcome = OutcomeStale
		case wasPrepared:
			repo.Outcome = OutcomeSkipped
		default:
			repo.Outcome = OutcomeFailed
		}
	}
	if err := record.Validate(); err != nil {
		return ManagedArtifactRecord{}, err
	}
	return record, nil
}

func validateManagedArtifactEvidence(plan managedartifact.Plan, prepared []managedartifact.PreparedCommit, pushes []managedartifact.PushResult, executionErr error) error {
	plannedByUID := make(map[string]managedartifact.RepositoryPlan, len(plan.Repositories))
	for _, planned := range plan.Repositories {
		plannedByUID[planned.UID] = planned
	}
	preparedByUID := make(map[string]managedartifact.PreparedCommit, len(prepared))
	for i, candidate := range prepared {
		planned, ok := plannedByUID[candidate.UID]
		if !ok {
			return fmt.Errorf("prepared commit %d references unknown uid %q", i, candidate.UID)
		}
		if _, duplicate := preparedByUID[candidate.UID]; duplicate {
			return fmt.Errorf("prepared commit %d duplicates uid %q", i, candidate.UID)
		}
		if candidate.ID != planned.ID || candidate.BaseOID != planned.BaseOID || !oidPattern.MatchString(candidate.TreeOID) || !oidPattern.MatchString(candidate.CommitOID) {
			return fmt.Errorf("prepared commit %d does not match reviewed repository %q", i, planned.ID)
		}
		preparedByUID[candidate.UID] = candidate
	}
	pushByUID := make(map[string]managedartifact.PushResult, len(pushes))
	for i, push := range pushes {
		planned, ok := plannedByUID[push.UID]
		if !ok {
			return fmt.Errorf("push result %d references unknown uid %q", i, push.UID)
		}
		if _, duplicate := pushByUID[push.UID]; duplicate {
			return fmt.Errorf("push result %d duplicates uid %q", i, push.UID)
		}
		candidate, wasPrepared := preparedByUID[push.UID]
		if !wasPrepared || push.ID != planned.ID || push.Branch != planned.Target.Branch || push.BaseOID != planned.BaseOID || push.CommitOID != candidate.CommitOID {
			return fmt.Errorf("push result %d does not match prepared reviewed repository %q", i, planned.ID)
		}
		pushByUID[push.UID] = push
	}
	if executionErr == nil {
		if len(prepared) != len(plan.Repositories) || len(pushes) != len(plan.Repositories) {
			return fmt.Errorf("successful managed artifact execution requires one prepared commit and push result per repository")
		}
		for _, push := range pushes {
			if !push.Pushed {
				return fmt.Errorf("successful managed artifact execution contains failed push for uid %q", push.UID)
			}
		}
	}
	return nil
}

func managedArtifactPlanRef(plan managedartifact.Plan) (ManagedArtifactPlanRef, error) {
	encoded, err := plan.Marshal()
	if err != nil {
		return ManagedArtifactPlanRef{}, fmt.Errorf("serialize managed artifact plan: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return ManagedArtifactPlanRef{Version: plan.Version, Kind: plan.Kind, SHA256: hex.EncodeToString(digest[:])}, nil
}

func managedArtifactRepositories(plan managedartifact.Plan) []ManagedArtifactRepository {
	repositories := make([]ManagedArtifactRepository, 0, len(plan.Repositories))
	for _, planned := range plan.Repositories {
		action := planned.Actions[0]
		repositories = append(repositories, ManagedArtifactRepository{
			UID:           planned.UID,
			ID:            planned.ID,
			Provider:      planned.Target.Provider,
			Path:          planned.Target.Path,
			Branch:        planned.Target.Branch,
			BaseOID:       planned.BaseOID,
			DesiredMode:   action.Desired.Mode,
			DesiredSHA256: action.Desired.SHA256,
		})
	}
	return repositories
}

func (r ManagedArtifactRecord) Marshal() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(r, "", "  ")
}

func ParseManagedArtifact(data []byte) (ManagedArtifactRecord, error) {
	var record ManagedArtifactRecord
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return ManagedArtifactRecord{}, fmt.Errorf("decode managed artifact execution record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ManagedArtifactRecord{}, fmt.Errorf("decode managed artifact execution record: trailing JSON value")
		}
		return ManagedArtifactRecord{}, fmt.Errorf("decode managed artifact execution record: trailing data: %w", err)
	}
	if err := record.Validate(); err != nil {
		return ManagedArtifactRecord{}, err
	}
	return record, nil
}

func (r ManagedArtifactRecord) Validate() error {
	if r.Version != ManagedArtifactVersion || r.Kind != ManagedArtifactKind {
		return fmt.Errorf("unsupported managed artifact execution record version or kind")
	}
	if !validIdentifier(r.ExecutionID) || !validPhase(r.Phase) || r.Mode != ModeApply {
		return fmt.Errorf("managed artifact execution record has invalid execution metadata")
	}
	if r.Plan.Version != managedartifact.PlanVersion || r.Plan.Kind != managedartifact.PlanKind || !digestPattern.MatchString(r.Plan.SHA256) {
		return fmt.Errorf("managed artifact plan reference is invalid")
	}
	if len(r.Repositories) == 0 {
		return fmt.Errorf("managed artifact execution record requires at least one repository")
	}
	if r.Phase == PhaseIntent {
		if r.Outcome != OutcomePlanned || r.FailureStage != "" {
			return fmt.Errorf("managed artifact intent requires planned outcome without failure stage")
		}
	} else {
		if r.Outcome != OutcomeApplied && r.Outcome != OutcomeFailed && r.Outcome != OutcomeStale {
			return fmt.Errorf("managed artifact result has unsupported overall outcome %q", r.Outcome)
		}
		if r.Outcome == OutcomeApplied && r.FailureStage != "" {
			return fmt.Errorf("applied managed artifact result must not define failure stage")
		}
		if r.FailureStage != "" && r.FailureStage != "PREPARE" && r.FailureStage != "PUSH" && r.FailureStage != "STALE" {
			return fmt.Errorf("managed artifact result has unsupported failure stage %q", r.FailureStage)
		}
	}
	seen := make(map[string]struct{}, len(r.Repositories))
	appliedCount := 0
	staleCount := 0
	for i, repo := range r.Repositories {
		if !validIdentifier(repo.UID) || !validIdentifier(repo.ID) || !validIdentifier(repo.Provider) {
			return fmt.Errorf("managed artifact repository %d has invalid identity", i)
		}
		if _, exists := seen[repo.UID]; exists {
			return fmt.Errorf("managed artifact repository %d duplicates uid %q", i, repo.UID)
		}
		seen[repo.UID] = struct{}{}
		if err := validateProviderPath(repo.Path); err != nil {
			return fmt.Errorf("managed artifact repository %d path: %w", i, err)
		}
		if err := validateBranch(repo.Branch); err != nil {
			return fmt.Errorf("managed artifact repository %d branch: %w", i, err)
		}
		if !oidPattern.MatchString(repo.BaseOID) || !digestPattern.MatchString(repo.DesiredSHA256) {
			return fmt.Errorf("managed artifact repository %d has invalid base or desired digest", i)
		}
		if repo.DesiredMode != "100644" && repo.DesiredMode != "100755" {
			return fmt.Errorf("managed artifact repository %d has invalid desired mode", i)
		}
		if repo.PreparedCommit != "" && !oidPattern.MatchString(repo.PreparedCommit) {
			return fmt.Errorf("managed artifact repository %d has invalid prepared commit", i)
		}
		if r.Phase == PhaseIntent {
			if repo.Outcome != OutcomePlanned || repo.PreparedCommit != "" || repo.Pushed {
				return fmt.Errorf("managed artifact intent repository %d must remain planned and unprepared", i)
			}
			continue
		}
		if repo.Outcome != OutcomeApplied && repo.Outcome != OutcomeFailed && repo.Outcome != OutcomeSkipped && repo.Outcome != OutcomeStale {
			return fmt.Errorf("managed artifact result repository %d has unsupported outcome %q", i, repo.Outcome)
		}
		if repo.Pushed != (repo.Outcome == OutcomeApplied) {
			return fmt.Errorf("managed artifact result repository %d pushed flag disagrees with outcome", i)
		}
		if repo.Pushed && repo.PreparedCommit == "" {
			return fmt.Errorf("managed artifact result repository %d pushed outcome requires prepared commit", i)
		}
		if repo.Outcome == OutcomeApplied {
			appliedCount++
		}
		if repo.Outcome == OutcomeStale {
			staleCount++
		}
	}
	if r.Phase == PhaseResult {
		if r.Outcome == OutcomeApplied && appliedCount != len(r.Repositories) {
			return fmt.Errorf("applied managed artifact result requires every repository to be applied")
		}
		if r.Outcome == OutcomeStale && staleCount != len(r.Repositories) {
			return fmt.Errorf("stale managed artifact result requires every repository to be stale")
		}
		if r.Outcome == OutcomeFailed && appliedCount == len(r.Repositories) {
			return fmt.Errorf("failed managed artifact result cannot report every repository applied")
		}
	}
	return nil
}
