package posture

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// CollectLocalGitStorage reads only locally materialized Git object-store facts.
// The supplied repository name is an operator assertion; no remote identity is
// inferred or verified and no canonical/history-size claim follows from it.
func CollectLocalGitStorage(ctx context.Context, checkoutPath, repository string) (StorageInventory, error) {
	inventory := NewStorageInventory(repository)
	if _, _, err := splitGitHubFullName(repository); err != nil {
		return StorageInventory{}, err
	}
	if strings.TrimSpace(checkoutPath) == "" {
		return StorageInventory{}, fmt.Errorf("local checkout path is required")
	}

	output, code, err := storageGitRead(ctx, checkoutPath, "count-objects", "-v")
	if err != nil || code != 0 {
		return StorageInventory{}, storageCommandError("count-objects", code, err)
	}
	values, err := parseGitObjectCounts(output)
	if err != nil {
		return StorageInventory{}, err
	}
	countEvidence := Evidence{Source: "git-local", Reference: "count-objects -v", Detail: "local materialized objects only; not canonical repository size"}
	inventory.Objects.LooseCount = Observed(values["count"], countEvidence)
	inventory.Objects.LooseBytes = Observed(values["size"]*1024, countEvidence)
	inventory.Objects.PackedCount = Observed(values["in-pack"], countEvidence)
	inventory.Objects.PackedBytes = Observed(values["size-pack"]*1024, countEvidence)
	inventory.Objects.PackCount = Observed(values["packs"], countEvidence)

	shallowOutput, code, err := storageGitRead(ctx, checkoutPath, "rev-parse", "--is-shallow-repository")
	if err != nil || code != 0 {
		return StorageInventory{}, storageCommandError("rev-parse", code, err)
	}
	stateEvidence := Evidence{Source: "git-local", Reference: "rev-parse --is-shallow-repository"}
	switch strings.TrimSpace(shallowOutput) {
	case "true":
		inventory.GitState.Shallow = Observed(true, stateEvidence)
	case "false":
		inventory.GitState.Shallow = Observed(false, stateEvidence)
	default:
		return StorageInventory{}, fmt.Errorf("git returned an invalid shallow-repository state")
	}

	// These are configuration observations, not a proof that all promised or
	// remotely reachable objects have been materialized.
	promisorOutput, code, err := storageGitRead(ctx, checkoutPath, "config", "--local", "--get-regexp", `^remote\..*\.promisor$`)
	if err != nil || (code != 0 && code != 1) {
		return StorageInventory{}, storageCommandError("config", code, err)
	}
	promisor := false
	if code == 0 {
		for _, line := range strings.Split(strings.TrimSpace(promisorOutput), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return StorageInventory{}, fmt.Errorf("invalid promisor configuration observation")
			}
			if strings.EqualFold(fields[1], "true") {
				promisor = true
			}
		}
	}
	extension, code, err := storageGitRead(ctx, checkoutPath, "config", "--local", "--get", "extensions.partialclone")
	if err != nil || (code != 0 && code != 1) {
		return StorageInventory{}, storageCommandError("config", code, err)
	}
	if code == 0 && strings.TrimSpace(extension) != "" {
		promisor = true
	}
	inventory.GitState.PromisorConfigured = Observed(promisor, Evidence{
		Source: "git-local", Reference: "local promisor/partial-clone configuration",
		Detail: "absence of configuration is not proof of complete remote object or history coverage",
	})
	inventory.Evidence = []Evidence{countEvidence}
	if err := inventory.Validate(); err != nil {
		return StorageInventory{}, err
	}
	return inventory, nil
}

func storageCommandError(command string, code int, err error) error {
	if err != nil {
		return fmt.Errorf("git %s could not be executed: %w", command, err)
	}
	return fmt.Errorf("git %s failed with exit code %d", command, code)
}

// storageGitRead intentionally does not surface Git stderr: it may contain
// local paths or remote URLs with embedded credentials.
func storageGitRead(ctx context.Context, path string, args ...string) (string, int, error) {
	argv := append([]string{"-C", path}, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	env := make([]string, 0, len(os.Environ())+5)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY",
			"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_CONFIG_COUNT",
			"GIT_CONFIG_PARAMETERS", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM":
			continue
		}
		if strings.HasPrefix(key, "GIT_CONFIG_KEY_") || strings.HasPrefix(key, "GIT_CONFIG_VALUE_") {
			continue
		}
		env = append(env, value)
	}
	cmd.Env = append(env,
		"GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	output, err := cmd.Output()
	if err == nil {
		return string(output), 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return "", exit.ExitCode(), nil
	}
	return "", -1, err
}

func parseGitObjectCounts(output string) (map[string]int64, error) {
	keys := map[string]bool{"count": true, "size": true, "in-pack": true, "packs": true, "size-pack": true}
	values := make(map[string]int64, len(keys))
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		key, raw, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || !keys[key] {
			continue
		}
		if _, duplicate := values[key]; duplicate {
			return nil, fmt.Errorf("duplicate git object counter %q", key)
		}
		number, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil || number < 0 || ((key == "size" || key == "size-pack") && number > math.MaxInt64/1024) {
			return nil, fmt.Errorf("invalid git object counter %q", key)
		}
		values[key] = number
	}
	for key := range keys {
		if _, exists := values[key]; !exists {
			return nil, fmt.Errorf("missing git object counter %q", key)
		}
	}
	return values, nil
}
