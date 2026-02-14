package worldmap

import (
	"game_main/common"
	"game_main/world/coords"
	"math"
	"time"

	opensimplex "github.com/ojrac/opensimplex-go"
)

// StrategicOverworldConfig holds parameters for the strategic overworld generator
type StrategicOverworldConfig struct {
	// Noise
	ElevationOctaves int
	ElevationScale   float64
	MoistureOctaves  int
	MoistureScale    float64
	Persistence      float64 // Amplitude decay per octave
	Lacunarity       float64 // Frequency multiplier per octave

	// Biome thresholds
	WaterThresh    float64
	MountainThresh float64
	ForestMoisture float64
	SwampMoisture  float64

	// POI counts (from nodeDefinitions.json categories)
	TownCount       int
	TempleCount     int
	GuildHallCount  int
	WatchtowerCount int
	POIMinDistance  int

	// Faction
	FactionCount      int
	FactionMinSpacing int

	Seed int64
}

// DefaultStrategicOverworldConfig returns sensible defaults
func DefaultStrategicOverworldConfig() StrategicOverworldConfig {
	return StrategicOverworldConfig{
		ElevationOctaves: 4,
		ElevationScale:   0.035,
		MoistureOctaves:  3,
		MoistureScale:    0.045,
		Persistence:      0.5,
		Lacunarity:       2.0,

		WaterThresh:    0.28,
		MountainThresh: 0.72,
		ForestMoisture: 0.55,
		SwampMoisture:  0.70,

		TownCount:       3,
		TempleCount:     2,
		GuildHallCount:  2,
		WatchtowerCount: 3,
		POIMinDistance:  12,

		FactionCount:      4,
		FactionMinSpacing: 25,

		Seed: 0,
	}
}

// StrategicOverworldGenerator generates large-scale world maps with multi-octave noise,
// biome classification, terrain-aware POI placement, and faction starting positions.
type StrategicOverworldGenerator struct {
	config StrategicOverworldConfig
}

// NewStrategicOverworldGenerator creates a new strategic overworld generator
func NewStrategicOverworldGenerator(config StrategicOverworldConfig) *StrategicOverworldGenerator {
	return &StrategicOverworldGenerator{config: config}
}

func (g *StrategicOverworldGenerator) Name() string {
	return "overworld"
}

func (g *StrategicOverworldGenerator) Description() string {
	return "Strategic world map with multi-octave terrain, typed POIs, and faction starting positions"
}

func (g *StrategicOverworldGenerator) Generate(width, height int, images TileImageSet) GenerationResult {
	result := GenerationResult{
		Tiles:                 createEmptyTiles(width, height, images),
		Rooms:                 make([]Rect, 0),
		ValidPositions:        make([]coords.LogicalPosition, 0),
		POIs:                  make([]POIData, 0),
		FactionStartPositions: make([]FactionStartPosition, 0),
		BiomeMap:              make([]Biome, width*height),
	}

	seed := g.config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	// Step 1: Generate fBm noise maps (different seeds for independent fields)
	elevationMap := g.generateFBmMap(width, height, g.config.ElevationScale, g.config.ElevationOctaves, seed)
	moistureMap := g.generateFBmMap(width, height, g.config.MoistureScale, g.config.MoistureOctaves, seed+1)

	// Step 2: Continent shaping - radial falloff from center
	g.applyContinentShaping(elevationMap, width, height)

	// Step 3: Biome classification and tile creation
	g.classifyBiomes(&result, width, height, elevationMap, moistureMap, images)

	// Step 4: Connectivity verification - ensure all walkable areas are connected
	terrainMap := g.buildTerrainMap(&result, width, height)
	ensureTerrainConnectivity(terrainMap, width, height)
	g.applyConnectivityFixes(&result, terrainMap, width, height, elevationMap, moistureMap, images)

	// Step 5: Faction starting positions
	g.placeFactionStartPositions(&result, width, height, elevationMap)

	// Step 6: Typed POI placement
	g.placeTypedPOIs(&result, width, height, elevationMap, moistureMap, images)

	return result
}

