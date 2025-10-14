package squads

import (
	"fmt"

	"game_main/common"
	"game_main/entitytemplates"

	"github.com/bytearena/ecs"
)

var SquadsManager common.EntityManager
var Units = make([]UnitTemplate, 0, len(entitytemplates.MonsterTemplates))

// InitSquadComponents registers all squad-related components with the ECS manager.
// Call this during game initialization.
func InitSquadComponents(squadManager common.EntityManager) {
	SquadComponent = squadManager.World.NewComponent()
	SquadMemberComponent = squadManager.World.NewComponent()
	GridPositionComponent = squadManager.World.NewComponent()
	UnitRoleComponent = squadManager.World.NewComponent()
	CoverComponent = squadManager.World.NewComponent()
	LeaderComponent = squadManager.World.NewComponent()
	TargetRowComponent = squadManager.World.NewComponent()
	AbilitySlotComponent = squadManager.World.NewComponent()
	CooldownTrackerComponent = squadManager.World.NewComponent()
}

// InitSquadTags creates tags for querying squad-related entities
// Call this after InitSquadComponents
func InitSquadTags(squadManager common.EntityManager) {
	SquadTag = ecs.BuildTag(SquadComponent)
	SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
	LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

	squadManager.Tags["squad"] = SquadTag
	squadManager.Tags["squadmember"] = SquadMemberTag
	squadManager.Tags["leader"] = LeaderTag
}

func InitializeSquadData() error {
	SquadsManager = *common.NewEntityManager()
	InitSquadComponents(SquadsManager)
	InitSquadTags(SquadsManager)
	if err := InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to initialize units: %w", err)
	}
	return nil
}
