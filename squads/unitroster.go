package squads

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// UnitRosterComponent marks the UnitRoster component
var UnitRosterComponent *ecs.Component

// UnitRosterEntry represents a unit owned by the player
type UnitRosterEntry struct {
	UnitEntityID ecs.EntityID // The actual unit entity
	TemplateName string       // Name of the template this was created from
	IsInSquad    bool         // Whether this unit is currently assigned to a squad
	SquadID      ecs.EntityID // Squad this unit belongs to (0 if not in squad)
}

// UnitRoster tracks all units owned by the player
type UnitRoster struct {
	Units    []UnitRosterEntry // All owned units
	MaxUnits int               // Maximum number of units player can own
}

// NewUnitRoster creates a new unit roster with a maximum capacity
func NewUnitRoster(maxUnits int) *UnitRoster {
	return &UnitRoster{
		Units:    make([]UnitRosterEntry, 0),
		MaxUnits: maxUnits,
	}
}

// CanAddUnit checks if roster has space for another unit
func (ur *UnitRoster) CanAddUnit() bool {
	return len(ur.Units) < ur.MaxUnits
}

// AddUnit adds a unit to the roster
// Returns error if roster is full
func (ur *UnitRoster) AddUnit(unitEntityID ecs.EntityID, templateName string) error {
	if !ur.CanAddUnit() {
		return fmt.Errorf("roster is full: %d/%d units", len(ur.Units), ur.MaxUnits)
	}

	entry := UnitRosterEntry{
		UnitEntityID: unitEntityID,
		TemplateName: templateName,
		IsInSquad:    false,
		SquadID:      0,
	}

	ur.Units = append(ur.Units, entry)
	return nil
}

// RemoveUnit removes a unit from the roster by entity ID
// Returns true if unit was found and removed
func (ur *UnitRoster) RemoveUnit(unitEntityID ecs.EntityID) bool {
	for i, entry := range ur.Units {
		if entry.UnitEntityID == unitEntityID {
			// Remove by replacing with last element and truncating
			ur.Units[i] = ur.Units[len(ur.Units)-1]
			ur.Units = ur.Units[:len(ur.Units)-1]
			return true
		}
	}
	return false
}

// GetAvailableUnits returns all units not currently in a squad
func (ur *UnitRoster) GetAvailableUnits() []UnitRosterEntry {
	available := make([]UnitRosterEntry, 0)
	for _, entry := range ur.Units {
		if !entry.IsInSquad {
			available = append(available, entry)
		}
	}
	return available
}

// MarkUnitInSquad marks a unit as being assigned to a squad
func (ur *UnitRoster) MarkUnitInSquad(unitEntityID ecs.EntityID, squadID ecs.EntityID) error {
	for i := range ur.Units {
		if ur.Units[i].UnitEntityID == unitEntityID {
			ur.Units[i].IsInSquad = true
			ur.Units[i].SquadID = squadID
			return nil
		}
	}
	return fmt.Errorf("unit not found in roster: %d", unitEntityID)
}

// MarkUnitAvailable marks a unit as no longer assigned to a squad
func (ur *UnitRoster) MarkUnitAvailable(unitEntityID ecs.EntityID) error {
	for i := range ur.Units {
		if ur.Units[i].UnitEntityID == unitEntityID {
			ur.Units[i].IsInSquad = false
			ur.Units[i].SquadID = 0
			return nil
		}
	}
	return fmt.Errorf("unit not found in roster: %d", unitEntityID)
}

// GetUnitCount returns current/max unit counts
func (ur *UnitRoster) GetUnitCount() (int, int) {
	return len(ur.Units), ur.MaxUnits
}

// GetPlayerRoster retrieves player's unit roster from ECS
// Returns nil if player has no roster component
func GetPlayerRoster(playerID ecs.EntityID, manager *common.EntityManager) *UnitRoster {
	return common.GetComponentTypeByID[*UnitRoster](manager, playerID, UnitRosterComponent)
}

