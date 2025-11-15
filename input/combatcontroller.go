package input

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/graphics"
	"game_main/worldmap"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type CombatController struct {
	ecsManager  *common.EntityManager
	playerData  *common.PlayerData
	gameMap     *worldmap.GameMap
	sharedState *SharedInputState
}

func NewCombatController(ecsManager *common.EntityManager, playerData *common.PlayerData,
	gameMap *worldmap.GameMap, sharedState *SharedInputState) *CombatController {
	return &CombatController{
		ecsManager:  ecsManager,
		playerData:  playerData,
		gameMap:     gameMap,
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

	// Get throwable item component directly (no wrapper)
	var item *gear.Item
	if cc.playerData.Throwables.ThrowableItemEntityID != 0 {
		item = gear.GetItemByID(cc.ecsManager.World, cc.playerData.Throwables.ThrowableItemEntityID)
	}
	if item == nil {
		return false
	}

	throwable := item.GetThrowableAction()
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

			// Get inventory component from player entity and remove thrown item
			inv := common.GetComponentTypeByID[*gear.Inventory](cc.ecsManager, cc.playerData.PlayerEntityID, gear.InventoryComponent)
			if inv != nil {
				gear.RemoveItem(cc.ecsManager.World, inv, cc.playerData.Throwables.ThrowableItemIndex)
			}

			cc.applyThrowable(item, throwable.Shape, cc.playerData.Pos)

			// Clear throwing state
			cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
			cc.gameMap.ApplyColorMatrix(indices, graphics.NewEmptyMatrix())
			cc.playerData.InputStates.IsThrowing = false
			return true
		}
	}

	// Cancel throwing
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
		cc.playerData.InputStates.IsThrowing = false
		return true
	}

	return false
}

// REMOVED: playerSelectRangedTarget - ranged weapon combat replaced by squad system

// REMOVED: drawRangedAttackAOE - ranged weapon combat replaced by squad system

func (cc *CombatController) drawThrowableAOE() {
	cursorX, cursorY := graphics.CursorPosition(*cc.playerData.Pos)

	// Type assert the interface{} to graphics.TileBasedShape
	s, ok := cc.playerData.Throwables.ThrowingAOEShape.(graphics.TileBasedShape)
	if !ok {
		return
	}

	var indices []int
	if cursorX != cc.sharedState.PrevCursor.X || cursorY != cc.sharedState.PrevCursor.Y {
		if cc.sharedState.PrevCursor.X != 0 && cc.sharedState.PrevCursor.Y != 0 {
			cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevThrowInds, graphics.NewEmptyMatrix())
		}
	}

	// Get throwable item component directly (no wrapper)
	var item *gear.Item
	if cc.playerData.Throwables.ThrowableItemEntityID != 0 {
		item = gear.GetItemByID(cc.ecsManager.World, cc.playerData.Throwables.ThrowableItemEntityID)
	}
	if item == nil {
		return
	}

	throwable := item.GetThrowableAction()
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
