package worldmap

import (
	"game_main/common"
	"game_main/visual/graphics"
	"game_main/world/coords"
	"math"
)

// CavernGenerator creates organic cave layouts using cellular automata with seeded chambers.
// Designed for squad-based tactical combat: large chambers for multi-squad battles,
// connected by passages wide enough for squads (3-5 tiles) but narrow enough for chokepoints.
type CavernGenerator struct {
	config CavernConfig
}

// CavernConfig holds tuning parameters for cavern generation.
type CavernConfig struct {
	FillDensity      float64 // Initial random wall density (0.42 = ~55-58% walkable after CA)
	NumChambers      int     // Target number of seeded chambers
	MinChamberRadius int     // Minimum chamber radius in tiles
	MaxChamberRadius int     // Maximum chamber radius in tiles
	CAIterations     int     // Number of cellular automata smoothing passes
	MinPassageWidth  int     // Minimum traversable passage width in tiles
	PillarDensity    float64 // Density of cover pillars inside chambers (0.0-1.0)
	BorderThickness  int     // Solid wall border around map edges
}

// chamber tracks a seeded chamber's center and radius for re-stamping.
type chamber struct {
	cx, cy, radius int
}

// DefaultCavernConfig returns tuned defaults for squad-based cave combat.
func DefaultCavernConfig() CavernConfig {
	return CavernConfig{
		FillDensity:      0.42,
		NumChambers:      5,
		MinChamberRadius: 8,
		MaxChamberRadius: 12,
		CAIterations:     6,
		MinPassageWidth:  3,
		PillarDensity:    0.03,
		BorderThickness:  2,
	}
}

func (g *CavernGenerator) Name() string {
	return "cavern"
}

func (g *CavernGenerator) Description() string {
	return "Organic cave layouts with large chambers for squad-based tactical combat"
}

func (g *CavernGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:          createEmptyTiles(width, height, images),
		Rooms:          make([]Rect, 0),
		ValidPositions: make([]coords.LogicalPosition, 0),
	}

	// Step 1: Initialize terrain map (all solid)
	terrainMap := make([]bool, width*height)

	// Step 2: Seed guaranteed chambers
	chambers := g.seedChambers(terrainMap, width, height)

	// Step 3: Random fill outside chambers
	g.randomFillOutsideChambers(terrainMap, chambers, width, height)

	// Step 4: Cellular automata smoothing
	for i := 0; i < g.config.CAIterations; i++ {
		terrainMap = g.cellularAutomataStep(terrainMap, width, height)
	}

	// Step 5: Re-stamp chambers (CA may have eroded edges)
	g.reStampChambers(terrainMap, chambers, width, height)

	// Step 6: Widen narrow passages (2 passes)
	for pass := 0; pass < 2; pass++ {
		g.widenPassages(terrainMap, width, height)
	}

	// Step 7: Enforce map border walls
	g.enforceBorders(terrainMap, width, height)

	// Step 8: Ensure connectivity
	ensureTerrainConnectivity(terrainMap, width, height)
	// One more widening pass on connectivity corridors
	g.widenPassages(terrainMap, width, height)
	// Re-enforce borders after widening
	g.enforceBorders(terrainMap, width, height)

	// Step 9: Post-processing
	g.placePillars(terrainMap, chambers, width, height)
	factionStarts := g.placeFactionStarts(terrainMap, width, height)
	g.convertToTiles(&result, terrainMap, width, height, images)

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

// seedChambers divides the map into a sector grid and places one circular chamber per sector.
func (g *CavernGenerator) seedChambers(terrainMap []bool, width, height int) []chamber {
	// 3x2 grid of sectors for a 100x80 map, adaptive for other sizes
	cols := 3
	rows := 2
	sectorW := width / cols
	sectorH := height / rows

	chambers := make([]chamber, 0, cols*rows)

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			// Sector boundaries
			sxMin := col * sectorW
			syMin := row * sectorH
			sxMax := sxMin + sectorW
			syMax := syMin + sectorH

			radius := common.GetRandomBetween(g.config.MinChamberRadius, g.config.MaxChamberRadius)

			// Random center within sector, clamped so circle stays in bounds
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

			// Carve circle into terrain map
			g.carveCircle(terrainMap, cx, cy, radius, width, height)

			chambers = append(chambers, chamber{cx: cx, cy: cy, radius: radius})
		}
	}

	// Optionally drop one chamber for variety (4-6 chambers)
	if len(chambers) > 4 && common.GetDiceRoll(100) <= 30 {
		dropIdx := common.GetRandomBetween(0, len(chambers)-1)
		// Fill it back in
		ch := chambers[dropIdx]
		g.fillCircle(terrainMap, ch.cx, ch.cy, ch.radius, width, height)
		chambers = append(chambers[:dropIdx], chambers[dropIdx+1:]...)
	}

	return chambers
}

