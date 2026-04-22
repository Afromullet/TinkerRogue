#!/usr/bin/env bash
# complexity_report.sh - Complexity hotspots report for TinkerRogue.
#
# Runs three tools against the project and produces a sectioned text report:
#   - gocyclo     (github.com/fzipp/gocyclo)      — cyclomatic complexity
#   - gocognit    (github.com/uudashr/gocognit)   — cognitive complexity
#   - golangci-lint with funlen/nestif/dupl/maintidx
#       — function length, nested conditionals, duplication, maintainability index
#
# Usage:
#   bash tools/scripts/complexity_report.sh [flags]
#
# Flags:
#   --top N            Top-N functions/files to list (default 50)
#   --over N           Threshold for the "over" section (default 15)
#   --include-tests    Include _test.go files
#   --skip-lint        Skip the golangci-lint pass (faster; gocyclo+gocognit only)
#   --output PATH      Output file (default resources/docs/complexity_report.txt)
#   --stdout           Print to stdout instead of writing to file
#   -h, --help         Show this help

set -euo pipefail

TOP=50
OVER=15
INCLUDE_TESTS=0
SKIP_LINT=0
OUTPUT=""
TO_STDOUT=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --top)           TOP="$2"; shift 2 ;;
        --over)          OVER="$2"; shift 2 ;;
        --include-tests) INCLUDE_TESTS=1; shift ;;
        --skip-lint)     SKIP_LINT=1; shift ;;
        --output)        OUTPUT="$2"; shift 2 ;;
        --stdout)        TO_STDOUT=1; shift ;;
        -h|--help)       sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
        *) echo "Unknown flag: $1" >&2; exit 2 ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

if ! command -v gocyclo >/dev/null 2>&1; then
    echo "error: gocyclo not found on PATH" >&2
    echo "install with: go install github.com/fzipp/gocyclo/cmd/gocyclo@latest" >&2
    exit 1
fi
if ! command -v gocognit >/dev/null 2>&1; then
    echo "error: gocognit not found on PATH" >&2
    echo "install with: go install github.com/uudashr/gocognit/cmd/gocognit@latest" >&2
    exit 1
fi
if [[ "$SKIP_LINT" -eq 0 ]] && ! command -v golangci-lint >/dev/null 2>&1; then
    echo "error: golangci-lint not found on PATH" >&2
    echo "install: https://golangci-lint.run/usage/install/  (or rerun with --skip-lint)" >&2
    exit 1
fi

# gocyclo includes test files by default; we filter via -ignore.
# gocognit excludes test files by default; pass -test to include, then filter the same way.
CYCLO_ARGS=()
COGNIT_ARGS=(-test)
if [[ "$INCLUDE_TESTS" -eq 0 ]]; then
    CYCLO_ARGS+=(-ignore '_test\.go')
    COGNIT_ARGS=(-ignore '_test\.go')
fi

if [[ -z "$OUTPUT" ]]; then
    OUTPUT="resources/docs/complexity_report.txt"
fi

TIMESTAMP="$(date '+%Y-%m-%d %H:%M:%S')"
GIT_SHA="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"

# -over 0 lists every function; reuse the output to derive avg + bucket counts.
# Both tools exit 1 when the over-set is non-empty, which is always true for -over 0;
# the `|| true` swallows that.
ALL_CYCLO="$(gocyclo "${CYCLO_ARGS[@]}" -over 0 . || true)"
ALL_COGNIT="$(gocognit "${COGNIT_ARGS[@]}" -over 0 . || true)"

