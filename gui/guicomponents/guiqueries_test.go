package guicomponents

import (
	"game_main/common"
	"game_main/coords"
	"game_main/systems"
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
	common.MonsterComponent = ecsManager.World.NewComponent()

	// Initialize tags
	ecsManager.WorldTags["monsters"] = ecs.BuildTag(common.MonsterComponent)

	// Initialize global position system
	common.GlobalPositionSystem = systems.NewPositionSystem(ecsManager.World)

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

	// Add monster or player tag
	if isMonster {
		entity.AddComponent(common.MonsterComponent, &common.Monster{})
	} else {
		entity.AddComponent(common.PlayerComponent, &common.Player{})
	}

	entityID := entity.GetID()

	// Register position in global system
	common.GlobalPositionSystem.AddEntity(entityID, pos)

	return entityID
}

func TestGetCreatureAtPosition_Success(t *testing.T) {
	queries, ecsManager := setupTestQueries()

	// Create a test creature
	pos := coords.LogicalPosition{X: 5, Y: 10}
	creatureID := createTestCreature(ecsManager.World, "Test Monster", pos, true)

	// Query for creature at position
	creatureInfo := queries.GetCreatureAtPosition(pos)

	// Verify result
	if creatureInfo == nil {
		t.Fatal("Expected creature info, got nil")
	}

	if creatureInfo.ID != creatureID {
		t.Errorf("Expected creature ID %d, got %d", creatureID, creatureInfo.ID)
	}

	if creatureInfo.Name != "Test Monster" {
		t.Errorf("Expected name 'Test Monster', got '%s'", creatureInfo.Name)
	}

	if !creatureInfo.IsMonster {
		t.Error("Expected creature to be marked as monster")
	}

	if creatureInfo.IsPlayer {
		t.Error("Expected creature to not be marked as player")
	}

	if creatureInfo.CurrentHP != 50 {
		t.Errorf("Expected CurrentHP 50, got %d", creatureInfo.CurrentHP)
	}

	if creatureInfo.MaxHP != 100 {
		t.Errorf("Expected MaxHP 100, got %d", creatureInfo.MaxHP)
	}

	if creatureInfo.Strength != 10 {
		t.Errorf("Expected Strength 10, got %d", creatureInfo.Strength)
	}
}

func TestGetCreatureAtPosition_NoCreature(t *testing.T) {
	queries, _ := setupTestQueries()

	// Query for creature at empty position
	pos := coords.LogicalPosition{X: 5, Y: 10}
	creatureInfo := queries.GetCreatureAtPosition(pos)

	// Verify nil result
	if creatureInfo != nil {
		t.Errorf("Expected nil for empty position, got creature: %v", creatureInfo)
	}
}

func TestGetCreatureAtPosition_Player(t *testing.T) {
	queries, ecsManager := setupTestQueries()

	// Create a player entity
	pos := coords.LogicalPosition{X: 3, Y: 7}
	playerID := createTestCreature(ecsManager.World, "Hero", pos, false)

	// Query for creature at position
	creatureInfo := queries.GetCreatureAtPosition(pos)

	// Verify result
	if creatureInfo == nil {
		t.Fatal("Expected creature info, got nil")
	}

	if creatureInfo.ID != playerID {
		t.Errorf("Expected player ID %d, got %d", playerID, creatureInfo.ID)
	}

	if creatureInfo.Name != "Hero" {
		t.Errorf("Expected name 'Hero', got '%s'", creatureInfo.Name)
	}

	if creatureInfo.IsMonster {
		t.Error("Expected player to not be marked as monster")
	}

	if !creatureInfo.IsPlayer {
		t.Error("Expected player to be marked as player")
	}
}

func TestGetCreatureAtPosition_NoAttributes(t *testing.T) {
	queries, ecsManager := setupTestQueries()

	// Create creature without attributes
	pos := coords.LogicalPosition{X: 2, Y: 4}
	entity := ecsManager.World.NewEntity()
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: "Simple Entity"})
	entity.AddComponent(common.PositionComponent, &pos)
	entity.AddComponent(common.MonsterComponent, &common.Monster{})

	entityID := entity.GetID()
	common.GlobalPositionSystem.AddEntity(entityID, pos)

	// Query for creature
	creatureInfo := queries.GetCreatureAtPosition(pos)

	// Verify we get basic info even without attributes
	if creatureInfo == nil {
		t.Fatal("Expected creature info, got nil")
	}

	if creatureInfo.Name != "Simple Entity" {
		t.Errorf("Expected name 'Simple Entity', got '%s'", creatureInfo.Name)
	}

	if creatureInfo.MaxHP != 0 {
		t.Errorf("Expected MaxHP 0 (no attributes), got %d", creatureInfo.MaxHP)
	}

	if !creatureInfo.IsMonster {
		t.Error("Expected entity to be marked as monster")
	}
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
