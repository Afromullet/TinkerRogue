package framework

import "github.com/hajimehoshi/ebiten/v2"

// CommonBindings returns bindings shared across most modes (just ESC -> Cancel).
func CommonBindings() *ActionMap {
	return NewActionMap("common").
		Bind(ebiten.KeyEscape, ActionCancel)
}

// DefaultUndoRedoBindings returns standard Ctrl+Z / Ctrl+Y bindings.
func DefaultUndoRedoBindings() *ActionMap {
	return NewActionMap("undo_redo").
		BindMod(ebiten.KeyZ, ModCtrl, ActionUndo).
		BindMod(ebiten.KeyY, ModCtrl, ActionRedo)
}

// DefaultCombatBindings returns all combat mode key bindings.
func DefaultCombatBindings() *ActionMap {
	return NewActionMap("combat").
		Bind(ebiten.KeyA, ActionAttackMode).
		Bind(ebiten.KeyM, ActionMoveMode).
		Bind(ebiten.KeyS, ActionSpellPanel).
		Bind(ebiten.KeyD, ActionArtifactPanel).
		Bind(ebiten.KeyI, ActionInspectMode).
		Bind(ebiten.KeyTab, ActionCycleSquad).
		Bind(ebiten.KeySpace, ActionEndTurn).
		Bind(ebiten.Key1, ActionSelectTarget1).
		Bind(ebiten.Key2, ActionSelectTarget2).
		Bind(ebiten.Key3, ActionSelectTarget3).
		Bind(ebiten.KeyH, ActionThreatToggle).
		BindMod(ebiten.KeyH, ModShift, ActionThreatCycleFact).
		Bind(ebiten.KeyControlRight, ActionHealthBarToggle).
		Bind(ebiten.KeyL, ActionLayerToggle).
		BindMod(ebiten.KeyL, ModShift, ActionLayerCycleMode).
		BindMod(ebiten.KeyZ, ModCtrl, ActionUndoMove).
		BindMod(ebiten.KeyK, ModCtrl, ActionDebugKillAll).
		Bind(ebiten.KeyEscape, ActionCancel).
		BindMouse(ebiten.MouseButtonLeft, ActionMouseClick)
}

// DefaultOverworldBindings returns all overworld mode key bindings.
func DefaultOverworldBindings() *ActionMap {
	return NewActionMap("overworld").
		Bind(ebiten.KeyEscape, ActionCancel).
		Bind(ebiten.KeyN, ActionNodePlacement).
		Bind(ebiten.KeySpace, ActionEndOverworldTurn).
		Bind(ebiten.KeyEnter, ActionEndOverworldTurn).
		Bind(ebiten.KeyM, ActionOverworldMove).
		Bind(ebiten.KeyTab, ActionCycleCommander).
		Bind(ebiten.KeyI, ActionToggleInfluence).
		Bind(ebiten.KeyG, ActionGarrison).
		Bind(ebiten.KeyR, ActionRecruitCommander).
		Bind(ebiten.KeyS, ActionSquadManagement).
		Bind(ebiten.KeyE, ActionEngageThreat).
		BindMouse(ebiten.MouseButtonLeft, ActionMouseClick)
}

// DefaultSquadEditorBindings returns squad editor mode key bindings.
func DefaultSquadEditorBindings() *ActionMap {
	return NewActionMap("squad_editor").
		Bind(ebiten.KeyEscape, ActionCancel).
		Bind(ebiten.KeyU, ActionToggleUnits).
		Bind(ebiten.KeyR, ActionToggleRoster).
		Bind(ebiten.KeyN, ActionNewSquad).
		Bind(ebiten.KeyV, ActionToggleAttackPattern).
		Bind(ebiten.KeyB, ActionToggleSupportPattern).
		Bind(ebiten.KeyTab, ActionCycleCommanderEditor)
}

