package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"repoctl/internal/config"
	"repoctl/internal/posture"
)

func TestDefaultManifestIsAssumptionFreeAndPinned(t *testing.T) {
	manifest := DefaultManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if got := CIAplicability(manifest); got != "unresolved" {
		t.Fatalf("CIApplicability() = %q, want unresolved", got)
	}
	claims := VerifyAuthority(manifest)
	if len(claims) != 1 || claims[0].DecisionKey != "repository.defaultBranch" || !claims[0].Verified {
		t.Fatalf("VerifyAuthority() = %#v, want verified default branch only", claims)
	}
	if manifest.Schema != PinnedSchemaURL() {
		t.Fatalf("schema = %q, want pinned authoritative schema", manifest.Schema)
	}
}

func TestManifestRejectsDuplicateDecisionKeys(t *testing.T) {
	manifest := DefaultManifest()
	manifest.Decisions = append(manifest.Decisions, Decision{Key: "repository.visibility", State: "unresolved"})
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate() succeeded for duplicate decision keys")
	}
}

func TestManifestRejectsInvalidResolvedValue(t *testing.T) {
	manifest := DefaultManifest()
	for i := range manifest.Decisions {
		if manifest.Decisions[i].Key == "repository.visibility" {
			manifest.Decisions[i] = Decision{
				Key:   "repository.visibility",
				State: "resolved",
				Value: map[string]interface{}{"private": true},
				Provenance: &Provenance{
					Authority: "human",
					Subject:   "owner",
					Reference: "test",
				},
			}
		}
	}
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate() succeeded for invalid repository.visibility value")
	}
}

func TestInitIsCreateOnly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, DefaultProposalPath)
	if err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := Init(root); err == nil {
		t.Fatal("second Init() succeeded, want create-only failure")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseManifest(data); err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
}

func TestInitRejectsSymlinkParentEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".repora")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := Init(root); err == nil {
		t.Fatal("Init() succeeded through symlinked .repora parent")
	}
	if _, err := os.Stat(filepath.Join(outside, "bootstrap.proposed.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside proposal exists or stat failed unexpectedly: %v", err)
	}
}

