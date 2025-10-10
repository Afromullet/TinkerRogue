# Squad Combat System - Corrected (Uses Native ECS Entity IDs)



```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/randgen"
	"game_main/squad"
	"math/rand"
)

// CombatResult - ✅ Uses ecs.EntityID (native type) instead of entity pointers
type CombatResult struct {
	TotalDamage  int
	UnitsKilled  []ecs.EntityID           // ✅ Native IDs
	DamageByUnit map[ecs.EntityID]int     // ✅ Native IDs
}

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, ecsmanager *common.EntityManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Query for attacker unit IDs (not pointers!)
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, ecsmanager)

	// Process each attacker unit
	for _, attackerID := range attackerUnitIDs {
		attackerUnit := FindUnitByID(attackerID, ecsmanager)
		if attackerUnit == nil {
			continue
		}

		// Check if unit is alive
		attackerAttr := common.GetAttributes(attackerUnit)
		if attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Get targeting data
		if !attackerUnit.HasComponent(squad.TargetRowComponent) {
			continue
		}

		targetRowData := common.GetComponentType[*squad.TargetRowData](attackerUnit, squad.TargetRowComponent)

		var actualTargetIDs []ecs.EntityID

		// Handle targeting based on mode
		if targetRowData.Mode == squad.TargetModeCellBased {
			// Cell-based targeting: hit specific grid cells
			for _, cell := range targetRowData.TargetCells {
				row, col := cell[0], cell[1]
				cellTargetIDs := GetUnitIDsAtGridPosition(defenderSquadID, row, col, ecsmanager)
				actualTargetIDs = append(actualTargetIDs, cellTargetIDs...)
			}
		} else {
			// Row-based targeting: hit entire row(s)
			for _, targetRow := range targetRowData.TargetRows {
				targetIDs := GetUnitIDsInRow(defenderSquadID, targetRow, ecsmanager)

				if len(targetIDs) == 0 {
					continue
				}

				if targetRowData.IsMultiTarget {
					maxTargets := targetRowData.MaxTargets
					if maxTargets == 0 || maxTargets > len(targetIDs) {
						actualTargetIDs = append(actualTargetIDs, targetIDs...)
					} else {
						actualTargetIDs = append(actualTargetIDs, selectRandomTargetIDs(targetIDs, maxTargets)...)
					}
				} else {
					actualTargetIDs = append(actualTargetIDs, selectLowestHPTargetID(targetIDs, ecsmanager))
				}
			}
		}

		// Apply damage to each selected target
		for _, defenderID := range actualTargetIDs {
			damage := calculateUnitDamageByID(attackerID, defenderID, ecsmanager)
			applyDamageToUnitByID(defenderID, damage, result, ecsmanager)
		}
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	return result
}

// calculateUnitDamageByID - ✅ Works with ecs.EntityID
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, ecsmanager *common.EntityManager) int {
	attackerUnit := FindUnitByID(attackerID, ecsmanager)
	defenderUnit := FindUnitByID(defenderID, ecsmanager)

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
	if attackerUnit.HasComponent(squad.UnitRoleComponent) {
		roleData := common.GetComponentType[*squad.UnitRoleData](attackerUnit, squad.UnitRoleComponent)
		baseDamage = applyRoleModifier(baseDamage, roleData.Role)
	}

	// Apply defense
	totalDamage := baseDamage - defenderAttr.TotalProtection
	if totalDamage < 1 {
		totalDamage = 1 // Minimum damage
	}

	// Apply cover (damage reduction from units in front)
	coverReduction := CalculateTotalCover(defenderID, ecsmanager)
	if coverReduction > 0.0 {
		totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))
		if totalDamage < 1 {
			totalDamage = 1 // Minimum damage even with cover
		}
	}

	return totalDamage
}

// applyRoleModifier adjusts damage based on unit role
func applyRoleModifier(damage int, role squad.UnitRole) int {
	switch role {
	case squad.RoleTank:
		return int(float64(damage) * 0.8) // -20% (tanks don't deal high damage)
	case squad.RoleDPS:
		return int(float64(damage) * 1.3) // +30% (damage dealers)
	case squad.RoleSupport:
		return int(float64(damage) * 0.6) // -40% (support units are weak attackers)
	default:
		return damage
	}
}

// applyDamageToUnitByID - ✅ Uses ecs.EntityID
func applyDamageToUnitByID(unitID ecs.EntityID, damage int, result *CombatResult, ecsmanager *common.EntityManager) {
	unit := FindUnitByID(unitID, ecsmanager)
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

// selectLowestHPTargetID - ✅ Works with ecs.EntityID
func selectLowestHPTargetID(unitIDs []ecs.EntityID, ecsmanager *common.EntityManager) ecs.EntityID {
	if len(unitIDs) == 0 {
		return 0
	}

	lowestID := unitIDs[0]
	lowestUnit := FindUnitByID(lowestID, ecsmanager)
	if lowestUnit == nil {
		return 0
	}
	lowestHP := common.GetAttributes(lowestUnit).CurrentHealth

	for _, unitID := range unitIDs[1:] {
		unit := FindUnitByID(unitID, ecsmanager)
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
func CalculateTotalCover(defenderID ecs.EntityID, ecsmanager *common.EntityManager) float64 {
	defenderUnit := FindUnitByID(defenderID, ecsmanager)
	if defenderUnit == nil {
		return 0.0
	}

	// Get defender's position and squad
	if !defenderUnit.HasComponent(squad.GridPositionComponent) || !defenderUnit.HasComponent(squad.SquadMemberComponent) {
		return 0.0
	}

	defenderPos := common.GetComponentType[*squad.GridPositionData](defenderUnit, squad.GridPositionComponent)
	defenderSquadData := common.GetComponentType[*squad.SquadMemberData](defenderUnit, squad.SquadMemberComponent)
	defenderSquadID := defenderSquadData.SquadID

	// Get all units providing cover
	coverProviders := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, ecsmanager)

	// Sum all cover bonuses (stacking additively)
	totalCover := 0.0
	for _, providerID := range coverProviders {
		providerUnit := FindUnitByID(providerID, ecsmanager)
		if providerUnit == nil {
			continue
		}

		// Check if provider has cover component
		if !providerUnit.HasComponent(squad.CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*squad.CoverData](providerUnit, squad.CoverComponent)

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
func GetCoverProvidersFor(defenderID ecs.EntityID, defenderSquadID ecs.EntityID, defenderPos *squad.GridPositionData, ecsmanager *common.EntityManager) []ecs.EntityID {
	var providers []ecs.EntityID

	// Get all columns the defender occupies
	defenderCols := make(map[int]bool)
	for c := defenderPos.AnchorCol; c < defenderPos.AnchorCol+defenderPos.Width && c < 3; c++ {
		defenderCols[c] = true
	}

	// Get all units in the same squad
	allUnitIDs := GetUnitIDsInSquad(defenderSquadID, ecsmanager)

	for _, unitID := range allUnitIDs {
		// Don't provide cover to yourself
		if unitID == defenderID {
			continue
		}

		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		// Check if unit has cover component
		if !unit.HasComponent(squad.CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*squad.CoverData](unit, squad.CoverComponent)

		// Get unit's position
		if !unit.HasComponent(squad.GridPositionComponent) {
			continue
		}

		unitPos := common.GetComponentType[*squad.GridPositionData](unit, squad.GridPositionComponent)

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
```

