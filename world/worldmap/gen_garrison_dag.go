package worldmap

import "game_main/common"

// Room type constants for garrison raid floors
const (
	GarrisonRoomBarracks    = "barracks"
	GarrisonRoomGuardPost   = "guard_post"
	GarrisonRoomArmory      = "armory"
	GarrisonRoomCommandPost = "command_post"
	GarrisonRoomPatrolRoute = "patrol_route"
	GarrisonRoomMageTower   = "mage_tower"
	GarrisonRoomRestRoom    = "rest_room"
	GarrisonRoomStairs      = "stairs"
)

// FloorNode represents a room in the abstract DAG
type FloorNode struct {
	ID             int
	RoomType       string
	Children       []int // IDs of downstream nodes
	Parents        []int // IDs of upstream nodes
	OnCriticalPath bool
	MinWidth       int
	MaxWidth       int
	MinHeight      int
	MaxHeight      int
}

// FloorDAG is the abstract graph for one garrison floor
type FloorDAG struct {
	Nodes        []*FloorNode
	EntryNodeID  int
	StairsNodeID int
}

// garrisonRoomSizes maps room types to {minW, maxW, minH, maxH}
var garrisonRoomSizes = map[string][4]int{
	GarrisonRoomBarracks:    {10, 14, 8, 11},
	GarrisonRoomGuardPost:   {8, 11, 7, 9},
	GarrisonRoomArmory:      {10, 13, 8, 11},
	GarrisonRoomCommandPost: {10, 13, 8, 11},
	GarrisonRoomPatrolRoute: {12, 16, 7, 9},
	GarrisonRoomMageTower:   {9, 11, 10, 12},
	GarrisonRoomRestRoom:    {7, 9, 6, 8},
	GarrisonRoomStairs:      {6, 8, 6, 8},
}

// FloorScalingEntry defines per-floor generation parameters
type FloorScalingEntry struct {
	MinCriticalPath int
	MaxCriticalPath int
	MinTotalRooms   int
	MaxTotalRooms   int
	AllowedTypes    []string
}

var garrisonFloorScaling = map[int]FloorScalingEntry{
	1: {3, 4, 6, 8, []string{
		GarrisonRoomGuardPost, GarrisonRoomBarracks, GarrisonRoomPatrolRoute, GarrisonRoomRestRoom,
		GarrisonRoomArmory,
	}},
	2: {3, 4, 7, 9, []string{
		GarrisonRoomGuardPost, GarrisonRoomBarracks, GarrisonRoomPatrolRoute, GarrisonRoomRestRoom,
		GarrisonRoomArmory,
	}},
	3: {4, 5, 8, 10, []string{
		GarrisonRoomGuardPost, GarrisonRoomBarracks, GarrisonRoomPatrolRoute, GarrisonRoomRestRoom,
		GarrisonRoomArmory, GarrisonRoomCommandPost, GarrisonRoomMageTower,
	}},
	4: {3, 4, 7, 9, []string{
		GarrisonRoomGuardPost, GarrisonRoomBarracks, GarrisonRoomPatrolRoute, GarrisonRoomRestRoom,
		GarrisonRoomArmory, GarrisonRoomCommandPost, GarrisonRoomMageTower,
	}},
	5: {3, 4, 6, 8, []string{
		GarrisonRoomGuardPost, GarrisonRoomBarracks, GarrisonRoomPatrolRoute, GarrisonRoomRestRoom,
		GarrisonRoomArmory, GarrisonRoomCommandPost, GarrisonRoomMageTower,
	}},
}

// SetGarrisonRoomSizes replaces the room size table with values from external config.
func SetGarrisonRoomSizes(sizes map[string][4]int) {
	garrisonRoomSizes = sizes
}

// SetGarrisonFloorScaling replaces the floor scaling table with values from external config.
func SetGarrisonFloorScaling(scaling map[int]FloorScalingEntry) {
	garrisonFloorScaling = scaling
}

