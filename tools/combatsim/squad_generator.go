package combatsim

import (
	"fmt"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"math/rand"
	"time"
)

// SquadCompositionGenerator generates diverse squad compositions for balance testing.
type SquadCompositionGenerator struct {
	templates []squads.UnitTemplate
	rng       *rand.Rand
}

// NewSquadCompositionGenerator creates a new squad generator with available templates.
func NewSquadCompositionGenerator() *SquadCompositionGenerator {
	return &SquadCompositionGenerator{
		templates: squads.Units, // Use global Units slice
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewSquadCompositionGeneratorWithSeed creates a generator with a specific seed for reproducibility.
func NewSquadCompositionGeneratorWithSeed(seed int64) *SquadCompositionGenerator {
	return &SquadCompositionGenerator{
		templates: squads.Units,
		rng:       rand.New(rand.NewSource(seed)),
	}
}

// GenerateRandomSquad creates a squad with random unit selection and role balance.
// Ensures at least 1 tank-role unit and 3-9 total units.
func (g *SquadCompositionGenerator) GenerateRandomSquad(name string, posX, posY int) SquadSetup {
	numUnits := g.rng.Intn(7) + 3 // 3-9 units

	// Pick a random formation
	formation := g.randomFormationType()
	formationTemplate := GetFormationTemplate(formation)

	// Generate units
	units := g.generateUnitsWithRoleBalance(numUnits)

	// Apply formation
	units = ApplyFormationToSquad(units, formationTemplate)

	// Randomly assign leader
	g.assignRandomLeader(units)

	return SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: posX, Y: posY},
	}
}

// GenerateBalancedSquad creates a squad with mixed frontline/backline units.
func (g *SquadCompositionGenerator) GenerateBalancedSquad(name string, posX, posY int) SquadSetup {
	units := make([]UnitConfig, 0)

	// 2 tanks
	units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 0)...)
	units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 1)...)

	// 2-3 DPS
	numDPS := g.rng.Intn(2) + 2
	for i := 0; i < numDPS; i++ {
		units = append(units, g.createRandomUnitByRole(squads.RoleDPS, 0, 0)...)
	}

	// 1-2 Support
	numSupport := g.rng.Intn(2) + 1
	for i := 0; i < numSupport; i++ {
		units = append(units, g.createRandomUnitByRole(squads.RoleSupport, 0, 0)...)
	}

	// Apply balanced formation
	formation := GetFormationTemplate(FormationBalanced)
	units = ApplyFormationToSquad(units, formation)

	// Assign leader
	g.assignRandomLeader(units)

	return SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: posX, Y: posY},
	}
}

// GenerateRangedSquad creates a ranged-focused composition.
func (g *SquadCompositionGenerator) GenerateRangedSquad(name string, posX, posY int) SquadSetup {
	units := make([]UnitConfig, 0)

	// 1-2 tanks for frontline
	numTanks := g.rng.Intn(2) + 1
	for i := 0; i < numTanks; i++ {
		units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 0)...)
	}

	// 4-6 ranged units (high AttackRange)
	numRanged := g.rng.Intn(3) + 4
	for i := 0; i < numRanged; i++ {
		rangedUnit := g.pickRandomRangedTemplate()
		if rangedUnit != nil {
			units = append(units, UnitConfig{
				TemplateName: rangedUnit.Name,
				GridRow:      0,
				GridCol:      0,
				IsLeader:     false,
			})
		}
	}

	// Apply ranged formation
	formation := GetFormationTemplate(FormationRanged)
	units = ApplyFormationToSquad(units, formation)

	// Assign leader
	g.assignRandomLeader(units)

	return SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: posX, Y: posY},
	}
}

// GenerateMagicSquad creates a magic-focused composition.
func (g *SquadCompositionGenerator) GenerateMagicSquad(name string, posX, posY int) SquadSetup {
	units := make([]UnitConfig, 0)

	// 1 tank for protection
	units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 0)...)

	// 4-6 magic users
	numMagic := g.rng.Intn(3) + 4
	for i := 0; i < numMagic; i++ {
		magicUnit := g.pickRandomMagicTemplate()
		if magicUnit != nil {
			units = append(units, UnitConfig{
				TemplateName: magicUnit.Name,
				GridRow:      0,
				GridCol:      0,
				IsLeader:     false,
			})
		}
	}

	// 1-2 support
	numSupport := g.rng.Intn(2) + 1
	for i := 0; i < numSupport; i++ {
		units = append(units, g.createRandomUnitByRole(squads.RoleSupport, 0, 0)...)
	}

	// Apply defensive formation (protect casters)
	formation := GetFormationTemplate(FormationDefensive)
	units = ApplyFormationToSquad(units, formation)

	// Assign leader
	g.assignRandomLeader(units)

	return SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: posX, Y: posY},
	}
}

// GenerateRoleSpecificSquad creates a squad focused on a specific role.
func (g *SquadCompositionGenerator) GenerateRoleSpecificSquad(name string, role squads.UnitRole, posX, posY int) SquadSetup {
	units := make([]UnitConfig, 0)

	// Ensure at least 1 tank if not generating tanks
	if role != squads.RoleTank {
		units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 0)...)
	}

	// 5-8 units of the specified role
	numRoleUnits := g.rng.Intn(4) + 5
	for i := 0; i < numRoleUnits; i++ {
		units = append(units, g.createRandomUnitByRole(role, 0, 0)...)
	}

	// Apply appropriate formation
	var formation FormationType
	switch role {
	case squads.RoleTank:
		formation = FormationAggressive
	case squads.RoleDPS:
		formation = FormationAggressive
	case squads.RoleSupport:
		formation = FormationDefensive
	default:
		formation = FormationBalanced
	}

	formationTemplate := GetFormationTemplate(formation)
	units = ApplyFormationToSquad(units, formationTemplate)

	// Assign leader
	g.assignRandomLeader(units)

	return SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: posX, Y: posY},
	}
}

