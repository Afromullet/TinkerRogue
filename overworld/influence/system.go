package influence

import (
	"fmt"
	"math"
	"sort"

	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// UpdateInfluenceInteractions resolves all influence interactions between overlapping nodes.
// Should be called once per tick, before threat and faction updates consume the results.
func UpdateInfluenceInteractions(manager *common.EntityManager, currentTick int64) {
	// 1. Clear stale interactions on all entities that had them
	clearStaleInteractions(manager)

	// 2. Find all overlapping node pairs
	pairs := FindOverlappingNodes(manager)
	if len(pairs) == 0 {
		return
	}

	// 3. For each pair: classify, calculate, record
	for _, pair := range pairs {
		interactionType := ClassifyInteraction(manager, pair.EntityA, pair.EntityB)
		effectStrength := CalculateInteractionModifier(manager, interactionType, pair.EntityA, pair.EntityB)

		// Add interaction to entity A
		addInteraction(manager, pair.EntityA, core.NodeInteraction{
			TargetID:     pair.EntityB.GetID(),
			Relationship: interactionType,
			Modifier:     effectStrength,
			Distance:     pair.Distance,
		})

		// Add reciprocal interaction to entity B
		addInteraction(manager, pair.EntityB, core.NodeInteraction{
			TargetID:     pair.EntityA.GetID(),
			Relationship: interactionType,
			Modifier:     effectStrength,
			Distance:     pair.Distance,
		})

		// Log significant new interactions
		logInteractionEvent(interactionType, pair, currentTick)
	}

	// 4. Calculate NetModifier for each entity with interactions
	finalizeNetModifiers(manager)
}

// clearStaleInteractions resets interaction data on all entities that have it.
func clearStaleInteractions(manager *common.EntityManager) {
	for _, result := range manager.World.Query(core.InteractionTag) {
		data := common.GetComponentType[*core.InteractionData](result.Entity, core.InteractionComponent)
		if data != nil {
			data.Interactions = data.Interactions[:0]
			data.NetModifier = 1.0
		}
	}
}

// addInteraction ensures the entity has an InteractionComponent and appends the interaction.
func addInteraction(manager *common.EntityManager, entity *ecs.Entity, interaction core.NodeInteraction) {
	entityID := entity.GetID()

	if !manager.HasComponent(entityID, core.InteractionComponent) {
		entity.AddComponent(core.InteractionComponent, &core.InteractionData{
			Interactions: []core.NodeInteraction{interaction},
			NetModifier:  1.0,
		})
		return
	}

	data := common.GetComponentType[*core.InteractionData](entity, core.InteractionComponent)
	if data != nil {
		data.Interactions = append(data.Interactions, interaction)
	}
}

// finalizeNetModifiers computes the combined modifier by keeping the top 2
// strongest interactions per type (simple sum, no exponential decay).
// NetModifier = 1.0 + sum of top-2 modifiers per type group.
func finalizeNetModifiers(manager *common.EntityManager) {
	for _, result := range manager.World.Query(core.InteractionTag) {
		data := common.GetComponentType[*core.InteractionData](result.Entity, core.InteractionComponent)
		if data == nil || len(data.Interactions) == 0 {
			data.NetModifier = 1.0
			continue
		}

		// Group interactions by type
		groups := make(map[core.InteractionType][]float64)
		for _, interaction := range data.Interactions {
			groups[interaction.Relationship] = append(groups[interaction.Relationship], interaction.Modifier)
		}

		// Keep top 2 strongest per type, simple sum
		netEffect := 0.0
		for _, modifiers := range groups {
			// Sort by absolute value descending so strongest effects are first
			sort.Slice(modifiers, func(i, j int) bool {
				return math.Abs(modifiers[i]) > math.Abs(modifiers[j])
			})
			limit := len(modifiers)
			if limit > 2 {
				limit = 2
			}
			for i := 0; i < limit; i++ {
				netEffect += modifiers[i]
			}
		}

		data.NetModifier = 1.0 + netEffect
	}
}

// logInteractionEvent logs significant interaction formation events.
func logInteractionEvent(interactionType core.InteractionType, pair NodePair, currentTick int64) {
	switch interactionType {
	case core.InteractionSynergy:
		core.LogEvent(core.EventInfluenceSynergy, currentTick, pair.EntityA.GetID(),
			fmt.Sprintf("Synergy cluster: nodes %d and %d (dist %d)",
				pair.EntityA.GetID(), pair.EntityB.GetID(), pair.Distance), nil)
	case core.InteractionCompetition:
		core.LogEvent(core.EventInfluenceCompetition, currentTick, pair.EntityA.GetID(),
			fmt.Sprintf("Faction rivalry: nodes %d and %d (dist %d)",
				pair.EntityA.GetID(), pair.EntityB.GetID(), pair.Distance), nil)
	case core.InteractionSuppression:
		core.LogEvent(core.EventInfluenceSuppression, currentTick, pair.EntityA.GetID(),
			fmt.Sprintf("Player suppression: nodes %d and %d (dist %d)",
				pair.EntityA.GetID(), pair.EntityB.GetID(), pair.Distance), nil)
	}
}
