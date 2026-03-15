package guispells

import (
	"game_main/tactical/combat"
	"game_main/tactical/spells"
	"game_main/templates"
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// SpellCastingHandler manages the full spell casting workflow:
// spell list display, targeting mode, and execution.
type SpellCastingHandler struct {
	deps *SpellCastingDeps

	// AoE targeting state
	activeShape graphics.TileBasedShape
	prevIndices []int // for clearing highlights
}

// NewSpellCastingHandler creates a new spell casting handler.
func NewSpellCastingHandler(deps *SpellCastingDeps) *SpellCastingHandler {
	return &SpellCastingHandler{
		deps: deps,
	}
}

// EnterSpellMode sets the spell mode flag. The handler is the single authority
// for all InSpellMode transitions (enter, cancel, and post-cast clear).
func (h *SpellCastingHandler) EnterSpellMode() {
	h.deps.BattleState.InSpellMode = true
}

// SelectSpell validates mana and enters targeting based on spell type.
func (h *SpellCastingHandler) SelectSpell(spellID string) {
	commanderID := h.deps.Encounter.GetRosterOwnerID()
	if commanderID == 0 {
		return
	}

	if !spells.HasEnoughMana(commanderID, spellID, h.deps.ECSManager) {
		return
	}

	spell := templates.GetSpellDefinition(spellID)
	if spell == nil {
		return
	}

	h.deps.BattleState.SelectedSpellID = spellID

	if !spell.IsSingleTarget() {
		// Initialize AoE shape targeting
		h.activeShape = createAoEShape(spell)
	}
}

// IsInSpellMode returns true if spell mode is active (spell list or targeting).
func (h *SpellCastingHandler) IsInSpellMode() bool {
	return h.deps.BattleState.InSpellMode
}

// IsAoETargeting returns true if currently in AoE targeting mode.
func (h *SpellCastingHandler) IsAoETargeting() bool {
	return h.activeShape != nil
}

// HasSelectedSpell returns true if a spell has been selected for targeting.
func (h *SpellCastingHandler) HasSelectedSpell() bool {
	return h.deps.BattleState.SelectedSpellID != ""
}

// CancelSpellMode clears all spell state and overlays.
func (h *SpellCastingHandler) CancelSpellMode() {
	h.ClearOverlay()
	h.deps.BattleState.InSpellMode = false
	h.deps.BattleState.SelectedSpellID = ""
}

// GetAvailableSpells returns spells the commander can cast (checks mana).
func (h *SpellCastingHandler) GetAvailableSpells() []*templates.SpellDefinition {
	commanderID := h.deps.Encounter.GetRosterOwnerID()
	if commanderID == 0 {
		return nil
	}
	return spells.GetCastableSpells(commanderID, h.deps.ECSManager)
}

// GetAllSpells returns all spells in the commander's spellbook.
func (h *SpellCastingHandler) GetAllSpells() []*templates.SpellDefinition {
	commanderID := h.deps.Encounter.GetRosterOwnerID()
	if commanderID == 0 {
		return nil
	}
	return spells.GetAllSpells(commanderID, h.deps.ECSManager)
}

// GetCommanderMana returns the commander's current and max mana.
func (h *SpellCastingHandler) GetCommanderMana() (current, max int) {
	commanderID := h.deps.Encounter.GetRosterOwnerID()
	if commanderID == 0 {
		return 0, 0
	}
	mana := spells.GetManaData(commanderID, h.deps.ECSManager)
	if mana == nil {
		return 0, 0
	}
	return mana.CurrentMana, mana.MaxMana
}

// --- Targeting ---

// HandleSingleTargetClick processes a click during single-target spell casting.
func (h *SpellCastingHandler) HandleSingleTargetClick(mouseX, mouseY int) {
	if h.deps.PlayerPos == nil {
		return
	}

	clickedPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *h.deps.PlayerPos)
	clickedSquadID := combat.GetSquadAtPosition(clickedPos, h.deps.ECSManager)

	if clickedSquadID == 0 {
		return
	}

	encounterID := h.deps.Encounter.GetCurrentEncounterID()
	if !h.deps.Queries.IsEnemySquadInEncounter(clickedSquadID, encounterID) {
		return
	}

	h.executeSpellOnTargets([]ecs.EntityID{clickedSquadID}, nil)
}