// SelectRandomComposition selects a random composition strategy with weighted probabilities.
// Returns one of: random, balanced, ranged, magic, or role-specific.
func (g *SquadCompositionGenerator) SelectRandomComposition(name string, posX, posY int) SquadSetup {
	// Weighted selection: balanced and random are more common
	roll := g.rng.Intn(100)

	switch {
	case roll < 30:
		return g.GenerateRandomSquad(name, posX, posY)
	case roll < 60:
		return g.GenerateBalancedSquad(name, posX, posY)
	case roll < 75:
		return g.GenerateRangedSquad(name, posX, posY)
	case roll < 85:
		return g.GenerateMagicSquad(name, posX, posY)
	default:
		// Role-specific: pick a random role
		roles := []squads.UnitRole{squads.RoleTank, squads.RoleDPS, squads.RoleSupport}
		role := roles[g.rng.Intn(len(roles))]
		return g.GenerateRoleSpecificSquad(name, role, posX, posY)
	}
}

// --- Helper methods ---

// generateUnitsWithRoleBalance creates units ensuring at least 1 tank.
func (g *SquadCompositionGenerator) generateUnitsWithRoleBalance(numUnits int) []UnitConfig {
	units := make([]UnitConfig, 0, numUnits)

	// First unit is always a tank
	units = append(units, g.createRandomUnitByRole(squads.RoleTank, 0, 0)...)

	// Remaining units are random
	for i := 1; i < numUnits; i++ {
		template := g.pickRandomTemplate()
		if template != nil {
			units = append(units, UnitConfig{
				TemplateName: template.Name,
				GridRow:      0,
				GridCol:      0,
				IsLeader:     false,
			})
		}
	}

	return units
}

// createRandomUnitByRole creates a unit config for a specific role.
func (g *SquadCompositionGenerator) createRandomUnitByRole(role squads.UnitRole, row, col int) []UnitConfig {
	template := g.pickRandomTemplateByRole(role)
	if template == nil {
		return []UnitConfig{}
	}

	return []UnitConfig{
		{
			TemplateName: template.Name,
			GridRow:      row,
			GridCol:      col,
			IsLeader:     false,
		},
	}
}

// pickRandomTemplate picks a random unit template.
func (g *SquadCompositionGenerator) pickRandomTemplate() *squads.UnitTemplate {
	if len(g.templates) == 0 {
		return nil
	}
	idx := g.rng.Intn(len(g.templates))
	return &g.templates[idx]
}

// pickRandomTemplateByRole picks a random template with the specified role.
func (g *SquadCompositionGenerator) pickRandomTemplateByRole(role squads.UnitRole) *squads.UnitTemplate {
	candidates := make([]*squads.UnitTemplate, 0)
	for i := range g.templates {
		if g.templates[i].Role == role {
			candidates = append(candidates, &g.templates[i])
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	idx := g.rng.Intn(len(candidates))
	return candidates[idx]
}

// pickRandomRangedTemplate picks a template with high attack range (>= 3).
func (g *SquadCompositionGenerator) pickRandomRangedTemplate() *squads.UnitTemplate {
	candidates := make([]*squads.UnitTemplate, 0)
	for i := range g.templates {
		if g.templates[i].AttackRange >= 3 {
			candidates = append(candidates, &g.templates[i])
		}
	}

	if len(candidates) == 0 {
		// Fallback to any template
		return g.pickRandomTemplate()
	}

	idx := g.rng.Intn(len(candidates))
	return candidates[idx]
}

// pickRandomMagicTemplate picks a template with Magic attack type.
func (g *SquadCompositionGenerator) pickRandomMagicTemplate() *squads.UnitTemplate {
	candidates := make([]*squads.UnitTemplate, 0)
	for i := range g.templates {
		if g.templates[i].AttackType == squads.AttackTypeMagic {
			candidates = append(candidates, &g.templates[i])
		}
	}

	if len(candidates) == 0 {
		// Fallback to high magic attribute
		for i := range g.templates {
			if g.templates[i].Attributes.Magic >= 50 {
				candidates = append(candidates, &g.templates[i])
			}
		}
	}

	if len(candidates) == 0 {
		// Final fallback to any template
		return g.pickRandomTemplate()
	}

	idx := g.rng.Intn(len(candidates))
	return candidates[idx]
}

// assignRandomLeader randomly assigns one unit as the squad leader.
func (g *SquadCompositionGenerator) assignRandomLeader(units []UnitConfig) {
	if len(units) == 0 {
		return
	}

	// Pick a random unit to be the leader
	leaderIdx := g.rng.Intn(len(units))
	units[leaderIdx].IsLeader = true
}

// randomFormationType returns a random formation type.
func (g *SquadCompositionGenerator) randomFormationType() FormationType {
	formations := []FormationType{
		FormationStandard,
		FormationDefensive,
		FormationAggressive,
		FormationRanged,
		FormationBalanced,
	}
	return formations[g.rng.Intn(len(formations))]
}

// ValidateSquadSetup checks if a squad setup is valid for combat.
func ValidateSquadSetup(setup SquadSetup) error {
	if len(setup.Units) == 0 {
		return fmt.Errorf("squad %s has no units", setup.Name)
	}

	// Check for at least one leader
	hasLeader := false
	for _, unit := range setup.Units {
		if unit.IsLeader {
			hasLeader = true
			break
		}
	}

	if !hasLeader {
		return fmt.Errorf("squad %s has no leader", setup.Name)
	}

	return nil
}
