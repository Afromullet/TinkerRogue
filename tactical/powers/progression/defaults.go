package progression

// StartingUnlockedPerks returns the perk IDs every new Player entity starts with.
// Covers Tank + DPS + Support roles so any starter squad has something to equip.
func StartingUnlockedPerks() []string {
	return []string{
		"brace_for_impact",
		"reckless_assault",
		"shieldwall_discipline",
		"field_medic",
	}
}

// StartingUnlockedSpells returns the spell IDs every new Player entity starts with.
// Low-cost damage spells so a starter mage leader is functional turn one.
func StartingUnlockedSpells() []string {
	return []string{
		"spark",
		"singe",
		"frost_snap",
	}
}

// NewProgressionData creates a zeroed ProgressionData seeded with the starter library.
func NewProgressionData() *ProgressionData {
	perks := StartingUnlockedPerks()
	spells := StartingUnlockedSpells()
	return &ProgressionData{
		ArcanaPoints:     0,
		SkillPoints:      0,
		UnlockedSpellIDs: append([]string(nil), spells...),
		UnlockedPerkIDs:  append([]string(nil), perks...),
	}
}
