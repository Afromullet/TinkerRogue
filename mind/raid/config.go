package raid

import (
	"encoding/json"
	"fmt"
	"os"

	"game_main/world/coords"
)

// RaidConfig holds the loaded raid configuration. Set by LoadRaidConfig.
var RaidConfig *RaidConfiguration

// RaidConfiguration mirrors the raidconfig.json structure.
type RaidConfiguration struct {
	Raid                RaidSettings                `json:"raid"`
	Recovery            RecoverySettings            `json:"recovery"`
	Alert               AlertSettings               `json:"alert"`
	Morale              MoraleSettings              `json:"morale"`
	Rewards             RewardSettings              `json:"rewards"`
	ArchetypeAssignment ArchetypeAssignmentSettings `json:"archetypeAssignment"`
}

type RaidSettings struct {
	MaxPlayerSquads         int `json:"maxPlayerSquads"`
	MaxDeployedPerEncounter int `json:"maxDeployedPerEncounter"`
	DefaultFloorCount       int `json:"defaultFloorCount"`
	ReservesPerFloor        int `json:"reservesPerFloor"`
	ExtraReservesAfterFloor int `json:"extraReservesAfterFloor"`
	CombatPositionX         int `json:"combatPositionX"`
	CombatPositionY         int `json:"combatPositionY"`
}

type RecoverySettings struct {
	DeployedHPPercent      int `json:"deployedHPPercent"`
	ReserveHPPercent       int `json:"reserveHPPercent"`
	BetweenFloorMoraleBonus int `json:"betweenFloorMoraleBonus"`
	VictoryMoraleBonus     int `json:"victoryMoraleBonus"`
	RestRoomMoraleBonus    int `json:"restRoomMoraleBonus"`
	RestRoomHPPercent      int `json:"restRoomHPPercent"`
	UnitDeathMoralePenalty int `json:"unitDeathMoralePenalty"`
	DefeatMoralePenalty    int `json:"defeatMoralePenalty"`
}

type AlertSettings struct {
	Levels []AlertLevelConfig `json:"levels"`
}

type AlertLevelConfig struct {
	Level              int    `json:"level"`
	Name               string `json:"name"`
	EncounterThreshold int    `json:"encounterThreshold"`
	ArmorBonus         int    `json:"armorBonus"`
	StrengthBonus      int    `json:"strengthBonus"`
	WeaponBonus        int    `json:"weaponBonus"`
	ActivatesReserves  bool   `json:"activatesReserves"`
}

type MoraleSettings struct {
	Thresholds []MoraleThresholdConfig `json:"thresholds"`
}

type MoraleThresholdConfig struct {
	MinMorale  int `json:"minMorale"`
	MaxMorale  int `json:"maxMorale"`
	DexPenalty int `json:"dexPenalty"`
	StrPenalty int `json:"strPenalty"`
}

type RewardSettings struct {
	CommandPostManaRestore int `json:"commandPostManaRestore"`
}

type ArchetypeAssignmentSettings struct {
	CriticalPathArchetypes []string `json:"criticalPathArchetypes"`
	BranchArchetypes       []string `json:"branchArchetypes"`
	EliteArchetypes        []string `json:"eliteArchetypes"`
	EliteFloorThreshold    int      `json:"eliteFloorThreshold"`
	ReserveArchetypes      []string `json:"reserveArchetypes"`
}

// LoadRaidConfig reads and parses raidconfig.json into RaidConfig.
func LoadRaidConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read raid config: %w", err)
	}

	var cfg RaidConfiguration
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse raid config: %w", err)
	}

	RaidConfig = &cfg
	return nil
}

// GetAlertLevel returns the alert config for a given level. Returns nil if not found.
func GetAlertLevel(level int) *AlertLevelConfig {
	if RaidConfig == nil {
		return nil
	}
	for i := range RaidConfig.Alert.Levels {
		if RaidConfig.Alert.Levels[i].Level == level {
			return &RaidConfig.Alert.Levels[i]
		}
	}
	return nil
}

