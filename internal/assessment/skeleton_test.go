package assessment

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewSkeletonMatchesCommittedTemplate(t *testing.T) {
	skeleton := NewSkeleton()
	if err := skeleton.Validate(); err != nil {
		t.Fatalf("NewSkeleton().Validate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join("..", "..", "templates", "assessments", "repository-assessment-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	committed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(committed template) error = %v", err)
	}
	if !reflect.DeepEqual(skeleton, committed) {
		t.Fatalf("NewSkeleton() does not match committed template\nnew: %#v\ncommitted: %#v", skeleton, committed)
	}
}