// BuildGarrisonDAG constructs the abstract DAG for a garrison floor.
// The DAG has a critical path of combat rooms from entry to stairs,
// with optional branch chains off critical-path nodes.
func BuildGarrisonDAG(floorNumber int) *FloorDAG {
	scaling, ok := garrisonFloorScaling[floorNumber]
	if !ok {
		scaling = garrisonFloorScaling[1]
	}

	dag := &FloorDAG{
		Nodes: make([]*FloorNode, 0),
	}
	nextID := 0

	// Step 1: Build critical path
	criticalPathLen := common.GetRandomBetween(scaling.MinCriticalPath, scaling.MaxCriticalPath)

	// Entry node is always a guard post
	entryNode := newFloorNode(nextID, GarrisonRoomGuardPost, true)
	dag.Nodes = append(dag.Nodes, entryNode)
	dag.EntryNodeID = nextID
	nextID++

	// Mid-path combat rooms
	combatTypes := filterCombatTypes(scaling.AllowedTypes)
	prevNodeID := entryNode.ID
	for i := 1; i < criticalPathLen; i++ {
		roomType := combatTypes[common.RandomInt(len(combatTypes))]
		node := newFloorNode(nextID, roomType, true)
		node.Parents = append(node.Parents, prevNodeID)
		dag.Nodes[prevNodeID].Children = append(dag.Nodes[prevNodeID].Children, nextID)
		dag.Nodes = append(dag.Nodes, node)
		prevNodeID = nextID
		nextID++
	}

	// Stairs node at end of critical path
	stairsNode := newFloorNode(nextID, GarrisonRoomStairs, true)
	stairsNode.Parents = append(stairsNode.Parents, prevNodeID)
	dag.Nodes[prevNodeID].Children = append(dag.Nodes[prevNodeID].Children, nextID)
	dag.Nodes = append(dag.Nodes, stairsNode)
	dag.StairsNodeID = nextID
	nextID++

	// Step 2: Attach branch chains off critical-path nodes (not stairs)
	currentTotal := len(dag.Nodes)
	targetTotal := common.GetRandomBetween(scaling.MinTotalRooms, scaling.MaxTotalRooms)
	// Ensure target is at least current (scaling min could be less than critical path + stairs)
	if targetTotal < currentTotal {
		targetTotal = currentTotal
	}
	branchTypes := filterBranchTypes(scaling.AllowedTypes)

	criticalPathNodes := make([]int, 0)
	for _, n := range dag.Nodes {
		if n.OnCriticalPath && n.ID != dag.StairsNodeID {
			criticalPathNodes = append(criticalPathNodes, n.ID)
		}
	}

	hasRestRoom := false
	for currentTotal < targetTotal && len(criticalPathNodes) > 0 && len(branchTypes) > 0 {
		// Pick a random critical path node to branch from
		parentID := criticalPathNodes[common.RandomInt(len(criticalPathNodes))]

		// Pick room type for branch
		roomType := branchTypes[common.RandomInt(len(branchTypes))]
		if roomType == GarrisonRoomRestRoom && hasRestRoom {
			// At most 1 rest room per floor
			nonRest := filterNonRestTypes(branchTypes)
			if len(nonRest) == 0 {
				break
			}
			roomType = nonRest[common.RandomInt(len(nonRest))]
		}
		if roomType == GarrisonRoomRestRoom {
			hasRestRoom = true
		}

		branchNode := newFloorNode(nextID, roomType, false)
		branchNode.Parents = append(branchNode.Parents, parentID)
		dag.Nodes[parentID].Children = append(dag.Nodes[parentID].Children, nextID)
		dag.Nodes = append(dag.Nodes, branchNode)

		// Step 3: Optionally reconnect branch to a downstream critical-path node
		// This creates diamond/merge structures in the DAG
		downstream := findDownstreamCriticalNodes(dag, parentID)
		if len(downstream) > 0 && common.GetDiceRoll(2) == 1 {
			reconnectID := downstream[common.RandomInt(len(downstream))]
			branchNode.Children = append(branchNode.Children, reconnectID)
			dag.Nodes[reconnectID].Parents = append(dag.Nodes[reconnectID].Parents, branchNode.ID)
		}

		nextID++
		currentTotal++

		// Step 4: 40% chance to chain a second room off this branch room
		if currentTotal < targetTotal && common.GetRandomBetween(1, 10) <= 4 {
			chainType := branchTypes[common.RandomInt(len(branchTypes))]
			if chainType == GarrisonRoomRestRoom && hasRestRoom {
				nonRest := filterNonRestTypes(branchTypes)
				if len(nonRest) > 0 {
					chainType = nonRest[common.RandomInt(len(nonRest))]
				} else {
					chainType = ""
				}
			}
			if chainType != "" {
				if chainType == GarrisonRoomRestRoom {
					hasRestRoom = true
				}
				chainNode := newFloorNode(nextID, chainType, false)
				chainNode.Parents = append(chainNode.Parents, branchNode.ID)
				branchNode.Children = append(branchNode.Children, nextID)
				dag.Nodes = append(dag.Nodes, chainNode)
				nextID++
				currentTotal++
			}
		}
	}

	return dag
}

