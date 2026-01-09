package guisquads

import (
	"fmt"
	"image/color"

	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/guiresources"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// SquadListComponent manages a container displaying squad buttons with filtering and selection
type SquadListComponent struct {
	container      *widget.Container
	queries        *framework.GUIQueries
	filter         framework.SquadFilter
	onSelect       func(squadID ecs.EntityID)
	listLabel      *widget.Text                    // First child is the label
	filteredSquads []ecs.EntityID                  // Currently displayed squad IDs (for change detection)
	buttons        map[ecs.EntityID]*widget.Button // NEW: Reuse buttons between refreshes
	noSquadsText   *widget.Text                    // NEW: Cache "AI Turn" message widget
}

// NewSquadListComponent creates a reusable squad list updater for a container
func NewSquadListComponent(
	container *widget.Container,
	queries *framework.GUIQueries,
	filter framework.SquadFilter,
	onSelect func(ecs.EntityID),
) *SquadListComponent {
	return &SquadListComponent{
		container:      container,
		queries:        queries,
		filter:         filter,
		onSelect:       onSelect,
		filteredSquads: make([]ecs.EntityID, 0),
		buttons:        make(map[ecs.EntityID]*widget.Button), // Initialize button cache
		noSquadsText:   nil,
	}
}

// Refresh updates the container with current squad buttons
func (slc *SquadListComponent) Refresh() {
	if slc.container == nil {
		return
	}

	// Get all squads and apply filter
	allSquads := slc.queries.SquadCache.FindAllSquads()
	newFilteredSquads := make([]ecs.EntityID, 0, len(allSquads))

	for _, squadID := range allSquads {
		squadInfo := slc.queries.GetSquadInfo(squadID)
		if squadInfo == nil || !slc.filter(squadInfo) {
			continue
		}
		newFilteredSquads = append(newFilteredSquads, squadID)
	}

	if !slc.squadListChanged(newFilteredSquads) {
		// FAST PATH: No change - just update button labels if needed
		slc.updateButtonLabels(newFilteredSquads)
		return
	}

	// SLOW PATH: Squad list changed - update widgets
	slc.updateButtonWidgets(newFilteredSquads)
	slc.filteredSquads = newFilteredSquads
}

// squadListChanged checks if the filtered squad list has changed
func (slc *SquadListComponent) squadListChanged(newSquads []ecs.EntityID) bool {
	if len(slc.filteredSquads) != len(newSquads) {
		return true
	}
	for i := range slc.filteredSquads {
		if slc.filteredSquads[i] != newSquads[i] {
			return true
		}
	}
	return false
}

// updateButtonLabels updates button text without recreating widgets (FAST)
func (slc *SquadListComponent) updateButtonLabels(squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		button, exists := slc.buttons[squadID]
		if !exists {
			continue
		}

		squadInfo := slc.queries.GetSquadInfo(squadID)
		if squadInfo == nil {
			continue
		}

		// Update button text if it changed (Text widget will remeasure on next render, not now)
		textWidget := button.Text()
		if textWidget != nil && textWidget.Label != squadInfo.Name {
			textWidget.Label = squadInfo.Name
		}
	}
}

// updateButtonWidgets recreates the widget list when squad list changes (SLOW)
func (slc *SquadListComponent) updateButtonWidgets(squadIDs []ecs.EntityID) {
	// Remove buttons for squads no longer in list
	for squadID, button := range slc.buttons {
		if !slc.containsSquad(squadIDs, squadID) {
			slc.container.RemoveChild(button)
			delete(slc.buttons, squadID)
		}
	}

	// Remove "AI Turn" message if present
	if slc.noSquadsText != nil {
		slc.container.RemoveChild(slc.noSquadsText)
		slc.noSquadsText = nil
	}

	// Rebuild container children in correct order
	slc.container.RemoveChildren()
	if slc.listLabel != nil {
		slc.container.AddChild(slc.listLabel)
	}

	// Add or reorder buttons
	for _, squadID := range squadIDs {
		button, exists := slc.buttons[squadID]

		if !exists {
			// Create new button ONLY if not in cache
			squadInfo := slc.queries.GetSquadInfo(squadID)
			if squadInfo == nil {
				continue
			}

			localSquadID := squadID // Capture for closure

			button = builders.CreateButtonWithConfig(builders.ButtonConfig{
				Text: squadInfo.Name,
				OnClick: func() {
					if slc.onSelect != nil {
						slc.onSelect(localSquadID)
					}
				},
			})

			slc.buttons[squadID] = button
		}

		slc.container.AddChild(button)
	}

	// If no squads match filter, show AI turn message
	if len(squadIDs) == 0 {
		// Create "AI Turn" message once and cache it
		if slc.noSquadsText == nil {
			slc.noSquadsText = builders.CreateTextWithConfig(builders.TextConfig{
				Text:     "AI Turn",
				FontFace: guiresources.SmallFace,
				Color:    color.Gray{Y: 128},
			})
		}
		slc.container.AddChild(slc.noSquadsText)
	}
}

