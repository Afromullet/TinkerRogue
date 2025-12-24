# Worldmap Tile Image Selection Guide

This document explains how each map generator in the worldmap package selects which tile images to use.

---

## Common Image Selection Pattern

All generators follow this general approach:

1. **Start with defaults**: Use `images.WallImages` and `images.FloorImages` from the `TileImageSet`
2. **Check for biome-specific images**: If applicable, look up `images.BiomeImages[biome]`
3. **Override if available**: Replace defaults with biome-specific images if they exist
4. **Random selection**: Pick a random image from the available array using `common.GetRandomBetween(0, len(images)-1)`

---

## Generator-Specific Implementations

### 1. rooms_corridors (Classic Roguelike)

**Biome System**: None - does not use biomes

**Image Selection**:
- Uses **default images only** (no biome-specific images)
- Wall images: `images.WallImages`
- Floor images: `images.FloorImages`

**Selection Flow**:
```
createEmptyTiles() → Randomly selects from images.WallImages
carveRoom()        → Randomly selects from images.FloorImages
carveHorizontalTunnel() → Randomly selects from images.FloorImages
carveVerticalTunnel()   → Randomly selects from images.FloorImages
```

**Key Characteristics**:
- Simplest image selection
- All walls look similar, all floors look similar
- No environmental variety

---

### 2. overworld (Noise-Based World Map)

**Biome System**: Per-tile biome determination using elevation + moisture noise

**Image Selection**:
- **Biome determination**: Each tile's biome is determined by:
  - Elevation noise value (0.0-1.0)
  - Moisture noise value (0.0-1.0)
  - Thresholds: Water (<0.3), Mountain (>0.6), Forest (moisture >0.55), Desert, Grassland
- **Image lookup**: Attempts biome-specific images with fallback to defaults

**Selection Flow** (per tile):
```go
// 1. Start with defaults
wallImages := images.WallImages
floorImages := images.FloorImages

// 2. Check for biome-specific images
biomeTileSet := images.BiomeImages[biome]
if biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        wallImages = biomeTileSet.WallImages
    }
    if len(biomeTileSet.FloorImages) > 0 {
        floorImages = biomeTileSet.FloorImages
    }
}

// 3. Select random image from chosen array
randomImage = wallImages[common.GetRandomBetween(0, len(wallImages)-1)]
```

**Biome Usage**:
- BiomeSwamp (water) → Walls (impassable)
- BiomeMountain → Walls (impassable)
- BiomeDesert → Floors (walkable)
- BiomeForest → Floors (walkable)
- BiomeGrassland → Floors (walkable)

**Key Characteristics**:
- Each tile can have a different biome
- Natural-looking biome transitions
- Visual variety across the map

---

### 3. tactical_biome (Cellular Automata Tactical)

**Biome System**: Single random biome per map

**Image Selection**:
- **Biome determination**: One biome chosen randomly at generation start from:
  - BiomeGrassland, BiomeForest, BiomeDesert, BiomeMountain, BiomeSwamp
- **Image lookup**: Same pattern as overworld (biome-specific with fallback)

**Selection Flow**:
```go
// 1. Select biome once (entire map uses same biome)
biome := selectBiome() // Random choice

// 2. Get biome-specific images (same pattern as overworld)
biomeTileSet := images.BiomeImages[biome]
wallImages := images.WallImages
floorImages := images.FloorImages

if biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        wallImages = biomeTileSet.WallImages
    }
    if len(biomeTileSet.FloorImages) > 0 {
        floorImages = biomeTileSet.FloorImages
    }
}

// 3. Random selection per tile from biome's image arrays
```

**Key Characteristics**:
- Entire map uses one consistent biome
- All tiles share the same image palette
- Uniform aesthetic per map

---

### 4. hybrid_tactical (Multi-Layer Perlin + Domain Warping + Voronoi)

**Biome System**: Per-tile biome determination using warped noise

**Image Selection**:
- **Biome determination**: Each tile's biome determined by:
  - Warped elevation noise
  - Moisture noise
  - Combined with domain warping for organic shapes
- **Image lookup**: Identical to overworld pattern

**Selection Flow**:
```go
// 1. Determine biome per tile
biome := biomeMap[y][x]

// 2. Get biome-specific images (same as overworld)
wallImages := images.WallImages
floorImages := images.FloorImages

biomeTileSet := images.BiomeImages[biome]
if biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        wallImages = biomeTileSet.WallImages
    }
    if len(biomeTileSet.FloorImages) > 0 {
        floorImages = biomeTileSet.FloorImages
    }
}

// 3. Random selection from biome's images
```

**Key Characteristics**:
- Variable biomes across the map
- Organic biome transitions (domain warping)
- Same image selection logic as overworld

---

### 5. wavelet_procedural (Wavelet Noise Procedural)

