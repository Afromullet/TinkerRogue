package input

import (
	"game_main/common"
	"game_main/coords"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Actions the player can perform in debug mode. Currently not checking if in debug mode

func PlayerDebugActions(pl *common.PlayerData) {

	//teleport to cursor
	if inpututil.IsKeyJustReleased(ebiten.KeyY) {

		cursorX, cursorY := ebiten.CursorPosition()

		// Use unified coordinate transformation - handles scrolling mode automatically
		centerPos := coords.LogicalPosition{X: pl.Pos.X, Y: pl.Pos.Y}
		logicalPos := coords.CoordManager.ScreenToLogical(cursorX, cursorY, &centerPos)

		pl.Pos.X = logicalPos.X
		pl.Pos.Y = logicalPos.Y

	}

}
