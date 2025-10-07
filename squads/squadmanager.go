package squads

import (
	"fmt"

	"game_main/entitytemplates"

	"github.com/bytearena/ecs"
)

var SquadsManager SquadECSManager
var Units = make([]UnitTemplate, 0, len(entitytemplates.MonsterTemplates))

type SquadECSManager struct {
	Manager *ecs.Manager
	Tags    map[string]ecs.Tag
}

func NewSquadECSManager() *SquadECSManager {

	return &SquadECSManager{
		Manager: ecs.NewManager(),
		Tags:    make(map[string]ecs.Tag),
	}

}

// InitSquadComponents registers all squad-related components with the ECS manager.
// Call this during game initialization.
func InitSquadComponents(squadManager SquadECSManager) {
	SquadComponent = squadManager.Manager.NewComponent()
	SquadMemberComponent = squadManager.Manager.NewComponent()
	GridPositionComponent = squadManager.Manager.NewComponent()
	UnitRoleComponent = squadManager.Manager.NewComponent()
	LeaderComponent = squadManager.Manager.NewComponent()
	TargetRowComponent = squadManager.Manager.NewComponent()
	AbilitySlotComponent = squadManager.Manager.NewComponent()
	CooldownTrackerComponent = squadManager.Manager.NewComponent()
}

// InitSquadTags creates tags for querying squad-related entities
// Call this after InitSquadComponents
func InitSquadTags(squadManager SquadECSManager) {
	SquadTag = ecs.BuildTag(SquadComponent)
	SquadMemberTag = ecs.BuildTag(SquadMemberComponent)
	LeaderTag = ecs.BuildTag(LeaderComponent, SquadMemberComponent)

	squadManager.Tags["squad"] = SquadTag
	squadManager.Tags["squadmember"] = SquadMemberTag
	squadManager.Tags["leader"] = LeaderTag
}

func InitializeSquadData() error {
	SquadsManager = *NewSquadECSManager()
	InitSquadComponents(SquadsManager)
	InitSquadTags(SquadsManager)
	if err := InitUnitTemplatesFromJSON(); err != nil {
		return fmt.Errorf("failed to initialize units: %w", err)
	}
	return nil
}
