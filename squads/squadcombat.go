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

	// Initialize combat log with squad info
	combatLog := initializeCombatLog(attackerSquadID, defenderSquadID, squadmanager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		return result
	}

	// Snapshot units that will participate (for logging)
	combatLog.AttackingUnits = snapshotAttackingUnits(attackerSquadID, combatLog.SquadDistance, squadmanager)

	// Process each attacking unit
	attackIndex := 0
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

	for _, attackerID := range attackerUnitIDs {
		// Check if unit can attack (alive and in range)
		if !canUnitAttack(attackerID, combatLog.SquadDistance, squadmanager) {
			continue
		}

		// Get targets for this attacker
		targetIDs := SelectTargetUnits(attackerID, defenderSquadID, squadmanager)

		// Attack each target
		attackIndex = processAttackOnTargets(attackerID, targetIDs, result, combatLog, attackIndex, squadmanager)
	}

	// Update defender squad destroyed status once after all attacks (performance optimization)
	UpdateSquadDestroyedStatus(defenderSquadID, squadmanager)

	// Finalize combat log with summary
	finalizeCombatLog(result, combatLog, defenderSquadID, squadmanager)

	result.CombatLog = combatLog
	return result
}

// initializeCombatLog creates the combat log structure with squad information
func initializeCombatLog(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *CombatLog {
	return &CombatLog{
		AttackerSquadID:   attackerSquadID,
		DefenderSquadID:   defenderSquadID,
		AttackerSquadName: GetSquadName(attackerSquadID, manager),
		DefenderSquadName: GetSquadName(defenderSquadID, manager),
		SquadDistance:     GetSquadDistance(attackerSquadID, defenderSquadID, manager),
		AttackEvents:      []AttackEvent{},
		AttackingUnits:    []UnitSnapshot{},
	}
}

// snapshotAttackingUnits captures attacking unit info before combat for logging
// Optimized: Batches all component lookups into single GetEntityByID call per unit.
func snapshotAttackingUnits(squadID ecs.EntityID, squadDistance int, manager *common.EntityManager) []UnitSnapshot {
	var snapshots []UnitSnapshot
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		// Check if unit can participate
		if !canUnitAttack(unitID, squadDistance, manager) {
			continue
		}

		// OPTIMIZATION: Get entity once, extract all components (Name, GridPosition, AttackRange, Role)
		// This replaces 4+ separate GetEntityByID calls with just 1
		entity := common.FindEntityByID(manager, unitID)
		if entity == nil {
			continue
		}

		// Extract all needed components from entity
		name := common.GetComponentType[*common.Name](entity, common.NameComponent)
		gridPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
		rangeData := common.GetComponentType[*AttackRangeData](entity, AttackRangeComponent)
		roleData := common.GetComponentType[*UnitRoleData](entity, UnitRoleComponent)

		unitName := "Unknown"
		if name != nil {
			unitName = name.NameStr
		}

		row, col := 0, 0
		if gridPos != nil {
			row, col = gridPos.AnchorRow, gridPos.AnchorCol
		}

		attackRange := 0
		if rangeData != nil {
			attackRange = rangeData.Range
		}

		roleName := "Unknown"
		if roleData != nil {
			roleName = roleData.Role.String()
		}

		snapshot := UnitSnapshot{
			UnitID:      unitID,
			UnitName:    unitName,
			GridRow:     row,
			GridCol:     col,
			AttackRange: attackRange,
			RoleName:    roleName,
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots
}

// finalizeCombatLog calculates summary statistics for the combat log
func finalizeCombatLog(result *CombatResult, log *CombatLog, defenderSquadID ecs.EntityID, manager *common.EntityManager) {
	result.TotalDamage = sumDamageMap(result.DamageByUnit)
	log.TotalDamage = result.TotalDamage
	log.UnitsKilled = len(result.UnitsKilled)
	log.DefenderStatus = calculateSquadStatus(defenderSquadID, manager)
}

// canUnitAttack checks if a unit is alive, can act, and within attack range
// Optimized: Batches all component lookups into single GetEntityByID call (3 calls → 1).
func canUnitAttack(attackerID ecs.EntityID, squadDistance int, manager *common.EntityManager) bool {
	// OPTIMIZATION: Get entity once, extract Attributes and AttackRange
	// This replaces 3 separate GetEntityByID calls with just 1
	entity := common.FindEntityByID(manager, attackerID)
	if entity == nil {
		return false
	}

	// Check if unit is alive and can act
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil || attr.CurrentHealth <= 0 || !attr.CanAct {
		return false
	}

	// Check if unit has attack range component and is within range
	if !entity.HasComponent(AttackRangeComponent) {
		return false
	}

	rangeData := common.GetComponentType[*AttackRangeData](entity, AttackRangeComponent)
	return rangeData != nil && rangeData.Range >= squadDistance
}

// SelectTargetUnits determines targets based on attack type (public for GUI and internal use)
// Optimized: Batches component lookups into single GetEntityByID call (2 calls → 1).
func SelectTargetUnits(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// OPTIMIZATION: Get entity once, check for TargetRowComponent
	// This replaces 2 separate GetEntityByID calls with just 1
	entity := common.FindEntityByID(manager, attackerID)
	if entity == nil {
		return []ecs.EntityID{}
	}

	// Check if attacker has targeting component
	if !entity.HasComponent(TargetRowComponent) {
		return []ecs.EntityID{}
	}

	targetData := common.GetComponentType[*TargetRowData](entity, TargetRowComponent)
	if targetData == nil {
		return []ecs.EntityID{}
	}

	switch targetData.AttackType {
	case AttackTypeMeleeRow:
		return selectMeleeRowTargets(attackerID, defenderSquadID, manager)
	case AttackTypeMeleeColumn:
		return selectMeleeColumnTargets(attackerID, defenderSquadID, manager)
	case AttackTypeRanged:
		return selectRangedTargets(attackerID, defenderSquadID, manager)
	case AttackTypeMagic:
		return selectMagicTargets(defenderSquadID, targetData.TargetCells, manager)
	default:
		return []ecs.EntityID{}
	}
}

// selectMeleeRowTargets targets front row (row 0), piercing to next row if empty
// Always targets all units in the row (up to 3)
func selectMeleeRowTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// Try each row starting from front (0 → 1 → 2)
	for row := 0; row <= 2; row++ {
		targets := getUnitsInRow(defenderSquadID, row, manager)

		if len(targets) > 0 {
			return targets // Return all units in the row
		}
	}

	return []ecs.EntityID{}
}

// selectMeleeColumnTargets targets column directly across from attacker, wrapping to adjacent columns if empty
// Targets exactly 1 unit (spear-type attack)
// Optimized: Batches component lookups for attacker and defenders.
func selectMeleeColumnTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// OPTIMIZATION: Get attacker entity once
	attackerEntity := common.FindEntityByID(manager, attackerID)
	if attackerEntity == nil {
		return []ecs.EntityID{}
	}

	attackerPos := common.GetComponentType[*GridPositionData](attackerEntity, GridPositionComponent)
	if attackerPos == nil {
		return []ecs.EntityID{}
	}

	attackerCol := attackerPos.AnchorCol

	// Try columns starting from attacker's column, wrapping around
	// Example: attackerCol=1 → try columns 1, 2, 0
	for offset := 0; offset < 3; offset++ {
		col := (attackerCol + offset) % 3

		// For each column, search through all rows to find any ALIVE unit
		for row := 0; row <= 2; row++ {
			cellUnits := GetUnitIDsAtGridPosition(defenderSquadID, row, col, manager)

			for _, unitID := range cellUnits {
				// OPTIMIZATION: Get entity once for health check
				entity := common.FindEntityByID(manager, unitID)
				if entity == nil {
					continue
				}

				// Check if unit is alive
				attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
				if attr != nil && attr.CurrentHealth > 0 {
					// Return the first alive unit found
					return []ecs.EntityID{unitID}
				}
			}
		}
	}

	return []ecs.EntityID{}
}

