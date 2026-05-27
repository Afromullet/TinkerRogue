package worldgen

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/world/worldmapcore"
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

func (g *StrategicOverworldGenerator) Generate(ctx worldmapcore.GenContext, images worldmapcore.TileImageSet) worldmapcore.GenerationResult {
	width, height := ctx.Width, ctx.Height

	result := worldmapcore.GenerationResult{
		Tiles:                 CreateEmptyTiles(ctx, images),
		Rooms:                 make([]worldmapcore.Rect, 0),
		ValidPositions:        make([]coords.LogicalPosition, 0),
		POIs:                  make([]worldmapcore.POIData, 0),
		FactionStartPositions: make([]worldmapcore.FactionStartPosition, 0),
		BiomeMap:              make([]worldmapcore.Biome, width*height),
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
	EnsureTerrainConnectivity(terrainMap, width, height)
	g.applyConnectivityFixes(&result, terrainMap, width, height, elevationMap, moistureMap, images)

	// Step 5: Faction starting positions
	g.placeFactionStartPositions(&result, width, height, elevationMap)

	// Step 6: Typed POI placement
	g.placeTypedPOIs(&result, width, height, elevationMap, moistureMap, images)

	return result
}

// generateFBmMap creates a fractal Brownian motion noise map using multi-octave OpenSimplex noise.
// Returns a flat row-major slice indexed via PositionToIndex(x, y).
func (g *StrategicOverworldGenerator) generateFBmMap(width, height int, baseScale float64, octaves int, seed int64) []float64 {
	noiseMap := make([]float64, width*height)

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
			noiseMap[PositionToIndex(x, y)] = normalized
		}
	}

	return noiseMap
}

// applyContinentShaping applies radial distance falloff so map edges trend toward water
func (g *StrategicOverworldGenerator) applyContinentShaping(elevationMap []float64, width, height int) {
	centerX := float64(width) / 2.0
	centerY := float64(height) / 2.0
	maxDist := math.Sqrt(centerX*centerX + centerY*centerY)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dx := float64(x) - centerX
			dy := float64(y) - centerY
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist // 0-1

			idx := PositionToIndex(x, y)
			// Reduce elevation near edges
			elevationMap[idx] *= 1.0 - (dist * 0.6)
			if elevationMap[idx] < 0 {
				elevationMap[idx] = 0
			}
		}
	}
}

// classifyBiomes converts elevation/moisture maps to biome-classified tiles
func (g *StrategicOverworldGenerator) classifyBiomes(result *worldmapcore.GenerationResult, width, height int, elevationMap, moistureMap []float64, images worldmapcore.TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := PositionToIndex(x, y)
			if index < 0 || index >= len(result.Tiles) {
				continue
			}

			elevation := elevationMap[index]
			moisture := moistureMap[index]
			biome := g.determineBiome(elevation, moisture)

			logicalPos := coords.LogicalPosition{X: x, Y: y}

			tile := result.Tiles[index]
			tile.Biome = biome
			result.BiomeMap[index] = biome

			wallImages, floorImages := GetBiomeImages(images, biome)

			switch biome {
			case worldmapcore.BiomeSwamp, worldmapcore.BiomeMountain:
				tile.TileType = worldmapcore.WALL
				tile.Blocked = true
				tile.Image = SelectRandomImage(wallImages)

			case worldmapcore.BiomeDesert, worldmapcore.BiomeForest, worldmapcore.BiomeGrassland:
				tile.TileType = worldmapcore.FLOOR
				tile.Blocked = false
				tile.Image = SelectRandomImage(floorImages)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// determineBiome classifies terrain based on elevation and moisture
func (g *StrategicOverworldGenerator) determineBiome(elevation, moisture float64) worldmapcore.Biome {
	if elevation < g.config.WaterThresh {
		return worldmapcore.BiomeSwamp
	}
	if elevation > g.config.MountainThresh {
		return worldmapcore.BiomeMountain
	}

	// Low-elevation high-moisture = swamp
	if moisture > g.config.SwampMoisture && elevation < 0.40 {
		return worldmapcore.BiomeSwamp
	}

	// High-ish elevation, dry = desert
	if elevation > 0.60 && moisture < 0.35 {
		return worldmapcore.BiomeDesert
	}

	// Wet = forest
	if moisture > g.config.ForestMoisture {
		return worldmapcore.BiomeForest
	}

	return worldmapcore.BiomeGrassland
}

// buildTerrainMap creates a boolean walkability map from current tile state
func (g *StrategicOverworldGenerator) buildTerrainMap(result *worldmapcore.GenerationResult, width, height int) []bool {
	terrainMap := make([]bool, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := PositionToIndex(x, y)
			if idx < len(result.Tiles) {
				terrainMap[idx] = !result.Tiles[idx].Blocked
			}
		}
	}
	return terrainMap
}

// applyConnectivityFixes updates tiles that were carved by connectivity to be walkable
func (g *StrategicOverworldGenerator) applyConnectivityFixes(result *worldmapcore.GenerationResult, terrainMap []bool, width, height int, elevationMap, moistureMap []float64, images worldmapcore.TileImageSet) {
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := PositionToIndex(x, y)
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
				tile.Biome = worldmapcore.BiomeGrassland
				result.BiomeMap[idx] = worldmapcore.BiomeGrassland
				tile.TileType = worldmapcore.FLOOR
				tile.Blocked = false

				_, floorImages := GetBiomeImages(images, worldmapcore.BiomeGrassland)
				tile.Image = SelectRandomImage(floorImages)
				result.ValidPositions = append(result.ValidPositions, logicalPos)
			}
		}
	}
}

