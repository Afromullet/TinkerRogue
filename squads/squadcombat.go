package squads

import (
	"fmt"
	"game_main/common"
	"game_main/randgen"
	"math/rand/v2"

	"github.com/bytearena/ecs"
)

// CombatResult - ✅ Uses ecs.EntityID (native type) instead of entity pointers
type CombatResult struct {
	TotalDamage  int
	UnitsKilled  []ecs.EntityID       // ✅ Native IDs
	DamageByUnit map[ecs.EntityID]int // ✅ Native IDs
}

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *SquadECSManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Query for attacker unit IDs (not pointers!)
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

	// Process each attacker unit
	for _, attackerID := range attackerUnitIDs {
		attackerUnit := FindUnitByID(attackerID, squadmanager)
		if attackerUnit == nil {
			continue
		}

		// Check if unit is alive
		attackerAttr := common.GetAttributes(attackerUnit)
		if attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Get targeting data
		if !attackerUnit.HasComponent(TargetRowComponent) {
			continue
		}

		targetRowData := common.GetComponentType[*TargetRowData](attackerUnit, TargetRowComponent)

		var actualTargetIDs []ecs.EntityID

		// Handle targeting based on mode
		if targetRowData.Mode == TargetModeCellBased {
			// Cell-based targeting: hit specific grid cells
			for _, cell := range targetRowData.TargetCells {
				row, col := cell[0], cell[1]
				cellTargetIDs := GetUnitIDsAtGridPosition(defenderSquadID, row, col, squadmanager)
				actualTargetIDs = append(actualTargetIDs, cellTargetIDs...)
			}
		} else {
			// Row-based targeting: hit entire row(s)
			for _, targetRow := range targetRowData.TargetRows {
				targetIDs := GetUnitIDsInRow(defenderSquadID, targetRow, squadmanager)

				if len(targetIDs) == 0 {
					continue
				}

				//TODO, handle multitarget seletion a better way. Figure out whether we still want that.
				//I am thinking just cell based will do it
				if targetRowData.IsMultiTarget {
					maxTargets := targetRowData.MaxTargets
					if maxTargets == 0 || maxTargets > len(targetIDs) {
						actualTargetIDs = append(actualTargetIDs, targetIDs...)
					} else {
						actualTargetIDs = append(actualTargetIDs, selectRandomTargetIDs(targetIDs, maxTargets)...)
					}
				} else {
					actualTargetIDs = append(actualTargetIDs, selectLowestHPTargetID(targetIDs, squadmanager))
				}
			}
		}

		//TODO this is where we should add hit chance
		// Apply damage to each selected target
		for _, defenderID := range actualTargetIDs {
			damage := calculateUnitDamageByID(attackerID, defenderID, squadmanager)
			applyDamageToUnitByID(defenderID, damage, result, squadmanager)
		}
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	return result
}

// calculateUnitDamageByID - TODO, calculate damage based off unit attributes
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *SquadECSManager) int {
	attackerUnit := FindUnitByID(attackerID, squadmanager)
	defenderUnit := FindUnitByID(defenderID, squadmanager)

	if attackerUnit == nil || defenderUnit == nil {
		return 0
	}

	attackerAttr := common.GetAttributes(attackerUnit)
	defenderAttr := common.GetAttributes(defenderUnit)

	// Base damage (adapt to existing weapon system)
	baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus

	// d20 variance (reuse existing logic)
	roll := randgen.GetDiceRoll(20)
	if roll >= 18 {
		baseDamage = int(float64(baseDamage) * 1.5) // Critical
	} else if roll <= 3 {
		baseDamage = baseDamage / 2 // Weak hit
	}

	// Apply role modifiers
	if attackerUnit.HasComponent(UnitRoleComponent) {

		baseDamage = 1 //Todo, calculate this from attributes
	}

	// Apply defense
	totalDamage := baseDamage - defenderAttr.TotalProtection
	if totalDamage < 1 {
		totalDamage = 1 // Minimum damage
	}

	// Apply cover (damage reduction from units in front)
	coverReduction := CalculateTotalCover(defenderID, squadmanager)
	if coverReduction > 0.0 {
		totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))
		if totalDamage < 1 {
			totalDamage = 1 // Minimum damage even with cover
		}
	}

	return totalDamage
}

// applyDamageToUnitByID - ✅ Uses ecs.EntityID
func applyDamageToUnitByID(unitID ecs.EntityID, damage int, result *CombatResult, squadmanager *SquadECSManager) {
	unit := FindUnitByID(unitID, squadmanager)
	if unit == nil {
		return
	}

	attr := common.GetAttributes(unit)
	attr.CurrentHealth -= damage
	result.DamageByUnit[unitID] = damage

	if attr.CurrentHealth <= 0 {
		result.UnitsKilled = append(result.UnitsKilled, unitID)
	}
}

// selectLowestHPTargetID - TODO, don't think I will want this kind of targeting
func selectLowestHPTargetID(unitIDs []ecs.EntityID, squadmanager *SquadECSManager) ecs.EntityID {
	if len(unitIDs) == 0 {
		return 0
	}

	lowestID := unitIDs[0]
	lowestUnit := FindUnitByID(lowestID, squadmanager)
	if lowestUnit == nil {
		return 0
	}
	lowestHP := common.GetAttributes(lowestUnit).CurrentHealth

	for _, unitID := range unitIDs[1:] {
		unit := FindUnitByID(unitID, squadmanager)
		if unit == nil {
			continue
		}

		hp := common.GetAttributes(unit).CurrentHealth
		if hp < lowestHP {
			lowestID = unitID
			lowestHP = hp
		}
	}

	return lowestID
}

