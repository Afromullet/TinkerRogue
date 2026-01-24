package combatsim

import (
	"game_main/tactical/squads"
	"game_main/templates"
	"testing"
)

// TestGenerateRandomSquad_ValidComposition verifies random squads are valid
func TestGenerateRandomSquad_ValidComposition(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()

	// Generate multiple squads to test randomness
	for i := 0; i < 10; i++ {
		setup := generator.GenerateRandomSquad("TestSquad", 0, 0)

		// Verify squad has units
		if len(setup.Units) < 3 {
			t.Errorf("Squad %d has too few units: %d", i, len(setup.Units))
		}

		if len(setup.Units) > 9 {
			t.Errorf("Squad %d has too many units: %d", i, len(setup.Units))
		}

		// Verify squad name
		if setup.Name != "TestSquad" {
			t.Errorf("Expected squad name 'TestSquad', got '%s'", setup.Name)
		}
	}
}

// TestGenerateRandomSquad_HasLeader verifies leader assignment
func TestGenerateRandomSquad_HasLeader(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()

	// Generate squad
	setup := generator.GenerateRandomSquad("TestSquad", 0, 0)

	// Count leaders
	leaderCount := 0
	for _, unit := range setup.Units {
		if unit.IsLeader {
			leaderCount++
		}
	}

	if leaderCount == 0 {
		t.Error("Squad has no leader")
	}

	if leaderCount > 1 {
		t.Errorf("Squad has multiple leaders: %d", leaderCount)
	}
}

// TestGenerateRandomSquad_HasTank verifies at least one tank is present
func TestGenerateRandomSquad_HasTank(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()

	// Generate multiple squads
	for i := 0; i < 10; i++ {
		setup := generator.GenerateRandomSquad("TestSquad", 0, 0)

		// Check if at least one unit is a tank
		hasTank := false
		for _, unit := range setup.Units {
			// Find the template to check role
			for _, template := range squads.Units {
				if template.Name == unit.TemplateName && template.Role == squads.RoleTank {
					hasTank = true
					break
				}
			}
			if hasTank {
				break
			}
		}

		if !hasTank {
			t.Errorf("Squad %d has no tank units", i)
		}
	}
}

// TestGenerationStrategies tests each generation strategy produces valid squads
func TestGenerationStrategies(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()

	strategies := []struct {
		name string
		fn   func(string, int, int) SquadSetup
	}{
		{"Random", generator.GenerateRandomSquad},
		{"Balanced", generator.GenerateBalancedSquad},
		{"Ranged", generator.GenerateRangedSquad},
		{"Magic", generator.GenerateMagicSquad},
	}

	for _, strategy := range strategies {
		t.Run(strategy.name, func(t *testing.T) {
			setup := strategy.fn("TestSquad", 0, 0)

			// Verify basic validity
			if len(setup.Units) == 0 {
				t.Errorf("%s strategy produced empty squad", strategy.name)
			}

			// Verify leader exists
			hasLeader := false
			for _, unit := range setup.Units {
				if unit.IsLeader {
					hasLeader = true
					break
				}
			}

			if !hasLeader {
				t.Errorf("%s strategy produced squad without leader", strategy.name)
			}

			// Verify all templates exist
			for _, unit := range setup.Units {
				found := false
				for _, template := range squads.Units {
					if template.Name == unit.TemplateName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s strategy used unknown template: %s", strategy.name, unit.TemplateName)
				}
			}
		})
	}
}

// TestGenerateBalancedSquad_HasMixedRoles verifies balanced squads have role variety
func TestGenerateBalancedSquad_HasMixedRoles(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()
	setup := generator.GenerateBalancedSquad("TestSquad", 0, 0)

	// Count roles
	roleCounts := make(map[squads.UnitRole]int)
	for _, unit := range setup.Units {
		for _, template := range squads.Units {
			if template.Name == unit.TemplateName {
				roleCounts[template.Role]++
				break
			}
		}
	}

	// Balanced squad should have at least 2 different roles
	if len(roleCounts) < 2 {
		t.Errorf("Balanced squad has only %d role types, expected at least 2", len(roleCounts))
	}

	// Should have tanks
	if roleCounts[squads.RoleTank] == 0 {
		t.Error("Balanced squad has no tanks")
	}
}

