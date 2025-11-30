package squads

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CombatResult - ✅ Uses ecs.EntityID (native type) instead of entity pointers
type CombatResult struct {
	TotalDamage  int
	UnitsKilled  []ecs.EntityID       // ✅ Native IDs
	DamageByUnit map[ecs.EntityID]int // ✅ Native IDs
	CombatLog    *CombatLog           // Detailed event log for display
}

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
// Only units within their attack range of the target squad can participate
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Initialize combat log
	combatLog := &CombatLog{
		AttackerSquadID:   attackerSquadID,
		DefenderSquadID:   defenderSquadID,
		AttackerSquadName: GetSquadName(attackerSquadID, squadmanager),
		DefenderSquadName: GetSquadName(defenderSquadID, squadmanager),
		SquadDistance:     GetSquadDistance(attackerSquadID, defenderSquadID, squadmanager),
		AttackEvents:      []AttackEvent{},
		AttackingUnits:    []UnitSnapshot{},
	}

	// Calculate distance between squads for range checking
	squadDistance := combatLog.SquadDistance
	if squadDistance < 0 {
		// Invalid squad positions, cannot attack
		result.CombatLog = combatLog
		return result
	}

	// Query for attacker unit IDs (not pointers!)
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

	// Capture attacking units snapshot (before actual combat)
	for _, attackerID := range attackerUnitIDs {
		attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
		if attackerAttr == nil || attackerAttr.CurrentHealth <= 0 {
			continue
		}

		if !squadmanager.HasComponentByIDWithTag(attackerID, SquadMemberTag, AttackRangeComponent) {
			continue
		}
		rangeData := common.GetComponentTypeByID[*AttackRangeData](squadmanager, attackerID, AttackRangeComponent)
		if rangeData.Range < squadDistance {
			continue
		}

		// Add to attacking units snapshot
		identity := GetUnitIdentity(attackerID, squadmanager)
		roleData := common.GetComponentTypeByID[*UnitRoleData](squadmanager, attackerID, UnitRoleComponent)
		roleName := "Unknown"
		if roleData != nil {
			roleName = roleData.Role.String()
		}

		snapshot := UnitSnapshot{
			UnitID:      attackerID,
			UnitName:    identity.Name,
			GridRow:     identity.GridRow,
			GridCol:     identity.GridCol,
			AttackRange: rangeData.Range,
			RoleName:    roleName,
		}
		combatLog.AttackingUnits = append(combatLog.AttackingUnits, snapshot)
	}

	// Attack index for sequential numbering
	attackIndex := 0

	// Process each attacker unit
	for _, attackerID := range attackerUnitIDs {
		// Check if unit is alive
		attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
		if attackerAttr == nil || attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Check if unit is within attack range
		if !squadmanager.HasComponentByIDWithTag(attackerID, SquadMemberTag, AttackRangeComponent) {
			continue
		}
		rangeData := common.GetComponentTypeByID[*AttackRangeData](squadmanager, attackerID, AttackRangeComponent)
		if rangeData.Range < squadDistance {
			// Unit is out of range, cannot attack
			continue
		}

		// Get targeting data
		if !squadmanager.HasComponentByIDWithTag(attackerID, SquadMemberTag, TargetRowComponent) {
			continue
		}

		targetRowData := common.GetComponentTypeByID[*TargetRowData](squadmanager, attackerID, TargetRowComponent)

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

		// Apply damage to each selected target and capture events
		multiTargetIndex := 1
		for _, defenderID := range actualTargetIDs {
			attackIndex++

			// Calculate damage with event capture
			damage, event := calculateUnitDamageByID(attackerID, defenderID, squadmanager)

			// Add targeting info to event
			defenderPos := common.GetComponentTypeByID[*GridPositionData](squadmanager, defenderID, GridPositionComponent)
			event.AttackIndex = attackIndex
			if defenderPos != nil {
				event.TargetInfo.TargetRow = defenderPos.AnchorRow
				event.TargetInfo.TargetCol = defenderPos.AnchorCol
			}
			event.TargetInfo.IsMultiTarget = targetRowData.IsMultiTarget
			if targetRowData.IsMultiTarget {
				event.TargetInfo.MultiTargetIndex = multiTargetIndex
				multiTargetIndex++
			}
			if targetRowData.Mode == TargetModeCellBased {
				event.TargetInfo.TargetMode = "cell"
			} else {
				event.TargetInfo.TargetMode = "row"
			}

			// Apply damage
			applyDamageToUnitByID(defenderID, damage, result, squadmanager)

			// Store event
			combatLog.AttackEvents = append(combatLog.AttackEvents, *event)
		}
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	// Calculate combat summary
	combatLog.TotalDamage = result.TotalDamage
	combatLog.UnitsKilled = len(result.UnitsKilled)
	combatLog.DefenderStatus = calculateSquadStatus(defenderSquadID, squadmanager)

	result.CombatLog = combatLog
	return result
}

