package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
	"math"
	"time"

	opensimplex "github.com/ojrac/opensimplex-go"
)

// CavernGenerator creates organic cave layouts using noise-distorted chambers,
// drunkard's walk tunnels, and two-phase cellular automata.
// Designed for enclosed, tactical cave combat with varied tunnel widths and chokepoints.
type CavernGenerator struct {
	config CavernConfig
}

// CavernConfig holds tuning parameters for cavern generation.
type CavernConfig struct {
	FillDensity       float64 // Wall density for random fill outside chambers (0.62 = 38% open)
	NumChambers       int     // Target number of seeded chambers (used for drop logic)
	MinChamberRadius  int     // Minimum chamber bounding radius
	MaxChamberRadius  int     // Maximum chamber bounding radius
	NoiseScale        float64 // OpenSimplex scale for chamber shape distortion
	ShapeThreshold    float64 // Noise+distance threshold for chamber carving
	CAPassesPhase1    int     // Aggressive CA sculpting passes
	CAPassesPhase2    int     // Gentle CA cleanup passes
	ErosionPasses     int     // Erosion/accretion smoothing passes
	TunnelBias        float64 // Drunkard's walk directional bias toward target (0.0-1.0)
	PillarDensity     float64 // Density of 2x2 cover pillars inside chambers
	StalactiteDensity float64 // Chance for 1x1 wall features near edges
	BorderThickness   int     // Solid wall border around map edges
	TargetWalkableMin float64 // Minimum acceptable walkable ratio
	TargetWalkableMax float64 // Maximum acceptable walkable ratio
	Seed              int64   // 0 = use time-based seed
}

// chamber tracks a seeded chamber's center and bounding radius.
type chamber struct {
	cx, cy, radius int
}

// mstEdge represents a connection between two chambers.
type mstEdge struct {
	from, to int
}

// DefaultCavernConfig returns tuned defaults for organic cave generation.
func DefaultCavernConfig() CavernConfig {
	return CavernConfig{
		FillDensity:       0.62,
		NumChambers:       5,
		MinChamberRadius:  6,
		MaxChamberRadius:  11,
		NoiseScale:        0.15,
		ShapeThreshold:    0.38,
		CAPassesPhase1:    4,
		CAPassesPhase2:    2,
		ErosionPasses:     2,
		TunnelBias:        0.60,
		PillarDensity:     0.04,
		StalactiteDensity: 0.02,
		BorderThickness:   2,
		TargetWalkableMin: 0.42,
		TargetWalkableMax: 0.50,
	}
}

// NewCavernGenerator creates a new cavern generator with the given config.
func NewCavernGenerator(config CavernConfig) *CavernGenerator {
	return &CavernGenerator{config: config}
}

func (g *CavernGenerator) Name() string {
	return "cavern"
}

func (g *CavernGenerator) Description() string {
	return "Organic cave layouts with irregular chambers and winding tunnels for tactical combat"
}