// DefaultFloorCount returns the configured default floor count, defaulting to 3.
func DefaultFloorCount() int {
	if RaidConfig != nil && RaidConfig.Raid.DefaultFloorCount > 0 {
		return RaidConfig.Raid.DefaultFloorCount
	}
	return 3
}

// MaxPlayerSquads returns the configured max player squads, defaulting to 4.
func MaxPlayerSquads() int {
	if RaidConfig != nil && RaidConfig.Raid.MaxPlayerSquads > 0 {
		return RaidConfig.Raid.MaxPlayerSquads
	}
	return 4
}

// MaxDeployedPerEncounter returns the configured max deployed squads, defaulting to 3.
func MaxDeployedPerEncounter() int {
	if RaidConfig != nil && RaidConfig.Raid.MaxDeployedPerEncounter > 0 {
		return RaidConfig.Raid.MaxDeployedPerEncounter
	}
	return 3
}

// CombatPosition returns the configured combat position, defaulting to (50, 40).
func CombatPosition() coords.LogicalPosition {
	if RaidConfig != nil && (RaidConfig.Raid.CombatPositionX > 0 || RaidConfig.Raid.CombatPositionY > 0) {
		return coords.LogicalPosition{X: RaidConfig.Raid.CombatPositionX, Y: RaidConfig.Raid.CombatPositionY}
	}
	return coords.LogicalPosition{X: 50, Y: 40}
}

// ReserveCountForFloor returns the number of reserve squads for a given floor number.
func ReserveCountForFloor(floorNumber int) int {
	base := 1
	threshold := 3
	if RaidConfig != nil {
		if RaidConfig.Raid.ReservesPerFloor > 0 {
			base = RaidConfig.Raid.ReservesPerFloor
		}
		if RaidConfig.Raid.ExtraReservesAfterFloor > 0 {
			threshold = RaidConfig.Raid.ExtraReservesAfterFloor
		}
	}
	if floorNumber >= threshold {
		return base + 1
	}
	return base
}

// CriticalPathArchetypes returns the configured critical path archetype list.
func CriticalPathArchetypes() []string {
	if RaidConfig != nil && len(RaidConfig.ArchetypeAssignment.CriticalPathArchetypes) > 0 {
		return RaidConfig.ArchetypeAssignment.CriticalPathArchetypes
	}
	return []string{"chokepoint_guard", "shield_wall", "orc_vanguard"}
}

// BranchArchetypes returns the configured branch archetype list.
func BranchArchetypes() []string {
	if RaidConfig != nil && len(RaidConfig.ArchetypeAssignment.BranchArchetypes) > 0 {
		return RaidConfig.ArchetypeAssignment.BranchArchetypes
	}
	return []string{"ranged_battery", "fast_response", "ambush_pack"}
}

// EliteArchetypes returns the configured elite archetype list.
func EliteArchetypes() []string {
	if RaidConfig != nil && len(RaidConfig.ArchetypeAssignment.EliteArchetypes) > 0 {
		return RaidConfig.ArchetypeAssignment.EliteArchetypes
	}
	return []string{"mage_tower", "command_post"}
}

// EliteFloorThreshold returns the floor number at which elite archetypes unlock.
func EliteFloorThreshold() int {
	if RaidConfig != nil && RaidConfig.ArchetypeAssignment.EliteFloorThreshold > 0 {
		return RaidConfig.ArchetypeAssignment.EliteFloorThreshold
	}
	return 3
}

// ReserveArchetypes returns the configured reserve archetype list.
func ReserveArchetypes() []string {
	if RaidConfig != nil && len(RaidConfig.ArchetypeAssignment.ReserveArchetypes) > 0 {
		return RaidConfig.ArchetypeAssignment.ReserveArchetypes
	}
	return []string{"fast_response", "ambush_pack"}
}

// GetMoraleThreshold returns the morale debuff tier for a given morale value.
// Returns nil if no threshold matches.
func GetMoraleThreshold(morale int) *MoraleThresholdConfig {
	if RaidConfig == nil {
		return nil
	}
	for i := range RaidConfig.Morale.Thresholds {
		t := &RaidConfig.Morale.Thresholds[i]
		if morale >= t.MinMorale && morale <= t.MaxMorale {
			return t
		}
	}
	return nil
}
