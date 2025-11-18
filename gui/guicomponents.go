package gui

import (
	"fmt"
	"image/color"
	"game_main/common"
	"game_main/gear"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// SquadListComponent manages a container displaying squad buttons with filtering and selection
type SquadListComponent struct {
	container      *widget.Container
	queries        *GUIQueries
	filter         SquadFilter
	onSelect       func(squadID ecs.EntityID)
	listLabel      *widget.Text // First child is the label
	filteredSquads []ecs.EntityID // Cache for squad IDs
	squadButtons   []*widget.Button // Cache for squad buttons
}

// SquadFilter determines which squads to show
type SquadFilter func(squadInfo *SquadInfo) bool

// NewSquadListComponent creates a reusable squad list updater for a container
func NewSquadListComponent(
	container *widget.Container,
	queries *GUIQueries,
	filter SquadFilter,
	onSelect func(ecs.EntityID),
) *SquadListComponent {
	return &SquadListComponent{
		container:      container,
		queries:        queries,
		filter:         filter,
		onSelect:       onSelect,
		filteredSquads: make([]ecs.EntityID, 0),
		squadButtons:   make([]*widget.Button, 0),
	}
}

// Refresh updates the container with current squad buttons
func (slc *SquadListComponent) Refresh() {
	if slc.container == nil {
		return
	}

	// Remove old buttons, keep label (first child)
	children := slc.container.Children()
	for i := len(children) - 1; i >= 1; i-- {
		slc.container.RemoveChild(children[i])
	}
	slc.squadButtons = make([]*widget.Button, 0)
	slc.filteredSquads = make([]ecs.EntityID, 0)

	// Get all squads
	allSquads := slc.queries.FindAllSquads()

	// Filter squads and create buttons
	for _, squadID := range allSquads {
		squadInfo := slc.queries.GetSquadInfo(squadID)
		if squadInfo == nil || !slc.filter(squadInfo) {
			continue
		}

		slc.filteredSquads = append(slc.filteredSquads, squadID)

		// Create button for this squad
		localSquadID := squadID // Capture for closure
		squadName := slc.queries.GetSquadName(squadID)

		button := CreateButtonWithConfig(ButtonConfig{
			Text: squadName,
			OnClick: func() {
				if slc.onSelect != nil {
					slc.onSelect(localSquadID)
				}
			},
		})

		slc.container.AddChild(button)
		slc.squadButtons = append(slc.squadButtons, button)
	}

	// If no squads match filter, show AI turn message
	if len(slc.squadButtons) == 0 {
		noSquadsText := CreateTextWithConfig(TextConfig{
			Text:     "AI Turn",
			FontFace: SmallFace,
			Color:    color.Gray{Y: 128},
		})
		slc.container.AddChild(noSquadsText)
	}
}

// Common filter functions (delegated to GUIQueries for centralized filtering)

// AliveSquadsOnly filters to only non-destroyed squads
// Deprecated: Use GUIQueries.FilterSquadsAlive() for new code
func AliveSquadsOnly(info *SquadInfo) bool {
	return !info.IsDestroyed
}

// PlayerSquadsOnly returns a filter function for player-controlled faction squads
// Deprecated: Use GUIQueries.FilterSquadsByPlayer() for new code
func PlayerSquadsOnly(queries *GUIQueries) SquadFilter {
	return queries.FilterSquadsByPlayer()
}

// FactionSquadsOnly returns a filter function for squads in a specific faction
// Deprecated: Use GUIQueries.FilterSquadsByFaction() for new code
func FactionSquadsOnly(factionID ecs.EntityID) SquadFilter {
	// This requires GUIQueries to get the closure behavior, so it can't be delegated directly
	// Callers should use FilterSquadsByFaction(factionID) instead
	return func(info *SquadInfo) bool {
		return !info.IsDestroyed && info.FactionID == factionID
	}
}

// ===== DETAIL PANEL COMPONENT =====

// DetailPanelComponent manages a text widget showing entity details
type DetailPanelComponent struct {
	textWidget *widget.Text
	queries    *GUIQueries
	formatter  DetailFormatter
}

// DetailFormatter converts entity data to display text
type DetailFormatter func(data interface{}) string

// NewDetailPanelComponent creates a reusable detail panel updater for Text widgets
func NewDetailPanelComponent(
	textWidget *widget.Text,
	queries *GUIQueries,
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
	info := data.(*SquadInfo)
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
	info := data.(*FactionInfo)
	return fmt.Sprintf("%s\n\nSquads: %d/%d\nMana: %d/%d",
		info.Name,
		info.AliveSquadCount, len(info.SquadIDs),
		info.CurrentMana, info.MaxMana)
}

func getSquadStatus(info *SquadInfo) string {
	if info.HasActed {
		return "Acted"
	} else if info.HasMoved {
		return "Moved"
	} else {
		return "Ready"
	}
}

// ===== TEXT DISPLAY COMPONENT =====

// TextDisplayComponent manages a text widget with periodic updates
type TextDisplayComponent struct {
	textWidget *widget.Text
	formatter  TextDisplayFormatter
}

// TextDisplayFormatter converts data to display text
type TextDisplayFormatter func() string

// NewTextDisplayComponent creates a text display updater
func NewTextDisplayComponent(
	textWidget *widget.Text,
	formatter TextDisplayFormatter,
) *TextDisplayComponent {
	return &TextDisplayComponent{
		textWidget: textWidget,
		formatter:  formatter,
	}
}

// Refresh updates the text display
func (tdc *TextDisplayComponent) Refresh() {
	if tdc.textWidget == nil || tdc.formatter == nil {
		return
	}
	tdc.textWidget.Label = tdc.formatter()
}

// SetText sets arbitrary text directly
func (tdc *TextDisplayComponent) SetText(text string) {
	if tdc.textWidget != nil {
		tdc.textWidget.Label = text
	}
}

// ===== PANEL LIST COMPONENT =====

// PanelListComponent manages a container with panels for each item
type PanelListComponent struct {
	container      *widget.Container
	queries        *GUIQueries
	panelBuilders  *PanelBuilders
	panelFactory   PanelFactory
	filter         ItemFilter
	activePanels   []*widget.Container
}

// PanelFactory creates a panel for an entity
type PanelFactory func(queries *GUIQueries, squadID ecs.EntityID) *widget.Container

// ItemFilter determines which items to show
type ItemFilter func(squadID ecs.EntityID) bool

// NewPanelListComponent creates a panel list updater
func NewPanelListComponent(
	container *widget.Container,
	queries *GUIQueries,
	panelBuilders *PanelBuilders,
	panelFactory PanelFactory,
	filter ItemFilter,
) *PanelListComponent {
	return &PanelListComponent{
		container:     container,
		queries:       queries,
		panelBuilders: panelBuilders,
		panelFactory:  panelFactory,
		filter:        filter,
		activePanels:  make([]*widget.Container, 0),
	}
}

// Refresh rebuilds the panel list with current data
func (plc *PanelListComponent) Refresh() {
	if plc.container == nil {
		return
	}

	// Clear existing panels (keep non-panel children)
	plc.activePanels = make([]*widget.Container, 0)

	// Get all squads
	allSquads := plc.queries.FindAllSquads()

	// Create panel for each filtered squad
	for _, squadID := range allSquads {
		if plc.filter != nil && !plc.filter(squadID) {
			continue
		}

		panel := plc.panelFactory(plc.queries, squadID)
		if panel != nil {
			plc.container.AddChild(panel)
			plc.activePanels = append(plc.activePanels, panel)
		}
	}
}

// Clear removes all panels from the container
func (plc *PanelListComponent) Clear() {
	if plc.container == nil {
		return
	}

	for _, panel := range plc.activePanels {
		plc.container.RemoveChild(panel)
	}
	plc.activePanels = make([]*widget.Container, 0)
}

// ===== BUTTON LIST COMPONENT =====

// ButtonListComponent manages a container with buttons for each item
type ButtonListComponent struct {
	container      *widget.Container
	queries        *GUIQueries
	buttonFactory  ButtonFactory
	filter         ItemFilter
	activeButtons  []*widget.Button
	filteredItems  []ecs.EntityID
}

// ButtonFactory creates a button for an entity
type ButtonFactory func(queries *GUIQueries, squadID ecs.EntityID, onSelect func(ecs.EntityID)) *widget.Button

// NewButtonListComponent creates a button list updater
func NewButtonListComponent(
	container *widget.Container,
	queries *GUIQueries,
	buttonFactory ButtonFactory,
	filter ItemFilter,
) *ButtonListComponent {
	return &ButtonListComponent{
		container:     container,
		queries:       queries,
		buttonFactory: buttonFactory,
		filter:        filter,
		activeButtons: make([]*widget.Button, 0),
		filteredItems: make([]ecs.EntityID, 0),
	}
}

// Refresh rebuilds the button list with current data
func (blc *ButtonListComponent) Refresh(onSelect func(ecs.EntityID)) {
	if blc.container == nil {
		return
	}

	// Clear existing buttons
	for _, btn := range blc.activeButtons {
		blc.container.RemoveChild(btn)
	}
	blc.activeButtons = make([]*widget.Button, 0)
	blc.filteredItems = make([]ecs.EntityID, 0)

	// Get all squads
	allSquads := blc.queries.FindAllSquads()

	// Create button for each filtered squad
	for _, squadID := range allSquads {
		if blc.filter != nil && !blc.filter(squadID) {
			continue
		}

		localSquadID := squadID // Capture for closure
		button := blc.buttonFactory(blc.queries, squadID, func(id ecs.EntityID) {
			if onSelect != nil {
				onSelect(id)
			}
		})

		if button != nil {
			blc.container.AddChild(button)
			blc.activeButtons = append(blc.activeButtons, button)
			blc.filteredItems = append(blc.filteredItems, localSquadID)
		}
	}
}

// Clear removes all buttons from the container
func (blc *ButtonListComponent) Clear() {
	if blc.container == nil {
		return
	}

	for _, btn := range blc.activeButtons {
		blc.container.RemoveChild(btn)
	}
	blc.activeButtons = make([]*widget.Button, 0)
	blc.filteredItems = make([]ecs.EntityID, 0)
}

// ===== COLOR UTILITY COMPONENT =====

// ColorLabelComponent displays text with a specific color (used for status indicators)
type ColorLabelComponent struct {
	text  *widget.Text
	color color.Color
}

// NewColorLabelComponent creates a colored text label
func NewColorLabelComponent(
	textContent string,
	textColor color.Color,
) *ColorLabelComponent {
	return &ColorLabelComponent{
		text: CreateTextWithConfig(TextConfig{
			Text:     textContent,
			FontFace: SmallFace,
			Color:    textColor,
		}),
		color: textColor,
	}
}

// SetText updates the label text
func (clc *ColorLabelComponent) SetText(text string) {
	if clc.text != nil {
		clc.text.Label = text
	}
}

// SetColor updates the text color by recreating the text widget with new color
func (clc *ColorLabelComponent) SetColor(newColor color.Color) {
	clc.color = newColor
	if clc.text != nil {
		// Note: widget.Text doesn't support changing color after creation
		// To implement this properly would require replacing the widget in its parent container
		// For now, we just track the color change
	}
}

// GetWidget returns the underlying text widget
func (clc *ColorLabelComponent) GetWidget() *widget.Text {
	return clc.text
}

// ===== ITEM LIST COMPONENT =====

// ItemListComponent manages an inventory list widget with filtering
type ItemListComponent struct {
	listWidget    *widget.List
	queries       *GUIQueries
	ecsManager    *common.EntityManager
	playerEntityID ecs.EntityID
	currentFilter string
}

// NewItemListComponent creates a reusable inventory list component
func NewItemListComponent(
	listWidget *widget.List,
	queries *GUIQueries,
	ecsManager *common.EntityManager,
	playerEntityID ecs.EntityID,
) *ItemListComponent {
	return &ItemListComponent{
		listWidget:     listWidget,
		queries:        queries,
		ecsManager:     ecsManager,
		playerEntityID: playerEntityID,
		currentFilter:  "All",
	}
}

// SetFilter updates the current filter and refreshes the list
func (ilc *ItemListComponent) SetFilter(filter string) {
	ilc.currentFilter = filter
	ilc.Refresh()
}

// Refresh updates the list with items based on current filter
func (ilc *ItemListComponent) Refresh() {
	if ilc.listWidget == nil || ilc.ecsManager == nil {
		return
	}

	// Get inventory from player entity
	inv := common.GetComponentTypeByID[*gear.Inventory](ilc.ecsManager, ilc.playerEntityID, gear.InventoryComponent)
	if inv == nil {
		ilc.listWidget.SetEntries([]interface{}{"No inventory available"})
		return
	}

	var entries []interface{}

	// Query inventory based on current filter
	switch ilc.currentFilter {
	case "Throwables":
		// Get throwable items
		throwableEntries := gear.GetThrowableItems(ilc.ecsManager.World, inv, []int{})
		if len(throwableEntries) == 0 {
			entries = []interface{}{"No throwable items"}
		} else {
			entries = make([]interface{}, len(throwableEntries))
			for i, e := range throwableEntries {
				entries[i] = e
			}
		}

	case "All":
		// Get all items
		allEntries := gear.GetInventoryForDisplay(ilc.ecsManager.World, inv, []int{})
		if len(allEntries) == 0 {
			entries = []interface{}{"Inventory is empty"}
		} else {
			entries = make([]interface{}, len(allEntries))
			for i, e := range allEntries {
				entries[i] = e
			}
		}

	default:
		// Placeholder for other filters
		entries = []interface{}{fmt.Sprintf("Filter '%s' not yet implemented", ilc.currentFilter)}
	}

	ilc.listWidget.SetEntries(entries)
}

// ===== STATS DISPLAY COMPONENT =====

// StatsDisplayComponent manages a text widget displaying player statistics
type StatsDisplayComponent struct {
	textWidget *widget.TextArea
	formatter  StatsFormatter
}

// StatsFormatter converts player data to display text
type StatsFormatter func(*common.PlayerData, *common.EntityManager) string

// NewStatsDisplayComponent creates a stats display component
func NewStatsDisplayComponent(
	textWidget *widget.TextArea,
	formatter StatsFormatter,
) *StatsDisplayComponent {
	return &StatsDisplayComponent{
		textWidget: textWidget,
		formatter:  formatter,
	}
}

// RefreshStats updates the stats display
func (sdc *StatsDisplayComponent) RefreshStats(playerData *common.PlayerData, ecsManager *common.EntityManager) {
	if sdc.textWidget == nil {
		return
	}

	if playerData == nil {
		sdc.textWidget.SetText("No player data available")
		return
	}

	if sdc.formatter != nil {
		sdc.textWidget.SetText(sdc.formatter(playerData, ecsManager))
	} else {
		// Default formatter - display player attributes
		sdc.textWidget.SetText(playerData.PlayerAttributes(ecsManager).DisplayString())
	}
}

// SetText sets arbitrary text in the stats display
func (sdc *StatsDisplayComponent) SetText(text string) {
	if sdc.textWidget != nil {
		sdc.textWidget.SetText(text)
	}
}