func (g *CavernGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Step 0: Initialize all-wall terrain map and noise
	terrainMap := make([]bool, width*height)
	seed := g.config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	noise := opensimplex.New(seed)

	// Step 1: Seed noise-shaped chambers
	chambers := g.seedChambers(terrainMap, width, height, noise)

	// Step 2: Dense random fill outside chambers
	g.randomFillOutsideChambers(terrainMap, width, height)

	// Step 3+4: Build MST and carve drunkard's walk tunnels
	edges := g.buildMST(chambers)
	for _, edge := range edges {
		g.carveDrunkardTunnel(terrainMap, width, height, chambers[edge.from], chambers[edge.to])
	}

	// Step 5: Phase 1 CA - aggressive sculpting
	for i := 0; i < g.config.CAPassesPhase1; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height, false)
	}

	// Step 6: Phase 2 CA - gentle cleanup
	for i := 0; i < g.config.CAPassesPhase2; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height, true)
	}

	// Step 7: Erosion/accretion smoothing
	for i := 0; i < g.config.ErosionPasses; i++ {
		g.erosionAccretionPass(terrainMap, width, height)
	}

	// Step 8: Enforce borders
	g.enforceBorders(terrainMap, width, height)

	// Step 9: Connectivity safety net
	ensureTerrainConnectivity(terrainMap, width, height)
	// If connectivity carved L-shaped corridors, smooth them
	terrainMap = g.cellularAutomataStep(terrainMap, width, height, true)
	g.enforceBorders(terrainMap, width, height)

	// Step 10: Walkable ratio check and correction
	g.checkWalkableRatio(terrainMap, width, height)

	// Step 11: Terrain features
	g.placePillars(terrainMap, chambers, width, height)
	g.placeStalactites(terrainMap, width, height)

	// Step 12: Faction starts
	factionStarts := g.placeFactionStarts(terrainMap, width, height)

	// Step 13: Convert to tiles
	convertTerrainMapToTiles(&result, terrainMap, width, height, images, BiomeMountain)

	// Record chambers as rooms for compatibility
	for _, ch := range chambers {
		result.Rooms = append(result.Rooms, NewRect(
			ch.cx-ch.radius, ch.cy-ch.radius,
			ch.radius*2, ch.radius*2,
		))
	}

	// Record faction start positions
	for i, pos := range factionStarts {
		result.FactionStartPositions = append(result.FactionStartPositions, FactionStartPosition{
			Position: pos,
			Biome:    BiomeMountain,
			Sector:   i,
		})
	}

	return result
}

// seedChambers divides the map into a 3x2 sector grid and places noise-distorted chambers.
func (g *CavernGenerator) seedChambers(terrainMap []bool, width, height int, noise opensimplex.Noise) []chamber {
	cols := 3
	rows := 2
	sectorW := width / cols
	sectorH := height / rows

	chambers := make([]chamber, 0, cols*rows)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			sxMin := col * sectorW
			syMin := row * sectorH
			sxMax := sxMin + sectorW
			syMax := syMin + sectorH

			radius := common.GetRandomBetween(g.config.MinChamberRadius, g.config.MaxChamberRadius)

			cx := common.GetRandomBetween(sxMin+radius+1, sxMax-radius-1)
			cy := common.GetRandomBetween(syMin+radius+1, syMax-radius-1)

			// Clamp to map bounds
			if cx-radius < g.config.BorderThickness {
				cx = g.config.BorderThickness + radius
			}
			if cy-radius < g.config.BorderThickness {
				cy = g.config.BorderThickness + radius
			}
			if cx+radius >= width-g.config.BorderThickness {
				cx = width - g.config.BorderThickness - radius - 1
			}
			if cy+radius >= height-g.config.BorderThickness {
				cy = height - g.config.BorderThickness - radius - 1
			}

			g.carveNoiseShape(terrainMap, cx, cy, radius, width, height, noise)
			chambers = append(chambers, chamber{cx: cx, cy: cy, radius: radius})
		}
	}

	// 30% chance to drop one chamber for variety
	if len(chambers) > 4 && common.GetDiceRoll(100) <= 30 {
		dropIdx := common.GetRandomBetween(0, len(chambers)-1)
		ch := chambers[dropIdx]
		g.setCircularRegion(terrainMap, ch.cx, ch.cy, ch.radius, width, height, false)
		chambers = append(chambers[:dropIdx], chambers[dropIdx+1:]...)
	}

	return chambers
}

// carveNoiseShape creates an irregular, amoeba-like chamber using OpenSimplex noise.
// For each tile in the bounding box, a combined distance+noise value determines carving.
func (g *CavernGenerator) carveNoiseShape(terrainMap []bool, cx, cy, radius, width, height int, noise opensimplex.Noise) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}

			dx := float64(x - cx)
			dy := float64(y - cy)
			distFromCenter := math.Sqrt(dx*dx+dy*dy) / float64(radius)

			if distFromCenter > 1.3 {
				continue // Skip tiles far outside bounding radius
			}

			noiseVal := noise.Eval2(float64(x)*g.config.NoiseScale, float64(y)*g.config.NoiseScale)
			// noiseVal is in [-1, 1], normalize to [0, 1]
			normalizedNoise := noiseVal*0.5 + 0.5

			// Combine: distance falloff (70% weight) + noise (30% weight)
			value := (1.0-distFromCenter)*0.7 + normalizedNoise*0.3

			if value > g.config.ShapeThreshold {
				terrainMap[y*width+x] = true
			}
		}
	}
}

