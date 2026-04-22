package progression

import (
	"errors"
	"fmt"
	"game_main/core/common"
	"game_main/tactical/powers/perks"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

var (
	ErrNotEnoughPoints   = errors.New("not enough points")
	ErrUnknownPerk       = errors.New("unknown perk")
	ErrUnknownSpell      = errors.New("unknown spell")
	ErrNoProgressionData = errors.New("no progression data")
)

// library describes one progression axis (perks vs. spells): which slice on
// ProgressionData stores its unlocked IDs and which currency field funds
// unlocks. Declaring one value per axis removes the parallel code paths that
// would otherwise diverge between spells and perks.
type library struct {
	unlocked     func(*ProgressionData) *[]string
	currency     func(*ProgressionData) *int
	currencyName string
}

var (
	perkLib = library{
		unlocked:     func(d *ProgressionData) *[]string { return &d.UnlockedPerkIDs },
		currency:     func(d *ProgressionData) *int { return &d.SkillPoints },
		currencyName: "skill",
	}
	spellLib = library{
		unlocked:     func(d *ProgressionData) *[]string { return &d.UnlockedSpellIDs },
		currency:     func(d *ProgressionData) *int { return &d.ArcanaPoints },
		currencyName: "arcana",
	}
)

// GetProgression returns the ProgressionData for any entity carrying a
// ProgressionComponent, or nil if absent. In the current design that entity is
// a Commander, but the API is entity-agnostic.
func GetProgression(ownerID ecs.EntityID, manager *common.EntityManager) *ProgressionData {
	return common.GetComponentTypeByID[*ProgressionData](manager, ownerID, ProgressionComponent)
}

// isUnlocked reports whether itemID is present in the slice selected by lib.
func (lib library) isUnlocked(ownerID ecs.EntityID, itemID string, manager *common.EntityManager) bool {
	data := GetProgression(ownerID, manager)
	if data == nil {
		return false
	}
	for _, id := range *lib.unlocked(data) {
		if id == itemID {
			return true
		}
	}
	return false
}

// unlock deducts unlockCost from the library's currency and appends itemID to
// its unlocked list. Idempotent if already unlocked.
func (lib library) unlock(ownerID ecs.EntityID, itemID string, unlockCost int, manager *common.EntityManager) error {
	data := GetProgression(ownerID, manager)
	if data == nil {
		return ErrNoProgressionData
	}
	list := lib.unlocked(data)
	for _, id := range *list {
		if id == itemID {
			return nil
		}
	}
	points := lib.currency(data)
	if *points < unlockCost {
		return fmt.Errorf("%w: need %d %s, have %d", ErrNotEnoughPoints, unlockCost, lib.currencyName, *points)
	}
	*points -= unlockCost
	*list = append(*list, itemID)
	return nil
}

// addPoints grants a positive amount to an owner's currency. No-op on missing data.
func (lib library) addPoints(ownerID ecs.EntityID, amount int, manager *common.EntityManager) {
	if amount <= 0 {
		return
	}
	data := GetProgression(ownerID, manager)
	if data == nil {
		return
	}
	*lib.currency(data) += amount
}

// === Public API — thin typed wrappers over the shared library helpers. ===

// IsPerkUnlocked reports whether the owner has the given perk in their library.
func IsPerkUnlocked(ownerID ecs.EntityID, perkID perks.PerkID, manager *common.EntityManager) bool {
	return perkLib.isUnlocked(ownerID, string(perkID), manager)
}

// IsSpellUnlocked reports whether the owner has the given spell in their library.
func IsSpellUnlocked(ownerID ecs.EntityID, spellID templates.SpellID, manager *common.EntityManager) bool {
	return spellLib.isUnlocked(ownerID, string(spellID), manager)
}

// UnlockPerk spends SkillPoints to add a perk to the library. Idempotent.
func UnlockPerk(ownerID ecs.EntityID, perkID perks.PerkID, manager *common.EntityManager) error {
	def := perks.GetPerkDefinition(perkID)
	if def == nil {
		return fmt.Errorf("%w: %s", ErrUnknownPerk, perkID)
	}
	return perkLib.unlock(ownerID, string(perkID), def.UnlockCost, manager)
}

// UnlockSpell spends ArcanaPoints to add a spell to the library. Idempotent.
func UnlockSpell(ownerID ecs.EntityID, spellID templates.SpellID, manager *common.EntityManager) error {
	def := templates.GetSpellDefinition(spellID)
	if def == nil {
		return fmt.Errorf("%w: %s", ErrUnknownSpell, spellID)
	}
	return spellLib.unlock(ownerID, string(spellID), def.UnlockCost, manager)
}

// AddArcanaPoints grants Arcana Points to an owner's progression.
func AddArcanaPoints(ownerID ecs.EntityID, amount int, manager *common.EntityManager) {
	spellLib.addPoints(ownerID, amount, manager)
}

// AddSkillPoints grants Skill Points to an owner's progression.
func AddSkillPoints(ownerID ecs.EntityID, amount int, manager *common.EntityManager) {
	perkLib.addPoints(ownerID, amount, manager)
}
