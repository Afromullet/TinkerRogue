package squads

import (
	"game_main/common"
	"game_main/config"

	"github.com/bytearena/ecs"
)

// CombatResult - Unified result type for combat operations
// Contains both combat execution data and orchestration status
type CombatResult struct {
	// Orchestration status (set by combat action system)
	Success         bool
	ErrorReason     string
	TargetDestroyed bool

	// Combat execution data (set by combat calculation logic)
	TotalDamage  int
	UnitsKilled  []ecs.EntityID
	DamageByUnit map[ecs.EntityID]int
	CombatLog    *CombatLog // Contains AttackerSquadName, DefenderSquadName for display
}

// Only units within their attack range of the target squad can participate
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Initialize combat log with squad info
	combatLog := InitializeCombatLog(attackerSquadID, defenderSquadID, squadmanager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		return result
	}

	// Snapshot units that will participate (for logging)
	combatLog.AttackingUnits = SnapshotAttackingUnits(attackerSquadID, combatLog.SquadDistance, squadmanager)
	combatLog.DefendingUnits = SnapshotAllUnits(defenderSquadID, squadmanager)

	// Process each attacking unit
	attackIndex := 0
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

	for _, attackerID := range attackerUnitIDs {
		// Check if unit can attack (alive and in range)
		if !CanUnitAttack(attackerID, combatLog.SquadDistance, squadmanager) {
			continue
		}

		// Get targets for this attacker
		targetIDs := SelectTargetUnits(attackerID, defenderSquadID, squadmanager)

		// Attack each target
		attackIndex = ProcessAttackOnTargets(attackerID, targetIDs, result, combatLog, attackIndex, squadmanager)
	}

	// Apply recorded damage to units (for backward compatibility with tests/simulator)
	ApplyRecordedDamage(result, squadmanager)

	UpdateSquadDestroyedStatus(defenderSquadID, squadmanager)

	// Finalize combat log with summary
	FinalizeCombatLog(result, combatLog, defenderSquadID, attackerSquadID, squadmanager)

	result.CombatLog = combatLog
	return result
}

// ExecuteSquadCounterattack executes a counterattack from defender to attacker
// Counterattacks have reduced damage (50%) and lower hit chance (-20%)
// Only units within their attack range of the target squad can participate
func ExecuteSquadCounterattack(defenderSquadID, attackerSquadID ecs.EntityID, squadmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Initialize combat log
	combatLog := InitializeCombatLog(defenderSquadID, attackerSquadID, squadmanager)
	if combatLog.SquadDistance < 0 {
		result.CombatLog = combatLog
		return result
	}

	// Snapshot units
	combatLog.AttackingUnits = SnapshotAttackingUnits(defenderSquadID, combatLog.SquadDistance, squadmanager)
	combatLog.DefendingUnits = SnapshotAllUnits(attackerSquadID, squadmanager)

	// Process each counterattacking unit
	attackIndex := 0
	defenderUnitIDs := GetUnitIDsInSquad(defenderSquadID, squadmanager)

	for _, counterAttackerID := range defenderUnitIDs {
		// Check if unit can counterattack (alive, in range)
		if !CanUnitAttack(counterAttackerID, combatLog.SquadDistance, squadmanager) {
			continue
		}

		// Get targets (same targeting logic as normal attacks)
		targetIDs := SelectTargetUnits(counterAttackerID, attackerSquadID, squadmanager)

		// Counterattack each target with penalties
		attackIndex = ProcessCounterattackOnTargets(counterAttackerID, targetIDs, result, combatLog, attackIndex, squadmanager)
	}

	// Apply recorded damage to units (for backward compatibility with tests/simulator)
	ApplyRecordedDamage(result, squadmanager)

	UpdateSquadDestroyedStatus(attackerSquadID, squadmanager)

	// Finalize combat log
	FinalizeCombatLog(result, combatLog, attackerSquadID, defenderSquadID, squadmanager)

	result.CombatLog = combatLog
	return result
}