// setCircularRegion sets all tiles within a circle to the given value.
func (g *CavernGenerator) setCircularRegion(terrainMap []bool, cx, cy, radius, width, height int, value bool) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				terrainMap[y*width+x] = value
			}
		}
	}
}

// randomFillOutsideChambers opens random tiles outside already-carved areas.
func (g *CavernGenerator) randomFillOutsideChambers(terrainMap []bool, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if terrainMap[idx] {
				continue // Already carved
			}

			randomPercent := float64(common.GetDiceRoll(100)) / 100.0
			if randomPercent > g.config.FillDensity {
				terrainMap[idx] = true
			}
		}
	}
}

// buildMST creates a minimum spanning tree connecting all chambers using Prim's algorithm.
// Also adds redundant edges with 40% chance for tactical flanking loops.
func (g *CavernGenerator) buildMST(chambers []chamber) []mstEdge {
	n := len(chambers)
	if n < 2 {
		return nil
	}

	// Precompute distances between all chamber pairs
	dist := func(a, b int) float64 {
		dx := float64(chambers[a].cx - chambers[b].cx)
		dy := float64(chambers[a].cy - chambers[b].cy)
		return math.Sqrt(dx*dx + dy*dy)
	}

	// Prim's algorithm
	inMST := make([]bool, n)
	inMST[0] = true
	mstCount := 1

	var mstEdges []mstEdge
	var nonMSTEdges []mstEdge

	for mstCount < n {
		bestDist := math.MaxFloat64
		bestFrom := -1
		bestTo := -1

		for i := 0; i < n; i++ {
			if !inMST[i] {
				continue
			}
			for j := 0; j < n; j++ {
				if inMST[j] {
					continue
				}
				d := dist(i, j)
				if d < bestDist {
					bestDist = d
					bestFrom = i
					bestTo = j
				}
			}
		}

		if bestFrom >= 0 {
			inMST[bestTo] = true
			mstCount++
			mstEdges = append(mstEdges, mstEdge{from: bestFrom, to: bestTo})
		}
	}

	// Collect non-MST edges and add some for redundancy
	mstSet := make(map[[2]int]bool)
	for _, e := range mstEdges {
		a, b := e.from, e.to
		if a > b {
			a, b = b, a
		}
		mstSet[[2]int{a, b}] = true
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if !mstSet[[2]int{i, j}] {
				nonMSTEdges = append(nonMSTEdges, mstEdge{from: i, to: j})
			}
		}
	}

	// 40% chance to add each non-MST edge
	for _, e := range nonMSTEdges {
		if common.GetDiceRoll(100) <= 40 {
			mstEdges = append(mstEdges, e)
		}
	}

	return mstEdges
}

