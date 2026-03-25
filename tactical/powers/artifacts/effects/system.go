package effects

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// ApplyEffect adds an effect to an entity, immediately modifying the target stat.
// Creates the ActiveEffectsComponent if it doesn't exist on the entity.
func ApplyEffect(entityID ecs.EntityID, effect ActiveEffect, manager *common.EntityManager) {
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return
	}

	// Skip dead units
	if attr.CurrentHealth <= 0 {
		return
	}

	// Apply the stat modifier immediately
	applyModifierToStat(attr, effect.Stat, effect.Modifier)

	// Get or create the effects component
	var effectsData *ActiveEffectsData
	if entity.HasComponent(ActiveEffectsComponent) {
		effectsData = common.GetComponentType[*ActiveEffectsData](entity, ActiveEffectsComponent)
	} else {
		effectsData = &ActiveEffectsData{}
		entity.AddComponent(ActiveEffectsComponent, effectsData)
	}

	effectsData.Effects = append(effectsData.Effects, effect)

	fmt.Printf("[EFFECT] Applied %s to entity %d: %+d to stat %d (%d turns)\n",
		effect.Name, entityID, effect.Modifier, effect.Stat, effect.RemainingTurns)
}

// ApplyEffectToUnits applies an effect to a list of unit entity IDs.
// Callers provide the unit IDs (e.g., from squads.GetUnitIDsInSquad).
func ApplyEffectToUnits(unitIDs []ecs.EntityID, effect ActiveEffect, manager *common.EntityManager) {
	for _, unitID := range unitIDs {
		ApplyEffect(unitID, effect, manager)
	}
}

// TickEffects decrements duration on all effects for an entity.
// When an effect expires (RemainingTurns reaches 0), it reverses the stat modifier
// and removes the effect. Permanent effects (RemainingTurns == -1) are skipped.
func TickEffects(entityID ecs.EntityID, manager *common.EntityManager) {
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return
	}

	if !entity.HasComponent(ActiveEffectsComponent) {
		return
	}

	effectsData := common.GetComponentType[*ActiveEffectsData](entity, ActiveEffectsComponent)
	if effectsData == nil || len(effectsData.Effects) == 0 {
		return
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return
	}

	// Filter in-place: keep non-expired effects
	kept := effectsData.Effects[:0]
	for i := range effectsData.Effects {
		e := &effectsData.Effects[i]

		// Permanent effects never tick
		if e.RemainingTurns == -1 {
			kept = append(kept, *e)
			continue
		}

		e.RemainingTurns--
		if e.RemainingTurns <= 0 {
			// Effect expired — reverse the modifier
			reverseModifierFromStat(attr, e.Stat, e.Modifier)
			fmt.Printf("[EFFECT] Expired %s on entity %d\n", e.Name, entityID)
		} else {
			kept = append(kept, *e)
		}
	}
	effectsData.Effects = kept
}

// TickEffectsForUnits ticks effects for a list of unit entity IDs.
// Callers provide the unit IDs (e.g., from squads.GetUnitIDsInSquad).
func TickEffectsForUnits(unitIDs []ecs.EntityID, manager *common.EntityManager) {
	for _, unitID := range unitIDs {
		TickEffects(unitID, manager)
	}
}

// RemoveAllEffects removes all effects from an entity, reversing all stat modifiers.
func RemoveAllEffects(entityID ecs.EntityID, manager *common.EntityManager) {
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return
	}

	if !entity.HasComponent(ActiveEffectsComponent) {
		return
	}

	effectsData := common.GetComponentType[*ActiveEffectsData](entity, ActiveEffectsComponent)
	if effectsData == nil {
		return
	}

	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil {
		return
	}

	for _, e := range effectsData.Effects {
		reverseModifierFromStat(attr, e.Stat, e.Modifier)
	}
	effectsData.Effects = nil
}

// applyModifierToStat adds a modifier to the corresponding Attributes field.
func applyModifierToStat(attr *common.Attributes, stat StatType, modifier int) {
	switch stat {
	case StatStrength:
		attr.Strength += modifier
	case StatDexterity:
		attr.Dexterity += modifier
	case StatMagic:
		attr.Magic += modifier
	case StatLeadership:
		attr.Leadership += modifier
	case StatArmor:
		attr.Armor += modifier
	case StatWeapon:
		attr.Weapon += modifier
	case StatMovementSpeed:
		attr.MovementSpeed += modifier
	case StatAttackRange:
		attr.AttackRange += modifier
	}
}

// reverseModifierFromStat subtracts a modifier from the corresponding Attributes field.
func reverseModifierFromStat(attr *common.Attributes, stat StatType, modifier int) {
	applyModifierToStat(attr, stat, -modifier)
}
