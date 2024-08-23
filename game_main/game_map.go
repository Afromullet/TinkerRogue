package main

import (
	"errors"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/norendren/go-fov/fov"
)

type ValidPositions struct {
	positions []Position
}

func (v *ValidPositions) Add(x int, y int) {

	v.positions = append(v.positions, Position{x, y})
}

func (v *ValidPositions) Get(index int) *Position {
	return &v.positions[index]
}

var floor *ebiten.Image = nil
var wall *ebiten.Image = nil
var validPositions ValidPositions

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

type TileType int

const (
	WALL TileType = iota
	FLOOR
)

type Tile struct {
	PixelX       int
	PixelY       int
	Blocked      bool
	Image        *ebiten.Image
	tileContents TileContents
	TileType     TileType
	IsRevealed   bool
}

// Holds any entities that are on a tile, whether it's items, creatures, etc.
type TileContents struct {
	entities *[]ecs.Entity
}

// Holds the Map Information
type GameMap struct {
	Tiles         []*Tile
	Rooms         []Rect
	PlayerVisible *fov.View
}

// Creating a new Map
func NewGameMap() GameMap {
	loadTileImages()
	validPositions = ValidPositions{
		positions: make([]Position, 0),
	}

	g := GameMap{}
	g.Tiles = g.createTiles()
	g.Rooms = make([]Rect, 0)
	g.PlayerVisible = fov.New()
	g.GenerateLevelTiles()

	return g
}

func (gameMap *GameMap) GetTile(pos *Position) *Tile {

	index := GetIndexFromXY(pos.X, pos.Y)
	return gameMap.Tiles[index]

}

func (gameMap *GameMap) GetStartingPosition() Position {
	x, y := gameMap.Rooms[0].Center()

	return Position{
		X: x,
		Y: y,
	}
}

// This pointer to the entity Parameter is shared with wherever it is passed from.
// Mainly, it's there because the entity manager has to keep track of all entities
// We do not want to remove it from the manager when it's on a tile
func (gameMap *GameMap) AddEntityToTile(entity *ecs.Entity, pos *Position) {

	tile := gameMap.GetTile(pos)

	if tile.tileContents.entities == nil {
		tile.tileContents.entities = new([]ecs.Entity)
	}

	*tile.tileContents.entities = append(*tile.tileContents.entities, *entity)
}

// This removes the item at the specified index from the tile.
// Removes the entity from the tile and returns it. It still exists in the manager
// DANGER - The caller will have to store the entity somewhere else. Although is still exists in the manager,
// It will be gone from the tile when this is called.
func (gameMap *GameMap) GrabItemFromTile(index int, pos *Position) (*ecs.Entity, error) {

	tile := gameMap.GetTile(pos)

	if tile.tileContents.entities == nil {
		return nil, errors.New("entities slice is nil")
	}

	entities := *tile.tileContents.entities

	if index < 0 || index >= len(entities) {
		return nil, errors.New("index out of range")
	}

	entity := entities[index]

	*tile.tileContents.entities = append(entities[:index], entities[index+1:]...)

	return &entity, nil
}

func (gameMap *GameMap) DrawLevel(screen *ebiten.Image) {
	gd := NewScreenData()

	for x := 0; x < gd.ScreenWidth; x++ {
		//for y := 0; y < gd.ScreenHeight; y++ {
		for y := 0; y < levelHeight; y++ {

			idx := GetIndexFromXY(x, y)
			tile := gameMap.Tiles[idx]
			isVis := gameMap.PlayerVisible.IsVisible(x, y)

			if isVis {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
				screen.DrawImage(tile.Image, op)
				gameMap.Tiles[idx].IsRevealed = true
			} else if tile.IsRevealed {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))

				op.ColorM.Translate(100, 100, 100, 0.35)
				screen.DrawImage(tile.Image, op)
			}

			/*
				if gameMap.PlayerVisible.IsVisible(x, y) {
					tile := gameMap.Tiles[GetIndexFromXY(x, y)]
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(tile.PixelX), float64(tile.PixelY))
					screen.DrawImage(tile.Image, op)

				}
			*/
		}
	}
}