// carveDrunkardTunnel creates an organic tunnel between two chambers using biased random walk.
// The tunnel width varies between 1 and 2 tile radius for natural chokepoints.
func (g *CavernGenerator) carveDrunkardTunnel(terrainMap []bool, width, height int, src, dst chamber) {
	x, y := src.cx, src.cy
	carveRadius := 1
	stepsUntilToggle := common.GetRandomBetween(15, 25)
	stepCount := 0

	for steps := 0; steps < 300; steps++ {
		// Carve circle at current position
		for dy := -carveRadius; dy <= carveRadius; dy++ {
			for dx := -carveRadius; dx <= carveRadius; dx++ {
				if dx*dx+dy*dy > carveRadius*carveRadius {
					continue
				}
				nx, ny := x+dx, y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					terrainMap[ny*width+nx] = true
				}
			}
		}

		// Check if close enough to target
		tdx := dst.cx - x
		tdy := dst.cy - y
		distSq := tdx*tdx + tdy*tdy
		if distSq <= 9 { // Within 3 tiles
			break
		}

		// Move: biased toward target or random
		if float64(common.GetDiceRoll(100))/100.0 <= g.config.TunnelBias {
			// Move toward target - pick the axis with larger distance
			absDx := tdx
			if absDx < 0 {
				absDx = -absDx
			}
			absDy := tdy
			if absDy < 0 {
				absDy = -absDy
			}

			if absDx >= absDy {
				if tdx > 0 {
					x++
				} else {
					x--
				}
			} else {
				if tdy > 0 {
					y++
				} else {
					y--
				}
			}
		} else {
			// Random cardinal direction
			dir := common.GetRandomBetween(0, 3)
			switch dir {
			case 0:
				x++
			case 1:
				x--
			case 2:
				y++
			case 3:
				y--
			}
		}

		// Clamp to map bounds
		if x < 1 {
			x = 1
		}
		if y < 1 {
			y = 1
		}
		if x >= width-1 {
			x = width - 2
		}
		if y >= height-1 {
			y = height - 2
		}

		// Toggle carve radius for width variation
		stepCount++
		if stepCount >= stepsUntilToggle {
			if carveRadius == 1 {
				carveRadius = 2
			} else {
				carveRadius = 1
			}
			stepsUntilToggle = common.GetRandomBetween(15, 25)
			stepCount = 0
		}
	}
}

// cellularAutomataStep applies one CA iteration.
// When gentle=false (aggressive): 5+ wall neighbors -> wall, else floor.
// When gentle=true: 5+ wall neighbors -> wall, 2 or fewer -> floor, 3-4 -> unchanged (stability band).
func (g *CavernGenerator) cellularAutomataStep(terrainMap []bool, width, height int, gentle bool) []bool {
	newMap := make([]bool, len(terrainMap))
	if gentle {
		copy(newMap, terrainMap)
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x

			wallCount := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= width || ny < 0 || ny >= height {
						wallCount++
						continue
					}
					if !terrainMap[ny*width+nx] {
						wallCount++
					}
				}
			}

			if gentle {
				if wallCount >= 5 {
					newMap[idx] = false // Wall
				} else if wallCount <= 2 {
					newMap[idx] = true // Floor
				}
				// 3-4 wall neighbors: unchanged (stability band)
			} else {
				newMap[idx] = (wallCount < 5)
			}
		}
	}

	return newMap
}

// erosionAccretionPass smooths wall contours for natural cave walls.
// Erosion: wall with 5+ walkable neighbors -> 50% chance floor.
// Accretion: floor with 7+ wall neighbors -> wall.
func (g *CavernGenerator) erosionAccretionPass(terrainMap []bool, width, height int) {
	changes := make([]struct {
		idx int
		val bool
	}, 0)

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			idx := y*width + x

			walkableCount := 0
			wallCount := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx < 0 || nx >= width || ny < 0 || ny >= height {
						wallCount++
						continue
					}
					if terrainMap[ny*width+nx] {
						walkableCount++
					} else {
						wallCount++
					}
				}
			}

			if !terrainMap[idx] && walkableCount >= 5 {
				// Erosion: wall tile surrounded by mostly walkable -> 50% become floor
				if common.GetDiceRoll(100) <= 50 {
					changes = append(changes, struct {
						idx int
						val bool
					}{idx, true})
				}
			} else if terrainMap[idx] && wallCount >= 7 {
				// Accretion: floor tile surrounded by mostly wall -> become wall
				changes = append(changes, struct {
					idx int
					val bool
				}{idx, false})
			}
		}
	}

	for _, c := range changes {
		terrainMap[c.idx] = c.val
	}
}

// enforceBorders sets outer tiles on all edges to wall.
func (g *CavernGenerator) enforceBorders(terrainMap []bool, width, height int) {
	border := g.config.BorderThickness
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if x < border || x >= width-border || y < border || y >= height-border {
				terrainMap[y*width+x] = false
			}
		}
	}
}

