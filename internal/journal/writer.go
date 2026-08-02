package journal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DirectoryName = ".repora/journal"

var ErrRecordExists = errors.New("journal record already exists")

// Writer persists validated records beneath a caller-owned root directory.
// It never overwrites an existing record and returns a slash-separated path
// relative to that root for safe display and later lookup.
type Writer struct {
	Root string
}

func (w Writer) Write(record Record) (string, error) {
	encoded, err := record.Marshal()
	if err != nil {
		return "", fmt.Errorf("validate journal record: %w", err)
	}
	if strings.TrimSpace(w.Root) == "" {
		return "", fmt.Errorf("journal root is required")
	}

	root, err := resolveJournalRoot(w.Root)
	if err != nil {
		return "", err
	}
	reporaDir, err := ensurePrivateDirectory(root, ".repora")
	if err != nil {
		return "", fmt.Errorf("prepare journal parent: %w", err)
	}
	journalDir, err := ensurePrivateDirectory(reporaDir, "journal")
	if err != nil {
		return "", fmt.Errorf("prepare journal directory: %w", err)
	}
	if !isContained(root, journalDir) {
		return "", fmt.Errorf("journal directory escapes root")
	}

	name := record.Repository.UID + "--" + record.ExecutionID + ".json"
	reference := filepath.ToSlash(filepath.Join(DirectoryName, name))
	finalPath := filepath.Join(journalDir, name)
	if filepath.Dir(finalPath) != journalDir {
		return "", fmt.Errorf("journal filename escapes journal directory")
	}

	temp, err := os.CreateTemp(journalDir, ".record-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create journal temporary file: %w", err)
	}
	tempPath := temp.Name()
	published := false
	defer func() {
		_ = temp.Close()
		if !published {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		return "", fmt.Errorf("secure journal temporary file: %w", err)
	}
	if _, err := temp.Write(append(encoded, '\n')); err != nil {
		return "", fmt.Errorf("write journal record: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return "", fmt.Errorf("sync journal record: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("close journal record: %w", err)
	}

	if err := os.Link(tempPath, finalPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return reference, fmt.Errorf("%w: %s", ErrRecordExists, reference)
		}
		return "", fmt.Errorf("commit journal record: %w", err)
	}
	published = true

	cleanupErr := os.Remove(tempPath)
	syncErr := syncDirectory(journalDir)
	if err := errors.Join(cleanupErr, syncErr); err != nil {
		return reference, fmt.Errorf("journal record published but post-commit cleanup or sync failed: %w", err)
	}

	return reference, nil
}

func resolveJournalRoot(value string) (string, error) {
	root, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve journal root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve journal root symlinks: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("inspect journal root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("journal root is not a directory")
	}
	return root, nil
}

// ensurePrivateDirectory creates exactly one child directory. Existing symbolic
// links and non-directories are rejected before any descendant is created.
func ensurePrivateDirectory(parent, name string) (string, error) {
	path := filepath.Join(parent, name)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil {
			return "", fmt.Errorf("create %q: %w", path, err)
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return "", fmt.Errorf("inspect %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("journal path component %q must not be a symlink", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("journal path component %q is not a directory", path)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return "", fmt.Errorf("secure %q: %w", path, err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", path, err)
	}
	if !isContained(parent, resolved) {
		return "", fmt.Errorf("journal path component %q escapes parent", path)
	}
	return resolved, nil
}

func isContained(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open journal directory: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("sync journal directory: %w", err)
	}
	return nil
}
