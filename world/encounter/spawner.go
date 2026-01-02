package encounter

import (
	"game_main/common"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// getEncounterImage returns the sprite path based on encounter type
func getEncounterImage(encounterType string) string {
	switch encounterType {
	case "goblin_basic":
		return "../assets/creatures/goblin.png"
	case "bandit_basic":
		return "../assets/creatures/human.png"
	case "beast_basic":
		return "../assets/creatures/anaconda.png"
	case "orc_basic":
		return "../assets/creatures/orc.png"
	default:
		return "../assets/creatures/goblin.png" // Fallback
	}
}

// SpawnRandomEncounter creates an encounter entity at a specific position
func SpawnRandomEncounter(
	manager *common.EntityManager,
	position coords.LogicalPosition,
	name string,
	level int,
	encounterType string,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	entity.AddComponent(OverworldEncounterComponent, &OverworldEncounterData{
		Name:          name,
		Level:         level,
		EncounterType: encounterType,
		IsDefeated:    false,
	})

	// Add position component
	entity.AddComponent(common.PositionComponent, &position)

	// Register in GlobalPositionSystem for O(1) collision detection
	common.GlobalPositionSystem.AddEntity(entity.GetID(), position)

	// Add renderable component so encounter is visible on map
	imagePath := getEncounterImage(encounterType)
	img, _, err := ebitenutil.NewImageFromFile(imagePath)
	if err != nil {
		log.Printf("Warning: Could not load encounter image at %s: %v\n", imagePath, err)
	} else {
		entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
			Image:   img,
			Visible: true,
		})
	}

	return entity.GetID()
}

// SpawnTestEncounters creates 3-5 random encounters for testing
func SpawnTestEncounters(manager *common.EntityManager, playerStartPos coords.LogicalPosition) {
	encounters := []struct {
		offsetX int
		offsetY int
		name    string
		level   int
		encType string
	}{
		{0, -2, "Goblin Patrol", 1, "goblin_basic"},  // 2 tiles North
		{2, 0, "Bandit Ambush", 2, "bandit_basic"},   // 2 tiles East
		{0, 2, "Wild Beast", 1, "beast_basic"},       // 2 tiles South
		{-2, 0, "Orc Raiders", 3, "orc_basic"},       // 2 tiles West
	}

	for _, enc := range encounters {
		pos := coords.LogicalPosition{
			X: clampPosition(playerStartPos.X+enc.offsetX, 0, 99),
			Y: clampPosition(playerStartPos.Y+enc.offsetY, 0, 79),
		}
		SpawnRandomEncounter(manager, pos, enc.name, enc.level, enc.encType)
	}
}

// clampPosition clamps position values within map bounds
func clampPosition(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
