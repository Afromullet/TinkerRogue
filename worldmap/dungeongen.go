package worldmap

import (
	"errors"
	"fmt"
	"game_main/common"
	"game_main/graphics"

	"game_main/randgen"
	"image/color"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/norendren/go-fov/fov"
)

var ValidPos ValidPositions

// ValidPosition stores the position of anything that a player or creature can move onto
// Todo determine whether we really need this. Figure otu why you have ValidPOsitions and solve it a better way
type ValidPositions struct {
	Pos []common.Position
}

func (v *ValidPositions) Add(x int, y int) {

	newpos := common.Position{X: x, Y: y}
	v.Pos = append(v.Pos, newpos)
}

func (v *ValidPositions) Get(index int) *common.Position {
	return &v.Pos[index]
}

// A rect is used to create rooms on the map
type Rect struct {
	X1 int
	X2 int
	Y1 int
	Y2 int
}

func (r Rect) IsInRoom(x, y int) bool {
	if x >= r.X1 && x <= r.X2 {
		if y >= r.Y1 && y <= r.Y2 {
			return true
		}
	}
	return false
}

func (r Rect) GetCoordinates() []common.Position {

	//Adding +1 and -1 so we don't get teh walls
	pos := make([]common.Position, 0)
	for y := r.Y1 + 1; y <= r.Y2-1; y++ {
		for x := r.X1 + 1; x <= r.X2-1; x++ {
			pos = append(pos, common.Position{X: x, Y: y})
		}
	}

	return pos
}

// Here temporarily for doing some basic monster spawning in the spawning package
// Do not want to spawn monsters in the center of the room
func (r Rect) GetCoordinatesWithoutCenter() []common.Position {

	//Adding +1 and -1 so we don't get teh walls
	pos := make([]common.Position, 0)
	for y := r.Y1 + 1; y <= r.Y2-1; y++ {
		for x := r.X1 + 1; x <= r.X2-1; x++ {
			centerX, centerY := r.Center()

			if centerX != x && centerY != y {
				pos = append(pos, common.Position{X: x, Y: y})
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
}

func NewGameMap() GameMap {
	loadTileImages()
	ValidPos = ValidPositions{
		Pos: make([]common.Position, 0),
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
	fmt.Println("Changing map")
	newGameMap := NewGameMap()

	//Not letting players go back up for now
	//startX, startY := newGameMap.Rooms[0].Center()
	//ind := graphics.IndexFromXY(startX, startY)
	//newGameMap.Tiles[ind].TileType = STAIRS_DOWN
	//newGameMap.Tiles[ind].image = stairs_down

	*gm = newGameMap
}

func (gameMap *GameMap) Tile(pos *common.Position) *Tile {

	index := graphics.IndexFromXY(pos.X, pos.Y)
	return gameMap.Tiles[index]

}

func (gameMap *GameMap) StartingPosition() common.Position {
	x, y := gameMap.Rooms[0].Center()

	return common.Position{
		X: x,
		Y: y,
	}
}

// The Entity Manager continues to track an entity when it is added to a tile.
// Since a tile has a position, we use the pos parameter to determine which tile to add it to
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *common.Position) {

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
func (gameMap *GameMap) RemoveItemFromTile(index int, pos *common.Position) (*ecs.Entity, error) {

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

func (gameMap *GameMap) DrawLevelSection(screen *ebiten.Image, revealAllTiles bool, pos *common.Position, size int) {

	x, y := common.PixelsFromPosition(pos, graphics.ScreenInfo.TileWidth, graphics.ScreenInfo.TileWidth)
	shape := graphics.TileSquare{
		PixelX: x,
		PixelY: y,
		Size:   size,
	}

	indices := shape.GetIndices()

	var cs = ebiten.ColorScale{}

	for idx := range indices {

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

// The color matrix draws on tiles.
// Right now it's only used for showing the AOE of throwable items
func (gameMap *GameMap) DrawLevel(screen *ebiten.Image, revealAllTiles bool) {

	var cs = ebiten.ColorScale{}

	for x := 0; x < graphics.ScreenInfo.DungeonWidth; x++ {
		//for y := 0; y < gd.ScreenHeight; y++ {
		for y := 0; y < graphics.ScreenInfo.DungeonHeight; y++ {

			idx := graphics.IndexFromXY(x, y)
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

			index = graphics.IndexFromXY(x, y)

			pos := common.Position{X: x, Y: y}
			wallImg := wallImgs[randgen.GetRandomBetween(0, len(wallImgs)-1)]
			//tile := NewTile(x*graphics.ScreenInfo.TileWidth, y*graphics.ScreenInfo.TileHeight, pos, true, wall, WALL, false)
			tile := NewTile(x*graphics.ScreenInfo.TileWidth, y*graphics.ScreenInfo.TileHeight, pos, true, wallImg, WALL, false)

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

			index := graphics.IndexFromXY(x, y)
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
		index := graphics.IndexFromXY(x, y)
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
		index := graphics.IndexFromXY(x, y)

		if index > 0 && index < graphics.ScreenInfo.DungeonWidth*graphics.ScreenInfo.DungeonHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR
			gameMap.Tiles[index].image = floorImgs[randgen.GetRandomBetween(0, len(floorImgs)-1)]

			ValidPos.Add(x, y)
		}
	}
}

// Plce the stairs in the center of a randoom room.
// The center of the room SHOULD not be blocked.
// Even if it is, that's not something to worry about now, since this is a short term approach
func (gm *GameMap) PlaceStairs() {

	//Starts at 1 so we don't create stairs in the starting room
	randRoom := randgen.GetRandomBetween(1, len(gm.Rooms)-1)

	x, y := gm.Rooms[randRoom].Center()

	ind := graphics.IndexFromXY(x, y)

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
func (gameMap GameMap) IsOpaque(x, y int) bool {
	idx := graphics.IndexFromXY(x, y)
	return gameMap.Tiles[idx].TileType == WALL
}
