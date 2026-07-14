package squadcore

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// ========================================
// LEADER COMPONENT HELPERS
// ========================================

// AddLeaderComponents adds the LeaderComponent to a unit entity.
func AddLeaderComponents(entity *ecs.Entity) {
	entity.AddComponent(LeaderComponent, &LeaderData{
		Leadership: 10,
		Experience: 0,
	})
}

// RemoveLeaderComponents removes the LeaderComponent from a unit entity.
func RemoveLeaderComponents(entity *ecs.Entity) {
	if entity.HasComponent(LeaderComponent) {
		entity.RemoveComponent(LeaderComponent)
	}
}

// ========================================
// GRID VALIDATION HELPERS
// ========================================

// ValidateGridPlacement validates that a unit of the given size fits within the
// squad grid at the specified anchor position. width/height of 0 are treated as 1.
// Used by all squad creation and placement paths.
func ValidateGridPlacement(row, col, width, height int) error {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	if row < 0 || col < 0 {
		return fmt.Errorf("invalid anchor position (%d, %d)", row, col)
	}
	if width > SquadGridSize {
		return fmt.Errorf("invalid grid width %d: must be 1-%d", width, SquadGridSize)
	}
	if height > SquadGridSize {
		return fmt.Errorf("invalid grid height %d: must be 1-%d", height, SquadGridSize)
	}
	if row+height > SquadGridSize || col+width > SquadGridSize {
		return fmt.Errorf("unit would extend outside grid at position (%d, %d) with size %dx%d",
			row, col, width, height)
	}
	return nil
}

// ValidateGridAnchor validates that an anchor cell (row, col) is within the squad grid.
// Equivalent to ValidateGridPlacement with width=1, height=1.
func ValidateGridAnchor(row, col int) error {
	return ValidateGridPlacement(row, col, 1, 1)
}

// hideUnitRenderable marks a unit's renderable as invisible. Units inside a squad
// never render themselves on the world map; only the squad entity does.
// No-op if the unit has no RenderableComponent.
func hideUnitRenderable(unitEntity *ecs.Entity) {
	r := common.GetComponentType[*common.Renderable](unitEntity, common.RenderableComponent)
	if r != nil {
		r.Visible = false
	}
}

// attachUnitToSquadAtGrid is the shared tail of all unit-placement paths:
// link the unit to its squad, write its grid anchor, hide its renderable, and
// promote it to leader (refreshing the squad's renderable) when the squad
// has none. Callers must validate capacity and occupancy first.
func attachUnitToSquadAtGrid(
	squadID ecs.EntityID,
	unitEntity *ecs.Entity,
	gridRow, gridCol int,
	manager *common.EntityManager,
) {
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	if gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent); gridPos != nil {
		gridPos.AnchorRow = gridRow
		gridPos.AnchorCol = gridCol
	}

	hideUnitRenderable(unitEntity)

	if GetLeaderID(squadID, manager) == 0 {
		AddLeaderComponents(unitEntity)
		if squadEntity := GetSquadEntity(squadID, manager); squadEntity != nil {
			SetSquadRenderableFromLeader(squadID, squadEntity, manager)
		}
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
		MaxUnits:   SquadMaxUnits,
		IsDeployed: false, // New squads start in reserves (not on map)
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

	return squadID
}

