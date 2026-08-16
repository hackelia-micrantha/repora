package posture

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var inlineMarkdownLink = regexp.MustCompile(`\[[^\]]*\]\(\s*<?([^\s)>]+)>?(?:\s+[^)]*)?\)`)

func documentationPresence(documentPath string, tree GitHubTree, entries map[string]GitHubTreeEntry, evidence Evidence) Fact[bool] {
	entry, exists := entries[documentPath]
	if exists {
		if entry.Type == "blob" {
			return Observed(true, evidence)
		}
		return Observed(false, evidenceWithDetail(evidence, fmt.Sprintf("%s exists but is not a blob", documentPath)))
	}
	if tree.Truncated {
		return Unknown[bool](evidenceWithDetail(evidence, fmt.Sprintf("Git tree is truncated; %s presence is unknown", documentPath)))
	}
	return Observed(false, evidence)
}

func loadDocumentationContent(ctx context.Context, reader GitHubReader, fullName, documentPath string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (documentContent, error) {
	entry, exists := entries[documentPath]
	if !exists {
		if tree.Truncated {
			return documentContent{State: StateUnknown, Evidence: evidenceWithDetail(treeEvidence, fmt.Sprintf("Git tree is truncated; %s presence is unknown", documentPath))}, nil
		}
		return documentContent{State: StateObserved, Evidence: treeEvidence}, nil
	}
	if entry.Type != "blob" {
		return documentContent{State: StateObserved, Evidence: evidenceWithDetail(treeEvidence, fmt.Sprintf("%s exists but is not a blob", documentPath))}, nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return documentContent{}, err
	}
	if !obs.Available {
		return documentContent{State: StateUnavailable, Evidence: obs.Evidence}, nil
	}
	if len(data) > maxDocumentationBytes {
		return documentContent{State: StateUnknown, Evidence: evidenceWithDetail(obs.Evidence, fmt.Sprintf("%s exceeds %d-byte normalization limit", documentPath, maxDocumentationBytes))}, nil
	}
	return documentContent{State: StateObserved, Data: data, Evidence: obs.Evidence}, nil
}

func contentMatchFact(content documentContent, observed bool) Fact[bool] {
	switch content.State {
	case StateObserved:
		if content.Data == nil {
			return Observed(false, content.Evidence)
		}
		return Observed(observed, content.Evidence)
	case StateUnavailable:
		return Unavailable[bool](content.Evidence)
	default:
		return Unknown[bool](content.Evidence)
	}
}

func markdownHeadings(data []byte) map[string]struct{} {
	result := map[string]struct{}{}
	inFence := false
	fence := ""
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fence = marker
			} else if marker == fence {
				inFence = false
				fence = ""
			}
			continue
		}
		if inFence {
			continue
		}
		candidate := strings.TrimLeft(line, " \t")
		count := 0
		for count < len(candidate) && candidate[count] == '#' {
			count++
		}
		if count == 0 || count > 6 || count >= len(candidate) || candidate[count] != ' ' {
			continue
		}
		heading := normalizeHeading(strings.TrimSpace(candidate[count+1:]))
		if heading != "" {
			result[heading] = struct{}{}
		}
	}
	return result
}

func normalizeHeading(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.TrimRight(value, "#"))
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func markdownRepositoryLinks(readmePath string, data []byte) map[string]struct{} {
	result := map[string]struct{}{}
	for _, match := range inlineMarkdownLink.FindAllStringSubmatch(string(data), -1) {
		if len(match) != 2 {
			continue
		}
		target := strings.TrimSpace(match[1])
		lower := strings.ToLower(target)
		if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(lower, "mailto:") {
			continue
		}
		if colon := strings.IndexByte(target, ':'); colon >= 0 {
			continue
		}
		if index := strings.IndexAny(target, "?#"); index >= 0 {
			target = target[:index]
		}
		if target == "" || strings.HasPrefix(target, "/") {
			continue
		}
		resolved := path.Clean(path.Join(path.Dir(readmePath), target))
		if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") {
			continue
		}
		result[resolved] = struct{}{}
	}
	return result
}

func factForState[T any](state FactState, evidence Evidence) Fact[T] {
	if state == StateUnavailable {
		return Unavailable[T](evidence)
	}
	return Unknown[T](evidence)
}

func evidenceWithDetail(evidence Evidence, detail string) Evidence {
	evidence.Detail = strings.TrimSpace(detail)
	return evidence
}

func sanitizeDocumentationError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.ReplaceAll(err.Error(), "\n", " ")
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 512 {
		message = message[:512] + "…"
	}
	return message
}

func setDocumentationUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.DefaultBranch = Unavailable[string](evidence)
	setDocumentationAfterBranchUnavailable(inventory, evidence)
}

func setDocumentationAfterBranchUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.DefaultCommit = Unavailable[string](evidence)
	setDocumentationAfterTreeUnavailable(inventory, evidence)
}

func setDocumentationAfterTreeUnavailable(inventory *DocumentationInventory, evidence Evidence) {
	inventory.ProfileDeclared = Unavailable[bool](evidence)
	inventory.ProfileName = Unavailable[string](evidence)
	inventory.READMEPresent = Unavailable[bool](evidence)
	inventory.RoutingMetadataPresent = Unavailable[bool](evidence)
	inventory.RoutingMetadataValid = Unavailable[bool](evidence)
}
