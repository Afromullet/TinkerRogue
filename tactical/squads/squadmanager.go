package squads

import (
	"fmt"

	"game_main/common"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

var Units = make([]UnitTemplate, 0, len(templates.MonsterTemplates))

// init registers the squads subsystem with the ECS component registry.
// This allows the squads package to self-register its components without
// game_main needing to know about squad internals.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitSquadComponents(em)
		InitSquadTags(em)
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
	AbilitySlotComponent = squadManager.World.NewComponent()
	CooldownTrackerComponent = squadManager.World.NewComponent()
	AttackRangeComponent = squadManager.World.NewComponent()
	MovementSpeedComponent = squadManager.World.NewComponent()
	UnitRosterComponent = squadManager.World.NewComponent()
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

// InitializeSquadData initializes squad components and templates in the provided EntityManager.
// This should be called with the game's main EntityManager (&g.em) so squads exist in the same ECS world.
func InitializeSquadData(manager *common.EntityManager) error {
	InitSquadComponents(manager)
	InitSquadTags(manager)
	if err := InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to initialize units: %w", err)
	}
	return nil
}
