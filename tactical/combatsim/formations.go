package combatsim

import (
	"fmt"
	"game_main/tactical/squads"
)

// FormationType defines tactical formation patterns
type FormationType int

const (
	FormationStandard FormationType = iota // Tank front, DPS front, Support mid, Ranged back
	FormationDefensive                     // Tank front, all others back for maximum cover
	FormationAggressive                    // DPS front, Tank mid for cover, Support/Ranged back
	FormationRanged                        // Tank front, everything else back (ranged-heavy)
	FormationBalanced                      // Mixed positioning for flexibility
)

// FormationTemplate defines row assignments by unit role
type FormationTemplate struct {
	Name       string
	TankRow    int // Row for Tank role units
	DPSRow     int // Row for DPS role units
	SupportRow int // Row for Support role units
	RangedRow  int // Row for ranged units (AttackRange > 1)
}

// GetFormationTemplate returns the formation template for the given type
func GetFormationTemplate(formType FormationType) FormationTemplate {
	switch formType {
	case FormationStandard:
		return FormationTemplate{
			Name:       "Standard",
			TankRow:    0, // Tanks in front (provide cover)
			DPSRow:     0, // DPS in front with tanks
			SupportRow: 1, // Support in middle (receive cover)
			RangedRow:  2, // Ranged in back (receive cover)
		}
	case FormationDefensive:
		return FormationTemplate{
			Name:       "Defensive",
			TankRow:    0, // Tanks in front (provide cover)
			DPSRow:     1, // DPS in middle (receive cover)
			SupportRow: 2, // Support in back (receive cover)
			RangedRow:  2, // Ranged in back (receive cover)
		}
	case FormationAggressive:
		return FormationTemplate{
			Name:       "Aggressive",
			TankRow:    1, // Tanks in middle (provide cover to DPS)
			DPSRow:     0, // DPS in front (high damage output)
			SupportRow: 2, // Support in back
			RangedRow:  2, // Ranged in back
		}
	case FormationRanged:
		return FormationTemplate{
			Name:       "Ranged",
			TankRow:    0, // Tanks in front (provide cover)
			DPSRow:     1, // DPS in middle
			SupportRow: 2, // Support in back
			RangedRow:  2, // All ranged in back (receive cover)
		}
	case FormationBalanced:
		return FormationTemplate{
			Name:       "Balanced",
			TankRow:    0, // Tanks in front
			DPSRow:     1, // DPS in middle
			SupportRow: 1, // Support in middle
			RangedRow:  2, // Ranged in back
		}
	default:
		// Return standard as fallback
		return GetFormationTemplate(FormationStandard)
	}
}

// ApplyFormationToSquad adjusts unit positions based on formation template
// It modifies GridRow assignments while preserving GridCol positions
func ApplyFormationToSquad(units []UnitConfig, formation FormationTemplate) []UnitConfig {
	result := make([]UnitConfig, len(units))

	for i, unitConfig := range units {
		// Copy the unit config
		result[i] = unitConfig.Clone()

		// Find the unit template
		template := findUnitTemplateByName(unitConfig.TemplateName)
		if template == nil {
			// Keep original row if template not found
			continue
		}

		// Assign row based on unit characteristics
		// Ranged units (AttackRange > 1) go to ranged row regardless of role
		if template.AttackRange > 1 {
			result[i].GridRow = formation.RangedRow
			continue
		}

		// Otherwise, assign by role
		switch template.Role {
		case squads.RoleTank:
			result[i].GridRow = formation.TankRow
		case squads.RoleDPS:
			result[i].GridRow = formation.DPSRow
		case squads.RoleSupport:
			result[i].GridRow = formation.SupportRow
		default:
			// Keep original row for unknown roles
		}
	}

	return result
}

// findUnitTemplateByName searches the global Units list for a template by name
func findUnitTemplateByName(name string) *squads.UnitTemplate {
	for i := range squads.Units {
		if squads.Units[i].Name == name {
			return &squads.Units[i]
		}
	}
	return nil
}

// FormationValidation contains validation results for a formation
type FormationValidation struct {
	HasDepth          bool    // Multiple rows used
	HasCoverProviders bool    // At least one unit with CoverValue > 0
	HasReceivers      bool    // At least one unit that can receive cover
	CoverPotential    float64 // Sum of all CoverValue
}

// ValidateFormation checks if formation enables cover mechanics
func ValidateFormation(units []UnitConfig) FormationValidation {
	validation := FormationValidation{
		HasDepth:          false,
		HasCoverProviders: false,
		HasReceivers:      false,
		CoverPotential:    0.0,
	}

	// Check row distribution
	rowCounts := make(map[int]int)
	for _, unit := range units {
		rowCounts[unit.GridRow]++
	}
	validation.HasDepth = len(rowCounts) > 1

	// Check for cover providers and receivers
	for i, unit := range units {
		template := findUnitTemplateByName(unit.TemplateName)
		if template == nil {
			continue
		}

		// Check if this unit provides cover
		if template.CoverValue > 0 {
			validation.HasCoverProviders = true
			validation.CoverPotential += template.CoverValue
		}

		// Check if this unit can receive cover from another unit
		// (i.e., is there a cover provider in a lower row in the same column or nearby)
		for j, other := range units {
			if i == j {
				continue // Don't compare with self
			}

			otherTemplate := findUnitTemplateByName(other.TemplateName)
			if otherTemplate == nil || otherTemplate.CoverValue == 0 {
				continue // Other unit doesn't provide cover
			}

			// Check if other unit is in front of this unit (lower row number)
			if other.GridRow < unit.GridRow {
				validation.HasReceivers = true
				break
			}
		}
	}

	return validation
}

// String returns a human-readable summary of the validation
func (v FormationValidation) String() string {
	return fmt.Sprintf("Formation: Depth=%v, Providers=%v, Receivers=%v, Potential=%.2f",
		v.HasDepth, v.HasCoverProviders, v.HasReceivers, v.CoverPotential)
}