// carveCircle sets all tiles within a circle to walkable.
func (g *CavernGenerator) carveCircle(terrainMap []bool, cx, cy, radius, width, height int) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				terrainMap[y*width+x] = true
			}
		}
	}
}

// fillCircle sets all tiles within a circle to wall (for dropping a chamber).
func (g *CavernGenerator) fillCircle(terrainMap []bool, cx, cy, radius, width, height int) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				terrainMap[y*width+x] = false
			}
		}
	}
}

// randomFillOutsideChambers opens random tiles outside chamber areas.
func (g *CavernGenerator) randomFillOutsideChambers(terrainMap []bool, chambers []chamber, width, height int) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			// Skip already-carved chamber tiles
			if terrainMap[idx] {
				continue
			}

			// Random open with probability (1 - FillDensity)
			randomPercent := float64(common.GetDiceRoll(100)) / 100.0
			if randomPercent > g.config.FillDensity {
				terrainMap[idx] = true
			}
		}
	}
}

// cellularAutomataStep applies one CA iteration.
// Rule: 5+ wall neighbors out of 8 → wall; else floor.
func (g *CavernGenerator) cellularAutomataStep(terrainMap []bool, width, height int) []bool {
	newMap := make([]bool, len(terrainMap))

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

			newMap[idx] = (wallCount < 5)
		}
	}

	return newMap
}

// reStampChambers re-carves chambers at radius-1 to preserve interiors after CA smoothing.
func (g *CavernGenerator) reStampChambers(terrainMap []bool, chambers []chamber, width, height int) {
	for _, ch := range chambers {
		innerRadius := ch.radius - 1
		if innerRadius < 3 {
			innerRadius = 3
		}
		g.carveCircle(terrainMap, ch.cx, ch.cy, innerRadius, width, height)
	}
}

// widenPassages scans for narrow passages and expands them.
// For every walkable tile, checks 5x5 neighborhood. If fewer than 8 of 25
// tiles are walkable, opens wall tiles in a 3x3 area around it.
func (g *CavernGenerator) widenPassages(terrainMap []bool, width, height int) {
	totalTiles := width * height

	// Safety cap: don't exceed 70% walkable
	maxWalkable := int(float64(totalTiles) * 0.70)
	walkableCount := 0
	for _, v := range terrainMap {
		if v {
			walkableCount++
		}
	}

	// Collect tiles that need widening first, then apply
	type widenTarget struct{ x, y int }
	var targets []widenTarget

	for y := 2; y < height-2; y++ {
		for x := 2; x < width-2; x++ {
			idx := y*width + x
			if !terrainMap[idx] {
				continue
			}

			// Count walkable in 5x5 neighborhood
			openCount := 0
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						if terrainMap[ny*width+nx] {
							openCount++
						}
					}
				}
			}

			if openCount < 8 {
				targets = append(targets, widenTarget{x, y})
			}
		}
	}

	// Apply widening
	for _, t := range targets {
		if walkableCount >= maxWalkable {
			break
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := t.x+dx, t.y+dy
				if nx >= 0 && nx < width && ny >= 0 && ny < height {
					nidx := ny*width + nx
					if !terrainMap[nidx] {
						terrainMap[nidx] = true
						walkableCount++
					}
				}
			}
		}
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

// placePillars places 2x2 wall clusters inside chambers for tactical cover.
func (g *CavernGenerator) placePillars(terrainMap []bool, chambers []chamber, width, height int) {
	for _, ch := range chambers {
		// Number of pillars proportional to chamber area
		area := math.Pi * float64(ch.radius) * float64(ch.radius)
		numPillars := int(area * g.config.PillarDensity)

		for i := 0; i < numPillars; i++ {
			// Random position within inner 60% of chamber radius
			innerR := int(float64(ch.radius) * 0.6)
			if innerR < 2 {
				continue
			}

			// Pick random offset from center
			px := ch.cx + common.GetRandomBetween(-innerR, innerR)
			py := ch.cy + common.GetRandomBetween(-innerR, innerR)

			// Verify it's within the chamber circle
			dx := px - ch.cx
			dy := py - ch.cy
			if dx*dx+dy*dy > innerR*innerR {
				continue
			}

			// Place 2x2 pillar
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					nx, ny := px+dx, py+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						terrainMap[ny*width+nx] = false
					}
				}
			}
		}
	}
}

