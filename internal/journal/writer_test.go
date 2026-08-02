package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestWriterPersistsValidatedRecordAppendOnly(t *testing.T) {
	root := t.TempDir()
	record, err := FromPlan("run-001", ModePlan, testArtifact())
	if err != nil {
		t.Fatal(err)
	}

	reference, err := (Writer{Root: root}).Write(record)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	wantReference := ".repora/journal/" + record.Repository.UID + "--run-001.json"
	if reference != wantReference {
		t.Fatalf("reference = %q, want %q", reference, wantReference)
	}
	if filepath.IsAbs(reference) || strings.Contains(reference, "..") {
		t.Fatalf("reference is not safe: %q", reference)
	}

	path := filepath.Join(root, filepath.FromSlash(reference))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Parse(data)
	if err != nil {
		t.Fatalf("persisted record does not parse: %v", err)
	}
	if decoded.ExecutionID != record.ExecutionID || decoded.Plan.SHA256 != record.Plan.SHA256 {
		t.Fatalf("persisted record = %#v, want %#v", decoded, record)
	}

	collisionReference, err := (Writer{Root: root}).Write(record)
	if !errors.Is(err, ErrRecordExists) {
		t.Fatalf("second Write error = %v, want ErrRecordExists", err)
	}
	if collisionReference != wantReference {
		t.Fatalf("collision reference = %q, want %q", collisionReference, wantReference)
	}
}

func TestWriterInitializesDirectoriesConcurrently(t *testing.T) {
	root := t.TempDir()
	const count = 8

	start := make(chan struct{})
	errorsCh := make(chan error, count)
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			record, err := FromPlan(fmt.Sprintf("run-%03d", i), ModePlan, testArtifact())
			if err == nil {
				_, err = (Writer{Root: root}).Write(record)
			}
			errorsCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errorsCh)

	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent Write returned error: %v", err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(DirectoryName)))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != count {
		t.Fatalf("journal entry count = %d, want %d", len(entries), count)
	}
}

func TestWriterUsesRestrictivePermissionsAndCleansTemporaryFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions are not meaningful on Windows")
	}
	root := t.TempDir()
	record, err := FromPlan("run-002", ModeDryRun, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	reference, err := (Writer{Root: root}).Write(record)
	if err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(root, filepath.FromSlash(DirectoryName))
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("directory permissions = %o, want 700", got)
	}
	fileInfo, err := os.Stat(filepath.Join(root, filepath.FromSlash(reference)))
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("file permissions = %o, want 600", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".record-") {
			t.Fatalf("temporary file was not cleaned up: %s", entry.Name())
		}
	}
}

func TestWriterRejectsInvalidInputBeforeCreatingJournal(t *testing.T) {
	root := t.TempDir()
	record, err := FromPlan("run-003", ModePlan, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	record.ExecutionID = "../escape"
	if _, err := (Writer{Root: root}).Write(record); err == nil || !strings.Contains(err.Error(), "validate journal record") {
		t.Fatalf("Write error = %v, want validation failure", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(DirectoryName))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("journal directory exists after invalid input: %v", err)
	}
	if _, err := (Writer{}).Write(mustRecord(t)); err == nil || !strings.Contains(err.Error(), "root is required") {
		t.Fatalf("empty-root error = %v", err)
	}
}

func TestWriterRejectsSymlinkedReporaParentBeforeOutsideMutation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup varies on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".repora")); err != nil {
		t.Fatal(err)
	}

	if _, err := (Writer{Root: root}).Write(mustRecord(t)); err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("Write error = %v, want parent symlink rejection", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "journal")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside journal directory was created before rejection: %v", err)
	}
}

func TestWriterRejectsJournalDirectorySymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup varies on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".repora"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, filepath.FromSlash(DirectoryName))); err != nil {
		t.Fatal(err)
	}
	if _, err := (Writer{Root: root}).Write(mustRecord(t)); err == nil || !strings.Contains(err.Error(), "must not be a symlink") {
		t.Fatalf("Write error = %v, want symlink rejection", err)
	}
}

func TestWriterRejectsNonDirectoryPathComponent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".repora"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Writer{Root: root}).Write(mustRecord(t)); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("Write error = %v, want non-directory rejection", err)
	}
}

func mustRecord(t *testing.T) Record {
	t.Helper()
	record, err := FromPlan("run-004", ModePlan, testArtifact())
	if err != nil {
		t.Fatal(err)
	}
	return record
}
