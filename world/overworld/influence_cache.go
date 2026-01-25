package overworld

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// InfluenceCache optimizes influence calculations using dirty flagging
// Caches influence values per tile and only recalculates when threats change
type InfluenceCache struct {
	// Cache structure: map[tileIndex]influenceValue
	cachedInfluence map[int]float64

	// Dirty flag: true if cache needs rebuild
	isDirty bool

	// Tracked threats: entityID -> last known intensity
	trackedThreats map[ecs.EntityID]int

	// Map dimensions
	mapWidth  int
	mapHeight int
}

// NewInfluenceCache creates an optimized influence cache
func NewInfluenceCache(mapWidth, mapHeight int) *InfluenceCache {
	return &InfluenceCache{
		cachedInfluence: make(map[int]float64),
		isDirty:         true, // Start dirty to force initial calculation
		trackedThreats:  make(map[ecs.EntityID]int),
		mapWidth:        mapWidth,
		mapHeight:       mapHeight,
	}
}

// GetInfluenceAt returns cached influence value for a tile
// Rebuilds cache if dirty flag is set
func (ic *InfluenceCache) GetInfluenceAt(
	manager *common.EntityManager,
	position coords.LogicalPosition,
) float64 {
	// Rebuild if dirty
	if ic.isDirty {
		ic.RebuildCache(manager)
	}

	// Convert position to index
	tileIndex := ic.positionToIndex(position)

	// Return cached value (0.0 if not found)
	if influence, exists := ic.cachedInfluence[tileIndex]; exists {
		return influence
	}

	return 0.0
}

// MarkDirty flags cache for rebuild
// Call this when threats spawn, evolve, or are destroyed
func (ic *InfluenceCache) MarkDirty() {
	ic.isDirty = true
}

// RebuildCache recalculates influence for all tiles
// Only called when isDirty == true
func (ic *InfluenceCache) RebuildCache(manager *common.EntityManager) {
	// Clear old cache
	ic.cachedInfluence = make(map[int]float64)

	// Track threat changes
	currentThreats := make(map[ecs.EntityID]int)

	// Iterate all threats and calculate influence
	for _, result := range manager.World.Query(ThreatNodeTag) {
		entity := result.Entity
		threatID := entity.GetID()

		threatData := common.GetComponentType[*ThreatNodeData](entity, ThreatNodeComponent)
		influenceData := common.GetComponentType[*InfluenceData](entity, InfluenceComponent)
		posData := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)

		if threatData == nil || influenceData == nil || posData == nil {
			continue
		}

		// Track this threat
		currentThreats[threatID] = threatData.Intensity

		// Calculate influence radius
		radius := influenceData.Radius
		effectStrength := influenceData.EffectStrength

		// Apply influence to all tiles in radius
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				targetPos := coords.LogicalPosition{
					X: posData.X + dx,
					Y: posData.Y + dy,
				}

				// Check bounds
				if targetPos.X < 0 || targetPos.X >= ic.mapWidth ||
					targetPos.Y < 0 || targetPos.Y >= ic.mapHeight {
					continue
				}

				// Calculate distance
				distance := posData.ManhattanDistance(&targetPos)
				if distance > radius {
					continue
				}

				// Calculate influence falloff
				falloff := 1.0 - (float64(distance) / float64(radius+1))
				influence := effectStrength * falloff

				// Add to cached influence (multiple threats can overlap)
				tileIndex := ic.positionToIndex(targetPos)
				ic.cachedInfluence[tileIndex] += influence
			}
		}
	}

	// Update tracked threats
	ic.trackedThreats = currentThreats

	// Cache is now clean
	ic.isDirty = false
}