---



### File: `systems/squadabilities.go`

**Ability trigger system - queries for leaders, uses native entity IDs**

```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/squad"
)

// CheckAndTriggerAbilities - ✅ Works with ecs.EntityID
func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	// Find leader via query (not stored reference)
	leaderID := GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return // No leader, no abilities
	}

	leaderEntity := FindUnitByID(leaderID, ecsmanager)
	if leaderEntity == nil {
		return
	}

	if !leaderEntity.HasComponent(squad.AbilitySlotComponent) {
		return
	}

	if !leaderEntity.HasComponent(squad.CooldownTrackerComponent) {
		return
	}

	abilityData := common.GetComponentType[*squad.AbilitySlotData](leaderEntity, squad.AbilitySlotComponent)
	cooldownData := common.GetComponentType[*squad.CooldownTrackerData](leaderEntity, squad.CooldownTrackerComponent)

	// Check each ability slot
	for i := 0; i < 4; i++ {
		slot := &abilityData.Slots[i]

		if !slot.IsEquipped || cooldownData.Cooldowns[i] > 0 || slot.HasTriggered {
			continue
		}

		// Evaluate trigger condition
		triggered := evaluateTrigger(slot, squadID, ecsmanager)
		if !triggered {
			continue
		}

		// Execute ability
		executeAbility(slot, squadID, ecsmanager)

		// Set cooldown
		cooldownData.Cooldowns[i] = cooldownData.MaxCooldowns[i]

		// Mark as triggered
		slot.HasTriggered = true
	}

	// Tick down cooldowns
	for i := 0; i < 4; i++ {
		if cooldownData.Cooldowns[i] > 0 {
			cooldownData.Cooldowns[i]--
		}
	}
}

// evaluateTrigger checks if a condition is met
func evaluateTrigger(slot *squad.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return false
	}

	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	switch slot.TriggerType {
	case squad.TRIGGER_SQUAD_HP_BELOW:
		avgHP := calculateAverageHP(squadID, ecsmanager)
		return avgHP < slot.Threshold

	case squad.TRIGGER_TURN_COUNT:
		return squadData.TurnCount == int(slot.Threshold)

	case squad.TRIGGER_COMBAT_START:
		return squadData.TurnCount == 1

	case squad.TRIGGER_ENEMY_COUNT:
		enemyCount := countEnemySquads(ecsmanager)
		return float64(enemyCount) >= slot.Threshold

	case squad.TRIGGER_MORALE_BELOW:
		return float64(squadData.Morale) < slot.Threshold

	default:
		return false
	}
}

// calculateAverageHP computes the squad's average HP as a percentage (0.0 - 1.0)
func calculateAverageHP(squadID ecs.EntityID, ecsmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	totalHP := 0
	totalMaxHP := 0

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		totalHP += attr.CurrentHealth
		totalMaxHP += attr.MaxHealth
	}

	if totalMaxHP == 0 {
		return 0.0
	}

	return float64(totalHP) / float64(totalMaxHP)
}

// countEnemySquads counts the number of enemy squads on the map
func countEnemySquads(ecsmanager *common.EntityManager) int {
	count := 0
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

		// Assume enemy squads don't have "Player" prefix (adjust based on your naming)
		if len(squadData.Name) > 0 && squadData.Name[0] != 'P' {
			count++
		}
	}
	return count
}

// executeAbility triggers the ability effect
// Data-driven approach: reads ability params, applies effects
func executeAbility(slot *squad.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	params := squad.GetAbilityParams(slot.AbilityType)

	switch slot.AbilityType {
	case squad.ABILITY_RALLY:
		applyRallyEffect(squadID, params, ecsmanager)
	case squad.ABILITY_HEAL:
		applyHealEffect(squadID, params, ecsmanager)
	case squad.ABILITY_BATTLE_CRY:
		applyBattleCryEffect(squadID, params, ecsmanager)
	case squad.ABILITY_FIREBALL:
		applyFireballEffect(squadID, params, ecsmanager)
	}
}

// --- Ability Implementations (Data-Driven) ---

// RallyEffect: Temporary damage buff to own squad
func applyRallyEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			attr.DamageBonus += params.DamageBonus
			// TODO: Track buff duration (requires turn/buff system)
		}
	}

	fmt.Printf("[ABILITY] Rally! +%d damage for %d turns\n", params.DamageBonus, params.Duration)
}

// HealEffect: Restore HP to own squad
func applyHealEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	healed := 0
	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		// Cap at max HP
		attr.CurrentHealth += params.HealAmount
		if attr.CurrentHealth > attr.MaxHealth {
			attr.CurrentHealth = attr.MaxHealth
		}
		healed++
	}

	fmt.Printf("[ABILITY] Healing Aura! %d units restored %d HP\n", healed, params.HealAmount)
}

// BattleCryEffect: First turn buff (morale + damage)
func applyBattleCryEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	// Boost morale
	squadData.Morale += params.MoraleBonus

	// Boost damage
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)
	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth > 0 {
			attr.DamageBonus += params.DamageBonus
		}
	}

	fmt.Printf("[ABILITY] Battle Cry! Morale and damage increased!\n")
}

// FireballEffect: AOE damage to enemy squad
func applyFireballEffect(squadID ecs.EntityID, params squad.AbilityParams, ecsmanager *common.EntityManager) {
	// Find first enemy squad (simplified targeting)
	var targetSquadID ecs.EntityID
	for _, result := range ecsmanager.World.Query(squad.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

		if squadData.SquadID != squadID {
			targetSquadID = squadData.SquadID
			break
		}
	}

	if targetSquadID == 0 {
		return // No targets
	}

	unitIDs := GetUnitIDsInSquad(targetSquadID, ecsmanager)
	killed := 0

	for _, unitID := range unitIDs {
		unit := FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		attr := common.GetAttributes(unit)
		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.CurrentHealth -= params.BaseDamage
		if attr.CurrentHealth <= 0 {
			killed++
		}
	}

	fmt.Printf("[ABILITY] Fireball! %d damage dealt, %d units killed\n", params.BaseDamage, killed)
}
```

---


