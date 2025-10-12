// Package worldmap handles game world generation, map management, and spatial operations.
// It provides dungeon generation algorithms, room management, tile systems,
// field of view calculations, and map-based entity placement and retrieval.
package worldmap

import (
	"errors"

	"game_main/common"
	"game_main/coords"
	"game_main/graphics"

	"game_main/randgen"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/norendren/go-fov/fov"
)

var ValidPos ValidPositions

// ValidPositions stores positions where players or creatures can move.
// TODO: Determine if this is still needed and find a better approach if possible.
type ValidPositions struct {
	Pos []coords.LogicalPosition
}

// Add appends a new position to the valid positions list.
func (v *ValidPositions) Add(x int, y int) {

	newpos := coords.LogicalPosition{X: x, Y: y}
	v.Pos = append(v.Pos, newpos)
}

// Get returns a pointer to the position at the specified index.
func (v *ValidPositions) Get(index int) *coords.LogicalPosition {
	return &v.Pos[index]
}

// Rect represents a rectangular room or area on the game map.
type Rect struct {
	X1 int
	X2 int
	Y1 int
	Y2 int
}

// IsInRoom checks if the given coordinates are within this room's boundaries.
func (r Rect) IsInRoom(x, y int) bool {
	if x >= r.X1 && x <= r.X2 {
		if y >= r.Y1 && y <= r.Y2 {
			return true
		}
	}
	return false
}

// GetCoordinates returns all floor positions within the room (excluding walls).
func (r Rect) GetCoordinates() []coords.LogicalPosition {

	//Adding +1 and -1 so we don't get the walls
	pos := make([]coords.LogicalPosition, 0)
	for y := r.Y1 + 1; y <= r.Y2-1; y++ {
		for x := r.X1 + 1; x <= r.X2-1; x++ {
			pos = append(pos, coords.LogicalPosition{X: x, Y: y})
		}
	}

	return pos
}

// GetCoordinatesWithoutCenter returns all floor positions except the center.
// Used for monster spawning to avoid placing monsters in room centers.
// TODO: This is a temporary solution for spawning logic.
func (r Rect) GetCoordinatesWithoutCenter() []coords.LogicalPosition {

	//Adding +1 and -1 so we don't get the walls
	pos := make([]coords.LogicalPosition, 0)
	for y := r.Y1 + 1; y <= r.Y2-1; y++ {
		for x := r.X1 + 1; x <= r.X2-1; x++ {
			centerX, centerY := r.Center()

			if centerX != x && centerY != y {
				pos = append(pos, coords.LogicalPosition{X: x, Y: y})
			}
		}
	}

	return pos
}

func (r *Rect) Center() (int, int) {
	centerX := (r.X1 + r.X2) / 2
	centerY := (r.Y1 + r.Y2) / 2
	return centerX, centerY
}

func NewRect(x int, y int, width int, height int) Rect {
	return Rect{
		X1: x,
		Y1: y,
		X2: x + width,
		Y2: y + height,
	}
}

func (r *Rect) Intersect(other Rect) bool {
	return (r.X1 <= other.X2 && r.X2 >= other.X1 && r.Y1 <= other.Y1 && r.Y2 >= other.Y1)
}

// Holds the Map Information
type GameMap struct {
	Tiles         []*Tile
	Rooms         []Rect
	PlayerVisible *fov.View
	NumTiles      int
	RightEdgeX    int
	RightEdgeY    int
}

func NewGameMap() GameMap {
	loadTileImages()
	ValidPos = ValidPositions{
		Pos: make([]coords.LogicalPosition, 0),
	}

	dungeonMap := GameMap{}
	dungeonMap.Tiles = dungeonMap.CreateTiles()
	dungeonMap.Rooms = make([]Rect, 0)
	dungeonMap.PlayerVisible = fov.New()
	dungeonMap.NumTiles = len(dungeonMap.Tiles)

	dungeonMap.GenerateLevelTiles()
	dungeonMap.PlaceStairs()

	return dungeonMap
}

// Todo need to add
// 1) Check to make sure we are at stairs
// 2) Clear all old entities. Store them somewhere else
// 3) Place the player in the starting room of the next map
// 4) Need to add Up Stairs function too
// 5) Figure out what else you have to add
func GoDownStairs(gm *GameMap) {

	//Need to remove all entities from the old map
	newGameMap := NewGameMap()

	//Not letting players go back up for now
	//startX, startY := newGameMap.Rooms[0].Center()
	//ind := graphics.IndexFromXY(startX, startY)
	//newGameMap.Tiles[ind].TileType = STAIRS_DOWN
	//newGameMap.Tiles[ind].image = stairs_down

	*gm = newGameMap
}