# Run golangci-lint once with complexity-focused linters. We deliberately use
# --no-config (ignore any project .golangci.yml) so results are reproducible
# regardless of repo lint config. Enabled linters:
#   funlen   — function length (statements/lines)
#   nestif   — deeply nested if-statements
#   dupl     — near-duplicate code blocks
#   maintidx — maintainability index (lower = worse)
LINT_OUTPUT=""
if [[ "$SKIP_LINT" -eq 0 ]]; then
    LINT_TESTS_FLAG="--tests=false"
    [[ "$INCLUDE_TESTS" -eq 1 ]] && LINT_TESTS_FLAG="--tests=true"
    LINT_OUTPUT="$(golangci-lint run \
        --no-config \
        --disable-all \
        -E funlen,nestif,dupl,maintidx \
        --issues-exit-code=0 \
        --print-issued-lines=false \
        "$LINT_TESTS_FLAG" \
        --timeout=5m \
        ./... 2>/dev/null || true)"
    # Keep only the one-line findings (file:line:col: msg (linter)); drop noise.
    LINT_OUTPUT="$(echo "$LINT_OUTPUT" | grep -E '\([a-z]+\)[[:space:]]*$' || true)"
fi

# Render one tool's four sections. Args: <title> <tool> <all_output>
render_tool() {
    local title="$1" tool="$2" all="$3"
    local -a args
    if [[ "$tool" == "gocyclo" ]]; then
        args=("${CYCLO_ARGS[@]}")
    else
        args=("${COGNIT_ARGS[@]}")
    fi

    echo "########################################################################"
    echo "#  $title"
    echo "########################################################################"
    echo

    echo "------------------------------------------------------------------------"
    echo "  Average complexity"
    echo "------------------------------------------------------------------------"
    local avg
    avg="$(echo "$all" | awk 'NF { sum += $1; n++ } END { if (n) printf "%.2f (over %d functions)", sum/n, n; else print "n/a" }')"
    echo "  Average: $avg"
    echo

    echo "------------------------------------------------------------------------"
    echo "  Top $TOP most complex functions (hot spots)"
    echo "------------------------------------------------------------------------"
    "$tool" "${args[@]}" -top "$TOP" . || true
    echo

    echo "------------------------------------------------------------------------"
    echo "  All functions with complexity > $OVER"
    echo "------------------------------------------------------------------------"
    local over_out
    over_out="$("$tool" "${args[@]}" -over "$OVER" . || true)"
    if [[ -z "$over_out" ]]; then
        echo "  (none — no function exceeds $OVER)"
    else
        echo "$over_out"
        echo
        echo "  Total functions over $OVER: $(echo "$over_out" | wc -l | tr -d ' ')"
    fi
    echo

    echo "------------------------------------------------------------------------"
    echo "  Distribution by complexity bucket"
    echo "------------------------------------------------------------------------"
    echo "$all" | awk '
        NF == 0 { next }
        {
            c = $1 + 0
            total++
            if      (c <= 5)  b1++
            else if (c <= 10) b2++
            else if (c <= 15) b3++
            else if (c <= 20) b4++
            else              b5++
        }
        END {
            if (total == 0) { print "  (no functions found)"; exit }
            printf "  %-10s %8s %8s\n", "Bucket", "Count", "Percent"
            printf "  %-10s %8s %8s\n", "------", "-----", "-------"
            printf "  %-10s %8d %7.1f%%\n", "1-5",    b1+0, (b1/total)*100
            printf "  %-10s %8d %7.1f%%\n", "6-10",   b2+0, (b2/total)*100
            printf "  %-10s %8d %7.1f%%\n", "11-15",  b3+0, (b3/total)*100
            printf "  %-10s %8d %7.1f%%\n", "16-20",  b4+0, (b4/total)*100
            printf "  %-10s %8d %7.1f%%\n", "21+",    b5+0, (b5/total)*100
            printf "  %-10s %8d\n", "Total", total
        }'
    echo
}

