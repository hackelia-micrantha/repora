package managedartifactplan

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"repoctl/internal/managedartifact"
)

func TestArtifactMarshalParseRoundTrip(t *testing.T) {
	artifact := validNewREADMEArtifact()
	data, err := artifact.Marshal()
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if !reflect.DeepEqual(parsed, artifact) {
		t.Fatalf("round trip mismatch\ngot: %#v\nwant: %#v", parsed, artifact)
	}
	second, err := parsed.Marshal()
	if err != nil {
		t.Fatalf("second Marshal error = %v", err)
	}
	if string(second) != string(data) {
		t.Fatalf("serialization is not deterministic\nfirst: %s\nsecond: %s", data, second)
	}
}

func TestArtifactAllowsExplicitEmptyRepositoryList(t *testing.T) {
	artifact := Artifact{Kind: Kind, Version: Version, Repositories: []RepositoryPlan{}}
	if err := artifact.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}
	data, err := artifact.Marshal()
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	if _, err := Parse(data); err != nil {
		t.Fatalf("Parse error = %v", err)
	}
}

func TestArtifactRejectsNilRepositoryList(t *testing.T) {
	artifact := Artifact{Kind: Kind, Version: Version}
	if err := artifact.Validate(); err == nil {
		t.Fatal("Validate accepted missing repositories array")
	}
}

func TestArtifactAcceptsExistingExecutableREADMEWhenModeIsPreserved(t *testing.T) {
	artifact := validExistingREADMEArtifact(ModeExecutable)
	if err := artifact.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestArtifactSemanticValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Artifact)
		wantErr string
	}{
		{
			name: "unsupported kind",
			mutate: func(a *Artifact) {
				a.Kind = "other"
			},
			wantErr: "unsupported managed artifact plan contract",
		},
		{
			name: "unsafe provider path",
			mutate: func(a *Artifact) {
				a.Repositories[0].Target.Path = "../repora"
			},
			wantErr: "target path",
		},
		{
			name: "invalid branch",
			mutate: func(a *Artifact) {
				a.Repositories[0].Target.Branch = "main..old"
			},
			wantErr: "target branch",
		},
		{
			name: "invalid base oid",
			mutate: func(a *Artifact) {
				a.Repositories[0].BaseOID = "deadbeef"
			},
			wantErr: "base_oid",
		},
		{
			name: "missing action",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions = nil
			},
			wantErr: "exactly one README action",
		},
		{
			name: "multiple actions",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions = append(a.Repositories[0].Actions, a.Repositories[0].Actions[0])
			},
			wantErr: "exactly one README action",
		},
		{
			name: "unsupported action type",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Type = "WRITE_FILE"
			},
			wantErr: "unsupported action type",
		},
		{
			name: "wrong output path",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Path = "docs/README.md"
			},
			wantErr: "managed README path",
		},
		{
			name: "absent observed state carries mode",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Observed.Mode = ModeRegular
			},
			wantErr: "absent README must not include",
		},
		{
			name: "present observed state missing digest",
			mutate: func(a *Artifact) {
				action := &a.Repositories[0].Actions[0]
				action.Observed = ObservedState{Present: true, Mode: ModeRegular}
			},
			wantErr: "observed: sha256",
		},
		{
			name: "new README executable mode",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Desired.Mode = ModeExecutable
			},
			wantErr: "new README must use mode",
		},
		{
			name: "existing README mode change",
			mutate: func(a *Artifact) {
				*a = validExistingREADMEArtifact(ModeExecutable)
				a.Repositories[0].Actions[0].Desired.Mode = ModeRegular
			},
			wantErr: "must preserve observed mode",
		},
		{
			name: "no-op content",
			mutate: func(a *Artifact) {
				*a = validExistingREADMEArtifact(ModeRegular)
				action := &a.Repositories[0].Actions[0]
				action.Desired.Content = "# Old\n"
				action.Desired.SHA256 = action.Observed.SHA256
			},
			wantErr: "must change observed content",
		},
		{
			name: "desired digest mismatch",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Desired.SHA256 = strings.Repeat("0", 64)
			},
			wantErr: "does not match desired content",
		},
		{
			name: "uppercase desired digest",
			mutate: func(a *Artifact) {
				action := &a.Repositories[0].Actions[0]
				action.Desired.SHA256 = strings.ToUpper(action.Desired.SHA256)
			},
			wantErr: "lowercase SHA-256",
		},
		{
			name: "invalid template digest",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].TemplateSHA256 = "abc"
			},
			wantErr: "template_sha256",
		},
		{
			name: "CR desired content",
			mutate: func(a *Artifact) {
				action := &a.Repositories[0].Actions[0]
				action.Desired.Content = "# Repora\r\n"
				action.Desired.SHA256 = SHA256([]byte(action.Desired.Content))
			},
			wantErr: "normalized LF",
		},
		{
			name: "oversized desired content",
			mutate: func(a *Artifact) {
				action := &a.Repositories[0].Actions[0]
				action.Desired.Content = strings.Repeat("x", managedartifact.MaxTextBytes+1)
				action.Desired.SHA256 = SHA256([]byte(action.Desired.Content))
			},
			wantErr: "README limit",
		},
		{
			name: "wrong diff labels",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Diff = "--- /tmp/a\n+++ /tmp/b\n"
			},
			wantErr: "fixed README path labels",
		},
		{
			name: "diff control sequence",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Diff += "\x1b[31mred\x1b[0m\n"
			},
			wantErr: "unsupported control data",
		},
		{
			name: "oversized diff",
			mutate: func(a *Artifact) {
				a.Repositories[0].Actions[0].Diff = "--- a/README.md\n+++ b/README.md\n" + strings.Repeat("x", MaxDiffBytes)
			},
			wantErr: "diff exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := validNewREADMEArtifact()
			tt.mutate(&artifact)
			err := artifact.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestArtifactRejectsDuplicateIdentityAndTarget(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*RepositoryPlan)
		wantErr string
	}{
		{
			name: "uid",
			mutate: func(repo *RepositoryPlan) {
				repo.ID = "repora-two"
				repo.Target.Path = "micrantha/repora-two"
			},
			wantErr: "duplicate uid",
		},
		{
			name: "id",
			mutate: func(repo *RepositoryPlan) {
				repo.UID = "repo.repora-two"
				repo.Target.Path = "micrantha/repora-two"
			},
			wantErr: "duplicate id",
		},
		{
			name: "target",
			mutate: func(repo *RepositoryPlan) {
				repo.UID = "repo.repora-two"
				repo.ID = "repora-two"
			},
			wantErr: "duplicate target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := validNewREADMEArtifact()
			second := artifact.Repositories[0]
			second.Actions = append([]Action(nil), second.Actions...)
			tt.mutate(&second)
			artifact.Repositories = append(artifact.Repositories, second)
			err := artifact.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseRejectsStructuralJSONDefects(t *testing.T) {
	valid, err := validNewREADMEArtifact().Marshal()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{
			name: "unknown field",
			mutate: func(root map[string]any) {
				root["unexpected"] = true
			},
			wantErr: "unknown field",
		},
		{
			name: "null field",
			mutate: func(root map[string]any) {
				desiredMap(root)["content"] = nil
			},
			wantErr: "must not be null",
		},
		{
			name: "missing observed present",
			mutate: func(root map[string]any) {
				delete(observedMap(root), "present")
			},
			wantErr: "observed.present field is required",
		},
		{
			name: "missing desired content",
			mutate: func(root map[string]any) {
				delete(desiredMap(root), "content")
			},
			wantErr: "desired.content field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal(valid, &root); err != nil {
				t.Fatal(err)
			}
			tt.mutate(root)
			data, err := json.Marshal(root)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(data)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Parse error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}

	if _, err := Parse(append(valid, []byte("\n{}")...)); err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("Parse trailing value error = %v", err)
	}
}

