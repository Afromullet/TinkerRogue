package guisquads

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/tactical/perks"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// PerkMode provides perk equip/unequip management in a separate screen.
// Opened from the squad editor via the "Perks" button or K key.
//
// Code organization:
// - perkmode.go: Lifecycle, navigation, tab switching
// - perk_panels_registry.go: Panel registrations via init()
// - perk_refresh.go: UI refresh logic
type PerkMode struct {
	framework.BaseMode

	// Active level selection
	activeLevel string // "squad", "unit", or "commander"

	// Tab switching
	activeTab string // "available" or "equipped"

	// Squad navigation
	currentSquadIndex int
	allSquadIDs       []ecs.EntityID

	// Unit navigation within current squad
	currentUnitIndex int
	allUnitIDs       []ecs.EntityID

	// Commander
	commanderID ecs.EntityID

	// Available tab widgets
	availableContent *widget.Container
	availableList    *widget.List
	availableTitle   *widget.Text
	availableDetail  *widget.TextArea
	slotButtonRow    *widget.Container

	// Equipped tab widgets
	equippedContent *widget.Container
	equippedList    *widget.List
	equippedTitle   *widget.Text
	equippedDetail  *widget.TextArea
	unequipButton   *widget.Button

	// Navigation widgets
	squadCounterLabel *widget.Text
	prevSquadButton   *widget.Button
	nextSquadButton   *widget.Button

	// Level buttons
	squadLevelButton     *widget.Button
	unitLevelButton      *widget.Button
	commanderLevelButton *widget.Button

	// Unit navigation widgets
	unitNavContainer *widget.Container
	unitNameLabel    *widget.Text
	prevUnitButton   *widget.Button
	nextUnitButton   *widget.Button

	// Selection state
	selectedAvailablePerkID string
	selectedEquipSlotIndex  int

	// Perk ID tracking parallel to available list entries
	availablePerkIDs []string
}

func NewPerkMode(modeManager *framework.UIModeManager) *PerkMode {
	mode := &PerkMode{
		currentSquadIndex:      0,
		allSquadIDs:            make([]ecs.EntityID, 0),
		allUnitIDs:             make([]ecs.EntityID, 0),
		activeLevel:            "squad",
		activeTab:              "available",
		selectedEquipSlotIndex: -1,
	}
	mode.SetModeName("perk_manager")
	mode.SetReturnMode("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode)
	return mode
}

func (pm *PerkMode) Initialize(ctx *framework.UIContext) error {
	err := framework.NewModeBuilder(&pm.BaseMode, framework.ModeConfig{
		ModeName:    "perk_manager",
		ReturnMode:  "squad_editor",
		StatusLabel: true,
	}).Build(ctx)

	if err != nil {
		return err
	}

	if err := pm.BuildPanels(
		PerkPanelSelector,
		PerkPanelContent,
	); err != nil {
		return err
	}

	pm.initializeWidgetReferences()
	return nil
}

func (pm *PerkMode) initializeWidgetReferences() {
	// Squad navigation
	pm.prevSquadButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "prevSquadButton")
	pm.nextSquadButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "nextSquadButton")
	pm.squadCounterLabel = framework.GetPanelWidget[*widget.Text](pm.Panels, PerkPanelSelector, "squadCounterLabel")

	// Level buttons
	pm.squadLevelButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "squadLevelButton")
	pm.unitLevelButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "unitLevelButton")
	pm.commanderLevelButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "commanderLevelButton")

	// Unit navigation
	pm.unitNavContainer = framework.GetPanelWidget[*widget.Container](pm.Panels, PerkPanelSelector, "unitNavContainer")
	pm.unitNameLabel = framework.GetPanelWidget[*widget.Text](pm.Panels, PerkPanelSelector, "unitNameLabel")
	pm.prevUnitButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "prevUnitButton")
	pm.nextUnitButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelSelector, "nextUnitButton")

	// Available tab
	pm.availableContent = framework.GetPanelWidget[*widget.Container](pm.Panels, PerkPanelContent, "availableContent")
	pm.availableList = framework.GetPanelWidget[*widget.List](pm.Panels, PerkPanelContent, "availableList")
	pm.availableTitle = framework.GetPanelWidget[*widget.Text](pm.Panels, PerkPanelContent, "availableTitle")
	pm.availableDetail = framework.GetPanelWidget[*widget.TextArea](pm.Panels, PerkPanelContent, "availableDetail")
	pm.slotButtonRow = framework.GetPanelWidget[*widget.Container](pm.Panels, PerkPanelContent, "slotButtonRow")

	// Equipped tab
	pm.equippedContent = framework.GetPanelWidget[*widget.Container](pm.Panels, PerkPanelContent, "equippedContent")
	pm.equippedList = framework.GetPanelWidget[*widget.List](pm.Panels, PerkPanelContent, "equippedList")
	pm.equippedTitle = framework.GetPanelWidget[*widget.Text](pm.Panels, PerkPanelContent, "equippedTitle")
	pm.equippedDetail = framework.GetPanelWidget[*widget.TextArea](pm.Panels, PerkPanelContent, "equippedDetail")
	pm.unequipButton = framework.GetPanelWidget[*widget.Button](pm.Panels, PerkPanelContent, "unequipButton")
}

