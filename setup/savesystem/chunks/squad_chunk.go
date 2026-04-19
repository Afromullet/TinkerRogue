package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/setup/savesystem"
	"game_main/tactical/powers/spells"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/tactical/squads/unitprogression"
	"game_main/templates"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&SquadChunk{})
}

// SquadChunk saves/loads all squad entities and their unit members.
type SquadChunk struct{}

func (c *SquadChunk) ChunkID() string   { return "squads" }
func (c *SquadChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedSquadChunkData struct {
	Squads []savedSquad `json:"squads"`
}

type savedSquad struct {
	EntityID           ecs.EntityID       `json:"entityID"`
	Name               string             `json:"name"`
	Formation          int                `json:"formation"`
	Morale             int                `json:"morale"`
	SquadLevel         int                `json:"squadLevel"`
	TurnCount          int                `json:"turnCount"`
	MaxUnits           int                `json:"maxUnits"`
	IsDeployed         bool               `json:"isDeployed"`
	GarrisonedAtNodeID ecs.EntityID       `json:"garrisonedAtNodeID"`
	Position           savedPosition      `json:"position"`
	Members            []savedSquadMember `json:"members"`
	Mana               *savedSquadMana    `json:"mana,omitempty"`
	SpellBook          *savedSpellBook    `json:"spellBook,omitempty"`
}

type savedSquadMana struct {
	CurrentMana int `json:"currentMana"`
	MaxMana     int `json:"maxMana"`
}

type savedSpellBook struct {
	SpellIDs []string `json:"spellIDs"`
}

func spellIDsToStrings(ids []templates.SpellID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

func stringsToSpellIDs(ids []string) []templates.SpellID {
	out := make([]templates.SpellID, len(ids))
	for i, id := range ids {
		out[i] = templates.SpellID(id)
	}
	return out
}

type savedSquadMember struct {
	EntityID      ecs.EntityID       `json:"entityID"`
	SquadID       ecs.EntityID       `json:"squadID"`
	Name          string             `json:"name"`
	UnitType      string             `json:"unitType"`
	Attrs         savedAttributes    `json:"attributes"`
	GridPos       savedGridPosition  `json:"gridPosition"`
	Role          int                `json:"role"`
	TargetRow     *savedTargetRow    `json:"targetRow,omitempty"`
	Cover         *savedCover        `json:"cover,omitempty"`
	AttackRange   int                `json:"attackRange"`
	MovementSpeed int                `json:"movementSpeed"`
	Experience    *savedExperience   `json:"experience,omitempty"`
	StatGrowth    *savedStatGrowth   `json:"statGrowth,omitempty"`
	Leader        *savedLeader       `json:"leader,omitempty"`
	AbilitySlots  *savedAbilitySlots `json:"abilitySlots,omitempty"`
	Cooldowns     *savedCooldowns    `json:"cooldowns,omitempty"`
}

type savedGridPosition struct {
	AnchorRow int `json:"anchorRow"`
	AnchorCol int `json:"anchorCol"`
	Width     int `json:"width"`
	Height    int `json:"height"`
}

type savedTargetRow struct {
	AttackType  int      `json:"attackType"`
	TargetCells [][2]int `json:"targetCells,omitempty"`
}

type savedCover struct {
	CoverValue     float64 `json:"coverValue"`
	CoverRange     int     `json:"coverRange"`
	RequiresActive bool    `json:"requiresActive"`
}

type savedExperience struct {
	Level         int `json:"level"`
	CurrentXP     int `json:"currentXP"`
	XPToNextLevel int `json:"xpToNextLevel"`
}

type savedStatGrowth struct {
	Strength   string `json:"strength"`
	Dexterity  string `json:"dexterity"`
	Magic      string `json:"magic"`
	Leadership string `json:"leadership"`
	Armor      string `json:"armor"`
	Weapon     string `json:"weapon"`
}

type savedLeader struct {
	Leadership int `json:"leadership"`
	Experience int `json:"experience"`
}

type savedAbilitySlots struct {
	Slots [4]savedAbilitySlot `json:"slots"`
}

type savedAbilitySlot struct {
	AbilityType  int     `json:"abilityType"`
	TriggerType  int     `json:"triggerType"`
	Threshold    float64 `json:"threshold"`
	HasTriggered bool    `json:"hasTriggered"`
	IsEquipped   bool    `json:"isEquipped"`
}

type savedCooldowns struct {
	Cooldowns    [4]int `json:"cooldowns"`
	MaxCooldowns [4]int `json:"maxCooldowns"`
}

// --- Save ---

func (c *SquadChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	chunkData := savedSquadChunkData{}

	for _, result := range em.World.Query(squadcore.SquadTag) {
		entity := result.Entity
		squadData := common.GetComponentType[*squadcore.SquadData](entity, squadcore.SquadComponent)
		if squadData == nil {
			continue
		}

		ss := savedSquad{
			EntityID:           entity.GetID(),
			Name:               squadData.Name,
			Formation:          int(squadData.Formation),
			Morale:             squadData.Morale,
			SquadLevel:         squadData.SquadLevel,
			TurnCount:          squadData.TurnCount,
			MaxUnits:           squadData.MaxUnits,
			IsDeployed:         squadData.IsDeployed,
			GarrisonedAtNodeID: squadData.GarrisonedAtNodeID,
		}

		if pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); pos != nil {
			ss.Position = positionToSaved(pos)
		}

		// Save mana and spellbook if present
		if mana := common.GetComponentType[*spells.ManaData](entity, spells.ManaComponent); mana != nil {
			ss.Mana = &savedSquadMana{CurrentMana: mana.CurrentMana, MaxMana: mana.MaxMana}
		}
		if sb := common.GetComponentType[*spells.SpellBookData](entity, spells.SpellBookComponent); sb != nil {
			ss.SpellBook = &savedSpellBook{SpellIDs: spellIDsToStrings(sb.SpellIDs)}
		}

		// Save all member units
		unitIDs := squadcore.GetUnitIDsInSquad(entity.GetID(), em)
		for _, unitID := range unitIDs {
			unitEntity := em.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			sm := saveSquadMember(unitEntity, unitID, em)
			ss.Members = append(ss.Members, sm)
		}

		chunkData.Squads = append(chunkData.Squads, ss)
	}

	return json.Marshal(chunkData)
}

func saveSquadMember(entity *ecs.Entity, entityID ecs.EntityID, em *common.EntityManager) savedSquadMember {
	sm := savedSquadMember{EntityID: entityID}

	if memberData := common.GetComponentType[*squadcore.SquadMemberData](entity, squadcore.SquadMemberComponent); memberData != nil {
		sm.SquadID = memberData.SquadID
	}

	if nameData := common.GetComponentType[*common.Name](entity, common.NameComponent); nameData != nil {
		sm.Name = nameData.NameStr
	}

	if utData := common.GetComponentType[*squadcore.UnitTypeData](entity, squadcore.UnitTypeComponent); utData != nil {
		sm.UnitType = utData.UnitType
	}

	if attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent); attr != nil {
		sm.Attrs = attributesToSaved(attr)
	}

	if gp := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent); gp != nil {
		sm.GridPos = savedGridPosition{
			AnchorRow: gp.AnchorRow, AnchorCol: gp.AnchorCol,
			Width: gp.Width, Height: gp.Height,
		}
	}

	if roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent); roleData != nil {
		sm.Role = int(roleData.Role)
	}

	if trData := common.GetComponentType[*squadcore.TargetRowData](entity, squadcore.TargetRowComponent); trData != nil {
		sm.TargetRow = &savedTargetRow{
			AttackType:  int(trData.AttackType),
			TargetCells: trData.TargetCells,
		}
	}

	if coverData := common.GetComponentType[*squadcore.CoverData](entity, squadcore.CoverComponent); coverData != nil {
		sm.Cover = &savedCover{
			CoverValue: coverData.CoverValue, CoverRange: coverData.CoverRange,
			RequiresActive: coverData.RequiresActive,
		}
	}

	if arData := common.GetComponentType[*squadcore.AttackRangeData](entity, squadcore.AttackRangeComponent); arData != nil {
		sm.AttackRange = arData.Range
	}

	if msData := common.GetComponentType[*squadcore.MovementSpeedData](entity, squadcore.MovementSpeedComponent); msData != nil {
		sm.MovementSpeed = msData.Speed
	}

	if expData := common.GetComponentType[*unitprogression.ExperienceData](entity, unitprogression.ExperienceComponent); expData != nil {
		sm.Experience = &savedExperience{
			Level: expData.Level, CurrentXP: expData.CurrentXP,
			XPToNextLevel: expData.XPToNextLevel,
		}
	}

	if sgData := common.GetComponentType[*unitprogression.StatGrowthData](entity, unitprogression.StatGrowthComponent); sgData != nil {
		sm.StatGrowth = &savedStatGrowth{
			Strength: string(sgData.Strength), Dexterity: string(sgData.Dexterity),
			Magic: string(sgData.Magic), Leadership: string(sgData.Leadership),
			Armor: string(sgData.Armor), Weapon: string(sgData.Weapon),
		}
	}

	if leaderData := common.GetComponentType[*squadcore.LeaderData](entity, squadcore.LeaderComponent); leaderData != nil {
		sm.Leader = &savedLeader{
			Leadership: leaderData.Leadership, Experience: leaderData.Experience,
		}
	}

	if abilityData := common.GetComponentType[*squadcore.AbilitySlotData](entity, squadcore.AbilitySlotComponent); abilityData != nil {
		sa := &savedAbilitySlots{}
		for i, slot := range abilityData.Slots {
			sa.Slots[i] = savedAbilitySlot{
				AbilityType: int(slot.AbilityType), TriggerType: int(slot.TriggerType),
				Threshold: slot.Threshold, HasTriggered: slot.HasTriggered,
				IsEquipped: slot.IsEquipped,
			}
		}
		sm.AbilitySlots = sa
	}

	if cdData := common.GetComponentType[*squadcore.CooldownTrackerData](entity, squadcore.CooldownTrackerComponent); cdData != nil {
		sm.Cooldowns = &savedCooldowns{
			Cooldowns: cdData.Cooldowns, MaxCooldowns: cdData.MaxCooldowns,
		}
	}

	return sm
}

