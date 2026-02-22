package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/savesystem"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&SquadChunk{})
}

// SquadChunk saves/loads all squad entities and their unit members.
type SquadChunk struct{}

func (c *SquadChunk) ChunkID() string  { return "squads" }
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
}

type savedSquadMember struct {
	EntityID      ecs.EntityID         `json:"entityID"`
	SquadID       ecs.EntityID         `json:"squadID"`
	Name          string               `json:"name"`
	UnitType      string               `json:"unitType"`
	Attrs         savedAttributes      `json:"attributes"`
	GridPos       savedGridPosition    `json:"gridPosition"`
	Role          int                  `json:"role"`
	TargetRow     *savedTargetRow      `json:"targetRow,omitempty"`
	Cover         *savedCover          `json:"cover,omitempty"`
	AttackRange   int                  `json:"attackRange"`
	MovementSpeed int                  `json:"movementSpeed"`
	Experience    *savedExperience     `json:"experience,omitempty"`
	StatGrowth    *savedStatGrowth     `json:"statGrowth,omitempty"`
	Leader        *savedLeader         `json:"leader,omitempty"`
	AbilitySlots  *savedAbilitySlots   `json:"abilitySlots,omitempty"`
	Cooldowns     *savedCooldowns      `json:"cooldowns,omitempty"`
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

	for _, result := range em.World.Query(squads.SquadTag) {
		entity := result.Entity
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
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

		// Save all member units
		unitIDs := squads.GetUnitIDsInSquad(entity.GetID(), em)
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

	if memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent); memberData != nil {
		sm.SquadID = memberData.SquadID
	}

	if nameData := common.GetComponentType[*common.Name](entity, common.NameComponent); nameData != nil {
		sm.Name = nameData.NameStr
	}

	if utData := common.GetComponentType[*squads.UnitTypeData](entity, squads.UnitTypeComponent); utData != nil {
		sm.UnitType = utData.UnitType
	}

	if attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent); attr != nil {
		sm.Attrs = attributesToSaved(attr)
	}

	if gp := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent); gp != nil {
		sm.GridPos = savedGridPosition{
			AnchorRow: gp.AnchorRow, AnchorCol: gp.AnchorCol,
			Width: gp.Width, Height: gp.Height,
		}
	}

	if roleData := common.GetComponentType[*squads.UnitRoleData](entity, squads.UnitRoleComponent); roleData != nil {
		sm.Role = int(roleData.Role)
	}

	if trData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent); trData != nil {
		sm.TargetRow = &savedTargetRow{
			AttackType:  int(trData.AttackType),
			TargetCells: trData.TargetCells,
		}
	}

	if coverData := common.GetComponentType[*squads.CoverData](entity, squads.CoverComponent); coverData != nil {
		sm.Cover = &savedCover{
			CoverValue: coverData.CoverValue, CoverRange: coverData.CoverRange,
			RequiresActive: coverData.RequiresActive,
		}
	}

	if arData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent); arData != nil {
		sm.AttackRange = arData.Range
	}

	if msData := common.GetComponentType[*squads.MovementSpeedData](entity, squads.MovementSpeedComponent); msData != nil {
		sm.MovementSpeed = msData.Speed
	}

	if expData := common.GetComponentType[*squads.ExperienceData](entity, squads.ExperienceComponent); expData != nil {
		sm.Experience = &savedExperience{
			Level: expData.Level, CurrentXP: expData.CurrentXP,
			XPToNextLevel: expData.XPToNextLevel,
		}
	}

	if sgData := common.GetComponentType[*squads.StatGrowthData](entity, squads.StatGrowthComponent); sgData != nil {
		sm.StatGrowth = &savedStatGrowth{
			Strength: string(sgData.Strength), Dexterity: string(sgData.Dexterity),
			Magic: string(sgData.Magic), Leadership: string(sgData.Leadership),
			Armor: string(sgData.Armor), Weapon: string(sgData.Weapon),
		}
	}

	if leaderData := common.GetComponentType[*squads.LeaderData](entity, squads.LeaderComponent); leaderData != nil {
		sm.Leader = &savedLeader{
			Leadership: leaderData.Leadership, Experience: leaderData.Experience,
		}
	}

	if abilityData := common.GetComponentType[*squads.AbilitySlotData](entity, squads.AbilitySlotComponent); abilityData != nil {
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

	if cdData := common.GetComponentType[*squads.CooldownTrackerData](entity, squads.CooldownTrackerComponent); cdData != nil {
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
			AddComponent(squads.SquadComponent, &squads.SquadData{
				SquadID:            newSquadID,
				Name:               ss.Name,
				Formation:          squads.FormationType(ss.Formation),
				Morale:             ss.Morale,
				SquadLevel:         ss.SquadLevel,
				TurnCount:          ss.TurnCount,
				MaxUnits:           ss.MaxUnits,
				IsDeployed:         ss.IsDeployed,
				GarrisonedAtNodeID: ss.GarrisonedAtNodeID, // remapped later
			}).
			AddComponent(common.PositionComponent, &pos)

		idMap.Register(ss.EntityID, newSquadID)

		// Register squad with position system
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.AddEntity(newSquadID, pos)
		}

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
		AddComponent(squads.SquadMemberComponent, &squads.SquadMemberData{
			SquadID: newSquadID, // Already new ID
		}).
		AddComponent(common.NameComponent, &common.Name{NameStr: sm.Name}).
		AddComponent(common.AttributeComponent, &attr).
		AddComponent(squads.GridPositionComponent, &squads.GridPositionData{
			AnchorRow: sm.GridPos.AnchorRow, AnchorCol: sm.GridPos.AnchorCol,
			Width: sm.GridPos.Width, Height: sm.GridPos.Height,
		}).
		AddComponent(squads.UnitRoleComponent, &squads.UnitRoleData{
			Role: squads.UnitRole(sm.Role),
		}).
		AddComponent(squads.AttackRangeComponent, &squads.AttackRangeData{
			Range: sm.AttackRange,
		}).
		AddComponent(squads.MovementSpeedComponent, &squads.MovementSpeedData{
			Speed: sm.MovementSpeed,
		}).
		AddComponent(squads.UnitTypeComponent, &squads.UnitTypeData{
			UnitType: sm.UnitType,
		})

	if sm.TargetRow != nil {
		entity.AddComponent(squads.TargetRowComponent, &squads.TargetRowData{
			AttackType:  squads.AttackType(sm.TargetRow.AttackType),
			TargetCells: sm.TargetRow.TargetCells,
		})
	}

	if sm.Cover != nil {
		entity.AddComponent(squads.CoverComponent, &squads.CoverData{
			CoverValue: sm.Cover.CoverValue, CoverRange: sm.Cover.CoverRange,
			RequiresActive: sm.Cover.RequiresActive,
		})
	}

	if sm.Experience != nil {
		entity.AddComponent(squads.ExperienceComponent, &squads.ExperienceData{
			Level: sm.Experience.Level, CurrentXP: sm.Experience.CurrentXP,
			XPToNextLevel: sm.Experience.XPToNextLevel,
		})
	}

	if sm.StatGrowth != nil {
		entity.AddComponent(squads.StatGrowthComponent, &squads.StatGrowthData{
			Strength:   squads.GrowthGrade(sm.StatGrowth.Strength),
			Dexterity:  squads.GrowthGrade(sm.StatGrowth.Dexterity),
			Magic:      squads.GrowthGrade(sm.StatGrowth.Magic),
			Leadership: squads.GrowthGrade(sm.StatGrowth.Leadership),
			Armor:      squads.GrowthGrade(sm.StatGrowth.Armor),
			Weapon:     squads.GrowthGrade(sm.StatGrowth.Weapon),
		})
	}

	if sm.Leader != nil {
		entity.AddComponent(squads.LeaderComponent, &squads.LeaderData{
			Leadership: sm.Leader.Leadership, Experience: sm.Leader.Experience,
		})
	}

	if sm.AbilitySlots != nil {
		asd := &squads.AbilitySlotData{}
		for i, slot := range sm.AbilitySlots.Slots {
			asd.Slots[i] = squads.AbilitySlot{
				AbilityType: squads.AbilityType(slot.AbilityType),
				TriggerType: squads.TriggerType(slot.TriggerType),
				Threshold:   slot.Threshold, HasTriggered: slot.HasTriggered,
				IsEquipped: slot.IsEquipped,
			}
		}
		entity.AddComponent(squads.AbilitySlotComponent, asd)
	}

	if sm.Cooldowns != nil {
		entity.AddComponent(squads.CooldownTrackerComponent, &squads.CooldownTrackerData{
			Cooldowns: sm.Cooldowns.Cooldowns, MaxCooldowns: sm.Cooldowns.MaxCooldowns,
		})
	}

	idMap.Register(sm.EntityID, newUnitID)
}

// --- RemapIDs ---

func (c *SquadChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	// Remap GarrisonedAtNodeID on squad entities
	for _, result := range em.World.Query(squads.SquadTag) {
		squadData := common.GetComponentType[*squads.SquadData](result.Entity, squads.SquadComponent)
		if squadData != nil && squadData.GarrisonedAtNodeID != 0 {
			squadData.GarrisonedAtNodeID = idMap.Remap(squadData.GarrisonedAtNodeID)
		}
	}

	// SquadMemberData.SquadID was already set to newSquadID during Load,
	// so no remapping needed for members.
	return nil
}
