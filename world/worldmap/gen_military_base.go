package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
)

// MilitaryBaseGenerator creates structured outdoor military base layouts.
// Produces a perimeter wall with gate, guard towers, drill yard, and supply areas.
type MilitaryBaseGenerator struct {
	config MilitaryBaseConfig
}

// MilitaryBaseConfig holds tuning parameters for military base generation.
type MilitaryBaseConfig struct {
	Biome             Biome   // Tileset biome (default BiomeGrassland)
	PerimeterInset    int     // Tiles from map edge to perimeter wall
	WallThickness     int     // Perimeter wall thickness
	GateWidth         int     // Gate opening width in tiles
	GateSide          int     // 0=south, 1=north, 2=east, 3=west
	NumGuardTowers    int     // Guard towers to place (corners first)
	GuardTowerSize    int     // Tower footprint NxN
	DrillYardMinRatio float64 // Min fraction of interior for drill yard
	NumSupplyAreas    int     // Supply/armory zones
	SupplyAreaMinSize int     // Min dimension of supply zone
	SupplyAreaMaxSize int     // Max dimension of supply zone
	CoverDensity      float64 // Sparse cover in drill yard
	NumPOIScatter     int     // Decorative town POI tiles
	BorderThickness   int     // Solid wall border around map edges
}

// DefaultMilitaryBaseConfig returns tuned defaults for military base generation.
func DefaultMilitaryBaseConfig() MilitaryBaseConfig {
	return MilitaryBaseConfig{
		Biome:             BiomeGrassland,
		PerimeterInset:    6,
		WallThickness:     1,
		GateWidth:         4,
		GateSide:          0, // south
		NumGuardTowers:    4,
		GuardTowerSize:    3,
		DrillYardMinRatio: 0.30,
		NumSupplyAreas:    2,
		SupplyAreaMinSize: 5,
		SupplyAreaMaxSize: 8,
		CoverDensity:      0.02,
		NumPOIScatter:     5,
		BorderThickness:   2,
	}
}

// NewMilitaryBaseGenerator creates a new military base generator with the given config.
func NewMilitaryBaseGenerator(config MilitaryBaseConfig) *MilitaryBaseGenerator {
	return &MilitaryBaseGenerator{config: config}
}

func (g *MilitaryBaseGenerator) Name() string { return "military_base" }
func (g *MilitaryBaseGenerator) Description() string {
	return "Structured outdoor military base with perimeter wall, gate, guard towers, and drill yard"
}

func (g *MilitaryBaseGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	cfg := g.config
	terrainMap := make([]bool, width*height)

	// Step 1: Initialize — all walkable except border
	g.initializeTerrain(terrainMap, width, height, cfg.BorderThickness)

	// Step 2: Build perimeter wall
	perimeterRect := g.buildPerimeterWall(terrainMap, width, height, cfg)

	// Step 3: Carve gate
	g.carveGate(terrainMap, width, height, cfg, perimeterRect)

	// Step 4: Place guard towers
	g.placeGuardTowers(terrainMap, width, height, cfg, perimeterRect)

	// Step 5: Place supply areas
	supplyRects := g.placeSupplyAreas(terrainMap, width, height, cfg, perimeterRect)

	// Step 6: Scatter drill yard cover
	drillYardRect := g.getDrillYardRect(cfg, perimeterRect)
	g.scatterDrillYardCover(terrainMap, width, height, cfg, drillYardRect)

	// Step 7: Ensure connectivity
	ensureTerrainConnectivity(terrainMap, width, height)

	// Step 8: Scatter decorative POIs
	poiPositions := g.scatterPOIs(terrainMap, width, height, cfg, perimeterRect)

	// Step 9: Place faction starts
	factionStarts := g.placeFactionStarts(terrainMap, width, height, cfg, perimeterRect)

	// Step 10: Convert to tiles
	convertTerrainMapToTiles(&result, terrainMap, width, height, images, cfg.Biome)

	// Place POI images on tiles
	for _, pos := range poiPositions {
		idx := positionToIndex(pos.X, pos.Y, width)
		if poiImg, ok := images.POIImages[POITown]; ok {
			result.Tiles[idx].POIType = POITown
			result.Tiles[idx].Image = poiImg
		}
	}

	// Step 11: Record rooms
	result.Rooms = append(result.Rooms, perimeterRect)
	result.Rooms = append(result.Rooms, drillYardRect)
	for _, sr := range supplyRects {
		result.Rooms = append(result.Rooms, sr)
	}

	// Record faction start positions
	for i, pos := range factionStarts {
		result.FactionStartPositions = append(result.FactionStartPositions, FactionStartPosition{
			Position: pos,
			Biome:    cfg.Biome,
			Sector:   i,
		})
	}

	return result
}

