package squadservices

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// UnitPurchaseService encapsulates all unit purchasing game logic
type UnitPurchaseService struct {
	entityManager *common.EntityManager
}

// NewUnitPurchaseService creates a new unit purchase service
func NewUnitPurchaseService(manager *common.EntityManager) *UnitPurchaseService {
	return &UnitPurchaseService{
		entityManager: manager,
	}
}

// PurchaseResult contains information about a purchase transaction
type PurchaseResult struct {
	Success        bool
	Error          string
	UnitID         ecs.EntityID
	UnitName       string
	CostPaid       int
	RemainingGold  int
	RosterCount    int
	RosterCapacity int
}

// PurchaseValidationResult contains validation information
type PurchaseValidationResult struct {
	CanPurchase    bool
	Error          string
	PlayerGold     int
	UnitCost       int
	RosterCount    int
	RosterCapacity int
}

// PlayerPurchaseInfo contains read-only purchase information
type PlayerPurchaseInfo struct {
	Gold           int
	RosterCount    int
	RosterCapacity int
	CanAddUnit     bool
}

// GetAvailableUnitsForPurchase returns all unit templates available for purchase
func (ups *UnitPurchaseService) GetAvailableUnitsForPurchase() []squads.UnitTemplate {
	// Return all templates (in future, could filter by tier, availability, etc.)
	templates := make([]squads.UnitTemplate, len(squads.Units))
	copy(templates, squads.Units)
	return templates
}

// GetUnitCost calculates the cost of a unit template
func (ups *UnitPurchaseService) GetUnitCost(template squads.UnitTemplate) int {
	// Simple cost formula based on unit name hash
	// TODO: Add cost field to UnitTemplate or JSON data
	baseCost := 100
	for _, c := range template.Name {
		baseCost += int(c) % 50
	}
	return baseCost
}

// CanPurchaseUnit validates if player can purchase a unit
func (ups *UnitPurchaseService) CanPurchaseUnit(playerID ecs.EntityID, template squads.UnitTemplate) *PurchaseValidationResult {
	result := &PurchaseValidationResult{}

	// Get player resources
	resources := common.GetPlayerResources(playerID, ups.entityManager)
	if resources == nil {
		result.Error = "player resources not found"
		return result
	}

	// Get player roster
	roster := squads.GetPlayerRoster(playerID, ups.entityManager)
	if roster == nil {
		result.Error = "player roster not found"
		return result
	}

	// Get unit cost
	cost := ups.GetUnitCost(template)

	// Fill in info
	result.PlayerGold = resources.Gold
	result.UnitCost = cost
	result.RosterCount, result.RosterCapacity = roster.GetUnitCount()

	// Check affordability
	if !resources.CanAfford(cost) {
		result.Error = fmt.Sprintf("insufficient gold: need %d, have %d", cost, resources.Gold)
		return result
	}

	// Check roster capacity
	if !roster.CanAddUnit() {
		result.Error = fmt.Sprintf("roster is full: %d/%d units", result.RosterCount, result.RosterCapacity)
		return result
	}

	// Validation passed
	result.CanPurchase = true
	return result
}

// PurchaseUnit handles the complete purchase transaction atomically with rollback on failure
func (ups *UnitPurchaseService) PurchaseUnit(playerID ecs.EntityID, template squads.UnitTemplate) *PurchaseResult {
	result := &PurchaseResult{
		UnitName: template.Name,
	}

	// Validate purchase
	validation := ups.CanPurchaseUnit(playerID, template)
	if !validation.CanPurchase {
		result.Error = validation.Error
		return result
	}

	// Get resources and roster
	resources := common.GetPlayerResources(playerID, ups.entityManager)
	roster := squads.GetPlayerRoster(playerID, ups.entityManager)
	cost := ups.GetUnitCost(template)

	// Step 1: Create unit entity from template
	unitEntity, err := squads.CreateUnitEntity(ups.entityManager, template)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create unit: %v", err)
		return result
	}
	unitID := unitEntity.GetID()

	// Step 2: Add to roster (with rollback on failure)
	if err := roster.AddUnit(unitID, template.Name); err != nil {
		// Rollback: Dispose entity
		ups.entityManager.World.DisposeEntities(unitEntity)
		result.Error = fmt.Sprintf("failed to add to roster: %v", err)
		return result
	}

	// Step 3: Spend gold (with rollback on failure)
	if err := resources.SpendGold(cost); err != nil {
		// Rollback: Remove from roster and dispose entity
		roster.RemoveUnit(unitID)
		ups.entityManager.World.DisposeEntities(unitEntity)
		result.Error = fmt.Sprintf("failed to spend gold: %v", err)
		return result
	}

	// Transaction successful - populate result
	result.Success = true
	result.UnitID = unitID
	result.CostPaid = cost
	result.RemainingGold = resources.Gold
	rosterCount, rosterCapacity := roster.GetUnitCount()
	result.RosterCount = rosterCount
	result.RosterCapacity = rosterCapacity

	return result
}

// GetPlayerPurchaseInfo returns read-only information about player's purchasing state
func (ups *UnitPurchaseService) GetPlayerPurchaseInfo(playerID ecs.EntityID) *PlayerPurchaseInfo {
	info := &PlayerPurchaseInfo{}

	// Get player resources
	resources := common.GetPlayerResources(playerID, ups.entityManager)
	if resources != nil {
		info.Gold = resources.Gold
	}

	// Get player roster
	roster := squads.GetPlayerRoster(playerID, ups.entityManager)
	if roster != nil {
		info.RosterCount, info.RosterCapacity = roster.GetUnitCount()
		info.CanAddUnit = roster.CanAddUnit()
	}

	return info
}

// GetUnitOwnedCount returns how many units of a template the player owns
func (ups *UnitPurchaseService) GetUnitOwnedCount(playerID ecs.EntityID, templateName string) (totalOwned, available int) {
	roster := squads.GetPlayerRoster(playerID, ups.entityManager)
	if roster == nil {
		return 0, 0
	}

	entry, exists := roster.Units[templateName]
	if !exists {
		return 0, 0
	}

	available = roster.GetAvailableCount(templateName)
	return entry.TotalOwned, available
}
