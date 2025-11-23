package common

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// PlayerResourcesComponent marks the PlayerResources component
var PlayerResourcesComponent *ecs.Component

// PlayerResources tracks player's currencies and resources
type PlayerResources struct {
	Gold int // Primary currency for purchasing units
}

// NewPlayerResources creates a new player resources instance with starting values
func NewPlayerResources(startingGold int) *PlayerResources {
	return &PlayerResources{
		Gold: startingGold,
	}
}

// CanAfford checks if player has enough gold for a purchase
func (pr *PlayerResources) CanAfford(cost int) bool {
	return pr.Gold >= cost
}

// SpendGold deducts gold from player resources
// Returns error if insufficient funds
func (pr *PlayerResources) SpendGold(amount int) error {
	if !pr.CanAfford(amount) {
		return fmt.Errorf("insufficient gold: have %d, need %d", pr.Gold, amount)
	}
	pr.Gold -= amount
	return nil
}

// AddGold adds gold to player resources
func (pr *PlayerResources) AddGold(amount int) {
	pr.Gold += amount
}

// GetPlayerResources retrieves player resources from ECS
// Returns nil if player has no resources component
func GetPlayerResources(playerID ecs.EntityID, manager *EntityManager) *PlayerResources {
	return GetComponentTypeByID[*PlayerResources](manager, playerID, PlayerResourcesComponent)
}