// initializeTerrain sets all tiles walkable except the border edges.
func (g *MilitaryBaseGenerator) initializeTerrain(terrainMap []bool, width, height, border int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < border || x >= width-border || y < border || y >= height-border {
				terrainMap[y*width+x] = false
			} else {
				terrainMap[y*width+x] = true
			}
		}
	}
}

// buildPerimeterWall stamps a wall rectangle inset from the map edge.
// Returns the perimeter rect (inner edge of the wall).
func (g *MilitaryBaseGenerator) buildPerimeterWall(terrainMap []bool, width, height int, cfg MilitaryBaseConfig) Rect {
	inset := cfg.PerimeterInset
	thick := cfg.WallThickness

	// Outer edge of perimeter
	ox1 := inset
	oy1 := inset
	ox2 := width - inset - 1
	oy2 := height - inset - 1

	// Stamp wall on all 4 sides
	for y := oy1; y <= oy2; y++ {
		for x := ox1; x <= ox2; x++ {
			// Check if this tile is within the wall thickness of any edge
			inTop := y >= oy1 && y < oy1+thick
			inBottom := y > oy2-thick && y <= oy2
			inLeft := x >= ox1 && x < ox1+thick
			inRight := x > ox2-thick && x <= ox2

			if inTop || inBottom || inLeft || inRight {
				terrainMap[y*width+x] = false
			}
		}
	}

	// Return the interior rect (inside the wall)
	return NewRect(ox1+thick, oy1+thick, (ox2-thick)-(ox1+thick), (oy2-thick)-(oy1+thick))
}

// carveGate opens a gate on the specified side and creates a 3-tile approach corridor outside.
func (g *MilitaryBaseGenerator) carveGate(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, perimRect Rect) {
	inset := cfg.PerimeterInset
	thick := cfg.WallThickness
	gw := cfg.GateWidth

	switch cfg.GateSide {
	case 0: // South
		centerX := (perimRect.X1 + perimRect.X2) / 2
		gateStart := centerX - gw/2
		wallY := height - inset - 1
		// Carve through wall
		for dy := 0; dy < thick; dy++ {
			for x := gateStart; x < gateStart+gw; x++ {
				if x >= 0 && x < width && wallY-dy >= 0 {
					terrainMap[(wallY-dy)*width+x] = true
				}
			}
		}
		// Approach corridor outside wall
		for dy := 1; dy <= 3; dy++ {
			for x := gateStart; x < gateStart+gw; x++ {
				y := wallY + dy
				if x >= 0 && x < width && y >= 0 && y < height {
					terrainMap[y*width+x] = true
				}
			}
		}

	case 1: // North
		centerX := (perimRect.X1 + perimRect.X2) / 2
		gateStart := centerX - gw/2
		wallY := inset
		for dy := 0; dy < thick; dy++ {
			for x := gateStart; x < gateStart+gw; x++ {
				if x >= 0 && x < width && wallY+dy < height {
					terrainMap[(wallY+dy)*width+x] = true
				}
			}
		}
		for dy := 1; dy <= 3; dy++ {
			for x := gateStart; x < gateStart+gw; x++ {
				y := wallY - dy
				if x >= 0 && x < width && y >= 0 && y < height {
					terrainMap[y*width+x] = true
				}
			}
		}

	case 2: // East
		centerY := (perimRect.Y1 + perimRect.Y2) / 2
		gateStart := centerY - gw/2
		wallX := width - inset - 1
		for dx := 0; dx < thick; dx++ {
			for y := gateStart; y < gateStart+gw; y++ {
				if y >= 0 && y < height && wallX-dx >= 0 {
					terrainMap[y*width+(wallX-dx)] = true
				}
			}
		}
		for dx := 1; dx <= 3; dx++ {
			for y := gateStart; y < gateStart+gw; y++ {
				x := wallX + dx
				if x >= 0 && x < width && y >= 0 && y < height {
					terrainMap[y*width+x] = true
				}
			}
		}

	case 3: // West
		centerY := (perimRect.Y1 + perimRect.Y2) / 2
		gateStart := centerY - gw/2
		wallX := inset
		for dx := 0; dx < thick; dx++ {
			for y := gateStart; y < gateStart+gw; y++ {
				if y >= 0 && y < height && wallX+dx < width {
					terrainMap[y*width+(wallX+dx)] = true
				}
			}
		}
		for dx := 1; dx <= 3; dx++ {
			for y := gateStart; y < gateStart+gw; y++ {
				x := wallX - dx
				if x >= 0 && x < width && y >= 0 && y < height {
					terrainMap[y*width+x] = true
				}
			}
		}
	}
}

