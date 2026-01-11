package framework

import (
	"fmt"

	"game_main/gui/builders"

	"github.com/hajimehoshi/ebiten/v2"
)

// ModeConfig provides declarative configuration for mode initialization.
// This eliminates repetitive initialization code in each mode.
//
// Usage:
//
//	func (m *MyMode) Initialize(ctx *UIContext) error {
//	    err := NewModeBuilder(&m.BaseMode, ModeConfig{
//	        ModeName:   "my_mode",
//	        ReturnMode: "parent_mode",
//	        Hotkeys:    []HotkeySpec{{Key: ebiten.KeyI, TargetMode: "inventory"}},
//	    }).Build(ctx)
//	    if err != nil {
//	        return err
//	    }
//
//	    // Build panels from registry
//	    return m.BuildPanels(MyPanelType1, MyPanelType2)
//	}
type ModeConfig struct {
	ModeName   string
	ReturnMode string // Mode to return to on ESC (empty if no return mode)

	Hotkeys     []HotkeySpec
	StatusLabel bool   // Whether to create a status label
	Commands    bool   // Whether to enable command history
	OnRefresh   func() // Callback for command history refresh (required if Commands=true)
}

// HotkeySpec defines a keyboard shortcut that transitions to a target mode
type HotkeySpec struct {
	Key        ebiten.Key
	TargetMode string
}

// ModeBuilder constructs UI modes using declarative configuration.
// Eliminates repetitive initialization boilerplate across mode implementations.
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
//
// Steps performed:
// 1. Set mode name and return mode
// 2. Initialize BaseMode infrastructure
// 3. Register hotkeys
// 4. Create status label (if configured)
// 5. Initialize command history (if configured)
//
// After Build() completes, call BuildPanels() to add panels from the registry.
func (mb *ModeBuilder) Build(ctx *UIContext) error {
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

	// Create status label if configured
	if mb.config.StatusLabel {
		mb.baseMode.StatusLabel = builders.CreateSmallLabel("")
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