// --- Load ---

func (c *SquadChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	var chunkData savedSquadChunkData
	if err := json.Unmarshal(data, &chunkData); err != nil {
		return fmt.Errorf("failed to unmarshal squad data: %w", err)
	}

	for _, ss := range chunkData.Squads {
		// Create squad entity
		pos := savedToPosition(ss.Position)
		squadEntity := em.World.NewEntity()
		newSquadID := squadEntity.GetID()

		squadEntity.
			AddComponent(squadcore.SquadComponent, &squadcore.SquadData{
				SquadID:            newSquadID,
				Name:               ss.Name,
				Formation:          squadcore.FormationType(ss.Formation),
				Morale:             ss.Morale,
				SquadLevel:         ss.SquadLevel,
				TurnCount:          ss.TurnCount,
				MaxUnits:           ss.MaxUnits,
				IsDeployed:         ss.IsDeployed,
				GarrisonedAtNodeID: ss.GarrisonedAtNodeID, // remapped later
			})

		// Atomically add position component and register with position system
		em.RegisterEntityPosition(squadEntity, pos)

		// Restore mana and spellbook if present
		if ss.Mana != nil {
			squadEntity.AddComponent(spells.ManaComponent, &spells.ManaData{
				CurrentMana: ss.Mana.CurrentMana,
				MaxMana:     ss.Mana.MaxMana,
			})
		}
		if ss.SpellBook != nil {
			squadEntity.AddComponent(spells.SpellBookComponent, &spells.SpellBookData{
				SpellIDs: stringsToSpellIDs(ss.SpellBook.SpellIDs),
			})
		}

		idMap.Register(ss.EntityID, newSquadID)

		// Create member units
		for _, sm := range ss.Members {
			loadSquadMember(em, sm, newSquadID, idMap)
		}
	}

	return nil
}

