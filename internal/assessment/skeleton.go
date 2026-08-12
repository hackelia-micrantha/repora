package assessment

// NewSkeleton returns the canonical v1 repository-assessment template.
func NewSkeleton() Report {
	dirty := false
	score := 0
	createdAt := "1970-01-01T00:00:00Z"

	return Report{
		Kind:    ReportKind,
		Version: ReportVersion,
		ID:      "replace-me",
		Title:   "Repository assessment",
		Scope:   []string{"quality"},
		Summary: "TODO: summarize the assessment scope and outcome.",
		Snapshot: Snapshot{
			Kind:    "repora.repository-snapshot",
			Version: 1,
			Repository: Repository{
				FullName: "owner/repository",
			},
			Revision: Revision{
				Commit: "0000000",
				Dirty:  &dirty,
			},
			CapturedAt: createdAt,
		},
		Findings: []Finding{},
		Evidence: []Evidence{},
		Scorecard: Scorecard{
			Kind:    "repora.scorecard",
			Version: 1,
			Dimensions: []Dimension{
				{
					Name:        "documentation",
					Score:       &score,
					Rationale:   "TODO: replace this placeholder dimension or provide evidence-backed rationale.",
					EvidenceIDs: []string{},
				},
			},
		},
		Metadata: &Metadata{
			CreatedBy: "template",
			CreatedAt: &createdAt,
			Notes:     "Template only. Replace placeholder snapshot, scope, scorecard, findings, and evidence before treating this as an assessment.",
		},
	}
}
