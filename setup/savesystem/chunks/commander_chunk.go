package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/core/common"
	"game_main/setup/savesystem"
	"game_main/tactical/commander"
	rstr "game_main/tactical/squads/roster"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&CommanderChunk{})
}

// CommanderChunk saves/loads commander entities with their action state,
// position, attributes, squad roster, mana, and spellbook.
type CommanderChunk struct{}

func (c *CommanderChunk) ChunkID() string  { return "commanders" }
func (c *CommanderChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedCommanderChunkData struct {
	Commanders []savedCommander `json:"commanders"`
}

type savedCommander struct {
	EntityID      ecs.EntityID        `json:"entityID"`
	Name          string              `json:"name"`
	IsActive      bool                `json:"isActive"`
	Position      savedPosition       `json:"position"`
	Attrs         savedAttributes     `json:"attributes"`
	ActionState   *savedActionState   `json:"actionState,omitempty"`
	SquadRoster   *savedSquadRoster   `json:"squadRoster,omitempty"`
}

type savedActionState struct {
	HasMoved          bool `json:"hasMoved"`
	HasActed          bool `json:"hasActed"`
	MovementRemaining int  `json:"movementRemaining"`
}

type savedSquadRoster struct {
	OwnedSquads []ecs.EntityID `json:"ownedSquads"`
	MaxSquads   int            `json:"maxSquads"`
}

// --- Save ---

func (c *CommanderChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	chunkData := savedCommanderChunkData{}

	for _, result := range em.World.Query(commander.CommanderTag) {
		entity := result.Entity
		cmdData := common.GetComponentType[*commander.CommanderData](entity, commander.CommanderComponent)
		if cmdData == nil {
			continue
		}

		sc := savedCommander{
			EntityID: entity.GetID(),
			Name:     cmdData.Name,
			IsActive: cmdData.IsActive,
		}

		if pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); pos != nil {
			sc.Position = positionToSaved(pos)
		}

		if attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent); attr != nil {
			sc.Attrs = attributesToSaved(attr)
		}

		if actionState := common.GetComponentType[*commander.CommanderActionStateData](entity, commander.CommanderActionStateComponent); actionState != nil {
			sc.ActionState = &savedActionState{
				HasMoved:          actionState.HasMoved,
				HasActed:          actionState.HasActed,
				MovementRemaining: actionState.MovementRemaining,
			}
		}

		if roster := common.GetComponentType[*rstr.SquadRoster](entity, rstr.SquadRosterComponent); roster != nil {
			sc.SquadRoster = &savedSquadRoster{
				OwnedSquads: make([]ecs.EntityID, len(roster.OwnedSquads)),
				MaxSquads:   roster.MaxSquads,
			}
			copy(sc.SquadRoster.OwnedSquads, roster.OwnedSquads)
		}

		chunkData.Commanders = append(chunkData.Commanders, sc)
	}

	return json.Marshal(chunkData)
}

// --- Load ---

func (c *CommanderChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	var chunkData savedCommanderChunkData
	if err := json.Unmarshal(data, &chunkData); err != nil {
		return fmt.Errorf("failed to unmarshal commander data: %w", err)
	}

	for _, sc := range chunkData.Commanders {
		pos := savedToPosition(sc.Position)
		attr := savedToAttributes(sc.Attrs)

		entity := em.World.NewEntity()
		newID := entity.GetID()

		entity.
			AddComponent(commander.CommanderComponent, &commander.CommanderData{
				CommanderID: newID,
				Name:        sc.Name,
				IsActive:    sc.IsActive,
			}).
			AddComponent(common.AttributeComponent, &attr)

		// Atomically add position component and register with position system
		em.RegisterEntityPosition(entity, pos)

		if sc.ActionState != nil {
			entity.AddComponent(commander.CommanderActionStateComponent, &commander.CommanderActionStateData{
				CommanderID:       newID,
				HasMoved:          sc.ActionState.HasMoved,
				HasActed:          sc.ActionState.HasActed,
				MovementRemaining: sc.ActionState.MovementRemaining,
			})
		}

		if sc.SquadRoster != nil {
			roster := rstr.NewSquadRoster(sc.SquadRoster.MaxSquads)
			// Copy old IDs; they'll be remapped in RemapIDs
			roster.OwnedSquads = make([]ecs.EntityID, len(sc.SquadRoster.OwnedSquads))
			copy(roster.OwnedSquads, sc.SquadRoster.OwnedSquads)
			entity.AddComponent(rstr.SquadRosterComponent, roster)
		}

		idMap.Register(sc.EntityID, newID)

	}

	return nil
}

// --- RemapIDs ---

func (c *CommanderChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	for _, result := range em.World.Query(commander.CommanderTag) {
		entity := result.Entity

		// Remap squad roster IDs
		if roster := common.GetComponentType[*rstr.SquadRoster](entity, rstr.SquadRosterComponent); roster != nil {
			roster.OwnedSquads = idMap.RemapSlice(roster.OwnedSquads)
		}
	}

	return nil
}
