package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

type FactionManager struct {
	manager *common.EntityManager
}

func NewFactionManager(manager *common.EntityManager) *FactionManager {
	return &FactionManager{
		manager: manager,
	}
}

func (fm *FactionManager) CreateFaction(name string, isPlayer bool) ecs.EntityID {
	faction := fm.manager.World.NewEntity()
	factionID := faction.GetID()

	faction.AddComponent(FactionComponent, &FactionData{
		FactionID:          factionID,
		Name:               name,
		Mana:               100,
		MaxMana:            100,
		IsPlayerControlled: isPlayer,
	})

	return factionID
}

func (fm *FactionManager) AddSquadToFaction(factionID, squadID ecs.EntityID, position coords.LogicalPosition) error {

	faction := findFactionByID(factionID, fm.manager)
	if faction == nil {
		return fmt.Errorf("faction %d not found", factionID)
	}

	squad := common.FindEntityByIDWithTag(fm.manager, squadID, squads.SquadTag)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Create MapPositionData to establish relationship
	mapPosEntity := fm.manager.World.NewEntity()
	mapPosEntity.AddComponent(MapPositionComponent, &MapPositionData{
		SquadID:   squadID,
		Position:  position,
		FactionID: factionID,
	})

	// Add PositionComponent to squad entity for compatibility with existing squad combat system
	if !squad.HasComponent(common.PositionComponent) {
		posPtr := new(coords.LogicalPosition)
		*posPtr = position
		squad.AddComponent(common.PositionComponent, posPtr)
	}

	// Register in PositionSystem
	common.GlobalPositionSystem.AddEntity(squadID, position)

	return nil
}

func (fm *FactionManager) GetFactionSquads(factionID ecs.EntityID) []ecs.EntityID {
	var squadIDs []ecs.EntityID

	// Query all MapPositionData entities
	for _, result := range fm.manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
		if mapPos.FactionID == factionID {
			squadIDs = append(squadIDs, mapPos.SquadID)
		}
	}

	return squadIDs
}

func (fm *FactionManager) RemoveSquadFromFaction(factionID, squadID ecs.EntityID) error {
	// Find MapPositionData entity for this squad
	mapPosEntity := findMapPositionEntity(squadID, fm.manager)
	if mapPosEntity == nil {
		return fmt.Errorf("squad %d not found on map", squadID)
	}

	// Verify squad belongs to this faction
	mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
	if mapPos.FactionID != factionID {
		return fmt.Errorf("squad %d does not belong to faction %d", squadID, factionID)
	}

	// Get position before removal for PositionSystem cleanup
	position := mapPos.Position

	// Remove MapPositionData entity from ECS
	fm.manager.World.DisposeEntities(mapPosEntity)

	// Remove from PositionSystem spatial grid
	common.GlobalPositionSystem.RemoveEntity(squadID, position)

	return nil
}

func (fm *FactionManager) GetFactionMana(factionID ecs.EntityID) (current, max int) {
	// Find faction entity
	faction := findFactionByID(factionID, fm.manager)
	if faction == nil {
		return 0, 0 // Faction not found
	}

	// Get faction data
	factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
	return factionData.Mana, factionData.MaxMana
}
