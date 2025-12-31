package battlelog

import (
	"fmt"
	"game_main/tactical/squads"
	"sort"
	"strings"

	"github.com/bytearena/ecs"
)

// GenerateEngagementSummary creates per-unit summaries from CombatLog.
// Aggregates AttackEvents by attacker to produce high-level action summaries.
func GenerateEngagementSummary(log *squads.CombatLog) *EngagementSummary {
	if log == nil {
		return &EngagementSummary{
			AttackerSummaries: []UnitActionSummary{},
			DefenderSummaries: []UnitActionSummary{},
		}
	}

	// Build attacker summaries
	attackerSummaries := make([]UnitActionSummary, 0, len(log.AttackingUnits))
	for _, unitSnapshot := range log.AttackingUnits {
		summary := buildUnitSummary(unitSnapshot.UnitID, unitSnapshot, log.AttackEvents)
		attackerSummaries = append(attackerSummaries, summary)
	}

	// Build defender summaries from counterattack events
	defenderSummaries := make([]UnitActionSummary, 0, len(log.DefendingUnits))
	for _, unitSnapshot := range log.DefendingUnits {
		// Filter counterattack events for this defender
		counterEvents := filterCounterattackEventsByAttacker(unitSnapshot.UnitID, log.AttackEvents)

		// Only create summary if this unit counterattacked
		if len(counterEvents) > 0 {
			summary := buildUnitSummary(unitSnapshot.UnitID, unitSnapshot, counterEvents)
			defenderSummaries = append(defenderSummaries, summary)
		}
	}

	return &EngagementSummary{
		AttackerSummaries: attackerSummaries,
		DefenderSummaries: defenderSummaries,
	}
}

// buildUnitSummary aggregates AttackEvents for a specific attacker.
func buildUnitSummary(unitID ecs.EntityID, unitSnapshot squads.UnitSnapshot, events []squads.AttackEvent) UnitActionSummary {
	// Filter events for this attacker
	unitEvents := filterEventsByAttacker(unitID, events)

	// Initialize summary
	summary := UnitActionSummary{
		UnitID:   unitID,
		UnitName: unitSnapshot.UnitName,
		Role:     unitSnapshot.RoleName,
		GridPos: GridPosition{
			Row: unitSnapshot.GridRow,
			Col: unitSnapshot.GridCol,
		},
		TargetedRows:    []int{},
		TargetedColumns: []int{},
		TargetedCells:   []GridPosition{},
		UnitsEngaged:    []UnitEngagement{},
	}

	// Early return if no events
	if len(unitEvents) == 0 {
		summary.Summary = fmt.Sprintf("%s did not attack", unitSnapshot.UnitName)
		return summary
	}

	// Aggregate targeting data
	summary.TargetedRows = uniqueRows(unitEvents)
	summary.TargetedColumns = uniqueCols(unitEvents)
	summary.TargetedCells = uniqueCells(unitEvents)
	summary.TargetMode = getTargetMode(unitEvents)

	// Build per-target engagement list
	engagementMap := make(map[ecs.EntityID]*UnitEngagement)
	for _, event := range unitEvents {
		if existing, ok := engagementMap[event.DefenderID]; ok {
			// Update existing engagement (multiple attacks on same target)
			existing.DamageDealt += event.FinalDamage
			if event.WasKilled {
				existing.WasKilled = true
			}
			// Keep highest outcome priority (CRITICAL > HIT > DODGE > MISS)
			if outcomeRank(event.HitResult.Type) > outcomeRank(parseOutcome(existing.Outcome)) {
				existing.Outcome = formatOutcome(event.HitResult.Type)
			}
		} else {
			// New engagement
			engagementMap[event.DefenderID] = &UnitEngagement{
				TargetID:    event.DefenderID,
				TargetName:  getTargetName(event, events),
				Outcome:     formatOutcome(event.HitResult.Type),
				DamageDealt: event.FinalDamage,
				WasKilled:   event.WasKilled,
			}
		}
	}

	// Convert map to sorted slice
	summary.UnitsEngaged = make([]UnitEngagement, 0, len(engagementMap))
	for _, eng := range engagementMap {
		summary.UnitsEngaged = append(summary.UnitsEngaged, *eng)
	}

	// Count outcomes
	for _, event := range unitEvents {
		summary.TotalAttacks++
		summary.TotalDamage += event.FinalDamage

		if event.WasKilled {
			summary.UnitsKilled++
		}

		switch event.HitResult.Type {
		case squads.HitTypeNormal:
			summary.Hits++
		case squads.HitTypeCritical:
			summary.Criticals++
			summary.Hits++ // Criticals also count as hits
		case squads.HitTypeCounterattack:
			summary.Hits++ // Counterattacks count as hits
		case squads.HitTypeMiss:
			summary.Misses++
		case squads.HitTypeDodge:
			summary.Dodges++
		}
	}

	// Generate human-readable summary
	summary.Summary = generateSummaryText(&summary)

	return summary
}