// generateFBmMap creates a fractal Brownian motion noise map using multi-octave OpenSimplex noise
func (g *StrategicOverworldGenerator) generateFBmMap(width, height int, baseScale float64, octaves int, seed int64) [][]float64 {
	noiseMap := make([][]float64, height)
	for i := range noiseMap {
		noiseMap[i] = make([]float64, width)
	}

	noise := opensimplex.New(seed)

	// Compute maxAmplitude for normalization (geometric series)
	maxAmplitude := 0.0
	amp := 1.0
	for o := 0; o < octaves; o++ {
		maxAmplitude += amp
		amp *= g.config.Persistence
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := 0.0
			amplitude := 1.0
			frequency := baseScale

			for o := 0; o < octaves; o++ {
				value += amplitude * noise.Eval2(float64(x)*frequency, float64(y)*frequency)
				amplitude *= g.config.Persistence
				frequency *= g.config.Lacunarity
			}

			// Normalize to [0, 1]
			normalized := (value/maxAmplitude + 1.0) / 2.0
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1.0 {
				normalized = 1.0
			}
			noiseMap[y][x] = normalized
		}
	}

	return noiseMap
}

// applyContinentShaping applies radial distance falloff so map edges trend toward water
func (g *StrategicOverworldGenerator) applyContinentShaping(elevationMap [][]float64, width, height int) {
	centerX := float64(width) / 2.0
	centerY := float64(height) / 2.0
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist // 0-1

			// Reduce elevation near edges
			elevationMap[y][x] *= 1.0 - (dist * 0.6)
			if elevationMap[y][x] < 0 {
				elevationMap[y][x] = 0
			}
		}
	}
}

