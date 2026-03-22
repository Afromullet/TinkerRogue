package unitdefs

// FilterUnits returns all units from the global Units slice matching the predicate.
func FilterUnits(predicate func(UnitTemplate) bool) []UnitTemplate {
	var filtered []UnitTemplate
	for _, unit := range Units {
		if predicate(unit) {
			filtered = append(filtered, unit)
		}
	}
	return filtered
}

// FilterByAttackRange returns units with attack range >= minRange
func FilterByAttackRange(minRange int) []UnitTemplate {
	return FilterUnits(func(u UnitTemplate) bool { return u.AttackRange >= minRange })
}

// FilterByMaxAttackRange returns units with attack range <= maxRange
func FilterByMaxAttackRange(maxRange int) []UnitTemplate {
	return FilterUnits(func(u UnitTemplate) bool { return u.AttackRange <= maxRange })
}

// FilterByMinMovementSpeed returns units with movement speed >= minSpeed
func FilterByMinMovementSpeed(minSpeed int) []UnitTemplate {
	return FilterUnits(func(u UnitTemplate) bool { return u.MovementSpeed >= minSpeed })
}

// FilterByAttackType returns units matching the specified attack type
func FilterByAttackType(attackType AttackType) []UnitTemplate {
	return FilterUnits(func(u UnitTemplate) bool { return u.AttackType == attackType })
}
