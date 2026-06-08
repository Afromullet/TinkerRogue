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
func (s *ResourceStockpile) CanAffordGold(cost int) bool {
	return s.Gold >= cost
}

// SpendGold deducts gold from the stockpile. Returns error if insufficient funds.
func (s *ResourceStockpile) SpendGold(amount int) error {
	if !s.CanAffordGold(amount) {
		return fmt.Errorf("insufficient gold: have %d, need %d", s.Gold, amount)
	}
	s.Gold -= amount
	return nil
}

// AddGold adds gold to the stockpile
func (s *ResourceStockpile) AddGold(amount int) {
	s.Gold += amount
}

// CanAffordMaterials checks if the stockpile has enough iron, wood, and stone
func (s *ResourceStockpile) CanAffordMaterials(iron, wood, stone int) bool {
	return s.Iron >= iron &&
		s.Wood >= wood &&
		s.Stone >= stone
}

// SpendMaterials deducts iron, wood, and stone. Returns error if insufficient.
func (s *ResourceStockpile) SpendMaterials(iron, wood, stone int) error {
	if !s.CanAffordMaterials(iron, wood, stone) {
		return fmt.Errorf("insufficient resources: have %d/%d/%d, need %d/%d/%d (Iron/Wood/Stone)",
			s.Iron, s.Wood, s.Stone,
			iron, wood, stone)
	}
	s.Iron -= iron
	s.Wood -= wood
	s.Stone -= stone
	return nil
}

// AddMaterials adds iron, wood, and stone to the stockpile
func (s *ResourceStockpile) AddMaterials(iron, wood, stone int) {
	s.Iron += iron
	s.Wood += wood
	s.Stone += stone
}

// GetResourceStockpile retrieves the unified resource stockpile for an entity.
// Returns nil if the entity has no ResourceStockpileComponent.
func GetResourceStockpile(entityID ecs.EntityID, manager *EntityManager) *ResourceStockpile {
	return GetComponentTypeByID[*ResourceStockpile](manager, entityID, ResourceStockpileComponent)
}