// placeGuardTowers places NxN wall clusters at perimeter corners (and midpoints if >4).
func (g *MilitaryBaseGenerator) placeGuardTowers(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, perimRect Rect) {
	inset := cfg.PerimeterInset
	ts := cfg.GuardTowerSize

	// Outer edge of perimeter wall
	ox1 := inset
	oy1 := inset
	ox2 := width - inset - 1
	oy2 := height - inset - 1

	// Corner positions (top-left of each tower)
	corners := [][2]int{
		{ox1 - ts/2, oy1 - ts/2},             // top-left
		{ox2 - ts/2, oy1 - ts/2},             // top-right
		{ox1 - ts/2, oy2 - ts/2},             // bottom-left
		{ox2 - ts/2, oy2 - ts/2},             // bottom-right
	}

	placeTower := func(tx, ty int) {
		for dy := 0; dy < ts; dy++ {
			for dx := 0; dx < ts; dx++ {
				nx, ny := tx+dx, ty+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					terrainMap[ny*width+nx] = false
				}
			}
		}
	}

	numToPlace := cfg.NumGuardTowers
	if numToPlace > len(corners) {
		numToPlace = len(corners)
	}
	for i := 0; i < numToPlace; i++ {
		placeTower(corners[i][0], corners[i][1])
	}

	// Additional towers at midpoints if requested
	if cfg.NumGuardTowers > 4 {
		midpoints := [][2]int{
			{(ox1 + ox2) / 2 - ts/2, oy1 - ts/2},     // top-mid
			{(ox1 + ox2) / 2 - ts/2, oy2 - ts/2},     // bottom-mid
			{ox1 - ts/2, (oy1 + oy2) / 2 - ts/2},     // left-mid
			{ox2 - ts/2, (oy1 + oy2) / 2 - ts/2},     // right-mid
		}
		extra := cfg.NumGuardTowers - 4
		if extra > len(midpoints) {
			extra = len(midpoints)
		}
		for i := 0; i < extra; i++ {
			placeTower(midpoints[i][0], midpoints[i][1])
		}
	}
}

