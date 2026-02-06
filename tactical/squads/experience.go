package squads

import (
	"game_main/common"
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
// Also recalculates MaxHealth since Strength may have changed.
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

	// Roll each stat independently against its growth chance
	if rng.Intn(100) < GrowthChance(growthData.Strength) {
		attr.Strength++
	}
	if rng.Intn(100) < GrowthChance(growthData.Dexterity) {
		attr.Dexterity++
	}
	if rng.Intn(100) < GrowthChance(growthData.Magic) {
		attr.Magic++
	}
	if rng.Intn(100) < GrowthChance(growthData.Leadership) {
		attr.Leadership++
	}
	if rng.Intn(100) < GrowthChance(growthData.Armor) {
		attr.Armor++
	}
	if rng.Intn(100) < GrowthChance(growthData.Weapon) {
		attr.Weapon++
	}

	// Recalculate MaxHealth since Strength may have changed
	attr.MaxHealth = attr.GetMaxHealth()
}

// GetExperienceData returns the ExperienceData for a unit, or nil if not found.
func GetExperienceData(unitID ecs.EntityID, manager *common.EntityManager) *ExperienceData {
	return common.GetComponentTypeByID[*ExperienceData](manager, unitID, ExperienceComponent)
}

// GetStatGrowthData returns the StatGrowthData for a unit, or nil if not found.
func GetStatGrowthData(unitID ecs.EntityID, manager *common.EntityManager) *StatGrowthData {
	return common.GetComponentTypeByID[*StatGrowthData](manager, unitID, StatGrowthComponent)
}
