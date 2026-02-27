package main

import (
	"fmt"
	"strings"
)

// RenderEngagement renders a complete engagement with grids and attack flow.
func RenderEngagement(eng EngagementRecord) string {
	var sb strings.Builder

	// Engagement header
	sb.WriteString(renderEngagementHeader(eng))
	sb.WriteString("\n")

	// Attacker formation grid
	sb.WriteString("ATTACKER FORMATION:\n")
	sb.WriteString(RenderSquadGrid(eng.CombatLog.AttackingUnits,
		eng.CombatLog.AttackerSquadName, "A"))
	sb.WriteString("\n")

	// Defender formation grid
	sb.WriteString("DEFENDER FORMATION:\n")
	sb.WriteString(RenderSquadGrid(eng.CombatLog.DefendingUnits,
		eng.CombatLog.DefenderSquadName, "D"))
	sb.WriteString("\n")

	// Attack flow
	sb.WriteString("ATTACK FLOW:\n")
	sb.WriteString("────────────────────────────────────────────────────\n")

	if eng.Summary != nil && len(eng.Summary.AttackerSummaries) > 0 {
		for _, summary := range eng.Summary.AttackerSummaries {
			sb.WriteString(renderUnitSummary(summary, "A"))
		}
	} else {
		sb.WriteString("  (No attacks recorded)\n")
	}

	sb.WriteString("────────────────────────────────────────────────────\n")
	sb.WriteString("\n")

	// Heal flow (only when heal events exist)
	if eng.CombatLog != nil && len(eng.CombatLog.HealEvents) > 0 {
		sb.WriteString("HEAL FLOW:\n")
		sb.WriteString("────────────────────────────────────────────────────\n")
		sb.WriteString(renderHealSummary(eng))
		sb.WriteString("────────────────────────────────────────────────────\n")
		sb.WriteString("\n")
	}

	// Counterattack flow
	if eng.Summary != nil && len(eng.Summary.DefenderSummaries) > 0 {
		sb.WriteString("COUNTERATTACK FLOW:\n")
		sb.WriteString("────────────────────────────────────────────────────\n")

		for _, summary := range eng.Summary.DefenderSummaries {
			sb.WriteString(renderUnitSummary(summary, "D"))
		}

		sb.WriteString("────────────────────────────────────────────────────\n")
		sb.WriteString("\n")
	}

	// Engagement totals
	sb.WriteString(renderEngagementTotals(eng.Summary))
	sb.WriteString("════════════════════════════════════════════════════\n")

	return sb.String()
}

// renderEngagementHeader creates a boxed header for an engagement.
func renderEngagementHeader(eng EngagementRecord) string {
	var sb strings.Builder

	// Header line
	title := fmt.Sprintf("ENGAGEMENT #%d (Round %d)", eng.Index, eng.Round)

	// Squad info line
	attackerName := eng.CombatLog.AttackerSquadName
	defenderName := eng.CombatLog.DefenderSquadName
	distance := eng.CombatLog.SquadDistance

	squadInfo := fmt.Sprintf("%s → %s (%d tiles)",
		attackerName, defenderName, distance)

	// Calculate box width (max of title and squad info, minimum 50)
	width := len(title)
	if len(squadInfo) > width {
		width = len(squadInfo)
	}
	if width < 50 {
		width = 50
	}

	// Top border
	sb.WriteString("┌─── ")
	sb.WriteString(title)
	sb.WriteString(" ")
	sb.WriteString(strings.Repeat("─", width-len(title)-1))
	sb.WriteString("┐\n")

	// Content line
	sb.WriteString("│ ")
	sb.WriteString(squadInfo)
	sb.WriteString(strings.Repeat(" ", width-len(squadInfo)))
	sb.WriteString("│\n")

	// Bottom border
	sb.WriteString("└")
	sb.WriteString(strings.Repeat("─", width+4))
	sb.WriteString("┘\n")

	return sb.String()
}