// placeSupplyAreas creates rectangular supply zones in interior corners opposite the gate.
// Each supply area contains barricade rows and 2x2 crate clusters.
func (g *MilitaryBaseGenerator) placeSupplyAreas(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, perimRect Rect) []Rect {
	supplyRects := make([]Rect, 0, cfg.NumSupplyAreas)

	// Choose corners opposite the gate for supply areas
	type corner struct{ x, y int }
	var candidates []corner

	switch cfg.GateSide {
	case 0: // Gate south → supply in top corners
		candidates = []corner{
			{perimRect.X1 + 1, perimRect.Y1 + 1},
			{perimRect.X2 - cfg.SupplyAreaMaxSize - 1, perimRect.Y1 + 1},
		}
	case 1: // Gate north → supply in bottom corners
		candidates = []corner{
			{perimRect.X1 + 1, perimRect.Y2 - cfg.SupplyAreaMaxSize - 1},
			{perimRect.X2 - cfg.SupplyAreaMaxSize - 1, perimRect.Y2 - cfg.SupplyAreaMaxSize - 1},
		}
	case 2: // Gate east → supply in left corners
		candidates = []corner{
			{perimRect.X1 + 1, perimRect.Y1 + 1},
			{perimRect.X1 + 1, perimRect.Y2 - cfg.SupplyAreaMaxSize - 1},
		}
	case 3: // Gate west → supply in right corners
		candidates = []corner{
			{perimRect.X2 - cfg.SupplyAreaMaxSize - 1, perimRect.Y1 + 1},
			{perimRect.X2 - cfg.SupplyAreaMaxSize - 1, perimRect.Y2 - cfg.SupplyAreaMaxSize - 1},
		}
	}

	for i := 0; i < cfg.NumSupplyAreas && i < len(candidates); i++ {
		sw := common.GetRandomBetween(cfg.SupplyAreaMinSize, cfg.SupplyAreaMaxSize)
		sh := common.GetRandomBetween(cfg.SupplyAreaMinSize, cfg.SupplyAreaMaxSize)

		sx := candidates[i].x
		sy := candidates[i].y

		// Clamp to interior
		if sx+sw > perimRect.X2-1 {
			sw = perimRect.X2 - 1 - sx
		}
		if sy+sh > perimRect.Y2-1 {
			sh = perimRect.Y2 - 1 - sy
		}
		if sw < cfg.SupplyAreaMinSize || sh < cfg.SupplyAreaMinSize {
			continue
		}

		supplyRect := NewRect(sx, sy, sw, sh)
		supplyRects = append(supplyRects, supplyRect)

		// Place barricade rows (1-wide wall segments, 2-3 tiles long)
		numBarricades := common.GetRandomBetween(2, 3)
		for b := 0; b < numBarricades; b++ {
			bx := common.GetRandomBetween(sx+1, sx+sw-3)
			by := common.GetRandomBetween(sy+1, sy+sh-2)
			bLen := common.GetRandomBetween(2, 3)
			// Horizontal barricade
			for dx := 0; dx < bLen; dx++ {
				nx := bx + dx
				if nx >= 0 && nx < width && by >= 0 && by < height {
					terrainMap[by*width+nx] = false
				}
			}
		}

		// Place 2x2 crate clusters with 2-tile gaps
		numCrates := common.GetRandomBetween(1, 2)
		for c := 0; c < numCrates; c++ {
			cx := common.GetRandomBetween(sx+1, sx+sw-3)
			cy := common.GetRandomBetween(sy+1, sy+sh-3)
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					nx, ny := cx+dx, cy+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						terrainMap[ny*width+nx] = false
					}
				}
			}
		}
	}

	return supplyRects
}

// getDrillYardRect returns the central drill yard area inside the perimeter.
func (g *MilitaryBaseGenerator) getDrillYardRect(cfg MilitaryBaseConfig, perimRect Rect) Rect {
	interiorW := perimRect.X2 - perimRect.X1
	interiorH := perimRect.Y2 - perimRect.Y1

	// Drill yard is the central portion of the interior
	marginX := interiorW / 5
	marginY := interiorH / 5

	return NewRect(
		perimRect.X1+marginX,
		perimRect.Y1+marginY,
		interiorW-2*marginX,
		interiorH-2*marginY,
	)
}

// scatterDrillYardCover places sparse 2x2 wall pillars in the drill yard with minimum spacing.
func (g *MilitaryBaseGenerator) scatterDrillYardCover(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, drillRect Rect) {
	drillArea := (drillRect.X2 - drillRect.X1) * (drillRect.Y2 - drillRect.Y1)
	numPillars := int(float64(drillArea) * cfg.CoverDensity)
	if numPillars < 1 {
		numPillars = 1
	}

	minSpacing := 4
	placed := make([][2]int, 0, numPillars)

	for attempt := 0; attempt < numPillars*20 && len(placed) < numPillars; attempt++ {
		px := common.GetRandomBetween(drillRect.X1+1, drillRect.X2-3)
		py := common.GetRandomBetween(drillRect.Y1+1, drillRect.Y2-3)

		if isTooCloseToAny(px, py, placed, minSpacing) {
			continue
		}

		if tryPlace2x2PillarOnTerrain(terrainMap, px, py, width, height) {
			placed = append(placed, [2]int{px, py})
		}
	}
}

