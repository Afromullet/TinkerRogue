package influence

import (
	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// ClassifyInteraction determines the interaction type between two nodes.
// Uses OwnerID from the unified OverworldNodeData for classification.
func ClassifyInteraction(manager *common.EntityManager, entityA, entityB *ecs.Entity) core.InteractionType {
	dataA := common.GetComponentType[*core.OverworldNodeData](entityA, core.OverworldNodeComponent)
	dataB := common.GetComponentType[*core.OverworldNodeData](entityB, core.OverworldNodeComponent)

	if dataA == nil || dataB == nil {
		return core.InteractionCompetition
	}

	aHostile := core.IsHostileOwner(dataA.OwnerID)
	bHostile := core.IsHostileOwner(dataB.OwnerID)

	// Both hostile (threat factions)
	if aHostile && bHostile {
		if dataA.OwnerID == dataB.OwnerID {
			return core.InteractionSynergy
		}
		return core.InteractionCompetition
	}

	// Both friendly/neutral (player or neutral)
	if !aHostile && !bHostile {
		return core.InteractionPlayerBoost
	}

	// Mixed (one friendly/neutral + one hostile) -> suppression
	return core.InteractionSuppression
}

// CalculateInteractionModifier returns the additive modifier for a given interaction.
// Positive = boost, negative = suppress. Added to 1.0 base during finalization.
func CalculateInteractionModifier(
	manager *common.EntityManager,
	interaction core.InteractionType,
	entityA, entityB *ecs.Entity,
) float64 {
	switch interaction {
	case core.InteractionSynergy:
		return calculateSynergyBonus()
	case core.InteractionCompetition:
		return calculateCompetitionPenalty()
	case core.InteractionSuppression:
		return calculateSuppressionPenalty(manager, entityA, entityB)
	case core.InteractionPlayerBoost:
		return calculatePlayerSynergyBonus(entityA, entityB)
	default:
		return 0.0
	}
}

// calculateSynergyBonus returns flat growth bonus for same-faction threats.
func calculateSynergyBonus() float64 {
	return getSynergyGrowthBonus()
}

// calculateCompetitionPenalty returns flat growth penalty for rival-faction threats.
func calculateCompetitionPenalty() float64 {
	return -getCompetitionGrowthPenalty()
}

// calculateSuppressionPenalty returns growth penalty from player/neutral nodes on threats.
// Scaled by node type multiplier. Uses unified OverworldNodeData.
func calculateSuppressionPenalty(manager *common.EntityManager, entityA, entityB *ecs.Entity) float64 {
	// Find the friendly/neutral entity (the suppressor)
	dataA := common.GetComponentType[*core.OverworldNodeData](entityA, core.OverworldNodeComponent)
	dataB := common.GetComponentType[*core.OverworldNodeData](entityB, core.OverworldNodeComponent)

	var suppressorData *core.OverworldNodeData
	if dataA != nil && !core.IsHostileOwner(dataA.OwnerID) {
		suppressorData = dataA
	} else if dataB != nil && !core.IsHostileOwner(dataB.OwnerID) {
		suppressorData = dataB
	}

	nodeTypeMult := 1.0
	if suppressorData != nil {
		nodeTypeMult = getSuppressionNodeTypeMultiplier(suppressorData.NodeTypeID)
	}

	return -getSuppressionGrowthPenalty() * nodeTypeMult
}

// calculatePlayerSynergyBonus computes bonus for adjacent friendly/neutral nodes.
// Returns base bonus, or complementary bonus if types are a complementary pair.
// Uses unified OverworldNodeData.
func calculatePlayerSynergyBonus(entityA, entityB *ecs.Entity) float64 {
	dataA := common.GetComponentType[*core.OverworldNodeData](entityA, core.OverworldNodeComponent)
	dataB := common.GetComponentType[*core.OverworldNodeData](entityB, core.OverworldNodeComponent)
	if dataA != nil && dataB != nil && isComplementaryPair(dataA.NodeTypeID, dataB.NodeTypeID) {
		return getPlayerSynergyComplementaryBonus()
	}

	return getPlayerSynergyBaseBonus()
}

// isComplementaryPair checks if two node types form a complementary pair from config.
func isComplementaryPair(typeA, typeB string) bool {
	for _, pair := range getComplementaryPairs() {
		if len(pair) != 2 {
			continue
		}
		if (pair[0] == typeA && pair[1] == typeB) || (pair[0] == typeB && pair[1] == typeA) {
			return true
		}
	}
	return false
}
