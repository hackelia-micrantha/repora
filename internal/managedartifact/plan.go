package managedartifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	PlanKind          = "repora.io/managed-artifact-plan"
	PlanVersion       = 1
	ActionWriteREADME = "WRITE_README"
	READMEPath        = "README.md"
	MaxDiffBytes      = 2*MaxTextBytes + 64*1024
)

var (
	planIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	planOIDPattern        = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)
	sha256Pattern         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Plan struct {
	Kind         string           `json:"kind"`
	Version      int              `json:"version"`
	Repositories []RepositoryPlan `json:"repositories"`
}

type RepositoryPlan struct {
	UID     string   `json:"uid"`
	ID      string   `json:"id"`
	Target  Target   `json:"target"`
	BaseOID string   `json:"base_oid"`
	Actions []Action `json:"actions"`
}

type Target struct {
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Branch   string `json:"branch"`
}

type Action struct {
	Type           string        `json:"type"`
	Path           string        `json:"path"`
	Observed       ObservedState `json:"observed"`
	Desired        DesiredState  `json:"desired"`
	TemplateSHA256 string        `json:"template_sha256"`
	Diff           string        `json:"diff"`
}

type ObservedState struct {
	Present *bool  `json:"present"`
	Mode    string `json:"mode,omitempty"`
	SHA256  string `json:"sha256,omitempty"`
}

type DesiredState struct {
	Mode    string  `json:"mode"`
	SHA256  string  `json:"sha256"`
	Content *string `json:"content"`
}

func ParsePlan(data []byte) (Plan, error) {
	if err := rejectPlanNulls(data); err != nil {
		return Plan{}, err
	}

	var plan Plan
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("decode managed artifact plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Plan{}, fmt.Errorf("decode managed artifact plan: trailing JSON value")
		}
		return Plan{}, fmt.Errorf("decode managed artifact plan: trailing data: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func (p Plan) Marshal() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(p, "", "  ")
}

func (p Plan) Validate() error {
	if p.Kind != PlanKind || p.Version != PlanVersion {
		return fmt.Errorf("unsupported managed artifact plan contract: kind=%q version=%d", p.Kind, p.Version)
	}
	if p.Repositories == nil {
		return fmt.Errorf("managed artifact plan repositories array is required")
	}

	seenUIDs := make(map[string]struct{}, len(p.Repositories))
	seenIDs := make(map[string]struct{}, len(p.Repositories))
	seenTargets := make(map[string]struct{}, len(p.Repositories))
	for i, repo := range p.Repositories {
		if !validPlanIdentifier(repo.UID) || !validPlanIdentifier(repo.ID) {
			return fmt.Errorf("repository %d requires valid uid and id", i)
		}
		if _, exists := seenUIDs[repo.UID]; exists {
			return fmt.Errorf("duplicate managed artifact repository uid %q", repo.UID)
		}
		seenUIDs[repo.UID] = struct{}{}
		if _, exists := seenIDs[repo.ID]; exists {
			return fmt.Errorf("duplicate managed artifact repository id %q", repo.ID)
		}
		seenIDs[repo.ID] = struct{}{}
		if err := validateTarget(repo.Target); err != nil {
			return fmt.Errorf("repository %d target: %w", i, err)
		}
		targetKey := repo.Target.Provider + ":" + repo.Target.Path + "#" + repo.Target.Branch
		if _, exists := seenTargets[targetKey]; exists {
			return fmt.Errorf("duplicate managed artifact target %q", targetKey)
		}
		seenTargets[targetKey] = struct{}{}
		if !planOIDPattern.MatchString(repo.BaseOID) {
			return fmt.Errorf("repository %d base_oid must be a 40- or 64-character hexadecimal object ID", i)
		}
		if len(repo.Actions) != 1 {
			return fmt.Errorf("repository %d must contain exactly one README action in managed artifact plan v1", i)
		}
		if err := validateAction(repo.Actions[0]); err != nil {
			return fmt.Errorf("repository %d action 0: %w", i, err)
		}
	}
	return nil
}

func validateAction(action Action) error {
	if action.Type != ActionWriteREADME {
		return fmt.Errorf("unsupported action type %q", action.Type)
	}
	if action.Path != READMEPath {
		return fmt.Errorf("managed artifact path must be %q", READMEPath)
	}
	if err := validateObservedState(action.Observed); err != nil {
		return err
	}
	if err := validateDesiredState(action.Observed, action.Desired); err != nil {
		return err
	}
	if !sha256Pattern.MatchString(action.TemplateSHA256) {
		return fmt.Errorf("template_sha256 must be a lowercase SHA-256 digest")
	}
	if err := validateReviewDiff(action.Diff); err != nil {
		return err
	}
	if action.Observed.Present != nil && *action.Observed.Present && action.Observed.Mode == action.Desired.Mode && action.Observed.SHA256 == action.Desired.SHA256 {
		return fmt.Errorf("managed artifact plan must not contain a no-op README action")
	}
	return nil
}

func validateObservedState(state ObservedState) error {
	if state.Present == nil {
		return fmt.Errorf("observed present field is required")
	}
	if *state.Present {
		if !validGitMode(state.Mode) {
			return fmt.Errorf("observed README mode must be 100644 or 100755 when present")
		}
		if !sha256Pattern.MatchString(state.SHA256) {
			return fmt.Errorf("observed README sha256 must be a lowercase SHA-256 digest when present")
		}
		return nil
	}
	if state.Mode != "" || state.SHA256 != "" {
		return fmt.Errorf("absent observed README must not define mode or sha256")
	}
	return nil
}

func validateDesiredState(observed ObservedState, desired DesiredState) error {
	if !validGitMode(desired.Mode) {
		return fmt.Errorf("desired README mode must be 100644 or 100755")
	}
	if desired.Content == nil {
		return fmt.Errorf("desired README content field is required")
	}
	content := *desired.Content
	if len(content) > MaxTextBytes {
		return fmt.Errorf("desired README content exceeds %d-byte limit", MaxTextBytes)
	}
	if err := validateNormalizedText("desired README content", content); err != nil {
		return err
	}
	if !sha256Pattern.MatchString(desired.SHA256) {
		return fmt.Errorf("desired README sha256 must be a lowercase SHA-256 digest")
	}
	if got := DigestSHA256([]byte(content)); got != desired.SHA256 {
		return fmt.Errorf("desired README sha256 does not match content")
	}
	if observed.Present != nil && *observed.Present {
		if desired.Mode != observed.Mode {
			return fmt.Errorf("desired README mode must preserve observed regular-file mode")
		}
	} else if desired.Mode != "100644" {
		return fmt.Errorf("new README must use mode 100644")
	}
	return nil
}

func validateTarget(target Target) error {
	if !validPlanIdentifier(target.Provider) {
		return fmt.Errorf("provider must be a symbolic identifier")
	}
	if err := validatePlanProviderPath(target.Path); err != nil {
		return err
	}
	return validatePlanBranch(target.Branch)
}

func validatePlanProviderPath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") {
		return fmt.Errorf("provider path contains an unsafe segment")
	}
	parts := strings.Split(value, "/")
	if len(parts) < 2 {
		return fmt.Errorf("provider path must include an owner or namespace")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\:@?#`) || strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("provider path contains an unsafe segment")
		}
	}
	return nil
}

func validatePlanBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") || strings.Contains(branch, "..") || strings.Contains(branch, "//") || strings.Contains(branch, "@{") {
		return fmt.Errorf("branch is not a valid symbolic ref name")
	}
	if strings.ContainsAny(branch, " ~^:?*[\\") {
		return fmt.Errorf("branch is not a valid symbolic ref name")
	}
	for _, segment := range strings.Split(branch, "/") {
		if segment == "" || strings.HasPrefix(segment, ".") || strings.HasSuffix(segment, ".lock") {
			return fmt.Errorf("branch is not a valid symbolic ref name")
		}
	}
	return nil
}

func validateReviewDiff(diff string) error {
	if diff == "" {
		return fmt.Errorf("README review diff is required")
	}
	if len(diff) > MaxDiffBytes {
		return fmt.Errorf("README review diff exceeds %d-byte limit", MaxDiffBytes)
	}
	if !utf8.ValidString(diff) {
		return fmt.Errorf("README review diff must be valid UTF-8")
	}
	for _, r := range diff {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("README review diff contains unsupported control character U+%04X", r)
		}
	}
	const prefix = "--- a/README.md\n+++ b/README.md\n@@ "
	if !strings.HasPrefix(diff, prefix) {
		return fmt.Errorf("README review diff must use fixed README.md labels and a unified hunk")
	}
	if !strings.HasSuffix(diff, "\n") {
		return fmt.Errorf("README review diff must end with LF")
	}
	return nil
}

func validateNormalizedText(label, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", label)
	}
	for _, r := range value {
		if r == '\r' {
			return fmt.Errorf("%s must use LF line endings", label)
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("%s contains unsupported control character U+%04X", label, r)
		}
	}
	return nil
}

func validGitMode(mode string) bool {
	return mode == "100644" || mode == "100755"
}

func validPlanIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && planIdentifierPattern.MatchString(value)
}

func DigestSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func rejectPlanNulls(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode managed artifact plan: %w", err)
	}
	return walkPlanNulls(value, "$")
}

func walkPlanNulls(value any, path string) error {
	switch current := value.(type) {
	case nil:
		return fmt.Errorf("managed artifact plan field %s must not be null", path)
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := walkPlanNulls(current[key], path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range current {
			if err := walkPlanNulls(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}