// selectRandomTargetIDs - ✅ Works with ecs.EntityID
func selectRandomTargetIDs(unitIDs []ecs.EntityID, count int) []ecs.EntityID {
	if count >= len(unitIDs) {
		return unitIDs
	}

	// Shuffle and take first N
	shuffled := make([]ecs.EntityID, len(unitIDs))
	copy(shuffled, unitIDs)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

func sumDamageMap(damageMap map[ecs.EntityID]int) int {
	total := 0
	for _, dmg := range damageMap {
		total += dmg
	}
	return total
}

// ========================================
// COVER SYSTEM FUNCTIONS
// ========================================

// CalculateTotalCover calculates the total damage reduction from all units providing cover to the defender
// Cover bonuses stack additively (e.g., 0.25 + 0.15 = 0.40 total reduction)
// Returns a value between 0.0 (no cover) and 1.0 (100% damage reduction, capped)
func CalculateTotalCover(defenderID ecs.EntityID, squadmanager *SquadECSManager) float64 {
	defenderUnit := FindUnitByID(defenderID, squadmanager)
	if defenderUnit == nil {
		return 0.0
	}

	// Get defender's position and squad
	if !defenderUnit.HasComponent(GridPositionComponent) || !defenderUnit.HasComponent(SquadMemberComponent) {
		return 0.0
	}

	defenderPos := common.GetComponentType[*GridPositionData](defenderUnit, GridPositionComponent)
	defenderSquadData := common.GetComponentType[*SquadMemberData](defenderUnit, SquadMemberComponent)
	defenderSquadID := defenderSquadData.SquadID

	// Get all units providing cover
	coverProviders := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	// Sum all cover bonuses (stacking additively)
	totalCover := 0.0
	for _, providerID := range coverProviders {
		providerUnit := FindUnitByID(providerID, squadmanager)
		if providerUnit == nil {
			continue
		}

		// Check if provider has cover component
		if !providerUnit.HasComponent(CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*CoverData](providerUnit, CoverComponent)

		// Check if provider is active (alive and not stunned)
		isActive := true
		if coverData.RequiresActive {
			attr := common.GetAttributes(providerUnit)
			isActive = attr.CurrentHealth > 0
			// TODO: Add stun/disable status check when status effects are implemented
		}

		totalCover += coverData.GetCoverBonus(isActive)
	}

	// Cap at 100% reduction (though in practice this should be very rare)
	if totalCover > 1.0 {
		totalCover = 1.0
	}

	return totalCover
}

// GetCoverProvidersFor finds all units in the same squad that provide cover to the defender
// Cover is provided by units in front (lower row number) within the same column(s)
// Multi-cell units provide cover to all columns they occupy
func GetCoverProvidersFor(defenderID ecs.EntityID, defenderSquadID ecs.EntityID, defenderPos *GridPositionData, squadmanager *SquadECSManager) []ecs.EntityID {
	var providers []ecs.EntityID

	// Get all columns the defender occupies
	defenderCols := make(map[int]bool)
	for c := defenderPos.AnchorCol; c < defenderPos.AnchorCol+defenderPos.Width && c < 3; c++ {
		defenderCols[c] = true
	}

	// Get all units in the same squad
	allUnitIDs := GetUnitIDsInSquad(defenderSquadID, squadmanager)

	for _, unitID := range allUnitIDs {
		// Don't provide cover to yourself
		if unitID == defenderID {
			continue
		}

		unit := FindUnitByID(unitID, squadmanager)
		if unit == nil {
			continue
		}

		// Check if unit has cover component
		if !unit.HasComponent(CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*CoverData](unit, CoverComponent)

		// Get unit's position
		if !unit.HasComponent(GridPositionComponent) {
			continue
		}

		unitPos := common.GetComponentType[*GridPositionData](unit, GridPositionComponent)

		// Check if unit is in front of defender (lower row number)
		// Unit must be at least 1 row in front to provide cover
		if unitPos.AnchorRow >= defenderPos.AnchorRow {
			continue
		}

		// Check if unit is within cover range
		rowDistance := defenderPos.AnchorRow - unitPos.AnchorRow
		if rowDistance > coverData.CoverRange {
			continue
		}

		// Check if unit occupies any column the defender is in
		unitCols := make(map[int]bool)
		for c := unitPos.AnchorCol; c < unitPos.AnchorCol+unitPos.Width && c < 3; c++ {
			unitCols[c] = true
		}

		// Check for column overlap
		hasOverlap := false
		for col := range defenderCols {
			if unitCols[col] {
				hasOverlap = true
				break
			}
		}

		if hasOverlap {
			providers = append(providers, unitID)
		}
	}

	return providers
}

func displayCombatResult(result *CombatResult, squadmanager *SquadECSManager) {
	fmt.Printf("  Total damage: %d\n", result.TotalDamage)
	fmt.Printf("  Units killed: %d\n", len(result.UnitsKilled))

	// ✅ Result uses native entity IDs
	for unitID, dmg := range result.DamageByUnit {
		unit := FindUnitByID(unitID, squadmanager)
		if unit == nil {
			continue
		}
		name := common.GetComponentType[*common.Name](unit, common.NameComponent)
		fmt.Printf("    %s took %d damage\n", name.NameStr, dmg)
	}
}

func displaySquadStatus(squadID ecs.EntityID, squadmanager *SquadECSManager) {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	fmt.Printf("\n%s Status:\n", squadData.Name)

	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	alive := 0

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, squadmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			alive++
			name := common.GetComponentType[*common.Name](unit, common.NameComponent)
			fmt.Printf("  %s: %d/%d HP\n", name.NameStr, attr.CurrentHealth, attr.MaxHealth)
		}
	}

	fmt.Printf("Total alive: %d\n", alive)
}