# Render the golangci-lint findings: counts per linter, top-N files by issue
# count, and the full list grouped by linter. Input: $LINT_OUTPUT (one finding
# per line in the form "path:line:col: message (linter)").
render_lint() {
    echo "########################################################################"
    echo "#  PART C — Structural Complexity (golangci-lint: funlen/nestif/dupl/maintidx)"
    echo "########################################################################"
    echo

    if [[ -z "$LINT_OUTPUT" ]]; then
        echo "  (no findings from funlen/nestif/dupl/maintidx)"
        echo
        return
    fi

    local total
    total="$(echo "$LINT_OUTPUT" | wc -l | tr -d ' ')"

    echo "------------------------------------------------------------------------"
    echo "  Findings by linter"
    echo "------------------------------------------------------------------------"
    echo "  Total findings: $total"
    echo
    echo "$LINT_OUTPUT" | awk '
        match($0, /\(([a-z]+)\)[[:space:]]*$/, m) { counts[m[1]]++ }
        END {
            printf "  %-12s %8s\n", "Linter", "Count"
            printf "  %-12s %8s\n", "------", "-----"
            for (k in counts) printf "  %-12s %8d\n", k, counts[k] | "sort -k2 -rn"
        }'
    echo
    echo "  Legend:"
    echo "    funlen    — function too long (>60 lines or >40 statements by default)"
    echo "    nestif    — deeply nested if-blocks (default min complexity 5)"
    echo "    dupl      — near-duplicate code blocks (default threshold 150 tokens)"
    echo "    maintidx  — low maintainability index (composite of cyclomatic + halstead + LOC)"
    echo

    echo "------------------------------------------------------------------------"
    echo "  Top $TOP files by finding count"
    echo "------------------------------------------------------------------------"
    echo "$LINT_OUTPUT" | awk -F: '{ print $1 }' \
        | sort | uniq -c | sort -rn | head -n "$TOP" \
        | awk '{ printf "  %4d  %s\n", $1, $2 }'
    echo

    echo "------------------------------------------------------------------------"
    echo "  All findings (grouped by linter)"
    echo "------------------------------------------------------------------------"
    # Sort so findings group by linter (trailing "(linter)") then by file.
    echo "$LINT_OUTPUT" | awk '
        {
            if (match($0, /\(([a-z]+)\)[[:space:]]*$/, m)) linter = m[1]
            else linter = "other"
            print linter "\t" $0
        }' | sort -k1,1 -k2,2 | awk -F'\t' '
        $1 != prev { if (prev != "") print ""; printf "  [%s]\n", $1; prev = $1 }
        { print "    " $2 }'
    echo
}

render() {
    echo "========================================================================"
    echo "  TinkerRogue Complexity Report (Cyclomatic + Cognitive)"
    echo "========================================================================"
    echo "  Generated: $TIMESTAMP"
    echo "  Branch:    $GIT_BRANCH"
    echo "  Commit:    $GIT_SHA"
    if [[ "$INCLUDE_TESTS" -eq 1 ]]; then
        echo "  Scope:     all .go files (including _test.go)"
    else
        echo "  Scope:     production .go files (_test.go excluded)"
    fi
    if [[ "$SKIP_LINT" -eq 1 ]]; then
        echo "  Tools:     gocyclo (cyclomatic), gocognit (cognitive)  [lint skipped]"
    else
        echo "  Tools:     gocyclo (cyclomatic), gocognit (cognitive), golangci-lint (structural)"
    fi
    echo "  Note:      cyclomatic counts branch paths; cognitive penalizes nesting."
    echo "             A function can be low on one and high on the other."
    echo

    render_tool "PART A — Cyclomatic Complexity (gocyclo)" "gocyclo" "$ALL_CYCLO"
    render_tool "PART B — Cognitive Complexity (gocognit)" "gocognit" "$ALL_COGNIT"
    if [[ "$SKIP_LINT" -eq 0 ]]; then
        render_lint
    fi

    echo "========================================================================"
    echo "  End of report"
    echo "========================================================================"
}

if [[ "$TO_STDOUT" -eq 1 ]]; then
    render
else
    mkdir -p "$(dirname "$OUTPUT")"
    render > "$OUTPUT"
    echo "Wrote complexity report to: $OUTPUT"
fi