func validNewREADMEArtifact() Artifact {
	content := "# Repora\n"
	return Artifact{
		Kind:    Kind,
		Version: Version,
		Repositories: []RepositoryPlan{
			{
				UID: "repo.repora",
				ID:  "repora",
				Target: Target{
					Provider: "gitlab",
					Path:     "micrantha/repora",
					Branch:   "main",
				},
				BaseOID: strings.Repeat("a", 40),
				Actions: []Action{
					{
						Type:     ActionWriteREADME,
						Path:     READMEPath,
						Observed: ObservedState{Present: false},
						Desired: DesiredState{
							Mode:    ModeRegular,
							SHA256:  SHA256([]byte(content)),
							Content: content,
						},
						TemplateSHA256: SHA256([]byte("# {{value.title}}\n")),
						Diff:           "--- a/README.md\n+++ b/README.md\n@@ -0,0 +1 @@\n+# Repora\n",
					},
				},
			},
		},
	}
}

func validExistingREADMEArtifact(mode string) Artifact {
	artifact := validNewREADMEArtifact()
	action := &artifact.Repositories[0].Actions[0]
	action.Observed = ObservedState{
		Present: true,
		Mode:    mode,
		SHA256:  SHA256([]byte("# Old\n")),
	}
	action.Desired.Mode = mode
	action.Desired.Content = "# New\n"
	action.Desired.SHA256 = SHA256([]byte(action.Desired.Content))
	action.Diff = "--- a/README.md\n+++ b/README.md\n@@ -1 +1 @@\n-# Old\n+# New\n"
	return artifact
}

func observedMap(root map[string]any) map[string]any {
	repositories := root["repositories"].([]any)
	repository := repositories[0].(map[string]any)
	actions := repository["actions"].([]any)
	action := actions[0].(map[string]any)
	return action["observed"].(map[string]any)
}

func desiredMap(root map[string]any) map[string]any {
	repositories := root["repositories"].([]any)
	repository := repositories[0].(map[string]any)
	actions := repository["actions"].([]any)
	action := actions[0].(map[string]any)
	return action["desired"].(map[string]any)
}
