package spells

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
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
	result := &SpellCastResult{
		SpellID:      spellID,
		DamageByUnit: make(map[ecs.EntityID]int),
	}

	// Look up spell definition
	spell := templates.GetSpellDefinition(spellID)
	if spell == nil {
		result.ErrorReason = fmt.Sprintf("unknown spell: %s", spellID)
		return result
	}
	result.SpellName = spell.Name

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
	book := GetSpellBook(casterEntityID, manager)
	if book == nil {
		result.ErrorReason = "caster has no spellbook"
		return result
	}
	hasSpell := false
	for _, id := range book.SpellIDs {
		if id == spellID {
			hasSpell = true
			break
		}
	}
	if !hasSpell {
		result.ErrorReason = fmt.Sprintf("spell %s not in caster's spellbook", spellID)
		return result
	}

	// Deduct mana
	mana.CurrentMana -= spell.ManaCost

	// Apply damage to each target squad
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
			damage := spell.Damage - defense
			if damage < 1 {
				damage = 1
			}

			attr.CurrentHealth -= damage
			if attr.CurrentHealth < 0 {
				attr.CurrentHealth = 0
			}

			result.DamageByUnit[unitID] = damage
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

	result.Success = true
	fmt.Printf("Spell cast: %s dealt %d total damage to %d squads (%d destroyed)\n",
		spell.Name, result.TotalDamageDealt, len(result.AffectedSquadIDs), len(result.SquadsDestroyed))

	return result
}
