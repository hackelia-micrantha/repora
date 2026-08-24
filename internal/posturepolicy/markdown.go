package posturepolicy

import (
	"fmt"
	"sort"
	"strings"
)

func RenderMarkdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Repository posture report\n\n")
	fmt.Fprintf(&b, "- Repository: `%s`\n", report.Repository)
	fmt.Fprintf(&b, "- Profile: `%s`\n\n", report.ProfileID)

	counts := SummaryBySeverity(report)
	b.WriteString("## Findings summary\n\n")
	for _, severity := range []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
		fmt.Fprintf(&b, "- %s: %d\n", severity, counts[severity])
	}
	b.WriteString("\n")

	areas := map[string][]Evaluation{}
	for _, evaluation := range report.Evaluations {
		areas[evaluation.Area] = append(areas[evaluation.Area], evaluation)
	}
	areaNames := make([]string, 0, len(areas))
	for area := range areas {
		areaNames = append(areaNames, area)
	}
	sort.Strings(areaNames)

	b.WriteString("## Policy evaluation\n\n")
	for _, area := range areaNames {
		fmt.Fprintf(&b, "### %s\n\n", area)
		items := areas[area]
		sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
		for _, evaluation := range items {
			fmt.Fprintf(&b, "#### %s — %s\n\n", evaluation.RuleID, evaluation.Title)
			fmt.Fprintf(&b, "- Status: **%s**\n", evaluation.Status)
			fmt.Fprintf(&b, "- Severity: `%s`\n", evaluation.Severity)
			fmt.Fprintf(&b, "- Fact: `%s`\n", evaluation.Fact)
			if len(evaluation.Expected) > 0 {
				fmt.Fprintf(&b, "- Expected: `%s`\n", compactJSON(evaluation.Expected))
			}
			if len(evaluation.Observed) > 0 {
				fmt.Fprintf(&b, "- Observed: `%s`\n", compactJSON(evaluation.Observed))
			}
			if evaluation.Exception != nil {
				fmt.Fprintf(&b, "- Exception: owner `%s`, expires `%s`, reason: %s\n", evaluation.Exception.Owner, evaluation.Exception.Expires, evaluation.Exception.Reason)
			}
			if evaluation.ExceptionGap != "" {
				fmt.Fprintf(&b, "- Exception status: **%s**\n", evaluation.ExceptionGap)
			}
			if len(evaluation.Evidence) > 0 {
				b.WriteString("- Evidence:\n")
				for _, evidence := range evaluation.Evidence {
					fmt.Fprintf(&b, "  - `%s` `%s`", evidence.Source, evidence.Reference)
					if evidence.Detail != "" {
						fmt.Fprintf(&b, ": %s", evidence.Detail)
					}
					b.WriteString("\n")
				}
			}
			if len(evaluation.Remediation) > 0 {
				b.WriteString("- Remediation options:\n")
				for _, remediation := range evaluation.Remediation {
					fmt.Fprintf(&b, "  - %s\n", remediation)
				}
			}
			b.WriteString("\n")
		}
	}

	unsupported := Unsupported(report)
	b.WriteString("## Unknown or unavailable evidence\n\n")
	if len(unsupported) == 0 {
		b.WriteString("None.\n")
	} else {
		for _, evaluation := range unsupported {
			fmt.Fprintf(&b, "- `%s` (%s): `%s` is %s\n", evaluation.RuleID, evaluation.Area, evaluation.Fact, evaluation.Status)
		}
	}

	return b.String()
}

func compactJSON(value []byte) string {
	return strings.Join(strings.Fields(string(value)), " ")
}
