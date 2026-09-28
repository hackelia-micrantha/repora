package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ManifestKind       = "micrantha.repository-bootstrap"
	ManifestVersion    = 1
	InspectionKind     = "repora.bootstrap-inspection"
	InspectionVersion  = 1
	PlanKind           = "repora.bootstrap-plan"
	PlanVersion        = 1
	ApplyResultKind    = "repora.bootstrap-apply-result"
	ApplyResultVersion = 1

	ContractRepository   = "hackelia-micrantha/.github"
	ContractRevision     = "af0ec6581e7593b4e5cb8a5ada4294cde86115e8"
	ContractSchemaPath   = "metadata/repository-bootstrap.schema.json"
	ContractTemplatePath = "docs/standards/templates/repository-bootstrap.json"
	DefaultManifestPath  = ".repora/bootstrap.json"
	DefaultProposalPath  = ".repora/bootstrap.proposed.json"
)

var ErrStale = errors.New("bootstrap plan is stale")

type Provenance struct {
	Authority string `json:"authority"`
	Subject   string `json:"subject"`
	Reference string `json:"reference"`
	Rationale string `json:"rationale,omitempty"`
}

type Decision struct {
	Key        string      `json:"key"`
	State      string      `json:"state"`
	Value      interface{} `json:"value,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`
	Question   string      `json:"question,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Notes      string      `json:"notes,omitempty"`
}

type Manifest struct {
	Schema        string         `json:"$schema,omitempty"`
	SchemaVersion int            `json:"schemaVersion"`
	Kind          string         `json:"kind"`
	PolicySources []PolicySource `json:"policySources,omitempty"`
	Decisions     []Decision     `json:"decisions"`
}

type PolicySource struct {
	Authority string `json:"authority"`
	Reference string `json:"reference"`
}

type ContractRef struct {
	Repository   string `json:"repository"`
	Revision     string `json:"revision"`
	SchemaPath   string `json:"schema_path"`
	TemplatePath string `json:"template_path"`
}

type Observation struct {
	Key      string `json:"key"`
	State    string `json:"state"`
	Value    bool   `json:"value"`
	Evidence string `json:"evidence"`
	SHA256   string `json:"sha256,omitempty"`
}

type Inspection struct {
	Kind           string        `json:"kind"`
	Version        int           `json:"version"`
	Root           string        `json:"root"`
	SnapshotSHA256 string        `json:"snapshot_sha256"`
	Observations   []Observation `json:"observations"`
}

type AuthorityClaim struct {
	DecisionKey string     `json:"decision_key"`
	Provenance  Provenance `json:"provenance"`
	Verified    bool       `json:"verified"`
	Basis       string     `json:"basis,omitempty"`
}

type Action struct {
	Type      string `json:"type"`
	Target    string `json:"target"`
	SHA256    string `json:"sha256,omitempty"`
	Supported bool   `json:"supported"`
}

type BlockedAction struct {
	Type         string   `json:"type"`
	Target       string   `json:"target,omitempty"`
	Reasons      []string `json:"reasons"`
	DecisionKeys []string `json:"decision_keys,omitempty"`
}

type Plan struct {
	Kind            string           `json:"kind"`
	Version         int              `json:"version"`
	Contract        ContractRef      `json:"contract"`
	Root            string           `json:"root"`
	SnapshotSHA256  string           `json:"snapshot_sha256"`
	ManifestPath    string           `json:"manifest_path"`
	ManifestSHA256  string           `json:"manifest_sha256"`
	CIApplicability string           `json:"ci_applicability"`
	AuthorityClaims []AuthorityClaim `json:"authority_claims"`
	Actions         []Action         `json:"actions"`
	BlockedActions  []BlockedAction  `json:"blocked_actions"`
	UnresolvedKeys  []string         `json:"unresolved_keys"`
}

type ApplyResult struct {
	Kind           string `json:"kind"`
	Version        int    `json:"version"`
	PlanSHA256     string `json:"plan_sha256"`
	Outcome        string `json:"outcome"`
	Target         string `json:"target"`
	ManifestSHA256 string `json:"manifest_sha256"`
}

func Contract() ContractRef {
	return ContractRef{
		Repository:   ContractRepository,
		Revision:     ContractRevision,
		SchemaPath:   ContractSchemaPath,
		TemplatePath: ContractTemplatePath,
	}
}

func PinnedSchemaURL() string {
	return "https://raw.githubusercontent.com/" + ContractRepository + "/" + ContractRevision + "/" + ContractSchemaPath
}

func DefaultManifest() Manifest {
	return Manifest{
		Schema:        PinnedSchemaURL(),
		SchemaVersion: ManifestVersion,
		Kind:          ManifestKind,
		Decisions:     defaultDecisions(),
	}
}

func defaultDecisions() []Decision {
	unresolved := func(key, question string) Decision {
		return Decision{Key: key, State: "unresolved", Question: question}
	}
	return []Decision{
		unresolved("repository.name", "What repository name is explicitly authorized?"),
		unresolved("repository.purpose", "What responsibility does this repository own?"),
		unresolved("project.identity", "Does this repository belong to an existing project identity or establish a new one?"),
		unresolved("repository.visibility", "Should this repository be public, private, or internal?"),
		unresolved("repository.classification", "Which Micrantha repository classification applies?"),
		unresolved("repository.maturity", "Which lifecycle state applies at creation?"),
		{
			Key:   "repository.defaultBranch",
			State: "resolved",
			Value: "main",
			Provenance: &Provenance{
				Authority: "organization-policy",
				Subject:   "micrantha-organization-policy",
				Reference: "Micrantha organization default branch policy",
			},
		},
		unresolved("repository.license", "What license, if any, is authorized for this repository?"),
		unresolved("repository.sourceExposure", "What source exposure is intended?"),
		unresolved("repository.role", "What repository-topology role does this surface have?"),
		unresolved("repository.distributionMode", "What distribution mode, if any, does this repository represent?"),
		unresolved("repository.topology", "What explicitly designed repository topology applies?"),
		unresolved("implementation.language", "What implementation language is intentionally selected?"),
		unresolved("implementation.runtime", "What runtime, if any, is intentionally selected?"),
		unresolved("implementation.buildSystem", "What build system, if any, is intentionally selected?"),
		unresolved("implementation.packageManager", "What package manager, if any, is intentionally selected?"),
		unresolved("interface.cli", "Does this repository intentionally provide a CLI?"),
		unresolved("interface.service", "Does this repository intentionally provide a service/API?"),
		unresolved("interface.library", "Does this repository intentionally provide a reusable library/SDK?"),
		unresolved("interface.website", "Does this repository intentionally provide a website or web application?"),
		unresolved("interface.mobile", "Does this repository intentionally provide a mobile application or mobile SDK?"),
		unresolved("delivery.ciProvider", "What CI implementation, if any, is explicitly selected?"),
		unresolved("delivery.releaseModel", "What release model, if any, applies?"),
		unresolved("delivery.distribution", "How, if at all, will consumers acquire releases or packages?"),
		unresolved("security.profile", "What project-specific security profile or additional controls apply beyond organization standards?"),
	}
}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := strictDecode(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode bootstrap manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) Marshal() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode bootstrap manifest: %w", err)
	}
	return append(data, '\n'), nil
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestVersion || m.Kind != ManifestKind {
		return fmt.Errorf("unsupported bootstrap manifest contract: kind=%q schemaVersion=%d", m.Kind, m.SchemaVersion)
	}
	if len(m.Decisions) == 0 {
		return fmt.Errorf("bootstrap manifest decisions must be non-empty")
	}
	seenPolicySources := make(map[string]struct{}, len(m.PolicySources))
	for i, source := range m.PolicySources {
		switch source.Authority {
		case "organization-policy", "project-policy":
		default:
			return fmt.Errorf("policy source %d has invalid authority %q", i, source.Authority)
		}
		if strings.TrimSpace(source.Reference) == "" {
			return fmt.Errorf("policy source %d requires reference", i)
		}
		identity := source.Authority + "\x00" + source.Reference
		if _, exists := seenPolicySources[identity]; exists {
			return fmt.Errorf("duplicate policy source %q", source.Reference)
		}
		seenPolicySources[identity] = struct{}{}
	}

	seen := make(map[string]struct{}, len(m.Decisions))
	for i, decision := range m.Decisions {
		if strings.TrimSpace(decision.Key) == "" || !strings.Contains(decision.Key, ".") {
			return fmt.Errorf("decision %d requires a dotted key", i)
		}
		if _, ok := seen[decision.Key]; ok {
			return fmt.Errorf("duplicate bootstrap decision key %q", decision.Key)
		}
		seen[decision.Key] = struct{}{}
		if err := validateDecision(decision); err != nil {
			return fmt.Errorf("decision %q: %w", decision.Key, err)
		}
	}
	return nil
}

func validateDecision(d Decision) error {
	switch d.State {
	case "resolved":
		if d.Provenance == nil {
			return fmt.Errorf("resolved decision requires provenance")
		}
		if err := validateProvenance(*d.Provenance); err != nil {
			return err
		}
		if d.Question != "" || d.Reason != "" {
			return fmt.Errorf("resolved decision must not carry unresolved fields")
		}
		return validateResolvedValue(d.Key, d.Value)
	case "unresolved":
		if d.Value != nil || d.Provenance != nil || d.Reason != "" {
			return fmt.Errorf("unresolved decision must not carry value, provenance, or reason")
		}
		return nil
	case "not-applicable":
		if d.Value != nil {
			return fmt.Errorf("not-applicable decision must not carry value")
		}
		if strings.TrimSpace(d.Reason) == "" {
			return fmt.Errorf("not-applicable decision requires reason")
		}
		if d.Provenance == nil {
			return fmt.Errorf("not-applicable decision requires provenance")
		}
		return validateProvenance(*d.Provenance)
	default:
		return fmt.Errorf("invalid state %q", d.State)
	}
}

func validateProvenance(p Provenance) error {
	switch p.Authority {
	case "human", "organization-policy", "project-policy":
	default:
		return fmt.Errorf("invalid provenance authority %q", p.Authority)
	}
	if strings.TrimSpace(p.Subject) == "" || strings.TrimSpace(p.Reference) == "" {
		return fmt.Errorf("provenance requires subject and reference")
	}
	return nil
}

var stringDecisionKeys = map[string]struct{}{
	"repository.name":               {},
	"repository.purpose":            {},
	"project.identity":              {},
	"repository.classification":     {},
	"repository.maturity":           {},
	"repository.license":            {},
	"repository.sourceExposure":     {},
	"repository.role":               {},
	"repository.distributionMode":   {},
	"repository.topology":           {},
	"implementation.language":       {},
	"implementation.runtime":        {},
	"implementation.buildSystem":    {},
	"implementation.packageManager": {},
	"delivery.ciProvider":           {},
	"delivery.releaseModel":         {},
	"delivery.distribution":         {},
	"security.profile":              {},
}

var boolDecisionKeys = map[string]struct{}{
	"interface.cli":     {},
	"interface.service": {},
	"interface.library": {},
	"interface.website": {},
	"interface.mobile":  {},
}

func validateResolvedValue(key string, value interface{}) error {
	if key == "repository.visibility" {
		v, ok := value.(string)
		if !ok || (v != "public" && v != "private" && v != "internal") {
			return fmt.Errorf("repository.visibility requires public, private, or internal")
		}
		return nil
	}
	if key == "repository.defaultBranch" {
		v, ok := value.(string)
		if !ok || strings.TrimSpace(v) == "" || strings.IndexFunc(v, func(r rune) bool { return r == ' ' || r == '\t' || r == '\r' || r == '\n' }) >= 0 {
			return fmt.Errorf("repository.defaultBranch requires a non-empty token")
		}
		return nil
	}
	if _, ok := boolDecisionKeys[key]; ok {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s requires a boolean value", key)
		}
		return nil
	}
	if _, ok := stringDecisionKeys[key]; ok {
		v, ok := value.(string)
		if !ok || strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s requires a non-empty string value", key)
		}
		return nil
	}
	return fmt.Errorf("resolved extension key %q lacks v1 value semantics", key)
}

func VerifyAuthority(m Manifest) []AuthorityClaim {
	claims := make([]AuthorityClaim, 0)
	for _, decision := range m.Decisions {
		if decision.State != "resolved" && decision.State != "not-applicable" {
			continue
		}
		claim := AuthorityClaim{DecisionKey: decision.Key, Provenance: *decision.Provenance}
		if decision.Key == "repository.defaultBranch" && decision.State == "resolved" && decision.Provenance.Authority == "organization-policy" && decision.Provenance.Subject == "micrantha-organization-policy" && decision.Provenance.Reference == "Micrantha organization default branch policy" {
			if value, ok := decision.Value.(string); ok && value == "main" {
				claim.Verified = true
				claim.Basis = ContractRepository + "@" + ContractRevision
			}
		}
		claims = append(claims, claim)
	}
	return claims
}

func CIAplicability(m Manifest) string {
	for _, decision := range m.Decisions {
		if decision.Key != "delivery.ciProvider" {
			continue
		}
		switch decision.State {
		case "resolved":
			return "applicable"
		case "not-applicable":
			return "not-applicable"
		default:
			return "unresolved"
		}
	}
	return "unresolved"
}

func Inspect(root string) (Inspection, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Inspection{}, fmt.Errorf("resolve bootstrap root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return Inspection{}, fmt.Errorf("stat bootstrap root: %w", err)
	}
	if !info.IsDir() {
		return Inspection{}, fmt.Errorf("bootstrap root must be a directory")
	}

	paths := []struct {
		key  string
		path string
	}{
		{"bootstrap_manifest_present", DefaultManifestPath},
		{"bootstrap_proposal_present", DefaultProposalPath},
		{"flake_present", "flake.nix"},
		{"flake_lock_present", "flake.lock"},
		{"license_present", "LICENSE"},
		{"readme_present", "README.md"},
	}
	observations := make([]Observation, 0, len(paths)+1)
	for _, item := range paths {
		obs, err := observeFile(absRoot, item.key, item.path)
		if err != nil {
			return Inspection{}, err
		}
		observations = append(observations, obs)
	}
	workflowObs, err := observeWorkflows(absRoot)
	if err != nil {
		return Inspection{}, err
	}
	observations = append(observations, workflowObs)
	sort.Slice(observations, func(i, j int) bool { return observations[i].Key < observations[j].Key })

	inspection := Inspection{
		Kind:         InspectionKind,
		Version:      InspectionVersion,
		Root:         absRoot,
		Observations: observations,
	}
	inspection.SnapshotSHA256 = inspectionDigest(inspection)
	return inspection, nil
}

func observeFile(root, key, relative string) (Observation, error) {
	full := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Observation{Key: key, State: "observed", Value: false, Evidence: relative}, nil
		}
		return Observation{}, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Observation{}, fmt.Errorf("inspect %s: symlink is not accepted at bootstrap boundary", relative)
	}
	if !info.Mode().IsRegular() {
		return Observation{}, fmt.Errorf("inspect %s: expected regular file", relative)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return Observation{}, fmt.Errorf("inspect %s: %w", relative, err)
	}
	return Observation{Key: key, State: "observed", Value: true, Evidence: relative, SHA256: digest(data)}, nil
}

func observeWorkflows(root string) (Observation, error) {
	dir := filepath.Join(root, ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Observation{Key: "workflows_present", State: "observed", Value: false, Evidence: ".github/workflows"}, nil
		}
		return Observation{}, fmt.Errorf("inspect workflows: %w", err)
	}
	names := make([]string, 0)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return Observation{}, fmt.Errorf("inspect workflows: symlink %q is not accepted", entry.Name())
		}
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return Observation{}, fmt.Errorf("inspect workflow %s: %w", name, err)
		}
		fmt.Fprintf(h, "%s\x00%s\x00", name, digest(data))
	}
	obs := Observation{Key: "workflows_present", State: "observed", Value: len(names) > 0, Evidence: ".github/workflows"}
	if len(names) > 0 {
		obs.SHA256 = hex.EncodeToString(h.Sum(nil))
	}
	return obs, nil
}

func inspectionDigest(inspection Inspection) string {
	h := sha256.New()
	for _, obs := range inspection.Observations {
		fmt.Fprintf(h, "%s\x00%t\x00%s\x00%s\x00", obs.Key, obs.Value, obs.Evidence, obs.SHA256)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func BuildPlan(root, manifestPath string) (Plan, error) {
	inspection, err := Inspect(root)
	if err != nil {
		return Plan{}, err
	}
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return Plan{}, fmt.Errorf("inspect bootstrap manifest input: %w", err)
	}
	if manifestInfo.Mode()&os.ModeSymlink != 0 || !manifestInfo.Mode().IsRegular() {
		return Plan{}, fmt.Errorf("bootstrap manifest input must be a regular non-symlink file")
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return Plan{}, fmt.Errorf("read bootstrap manifest: %w", err)
	}
	manifest, err := ParseManifest(manifestData)
	if err != nil {
		return Plan{}, err
	}
	claims := VerifyAuthority(manifest)
	unresolved := make([]string, 0)
	for _, decision := range manifest.Decisions {
		if decision.State == "unresolved" {
			unresolved = append(unresolved, decision.Key)
		}
	}
	sort.Strings(unresolved)

	plan := Plan{
		Kind:            PlanKind,
		Version:         PlanVersion,
		Contract:        Contract(),
		Root:            inspection.Root,
		SnapshotSHA256:  inspection.SnapshotSHA256,
		ManifestPath:    manifestPath,
		ManifestSHA256:  digest(manifestData),
		CIApplicability: CIAplicability(manifest),
		AuthorityClaims: claims,
		Actions:         []Action{},
		BlockedActions:  []BlockedAction{},
		UnresolvedKeys:  unresolved,
	}

	target := filepath.Join(inspection.Root, filepath.FromSlash(DefaultManifestPath))
	present := observationValue(inspection, "bootstrap_manifest_present")
	if present {
		targetData, err := os.ReadFile(target)
		if err != nil {
			return Plan{}, fmt.Errorf("read existing bootstrap manifest: %w", err)
		}
		if digest(targetData) != plan.ManifestSHA256 {
			plan.BlockedActions = append(plan.BlockedActions, BlockedAction{
				Type:    "WRITE_BOOTSTRAP_MANIFEST",
				Target:  DefaultManifestPath,
				Reasons: []string{"target already exists with different content; bootstrap apply is create-only"},
			})
		}
	} else {
		// Persisting the manifest records claims as data; it does not exercise those
		// decisions. Dependent effects remain blocked until authority is verified.
		plan.Actions = append(plan.Actions, Action{Type: "WRITE_BOOTSTRAP_MANIFEST", Target: DefaultManifestPath, SHA256: plan.ManifestSHA256, Supported: true})
	}

	plan.BlockedActions = append(plan.BlockedActions, decisionBlockers(manifest, claims)...)
	return plan, plan.Validate()
}

func decisionBlockers(m Manifest, claims []AuthorityClaim) []BlockedAction {
	byKey := make(map[string]Decision, len(m.Decisions))
	for _, d := range m.Decisions {
		byKey[d.Key] = d
	}
	verified := make(map[string]bool, len(claims))
	for _, claim := range claims {
		verified[claim.DecisionKey] = claim.Verified
	}
	block := func(action string, keys ...string) BlockedAction {
		missing := []string{}
		unverified := []string{}
		for _, key := range keys {
			d, ok := byKey[key]
			if !ok || d.State == "unresolved" {
				missing = append(missing, key)
				continue
			}
			if (d.State == "resolved" || d.State == "not-applicable") && !verified[key] {
				unverified = append(unverified, key)
			}
		}
		reasons := []string{"capability is intentionally not executable in bootstrap v1"}
		decisionKeys := append([]string{}, missing...)
		decisionKeys = append(decisionKeys, unverified...)
		if len(missing) > 0 {
			reasons = append(reasons, "required decisions remain unresolved")
		}
		if len(unverified) > 0 {
			reasons = append(reasons, "decision provenance is recorded but not independently verified")
		}
		sort.Strings(decisionKeys)
		return BlockedAction{Type: action, Reasons: reasons, DecisionKeys: decisionKeys}
	}
	out := []BlockedAction{
		block("PROVIDER_CREATE", "repository.name", "repository.visibility", "repository.purpose"),
		block("WRITE_LICENSE", "repository.license"),
		block("SCAFFOLD_IMPLEMENTATION", "implementation.language", "implementation.buildSystem"),
		block("CONFIGURE_RELEASE", "delivery.releaseModel", "delivery.distribution"),
	}
	ci := block("CONFIGURE_FLAKE_FIRST_CI", "delivery.ciProvider")
	if d, ok := byKey["delivery.ciProvider"]; !ok || d.State == "unresolved" {
		ci.Reasons = append(ci.Reasons, "CI applicability remains unresolved; observed workflows or flakes do not resolve it")
	} else if d.State == "not-applicable" {
		ci.Reasons = append(ci.Reasons, "CI is explicitly declared not applicable; no CI mutation is planned")
	} else {
		ci.Reasons = append(ci.Reasons, "CI is explicitly declared applicable; flake-first policy applies once authority is verified")
	}
	out = append(out, ci)
	return out
}

func observationValue(inspection Inspection, key string) bool {
	for _, observation := range inspection.Observations {
		if observation.Key == key {
			return observation.Value
		}
	}
	return false
}

func (p Plan) Validate() error {
	if p.Kind != PlanKind || p.Version != PlanVersion {
		return fmt.Errorf("unsupported bootstrap plan contract: kind=%q version=%d", p.Kind, p.Version)
	}
	if p.Contract != Contract() {
		return fmt.Errorf("bootstrap plan contract reference does not match pinned contract")
	}
	if strings.TrimSpace(p.Root) == "" || strings.TrimSpace(p.ManifestPath) == "" {
		return fmt.Errorf("bootstrap plan root and manifest_path are required")
	}
	if !validDigest(p.SnapshotSHA256) || !validDigest(p.ManifestSHA256) {
		return fmt.Errorf("bootstrap plan requires lowercase SHA-256 identities")
	}
	switch p.CIApplicability {
	case "applicable", "not-applicable", "unresolved":
	default:
		return fmt.Errorf("invalid ci_applicability %q", p.CIApplicability)
	}
	if p.Actions == nil || p.BlockedActions == nil || p.AuthorityClaims == nil || p.UnresolvedKeys == nil {
		return fmt.Errorf("bootstrap plan arrays are required")
	}
	for _, action := range p.Actions {
		if action.Type != "WRITE_BOOTSTRAP_MANIFEST" || action.Target != DefaultManifestPath || !action.Supported || !validDigest(action.SHA256) {
			return fmt.Errorf("bootstrap plan contains unsupported executable action")
		}
	}
	return nil
}

func (p Plan) Marshal() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode bootstrap plan: %w", err)
	}
	return append(data, '\n'), nil
}

func ParsePlan(data []byte) (Plan, error) {
	var plan Plan
	if err := strictDecode(data, &plan); err != nil {
		return Plan{}, fmt.Errorf("decode bootstrap plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func Apply(plan Plan) (ApplyResult, error) {
	planData, err := plan.Marshal()
	if err != nil {
		return ApplyResult{}, err
	}
	result := ApplyResult{
		Kind:           ApplyResultKind,
		Version:        ApplyResultVersion,
		PlanSHA256:     digest(planData),
		Outcome:        "blocked",
		Target:         DefaultManifestPath,
		ManifestSHA256: plan.ManifestSHA256,
	}
	inspection, err := Inspect(plan.Root)
	if err != nil {
		return result, err
	}
	if inspection.SnapshotSHA256 != plan.SnapshotSHA256 {
		return result, fmt.Errorf("%w: observed repository state changed", ErrStale)
	}
	manifestInfo, err := os.Lstat(plan.ManifestPath)
	if err != nil {
		return result, fmt.Errorf("inspect bootstrap manifest for apply: %w", err)
	}
	if manifestInfo.Mode()&os.ModeSymlink != 0 || !manifestInfo.Mode().IsRegular() {
		return result, fmt.Errorf("%w: bootstrap manifest input is no longer a regular non-symlink file", ErrStale)
	}
	manifestData, err := os.ReadFile(plan.ManifestPath)
	if err != nil {
		return result, fmt.Errorf("read bootstrap manifest for apply: %w", err)
	}
	if digest(manifestData) != plan.ManifestSHA256 {
		return result, fmt.Errorf("%w: manifest input changed", ErrStale)
	}
	manifest, err := ParseManifest(manifestData)
	if err != nil {
		return result, err
	}
	claims := VerifyAuthority(manifest)
	if !sameClaims(plan.AuthorityClaims, claims) {
		return result, fmt.Errorf("%w: authority evidence changed or cannot be revalidated", ErrStale)
	}
	if len(plan.Actions) == 0 {
		result.Outcome = "no-op"
		return result, nil
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != "WRITE_BOOTSTRAP_MANIFEST" {
		return result, fmt.Errorf("bootstrap apply only supports one create-only manifest action")
	}
	target := filepath.Join(plan.Root, filepath.FromSlash(DefaultManifestPath))
	parentInfo, err := os.Stat(filepath.Dir(target))
	if err != nil {
		return result, fmt.Errorf("%w: bootstrap manifest parent changed or is unavailable", ErrStale)
	}
	if !parentInfo.IsDir() {
		return result, fmt.Errorf("%w: bootstrap manifest parent is not a directory", ErrStale)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return result, fmt.Errorf("%w: bootstrap manifest target now exists", ErrStale)
		}
		return result, fmt.Errorf("create bootstrap manifest: %w", err)
	}
	writeErr := func() error {
		if _, err := file.Write(manifestData); err != nil {
			return err
		}
		return file.Sync()
	}()
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(target)
		return result, fmt.Errorf("write bootstrap manifest: %w", writeErr)
	}
	if closeErr != nil {
		return result, fmt.Errorf("close bootstrap manifest: %w", closeErr)
	}
	result.Outcome = "created"
	return result, nil
}

func sameClaims(a, b []AuthorityClaim) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Init(path string) error {
	manifest := DefaultManifest()
	data, err := manifest.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create bootstrap manifest parent: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("bootstrap manifest already exists: %s", path)
		}
		return fmt.Errorf("create bootstrap manifest: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write bootstrap manifest: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close bootstrap manifest: %w", err)
	}
	return nil
}

func strictDecode(data []byte, out interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return fmt.Errorf("trailing data: %w", err)
	}
	return nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

[executed on device: 76a4bdf5fc1b (a7fd9f41-8002-4c03-ac43-498109dd9775)]