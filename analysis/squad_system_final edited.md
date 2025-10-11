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

