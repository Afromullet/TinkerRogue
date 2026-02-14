package squads

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// UnitRosterComponent marks the UnitRoster component
var UnitRosterComponent *ecs.Component

// UnitRosterEntry represents a count of units of a specific template
type UnitRosterEntry struct {
	TemplateName  string               // Name of the template
	TotalOwned    int                  // Total number of this unit type owned
	UnitsInSquads map[ecs.EntityID]int // Map of squadID -> count of units in that squad
	UnitEntities  []ecs.EntityID       // Individual unit entity IDs for this template
}

// UnitRoster tracks all units owned by the player
type UnitRoster struct {
	Units    map[string]*UnitRosterEntry // Map of template name -> roster entry
	MaxUnits int                         // Maximum number of units player can own
}

// NewUnitRoster creates a new unit roster with a maximum capacity
func NewUnitRoster(maxUnits int) *UnitRoster {
	return &UnitRoster{
		Units:    make(map[string]*UnitRosterEntry),
		MaxUnits: maxUnits,
	}
}

// getTotalUnitCount returns total number of individual units owned
func (ur *UnitRoster) getTotalUnitCount() int {
	total := 0
	for _, entry := range ur.Units {
		total += entry.TotalOwned
	}
	return total
}

// CanAddUnit checks if roster has space for another unit
func (ur *UnitRoster) CanAddUnit() bool {
	return ur.getTotalUnitCount() < ur.MaxUnits
}

// AddUnit adds a unit to the roster
// Returns error if roster is full
func (ur *UnitRoster) AddUnit(unitEntityID ecs.EntityID, templateName string) error {
	if !ur.CanAddUnit() {
		return fmt.Errorf("roster is full: %d/%d units", ur.getTotalUnitCount(), ur.MaxUnits)
	}

	// Get or create entry for this template
	entry, exists := ur.Units[templateName]
	if !exists {
		entry = &UnitRosterEntry{
			TemplateName:  templateName,
			TotalOwned:    0,
			UnitsInSquads: make(map[ecs.EntityID]int),
			UnitEntities:  make([]ecs.EntityID, 0),
		}
		ur.Units[templateName] = entry
	}

	entry.TotalOwned++
	entry.UnitEntities = append(entry.UnitEntities, unitEntityID)
	return nil
}

// RemoveUnit removes a unit from the roster by entity ID
// Returns true if unit was found and removed
func (ur *UnitRoster) RemoveUnit(unitEntityID ecs.EntityID) bool {
	for templateName, entry := range ur.Units {
		for i, id := range entry.UnitEntities {
			if id == unitEntityID {
				// Remove entity ID
				entry.UnitEntities[i] = entry.UnitEntities[len(entry.UnitEntities)-1]
				entry.UnitEntities = entry.UnitEntities[:len(entry.UnitEntities)-1]
				entry.TotalOwned--

				// Remove entry if no units left
				if entry.TotalOwned == 0 {
					delete(ur.Units, templateName)
				}
				return true
			}
		}
	}
	return false
}

// GetAvailableUnits returns all unit types that have available units (not in squad)
func (ur *UnitRoster) GetAvailableUnits() []*UnitRosterEntry {
	available := make([]*UnitRosterEntry, 0)
	for _, entry := range ur.Units {
		if ur.GetAvailableCount(entry.TemplateName) > 0 {
			available = append(available, entry)
		}
	}
	return available
}

// GetAvailableCount returns how many units of a template are available (not in squad)
func (ur *UnitRoster) GetAvailableCount(templateName string) int {
	entry, exists := ur.Units[templateName]
	if !exists {
		return 0
	}

	inSquadCount := 0
	for _, count := range entry.UnitsInSquads {
		inSquadCount += count
	}

	return entry.TotalOwned - inSquadCount
}

// MarkUnitInSquad marks a unit as being assigned to a squad
func (ur *UnitRoster) MarkUnitInSquad(unitEntityID ecs.EntityID, squadID ecs.EntityID) error {
	// Find which template this unit belongs to
	for _, entry := range ur.Units {
		for _, id := range entry.UnitEntities {
			if id == unitEntityID {
				entry.UnitsInSquads[squadID]++
				return nil
			}
		}
	}
	return fmt.Errorf("unit not found in roster: %d", unitEntityID)
}

// MarkUnitAvailable marks a unit as no longer assigned to a squad
func (ur *UnitRoster) MarkUnitAvailable(unitEntityID ecs.EntityID) error {
	// Find which template and squad this unit belongs to
	for _, entry := range ur.Units {
		for _, id := range entry.UnitEntities {
			if id == unitEntityID {
				// Find and decrement the squad count
				for squadID, count := range entry.UnitsInSquads {
					if count > 0 {
						entry.UnitsInSquads[squadID]--
						if entry.UnitsInSquads[squadID] == 0 {
							delete(entry.UnitsInSquads, squadID)
						}
						return nil
					}
				}
				return fmt.Errorf("unit %d not marked as in any squad", unitEntityID)
			}
		}
	}
	return fmt.Errorf("unit not found in roster: %d", unitEntityID)
}

// GetUnitCount returns current/max unit counts
func (ur *UnitRoster) GetUnitCount() (int, int) {
	return ur.getTotalUnitCount(), ur.MaxUnits
}

// GetUnitEntityForTemplate gets an available unit entity ID for placing in squad
// Returns 0 if no available units of this template
func (ur *UnitRoster) GetUnitEntityForTemplate(templateName string) ecs.EntityID {
	entry, exists := ur.Units[templateName]
	if !exists || ur.GetAvailableCount(templateName) == 0 {
		return 0
	}

	// Return first available entity
	// In future, could track which specific entities are in squads
	// For now, just return first entity from the list
	if len(entry.UnitEntities) > 0 {
		return entry.UnitEntities[0]
	}
	return 0
}

// GetPlayerRoster retrieves player's unit roster from ECS
// Returns nil if player has no roster component
func GetPlayerRoster(playerID ecs.EntityID, manager *common.EntityManager) *UnitRoster {
	return common.GetComponentTypeByID[*UnitRoster](manager, playerID, UnitRosterComponent)
}

// RegisterSquadUnitInRoster registers an existing squad unit in the roster
// Used to backfill roster with units that were created before roster tracking
// Returns error if roster is full or unit data is invalid
func RegisterSquadUnitInRoster(roster *UnitRoster, unitID ecs.EntityID, squadID ecs.EntityID, manager *common.EntityManager) error {
	// Get unit name to determine template
	nameStr := "Unknown"
	if nameComp, ok := manager.GetComponent(unitID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			nameStr = name.NameStr
		}
	}

	// Add unit to roster (will increment TotalOwned and add to UnitEntities)
	if err := roster.AddUnit(unitID, nameStr); err != nil {
		return fmt.Errorf("failed to add unit to roster: %w", err)
	}

	// Mark as in squad (will increment UnitsInSquads count)
	if err := roster.MarkUnitInSquad(unitID, squadID); err != nil {
		return fmt.Errorf("failed to mark unit in squad: %w", err)
	}

	return nil
}