// placeFactionStartPositions divides the map into sectors and finds optimal starting positions
func (g *StrategicOverworldGenerator) placeFactionStartPositions(result *worldmapcore.GenerationResult, width, height int, elevationMap []float64) {
	// Build terrain map for openness scoring
	terrainMap := g.buildTerrainMap(result, width, height)

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
				idx := PositionToIndex(x, y)
				if idx < 0 || idx >= len(result.Tiles) || result.Tiles[idx].Blocked {
					continue
				}

				// Only consider grassland or forest
				biome := result.BiomeMap[idx]
				if biome != worldmapcore.BiomeGrassland && biome != worldmapcore.BiomeForest {
					continue
				}

				// Score by number of walkable neighbors in 5-tile radius
				score := ScoreTerrainOpenness(terrainMap, x, y, 5, width, height)
				if score > bestScore {
					bestScore = score
					bestPos = coords.LogicalPosition{X: x, Y: y}
				}
			}
		}

		if bestScore > 0 {
			result.FactionStartPositions = append(result.FactionStartPositions, worldmapcore.FactionStartPosition{
				Position: bestPos,
				Biome:    result.BiomeMap[PositionToIndex(bestPos.X, bestPos.Y)],
				Sector:   i,
			})
		}
	}
}

// placeTypedPOIs places terrain-aware POIs in order: towns, temples, watchtowers, guild halls
func (g *StrategicOverworldGenerator) placeTypedPOIs(result *worldmapcore.GenerationResult, width, height int, elevationMap, moistureMap []float64, images worldmapcore.TileImageSet) {
	placedPositions := make([]coords.LogicalPosition, 0)

	// Towns first (guild halls depend on them)
	placedTowns := g.placePOIs(result, elevationMap, moistureMap, images, poiPlacementRule{
		nodeID:  worldmapcore.POITown,
		count:   g.config.TownCount,
		minDist: g.config.POIMinDistance,
	}, placedPositions)
	placedPositions = append(placedPositions, placedTowns...)

	// Temples
	placedTemples := g.placePOIs(result, elevationMap, moistureMap, images, poiPlacementRule{
		nodeID:  worldmapcore.POITemple,
		count:   g.config.TempleCount,
		minDist: 15,
	}, placedPositions)
	placedPositions = append(placedPositions, placedTemples...)

	// Watchtowers
	placedWatchtowers := g.placePOIs(result, elevationMap, moistureMap, images, poiPlacementRule{
		nodeID:  worldmapcore.POIWatchtower,
		count:   g.config.WatchtowerCount,
		minDist: 10,
	}, placedPositions)
	placedPositions = append(placedPositions, placedWatchtowers...)

	// Guild halls — same placement, but constrained to be within 20 tiles of a town.
	g.placePOIs(result, elevationMap, moistureMap, images, poiPlacementRule{
		nodeID:  worldmapcore.POIGuildHall,
		count:   g.config.GuildHallCount,
		minDist: g.config.POIMinDistance,
		accept:  withinDistance(placedTowns, 20.0),
	}, placedPositions)
}

