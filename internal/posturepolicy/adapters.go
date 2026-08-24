package posturepolicy

import (
	"encoding/json"
	"fmt"

	"repoctl/internal/posture"
)

func AddInventory(inputs *Inputs, inventory posture.Inventory) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	if err := requireRepository(inputs, inventory.Repository.FullName); err != nil {
		return err
	}
	facts := inventory.RepositoryFacts
	entries := map[string]FactInput{}
	addConverted(entries, "repository.default_branch", facts.DefaultBranch)
	addConverted(entries, "repository.default_branch_protected", facts.DefaultBranchProtected)
	addConverted(entries, "repository.required_status_checks", facts.RequiredStatusChecks)
	addConverted(entries, "repository.required_reviews", facts.RequiredReviews)
	addConverted(entries, "repository.force_push_protected", facts.ForcePushProtected)
	addConverted(entries, "repository.deletion_protected", facts.DeletionProtected)
	addConverted(entries, "repository.codeowners_present", facts.CODEOWNERSPresent)
	addConverted(entries, "repository.security_md_present", facts.SecurityMDPresent)
	addConverted(entries, "repository.license_present", facts.LicensePresent)
	addConverted(entries, "repository.issue_template_present", facts.IssueTemplatePresent)
	addConverted(entries, "repository.pull_request_template_present", facts.PullRequestTemplatePresent)
	addConverted(entries, "repository.dependency_update_automation", facts.DependencyAutomation)
	addConverted(entries, "repository.workflow_paths", facts.WorkflowPaths)
	entries["ci.workflows_state"] = stateInput(inventory.WorkflowsState, string(inventory.WorkflowsState), inventory.Evidence)
	for _, workflow := range inventory.Workflows {
		prefix := "ci.workflow." + workflow.Path
		entries[prefix+".state"] = stateInput(workflow.State, string(workflow.State), workflow.Evidence)
		entries[prefix+".uses_pull_request_target"] = stateInput(workflow.State, workflow.UsesPullRequestTarget, workflow.Evidence)
		entries[prefix+".permissions_declared"] = stateInput(workflow.State, workflow.Permissions.Declared, workflow.Evidence)
		entries[prefix+".permissions_default"] = stateInput(workflow.State, workflow.Permissions.Default, workflow.Evidence)
		for _, job := range workflow.Jobs {
			jobPrefix := prefix + ".job." + job.Name
			addConverted(entries, jobPrefix+".self_hosted_label", job.SelfHostedLabel)
			entries[jobPrefix+".runs_on"] = stateInput(workflow.State, job.RunsOn, workflow.Evidence)
			entries[jobPrefix+".permissions_declared"] = stateInput(workflow.State, job.Permissions.Declared, workflow.Evidence)
			for _, action := range job.Actions {
				actionPrefix := jobPrefix + ".action." + action.Uses
				entries[actionPrefix+".pinning"] = stateInput(workflow.State, action.Pinning, workflow.Evidence)
				entries[actionPrefix+".third_party"] = stateInput(workflow.State, action.ThirdParty, workflow.Evidence)
			}
		}
	}
	return addEntries(inputs, entries)
}

func AddDocumentation(inputs *Inputs, inventory posture.DocumentationInventory) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	if err := requireRepository(inputs, inventory.Repository.FullName); err != nil {
		return err
	}
	entries := map[string]FactInput{}
	addConverted(entries, "documentation.default_branch", inventory.DefaultBranch)
	addConverted(entries, "documentation.default_commit", inventory.DefaultCommit)
	addConverted(entries, "documentation.profile_declared", inventory.ProfileDeclared)
	addConverted(entries, "documentation.profile_name", inventory.ProfileName)
	addConverted(entries, "documentation.readme_present", inventory.READMEPresent)
	addConverted(entries, "documentation.routing_metadata_present", inventory.RoutingMetadataPresent)
	addConverted(entries, "documentation.routing_trust_metadata_usable", inventory.RoutingTrustMetadataUsable)
	for _, document := range inventory.Documents {
		prefix := "documentation.document." + document.Path
		addConverted(entries, prefix+".present", document.Present)
		addConverted(entries, prefix+".trust_tier", document.TrustTier)
	}
	for _, section := range inventory.READMESections {
		addConverted(entries, "documentation.readme_section."+section.Section+".present", section.Present)
	}
	for _, link := range inventory.READMELinks {
		addConverted(entries, "documentation.readme_link."+link.Target+".present", link.Present)
	}
	for _, marker := range inventory.ContentMarkers {
		addConverted(entries, "documentation.content_marker."+marker.ID+".present", marker.Present)
	}
	return addEntries(inputs, entries)
}

