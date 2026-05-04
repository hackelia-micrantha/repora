package plan

import (
	"repoctl/internal/config"
	"repoctl/internal/status"
)

type Output struct {
	Plan []RepoPlan `json:"plan"`
}

type RepoPlan struct {
	ID      string   `json:"id"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	Behind      int    `json:"behind"`
	Destructive bool   `json:"destructive"`
}

func NewOutput(spec config.Spec, results []status.Result, ok []bool) Output {
	out := Output{Plan: make([]RepoPlan, 0, len(spec.Repos))}
	for i, repo := range spec.Repos {
		if !ok[i] {
			continue
		}
		out.Plan = append(out.Plan, NewRepoPlan(repo, results[i]))
	}
	return out
}

func NewRepoPlan(repo config.Repo, result status.Result) RepoPlan {
	repoPlan := RepoPlan{
		ID:      repo.ID,
		Actions: []Action{},
	}
	if result.State == status.StateBehind {
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        "PUSH_MIRROR",
			Target:      repo.Mirrors[0].Provider,
			Behind:      result.Behind,
			Destructive: false,
		})
	}
	return repoPlan
}
