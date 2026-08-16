package posture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type documentContent struct {
	State    FactState
	Data     []byte
	Evidence Evidence
}

type trustRule struct {
	Tier     string
	Patterns []string
	Index    int
}

type trustRouter struct {
	Rules []trustRule
}

func CollectGitHubDocumentation(ctx context.Context, reader GitHubReader, fullName string) (DocumentationInventory, error) {
	if _, _, err := splitGitHubFullName(fullName); err != nil {
		return DocumentationInventory{}, err
	}
	inventory := newDocumentationInventory(fullName)
	repository, repoObs, err := reader.Repository(ctx, fullName)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, repoObs.Evidence)
	if !repoObs.Available {
		setDocumentationUnavailable(&inventory, repoObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultBranch = Observed(repository.DefaultBranch, repoObs.Evidence)

	branch, branchObs, err := reader.Branch(ctx, fullName, repository.DefaultBranch)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, branchObs.Evidence)
	if !branchObs.Available {
		setDocumentationAfterBranchUnavailable(&inventory, branchObs.Evidence)
		return inventory, inventory.Validate()
	}
	inventory.DefaultCommit = Observed(branch.CommitSHA, branchObs.Evidence)

	tree, treeObs, err := reader.Tree(ctx, fullName, branch.TreeSHA)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.Evidence = append(inventory.Evidence, treeObs.Evidence)
	if !treeObs.Available {
		setDocumentationAfterTreeUnavailable(&inventory, treeObs.Evidence)
		return inventory, inventory.Validate()
	}
	entries := make(map[string]GitHubTreeEntry, len(tree.Entries))
	for _, entry := range tree.Entries {
		entries[entry.Path] = entry
	}

	profile, profileEvidence, profileState, declared, err := loadDocumentationProfile(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.ProfileDeclared = declared
	if profileState == StateObserved {
		inventory.ProfileName = Observed(profile.Name, profileEvidence)
		inventory.READMEPath = profile.README.Path
	} else {
		inventory.ProfileName = factForState[string](profileState, profileEvidence)
	}

	router, routerState, routerEvidence, routerPresent, routerUsable, err := loadTrustRouter(ctx, reader, fullName, tree, entries, treeObs.Evidence)
	if err != nil {
		return DocumentationInventory{}, err
	}
	inventory.RoutingMetadataPresent = routerPresent
	inventory.RoutingTrustMetadataUsable = routerUsable
	if routerEvidence.Source != "" {
		inventory.Evidence = append(inventory.Evidence, routerEvidence)
	}

	if profileState != StateObserved {
		inventory.READMEPresent = factForState[bool](profileState, profileEvidence)
		return inventory, inventory.Validate()
	}

	contentCache := map[string]documentContent{}
	loadContent := func(documentPath string) (documentContent, error) {
		if cached, ok := contentCache[documentPath]; ok {
			return cached, nil
		}
		content, err := loadDocumentationContent(ctx, reader, fullName, documentPath, tree, entries, treeObs.Evidence)
		if err != nil {
			return documentContent{}, err
		}
		contentCache[documentPath] = content
		return content, nil
	}

	documentPaths := sortedUnique(append(append([]string{}, profile.Documents...), profile.README.Path))
	for _, documentPath := range documentPaths {
		present := documentationPresence(documentPath, tree, entries, treeObs.Evidence)
		trust := classifyTrustFact(documentPath, router, routerState, routerEvidence)
		inventory.Documents = append(inventory.Documents, DocumentationDocumentFact{Path: documentPath, Present: present, TrustTier: trust})
		if documentPath == profile.README.Path {
			inventory.READMEPresent = present
		}
	}
	if inventory.READMEPresent.State == "" {
		inventory.READMEPresent = documentationPresence(profile.README.Path, tree, entries, treeObs.Evidence)
	}

	readmeContent, err := loadContent(profile.README.Path)
	if err != nil {
		return DocumentationInventory{}, err
	}
	headings := map[string]struct{}{}
	links := map[string]struct{}{}
	if readmeContent.State == StateObserved && readmeContent.Data != nil {
		headings = markdownHeadings(readmeContent.Data)
		links = markdownRepositoryLinks(profile.README.Path, readmeContent.Data)
	}
	for _, section := range profile.README.Sections {
		_, found := headings[normalizeHeading(section)]
		inventory.READMESections = append(inventory.READMESections, DocumentationSectionFact{
			Section: section,
			Present: contentMatchFact(readmeContent, found),
		})
	}
	for _, target := range profile.README.Links {
		_, found := links[target]
		inventory.READMELinks = append(inventory.READMELinks, DocumentationLinkFact{
			Target:  target,
			Present: contentMatchFact(readmeContent, found),
		})
	}
	for _, marker := range profile.ContentMarkers {
		content, err := loadContent(marker.Path)
		if err != nil {
			return DocumentationInventory{}, err
		}
		digest := sha256.Sum256([]byte(marker.Contains))
		inventory.ContentMarkers = append(inventory.ContentMarkers, DocumentationMarkerFact{
			ID:             marker.ID,
			Path:           marker.Path,
			ExpectedSHA256: hex.EncodeToString(digest[:]),
			Present:        contentMatchFact(content, content.Data != nil && strings.Contains(string(content.Data), marker.Contains)),
		})
	}
	return inventory, inventory.Validate()
}

func loadDocumentationProfile(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (DocumentationProfile, Evidence, FactState, Fact[bool], error) {
	entry, exists := entries[documentationProfilePath]
	if !exists {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; documentation profile presence is unknown")
			return DocumentationProfile{}, evidence, StateUnknown, Unknown[bool](evidence), nil
		}
		profile := DefaultDocumentationProfile()
		evidence := Evidence{Source: "repora.builtin", Reference: "documentation-profile:baseline"}
		return profile, evidence, StateObserved, Observed(false, treeEvidence), nil
	}
	if entry.Type != "blob" {
		evidence := evidenceWithDetail(treeEvidence, "documentation profile path exists but is not a blob")
		return DocumentationProfile{}, evidence, StateUnknown, Observed(true, treeEvidence), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return DocumentationProfile{}, Evidence{}, "", Fact[bool]{}, err
	}
	if !obs.Available {
		return DocumentationProfile{}, obs.Evidence, StateUnavailable, Observed(true, treeEvidence), nil
	}
	profile, err := ParseDocumentationProfile(data)
	if err != nil {
		evidence := evidenceWithDetail(obs.Evidence, "declared documentation posture profile is malformed or unsupported")
		return DocumentationProfile{}, evidence, StateUnknown, Observed(true, treeEvidence), nil
	}
	return profile, obs.Evidence, StateObserved, Observed(true, treeEvidence), nil
}

func loadTrustRouter(ctx context.Context, reader GitHubReader, fullName string, tree GitHubTree, entries map[string]GitHubTreeEntry, treeEvidence Evidence) (trustRouter, FactState, Evidence, Fact[bool], Fact[bool], error) {
	entry, exists := entries[documentRouterPath]
	if !exists {
		if tree.Truncated {
			evidence := evidenceWithDetail(treeEvidence, "Git tree is truncated; document routing trust metadata presence is unknown")
			return trustRouter{}, StateUnknown, evidence, Unknown[bool](evidence), Unknown[bool](evidence), nil
		}
		evidence := evidenceWithDetail(treeEvidence, "document router is not present")
		return trustRouter{}, StateUnknown, evidence, Observed(false, treeEvidence), Unknown[bool](evidence), nil
	}
	if entry.Type != "blob" {
		evidence := evidenceWithDetail(treeEvidence, "document router path exists but is not a blob")
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	data, obs, err := reader.Blob(ctx, fullName, entry.SHA)
	if err != nil {
		return trustRouter{}, "", Evidence{}, Fact[bool]{}, Fact[bool]{}, err
	}
	if !obs.Available {
		return trustRouter{}, StateUnavailable, obs.Evidence, Observed(true, treeEvidence), Unavailable[bool](obs.Evidence), nil
	}
	if len(data) > maxDocumentationBytes {
		evidence := evidenceWithDetail(obs.Evidence, fmt.Sprintf("document routing trust metadata exceeds %d-byte normalization limit", maxDocumentationBytes))
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	router, err := parseTrustRouter(data)
	if err != nil {
		evidence := evidenceWithDetail(obs.Evidence, "document routing trust metadata is malformed or unsupported")
		return trustRouter{}, StateUnknown, evidence, Observed(true, treeEvidence), Observed(false, evidence), nil
	}
	return router, StateObserved, obs.Evidence, Observed(true, treeEvidence), Observed(true, obs.Evidence), nil
}