// gridRow and gridCol are the row and col we want to anchor the unit at
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *common.EntityManager,
	unit unitdefs.UnitTemplate,
	gridRow, gridCol int) (ecs.EntityID, error) {

	// Validate position using the provided parameters, not unit template values
	if err := ValidateGridAnchor(gridRow, gridCol); err != nil {
		return 0, err
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

	attachUnitToSquadAtGrid(squadID, unitEntity, gridRow, gridCol, squadmanager)

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

// PlaceUnitInSquad places an existing entity into a squad at the specified grid position.
// Unlike AddUnitToSquad, this does NOT create a new entity — it reuses the given entity.
// The entity must already have GridPositionComponent and AttributeComponent.
func PlaceUnitInSquad(squadID ecs.EntityID, unitEntityID ecs.EntityID, manager *common.EntityManager, gridRow, gridCol int) error {
	// Validate grid position
	if err := ValidateGridAnchor(gridRow, gridCol); err != nil {
		return err
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, manager)
	if len(existingUnitIDs) > 0 {
		return fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Check capacity
	attr := common.GetComponentTypeByID[*common.Attributes](manager, unitEntityID, common.AttributeComponent)
	if attr == nil {
		return fmt.Errorf("unit entity has no attributes")
	}
	unitCapacityCost := attr.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, manager) {
		remaining := GetSquadRemainingCapacity(squadID, manager)
		return fmt.Errorf("insufficient squad capacity: need %.2f, have %.2f remaining",
			unitCapacityCost, remaining)
	}

	unitEntity := manager.FindEntityByID(unitEntityID)
	if unitEntity == nil {
		return fmt.Errorf("unit entity %d not found", unitEntityID)
	}

	// Caller contract: entity must already have GridPositionComponent.
	if !unitEntity.HasComponent(GridPositionComponent) {
		return fmt.Errorf("unit entity has no GridPositionComponent")
	}

	attachUnitToSquadAtGrid(squadID, unitEntity, gridRow, gridCol, manager)
	return nil
}

// UnassignUnitFromSquad removes a unit from its squad WITHOUT disposing the entity.
// The entity stays alive in the ECS world (remains in roster pool).
func UnassignUnitFromSquad(unitEntityID ecs.EntityID, manager *common.EntityManager) error {
	if !manager.HasComponent(unitEntityID, SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	unitEntity := manager.FindEntityByID(unitEntityID)
	if unitEntity == nil {
		return fmt.Errorf("unit entity %d not found", unitEntityID)
	}

	// If this unit is the leader, strip leader components to prevent orphaning
	if unitEntity.HasComponent(LeaderComponent) {
		RemoveLeaderComponents(unitEntity)
	}

	// Remove squad membership
	unitEntity.RemoveComponent(SquadMemberComponent)

	// Reset grid position anchor to 0,0
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	if gridPos != nil {
		gridPos.AnchorRow = 0
		gridPos.AnchorCol = 0
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

	// Validate new anchor position and that unit fits within grid
	if err := ValidateGridPlacement(newRow, newCol, gridPosData.CellWidth, gridPosData.CellHeight); err != nil {
		return err
	}

	memberData := common.GetComponentTypeByID[*SquadMemberData](ecsmanager, unitEntityID, SquadMemberComponent)

	// Check if ANY cell at new position is occupied (excluding this unit itself)
	for r := newRow; r < newRow+gridPosData.CellHeight; r++ {
		for c := newCol; c < newCol+gridPosData.CellWidth; c++ {
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

// CreateSquadFromTemplate
func CreateSquadFromTemplate(
	ecsmanager *common.EntityManager,
	squadName string,
	formation FormationType,
	worldPos coords.LogicalPosition,
	unitTemplates []unitdefs.UnitTemplate,
) ecs.EntityID {
	squadEntity := ecsmanager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:    squadID,
		Name:       squadName,
		Formation:  formation,
		Morale:     100,
		TurnCount:  0,
		MaxUnits:   SquadMaxUnits,
		IsDeployed: false,
	})
	squadEntity.AddComponent(common.PositionComponent, &worldPos)

	var occupied [SquadGridSize][SquadGridSize]bool
	for _, template := range unitTemplates {
		placeTemplateUnit(ecsmanager, squadID, worldPos, template, &occupied)
	}

	// Set squad's renderable to the leader's sprite so it appears on the world map.
	SetSquadRenderableFromLeader(squadID, squadEntity, ecsmanager)

	return squadID
}

// placeTemplateUnit validates, creates, and registers a single template-driven
// unit inside a squad-under-construction, mutating `occupied` to mark its cells.
// Logs a warning and returns without mutating state if the unit cannot be placed.
func placeTemplateUnit(
	ecsmanager *common.EntityManager,
	squadID ecs.EntityID,
	worldPos coords.LogicalPosition,
	template unitdefs.UnitTemplate,
	occupied *[SquadGridSize][SquadGridSize]bool,
) {
	width := template.GridWidth
	if width == 0 {
		width = 1
	}
	height := template.GridHeight
	if height == 0 {
		height = 1
	}

	if err := ValidateGridPlacement(template.GridRow, template.GridCol, width, height); err != nil {
		fmt.Printf("Warning: %v, skipping\n", err)
		return
	}

	for r := template.GridRow; r < template.GridRow+height; r++ {
		for c := template.GridCol; c < template.GridCol+width; c++ {
			if occupied[r][c] {
				fmt.Printf("Warning: Cell (%d,%d) already occupied, cannot place %dx%d unit at (%d,%d)\n",
					r, c, width, height, template.GridRow, template.GridCol)
				return
			}
		}
	}

	unitEntity, err := templates.CreateCreatureEntity(ecsmanager, template.EntityConfig, template.EntityData)
	if err != nil {
		fmt.Printf("Warning: %v, skipping\n", err)
		return
	}
	unitEntity.AddComponent(common.NameComponent, &common.Name{
		NameStr: templates.GenerateName("default", template.UnitType),
	})
	hideUnitRenderable(unitEntity)

	// Re-bind world position to the squad's tile (CreateCreatureEntity defaults to 0,0).
	unitEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{X: worldPos.X, Y: worldPos.Y})

	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{SquadID: squadID})
	ApplyUnitComponents(unitEntity, template, template.GridRow, template.GridCol)
	if template.IsLeader {
		AddLeaderComponents(unitEntity)
	}

	for r := template.GridRow; r < template.GridRow+height; r++ {
		for c := template.GridCol; c < template.GridCol+width; c++ {
			occupied[r][c] = true
		}
	}
}

// SetSquadRenderableFromLeader copies the leader unit's sprite to the squad entity.
// This makes the squad render on the world map using the leader's image.
// If no leader is found, the squad will have no renderable (won't display on map).
func SetSquadRenderableFromLeader(squadID ecs.EntityID, squadEntity *ecs.Entity, ecsmanager *common.EntityManager) {
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
	leaderRenderable := common.GetComponentType[*common.Renderable](leaderEntity, common.RenderableComponent)
	if leaderRenderable == nil || leaderRenderable.Image == nil {
		return
	}

	// Add/update the squad's renderable with the leader's image
	squadEntity.AddComponent(common.RenderableComponent, &common.Renderable{
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

// ResetSquadDeployment unregisters the squad and its units from the spatial
// position system and clears the IsDeployed flag. Used when a squad leaves the
// tactical map (combat exit, garrison-defenders returning to their node) but
// the squad entities themselves must survive in ECS for future deployment.
//
// Counterpart to MarkSquadsDeployed in mind/combatlifecycle/enrollment.go.
func ResetSquadDeployment(manager *common.EntityManager, squadEntity *ecs.Entity) {
	if squadEntity == nil {
		return
	}

	manager.UnregisterEntityPosition(squadEntity)

	for _, unitID := range GetUnitIDsInSquad(squadEntity.GetID(), manager) {
		unitEntity := manager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}
		manager.UnregisterEntityPosition(unitEntity)
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData != nil {
		squadData.IsDeployed = false
	}
}
