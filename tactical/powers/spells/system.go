package spells

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/effects"
	"game_main/tactical/squads/squadcore"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// AddSpellCapabilityToSquad attaches ManaComponent and SpellBookComponent to a squad entity.
// Call this after squad creation when the leader's unit type has spells.
// Does nothing if spellIDs is empty.
func AddSpellCapabilityToSquad(squadID ecs.EntityID, manager *common.EntityManager, startingMana, maxMana int, spellIDs []string) {
	if len(spellIDs) == 0 {
		return
	}
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return
	}
	entity.AddComponent(ManaComponent, &ManaData{
		CurrentMana: startingMana,
		MaxMana:     maxMana,
	})
	entity.AddComponent(SpellBookComponent, &SpellBookData{
		SpellIDs: spellIDs,
	})
}

// InitSquadSpellsFromLeader checks the squad's leader unit type and adds spell capability
// if that unit type has spells defined in UnitSpellRegistry. Call after CreateSquadFromTemplate.
func InitSquadSpellsFromLeader(squadID ecs.EntityID, manager *common.EntityManager) {
	leaderID := squadcore.GetLeaderID(squadID, manager)
	if leaderID == 0 {
		return
	}
	leaderUnitType := squadcore.GetUnitType(leaderID, manager)
	if leaderUnitType == "" {
		return
	}
	spellIDs := templates.GetSpellsForUnitType(leaderUnitType)
	if len(spellIDs) == 0 {
		return
	}
	cfg := templates.GameConfig.Commander
	AddSpellCapabilityToSquad(squadID, manager, cfg.StartingMana, cfg.MaxMana, spellIDs)
}

// ExecuteSpellCast performs a spell cast from a squad against target squads.
// It validates mana, deducts cost, applies damage to all alive units in targets,
// and removes destroyed squads from the map.
//
// Design decision: spells intentionally bypass perk hooks. Perks modify the
// physical combat pipeline (attack/counter/cover); spells operate in a separate
// mana-gated power layer. This matches the four-layer power system design where
// each layer turns a distinct knob. If spell-perk interaction is desired in the
// future, add an OnSpellCast hook point (see PERKS_AND_HOOKS_COMBINED_ANALYSIS.md
// Tier 3 recommendations).
func ExecuteSpellCast(
	casterEntityID ecs.EntityID,
	spellID string,
	targetSquadIDs []ecs.EntityID,
	manager *common.EntityManager,
) *SpellCastResult {
	result := &SpellCastResult{}

	// Verify combat is active
	if !combatstate.IsCombatActive(manager) {
		result.ErrorReason = "no active combat"
		return result
	}

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

	// Validate mana
	if mana.CurrentMana < spell.ManaCost {
		result.ErrorReason = fmt.Sprintf("insufficient mana: have %d, need %d", mana.CurrentMana, spell.ManaCost)
		return result
	}

	// Validate spell is in spellbook
	if !HasSpellInBook(casterEntityID, spellID, manager) {
		result.ErrorReason = fmt.Sprintf("spell %s not in caster's spellbook", spellID)
		return result
	}

	// Deduct mana
	mana.CurrentMana -= spell.ManaCost

	// Apply effect based on spell type
	switch spell.EffectType {
	case templates.EffectDamage:
		applyDamageSpell(spell, targetSquadIDs, result, manager)
	case templates.EffectBuff, templates.EffectDebuff:
		applyBuffDebuffSpell(spell, targetSquadIDs, result, manager)
	}

	result.Success = true
	return result
}

// applyDamageSpell applies damage to all units in target squads.
func applyDamageSpell(
	spell *templates.SpellDefinition,
	targetSquadIDs []ecs.EntityID,
	result *SpellCastResult,
	manager *common.EntityManager,
) {
	for _, squadID := range targetSquadIDs {
		unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
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
			damage := spell.Damage - defense
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
		if squadcore.IsSquadDestroyed(squadID, manager) {
			result.SquadsDestroyed = append(result.SquadsDestroyed, squadID)
			if err := combatstate.RemoveSquadFromMap(squadID, manager); err != nil {
				fmt.Printf("Warning: failed to remove destroyed squad %d from map: %v\n", squadID, err)
			}
		}
	}

	fmt.Printf("Spell cast: %s dealt %d total damage to %d squads (%d destroyed)\n",
		spell.Name, result.TotalDamageDealt, len(result.AffectedSquadIDs), len(result.SquadsDestroyed))
}

// applyBuffDebuffSpell applies stat modifiers to all units in target squads.
func applyBuffDebuffSpell(
	spell *templates.SpellDefinition,
	targetSquadIDs []ecs.EntityID,
	result *SpellCastResult,
	manager *common.EntityManager,
) {
	effectsApplied := 0
	for _, squadID := range targetSquadIDs {
		unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
		for _, mod := range spell.StatModifiers {
			statType, err := effects.ParseStatType(mod.Stat)
			if err != nil {
				fmt.Printf("WARNING: spell %q has invalid stat modifier: %v\n", spell.Name, err)
				continue
			}
			effect := effects.ActiveEffect{
				Name:           spell.Name,
				Source:         effects.SourceSpell,
				Stat:           statType,
				Modifier:       mod.Modifier,
				RemainingTurns: spell.Duration,
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
