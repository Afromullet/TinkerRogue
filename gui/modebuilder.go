package gui

import (
	"fmt"

	"game_main/gui/core"
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ModeConfig provides declarative configuration for mode initialization.
// This eliminates 70-100 lines of repetitive initialization code in each mode.
//
// Usage:
//
//	func (m *MyMode) Initialize(ctx *core.UIContext) error {
//	    return NewModeBuilder(&m.BaseMode, ModeConfig{
//	        ModeName: "my_mode",
//	        ReturnMode: "parent_mode",
//	        Hotkeys: []HotkeySpec{{Key: ebiten.KeyI, TargetMode: "inventory"}},
//	        Panels: []PanelSpec{{SpecName: "message_log"}},
//	        Buttons: []ButtonGroupSpec{{Position: widgets.BottomCenter(), Buttons: myButtons}},
//	    }).Build(ctx)
//	}
type ModeConfig struct {
	ModeName   string
	ReturnMode string // Mode to return to on ESC (empty if no return mode)

	Hotkeys      []HotkeySpec
	Panels       []PanelSpec
	Buttons      []ButtonGroupSpec
	StatusLabel  bool // Whether to create a status label
	Commands     bool // Whether to enable command history
	OnRefresh    func() // Callback for command history refresh (required if Commands=true)
}

// HotkeySpec defines a keyboard shortcut that transitions to a target mode
type HotkeySpec struct {
	Key        ebiten.Key
	TargetMode string
}

// PanelSpec defines a panel to create during mode initialization.
// Use one of three approaches:
// 1. TypedPanel with SpecName (recommended): Specify PanelType + SpecName for standard panels with content
// 2. SpecName only: For simple container panels without content
// 3. CustomBuild: For complex custom panels
type PanelSpec struct {
	// Type-based panel creation (recommended)
	PanelType   widgets.PanelType         // Type of panel (Simple, Detail, List)
	SpecName    string                    // Panel specification name from widgets.StandardPanels
	DetailText  string                    // Initial text for detail panels
	ListConfig  *widgets.ListConfig       // List configuration for list panels

	// Widget references (populated after creation for access by mode)
	TextArea    *widget.TextArea          // Reference to created TextArea (for PanelTypeDetail)
	List        *widget.List              // Reference to created List (for PanelTypeList)

	// Legacy/custom approaches
	CustomBuild func() *widget.Container  // Optional: Custom panel builder function
	OnCreate    func(*widget.Container)   // DEPRECATED: Use PanelType instead
}

// ButtonGroupSpec defines a group of buttons positioned together
type ButtonGroupSpec struct {
	Position widgets.PanelOption    // Position (e.g., widgets.BottomCenter())
	Buttons  []widgets.ButtonSpec   // Button specifications
}

// ModeBuilder constructs UI modes using declarative configuration.
// Eliminates repetitive initialization boilerplate across 10+ mode implementations.
type ModeBuilder struct {
	baseMode *BaseMode
	config   ModeConfig
}

// NewModeBuilder creates a builder for the given BaseMode and configuration.
// The baseMode should be embedded in your mode struct (e.g., &myMode.BaseMode).
func NewModeBuilder(baseMode *BaseMode, config ModeConfig) *ModeBuilder {
	return &ModeBuilder{
		baseMode: baseMode,
		config:   config,
	}
}

// Build initializes the mode according to the configuration.
// This replaces 70-100 lines of manual initialization with declarative config.
//
// Steps performed:
// 1. Set mode name and return mode
// 2. Initialize BaseMode infrastructure
// 3. Register hotkeys
// 4. Build panels
// 5. Build button groups
// 6. Create status label (if configured)
// 7. Initialize command history (if configured)
func (mb *ModeBuilder) Build(ctx *core.UIContext) error {
	// Set mode properties
	mb.baseMode.SetModeName(mb.config.ModeName)
	if mb.config.ReturnMode != "" {
		mb.baseMode.SetReturnMode(mb.config.ReturnMode)
	}

	// Initialize common mode infrastructure
	mb.baseMode.InitializeBase(ctx)

	// Register hotkeys for mode transitions
	for _, hk := range mb.config.Hotkeys {
		mb.baseMode.RegisterHotkey(hk.Key, hk.TargetMode)
	}

	// Build panels
	if err := mb.buildPanels(); err != nil {
		return fmt.Errorf("failed to build panels: %w", err)
	}

	// Build button groups
	if err := mb.buildButtonGroups(); err != nil {
		return fmt.Errorf("failed to build button groups: %w", err)
	}

	// Create status label if configured
	if mb.config.StatusLabel {
		mb.baseMode.StatusLabel = widgets.CreateSmallLabel("")
		// Position status label below other content (modes can reposition if needed)
		mb.baseMode.RootContainer.AddChild(mb.baseMode.StatusLabel)
	}

	// Initialize command history if configured
	if mb.config.Commands {
		if mb.config.OnRefresh == nil {
			return fmt.Errorf("Commands=true requires OnRefresh callback")
		}
		mb.baseMode.InitializeCommandHistory(mb.config.OnRefresh)
	}

	return nil
}

// buildPanels creates all configured panels and adds them to the root container
func (mb *ModeBuilder) buildPanels() error {
	for i, panelSpec := range mb.config.Panels {
		var panel *widget.Container

		// Approach 1: TypedPanel with BuildTypedPanel (recommended)
		if panelSpec.PanelType != 0 || (panelSpec.SpecName != "" && panelSpec.DetailText != "") {
			result := mb.baseMode.PanelBuilders.BuildTypedPanel(widgets.TypedPanelConfig{
				Type:       panelSpec.PanelType,
				SpecName:   panelSpec.SpecName,
				DetailText: panelSpec.DetailText,
				ListConfig: panelSpec.ListConfig,
			})

			if result.Panel == nil {
				return fmt.Errorf("panel %d: failed to build typed panel with spec '%s'", i, panelSpec.SpecName)
			}

			panel = result.Panel

			// Store widget references for mode access
			mb.config.Panels[i].TextArea = result.TextArea
			mb.config.Panels[i].List = result.List

			// Store widgets in BaseMode.PanelWidgets map by SpecName
			if panelSpec.SpecName != "" {
				if result.TextArea != nil {
					mb.baseMode.PanelWidgets[panelSpec.SpecName] = result.TextArea
				}
				if result.List != nil {
					mb.baseMode.PanelWidgets[panelSpec.SpecName] = result.List
				}
			}

		// Approach 2: CustomBuild for complex panels
		} else if panelSpec.CustomBuild != nil {
			panel = panelSpec.CustomBuild()

		// Approach 3: Simple panel from SpecName
		} else if panelSpec.SpecName != "" {
			result := mb.baseMode.PanelBuilders.BuildTypedPanel(widgets.TypedPanelConfig{
				Type:     widgets.PanelTypeSimple,
				SpecName: panelSpec.SpecName,
			})

			if result.Panel == nil {
				return fmt.Errorf("panel %d: specification '%s' not found in StandardPanels", i, panelSpec.SpecName)
			}

			panel = result.Panel

		} else {
			return fmt.Errorf("panel %d: must specify either SpecName, PanelType, or CustomBuild", i)
		}

		// Call post-creation hook if provided (deprecated - use PanelType instead)
		if panelSpec.OnCreate != nil {
			panelSpec.OnCreate(panel)
		}

		// Add panel to root container
		mb.baseMode.RootContainer.AddChild(panel)
	}

	return nil
}

// buildButtonGroups creates all configured button groups and adds them to the root container
func (mb *ModeBuilder) buildButtonGroups() error {
	// Skip if no button groups configured
	if len(mb.config.Buttons) == 0 {
		return nil
	}

	// Verify PanelBuilders is initialized
	if mb.baseMode.PanelBuilders == nil {
		return fmt.Errorf("PanelBuilders is nil - ensure InitializeBase() was called")
	}

	for i, btnGroup := range mb.config.Buttons {
		if len(btnGroup.Buttons) == 0 {
			return fmt.Errorf("button group %d: must have at least one button", i)
		}

		// Verify Position is not nil
		if btnGroup.Position == nil {
			return fmt.Errorf("button group %d: Position is required", i)
		}

		container := CreateActionButtonGroup(
			mb.baseMode.PanelBuilders,
			btnGroup.Position,
			btnGroup.Buttons,
		)

		mb.baseMode.RootContainer.AddChild(container)
	}

	return nil
}
