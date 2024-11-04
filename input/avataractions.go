package input

import (
	"fmt"
	"game_main/avatar"
	"game_main/combat"
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/gui"
	"game_main/monsters"
	"game_main/rendering"
	"game_main/worldmap"

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
func ApplyThrowable(ecsmanager *common.EntityManager, item *gear.Item, pl *avatar.PlayerData, shape graphics.TileBasedShape, playerUI *gui.PlayerUI, throwerPos *common.Position) {

	t := item.ItemEffect(gear.THROWABLE_NAME).(*gear.Throwable)

	if t.VX != nil {

		t.VX.ResetVX()
		graphics.AddVXArea(graphics.NewVisualEffectArea(pl.Pos.X, pl.Pos.Y, t.Shape, t.VX))

	}

	//t.ReadyThrowAreaVX()
	pos := graphics.GetTilePositions(t.Shape.GetIndices(), graphics.ScreenInfo.DungeonWidth)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {

		curPos := c.Components[common.PositionComponent].(*common.Position)
		crea := c.Components[monsters.CreatureComponent].(*monsters.Creature)

		for _, p := range pos {

			if curPos.IsEqual(&p) && curPos.InRange(throwerPos, t.ThrowingRange) {
				crea.AddEffects(item.Properties)
				pl.InputStates.IsThrowing = false //Hit at least one target. Once returning, we can clear GUI elements by checking this far
			}
		}

	}

}

// todo remove game type from function params
// This changes a lot of state in different parts. Todo refactor
func HandlePlayerThrowable(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, playerUI *gui.PlayerUI) {

	if pl.InputStates.IsShooting {
		return
	}

	if pl.InputStates.IsThrowing {

		throwable := pl.Throwables.ThrowableItem.ItemEffect(gear.THROWABLE_NAME).(*gear.Throwable)

		updater := UpdateDirection(&throwable.Shape)

		throwable.Shape.UpdateShape(updater)

		gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix()) //Clears previously applied rotation if there is any
		DrawThrowableAOE(pl, gm)

		//Press middle mouse button to throw

		//if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1)
		if inpututil.IsKeyJustReleased(ebiten.KeyT) {

			if throwable.InRange(pl.Pos) {
				indices := throwable.Shape.GetIndices()

				//todo add check here that only lets someone throw if the area is in range. TileBasedShapes
				//Need a "getorigin" or "getstart" function

				pl.Throwables.RemoveThrownItem(pl.Inventory)

				ApplyThrowable(ecsmanager, pl.Throwables.ThrowableItem, pl, pl.Throwables.ThrowingAOEShape, playerUI, pl.Pos)

				// Calling this again removes the item for the GUI
				playerUI.ItemsUI.ThrowableItemDisplay.DisplayInventory()

				//Todo does not work to clear throwing GUI elements

				gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
				gm.ApplyColorMatrix(indices, graphics.NewEmptyMatrix())
				playerUI.SetThrowableItemSelected(false) //TOdo this is a problem

			}

		}

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			gm.ApplyColorMatrix(PrevThrowInds, graphics.NewEmptyMatrix())
			playerUI.SetThrowableItemSelected(false) //TOdo this is a problem
			pl.InputStates.IsThrowing = false

		}
	}

}

func HandlePlayerRangedAttack(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap) {

	if pl.InputStates.IsShooting {

		DrawRangedAttackAOE(pl, gm)

		//Cancel throwing
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {

			pl.InputStates.IsShooting = false
			gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())
			//log.Println("Removing throwable")

		}

		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton1) {

			combat.RangedAttackSystem(ecsmanager, pl, gm, pl.Pos)

		}

	}

}
func PlayerSelectRangedTarget(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	gm.ApplyColorMatrix(PrevRangedAttInds, graphics.NewEmptyMatrix())

	pl.InputStates.IsShooting = true
	pl.Equipment.PrepareRangedAttack()
	//DrawRangedAttackAOE(pl, gm) //Leaving this here for now. Not

}

// Not making this a function of worldmap.GameMap since right now only the player uses it
func IsCreatureOnTile(ecsmanager *common.EntityManager, pos *common.Position, gm *worldmap.GameMap) bool {

	index := graphics.CoordTransformer.IndexFromLogicalXY(pos.X, pos.Y)

	nextTile := gm.Tiles[index]

	if nextTile.Blocked && common.GetCreatureAtPosition(ecsmanager, pos) != nil {
		return true

	}

	return false

}

func MovePlayer(ecsmanager *common.EntityManager, pl *avatar.PlayerData, gm *worldmap.GameMap, xOffset, yOffset int) {

	nextPosition := common.Position{
		X: pl.Pos.X + xOffset,
		Y: pl.Pos.Y + yOffset,
	}

	index := graphics.CoordTransformer.IndexFromLogicalXY(nextPosition.X, nextPosition.Y)
	nextTile := gm.Tiles[index]

	index = graphics.CoordTransformer.IndexFromLogicalXY(pl.Pos.X, pl.Pos.Y)
	oldTile := gm.Tiles[index]

	if !nextTile.Blocked {

		gm.PlayerVisible.Compute(gm, pl.Pos.X, pl.Pos.Y, 8)

		pl.Pos.X = nextPosition.X
		pl.Pos.Y = nextPosition.Y
		nextTile.Blocked = true
		oldTile.Blocked = false
		fmt.Println("Moving")

	} else {
		//Determine if the tile is blocked because there's a creature
		if common.GetCreatureAtPosition(ecsmanager, &nextPosition) != nil {

			combat.MeleeAttackSystem(ecsmanager, pl, gm, pl.Pos, &nextPosition)
		}

	}

}

func PlayerPickupItem(pl *avatar.PlayerData, gm *worldmap.GameMap) {

	itemFromTile, _ := gm.RemoveItemFromTile(0, pl.Pos)

	if itemFromTile != nil {

		renderable := common.GetComponentType[*rendering.Renderable](itemFromTile, rendering.RenderableComponent)
		renderable.Visible = false
		pl.Inventory.AddItem(itemFromTile)
	}

}
