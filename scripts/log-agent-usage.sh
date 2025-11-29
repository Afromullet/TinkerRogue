#!/bin/bash
# Agent Usage Tracking Hook for Claude Code
# Logs SubagentStop events to CSV for analytics

# CSV file location - relative to project root
CSV_FILE="analysis/agent-usage.csv"
DEBUG_LOG="analysis/hook-debug.log"

# Ensure analysis directory exists
mkdir -p "$(dirname "$CSV_FILE")"

# Generate timestamp (ISO 8601 format)
TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S")

# Read JSON input from stdin
INPUT_JSON=$(cat)

# Debug logging
echo "=== SubagentStop Hook Debug $(date) ===" >> "$DEBUG_LOG"
echo "Input JSON: $INPUT_JSON" >> "$DEBUG_LOG"

# Extract transcript path from JSON using jq (if available) or grep/sed fallback
if command -v jq &> /dev/null; then
    TRANSCRIPT_PATH=$(echo "$INPUT_JSON" | jq -r '.transcript_path // empty')
else
    # Fallback parsing without jq
    TRANSCRIPT_PATH=$(echo "$INPUT_JSON" | grep -o '"transcript_path":"[^"]*"' | sed 's/"transcript_path":"\([^"]*\)"/\1/')
fi

echo "Transcript path: $TRANSCRIPT_PATH" >> "$DEBUG_LOG"

# Initialize CSV with header if it doesn't exist
if [ ! -f "$CSV_FILE" ]; then
    echo "Timestamp,Agent Name,Model,Duration,Session ID" > "$CSV_FILE"
fi

# Parse transcript to extract agent metadata
AGENT_NAME="unknown"
MODEL="unknown"
DURATION=0
SESSION_ID="unknown"

if [ -n "$TRANSCRIPT_PATH" ] && [ -f "$TRANSCRIPT_PATH" ]; then
    echo "Parsing transcript file..." >> "$DEBUG_LOG"

    # Extract session ID from JSON input
    if command -v jq &> /dev/null; then
        SESSION_ID=$(echo "$INPUT_JSON" | jq -r '.session_id // "unknown"')
    else
        SESSION_ID=$(echo "$INPUT_JSON" | grep -o '"session_id":"[^"]*"' | sed 's/"session_id":"\([^"]*\)"/\1/')
    fi

    # Read last few lines of transcript to find agent metadata
    # Look for Task tool invocations with subagent_type
    LAST_LINES=$(tail -n 50 "$TRANSCRIPT_PATH" 2>/dev/null || echo "")

    if command -v jq &> /dev/null; then
        # Extract subagent_type from last Task tool use
        AGENT_NAME=$(echo "$LAST_LINES" | jq -s '
            [.[] | select(.type == "tool_use" and .name == "Task") | .input.subagent_type]
            | last // "unknown"' 2>/dev/null || echo "unknown")

        # Extract model if available
        MODEL=$(echo "$LAST_LINES" | jq -s '
            [.[] | select(.type == "tool_use" and .name == "Task") | .input.model]
            | last // "unknown"' 2>/dev/null || echo "unknown")
    else
        # Fallback: grep for subagent_type
        AGENT_NAME=$(echo "$LAST_LINES" | grep -o '"subagent_type":"[^"]*"' | tail -1 | sed 's/"subagent_type":"\([^"]*\)"/\1/' || echo "unknown")
    fi

    echo "Extracted agent name: $AGENT_NAME" >> "$DEBUG_LOG"
    echo "Extracted model: $MODEL" >> "$DEBUG_LOG"
    echo "Session ID: $SESSION_ID" >> "$DEBUG_LOG"
else
    echo "Transcript file not found or path empty" >> "$DEBUG_LOG"
fi

# Append agent execution data to CSV
echo "\"${TIMESTAMP}\",\"${AGENT_NAME}\",\"${MODEL}\",${DURATION},\"${SESSION_ID}\"" >> "$CSV_FILE"

echo "Logged to CSV successfully" >> "$DEBUG_LOG"
echo "" >> "$DEBUG_LOG"

# Exit successfully
exit 0
