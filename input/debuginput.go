package input

import (
	"fmt"
	"game_main/avatar"
	"game_main/common"
	"game_main/graphics"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Actions the player can perform in debug mode. Currently not checking if in debug mode

func PlayerDebugActions(pl *avatar.PlayerData) {

	//teleport to cursor
	if inpututil.IsKeyJustReleased(ebiten.KeyY) {

		cursorX, cursorY := ebiten.CursorPosition()

		var gridX, gridY = 0, 0
		if graphics.MAP_SCROLLING_ENABLED {
			logicalX, logicalY := graphics.TransformPixelPosition(cursorX, cursorY, pl.Pos.X, pl.Pos.Y, graphics.ScreenInfo)
			gridX = int(math.Round(float64(logicalX) / float64(graphics.ScreenInfo.TileSize)))
			gridY = int(math.Round(float64(logicalY) / float64(graphics.ScreenInfo.TileSize)))

		} else {
			gridX, gridY = graphics.LogicalXYFromPixels(cursorX, cursorY)
		}

		pl.Pos.X = gridX
		pl.Pos.Y = gridY

	}

	//teleport to cursor
	if inpututil.IsKeyJustReleased(ebiten.KeyU) {

		attr := common.GetComponentType[*common.Attributes](pl.PlayerEntity, common.AttributeComponent)
		fmt.Println("Printing Current Attributes ", attr)

	}

}
