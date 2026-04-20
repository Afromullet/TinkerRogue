package spawning

import (
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/templates"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// GeneratePositionsAroundPoint creates positions distributed around a center point.
// arcStart/arcEnd define the angle range in radians (0 to 2*Pi for full circle).
// minDistance/maxDistance define the radius range from center.
func GeneratePositionsAroundPoint(
	center coords.LogicalPosition,
	count int,
	arcStart, arcEnd float64,
	minDistance, maxDistance int,
) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, count)
	arcRange := arcEnd - arcStart
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for i := 0; i < count; i++ {
		angle := arcStart + (float64(i)/float64(count))*arcRange
		distance := minDistance + (i % (maxDistance - minDistance + 1))

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		pos := coords.LogicalPosition{
			X: clampPosition(center.X+offsetX, 0, mapWidth-1),
			Y: clampPosition(center.Y+offsetY, 0, mapHeight-1),
		}
		positions[i] = pos
	}
	return positions
}

// GeneratePlayerSquadPositions creates positions for player squads around a starting point.
// Uses a forward-facing arc (-Pi/2 to Pi/2) at PlayerMinDistance..PlayerMaxDistance.
func GeneratePlayerSquadPositions(startPos coords.LogicalPosition, count int) []coords.LogicalPosition {
	return GeneratePositionsAroundPoint(startPos, count, -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
}

func clampPosition(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// clampPowerTarget applies min/max bounds from the difficulty modifier to a raw power target.
// If raw is zero or negative, returns the minimum. If above max, returns the max.
func clampPowerTarget(raw float64, mod templates.JSONEncounterDifficulty) float64 {
	if raw <= 0.0 {
		return mod.MinTargetPower
	}
	if raw > mod.MaxTargetPower {
		return mod.MaxTargetPower
	}
	return raw
}

// getDifficultyModifier retrieves difficulty settings for a given encounter level.
// Falls back to level 3 (fair fight) if level is invalid.
// Applies global difficulty overlay (power scale, squad/unit count offsets).
func getDifficultyModifier(level int) templates.JSONEncounterDifficulty {
	var result templates.JSONEncounterDifficulty

	found := false
	for _, t := range templates.EncounterDifficultyTemplates {
		if t.Level == level {
			result = t
			found = true
			break
		}
	}

	if !found {
		for _, t := range templates.EncounterDifficultyTemplates {
			if t.Level == 3 {
				result = t
				found = true
				break
			}
		}
	}

	if !found {
		result = templates.JSONEncounterDifficulty{
			Level:            3,
			Name:             "Fair Fight",
			PowerMultiplier:  1.0,
			SquadCount:       4,
			MinUnitsPerSquad: 3,
			MaxUnitsPerSquad: 5,
			MinTargetPower:   50.0,
			MaxTargetPower:   2000.0,
		}
	}

	diff := templates.GlobalDifficulty.Encounter()
	result.PowerMultiplier *= diff.PowerMultiplierScale

	result.SquadCount += diff.SquadCountOffset
	if result.SquadCount < 1 {
		result.SquadCount = 1
	}

	result.MinUnitsPerSquad += diff.MinUnitsPerSquadOffset
	if result.MinUnitsPerSquad < 1 {
		result.MinUnitsPerSquad = 1
	}

	result.MaxUnitsPerSquad += diff.MaxUnitsPerSquadOffset
	if result.MaxUnitsPerSquad < result.MinUnitsPerSquad {
		result.MaxUnitsPerSquad = result.MinUnitsPerSquad
	}

	return result
}

// calculateTargetPower computes the clamped enemy power target from a set of squad IDs.
// Averages squad power across the given IDs, scales by difficulty, and clamps to bounds.
func calculateTargetPower(
	manager *common.EntityManager,
	squadIDs []ecs.EntityID,
	config *evaluation.PowerConfig,
	difficultyMod templates.JSONEncounterDifficulty,
) float64 {
	totalPower := 0.0
	for _, squadID := range squadIDs {
		totalPower += evaluation.CalculateSquadPower(squadID, manager, config)
	}
	avgPower := totalPower / float64(len(squadIDs))
	return clampPowerTarget(avgPower*difficultyMod.PowerMultiplier, difficultyMod)
}
