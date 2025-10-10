package squads

/*
// CombatResult - ✅ Uses ecs.EntityID (native type) instead of entity pointers
type CombatResult struct {
	TotalDamage  int
	UnitsKilled  []ecs.EntityID       // ✅ Native IDs
	DamageByUnit map[ecs.EntityID]int // ✅ Native IDs
}

// ExecuteSquadAttack performs row-based combat between two squads
// ✅ Works with ecs.EntityID internally
func ExecuteSquadAttack(attackerSquadID, defenderSquadID ecs.EntityID, squadmanager *SquadECSManager) *CombatResult {
	result := &CombatResult{
		DamageByUnit: make(map[ecs.EntityID]int),
		UnitsKilled:  []ecs.EntityID{},
	}

	// Query for attacker unit IDs (not pointers!)
	attackerUnitIDs := GetUnitIDsInSquad(attackerSquadID, squadmanager)

	// Process each attacker unit
	for _, attackerID := range attackerUnitIDs {
		attackerUnit := FindUnitByID(attackerID, squadmanager)
		if attackerUnit == nil {
			continue
		}

		// Check if unit is alive
		attackerAttr := common.GetAttributes(attackerUnit)
		if attackerAttr.CurrentHealth <= 0 {
			continue
		}

		// Get targeting data
		if !attackerUnit.HasComponent(TargetRowComponent) {
			continue
		}

		targetRowData := common.GetComponentType[*TargetRowData](attackerUnit, TargetRowComponent)

		var actualTargetIDs []ecs.EntityID

		// Handle targeting based on mode
		if targetRowData.Mode == TargetModeCellBased {
			// Cell-based targeting: hit specific grid cells
			for _, cell := range targetRowData.TargetCells {
				row, col := cell[0], cell[1]
				cellTargetIDs := GetUnitIDsAtGridPosition(defenderSquadID, row, col, squadmanager)
				actualTargetIDs = append(actualTargetIDs, cellTargetIDs...)
			}
		} else {
			// Row-based targeting: hit entire row(s)
			for _, targetRow := range targetRowData.TargetRows {
				targetIDs := GetUnitIDsInRow(defenderSquadID, targetRow, squadmanager)

				if len(targetIDs) == 0 {
					continue
				}

				//TODO, handle multitarget seletion a better way. Figure out whether we still want that.
				//I am thinking just cell based will do it
				if targetRowData.IsMultiTarget {
					maxTargets := targetRowData.MaxTargets
					if maxTargets == 0 || maxTargets > len(targetIDs) {
						actualTargetIDs = append(actualTargetIDs, targetIDs...)
					} else {
						actualTargetIDs = append(actualTargetIDs, selectRandomTargetIDs(targetIDs, maxTargets)...)
					}
				} else {
					actualTargetIDs = append(actualTargetIDs, selectLowestHPTargetID(targetIDs, squadmanager))
				}
			}
		}

		//TODO this is where we should add hit chance
		// Apply damage to each selected target
		for _, defenderID := range actualTargetIDs {
			damage := calculateUnitDamageByID(attackerID, defenderID, squadmanager)
			applyDamageToUnitByID(defenderID, damage, result, squadmanager)
		}
	}

	result.TotalDamage = sumDamageMap(result.DamageByUnit)

	return result
}

// calculateUnitDamageByID - TODO, calculate damage based off unit attributes
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *SquadECSManager) int {
	attackerUnit := FindUnitByID(attackerID, squadmanager)
	defenderUnit := FindUnitByID(defenderID, squadmanager)

	if attackerUnit == nil || defenderUnit == nil {
		return 0
	}

	attackerAttr := common.GetAttributes(attackerUnit)
	defenderAttr := common.GetAttributes(defenderUnit)

	// Base damage (adapt to existing weapon system)
	baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus

	// d20 variance (reuse existing logic)
	roll := randgen.GetDiceRoll(20)
	if roll >= 18 {
		baseDamage = int(float64(baseDamage) * 1.5) // Critical
	} else if roll <= 3 {
		baseDamage = baseDamage / 2 // Weak hit
	}

	// Apply role modifiers
	if attackerUnit.HasComponent(UnitRoleComponent) {

		baseDamage = 1 //Todo, calculate this from attributes
	}

	// Apply defense
	totalDamage := baseDamage - defenderAttr.TotalProtection
	if totalDamage < 1 {
		totalDamage = 1 // Minimum damage
	}

	// Apply cover (damage reduction from units in front)
	coverReduction := CalculateTotalCover(defenderID, squadmanager)
	if coverReduction > 0.0 {
		totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))
		if totalDamage < 1 {
			totalDamage = 1 // Minimum damage even with cover
		}
	}

	return totalDamage
}
*/
