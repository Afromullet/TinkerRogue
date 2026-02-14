package spells

import (
	"game_main/common"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// GetManaData returns the ManaData for an entity, or nil if not found.
func GetManaData(entityID ecs.EntityID, manager *common.EntityManager) *ManaData {
	return common.GetComponentTypeByID[*ManaData](manager, entityID, ManaComponent)
}

// GetSpellBook returns the SpellBookData for an entity, or nil if not found.
func GetSpellBook(entityID ecs.EntityID, manager *common.EntityManager) *SpellBookData {
	return common.GetComponentTypeByID[*SpellBookData](manager, entityID, SpellBookComponent)
}

// HasEnoughMana checks if an entity has enough mana to cast the given spell.
func HasEnoughMana(entityID ecs.EntityID, spellID string, manager *common.EntityManager) bool {
	mana := GetManaData(entityID, manager)
	if mana == nil {
		return false
	}
	spell := templates.GetSpellDefinition(spellID)
	if spell == nil {
		return false
	}
	return mana.CurrentMana >= spell.ManaCost
}

// GetCastableSpells returns all spells the entity can currently afford to cast.
func GetCastableSpells(entityID ecs.EntityID, manager *common.EntityManager) []*templates.SpellDefinition {
	mana := GetManaData(entityID, manager)
	if mana == nil {
		return nil
	}
	book := GetSpellBook(entityID, manager)
	if book == nil {
		return nil
	}

	var castable []*templates.SpellDefinition
	for _, id := range book.SpellIDs {
		spell := templates.GetSpellDefinition(id)
		if spell != nil && mana.CurrentMana >= spell.ManaCost {
			castable = append(castable, spell)
		}
	}
	return castable
}

// HasSpellInBook checks if a spell is in the entity's spellbook.
func HasSpellInBook(entityID ecs.EntityID, spellID string, manager *common.EntityManager) bool {
	book := GetSpellBook(entityID, manager)
	if book == nil {
		return false
	}
	for _, id := range book.SpellIDs {
		if id == spellID {
			return true
		}
	}
	return false
}

// GetAllSpells returns all spells in the entity's spellbook (regardless of mana).
func GetAllSpells(entityID ecs.EntityID, manager *common.EntityManager) []*templates.SpellDefinition {
	book := GetSpellBook(entityID, manager)
	if book == nil {
		return nil
	}

	var result []*templates.SpellDefinition
	for _, id := range book.SpellIDs {
		spell := templates.GetSpellDefinition(id)
		if spell != nil {
			result = append(result, spell)
		}
	}
	return result
}
