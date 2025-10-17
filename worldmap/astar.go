// Package pathfinding implements the A* algorithm for finding optimal paths in the game world.
// It provides pathfinding capabilities for AI movement, player navigation assistance,
// and other game mechanics that require shortest-path calculations on the game map.
package worldmap

import (
	"errors"
	"game_main/coords"
	"game_main/graphics"

	"reflect"
)

// node represents a point in the A* pathfinding algorithm.
// g is the total distance from the start node.
// h is the estimated distance to the goal (heuristic).
// f is the total cost (g + h).
type node struct {
	Parent   *node
	Position *coords.LogicalPosition
	g        int
	h        int
	f        int
}

// isEqual compares two nodes for positional equality.
func (n *node) isEqual(other *node) bool {
	return n.Position.IsEqual(other.Position)
}

// newNode creates a new pathfinding node with the given parent and position.
// Initializes all cost values to zero.
func newNode(parent *node, position *coords.LogicalPosition) *node {
	n := node{}
	n.Parent = parent
	n.Position = position
	n.g = 0
	n.h = 0
	n.f = 0

	return &n
}

// reverseSlice reverses any slice in-place using reflection.
// Panics if the provided data is not a slice.
func reverseSlice(data interface{}) {
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		panic(errors.New("data must be a slice type"))
	}
	valueLen := value.Len()
	for i := 0; i <= int((valueLen-1)/2); i++ {
		reverseIndex := valueLen - 1 - i
		tmp := value.Index(reverseIndex).Interface()
		value.Index(reverseIndex).Set(value.Index(i))
		value.Index(i).Set(reflect.ValueOf(tmp))
	}
}

// isInSlice checks if a target node exists in a slice of nodes.
// Uses position-based equality comparison.
func isInSlice(s []*node, target *node) bool {
	for _, n := range s {
		if n.isEqual(target) {
			return true
		}
	}
	return false
}

// AStar implements the A* pathfinding algorithm.
type AStar struct{}

// GetPath finds the shortest path between start and end positions using A* algorithm.
// Returns a slice of positions representing the path, or empty slice if no path exists.
// The ignoreWalls parameter allows pathfinding through walls when true.
// TODO: gameMap should be a pointer for better performance.
func (as AStar) GetPath(gameMap GameMap, start *coords.LogicalPosition, end *coords.LogicalPosition, ignoreWalls bool) []coords.LogicalPosition {

	openList := make([]*node, 0)
	closedList := make([]*node, 0)

	//Create our starting point
	startNode := newNode(nil, start)
	startNode.g = 0
	startNode.h = 0
	startNode.f = 0

	//Create this node just for ease of dropping into our isEqual function to see if we are at the end
	endNodePlaceholder := newNode(nil, end)

	openList = append(openList, startNode)

	for {
		if len(openList) == 0 {
			break
		}
		//Get the current node
		currentNode := openList[0]
		currentIndex := 0

		//Get the node with the smallest f value
		for index, item := range openList {
			if item.f < currentNode.f {
				currentNode = item
				currentIndex = index
			}
		}

		//Move from open to closed list
		openList = append(openList[:currentIndex], openList[currentIndex+1:]...)
		closedList = append(closedList, currentNode)

		//Check to see if we reached our end
		//If so, we are done here
		if currentNode.isEqual(endNodePlaceholder) {
			path := make([]coords.LogicalPosition, 0)
			current := currentNode
			for {
				if current == nil {
					break
				}
				path = append(path, *current.Position)
				current = current.Parent
			}
			//Reverse the Path and Return it
			reverseSlice(path)
			return path
		}

		//Ok, if we are here, we are not finished yet

		edges := make([]*node, 0)
		//Now we get each node in the four cardinal directions
		//Note:  If you wish to add Diagonal movement, you can do so by getting all 8 positions
		if currentNode.Position.Y > 0 {
			logicalPos := coords.LogicalPosition{X: currentNode.Position.X, Y: currentNode.Position.Y - 1}
			tile := gameMap.Tiles[coords.CoordManager.LogicalToIndex(logicalPos)]
			if ignoreWalls || tile.TileType != WALL {
				//The location is in the map bounds and is walkable
				upNodePosition := coords.LogicalPosition{
					X: currentNode.Position.X,
					Y: currentNode.Position.Y - 1,
				}
				newNode := newNode(currentNode, &upNodePosition)
				edges = append(edges, newNode)

			}

		}
		if currentNode.Position.Y < graphics.ScreenInfo.DungeonHeight {
			logicalPos := coords.LogicalPosition{X: currentNode.Position.X, Y: currentNode.Position.Y + 1}
			tile := gameMap.Tiles[coords.CoordManager.LogicalToIndex(logicalPos)]
			if ignoreWalls || tile.TileType != WALL {
				//The location is in the map bounds and is walkable
				downNodePosition := coords.LogicalPosition{
					X: currentNode.Position.X,
					Y: currentNode.Position.Y + 1,
				}
				newNode := newNode(currentNode, &downNodePosition)
				edges = append(edges, newNode)

			}

		}
		if currentNode.Position.X > 0 {
			logicalPos := coords.LogicalPosition{X: currentNode.Position.X - 1, Y: currentNode.Position.Y}
			tile := gameMap.Tiles[coords.CoordManager.LogicalToIndex(logicalPos)]
			if ignoreWalls || tile.TileType != WALL {
				//The location is in the map bounds and is walkable
				leftNodePosition := coords.LogicalPosition{
					X: currentNode.Position.X - 1,
					Y: currentNode.Position.Y,
				}
				newNode := newNode(currentNode, &leftNodePosition)
				edges = append(edges, newNode)

			}

		}
		if currentNode.Position.X < graphics.ScreenInfo.DungeonWidth {
			logicalPos := coords.LogicalPosition{X: currentNode.Position.X + 1, Y: currentNode.Position.Y}
			tile := gameMap.Tiles[coords.CoordManager.LogicalToIndex(logicalPos)]
			if ignoreWalls && tile.TileType != WALL {
				//The location is in the map bounds and is walkable
				rightNodePosition := coords.LogicalPosition{
					X: currentNode.Position.X + 1,
					Y: currentNode.Position.Y,
				}
				newNode := newNode(currentNode, &rightNodePosition)
				edges = append(edges, newNode)

			}

		}

		//Now we iterate through the edges and put them in the open list.
		for _, edge := range edges {
			if isInSlice(closedList, edge) {
				continue
			}

			edge.g = currentNode.g + 1
			edge.h = edge.Position.ChebyshevDistance(endNodePlaceholder.Position)
			//edge.h = edge.Position.ManhattanDistance(endNodePlaceholder.Position) todo see which one oyu want to use
			edge.f = edge.g + edge.h

			if isInSlice(openList, edge) {
				//Loop through and check g values
				isFurther := false
				for _, n := range openList {
					if edge.g > n.g {
						isFurther = true
						break
					}
				}

				if isFurther {
					continue
				}

			}
			openList = append(openList, edge)
		}

	}

	return nil
}

// Creates a slice of Positions from p to other. Uses AStar to build the path
func BuildPath(gm *GameMap, start *coords.LogicalPosition, other *coords.LogicalPosition) []coords.LogicalPosition {

	astar := AStar{}
	return astar.GetPath(*gm, start, other, false)

}