```go
package systems

import (
	"fmt"
	"github.com/bytearena/ecs"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"
	"game_main/squad"
)


// CreateSquadFromTemplate - ✅ Returns ecs.EntityID (native type)
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation squad.FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []UnitTemplate,
) ecs.EntityID {

	// Create squad entity
	squadEntity := ecsmanager.World.NewEntity()

	// ✅ Get native entity ID
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(squad.SquadComponent, &squad.SquadData{
		SquadID:   squadID,
		Name:      squadName,
		Formation: formation,
		Morale:    100,
		TurnCount: 0,
		MaxUnits:  9,
	})
	squadEntity.AddComponent(common.PositionComponent, &worldPos)

	// Track occupied grid positions (keyed by "row,col")
	occupied := make(map[string]bool)

	// Create units
	for _, template := range unitTemplates {
		// Default to 1x1 if not specified
		width := template.GridWidth
		if width == 0 {
			width = 1
		}
		height := template.GridHeight
		if height == 0 {
			height = 1
		}

		// Validate that unit fits within 3x3 grid
		if template.GridRow < 0 || template.GridCol < 0 {
			fmt.Printf("Warning: Invalid anchor position (%d, %d), skipping\n", template.GridRow, template.GridCol)
			continue
		}
		if template.GridRow+height > 3 || template.GridCol+width > 3 {
			fmt.Printf("Warning: Unit extends outside grid (anchor=%d,%d, size=%dx%d), skipping\n",
				template.GridRow, template.GridCol, width, height)
			continue
		}

		// Check if ANY cell this unit would occupy is already occupied
		canPlace := true
		var cellsToOccupy [][2]int
		for r := template.GridRow; r < template.GridRow+height; r++ {
			for c := template.GridCol; c < template.GridCol+width; c++ {
				key := fmt.Sprintf("%d,%d", r, c)
				if occupied[key] {
					canPlace = false
					fmt.Printf("Warning: Cell (%d,%d) already occupied, cannot place %dx%d unit at (%d,%d)\n",
						r, c, width, height, template.GridRow, template.GridCol)
					break
				}
				cellsToOccupy = append(cellsToOccupy, [2]int{r, c})
			}
			if !canPlace {
				break
			}
		}

		if !canPlace {
			continue
		}

		// Create unit entity
		unitEntity := entitytemplates.CreateEntityFromTemplate(
			*ecsmanager,
			template.EntityConfig,
			template.EntityData,
		)

		// Add squad membership (uses ID, not entity pointer)
		unitEntity.AddComponent(squad.SquadMemberComponent, &squad.SquadMemberData{
			SquadID: squadID,  // ✅ Native entity ID
		})

		// Add grid position (supports multi-cell)
		unitEntity.AddComponent(squad.GridPositionComponent, &squad.GridPositionData{
			AnchorRow: template.GridRow,
			AnchorCol: template.GridCol,
			Width:     width,
			Height:    height,
		})

		// Add role
		unitEntity.AddComponent(squad.UnitRoleComponent, &squad.UnitRoleData{
			Role: template.Role,
		})

		// Add targeting data (supports both row-based and cell-based modes)
		targetMode := squad.TargetModeRowBased
		if template.TargetMode == "cell" {
			targetMode = squad.TargetModeCellBased
		}

		unitEntity.AddComponent(squad.TargetRowComponent, &squad.TargetRowData{
			Mode:          targetMode,
			TargetRows:    template.TargetRows,
			IsMultiTarget: template.IsMultiTarget,
			MaxTargets:    template.MaxTargets,
			TargetCells:   template.TargetCells,
		})

		// Add cover component if unit provides cover
		if template.CoverValue > 0.0 {
			unitEntity.AddComponent(squad.CoverComponent, &squad.CoverData{
				CoverValue:     template.CoverValue,
				CoverRange:     template.CoverRange,
				RequiresActive: template.RequiresActive,
			})
		}

		// Add leader component if needed
		if template.IsLeader {
			unitEntity.AddComponent(squad.LeaderComponent, &squad.LeaderData{
				Leadership: 10,
				Experience: 0,
			})

			// Add ability slots
			unitEntity.AddComponent(squad.AbilitySlotComponent, &squad.AbilitySlotData{
				Slots: [4]squad.AbilitySlot{},
			})

			// Add cooldown tracker
			unitEntity.AddComponent(squad.CooldownTrackerComponent, &squad.CooldownTrackerData{
				Cooldowns:    [4]int{0, 0, 0, 0},
				MaxCooldowns: [4]int{0, 0, 0, 0},
			})
		}

		// Mark ALL cells as occupied
		for _, cell := range cellsToOccupy {
			key := fmt.Sprintf("%d,%d", cell[0], cell[1])
			occupied[key] = true
		}
	}

	return squadID // ✅ Return native entity ID
}





// EquipAbilityToLeader - ✅ Accepts ecs.EntityID (native type)
func EquipAbilityToLeader(
	leaderEntityID ecs.EntityID,
	slotIndex int,
	abilityType squad.AbilityType,
	triggerType squad.TriggerType,
	threshold float64,
	ecsmanager *common.EntityManager,
) error {

	if slotIndex < 0 || slotIndex >= 4 {
		return fmt.Errorf("invalid slot %d", slotIndex)
	}

	leaderEntity := FindUnitByID(leaderEntityID, ecsmanager)
	if leaderEntity == nil {
		return fmt.Errorf("leader entity not found")
	}

	if !leaderEntity.HasComponent(squad.LeaderComponent) {
		return fmt.Errorf("entity is not a leader")
	}

	// Get ability params
	params := squad.GetAbilityParams(abilityType)

	// Update ability slot
	abilityData := common.GetComponentType[*squad.AbilitySlotData](leaderEntity, squad.AbilitySlotComponent)
	abilityData.Slots[slotIndex] = squad.AbilitySlot{
		AbilityType:  abilityType,
		TriggerType:  triggerType,
		Threshold:    threshold,
		HasTriggered: false,
		IsEquipped:   true,
	}

	// Update cooldown tracker
	cooldownData := common.GetComponentType[*squad.CooldownTrackerData](leaderEntity, squad.CooldownTrackerComponent)
	cooldownData.MaxCooldowns[slotIndex] = params.BaseCooldown
	cooldownData.Cooldowns[slotIndex] = 0

	return nil
}

// MoveUnitInSquad - ✅ Accepts ecs.EntityID (native type)
// ✅ Supports multi-cell units - validates all cells at new position
func MoveUnitInSquad(unitEntityID ecs.EntityID, newRow, newCol int, ecsmanager *common.EntityManager) error {
	unitEntity := FindUnitByID(unitEntityID, ecsmanager)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(squad.SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	gridPosData := common.GetComponentType[*squad.GridPositionData](unitEntity, squad.GridPositionComponent)

	// Validate new anchor position is in bounds
	if newRow < 0 || newCol < 0 {
		return fmt.Errorf("invalid anchor position (%d, %d)", newRow, newCol)
	}

	// Validate unit fits within grid at new position
	if newRow+gridPosData.Height > 3 || newCol+gridPosData.Width > 3 {
		return fmt.Errorf("unit would extend outside grid at position (%d, %d) with size %dx%d",
			newRow, newCol, gridPosData.Width, gridPosData.Height)
	}

	memberData := common.GetComponentType[*squad.SquadMemberData](unitEntity, squad.SquadMemberComponent)

	// Check if ANY cell at new position is occupied (excluding this unit itself)
	for r := newRow; r < newRow+gridPosData.Height; r++ {
		for c := newCol; c < newCol+gridPosData.Width; c++ {
			existingUnitIDs := GetUnitIDsAtGridPosition(memberData.SquadID, r, c, ecsmanager)
			for _, existingID := range existingUnitIDs {
				if existingID != unitEntityID {
					return fmt.Errorf("cell (%d, %d) already occupied by another unit", r, c)
				}
			}
		}
	}

	// Update grid position (anchor only, width/height remain the same)
	gridPosData.AnchorRow = newRow
	gridPosData.AnchorCol = newCol

	return nil
}
```

