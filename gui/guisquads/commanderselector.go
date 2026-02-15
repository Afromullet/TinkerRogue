package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/commander"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// CommanderSelector manages prev/next cycling through commanders with a label display.
type CommanderSelector struct {
	AllIDs     []ecs.EntityID
	CurrentIdx int
	Label      *widget.Text
	PrevBtn    *widget.Button
	NextBtn    *widget.Button
}

// NewCommanderSelector creates a selector bound to the given widgets.
func NewCommanderSelector(label *widget.Text, prevBtn, nextBtn *widget.Button) *CommanderSelector {
	return &CommanderSelector{
		Label:   label,
		PrevBtn: prevBtn,
		NextBtn: nextBtn,
	}
}

// Load enumerates all commanders for the given player and syncs the index to selectedID.
func (cs *CommanderSelector) Load(playerID ecs.EntityID, selectedID ecs.EntityID, manager *common.EntityManager) {
	cs.AllIDs = commander.GetAllCommanders(playerID, manager)
	cs.CurrentIdx = 0
	for i, id := range cs.AllIDs {
		if id == selectedID {
			cs.CurrentIdx = i
			break
		}
	}
	cs.UpdateLabel(manager)
	cs.UpdateButtons()
}

// CurrentID returns the currently selected commander's entity ID.
func (cs *CommanderSelector) CurrentID() ecs.EntityID {
	if len(cs.AllIDs) == 0 {
		return 0
	}
	return cs.AllIDs[cs.CurrentIdx]
}

// ShowPrevious cycles to the previous commander. Calls onSwitch with the new ID if changed.
func (cs *CommanderSelector) ShowPrevious(manager *common.EntityManager, onSwitch func(ecs.EntityID)) {
	if len(cs.AllIDs) <= 1 {
		return
	}
	cs.CurrentIdx--
	if cs.CurrentIdx < 0 {
		cs.CurrentIdx = len(cs.AllIDs) - 1
	}
	onSwitch(cs.CurrentID())
	cs.UpdateLabel(manager)
	cs.UpdateButtons()
}

// ShowNext cycles to the next commander. Calls onSwitch with the new ID if changed.
func (cs *CommanderSelector) ShowNext(manager *common.EntityManager, onSwitch func(ecs.EntityID)) {
	if len(cs.AllIDs) <= 1 {
		return
	}
	cs.CurrentIdx++
	if cs.CurrentIdx >= len(cs.AllIDs) {
		cs.CurrentIdx = 0
	}
	onSwitch(cs.CurrentID())
	cs.UpdateLabel(manager)
	cs.UpdateButtons()
}

// UpdateLabel refreshes the commander name display.
func (cs *CommanderSelector) UpdateLabel(manager *common.EntityManager) {
	if cs.Label == nil {
		return
	}
	if len(cs.AllIDs) == 0 {
		cs.Label.Label = "No Commanders"
		return
	}
	cmdrData := commander.GetCommanderData(cs.AllIDs[cs.CurrentIdx], manager)
	if cmdrData != nil {
		cs.Label.Label = fmt.Sprintf("Commander: %s", cmdrData.Name)
	} else {
		cs.Label.Label = "Commander: ???"
	}
}

// UpdateButtons disables prev/next when only one commander exists.
func (cs *CommanderSelector) UpdateButtons() {
	hasMultiple := len(cs.AllIDs) > 1
	if cs.PrevBtn != nil {
		cs.PrevBtn.GetWidget().Disabled = !hasMultiple
	}
	if cs.NextBtn != nil {
		cs.NextBtn.GetWidget().Disabled = !hasMultiple
	}
}