// renderUnitSummary renders attack flow for a single unit.
func renderUnitSummary(summary UnitActionSummary, prefix string) string {
	var sb strings.Builder

	// Unit header: [ID] Name (Role) at (row,col)
	roleInfo := ""
	if summary.Role != "" {
		roleInfo = fmt.Sprintf(" (%s)", summary.Role)
	}

	sb.WriteString(fmt.Sprintf("[%s%d] %s%s at (%d,%d)\n",
		prefix, summary.UnitID, summary.UnitName, roleInfo,
		summary.GridPos.Row, summary.GridPos.Col))

	// Targeting info
	if len(summary.TargetedRows) > 0 {
		sb.WriteString(fmt.Sprintf("  Targeted rows: %s\n",
			formatIntSlice(summary.TargetedRows)))
	} else if len(summary.TargetedColumns) > 0 {
		sb.WriteString(fmt.Sprintf("  Targeted columns: %s\n",
			formatIntSlice(summary.TargetedColumns)))
	} else if len(summary.TargetedCells) > 0 {
		sb.WriteString("  Targeted cells: ")
		sb.WriteString(formatCellSlice(summary.TargetedCells))
		sb.WriteString("\n")
	}

	// Per-target engagement details
	if len(summary.UnitsEngaged) > 0 {
		for i, eng := range summary.UnitsEngaged {
			// Choose tree character based on position
			prefix := "├─"
			if i == len(summary.UnitsEngaged)-1 {
				prefix = "└─"
			}

			// Format engagement line
			engLine := fmt.Sprintf("  %s %s %s",
				prefix, eng.Outcome, eng.TargetName)

			if eng.DamageDealt > 0 {
				engLine += fmt.Sprintf(" for %d damage", eng.DamageDealt)
			}

			if eng.WasKilled {
				engLine += " (killed)"
			}

			sb.WriteString(engLine)
			sb.WriteString("\n")
		}
	}

	// Summary line
	sb.WriteString(fmt.Sprintf("  ➤ %d attack", summary.TotalAttacks))
	if summary.TotalAttacks != 1 {
		sb.WriteString("s")
	}
	sb.WriteString(": ")

	// Outcome breakdown
	outcomes := []string{}
	if summary.Hits > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d hit", summary.Hits))
		if summary.Hits > 1 {
			outcomes[len(outcomes)-1] += "s"
		}
	}
	if summary.Misses > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d miss", summary.Misses))
		if summary.Misses > 1 {
			outcomes[len(outcomes)-1] += "es"
		}
	}
	if summary.Dodges > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d dodge", summary.Dodges))
		if summary.Dodges > 1 {
			outcomes[len(outcomes)-1] += "s"
		}
	}
	if summary.Criticals > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d critical", summary.Criticals))
		if summary.Criticals > 1 {
			outcomes[len(outcomes)-1] += "s"
		}
	}

	if len(outcomes) > 0 {
		sb.WriteString(strings.Join(outcomes, ", "))
	} else {
		sb.WriteString("no outcomes")
	}

	if summary.TotalDamage > 0 {
		sb.WriteString(fmt.Sprintf(", %d damage", summary.TotalDamage))
	}

	if summary.TotalHealing > 0 {
		sb.WriteString(fmt.Sprintf(", %d healing (%d units)", summary.TotalHealing, summary.UnitsHealed))
	}

	sb.WriteString("\n\n")

	return sb.String()
}

// renderHealSummary renders heal flow grouped by healer.
func renderHealSummary(eng EngagementRecord) string {
	if eng.CombatLog == nil || len(eng.CombatLog.HealEvents) == 0 {
		return ""
	}

	// Build name lookup from both sides
	nameMap := make(map[int64]string)
	roleMap := make(map[int64]string)
	posMap := make(map[int64][2]int)
	for _, u := range eng.CombatLog.AttackingUnits {
		nameMap[u.UnitID] = u.UnitName
		roleMap[u.UnitID] = u.RoleName
		posMap[u.UnitID] = [2]int{u.GridRow, u.GridCol}
	}
	for _, u := range eng.CombatLog.DefendingUnits {
		nameMap[u.UnitID] = u.UnitName
		roleMap[u.UnitID] = u.RoleName
		posMap[u.UnitID] = [2]int{u.GridRow, u.GridCol}
	}

	// Group heals by healer
	type healGroup struct {
		healerID int64
		events   []HealEvent
	}
	healerOrder := []int64{}
	healerMap := make(map[int64]*healGroup)
	for _, h := range eng.CombatLog.HealEvents {
		if _, exists := healerMap[h.HealerID]; !exists {
			healerMap[h.HealerID] = &healGroup{healerID: h.HealerID}
			healerOrder = append(healerOrder, h.HealerID)
		}
		healerMap[h.HealerID].events = append(healerMap[h.HealerID].events, h)
	}

	var sb strings.Builder
	for _, hid := range healerOrder {
		group := healerMap[hid]
		healerName := nameMap[hid]
		if healerName == "" {
			healerName = fmt.Sprintf("Unit_%d", hid)
		}
		role := roleMap[hid]
		pos := posMap[hid]

		roleInfo := ""
		if role != "" {
			roleInfo = fmt.Sprintf(" (%s)", role)
		}

		sb.WriteString(fmt.Sprintf("[A%d] %s%s at (%d,%d)\n", hid, healerName, roleInfo, pos[0], pos[1]))

		totalHealing := 0
		for i, h := range group.events {
			targetName := nameMap[h.TargetID]
			if targetName == "" {
				targetName = fmt.Sprintf("Unit_%d", h.TargetID)
			}

			prefix := "├─"
			if i == len(group.events)-1 {
				prefix = "└─"
			}

			sb.WriteString(fmt.Sprintf("  %s HEAL %s for %d HP (%d→%d)\n",
				prefix, targetName, h.HealAmount, h.TargetHPBefore, h.TargetHPAfter))
			totalHealing += h.HealAmount
		}

		healCount := len(group.events)
		sb.WriteString(fmt.Sprintf("  ➤ %d heal", healCount))
		if healCount != 1 {
			sb.WriteString("s")
		}
		sb.WriteString(fmt.Sprintf(": %d total healing\n\n", totalHealing))
	}

	return sb.String()
}

