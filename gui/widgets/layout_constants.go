// Package gui - Layout configuration constants
package widgets

// ========================================
// PANEL WIDTH PERCENTAGES
// ========================================

const (
	// Narrow side panels (stats display, faction info)
	PanelWidthNarrow = 0.15

	// Standard side panels (squad lists, inventory filters)
	PanelWidthStandard = 0.2

	// Medium panels (filters, secondary content)
	PanelWidthMedium = 0.3

	// Wide panels (top bars, main content areas)
	PanelWidthWide = 0.4

	// Extra wide panels (detail views, full-width content)
	PanelWidthExtraWide = 0.45
)

// ========================================
// PANEL HEIGHT PERCENTAGES
// ========================================

const (
	// Tiny panels (top bars, button containers)
	PanelHeightTiny = 0.08

	// Small panels (faction info, header sections)
	PanelHeightSmall = 0.12

	// Quarter screen height
	PanelHeightQuarter = 0.25

	// Third of screen height
	PanelHeightThird = 0.33

	// Half screen height
	PanelHeightHalf = 0.5

	// Tall panels (detail views, list containers)
	PanelHeightTall = 0.75

	// Extra tall panels (between half and full)
	PanelHeightExtraTall = 0.6

	// Nearly full screen (main content areas)
	PanelHeightFull = 0.85
)

// ========================================
// STANDARD PADDING PERCENTAGES
// ========================================

const (
	// Extra small padding (replaces 8-10px hardcoded values)
	PaddingExtraSmall = 0.0125 // ~10px at 800px screen

	// Tight padding for most panels (replaces 15px hardcoded values)
	PaddingTight = 0.015 // ~12px at 800px screen, ~18px at 1200px

	// Standard padding between elements (replaces 20px hardcoded values)
	PaddingStandard = 0.02 // ~16px at 800px, ~24px at 1200px

	// Loose padding for spacious layouts (replaces 30px hardcoded values)
	PaddingLoose = 0.03 // ~24px at 800px, ~36px at 1200px

	// Vertical offset for stacking widgets (replaces 80px top offset)
	PaddingStackedWidget = 0.08 // ~64px at 800px screen, ~96px at 1200px
)

// ========================================
// LAYOUT OFFSET CONSTANTS
// ========================================

const (
	// Space reserved at bottom for button containers
	BottomButtonOffset = 0.08
)

// ========================================
// UI MODE LAYOUT SPECIFICATIONS
// ========================================

// ExplorationModeLayout defines layout constants for exploration mode
const (
	// Map display and main content
	ExplorationMapWidth = 0.7
)

// CombatModeLayout defines layout constants for combat mode
// Layout structure:
//   Top: TurnOrder (center, 8%), FactionInfo (top-left, 10%)
//   Left: SquadList (35%) + SquadDetail (25%) = 60% middle section
//   Bottom: ActionButtons (center, 15%), CombatLog (right, 15%)
const (
	// Panel widths
	CombatTurnOrderWidth    = 0.4  // Top center turn order bar
	CombatFactionInfoWidth  = 0.18 // Top left faction info
	CombatSquadListWidth    = 0.18 // Left side squad list
	CombatSquadDetailWidth  = 0.18 // Left side squad detail
	CombatLogWidth          = 0.22 // Bottom right combat log
	CombatActionButtonWidth = 0.35 // Bottom center buttons

	// Panel heights
	CombatTurnOrderHeight   = 0.08 // Top bar (8% from top)
	CombatFactionInfoHeight = 0.10 // Faction info (10% at very top)
	CombatSquadListHeight   = 0.35 // Squad list (35% of middle area)
	CombatSquadDetailHeight = 0.25 // Squad detail (25% of middle area)
	CombatLogHeight         = 0.15 // Combat log (15% at bottom)
	CombatActionButtonHeight = 0.08 // Button strip (8%)
)

// InventoryModeLayout defines layout constants for inventory mode
const (
	// Item list (main area)
	InventoryListWidth  = 0.5
	InventoryListHeight = PanelHeightTall
)

// SquadManagementLayout defines layout constants for squad management mode
const (
	// Center squad panel
	SquadMgmtPanelWidth  = 0.6
	SquadMgmtPanelHeight = 0.5

	// Navigation bar (Previous/Next buttons)
	SquadMgmtNavWidth  = 0.5
	SquadMgmtNavHeight = 0.08

	// Command bar (Disband, Merge, Undo, Redo)
	SquadMgmtCmdWidth  = 0.6
	SquadMgmtCmdHeight = 0.08

	// Status label
	SquadMgmtStatusHeight = 0.05

	// Action buttons (bottom)
	SquadMgmtButtonHeight = 0.08
)

// SquadBuilderLayout defines layout constants for squad builder mode
const (
	// Grid area (left side)
	SquadBuilderGridWidth = 0.5

	// Unit list panel (right side)
	SquadBuilderUnitListWidth  = 0.25
	SquadBuilderUnitListHeight = PanelHeightTall

	// Squad info panel (right side, above unit list)
	SquadBuilderInfoWidth  = 0.25
	SquadBuilderInfoHeight = 0.2
)

// SquadEditorLayout defines layout constants for squad editor mode
const (
	// Navigation bar (top-center)
	SquadEditorNavHeight = 0.08

	// Squad selector list (left)
	SquadEditorSquadListWidth  = 0.25
	SquadEditorSquadListHeight = 0.7

	// Unit list (center-left)
	SquadEditorUnitListWidth  = 0.25
	SquadEditorUnitListHeight = 0.7

	// Roster list (center-right)
	SquadEditorRosterListWidth  = 0.25
	SquadEditorRosterListHeight = 0.7
)

// UnitPurchaseLayout defines layout constants for unit purchase mode
const (
	// Unit list (left side)
	UnitPurchaseListWidth  = 0.35
	UnitPurchaseListHeight = 0.7

	// Detail panel (right side)
	UnitPurchaseDetailWidth  = 0.35
	UnitPurchaseDetailHeight = 0.6

	// Resource display (top-center)
	UnitPurchaseResourceWidth  = 0.25
	UnitPurchaseResourceHeight = 0.08
)

// SquadDeploymentLayout defines layout constants for squad deployment mode
const (
	// Squad list panel (left)
	SquadDeployListWidth  = 0.3
	SquadDeployListHeight = 0.7

	// Instruction text (top-center)
	SquadDeployInstructWidth  = 0.5
	SquadDeployInstructHeight = 0.15
)

// FormationEditorLayout defines layout constants for formation editor mode
const (
	// Formation grid preview (center)
	FormationGridWidth  = 0.5
	FormationGridHeight = 0.6

	// Squad selector (left)
	FormationSquadListWidth  = 0.2
	FormationSquadListHeight = 0.7

	// Unit palette (right)
	FormationPaletteWidth  = 0.2
	FormationPaletteHeight = 0.7
)

// InfoModeLayout defines layout constants for info display mode
const (
	// Main content area
	InfoContentWidth = PanelWidthExtraWide
)
