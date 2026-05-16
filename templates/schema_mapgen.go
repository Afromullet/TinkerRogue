package templates

// JSONMapGenConfig is the root container for map generation configuration.
type JSONMapGenConfig struct {
	Generators JSONMapGenerators `json:"generators"`
}

// JSONMapGenerators holds per-generator configuration sections.
type JSONMapGenerators struct {
	RoomsCorridors *JSONRoomsCorridorsConfig `json:"rooms_corridors,omitempty"`
	Cavern         *JSONCavernConfig         `json:"cavern,omitempty"`
	Overworld      *JSONOverworldGenConfig   `json:"overworld,omitempty"`
	GarrisonRaid   *JSONGarrisonRaidConfig   `json:"garrison_raid,omitempty"`
}

// JSONRoomsCorridorsConfig holds rooms-and-corridors generator parameters.
type JSONRoomsCorridorsConfig struct {
	MinRoomSize int `json:"minRoomSize"`
	MaxRoomSize int `json:"maxRoomSize"`
	MaxRooms    int `json:"maxRooms"`
}

// JSONCavernConfig holds cavern generator parameters.
type JSONCavernConfig struct {
	FillDensity       float64 `json:"fillDensity"`
	NumChambers       int     `json:"numChambers"`
	MinChamberRadius  int     `json:"minChamberRadius"`
	MaxChamberRadius  int     `json:"maxChamberRadius"`
	NoiseScale        float64 `json:"noiseScale"`
	ShapeThreshold    float64 `json:"shapeThreshold"`
	CAPassesPhase1    int     `json:"caPassesPhase1"`
	CAPassesPhase2    int     `json:"caPassesPhase2"`
	ErosionPasses     int     `json:"erosionPasses"`
	TunnelBias        float64 `json:"tunnelBias"`
	PillarDensity     float64 `json:"pillarDensity"`
	StalactiteDensity float64 `json:"stalactiteDensity"`
	BorderThickness   int     `json:"borderThickness"`
	TargetWalkableMin float64 `json:"targetWalkableMin"`
	TargetWalkableMax float64 `json:"targetWalkableMax"`
}

// JSONOverworldGenConfig holds strategic overworld generator parameters.
type JSONOverworldGenConfig struct {
	ElevationOctaves  int     `json:"elevationOctaves"`
	ElevationScale    float64 `json:"elevationScale"`
	MoistureOctaves   int     `json:"moistureOctaves"`
	MoistureScale     float64 `json:"moistureScale"`
	Persistence       float64 `json:"persistence"`
	Lacunarity        float64 `json:"lacunarity"`
	WaterThresh       float64 `json:"waterThresh"`
	MountainThresh    float64 `json:"mountainThresh"`
	ForestMoisture   float64 `json:"forestMoisture"`
	SwampMoisture     float64 `json:"swampMoisture"`
	TownCount         int     `json:"townCount"`
	TempleCount       int     `json:"templeCount"`
	GuildHallCount    int     `json:"guildHallCount"`
	WatchtowerCount   int     `json:"watchtowerCount"`
	POIMinDistance    int     `json:"poiMinDistance"`
	FactionCount      int     `json:"factionCount"`
	FactionMinSpacing int     `json:"factionMinSpacing"`
}

// JSONGarrisonRaidConfig holds garrison raid generator parameters.
type JSONGarrisonRaidConfig struct {
	RoomSizes    map[string]JSONRoomSize   `json:"roomSizes"`
	FloorScaling []JSONFloorScaling        `json:"floorScaling"`
	SpawnCounts  map[string]JSONSpawnCount `json:"spawnCounts"`
}

// JSONRoomSize holds min/max width and height for a garrison room type.
type JSONRoomSize struct {
	MinW int `json:"minW"`
	MaxW int `json:"maxW"`
	MinH int `json:"minH"`
	MaxH int `json:"maxH"`
}

// JSONFloorScaling holds per-floor generation parameters for garrison raids.
type JSONFloorScaling struct {
	Floor        int      `json:"floor"`
	MinCritPath  int      `json:"minCritPath"`
	MaxCritPath  int      `json:"maxCritPath"`
	MinTotal     int      `json:"minTotal"`
	MaxTotal     int      `json:"maxTotal"`
	AllowedTypes []string `json:"allowedTypes"`
}

// JSONSpawnCount holds spawn count ranges for player and defender per room type.
type JSONSpawnCount struct {
	MinPlayer   int `json:"minPlayer"`
	MaxPlayer   int `json:"maxPlayer"`
	MinDefender int `json:"minDefender"`
	MaxDefender int `json:"maxDefender"`
}