// poiPlacementRule parameterizes placePOIs. accept (optional) adds an extra
// per-candidate predicate on top of terrain rules and spacing.
type poiPlacementRule struct {
	nodeID  string
	count   int
	minDist int
	accept  func(pos coords.LogicalPosition) bool
}

// withinDistance returns an accept predicate that requires the candidate to lie
// within maxDist of at least one anchor.
func withinDistance(anchors []coords.LogicalPosition, maxDist float64) func(coords.LogicalPosition) bool {
	if len(anchors) == 0 {
		// No anchors -> never accept
		return func(coords.LogicalPosition) bool { return false }
	}
	return func(pos coords.LogicalPosition) bool {
		for _, a := range anchors {
			if euclideanDistance(pos, a) <= maxDist {
				return true
			}
		}
		return false
	}
}

// placePOIs is the shared POI placement loop: random-pick-and-retry with
// terrain validation, distance spacing, and an optional accept predicate.
func (g *StrategicOverworldGenerator) placePOIs(
	result *worldmapcore.GenerationResult,
	elevationMap, moistureMap []float64,
	images worldmapcore.TileImageSet,
	rule poiPlacementRule,
	existingPOIs []coords.LogicalPosition,
) []coords.LogicalPosition {
	placed := make([]coords.LogicalPosition, 0)
	maxAttempts := rule.count * 50

	for attempt := 0; attempt < maxAttempts && len(placed) < rule.count; attempt++ {
		if len(result.ValidPositions) == 0 {
			break
		}

		validIdx := common.GetRandomBetween(0, len(result.ValidPositions)-1)
		pos := result.ValidPositions[validIdx]
		idx := PositionToIndex(pos.X, pos.Y)

		if idx < 0 || idx >= len(result.Tiles) || result.Tiles[idx].Blocked {
			continue
		}

		biome := result.BiomeMap[idx]

		if !g.isValidPOITerrain(rule.nodeID, biome, elevationMap[idx], moistureMap[idx]) {
			continue
		}

		if rule.accept != nil && !rule.accept(pos) {
			continue
		}

		if IsTooCloseEuclidean(pos, existingPOIs, rule.minDist) || IsTooCloseEuclidean(pos, placed, rule.minDist) {
			continue
		}

		result.POIs = append(result.POIs, worldmapcore.POIData{
			Position: pos,
			NodeID:   rule.nodeID,
			Biome:    biome,
		})

		if poiImg, ok := images.POIImages[rule.nodeID]; ok {
			result.Tiles[idx].POIType = rule.nodeID
			result.Tiles[idx].Image = poiImg
		}

		// Add as a 1x1 Rect for backward compat with StartingPosition()/PlaceStairs()
		result.Rooms = append(result.Rooms, worldmapcore.NewRect(pos.X, pos.Y, 1, 1))
		placed = append(placed, pos)
	}

	return placed
}

// isValidPOITerrain checks terrain-specific placement rules per POI type
func (g *StrategicOverworldGenerator) isValidPOITerrain(nodeID string, biome worldmapcore.Biome, elevation, moisture float64) bool {
	switch nodeID {
	case worldmapcore.POITown:
		// Towns: grassland or forest edge
		return biome == worldmapcore.BiomeGrassland || biome == worldmapcore.BiomeForest

	case worldmapcore.POITemple:
		// Temples: elevated terrain or desert, prefer isolated
		return (elevation > 0.55 && biome != worldmapcore.BiomeSwamp && biome != worldmapcore.BiomeMountain) || biome == worldmapcore.BiomeDesert

	case worldmapcore.POIWatchtower:
		// Watchtowers: elevated walkable terrain near mountain borders
		return elevation > 0.50 && biome != worldmapcore.BiomeSwamp && biome != worldmapcore.BiomeMountain

	default:
		// Guild halls and others: any walkable
		return biome != worldmapcore.BiomeSwamp && biome != worldmapcore.BiomeMountain
	}
}

// euclideanDistance computes Euclidean distance between two positions.
func euclideanDistance(a, b coords.LogicalPosition) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// Register this generator on package initialization
func init() {
	RegisterGenerator(NewStrategicOverworldGenerator(DefaultStrategicOverworldConfig()))
}
