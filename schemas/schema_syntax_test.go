package schemas_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCommittedSchemasAreValidJSON(t *testing.T) {
	paths, err := filepath.Glob("*.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no committed schemas found")
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var document any
			if err := json.Unmarshal(data, &document); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
		})
	}
}