func (pm *PerkMode) Enter(fromMode framework.UIMode) error {
	pm.syncSquadOrderFromRoster()
	pm.currentSquadIndex = 0
	pm.activeLevel = "squad"
	pm.activeTab = "available"

	// Load commander ID
	pm.commanderID = pm.Context.GetSquadRosterOwnerID()

	if len(pm.allSquadIDs) == 0 {
		pm.SetStatus("No squads available")
	} else {
		pm.loadUnitsForCurrentSquad()
		pm.refreshAllUI()
	}

	// Ensure available tab is visible on entry
	pm.availableContent.GetWidget().Visibility = widget.Visibility_Show
	pm.equippedContent.GetWidget().Visibility = widget.Visibility_Hide

	// Hide unit nav initially (squad level)
	pm.unitNavContainer.GetWidget().Visibility = widget.Visibility_Hide

	return nil
}

func (pm *PerkMode) Exit(toMode framework.UIMode) error {
	pm.selectedAvailablePerkID = ""
	pm.selectedEquipSlotIndex = -1
	return nil
}

func (pm *PerkMode) Update(deltaTime float64) error {
	return nil
}

func (pm *PerkMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (pm *PerkMode) HandleInput(inputState *framework.InputState) bool {
	if pm.HandleCommonInput(inputState) {
		return true
	}

	// Left/Right arrows cycle squads
	if inputState.KeysJustPressed[ebiten.KeyLeft] {
		pm.cycleSquad(-1)
		return true
	}
	if inputState.KeysJustPressed[ebiten.KeyRight] {
		pm.cycleSquad(1)
		return true
	}

	// Up/Down cycle units (when in unit level)
	if pm.activeLevel == "unit" {
		if inputState.KeysJustPressed[ebiten.KeyUp] {
			pm.cycleUnit(-1)
			return true
		}
		if inputState.KeysJustPressed[ebiten.KeyDown] {
			pm.cycleUnit(1)
			return true
		}
	}

	return false
}

// === Navigation ===

func (pm *PerkMode) currentSquadID() ecs.EntityID {
	if len(pm.allSquadIDs) == 0 {
		return 0
	}
	return pm.allSquadIDs[pm.currentSquadIndex]
}

func (pm *PerkMode) cycleSquad(delta int) {
	if len(pm.allSquadIDs) == 0 {
		return
	}
	pm.currentSquadIndex = (pm.currentSquadIndex + delta + len(pm.allSquadIDs)) % len(pm.allSquadIDs)
	pm.loadUnitsForCurrentSquad()
	pm.updateSquadCounter()
	pm.refreshActiveTab()
}

func (pm *PerkMode) cycleUnit(delta int) {
	if len(pm.allUnitIDs) == 0 {
		return
	}
	pm.currentUnitIndex = (pm.currentUnitIndex + delta + len(pm.allUnitIDs)) % len(pm.allUnitIDs)
	pm.updateUnitLabel()
	pm.refreshActiveTab()
}

func (pm *PerkMode) updateSquadCounter() {
	if pm.squadCounterLabel != nil && len(pm.allSquadIDs) > 0 {
		squadName := squads.GetSquadName(pm.currentSquadID(), pm.Context.ECSManager)
		pm.squadCounterLabel.Label = fmt.Sprintf("%s (%d/%d)", squadName, pm.currentSquadIndex+1, len(pm.allSquadIDs))
	}
	pm.updateSquadNavigationButtons()
}

func (pm *PerkMode) updateSquadNavigationButtons() {
	hasMultipleSquads := len(pm.allSquadIDs) > 1
	if pm.prevSquadButton != nil {
		pm.prevSquadButton.GetWidget().Disabled = !hasMultipleSquads
	}
	if pm.nextSquadButton != nil {
		pm.nextSquadButton.GetWidget().Disabled = !hasMultipleSquads
	}
}

func (pm *PerkMode) updateUnitLabel() {
	if pm.unitNameLabel == nil || len(pm.allUnitIDs) == 0 {
		return
	}
	unitID := pm.allUnitIDs[pm.currentUnitIndex]
	name := common.GetEntityName(pm.Context.ECSManager, unitID, "Unknown")
	pm.unitNameLabel.Label = fmt.Sprintf("%s (%d/%d)", name, pm.currentUnitIndex+1, len(pm.allUnitIDs))
}

func (pm *PerkMode) updateUnitNavigationButtons() {
	hasMultipleUnits := len(pm.allUnitIDs) > 1
	if pm.prevUnitButton != nil {
		pm.prevUnitButton.GetWidget().Disabled = !hasMultipleUnits
	}
	if pm.nextUnitButton != nil {
		pm.nextUnitButton.GetWidget().Disabled = !hasMultipleUnits
	}
}

func (pm *PerkMode) syncSquadOrderFromRoster() {
	rosterOwnerID := pm.Context.GetSquadRosterOwnerID()
	manager := pm.Context.ECSManager

	roster := squads.GetPlayerSquadRoster(rosterOwnerID, manager)
	if roster == nil {
		return
	}

	pm.allSquadIDs = make([]ecs.EntityID, len(roster.OwnedSquads))
	copy(pm.allSquadIDs, roster.OwnedSquads)
}

func (pm *PerkMode) loadUnitsForCurrentSquad() {
	if len(pm.allSquadIDs) == 0 {
		pm.allUnitIDs = nil
		pm.currentUnitIndex = 0
		return
	}
	pm.allUnitIDs = squads.GetUnitIDsInSquad(pm.currentSquadID(), pm.Context.ECSManager)
	pm.currentUnitIndex = 0
}

// switchLevel changes between squad/unit/commander perk views
func (pm *PerkMode) switchLevel(levelName string) {
	if pm.activeLevel == levelName {
		return
	}
	pm.activeLevel = levelName
	pm.selectedAvailablePerkID = ""
	pm.selectedEquipSlotIndex = -1

	// Show/hide unit navigation
	if pm.unitNavContainer != nil {
		if levelName == "unit" {
			pm.unitNavContainer.GetWidget().Visibility = widget.Visibility_Show
			pm.updateUnitLabel()
			pm.updateUnitNavigationButtons()
		} else {
			pm.unitNavContainer.GetWidget().Visibility = widget.Visibility_Hide
		}
	}

	pm.refreshActiveTab()
}

// getCurrentEntityID returns the entity being edited based on activeLevel
func (pm *PerkMode) getCurrentEntityID() ecs.EntityID {
	switch pm.activeLevel {
	case "squad":
		return pm.currentSquadID()
	case "unit":
		if len(pm.allUnitIDs) == 0 {
			return 0
		}
		return pm.allUnitIDs[pm.currentUnitIndex]
	case "commander":
		return pm.commanderID
	default:
		return 0
	}
}

// getCurrentPerkLevel returns the PerkLevel for the active level
func (pm *PerkMode) getCurrentPerkLevel() perks.PerkLevel {
	switch pm.activeLevel {
	case "squad":
		return perks.PerkLevelSquad
	case "unit":
		return perks.PerkLevelUnit
	case "commander":
		return perks.PerkLevelCommander
	default:
		return perks.PerkLevelSquad
	}
}

// getSlotCount returns the number of perk slots for the active level
func (pm *PerkMode) getSlotCount() int {
	switch pm.activeLevel {
	case "squad":
		return 3
	case "unit":
		return 2
	case "commander":
		return 3
	default:
		return 0
	}
}

// === Tab Switching ===

func (pm *PerkMode) switchTab(tabName string) {
	if pm.activeTab == tabName {
		return
	}
	pm.activeTab = tabName

	pm.availableContent.GetWidget().Visibility = widget.Visibility_Hide
	pm.equippedContent.GetWidget().Visibility = widget.Visibility_Hide

	switch tabName {
	case "available":
		pm.availableContent.GetWidget().Visibility = widget.Visibility_Show
		pm.refreshAvailable()
	case "equipped":
		pm.equippedContent.GetWidget().Visibility = widget.Visibility_Show
		pm.refreshEquipped()
	}
}
