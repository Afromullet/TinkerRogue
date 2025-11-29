#!/bin/bash
# Agent Usage Tracking Hook for Claude Code
# Logs SubagentStop events to CSV for analytics

# Exit on error (but catch errors to ensure clean exit)
set -e

# CSV file location
CSV_FILE="${CLAUDE_PROJECT_DIR}/.claude/agent-usage.csv"

# Ensure .claude directory exists
mkdir -p "$(dirname "$CSV_FILE")"

# Generate timestamp (ISO 8601 format)
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S")

# Extract agent metadata from environment variables
# Note: Exact variable names may differ - these are best guesses
AGENT_NAME="${CLAUDE_AGENT_NAME:-${SUBAGENT_TYPE:-unknown}}"
STATUS="${CLAUDE_AGENT_STATUS:-${AGENT_STATUS:-complete}}"
DURATION="${CLAUDE_AGENT_DURATION:-${AGENT_DURATION:-0}}"

# Optional: Debug logging (uncomment to see available environment variables)
# DEBUG_LOG="${CLAUDE_PROJECT_DIR}/.claude/hook-debug.log"
# echo "=== SubagentStop Hook Debug $(date) ===" >> "$DEBUG_LOG"
# env | grep -E "(CLAUDE|AGENT|SUBAGENT)" >> "$DEBUG_LOG" 2>/dev/null || true
# echo "" >> "$DEBUG_LOG"

# Initialize CSV with header if it doesn't exist
if [ ! -f "$CSV_FILE" ]; then
    echo "Timestamp,Agent Name,Status,Duration" > "$CSV_FILE"
fi

# Append agent execution data to CSV
# Use double quotes for fields that might contain commas or spaces
echo "\"${TIMESTAMP}\",\"${AGENT_NAME}\",\"${STATUS}\",${DURATION}" >> "$CSV_FILE"

# Exit successfully (don't block agent execution even if logging fails)
exit 0
