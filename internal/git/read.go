package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

// TreeEntry is one exact entry from a Git tree.
type TreeEntry struct {
	Mode string
	Type string
	OID  string
	Path string
}

// ReadTreeEntry resolves one exact path at rev without checking out a worktree.
// The bool result is false when the path is absent.
func (Client) ReadTreeEntry(repoPath, rev, treePath string) (TreeEntry, bool, error) {
	out, err := outputBytes(repoPath, "ls-tree", "-z", "--full-tree", rev, "--", treePath)
	if err != nil {
		return TreeEntry{}, false, err
	}
	if len(out) == 0 {
		return TreeEntry{}, false, nil
	}

	records := bytes.Split(out, []byte{0})
	if len(records) != 2 || len(records[1]) != 0 {
		return TreeEntry{}, false, fmt.Errorf("git ls-tree returned multiple or unterminated records for %q", treePath)
	}
	parts := bytes.SplitN(records[0], []byte{'\t'}, 2)
	if len(parts) != 2 {
		return TreeEntry{}, false, fmt.Errorf("git ls-tree returned malformed entry for %q", treePath)
	}
	fields := strings.Fields(string(parts[0]))
	if len(fields) != 3 {
		return TreeEntry{}, false, fmt.Errorf("git ls-tree returned malformed metadata for %q", treePath)
	}
	entry := TreeEntry{
		Mode: fields[0],
		Type: fields[1],
		OID:  fields[2],
		Path: string(parts[1]),
	}
	if entry.Path != treePath {
		return TreeEntry{}, false, fmt.Errorf("git ls-tree returned unexpected path %q for %q", entry.Path, treePath)
	}
	return entry, true, nil
}

// ReadBlobBounded reads an exact Git blob only after checking its declared
// object size. This prevents callers from materializing an unbounded blob in
// memory merely to reject it afterward.
func (Client) ReadBlobBounded(repoPath, oid string, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 {
		return nil, fmt.Errorf("blob byte limit must be non-negative")
	}
	sizeData, err := outputBytes(repoPath, "cat-file", "-s", oid)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64)
	if err != nil || size < 0 {
		return nil, fmt.Errorf("git cat-file returned invalid blob size for %q", oid)
	}
	if size > maxBytes {
		return nil, fmt.Errorf("git blob %q exceeds %d-byte limit", oid, maxBytes)
	}
	data, err := outputBytes(repoPath, "cat-file", "blob", oid)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != size {
		return nil, fmt.Errorf("git blob %q size changed while reading: expected %d bytes, got %d", oid, size, len(data))
	}
	return data, nil
}

func outputBytes(repoPath string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := gitCommandContext(ctx, repoPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		command := redactSensitive(strings.Join(args, " "))
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("git %s: %w", command, ctxErr)
		}
		detail := strings.TrimSpace(redactSensitive(string(out)))
		if detail != "" {
			return nil, fmt.Errorf("git %s: %w: %s", command, err, detail)
		}
		return nil, fmt.Errorf("git %s: %w", command, err)
	}
	return out, nil
}
