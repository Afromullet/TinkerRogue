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
// Use either SpecName (from StandardPanels) OR CustomBuild (custom function).
type PanelSpec struct {
	SpecName    string                       // Panel specification name from widgets.StandardPanels
	CustomBuild func() *widget.Container     // Optional: Custom panel builder function
	OnCreate    func(*widget.Container)      // Optional: Post-creation callback (e.g., to add children)
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

		// Use custom builder or standard specification
		if panelSpec.CustomBuild != nil {
			panel = panelSpec.CustomBuild()
		} else if panelSpec.SpecName != "" {
			panel = widgets.CreateStandardPanel(mb.baseMode.PanelBuilders, panelSpec.SpecName)
			if panel == nil {
				return fmt.Errorf("panel %d: specification '%s' not found in StandardPanels", i, panelSpec.SpecName)
			}
		} else {
			return fmt.Errorf("panel %d: must specify either SpecName or CustomBuild", i)
		}

		// Call post-creation hook if provided
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
	for i, btnGroup := range mb.config.Buttons {
		if len(btnGroup.Buttons) == 0 {
			return fmt.Errorf("button group %d: must have at least one button", i)
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
