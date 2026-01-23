package framework

import (
	"game_main/common"
	"game_main/world/coords"

	"testing"

	"github.com/bytearena/ecs"
)

// setupTestQueries creates a test environment with GUIQueries
func setupTestQueries() (*GUIQueries, *common.EntityManager) {
	ecsManager := common.NewEntityManager()

	// Initialize required components
	common.PositionComponent = ecsManager.World.NewComponent()
	common.NameComponent = ecsManager.World.NewComponent()
	common.AttributeComponent = ecsManager.World.NewComponent()
	common.PlayerComponent = ecsManager.World.NewComponent()

	// Initialize global position system
	common.GlobalPositionSystem = common.NewPositionSystem(ecsManager.World)

	queries := NewGUIQueries(ecsManager)
	return queries, ecsManager
}

// createTestCreature creates a test creature entity
func createTestCreature(manager *ecs.Manager, name string, pos coords.LogicalPosition, isMonster bool) ecs.EntityID {
	entity := manager.NewEntity()

	// Add name component
	entity.AddComponent(common.NameComponent, &common.Name{
		NameStr: name,
	})

	// Add position component
	entity.AddComponent(common.PositionComponent, &pos)

	// Add attributes component
	entity.AddComponent(common.AttributeComponent, &common.Attributes{
		CurrentHealth: 50,
		MaxHealth:     100,
		Strength:      10,
		Dexterity:     15,
		Magic:         8,
		Leadership:    12,
		Armor:         5,
		Weapon:        7,
	})

	entityID := entity.GetID()

	// Register position in global system
	common.GlobalPositionSystem.AddEntity(entityID, pos)

	return entityID
}

func TestGetTileInfo_Basic(t *testing.T) {
	queries, _ := setupTestQueries()

	// Query tile info
	pos := coords.LogicalPosition{X: 8, Y: 12}
	tileInfo := queries.GetTileInfo(pos)

	// Verify result
	if tileInfo == nil {
		t.Fatal("Expected tile info, got nil")
	}

	if tileInfo.Position.X != 8 || tileInfo.Position.Y != 12 {
		t.Errorf("Expected position (8, 12), got (%d, %d)", tileInfo.Position.X, tileInfo.Position.Y)
	}

	if tileInfo.TileType != "Floor" {
		t.Errorf("Expected TileType 'Floor', got '%s'", tileInfo.TileType)
	}

	if tileInfo.MovementCost != 1 {
		t.Errorf("Expected MovementCost 1, got %d", tileInfo.MovementCost)
	}

	if !tileInfo.IsWalkable {
		t.Error("Expected tile to be walkable")
	}
}

func TestGetTileInfo_WithEntity(t *testing.T) {
	queries, ecsManager := setupTestQueries()

	// Create entity at position
	pos := coords.LogicalPosition{X: 6, Y: 9}
	entityID := createTestCreature(ecsManager.World, "Occupant", pos, true)

	// Query tile info
	tileInfo := queries.GetTileInfo(pos)

	// Verify entity is reported
	if !tileInfo.HasEntity {
		t.Error("Expected tile to have entity")
	}

	if tileInfo.EntityID != entityID {
		t.Errorf("Expected entity ID %d, got %d", entityID, tileInfo.EntityID)
	}
}

func TestGetTileInfo_EmptyTile(t *testing.T) {
	queries, _ := setupTestQueries()

	// Query empty tile
	pos := coords.LogicalPosition{X: 20, Y: 30}
	tileInfo := queries.GetTileInfo(pos)

	// Verify no entity
	if tileInfo.HasEntity {
		t.Error("Expected tile to have no entity")
	}

	if tileInfo.EntityID != 0 {
		t.Errorf("Expected entity ID 0, got %d", tileInfo.EntityID)
	}
}
