#!/usr/bin/env bash
# view-usage.sh - Display Claude Code agent/skill usage statistics

set -euo pipefail

STATS_FILE="$(dirname "$0")/../.claude/usage-stats.txt"

if [ ! -f "$STATS_FILE" ] || [ ! -s "$STATS_FILE" ]; then
    echo "No usage data yet."
    echo "Stats will appear after agents or skills are used."
    exit 0
fi

echo "=== Claude Code Usage Stats ==="
echo ""

# Agents section
AGENTS=$(grep "^agent:" "$STATS_FILE" 2>/dev/null | sort -t' ' -k2 -rn || true)
if [ -n "$AGENTS" ]; then
    echo "-- Agents --"
    AGENT_TOTAL=0
    while IFS=' ' read -r key count; do
        name="${key#agent:}"
        printf "  %-25s %d\n" "$name" "$count"
        AGENT_TOTAL=$((AGENT_TOTAL + count))
    done <<< "$AGENTS"
    echo "  -------------------------"
    printf "  %-25s %d\n" "TOTAL" "$AGENT_TOTAL"
    echo ""
fi

# Skills section
SKILLS=$(grep "^skill:" "$STATS_FILE" 2>/dev/null | sort -t' ' -k2 -rn || true)
if [ -n "$SKILLS" ]; then
    echo "-- Skills --"
    SKILL_TOTAL=0
    while IFS=' ' read -r key count; do
        name="${key#skill:}"
        printf "  %-25s %d\n" "$name" "$count"
        SKILL_TOTAL=$((SKILL_TOTAL + count))
    done <<< "$SKILLS"
    echo "  -------------------------"
    printf "  %-25s %d\n" "TOTAL" "$SKILL_TOTAL"
    echo ""
fi

# Grand total
ALL_TOTAL=$(awk '{sum += $2} END {print sum}' "$STATS_FILE")
echo "=== Grand Total: ${ALL_TOTAL} ==="