// CheckForChanges detects if threats have changed since last cache
// Returns true if cache should be marked dirty
func (ic *InfluenceCache) CheckForChanges(manager *common.EntityManager) bool {
	currentThreats := make(map[ecs.EntityID]int)

	// Scan current threats
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil {
			currentThreats[result.Entity.GetID()] = threatData.Intensity
		}
	}

	// Check if threat count changed
	if len(currentThreats) != len(ic.trackedThreats) {
		return true
	}

	// Check if any threat intensity changed
	for threatID, intensity := range currentThreats {
		if oldIntensity, exists := ic.trackedThreats[threatID]; !exists || oldIntensity != intensity {
			return true
		}
	}

	return false
}

// Update checks for changes and marks dirty if needed
// Call this every tick or before querying influence
func (ic *InfluenceCache) Update(manager *common.EntityManager) {
	if ic.CheckForChanges(manager) {
		ic.MarkDirty()
	}
}

// GetTilesAboveInfluence returns all tiles with influence >= threshold
// Useful for finding high-danger zones
func (ic *InfluenceCache) GetTilesAboveInfluence(
	manager *common.EntityManager,
	threshold float64,
) []coords.LogicalPosition {
	// Rebuild if dirty
	if ic.isDirty {
		ic.RebuildCache(manager)
	}

	var tiles []coords.LogicalPosition

	for tileIndex, influence := range ic.cachedInfluence {
		if influence >= threshold {
			pos := ic.indexToPosition(tileIndex)
			tiles = append(tiles, pos)
		}
	}

	return tiles
}

// GetMaxInfluence returns the highest influence value on the map
func (ic *InfluenceCache) GetMaxInfluence(manager *common.EntityManager) float64 {
	// Rebuild if dirty
	if ic.isDirty {
		ic.RebuildCache(manager)
	}

	maxInfluence := 0.0
	for _, influence := range ic.cachedInfluence {
		if influence > maxInfluence {
			maxInfluence = influence
		}
	}

	return maxInfluence
}

// positionToIndex converts logical position to cache key
func (ic *InfluenceCache) positionToIndex(pos coords.LogicalPosition) int {
	return pos.Y*ic.mapWidth + pos.X
}

// indexToPosition converts cache key back to position
func (ic *InfluenceCache) indexToPosition(index int) coords.LogicalPosition {
	return coords.LogicalPosition{
		X: index % ic.mapWidth,
		Y: index / ic.mapWidth,
	}
}

// GetCacheSize returns number of cached tiles
func (ic *InfluenceCache) GetCacheSize() int {
	return len(ic.cachedInfluence)
}

// IsDirty returns dirty flag state
func (ic *InfluenceCache) IsDirty() bool {
	return ic.isDirty
}

// Clear empties the cache
func (ic *InfluenceCache) Clear() {
	ic.cachedInfluence = make(map[int]float64)
	ic.trackedThreats = make(map[ecs.EntityID]int)
	ic.isDirty = true
}

// CreateInfluenceCacheEntity creates an ECS-managed singleton for the influence cache
func CreateInfluenceCacheEntity(manager *common.EntityManager, mapWidth, mapHeight int) ecs.EntityID {
	entity := manager.World.NewEntity()
	entityID := entity.GetID()

	cache := NewInfluenceCache(mapWidth, mapHeight)
	cacheData := &InfluenceCacheData{
		Cache: cache,
	}

	entity.AddComponent(InfluenceCacheComponent, cacheData)

	return entityID
}

// GetInfluenceCache retrieves the singleton influence cache
func GetInfluenceCache(manager *common.EntityManager) *InfluenceCache {
	for _, result := range manager.World.Query(InfluenceCacheTag) {
		cacheData := common.GetComponentType[*InfluenceCacheData](result.Entity, InfluenceCacheComponent)
		if cacheData != nil {
			return cacheData.Cache
		}
	}
	return nil
}

// GetOrCreateInfluenceCache retrieves the singleton or creates one if missing
func GetOrCreateInfluenceCache(manager *common.EntityManager, mapWidth, mapHeight int) *InfluenceCache {
	cache := GetInfluenceCache(manager)
	if cache == nil {
		CreateInfluenceCacheEntity(manager, mapWidth, mapHeight)
		cache = GetInfluenceCache(manager)
	}
	return cache
}
