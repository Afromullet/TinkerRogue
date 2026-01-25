package overworld

import (
	"fmt"
	"game_main/common"
	testfx "game_main/testing"
	"game_main/world/coords"
)

// ExampleInitializeOverworld demonstrates how to set up the overworld system
func ExampleInitializeOverworld() {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create tick state (singleton)
	CreateTickStateEntity(manager)

	// Verify tick state was created
	tickState := GetTickState(manager)
	fmt.Printf("Tick state created: %v\n", tickState != nil)
	fmt.Printf("Starting tick: %d\n", tickState.CurrentTick)
}

// ExampleCreateThreatNode demonstrates how to spawn threat nodes
func ExampleCreateThreatNode() {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	tickState := GetTickState(manager)
	currentTick := int64(0)
	if tickState != nil {
		currentTick = tickState.CurrentTick
	}

	// Create various threat types
	CreateThreatNode(manager, coords.LogicalPosition{X: 50, Y: 50}, ThreatNecromancer, 1, currentTick)
	CreateThreatNode(manager, coords.LogicalPosition{X: 60, Y: 45}, ThreatBanditCamp, 2, currentTick)
	CreateThreatNode(manager, coords.LogicalPosition{X: 55, Y: 55}, ThreatCorruption, 1, currentTick)

	// Count created threats
	count := CountThreatNodes(manager)
	fmt.Printf("Created %d threat nodes\n", count)
}

// ExampleAdvanceTick demonstrates how to advance the game tick
func ExampleAdvanceTick() {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Get initial tick
	tickState := GetTickState(manager)
	initialTick := tickState.CurrentTick
	fmt.Printf("Initial tick: %d\n", initialTick)

	// Advance one tick
	AdvanceTick(manager)

	// Check new tick
	tickState = GetTickState(manager)
	fmt.Printf("After advance: %d\n", tickState.CurrentTick)
}

// ExampleQueryThreats demonstrates various threat query functions
func ExampleQueryThreats() {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	tickState := GetTickState(manager)
	currentTick := tickState.CurrentTick

	// Create test threats
	CreateThreatNode(manager, coords.LogicalPosition{X: 50, Y: 50}, ThreatNecromancer, 2, currentTick)
	CreateThreatNode(manager, coords.LogicalPosition{X: 52, Y: 52}, ThreatNecromancer, 3, currentTick)
	CreateThreatNode(manager, coords.LogicalPosition{X: 100, Y: 100}, ThreatBanditCamp, 1, currentTick)

	// Query all threats
	allThreats := GetAllThreatNodes(manager)
	fmt.Printf("Total threats: %d\n", len(allThreats))

	// Query threats in radius
	centerPos := coords.LogicalPosition{X: 50, Y: 50}
	nearbyThreats := GetThreatsInRadius(manager, centerPos, 5)
	fmt.Printf("Threats within radius 5: %d\n", len(nearbyThreats))

	// Query threats by type
	necromancers := GetThreatsByType(manager, ThreatNecromancer)
	fmt.Printf("Necromancer threats: %d\n", len(necromancers))

	// Calculate average intensity
	avgIntensity := CalculateAverageIntensity(manager)
	fmt.Printf("Average intensity: %.1f\n", avgIntensity)
}

// ExampleCreateFaction demonstrates how to create and query factions
func ExampleCreateFaction() {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)
	CreateTickStateEntity(manager)

	// Create a faction
	factionPos := coords.LogicalPosition{X: 50, Y: 50}
	factionID := CreateFaction(manager, FactionNecromancers, factionPos, 10)

	// Query factions
	count := CountFactions(manager)
	fmt.Printf("Created %d faction(s)\n", count)

	faction := GetFactionByID(manager, factionID)
	if faction != nil {
		fmt.Printf("Faction type: Necromancers\n")
		fmt.Printf("Strength: %d\n", faction.Strength)
	}
}