// selectRangedTargets targets same row as attacker (all units), with fallback logic
// Optimized: Batches component lookup for attacker.
func selectRangedTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// OPTIMIZATION: Get attacker entity once
	attackerEntity := common.FindEntityByID(manager, attackerID)
	if attackerEntity == nil {
		return []ecs.EntityID{}
	}

	attackerPos := common.GetComponentType[*GridPositionData](attackerEntity, GridPositionComponent)
	if attackerPos == nil {
		return []ecs.EntityID{}
	}

	attackerRow := attackerPos.AnchorRow

	// Try same row as attacker - return ALL units in row
	targets := getUnitsInRow(defenderSquadID, attackerRow, manager)
	if len(targets) > 0 {
		return targets
	}

	// Fallback: lowest armor, furthest row, leftmost column tiebreaker
	return selectLowestArmorTarget(defenderSquadID, manager)
}

// selectMagicTargets uses cell-based patterns WITHOUT pierce-through
func selectMagicTargets(defenderSquadID ecs.EntityID, targetCells [][2]int, manager *common.EntityManager) []ecs.EntityID {
	var targets []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for _, cell := range targetCells {
		row, col := cell[0], cell[1]

		// Get units at exact cell (no pierce)
		cellTargets := GetUnitIDsAtGridPosition(defenderSquadID, row, col, manager)

		for _, unitID := range cellTargets {
			if !seen[unitID] {
				targets = append(targets, unitID)
				seen[unitID] = true
			}
		}
	}

	return targets
}

