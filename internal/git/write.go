package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	reporaCommitName  = "Repora"
	reporaCommitEmail = "repora@localhost.invalid"
)

// WriteBlob writes content as an otherwise-unreferenced Git blob in the local
// object database and returns its object ID.
func (Client) WriteBlob(repoPath string, content []byte) (string, error) {
	out, err := outputBytesWithInput(repoPath, content, nil, "hash-object", "-w", "--stdin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// BuildTreeWithRootBlob creates a tree derived from baseOID with exactly one
// root entry replaced or added. It does not update an index or any ref.
func (Client) BuildTreeWithRootBlob(repoPath, baseOID, treePath, mode, blobOID string) (string, error) {
	if treePath == "" || strings.ContainsAny(treePath, "/\x00\r\n") {
		return "", fmt.Errorf("tree path must be one non-empty root path")
	}
	if mode != "100644" && mode != "100755" {
		return "", fmt.Errorf("root blob mode must be 100644 or 100755")
	}

	current, err := outputBytes(repoPath, "ls-tree", "-z", "--full-tree", baseOID)
	if err != nil {
		return "", err
	}
	var input bytes.Buffer
	for _, record := range bytes.Split(current, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		tab := bytes.IndexByte(record, '\t')
		if tab < 0 {
			return "", fmt.Errorf("git ls-tree returned malformed root entry")
		}
		if string(record[tab+1:]) == treePath {
			continue
		}
		input.Write(record)
		input.WriteByte(0)
	}
	fmt.Fprintf(&input, "%s blob %s\t%s", mode, blobOID, treePath)
	input.WriteByte(0)

	out, err := outputBytesWithInput(repoPath, input.Bytes(), nil, "mktree", "-z")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CreateCommitObject creates an otherwise-unreferenced local commit object.
// Author and committer identity are fixed to Repora; both dates use one current
// UTC instant. No ref is created or updated.
func (Client) CreateCommitObject(repoPath, treeOID, parentOID, message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" || strings.ContainsRune(message, '\x00') {
		return "", fmt.Errorf("commit message is required and must not contain NUL")
	}
	now := time.Now().UTC()
	date := fmt.Sprintf("%d +0000", now.Unix())
	env := []gitEnvOverride{
		{Key: "GIT_AUTHOR_NAME", Value: reporaCommitName},
		{Key: "GIT_AUTHOR_EMAIL", Value: reporaCommitEmail},
		{Key: "GIT_AUTHOR_DATE", Value: date},
		{Key: "GIT_COMMITTER_NAME", Value: reporaCommitName},
		{Key: "GIT_COMMITTER_EMAIL", Value: reporaCommitEmail},
		{Key: "GIT_COMMITTER_DATE", Value: date},
	}
	out, err := outputBytesWithInput(repoPath, nil, env, "commit-tree", treeOID, "-p", parentOID, "-m", message, "--no-gpg-sign")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ChangedPaths returns the exact recursively changed paths between two commit
// objects without updating repository state.
func (Client) ChangedPaths(repoPath, baseOID, commitOID string) ([]string, error) {
	out, err := outputBytes(repoPath, "diff-tree", "--no-commit-id", "--name-only", "-r", "-z", baseOID, commitOID)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		paths = append(paths, string(raw))
	}
	return paths, nil
}

type gitEnvOverride struct {
	Key   string
	Value string
}

func outputBytesWithInput(repoPath string, input []byte, overrides []gitEnvOverride, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := gitCommandContext(ctx, repoPath, args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	if len(overrides) > 0 {
		cmd.Env = applyGitEnvOverrides(os.Environ(), overrides)
	}
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

func applyGitEnvOverrides(base []string, overrides []gitEnvOverride) []string {
	replaced := make(map[string]struct{}, len(overrides))
	for _, override := range overrides {
		replaced[override.Key] = struct{}{}
	}
	env := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		key, _, _ := strings.Cut(item, "=")
		if _, replace := replaced[key]; replace {
			continue
		}
		env = append(env, item)
	}
	for _, override := range overrides {
		env = append(env, override.Key+"="+override.Value)
	}
	return env
}
