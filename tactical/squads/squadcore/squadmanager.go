package squadcore

import (
	"fmt"

	"game_main/core/common"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

// Package-level ECS Views for zero-allocation squad queries.
// Initialized once during subsystem registration; automatically maintained by the ECS library.
// All canonical squad queries in squadqueries.go iterate these views directly.
var (
	squadView       *ecs.View // All SquadTag entities
	squadMemberView *ecs.View // All SquadMemberTag entities
	leaderView      *ecs.View // All LeaderTag entities
)

// init registers the squads subsystem with the ECS component registry.
// This allows the squads package to self-register its components without
// game_main needing to know about squad internals.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitSquadComponents(em)
		InitSquadTags(em)
		squadView = em.World.CreateView(SquadTag)
		squadMemberView = em.World.CreateView(SquadMemberTag)
		leaderView = em.World.CreateView(LeaderTag)
	})
}

// InitSquadComponents registers all squad-related components with the ECS manager.
// Call this during game initialization.
func InitSquadComponents(squadManager *common.EntityManager) {
	SquadComponent = squadManager.World.NewComponent()
	SquadMemberComponent = squadManager.World.NewComponent()
	GridPositionComponent = squadManager.World.NewComponent()
	UnitRoleComponent = squadManager.World.NewComponent()
	CoverComponent = squadManager.World.NewComponent()
	LeaderComponent = squadManager.World.NewComponent()
	TargetRowComponent = squadManager.World.NewComponent()
	AttackRangeComponent = squadManager.World.NewComponent()
	MovementSpeedComponent = squadManager.World.NewComponent()
	UnitTypeComponent = squadManager.World.NewComponent()
}

// InitSquadTags creates tags for querying squad-related entities
// Call this after InitSquadComponents
func InitSquadTags(squadManager *common.EntityManager) {
	SquadTag = ecs.BuildTag(SquadComponent)
	SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
	LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

	squadManager.WorldTags["squad"] = SquadTag
	squadManager.WorldTags["squadmember"] = SquadMemberTag
	squadManager.WorldTags["leader"] = LeaderTag
}

// InitializeSquadData initializes squad templates (unit data from JSON).
// Components and tags are auto-initialized via init() and common.InitializeSubsystems().
// This should be called with the game's main EntityManager for template loading only.
func InitializeSquadData(manager *common.EntityManager) error {
	if err := unitdefs.InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to initialize units: %w", err)
	}
	return nil
}
