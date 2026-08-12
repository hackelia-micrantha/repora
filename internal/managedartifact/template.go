package managedartifact

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// LoadTemplate reads a managed README template relative to the physical
// directory containing the configuration file. The resolved template must
// remain inside that directory and be a bounded regular file.
func LoadTemplate(configPath, templatePath string) ([]byte, error) {
	if strings.TrimSpace(configPath) == "" {
		return nil, fmt.Errorf("configuration path is required")
	}
	if err := validateTemplateReference(templatePath); err != nil {
		return nil, err
	}

	absoluteConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration path: %w", err)
	}
	resolvedConfig, err := filepath.EvalSymlinks(absoluteConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration path: %w", err)
	}
	configInfo, err := os.Stat(resolvedConfig)
	if err != nil {
		return nil, fmt.Errorf("stat configuration path: %w", err)
	}
	if !configInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("configuration path must resolve to a regular file")
	}
	root := filepath.Dir(resolvedConfig)

	candidate := filepath.Join(root, filepath.FromSlash(templatePath))
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

func validateTemplateReference(value string) error {
	if value == "" || value != strings.TrimSpace(value) {
		return fmt.Errorf("README template path must be a canonical configuration-root-relative path")
	}
	if path.IsAbs(value) || filepath.IsAbs(filepath.FromSlash(value)) || strings.HasPrefix(value, "~") || strings.ContainsAny(value, "\\:\x00\r\n\t") {
		return fmt.Errorf("README template path must be a portable configuration-root-relative path")
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029' {
			return fmt.Errorf("README template path contains unsafe display/control character U+%04X", r)
		}
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != value {
		return fmt.Errorf("README template path must not contain traversal or redundant segments")
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("README template path contains an invalid segment")
		}
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