func AddHooks(inputs *Inputs, inventory posture.HooksInventory) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	if err := requireRepository(inputs, inventory.Repository.FullName); err != nil {
		return err
	}
	entries := map[string]FactInput{}
	addConverted(entries, "hooks.default_branch", inventory.DefaultBranch)
	addConverted(entries, "hooks.default_commit", inventory.DefaultCommit)
	addConverted(entries, "hooks.profile_declared", inventory.ProfileDeclared)
	addConverted(entries, "hooks.manager", inventory.Manager)
	addConverted(entries, "hooks.bootstrap_instructions_present", inventory.BootstrapPresent)
	addConverted(entries, "hooks.bypass_documentation_present", inventory.BypassPresent)
	for _, entrypoint := range inventory.Entrypoints {
		prefix := "hooks.entrypoint." + entrypoint.Path
		addConverted(entries, prefix+".configured", entrypoint.Configured)
		addConverted(entries, prefix+".executable", entrypoint.Executable)
		addConverted(entries, prefix+".network_loaded", entrypoint.NetworkLoaded)
	}
	for _, check := range inventory.RequiredChecks {
		prefix := "hooks.required_check." + check.Name
		addConverted(entries, prefix+".configured", check.Configured)
		addConverted(entries, prefix+".ci_covered", check.CICovered)
	}
	return addEntries(inputs, entries)
}

func AddCommits(inputs *Inputs, inventory posture.CommitInventory) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	if err := requireRepository(inputs, inventory.Repository.FullName); err != nil {
		return err
	}
	entries := map[string]FactInput{}
	addConverted(entries, "commits.default_branch", inventory.DefaultBranch)
	addConverted(entries, "commits.default_commit", inventory.DefaultCommit)
	addConverted(entries, "commits.profile_declared", inventory.ProfileDeclared)
	addConverted(entries, "commits.history_limit", inventory.HistoryLimit)
	addConverted(entries, "commits.history_truncated", inventory.HistoryTruncated)
	addConverted(entries, "commits.file_count_threshold", inventory.FileCountThreshold)
	addConverted(entries, "commits.changed_lines_threshold", inventory.ChangedLinesThreshold)
	addConverted(entries, "commits.sensitive_path_patterns", inventory.SensitivePathPatterns)
	addConverted(entries, "commits.signed_tag_count", inventory.SignedTagCount)
	addConverted(entries, "commits.unsigned_tag_count", inventory.UnsignedTagCount)
	addConverted(entries, "commits.release_boundary_change_count", inventory.ReleaseBoundaryChangeCount)
	entries["commits.observed_count"] = observedInput(len(inventory.Commits), inventory.Evidence)
	for _, commit := range inventory.Commits {
		prefix := "commits.commit." + commit.SHA
		addConverted(entries, prefix+".merge_commit", commit.MergeCommit)
		addConverted(entries, prefix+".signature_verification", commit.SignatureVerification)
		addConverted(entries, prefix+".changed_lines", commit.ChangedLines)
		addConverted(entries, prefix+".observed_file_count", commit.ObservedFileCount)
		addConverted(entries, prefix+".files_complete", commit.FilesComplete)
		addConverted(entries, prefix+".file_count_threshold_exceeded", commit.FileCountThresholdExceeded)
		addConverted(entries, prefix+".changed_lines_threshold_exceeded", commit.ChangedLinesThresholdExceeded)
		addConverted(entries, prefix+".sensitive_paths_changed", commit.SensitivePathsChanged)
		addConverted(entries, prefix+".associated_pull_requests", commit.AssociatedPullRequests)
		addConverted(entries, prefix+".direct_to_default_branch", commit.DirectToDefaultBranch)
		addConverted(entries, prefix+".unreviewed_change", commit.UnreviewedChange)
	}
	return addEntries(inputs, entries)
}

func AddMirrors(inputs *Inputs, inventory posture.MirrorInventory, repoUID string) error {
	if err := inventory.Validate(); err != nil {
		return err
	}
	var selected *posture.MirrorRepositoryFacts
	for idx := range inventory.Repos {
		if inventory.Repos[idx].UID != repoUID {
			continue
		}
		if selected != nil {
			return fmt.Errorf("mirror posture contains duplicate repository uid %q", repoUID)
		}
		selected = &inventory.Repos[idx]
	}
	if selected == nil {
		return fmt.Errorf("mirror posture repository uid %q was not found", repoUID)
	}
	return addMirrorRepository(inputs, *selected)
}

