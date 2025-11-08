package worldmap

import (
	"game_main/common"
	"game_main/coords"
)

// BSPGenerator implements Binary Space Partitioning dungeon generation
// Creates more structured, architectural layouts compared to rooms-and-corridors
type BSPGenerator struct {
	config        GeneratorConfig
	minSplitSize  int
	maxSplitDepth int
}

// NewBSPGenerator creates a new BSP tree generator
func NewBSPGenerator(config GeneratorConfig) *BSPGenerator {
	return &BSPGenerator{
		config:        config,
		minSplitSize:  15, // Minimum area size before stopping splits
		maxSplitDepth: 4,  // Maximum recursion depth
	}
}

func (g *BSPGenerator) Name() string {
	return "bsp"
}

func (g *BSPGenerator) Description() string {
	return "Binary Space Partitioning: structured architectural layouts with large rooms"
}

type BSPNode struct {
	x, y, w, h  int
	left, right *BSPNode
	room        *Rect
}

func (g *BSPGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Create root node spanning entire map
	root := &BSPNode{x: 1, y: 1, w: width - 2, h: height - 2}

	// Recursively split space
	g.splitNode(root, 0)

	// Create rooms in leaf nodes
	g.createRoomsInTree(root, &result, images)

	// Connect adjacent rooms
	g.connectRoomsInTree(root, &result, images)

	return result
}

func (g *BSPGenerator) splitNode(node *BSPNode, depth int) {
	// Stop if too deep or area too small
	if depth >= g.maxSplitDepth || node.w < g.minSplitSize || node.h < g.minSplitSize {
		return
	}

	// Decide split orientation based on area shape
	splitHorizontally := false
	if node.w > node.h && float64(node.w)/float64(node.h) >= 1.25 {
		splitHorizontally = false
	} else if node.h > node.w && float64(node.h)/float64(node.w) >= 1.25 {
		splitHorizontally = true
	} else {
		splitHorizontally = common.GetDiceRoll(2) == 1
	}

	// Calculate split position
	if splitHorizontally {
		split := common.GetRandomBetween(g.minSplitSize, node.h-g.minSplitSize)
		node.left = &BSPNode{x: node.x, y: node.y, w: node.w, h: split}
		node.right = &BSPNode{x: node.x, y: node.y + split, w: node.w, h: node.h - split}
	} else {
		split := common.GetRandomBetween(g.minSplitSize, node.w-g.minSplitSize)
		node.left = &BSPNode{x: node.x, y: node.y, w: split, h: node.h}
		node.right = &BSPNode{x: node.x + split, y: node.y, w: node.w - split, h: node.h}
	}

	// Recursively split children
	g.splitNode(node.left, depth+1)
	g.splitNode(node.right, depth+1)
}

func (g *BSPGenerator) createRoomsInTree(node *BSPNode, result *GenerationResult, images TileImageSet) {
	if node.left != nil {
		g.createRoomsInTree(node.left, result, images)
	}
	if node.right != nil {
		g.createRoomsInTree(node.right, result, images)
	}

	// Leaf node - create room
	if node.left == nil && node.right == nil {
		roomW := common.GetRandomBetween(g.config.MinRoomSize, min(g.config.MaxRoomSize, node.w-2))
		roomH := common.GetRandomBetween(g.config.MinRoomSize, min(g.config.MaxRoomSize, node.h-2))
		roomX := node.x + common.GetRandomBetween(1, node.w-roomW-1)
		roomY := node.y + common.GetRandomBetween(1, node.h-roomH-1)

		room := NewRect(roomX, roomY, roomW, roomH)
		node.room = &room

		carveRoom(result, room, images)
		result.Rooms = append(result.Rooms, room)
	}
}

func (g *BSPGenerator) connectRoomsInTree(node *BSPNode, result *GenerationResult, images TileImageSet) {
	if node.left == nil || node.right == nil {
		return
	}

	// Recursively connect children first
	g.connectRoomsInTree(node.left, result, images)
	g.connectRoomsInTree(node.right, result, images)

	// Connect left and right subtrees
	leftRoom := g.getRandomLeafRoom(node.left)
	rightRoom := g.getRandomLeafRoom(node.right)

	if leftRoom != nil && rightRoom != nil {
		x1, y1 := leftRoom.Center()
		x2, y2 := rightRoom.Center()

		// Create corridor
		carveHorizontalTunnel(result, x1, x2, y1, images)
		carveVerticalTunnel(result, y1, y2, x2, images)
	}
}

func (g *BSPGenerator) getRandomLeafRoom(node *BSPNode) *Rect {
	if node == nil {
		return nil
	}

	if node.room != nil {
		return node.room
	}

	// Randomly choose left or right subtree
	if common.GetDiceRoll(2) == 1 {
		room := g.getRandomLeafRoom(node.left)
		if room != nil {
			return room
		}
		return g.getRandomLeafRoom(node.right)
	} else {
		room := g.getRandomLeafRoom(node.right)
		if room != nil {
			return room
		}
		return g.getRandomLeafRoom(node.left)
	}
}




// Register BSP generator
func init() {
	RegisterGenerator(NewBSPGenerator(DefaultConfig()))
}
