package input

import (
	"fmt"
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
func ApplyThrowable(ecsmanager *common.EntityManager, item *equipment.Item, shape graphics.TileBasedShape, throwerPos *common.Position) {

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
		fmt.Println("Throwing range ", t.ThrowingRange)
		for _, p := range pos {

			if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
				crea.AddEffects(item.Properties)
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

// todo remove game type from function params
func HandlePlayerThrowable(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI) {

	if playerUI.IsThrowableItemSelected() {

		DrawThrowableAOE(pl, gm)

		//Press middle mouse button to throw

		//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1)
		if inpututil.IsKeyJustReleased(ebiten.KeyT) {

			log.Println("Throwing item")

			pl.ThrowPreparedItem(pl.Inv)

			ApplyThrowable(ecsmanager, pl.ThrowableItem, pl.ThrowingAOEShape, pl.Pos)

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			log.Println("Removing throwable")
			gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
			playerUI.SetThrowableItemSelected(false) //TOdo this is a problem

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

	if pl.Targeting {

		msg := common.GetComponentType[*common.UserMessage](pl.PlayerEntity, common.UsrMsg)

		msg.GameStateMessage = "Shooting"
		DrawRangedAttackAOE(pl, gm)

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			pl.Targeting = false
			gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())
			//log.Println("Removing throwable")

		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {

			combat.RangedAttackSystem(ecsmanager, pl, gm, pl.Pos)

		}

	}

}