// scatterPOIs places decorative town POI tiles at random walkable positions inside the perimeter.
func (g *MilitaryBaseGenerator) scatterPOIs(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, perimRect Rect) []coords.LogicalPosition {
	positions := make([]coords.LogicalPosition, 0, cfg.NumPOIScatter)
	placedCoords := make([][2]int, 0, cfg.NumPOIScatter)
	minSpacing := 4

	for attempt := 0; attempt < cfg.NumPOIScatter*30 && len(positions) < cfg.NumPOIScatter; attempt++ {
		x := common.GetRandomBetween(perimRect.X1+2, perimRect.X2-2)
		y := common.GetRandomBetween(perimRect.Y1+2, perimRect.Y2-2)

		if x < 0 || x >= width || y < 0 || y >= height {
			continue
		}
		if !terrainMap[y*width+x] {
			continue
		}

		// Check minimum spacing from other POIs
		if isTooCloseToAny(x, y, placedCoords, minSpacing) {
			continue
		}

		positions = append(positions, coords.LogicalPosition{X: x, Y: y})
		placedCoords = append(placedCoords, [2]int{x, y})
	}

	return positions
}

// placeFactionStarts places attacker outside the gate and defender inside the base rear.
func (g *MilitaryBaseGenerator) placeFactionStarts(terrainMap []bool, width, height int, cfg MilitaryBaseConfig, perimRect Rect) []coords.LogicalPosition {
	var attackerRegion, defenderRegion [4]int // xMin, xMax, yMin, yMax

	inset := cfg.PerimeterInset

	switch cfg.GateSide {
	case 0: // South gate: attacker below wall, defender in top half
		attackerRegion = [4]int{perimRect.X1, perimRect.X2, height - inset + 1, height - cfg.BorderThickness - 1}
		defenderRegion = [4]int{perimRect.X1 + 2, perimRect.X2 - 2, perimRect.Y1 + 1, (perimRect.Y1 + perimRect.Y2) / 2}
	case 1: // North gate: attacker above wall, defender in bottom half
		attackerRegion = [4]int{perimRect.X1, perimRect.X2, cfg.BorderThickness, inset - 1}
		defenderRegion = [4]int{perimRect.X1 + 2, perimRect.X2 - 2, (perimRect.Y1 + perimRect.Y2) / 2, perimRect.Y2 - 1}
	case 2: // East gate: attacker right of wall, defender in left half
		attackerRegion = [4]int{width - inset + 1, width - cfg.BorderThickness - 1, perimRect.Y1, perimRect.Y2}
		defenderRegion = [4]int{perimRect.X1 + 1, (perimRect.X1 + perimRect.X2) / 2, perimRect.Y1 + 2, perimRect.Y2 - 2}
	case 3: // West gate: attacker left of wall, defender in right half
		attackerRegion = [4]int{cfg.BorderThickness, inset - 1, perimRect.Y1, perimRect.Y2}
		defenderRegion = [4]int{(perimRect.X1 + perimRect.X2) / 2, perimRect.X2 - 1, perimRect.Y1 + 2, perimRect.Y2 - 2}
	}

	scanRadius := 5
	attacker := findBestOpenPosition(terrainMap, width, height, attackerRegion[0], attackerRegion[1], attackerRegion[2], attackerRegion[3], scanRadius)
	defender := findBestOpenPosition(terrainMap, width, height, defenderRegion[0], defenderRegion[1], defenderRegion[2], defenderRegion[3], scanRadius)

	var starts []coords.LogicalPosition
	if attacker.X >= 0 && defender.X >= 0 {
		starts = append(starts, attacker, defender)
	}
	return starts
}


func init() {
	RegisterGenerator(&MilitaryBaseGenerator{config: DefaultMilitaryBaseConfig()})
}