// HandleAoETargetingFrame updates the shape overlay to follow the mouse.
func (h *SpellCastingHandler) HandleAoETargetingFrame(mouseX, mouseY int) {
	if h.activeShape == nil || h.deps.PlayerPos == nil || h.deps.GameMap == nil {
		return
	}

	// Clear previous overlay
	if len(h.prevIndices) > 0 {
		h.deps.GameMap.ApplyColorMatrix(h.prevIndices, graphics.NewEmptyMatrix())
	}

	// Convert mouse to logical via global CoordManager (correct screen dimensions),
	// then back to pixel for shape positioning.
	logicalPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *h.deps.PlayerPos)
	pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)

	// Update shape position and get indices
	h.activeShape.UpdatePosition(pixelPos.X, pixelPos.Y)
	indices := h.activeShape.GetIndices()

	// Apply spell overlay color to each tile
	spellOverlay := graphics.ColorMatrix{R: 0.8, G: 0.2, B: 0.8, A: 0.4, ApplyMatrix: true}
	for _, idx := range indices {
		if idx >= 0 && idx < h.deps.GameMap.NumTiles {
			h.deps.GameMap.ApplyColorMatrixToIndex(idx, spellOverlay)
		}
	}

	h.prevIndices = indices
}

// HandleAoEConfirmClick gathers squads in the AoE area and executes the spell.
func (h *SpellCastingHandler) HandleAoEConfirmClick(mouseX, mouseY int) {
	if h.activeShape == nil || h.deps.PlayerPos == nil {
		return
	}

	// Convert mouse to logical via global CoordManager (correct screen dimensions),
	// then back to pixel for shape positioning.
	logicalPos := graphics.MouseToLogicalPosition(mouseX, mouseY, *h.deps.PlayerPos)
	pixelPos := coords.CoordManager.LogicalToPixel(logicalPos)
	h.activeShape.UpdatePosition(pixelPos.X, pixelPos.Y)

	// Gather squads in the affected tiles
	indices := h.activeShape.GetIndices()
	squadSet := make(map[ecs.EntityID]bool)

	for _, idx := range indices {
		logicalPos := coords.CoordManager.IndexToLogical(idx)
		squadID := combat.GetSquadAtPosition(logicalPos, h.deps.ECSManager)
		if squadID != 0 && h.deps.Queries.IsEnemySquadInEncounter(squadID, h.deps.Encounter.GetCurrentEncounterID()) {
			squadSet[squadID] = true
		}
	}

	if len(squadSet) == 0 {
		return
	}

	targetIDs := make([]ecs.EntityID, 0, len(squadSet))
	for id := range squadSet {
		targetIDs = append(targetIDs, id)
	}

	h.executeSpellOnTargets(targetIDs, &logicalPos)
}

// RotateShapeLeft rotates the AoE shape counterclockwise.
func (h *SpellCastingHandler) RotateShapeLeft() {
	if baseShape, ok := h.activeShape.(*graphics.BaseShape); ok {
		if baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateLeft(*baseShape.Direction)
		}
	}
}

// RotateShapeRight rotates the AoE shape clockwise.
func (h *SpellCastingHandler) RotateShapeRight() {
	if baseShape, ok := h.activeShape.(*graphics.BaseShape); ok {
		if baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateRight(*baseShape.Direction)
		}
	}
}

