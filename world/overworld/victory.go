package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// VictoryCondition represents win/loss state
type VictoryCondition int

const (
	VictoryNone          VictoryCondition = iota // Game in progress
	VictoryPlayerWins                            // Player eliminated all threats/factions
	VictoryPlayerLoses                           // Player overwhelmed
	VictoryTimeLimit                             // Survived N ticks
	VictoryFactionDefeat                         // Defeated specific faction
)

func (v VictoryCondition) String() string {
	switch v {
	case VictoryNone:
		return "In Progress"
	case VictoryPlayerWins:
		return "Victory!"
	case VictoryPlayerLoses:
		return "Defeat"
	case VictoryTimeLimit:
		return "Survival Victory"
	case VictoryFactionDefeat:
		return "Faction Defeated"
	default:
		return "Unknown"
	}
}

// VictoryStateData tracks victory condition progress
type VictoryStateData struct {
	Condition         VictoryCondition
	TicksToSurvive    int64 // For survival victory
	TargetFactionType FactionType
	VictoryAchieved   bool
	DefeatReason      string
}

// SquadChecker is an interface for checking squad status without circular dependency
// This interface allows the overworld package to query squad status without importing the squads package
type SquadChecker interface {
	// HasActiveSquads returns true if the player has any squads with living units
	HasActiveSquads(manager *common.EntityManager) bool
}

// squadChecker is the injected implementation for squad checking
// Set this in main package initialization via SetSquadChecker()
var squadChecker SquadChecker

// SetSquadChecker injects the squad checking implementation
// Call this from main package after squads package is initialized
func SetSquadChecker(checker SquadChecker) {
	squadChecker = checker
}

// CheckVictoryCondition evaluates if player has won or lost
func CheckVictoryCondition(manager *common.EntityManager) VictoryCondition {
	// Get victory state (if configured)
	victoryState := GetVictoryState(manager)

	// Check defeat conditions first (highest priority)
	if IsPlayerDefeated(manager) {
		if victoryState != nil {
			victoryState.Condition = VictoryPlayerLoses
			victoryState.VictoryAchieved = true
			victoryState.DefeatReason = GetDefeatReason(manager)
		}

		// Log defeat event
		tickState := GetTickState(manager)
		currentTick := int64(0)
		if tickState != nil {
			currentTick = tickState.CurrentTick
		}
		LogEvent(EventDefeat, currentTick, 0, GetDefeatReason(manager))

		return VictoryPlayerLoses
	}

	// Check survival victory first (if configured) - takes priority over threat elimination
	if victoryState != nil && victoryState.TicksToSurvive > 0 {
		tickState := GetTickState(manager)
		if tickState != nil && tickState.CurrentTick >= victoryState.TicksToSurvive {
			victoryState.Condition = VictoryTimeLimit
			victoryState.VictoryAchieved = true

			// Log victory event
			LogEvent(EventVictory, tickState.CurrentTick, 0,
				formatEventString("Victory! Survived %d ticks", victoryState.TicksToSurvive))

			return VictoryTimeLimit
		}
		// Still surviving - game continues
		return VictoryNone
	}

	// Check threat elimination victory (only if no survival condition)
	if HasPlayerEliminatedAllThreats(manager) {
		if victoryState != nil {
			victoryState.Condition = VictoryPlayerWins
			victoryState.VictoryAchieved = true
		}

		// Log victory event
		tickState := GetTickState(manager)
		currentTick := int64(0)
		if tickState != nil {
			currentTick = tickState.CurrentTick
		}
		LogEvent(EventVictory, currentTick, 0, "Victory! All threats eliminated")

		return VictoryPlayerWins
	}

	// Check faction-specific victory (if configured)
	if victoryState != nil && victoryState.TargetFactionType != FactionType(0) {
		if HasPlayerDefeatedFactionType(manager, victoryState.TargetFactionType) {
			victoryState.Condition = VictoryFactionDefeat
			victoryState.VictoryAchieved = true

			// Log victory event
			tickState := GetTickState(manager)
			currentTick := int64(0)
			if tickState != nil {
				currentTick = tickState.CurrentTick
			}
			LogEvent(EventVictory, currentTick, 0,
				formatEventString("Victory! Defeated all %s factions", victoryState.TargetFactionType.String()))

			return VictoryFactionDefeat
		}
	}

	return VictoryNone
}

