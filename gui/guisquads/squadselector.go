package guisquads

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// SquadSelector manages prev/next cycling through squads with a label display.
// Mirrors CommanderSelector's design for squad-level navigation.
type SquadSelector struct {
	AllIDs     []ecs.EntityID
	CurrentIdx int
	Label      *widget.Text
	PrevBtn    *widget.Button
	NextBtn    *widget.Button
}

// NewSquadSelector creates a selector bound to the given widgets. Nil-safe for unused widgets.
func NewSquadSelector(label *widget.Text, prevBtn, nextBtn *widget.Button) *SquadSelector {
	return &SquadSelector{
		Label:   label,
		PrevBtn: prevBtn,
		NextBtn: nextBtn,
	}
}

// Load enumerates all squads from the roster owner and clamps the current index.
func (ss *SquadSelector) Load(rosterOwnerID ecs.EntityID, manager *common.EntityManager) {
	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		ss.AllIDs = nil
		ss.UpdateLabel()
		ss.UpdateButtons()
		return
	}

	ss.AllIDs = make([]ecs.EntityID, len(roster.OwnedSquads))
	copy(ss.AllIDs, roster.OwnedSquads)

	if ss.CurrentIdx >= len(ss.AllIDs) {
		ss.CurrentIdx = 0
	}

	ss.UpdateLabel()
	ss.UpdateButtons()
}

// CurrentID returns the currently selected squad's entity ID.
func (ss *SquadSelector) CurrentID() ecs.EntityID {
	if len(ss.AllIDs) == 0 {
		return 0
	}
	return ss.AllIDs[ss.CurrentIdx]
}

// Cycle advances the squad index by delta (-1 for previous, +1 for next).
// Calls onSwitch after changing if provided.
func (ss *SquadSelector) Cycle(delta int, onSwitch func()) {
	if len(ss.AllIDs) <= 1 {
		return
	}
	ss.CurrentIdx = (ss.CurrentIdx + delta + len(ss.AllIDs)) % len(ss.AllIDs)
	if onSwitch != nil {
		onSwitch()
	}
	ss.UpdateLabel()
	ss.UpdateButtons()
}

// SelectByID finds the squad with the given ID and sets it as current.
// Returns true if found.
func (ss *SquadSelector) SelectByID(entityID ecs.EntityID) bool {
	for i, id := range ss.AllIDs {
		if id == entityID {
			ss.CurrentIdx = i
			return true
		}
	}
	return false
}

// ResetIndex sets the current index to 0.
func (ss *SquadSelector) ResetIndex() {
	ss.CurrentIdx = 0
}

// HasSquads returns true if there is at least one squad loaded.
func (ss *SquadSelector) HasSquads() bool {
	return len(ss.AllIDs) > 0
}

// UpdateLabel refreshes the "Squad X of Y" display.
func (ss *SquadSelector) UpdateLabel() {
	if ss.Label == nil {
		return
	}
	if len(ss.AllIDs) == 0 {
		ss.Label.Label = "No Squads"
		return
	}
	ss.Label.Label = fmt.Sprintf("Squad %d of %d", ss.CurrentIdx+1, len(ss.AllIDs))
}

// UpdateButtons disables prev/next when only one squad exists.
func (ss *SquadSelector) UpdateButtons() {
	hasMultiple := len(ss.AllIDs) > 1
	if ss.PrevBtn != nil {
		ss.PrevBtn.GetWidget().Disabled = !hasMultiple
	}
	if ss.NextBtn != nil {
		ss.NextBtn.GetWidget().Disabled = !hasMultiple
	}
}
