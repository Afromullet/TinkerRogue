package input

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Actions the player can perform in debug mode. Currently not checking if in debug mode

func PlayerDebugActions(pl *avatar.PlayerData) {

	//teleport to cursor
	if inpututil.IsKeyJustReleased(ebiten.KeyY) {

		cursorX, cursorY := ebiten.CursorPosition()

		gridX, gridY := graphics.LogicalXYFromPixels(cursorX, cursorY)

		pl.Pos.X = gridX
		pl.Pos.Y = gridY

	}

	//teleport to cursor
	if inpututil.IsKeyJustReleased(ebiten.KeyU) {

		attr := common.GetComponentType[*common.Attributes](pl.PlayerEntity, common.AttributeComponent)
		fmt.Println("Printing Current Attributes ", attr)

	}

}