func (gameMap *GameMap) Tile(pos *coords.LogicalPosition) *Tile {

	logicalPos := coords.LogicalPosition{X: pos.X, Y: pos.Y}
	index := coords.CoordManager.LogicalToIndex(logicalPos)
	return gameMap.Tiles[index]

}

func (gameMap *GameMap) StartingPosition() coords.LogicalPosition {
	x, y := gameMap.Rooms[0].Center()

	return coords.LogicalPosition{
		X: x,
		Y: y,
	}
}

// The Entity Manager continues to track an entity when it is added to a tile.
// Since a tile has a position, we use the pos parameter to determine which tile to add it to
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *coords.LogicalPosition) {

	tile := gameMap.Tile(pos)

	if tile.tileContents.entities == nil {

		tile.tileContents.entities = make([]*ecs.Entity, 0)
	}

	tile.tileContents.entities = append(tile.tileContents.entities, entity)
}

// This removes the item at the specified index from the tile.
// Right now it's just used for the inventory
// The item is removed from the tile but still exists in the entity manager.
// Since this removes the item from tile.tileContents, the caller will have to store it somewhere
// Otherwise, it'll only exist in the entity manager
func (gameMap *GameMap) RemoveItemFromTile(index int, pos *coords.LogicalPosition) (*ecs.Entity, error) {

	tile := gameMap.Tile(pos)

	if tile.tileContents.entities == nil {
		return nil, errors.New("entities slice is nil")
	}

	entities := tile.tileContents.entities

	if index < 0 || index >= len(entities) {
		return nil, errors.New("index out of range")
	}

	entity := entities[index]

	tile.tileContents.entities = append(entities[:index], entities[index+1:]...)

	return entity, nil
}

func (gameMap *GameMap) DrawLevelCenteredSquare(screen *ebiten.Image, playerPos *coords.LogicalPosition, size int, revealAllTiles bool) {

	var cs = ebiten.ColorScale{}
	sq := coords.NewDrawableSection(playerPos.X, playerPos.Y, size)

	//centerOffsetX, centerOffsetY := graphics.CenterOffset(playerPos.X, playerPos.Y)

	// Get the dimensions of the screen
	gameMap.RightEdgeX = 0
	gameMap.RightEdgeY = 0
	// Initialize with smallest possible value
	// Draw the square section centered on the screen
	for x := sq.StartX; x <= sq.EndX; x++ {
		for y := sq.StartY; y <= sq.EndY; y++ {
			// Skip coordinates outside map bounds
			if x < 0 || x >= graphics.ScreenInfo.DungeonWidth || y < 0 || y >= graphics.ScreenInfo.DungeonHeight {
				continue
			}

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := gameMap.Tiles[idx]

			isVis := gameMap.PlayerVisible.IsVisible(x, y)

			if revealAllTiles {
				isVis = true
			}

			op := &ebiten.DrawImageOptions{}

			if isVis {
				tile.IsRevealed = true
			} else if tile.IsRevealed {
				// Apply color modification to darken out-of-FOV tiles
				op.ColorScale.ScaleWithColor(color.RGBA{1, 1, 1, 1})
			}

			if isVis || tile.IsRevealed {
				// Apply scaling first
				op.GeoM.Scale(float64(graphics.ScreenInfo.ScaleFactor), float64(graphics.ScreenInfo.ScaleFactor))
				offsetX, offsetY := graphics.OffsetFromCenter(playerPos.X, playerPos.Y, tile.PixelX, tile.PixelY, graphics.ScreenInfo)
				op.GeoM.Translate(offsetX, offsetY)

				// Calculate the right edge of this tile
				tileRightEdge := int(offsetX + float64(tile.image.Bounds().Dx()*graphics.ScreenInfo.ScaleFactor))
				if tileRightEdge > gameMap.RightEdgeX {
					gameMap.RightEdgeX = tileRightEdge
				}

				// Calculate the top edge of this tile
				tileTopEdge := int(offsetY)
				if tileTopEdge < gameMap.RightEdgeY {
					gameMap.RightEdgeY = tileTopEdge
				}
			}

			if !tile.cm.IsEmpty() {
				cs.SetR(tile.cm.R)
				cs.SetG(tile.cm.G)
				cs.SetB(tile.cm.B)
				cs.SetA(tile.cm.A)
				op.ColorScale.ScaleWithColorScale(cs)
			}

			screen.DrawImage(tile.image, op)
		}
	}

}