// IsPlayerDefeated checks if player has lost
func IsPlayerDefeated(manager *common.EntityManager) bool {
	// Player loses if threat influence is too high
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > 100.0 {
		return true
	}

	// Player loses if too many high-intensity threats exist
	highIntensityCount := 0
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil && threatData.Intensity >= 8 {
			highIntensityCount++
		}
	}

	if highIntensityCount >= 10 {
		return true // 10+ tier-8 threats = overwhelming
	}

	// Player loses if all squads destroyed
	if HasPlayerLostAllSquads(manager) {
		return true
	}

	return false
}

// GetDefeatReason returns a human-readable defeat reason
func GetDefeatReason(manager *common.EntityManager) string {
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > 100.0 {
		return formatEventString("Defeat! Overwhelmed by threat influence (%.1f)", totalInfluence)
	}

	highIntensityCount := 0
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil && threatData.Intensity >= 8 {
			highIntensityCount++
		}
	}

	if highIntensityCount >= 10 {
		return formatEventString("Defeat! Too many powerful threats (%d tier-8+ threats)", highIntensityCount)
	}

	if HasPlayerLostAllSquads(manager) {
		return "Defeat! All squads destroyed"
	}

	return "Defeat! Unknown reason"
}

// HasPlayerLostAllSquads checks if player has any surviving squads
func HasPlayerLostAllSquads(manager *common.EntityManager) bool {
	// If no squad checker is injected, assume player hasn't lost
	// (squad-based defeat is optional feature)
	if squadChecker == nil {
		return false
	}

	// Invert the checker result: HasActiveSquads=false means all squads lost
	return !squadChecker.HasActiveSquads(manager)
}

// HasPlayerEliminatedAllThreats checks if all threats are gone
func HasPlayerEliminatedAllThreats(manager *common.EntityManager) bool {
	threatCount := CountThreatNodes(manager)
	return threatCount == 0
}

// HasPlayerDefeatedFactionType checks if specific faction type is eliminated
func HasPlayerDefeatedFactionType(manager *common.EntityManager, factionType FactionType) bool {
	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil && factionData.FactionType == factionType {
			return false // Faction still exists
		}
	}
	return true // No factions of this type found
}

// GetTotalThreatInfluence calculates combined threat pressure
func GetTotalThreatInfluence(manager *common.EntityManager) float64 {
	total := 0.0

	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		influenceData := common.GetComponentType[*InfluenceData](result.Entity, InfluenceComponent)

		if threatData != nil && influenceData != nil {
			// Influence value = intensity × radius × strength
			influence := float64(threatData.Intensity) * float64(influenceData.Radius) * influenceData.EffectStrength
			total += influence
		}
	}

	return total
}

// CreateVictoryStateEntity creates singleton victory tracking entity
func CreateVictoryStateEntity(
	manager *common.EntityManager,
	ticksToSurvive int64,
	targetFactionType FactionType,
) ecs.EntityID {
	entity := manager.World.NewEntity()

	entity.AddComponent(VictoryStateComponent, &VictoryStateData{
		Condition:         VictoryNone,
		TicksToSurvive:    ticksToSurvive,
		TargetFactionType: targetFactionType,
		VictoryAchieved:   false,
		DefeatReason:      "",
	})

	return entity.GetID()
}

// GetVictoryState retrieves singleton victory state
func GetVictoryState(manager *common.EntityManager) *VictoryStateData {
	for _, result := range manager.World.Query(VictoryStateTag) {
		return common.GetComponentType[*VictoryStateData](result.Entity, VictoryStateComponent)
	}
	return nil
}

// GetVictoryProgress returns descriptive text about victory condition status
func GetVictoryProgress(manager *common.EntityManager) string {
	victoryState := GetVictoryState(manager)
	if victoryState == nil {
		return "No victory condition set"
	}

	if victoryState.VictoryAchieved {
		return victoryState.Condition.String()
	}

	// Show progress towards victory
	switch victoryState.Condition {
	case VictoryNone:
		threatCount := CountThreatNodes(manager)
		return formatString("Threats remaining: %d", threatCount)

	case VictoryTimeLimit:
		tickState := GetTickState(manager)
		if tickState != nil {
			remaining := victoryState.TicksToSurvive - tickState.CurrentTick
			return formatString("Survive %d more ticks", remaining)
		}

	case VictoryFactionDefeat:
		factionCount := len(GetFactionsByType(manager, victoryState.TargetFactionType))
		return formatString("%s factions remaining: %d", victoryState.TargetFactionType, factionCount)
	}

	return "In Progress"
}

// formatString is a simple helper to avoid fmt import
func formatString(format string, args ...interface{}) string {
	// Simplified - just return format for now
	// In real implementation, use fmt.Sprintf
	return format
}
