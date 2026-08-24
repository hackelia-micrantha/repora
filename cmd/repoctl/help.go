package main

import (
	"fmt"
	"io"
)

const helpText = `repoctl manages configured Git repository state and read-only repository evidence.

Usage:
  repoctl <command> [options]
  repoctl posture inventory OWNER/REPO
  repoctl posture docs OWNER/REPO
  repoctl posture hooks OWNER/REPO
  repoctl posture commits OWNER/REPO
  repoctl posture mirrors -f repora.yaml
  repoctl posture converge [--inventory FILE] [--docs FILE] [--hooks FILE] [--commits FILE] [--mirrors FILE --repo-uid UID]
  repoctl posture report --profile POLICY.json --facts FACTS.json --as-of YYYY-MM-DD [--format markdown|json]
  repoctl plan-readme -f repora.yaml [--artifact]
  repoctl apply-readme -f repora.yaml --plan-file FILE [--dry-run] [--json]
  repoctl validate-report FILE
  repoctl list-findings FILE
  repoctl generate-scorecard FILE
  repoctl assess FILE
  repoctl --help
  repoctl --version

Commands:
  status   inspect canonical and mirror default-branch state
  plan     show planned mirror updates or export an executable artifact
  apply    apply current observations or an exact plan artifact
  sync     alias for apply
  posture  collect, converge, and evaluate read-only repository posture evidence
  plan-readme  review managed README changes or export the exact managed-artifact plan
  apply-readme  dry-run or journaled exact-plan managed README apply
  validate-report  validate a repository assessment report without mutation
  list-findings    list validated assessment findings without mutation
  generate-scorecard  render validated scorecard dimensions without recalculation
  assess   create a new canonical assessment skeleton without overwriting files
  version  show embedded version and commit metadata
  help     show this help

Options for mirror commands:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --json
        print stabilized command JSON
  --artifact
        with plan, print the exact executable plan artifact as JSON
  --plan-file string
        with apply or sync, execute the exact artifact from this file
  --parallel int
        maximum concurrent repository operations (default 5)
  --continue-on-error
        continue processing repositories after an error
  --dry-run
        show what apply or sync would change without mutation
  --force
        allow destructive overwrites for ahead or diverged mirrors
  --debug
        print debug logs to stderr

Options for posture commands:
  OWNER/REPO
        GitHub repository to inspect with posture inventory, posture docs, posture hooks, or posture commits. Public repositories need no token; private/provider-protected evidence may use GITHUB_TOKEN or GH_TOKEN from the environment.
  posture mirrors -f string
        inspect mirror posture for repositories declared in SCHEMA-0001 YAML (default "repora.yaml")
  posture converge --inventory string
        strict repora.posture-inventory v1 JSON
  posture converge --docs string
        strict repora.posture-documentation v1 JSON
  posture converge --hooks string
        strict repora.posture-hooks v1 JSON
  posture converge --commits string
        strict repora.posture-commits v1 JSON
  posture converge --mirrors string
        strict repora.posture-mirrors v1 JSON; requires --repo-uid
  posture converge --repo-uid string
        repository uid to select from the mirror posture artifact
  posture report --profile string
        strict repora.posture-policy-profile v1 JSON; policy is external to repository-controlled observation profiles
  posture report --facts string
        strict repora.posture-policy-inputs v1 JSON containing normalized facts only
  posture report --as-of string
        explicit YYYY-MM-DD evaluation date used for deterministic exception-expiry handling
  posture report --format string
        markdown (default) or json

Posture inventory is GET-only. Provider fields unavailable under current access are emitted as unavailable facts rather than false negatives. It does not evaluate policy, create findings, run scanners, or mutate repository/provider state.

Posture docs is also GET-only. It observes document presence, configured README sections and links, exact content markers, and document-routing trust metadata. A target repository may declare observation targets in .repora/posture-documentation.yaml. The profile selects facts to observe; it does not assign severity or authorize remediation.

Posture hooks is GET-only. It observes common/custom hook configuration, optional .repora/posture-hooks.yaml expectations, required local-check coverage in GitHub Actions, bootstrap/bypass documentation, and bounded static network-load signals. It never installs or executes target-repository hook code, and CI remains the enforcement authority.

Posture commits is GET-only. It observes an explicitly bounded default-branch history window, commit signature verification state, merge shape, change size/file scope, configured sensitive-path matches, and optional commit-to-PR association. Repository-owned thresholds are observation parameters only; direct-push/unreviewed status, tag signatures, and release boundaries remain unknown unless evidence can prove them. It performs no productivity scoring, identity profiling, blame, intent inference, or history mutation.

Posture mirrors reuses the existing mirror reconciliation cache/status semantics. It may create or refresh Repora's local mirror cache and configured cache remotes, but it does not push, synchronize mirrors, publish releases, or mutate provider settings. It observes declared canonical/mirror identity, default-branch and commit drift, and preserves unavailable provider metadata. Tag and release drift remain explicit unknown facts in v1.

Posture converge is offline-only. It strictly validates supplied versioned collector artifacts, rejects duplicate source flags and repository-identity mismatches, preserves observed/unknown/unavailable states through the typed adapters, and emits deterministic repora.posture-policy-inputs v1 JSON. It does not re-scan repositories or contact providers.

Posture report is offline-only. It consumes normalized fact inputs and an external policy profile, preserves unknown/unavailable evidence, evaluates explicit expected-vs-observed rules and exceptions, and emits deterministic Markdown or JSON. It does not contact providers, re-scan repositories, mutate state, or calculate an opaque numeric score.

Options for plan-readme:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --artifact
        print the exact repora.io/managed-artifact-plan v1 JSON instead of human review output

Options for apply-readme:
  -f string
        path to SCHEMA-0001 YAML config (default "repora.yaml")
  --plan-file string
        exact repora.io/managed-artifact-plan v1 JSON to preflight or apply
  --dry-run
        revalidate current canonical state and show the reviewed diff without mutation
  --json
        on real apply, print repora.io/managed-artifact-apply-result v1 JSON

A real apply persists managed-artifact INTENT before candidate creation, creates verified local candidate commits, performs fresh stale preflight, pushes with an exact reviewed-base lease, and persists RESULT evidence. It does not use --force.

Global options:
  --version
        show embedded version and commit metadata
  -h, --help
        show this help
`

func isHelpRequest(args []string) bool {
	if len(args) != 1 {
		return false
	}

	switch args[0] {
	case "help", "-h", "--help":
		return true
	default:
		return false
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, helpText)
}
