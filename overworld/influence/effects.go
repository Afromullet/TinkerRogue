package influence

import (
	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// ClassifyInteraction determines the interaction type between two nodes.
func ClassifyInteraction(manager *common.EntityManager, entityA, entityB *ecs.Entity) core.InteractionType {
	aIsThreat := manager.HasComponent(entityA.GetID(), core.ThreatNodeComponent)
	bIsThreat := manager.HasComponent(entityB.GetID(), core.ThreatNodeComponent)
	aIsPlayer := manager.HasComponent(entityA.GetID(), core.PlayerNodeComponent)
	bIsPlayer := manager.HasComponent(entityB.GetID(), core.PlayerNodeComponent)

	// Both threats
	if aIsThreat && bIsThreat {
		if sameFaction(manager, entityA, entityB) {
			return core.InteractionSynergy
		}
		return core.InteractionCompetition
	}

	// Both player nodes
	if aIsPlayer && bIsPlayer {
		return core.InteractionPlayerBoost
	}

	// One player, one threat -> suppression
	if (aIsPlayer && bIsThreat) || (aIsThreat && bIsPlayer) {
		return core.InteractionSuppression
	}

	// Fallback (shouldn't happen with valid nodes)
	return core.InteractionCompetition
}

// sameFaction checks if two threat nodes belong to the same faction.
func sameFaction(manager *common.EntityManager, entityA, entityB *ecs.Entity) bool {
	threatA := common.GetComponentType[*core.ThreatNodeData](entityA, core.ThreatNodeComponent)
	threatB := common.GetComponentType[*core.ThreatNodeData](entityB, core.ThreatNodeComponent)
	if threatA == nil || threatB == nil {
		return false
	}

	nodeA := core.GetNodeRegistry().GetNodeByType(threatA.ThreatType)
	nodeB := core.GetNodeRegistry().GetNodeByType(threatB.ThreatType)
	if nodeA == nil || nodeB == nil {
		return false
	}

	return nodeA.FactionID != "" && nodeA.FactionID == nodeB.FactionID
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

// calculateSuppressionPenalty returns growth penalty from player nodes on threats.
// Scaled by node type multiplier.
func calculateSuppressionPenalty(manager *common.EntityManager, entityA, entityB *ecs.Entity) float64 {
	// Find which entity is the player node
	playerEntity := entityA
	if !manager.HasComponent(entityA.GetID(), core.PlayerNodeComponent) {
		playerEntity = entityB
	}

	// Get node type multiplier
	nodeTypeMult := 1.0
	playerData := common.GetComponentType[*core.PlayerNodeData](playerEntity, core.PlayerNodeComponent)
	if playerData != nil {
		nodeTypeMult = getSuppressionNodeTypeMultiplier(string(playerData.NodeTypeID))
	}

	return -getSuppressionGrowthPenalty() * nodeTypeMult
}

// calculatePlayerSynergyBonus computes bonus for adjacent player nodes.
// Returns base bonus, or complementary bonus if types are a complementary pair.
func calculatePlayerSynergyBonus(entityA, entityB *ecs.Entity) float64 {
	// Check if complementary pair
	playerA := common.GetComponentType[*core.PlayerNodeData](entityA, core.PlayerNodeComponent)
	playerB := common.GetComponentType[*core.PlayerNodeData](entityB, core.PlayerNodeComponent)
	if playerA != nil && playerB != nil && isComplementaryPair(string(playerA.NodeTypeID), string(playerB.NodeTypeID)) {
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
