package spells

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/effects"
	"game_main/tactical/perks"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// ExecuteSpellCast performs a spell cast from a commander against target squads.
// It validates mana, deducts cost, applies damage to all alive units in targets,
// and removes destroyed squads from the map.
func ExecuteSpellCast(
	casterEntityID ecs.EntityID,
	spellID string,
	targetSquadIDs []ecs.EntityID,
	manager *common.EntityManager,
) *SpellCastResult {
	result := &SpellCastResult{}

	// Look up spell definition
	spell := templates.GetSpellDefinition(spellID)
	if spell == nil {
		result.ErrorReason = fmt.Sprintf("unknown spell: %s", spellID)
		return result
	}

	// Get mana data
	mana := GetManaData(casterEntityID, manager)
	if mana == nil {
		result.ErrorReason = "caster has no mana component"
		return result
	}

	// Apply commander perk cost modifiers
	actualCost := perks.ModifySpellCost(casterEntityID, spell.ManaCost, manager)

	// Validate mana
	if mana.CurrentMana < actualCost {
		result.ErrorReason = fmt.Sprintf("insufficient mana: have %d, need %d", mana.CurrentMana, actualCost)
		return result
	}

	// Validate spell is in spellbook
	if !HasSpellInBook(casterEntityID, spellID, manager) {
		result.ErrorReason = fmt.Sprintf("spell %s not in caster's spellbook", spellID)
		return result
	}

	// Deduct mana
	mana.CurrentMana -= actualCost

	// Apply effect based on spell type
	switch spell.EffectType {
	case templates.EffectDamage:
		applyDamageSpell(casterEntityID, spell, targetSquadIDs, result, manager)
	case templates.EffectBuff, templates.EffectDebuff:
		applyBuffDebuffSpell(casterEntityID, spell, targetSquadIDs, result, manager)
	}

	result.Success = true
	return result
}

// applyDamageSpell applies damage to all units in target squads.
func applyDamageSpell(
	casterID ecs.EntityID,
	spell *templates.SpellDefinition,
	targetSquadIDs []ecs.EntityID,
	result *SpellCastResult,
	manager *common.EntityManager,
) {
	// Apply commander perk damage modifier
	spellDamage := perks.ModifySpellDamage(casterID, spell.Damage, manager)

	for _, squadID := range targetSquadIDs {
		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		squadDamaged := false

		for _, unitID := range unitIDs {
			unitEntity := manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}

			attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attr == nil || attr.CurrentHealth <= 0 {
				continue
			}

			// Calculate damage: spell damage minus unit's magic defense, minimum 1
			defense := attr.GetMagicDefense()
			damage := spellDamage - defense
			if damage < 1 {
				damage = 1
			}

			attr.CurrentHealth -= damage
			if attr.CurrentHealth < 0 {
				attr.CurrentHealth = 0
			}

			result.TotalDamageDealt += damage
			squadDamaged = true
		}

		if squadDamaged {
			result.AffectedSquadIDs = append(result.AffectedSquadIDs, squadID)
		}

		// Check if squad was destroyed
		if squads.IsSquadDestroyed(squadID, manager) {
			result.SquadsDestroyed = append(result.SquadsDestroyed, squadID)
			if err := combat.RemoveSquadFromMap(squadID, manager); err != nil {
				fmt.Printf("Warning: failed to remove destroyed squad %d from map: %v\n", squadID, err)
			}
		}
	}

	fmt.Printf("Spell cast: %s dealt %d total damage to %d squads (%d destroyed)\n",
		spell.Name, result.TotalDamageDealt, len(result.AffectedSquadIDs), len(result.SquadsDestroyed))
}

// applyBuffDebuffSpell applies stat modifiers to all units in target squads.
func applyBuffDebuffSpell(
	casterID ecs.EntityID,
	spell *templates.SpellDefinition,
	targetSquadIDs []ecs.EntityID,
	result *SpellCastResult,
	manager *common.EntityManager,
) {
	// Apply commander perk duration modifier
	duration := perks.ModifySpellDuration(casterID, spell.Duration, manager)

	effectsApplied := 0
	for _, squadID := range targetSquadIDs {
		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		for _, mod := range spell.StatModifiers {
			// Apply commander perk modifier value scaling
			modValue := perks.ModifySpellModifier(casterID, mod.Modifier, manager)
			effect := effects.ActiveEffect{
				Name:           spell.Name,
				Source:         effects.SourceSpell,
				Stat:           effects.ParseStatType(mod.Stat),
				Modifier:       modValue,
				RemainingTurns: duration,
			}
			effects.ApplyEffectToUnits(unitIDs, effect, manager)
			effectsApplied++
		}
		result.AffectedSquadIDs = append(result.AffectedSquadIDs, squadID)
	}

	effectType := "buff"
	if spell.EffectType != templates.EffectBuff {
		effectType = "debuff"
	}
	fmt.Printf("Spell cast: %s applied %d %s effects to %d squads\n",
		spell.Name, effectsApplied, effectType, len(result.AffectedSquadIDs))
}