// TestGenerateRangedSquad_HasRangedUnits verifies ranged focus
func TestGenerateRangedSquad_HasRangedUnits(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()
	setup := generator.GenerateRangedSquad("TestSquad", 0, 0)

	// Count ranged units (AttackRange >= 3)
	rangedCount := 0
	for _, unit := range setup.Units {
		for _, template := range squads.Units {
			if template.Name == unit.TemplateName && template.AttackRange >= 3 {
				rangedCount++
				break
			}
		}
	}

	// Should have multiple ranged units
	if rangedCount < 2 {
		t.Errorf("Ranged squad has only %d ranged units, expected at least 2", rangedCount)
	}
}

// TestGenerateMagicSquad_HasMagicUsers verifies magic focus
func TestGenerateMagicSquad_HasMagicUsers(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()
	setup := generator.GenerateMagicSquad("TestSquad", 0, 0)

	// Count magic users (AttackType == Magic or high Magic attribute)
	magicCount := 0
	for _, unit := range setup.Units {
		for _, template := range squads.Units {
			if template.Name == unit.TemplateName {
				if template.AttackType == squads.AttackTypeMagic || template.Attributes.Magic >= 50 {
					magicCount++
				}
				break
			}
		}
	}

	// Should have multiple magic users
	if magicCount < 2 {
		t.Errorf("Magic squad has only %d magic users, expected at least 2", magicCount)
	}
}

// TestSelectRandomComposition_Variety verifies varied composition selection
func TestSelectRandomComposition_Variety(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	generator := NewSquadCompositionGenerator()

	// Generate many squads and track squad sizes
	sizeVariety := make(map[int]bool)
	for i := 0; i < 20; i++ {
		setup := generator.SelectRandomComposition("TestSquad", 0, 0)
		sizeVariety[len(setup.Units)] = true
	}

	// Should see some variety in squad sizes
	if len(sizeVariety) < 2 {
		t.Error("SelectRandomComposition shows no variety in squad sizes")
	}
}

// TestValidateSquadSetup_EmptySquad verifies validation catches empty squads
func TestValidateSquadSetup_EmptySquad(t *testing.T) {
	setup := SquadSetup{
		Name:  "EmptySquad",
		Units: []UnitConfig{},
	}

	err := ValidateSquadSetup(setup)
	if err == nil {
		t.Error("Expected error for empty squad, got nil")
	}
}

// TestValidateSquadSetup_NoLeader verifies validation catches missing leader
func TestValidateSquadSetup_NoLeader(t *testing.T) {
	setup := SquadSetup{
		Name: "NoLeaderSquad",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: false},
		},
	}

	err := ValidateSquadSetup(setup)
	if err == nil {
		t.Error("Expected error for squad without leader, got nil")
	}
}

// TestValidateSquadSetup_ValidSquad verifies validation accepts valid squads
func TestValidateSquadSetup_ValidSquad(t *testing.T) {
	setup := SquadSetup{
		Name: "ValidSquad",
		Units: []UnitConfig{
			{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
			{TemplateName: "Fighter", GridRow: 0, GridCol: 1, IsLeader: false},
		},
	}

	err := ValidateSquadSetup(setup)
	if err != nil {
		t.Errorf("Expected valid squad to pass validation, got error: %v", err)
	}
}

// TestNewSquadCompositionGeneratorWithSeed verifies reproducibility
func TestNewSquadCompositionGeneratorWithSeed(t *testing.T) {
	// Initialize templates
	templates.ReadMonsterData()
	if err := squads.InitUnitTemplatesFromJSON(); err != nil {
		t.Fatalf("Failed to init templates: %v", err)
	}

	// Create two generators with same seed
	seed := int64(12345)
	gen1 := NewSquadCompositionGeneratorWithSeed(seed)
	gen2 := NewSquadCompositionGeneratorWithSeed(seed)

	// Generate squads
	setup1 := gen1.GenerateRandomSquad("Squad1", 0, 0)
	setup2 := gen2.GenerateRandomSquad("Squad1", 0, 0)

	// Should have same size
	if len(setup1.Units) != len(setup2.Units) {
		t.Errorf("Squads with same seed have different sizes: %d vs %d", len(setup1.Units), len(setup2.Units))
	}

	// Should have same unit templates (in same order)
	for i := 0; i < len(setup1.Units) && i < len(setup2.Units); i++ {
		if setup1.Units[i].TemplateName != setup2.Units[i].TemplateName {
			t.Errorf("Unit %d differs: %s vs %s", i, setup1.Units[i].TemplateName, setup2.Units[i].TemplateName)
		}
	}
}
