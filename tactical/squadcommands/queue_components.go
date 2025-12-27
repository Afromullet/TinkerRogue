package squadcommands

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CommandQueueData holds pending commands for a squad
// Processed one per turn by ProcessCommandQueues()
type CommandQueueData struct {
	Commands []SquadCommand // Ordered queue of pending commands
	Paused   bool           // If true, queue processing paused (animations, etc.)
}

var CommandQueueComponent *ecs.Component
var CommandQueueTag ecs.Tag

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitializeCommandQueueComponents(em)
	})
}

func InitializeCommandQueueComponents(manager *common.EntityManager) {
	CommandQueueComponent = manager.World.NewComponent()
	CommandQueueTag = ecs.BuildTag(CommandQueueComponent)
	manager.WorldTags["CommandQueue"] = CommandQueueTag
}