// ProcessCounterattackOnTargets applies counterattack damage with penalties
func ProcessCounterattackOnTargets(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	for _, defenderID := range targetIDs {
		attackIndex++

		// Calculate damage WITH COUNTERATTACK PENALTIES
		damage, event := calculateCounterattackDamage(attackerID, defenderID, manager)

		// Mark as counterattack
		event.IsCounterattack = true

		// Add targeting info
		defenderPos := common.GetComponentTypeByID[*GridPositionData](manager, defenderID, GridPositionComponent)
		event.AttackIndex = attackIndex
		if defenderPos != nil {
			event.TargetInfo.TargetRow = defenderPos.AnchorRow
			event.TargetInfo.TargetCol = defenderPos.AnchorCol
		}

		// Set target mode
		targetData := common.GetComponentTypeByID[*TargetRowData](manager, attackerID, TargetRowComponent)
		if targetData != nil {
			event.TargetInfo.TargetMode = targetData.AttackType.String()
		}

		// Apply damage
		recordDamageToUnit(defenderID, damage, result, manager)

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// calculateCounterattackDamage calculates damage with BOTH penalties:
// 1. Reduced hit chance (-20%)
// 2. Reduced damage (50% multiplier)
func calculateCounterattackDamage(attackerID, defenderID ecs.EntityID, squadmanager *common.EntityManager) (int, *AttackEvent) {
	attackerAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, attackerID, common.AttributeComponent)
	defenderAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, defenderID, common.AttributeComponent)

	event := &AttackEvent{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		IsCounterattack: true,
	}

	if defenderAttr != nil {
		event.DefenderHPBefore = defenderAttr.CurrentHealth
	}

	if attackerAttr == nil || defenderAttr == nil {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// PENALTY #1: Reduced hit chance (-20%)
	baseHitThreshold := attackerAttr.GetHitRate()
	hitThreshold := baseHitThreshold - config.COUNTERATTACK_HIT_PENALTY
	if hitThreshold < 0 {
		hitThreshold = 0
	}

	hitRoll, didHit := rollHit(hitThreshold)
	event.HitResult.HitRoll = hitRoll
	event.HitResult.HitThreshold = hitThreshold

	if !didHit {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Dodge roll (no penalty)
	dodgeThreshold := defenderAttr.GetDodgeChance()
	dodgeRoll, wasDodged := rollDodge(dodgeThreshold)
	event.HitResult.DodgeRoll = dodgeRoll
	event.HitResult.DodgeThreshold = dodgeThreshold

	if wasDodged {
		event.HitResult.Type = HitTypeDodge
		return 0, event
	}

	// Calculate base damage
	baseDamage := attackerAttr.GetPhysicalDamage()
	event.BaseDamage = baseDamage
	event.CritMultiplier = 1.0

	// Crit roll (no penalty)
	critThreshold := attackerAttr.GetCritChance()
	critRoll, wasCrit := rollCrit(critThreshold)
	event.HitResult.CritRoll = critRoll
	event.HitResult.CritThreshold = critThreshold

	if wasCrit {
		baseDamage = int(float64(baseDamage) * 1.5)
		event.CritMultiplier = 1.5
		event.HitResult.Type = HitTypeCritical
	} else {
		event.HitResult.Type = HitTypeCounterattack
	}

	// PENALTY #2: Reduced damage (50%)
	baseDamage = int(float64(baseDamage) * config.COUNTERATTACK_DAMAGE_MULTIPLIER)
	if baseDamage < 1 {
		baseDamage = 1
	}

	// Apply resistance (normal)
	resistance := defenderAttr.GetPhysicalResistance()
	event.ResistanceAmount = resistance
	totalDamage := baseDamage - resistance
	if totalDamage < 1 {
		totalDamage = 1
	}

	// Apply cover (normal)
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

// InitializeCombatLog creates the combat log structure with squad information
func InitializeCombatLog(attackerSquadID, defenderSquadID ecs.EntityID, manager *common.EntityManager) *CombatLog {
	return &CombatLog{
		AttackerSquadID:   attackerSquadID,
		DefenderSquadID:   defenderSquadID,
		AttackerSquadName: GetSquadName(attackerSquadID, manager),
		DefenderSquadName: GetSquadName(defenderSquadID, manager),
		SquadDistance:     GetSquadDistance(attackerSquadID, defenderSquadID, manager),
		AttackEvents:      []AttackEvent{},
		AttackingUnits:    []UnitSnapshot{},
		DefendingUnits:    []UnitSnapshot{},
	}
}

// snapshotUnits captures unit info before combat for logging
// If squadDistance >= 0, only includes units that can attack at that distance
// If squadDistance < 0, includes all units (used for defenders)
func snapshotUnits(squadID ecs.EntityID, squadDistance int, filterByRange bool, manager *common.EntityManager) []UnitSnapshot {
	var snapshots []UnitSnapshot
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		// Check if unit can participate (if filtering by range)
		if filterByRange && !CanUnitAttack(unitID, squadDistance, manager) {
			continue
		}

		entity := manager.FindEntityByID(unitID)
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

// SnapshotAttackingUnits captures attacking unit info before combat for logging
func SnapshotAttackingUnits(squadID ecs.EntityID, squadDistance int, manager *common.EntityManager) []UnitSnapshot {
	return snapshotUnits(squadID, squadDistance, true, manager)
}

// SnapshotAllUnits captures all units in a squad for logging (used for defenders)
func SnapshotAllUnits(squadID ecs.EntityID, manager *common.EntityManager) []UnitSnapshot {
	return snapshotUnits(squadID, -1, false, manager)
}

// FinalizeCombatLog calculates summary statistics for the combat log
// Now takes both attacker and defender IDs to handle counterattack deaths properly
func FinalizeCombatLog(result *CombatResult, log *CombatLog, defenderSquadID, attackerSquadID ecs.EntityID, manager *common.EntityManager) {
	result.TotalDamage = sumDamageMap(result.DamageByUnit)
	log.TotalDamage = result.TotalDamage
	log.UnitsKilled = len(result.UnitsKilled)
	log.DefenderStatus = calculateSquadStatus(defenderSquadID, manager)
}

// CanUnitAttack checks if a unit is alive, can act, and within attack range
func CanUnitAttack(attackerID ecs.EntityID, squadDistance int, manager *common.EntityManager) bool {
	entity := manager.FindEntityByID(attackerID)
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
func SelectTargetUnits(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	entity := manager.FindEntityByID(attackerID)
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

// selectMeleeColumnTargets targets column directly across from attacker, piercing to next column if empty
// Always targets all units in the column (piercing attack)
func selectMeleeColumnTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerEntity := manager.FindEntityByID(attackerID)
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
		targets := getUnitsInColumn(defenderSquadID, col, manager)

		if len(targets) > 0 {
			return targets // Return all units in the column
		}
	}

	return []ecs.EntityID{}
}

// selectRangedTargets targets same row as attacker (all units), with fallback logic
func selectRangedTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerEntity := manager.FindEntityByID(attackerID)
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
func getUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	// Check all columns in the row
	for col := 0; col <= 2; col++ {
		cellUnits := GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				entity := manager.FindEntityByID(unitID)
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

// Helper: Get all ALIVE units in a specific column
func getUnitsInColumn(squadID ecs.EntityID, col int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	// Check all rows in the column
	for row := 0; row <= 2; row++ {
		cellUnits := GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				entity := manager.FindEntityByID(unitID)
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
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
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

// ProcessAttackOnTargets applies damage to all targets and creates combat events
// Returns the updated attack index
func ProcessAttackOnTargets(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
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
		recordDamageToUnit(defenderID, damage, result, manager)

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// calculateUnitDamageByID calculates damage using new attribute system and returns detailed event data
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *common.EntityManager) (int, *AttackEvent) {
	attackerAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, attackerID, common.AttributeComponent)
	defenderAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, defenderID, common.AttributeComponent)

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
	hitRoll, didHit := rollHit(hitThreshold)
	event.HitResult.HitRoll = hitRoll
	event.HitResult.HitThreshold = hitThreshold

	if !didHit {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Dodge roll
	dodgeThreshold := defenderAttr.GetDodgeChance()
	dodgeRoll, wasDodged := rollDodge(dodgeThreshold)
	event.HitResult.DodgeRoll = dodgeRoll
	event.HitResult.DodgeThreshold = dodgeThreshold

	if wasDodged {
		event.HitResult.Type = HitTypeDodge
		return 0, event
	}

	// Get attacker's attack type to determine damage formula
	attackerTargetData := common.GetComponentTypeByID[*TargetRowData](squadmanager, attackerID, TargetRowComponent)

	// Calculate base damage based on attack type
	var baseDamage int
	var resistance int

	if attackerTargetData != nil && attackerTargetData.AttackType == AttackTypeMagic {
		// Magic damage path
		baseDamage = attackerAttr.GetMagicDamage()
		resistance = defenderAttr.GetMagicDefense()
	} else {
		// Physical damage path (Melee, Ranged, or fallback)
		baseDamage = attackerAttr.GetPhysicalDamage()
		resistance = defenderAttr.GetPhysicalResistance()
	}

	event.BaseDamage = baseDamage
	event.CritMultiplier = 1.0

	// Crit roll
	critThreshold := attackerAttr.GetCritChance()
	critRoll, wasCrit := rollCrit(critThreshold)
	event.HitResult.CritRoll = critRoll
	event.HitResult.CritThreshold = critThreshold

	if wasCrit {
		baseDamage = int(float64(baseDamage) * 1.5)
		event.CritMultiplier = 1.5
		event.HitResult.Type = HitTypeCritical
	} else {
		event.HitResult.Type = HitTypeNormal
	}

	// Apply resistance (now type-appropriate)
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

// rollHit returns the roll value and whether the attack hit
// Returns: (roll value, hit succeeded)
func rollHit(hitRate int) (roll int, hit bool) {
	roll = common.GetDiceRoll(100)
	hit = roll <= hitRate
	return
}

// rollCrit returns the roll value and whether the attack is a critical hit
// Returns: (roll value, crit succeeded)
func rollCrit(critChance int) (roll int, crit bool) {
	roll = common.GetDiceRoll(100)
	crit = roll <= critChance
	return
}

// rollDodge returns the roll value and whether the attack was dodged
// Returns: (roll value, dodge succeeded)
func rollDodge(dodgeChance int) (roll int, dodged bool) {
	roll = common.GetDiceRoll(100)
	dodged = roll <= dodgeChance
	return
}

// recordDamageToUnit records damage in the combat result without modifying HP (pure calculation)
func recordDamageToUnit(unitID ecs.EntityID, damage int, result *CombatResult, squadmanager *common.EntityManager) {
	// Accumulate damage (in case unit is hit multiple times)
	result.DamageByUnit[unitID] += damage

	// Check if unit would be killed (prediction based on current HP)
	attr := common.GetComponentTypeByID[*common.Attributes](squadmanager, unitID, common.AttributeComponent)
	if attr != nil {
		totalDamageTaken := result.DamageByUnit[unitID]
		if attr.CurrentHealth-totalDamageTaken <= 0 {
			// Only add to UnitsKilled once
			alreadyMarked := false
			for _, killedID := range result.UnitsKilled {
				if killedID == unitID {
					alreadyMarked = true
					break
				}
			}
			if !alreadyMarked {
				result.UnitsKilled = append(result.UnitsKilled, unitID)
			}
		}
	}
}

// ApplyRecordedDamage applies all recorded damage from result.DamageByUnit to actual unit HP
// This is called during orchestration phase after all combat calculations are complete
func ApplyRecordedDamage(result *CombatResult, squadmanager *common.EntityManager) {
	for unitID, damage := range result.DamageByUnit {
		attr := common.GetComponentTypeByID[*common.Attributes](squadmanager, unitID, common.AttributeComponent)
		if attr == nil {
			continue
		}
		attr.CurrentHealth -= damage
	}
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
		entity := manager.FindEntityByID(unitID)
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
		entity := squadmanager.FindEntityByID(unitID)
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
// Similar to CalculateTotalCover but includes provider details.
func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) CoverBreakdown {
	breakdown := CoverBreakdown{
		Providers: []CoverProvider{},
	}
	defenderEntity := squadmanager.FindEntityByID(defenderID)
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
		providerEntity := squadmanager.FindEntityByID(providerID)
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
