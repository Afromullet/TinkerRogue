package common

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// ResourceStockpileComponent marks the unified resource component for all entities (player + factions)
var ResourceStockpileComponent *ecs.Component

// ResourceStockpile tracks gold, iron, wood, and stone for any entity
type ResourceStockpile struct {
	Gold  int
	Iron  int
	Wood  int
	Stone int
}

// NewResourceStockpile creates a new resource stockpile with the given values
func NewResourceStockpile(gold, iron, wood, stone int) *ResourceStockpile {
	return &ResourceStockpile{
		Gold:  gold,
		Iron:  iron,
		Wood:  wood,
		Stone: stone,
	}
}

// CanAffordGold checks if the stockpile has enough gold
func CanAffordGold(stockpile *ResourceStockpile, cost int) bool {
	return stockpile.Gold >= cost
}

// SpendGold deducts gold from the stockpile. Returns error if insufficient funds.
func SpendGold(stockpile *ResourceStockpile, amount int) error {
	if !CanAffordGold(stockpile, amount) {
		return fmt.Errorf("insufficient gold: have %d, need %d", stockpile.Gold, amount)
	}
	stockpile.Gold -= amount
	return nil
}

// AddGold adds gold to the stockpile
func AddGold(stockpile *ResourceStockpile, amount int) {
	stockpile.Gold += amount
}

// CanAffordMaterials checks if the stockpile has enough iron, wood, and stone
func CanAffordMaterials(stockpile *ResourceStockpile, iron, wood, stone int) bool {
	return stockpile.Iron >= iron &&
		stockpile.Wood >= wood &&
		stockpile.Stone >= stone
}

// SpendMaterials deducts iron, wood, and stone. Returns error if insufficient.
func SpendMaterials(stockpile *ResourceStockpile, iron, wood, stone int) error {
	if !CanAffordMaterials(stockpile, iron, wood, stone) {
		return fmt.Errorf("insufficient resources: have %d/%d/%d, need %d/%d/%d (Iron/Wood/Stone)",
			stockpile.Iron, stockpile.Wood, stockpile.Stone,
			iron, wood, stone)
	}
	stockpile.Iron -= iron
	stockpile.Wood -= wood
	stockpile.Stone -= stone
	return nil
}

// AddMaterials adds iron, wood, and stone to the stockpile
func AddMaterials(stockpile *ResourceStockpile, iron, wood, stone int) {
	stockpile.Iron += iron
	stockpile.Wood += wood
	stockpile.Stone += stone
}

// GetResourceStockpile retrieves the unified resource stockpile for an entity.
// Returns nil if the entity has no ResourceStockpileComponent.
func GetResourceStockpile(entityID ecs.EntityID, manager *EntityManager) *ResourceStockpile {
	return GetComponentTypeByID[*ResourceStockpile](manager, entityID, ResourceStockpileComponent)
}
