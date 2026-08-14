package posture

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var actionSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func parseWorkflow(fullName, path string, data []byte, evidence Evidence) (Workflow, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return Workflow{}, fmt.Errorf("parse workflow YAML: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return Workflow{}, fmt.Errorf("workflow root must be a mapping")
	}
	root := document.Content[0]
	workflow := Workflow{
		Path:                  path,
		State:                 StateObserved,
		Permissions:           parsePermissions(mappingValue(root, "permissions")),
		UsesPullRequestTarget: eventPresent(mappingValue(root, "on"), "pull_request_target"),
		Jobs:                  []WorkflowJob{},
		Evidence:              []Evidence{evidence},
	}

	jobs := mappingValue(root, "jobs")
	if jobs == nil {
		return workflow, nil
	}
	if jobs.Kind != yaml.MappingNode {
		return Workflow{}, fmt.Errorf("workflow jobs must be a mapping")
	}

	type namedNode struct {
		name string
		node *yaml.Node
	}
	ordered := make([]namedNode, 0, len(jobs.Content)/2)
	for i := 0; i+1 < len(jobs.Content); i += 2 {
		if jobs.Content[i].Kind != yaml.ScalarNode || jobs.Content[i+1].Kind != yaml.MappingNode {
			continue
		}
		ordered = append(ordered, namedNode{name: jobs.Content[i].Value, node: jobs.Content[i+1]})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].name < ordered[j].name })

	for _, item := range ordered {
		workflow.Jobs = append(workflow.Jobs, parseWorkflowJob(fullName, path, item.name, item.node, evidence))
	}
	return workflow, nil
}

func parseWorkflowJob(fullName, path, name string, node *yaml.Node, evidence Evidence) WorkflowJob {
	runsOn := scalarStrings(mappingValue(node, "runs-on"))
	selfHosted := classifySelfHosted(runsOn, evidence)
	job := WorkflowJob{
		Name:        name,
		Permissions: parsePermissions(mappingValue(node, "permissions")),
		RunsOn:      runsOn,
		SelfHosted:  selfHosted,
		Actions:     []ActionReference{},
	}

	if uses := scalarValue(mappingValue(node, "uses")); uses != "" {
		job.Actions = append(job.Actions, classifyAction(fullName, uses))
	}
	steps := mappingValue(node, "steps")
	if steps != nil && steps.Kind == yaml.SequenceNode {
		for _, step := range steps.Content {
			if step.Kind != yaml.MappingNode {
				continue
			}
			if uses := scalarValue(mappingValue(step, "uses")); uses != "" {
				job.Actions = append(job.Actions, classifyAction(fullName, uses))
			}
	}
	return job
}

func parsePermissions(node *yaml.Node) Permissions {
	permissions := Permissions{Scopes: []PermissionScope{}}
	if node == nil {
		return permissions
	}
	permissions.Declared = true
	switch node.Kind {
	case yaml.ScalarNode:
		permissions.Default = strings.TrimSpace(node.Value)
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := scalarValue(node.Content[i])
			value := scalarValue(node.Content[i+1])
			if key != "" {
				permissions.Scopes = append(permissions.Scopes, PermissionScope{Scope: key, Access: value})
			}
		}
		sort.Slice(permissions.Scopes, func(i, j int) bool { return permissions.Scopes[i].Scope < permissions.Scopes[j].Scope })
	default:
		permissions.Default = "unknown"
	}
	return permissions
}

func eventPresent(node *yaml.Node, event string) bool {
	if node == nil {
		return false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value == event
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if scalarValue(child) == event {
				return true
			}
		}
	case yaml.MappingNode:
		return mappingValue(node, event) != nil
	}
	return false
}

func scalarStrings(node *yaml.Node) []string {
	if node == nil {
		return []string{}
	}
	values := []string{}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Value != "" {
			values = append(values, node.Value)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if value := scalarValue(child); value != "" {
				values = append(values, value)
			}
		}
	}
	return values
}

func classifySelfHosted(runsOn []string, evidence Evidence) Fact[bool] {
	if len(runsOn) == 0 {
		return Unknown[bool](evidence)
	}
	dynamic := false
	for _, label := range runsOn {
		trimmed := strings.TrimSpace(label)
		if strings.EqualFold(trimmed, "self-hosted") {
			return Observed(true, evidence)
		}
		if strings.Contains(trimmed, "${{") {
			dynamic = true
		}
	}
	if dynamic {
		return Unknown[bool](evidence)
	}
	return Observed(false, evidence)
}

func classifyAction(fullName, uses string) ActionReference {
	ref := ActionReference{Uses: uses, Pinning: "unversioned"}
	if strings.HasPrefix(uses, "./") {
		ref.Pinning = "local"
		return ref
	}
	if strings.HasPrefix(uses, "docker://") {
		ref.ThirdParty = true
		if strings.Contains(uses, "@sha256:") {
			ref.Pinning = "immutable-digest"
		} else {
			ref.Pinning = "mutable-ref"
		}
		return ref
	}

	at := strings.LastIndex(uses, "@")
	pathPart := uses
	versionPart := ""
	if at >= 0 {
		pathPart = uses[:at]
		versionPart = uses[at+1:]
	}
	segments := strings.Split(pathPart, "/")
	if len(segments) >= 2 {
		owner := strings.ToLower(segments[0])
		repository := strings.ToLower(segments[0] + "/" + segments[1])
		ref.ThirdParty = owner != "actions" && owner != "github" && repository != strings.ToLower(fullName)
	} else {
		ref.ThirdParty = true
	}
	if versionPart == "" {
		ref.Pinning = "unversioned"
	} else if actionSHA.MatchString(versionPart) {
		ref.Pinning = "immutable-sha"
	} else {
		ref.Pinning = "mutable-ref"
	}
	return ref
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if scalarValue(node.Content[i]) == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}
