package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// ChangeLeaderCommand changes the squad leader to a different unit
type ChangeLeaderCommand struct {
	manager    *common.EntityManager
	squadID    ecs.EntityID
	newLeaderID ecs.EntityID

	// Undo state
	oldLeaderID ecs.EntityID
}

func NewChangeLeaderCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	newLeaderID ecs.EntityID,
) *ChangeLeaderCommand {
	return &ChangeLeaderCommand{
		manager:     manager,
		squadID:     squadID,
		newLeaderID: newLeaderID,
	}
}

func (c *ChangeLeaderCommand) Validate() error {
	// Check squad exists
	squadEntity := squads.GetSquadEntity(c.squadID, c.manager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Check new leader is in squad
	if !c.manager.HasComponentByIDWithTag(c.newLeaderID, squads.SquadMemberTag, squads.SquadMemberComponent) {
		return fmt.Errorf("new leader is not in a squad")
	}

	memberData := common.GetComponentTypeByID[*squads.SquadMemberData](c.manager, c.newLeaderID, squads.SquadMemberComponent)
	if memberData == nil || memberData.SquadID != c.squadID {
		return fmt.Errorf("new leader is not in this squad")
	}

	// Check new leader is not already the leader
	isLeader := c.manager.HasComponentByIDWithTag(c.newLeaderID, squads.SquadMemberTag, squads.LeaderComponent)
	if isLeader {
		return fmt.Errorf("unit is already the leader")
	}

	return nil
}

func (c *ChangeLeaderCommand) Execute() error {
	// Find current leader
	c.oldLeaderID = squads.GetLeaderID(c.squadID, c.manager)

	// Remove leader component from old leader (if exists)
	if c.oldLeaderID != 0 {
		oldLeaderEntity := common.FindEntityByIDWithTag(c.manager, c.oldLeaderID, squads.SquadMemberTag)
		if oldLeaderEntity != nil {
			if oldLeaderEntity.HasComponent(squads.LeaderComponent) {
				oldLeaderEntity.RemoveComponent(squads.LeaderComponent)
			}
			if oldLeaderEntity.HasComponent(squads.AbilitySlotComponent) {
				oldLeaderEntity.RemoveComponent(squads.AbilitySlotComponent)
			}
			if oldLeaderEntity.HasComponent(squads.CooldownTrackerComponent) {
				oldLeaderEntity.RemoveComponent(squads.CooldownTrackerComponent)
			}
		}
	}

	// Add leader component to new leader
	newLeaderEntity := common.FindEntityByIDWithTag(c.manager, c.newLeaderID, squads.SquadMemberTag)
	if newLeaderEntity == nil {
		return fmt.Errorf("new leader entity not found")
	}

	// Add leader component with default values
	newLeaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{
		Leadership: 10,
		Experience: 0,
	})

	// Add ability slots
	newLeaderEntity.AddComponent(squads.AbilitySlotComponent, &squads.AbilitySlotData{
		Slots: [4]squads.AbilitySlot{},
	})

	// Add cooldown tracker
	newLeaderEntity.AddComponent(squads.CooldownTrackerComponent, &squads.CooldownTrackerData{
		Cooldowns:    [4]int{0, 0, 0, 0},
		MaxCooldowns: [4]int{0, 0, 0, 0},
	})

	// Update squad capacity based on new leader
	squads.UpdateSquadCapacity(c.squadID, c.manager)

	return nil
}

func (c *ChangeLeaderCommand) Undo() error {
	// Remove leader component from new leader
	newLeaderEntity := common.FindEntityByIDWithTag(c.manager, c.newLeaderID, squads.SquadMemberTag)
	if newLeaderEntity != nil {
		if newLeaderEntity.HasComponent(squads.LeaderComponent) {
			newLeaderEntity.RemoveComponent(squads.LeaderComponent)
		}
		if newLeaderEntity.HasComponent(squads.AbilitySlotComponent) {
			newLeaderEntity.RemoveComponent(squads.AbilitySlotComponent)
		}
		if newLeaderEntity.HasComponent(squads.CooldownTrackerComponent) {
			newLeaderEntity.RemoveComponent(squads.CooldownTrackerComponent)
		}
	}

	// Restore old leader (if there was one)
	if c.oldLeaderID != 0 {
		oldLeaderEntity := common.FindEntityByIDWithTag(c.manager, c.oldLeaderID, squads.SquadMemberTag)
		if oldLeaderEntity != nil {
			oldLeaderEntity.AddComponent(squads.LeaderComponent, &squads.LeaderData{
				Leadership: 10,
				Experience: 0,
			})

			oldLeaderEntity.AddComponent(squads.AbilitySlotComponent, &squads.AbilitySlotData{
				Slots: [4]squads.AbilitySlot{},
			})

			oldLeaderEntity.AddComponent(squads.CooldownTrackerComponent, &squads.CooldownTrackerData{
				Cooldowns:    [4]int{0, 0, 0, 0},
				MaxCooldowns: [4]int{0, 0, 0, 0},
			})
		}
	}

	// Update squad capacity
	squads.UpdateSquadCapacity(c.squadID, c.manager)

	return nil
}

func (c *ChangeLeaderCommand) Description() string {
	// Get unit names for better description
	newLeaderName := "Unit"
	if nameComp, ok := c.manager.GetComponent(c.newLeaderID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			newLeaderName = name.NameStr
		}
	}

	return fmt.Sprintf("Change leader to '%s'", newLeaderName)
}
