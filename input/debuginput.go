package input

import (
	"game_main/avatar"
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
			logicalX, logicalY := graphics.TransformPixelPosition(pl.Pos.X, pl.Pos.Y, cursorX, cursorY, graphics.ScreenInfo)
			gridX = int(math.Round(float64(logicalX) / float64(graphics.ScreenInfo.TileSize)))
			gridY = int(math.Round(float64(logicalY) / float64(graphics.ScreenInfo.TileSize)))

		} else {
			pixelPos := graphics.PixelPosition{X: cursorX, Y: cursorY}
			logicalPos := graphics.CoordManager.PixelToLogical(pixelPos)
			gridX, gridY = logicalPos.X, logicalPos.Y
		}

		pl.Pos.X = gridX
		pl.Pos.Y = gridY

	}

}
