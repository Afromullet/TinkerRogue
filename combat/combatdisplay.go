package combat

import (
	"fmt"
	"strings"

	"game_main/common"
	"game_main/squads"
)

// DisplayCombatLog formats and prints the full combat log to terminal
func DisplayCombatLog(log *squads.CombatLog, manager *common.EntityManager) {
	if log == nil {
		return
	}

	printSeparator()
	printHeader(log)
	printSeparator()

	printAttackingUnits(log.AttackingUnits)

	fmt.Println("\n--- ATTACK SEQUENCE ---")

	for _, event := range log.AttackEvents {
		printAttackEvent(event, manager)
	}

	fmt.Println("\n--- COMBAT SUMMARY ---")
	printSummary(log)
	printSeparator()
}

func printSeparator() {
	fmt.Println(strings.Repeat("=", 80))
}

func printHeader(log *squads.CombatLog) {
	fmt.Printf("COMBAT: %s attacks %s at range %d\n",
		log.AttackerSquadName,
		log.DefenderSquadName,
		log.SquadDistance)
}

func printAttackingUnits(units []squads.UnitSnapshot) {
	if len(units) == 0 {
		return
	}

	fmt.Printf("\n--- Attacking Units (%d in range) ---\n", len(units))
	for _, unit := range units {
		fmt.Printf("  ✓ %s (Row %d, Col %d, %s, Range %d)\n",
			unit.UnitName,
			unit.GridRow,
			unit.GridCol,
			unit.RoleName,
			unit.AttackRange)
	}
}

func printAttackEvent(event squads.AttackEvent, manager *common.EntityManager) {
	// Get identities
	attacker := squads.GetUnitIdentity(event.AttackerID, manager)
	defender := squads.GetUnitIdentity(event.DefenderID, manager)

	// Header with attacker → defender
	fmt.Printf("[%d] %s → %s (Row %d, Col %d)\n",
		event.AttackIndex,
		attacker.Name,
		defender.Name,
		event.TargetInfo.TargetRow,
		event.TargetInfo.TargetCol)

	// Handle miss/dodge
	if event.HitResult.Type == squads.HitTypeMiss {
		fmt.Printf("    MISS (rolled %d, needed ≤%d)\n",
			event.HitResult.HitRoll,
			event.HitResult.HitThreshold)
		fmt.Println()
		return
	}

	if event.HitResult.Type == squads.HitTypeDodge {
		fmt.Printf("    DODGED (rolled %d, defender dodge %d%%)\n",
			event.HitResult.DodgeRoll,
			event.HitResult.DodgeThreshold)
		fmt.Println()
		return
	}

	// Base damage line
	baseLine := fmt.Sprintf("    Base Damage: %d", event.BaseDamage)
	if event.HitResult.Type == squads.HitTypeCritical {
		baseLine += fmt.Sprintf(" × %.1f CRITICAL!", event.CritMultiplier)
	}
	fmt.Println(baseLine)

	// Resistance line
	if event.ResistanceAmount > 0 {
		fmt.Printf("    - Defender Resistance: -%d\n", event.ResistanceAmount)
	}

	// Cover lines
	if event.CoverReduction.TotalReduction > 0 {
		damageAfterResist := event.BaseDamage
		if event.CritMultiplier > 1.0 {
			damageAfterResist = int(float64(event.BaseDamage) * event.CritMultiplier)
		}
		damageAfterResist -= event.ResistanceAmount
		if damageAfterResist < 1 {
			damageAfterResist = 1
		}

		for _, provider := range event.CoverReduction.Providers {
			percentage := int(provider.CoverValue * 100)
			reduction := int(float64(damageAfterResist) * provider.CoverValue)
			fmt.Printf("    - Cover from %s: -%d (%d%% reduction)\n",
				provider.UnitName,
				reduction,
				percentage)
		}
	} else {
		fmt.Println("    - No Cover")
	}

	// Final damage
	fmt.Printf("    → FINAL DAMAGE: %d HP\n", event.FinalDamage)

	// Defender status
	if event.WasKilled {
		fmt.Printf("    [%s: KILLED]\n", defender.Name)
	} else {
		fmt.Printf("    [%s: %d/%d HP remaining]\n",
			defender.Name,
			event.DefenderHPAfter,
			defender.MaxHP)
	}

	fmt.Println()
}

func printSummary(log *squads.CombatLog) {
	fmt.Printf("Total Damage Dealt: %d\n", log.TotalDamage)
	fmt.Printf("Enemies Killed: %d\n", log.UnitsKilled)
	fmt.Printf("%s Status: %d/%d units alive (Avg HP: %d%%)\n",
		log.DefenderSquadName,
		log.DefenderStatus.AliveUnits,
		log.DefenderStatus.TotalUnits,
		log.DefenderStatus.AverageHP)
}