func TestApplyRejectsSymlinkParentSwap(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	proposal := filepath.Join(root, DefaultProposalPath)
	plan, err := BuildPlan(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	moved := filepath.Join(outside, "repora")
	if err := os.Rename(filepath.Join(root, ".repora"), moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, filepath.Join(root, ".repora")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err = Apply(plan)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("Apply() error = %v, want ErrStale", err)
	}
	if _, err := os.Stat(filepath.Join(moved, "bootstrap.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside bootstrap manifest exists or stat failed unexpectedly: %v", err)
	}
}

func TestPlanDoesNotInferCIApplicabilityFromObservedWorkflowOrFlake(t *testing.T) {
	root := t.TempDir()
	proposal := filepath.Join(root, DefaultProposalPath)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "flake.nix"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("name: ci\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, proposal)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.CIApplicability != "unresolved" {
		t.Fatalf("CIApplicability = %q, want unresolved", plan.CIApplicability)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != "WRITE_BOOTSTRAP_MANIFEST" {
		t.Fatalf("Actions = %#v, want create-only manifest action", plan.Actions)
	}
}

func TestPlanBindsAbsoluteManifestPath(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	plan, err := BuildPlan(".", DefaultProposalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(plan.Root) || !filepath.IsAbs(plan.ManifestPath) {
		t.Fatalf("plan paths must be absolute: root=%q manifest=%q", plan.Root, plan.ManifestPath)
	}
}

func TestPlanRejectsActionDigestMismatch(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, filepath.Join(root, DefaultProposalPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("Actions = %#v, want one action", plan.Actions)
	}
	plan.Actions[0].SHA256 = strings.Repeat("0", 64)
	if err := plan.Validate(); err == nil {
		t.Fatal("Validate() succeeded for mismatched action digest")
	}
}

func TestApplyCreatesOnlyAuthoritativeManifest(t *testing.T) {
	root := t.TempDir()
	proposal := filepath.Join(root, DefaultProposalPath)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(plan)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.Outcome != "created" {
		t.Fatalf("Outcome = %q, want created", result.Outcome)
	}
	data, err := os.ReadFile(filepath.Join(root, DefaultManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	if digest(data) != plan.ManifestSHA256 {
		t.Fatal("written manifest digest does not match reviewed plan")
	}
}

func TestApplyFailsStaleWhenObservedStateChanges(t *testing.T) {
	root := t.TempDir()
	proposal := filepath.Join(root, DefaultProposalPath)
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Apply(plan)
	if !errors.Is(err, ErrStale) {
		t.Fatalf("Apply() error = %v, want ErrStale", err)
	}
}

func TestApplyRejectsTamperedManifestDerivedPlanFields(t *testing.T) {
	root := t.TempDir()
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, filepath.Join(root, DefaultProposalPath))
	if err != nil {
		t.Fatal(err)
	}
	plan.CIApplicability = "applicable"
	if _, err := Apply(plan); !errors.Is(err, ErrStale) {
		t.Fatalf("Apply() error = %v, want ErrStale for tampered CI applicability", err)
	}

	plan, err = BuildPlan(root, filepath.Join(root, DefaultProposalPath))
	if err != nil {
		t.Fatal(err)
	}
	plan.UnresolvedKeys = []string{}
	if _, err := Apply(plan); !errors.Is(err, ErrStale) {
		t.Fatalf("Apply() error = %v, want ErrStale for tampered unresolved keys", err)
	}

	plan, err = BuildPlan(root, filepath.Join(root, DefaultProposalPath))
	if err != nil {
		t.Fatal(err)
	}
	plan.BlockedActions = []BlockedAction{}
	if _, err := Apply(plan); !errors.Is(err, ErrStale) {
		t.Fatalf("Apply() error = %v, want ErrStale for tampered blocked actions", err)
	}
}

func TestResolvedHumanClaimIsRecordedButDoesNotAuthorizeFutureEffects(t *testing.T) {
	root := t.TempDir()
	manifest := DefaultManifest()
	for i := range manifest.Decisions {
		if manifest.Decisions[i].Key == "repository.visibility" {
			manifest.Decisions[i] = Decision{
				Key:   "repository.visibility",
				State: "resolved",
				Value: "private",
				Provenance: &Provenance{
					Authority: "human",
					Subject:   "repository-owner",
					Reference: "decision-1",
				},
			}
		}
	}
	data, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	proposal := filepath.Join(root, DefaultProposalPath)
	if err := os.MkdirAll(filepath.Dir(proposal), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proposal, data, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("manifest persistence unexpectedly blocked: %#v", plan.BlockedActions)
	}
	found := false
	for _, blocked := range plan.BlockedActions {
		if blocked.Type != "PROVIDER_CREATE" {
			continue
		}
		for _, key := range blocked.DecisionKeys {
			if key == "repository.visibility" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("provider action did not remain blocked by unverified visibility claim: %#v", plan.BlockedActions)
	}
}

type fakeGitHubReader struct {
	tree  posture.GitHubTree
	trees map[string]posture.GitHubTree
	blobs map[string][]byte
}

func (f fakeGitHubReader) Repository(context.Context, string) (posture.GitHubRepository, posture.ReadObservation, error) {
	return posture.GitHubRepository{DefaultBranch: "main"}, available("repo"), nil
}

func (f fakeGitHubReader) Branch(context.Context, string, string) (posture.GitHubBranch, posture.ReadObservation, error) {
	return posture.GitHubBranch{Name: "main", CommitSHA: "1111111111111111111111111111111111111111", TreeSHA: "2222222222222222222222222222222222222222"}, available("branch"), nil
}

func (f fakeGitHubReader) BranchProtection(context.Context, string, string) (posture.GitHubProtection, posture.ReadObservation, error) {
	return posture.GitHubProtection{}, available("protection"), nil
}

func (f fakeGitHubReader) Tree(_ context.Context, repository string, _ string) (posture.GitHubTree, posture.ReadObservation, error) {
	if f.trees != nil {
		if tree, ok := f.trees[repository]; ok {
			return tree, available("tree"), nil
		}
	}
	return f.tree, available("tree"), nil
}

func (f fakeGitHubReader) Blob(_ context.Context, _ string, sha string) ([]byte, posture.ReadObservation, error) {
	data, ok := f.blobs[sha]
	if !ok {
		return nil, posture.ReadObservation{}, errors.New("missing fake blob")
	}
	return data, available("blob"), nil
}

func available(reference string) posture.ReadObservation {
	return posture.ReadObservation{Available: true, Evidence: posture.Evidence{Source: "test", Reference: reference}}
}

func TestDiscoveryDoesNotInferCIFromWorkflowAndClassifiesRescans(t *testing.T) {
	reader := fakeGitHubReader{
		tree: posture.GitHubTree{Entries: []posture.GitHubTreeEntry{
			{Path: ".github/workflows/ci.yml", Type: "blob", SHA: "a"},
			{Path: "flake.nix", Type: "blob", SHA: "b"},
		}},
		trees: map[string]posture.GitHubTree{
			"hackelia-micrantha/.github": {Entries: []posture.GitHubTreeEntry{
				{Path: "SECURITY.md", Type: "blob", SHA: "security"},
			}},
		},
		blobs: map[string][]byte{},
	}
	spec := config.Spec{Repos: []config.Repo{{
		ID:      "other",
		Mirrors: []config.Endpoint{{Provider: "github", Path: "hackelia-micrantha/other"}},
	}}}
	first, err := Discover(context.Background(), reader, spec, "hackelia-micrantha/new-repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Registration != "unregistered" || first.CIApplicability != "unresolved" || first.ScanState != "first-discovery" {
		t.Fatalf("unexpected first discovery: %#v", first)
	}
	if first.InheritedBaseline[0].State != "provider-inherited" {
		t.Fatalf("inherited baseline = %#v", first.InheritedBaseline)
	}
	second, err := Discover(context.Background(), reader, spec, "hackelia-micrantha/new-repo", &first)
	if err != nil {
		t.Fatal(err)
	}
	if second.ScanState != "unchanged" {
		t.Fatalf("ScanState = %q, want unchanged", second.ScanState)
	}
}

func TestDiscoveryRejectsTamperedPreviousFingerprint(t *testing.T) {
	reader := fakeGitHubReader{tree: posture.GitHubTree{}, blobs: map[string][]byte{}}
	first, err := Discover(context.Background(), reader, config.Spec{}, "hackelia-micrantha/new-repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	first.Registration = "registered"
	_, err = Discover(context.Background(), reader, config.Spec{}, "hackelia-micrantha/new-repo", &first)
	if err == nil {
		t.Fatal("Discover() accepted previous artifact whose evidence no longer matches its fingerprint")
	}
}

func TestDiscoveryDoesNotClaimInheritanceWithoutSourceEvidence(t *testing.T) {
	reader := fakeGitHubReader{tree: posture.GitHubTree{}, trees: map[string]posture.GitHubTree{}, blobs: map[string][]byte{}}
	discovery, err := Discover(context.Background(), reader, config.Spec{}, "hackelia-micrantha/new-repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range discovery.InheritedBaseline {
		if artifact.State == "provider-inherited" {
			t.Fatalf("unexpected provider-inherited state without source evidence: %#v", artifact)
		}
	}
}

func TestDiscoveryUsesExplicitManifestDecisionForCIApplicability(t *testing.T) {
	manifest := DefaultManifest()
	for i := range manifest.Decisions {
		if manifest.Decisions[i].Key == "delivery.ciProvider" {
			manifest.Decisions[i] = Decision{
				Key:        "delivery.ciProvider",
				State:      "resolved",
				Value:      "github-actions",
				Provenance: &Provenance{Authority: "human", Subject: "repository-owner", Reference: "decision-2"},
			}
		}
	}
	data, err := manifest.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	reader := fakeGitHubReader{
		tree:  posture.GitHubTree{Entries: []posture.GitHubTreeEntry{{Path: DefaultManifestPath, Type: "blob", SHA: "manifest"}}},
		blobs: map[string][]byte{"manifest": data},
	}
	discovery, err := Discover(context.Background(), reader, config.Spec{}, "hackelia-micrantha/new-repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if discovery.ManifestState != "valid" || discovery.CIApplicability != "applicable" {
		t.Fatalf("discovery = %#v", discovery)
	}
}