func addMirrorRepository(inputs *Inputs, repository posture.MirrorRepositoryFacts) error {
	entries := map[string]FactInput{}
	entries["mirrors.repo_id"] = observedInput(repository.ID, repository.Evidence)
	entries["mirrors.repo_uid"] = observedInput(repository.UID, repository.Evidence)
	addConverted(entries, "mirrors.mode", repository.Mode)
	addConverted(entries, "mirrors.direction", repository.Direction)
	addConverted(entries, "mirrors.canonical.default_branch", repository.Canonical.DefaultBranch)
	addConverted(entries, "mirrors.canonical.commit", repository.Canonical.Commit)
	addConverted(entries, "mirrors.canonical.visibility", repository.Canonical.Visibility)
	addConverted(entries, "mirrors.canonical.current_actor_push_permission", repository.Canonical.CurrentActorPushPermission)
	entries["mirrors.target_count"] = observedInput(len(repository.Mirrors), repository.Evidence)
	for _, mirror := range repository.Mirrors {
		prefix := "mirrors.target." + mirror.Identity.Provider + ":" + mirror.Identity.Path
		addConverted(entries, prefix+".cache_remote", mirror.CacheRemote)
		addConverted(entries, prefix+".default_branch", mirror.DefaultBranch)
		addConverted(entries, prefix+".default_branch_drift", mirror.DefaultBranchDrift)
		addConverted(entries, prefix+".commit", mirror.Commit)
		addConverted(entries, prefix+".divergence", mirror.Divergence)
		addConverted(entries, prefix+".ahead", mirror.Ahead)
		addConverted(entries, prefix+".behind", mirror.Behind)
		addConverted(entries, prefix+".visibility", mirror.Visibility)
		addConverted(entries, prefix+".current_actor_push_permission", mirror.CurrentActorPushPermission)
		addConverted(entries, prefix+".tag_drift", mirror.TagDrift)
		addConverted(entries, prefix+".release_drift", mirror.ReleaseDrift)
	}
	return addEntries(inputs, entries)
}

func requireRepository(inputs *Inputs, repository string) error {
	if inputs == nil {
		return fmt.Errorf("posture policy inputs are required")
	}
	if inputs.Kind == "" && inputs.Version == 0 && inputs.Repository == "" && inputs.Facts == nil {
		*inputs = NewInputs(repository)
	}
	if inputs.Repository != repository {
		return fmt.Errorf("posture artifact repository %q does not match policy inputs repository %q", repository, inputs.Repository)
	}
	return nil
}

func addEntries(inputs *Inputs, entries map[string]FactInput) error {
	if inputs == nil {
		return fmt.Errorf("posture policy inputs are required")
	}
	for name, fact := range entries {
		if inputs.Facts != nil {
			if _, exists := inputs.Facts[name]; exists {
				return fmt.Errorf("posture policy fact %q already exists", name)
			}
		}
		if err := validateFactInput(name, fact); err != nil {
			return err
		}
	}
	if inputs.Facts == nil {
		inputs.Facts = map[string]FactInput{}
	}
	for name, fact := range entries {
		inputs.Facts[name] = fact
	}
	return nil
}

func addConverted[T any](entries map[string]FactInput, name string, fact posture.Fact[T]) {
	entries[name] = convertedFact(fact)
}

func convertedFact[T any](fact posture.Fact[T]) FactInput {
	out := FactInput{State: fact.State, Evidence: append([]posture.Evidence(nil), fact.Evidence...)}
	if fact.State == posture.StateObserved && fact.Value != nil {
		raw, err := json.Marshal(*fact.Value)
		if err == nil {
			out.Value = raw
		}
	}
	return out
}

func observedInput(value any, evidence []posture.Evidence) FactInput {
	raw, _ := json.Marshal(value)
	return FactInput{State: posture.StateObserved, Value: raw, Evidence: append([]posture.Evidence(nil), evidence...)}
}

func stateInput(state posture.FactState, value any, evidence []posture.Evidence) FactInput {
	if state == posture.StateObserved {
		return observedInput(value, evidence)
	}
	return FactInput{State: state, Evidence: append([]posture.Evidence(nil), evidence...)}
}
