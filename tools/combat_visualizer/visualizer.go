package main

import (
	"fmt"
	"strings"
	"time"
)

// Visualize generates a complete ASCII visualization of a battle record.
func Visualize(record *BattleRecord) string {
	var sb strings.Builder

	// Battle header
	sb.WriteString(renderBattleHeader(record))
	sb.WriteString("\n")

	// Render each engagement
	if len(record.Engagements) == 0 {
		sb.WriteString("(No engagements recorded)\n\n")
	} else {
		for _, eng := range record.Engagements {
			sb.WriteString(RenderEngagement(eng))
			sb.WriteString("\n")
		}
	}

	// Battle footer
	sb.WriteString(renderBattleFooter())

	return sb.String()
}

// renderBattleHeader creates the battle header with metadata.
func renderBattleHeader(record *BattleRecord) string {
	var sb strings.Builder

	// Top separator
	sb.WriteString("═══════════════════════════════════════════════════════\n")

	// Battle ID
	sb.WriteString(fmt.Sprintf("BATTLE: %s\n", record.BattleID))

	// Timestamps
	startTime := record.StartTime.Format("2006-01-02 15:04:05")
	endTime := record.EndTime.Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf("Started: %s\n", startTime))
	sb.WriteString(fmt.Sprintf("Ended:   %s\n", endTime))

	// Duration
	duration := record.EndTime.Sub(record.StartTime)
	sb.WriteString(fmt.Sprintf("Duration: %s\n", formatDuration(duration)))

	// Rounds
	sb.WriteString(fmt.Sprintf("Rounds: %d\n", record.FinalRound))

	// Victor
	if record.VictorName != "" {
		sb.WriteString(fmt.Sprintf("Victor: %s\n", record.VictorName))
	} else {
		sb.WriteString("Victor: Unknown\n")
	}

	// Bottom separator
	sb.WriteString("═══════════════════════════════════════════════════════\n")

	return sb.String()
}

// renderBattleFooter creates the battle footer.
func renderBattleFooter() string {
	var sb strings.Builder

	sb.WriteString("════════════════════════════════════════════════════\n")
	sb.WriteString("BATTLE END\n")
	sb.WriteString("════════════════════════════════════════════════════\n")

	return sb.String()
}

// formatDuration formats a duration as human-readable string.
// Examples: "5s", "1m 30s", "2h 15m 30s"
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	var parts []string

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}
