package input

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"
	"game_main/gui"

	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type CombatController struct {
	ecsManager  *common.EntityManager
	playerData  *avatar.PlayerData
	gameMap     *worldmap.GameMap
	playerUI    *gui.PlayerUI
	sharedState *SharedInputState
}

func NewCombatController(ecsManager *common.EntityManager, playerData *avatar.PlayerData,
	gameMap *worldmap.GameMap, playerUI *gui.PlayerUI, sharedState *SharedInputState) *CombatController {
	return &CombatController{
		ecsManager:  ecsManager,
		playerData:  playerData,
		gameMap:     gameMap,
		playerUI:    playerUI,
		sharedState: sharedState,
	}
}

func (cc *CombatController) CanHandle() bool {
	return cc.playerData.InputStates.IsThrowing
	// Ranged weapon shooting removed - squad system handles combat
}

func (cc *CombatController) OnActivate() {
	// Could initialize combat-specific state here
}

func (cc *CombatController) OnDeactivate() {
	// Could clean up combat-specific state here
}

func (cc *CombatController) HandleInput() bool {
	inputHandled := false

	// Ranged attack mode removed - squad system handles combat

	// Handle throwing mode
	if cc.playerData.InputStates.IsThrowing {
		inputHandled = cc.handleThrowable() || inputHandled
	}

	// Ranged attack initiation removed - squad system handles combat

	return inputHandled
}

// REMOVED: handleRangedAttack - ranged weapon combat replaced by squad system

func (cc *CombatController) handleThrowable() bool {
	// IsShooting check removed - squad system handles combat

	throwable := cc.playerData.Throwables.ThrowableItem.GetThrowableAction()
	if throwable == nil {
		return false
	}

	// Handle rotation input directly for throwable shapes
	if baseShape, ok := throwable.Shape.(*graphics.BaseShape); ok {
		if inpututil.IsKeyJustReleased(ebiten.KeyDigit1) && baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateLeft(*baseShape.Direction)
		} else if inpututil.IsKeyJustReleased(ebiten.KeyDigit2) && baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateRight(*baseShape.Direction)
		}
	}

	cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
	cc.drawThrowableAOE()

	// Execute throw
	if inpututil.IsKeyJustReleased(ebiten.KeyT) {
		if throwable.InRange(cc.playerData.Pos) {
			indices := throwable.Shape.GetIndices()

			cc.playerData.Throwables.RemoveThrownItem(cc.playerData.Inventory)
			cc.applyThrowable(cc.playerData.Throwables.ThrowableItem, throwable.Shape, cc.playerData.Pos)

			cc.playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()
			cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
			cc.gameMap.ApplyColorMatrix(indices, graphics.NewEmptyMatrix())
			cc.playerUI.SetThrowableItemSelected(false)
			return true
		}
	}

	// Cancel throwing
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
		cc.playerUI.SetThrowableItemSelected(false)
		cc.playerData.InputStates.IsThrowing = false
		return true
	}

	return false
}

// REMOVED: playerSelectRangedTarget - ranged weapon combat replaced by squad system

// REMOVED: drawRangedAttackAOE - ranged weapon combat replaced by squad system

func (cc *CombatController) drawThrowableAOE() {
	cursorX, cursorY := graphics.CursorPosition(*cc.playerData.Pos)

	s := cc.playerData.Throwables.ThrowingAOEShape

	var indices []int
	if cursorX != cc.sharedState.PrevCursor.X || cursorY != cc.sharedState.PrevCursor.Y {
		if cc.sharedState.PrevCursor.X != 0 && cc.sharedState.PrevCursor.Y != 0 {
			cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
		}
	}

	throwable := cc.playerData.Throwables.ThrowableItem.GetThrowableAction()
	if throwable == nil {
		return
	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {
		logicalPos := coords.CoordManager.IndexToLogical(i)
		playerLogicalPos := coords.LogicalPosition{X: cc.playerData.Pos.X, Y: cc.playerData.Pos.Y}

		if logicalPos.InRange(&playerLogicalPos, throwable.ThrowingRange) {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)
		} else {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)
		}
	}

	cc.sharedState.PrevCursor.X, cc.sharedState.PrevCursor.Y = cursorX, cursorY
	cc.sharedState.PrevThrowInds = indices
}

func (cc *CombatController) applyThrowable(item *gear.Item, shape graphics.TileBasedShape, throwerPos *coords.LogicalPosition) {
	t := item.GetThrowableAction()
	if t == nil {
		return
	}

	// Execute the action using the new interface
	//appliedEffects := t.Execute(cc.playerData.Pos, throwerPos, cc.ecsManager.World, cc.ecsManager.WorldTags)

	//cc.playerData.InputStates.IsThrowing = false
	// TODO, for future. Eventually apply effects to a squad

	/*
		for _, c := range cc.ecsManager.World.Query(cc.ecsManager.WorldTags["monsters"]) {
			curPos := c.Components[common.PositionComponent].(*coords.LogicalPosition)

			pos := coords.CoordManager.GetTilePositionsAsCommon(t.Shape.GetIndices())
			for _, p := range pos {
				if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
					// Apply individual effects instead of the entire properties entity
					//for _, effect := range appliedEffects {
					//crea.StatEffectTracker.Add(effect)
					//	}
					cc.playerData.InputStates.IsThrowing = false
				}
			}

		}
	*/
}
