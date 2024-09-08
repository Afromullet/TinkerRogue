package main

import (
	"errors"
	"game_main/ecshelper"
	"game_main/graphics"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/norendren/go-fov/fov"
)

var validPositions ValidPositions

// ValidPosition stores the position of anything that a player or creature can move onto
// Todo determine whether we really need this. Figure otu why you have ValidPOsitions and solve it a better way
type ValidPositions struct {
	positions []ecshelper.Position
}

func (v *ValidPositions) Add(x int, y int) {

	v.positions = append(v.positions, ecshelper.Position{x, y})
}

func (v *ValidPositions) Get(index int) *ecshelper.Position {
	return &v.positions[index]
}

// A rect is used to create rooms on the map
type Rect struct {
	X1 int
	X2 int
	Y1 int
	Y2 int
}

func NewRect(x int, y int, width int, height int) Rect {
	return Rect{
		X1: x,
		Y1: y,
		X2: x + width,
		Y2: y + height,
	}
}

func (r *Rect) Center() (int, int) {
	centerX := (r.X1 + r.X2) / 2
	centerY := (r.Y1 + r.Y2) / 2
	return centerX, centerY
}

func (r *Rect) Intersect(other Rect) bool {
	return (r.X1 <= other.X2 && r.X2 >= other.X1 && r.Y1 <= other.Y1 && r.Y2 >= other.Y1)
}

// Holds the Map Information
type GameMap struct {
	Tiles         []*Tile
	Rooms         []Rect
	PlayerVisible *fov.View
}

func NewGameMap() GameMap {
	loadTileImages()
	validPositions = ValidPositions{
		positions: make([]ecshelper.Position, 0),
	}

	g := GameMap{}
	g.Tiles = g.createTiles()
	g.Rooms = make([]Rect, 0)
	g.PlayerVisible = fov.New()
	g.GenerateLevelTiles()

	return g
}

func (gameMap *GameMap) Tile(pos *ecshelper.Position) *Tile {

	index := graphics.IndexFromXY(pos.X, pos.Y)
	return gameMap.Tiles[index]

}

func (gameMap *GameMap) StartingPosition() ecshelper.Position {
	x, y := gameMap.Rooms[0].Center()

	return ecshelper.Position{
		X: x,
		Y: y,
	}
}

// The Entity Manager continues to track an entity when it is added to a tile.
// Since a tile has a position, we use the pos parameter to determine which tile to add it to
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *ecshelper.Position) {

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
func (gameMap *GameMap) RemoveItemFromTile(index int, pos *ecshelper.Position) (*ecs.Entity, error) {

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

// The color matrix draws on tiles.
// Right now it's only used for showing the AOE of throwable items
func (gameMap *GameMap) DrawLevel(screen *ebiten.Image) {
	gd := graphics.NewScreenData()

	var cs = ebiten.ColorScale{}

	for x := 0; x < gd.ScreenWidth; x++ {
		//for y := 0; y < gd.ScreenHeight; y++ {
		for y := 0; y < graphics.LevelHeight; y++ {

			idx := graphics.IndexFromXY(x, y)
			tile := gameMap.Tiles[idx]
			isVis := gameMap.PlayerVisible.IsVisible(x, y)

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

			screen.DrawImage(tile.Image, op)

		}
	}
}

func (gameMap *GameMap) createTiles() []*Tile {
	gd := graphics.NewScreenData()
	tiles := make([]*Tile, graphics.LevelHeight*gd.ScreenWidth)
	index := 0
	for x := 0; x < gd.ScreenWidth; x++ {
		for y := 0; y < graphics.LevelHeight; y++ {
			index = graphics.IndexFromXY(x, y)

			pos := ecshelper.Position{x, y}
			wallImg := wallImgs[GetRandomBetween(0, len(wallImgs)-1)]
			//tile := NewTile(x*gd.TileWidth, y*gd.TileHeight, pos, true, wall, WALL, false)
			tile := NewTile(x*gd.TileWidth, y*gd.TileHeight, pos, true, wallImg, WALL, false)

			tiles[index] = &tile
		}
	}
	return tiles
}

func (gameMap *GameMap) GenerateLevelTiles() {
	MIN_SIZE := 6
	MAX_SIZE := 10
	MAX_ROOMS := 30

	gd := graphics.NewScreenData()
	graphics.LevelHeight = gd.ScreenHeight - gd.UIHeight
	tiles := gameMap.createTiles()
	gameMap.Tiles = tiles
	contains_rooms := false

	for idx := 0; idx < MAX_ROOMS; idx++ {
		w := GetRandomBetween(MIN_SIZE, MAX_SIZE)
		h := GetRandomBetween(MIN_SIZE, MAX_SIZE)
		x := GetDiceRoll(gd.ScreenWidth - w - 1)
		y := GetDiceRoll(graphics.LevelHeight - h - 1)
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
				coinflip := GetDiceRoll(2)
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

			index := graphics.IndexFromXY(x, y)
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR

			//Select a random tile png
			gameMap.Tiles[index].Image = floorImgs[GetRandomBetween(0, len(floorImgs)-1)]

			validPositions.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createHorizontalTunnel(x1 int, x2 int, y int) {
	gd := graphics.NewScreenData()
	for x := min(x1, x2); x < max(x1, x2)+1; x++ {
		index := graphics.IndexFromXY(x, y)
		if index > 0 && index < gd.ScreenWidth*graphics.LevelHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR

			gameMap.Tiles[index].Image = floorImgs[GetRandomBetween(0, len(floorImgs)-1)]

			validPositions.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createVerticalTunnel(y1 int, y2 int, x int) {
	gd := graphics.NewScreenData()
	for y := min(y1, y2); y < max(y1, y2)+1; y++ {
		index := graphics.IndexFromXY(x, y)

		if index > 0 && index < gd.ScreenWidth*graphics.LevelHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR
			gameMap.Tiles[index].Image = floorImgs[GetRandomBetween(0, len(floorImgs)-1)]

			validPositions.Add(x, y)
		}
	}
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

	gameMap.Tiles[index].SetColorMatrix(m)

}

func (gameMap GameMap) InBounds(x, y int) bool {
	gd := graphics.NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > graphics.LevelHeight {
		return false
	}
	return true
}

// TODO: Change this to check for WALL, not blocked
func (gameMap GameMap) IsOpaque(x, y int) bool {
	idx := graphics.IndexFromXY(x, y)
	return gameMap.Tiles[idx].TileType == WALL
}
