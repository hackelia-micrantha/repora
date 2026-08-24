package git

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Client struct{}

const defaultGitTimeout = 30 * time.Second
const cacheDirectoryEnvironment = "REPORA_CACHE_DIR"

var gitTimeout = defaultGitTimeout

var urlCredentialPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://)([^/@\s]+)@`)
var scpCredentialPattern = regexp.MustCompile(`(?i)([^\s/@:]+):([^\s/@]+)@([a-z0-9.-]+)`)
var credentialValuePattern = regexp.MustCompile(`(?i)\b(password|passwd|token|access_token|oauth2)=([^\s&]+)`)

func MirrorPath(identity string) (string, error) {
	segment, err := SafePathSegment(identity)
	if err != nil {
		return "", err
	}
	root, err := cacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, segment+".git"), nil
}

func cacheRoot() (string, error) {
	if configured := os.Getenv(cacheDirectoryEnvironment); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("%s must be an absolute path", cacheDirectoryEnvironment)
		}
		return filepath.Clean(configured), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".cache", "repora"), nil
}

func SafePathSegment(identity string) (string, error) {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return "", fmt.Errorf("identity is required")
	}
	return "uid-" + base64.RawURLEncoding.EncodeToString([]byte(identity)), nil
}

func (Client) EnsureMirror(path, canonicalURL string) error {
	if err := rejectSymlinkComponents(path); err != nil {
		return fmt.Errorf("validate mirror path: %w", err)
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("validate mirror path: symlink component %q is not allowed", path)
		}
		if !info.IsDir() {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove invalid mirror path: %w", err)
			}
		} else {
			valid, err := isValidMirror(path)
			if err != nil {
				return fmt.Errorf("validate mirror repo: %w", err)
			}
			if valid {
				return nil
			}
			if err := removeMirrorPath(path); err != nil {
				return fmt.Errorf("remove corrupted mirror cache: %w", err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mirror repo: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create mirror cache: %w", err)
	}
	if err := run("", "clone", "--mirror", canonicalURL, path); err != nil {
		if cleanupErr := removeMirrorPath(path); cleanupErr != nil {
			return fmt.Errorf("clone mirror: %w; cleanup incomplete mirror: %v", err, cleanupErr)
		}
		return fmt.Errorf("clone mirror: %w", err)
	}
	return nil
}

func rejectSymlinkComponents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}
	volume := filepath.VolumeName(absolute)
	current := volume + string(os.PathSeparator)
	remainder := strings.TrimPrefix(absolute, current)
	for _, component := range strings.Split(remainder, string(os.PathSeparator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink component %q is not allowed", current)
		}
	}
	return nil
}

func removeMirrorPath(path string) error {
	if err := rejectSymlinkComponents(path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func isValidMirror(path string) (bool, error) {
	out, err := output(path, "rev-parse", "--is-bare-repository")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

func (Client) ConfigureRemote(repoPath, name, url string) error {
	if err := run(repoPath, "remote", "get-url", name); err == nil {
		return run(repoPath, "remote", "set-url", name, url)
	}
	return run(repoPath, "remote", "add", name, url)
}

func (Client) Fetch(repoPath, name string) error {
	return run(repoPath, "fetch", "--prune", name)
}

func (Client) SyncMirrorFromRemote(repoPath, remote string) error {
	return run(repoPath, "fetch", "--prune", remote, "+refs/*:refs/*")
}

func (Client) SetRemoteHead(repoPath, name string) error {
	return run(repoPath, "remote", "set-head", name, "-a")
}

func (Client) RevListLeftRightCount(repoPath, left, right string) (string, error) {
	return output(repoPath, "rev-list", "--left-right", "--count", left+"..."+right)
}

func (Client) RevParseShort(repoPath, rev string) (string, error) {
	return output(repoPath, "rev-parse", "--short", rev)
}

func (Client) PushMirror(repoPath, remote string) error {
	return run(repoPath, "push", "--mirror", remote)
}

func (Client) ResolveRevision(repoPath, rev string) (string, error) {
	out, err := output(repoPath, "rev-parse", rev)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (Client) ResolveRemoteHeadBranch(repoPath, remote string) (string, error) {
	out, err := output(repoPath, "symbolic-ref", "--short", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(out)
	prefix := remote + "/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix), nil
	}
	return ref, nil
}

func (Client) PushBranch(repoPath, remote, srcRef, dstBranch string) error {
	refspec := srcRef + ":refs/heads/" + dstBranch
	return run(repoPath, "push", remote, refspec)
}

func (Client) ForcePushBranchWithLease(repoPath, remote, srcRef, dstBranch, expectedOldOID string) error {
	lease := "--force-with-lease=refs/heads/" + dstBranch + ":" + expectedOldOID
	refspec := srcRef + ":refs/heads/" + dstBranch
	return run(repoPath, "push", remote, lease, refspec)
}

func run(repoPath string, args ...string) error {
	return runContext(context.Background(), repoPath, args...)
}

func runContext(parent context.Context, repoPath string, args ...string) error {
	ctx, cancel := context.WithTimeout(parent, gitTimeout)
	defer cancel()

	cmd := gitCommandContext(ctx, repoPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		command := redactSensitive(strings.Join(args, " "))
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("git %s: %w", command, ctxErr)
		}
		detail := strings.TrimSpace(redactSensitive(string(out)))
		if detail != "" {
			return fmt.Errorf("git %s: %w: %s", command, err, detail)
		}
		return fmt.Errorf("git %s: %w", command, err)
	}
	return nil
}

func output(repoPath string, args ...string) (string, error) {
	return outputContext(context.Background(), repoPath, args...)
}

func outputContext(parent context.Context, repoPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, gitTimeout)
	defer cancel()

	cmd := gitCommandContext(ctx, repoPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		command := redactSensitive(strings.Join(args, " "))
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", fmt.Errorf("git %s: %w", command, ctxErr)
		}
		return "", fmt.Errorf("git %s: %w: %s", command, err, bytes.TrimSpace([]byte(redactSensitive(string(out)))))
	}
	return string(out), nil
}

func redactSensitive(value string) string {
	value = urlCredentialPattern.ReplaceAllString(value, `${1}[REDACTED]@`)
	value = scpCredentialPattern.ReplaceAllString(value, `[REDACTED]@$3`)
	return credentialValuePattern.ReplaceAllString(value, `$1=[REDACTED]`)
}

func gitCommandContext(ctx context.Context, repoPath string, args ...string) *exec.Cmd {
	if repoPath != "" {
		args = append([]string{"-C", repoPath}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	configureGitCommand(cmd)
	return cmd
}
