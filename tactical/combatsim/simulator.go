package combatsim

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Simulator runs combat simulations
type Simulator struct {
	config SimulationConfig
}

// SimulationConfig contains simulation parameters
type SimulationConfig struct {
	Iterations int
	Verbose    bool
}

// NewSimulator creates a new combat simulator
func NewSimulator(config SimulationConfig) *Simulator {
	return &Simulator{
		config: config,
	}
}

// Run executes N combat simulations for a scenario
func (s *Simulator) Run(scenario CombatScenario) (*SimulationResult, error) {
	result := NewSimulationResult(scenario, s.config.Iterations)

	for i := 0; i < s.config.Iterations; i++ {
		metrics, err := s.runSingleCombat(scenario)
		if err != nil {
			return nil, fmt.Errorf("simulation %d failed: %w", i+1, err)
		}

		result.AddMetrics(metrics)

		if s.config.Verbose {
			result.CombatLogs = append(result.CombatLogs, metrics.Log)
		}
	}

	result.Finalize()
	return result, nil
}

// runSingleCombat executes a single isolated combat simulation
func (s *Simulator) runSingleCombat(scenario CombatScenario) (*CombatMetrics, error) {
	// Create isolated EntityManager for this run
	manager := common.NewEntityManager()

	// Initialize ECS components
	initializeComponents(manager)

	// Build squads
	attackerID, err := buildSquad(manager, scenario.AttackerSetup)
	if err != nil {
		return nil, fmt.Errorf("failed to build attacker squad: %w", err)
	}

	defenderID, err := buildSquad(manager, scenario.DefenderSetup)
	if err != nil {
		return nil, fmt.Errorf("failed to build defender squad: %w", err)
	}

	// Squad distance is already set via WorldPosition in buildSquad

	// Execute combat
	combatResult := squads.ExecuteSquadAttack(attackerID, defenderID, manager)

	// Extract metrics
	metrics := ExtractMetrics(
		combatResult,
		scenario.AttackerSetup.Name,
		scenario.DefenderSetup.Name,
		attackerID,
		defenderID,
	)

	// Cleanup entities (important for isolation)
	cleanupEntities(manager)

	return metrics, nil
}

// buildSquad creates a squad from a SquadSetup
func buildSquad(manager *common.EntityManager, setup SquadSetup) (ecs.EntityID, error) {
	// Create squad entity
	squadEntity := manager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadData := &squads.SquadData{
		SquadID:       squadID,
		Name:          setup.Name,
		Morale:        100,
		MaxUnits:      9,
		TurnCount:     0,
		Formation:     squads.FormationBalanced,
		UsedCapacity:  0,
		TotalCapacity: 6,
	}
	squadEntity.AddComponent(squads.SquadComponent, squadData)

	// Add position component for range calculations
	squadEntity.AddComponent(common.PositionComponent, &setup.WorldPosition)

	// Add units to squad
	for _, unitConfig := range setup.Units {
		// Find unit template by name
		template := findUnitTemplate(unitConfig.TemplateName)
		if template == nil {
			return 0, fmt.Errorf("unit template not found: %s", unitConfig.TemplateName)
		}

		// Create unit entity
		unitEntity := manager.World.NewEntity()

		// Add attributes
		attrs := template.Attributes
		unitEntity.AddComponent(common.AttributeComponent, &attrs)

		// Add name component
		unitEntity.AddComponent(common.NameComponent, &common.Name{NameStr: template.Name})

		// Add squad membership
		squadMemberData := &squads.SquadMemberData{
			SquadID: squadID,
		}
		unitEntity.AddComponent(squads.SquadMemberComponent, squadMemberData)

		// Add grid position
		gridPosData := &squads.GridPositionData{
			AnchorRow: unitConfig.GridRow,
			AnchorCol: unitConfig.GridCol,
			Width:     template.GridWidth,
			Height:    template.GridHeight,
		}
		unitEntity.AddComponent(squads.GridPositionComponent, gridPosData)

		// Add target row (targeting)
		targetRowData := &squads.TargetRowData{
			TargetCells: template.TargetCells,
		}
		unitEntity.AddComponent(squads.TargetRowComponent, targetRowData)

		// Add attack range if present
		if template.AttackRange > 0 {
			rangeData := &squads.AttackRangeData{
				Range: template.AttackRange,
			}
			unitEntity.AddComponent(squads.AttackRangeComponent, rangeData)
		}

		// Add cover if present
		if template.CoverValue > 0 {
			coverData := &squads.CoverData{
				CoverValue:     template.CoverValue,
				CoverRange:     template.CoverRange,
				RequiresActive: template.RequiresActive,
			}
			unitEntity.AddComponent(squads.CoverComponent, coverData)
		}

		// Add leader component if specified
		if unitConfig.IsLeader {
			leaderData := &squads.LeaderData{}
			unitEntity.AddComponent(squads.LeaderComponent, leaderData)
		}
	}

	return squadID, nil
}

// cleanupEntities disposes all entities in the manager
func cleanupEntities(manager *common.EntityManager) {
	// Get all entities
	entities := manager.World.Query(ecs.BuildTag())

	for _, result := range entities {
		manager.World.DisposeEntity(result.Entity)
	}
}

// findUnitTemplate finds a unit template by name
func findUnitTemplate(name string) *squads.UnitTemplate {
	for i := range squads.Units {
		if squads.Units[i].Name == name {
			return &squads.Units[i]
		}
	}
	return nil
}

// initializeComponents registers all ECS components needed for combat simulation
func initializeComponents(manager *common.EntityManager) {
	// Register core components
	common.PositionComponent = manager.World.NewComponent()
	common.NameComponent = manager.World.NewComponent()
	common.AttributeComponent = manager.World.NewComponent()

	// Initialize squad components and tags
	squads.InitSquadComponents(manager)
	squads.InitSquadTags(manager)
}
