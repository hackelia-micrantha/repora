package posturepolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"
)

func RenderMarkdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Repository posture report\n\n")
	fmt.Fprintf(&b, "- Repository: %s\n", inlineCode(report.Repository))
	fmt.Fprintf(&b, "- Profile: %s\n", inlineCode(report.ProfileID))
	fmt.Fprintf(&b, "- As of: %s\n\n", inlineCode(report.AsOf))

	counts := SummaryBySeverity(report)
	b.WriteString("## Findings summary\n\n")
	for _, severity := range []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
		fmt.Fprintf(&b, "- %s: %d\n", escapeMarkdownText(string(severity)), counts[severity])
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
		fmt.Fprintf(&b, "### %s\n\n", escapeMarkdownText(area))
		items := areas[area]
		sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
		for _, evaluation := range items {
			fmt.Fprintf(&b, "#### %s — %s\n\n", escapeMarkdownText(evaluation.RuleID), escapeMarkdownText(evaluation.Title))
			fmt.Fprintf(&b, "- Status: **%s**\n", escapeMarkdownText(string(evaluation.Status)))
			fmt.Fprintf(&b, "- Severity: %s\n", inlineCode(string(evaluation.Severity)))
			fmt.Fprintf(&b, "- Fact: %s\n", inlineCode(evaluation.Fact))
			if len(evaluation.Expected) > 0 {
				fmt.Fprintf(&b, "- Expected: %s\n", inlineCode(compactJSON(evaluation.Expected)))
			}
			if len(evaluation.Observed) > 0 {
				fmt.Fprintf(&b, "- Observed: %s\n", inlineCode(compactJSON(evaluation.Observed)))
			}
			if evaluation.Exception != nil {
				fmt.Fprintf(&b, "- Exception: owner %s, expires %s, reason: %s\n", inlineCode(evaluation.Exception.Owner), inlineCode(evaluation.Exception.Expires), escapeMarkdownText(evaluation.Exception.Reason))
			}
			if evaluation.ExceptionGap != "" {
				fmt.Fprintf(&b, "- Exception status: **%s**\n", escapeMarkdownText(evaluation.ExceptionGap))
			}
			if len(evaluation.Evidence) > 0 {
				b.WriteString("- Evidence:\n")
				for _, evidence := range evaluation.Evidence {
					fmt.Fprintf(&b, "  - %s %s", inlineCode(evidence.Source), inlineCode(evidence.Reference))
					if evidence.Detail != "" {
						fmt.Fprintf(&b, ": %s", escapeMarkdownText(evidence.Detail))
					}
					b.WriteString("\n")
				}
			}
			if len(evaluation.Remediation) > 0 {
				b.WriteString("- Remediation options:\n")
				for _, remediation := range evaluation.Remediation {
					fmt.Fprintf(&b, "  - %s\n", escapeMarkdownText(remediation))
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
			fmt.Fprintf(&b, "- %s (%s): %s is %s\n", inlineCode(evaluation.RuleID), escapeMarkdownText(evaluation.Area), inlineCode(evaluation.Fact), escapeMarkdownText(string(evaluation.Status)))
		}
	}

	return b.String()
}

func compactJSON(value []byte) string {
	var out bytes.Buffer
	if err := json.Compact(&out, value); err != nil {
		return string(value)
	}
	return out.String()
}

func inlineCode(value string) string {
	value = strings.NewReplacer("\r", `\r`, "\n", `\n`, "\t", `\t`).Replace(value)
	maxRun := 0
	currentRun := 0
	for _, r := range value {
		if r == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
		} else {
			currentRun = 0
		}
	}
	delimiter := strings.Repeat("`", maxRun+1)
	return delimiter + " " + value + " " + delimiter
}

func escapeMarkdownText(value string) string {
	value = html.EscapeString(value)
	value = strings.NewReplacer(
		"\\", `\\`,
		"\r", `\r`,
		"\n", `\n`,
		"\t", `\t`,
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"!", "\\!",
		"|", "\\|",
		">", "\\>",
	).Replace(value)
	return value
}
