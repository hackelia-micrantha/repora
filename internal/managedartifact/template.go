package managedartifact

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadTemplate reads a managed README template relative to the physical
// directory containing the configuration file. The resolved template must
// remain inside that directory and be a bounded regular file.
func LoadTemplate(configPath, templatePath string) ([]byte, error) {
	if strings.TrimSpace(templatePath) == "" {
		return nil, fmt.Errorf("README template path is required")
	}
	if filepath.IsAbs(templatePath) || strings.Contains(templatePath, `\`) {
		return nil, fmt.Errorf("README template path must be configuration-root-relative")
	}
	clean := filepath.Clean(filepath.FromSlash(templatePath))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("README template path escapes configuration root")
	}

	absoluteConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration path: %w", err)
	}
	resolvedConfig, err := filepath.EvalSymlinks(absoluteConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration path: %w", err)
	}
	root := filepath.Dir(resolvedConfig)

	candidate := filepath.Join(root, clean)
	resolvedTemplate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return nil, fmt.Errorf("resolve README template: %w", err)
	}
	if !pathWithin(root, resolvedTemplate) {
		return nil, fmt.Errorf("README template resolves outside configuration root")
	}

	file, err := os.Open(resolvedTemplate)
	if err != nil {
		return nil, fmt.Errorf("open README template: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat README template: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("README template must resolve to a regular file")
	}
	if info.Size() > MaxTextBytes {
		return nil, fmt.Errorf("README template exceeds %d-byte limit", MaxTextBytes)
	}

	data, err := io.ReadAll(io.LimitReader(file, MaxTextBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read README template: %w", err)
	}
	if len(data) > MaxTextBytes {
		return nil, fmt.Errorf("README template exceeds %d-byte limit", MaxTextBytes)
	}
	return data, nil
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
