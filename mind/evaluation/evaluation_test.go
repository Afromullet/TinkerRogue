package evaluation

import (
	"game_main/tactical/unitdefs"
	"testing"
)

func TestGetRoleMultiplierFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		role     unitdefs.UnitRole
		expected float64
	}{
		{"Tank role", unitdefs.RoleTank, 1.2},
		{"DPS role", unitdefs.RoleDPS, 1.5},
		{"Support role", unitdefs.RoleSupport, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRoleMultiplierFromConfig(tt.role)
			if result != tt.expected {
				t.Errorf("GetRoleMultiplierFromConfig(%v) = %v, want %v", tt.role, result, tt.expected)
			}
		})
	}
}

func TestGetRoleMultiplierFromConfig_UnknownRole(t *testing.T) {
	// Unknown role should return baseline 1.0
	unknownRole := unitdefs.UnitRole(999)
	result := GetRoleMultiplierFromConfig(unknownRole)
	if result != 1.0 {
		t.Errorf("GetRoleMultiplierFromConfig(unknown) = %v, want 1.0", result)
	}
}