func loadSquadMember(em *common.EntityManager, sm savedSquadMember, newSquadID ecs.EntityID, idMap *savesystem.EntityIDMap) {
	attr := savedToAttributes(sm.Attrs)

	entity := em.World.NewEntity()
	newUnitID := entity.GetID()

	entity.
		AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{
			SquadID: newSquadID, // Already new ID
		}).
		AddComponent(common.NameComponent, &common.Name{NameStr: sm.Name}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{
			AnchorRow: sm.GridPos.AnchorRow, AnchorCol: sm.GridPos.AnchorCol,
			Width: sm.GridPos.Width, Height: sm.GridPos.Height,
		}).
		AddComponent(squadcore.UnitRoleComponent, &squadcore.UnitRoleData{
			Role: unitdefs.UnitRole(sm.Role),
		}).
		AddComponent(squadcore.AttackRangeComponent, &squadcore.AttackRangeData{
			Range: sm.AttackRange,
		}).
		AddComponent(squadcore.MovementSpeedComponent, &squadcore.MovementSpeedData{
			Speed: sm.MovementSpeed,
		}).
		AddComponent(squadcore.UnitTypeComponent, &squadcore.UnitTypeData{
			UnitType: sm.UnitType,
		})

	if sm.TargetRow != nil {
		entity.AddComponent(squadcore.TargetRowComponent, &squadcore.TargetRowData{
			AttackType:  unitdefs.AttackType(sm.TargetRow.AttackType),
			TargetCells: sm.TargetRow.TargetCells,
		})
	}

	if sm.Cover != nil {
		entity.AddComponent(squadcore.CoverComponent, &squadcore.CoverData{
			CoverValue: sm.Cover.CoverValue, CoverRange: sm.Cover.CoverRange,
			RequiresActive: sm.Cover.RequiresActive,
		})
	}

	if sm.Experience != nil {
		entity.AddComponent(unitprogression.ExperienceComponent, &unitprogression.ExperienceData{
			Level: sm.Experience.Level, CurrentXP: sm.Experience.CurrentXP,
			XPToNextLevel: sm.Experience.XPToNextLevel,
		})
	}

	if sm.StatGrowth != nil {
		entity.AddComponent(unitprogression.StatGrowthComponent, &unitprogression.StatGrowthData{
			Strength:   unitprogression.GrowthGrade(sm.StatGrowth.Strength),
			Dexterity:  unitprogression.GrowthGrade(sm.StatGrowth.Dexterity),
			Magic:      unitprogression.GrowthGrade(sm.StatGrowth.Magic),
			Leadership: unitprogression.GrowthGrade(sm.StatGrowth.Leadership),
			Armor:      unitprogression.GrowthGrade(sm.StatGrowth.Armor),
			Weapon:     unitprogression.GrowthGrade(sm.StatGrowth.Weapon),
		})
	}

	if sm.Leader != nil {
		entity.AddComponent(squadcore.LeaderComponent, &squadcore.LeaderData{
			Leadership: sm.Leader.Leadership, Experience: sm.Leader.Experience,
		})
	}

	if sm.AbilitySlots != nil {
		asd := &squadcore.AbilitySlotData{}
		for i, slot := range sm.AbilitySlots.Slots {
			asd.Slots[i] = squadcore.AbilitySlot{
				AbilityType: squadcore.AbilityType(slot.AbilityType),
				TriggerType: squadcore.TriggerType(slot.TriggerType),
				Threshold:   slot.Threshold, HasTriggered: slot.HasTriggered,
				IsEquipped: slot.IsEquipped,
			}
		}
		entity.AddComponent(squadcore.AbilitySlotComponent, asd)
	}

	if sm.Cooldowns != nil {
		entity.AddComponent(squadcore.CooldownTrackerComponent, &squadcore.CooldownTrackerData{
			Cooldowns: sm.Cooldowns.Cooldowns, MaxCooldowns: sm.Cooldowns.MaxCooldowns,
		})
	}

	idMap.Register(sm.EntityID, newUnitID)
}

// --- RemapIDs ---

func (c *SquadChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	// Remap GarrisonedAtNodeID on squad entities
	for _, result := range em.World.Query(squadcore.SquadTag) {
		squadData := common.GetComponentType[*squadcore.SquadData](result.Entity, squadcore.SquadComponent)
		if squadData != nil && squadData.GarrisonedAtNodeID != 0 {
			squadData.GarrisonedAtNodeID = idMap.Remap(squadData.GarrisonedAtNodeID)
		}
	}

	// SquadMemberData.SquadID was already set to newSquadID during Load,
	// so no remapping needed for members.
	return nil
}
