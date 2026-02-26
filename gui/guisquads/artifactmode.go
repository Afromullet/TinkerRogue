package guisquads

import (
	"game_main/gui/builders"
	"game_main/gui/framework"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ArtifactMode provides inventory and equipment management in a separate screen.
// Opened from the squad editor via the "Artifacts" button.
//
// Code organization:
// - artifactmode.go: Lifecycle, navigation, tab switching
// - artifact_panels_registry.go: Panel registrations via init()
// - artifact_refresh.go: UI refresh logic
type ArtifactMode struct {
	framework.BaseMode

	// Tab switching
	activeTab string // "inventory" or "equipment"

	// Inventory tab widgets
	inventoryContent *widget.Container
	inventoryList    *widget.List
	inventoryTitle   *widget.Text
	inventoryDetail  *widget.TextArea
	inventoryButton  *widget.Button

	// Equipment tab widgets
	equipmentContent *widget.Container
	equipmentList    *widget.List
	equipmentTitle   *widget.Text
	equipmentDetail  *widget.TextArea
	equipmentButton  *widget.Button

	// Selection state
	selectedInventoryArtifact string
	selectedEquippedArtifact  string

	// Squad navigation
	squadSelector *SquadSelector
}

func NewArtifactMode(modeManager *framework.UIModeManager) *ArtifactMode {
	mode := &ArtifactMode{
		activeTab: "inventory",
	}
	mode.SetModeName("artifact_manager")
	mode.SetReturnMode("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode)
	return mode
}

func (am *ArtifactMode) Initialize(ctx *framework.UIContext) error {
	err := framework.NewModeBuilder(&am.BaseMode, framework.ModeConfig{
		ModeName:    "artifact_manager",
		ReturnMode:  "squad_editor",
		StatusLabel: true,
	}).Build(ctx)

	if err != nil {
		return err
	}

	if err := am.BuildPanels(
		ArtifactPanelSquadSelector,
		ArtifactPanelContent,
	); err != nil {
		return err
	}

	am.initializeWidgetReferences()

	// Add bottom navigation bar
	am.RootContainer.AddChild(am.buildNavigationActions())

	return nil
}

// initializeWidgetReferences populates mode fields from panel registry
func (am *ArtifactMode) initializeWidgetReferences() {
	// Squad selector navigation
	am.squadSelector = NewSquadSelector(
		framework.GetPanelWidget[*widget.Text](am.Panels, ArtifactPanelSquadSelector, "counterLabel"),
		framework.GetPanelWidget[*widget.Button](am.Panels, ArtifactPanelSquadSelector, "prevButton"),
		framework.GetPanelWidget[*widget.Button](am.Panels, ArtifactPanelSquadSelector, "nextButton"),
	)

	// Inventory tab
	am.inventoryContent = framework.GetPanelWidget[*widget.Container](am.Panels, ArtifactPanelContent, "inventoryContent")
	am.inventoryList = framework.GetPanelWidget[*widget.List](am.Panels, ArtifactPanelContent, "inventoryList")
	am.inventoryTitle = framework.GetPanelWidget[*widget.Text](am.Panels, ArtifactPanelContent, "inventoryTitle")
	am.inventoryDetail = framework.GetPanelWidget[*widget.TextArea](am.Panels, ArtifactPanelContent, "inventoryDetail")
	am.inventoryButton = framework.GetPanelWidget[*widget.Button](am.Panels, ArtifactPanelContent, "inventoryButton")

	// Equipment tab
	am.equipmentContent = framework.GetPanelWidget[*widget.Container](am.Panels, ArtifactPanelContent, "equipmentContent")
	am.equipmentList = framework.GetPanelWidget[*widget.List](am.Panels, ArtifactPanelContent, "equipmentList")
	am.equipmentTitle = framework.GetPanelWidget[*widget.Text](am.Panels, ArtifactPanelContent, "equipmentTitle")
	am.equipmentDetail = framework.GetPanelWidget[*widget.TextArea](am.Panels, ArtifactPanelContent, "equipmentDetail")
	am.equipmentButton = framework.GetPanelWidget[*widget.Button](am.Panels, ArtifactPanelContent, "equipmentButton")
}

func (am *ArtifactMode) Enter(fromMode framework.UIMode) error {
	am.squadSelector.ResetIndex()
	am.squadSelector.Load(am.Context.GetSquadRosterOwnerID(), am.Context.ECSManager)
	am.activeTab = "inventory"

	if !am.squadSelector.HasSquads() {
		am.SetStatus("No squads available")
	} else {
		am.refreshAllUI()
	}

	// Ensure inventory tab is visible on entry
	am.inventoryContent.GetWidget().Visibility = widget.Visibility_Show
	am.equipmentContent.GetWidget().Visibility = widget.Visibility_Hide

	return nil
}

func (am *ArtifactMode) Exit(toMode framework.UIMode) error {
	am.selectedInventoryArtifact = ""
	am.selectedEquippedArtifact = ""
	return nil
}

func (am *ArtifactMode) Update(deltaTime float64) error {
	return nil
}

func (am *ArtifactMode) Render(screen *ebiten.Image) {
	// No custom rendering needed
}

func (am *ArtifactMode) HandleInput(inputState *framework.InputState) bool {
	// ESC returns to squad editor
	if am.HandleCommonInput(inputState) {
		return true
	}

	// Left/Right arrows cycle squads
	if inputState.KeysJustPressed[ebiten.KeyLeft] {
		am.squadSelector.Cycle(-1, am.refreshActiveTab)
		return true
	}
	if inputState.KeysJustPressed[ebiten.KeyRight] {
		am.squadSelector.Cycle(1, am.refreshActiveTab)
		return true
	}

	// Tab switching hotkeys
	if inputState.KeysJustPressed[ebiten.KeyI] {
		am.switchTab("inventory")
		return true
	}
	if inputState.KeysJustPressed[ebiten.KeyE] {
		am.switchTab("equipment")
		return true
	}

	return false
}

// === Tab Switching ===

func (am *ArtifactMode) switchTab(tabName string) {
	if am.activeTab == tabName {
		return
	}
	am.activeTab = tabName

	am.inventoryContent.GetWidget().Visibility = widget.Visibility_Hide
	am.equipmentContent.GetWidget().Visibility = widget.Visibility_Hide

	switch tabName {
	case "inventory":
		am.inventoryContent.GetWidget().Visibility = widget.Visibility_Show
		am.refreshInventory()
	case "equipment":
		am.equipmentContent.GetWidget().Visibility = widget.Visibility_Show
		am.refreshEquipment()
	}
}

// buildNavigationActions creates bottom-right navigation buttons
func (am *ArtifactMode) buildNavigationActions() *widget.Container {
	spacing := int(float64(am.Layout.ScreenWidth) * specs.PaddingTight)
	bottomPad := int(float64(am.Layout.ScreenHeight) * specs.BottomButtonOffset)
	rightPad := int(float64(am.Layout.ScreenWidth) * specs.PaddingStandard)
	anchorLayout := builders.AnchorEndEnd(rightPad, bottomPad)

	return builders.CreateButtonGroup(builders.ButtonGroupConfig{
		Buttons: []builders.ButtonSpec{
			{Text: "Back (ESC)", OnClick: func() {
				if returnMode, exists := am.ModeManager.GetMode(am.GetReturnMode()); exists {
					am.ModeManager.RequestTransition(returnMode, "Back button pressed")
				}
			}},
		},
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    builders.NewResponsiveHorizontalPadding(am.Layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})
}