// placeFactionStarts finds good starting positions on opposite sides of the map.
func (g *CavernGenerator) placeFactionStarts(terrainMap []bool, width, height int) []coords.LogicalPosition {
	var starts []coords.LogicalPosition

	// Left zone: x < width/4
	leftBest := g.findBestStart(terrainMap, width, height, 0, width/4)
	// Right zone: x > 3*width/4
	rightBest := g.findBestStart(terrainMap, width, height, 3*width/4, width)

	if leftBest.X >= 0 && rightBest.X >= 0 {
		// Check minimum 40-tile separation
		dx := float64(rightBest.X - leftBest.X)
		dy := float64(rightBest.Y - leftBest.Y)
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist >= 40 {
			// Clear 6-tile radius around each start
			g.clearAreaAround(terrainMap, leftBest.X, leftBest.Y, 6, width, height)
			g.clearAreaAround(terrainMap, rightBest.X, rightBest.Y, 6, width, height)
			starts = append(starts, leftBest, rightBest)
		} else {
			// Separation too small, just use what we have
			g.clearAreaAround(terrainMap, leftBest.X, leftBest.Y, 6, width, height)
			g.clearAreaAround(terrainMap, rightBest.X, rightBest.Y, 6, width, height)
			starts = append(starts, leftBest, rightBest)
		}
	}

	return starts
}

// findBestStart scores walkable tiles in an x-range by openness and returns the best.
func (g *CavernGenerator) findBestStart(terrainMap []bool, width, height, xMin, xMax int) coords.LogicalPosition {
	bestScore := -1
	bestPos := coords.LogicalPosition{X: -1, Y: -1}

	scanRadius := 6

	for y := scanRadius; y < height-scanRadius; y++ {
		for x := xMin; x < xMax; x++ {
			idx := y*width + x
			if !terrainMap[idx] {
				continue
			}

			// Score by counting walkable neighbors in radius
			score := 0
			for dy := -scanRadius; dy <= scanRadius; dy++ {
				for dx := -scanRadius; dx <= scanRadius; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						if terrainMap[ny*width+nx] {
							score++
						}
					}
				}
			}

			if score > bestScore {
				bestScore = score
				bestPos = coords.LogicalPosition{X: x, Y: y}
			}
		}
	}

	return bestPos
}

// clearAreaAround opens a circular area around a position.
func (g *CavernGenerator) clearAreaAround(terrainMap []bool, cx, cy, radius, width, height int) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if x < 0 || x >= width || y < 0 || y >= height {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				terrainMap[y*width+x] = true
			}
		}
	}
}

// convertToTiles converts the boolean terrain map to actual Tile objects.
func (g *CavernGenerator) convertToTiles(result *GenerationResult, terrainMap []bool, width, height int, images TileImageSet) {
	wallImages, floorImages := getBiomeImages(images, BiomeMountain)

	// Initialize biome map
	result.BiomeMap = make([]Biome, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := positionToIndex(x, y, width)
			logicalPos := coords.LogicalPosition{X: x, Y: y}
			pixelX := x * graphics.ScreenInfo.TileSize
			pixelY := y * graphics.ScreenInfo.TileSize

			result.BiomeMap[idx] = BiomeMountain

			if terrainMap[idx] {
				// Walkable floor tile
				floorImage := selectRandomImage(floorImages)
				tile := NewTile(pixelX, pixelY, logicalPos, false, floorImage, FLOOR, false)
				// NewTile hardcodes Blocked=true and TileType=WALL, so set explicitly
				tile.Blocked = false
				tile.TileType = FLOOR
				tile.Biome = BiomeMountain
				result.Tiles[idx] = &tile
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			} else {
				// Wall tile
				wallImage := selectRandomImage(wallImages)
				tile := NewTile(pixelX, pixelY, logicalPos, true, wallImage, WALL, false)
				tile.Biome = BiomeMountain
				result.Tiles[idx] = &tile
			}
		}
	}
}

// Register on package initialization
func init() {
	RegisterGenerator(&CavernGenerator{config: DefaultCavernConfig()})
}
