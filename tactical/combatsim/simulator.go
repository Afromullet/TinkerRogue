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

	// Analysis options
	TrackTimeline bool   // Enable round-by-round tracking
	TrackUnits    bool   // Enable per-unit metrics
	AnalysisMode  string // "quick", "standard", "comprehensive"
}

// AnalysisMode constants
const (
	AnalysisModeQuick         = "quick"         // Win rate + basic stats only
	AnalysisModeStandard      = "standard"      // + per-unit performance + mechanics
	AnalysisModeComprehensive = "comprehensive" // + timeline + confidence + recommendations
)

// NewSimulator creates a new combat simulator
func NewSimulator(config SimulationConfig) *Simulator {
	return &Simulator{
		config: config,
	}
}

// GetIterations returns the configured iteration count
func (s *Simulator) GetIterations() int {
	return s.config.Iterations
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

// RunWithAnalysis executes simulations with comprehensive analysis
func (s *Simulator) RunWithAnalysis(scenario CombatScenario) (*SimulationResult, []TimelineData, [][]UnitPerformanceData, error) {
	result := NewSimulationResult(scenario, s.config.Iterations)
	timelines := make([]TimelineData, 0, s.config.Iterations)
	unitPerformances := make([][]UnitPerformanceData, 0, s.config.Iterations)

	for i := 0; i < s.config.Iterations; i++ {
		metrics, timeline, unitPerf, err := s.runSingleCombatWithTimeline(scenario)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("simulation %d failed: %w", i+1, err)
		}

		result.AddMetrics(metrics)

		if s.config.Verbose {
			result.CombatLogs = append(result.CombatLogs, metrics.Log)
		}

		if timeline != nil {
			timelines = append(timelines, *timeline)
		}

		if unitPerf != nil {
			unitPerformances = append(unitPerformances, unitPerf)
		}
	}

	result.Finalize()
	return result, timelines, unitPerformances, nil
}

// runSingleCombatWithTimeline executes combat with round-by-round tracking
func (s *Simulator) runSingleCombatWithTimeline(scenario CombatScenario) (*CombatMetrics, *TimelineData, []UnitPerformanceData, error) {
	// Create isolated EntityManager for this run
	manager := common.NewEntityManager()

	// Initialize ECS components
	initializeComponents(manager)

	// Build squads
	attackerID, err := buildSquad(manager, scenario.AttackerSetup)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build attacker squad: %w", err)
	}

	defenderID, err := buildSquad(manager, scenario.DefenderSetup)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build defender squad: %w", err)
	}

	// Initialize timeline tracking
	timeline := NewTimelineData()
	allCombatLogs := make([]*squads.CombatLog, 0)
	totalDamageDealt := 0
	totalDamageTaken := 0
	allUnitsKilled := make([]ecs.EntityID, 0)

	round := 0
	maxRounds := 20 // Safety limit

	// Initial snapshot
	unitPerf := s.captureInitialUnitState(attackerID, defenderID, manager)

	// Round-by-round combat execution
	for round < maxRounds {
		// Check if combat is complete
		complete, winner := IsCombatComplete(attackerID, defenderID, manager)
		if complete {
			timeline.Winner = winner
			break
		}

		round++

		// Capture pre-round state
		snapshot := CaptureRoundSnapshot(round, attackerID, defenderID, manager)

		// Execute one attack exchange (attacker attacks defender)
		attackerResult := squads.ExecuteSquadAttack(attackerID, defenderID, manager)
		UpdateSnapshotFromCombat(&snapshot, attackerResult)
		totalDamageDealt += attackerResult.TotalDamage
		allUnitsKilled = append(allUnitsKilled, attackerResult.UnitsKilled...)
		if attackerResult.CombatLog != nil {
			allCombatLogs = append(allCombatLogs, attackerResult.CombatLog)
		}

		// Defender counter-attacks (if still alive)
		defenderComplete, _ := IsCombatComplete(attackerID, defenderID, manager)
		if !defenderComplete {
			defenderResult := squads.ExecuteSquadAttack(defenderID, attackerID, manager)
			snapshot.DamageTakenThisRound = defenderResult.TotalDamage
			totalDamageTaken += defenderResult.TotalDamage
			// Units killed by defender counted as attacker losses
			if defenderResult.CombatLog != nil {
				allCombatLogs = append(allCombatLogs, defenderResult.CombatLog)
				for _, event := range defenderResult.CombatLog.AttackEvents {
					if event.HitResult.Type == squads.HitTypeCritical {
						snapshot.CritsThisRound++
					} else if event.HitResult.Type == squads.HitTypeDodge {
						snapshot.DodgesThisRound++
					}
				}
			}
		}

		// Update momentum after both attacks
		snapshot.Momentum = CalculateMomentum(attackerID, defenderID, manager)

		// Detect first blood
		if timeline.FirstBloodRound == 0 && snapshot.UnitsKilledThisRound > 0 {
			timeline.FirstBloodRound = round
		}

		timeline.Rounds = append(timeline.Rounds, snapshot)
	}

	timeline.CombatDuration = round
	timeline.TurningPoint = DetectTurningPoint(*timeline)

	// Update unit performance with final state
	s.updateUnitPerformanceFromCombat(unitPerf, allCombatLogs, round, manager)

	// Create combined metrics
	metrics := s.createCombinedMetrics(
		scenario,
		allCombatLogs,
		totalDamageDealt,
		totalDamageTaken,
		allUnitsKilled,
		round,
		timeline.Winner,
	)

	// Cleanup entities
	cleanupEntities(manager)

	return metrics, timeline, unitPerf, nil
}

