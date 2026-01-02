package combatsim

import (
	"fmt"
	"game_main/world/coords"
)

// CombatScenario defines a test scenario for combat simulation
type CombatScenario struct {
	Name          string
	AttackerSetup SquadSetup
	DefenderSetup SquadSetup
	SquadDistance int // Distance between squads (affects range)
}

// SquadSetup configures a squad for simulation
type SquadSetup struct {
	Name          string
	Units         []UnitConfig
	WorldPosition coords.LogicalPosition
}

// UnitConfig defines a unit in a squad for simulation
type UnitConfig struct {
	TemplateName       string // Reference to Units[] global
	GridRow            int
	GridCol            int
	IsLeader           bool
	AttributeOverrides map[string]int // Override specific attributes for sweeps
}

// ScenarioBuilder provides fluent API for creating combat scenarios
type ScenarioBuilder struct {
	scenario CombatScenario
}

// NewScenarioBuilder creates a new scenario builder
func NewScenarioBuilder(name string) *ScenarioBuilder {
	return &ScenarioBuilder{
		scenario: CombatScenario{
			Name:          name,
			SquadDistance: 1, // Default melee range
		},
	}
}

// WithAttacker configures the attacking squad
func (b *ScenarioBuilder) WithAttacker(name string, units []UnitConfig) *ScenarioBuilder {
	b.scenario.AttackerSetup = SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: 0, Y: 0}, // Default position
	}
	return b
}

// WithDefender configures the defending squad
func (b *ScenarioBuilder) WithDefender(name string, units []UnitConfig) *ScenarioBuilder {
	b.scenario.DefenderSetup = SquadSetup{
		Name:          name,
		Units:         units,
		WorldPosition: coords.LogicalPosition{X: 0, Y: 1}, // Default position (1 tile away)
	}
	return b
}

// WithDistance sets the squad distance
func (b *ScenarioBuilder) WithDistance(distance int) *ScenarioBuilder {
	b.scenario.SquadDistance = distance
	return b
}

// WithFormation applies the formation template to both attacker and defender squads
func (b *ScenarioBuilder) WithFormation(formType FormationType) *ScenarioBuilder {
	template := GetFormationTemplate(formType)
	b.scenario.AttackerSetup.Units = ApplyFormationToSquad(b.scenario.AttackerSetup.Units, template)
	b.scenario.DefenderSetup.Units = ApplyFormationToSquad(b.scenario.DefenderSetup.Units, template)
	return b
}

// WithAttackerFormation applies the formation template to the attacker squad only
func (b *ScenarioBuilder) WithAttackerFormation(formType FormationType) *ScenarioBuilder {
	template := GetFormationTemplate(formType)
	b.scenario.AttackerSetup.Units = ApplyFormationToSquad(b.scenario.AttackerSetup.Units, template)
	return b
}

// WithDefenderFormation applies the formation template to the defender squad only
func (b *ScenarioBuilder) WithDefenderFormation(formType FormationType) *ScenarioBuilder {
	template := GetFormationTemplate(formType)
	b.scenario.DefenderSetup.Units = ApplyFormationToSquad(b.scenario.DefenderSetup.Units, template)
	return b
}

// Build creates the final scenario
func (b *ScenarioBuilder) Build() CombatScenario {
	if b.scenario.Name == "" {
		b.scenario.Name = fmt.Sprintf("%s vs %s", b.scenario.AttackerSetup.Name, b.scenario.DefenderSetup.Name)
	}
	return b.scenario
}

// GetUnitCount returns the number of units in this setup
func (s *SquadSetup) GetUnitCount() int {
	return len(s.Units)
}

// Clone creates a deep copy of the scenario for mutation
func (s CombatScenario) Clone() CombatScenario {
	clone := CombatScenario{
		Name:          s.Name,
		SquadDistance: s.SquadDistance,
		AttackerSetup: s.AttackerSetup.Clone(),
		DefenderSetup: s.DefenderSetup.Clone(),
	}
	return clone
}

// Clone creates a deep copy of the squad setup
func (s SquadSetup) Clone() SquadSetup {
	clone := SquadSetup{
		Name:          s.Name,
		WorldPosition: s.WorldPosition,
		Units:         make([]UnitConfig, len(s.Units)),
	}

	for i, unit := range s.Units {
		clone.Units[i] = unit.Clone()
	}

	return clone
}

// Clone creates a deep copy of the unit config
func (u UnitConfig) Clone() UnitConfig {
	clone := UnitConfig{
		TemplateName:       u.TemplateName,
		GridRow:            u.GridRow,
		GridCol:            u.GridCol,
		IsLeader:           u.IsLeader,
		AttributeOverrides: make(map[string]int),
	}

	for k, v := range u.AttributeOverrides {
		clone.AttributeOverrides[k] = v
	}

	return clone
}
