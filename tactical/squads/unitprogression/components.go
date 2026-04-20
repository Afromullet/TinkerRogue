// Package unitprogression handles unit experience, leveling, and stat growth.
// It provides ECS components and functions for tracking XP, levels, and
// per-stat growth rates (Fire Emblem-style growth grades).
package unitprogression

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// Component variables for experience and stat growth
var (
	ExperienceComponent *ecs.Component
	StatGrowthComponent *ecs.Component
)

// init registers the unitprogression subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		ExperienceComponent = em.World.NewComponent()
		StatGrowthComponent = em.World.NewComponent()
	})
}

// ExperienceData tracks a unit's level and XP progress.
type ExperienceData struct {
	Level         int // Current level (starts at 1)
	CurrentXP     int // XP accumulated toward next level
	XPToNextLevel int // XP required to level up (fixed 100)
}

// GrowthGrade represents a stat growth rate grade.
type GrowthGrade string

const (
	GradeS GrowthGrade = "S" // 90% chance
	GradeA GrowthGrade = "A" // 75% chance
	GradeB GrowthGrade = "B" // 60% chance
	GradeC GrowthGrade = "C" // 45% chance
	GradeD GrowthGrade = "D" // 30% chance
	GradeE GrowthGrade = "E" // 15% chance
	GradeF GrowthGrade = "F" // 5% chance
)

// StatGrowthData defines per-stat growth rates for a unit.
// Each field is a GrowthGrade that determines the chance of +1 on level up.
type StatGrowthData struct {
	Strength   GrowthGrade
	Dexterity  GrowthGrade
	Magic      GrowthGrade
	Leadership GrowthGrade
	Armor      GrowthGrade
	Weapon     GrowthGrade
}