### Formation Presets

**File:** `squad/formations.go`

```go
package squad

// FormationPreset defines a quick-start squad configuration
type FormationPreset struct {
	Positions []FormationPosition
}

type FormationPosition struct {
	AnchorRow int
	AnchorCol int
	Role      UnitRole
	Target    []int
}

// GetFormationPreset returns predefined formation templates
func GetFormationPreset(formation FormationType) FormationPreset {
	switch formation {
	case FormationBalanced:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleSupport, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 0, Role: RoleDPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: RoleDPS, Target: []int{2}},
			},
		}

	case FormationDefensive:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 0, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 0, AnchorCol: 2, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleSupport, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleDPS, Target: []int{2}},
			},
		}

	case FormationOffensive:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 1, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 1, AnchorCol: 2, Role: RoleDPS, Target: []int{1}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleSupport, Target: []int{2}},
			},
		}

	case FormationRanged:
		return FormationPreset{
			Positions: []FormationPosition{
				{AnchorRow: 0, AnchorCol: 1, Role: RoleTank, Target: []int{0}},
				{AnchorRow: 1, AnchorCol: 0, Role: RoleDPS, Target: []int{1, 2}},
				{AnchorRow: 1, AnchorCol: 2, Role: RoleDPS, Target: []int{1, 2}},
				{AnchorRow: 2, AnchorCol: 0, Role: RoleDPS, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 1, Role: RoleSupport, Target: []int{2}},
				{AnchorRow: 2, AnchorCol: 2, Role: RoleDPS, Target: []int{2}},
			},
		}

	default:
		return FormationPreset{Positions: []FormationPosition{}}
	}
}
```

---

## Implementation Phases

### Phase 1: Core Components and Data Structures (6-8 hours)

**Deliverables:**
- `squad/components.go` - All component definitions (using ecs.EntityID)
- `squad/tags.go` - Tag initialization
- Component registration in game initialization

**Steps:**
1. Create `squad/` package directory
2. Define all component types and data structures (with ecs.EntityID)
3. Create `InitSquadComponents()` function
4. Add component registration to `game_main/main.go`
5. Create `InitSquadTags()` and register tags
6. Build and verify no compilation errors

**Code Example:**
```go
// In game_main/main.go, add to initialization:
func initializeGame() {
	// ... existing initialization

	// Register squad components
	squad.InitSquadComponents(ecsmanager.World)
	squad.InitSquadTags()

	// ... rest of initialization
}
```

**Testing:**
- Build succeeds: `go build -o game_main/game_main.exe game_main/*.go`
- No runtime panics on startup

### Phase 2: Query System (4-6 hours)

**Deliverables:**
- `systems/squadqueries.go` - Query functions for squad relationships

**Steps:**
1. Implement `GetUnitIDsInSquad()` (returns []ecs.EntityID)
2. Implement `GetSquadEntity()` (returns *ecs.Entity from query)
3. Implement `GetUnitIDsAtGridPosition()`
4. Implement `GetUnitIDsInRow()`
5. Implement `GetLeaderID()`
6. Implement `IsSquadDestroyed()`
7. Implement `FindUnitByID()` helper

**Testing:**
```go
func TestSquadQueries() {
	// Create test squad with units
	squadID := systems.CreateSquadFromTemplate(...)

	// Test queries
	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)
	assert.Greater(t, len(unitIDs), 0)

	frontRowIDs := systems.GetUnitIDsInRow(squadID, 0, ecsmanager)
	assert.Greater(t, len(frontRowIDs), 0)

	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	assert.NotEqual(t, ecs.EntityID(0), leaderID)
}
```

### Phase 3: Row-Based Combat System (8-10 hours)

**Deliverables:**
- `systems/squadcombat.go` - ExecuteSquadAttack, damage calculation, cover system
- Row-based targeting logic
- Integration with existing `PerformAttack()` concepts

**Steps:**
1. Implement `ExecuteSquadAttack()` function
2. Implement `calculateUnitDamageByID()` (adapt existing combat logic)
3. Implement `applyRoleModifier()` for Tank/DPS/Support
4. Implement `CalculateTotalCover()` and `GetCoverProvidersFor()` for cover system
5. Integrate cover reduction into damage calculation
6. Implement targeting logic (single-target vs multi-target)
7. Implement helper functions (selectLowestHPTargetID, selectRandomTargetIDs)
8. Add death tracking and unit removal

**Testing:**
- Create two squads with different roles
- Execute combat and verify damage distribution
- Verify row targeting (front row hit first, back row protected)
- Verify role modifiers apply correctly
- **Test cover mechanics:**
  - Verify front-line tanks reduce damage to back-line units
  - Test stacking cover (multiple units in same column)
  - Test dead units don't provide cover
  - Test multi-cell units providing cover to multiple columns
  - Test cover range limitations
- Test edge cases (empty rows, all dead, etc.)

### Phase 4: Automated Ability System (6-8 hours)

**Deliverables:**
- `systems/squadabilities.go` - Ability trigger checking and execution
- Built-in abilities: Rally, Heal, BattleCry, Fireball

**Steps:**
1. Implement `CheckAndTriggerAbilities()`
2. Implement `evaluateTrigger()` for condition checking
3. Implement `executeAbility()` for ability dispatch
4. Implement 4 example abilities (Rally, Heal, BattleCry, Fireball)
5. Add cooldown tick system
6. Implement `EquipAbilityToLeader()`

**Testing:**
- Equip abilities with different trigger conditions
- Simulate combat and verify triggers fire correctly
- Test cooldown system
- Verify condition thresholds (HP < 50%, turn count, etc.)

### Phase 5: Squad Creation and Management (4-6 hours)

**Deliverables:**
- `systems/squadcreation.go` - CreateSquadFromTemplate, AddUnitToSquad, etc.
- `squad/formations.go` - Formation presets

**Steps:**
1. Implement `CreateSquadFromTemplate()` (returns ecs.EntityID!)
2. Implement `AddUnitToSquad()`
3. Implement `RemoveUnitFromSquad()`
4. Implement `MoveUnitInSquad()`
5. Implement `EquipAbilityToLeader()`
6. Define formation presets in `squad/formations.go`

**Testing:**
- Create squads with varying sizes (1-9 units)
- Test with sparse formations (empty grid slots)
- Verify leader assignment
- Test formation presets
- Verify native entity IDs are returned

### Phase 6: Integration with Existing Systems (8-10 hours)

**Deliverables:**
- Updated `input/combatcontroller.go` for squad selection
- Updated `spawning/spawnmonsters.go` for squad spawning
- Updated rendering to show squad grid
- Simple cleanup (entity.Remove())

**Steps:**
1. Modify `CombatController.HandleClick()` to detect squad entities
2. Implement squad selection/targeting flow
3. Update spawning system to create enemy squads
4. Add squad rendering with grid overlay
5. Integrate `CheckAndTriggerAbilities()` into turn system
6. Update input handling for squad movement (on map)
7. Add squad destruction cleanup (just entity.Remove())