// containsSquad checks if a squadID is in the list
func (slc *SquadListComponent) containsSquad(squads []ecs.EntityID, squadID ecs.EntityID) bool {
	for _, id := range squads {
		if id == squadID {
			return true
		}
	}
	return false
}

// ===== DETAIL PANEL COMPONENT =====

// DetailPanelComponent manages a text widget showing entity details
type DetailPanelComponent struct {
	textWidget *widget.Text
	queries    *framework.GUIQueries
	formatter  DetailFormatter
}

// DetailFormatter converts entity data to display text
type DetailFormatter func(data interface{}) string

// NewDetailPanelComponent creates a reusable detail panel updater for Text widgets
func NewDetailPanelComponent(
	textWidget *widget.Text,
	queries *framework.GUIQueries,
	formatter DetailFormatter,
) *DetailPanelComponent {
	return &DetailPanelComponent{
		textWidget: textWidget,
		queries:    queries,
		formatter:  formatter,
	}
}

// ShowSquad displays squad details
func (dpc *DetailPanelComponent) ShowSquad(squadID ecs.EntityID) {
	if dpc.textWidget == nil {
		return
	}

	squadInfo := dpc.queries.GetSquadInfo(squadID)
	if squadInfo == nil {
		dpc.textWidget.Label = "Squad not found"
		return
	}

	if dpc.formatter != nil {
		dpc.textWidget.Label = dpc.formatter(squadInfo)
	} else {
		// Default formatter
		dpc.textWidget.Label = DefaultSquadFormatter(squadInfo)
	}
}

// ShowFaction displays faction details
func (dpc *DetailPanelComponent) ShowFaction(factionID ecs.EntityID) {
	if dpc.textWidget == nil {
		return
	}

	factionInfo := dpc.queries.GetFactionInfo(factionID)
	if factionInfo == nil {
		dpc.textWidget.Label = "Faction not found"
		return
	}

	if dpc.formatter != nil {
		dpc.textWidget.Label = dpc.formatter(factionInfo)
	} else {
		dpc.textWidget.Label = DefaultFactionFormatter(factionInfo)
	}
}

// SetText sets arbitrary text in the detail panel
func (dpc *DetailPanelComponent) SetText(text string) {
	if dpc.textWidget != nil {
		dpc.textWidget.Label = text
	}
}

// Default formatters

// DefaultSquadFormatter creates a formatted string from squad info
func DefaultSquadFormatter(data interface{}) string {
	info := data.(*framework.SquadInfo)
	status := getSquadStatus(info)
	return fmt.Sprintf("%s\n\nUnits: %d/%d\nHP: %d/%d\nMove: %d\nStatus: %s",
		info.Name,
		info.AliveUnits, info.TotalUnits,
		info.CurrentHP, info.MaxHP,
		info.MovementRemaining,
		status)
}

// DefaultFactionFormatter creates a formatted string from faction info
func DefaultFactionFormatter(data interface{}) string {
	info := data.(*framework.FactionInfo)
	return fmt.Sprintf("%s\n\nSquads: %d/%d\nMana: %d/%d",
		info.Name,
		info.AliveSquadCount, len(info.SquadIDs),
		info.CurrentMana, info.MaxMana)
}

func getSquadStatus(info *framework.SquadInfo) string {
	if info.HasActed {
		return "Acted"
	} else if info.HasMoved {
		return "Moved"
	} else {
		return "Ready"
	}
}