// The color matrix draws on tiles.
// Right now it's only used for showing the AOE of throwable items
func (gameMap *GameMap) DrawLevel(screen *ebiten.Image, revealAllTiles bool) {

	var cs = ebiten.ColorScale{}

	for x := 0; x < graphics.ScreenInfo.DungeonWidth; x++ {
		//for y := 0; y < gd.ScreenHeight; y++ {
		for y := 0; y < graphics.ScreenInfo.DungeonHeight; y++ {

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			idx := coords.CoordManager.LogicalToIndex(logicalPos)
			tile := gameMap.Tiles[idx]
			isVis := gameMap.PlayerVisible.IsVisible(x, y)

			if revealAllTiles {
				isVis = true
			}

			op := &ebiten.DrawImageOptions{}

			if isVis {
				op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
				gameMap.Tiles[idx].IsRevealed = true

			} else if tile.IsRevealed {

				op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))

				//Blackening out tiles that are out of Fov
				op.ColorScale.ScaleWithColor(color.RGBA{1, 1, 1, 1})

			}

			if !tile.cm.IsEmpty() {

				cs.SetR(tile.cm.R)
				cs.SetG(tile.cm.G)
				cs.SetB(tile.cm.B)
				cs.SetA(tile.cm.A)

				op.ColorScale.ScaleWithColorScale(cs)

			}

			screen.DrawImage(tile.image, op)

		}
	}
}

func (gameMap *GameMap) CreateTiles() []*Tile {

	tiles := make([]*Tile, graphics.ScreenInfo.DungeonWidth*graphics.ScreenInfo.DungeonHeight)
	index := 0

	for x := 0; x < graphics.ScreenInfo.DungeonWidth; x++ {
		for y := 0; y < graphics.ScreenInfo.DungeonHeight; y++ {

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index = coords.CoordManager.LogicalToIndex(logicalPos)

			pos := coords.LogicalPosition{X: x, Y: y}
			wallImg := wallImgs[randgen.GetRandomBetween(0, len(wallImgs)-1)]
			//tile := NewTile(x*graphics.ScreenInfo.TileWidth, y*graphics.ScreenInfo.TileHeight, pos, true, wall, WALL, false)
			tile := NewTile(x*graphics.ScreenInfo.TileSize, y*graphics.ScreenInfo.TileSize, pos, true, wallImg, WALL, false)

			tiles[index] = &tile
		}
	}
	return tiles
}

func (gameMap *GameMap) GenerateLevelTiles() {
	MIN_SIZE := 6
	MAX_SIZE := 10
	MAX_ROOMS := 30

	tiles := gameMap.CreateTiles()
	gameMap.Tiles = tiles
	contains_rooms := false

	for idx := 0; idx < MAX_ROOMS; idx++ {
		w := randgen.GetRandomBetween(MIN_SIZE, MAX_SIZE)
		h := randgen.GetRandomBetween(MIN_SIZE, MAX_SIZE)
		x := randgen.GetDiceRoll(graphics.ScreenInfo.DungeonWidth - w - 1)
		y := randgen.GetDiceRoll(graphics.ScreenInfo.DungeonHeight - h - 1)
		new_room := NewRect(x, y, w, h)

		okToAdd := true
		for _, otherRoom := range gameMap.Rooms {
			if new_room.Intersect(otherRoom) {
				okToAdd = false
				break
			}
		}

		if okToAdd {
			gameMap.createRoom(new_room)
			if contains_rooms {
				newX, newY := new_room.Center()
				prevX, prevY := gameMap.Rooms[len(gameMap.Rooms)-1].Center()
				coinflip := randgen.GetDiceRoll(2)
				if coinflip == 2 {
					gameMap.createHorizontalTunnel(prevX, newX, prevY)
					gameMap.createVerticalTunnel(prevY, newY, newX)

				} else {
					gameMap.createHorizontalTunnel(prevX, newX, newY)
					gameMap.createVerticalTunnel(prevY, newY, prevX)
				}

			}

			gameMap.Rooms = append(gameMap.Rooms, new_room)
			contains_rooms = true
		}
	}
}

