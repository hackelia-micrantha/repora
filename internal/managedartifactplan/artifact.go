// Package managedartifactplan defines the durable v1 plan contract for managed README changes.
package managedartifactplan

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
	"unicode/utf8"

	"repoctl/internal/managedartifact"
)

const (
	Kind           = "repora.io/managed-artifact-plan"
	Version        = 1
	ActionWriteREADME = "WRITE_README"
	READMEPath     = "README.md"
	ModeRegular    = "100644"
	ModeExecutable = "100755"
	MaxDiffBytes   = 4 << 20
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	oidPattern        = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)
	digestPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type Artifact struct {
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
	Present bool   `json:"present"`
	Mode    string `json:"mode,omitempty"`
	SHA256  string `json:"sha256,omitempty"`
}

type DesiredState struct {
	Mode    string `json:"mode"`
	SHA256  string `json:"sha256"`
	Content string `json:"content"`
}

func Parse(data []byte) (Artifact, error) {
	if err := rejectNullValues(data); err != nil {
		return Artifact{}, err
	}

	var artifact Artifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return Artifact{}, fmt.Errorf("decode managed artifact plan: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Artifact{}, err
	}
	if err := artifact.Validate(); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func (a Artifact) Marshal() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(a, "", "  ")
}

func (a Artifact) Validate() error {
	if a.Kind != Kind || a.Version != Version {
		return fmt.Errorf("unsupported managed artifact plan contract: kind=%q version=%d", a.Kind, a.Version)
	}
	if a.Repositories == nil {
		return fmt.Errorf("managed artifact plan repositories array is required")
	}

	seenUIDs := make(map[string]struct{}, len(a.Repositories))
	seenIDs := make(map[string]struct{}, len(a.Repositories))
	seenTargets := make(map[string]struct{}, len(a.Repositories))
	for i, repo := range a.Repositories {
		if err := repo.validate(); err != nil {
			return fmt.Errorf("repository[%d]: %w", i, err)
		}
		if _, exists := seenUIDs[repo.UID]; exists {
			return fmt.Errorf("repository[%d]: duplicate uid %q", i, repo.UID)
		}
		seenUIDs[repo.UID] = struct{}{}
		if _, exists := seenIDs[repo.ID]; exists {
			return fmt.Errorf("repository[%d]: duplicate id %q", i, repo.ID)
		}
		seenIDs[repo.ID] = struct{}{}
		targetKey := repo.Target.Provider + ":" + repo.Target.Path
		if _, exists := seenTargets[targetKey]; exists {
			return fmt.Errorf("repository[%d]: duplicate target %q", i, targetKey)
		}
		seenTargets[targetKey] = struct{}{}
	}
	return nil
}

func (r RepositoryPlan) validate() error {
	if !validIdentifier(r.UID) || !validIdentifier(r.ID) {
		return fmt.Errorf("valid uid and id are required")
	}
	if err := r.Target.validate(); err != nil {
		return err
	}
	if !oidPattern.MatchString(r.BaseOID) {
		return fmt.Errorf("base_oid must be a 40- or 64-character hexadecimal object id")
	}
	if len(r.Actions) != 1 {
		return fmt.Errorf("exactly one README action is required, got %d", len(r.Actions))
	}
	if err := r.Actions[0].validate(); err != nil {
		return fmt.Errorf("action: %w", err)
	}
	return nil
}

func (t Target) validate() error {
	if !validIdentifier(t.Provider) {
		return fmt.Errorf("target provider %q is invalid", t.Provider)
	}
	if err := validateProviderPath(t.Path); err != nil {
		return fmt.Errorf("target path: %w", err)
	}
	if err := validateBranch(t.Branch); err != nil {
		return fmt.Errorf("target branch: %w", err)
	}
	return nil
}