// DefaultArtifactBindings returns artifact mode key bindings.
func DefaultArtifactBindings() *ActionMap {
	return NewActionMap("artifact").
		Bind(ebiten.KeyEscape, ActionCancel).
		Bind(ebiten.KeyLeft, ActionPrevSquad).
		Bind(ebiten.KeyRight, ActionNextSquad).
		Bind(ebiten.KeyI, ActionTabInventory).
		Bind(ebiten.KeyE, ActionTabEquipment)
}

// DefaultNodePlacementBindings returns node placement mode key bindings.
func DefaultNodePlacementBindings() *ActionMap {
	return NewActionMap("node_placement").
		Bind(ebiten.KeyEscape, ActionCancel).
		Bind(ebiten.KeyTab, ActionCycleNodeType).
		Bind(ebiten.Key1, ActionSelectNodeType1).
		Bind(ebiten.Key2, ActionSelectNodeType2).
		Bind(ebiten.Key3, ActionSelectNodeType3).
		Bind(ebiten.Key4, ActionSelectNodeType4).
		BindMouse(ebiten.MouseButtonLeft, ActionMouseClick)
}

// DefaultRaidFloorMapBindings returns raid floor map key bindings.
func DefaultRaidFloorMapBindings() *ActionMap {
	return NewActionMap("raid_floormap").
		Bind(ebiten.Key1, ActionSelectRoom1).
		Bind(ebiten.Key2, ActionSelectRoom2).
		Bind(ebiten.Key3, ActionSelectRoom3).
		Bind(ebiten.Key4, ActionSelectRoom4).
		Bind(ebiten.Key5, ActionSelectRoom5).
		Bind(ebiten.Key6, ActionSelectRoom6).
		Bind(ebiten.Key7, ActionSelectRoom7).
		Bind(ebiten.Key8, ActionSelectRoom8).
		Bind(ebiten.Key9, ActionSelectRoom9).
		BindMouse(ebiten.MouseButtonLeft, ActionMouseClick)
}

// DefaultRaidDeployBindings returns raid deploy panel key bindings.
func DefaultRaidDeployBindings() *ActionMap {
	return NewActionMap("raid_deploy").
		Bind(ebiten.KeyEnter, ActionConfirm).
		Bind(ebiten.KeyEscape, ActionDeployBack)
}

// DefaultRaidSummaryBindings returns raid summary panel key bindings.
func DefaultRaidSummaryBindings() *ActionMap {
	return NewActionMap("raid_summary").
		Bind(ebiten.KeyEnter, ActionConfirm).
		Bind(ebiten.KeySpace, ActionDismiss)
}

// DefaultUnitPurchaseBindings returns unit purchase mode key bindings.
func DefaultUnitPurchaseBindings() *ActionMap {
	return MergeActionMaps("unit_purchase",
		CommonBindings(),
		DefaultUndoRedoBindings(),
	)
}

// DefaultCombatAnimationBindings returns combat animation mode key bindings.
func DefaultCombatAnimationBindings() *ActionMap {
	return NewActionMap("combat_animation").
		Bind(ebiten.KeySpace, ActionReplayAnimation).
		Bind(ebiten.KeyEscape, ActionCancel).
		BindMouse(ebiten.MouseButtonLeft, ActionMouseClick)
}

// DefaultCameraBindings returns camera/exploration movement key bindings.
// These use TriggerJustReleased to match the existing CameraController behavior.
func DefaultCameraBindings() *ActionMap {
	return NewActionMap("camera").
		BindRelease(ebiten.KeyW, ActionCameraMoveUp).
		BindRelease(ebiten.KeyS, ActionCameraMoveDown).
		BindRelease(ebiten.KeyA, ActionCameraMoveLeft).
		BindRelease(ebiten.KeyD, ActionCameraMoveRight).
		BindRelease(ebiten.KeyQ, ActionCameraMoveUpLeft).
		BindRelease(ebiten.KeyE, ActionCameraMoveUpRight).
		BindRelease(ebiten.KeyZ, ActionCameraMoveDownLeft).
		BindRelease(ebiten.KeyC, ActionCameraMoveDownRight).
		BindRelease(ebiten.KeyB, ActionCameraHighlight).
		BindRelease(ebiten.KeyM, ActionCameraToggleScroll)
}
