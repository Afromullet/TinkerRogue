package framework

import (
	"fmt"
	"game_main/gui/builders"
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// PanelType identifies a type of panel in the registry
type PanelType string

// Standard panel types used across modes
const (
	// Combat mode panels
	PanelTypeTurnOrder     PanelType = "turn_order"
	PanelTypeFactionInfo   PanelType = "faction_info"
	PanelTypeSquadList     PanelType = "squad_list"
	PanelTypeSquadDetail   PanelType = "squad_detail"
	PanelTypeCombatLog     PanelType = "combat_log"
	PanelTypeCombatActions PanelType = "combat_actions"
	PanelTypeLayerStatus   PanelType = "layer_status"

	// Squad editor panels
	PanelTypeSquadNavigation PanelType = "squad_navigation"
	PanelTypeSquadSelector   PanelType = "squad_selector"
	PanelTypeGridEditor      PanelType = "grid_editor"
	PanelTypeUnitList        PanelType = "unit_list"
	PanelTypeRosterList      PanelType = "roster_list"
	PanelTypeEditorActions   PanelType = "editor_actions"

	// Exploration mode panels
	PanelTypeMessageLog  PanelType = "message_log"
	PanelTypeExploreInfo PanelType = "explore_info"

	// Common panels
	PanelTypeStatus  PanelType = "status"
	PanelTypeActions PanelType = "actions"
)

// PanelContentType specifies what widget goes inside the panel
type PanelContentType int

const (
	ContentEmpty       PanelContentType = iota // Just container
	ContentText                                // Text label
	ContentTextArea                            // TextArea for scrollable text
	ContentList                                // List widget
	ContentButtonGroup                         // Button group
	ContentGrid                                // Grid layout (e.g., 3x3 formation)
	ContentCustom                              // Custom widget tree via callback
)

// PanelDescriptor defines how to build a panel
type PanelDescriptor struct {
	// SpecName references a StandardPanels entry for layout
	SpecName string

	// Content specifies what widget type goes inside
	Content PanelContentType

	// Position override (if not using spec)
	Position func(*specs.LayoutConfig) builders.PanelOption

	// Size override (if not using spec)
	Width  float64 // As fraction of screen width (0.0-1.0)
	Height float64 // As fraction of screen height (0.0-1.0)

	// OnCreate is called after panel is built to customize it
	// The PanelResult contains the panel and any created widgets
	OnCreate func(*PanelResult, UIMode) error
}

// PanelResult holds the built panel and its widgets
type PanelResult struct {
	// Container is the root panel container
	Container *widget.Container

	// Type identifies this panel
	Type PanelType

	// Widget references (populated based on ContentType)
	TextLabel *widget.Text
	TextArea  *widget.TextArea
	List      *widget.List
	Buttons   []*widget.Button
	GridCells [][]*widget.Button // For grid panels

	// Custom widgets (for ContentCustom)
	Custom map[string]interface{}
}

// Global panel registry
var panelRegistry = make(map[PanelType]PanelDescriptor)

// RegisterPanel adds a panel type to the global registry.
// Call this in init() functions to define panel types.
//
// Example:
//
//	func init() {
//	    RegisterPanel(PanelTypeTurnOrder, PanelDescriptor{
//	        SpecName: "turn_order",
//	        Content:  ContentText,
//	    })
//	}
func RegisterPanel(ptype PanelType, desc PanelDescriptor) {
	panelRegistry[ptype] = desc
}

// GetPanelDescriptor returns the descriptor for a panel type
func GetPanelDescriptor(ptype PanelType) (PanelDescriptor, bool) {
	desc, ok := panelRegistry[ptype]
	return desc, ok
}

// BuildRegisteredPanel creates a panel from the registry.
// Returns a PanelResult with the container and widget references.
func BuildRegisteredPanel(ptype PanelType, mode UIMode, pb *builders.PanelBuilders, layout *specs.LayoutConfig) (*PanelResult, error) {
	desc, ok := panelRegistry[ptype]
	if !ok {
		return nil, fmt.Errorf("unknown panel type: %s", ptype)
	}

	result := &PanelResult{
		Type:   ptype,
		Custom: make(map[string]interface{}),
	}

	// Build container from spec or custom position/size
	// Skip if OnCreate will handle container creation
	if desc.OnCreate == nil {
		if desc.SpecName != "" {
			if spec, exists := builders.StandardPanels[desc.SpecName]; exists {
				result.Container = pb.BuildPanel(
					spec.Position,
					builders.Size(spec.Width, spec.Height),
					builders.Padding(spec.Padding),
					spec.Layout,
				)
			} else {
				return nil, fmt.Errorf("unknown spec: %s", desc.SpecName)
			}
		} else if desc.Position != nil {
			// Use custom position/size
			width := desc.Width
			if width == 0 {
				width = 0.2
			}
			height := desc.Height
			if height == 0 {
				height = 0.1
			}

			result.Container = pb.BuildPanel(
				desc.Position(layout),
				builders.Size(width, height),
			)
		}
	}

	// Run custom setup first if defined (handles container and content)
	if desc.OnCreate != nil {
		if err := desc.OnCreate(result, mode); err != nil {
			return nil, fmt.Errorf("panel %s OnCreate failed: %w", ptype, err)
		}
	} else if desc.Content == ContentText {
		result.TextLabel = builders.CreateSmallLabel("")
		if result.Container != nil {
			result.Container.AddChild(result.TextLabel)
		}
	}

	return result, nil
}

// PanelRegistry manages built panels for a mode
type PanelRegistry struct {
	panels map[PanelType]*PanelResult
}

// NewPanelRegistry creates an empty panel registry
func NewPanelRegistry() *PanelRegistry {
	return &PanelRegistry{
		panels: make(map[PanelType]*PanelResult),
	}
}

// Add stores a built panel
func (pr *PanelRegistry) Add(result *PanelResult) {
	pr.panels[result.Type] = result
}

// Get retrieves a built panel by type
func (pr *PanelRegistry) Get(ptype PanelType) *PanelResult {
	return pr.panels[ptype]
}