// captureInitialUnitState captures initial unit state for performance tracking
func (s *Simulator) captureInitialUnitState(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) []UnitPerformanceData {
	perfs := make([]UnitPerformanceData, 0)

	for _, result := range manager.World.Query(squads.SquadMemberTag) {
		unitEntity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](unitEntity, squads.SquadMemberComponent)
		if memberData == nil {
			continue
		}

		// Only track units from our squads
		if memberData.SquadID != attackerID && memberData.SquadID != defenderID {
			continue
		}

		attrs := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
		nameComp := common.GetComponentType[*common.Name](unitEntity, common.NameComponent)
		gridPos := common.GetComponentType[*squads.GridPositionData](unitEntity, squads.GridPositionComponent)
		roleData := common.GetComponentType[*squads.UnitRoleData](unitEntity, squads.UnitRoleComponent)

		perf := UnitPerformanceData{
			UnitID:      unitEntity.GetID(),
			TurnOfDeath: -1, // Not dead yet
		}

		if attrs != nil {
			perf.StartingHP = attrs.CurrentHealth
			perf.EndingHP = attrs.CurrentHealth
		}

		if nameComp != nil {
			perf.TemplateName = nameComp.NameStr
		}

		if gridPos != nil {
			perf.GridRow = gridPos.AnchorRow
			perf.GridCol = gridPos.AnchorCol
		}

		if roleData != nil {
			perf.Role = roleData.Role.String()
		}

		perfs = append(perfs, perf)
	}

	return perfs
}

