package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/savesystem"
	"game_main/tactical/commander"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&PlayerChunk{})
}

// PlayerChunk saves/loads the player entity: position, attributes, resources,
// unit roster, and commander roster. Artifact inventory is handled by GearChunk.
type PlayerChunk struct{}

func (c *PlayerChunk) ChunkID() string  { return "player" }
func (c *PlayerChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedPlayer struct {
	EntityID  ecs.EntityID    `json:"entityID"`
	Position  savedPosition   `json:"position"`
	Attrs     savedAttributes `json:"attributes"`
	Resources savedResources  `json:"resources"`
	UnitRos   *savedUnitRoster `json:"unitRoster,omitempty"`
	CmdRos    *savedCmdRoster  `json:"commanderRoster,omitempty"`
}

type savedResources struct {
	Gold  int `json:"gold"`
	Iron  int `json:"iron"`
	Wood  int `json:"wood"`
	Stone int `json:"stone"`
}

type savedUnitRoster struct {
	MaxUnits int                    `json:"maxUnits"`
	Entries  []savedUnitRosterEntry `json:"entries"`
}

type savedUnitRosterEntry struct {
	UnitType      string               `json:"unitType"`
	TotalOwned    int                  `json:"totalOwned"`
	UnitEntities  []ecs.EntityID       `json:"unitEntities"`
	UnitsInSquads map[ecs.EntityID]int `json:"unitsInSquads"`
}

type savedCmdRoster struct {
	CommanderIDs  []ecs.EntityID `json:"commanderIDs"`
	MaxCommanders int            `json:"maxCommanders"`
}

// --- Save ---

func (c *PlayerChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	playerTag, ok := em.WorldTags["players"]
	if !ok {
		// No player tag registered — nothing to save
		return nil, nil
	}

	results := em.World.Query(playerTag)
	if len(results) == 0 {
		// No player entity exists — nothing to save
		return nil, nil
	}

	entity := results[0].Entity
	entityID := entity.GetID()

	sp := savedPlayer{EntityID: entityID}

	// Position
	if pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); pos != nil {
		sp.Position = positionToSaved(pos)
	}

	// Attributes
	if attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent); attr != nil {
		sp.Attrs = attributesToSaved(attr)
	}

	// Resources
	if res := common.GetComponentType[*common.ResourceStockpile](entity, common.ResourceStockpileComponent); res != nil {
		sp.Resources = savedResources{
			Gold: res.Gold, Iron: res.Iron, Wood: res.Wood, Stone: res.Stone,
		}
	}

	// Unit Roster
	if roster := common.GetComponentType[*squads.UnitRoster](entity, squads.UnitRosterComponent); roster != nil {
		sur := &savedUnitRoster{MaxUnits: roster.MaxUnits}
		for _, entry := range roster.Units {
			se := savedUnitRosterEntry{
				UnitType:      entry.UnitType,
				TotalOwned:    entry.TotalOwned,
				UnitEntities:  make([]ecs.EntityID, len(entry.UnitEntities)),
				UnitsInSquads: make(map[ecs.EntityID]int),
			}
			copy(se.UnitEntities, entry.UnitEntities)
			for k, v := range entry.UnitsInSquads {
				se.UnitsInSquads[k] = v
			}
			sur.Entries = append(sur.Entries, se)
		}
		sp.UnitRos = sur
	}

	// Commander Roster
	if cmdRoster := common.GetComponentType[*commander.CommanderRosterData](entity, commander.CommanderRosterComponent); cmdRoster != nil {
		sp.CmdRos = &savedCmdRoster{
			CommanderIDs:  make([]ecs.EntityID, len(cmdRoster.CommanderIDs)),
			MaxCommanders: cmdRoster.MaxCommanders,
		}
		copy(sp.CmdRos.CommanderIDs, cmdRoster.CommanderIDs)
	}

	return json.Marshal(sp)
}

// --- Load ---

func (c *PlayerChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	var sp savedPlayer
	if err := json.Unmarshal(data, &sp); err != nil {
		return fmt.Errorf("failed to unmarshal player data: %w", err)
	}

	pos := savedToPosition(sp.Position)
	attr := savedToAttributes(sp.Attrs)

	entity := em.World.NewEntity()
	entity.
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(common.PositionComponent, &pos).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(common.ResourceStockpileComponent, &common.ResourceStockpile{
			Gold: sp.Resources.Gold, Iron: sp.Resources.Iron,
			Wood: sp.Resources.Wood, Stone: sp.Resources.Stone,
		})

	newID := entity.GetID()
	idMap.Register(sp.EntityID, newID)

	// Rebuild unit roster (entity IDs remapped in RemapIDs)
	if sp.UnitRos != nil {
		roster := squads.NewUnitRoster(sp.UnitRos.MaxUnits)
		for _, se := range sp.UnitRos.Entries {
			entry := &squads.UnitRosterEntry{
				UnitType:      se.UnitType,
				TotalOwned:    se.TotalOwned,
				UnitEntities:  make([]ecs.EntityID, len(se.UnitEntities)),
				UnitsInSquads: make(map[ecs.EntityID]int),
			}
			copy(entry.UnitEntities, se.UnitEntities)
			for k, v := range se.UnitsInSquads {
				entry.UnitsInSquads[k] = v
			}
			roster.Units[se.UnitType] = entry
		}
		entity.AddComponent(squads.UnitRosterComponent, roster)
	}

	// Commander roster (IDs remapped in RemapIDs)
	if sp.CmdRos != nil {
		cmdRoster := &commander.CommanderRosterData{
			CommanderIDs:  make([]ecs.EntityID, len(sp.CmdRos.CommanderIDs)),
			MaxCommanders: sp.CmdRos.MaxCommanders,
		}
		copy(cmdRoster.CommanderIDs, sp.CmdRos.CommanderIDs)
		entity.AddComponent(commander.CommanderRosterComponent, cmdRoster)
	}

	// Rebuild "players" tag — during load, gameinit.go doesn't run,
	// so the chunk must ensure this tag exists for other systems to query.
	playersTag := ecs.BuildTag(common.PlayerComponent, common.PositionComponent)
	em.WorldTags["players"] = playersTag

	// Add to position system
	if common.GlobalPositionSystem != nil {
		common.GlobalPositionSystem.AddEntity(newID, pos)
	}

	return nil
}

// --- RemapIDs ---

func (c *PlayerChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	playerTag, ok := em.WorldTags["players"]
	if !ok {
		return nil
	}

	results := em.World.Query(playerTag)
	if len(results) == 0 {
		return nil
	}

	entity := results[0].Entity

	// Remap unit roster entity IDs
	if roster := common.GetComponentType[*squads.UnitRoster](entity, squads.UnitRosterComponent); roster != nil {
		for _, entry := range roster.Units {
			for i, oldID := range entry.UnitEntities {
				entry.UnitEntities[i] = idMap.Remap(oldID)
			}
			remapped := make(map[ecs.EntityID]int)
			for oldSquadID, count := range entry.UnitsInSquads {
				newSquadID := idMap.Remap(oldSquadID)
				if newSquadID != 0 {
					remapped[newSquadID] = count
				}
			}
			entry.UnitsInSquads = remapped
		}
	}

	// Remap commander roster IDs
	if cmdRoster := common.GetComponentType[*commander.CommanderRosterData](entity, commander.CommanderRosterComponent); cmdRoster != nil {
		cmdRoster.CommanderIDs = idMap.RemapSlice(cmdRoster.CommanderIDs)
	}

	return nil
}