// ClearOverlay removes the targeting overlay from the game map.
func (h *SpellCastingHandler) ClearOverlay() {
	if h.deps.GameMap != nil && len(h.prevIndices) > 0 {
		h.deps.GameMap.ApplyColorMatrix(h.prevIndices, graphics.NewEmptyMatrix())
	}
	h.activeShape = nil
	h.prevIndices = nil
}

// createAoEShape creates a TileBasedShape from the spell's shape definition.
func createAoEShape(spell *templates.SpellDefinition) graphics.TileBasedShape {
	if spell.Shape == nil {
		return graphics.CreateShapeFromConfig(nil)
	}
	return graphics.CreateShapeFromConfig(&graphics.ShapeConfig{
		Type:   spell.Shape.Type,
		Size:   spell.Shape.Size,
		Length: spell.Shape.Length,
		Width:  spell.Shape.Width,
		Height: spell.Shape.Height,
		Radius: spell.Shape.Radius,
	})
}

// --- Execution ---

// executeSpellOnTargets casts the selected spell on the given target squads.
// targetPos is the clicked position for AoE spells (nil for single-target).
func (h *SpellCastingHandler) executeSpellOnTargets(targetSquadIDs []ecs.EntityID, targetPos *coords.LogicalPosition) {
	spellID := h.deps.BattleState.SelectedSpellID
	if spellID == "" {
		return
	}

	spell := templates.GetSpellDefinition(spellID)
	if spell == nil {
		return
	}

	commanderID := h.deps.Encounter.GetRosterOwnerID()
	if commanderID == 0 {
		return
	}

	result := spells.ExecuteSpellCast(commanderID, spellID, targetSquadIDs, h.deps.ECSManager)

	if !result.Success {
		return
	}

	// Trigger VX at target positions
	triggerSpellVX(spell, targetSquadIDs, targetPos, h.deps)

	// Clear AoE overlay before changing mode flags
	h.ClearOverlay()

	// Set spell cast flag
	h.deps.BattleState.HasCastSpell = true

	// Clear spell mode
	h.deps.BattleState.InSpellMode = false
	h.deps.BattleState.SelectedSpellID = ""

	// Invalidate caches for affected squads
	for _, squadID := range result.AffectedSquadIDs {
		h.deps.Queries.MarkSquadDirty(squadID)
	}
	for _, squadID := range result.SquadsDestroyed {
		h.deps.Queries.InvalidateSquad(squadID)
	}
}

// triggerSpellVX creates visual effects at the target location.
// For AoE spells, targetPos is the clicked position; for single-target, it's nil and squad positions are used.
func triggerSpellVX(spell *templates.SpellDefinition, targetSquadIDs []ecs.EntityID, targetPos *coords.LogicalPosition, deps *SpellCastingDeps) {
	playerPos := deps.PlayerPos
	if playerPos == nil {
		return
	}

	// AoE: one visual effect area at the clicked position
	if spell.IsAoE() && spell.Shape != nil && targetPos != nil {
		pixelX := targetPos.X * graphics.ScreenInfo.TileSize
		pixelY := targetPos.Y * graphics.ScreenInfo.TileSize
		shape := createAoEShape(spell)
		shape.UpdatePosition(pixelX, pixelY)
		vx := graphics.CreateVisualEffectByType(spell.VXType, 0, 0, spell.VXDuration)
		area := graphics.NewVisualEffectArea(playerPos.X, playerPos.Y, shape, vx)
		graphics.AddVXArea(area)
		return
	}

	// Single target: VX at each squad's screen position
	for _, squadID := range targetSquadIDs {
		squadInfo := deps.Queries.GetSquadInfo(squadID)
		if squadInfo == nil || squadInfo.Position == nil {
			continue
		}

		pos := *squadInfo.Position
		sx, sy := coords.CoordManager.LogicalToScreen(pos, playerPos)
		vx := graphics.CreateVisualEffectByType(spell.VXType, int(sx), int(sy), spell.VXDuration)
		graphics.AddVX(vx)
	}
}
