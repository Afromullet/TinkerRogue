package encounter

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// MapZone defines a region on the map where encounters spawn
type MapZone struct {
	CenterX      int
	CenterY      int
	Radius       int
	BaseLevel    int // Difficulty tier for this zone
	EncounterMin int // Min encounters to spawn
	EncounterMax int // Max encounters to spawn
}

// SpawnRandomEncounters generates 10+ encounters across map zones
// Returns list of created encounter entity IDs
func SpawnRandomEncounters(
	manager *common.EntityManager,
	playerStartPos coords.LogicalPosition,
) []ecs.EntityID {
	zones := GenerateMapZones(playerStartPos)
	encounterIDs := []ecs.EntityID{}

	encounterTypes := []string{
		string(EncounterGoblinBasic),
		string(EncounterBanditBasic),
		string(EncounterBeastBasic),
		string(EncounterOrcBasic),
	}

	for _, zone := range zones {
		encounterCount := common.GetRandomBetween(zone.EncounterMin, zone.EncounterMax)

		for i := 0; i < encounterCount; i++ {
			// Random position within zone radius
			pos := generateRandomPositionInZone(zone, manager)

			// Random encounter type
			encType := encounterTypes[common.RandomInt(len(encounterTypes))]

			// Generate name with level indicator
			name := generateEncounterName(encType, zone.BaseLevel)

			// Spawn encounter entity
			encounterID := SpawnRandomEncounter(manager, pos, name, zone.BaseLevel, encType)
			encounterIDs = append(encounterIDs, encounterID)
		}
	}

	return encounterIDs
}

// GenerateMapZones creates difficulty regions radiating from player start
// Returns 4 zones with escalating difficulty
func GenerateMapZones(playerStartPos coords.LogicalPosition) []MapZone {
	return []MapZone{
		// Starting zone (easiest)
		{
			CenterX:      playerStartPos.X,
			CenterY:      playerStartPos.Y,
			Radius:       15,
			BaseLevel:    1,
			EncounterMin: 3,
			EncounterMax: 5,
		},

		// Mid-tier zone northwest (moderate difficulty)
		{
			CenterX:      clampPosition(playerStartPos.X-30, 0, 99),
			CenterY:      clampPosition(playerStartPos.Y-20, 0, 79),
			Radius:       12,
			BaseLevel:    2,
			EncounterMin: 2,
			EncounterMax: 4,
		},

		// Mid-tier zone southeast (hard)
		{
			CenterX:      clampPosition(playerStartPos.X+30, 0, 99),
			CenterY:      clampPosition(playerStartPos.Y+20, 0, 79),
			Radius:       12,
			BaseLevel:    3,
			EncounterMin: 2,
			EncounterMax: 3,
		},

		// Boss zone south (hardest)
		{
			CenterX:      playerStartPos.X,
			CenterY:      clampPosition(playerStartPos.Y+40, 0, 79),
			Radius:       10,
			BaseLevel:    5,
			EncounterMin: 1,
			EncounterMax: 2,
		},
	}
}

// generateRandomPositionInZone picks random position within zone, avoiding occupied tiles
func generateRandomPositionInZone(zone MapZone, manager *common.EntityManager) coords.LogicalPosition {
	// Try up to 10 times to find unoccupied position
	for attempts := 0; attempts < 10; attempts++ {
		// Use circular distribution
		angle := common.RandomFloat() * 2.0 * math.Pi
		distance := common.RandomInt(zone.Radius)

		offsetX := int(math.Round(float64(distance) * math.Cos(angle)))
		offsetY := int(math.Round(float64(distance) * math.Sin(angle)))

		x := clampPosition(zone.CenterX+offsetX, 0, 99)
		y := clampPosition(zone.CenterY+offsetY, 0, 79)

		pos := coords.LogicalPosition{X: x, Y: y}

		// Check if position is occupied
		if !isPositionOccupied(pos, manager) {
			return pos
		}
	}

	// Fallback to zone center if all attempts fail
	return coords.LogicalPosition{X: zone.CenterX, Y: zone.CenterY}
}

// isPositionOccupied checks if any entity exists at the given position
func isPositionOccupied(pos coords.LogicalPosition, manager *common.EntityManager) bool {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	return len(entityIDs) > 0
}

// generateEncounterName creates descriptive names with level indicators
func generateEncounterName(encounterType string, level int) string {
	prefixes := map[string][]string{
		string(EncounterGoblinBasic): {"Goblin Scouts", "Goblin Raiders", "Goblin Warband"},
		string(EncounterBanditBasic): {"Bandit Ambush", "Highwaymen", "Outlaw Gang"},
		string(EncounterBeastBasic):  {"Wild Beasts", "Feral Pack", "Prowling Hunters"},
		string(EncounterOrcBasic):    {"Orc Raiders", "Orc Warband", "Orc Champions"},
	}

	typeNames := prefixes[encounterType]
	if typeNames == nil || len(typeNames) == 0 {
		// Fallback for unknown types
		return fmt.Sprintf("Unknown Encounter (Lv.%d)", level)
	}

	baseName := typeNames[common.RandomInt(len(typeNames))]

	return fmt.Sprintf("%s (Lv.%d)", baseName, level)
}