func (gameMap *GameMap) createRoom(room Rect) {
	for y := room.Y1 + 1; y < room.Y2; y++ {
		for x := room.X1 + 1; x < room.X2; x++ {

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := coords.CoordManager.LogicalToIndex(logicalPos)
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR

			//Select a random tile png
			gameMap.Tiles[index].image = floorImgs[randgen.GetRandomBetween(0, len(floorImgs)-1)]

			ValidPos.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createHorizontalTunnel(x1 int, x2 int, y int) {

	for x := min(x1, x2); x < max(x1, x2)+1; x++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)
		if index > 0 && index < graphics.ScreenInfo.DungeonWidth*graphics.ScreenInfo.DungeonHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR

			gameMap.Tiles[index].image = floorImgs[randgen.GetRandomBetween(0, len(floorImgs)-1)]

			ValidPos.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createVerticalTunnel(y1 int, y2 int, x int) {

	for y := min(y1, y2); y < max(y1, y2)+1; y++ {
		logicalPos := coords.LogicalPosition{X: x, Y: y}
		index := coords.CoordManager.LogicalToIndex(logicalPos)

		if index > 0 && index < graphics.ScreenInfo.DungeonWidth*graphics.ScreenInfo.DungeonHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR
			gameMap.Tiles[index].image = floorImgs[randgen.GetRandomBetween(0, len(floorImgs)-1)]

			ValidPos.Add(x, y)
		}
	}
}

// Place the stairs in the center of a random room.
// The center of the room SHOULD not be blocked.
// Even if it is, that's not something to worry about now, since this is a short term approach
func (gm *GameMap) PlaceStairs() {

	//Starts at 1 so we don't create stairs in the starting room
	randRoom := randgen.GetRandomBetween(1, len(gm.Rooms)-1)

	x, y := gm.Rooms[randRoom].Center()

	logicalPos := coords.LogicalPosition{X: x, Y: y}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)

	gm.Tiles[ind].TileType = STAIRS_DOWN

	gm.Tiles[ind].image = stairs_down

}

// Applies the scaling ColorMatrix to the tiles at the Indices
func (gameMap *GameMap) ApplyColorMatrix(indices []int, m graphics.ColorMatrix) {

	for _, ind := range indices {

		if ind <= len(gameMap.Tiles) {
			gameMap.Tiles[ind].SetColorMatrix(m)
		}
	}

}

// Applies the scaling ColorMatrix to the tiles at the Indices
func (gameMap *GameMap) ApplyColorMatrixToIndex(index int, m graphics.ColorMatrix) {

	if index < gameMap.NumTiles {
		gameMap.Tiles[index].SetColorMatrix(m)
	}

}

func (gameMap GameMap) InBounds(x, y int) bool {

	if x < 0 || x > graphics.ScreenInfo.DungeonWidth || y < 0 || y > graphics.ScreenInfo.DungeonHeight {
		return false
	}
	return true
}

// TODO: Change this to check for WALL, not blocked
// Shouldn't this be a pointer?
func (gameMap GameMap) IsOpaque(x, y int) bool {
	logicalPos := coords.LogicalPosition{X: x, Y: y}
	idx := coords.CoordManager.LogicalToIndex(logicalPos)
	return gameMap.Tiles[idx].TileType == WALL
}

// Gets non blocked indices of a square centered at PixelX and PixelY
func (gameMap *GameMap) UnblockedIndices(pixelX, pixelY, size int) []int {
	inds := make([]int, 0)

	sq := graphics.NewSquare(pixelX, pixelY, common.NormalQuality).GetIndices()

	for _, i := range sq {

		if !gameMap.Tiles[i].Blocked {
			inds = append(inds, i)
		}

	}

	return inds

}

func (gameMap *GameMap) UnblockedLogicalCoords(pixelX, pixelY, size int) []coords.LogicalPosition {
	pos := make([]coords.LogicalPosition, 0)

	sq := graphics.NewSquare(pixelX, pixelY, common.NormalQuality).GetIndices()

	for _, i := range sq {

		if !gameMap.Tiles[i].Blocked {

			logicalPos := coords.CoordManager.IndexToLogical(i)
			x, y := logicalPos.X, logicalPos.Y
			pos = append(pos, coords.LogicalPosition{X: x, Y: y})

		}

	}

	return pos

}
