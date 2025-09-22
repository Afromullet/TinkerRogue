package input

import (
	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/gui"
	"game_main/monsters"
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
	return cc.playerData.InputStates.IsThrowing || cc.playerData.InputStates.IsShooting
}

func (cc *CombatController) OnActivate() {
	// Could initialize combat-specific state here
}

func (cc *CombatController) OnDeactivate() {
	// Could clean up combat-specific state here
}

func (cc *CombatController) HandleInput() bool {
	inputHandled := false

	// Handle ranged attack mode
	if cc.playerData.InputStates.IsShooting {
		inputHandled = cc.handleRangedAttack() || inputHandled
	}

	// Handle throwing mode
	if cc.playerData.InputStates.IsThrowing {
		inputHandled = cc.handleThrowable() || inputHandled
	}

	// Handle ranged attack initiation
	if inpututil.IsKeyJustReleased(ebiten.KeyF) {
		if !cc.playerData.InputStates.IsThrowing && cc.playerData.Equipment.EqRangedWeapon != nil {
			cc.playerSelectRangedTarget()
			cc.playerData.InputStates.HasKeyInput = true
			inputHandled = true
		}
	}

	return inputHandled
}

func (cc *CombatController) handleRangedAttack() bool {
	cc.drawRangedAttackAOE()

	// Cancel shooting
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		cc.playerData.InputStates.IsShooting = false
		cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevRangedAttInds, graphics.NewEmptyMatrix())
		return true
	}

	// Execute ranged attack
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {
		combat.RangedAttackSystem(cc.ecsManager, cc.playerData, cc.gameMap, cc.playerData.Pos)
		return true
	}

	return false
}

func (cc *CombatController) handleThrowable() bool {
	if cc.playerData.InputStates.IsShooting {
		return false
	}

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

func (cc *CombatController) playerSelectRangedTarget() {
	cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevRangedAttInds, graphics.NewEmptyMatrix())
	cc.playerData.InputStates.IsShooting = true
	cc.playerData.Equipment.PrepareRangedAttack()
}

func (cc *CombatController) drawRangedAttackAOE() {
	cursorX, cursorY := graphics.CursorPosition(*cc.playerData.Pos)

	s := cc.playerData.Equipment.RangedWeaponAOEShape
	rangedWep := common.GetComponentType[*gear.RangedWeapon](cc.playerData.Equipment.EqRangedWeapon, gear.RangedWeaponComponent)

	// Handle rotation input directly
	if baseShape, ok := rangedWep.TargetArea.(*graphics.BaseShape); ok {
		if inpututil.IsKeyJustReleased(ebiten.KeyDigit1) && baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateLeft(*baseShape.Direction)
		} else if inpututil.IsKeyJustReleased(ebiten.KeyDigit2) && baseShape.Direction != nil {
			*baseShape.Direction = graphics.RotateRight(*baseShape.Direction)
		}
	}

	var indices []int
	if cursorX != cc.sharedState.PrevCursor.X || cursorY != cc.sharedState.PrevCursor.Y {
		if cc.sharedState.PrevCursor.X != 0 && cc.sharedState.PrevCursor.Y != 0 {
			cc.gameMap.ApplyColorMatrix(cc.sharedState.PrevRangedAttInds, graphics.NewEmptyMatrix())
		}
	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {
		logicalPos := graphics.CoordManager.IndexToLogical(i)
		playerLogicalPos := graphics.LogicalPosition{X: cc.playerData.Pos.X, Y: cc.playerData.Pos.Y}

		if logicalPos.InRange(playerLogicalPos, cc.playerData.Equipment.RangedWeaponMaxDistance) {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)
		} else {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)
		}
	}

	cc.sharedState.PrevCursor.X, cc.sharedState.PrevCursor.Y = cursorX, cursorY
	cc.sharedState.PrevRangedAttInds = indices
}

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
		logicalPos := graphics.CoordManager.IndexToLogical(i)
		playerLogicalPos := graphics.LogicalPosition{X: cc.playerData.Pos.X, Y: cc.playerData.Pos.Y}

		if logicalPos.InRange(playerLogicalPos, throwable.ThrowingRange) {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)
		} else {
			cc.gameMap.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)
		}
	}

	cc.sharedState.PrevCursor.X, cc.sharedState.PrevCursor.Y = cursorX, cursorY
	cc.sharedState.PrevThrowInds = indices
}

func (cc *CombatController) applyThrowable(item *gear.Item, shape graphics.TileBasedShape, throwerPos *common.Position) {
	t := item.GetThrowableAction()
	if t == nil {
		return
	}

	// Execute the action using the new interface
	appliedEffects := t.Execute(cc.playerData.Pos, throwerPos, cc.ecsManager.World, cc.ecsManager.WorldTags)

	// Apply the effects to creatures
	for _, c := range cc.ecsManager.World.Query(cc.ecsManager.WorldTags["monsters"]) {
		curPos := c.Components[common.PositionComponent].(*common.Position)
		crea := c.Components[monsters.CreatureComponent].(*monsters.Creature)

		pos := graphics.CoordManager.GetTilePositionsAsCommon(t.Shape.GetIndices())
		for _, p := range pos {
			if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
				// Apply individual effects instead of the entire properties entity
				for _, effect := range appliedEffects {
					crea.StatEffectTracker.Add(effect)
				}
				cc.playerData.InputStates.IsThrowing = false
			}
		}
	}
}