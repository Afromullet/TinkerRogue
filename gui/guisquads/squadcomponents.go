package guisquads

import (
	"fmt"

	"game_main/gui/framework"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

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