// filterEventsByAttacker returns events where AttackerID matches.
func filterEventsByAttacker(unitID ecs.EntityID, events []squads.AttackEvent) []squads.AttackEvent {
	filtered := []squads.AttackEvent{}
	for _, event := range events {
		if event.AttackerID == unitID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// filterCounterattackEventsByAttacker returns counterattack events where AttackerID matches.
// Only returns events with IsCounterattack == true
func filterCounterattackEventsByAttacker(unitID ecs.EntityID, events []squads.AttackEvent) []squads.AttackEvent {
	filtered := []squads.AttackEvent{}
	for _, event := range events {
		if event.AttackerID == unitID && event.IsCounterattack {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// uniqueRows extracts unique TargetRow values.
func uniqueRows(events []squads.AttackEvent) []int {
	rowSet := make(map[int]bool)
	for _, event := range events {
		rowSet[event.TargetInfo.TargetRow] = true
	}

	rows := make([]int, 0, len(rowSet))
	for row := range rowSet {
		rows = append(rows, row)
	}
	sort.Ints(rows)
	return rows
}

// uniqueCols extracts unique TargetCol values.
func uniqueCols(events []squads.AttackEvent) []int {
	colSet := make(map[int]bool)
	for _, event := range events {
		colSet[event.TargetInfo.TargetCol] = true
	}

	cols := make([]int, 0, len(colSet))
	for col := range colSet {
		cols = append(cols, col)
	}
	sort.Ints(cols)
	return cols
}

// uniqueCells extracts unique (row, col) pairs for Magic attacks.
func uniqueCells(events []squads.AttackEvent) []GridPosition {
	cellSet := make(map[GridPosition]bool)
	for _, event := range events {
		if event.TargetInfo.TargetMode == "Magic" {
			cell := GridPosition{
				Row: event.TargetInfo.TargetRow,
				Col: event.TargetInfo.TargetCol,
			}
			cellSet[cell] = true
		}
	}

	cells := make([]GridPosition, 0, len(cellSet))
	for cell := range cellSet {
		cells = append(cells, cell)
	}
	return cells
}

// getTargetMode returns the primary attack mode (most common).
func getTargetMode(events []squads.AttackEvent) string {
	if len(events) == 0 {
		return ""
	}
	// Simplified: just return first event's mode
	return events[0].TargetInfo.TargetMode
}

// getTargetName looks up the defender's name from events.
// This is a helper to extract the name from the first matching event.
func getTargetName(event squads.AttackEvent, allEvents []squads.AttackEvent) string {
	// In a real implementation, you'd look this up from entity manager
	// For now, we use a simple placeholder based on DefenderID
	return fmt.Sprintf("Unit_%d", event.DefenderID)
}

// formatOutcome converts HitType to string.
func formatOutcome(hitType squads.HitType) string {
	return hitType.String()
}

// parseOutcome converts string back to HitType for ranking.
func parseOutcome(outcome string) squads.HitType {
	switch outcome {
	case "MISS":
		return squads.HitTypeMiss
	case "DODGE":
		return squads.HitTypeDodge
	case "HIT":
		return squads.HitTypeNormal
	case "CRITICAL":
		return squads.HitTypeCritical
	case "COUNTERATTACK":
		return squads.HitTypeCounterattack
	default:
		return squads.HitTypeMiss
	}
}

// outcomeRank returns priority rank for outcome (higher = better).
func outcomeRank(hitType squads.HitType) int {
	switch hitType {
	case squads.HitTypeCritical:
		return 3
	case squads.HitTypeCounterattack:
		return 2
	case squads.HitTypeNormal:
		return 2
	case squads.HitTypeDodge:
		return 1
	case squads.HitTypeMiss:
		return 0
	default:
		return 0
	}
}

// generateSummaryText creates a human-readable summary.
func generateSummaryText(summary *UnitActionSummary) string {
	if summary.TotalAttacks == 0 {
		return fmt.Sprintf("%s did not attack", summary.UnitName)
	}

	var parts []string

	// Targeting info
	if len(summary.TargetedRows) > 0 {
		rowsStr := formatIntSlice(summary.TargetedRows)
		parts = append(parts, fmt.Sprintf("attacked rows %s", rowsStr))
	} else if len(summary.TargetedColumns) > 0 {
		colsStr := formatIntSlice(summary.TargetedColumns)
		parts = append(parts, fmt.Sprintf("attacked columns %s", colsStr))
	} else if len(summary.TargetedCells) > 0 {
		parts = append(parts, "attacked specific cells")
	}

	// Per-target details
	targetDetails := []string{}
	for _, eng := range summary.UnitsEngaged {
		detail := ""
		if eng.WasKilled {
			detail = fmt.Sprintf("%s %s for %d (killed)", eng.Outcome, eng.TargetName, eng.DamageDealt)
		} else if eng.DamageDealt > 0 {
			detail = fmt.Sprintf("%s %s for %d", eng.Outcome, eng.TargetName, eng.DamageDealt)
		} else {
			detail = fmt.Sprintf("%s %s", eng.Outcome, eng.TargetName)
		}
		targetDetails = append(targetDetails, detail)
	}

	if len(targetDetails) > 0 {
		parts = append(parts, strings.Join(targetDetails, ", "))
	}

	// Outcome summary
	outcomeParts := []string{}
	if summary.Hits > 0 {
		outcomeParts = append(outcomeParts, fmt.Sprintf("%d hit", summary.Hits))
	}
	if summary.Criticals > 0 {
		outcomeParts = append(outcomeParts, fmt.Sprintf("%d critical", summary.Criticals))
	}
	if summary.Misses > 0 {
		outcomeParts = append(outcomeParts, fmt.Sprintf("%d miss", summary.Misses))
	}
	if summary.Dodges > 0 {
		outcomeParts = append(outcomeParts, fmt.Sprintf("%d dodge", summary.Dodges))
	}

	if len(outcomeParts) > 0 {
		parts = append(parts, strings.Join(outcomeParts, ", "))
	}

	// Total damage
	if summary.TotalDamage > 0 {
		parts = append(parts, fmt.Sprintf("%d total damage", summary.TotalDamage))
	}

	return fmt.Sprintf("%s: %s", summary.UnitName, strings.Join(parts, "; "))
}

// formatIntSlice formats an integer slice as comma-separated string.
func formatIntSlice(nums []int) string {
	strs := make([]string, len(nums))
	for i, n := range nums {
		strs[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(strs, ",")
}
