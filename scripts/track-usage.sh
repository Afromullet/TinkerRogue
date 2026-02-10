#!/usr/bin/env bash
# track-usage.sh - Claude Code hook script for tracking agent/skill usage
# Receives hook JSON on stdin, increments counters in .claude/usage-stats.txt

set -euo pipefail

STATS_FILE="$(dirname "$0")/../.claude/usage-stats.txt"

# Read hook JSON from stdin
INPUT=$(cat)

# Extract hook_event_name using grep (works for flat JSON)
EVENT=$(echo "$INPUT" | grep -oP '"hook_event_name"\s*:\s*"\K[^"]+' || echo "")

KEY=""

if [ "$EVENT" = "SubagentStop" ]; then
    # Try agent_type first, then subagent_type
    AGENT=$(echo "$INPUT" | grep -oP '"agent_type"\s*:\s*"\K[^"]+' || echo "")
    if [ -z "$AGENT" ]; then
        AGENT=$(echo "$INPUT" | grep -oP '"subagent_type"\s*:\s*"\K[^"]+' || echo "")
    fi
    if [ -n "$AGENT" ]; then
        KEY="agent:${AGENT}"
    fi

elif [ "$EVENT" = "PostToolUse" ]; then
    # Extract skill name from tool_input
    SKILL=$(echo "$INPUT" | grep -oP '"skill"\s*:\s*"\K[^"]+' || echo "")
    if [ -n "$SKILL" ]; then
        KEY="skill:${SKILL}"
    fi
fi

# If we got a valid key, increment the counter
if [ -n "$KEY" ]; then
    # Create stats file if it doesn't exist
    touch "$STATS_FILE"

    # Escape special regex chars in key for grep/sed
    ESCAPED_KEY=$(echo "$KEY" | sed 's/[.[\*^$()+?{|\\]/\\&/g')

    if grep -q "^${ESCAPED_KEY} " "$STATS_FILE" 2>/dev/null; then
        # Increment existing counter
        CURRENT=$(grep "^${ESCAPED_KEY} " "$STATS_FILE" | awk '{print $2}')
        NEW=$((CURRENT + 1))
        sed -i "s/^${ESCAPED_KEY} ${CURRENT}$/${ESCAPED_KEY} ${NEW}/" "$STATS_FILE"
    else
        # Add new entry
        echo "${KEY} 1" >> "$STATS_FILE"
    fi
fi