// classifyBiomes converts elevation/moisture maps to biome-classified tiles
func (g *StrategicOverworldGenerator) classifyBiomes(result *GenerationResult, width, height int, elevationMap, moistureMap [][]float64, images TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			elevation := elevationMap[y][x]
			moisture := moistureMap[y][x]
			biome := g.determineBiome(elevation, moisture)

			logicalPos := coords.LogicalPosition{X: x, Y: y}
			index := positionToIndex(x, y, width)

			if index < 0 || index >= len(result.Tiles) {
				continue
			}

			tile := result.Tiles[index]
			tile.Biome = biome
			result.BiomeMap[index] = biome

			wallImages, floorImages := getBiomeImages(images, biome)

			switch biome {
			case BiomeSwamp, BiomeMountain:
				tile.TileType = WALL
				tile.Blocked = true
				tile.Image = selectRandomImage(wallImages)

			case BiomeDesert, BiomeForest, BiomeGrassland:
				tile.TileType = FLOOR
				tile.Blocked = false
				tile.Image = selectRandomImage(floorImages)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// determineBiome classifies terrain based on elevation and moisture
func (g *StrategicOverworldGenerator) determineBiome(elevation, moisture float64) Biome {
	if elevation < g.config.WaterThresh {
		return BiomeSwamp
	}
	if elevation > g.config.MountainThresh {
		return BiomeMountain
	}

	// Low-elevation high-moisture = swamp
	if moisture > g.config.SwampMoisture && elevation < 0.40 {
		return BiomeSwamp
	}

	// High-ish elevation, dry = desert
	if elevation > 0.60 && moisture < 0.35 {
		return BiomeDesert
	}

	// Wet = forest
	if moisture > g.config.ForestMoisture {
		return BiomeForest
	}

	return BiomeGrassland
}

// buildTerrainMap creates a boolean walkability map from current tile state
func (g *StrategicOverworldGenerator) buildTerrainMap(result *GenerationResult, width, height int) []bool {
	terrainMap := make([]bool, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if idx < len(result.Tiles) {
				terrainMap[idx] = !result.Tiles[idx].Blocked
			}
		}
	}
	return terrainMap
}

// applyConnectivityFixes updates tiles that were carved by connectivity to be walkable
func (g *StrategicOverworldGenerator) applyConnectivityFixes(result *GenerationResult, terrainMap []bool, width, height int, elevationMap, moistureMap [][]float64, images TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if idx >= len(result.Tiles) {
				continue
			}

			tile := result.Tiles[idx]
			wasBlocked := tile.Blocked
			isNowWalkable := terrainMap[idx]

			// Only update tiles that connectivity carving changed
			if wasBlocked && isNowWalkable {
				logicalPos := coords.LogicalPosition{X: x, Y: y}

				// Carved corridors become grassland
				tile.Biome = BiomeGrassland
				result.BiomeMap[idx] = BiomeGrassland
				tile.TileType = FLOOR
				tile.Blocked = false

				_, floorImages := getBiomeImages(images, BiomeGrassland)
				tile.Image = selectRandomImage(floorImages)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// placeFactionStartPositions divides the map into sectors and finds optimal starting positions
func (g *StrategicOverworldGenerator) placeFactionStartPositions(result *GenerationResult, width, height int, elevationMap [][]float64) {
	// Define sectors as quadrant regions offset from corners
	type sectorBounds struct {
		minX, maxX, minY, maxY int
	}

	sectors := []sectorBounds{
		{10, 30, 10, 25},                                   // Sector 0: top-left
		{width - 30, width - 10, 10, 25},                   // Sector 1: top-right
		{width - 30, width - 10, height - 25, height - 10}, // Sector 2: bottom-right
		{10, 30, height - 25, height - 10},                 // Sector 3: bottom-left
	}

	for i := 0; i < g.config.FactionCount && i < len(sectors); i++ {
		s := sectors[i]
		bestPos := coords.LogicalPosition{}
		bestScore := -1

		// Clamp sector bounds to map dimensions
		if s.minX < 0 {
			s.minX = 0
		}
		if s.maxX >= width {
			s.maxX = width - 1
		}
		if s.minY < 0 {
			s.minY = 0
		}
		if s.maxY >= height {
			s.maxY = height - 1
		}

		for y := s.minY; y <= s.maxY; y++ {
			for x := s.minX; x <= s.maxX; x++ {
				idx := positionToIndex(x, y, width)
				if idx < 0 || idx >= len(result.Tiles) || result.Tiles[idx].Blocked {
					continue
				}

				// Only consider grassland or forest
				biome := result.BiomeMap[idx]
				if biome != BiomeGrassland && biome != BiomeForest {
					continue
				}

				// Score by number of walkable neighbors in 5-tile radius
				score := g.countWalkableNeighbors(result, x, y, 5, width, height)
				if score > bestScore {
					bestScore = score
					bestPos = coords.LogicalPosition{X: x, Y: y}
				}
			}
		}

		if bestScore > 0 {
			result.FactionStartPositions = append(result.FactionStartPositions, FactionStartPosition{
				Position: bestPos,
				Biome:    result.BiomeMap[positionToIndex(bestPos.X, bestPos.Y, width)],
				Sector:   i,
			})
		}
	}
}

// countWalkableNeighbors counts walkable tiles within a radius
func (g *StrategicOverworldGenerator) countWalkableNeighbors(result *GenerationResult, cx, cy, radius, width, height int) int {
	count := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := cx+dx, cy+dy
			if nx < 0 || nx >= width || ny < 0 || ny >= height {
				continue
			}
			idx := positionToIndex(nx, ny, width)
			if idx >= 0 && idx < len(result.Tiles) && !result.Tiles[idx].Blocked {
				count++
			}
		}
	}
	return count
}

// placeTypedPOIs places terrain-aware POIs in order: towns, temples, watchtowers, guild halls
func (g *StrategicOverworldGenerator) placeTypedPOIs(result *GenerationResult, width, height int, elevationMap, moistureMap [][]float64, images TileImageSet) {
	// Track placed POI positions for distance checks
	placedPositions := make([]coords.LogicalPosition, 0)

	// Towns first (guild halls depend on them)
	placedTowns := g.placePOIType(result, width, height, elevationMap, moistureMap, POITown, g.config.TownCount, g.config.POIMinDistance, placedPositions, images)
	placedPositions = append(placedPositions, placedTowns...)

	// Temples
	placedTemples := g.placePOIType(result, width, height, elevationMap, moistureMap, POITemple, g.config.TempleCount, 15, placedPositions, images)
	placedPositions = append(placedPositions, placedTemples...)

	// Watchtowers
	placedWatchtowers := g.placePOIType(result, width, height, elevationMap, moistureMap, POIWatchtower, g.config.WatchtowerCount, 10, placedPositions, images)
	placedPositions = append(placedPositions, placedWatchtowers...)

	// Guild halls (must be near towns)
	g.placeGuildHalls(result, width, height, elevationMap, placedTowns, placedPositions, images)
}

// placePOIType places POIs of a specific type using terrain-aware rules
func (g *StrategicOverworldGenerator) placePOIType(result *GenerationResult, width, height int, elevationMap, moistureMap [][]float64, nodeID string, count, minDist int, existingPOIs []coords.LogicalPosition, images TileImageSet) []coords.LogicalPosition {
	placed := make([]coords.LogicalPosition, 0)
	maxAttempts := count * 50

	for attempt := 0; attempt < maxAttempts && len(placed) < count; attempt++ {
		if len(result.ValidPositions) == 0 {
			break
		}

		// Pick random valid position
		validIdx := common.GetRandomBetween(0, len(result.ValidPositions)-1)
		pos := result.ValidPositions[validIdx]
		idx := positionToIndex(pos.X, pos.Y, width)

		if idx < 0 || idx >= len(result.Tiles) {
			continue
		}

		biome := result.BiomeMap[idx]
		elevation := elevationMap[pos.Y][pos.X]

		// Check terrain-specific placement rules
		if !g.isValidPOITerrain(nodeID, biome, elevation, moistureMap[pos.Y][pos.X]) {
			continue
		}

		// Check distance from all existing POIs
		if g.isTooCloseToAny(pos, existingPOIs, minDist) || g.isTooCloseToAny(pos, placed, minDist) {
			continue
		}

		// Place the POI
		result.POIs = append(result.POIs, POIData{
			Position: pos,
			NodeID:   nodeID,
			Biome:    biome,
		})

		// Set POI-specific tile image so the renderer draws it
		if poiImg, ok := images.POIImages[nodeID]; ok {
			result.Tiles[idx].POIType = nodeID
			result.Tiles[idx].Image = poiImg
		}

		// Add as a 1x1 Rect for backward compat with StartingPosition()/PlaceStairs()
		result.Rooms = append(result.Rooms, NewRect(pos.X, pos.Y, 1, 1))
		placed = append(placed, pos)
	}

	return placed
}

// placeGuildHalls places guild halls near existing towns
func (g *StrategicOverworldGenerator) placeGuildHalls(result *GenerationResult, width, height int, elevationMap [][]float64, townPositions, existingPOIs []coords.LogicalPosition, images TileImageSet) {
	if len(townPositions) == 0 {
		return
	}

	placed := 0
	maxAttempts := g.config.GuildHallCount * 50

	for attempt := 0; attempt < maxAttempts && placed < g.config.GuildHallCount; attempt++ {
		if len(result.ValidPositions) == 0 {
			break
		}

		validIdx := common.GetRandomBetween(0, len(result.ValidPositions)-1)
		pos := result.ValidPositions[validIdx]
		idx := positionToIndex(pos.X, pos.Y, width)

		if idx < 0 || idx >= len(result.Tiles) || result.Tiles[idx].Blocked {
			continue
		}

		// Must be within 20 tiles of a town
		nearTown := false
		for _, town := range townPositions {
			dist := g.distance(pos, town)
			if dist <= 20.0 {
				nearTown = true
				break
			}
		}
		if !nearTown {
			continue
		}

		// Check distance from other POIs
		if g.isTooCloseToAny(pos, existingPOIs, g.config.POIMinDistance) {
			continue
		}

		result.POIs = append(result.POIs, POIData{
			Position: pos,
			NodeID:   POIGuildHall,
			Biome:    result.BiomeMap[idx],
		})

		// Set POI-specific tile image so the renderer draws it
		if poiImg, ok := images.POIImages[POIGuildHall]; ok {
			result.Tiles[idx].POIType = POIGuildHall
			result.Tiles[idx].Image = poiImg
		}

		result.Rooms = append(result.Rooms, NewRect(pos.X, pos.Y, 1, 1))
		existingPOIs = append(existingPOIs, pos)
		placed++
	}
}

// isValidPOITerrain checks terrain-specific placement rules per POI type
func (g *StrategicOverworldGenerator) isValidPOITerrain(nodeID string, biome Biome, elevation, moisture float64) bool {
	switch nodeID {
	case POITown:
		// Towns: grassland or forest edge
		return biome == BiomeGrassland || biome == BiomeForest

	case POITemple:
		// Temples: elevated terrain or desert, prefer isolated
		return (elevation > 0.55 && biome != BiomeSwamp && biome != BiomeMountain) || biome == BiomeDesert

	case POIWatchtower:
		// Watchtowers: elevated walkable terrain near mountain borders
		return elevation > 0.50 && biome != BiomeSwamp && biome != BiomeMountain

	default:
		// Guild halls and others: any walkable
		return biome != BiomeSwamp && biome != BiomeMountain
	}
}

// isTooCloseToAny checks if pos is within minDist of any position in the list
func (g *StrategicOverworldGenerator) isTooCloseToAny(pos coords.LogicalPosition, positions []coords.LogicalPosition, minDist int) bool {
	for _, other := range positions {
		if g.distance(pos, other) < float64(minDist) {
			return true
		}
	}
	return false
}

// distance computes Euclidean distance between two positions
func (g *StrategicOverworldGenerator) distance(a, b coords.LogicalPosition) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewStrategicOverworldGenerator(DefaultStrategicOverworldConfig()))
}
