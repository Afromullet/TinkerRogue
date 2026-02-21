package squads

import (
	"fmt"
	"game_main/common"
	"game_main/templates"
	"game_main/visual/rendering"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ========================================
// LEADER COMPONENT HELPERS
// ========================================

// AddLeaderComponents adds all leader-related components to a unit entity.
// Includes LeaderComponent, AbilitySlotComponent, and CooldownTrackerComponent.
func AddLeaderComponents(entity *ecs.Entity) {
	entity.AddComponent(LeaderComponent, &LeaderData{
		Leadership: 10,
		Experience: 0,
	})

	entity.AddComponent(AbilitySlotComponent, &AbilitySlotData{
		Slots: [4]AbilitySlot{},
	})

	entity.AddComponent(CooldownTrackerComponent, &CooldownTrackerData{
		Cooldowns:    [4]int{0, 0, 0, 0},
		MaxCooldowns: [4]int{0, 0, 0, 0},
	})
}

// RemoveLeaderComponents removes all leader-related components from a unit entity.
func RemoveLeaderComponents(entity *ecs.Entity) {
	if entity.HasComponent(LeaderComponent) {
		entity.RemoveComponent(LeaderComponent)
	}
	if entity.HasComponent(AbilitySlotComponent) {
		entity.RemoveComponent(AbilitySlotComponent)
	}
	if entity.HasComponent(CooldownTrackerComponent) {
		entity.RemoveComponent(CooldownTrackerComponent)
	}
}

// ========================================
// SQUAD RELATED
// ========================================

// CreateEmptySquad creates a new empty squad and returns its ID
func CreateEmptySquad(squadmanager *common.EntityManager,
	squadName string) ecs.EntityID {

	squadEntity := squadmanager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:    squadID,
		Name:       squadName,
		Morale:     100,
		TurnCount:  0,
		MaxUnits:   9,
		IsDeployed: false, // New squads start in reserves (not on map)
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

	return squadID
}

// gridRow and gridCol are the row and col we want to anchor the unit at
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *common.EntityManager,
	unit UnitTemplate,
	gridRow, gridCol int) (ecs.EntityID, error) {

	// Validate position using the provided parameters, not unit template values
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		return 0, fmt.Errorf("invalid grid position (%d, %d)", gridRow, gridCol)
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, squadmanager)
	if len(existingUnitIDs) > 0 {
		return 0, fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Check capacity before adding unit
	unitCapacityCost := unit.Attributes.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, squadmanager) {
		remaining := GetSquadRemainingCapacity(squadID, squadmanager)
		return 0, fmt.Errorf("insufficient squad capacity: need %.2f, have %.2f remaining (unit %s costs %.2f)",
			unitCapacityCost, remaining, unit.UnitType, unitCapacityCost)
	}

	// Create unit entity (adds GridPositionComponent with default 0,0)
	unitEntity, err := CreateUnitEntity(squadmanager, unit)
	if err != nil {
		return 0, fmt.Errorf("invalid unit for %s: %w", unit.UnitType, err)
	}

	// Add SquadMemberComponent to link unit to squad
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	// Update GridPositionComponent with actual grid position
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPos.AnchorRow = gridRow
	gridPos.AnchorCol = gridCol

	return unitEntity.GetID(), nil
}

