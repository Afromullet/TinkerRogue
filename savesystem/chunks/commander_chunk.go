package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/savesystem"
	"game_main/tactical/commander"
	"game_main/tactical/spells"
	"game_main/tactical/squads"
	"game_main/world/coords"

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
	Mana          *savedMana          `json:"mana,omitempty"`
	SpellBook     *savedSpellBook     `json:"spellBook,omitempty"`
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

type savedMana struct {
	CurrentMana int `json:"currentMana"`
	MaxMana     int `json:"maxMana"`
}

type savedSpellBook struct {
	SpellIDs []string `json:"spellIDs"`
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

		if roster := common.GetComponentType[*squads.SquadRoster](entity, squads.SquadRosterComponent); roster != nil {
			sc.SquadRoster = &savedSquadRoster{
				OwnedSquads: make([]ecs.EntityID, len(roster.OwnedSquads)),
				MaxSquads:   roster.MaxSquads,
			}
			copy(sc.SquadRoster.OwnedSquads, roster.OwnedSquads)
		}

		if mana := common.GetComponentType[*spells.ManaData](entity, spells.ManaComponent); mana != nil {
			sc.Mana = &savedMana{CurrentMana: mana.CurrentMana, MaxMana: mana.MaxMana}
		}

		if sb := common.GetComponentType[*spells.SpellBookData](entity, spells.SpellBookComponent); sb != nil {
			sc.SpellBook = &savedSpellBook{
				SpellIDs: make([]string, len(sb.SpellIDs)),
			}
			copy(sc.SpellBook.SpellIDs, sb.SpellIDs)
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
			AddComponent(common.PositionComponent, &pos).
			AddComponent(common.AttributeComponent, &attr)

		if sc.ActionState != nil {
			entity.AddComponent(commander.CommanderActionStateComponent, &commander.CommanderActionStateData{
				CommanderID:       newID,
				HasMoved:          sc.ActionState.HasMoved,
				HasActed:          sc.ActionState.HasActed,
				MovementRemaining: sc.ActionState.MovementRemaining,
			})
		}

		if sc.SquadRoster != nil {
			roster := squads.NewSquadRoster(sc.SquadRoster.MaxSquads)
			// Copy old IDs; they'll be remapped in RemapIDs
			roster.OwnedSquads = make([]ecs.EntityID, len(sc.SquadRoster.OwnedSquads))
			copy(roster.OwnedSquads, sc.SquadRoster.OwnedSquads)
			entity.AddComponent(squads.SquadRosterComponent, roster)
		}

		if sc.Mana != nil {
			entity.AddComponent(spells.ManaComponent, &spells.ManaData{
				CurrentMana: sc.Mana.CurrentMana, MaxMana: sc.Mana.MaxMana,
			})
		}

		if sc.SpellBook != nil {
			spellIDs := make([]string, len(sc.SpellBook.SpellIDs))
			copy(spellIDs, sc.SpellBook.SpellIDs)
			entity.AddComponent(spells.SpellBookComponent, &spells.SpellBookData{
				SpellIDs: spellIDs,
			})
		}

		idMap.Register(sc.EntityID, newID)

		// Add to position system
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.AddEntity(newID, pos)
		}
	}

	return nil
}

// --- RemapIDs ---

func (c *CommanderChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	for _, result := range em.World.Query(commander.CommanderTag) {
		entity := result.Entity

		// Remap squad roster IDs
		if roster := common.GetComponentType[*squads.SquadRoster](entity, squads.SquadRosterComponent); roster != nil {
			roster.OwnedSquads = idMap.RemapSlice(roster.OwnedSquads)
		}
	}

	return nil
}