func (a Action) validate() error {
	if a.Type != ActionWriteREADME {
		return fmt.Errorf("unsupported action type %q", a.Type)
	}
	if a.Path != READMEPath {
		return fmt.Errorf("managed README path must be %q", READMEPath)
	}
	if err := a.Observed.validate(); err != nil {
		return fmt.Errorf("observed: %w", err)
	}
	if err := a.Desired.validate(); err != nil {
		return fmt.Errorf("desired: %w", err)
	}
	if a.Observed.Present {
		if a.Desired.Mode != a.Observed.Mode {
			return fmt.Errorf("desired mode %q must preserve observed mode %q", a.Desired.Mode, a.Observed.Mode)
		}
		if a.Desired.SHA256 == a.Observed.SHA256 {
			return fmt.Errorf("README action must change observed content")
		}
	} else if a.Desired.Mode != ModeRegular {
		return fmt.Errorf("new README must use mode %s", ModeRegular)
	}
	if !digestPattern.MatchString(a.TemplateSHA256) {
		return fmt.Errorf("template_sha256 must be a lowercase SHA-256 digest")
	}
	if err := validateDiff(a.Diff); err != nil {
		return err
	}
	return nil
}

func (s ObservedState) validate() error {
	if !s.Present {
		if s.Mode != "" || s.SHA256 != "" {
			return fmt.Errorf("absent README must not include mode or sha256")
		}
		return nil
	}
	if !validMode(s.Mode) {
		return fmt.Errorf("unsupported README mode %q", s.Mode)
	}
	if !digestPattern.MatchString(s.SHA256) {
		return fmt.Errorf("sha256 must be a lowercase SHA-256 digest")
	}
	return nil
}

func (s DesiredState) validate() error {
	if !validMode(s.Mode) {
		return fmt.Errorf("unsupported README mode %q", s.Mode)
	}
	if !digestPattern.MatchString(s.SHA256) {
		return fmt.Errorf("sha256 must be a lowercase SHA-256 digest")
	}
	if !utf8.ValidString(s.Content) {
		return fmt.Errorf("content must be valid UTF-8")
	}
	if strings.ContainsRune(s.Content, '\x00') {
		return fmt.Errorf("content must not contain NUL")
	}
	if strings.ContainsRune(s.Content, '\r') {
		return fmt.Errorf("content must use normalized LF line endings")
	}
	if len(s.Content) > managedartifact.MaxTextBytes {
		return fmt.Errorf("content exceeds %d-byte README limit", managedartifact.MaxTextBytes)
	}
	if got := SHA256([]byte(s.Content)); got != s.SHA256 {
		return fmt.Errorf("sha256 does not match desired content")
	}
	return nil
}

func validateDiff(diff string) error {
	if diff == "" {
		return fmt.Errorf("diff is required")
	}
	if !utf8.ValidString(diff) {
		return fmt.Errorf("diff must be valid UTF-8")
	}
	if len(diff) > MaxDiffBytes {
		return fmt.Errorf("diff exceeds %d-byte limit", MaxDiffBytes)
	}
	if strings.ContainsRune(diff, '\x00') || strings.ContainsRune(diff, '\r') || strings.ContainsRune(diff, '\x1b') {
		return fmt.Errorf("diff contains unsupported control data")
	}
	const header = "--- a/README.md\n+++ b/README.md\n"
	if !strings.HasPrefix(diff, header) {
		return fmt.Errorf("diff must use fixed README path labels")
	}
	return nil
}

func validMode(mode string) bool {
	return mode == ModeRegular || mode == ModeExecutable
}

func validIdentifier(value string) bool {
	return value != "" && value == strings.TrimSpace(value) && identifierPattern.MatchString(value)
}

func validateProviderPath(path string) error {
	if path == "" || path != strings.TrimSpace(path) || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("provider path contains an unsafe segment")
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("provider path must include an owner or namespace")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `\\:@?#`) || strings.ContainsAny(part, " \t\r\n") {
			return fmt.Errorf("provider path contains an unsafe segment")
		}
	}
	return nil
}

func validateBranch(branch string) error {
	if branch == "" || branch != strings.TrimSpace(branch) || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") || strings.Contains(branch, "..") || strings.Contains(branch, "//") || strings.Contains(branch, "@{") {
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

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func rejectNullValues(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode managed artifact plan: %w", err)
	}
	return walkNulls(value, "$")
}

func walkNulls(value any, path string) error {
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
			if err := walkNulls(current[key], path+"."+key); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range current {
			if err := walkNulls(child, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode managed artifact plan trailing data: %w", err)
	}
	return fmt.Errorf("decode managed artifact plan: multiple JSON values are not allowed")
}