func RemoveUnitFromSquad(unitEntityID ecs.EntityID, squadmanager *common.EntityManager) error {
	if !squadmanager.HasComponent(unitEntityID, SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	// Find the unit entity and dispose it
	unitEntity := squadmanager.FindEntityByID(unitEntityID)
	if unitEntity != nil {
		// Get position component if it exists (units typically don't have world positions)
		pos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
		// Use CleanDisposeEntity for consistent cleanup
		squadmanager.CleanDisposeEntity(unitEntity, pos)
	}

	return nil
}

func MoveUnitInSquad(unitEntityID ecs.EntityID, newRow, newCol int, ecsmanager *common.EntityManager) error {
	if !ecsmanager.HasComponent(unitEntityID, SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	gridPosData := common.GetComponentTypeByID[*GridPositionData](ecsmanager, unitEntityID, GridPositionComponent)
	if gridPosData == nil {
		return fmt.Errorf("unit entity not found")
	}

	// Validate new anchor position is in bounds
	if newRow < 0 || newCol < 0 {
		return fmt.Errorf("invalid anchor position (%d, %d)", newRow, newCol)
	}

	// Validate unit fits within grid at new position
	if newRow+gridPosData.Height > 3 || newCol+gridPosData.Width > 3 {
		return fmt.Errorf("unit would extend outside grid at position (%d, %d) with size %dx%d",
			newRow, newCol, gridPosData.Width, gridPosData.Height)
	}

	memberData := common.GetComponentTypeByID[*SquadMemberData](ecsmanager, unitEntityID, SquadMemberComponent)

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

// CreateSquadFromTemplate
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []UnitTemplate,
) ecs.EntityID {

	// Create squad entity
	squadEntity := ecsmanager.World.NewEntity()

	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:    squadID,
		Name:       squadName,
		Formation:  formation,
		Morale:     100,
		TurnCount:  0,
		MaxUnits:   9,
		IsDeployed: false, // New squads start in reserves (not on map)
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
		unitEntity := templates.CreateEntityFromTemplate(
			*ecsmanager,
			template.EntityConfig,
			template.EntityData,
		)

		// Make unit's renderable invisible - only the squad should render on the world map
		// Units are internal to squads; the squad entity shows the leader's sprite
		unitRenderable := common.GetComponentType[*rendering.Renderable](unitEntity, rendering.RenderableComponent)
		if unitRenderable != nil {
			unitRenderable.Visible = false
		}

		// Update unit's world position to match squad position
		// (CreateEntityFromTemplate sets position to 0,0 by default)
		// Re-add PositionComponent with correct world position
		unitEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
			X: worldPos.X,
			Y: worldPos.Y,
		})

		// Add squad membership (uses ID, not entity pointer)
		unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
			SquadID: squadID,
		})

		// Add grid position (supports multi-cell)
		unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
			AnchorRow: template.GridRow,
			AnchorCol: template.GridCol,
			Width:     width,
			Height:    height,
		})

		// Add role
		unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
			Role: template.Role,
		})

		// Add targeting data
		unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
			AttackType:  template.AttackType,
			TargetCells: template.TargetCells,
		})

		// Add cover component if unit provides cover
		if template.CoverValue > 0.0 {
			unitEntity.AddComponent(CoverComponent, &CoverData{
				CoverValue:     template.CoverValue,
				CoverRange:     template.CoverRange,
				RequiresActive: template.RequiresActive,
			})
		}

		// Add attack range component
		unitEntity.AddComponent(AttackRangeComponent, &AttackRangeData{
			Range: template.AttackRange,
		})

		// Add movement speed component
		unitEntity.AddComponent(MovementSpeedComponent, &MovementSpeedData{
			Speed: template.MovementSpeed,
		})

		// Add unit type component for roster grouping (preserves original type)
		unitEntity.AddComponent(UnitTypeComponent, &UnitTypeData{
			UnitType: template.UnitType,
		})

		// Add experience component (all units start at level 1 with 0 XP)
		unitEntity.AddComponent(ExperienceComponent, &ExperienceData{
			Level:         1,
			CurrentXP:     0,
			XPToNextLevel: 100,
		})

		// Add stat growth component
		unitEntity.AddComponent(StatGrowthComponent, &StatGrowthData{
			Strength:   template.StatGrowths.Strength,
			Dexterity:  template.StatGrowths.Dexterity,
			Magic:      template.StatGrowths.Magic,
			Leadership: template.StatGrowths.Leadership,
			Armor:      template.StatGrowths.Armor,
			Weapon:     template.StatGrowths.Weapon,
		})

		// Add leader component if needed
		if template.IsLeader {
			AddLeaderComponents(unitEntity)
		}

		// Mark ALL cells as occupied
		for _, cell := range cellsToOccupy {
			key := fmt.Sprintf("%d,%d", cell[0], cell[1])
			occupied[key] = true
		}
	}

	// Set squad's renderable to the leader's sprite
	// This makes the squad display the leader's image on the world map
	setSquadRenderableFromLeader(squadID, squadEntity, ecsmanager)

	return squadID
}

// setSquadRenderableFromLeader copies the leader unit's sprite to the squad entity.
// This makes the squad render on the world map using the leader's image.
// If no leader is found, the squad will have no renderable (won't display on map).
func setSquadRenderableFromLeader(squadID ecs.EntityID, squadEntity *ecs.Entity, ecsmanager *common.EntityManager) {
	// Find the leader unit
	leaderID := GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return
	}

	// Get the leader entity
	leaderEntity := ecsmanager.FindEntityByID(leaderID)
	if leaderEntity == nil {
		return
	}

	// Get the leader's renderable
	leaderRenderable := common.GetComponentType[*rendering.Renderable](leaderEntity, rendering.RenderableComponent)
	if leaderRenderable == nil || leaderRenderable.Image == nil {
		return
	}

	// Add/update the squad's renderable with the leader's image
	squadEntity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image:   leaderRenderable.Image,
		Visible: true,
	})
}

// ========================================
// ENTITY DISPOSAL FUNCTIONS
// ========================================

// DisposeDeadUnitsInSquad disposes all dead units (CurrentHealth <= 0) in a squad.
// Returns the number of units disposed.
// Use this to clean up dead units after combat while the squad survives.
func DisposeDeadUnitsInSquad(squadID ecs.EntityID, manager *common.EntityManager) int {
	unitIDs := GetUnitIDsInSquad(squadID, manager)
	disposedCount := 0

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth <= 0 {
			// Unit is dead, dispose it
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			manager.CleanDisposeEntity(entity, pos)
			disposedCount++
		}
	}

	return disposedCount
}

// DisposeSquadAndUnits disposes a squad entity and ALL its units (dead or alive).
// Use this when a squad is destroyed and needs complete cleanup.
// This removes all entities from the ECS world.
func DisposeSquadAndUnits(squadID ecs.EntityID, manager *common.EntityManager) {
	// Get the squad entity first (use direct ID lookup for reliability)
	squadEntity := manager.FindEntityByID(squadID)

	// Get all units before disposing (query won't work after squad is gone)
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	// Dispose all units first
	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		manager.CleanDisposeEntity(entity, pos)
	}

	// Now dispose the squad entity itself
	if squadEntity != nil {
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		manager.CleanDisposeEntity(squadEntity, pos)
	}
}
