package encounter

import (
	"game_main/common"
	"game_main/world/coords"
	"testing"
)

func TestCheckEncounterAtPosition(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize common components (required for PositionComponent)
	common.PositionComponent = manager.World.NewComponent()
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Initialize encounter components
	InitEncounterComponents(manager)
	InitEncounterTags(manager)

	// Spawn encounter at (10, 10)
	pos := coords.LogicalPosition{X: 10, Y: 10}
	encounterID := SpawnRandomEncounter(manager, pos, "Test Encounter", 1, "test")

	// Test collision detection
	result := CheckEncounterAtPosition(manager, pos)
	if result != encounterID {
		t.Errorf("Expected encounter ID %d, got %d", encounterID, result)
	}

	// Test no collision
	emptyPos := coords.LogicalPosition{X: 50, Y: 50}
	result = CheckEncounterAtPosition(manager, emptyPos)
	if result != 0 {
		t.Errorf("Expected no encounter, got ID %d", result)
	}
}

func TestDefeatedEncounterNotTriggered(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize common components
	common.PositionComponent = manager.World.NewComponent()
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Initialize encounter components
	InitEncounterComponents(manager)
	InitEncounterTags(manager)

	pos := coords.LogicalPosition{X: 10, Y: 10}
	encounterID := SpawnRandomEncounter(manager, pos, "Test", 1, "test")

	// Mark as defeated
	entity := manager.FindEntityByID(encounterID)
	encounterData := common.GetComponentType[*OverworldEncounterData](
		entity,
		OverworldEncounterComponent,
	)
	encounterData.IsDefeated = true

	// Should not trigger
	result := CheckEncounterAtPosition(manager, pos)
	if result != 0 {
		t.Errorf("Defeated encounter should not trigger, got ID %d", result)
	}
}

func TestSpawnRandomEncounter(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize common components
	common.PositionComponent = manager.World.NewComponent()
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Initialize encounter components
	InitEncounterComponents(manager)
	InitEncounterTags(manager)

	pos := coords.LogicalPosition{X: 25, Y: 30}
	name := "Goblin Patrol"
	level := 2
	encounterType := "goblin_basic"

	encounterID := SpawnRandomEncounter(manager, pos, name, level, encounterType)

	// Verify entity was created
	if encounterID == 0 {
		t.Fatal("SpawnRandomEncounter returned 0 entity ID")
	}

	// Verify entity has encounter component
	entity := manager.FindEntityByID(encounterID)
	if entity == nil {
		t.Fatal("Entity not found after spawning")
	}

	if !entity.HasComponent(OverworldEncounterComponent) {
		t.Error("Entity missing OverworldEncounterComponent")
	}

	// Verify encounter data
	encounterData := common.GetComponentType[*OverworldEncounterData](
		entity,
		OverworldEncounterComponent,
	)

	if encounterData == nil {
		t.Fatal("OverworldEncounterData is nil")
	}

	if encounterData.Name != name {
		t.Errorf("Expected name %s, got %s", name, encounterData.Name)
	}

	if encounterData.Level != level {
		t.Errorf("Expected level %d, got %d", level, encounterData.Level)
	}

	if encounterData.EncounterType != encounterType {
		t.Errorf("Expected type %s, got %s", encounterType, encounterData.EncounterType)
	}

	if encounterData.IsDefeated {
		t.Error("New encounter should not be marked as defeated")
	}

	// Verify position component
	if !entity.HasComponent(common.PositionComponent) {
		t.Error("Entity missing PositionComponent")
	}

	posData := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if posData == nil {
		t.Fatal("PositionData is nil")
	}

	if *posData != pos {
		t.Errorf("Expected position %+v, got %+v", pos, *posData)
	}
}

func TestSpawnTestEncounters(t *testing.T) {
	manager := common.NewEntityManager()

	// Initialize common components
	common.PositionComponent = manager.World.NewComponent()
	common.GlobalPositionSystem = common.NewPositionSystem(manager.World)

	// Initialize encounter components
	InitEncounterComponents(manager)
	InitEncounterTags(manager)

	playerStartPos := coords.LogicalPosition{X: 50, Y: 40}

	// Spawn test encounters
	SpawnTestEncounters(manager, playerStartPos)

	// Verify that encounters were created
	encounterCount := 0
	for range manager.World.Query(OverworldEncounterTag) {
		encounterCount++
	}

	if encounterCount != 4 {
		t.Errorf("Expected 4 encounters, got %d", encounterCount)
	}
}