**Biome System**: Per-tile biome determination using wavelet noise

**Image Selection**:
- **Biome determination**: Each tile's biome determined by:
  - Wavelet elevation noise
  - Wavelet moisture noise
  - Multiple frequency bands for detail
- **Image lookup**: Identical to overworld/hybrid pattern

**Selection Flow**:
```go
// Identical to hybrid_tactical
biome := biomeMap[y][x]

wallImages := images.WallImages
floorImages := images.FloorImages

biomeTileSet := images.BiomeImages[biome]
if biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        wallImages = biomeTileSet.WallImages
    }
    if len(biomeTileSet.FloorImages) > 0 {
        floorImages = biomeTileSet.FloorImages
    }
}
```

**Key Characteristics**:
- Per-tile biome variation
- Sharper biome boundaries (wavelet vs Perlin)
- Same fallback pattern as other biome-based generators

---

### 6. cave_tactical (Underground Cave Systems)

**Biome System**: Fixed biomes (not determined by noise)

**Image Selection**:
- **Biome assignment**:
  - All cave walls → `BiomeMountain` (rocky appearance)
  - All cave floors → `BiomeMountain` (default) or `BiomeSwamp` (for pools)
- **Two-stage lookup**: Gets mountain images, then optionally overrides floors with swamp for pools

**Selection Flow**:
```go
// 1. Fixed biome assignments
caveBiome := BiomeMountain
poolBiome := BiomeSwamp

// 2. Get cave wall/floor images from mountain biome
caveWallImages := images.WallImages
caveFloorImages := images.FloorImages

if biomeTileSet := images.BiomeImages[caveBiome]; biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        caveWallImages = biomeTileSet.WallImages
    }
    if len(biomeTileSet.FloorImages) > 0 {
        caveFloorImages = biomeTileSet.FloorImages
    }
}

// 3. Get pool images from swamp biome
poolFloorImages := caveFloorImages
if biomeTileSet := images.BiomeImages[poolBiome]; biomeTileSet != nil {
    if len(biomeTileSet.FloorImages) > 0 {
        poolFloorImages = biomeTileSet.FloorImages
    }
}

// 4. Select image based on terrain type
if terrainMap[idx] {
    // Floor tile
    floorImages := caveFloorImages
    if poolMap[idx] {
        floorImages = poolFloorImages  // Use swamp images for pools
    }
    image = floorImages[random]
}
```

**Key Characteristics**:
- Fixed biome selection (not dynamic)
- Two biomes used: Mountain for cave, Swamp for water pools
- Pool detection adds visual variety within caves

---

## Image Fallback Hierarchy

All generators follow this fallback pattern:

1. **Try biome-specific images**: `images.BiomeImages[biome].WallImages/FloorImages`
2. **Fall back to defaults**: `images.WallImages/FloorImages`
3. **Handle missing images**: Check `len(images) > 0` before selecting

Example fallback code (used by all biome-based generators):
```go
wallImages := images.WallImages  // Default
biomeTileSet := images.BiomeImages[biome]
if biomeTileSet != nil {
    if len(biomeTileSet.WallImages) > 0 {
        wallImages = biomeTileSet.WallImages  // Override with biome-specific
    }
}

// Now wallImages either contains biome-specific images or defaults
```

---

## Random Selection Method

All generators use the same random selection:
```go
if len(images) > 0 {
    selectedImage = images[common.GetRandomBetween(0, len(images)-1)]
}
```

- `common.GetRandomBetween(min, max)` returns inclusive random integer
- Each tile independently selects a random image from available array
- No spatial correlation in image selection (pure random per tile)

---

## Summary Table

| Generator | Biome System | Images Used | Variety |
|-----------|--------------|-------------|---------|
| rooms_corridors | None | Defaults only | None - uniform look |
| overworld | Per-tile (noise) | Biome-specific + defaults | High - varied biomes |
| tactical_biome | Single random | Biome-specific + defaults | Medium - one biome |
| hybrid_tactical | Per-tile (warped noise) | Biome-specific + defaults | High - varied biomes |
| wavelet_procedural | Per-tile (wavelet) | Biome-specific + defaults | High - varied biomes |
| cave_tactical | Fixed (Mountain/Swamp) | Biome-specific + defaults | Low - cave aesthetic |

---

## Asset Requirements

For generators to use biome-specific images, assets must be organized as:
```
assets/tiles/
├── floors/
│   ├── limestone/        (default)
│   ├── grassland/
│   ├── forest/
│   ├── desert/
│   ├── mountain/
│   └── swamp/
└── walls/
    ├── marble/           (default)
    ├── grassland/
    ├── forest/
    ├── desert/
    ├── mountain/
    └── swamp/
```

If biome-specific folders don't exist or are empty, generators automatically fall back to default images.