// Helper: Get all ALIVE units in a specific row
// Optimized: Uses direct entity lookup instead of GetAttributesByIDWithTag in loop.
func getUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	// Check all columns in the row
	for col := 0; col <= 2; col++ {
		cellUnits := GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				// OPTIMIZATION: Get entity once for health check
				entity := common.FindEntityByID(manager, unitID)
				if entity == nil {
					continue
				}

				// Filter out dead units
				attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
				if attr != nil && attr.CurrentHealth > 0 {
					units = append(units, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return units
}

// Helper: Select lowest armor target, furthest row on tie, leftmost column on further tie
func selectLowestArmorTarget(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	allUnits := GetUnitIDsInSquad(squadID, manager)

	if len(allUnits) == 0 {
		return []ecs.EntityID{}
	}

	// Find unit with lowest armor
	var bestTarget ecs.EntityID
	lowestArmor := int(^uint(0) >> 1) // Max int
	furthestRow := -1
	leftmostCol := 3 // Start with invalid column (max is 2)

	for _, unitID := range allUnits {
		attr := common.GetAttributesByIDWithTag(manager, unitID, SquadMemberTag)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		armor := attr.GetPhysicalResistance()
		pos := common.GetComponentTypeByID[*GridPositionData](manager, unitID, GridPositionComponent)
		if pos == nil {
			continue
		}

		row := pos.AnchorRow
		col := pos.AnchorCol

		// Select if:
		// 1. Lower armor, OR
		// 2. Same armor AND further row, OR
		// 3. Same armor AND same row AND more left column
		if armor < lowestArmor ||
			(armor == lowestArmor && row > furthestRow) ||
			(armor == lowestArmor && row == furthestRow && col < leftmostCol) {
			lowestArmor = armor
			furthestRow = row
			leftmostCol = col
			bestTarget = unitID
		}
	}

	if bestTarget == 0 {
		return []ecs.EntityID{}
	}

	return []ecs.EntityID{bestTarget}
}

// processAttackOnTargets applies damage to all targets and creates combat events
// Returns the updated attack index
func processAttackOnTargets(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	for _, defenderID := range targetIDs {
		attackIndex++

		// Calculate damage and create event
		damage, event := calculateUnitDamageByID(attackerID, defenderID, manager)

		// Add targeting info to event
		defenderPos := common.GetComponentTypeByID[*GridPositionData](manager, defenderID, GridPositionComponent)
		event.AttackIndex = attackIndex
		if defenderPos != nil {
			event.TargetInfo.TargetRow = defenderPos.AnchorRow
			event.TargetInfo.TargetCol = defenderPos.AnchorCol
		}

		// Set target mode from attacker's attack type
		targetData := common.GetComponentTypeByID[*TargetRowData](manager, attackerID, TargetRowComponent)
		if targetData != nil {
			event.TargetInfo.TargetMode = targetData.AttackType.String()
		}

		// Apply damage
		applyDamageToUnitByID(defenderID, damage, result, manager)

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
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

// applyDamageToUnitByID applies damage to a unit and tracks it in the combat result
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

	// Note: UpdateSquadDestroyedStatus is now called once per attack in ExecuteSquadAttack
	// instead of per damaged unit for better performance
}

func sumDamageMap(damageMap map[ecs.EntityID]int) int {
	total := 0
	for _, dmg := range damageMap {
		total += dmg
	}
	return total
}

// calculateSquadStatus summarizes squad health for combat log
// Optimized: Uses direct entity lookup in loop instead of GetAttributesByIDWithTag.
func calculateSquadStatus(squadID ecs.EntityID, manager *common.EntityManager) SquadStatus {
	unitIDs := GetUnitIDsInSquad(squadID, manager)
	aliveCount := 0
	totalHP := 0
	totalMaxHP := 0

	for _, unitID := range unitIDs {
		// OPTIMIZATION: Get entity once for attributes check
		entity := common.FindEntityByID(manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
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
// Optimized: Batches defender component lookups (4 calls → 1), optimizes provider loop.
func CalculateTotalCover(defenderID ecs.EntityID, squadmanager *common.EntityManager) float64 {
	// OPTIMIZATION: Get defender entity once, extract all components
	// This replaces 4 separate GetEntityByID calls with just 1
	defenderEntity := common.FindEntityByID(squadmanager, defenderID)
	if defenderEntity == nil {
		return 0.0
	}

	// Check if defender has required components
	if !defenderEntity.HasComponent(GridPositionComponent) || !defenderEntity.HasComponent(SquadMemberComponent) {
		return 0.0
	}

	// Get defender's position and squad from entity
	defenderPos := common.GetComponentType[*GridPositionData](defenderEntity, GridPositionComponent)
	defenderSquadData := common.GetComponentType[*SquadMemberData](defenderEntity, SquadMemberComponent)
	if defenderPos == nil || defenderSquadData == nil {
		return 0.0
	}
	defenderSquadID := defenderSquadData.SquadID

	// Get all units providing cover
	coverProviders := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	// Sum all cover bonuses (stacking additively)
	totalCover := 0.0
	for _, providerID := range coverProviders {
		// OPTIMIZATION: Get provider entity once for all component checks
		providerEntity := common.FindEntityByID(squadmanager, providerID)
		if providerEntity == nil {
			continue
		}

		// Check if provider has cover component
		if !providerEntity.HasComponent(CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*CoverData](providerEntity, CoverComponent)
		if coverData == nil {
			continue
		}

		// Check if provider is active (alive and not stunned)
		isActive := true
		if coverData.RequiresActive {
			attr := common.GetComponentType[*common.Attributes](providerEntity, common.AttributeComponent)
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
// Optimized: Batches component lookups in loop (4 calls → 1 per unit).
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

		// OPTIMIZATION: Get entity once, extract Cover and GridPosition components
		// This replaces 4 separate GetEntityByID calls with just 1
		entity := common.FindEntityByID(squadmanager, unitID)
		if entity == nil {
			continue
		}

		// Check if unit has cover component
		if !entity.HasComponent(CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*CoverData](entity, CoverComponent)
		if coverData == nil {
			continue
		}

		// Get unit's position
		if !entity.HasComponent(GridPositionComponent) {
			continue
		}

		unitPos := common.GetComponentType[*GridPositionData](entity, GridPositionComponent)
		if unitPos == nil {
			continue
		}

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
// Optimized: Batches defender and provider component lookups.
func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) CoverBreakdown {
	breakdown := CoverBreakdown{
		Providers: []CoverProvider{},
	}

	// OPTIMIZATION: Get defender entity once
	defenderEntity := common.FindEntityByID(squadmanager, defenderID)
	if defenderEntity == nil {
		return breakdown
	}

	if !defenderEntity.HasComponent(GridPositionComponent) || !defenderEntity.HasComponent(SquadMemberComponent) {
		return breakdown
	}

	defenderPos := common.GetComponentType[*GridPositionData](defenderEntity, GridPositionComponent)
	defenderSquadData := common.GetComponentType[*SquadMemberData](defenderEntity, SquadMemberComponent)
	if defenderPos == nil || defenderSquadData == nil {
		return breakdown
	}
	defenderSquadID := defenderSquadData.SquadID

	providerIDs := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	totalCover := 0.0
	for _, providerID := range providerIDs {
		// OPTIMIZATION: Get provider entity once, extract all needed components
		providerEntity := common.FindEntityByID(squadmanager, providerID)
		if providerEntity == nil {
			continue
		}

		if !providerEntity.HasComponent(CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*CoverData](providerEntity, CoverComponent)
		if coverData == nil {
			continue
		}

		// Check if active
		isActive := true
		if coverData.RequiresActive {
			attr := common.GetComponentType[*common.Attributes](providerEntity, common.AttributeComponent)
			if attr != nil {
				isActive = attr.CurrentHealth > 0
			}
		}

		coverValue := coverData.GetCoverBonus(isActive)
		if coverValue > 0 {
			// Get provider info for logging - extract from entity we already have
			name := common.GetComponentType[*common.Name](providerEntity, common.NameComponent)
			providerPos := common.GetComponentType[*GridPositionData](providerEntity, GridPositionComponent)

			unitName := "Unknown"
			if name != nil {
				unitName = name.NameStr
			}

			provider := CoverProvider{
				UnitID:     providerID,
				UnitName:   unitName,
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

// displaySquadStatus displays squad status for debugging
// Optimized: Batches component lookups per unit (2 calls → 1).
func displaySquadStatus(squadID ecs.EntityID, squadmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, squadmanager)
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	fmt.Printf("\n%s Status:\n", squadData.Name)

	unitIDs := GetUnitIDsInSquad(squadID, squadmanager)
	alive := 0

	for _, unitID := range unitIDs {
		// OPTIMIZATION: Get entity once, extract Attributes and Name
		// This replaces 2 separate GetEntityByID calls with just 1
		entity := common.FindEntityByID(squadmanager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth > 0 {
			alive++
			name := common.GetComponentType[*common.Name](entity, common.NameComponent)
			unitName := "Unknown"
			if name != nil {
				unitName = name.NameStr
			}
			fmt.Printf("  %s: %d/%d HP\n", unitName, attr.CurrentHealth, attr.MaxHealth)
		}
	}

	fmt.Printf("Total alive: %d\n", alive)
}