// calculateUnitDamageByID calculates damage using new attribute system and returns detailed event data
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *common.EntityManager) (int, *AttackEvent) {
	attackerAttr := common.GetAttributesByIDWithTag(squadmanager, attackerID, SquadMemberTag)
	defenderAttr := common.GetAttributesByIDWithTag(squadmanager, defenderID, SquadMemberTag)

	// Create event to track damage pipeline
	event := &AttackEvent{
		AttackerID: attackerID,
		DefenderID: defenderID,
	}

	if defenderAttr != nil {
		event.DefenderHPBefore = defenderAttr.CurrentHealth
	}

	if attackerAttr == nil || defenderAttr == nil {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Hit roll
	hitThreshold := attackerAttr.GetHitRate()
	hitRoll := common.GetDiceRoll(100)
	event.HitResult.HitRoll = hitRoll
	event.HitResult.HitThreshold = hitThreshold

	if hitRoll > hitThreshold {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Dodge roll
	dodgeThreshold := defenderAttr.GetDodgeChance()
	dodgeRoll := common.GetDiceRoll(100)
	event.HitResult.DodgeRoll = dodgeRoll
	event.HitResult.DodgeThreshold = dodgeThreshold

	if dodgeRoll <= dodgeThreshold {
		event.HitResult.Type = HitTypeDodge
		return 0, event
	}

	// Calculate base damage
	baseDamage := attackerAttr.GetPhysicalDamage()
	event.BaseDamage = baseDamage
	event.CritMultiplier = 1.0

	// Crit roll
	critThreshold := attackerAttr.GetCritChance()
	critRoll := common.GetDiceRoll(100)
	event.HitResult.CritRoll = critRoll
	event.HitResult.CritThreshold = critThreshold

	if critRoll <= critThreshold {
		baseDamage = int(float64(baseDamage) * 1.5)
		event.CritMultiplier = 1.5
		event.HitResult.Type = HitTypeCritical
	} else {
		event.HitResult.Type = HitTypeNormal
	}

	// Apply resistance
	resistance := defenderAttr.GetPhysicalResistance()
	event.ResistanceAmount = resistance
	totalDamage := baseDamage - resistance
	if totalDamage < 1 {
		totalDamage = 1
	}

	// Apply cover with detailed breakdown
	coverBreakdown := CalculateCoverBreakdown(defenderID, squadmanager)
	event.CoverReduction = coverBreakdown

	if coverBreakdown.TotalReduction > 0.0 {
		totalDamage = int(float64(totalDamage) * (1.0 - coverBreakdown.TotalReduction))
		if totalDamage < 1 {
			totalDamage = 1
		}
	}

	event.FinalDamage = totalDamage
	event.DefenderHPAfter = defenderAttr.CurrentHealth - totalDamage
	if event.DefenderHPAfter <= 0 {
		event.WasKilled = true
	}

	return totalDamage, event
}

// rollHit determines if an attack hits based on hit rate
func rollHit(hitRate int) bool {
	roll := common.GetDiceRoll(100)
	return roll <= hitRate
}

// rollCrit determines if an attack is a critical hit
func rollCrit(critChance int) bool {
	roll := common.GetDiceRoll(100)
	return roll <= critChance
}

// rollDodge determines if an attack is dodged
func rollDodge(dodgeChance int) bool {
	roll := common.GetDiceRoll(100)
	return roll <= dodgeChance
}

// applyDamageToUnitByID - ✅ Uses ecs.EntityID
func applyDamageToUnitByID(unitID ecs.EntityID, damage int, result *CombatResult, squadmanager *common.EntityManager) {
	attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
	if attr == nil {
		return
	}

	attr.CurrentHealth -= damage
	result.DamageByUnit[unitID] = damage

	if attr.CurrentHealth <= 0 {
		result.UnitsKilled = append(result.UnitsKilled, unitID)
	}

	// Update the squad's destroyed status cache after health change
	memberData := common.GetComponentTypeByID[*SquadMemberData](squadmanager, unitID, SquadMemberComponent)
	if memberData != nil {
		UpdateSquadDestroyedStatus(memberData.SquadID, squadmanager)
	}
}

// selectLowestHPTargetID - TODO, don't think I will want this kind of targeting
func selectLowestHPTargetID(unitIDs []ecs.EntityID, squadmanager *common.EntityManager) ecs.EntityID {
	if len(unitIDs) == 0 {
		return 0
	}

	lowestID := unitIDs[0]
	lowestAttr := common.GetAttributesByIDWithTag(squadmanager, lowestID, SquadMemberTag)
	if lowestAttr == nil {
		return 0
	}
	lowestHP := lowestAttr.CurrentHealth

	for _, unitID := range unitIDs[1:] {
		attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		hp := attr.CurrentHealth
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
	common.Shuffle(len(shuffled), func(i, j int) {
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

// calculateSquadStatus summarizes squad health for combat log
func calculateSquadStatus(squadID ecs.EntityID, manager *common.EntityManager) SquadStatus {
	unitIDs := GetUnitIDsInSquad(squadID, manager)
	aliveCount := 0
	totalHP := 0
	totalMaxHP := 0

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth > 0 {
			aliveCount++
			totalHP += attr.CurrentHealth
			totalMaxHP += attr.MaxHealth
		}
	}

	avgHP := 0
	if totalMaxHP > 0 {
		avgHP = (totalHP * 100) / totalMaxHP
	}

	return SquadStatus{
		AliveUnits: aliveCount,
		TotalUnits: len(unitIDs),
		AverageHP:  avgHP,
	}
}

// ========================================
// COVER SYSTEM FUNCTIONS
// ========================================

// CalculateTotalCover calculates the total damage reduction from all units providing cover to the defender
// Cover bonuses stack additively (e.g., 0.25 + 0.15 = 0.40 total reduction)
// Returns a value between 0.0 (no cover) and 1.0 (100% damage reduction, capped)
func CalculateTotalCover(defenderID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	// Check if defender exists with required components
	if !squadmanager.HasComponentByIDWithTag(defenderID, SquadMemberTag, GridPositionComponent) ||
		!squadmanager.HasComponentByIDWithTag(defenderID, SquadMemberTag, SquadMemberComponent) {
		return 0.0
	}

	// Get defender's position and squad
	defenderPos := common.GetComponentTypeByID[*GridPositionData](squadmanager, defenderID, GridPositionComponent)
	defenderSquadData := common.GetComponentTypeByID[*SquadMemberData](squadmanager, defenderID, SquadMemberComponent)
	defenderSquadID := defenderSquadData.SquadID

	// Get all units providing cover
	coverProviders := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	// Sum all cover bonuses (stacking additively)
	totalCover := 0.0
	for _, providerID := range coverProviders {
		// Check if provider has cover component
		if !squadmanager.HasComponentByIDWithTag(providerID, SquadMemberTag, CoverComponent) {
			continue
		}

		coverData := common.GetComponentTypeByID[*CoverData](squadmanager, providerID, CoverComponent)

		// Check if provider is active (alive and not stunned)
		isActive := true
		if coverData.RequiresActive {
			attr := common.GetAttributesByIDWithTag(squadmanager, providerID, SquadMemberTag)
			if attr != nil {
				isActive = attr.CurrentHealth > 0
			}
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
func GetCoverProvidersFor(defenderID ecs.EntityID, defenderSquadID ecs.EntityID, defenderPos *GridPositionData, squadmanager *common.EntityManager) []ecs.EntityID {
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

		// Check if unit has cover component
		if !squadmanager.HasComponentByIDWithTag(unitID, SquadMemberTag, CoverComponent) {
			continue
		}

		coverData := common.GetComponentTypeByID[*CoverData](squadmanager, unitID, CoverComponent)

		// Get unit's position
		if !squadmanager.HasComponentByIDWithTag(unitID, SquadMemberTag, GridPositionComponent) {
			continue
		}

		unitPos := common.GetComponentTypeByID[*GridPositionData](squadmanager, unitID, GridPositionComponent)

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

// CalculateCoverBreakdown returns detailed cover information for logging
// Similar to CalculateTotalCover but includes provider details
func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) CoverBreakdown {
	breakdown := CoverBreakdown{
		Providers: []CoverProvider{},
	}

	if !squadmanager.HasComponentByIDWithTag(defenderID, SquadMemberTag, GridPositionComponent) ||
		!squadmanager.HasComponentByIDWithTag(defenderID, SquadMemberTag, SquadMemberComponent) {
		return breakdown
	}

	defenderPos := common.GetComponentTypeByID[*GridPositionData](squadmanager, defenderID, GridPositionComponent)
	defenderSquadData := common.GetComponentTypeByID[*SquadMemberData](squadmanager, defenderID, SquadMemberComponent)
	defenderSquadID := defenderSquadData.SquadID

	providerIDs := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	totalCover := 0.0
	for _, providerID := range providerIDs {
		if !squadmanager.HasComponentByIDWithTag(providerID, SquadMemberTag, CoverComponent) {
			continue
		}

		coverData := common.GetComponentTypeByID[*CoverData](squadmanager, providerID, CoverComponent)

		// Check if active
		isActive := true
		if coverData.RequiresActive {
			attr := common.GetAttributesByIDWithTag(squadmanager, providerID, SquadMemberTag)
			if attr != nil {
				isActive = attr.CurrentHealth > 0
			}
		}

		coverValue := coverData.GetCoverBonus(isActive)
		if coverValue > 0 {
			// Get provider info for logging
			identity := GetUnitIdentity(providerID, squadmanager)
			providerPos := common.GetComponentTypeByID[*GridPositionData](squadmanager, providerID, GridPositionComponent)

			provider := CoverProvider{
				UnitID:     providerID,
				UnitName:   identity.Name,
				CoverValue: coverValue,
			}
			if providerPos != nil {
				provider.GridRow = providerPos.AnchorRow
				provider.GridCol = providerPos.AnchorCol
			}

			breakdown.Providers = append(breakdown.Providers, provider)
			totalCover += coverValue
		}
	}

	if totalCover > 1.0 {
		totalCover = 1.0
	}
	breakdown.TotalReduction = totalCover

	return breakdown
}

func displayCombatResult(result *CombatResult, squadmanager *common.EntityManager) {
	fmt.Printf("  Total damage: %d\n", result.TotalDamage)
	fmt.Printf("  Units killed: %d\n", len(result.UnitsKilled))

	// ✅ Result uses native entity IDs
	for unitID, dmg := range result.DamageByUnit {
		name := common.GetComponentTypeByID[*common.Name](squadmanager, unitID, common.NameComponent)
		if name == nil {
			continue
		}
		fmt.Printf("    %s took %d damage\n", name.NameStr, dmg)
	}
}

func displaySquadStatus(squadID ecs.EntityID, squadmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	fmt.Printf("\n%s Status:\n", squadData.Name)

	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	alive := 0

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(squadmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth > 0 {
			alive++
			name := common.GetComponentTypeByID[*common.Name](squadmanager, unitID, common.NameComponent)
			fmt.Printf("  %s: %d/%d HP\n", name.NameStr, attr.CurrentHealth, attr.MaxHealth)
		}
	}

	fmt.Printf("Total alive: %d\n", alive)
}