// createTiles creates a map of all walls as a baseline for carving out a level.
func (gameMap *GameMap) createTiles() []*Tile {
	gd := NewScreenData()
	tiles := make([]*Tile, levelHeight*gd.ScreenWidth)
	index := 0
	for x := 0; x < gd.ScreenWidth; x++ {
		for y := 0; y < levelHeight; y++ {
			index = GetIndexFromXY(x, y)

			tile := Tile{
				PixelX:     x * gd.TileWidth,
				PixelY:     y * gd.TileHeight,
				Blocked:    true,
				Image:      wall,
				TileType:   WALL,
				IsRevealed: false,
			}
			tiles[index] = &tile
		}
	}
	return tiles
}

// GenerateLevelTiles creates a new Dungeon Level Map.
func (gameMap *GameMap) GenerateLevelTiles() {
	MIN_SIZE := 6
	MAX_SIZE := 10
	MAX_ROOMS := 30

	gd := NewScreenData()
	levelHeight = gd.ScreenHeight - gd.UIHeight
	tiles := gameMap.createTiles()
	gameMap.Tiles = tiles
	contains_rooms := false

	for idx := 0; idx < MAX_ROOMS; idx++ {
		w := GetRandomBetween(MIN_SIZE, MAX_SIZE)
		h := GetRandomBetween(MIN_SIZE, MAX_SIZE)
		x := GetDiceRoll(gd.ScreenWidth - w - 1)
		y := GetDiceRoll(levelHeight - h - 1)
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

			index := GetIndexFromXY(x, y)
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR
			gameMap.Tiles[index].Image = floor
			validPositions.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createHorizontalTunnel(x1 int, x2 int, y int) {
	gd := NewScreenData()
	for x := min(x1, x2); x < max(x1, x2)+1; x++ {
		index := GetIndexFromXY(x, y)
		if index > 0 && index < gd.ScreenWidth*levelHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR

			gameMap.Tiles[index].Image = floor
			validPositions.Add(x, y)
		}
	}
}

func (gameMap *GameMap) createVerticalTunnel(y1 int, y2 int, x int) {
	gd := NewScreenData()
	for y := min(y1, y2); y < max(y1, y2)+1; y++ {
		index := GetIndexFromXY(x, y)

		if index > 0 && index < gd.ScreenWidth*levelHeight {
			gameMap.Tiles[index].Blocked = false
			gameMap.Tiles[index].TileType = FLOOR
			gameMap.Tiles[index].Image = floor
			validPositions.Add(x, y)
		}
	}
}

// GetIndexFromXY gets the index of the map array from a given X,Y TILE coordinate.
// This coordinate is logical tiles, not pixels.
func GetIndexFromXY(x int, y int) int {
	gd := NewScreenData()
	return (y * gd.ScreenWidth) + x
}

func GetIndexFromPixels(pixelX, pixelY int) int {

	gd := NewScreenData()

	gridX := pixelX / gd.TileWidth
	gridY := pixelY / gd.TileHeight

	idx := GetIndexFromXY(gridX, gridY)

	return idx
}

func GetPositionFromPixels(pixelX, pixelY int) Position {

	gd := NewScreenData()

	return Position{
		X: pixelX / gd.TileWidth,
		Y: pixelY / gd.TileHeight,
	}

}

func loadTileImages() {
	if floor != nil && wall != nil {
		return
	}
	var err error

	floor, _, err = ebitenutil.NewImageFromFile("assets//tiles/floor.png")
	if err != nil {
		log.Fatal(err)
	}

	wall, _, err = ebitenutil.NewImageFromFile("assets//tiles/wall.png")
	if err != nil {
		log.Fatal(err)
	}
}

func (gameMap GameMap) InBounds(x, y int) bool {
	gd := NewScreenData()
	if x < 0 || x > gd.ScreenWidth || y < 0 || y > levelHeight {
		return false
	}
	return true
}

// TODO: Change this to check for WALL, not blocked
func (gameMap GameMap) IsOpaque(x, y int) bool {
	idx := GetIndexFromXY(x, y)
	return gameMap.Tiles[idx].TileType == WALL
}
