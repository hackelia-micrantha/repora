package managedartifact

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommittedManagedArtifactPlanExampleParses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "managed-artifact-plan-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ParsePlan(data)
	if err != nil {
		t.Fatalf("ParsePlan(example) error = %v", err)
	}
	encoded, err := plan.Marshal()
	if err != nil {
		t.Fatalf("Marshal(example) error = %v", err)
	}
	roundTrip, err := ParsePlan(encoded)
	if err != nil {
		t.Fatalf("ParsePlan(Marshal(example)) error = %v", err)
	}
	if !reflect.DeepEqual(roundTrip, plan) {
		t.Fatalf("example round trip changed plan\ngot: %#v\nwant: %#v", roundTrip, plan)
	}
}
