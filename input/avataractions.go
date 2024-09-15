package input

import (
	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/gui"
	"game_main/monsters"
	"game_main/worldmap"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

/*
Anything related to player actions. Functions here were split between files.
It will be easier to track what changes I'm making by putting them in one file
Not too many player actions yet. They will be broken out as they evolve.

The way player ranged attacks and throwing is handled is different from monsters.
There's information that has to be displayed to the user, such as the AOE of their attacks
That's what this file is for

*/

// Applies the throwable
func ApplyThrowable(ecsmanager *common.EntityManager, item *equipment.Item, pl *avatar.PlayerData, shape graphics.TileBasedShape, playerUI *gui.PlayerUI, throwerPos *common.Position) {

	t := item.ItemEffect(equipment.THROWABLE_NAME).(*equipment.Throwable)

	if t.VX != nil {

		t.VX.ResetVX()
		graphics.AddVXArea(graphics.NewVisualEffectArea(t.Shape, t.VX))

	}

	//t.ReadyThrowAreaVX()
	pos := common.GetTilePositions(t.Shape.GetIndices())

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

		curPos := c.Components[common.PositionComponent].(*common.Position)
		crea := c.Components[monsters.CreatureComponent].(*monsters.Creature)

		for _, p := range pos {

			if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
				crea.AddEffects(item.Properties)
				pl.IsThrowing = false //Hit at least one target. Once returning, we can clear GUI elements by checking this far
			}
		}

	}

}

func DrawThrowableAOE(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	gd := graphics.NewScreenData()
	cursorX, cursorY := ebiten.CursorPosition()

	s := pl.ThrowingAOEShape

	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())

		}

	}

	throwable := pl.ThrowableItem.ItemEffect(equipment.THROWABLE_NAME).(*equipment.Throwable)

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		pos := common.PositionFromIndex(i, gd.ScreenWidth)

		if pos.InRange(pl.Pos, throwable.ThrowingRange) {
			gm.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			gm.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevThrowInds = indices

}

// Handle rotation if the shape has a direction
func UpdateDirection(throwable *equipment.Throwable, updater *graphics.ShapeUpdater) {

	newDir := graphics.GetDirection(throwable.Shape)
	if newDir != graphics.NoDirection {

		if inpututil.IsKeyJustReleased(ebiten.KeyDigit1) {

			newDir := graphics.GetDirection(throwable.Shape)
			newDir = graphics.RotateLeft(newDir)
			updater.Direction = newDir

		} else if inpututil.IsKeyJustReleased(ebiten.KeyDigit2) {
			newDir := graphics.GetDirection(throwable.Shape)
			newDir = graphics.RotateRight(newDir)
			updater.Direction = newDir
		}

	}
}

// todo remove game type from function params
// This changes a lot of state in different parts. Todo refactor
func HandlePlayerThrowable(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI) {

	if pl.IsShooting {
		return
	}

	if pl.IsThrowing {

		throwable := pl.ThrowableItem.ItemEffect(equipment.THROWABLE_NAME).(*equipment.Throwable)

		updater := graphics.ExtractShapeParams(throwable.Shape)
		UpdateDirection(throwable, &updater)

		gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix()) //Clears previously applied rotation if there is any

		throwable.Shape.UpdateShape(updater)

		DrawThrowableAOE(pl, gm)

		//Press middle mouse button to throw

		//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1)
		if inpututil.IsKeyJustReleased(ebiten.KeyT) {

			indices := throwable.Shape.GetIndices()

			pl.ThrowPreparedItem(pl.Inv)

			ApplyThrowable(ecsmanager, pl.ThrowableItem, pl, pl.ThrowingAOEShape, playerUI, pl.Pos)

			playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory(pl.Inv)

			//Todo does not work to clear throwing GUI elements
			if !pl.IsThrowing {
				gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
				gm.ApplyColorMatrix(indices, graphics.NewEmptyMatrix())
				playerUI.SetThrowableItemSelected(false) //TOdo this is a problem

			}

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			log.Println("Removing throwable")
			gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
			playerUI.SetThrowableItemSelected(false) //TOdo this is a problem
			pl.IsThrowing = false

		}
	}

}

func DrawRangedAttackAOE(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	gd := graphics.NewScreenData()
	cursorX, cursorY := ebiten.CursorPosition()

	s := pl.RangedWeaponAOEShape
	var indices []int
	if cursorX != prevCursorX || cursorY != prevCursorY {

		if prevCursorX != 0 && prevCursorY != 0 {
			gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

		}

	}

	s.UpdatePosition(cursorX, cursorY)
	indices = s.GetIndices()

	for _, i := range indices {

		pos := common.PositionFromIndex(i, gd.ScreenWidth)

		if pos.InRange(pl.Pos, pl.RangedWeaponMaxDistance) {
			gm.ApplyColorMatrixToIndex(i, graphics.GreenColorMatrix)

		} else {

			gm.ApplyColorMatrixToIndex(i, graphics.RedColorMatrix)

		}

	}

	prevCursorX, prevCursorY = cursorX, cursorY
	PrevRangedAttInds = indices

}

func HandlePlayerRangedAttack(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap) {

	if pl.IsShooting {

		msg := common.GetComponentType[*common.UserMessage](pl.PlayerEntity, common.UsrMsg)

		msg.GameStateMessage = "Shooting"
		DrawRangedAttackAOE(pl, gm)

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			pl.IsShooting = false
			gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())
			//log.Println("Removing throwable")

		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {

			combat.RangedAttackSystem(ecsmanager, pl, gm, pl.Pos)

		}

	}

}

func MovePlayer(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, xOffset, yOffset int) {

	nextPosition := common.Position{
		X: pl.Pos.X + xOffset,
		Y: pl.Pos.Y + yOffset,
	}

	index := graphics.IndexFromXY(nextPosition.X, nextPosition.Y)
	nextTile := gm.Tiles[index]

	index = graphics.IndexFromXY(pl.Pos.X, pl.Pos.Y)
	oldTile := gm.Tiles[index]

	if !nextTile.Blocked {
		gm.PlayerVisible.Compute(gm, pl.Pos.X, pl.Pos.Y, 8)
		pl.Pos.X = nextPosition.X
		pl.Pos.Y = nextPosition.Y
		nextTile.Blocked = true
		oldTile.Blocked = false

	} else {
		//Determine if the tyle is blocked because there's a creature

		c := combat.GetCreatureAtPosition(ecsmanager, &nextPosition)

		if c != nil {

			combat.MeleeAttackSystem(ecsmanager, pl, gm, pl.Pos, &nextPosition)
		}

	}

}

func PlayerPickupItem(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	itemFromTile, _ := gm.RemoveItemFromTile(0, pl.Pos)

	if itemFromTile != nil {
		pl.Inv.AddItem(itemFromTile)
	}

}

func PlayerSelectRangedTarget(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

	pl.IsShooting = true
	pl.PrepareRangedAttack()
	DrawRangedAttackAOE(pl, gm)

}
