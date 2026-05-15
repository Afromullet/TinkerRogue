// Package roster manages unit and squad rosters - tracking which units and squads
// a player owns, their assignment status, and capacity limits.
package roster

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// UnitRosterComponent marks the UnitRoster component
var UnitRosterComponent *ecs.Component

// UnitRosterEntry represents a count of units of a specific template
type UnitRosterEntry struct {
	UnitType      string               // Unit type identifier
	TotalOwned    int                  // Total number of this unit type owned
	UnitsInSquads map[ecs.EntityID]int // Map of squadID -> count of units in that squad
	UnitEntities  []ecs.EntityID       // Individual unit entity IDs for this template
}

// UnitRoster tracks all units owned by the player
type UnitRoster struct {
	Units    map[string]*UnitRosterEntry // Map of template name -> roster entry
	MaxUnits int                         // Maximum number of units player can own

	// entryByEntity maps a unit entity ID to the unitType key in Units, so
	// MarkUnitInSquad/MarkUnitAvailable can locate the owning entry in O(1)
	// instead of walking every entry's UnitEntities slice.
	entryByEntity map[ecs.EntityID]string
}

// NewUnitRoster creates a new unit roster with a maximum capacity
func NewUnitRoster(maxUnits int) *UnitRoster {
	return &UnitRoster{
		Units:         make(map[string]*UnitRosterEntry),
		MaxUnits:      maxUnits,
		entryByEntity: make(map[ecs.EntityID]string),
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
func (ur *UnitRoster) AddUnit(unitEntityID ecs.EntityID, unitType string) error {
	if !ur.CanAddUnit() {
		return fmt.Errorf("roster is full: %d/%d units", ur.getTotalUnitCount(), ur.MaxUnits)
	}

	entry, exists := ur.Units[unitType]
	if !exists {
		entry = &UnitRosterEntry{
			UnitType:      unitType,
			TotalOwned:    0,
			UnitsInSquads: make(map[ecs.EntityID]int),
			UnitEntities:  make([]ecs.EntityID, 0),
		}
		ur.Units[unitType] = entry
	}

	entry.TotalOwned++
	entry.UnitEntities = append(entry.UnitEntities, unitEntityID)
	ur.entryByEntity[unitEntityID] = unitType
	return nil
}

// RemoveUnit removes a unit from the roster by entity ID
func (ur *UnitRoster) RemoveUnit(unitEntityID ecs.EntityID) bool {
	templateName, ok := ur.entryByEntity[unitEntityID]
	if !ok {
		return false
	}
	entry, ok := ur.Units[templateName]
	if !ok {
		delete(ur.entryByEntity, unitEntityID)
		return false
	}

	for i, id := range entry.UnitEntities {
		if id == unitEntityID {
			entry.UnitEntities[i] = entry.UnitEntities[len(entry.UnitEntities)-1]
			entry.UnitEntities = entry.UnitEntities[:len(entry.UnitEntities)-1]
			entry.TotalOwned--
			delete(ur.entryByEntity, unitEntityID)

			if entry.TotalOwned == 0 {
				delete(ur.Units, templateName)
			}
			return true
		}
	}
	return false
}

// GetAvailableUnits returns all unit types that have available units (not in squad)
func (ur *UnitRoster) GetAvailableUnits() []*UnitRosterEntry {
	available := make([]*UnitRosterEntry, 0)
	for _, entry := range ur.Units {
		if ur.GetAvailableCount(entry.UnitType) > 0 {
			available = append(available, entry)
		}
	}
	return available
}

// GetAvailableCount returns how many units of a template are available (not in squad)
func (ur *UnitRoster) GetAvailableCount(unitType string) int {
	entry, exists := ur.Units[unitType]
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
	entry := ur.entryForUnit(unitEntityID)
	if entry == nil {
		return fmt.Errorf("unit not found in roster: %d", unitEntityID)
	}
	entry.UnitsInSquads[squadID]++
	return nil
}

// MarkUnitAvailable marks a unit as no longer assigned to a squad
func (ur *UnitRoster) MarkUnitAvailable(unitEntityID ecs.EntityID) error {
	entry := ur.entryForUnit(unitEntityID)
	if entry == nil {
		return fmt.Errorf("unit not found in roster: %d", unitEntityID)
	}
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

// entryForUnit returns the roster entry that owns the given unit entity, or nil
// if the unit isn't tracked. O(1) lookup via entryByEntity.
func (ur *UnitRoster) entryForUnit(unitEntityID ecs.EntityID) *UnitRosterEntry {
	unitType, ok := ur.entryByEntity[unitEntityID]
	if !ok {
		return nil
	}
	return ur.Units[unitType]
}

// GetUnitCount returns current/max unit counts
func (ur *UnitRoster) GetUnitCount() (int, int) {
	return ur.getTotalUnitCount(), ur.MaxUnits
}

// GetUnitEntityForTemplate gets an available unit entity ID for placing in squad.
// Returns the first entity that does NOT have SquadMemberComponent.
// Returns 0 if no available units of this type.
func (ur *UnitRoster) GetUnitEntityForTemplate(unitType string, manager *common.EntityManager) ecs.EntityID {
	entry, exists := ur.Units[unitType]
	if !exists {
		return 0
	}
	for _, id := range entry.UnitEntities {
		if isUnitAvailable(manager, id) {
			return id
		}
	}
	return 0
}

// RosterUnitEntry represents a single available unit for display in the roster list
type RosterUnitEntry struct {
	ID           ecs.EntityID
	Name         string
	TemplateName string
}

// GetAvailableUnitDetails returns individual entries for all available (not in squad) units.
func (ur *UnitRoster) GetAvailableUnitDetails(manager *common.EntityManager) []RosterUnitEntry {
	var results []RosterUnitEntry
	for templateName, entry := range ur.Units {
		for _, id := range entry.UnitEntities {
			if !isUnitAvailable(manager, id) {
				continue
			}
			results = append(results, RosterUnitEntry{
				ID:           id,
				Name:         common.GetEntityName(manager, id, "Unknown"),
				TemplateName: templateName,
			})
		}
	}
	return results
}

// isUnitAvailable reports whether a unit entity is not currently assigned to a squad.
// Shared by GetUnitEntityForTemplate and GetAvailableUnitDetails.
func isUnitAvailable(manager *common.EntityManager, unitID ecs.EntityID) bool {
	return !manager.HasComponent(unitID, squadcore.SquadMemberComponent)
}

// GetPlayerRoster retrieves player's unit roster from ECS
func GetPlayerRoster(playerID ecs.EntityID, manager *common.EntityManager) *UnitRoster {
	return common.GetComponentTypeByID[*UnitRoster](manager, playerID, UnitRosterComponent)
}

// RegisterSquadUnitInRoster registers an existing squad unit in the roster
func RegisterSquadUnitInRoster(roster *UnitRoster, unitID ecs.EntityID, squadID ecs.EntityID, manager *common.EntityManager) error {
	unitType := "Unknown"
	if utData := common.GetComponentTypeByID[*squadcore.UnitTypeData](manager, unitID, squadcore.UnitTypeComponent); utData != nil {
		unitType = utData.UnitType
	}

	if err := roster.AddUnit(unitID, unitType); err != nil {
		return fmt.Errorf("failed to add unit to roster: %w", err)
	}

	if err := roster.MarkUnitInSquad(unitID, squadID); err != nil {
		return fmt.Errorf("failed to mark unit in squad: %w", err)
	}

	return nil
}

// AssignUnitToSquad places an existing unit entity into a squad and updates
// roster bookkeeping atomically. If marking the roster fails after placement,
// the ECS-side placement is rolled back so the two stores cannot desync.
//
// Replaces the manual "PlaceUnitInSquad then MarkUnitInSquad" sequence used by
// AddUnitCommand and RemoveUnitCommand.Undo.
func AssignUnitToSquad(
	roster *UnitRoster,
	unitID ecs.EntityID,
	squadID ecs.EntityID,
	gridRow, gridCol int,
	manager *common.EntityManager,
) error {
	if roster == nil {
		return fmt.Errorf("roster is nil")
	}
	if err := squadcore.PlaceUnitInSquad(squadID, unitID, manager, gridRow, gridCol); err != nil {
		return fmt.Errorf("failed to place unit in squad: %w", err)
	}
	if err := roster.MarkUnitInSquad(unitID, squadID); err != nil {
		_ = squadcore.UnassignUnitFromSquad(unitID, manager)
		return fmt.Errorf("failed to mark unit in roster: %w", err)
	}
	return nil
}

// UnassignUnitFromSquad removes a unit from its squad and updates roster
// bookkeeping atomically. If the ECS-side unassign fails after the roster has
// been marked available, the roster bookkeeping is restored.
//
// Replaces the manual "MarkUnitAvailable then UnassignUnitFromSquad" sequence
// used by RemoveUnitCommand and AddUnitCommand.Undo.
func UnassignUnitFromSquad(
	roster *UnitRoster,
	unitID ecs.EntityID,
	squadID ecs.EntityID,
	manager *common.EntityManager,
) error {
	if roster == nil {
		return fmt.Errorf("roster is nil")
	}
	if err := roster.MarkUnitAvailable(unitID); err != nil {
		return fmt.Errorf("failed to mark unit available: %w", err)
	}
	if err := squadcore.UnassignUnitFromSquad(unitID, manager); err != nil {
		_ = roster.MarkUnitInSquad(unitID, squadID)
		return fmt.Errorf("failed to unassign unit from squad: %w", err)
	}
	return nil
}
