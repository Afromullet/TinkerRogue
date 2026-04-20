package core

import (
	"game_main/common"
)

// ResourceCost defines the iron, wood, and stone cost for an action.
type ResourceCost struct {
	Iron  int
	Wood  int
	Stone int
}

// CanAfford returns true if the stockpile has enough resources to cover the cost.
func CanAfford(stockpile *common.ResourceStockpile, cost ResourceCost) bool {
	return common.CanAffordMaterials(stockpile, cost.Iron, cost.Wood, cost.Stone)
}

// SpendResources deducts the cost from the stockpile.
// Returns an error if the stockpile cannot afford the cost.
func SpendResources(stockpile *common.ResourceStockpile, cost ResourceCost) error {
	return common.SpendMaterials(stockpile, cost.Iron, cost.Wood, cost.Stone)
}