**Code Example: Combat Controller (Updated)**
```go
// File: input/combatcontroller.go

func (c *CombatController) executeSquadCombat(attackerSquadID, defenderSquadID ecs.EntityID) {
	// Increment turn count
	attackerSquad := systems.GetSquadEntity(attackerSquadID, c.ecsmanager)
	attackerData := common.GetComponentType[*squad.SquadData](attackerSquad, squad.SquadComponent)
	attackerData.TurnCount++

	// Trigger abilities
	systems.CheckAndTriggerAbilities(attackerSquadID, c.ecsmanager)

	// Execute attack - ✅ Result uses native entity IDs
	result := systems.ExecuteSquadAttack(attackerSquadID, defenderSquadID, c.ecsmanager)

	// Display results
	c.showCombatResults(result)

	// Counter-attack if defender still alive
	if !systems.IsSquadDestroyed(defenderSquadID, c.ecsmanager) {
		systems.CheckAndTriggerAbilities(defenderSquadID, c.ecsmanager)
		counterResult := systems.ExecuteSquadAttack(defenderSquadID, attackerSquadID, c.ecsmanager)
		c.showCombatResults(counterResult)
	}

	// Cleanup destroyed squads
	c.checkSquadDestruction(attackerSquadID)
	c.checkSquadDestruction(defenderSquadID)
}

func (c *CombatController) checkSquadDestruction(squadID ecs.EntityID) {
	if systems.IsSquadDestroyed(squadID, c.ecsmanager) {
		// ✅ Get unit IDs using native method
		unitIDs := systems.GetUnitIDsInSquad(squadID, c.ecsmanager)
		for _, unitID := range unitIDs {
			unit := systems.FindUnitByID(unitID, c.ecsmanager)
			if unit != nil {
				unit.Remove()  // ✅ Simple cleanup, no registry
			}
		}

		// Remove squad entity
		squadEntity := systems.GetSquadEntity(squadID, c.ecsmanager)
		if squadEntity != nil {
			squadEntity.Remove()  // ✅ Simple cleanup
		}
	}
}
```

**Testing:**
- Click to select and target squads
- Verify combat resolves correctly
- Spawned enemy squads appear on map
- Squad grid renders when selected
- Abilities trigger during combat
- Dead squads are removed from map

---

## Integration Guide

### Game Initialization

**File:** `game_main/main.go`

```go
func main() {
	manager := ecs.NewManager()
	ecsmanager := &common.EntityManager{
		World:     manager,
		WorldTags: make(map[string]ecs.Tag),
	}

	// Register common components
	common.PositionComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()

	// Register squad components
	squad.InitSquadComponents(manager)
	squad.InitSquadTags()

	// ... rest of initialization
}
```

### Spawning Enemy Squads

**File:** `spawning/spawnmonsters.go`

```go
// SpawnEnemySquad creates an enemy squad based on level
func SpawnEnemySquad(ecsmanager *common.EntityManager, level int, worldPos coords.LogicalPosition) ecs.EntityID {
	var templates []systems.UnitTemplate

	if level <= 3 {
		// Early game: 3-5 weak units
		templates = []systems.UnitTemplate{
			{
				EntityType:    entitytemplates.EntityCreature,
				EntityConfig:  entitytemplates.EntityConfig{Name: "Goblin"},
				EntityData:    loadMonsterData("Goblin"),
				GridRow:       0, GridCol: 0,
				Role:          squad.RoleTank,
				TargetMode:    squad.TargetModeRowBased,
				TargetRows:    []int{0},
				IsMultiTarget: false,
				MaxTargets:    1,
			},
			// ... more units
		}
	} else if level <= 7 {
		// Mid game: 5-7 units with leader
		templates = []systems.UnitTemplate{
			{
				EntityType:   entitytemplates.EntityCreature,
				EntityConfig: entitytemplates.EntityConfig{Name: "Orc Warrior"},
				EntityData:   loadMonsterData("Orc"),
				GridRow:      0, GridCol: 1,
				Role:         squad.RoleTank,
				TargetMode:   squad.TargetModeRowBased,
				TargetRows:   []int{0},
				IsLeader:     true, // Add leader
			},
			// ... more units
		}
	}

	// Create squad (returns native entity ID!)
	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Enemy Squad",
		squad.FormationBalanced,
		worldPos,
		templates,
	)

	// Equip leader abilities if present
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID != 0 {
		equipEnemyLeaderAbilities(leaderID, ecsmanager)
	}

	return squadID
}

func equipEnemyLeaderAbilities(leaderID ecs.EntityID, ecsmanager *common.EntityManager) {
	// Equip 2 random abilities
	systems.EquipAbilityToLeader(
		leaderID,
		0, // Slot 0
		squad.ABILITY_RALLY,
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5, // 50% HP
		ecsmanager,
	)

	systems.EquipAbilityToLeader(
		leaderID,
		1, // Slot 1
		squad.ABILITY_BATTLE_CRY,
		squad.TRIGGER_COMBAT_START,
		0, // No threshold
		ecsmanager,
	)
}
```

---

## Complete Code Examples

### Example 1: Creating a Full Player Squad

