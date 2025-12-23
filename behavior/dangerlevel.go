package behavior

import (
	"game_main/combat"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

var ThreatLevelManager *FactionThreatLevelManager

// Keeps track of Each Factions Danger Level.
type FactionThreatLevelManager struct {
	manager  *common.EntityManager
	factions map[ecs.EntityID]*FactionThreatLevel
}

func NewFactionThreatLevelManager(manager *common.EntityManager) *FactionThreatLevelManager {
	return &FactionThreatLevelManager{
		manager:  manager,
		factions: make(map[ecs.EntityID]*FactionThreatLevel),
	}
}

func (ftlm *FactionThreatLevelManager) AddFaction(factionID ecs.EntityID) {

	if _, exists := ftlm.factions[factionID]; !exists {
		ftlm.factions[factionID] = NewFactionThreatRating(factionID, ftlm.manager)
	}

	ftlm.factions[factionID].UpdateThreatRatings()
}

func (ftlm *FactionThreatLevelManager) UpdateFaction(factionID ecs.EntityID) {

	ftlm.factions[factionID].UpdateThreatRatings()
}

func (ftlm *FactionThreatLevelManager) UpdateAllFactions() {

	factionIDs := combat.GetAllFactions(ftlm.manager)

	for _, ID := range factionIDs {
		ftlm.factions[ID].UpdateThreatRatings()
	}

}

type FactionThreatLevel struct {
	manager          *common.EntityManager
	factionID        ecs.EntityID
	squadDangerLevel map[ecs.EntityID]*SquadThreatLevel //Key is the squad ID. Value is the danger level
}

func NewFactionThreatRating(factionID ecs.EntityID, manager *common.EntityManager) *FactionThreatLevel {

	squadIDs := combat.GetSquadsForFaction(factionID, manager)

	ftl := &FactionThreatLevel{

		factionID:        factionID,
		squadDangerLevel: make(map[ecs.EntityID]*SquadThreatLevel, len(squadIDs)),
		manager:          manager,
	}

	for _, ID := range squadIDs {
		ftl.squadDangerLevel[ID] = NewSquadThreatLevel(ftl.manager, ID)
	}

	return ftl
}

func (ftr *FactionThreatLevel) UpdateThreatRatings() {

	squadIDs := combat.GetSquadsForFaction(ftr.factionID, ftr.manager)

	for _, squadID := range squadIDs {
		ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()
	}

}

func (ftr *FactionThreatLevel) UpdateThreatRatingForSquad(squadID ecs.EntityID) {
	ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()

}

type SquadThreatLevel struct {
	manager       *common.EntityManager
	squadID       ecs.EntityID
	DangerByRange map[int]float64 //Key is the range. Value is the danger level

}

func NewSquadThreatLevel(manager *common.EntityManager, squadID ecs.EntityID) *SquadThreatLevel {

	return &SquadThreatLevel{
		manager: manager,
		squadID: squadID,
	}
}

// CalculateFactionDangerLevels calculates danger levels for all squads in a faction in parallel.
// Returns a map of squadID -> danger levels by range.
func (stl *SquadThreatLevel) CalculateSquadDangerLevel() {
	unitIDs := squads.GetUnitIDsInSquad(stl.squadID, stl.manager)

	// First, collect all units with their attack data
	type unitData struct {
		entity         *ecs.Entity
		attackRange    int
		power          float64
		roleMultiplier float64
		isLeader       bool
		attackType     squads.AttackType
	}

	var units []unitData
	uniqueRanges := make(map[int]bool)

	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByID(stl.manager, unitID)
		if unitEntity == nil {
			continue
		}

		// Get unit role
		roleData := common.GetComponentType[*squads.UnitRoleData](unitEntity, squads.UnitRoleComponent)
		if roleData == nil {
			continue
		}

		// Get unit target row (attack type)
		targetRowData := common.GetComponentType[*squads.TargetRowData](unitEntity, squads.TargetRowComponent)
		if targetRowData == nil {
			continue
		}

		// Get attack range
		attackRangeData := common.GetComponentType[*squads.AttackRangeData](unitEntity, squads.AttackRangeComponent)
		attackRange := 1
		if attackRangeData != nil {
			attackRange = attackRangeData.Range
		}
		uniqueRanges[attackRange] = true

		// Get unit attributes
		attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		// Check if unit is leader by checking for LeaderComponent
		isLeader := unitEntity.HasComponent(squads.LeaderComponent)

		// Calculate base power from weapon and dexterity
		// Weapon damage is primary, dexterity improves hit rate
		basePower := float64(attr.Weapon + attr.Dexterity/2)

		// Get role multiplier
		roleMultiplier := stl.getRoleMultiplier(roleData.Role)

		units = append(units, unitData{
			entity:         unitEntity,
			attackRange:    attackRange,
			power:          basePower,
			roleMultiplier: roleMultiplier,
			isLeader:       isLeader,
			attackType:     targetRowData.AttackType,
		})
	}

	// Calculate danger at each attack range
	dangerByRange := make(map[int]float64)

	for attackRange := range uniqueRanges {
		var rangeDanger float64 = 0

		// Sum danger from units that can attack at this range
		for _, ud := range units {
			if ud.attackRange >= attackRange {
				// Apply leader bonus
				leaderBonus := 1.0
				if ud.isLeader {
					leaderBonus = 1.3
				}

				// Calculate unit danger contribution at this range
				unitDanger := ud.power * ud.roleMultiplier * leaderBonus
				rangeDanger += unitDanger
			}
		}

		dangerByRange[attackRange] = rangeDanger
	}

	// Apply composition bonus to each range
	compositionBonus := stl.calculateCompositionBonus()
	for range_, danger := range dangerByRange {
		dangerByRange[range_] = danger * compositionBonus
	}

	stl.DangerByRange = dangerByRange

}

// getRoleMultiplier returns a damage multiplier based on unit role
func (stl *SquadThreatLevel) getRoleMultiplier(role squads.UnitRole) float64 {
	switch role {
	case squads.RoleDPS:
		return 1.5 // Highest damage output
	case squads.RoleTank:
		return 1.2 // High durability
	case squads.RoleSupport:
		return 1.0 // Utility/healing
	default:
		return 1.0
	}
}

// calculateCompositionBonus returns a bonus multiplier based on attack type diversity
// Squads with diverse attack types (melee + ranged + magic) are more effective
func (stl *SquadThreatLevel) calculateCompositionBonus() float64 {
	unitIDs := squads.GetUnitIDsInSquad(stl.squadID, stl.manager)
	if len(unitIDs) == 0 {
		return 1.0
	}

	// Count different attack types in the squad
	attackTypeCount := make(map[squads.AttackType]int)
	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByID(stl.manager, unitID)
		if unitEntity == nil {
			continue
		}

		targetRowData := common.GetComponentType[*squads.TargetRowData](unitEntity, squads.TargetRowComponent)
		if targetRowData != nil {
			attackTypeCount[targetRowData.AttackType]++
		}
	}

	// Bonus for having diverse attack types
	// Pure squads (1 type) are less effective
	// Balanced squads (2-3 types) are stronger
	// Highly diverse (4 types) is ideal but rare
	uniqueTypes := len(attackTypeCount)
	switch uniqueTypes {
	case 1:
		return 0.8 // Weak - pure melee, pure ranged, or pure magic
	case 2:
		return 1.1 // Good - mixed strategies
	case 3:
		return 1.2 // Excellent - diverse
	case 4:
		return 1.3 // Optimal - all attack types
	default:
		return 1.0
	}
}