// renderEngagementTotals renders aggregate statistics for the engagement.
func renderEngagementTotals(summary *EngagementSummary) string {
	if summary == nil || len(summary.AttackerSummaries) == 0 {
		return "Engagement Summary:\n  (No data)\n"
	}

	// Aggregate totals across all attackers
	totalAttacks := 0
	totalHits := 0
	totalMisses := 0
	totalDodges := 0
	totalCriticals := 0
	totalDamage := 0
	totalKills := 0
	totalHealing := 0

	for _, s := range summary.AttackerSummaries {
		totalAttacks += s.TotalAttacks
		totalHits += s.Hits
		totalMisses += s.Misses
		totalDodges += s.Dodges
		totalCriticals += s.Criticals
		totalDamage += s.TotalDamage
		totalKills += s.UnitsKilled
		totalHealing += s.TotalHealing
	}

	// Aggregate totals for counterattacks
	counterAttacks := 0
	counterHits := 0
	counterMisses := 0
	counterDodges := 0
	counterCriticals := 0
	counterDamage := 0
	counterKills := 0

	for _, s := range summary.DefenderSummaries {
		counterAttacks += s.TotalAttacks
		counterHits += s.Hits
		counterMisses += s.Misses
		counterDodges += s.Dodges
		counterCriticals += s.Criticals
		counterDamage += s.TotalDamage
		counterKills += s.UnitsKilled
	}

	var sb strings.Builder
	sb.WriteString("Engagement Summary:\n")
	sb.WriteString(fmt.Sprintf("  Attacks: %d\n", totalAttacks))

	// Outcomes line
	sb.WriteString("  Outcomes: ")
	outcomes := []string{}
	if totalHits > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d hits", totalHits))
	}
	if totalMisses > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d misses", totalMisses))
	}
	if totalDodges > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d dodges", totalDodges))
	}
	if totalCriticals > 0 {
		outcomes = append(outcomes, fmt.Sprintf("%d criticals", totalCriticals))
	}
	sb.WriteString(strings.Join(outcomes, ", "))
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Total Damage: %d\n", totalDamage))
	if totalHealing > 0 {
		sb.WriteString(fmt.Sprintf("  Total Healing: %d\n", totalHealing))
	}
	sb.WriteString(fmt.Sprintf("  Units Killed: %d\n", totalKills))

	// Counterattack stats
	if counterAttacks > 0 {
		sb.WriteString(fmt.Sprintf("  Counterattacks: %d\n", counterAttacks))

		sb.WriteString("  Counter Outcomes: ")
		counterOutcomes := []string{}
		if counterHits > 0 {
			counterOutcomes = append(counterOutcomes, fmt.Sprintf("%d hits", counterHits))
		}
		if counterMisses > 0 {
			counterOutcomes = append(counterOutcomes, fmt.Sprintf("%d misses", counterMisses))
		}
		if counterDodges > 0 {
			counterOutcomes = append(counterOutcomes, fmt.Sprintf("%d dodges", counterDodges))
		}
		if counterCriticals > 0 {
			counterOutcomes = append(counterOutcomes, fmt.Sprintf("%d criticals", counterCriticals))
		}
		sb.WriteString(strings.Join(counterOutcomes, ", "))
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("  Counter Damage: %d\n", counterDamage))
		sb.WriteString(fmt.Sprintf("  Counter Kills: %d\n", counterKills))
	}

	return sb.String()
}

// formatIntSlice formats an integer slice as [1,2,3].
func formatIntSlice(nums []int) string {
	if len(nums) == 0 {
		return "[]"
	}

	strs := make([]string, len(nums))
	for i, n := range nums {
		strs[i] = fmt.Sprintf("%d", n)
	}

	return "[" + strings.Join(strs, ",") + "]"
}

// formatCellSlice formats grid positions as [(0,0),(1,1)].
func formatCellSlice(cells []GridPosition) string {
	if len(cells) == 0 {
		return "[]"
	}

	strs := make([]string, len(cells))
	for i, cell := range cells {
		strs[i] = fmt.Sprintf("(%d,%d)", cell.Row, cell.Col)
	}

	return "[" + strings.Join(strs, ",") + "]"
}