// updateUnitPerformanceFromCombat updates unit performance data from combat logs
func (s *Simulator) updateUnitPerformanceFromCombat(perfs []UnitPerformanceData, logs []*squads.CombatLog, finalRound int, manager *common.EntityManager) {
	// Create map for quick lookup
	perfMap := make(map[ecs.EntityID]*UnitPerformanceData)
	for i := range perfs {
		perfMap[perfs[i].UnitID] = &perfs[i]
	}

	// Process all combat logs
	for _, log := range logs {
		if log == nil {
			continue
		}

		for _, event := range log.AttackEvents {
			// Update attacker stats
			if attPerf, ok := perfMap[event.AttackerID]; ok {
				attPerf.AttacksAttempted++

				switch event.HitResult.Type {
				case squads.HitTypeNormal:
					attPerf.AttacksHit++
					attPerf.DamageDealt += event.FinalDamage
				case squads.HitTypeCritical:
					attPerf.AttacksCrit++
					attPerf.DamageDealt += event.FinalDamage
				case squads.HitTypeMiss:
					attPerf.AttacksMissed++
				case squads.HitTypeDodge:
					attPerf.AttacksDodged++
				}
			}

			// Update defender stats
			if defPerf, ok := perfMap[event.DefenderID]; ok {
				defPerf.DamageReceived += event.FinalDamage
				defPerf.DamageBlocked += event.ResistanceAmount

				if event.CoverReduction.TotalReduction > 0 {
					// Estimate blocked damage from cover based on base damage
					defPerf.DamageBlocked += int(event.CoverReduction.TotalReduction * float64(event.BaseDamage))
				}
			}

			// Track cover providers
			for _, provider := range event.CoverReduction.Providers {
				if cpPerf, ok := perfMap[provider.UnitID]; ok {
					cpPerf.CoverInstancesProvided++
					cpPerf.DamageReductionProvided += provider.CoverValue
				}
			}
		}
	}

	// Update final HP and death turn
	for i := range perfs {
		entity := manager.FindEntityByID(perfs[i].UnitID)
		if entity != nil {
			attrs := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
			if attrs != nil {
				perfs[i].EndingHP = attrs.CurrentHealth
				if attrs.CurrentHealth <= 0 {
					perfs[i].TurnOfDeath = finalRound // Approximate
				}
			}
		}
	}
}

// createCombinedMetrics creates CombatMetrics from accumulated combat data
func (s *Simulator) createCombinedMetrics(
	scenario CombatScenario,
	logs []*squads.CombatLog,
	totalDamageDealt, totalDamageTaken int,
	unitsKilled []ecs.EntityID,
	turnsElapsed int,
	winner string,
) *CombatMetrics {
	metrics := NewCombatMetrics()
	metrics.Winner = winner
	metrics.TurnsElapsed = turnsElapsed

	metrics.DamageDealt[scenario.AttackerSetup.Name] = totalDamageDealt
	metrics.DamageTaken[scenario.AttackerSetup.Name] = totalDamageTaken
	metrics.DamageDealt[scenario.DefenderSetup.Name] = totalDamageTaken
	metrics.DamageTaken[scenario.DefenderSetup.Name] = totalDamageDealt

	metrics.UnitsKilled[scenario.DefenderSetup.Name] = len(unitsKilled)

	// Count mechanics from all logs
	for _, log := range logs {
		if log == nil {
			continue
		}

		for _, event := range log.AttackEvents {
			switch event.HitResult.Type {
			case squads.HitTypeMiss:
				metrics.MissCount++
			case squads.HitTypeDodge:
				metrics.DodgeCount++
			case squads.HitTypeCritical:
				metrics.CritCount++
			case squads.HitTypeNormal:
				metrics.HitCount++
			}

			if event.CoverReduction.TotalReduction > 0 {
				metrics.CoverApplications++
				metrics.TotalCoverReduction += event.CoverReduction.TotalReduction
			}
		}
	}

	// Use first log for main log reference
	if len(logs) > 0 {
		metrics.Log = logs[0]
	}

	return metrics
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

		// Add attributes (with potential overrides for sweeps)
		attrs := template.Attributes
		applyAttributeOverrides(&attrs, unitConfig.AttributeOverrides)
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

// applyAttributeOverrides applies override values to attributes for parameter sweeps
func applyAttributeOverrides(attrs *common.Attributes, overrides map[string]int) {
	if overrides == nil {
		return
	}

	for attr, value := range overrides {
		switch attr {
		case "Strength":
			attrs.Strength = value
		case "Dexterity":
			attrs.Dexterity = value
		case "Magic":
			attrs.Magic = value
		case "Leadership":
			attrs.Leadership = value
		case "Armor":
			attrs.Armor = value
		case "Weapon":
			attrs.Weapon = value
		case "MovementSpeed":
			attrs.MovementSpeed = value
		case "AttackRange":
			attrs.AttackRange = value
		}
	}

	// Recalculate derived stats after override
	attrs.MaxHealth = attrs.GetMaxHealth()
	attrs.CurrentHealth = attrs.MaxHealth
}
