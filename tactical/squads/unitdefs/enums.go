// Package unitdefs defines unit types, roles, attack types, and templates.
// It is a leaf package that other tactical packages import for unit definitions
// without needing to depend on the full squads package.
package unitdefs

import "fmt"

// UnitRole defines a unit's combat role
type UnitRole int

const (
	RoleTank    UnitRole = iota // Takes hits first, high defense
	RoleDPS                     // High damage output
	RoleSupport                 // Buffs, heals, utility
	RoleError                   // This is an error
)

func (r UnitRole) String() string {
	switch r {
	case RoleTank:
		return "Tank"
	case RoleDPS:
		return "DPS"
	case RoleSupport:
		return "Support"
	default:
		return "Unknown"
	}
}

// AttackType defines how a unit selects targets
type AttackType int

const (
	AttackTypeMeleeRow    AttackType = iota // Targets front row (3 targets max)
	AttackTypeMeleeColumn                   // Targets column (1 target, spear-type)
	AttackTypeRanged                        // Targets same row as attacker
	AttackTypeMagic                         // Cell-based patterns
	AttackTypeHeal                          // Heals friendly units using targetCells
)

func (a AttackType) String() string {
	switch a {
	case AttackTypeMeleeRow:
		return "MeleeRow"
	case AttackTypeMeleeColumn:
		return "MeleeColumn"
	case AttackTypeRanged:
		return "Ranged"
	case AttackTypeMagic:
		return "Magic"
	case AttackTypeHeal:
		return "Heal"
	default:
		return "Unknown"
	}
}

// GetRole converts a role string from a JSON file to a UnitRole enum value.
func GetRole(roleString string) (UnitRole, error) {
	switch roleString {
	case "Tank":
		return RoleTank, nil
	case "DPS":
		return RoleDPS, nil
	case "Support":
		return RoleSupport, nil
	default:
		return 0, fmt.Errorf("invalid role: %q, expected Tank, DPS, or Support", roleString)
	}
}

// GetAttackType converts an attack type string from JSON to an AttackType enum value.
// If attackTypeString is empty, falls back to attackRange for backward compatibility.
func GetAttackType(attackTypeString string, attackRange int) (AttackType, error) {
	if attackTypeString != "" {
		switch attackTypeString {
		case "MeleeRow":
			return AttackTypeMeleeRow, nil
		case "MeleeColumn":
			return AttackTypeMeleeColumn, nil
		case "Ranged":
			return AttackTypeRanged, nil
		case "Magic":
			return AttackTypeMagic, nil
		case "Heal":
			return AttackTypeHeal, nil
		default:
			return 0, fmt.Errorf("invalid attackType: %q, expected MeleeRow, MeleeColumn, Ranged, Magic, or Heal", attackTypeString)
		}
	}

	// Fallback to attackRange for backward compatibility
	switch attackRange {
	case 0:
		return AttackTypeMeleeRow, nil
	case 1:
		return AttackTypeMeleeRow, nil
	case 3:
		return AttackTypeRanged, nil
	case 4:
		return AttackTypeMagic, nil
	default:
		return 0, fmt.Errorf("cannot determine attack type: attackType is empty and attackRange %d is invalid", attackRange)
	}
}