```go
func InitializePlayerSquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		// Row 0: Front line tanks
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Knight Captain",
				ImagePath: "knight.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Knight Captain",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         50,
					AttackBonus:       5,
					BaseArmorClass:    15,
					BaseProtection:    5,
					BaseDodgeChance:   10.0,
					BaseMovementSpeed: 5,
				},
			},
			GridRow:       0,
			GridCol:       1,
			Role:          squad.RoleTank,
			TargetMode:    squad.TargetModeRowBased,
			TargetRows:    []int{0}, // Attack front row
			IsMultiTarget: false,
			MaxTargets:    1,
			IsLeader:      true,
			CoverValue:    0.30,        // 30% damage reduction
			CoverRange:    2,           // Covers mid and back rows
			RequiresActive: true,       // Dead knights provide no cover
		},
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Shield Bearer",
				ImagePath: "shield.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createTankData(40, 4, 16, 6),
			GridRow:      0,
			GridCol:      0,
			Role:         squad.RoleTank,
			TargetRows:   []int{0},
			IsMultiTarget: false,
			MaxTargets:   1,
			CoverValue:    0.25,        // 25% damage reduction
			CoverRange:    2,           // Covers mid and back rows
			RequiresActive: true,       // Dead shields provide no cover
		},
		// Row 1: Mid-line DPS and support
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Battle Cleric",
				ImagePath: "cleric.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createSupportData(35, 3, 12, 2),
			GridRow:      1,
			GridCol:      1,
			Role:         squad.RoleSupport,
			TargetRows:   []int{1},
			IsMultiTarget: false,
			MaxTargets:   1,
			CoverValue:    0.10,        // 10% damage reduction
			CoverRange:    1,           // Only covers back row
			RequiresActive: true,       // Dead clerics provide no cover
		},
		// Row 2: Back-line ranged DPS
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Longbowman",
				ImagePath: "archer.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(25, 10, 11, 2),
			GridRow:      2,
			GridCol:      0,
			Role:         squad.RoleDPS,
			TargetRows:   []int{2}, // Snipe back row
			IsMultiTarget: false,
			MaxTargets:   1,
			// No cover (CoverValue: 0, default) - archers don't provide cover
		},
		{
			EntityType:   entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Fire Mage",
				ImagePath: "mage.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(20, 12, 10, 1),
			GridRow:      2,
			GridCol:      2,
			Role:         squad.RoleDPS,
			TargetRows:   []int{2}, // AOE back row
			IsMultiTarget: true,    // Hits all units in row
			MaxTargets:   0,        // Unlimited
			// No cover (CoverValue: 0, default) - mages don't provide cover
		},
	}

	// Create squad (returns native entity ID!)
	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Player's Legion",
		squad.FormationBalanced,
		coords.LogicalPosition{X: 10, Y: 10},
		templates,
	)

	// Setup leader abilities
	setupPlayerLeaderAbilities(squadID, ecsmanager)

	return squadID
}

func setupPlayerLeaderAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return
	}

	// Battle Cry: Triggers at combat start (turn 1)
	systems.EquipAbilityToLeader(
		leaderID,
		0,
		squad.ABILITY_BATTLE_CRY,
		squad.TRIGGER_COMBAT_START,
		0,
		ecsmanager,
	)

	// Heal: Triggers when squad HP drops below 50%
	systems.EquipAbilityToLeader(
		leaderID,
		1,
		squad.ABILITY_HEAL,
		squad.TRIGGER_SQUAD_HP_BELOW,
		0.5,
		ecsmanager,
	)

	// Rally: Triggers on turn 3
	systems.EquipAbilityToLeader(
		leaderID,
		2,
		squad.ABILITY_RALLY,
		squad.TRIGGER_TURN_COUNT,
		3.0,
		ecsmanager,
	)

	// Fireball: Triggers when 2+ enemy squads present
	systems.EquipAbilityToLeader(
		leaderID,
		3,
		squad.ABILITY_FIREBALL,
		squad.TRIGGER_ENEMY_COUNT,
		2.0,
		ecsmanager,
	)
}

// Helper functions to create JSON data
func createTankData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "Tank",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   15.0,
			BaseMovementSpeed: 4,
		},
	}
}

func createDPSData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "DPS",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   20.0,
			BaseMovementSpeed: 6,
		},
	}
}

func createSupportData(hp, atk, ac, prot int) entitytemplates.JSONMonster {
	return entitytemplates.JSONMonster{
		Name: "Support",
		Attributes: entitytemplates.JSONAttributes{
			MaxHealth:         hp,
			AttackBonus:       atk,
			BaseArmorClass:    ac,
			BaseProtection:    prot,
			BaseDodgeChance:   25.0,
			BaseMovementSpeed: 5,
		},
	}
}
```

### Example 2: Combat Simulation

```go
func TestCombatScenario() {
	// Initialize ECS
	manager := ecs.NewManager()
	ecsmanager := &common.EntityManager{
		World:     manager,
		WorldTags: make(map[string]ecs.Tag),
	}

	// Register components
	common.PositionComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()
	squad.InitSquadComponents(manager)
	squad.InitSquadTags()

	// Create two squads
	playerSquadID := InitializePlayerSquad(ecsmanager)
	enemySquadID := SpawnEnemySquad(ecsmanager, 5, coords.LogicalPosition{X: 15, Y: 15})

	// Simulate 5 turns of combat
	for turn := 1; turn <= 5; turn++ {
		fmt.Printf("\n--- TURN %d ---\n", turn)

		// Player squad attacks
		fmt.Println("Player squad attacks:")
		result := systems.ExecuteSquadAttack(playerSquadID, enemySquadID, ecsmanager)
		displayCombatResult(result, ecsmanager)

		// Check if enemy destroyed
		if systems.IsSquadDestroyed(enemySquadID, ecsmanager) {
			fmt.Println("Enemy squad destroyed!")
			break
		}

		// Enemy counter-attacks
		fmt.Println("Enemy squad counter-attacks:")
		counterResult := systems.ExecuteSquadAttack(enemySquadID, playerSquadID, ecsmanager)
		displayCombatResult(counterResult, ecsmanager)

		// Check if player destroyed
		if systems.IsSquadDestroyed(playerSquadID, ecsmanager) {
			fmt.Println("Player squad destroyed!")
			break
		}

		// Display squad status
		displaySquadStatus(playerSquadID, ecsmanager)
		displaySquadStatus(enemySquadID, ecsmanager)
	}
}

func displayCombatResult(result *systems.CombatResult, ecsmanager *common.EntityManager) {
	fmt.Printf("  Total damage: %d\n", result.TotalDamage)
	fmt.Printf("  Units killed: %d\n", len(result.UnitsKilled))

	// ✅ Result uses native entity IDs
	for unitID, dmg := range result.DamageByUnit {
		unit := systems.FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}
		name := common.GetComponentType[*common.Name](unit, common.NameComponent)
		fmt.Printf("    %s took %d damage\n", name.NameStr, dmg)
	}
}

func displaySquadStatus(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	squadEntity := systems.GetSquadEntity(squadID, ecsmanager)
	squadData := common.GetComponentType[*squad.SquadData](squadEntity, squad.SquadComponent)

	fmt.Printf("\n%s Status:\n", squadData.Name)

	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)
	alive := 0

	for _, unitID := range unitIDs {
		unit := systems.FindUnitByID(unitID, ecsmanager)
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
```

### Example 3: Dynamic Squad Modification

```go
// Add a new unit to an existing squad mid-game
func RecruitUnitToSquad(squadID ecs.EntityID, unitName string, gridRow, gridCol int, ecsmanager *common.EntityManager) error {
	// Create new unit entity
	unitEntity := entitytemplates.CreateEntityFromTemplate(
		*ecsmanager,
		entitytemplates.EntityConfig{
			Name:      unitName,
			ImagePath: "recruit.png",
			AssetDir:  "../assets/",
			Visible:   true,
		},
		createDPSData(30, 7, 12, 3),
	)

	// ✅ Get native entity ID
	unitEntityID := unitEntity.GetID()

	// Add to squad
	err := systems.AddUnitToSquad(
		squadID,
		unitEntityID,
		gridRow,
		gridCol,
		squad.RoleDPS,
		[]int{1}, // Target middle row
		false,    // Single-target
		1,        // Max 1 target
		ecsmanager,
	)

	if err != nil {
		return fmt.Errorf("failed to recruit unit: %v", err)
	}

	fmt.Printf("Recruited %s to squad at position (%d, %d)\n", unitName, gridRow, gridCol)
	return nil
}

// Swap unit positions within squad
func ReorganizeSquad(squadID ecs.EntityID, ecsmanager *common.EntityManager) error {
	// Find a back-line DPS unit
	unitIDs := systems.GetUnitIDsInSquad(squadID, ecsmanager)

	var dpsUnitID ecs.EntityID
	for _, unitID := range unitIDs {
		unit := systems.FindUnitByID(unitID, ecsmanager)
		if unit == nil {
			continue
		}

		roleData := common.GetComponentType[*squad.UnitRoleData](unit, squad.UnitRoleComponent)
		gridPos := common.GetComponentType[*squad.GridPositionData](unit, squad.GridPositionComponent)

		if roleData.Role == squad.RoleDPS && gridPos.AnchorRow == 2 {
			dpsUnitID = unitID
			break
		}
	}

	if dpsUnitID == 0 {
		return fmt.Errorf("no DPS unit found to move")
	}

	// Move to front line (tactical decision)
	err := systems.MoveUnitInSquad(dpsUnitID, 0, 2, ecsmanager) // Front right
	if err != nil {
		return fmt.Errorf("failed to move unit: %v", err)
	}

	fmt.Println("Moved DPS unit to front line for aggressive tactics")
	return nil
}
```