// checkWalkableRatio checks and corrects the walkable tile ratio.
// If above 55%: run 1 additional Phase 1 CA pass + re-enforce borders.
// If below 35%: run 1 erosion pass with relaxed threshold.
func (g *CavernGenerator) checkWalkableRatio(terrainMap []bool, width, height int) {
	totalTiles := width * height
	walkable := 0
	for _, v := range terrainMap {
		if v {
			walkable++
		}
	}

	ratio := float64(walkable) / float64(totalTiles)

	if ratio > g.config.TargetWalkableMax {
		// Too open: run aggressive CA pass to close things up
		result := g.cellularAutomataStep(terrainMap, width, height, false)
		copy(terrainMap, result)
		g.enforceBorders(terrainMap, width, height)
	} else if ratio < g.config.TargetWalkableMin {
		// Too closed: relax some walls
		for y := 1; y < height-1; y++ {
			for x := 1; x < width-1; x++ {
				idx := y*width + x
				if terrainMap[idx] {
					continue
				}

				walkableCount := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < width && ny >= 0 && ny < height {
							if terrainMap[ny*width+nx] {
								walkableCount++
							}
						}
					}
				}

				if walkableCount >= 4 && common.GetDiceRoll(100) <= 40 {
					terrainMap[idx] = true
				}
			}
		}
	}
}

// placePillars places 2x2 wall clusters inside chambers for tactical cover.
func (g *CavernGenerator) placePillars(terrainMap []bool, chambers []chamber, width, height int) {
	for _, ch := range chambers {
		area := math.Pi * float64(ch.radius) * float64(ch.radius)
		numPillars := int(area * g.config.PillarDensity)

		for i := 0; i < numPillars; i++ {
			innerR := int(float64(ch.radius) * 0.6)
			if innerR < 2 {
				continue
			}

			px := ch.cx + common.GetRandomBetween(-innerR, innerR)
			py := ch.cy + common.GetRandomBetween(-innerR, innerR)

			// Verify within bounding area
			dx := px - ch.cx
			dy := py - ch.cy
			if dx*dx+dy*dy > innerR*innerR {
				continue
			}

			tryPlace2x2PillarOnTerrain(terrainMap, px, py, width, height)
		}
	}
}

// placeStalactites adds 1x1 wall features near cave edges for partial cover and visual variety.
// For each floor tile adjacent to a wall, there's a small chance it becomes wall.
func (g *CavernGenerator) placeStalactites(terrainMap []bool, width, height int) {
	// Collect candidates first, then apply (avoid cascade)
	var candidates []int

	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			idx := y*width + x
			if !terrainMap[idx] {
				continue
			}

			// Check if adjacent to at least one wall
			adjacentWall := false
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						if !terrainMap[ny*width+nx] {
							adjacentWall = true
						}
					}
				}
			}

			if adjacentWall && float64(common.GetDiceRoll(1000))/1000.0 <= g.config.StalactiteDensity {
				candidates = append(candidates, idx)
			}
		}
	}

	for _, idx := range candidates {
		terrainMap[idx] = false
	}
}

// placeFactionStarts finds good starting positions on opposite sides of the map.
func (g *CavernGenerator) placeFactionStarts(terrainMap []bool, width, height int) []coords.LogicalPosition {
	var starts []coords.LogicalPosition

	leftBest := findBestOpenPosition(terrainMap, width, height, 0, width/4, 0, height, 6)
	rightBest := findBestOpenPosition(terrainMap, width, height, 3*width/4, width, 0, height, 6)

	if leftBest.X >= 0 && rightBest.X >= 0 {
		g.setCircularRegion(terrainMap, leftBest.X, leftBest.Y, 6, width, height, true)
		g.setCircularRegion(terrainMap, rightBest.X, rightBest.Y, 6, width, height, true)
		starts = append(starts, leftBest, rightBest)
	}

	return starts
}

// Register on package initialization
func init() {
	RegisterGenerator(&CavernGenerator{config: DefaultCavernConfig()})
}