func newFloorNode(id int, roomType string, onCriticalPath bool) *FloorNode {
	sizes := garrisonRoomSizes[roomType]
	return &FloorNode{
		ID:             id,
		RoomType:       roomType,
		Children:       make([]int, 0),
		Parents:        make([]int, 0),
		OnCriticalPath: onCriticalPath,
		MinWidth:       sizes[0],
		MaxWidth:       sizes[1],
		MinHeight:      sizes[2],
		MaxHeight:      sizes[3],
	}
}

// filterCombatTypes returns allowed types that are combat rooms
func filterCombatTypes(allowed []string) []string {
	combatSet := map[string]bool{
		GarrisonRoomGuardPost:   true,
		GarrisonRoomBarracks:    true,
		GarrisonRoomArmory:      true,
		GarrisonRoomCommandPost: true,
		GarrisonRoomPatrolRoute: true,
		GarrisonRoomMageTower:   true,
	}
	result := make([]string, 0)
	for _, a := range allowed {
		if combatSet[a] {
			result = append(result, a)
		}
	}
	if len(result) == 0 {
		return []string{GarrisonRoomGuardPost}
	}
	return result
}

// filterBranchTypes returns allowed types suitable for side branches
func filterBranchTypes(allowed []string) []string {
	branchSet := map[string]bool{
		GarrisonRoomRestRoom:    true,
		GarrisonRoomArmory:      true,
		GarrisonRoomCommandPost: true,
		GarrisonRoomMageTower:   true,
		GarrisonRoomPatrolRoute: true,
	}
	result := make([]string, 0)
	for _, a := range allowed {
		if branchSet[a] {
			result = append(result, a)
		}
	}
	return result
}

// filterNonRestTypes removes rest room from the list
func filterNonRestTypes(types []string) []string {
	result := make([]string, 0)
	for _, t := range types {
		if t != GarrisonRoomRestRoom {
			result = append(result, t)
		}
	}
	return result
}

// findDownstreamCriticalNodes finds the next critical-path node directly
// downstream from the given node (excluding the stairs node and the node itself).
// Limited to only the immediate next node to prevent shortcuts that skip floors.
func findDownstreamCriticalNodes(dag *FloorDAG, nodeID int) []int {
	// Only look at direct children on the critical path
	for _, childID := range dag.Nodes[nodeID].Children {
		child := dag.Nodes[childID]
		if child.OnCriticalPath && childID != dag.StairsNodeID && childID != nodeID {
			return []int{childID}
		}
	}

	// If no direct child is on critical path, check one level deeper
	for _, childID := range dag.Nodes[nodeID].Children {
		for _, grandchildID := range dag.Nodes[childID].Children {
			gc := dag.Nodes[grandchildID]
			if gc.OnCriticalPath && grandchildID != dag.StairsNodeID && grandchildID != nodeID {
				return []int{grandchildID}
			}
		}
	}

	return nil
}

// topologicalSort returns DAG nodes in topological order (parents before children).
func topologicalSort(dag *FloorDAG) []*FloorNode {
	inDegree := make(map[int]int)
	for _, n := range dag.Nodes {
		if _, ok := inDegree[n.ID]; !ok {
			inDegree[n.ID] = 0
		}
		for _, childID := range n.Children {
			inDegree[childID]++
		}
	}

	queue := make([]int, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	sorted := make([]*FloorNode, 0, len(dag.Nodes))
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		sorted = append(sorted, dag.Nodes[nodeID])

		for _, childID := range dag.Nodes[nodeID].Children {
			inDegree[childID]--
			if inDegree[childID] == 0 {
				queue = append(queue, childID)
			}
		}
	}

	return sorted
}

// dagDepth returns the depth of each node (longest path from any root).
func dagDepth(dag *FloorDAG) map[int]int {
	depth := make(map[int]int)
	sorted := topologicalSort(dag)
	for _, node := range sorted {
		maxParentDepth := -1
		for _, parentID := range node.Parents {
			if d, ok := depth[parentID]; ok && d > maxParentDepth {
				maxParentDepth = d
			}
		}
		depth[node.ID] = maxParentDepth + 1
	}
	return depth
}