### Example 4: Multi-Cell Unit Squad

```go
// Create a squad with multi-cell units (large creatures)
func CreateGiantSquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		// 2x2 Giant occupying front-left (rows 0-1, cols 0-1)
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Stone Giant",
				ImagePath: "giant.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Stone Giant",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         100,  // Large HP pool
					AttackBonus:       15,   // High damage
					BaseArmorClass:    12,
					BaseProtection:    8,
					BaseDodgeChance:   5.0,  // Slow, hard to dodge
					BaseMovementSpeed: 3,
				},
			},
			GridRow:    0,    // Anchor at top-left
			GridCol:    0,
			GridWidth:  2,    // ✅ 2 cells wide
			GridHeight: 2,    // ✅ 2 cells tall
			Role:       squad.RoleTank,
			TargetRows: []int{0, 1}, // Can hit front and mid rows (tall reach)
			IsMultiTarget: false,
			MaxTargets:    1,
			IsLeader:      true,
			CoverValue:    0.40,        // ✅ 40% cover - large unit provides excellent protection
			CoverRange:    1,           // Only covers back row (row 2) since giant occupies rows 0-1
			RequiresActive: true,       // Dead giants provide no cover
		},

		// 1x1 Archer in front-right
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Archer",
				ImagePath: "archer.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: createDPSData(30, 8, 11, 2),
			GridRow:    0,
			GridCol:    2,
			GridWidth:  1,  // ✅ Standard 1x1
			GridHeight: 1,
			Role:       squad.RoleDPS,
			TargetRows: []int{2}, // Snipe back row
			IsMultiTarget: false,
			MaxTargets:    1,
		},

		// 2x1 Cavalry unit (wide) in middle row
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Heavy Cavalry",
				ImagePath: "cavalry.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Heavy Cavalry",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         60,
					AttackBonus:       12,
					BaseArmorClass:    14,
					BaseProtection:    6,
					BaseDodgeChance:   10.0,
					BaseMovementSpeed: 7,
				},
			},
			GridRow:    1,
			GridCol:    2,    // Can't go in cols 0-1 (giant occupies them)
			GridWidth:  1,    // ✅ Only 1 cell available in row 1
			GridHeight: 1,
			Role:       squad.RoleDPS,
			TargetRows: []int{0}, // Charge front row
			IsMultiTarget: false,
			MaxTargets:    1,
		},

		// 3x1 Trebuchet in back row (full-width siege weapon)
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name:      "Trebuchet",
				ImagePath: "trebuchet.png",
				AssetDir:  "../assets/",
				Visible:   true,
			},
			EntityData: entitytemplates.JSONMonster{
				Name: "Trebuchet",
				Attributes: entitytemplates.JSONAttributes{
					MaxHealth:         40,
					AttackBonus:       20,  // Massive AOE damage
					BaseArmorClass:    10,
					BaseProtection:    2,
					BaseDodgeChance:   0.0,  // Can't dodge
					BaseMovementSpeed: 1,
				},
			},
			GridRow:    2,
			GridCol:    0,
			GridWidth:  3,    // ✅ Spans entire back row (cols 0-2)
			GridHeight: 1,
			Role:       squad.RoleDPS,
			TargetRows: []int{0, 1, 2}, // Hits all rows
			IsMultiTarget: true,  // AOE
			MaxTargets:    2,     // Max 2 targets per row
		},
	}

	/*
	Visual Grid Layout:
	Row 0: [Giant___|Giant___] [Archer]
	Row 1: [Giant___|Giant___] [Cavalry]
	Row 2: [Trebuchet___|Trebuchet___|Trebuchet___]
	*/

	squadID := systems.CreateSquadFromTemplate(
		ecsmanager,
		"Siege Battalion",
		squad.FORMATION_OFFENSIVE,
		coords.LogicalPosition{X: 20, Y: 20},
		templates,
	)

	// Equip leader abilities (Giant)
	leaderID := systems.GetLeaderID(squadID, ecsmanager)
	if leaderID != 0 {
		// Rally ability: Buff squad damage when HP drops
		systems.EquipAbilityToLeader(
			leaderID,
			0,
			squad.ABILITY_RALLY,
			squad.TRIGGER_SQUAD_HP_BELOW,
			0.6, // 60% HP
			ecsmanager,
		)
	}

	return squadID
}

// Test multi-cell unit targeting
func TestMultiCellTargeting(ecsmanager *common.EntityManager) {
	// Create giant squad
	giantSquadID := CreateGiantSquad(ecsmanager)

	// Create standard enemy squad
	enemySquadID := createStandardEnemySquad(ecsmanager)

	// Query front row of giant squad
	frontRowUnits := systems.GetUnitIDsInRow(giantSquadID, 0, ecsmanager)
	fmt.Printf("Front row has %d units\n", len(frontRowUnits))
	// Output: Front row has 2 units (Giant + Archer)

	// Query middle row of giant squad
	midRowUnits := systems.GetUnitIDsInRow(giantSquadID, 1, ecsmanager)
	fmt.Printf("Middle row has %d units\n", len(midRowUnits))
	// Output: Middle row has 2 units (Giant + Cavalry)
	// Note: Giant appears in BOTH front and middle row queries!

	// Query back row of giant squad
	backRowUnits := systems.GetUnitIDsInRow(giantSquadID, 2, ecsmanager)
	fmt.Printf("Back row has %d units\n", len(backRowUnits))
	// Output: Back row has 1 unit (Trebuchet)

	// Enemy attacks front row - can hit Giant or Archer
	fmt.Println("\nEnemy attacks front row:")
	result := systems.ExecuteSquadAttack(enemySquadID, giantSquadID, ecsmanager)
	// Giant will likely be targeted (larger HP pool = lower HP target)

	// Enemy attacks middle row - can hit Giant or Cavalry
	fmt.Println("\nEnemy attacks middle row:")
	// Giant is vulnerable from BOTH front and middle row attacks!
}

func createStandardEnemySquad(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Goblin", ImagePath: "goblin.png", AssetDir: "../assets/", Visible: true},
			EntityData: createDPSData(20, 5, 10, 1),
			GridRow: 0, GridCol: 0,
			GridWidth: 1, GridHeight: 1,
			Role: squad.RoleDPS,
			TargetRows: []int{0},
		},
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{Name: "Orc", ImagePath: "orc.png", AssetDir: "../assets/", Visible: true},
			EntityData: createTankData(40, 6, 13, 4),
			GridRow: 0, GridCol: 1,
			GridWidth: 1, GridHeight: 1,
			Role: squad.RoleTank,
			TargetRows: []int{0},
		},
	}

	return systems.CreateSquadFromTemplate(
		ecsmanager,
		"Goblin Warband",
		squad.FormationBalanced,
		coords.LogicalPosition{X: 25, Y: 20},
		templates,
	)
}
```

