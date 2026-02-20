package guiunitview

import (
	"fmt"
	"math/rand"
	"time"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// UnitViewMode displays read-only unit details (attributes, level, XP).
// Accessed from the squad editor via "View Unit" button.
type UnitViewMode struct {
	framework.BaseMode

	viewUnitID ecs.EntityID
	detailText *widget.TextArea
	rng        *rand.Rand
}

// NewUnitViewMode creates a new UnitViewMode instance.
func NewUnitViewMode(modeManager *framework.UIModeManager) *UnitViewMode {
	mode := &UnitViewMode{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	mode.SetModeName("unit_view")
	mode.SetReturnMode("squad_editor")
	mode.ModeManager = modeManager
	mode.SetSelf(mode)
	return mode
}

// SetUnitID sets the unit to display. Call before transitioning to this mode.
func (uvm *UnitViewMode) SetUnitID(unitID ecs.EntityID) {
	uvm.viewUnitID = unitID
}

// Initialize builds the mode UI using ModeBuilder and panel registry.
func (uvm *UnitViewMode) Initialize(ctx *framework.UIContext) error {
	err := framework.NewModeBuilder(&uvm.BaseMode, framework.ModeConfig{
		ModeName:   "unit_view",
		ReturnMode: "squad_editor",
	}).Build(ctx)
	if err != nil {
		return err
	}

	if err := uvm.BuildPanels(UnitViewPanelDetail); err != nil {
		return err
	}

	uvm.detailText = framework.GetPanelWidget[*widget.TextArea](uvm.Panels, UnitViewPanelDetail, "detailText")
	return nil
}

// Enter is called when switching to this mode. Refreshes the unit display.
func (uvm *UnitViewMode) Enter(fromMode framework.UIMode) error {
	uvm.refreshUnitDisplay()
	return nil
}

// HandleInput delegates to common input handling (ESC to return).
func (uvm *UnitViewMode) HandleInput(inputState *framework.InputState) bool {
	return uvm.HandleCommonInput(inputState)
}

// onAddXP awards 100 XP to the viewed unit and refreshes the display (debug).
func (uvm *UnitViewMode) onAddXP() {
	squads.AwardExperience(uvm.viewUnitID, 100, uvm.Context.ECSManager, uvm.rng)
	uvm.refreshUnitDisplay()
}

// refreshUnitDisplay populates the text area with unit details.
func (uvm *UnitViewMode) refreshUnitDisplay() {
	if uvm.detailText == nil {
		return
	}

	manager := uvm.Context.ECSManager
	unitID := uvm.viewUnitID

	name := common.GetEntityName(manager, unitID, "Unknown")

	// Unit type
	unitType := "Unknown"
	if utData := common.GetComponentTypeByID[*squads.UnitTypeData](manager, unitID, squads.UnitTypeComponent); utData != nil {
		unitType = utData.UnitType
	}

	// Attributes
	attrText := "  (no attributes)"
	if attrComp, ok := manager.GetComponent(unitID, common.AttributeComponent); ok {
		attr := attrComp.(*common.Attributes)
		attrText = fmt.Sprintf(
			"  Strength:   %d\n  Dexterity:  %d\n  Magic:      %d\n  Leadership: %d\n  Armor:      %d\n  Weapon:     %d\n  HP:         %d / %d",
			attr.Strength, attr.Dexterity, attr.Magic,
			attr.Leadership, attr.Armor, attr.Weapon,
			attr.CurrentHealth, attr.MaxHealth,
		)
	}

	// Experience
	expText := "  (no experience data)"
	if expData := squads.GetExperienceData(unitID, manager); expData != nil {
		expText = fmt.Sprintf(
			"  Level:  %d\n  XP:     %d / %d",
			expData.Level, expData.CurrentXP, expData.XPToNextLevel,
		)
	}

	text := fmt.Sprintf(
		"Name: %s\nType: %s\n\n--- Attributes ---\n%s\n\n--- Experience ---\n%s",
		name, unitType, attrText, expText,
	)

	uvm.detailText.SetText(text)
}
