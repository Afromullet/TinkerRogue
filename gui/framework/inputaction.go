package framework

// InputAction represents a semantic game action decoupled from physical keys.
// Modes check actions instead of raw key constants, enabling rebindable controls.
type InputAction int

const (
	ActionNone InputAction = iota

	// === Universal ===
	ActionCancel  // ESC - close panel, exit mode, cancel operation
	ActionConfirm // Enter - confirm selection, dismiss dialog
	ActionDismiss // Space - dismiss summary, end turn in some contexts

	// === Undo/Redo ===
	ActionUndo // Ctrl+Z
	ActionRedo // Ctrl+Y

	// === Combat ===
	ActionAttackMode       // A - toggle attack mode
	ActionMoveMode         // M - toggle move mode
	ActionSpellPanel       // S - toggle spell panel
	ActionArtifactPanel    // D - toggle artifact panel
	ActionInspectMode      // I - toggle inspect mode
	ActionCycleSquad       // Tab - cycle through squads
	ActionEndTurn          // Space - end current turn
	ActionSelectTarget1    // 1 - select enemy target 1
	ActionSelectTarget2    // 2 - select enemy target 2
	ActionSelectTarget3    // 3 - select enemy target 3
	ActionThreatToggle     // H - toggle threat heat map
	ActionThreatCycleFact  // Shift+H - cycle threat faction
	ActionHealthBarToggle  // CtrlRight - toggle health bars
	ActionLayerToggle      // L - toggle layer visualizer
	ActionLayerCycleMode   // Shift+L - cycle layer mode
	ActionUndoMove         // Ctrl+Z in combat context
	ActionDebugKillAll     // Ctrl+K - kill all enemies (debug)
	ActionAoERotateLeft    // 1 in AoE targeting
	ActionAoERotateRight   // 2 in AoE targeting

	// === Overworld ===
	ActionNodePlacement    // N - enter node placement mode
	ActionOverworldMove    // M - toggle movement mode
	ActionCycleCommander   // Tab - cycle to next commander
	ActionToggleInfluence  // I - toggle influence display
	ActionGarrison         // G - garrison management
	ActionRecruitCommander // R - recruit new commander
	ActionSquadManagement  // S - open squad management
	ActionEngageThreat     // E - engage threat at commander position
	ActionEndOverworldTurn // Space - end overworld turn
	ActionMouseClick       // Left mouse button click

	// === Squad Editor ===
	ActionToggleUnits          // U - toggle units panel
	ActionToggleRoster         // R - toggle roster panel
	ActionNewSquad             // N - create new squad
	ActionToggleAttackPattern  // V - toggle attack pattern view
	ActionToggleSupportPattern // B - toggle support pattern view
	ActionCycleCommanderEditor // Tab - cycle to next commander in editor

	// === Artifact Mode ===
	ActionPrevSquad       // Left arrow
	ActionNextSquad       // Right arrow
	ActionTabInventory    // I - inventory tab
	ActionTabEquipment    // E - equipment tab

	// === Node Placement ===
	ActionCycleNodeType   // Tab - cycle node types
	ActionSelectNodeType1 // 1
	ActionSelectNodeType2 // 2
	ActionSelectNodeType3 // 3
	ActionSelectNodeType4 // 4

	// === Raid ===
	ActionSelectRoom1 // 1
	ActionSelectRoom2 // 2
	ActionSelectRoom3 // 3
	ActionSelectRoom4 // 4
	ActionSelectRoom5 // 5
	ActionSelectRoom6 // 6
	ActionSelectRoom7 // 7
	ActionSelectRoom8 // 8
	ActionSelectRoom9 // 9
	ActionDeployBack  // ESC - return to floor map from deploy

	// === Combat Animation ===
	ActionReplayAnimation // Space - replay combat animation

	// === Camera (exploration) ===
	ActionCameraMoveUp        // W (on release)
	ActionCameraMoveDown      // S (on release)
	ActionCameraMoveLeft      // A (on release)
	ActionCameraMoveRight     // D (on release)
	ActionCameraMoveUpLeft    // Q (on release)
	ActionCameraMoveUpRight   // E (on release)
	ActionCameraMoveDownLeft  // Z (on release)
	ActionCameraMoveDownRight // C (on release)
	ActionCameraHighlight     // B (on release) - debug tile highlight
	ActionCameraToggleScroll  // M (on release) - toggle map scrolling
)
