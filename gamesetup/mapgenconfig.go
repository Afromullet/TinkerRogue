package gamesetup

import (
	"game_main/templates"
	"game_main/world/worldmap"
)

// InitMapGenConfigOverride registers a ConfigOverride hook that creates
// generators with JSON-configured parameters instead of code defaults.
// Must be called after templates.ReadGameData() has loaded the config.
func InitMapGenConfigOverride() {
	cfg := templates.MapGenConfigTemplate
	if cfg == nil {
		return // No JSON config loaded, use code defaults
	}

	worldmap.ConfigOverride = func(name string) worldmap.MapGenerator {
		switch name {
		case "rooms_corridors":
			return buildRoomsCorridorsGenerator(cfg)
		case "cavern":
			return buildCavernGenerator(cfg)
		case "overworld":
			return buildOverworldGenerator(cfg)
		case "military_base":
			return buildMilitaryBaseGenerator(cfg)
		case "garrison_raid":
			return buildGarrisonRaidGenerator(cfg)
		default:
			return nil // Unknown generator, fall back to registry
		}
	}

	// Apply garrison data table overrides (these are package-level vars, not per-instance)
	applyGarrisonTableOverrides(cfg)
}

func buildRoomsCorridorsGenerator(cfg *templates.JSONMapGenConfig) worldmap.MapGenerator {
	jrc := cfg.Generators.RoomsCorridors
	if jrc == nil {
		return nil
	}

	config := worldmap.DefaultRoomsCorridorsConfig()
	config.MinRoomSize = jrc.MinRoomSize
	config.MaxRoomSize = jrc.MaxRoomSize
	config.MaxRooms = jrc.MaxRooms
	return worldmap.NewRoomsAndCorridorsGenerator(config)
}

func buildCavernGenerator(cfg *templates.JSONMapGenConfig) worldmap.MapGenerator {
	jc := cfg.Generators.Cavern
	if jc == nil {
		return nil
	}

	config := worldmap.CavernConfig{
		FillDensity:       jc.FillDensity,
		NumChambers:       jc.NumChambers,
		MinChamberRadius:  jc.MinChamberRadius,
		MaxChamberRadius:  jc.MaxChamberRadius,
		NoiseScale:        jc.NoiseScale,
		ShapeThreshold:    jc.ShapeThreshold,
		CAPassesPhase1:    jc.CAPassesPhase1,
		CAPassesPhase2:    jc.CAPassesPhase2,
		ErosionPasses:     jc.ErosionPasses,
		TunnelBias:        jc.TunnelBias,
		PillarDensity:     jc.PillarDensity,
		StalactiteDensity: jc.StalactiteDensity,
		BorderThickness:   jc.BorderThickness,
		TargetWalkableMin: jc.TargetWalkableMin,
		TargetWalkableMax: jc.TargetWalkableMax,
	}
	return worldmap.NewCavernGenerator(config)
}

func buildOverworldGenerator(cfg *templates.JSONMapGenConfig) worldmap.MapGenerator {
	jow := cfg.Generators.Overworld
	if jow == nil {
		return nil
	}

	config := worldmap.StrategicOverworldConfig{
		ElevationOctaves:  jow.ElevationOctaves,
		ElevationScale:    jow.ElevationScale,
		MoistureOctaves:   jow.MoistureOctaves,
		MoistureScale:     jow.MoistureScale,
		Persistence:       jow.Persistence,
		Lacunarity:        jow.Lacunarity,
		WaterThresh:       jow.WaterThresh,
		MountainThresh:    jow.MountainThresh,
		ForestMoisture:    jow.ForestMoisture,
		SwampMoisture:     jow.SwampMoisture,
		TownCount:         jow.TownCount,
		TempleCount:       jow.TempleCount,
		GuildHallCount:    jow.GuildHallCount,
		WatchtowerCount:   jow.WatchtowerCount,
		POIMinDistance:     jow.POIMinDistance,
		FactionCount:      jow.FactionCount,
		FactionMinSpacing: jow.FactionMinSpacing,
	}
	return worldmap.NewStrategicOverworldGenerator(config)
}

func buildMilitaryBaseGenerator(cfg *templates.JSONMapGenConfig) worldmap.MapGenerator {
	jmb := cfg.Generators.MilitaryBase
	if jmb == nil {
		return nil
	}

	config := worldmap.MilitaryBaseConfig{
		Biome:             worldmap.BiomeFromString(jmb.Biome),
		PerimeterInset:    jmb.PerimeterInset,
		WallThickness:     jmb.WallThickness,
		GateWidth:         jmb.GateWidth,
		GateSide:          jmb.GateSide,
		NumGuardTowers:    jmb.NumGuardTowers,
		GuardTowerSize:    jmb.GuardTowerSize,
		DrillYardMinRatio: jmb.DrillYardMinRatio,
		NumSupplyAreas:    jmb.NumSupplyAreas,
		SupplyAreaMinSize: jmb.SupplyAreaMinSize,
		SupplyAreaMaxSize: jmb.SupplyAreaMaxSize,
		CoverDensity:      jmb.CoverDensity,
		NumPOIScatter:     jmb.NumPOIScatter,
		BorderThickness:   jmb.BorderThickness,
	}
	return worldmap.NewMilitaryBaseGenerator(config)
}

func buildGarrisonRaidGenerator(cfg *templates.JSONMapGenConfig) worldmap.MapGenerator {
	// Garrison raid uses code defaults for FloorNumber/Seed (runtime values).
	// The data tables (room sizes, floor scaling, spawn counts) are set via
	// applyGarrisonTableOverrides, not per-instance config.
	if cfg.Generators.GarrisonRaid == nil {
		return nil
	}
	return nil // Use registry default; tables are already overridden
}

func applyGarrisonTableOverrides(cfg *templates.JSONMapGenConfig) {
	jgr := cfg.Generators.GarrisonRaid
	if jgr == nil {
		return
	}

	// Override room sizes
	if len(jgr.RoomSizes) > 0 {
		sizes := make(map[string][4]int, len(jgr.RoomSizes))
		for key, rs := range jgr.RoomSizes {
			sizes[key] = [4]int{rs.MinW, rs.MaxW, rs.MinH, rs.MaxH}
		}
		worldmap.SetGarrisonRoomSizes(sizes)
	}

	// Override floor scaling
	if len(jgr.FloorScaling) > 0 {
		scaling := make(map[int]worldmap.FloorScalingEntry, len(jgr.FloorScaling))
		for _, fs := range jgr.FloorScaling {
			scaling[fs.Floor] = worldmap.FloorScalingEntry{
				MinCriticalPath: fs.MinCritPath,
				MaxCriticalPath: fs.MaxCritPath,
				MinTotalRooms:   fs.MinTotal,
				MaxTotalRooms:   fs.MaxTotal,
				AllowedTypes:    fs.AllowedTypes,
			}
		}
		worldmap.SetGarrisonFloorScaling(scaling)
	}

	// Override spawn counts
	if len(jgr.SpawnCounts) > 0 {
		counts := make(map[string][4]int, len(jgr.SpawnCounts))
		for key, sc := range jgr.SpawnCounts {
			counts[key] = [4]int{sc.MinPlayer, sc.MaxPlayer, sc.MinDefender, sc.MaxDefender}
		}
		worldmap.SetGarrisonSpawnCounts(counts)
	}
}
