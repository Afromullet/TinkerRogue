package unitprogression

import (
	"game_main/core/common"
	"math/rand"

	"github.com/bytearena/ecs"
)

// GrowthChance returns the percentage chance (0-100) of gaining +1 stat on level up
// for the given growth grade.
func GrowthChance(grade GrowthGrade) int {
	switch grade {
	case GradeS:
		return 90
	case GradeA:
		return 75
	case GradeB:
		return 60
	case GradeC:
		return 45
	case GradeD:
		return 30
	case GradeE:
		return 15
	case GradeF:
		return 5
	default:
		return 0
	}
}

// AwardExperience adds XP to a unit and processes any level ups that occur.
// Handles multi-level jumps when a large XP amount is awarded.
func AwardExperience(unitID ecs.EntityID, amount int, manager *common.EntityManager, rng *rand.Rand) {
	if amount <= 0 {
		return
	}

	expData := GetExperienceData(unitID, manager)
	if expData == nil {
		return
	}

	expData.CurrentXP += amount

	// Process level ups (supports multi-level jumps)
	for expData.CurrentXP >= expData.XPToNextLevel {
		expData.CurrentXP -= expData.XPToNextLevel
		expData.Level++
		processLevelUp(unitID, manager, rng)
	}
}

// processLevelUp rolls each stat against its growth rate and increments on success.
func processLevelUp(unitID ecs.EntityID, manager *common.EntityManager, rng *rand.Rand) {
	growthData := GetStatGrowthData(unitID, manager)
	if growthData == nil {
		return
	}

	entity := manager.FindEntityByID(unitID)
	if entity == nil {
		return
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return
	}

	// Roll each stat independently against its growth chance.
	// Adding a new growable stat requires only adding a row to this table
	// (plus matching fields on StatGrowthData and Attributes).
	growthRolls := []struct {
		grade GrowthGrade
		stat  *int
	}{
		{growthData.Strength, &attr.Strength},
		{growthData.Dexterity, &attr.Dexterity},
		{growthData.Magic, &attr.Magic},
		{growthData.Leadership, &attr.Leadership},
		{growthData.Armor, &attr.Armor},
		{growthData.Weapon, &attr.Weapon},
	}
	for _, roll := range growthRolls {
		if rng.Intn(100) < GrowthChance(roll.grade) {
			*roll.stat++
		}
	}
}

// GetExperienceData returns the ExperienceData for a unit, or nil if not found.
func GetExperienceData(unitID ecs.EntityID, manager *common.EntityManager) *ExperienceData {
	return common.GetComponentTypeByID[*ExperienceData](manager, unitID, ExperienceComponent)
}

// GetStatGrowthData returns the StatGrowthData for a unit, or nil if not found.
func GetStatGrowthData(unitID ecs.EntityID, manager *common.EntityManager) *StatGrowthData {
	return common.GetComponentTypeByID[*StatGrowthData](manager, unitID, StatGrowthComponent)
}
