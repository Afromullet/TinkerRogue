package squads

// FilterByAttackRange returns units with attack range >= minRange
func FilterByAttackRange(minRange int) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if unit.AttackRange >= minRange {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// FilterByMaxAttackRange returns units with attack range <= maxRange
func FilterByMaxAttackRange(maxRange int) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if unit.AttackRange <= maxRange {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// FilterByAttackType returns units matching the specified attack type
func FilterByAttackType(attackType AttackType) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if unit.AttackType == attackType {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}
