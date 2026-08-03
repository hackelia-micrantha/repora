package apply

import "repoctl/internal/status"

const DetailedOutputVersion = 3

type DetailedOutput struct {
	Kind    string           `json:"kind"`
	Version int              `json:"version"`
	Results []DetailedResult `json:"results"`
}

type DetailedResult struct {
	ID      string             `json:"id"`
	UID     string             `json:"uid"`
	State   status.State       `json:"state"`
	Applied bool               `json:"applied"`
	DryRun  bool               `json:"dry_run"`
	Actions []DetailedAction   `json:"actions"`
	Journal *JournalReferences `json:"journal,omitempty"`
	Error   string             `json:"error,omitempty"`
}

type DetailedAction struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Force   bool   `json:"force"`
	Before  string `json:"before"`
	Desired string `json:"desired"`
	After   string `json:"after,omitempty"`
	Outcome string `json:"outcome"`
	Error   string `json:"error,omitempty"`
}

func NewDetailedOutput(results []DetailedResult) DetailedOutput {
	if results == nil {
		results = []DetailedResult{}
	}
	return DetailedOutput{Kind: OutputKind, Version: DetailedOutputVersion, Results: results}
}