**Key Takeaways from Multi-Cell Example:**

1. **Large units occupy multiple cells** - The 2x2 Giant takes up 4 cells
2. **Row targeting affects multi-cell units** - Giant can be hit by attacks targeting row 0 OR row 1
3. **Deduplication works** - Giant appears only once in each row query despite occupying 2 rows
4. **Grid constraints enforced** - Cavalry can't fit in cols 0-1 of row 1 (Giant blocks it)
5. **Full-width units** - Trebuchet spans entire back row (3 cells wide)
6. **Strategic trade-offs** - Large units are easier to hit but have more HP
7. **Multi-column cover** - Giant provides 40% cover to both columns 0 and 1 in row 2 (Trebuchet benefits)

---

### Example 5: Cover System Demonstration

```go
// Demonstrate tactical cover mechanics with stacking bonuses
func TestCoverMechanics(ecsmanager *common.EntityManager) {
	// Create a defensive squad with overlapping cover
	defensiveSquad := createDefensiveSquadWithCover(ecsmanager)

	// Create attacking enemy squad
	enemySquad := createStandardEnemySquad(ecsmanager)

	// Enemy attacks back row - cover should reduce damage
	fmt.Println("=== Testing Cover Mechanics ===\n")

	// Get back row archer (receives cover from front-line tanks)
	backRowUnits := systems.GetUnitIDsInRow(defensiveSquad, 2, ecsmanager)
	archerID := backRowUnits[0]

	// Calculate cover for archer
	coverReduction := systems.CalculateTotalCover(archerID, ecsmanager)
	fmt.Printf("Archer cover reduction: %.0f%%\n", coverReduction * 100)
	// Output: Archer cover reduction: 55% (30% from Knight + 25% from Shield Bearer)

	// Execute attack - damage should be reduced by cover
	result := systems.ExecuteSquadAttack(enemySquad, defensiveSquad, ecsmanager)
	archerDamage := result.DamageByUnit[archerID]
	fmt.Printf("Damage to archer (with cover): %d\n", archerDamage)

	// Kill front-line tanks to remove cover
	knightID := systems.GetUnitIDsAtGridPosition(defensiveSquad, 0, 1, ecsmanager)[0]
	knightEntity := systems.FindUnitByID(knightID, ecsmanager)
	knightAttr := common.GetAttributes(knightEntity)
	knightAttr.CurrentHealth = 0 // Kill knight

	// Cover should now be reduced
	coverReductionAfterDeath := systems.CalculateTotalCover(archerID, ecsmanager)
	fmt.Printf("\nArcher cover after knight death: %.0f%%\n", coverReductionAfterDeath * 100)
	// Output: Archer cover after knight death: 25% (only Shield Bearer alive)

	// Attack again - more damage without full cover
	result2 := systems.ExecuteSquadAttack(enemySquad, defensiveSquad, ecsmanager)
	archerDamage2 := result2.DamageByUnit[archerID]
	fmt.Printf("Damage to archer (reduced cover): %d\n", archerDamage2)
	fmt.Printf("Damage increase: +%d (%.1f%%)\n",
		archerDamage2 - archerDamage,
		float64(archerDamage2 - archerDamage) / float64(archerDamage) * 100)
}

func createDefensiveSquadWithCover(ecsmanager *common.EntityManager) ecs.EntityID {
	templates := []systems.UnitTemplate{
		// Row 0: Heavy front-line tanks providing overlapping cover
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name: "Knight Captain", ImagePath: "knight.png",
				AssetDir: "../assets/", Visible: true,
			},
			EntityData: createTankData(50, 5, 15, 5),
			GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1,
			Role: squad.RoleTank,
			TargetRows: []int{0},
			IsMultiTarget: false,
			MaxTargets: 1,
			IsLeader: true,
			CoverValue: 0.30,       // 30% cover
			CoverRange: 2,          // Covers rows 1 and 2
			RequiresActive: true,
		},
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name: "Shield Bearer", ImagePath: "shield.png",
				AssetDir: "../assets/", Visible: true,
			},
			EntityData: createTankData(40, 4, 16, 6),
			GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1,
			Role: squad.RoleTank,
			TargetRows: []int{0},
			IsMultiTarget: false,
			MaxTargets: 1,
			CoverValue: 0.25,       // 25% cover
			CoverRange: 2,          // Covers rows 1 and 2
			RequiresActive: true,
		},

		// Row 1: Mid-line support with minor cover
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name: "Battle Cleric", ImagePath: "cleric.png",
				AssetDir: "../assets/", Visible: true,
			},
			EntityData: createSupportData(35, 3, 12, 2),
			GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1,
			Role: squad.RoleSupport,
			TargetRows: []int{1},
			IsMultiTarget: false,
			MaxTargets: 1,
			CoverValue: 0.10,       // 10% cover
			CoverRange: 1,          // Only covers row 2
			RequiresActive: true,
		},

		// Row 2: Back-line archers (receive stacking cover)
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name: "Longbowman", ImagePath: "archer.png",
				AssetDir: "../assets/", Visible: true,
			},
			EntityData: createDPSData(25, 10, 11, 2),
			GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1,
			Role: squad.RoleDPS,
			TargetRows: []int{2},
			IsMultiTarget: false,
			MaxTargets: 1,
			// No cover provided, but receives cover from Shield Bearer (col 0)
		},
		{
			EntityType: entitytemplates.EntityCreature,
			EntityConfig: entitytemplates.EntityConfig{
				Name: "Crossbowman", ImagePath: "archer.png",
				AssetDir: "../assets/", Visible: true,
			},
			EntityData: createDPSData(25, 10, 11, 2),
			GridRow: 2, GridCol: 1, GridWidth: 1, GridHeight: 1,
			Role: squad.RoleDPS,
			TargetRows: []int{2},
			IsMultiTarget: false,
			MaxTargets: 1,
			// No cover provided, but receives STACKING cover:
			// - 30% from Knight Captain (col 1)
			// - 25% from Shield Bearer (different column, no overlap)
			// - 10% from Battle Cleric (col 1)
			// Total: 40% cover (stacking from Knight + Cleric)
		},
	}

	/*
	Visual Grid Layout (with cover relationships):
	Row 0: [Shield(25%)_] [Knight(30%)_] [Empty]
			 |               |
			 v (col 0)       v (col 1)
	Row 1: [Empty]        [Cleric(10%)_] [Empty]
							|
							v (col 1)
	Row 2: [Longbow]      [Crossbow]     [Empty]
		   (25% cover)    (40% cover - stacking!)
	*/

	return systems.CreateSquadFromTemplate(
		ecsmanager,
		"Defensive Phalanx",
		squad.FormationDefensive,
		coords.LogicalPosition{X: 10, Y: 10},
		templates,
	)
}
```

