package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// PurchaseUnitCommand purchases a unit and adds it to the player's roster
type PurchaseUnitCommand struct {
	entityManager *common.EntityManager
	playerID      ecs.EntityID
	unitTemplate  squads.UnitTemplate
	costPaid      int

	// Captured state for undo
	purchasedUnitID ecs.EntityID
}

// NewPurchaseUnitCommand creates a new purchase unit command
func NewPurchaseUnitCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	unitTemplate squads.UnitTemplate,
) *PurchaseUnitCommand {
	return &PurchaseUnitCommand{
		entityManager: manager,
		playerID:      playerID,
		unitTemplate:  unitTemplate,
	}
}

// Validate checks if the unit can be purchased
func (cmd *PurchaseUnitCommand) Validate() error {
	if cmd.playerID == 0 {
		return fmt.Errorf("invalid player ID")
	}

	// Validate purchase
	validation := squads.CanPurchaseUnit(cmd.playerID, cmd.unitTemplate, cmd.entityManager)
	if !validation.CanPurchase {
		return fmt.Errorf("cannot purchase unit: %s", validation.Error)
	}

	return nil
}

// Execute performs the unit purchase
func (cmd *PurchaseUnitCommand) Execute() error {
	// Execute purchase transaction
	result := squads.PurchaseUnit(cmd.playerID, cmd.unitTemplate, cmd.entityManager)

	if !result.Success {
		return fmt.Errorf("purchase failed: %s", result.Error)
	}

	// Capture state for undo
	cmd.costPaid = result.CostPaid
	cmd.purchasedUnitID = result.UnitID

	return nil
}

// Undo refunds the unit and returns gold to player
func (cmd *PurchaseUnitCommand) Undo() error {
	if cmd.purchasedUnitID == 0 {
		return fmt.Errorf("no saved unit ID for undo")
	}

	// Refund the purchase
	result := squads.RefundUnitPurchase(cmd.playerID, cmd.purchasedUnitID, cmd.costPaid, cmd.entityManager)

	if !result.Success {
		return fmt.Errorf("failed to refund purchase: %s", result.Error)
	}

	return nil
}

// Description returns a human-readable description
func (cmd *PurchaseUnitCommand) Description() string {
	return fmt.Sprintf("Purchase %s for %d gold", cmd.unitTemplate.Name, cmd.costPaid)
}
