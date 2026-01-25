package overworld

import (
	"game_main/common"
	testfx "game_main/testing"
	"testing"
)

func TestGetPlayerResources(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player entity
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	// Add resources component
	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:       100,
		Experience: 50,
		Reputation: 10,
		Items:      []string{"sword", "shield"},
	})

	// Retrieve resources
	resources := GetPlayerResources(manager, playerID)

	if resources == nil {
		t.Fatal("Failed to retrieve player resources")
	}

	if resources.Gold != 100 {
		t.Errorf("Expected Gold=100, got %d", resources.Gold)
	}

	if resources.Experience != 50 {
		t.Errorf("Expected Experience=50, got %d", resources.Experience)
	}

	if resources.Reputation != 10 {
		t.Errorf("Expected Reputation=10, got %d", resources.Reputation)
	}

	if len(resources.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resources.Items))
	}
}

func TestGetPlayerResources_NoComponent(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player entity without resources component
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	resources := GetPlayerResources(manager, playerID)

	if resources != nil {
		t.Error("Expected nil when player has no resources component")
	}
}

func TestGetPlayerResources_InvalidID(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Query non-existent player
	resources := GetPlayerResources(manager, 9999)

	if resources != nil {
		t.Error("Expected nil for non-existent player ID")
	}
}

func TestGrantResources(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player entity
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	// Add initial resources
	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:       100,
		Experience: 50,
		Reputation: 0,
		Items:      []string{},
	})

	// Grant rewards
	rewards := RewardTable{
		Gold:       50,
		Experience: 25,
		Items:      []string{"potion", "scroll"},
	}

	err := GrantResources(manager, playerID, rewards)
	if err != nil {
		t.Fatalf("Failed to grant resources: %v", err)
	}

	// Verify resources updated
	resources := GetPlayerResources(manager, playerID)
	if resources.Gold != 150 {
		t.Errorf("Expected Gold=150 after grant, got %d", resources.Gold)
	}

	if resources.Experience != 75 {
		t.Errorf("Expected Experience=75 after grant, got %d", resources.Experience)
	}

	if len(resources.Items) != 2 {
		t.Errorf("Expected 2 items after grant, got %d", len(resources.Items))
	}
}

func TestGrantResources_CreatesComponent(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player entity WITHOUT resources component
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	// Grant rewards (should create component)
	rewards := RewardTable{
		Gold:       100,
		Experience: 50,
		Items:      []string{"item1"},
	}

	err := GrantResources(manager, playerID, rewards)
	if err != nil {
		t.Fatalf("Failed to grant resources: %v", err)
	}

	// Verify component was created
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		t.Fatal("Resources component should have been created")
	}

	if resources.Gold != 100 {
		t.Errorf("Expected Gold=100, got %d", resources.Gold)
	}

	if resources.Experience != 50 {
		t.Errorf("Expected Experience=50, got %d", resources.Experience)
	}

	if len(resources.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(resources.Items))
	}
}

func TestGrantResources_InvalidPlayerID(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Try to grant resources to non-existent player
	rewards := RewardTable{
		Gold:       100,
		Experience: 50,
	}

	err := GrantResources(manager, 9999, rewards)

	if err == nil {
		t.Error("Expected error when granting resources to non-existent player")
	}
}

func TestSpendGold(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player with resources
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:       100,
		Experience: 50,
	})

	// Spend gold
	err := SpendGold(manager, playerID, 30)

	if err != nil {
		t.Errorf("Expected successful gold spending, got error: %v", err)
	}

	// Verify gold reduced
	resources := GetPlayerResources(manager, playerID)
	if resources.Gold != 70 {
		t.Errorf("Expected Gold=70 after spending 30, got %d", resources.Gold)
	}
}

func TestSpendGold_InsufficientFunds(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player with limited resources
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:       50,
		Experience: 0,
	})

	// Try to spend more than available
	err := SpendGold(manager, playerID, 100)

	if err == nil {
		t.Error("Expected error when spending more gold than available")
	}

	// Verify gold unchanged
	resources := GetPlayerResources(manager, playerID)
	if resources.Gold != 50 {
		t.Errorf("Expected Gold=50 (unchanged), got %d", resources.Gold)
	}
}

func TestSpendGold_NoResources(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player without resources component
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	err := SpendGold(manager, playerID, 10)

	if err == nil {
		t.Error("Expected error when player has no resources component")
	}
}

func TestCanAfford(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player with resources
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold: 100,
	})

	// Check for amounts
	if !CanAfford(manager, playerID, 50) {
		t.Error("Expected true when player has enough gold")
	}

	if !CanAfford(manager, playerID, 100) {
		t.Error("Expected true when player has exactly enough gold")
	}

	if CanAfford(manager, playerID, 150) {
		t.Error("Expected false when player doesn't have enough gold")
	}
}

func TestCanAfford_NoResources(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player without resources
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	if CanAfford(manager, playerID, 10) {
		t.Error("Expected false when player has no resources component")
	}
}

func TestInitializePlayerResources(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player entity
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	// Initialize resources
	InitializePlayerResources(manager, playerID, 200)

	// Verify created correctly
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		t.Fatal("Failed to retrieve created resources")
	}

	if resources.Gold != 200 {
		t.Errorf("Expected Gold=200, got %d", resources.Gold)
	}

	if resources.Experience != 0 {
		t.Errorf("Expected Experience=0 initially, got %d", resources.Experience)
	}

	if resources.Reputation != 0 {
		t.Errorf("Expected Reputation=0 initially, got %d", resources.Reputation)
	}

	if len(resources.Items) != 0 {
		t.Errorf("Expected 0 items initially, got %d", len(resources.Items))
	}
}

func TestGrantResources_AccumulatesItems(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player with initial items
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:  100,
		Items: []string{"sword"},
	})

	// Grant more items
	rewards := RewardTable{
		Gold:  50,
		Items: []string{"shield", "potion"},
	}

	GrantResources(manager, playerID, rewards)

	// Verify items accumulated
	resources := GetPlayerResources(manager, playerID)
	if len(resources.Items) != 3 {
		t.Errorf("Expected 3 items total, got %d", len(resources.Items))
	}

	// Check items are in order
	expectedItems := []string{"sword", "shield", "potion"}
	for i, expected := range expectedItems {
		if resources.Items[i] != expected {
			t.Errorf("Expected item[%d]=%q, got %q", i, expected, resources.Items[i])
		}
	}
}

func TestSpendGold_ExactAmount(t *testing.T) {
	manager := testfx.NewTestEntityManager()
	common.InitializeSubsystems(manager)

	// Create player with exact amount
	playerEntity := manager.World.NewEntity()
	playerID := playerEntity.GetID()

	playerEntity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold: 100,
	})

	// Spend exact amount
	err := SpendGold(manager, playerID, 100)

	if err != nil {
		t.Errorf("Expected success when spending exact gold amount, got error: %v", err)
	}

	// Verify gold is now 0
	resources := GetPlayerResources(manager, playerID)
	if resources.Gold != 0 {
		t.Errorf("Expected Gold=0 after spending all, got %d", resources.Gold)
	}
}
