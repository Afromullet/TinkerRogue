package overworld

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// PlayerResourcesData tracks player's strategic resources
type PlayerResourcesData struct {
	Gold       int      // Currency for purchasing units/upgrades
	Experience int      // XP for squad leveling
	Reputation int      // Influence with factions
	Items      []string // Item IDs from rewards
}

// GetPlayerResources retrieves player's resource component
func GetPlayerResources(manager *common.EntityManager, playerID ecs.EntityID) *PlayerResourcesData {
	entity := manager.FindEntityByID(playerID)
	if entity == nil {
		return nil
	}

	return common.GetComponentType[*PlayerResourcesData](entity, PlayerResourcesComponent)
}

// GrantResources adds resources to player (replaces placeholder GrantRewards)
func GrantResources(manager *common.EntityManager, playerID ecs.EntityID, rewards RewardTable) error {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		// Create resources component if it doesn't exist
		entity := manager.FindEntityByID(playerID)
		if entity == nil {
			return fmt.Errorf("player entity %d not found", playerID)
		}

		resources = &PlayerResourcesData{
			Gold:       0,
			Experience: 0,
			Reputation: 0,
			Items:      []string{},
		}
		entity.AddComponent(PlayerResourcesComponent, resources)
	}

	// Add rewards
	resources.Gold += rewards.Gold
	resources.Experience += rewards.Experience
	resources.Items = append(resources.Items, rewards.Items...)

	fmt.Printf("Granted rewards to player %d: %d gold, %d XP, %d items\n",
		playerID, rewards.Gold, rewards.Experience, len(rewards.Items))

	return nil
}

// SpendGold deducts gold from player
func SpendGold(manager *common.EntityManager, playerID ecs.EntityID, amount int) error {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		return fmt.Errorf("player has no resources component")
	}

	if resources.Gold < amount {
		return fmt.Errorf("insufficient gold: have %d, need %d", resources.Gold, amount)
	}

	resources.Gold -= amount
	return nil
}

// CanAfford checks if player can afford a cost
func CanAfford(manager *common.EntityManager, playerID ecs.EntityID, cost int) bool {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		return false
	}

	return resources.Gold >= cost
}

// GetGold returns player's current gold
func GetGold(manager *common.EntityManager, playerID ecs.EntityID) int {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		return 0
	}

	return resources.Gold
}

// GetExperience returns player's current XP
func GetExperience(manager *common.EntityManager, playerID ecs.EntityID) int {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		return 0
	}

	return resources.Experience
}

// AddReputation modifies player's reputation
func AddReputation(manager *common.EntityManager, playerID ecs.EntityID, amount int) {
	resources := GetPlayerResources(manager, playerID)
	if resources != nil {
		resources.Reputation += amount

		// Cap reputation at -100 to +100
		if resources.Reputation > 100 {
			resources.Reputation = 100
		}
		if resources.Reputation < -100 {
			resources.Reputation = -100
		}
	}
}

// GetReputation returns player's current reputation
func GetReputation(manager *common.EntityManager, playerID ecs.EntityID) int {
	resources := GetPlayerResources(manager, playerID)
	if resources == nil {
		return 0
	}

	return resources.Reputation
}

// InitializePlayerResources creates starting resources for player
func InitializePlayerResources(manager *common.EntityManager, playerID ecs.EntityID, startingGold int) {
	entity := manager.FindEntityByID(playerID)
	if entity == nil {
		return
	}

	// Check if already has resources
	if GetPlayerResources(manager, playerID) != nil {
		return
	}

	entity.AddComponent(PlayerResourcesComponent, &PlayerResourcesData{
		Gold:       startingGold,
		Experience: 0,
		Reputation: 0,
		Items:      []string{},
	})

	fmt.Printf("Initialized player resources: %d gold\n", startingGold)
}
