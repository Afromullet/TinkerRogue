package main

import "game_main/combatsim"

// GetAllTestScenarios returns all tactical combat test scenarios
func GetAllTestScenarios() []combatsim.CombatScenario {
	return []combatsim.CombatScenario{
		createScenario_TankVsTank(),
		createScenario_DPSvsDPS(),
		createScenario_TankVsDPS(),
		createScenario_RangedVsMelee(),
		createScenario_MagicVsPhysical(),
		createScenario_SupportHeavy(),
		createScenario_BalancedMixed(),
		createScenario_MultiCellUnits(),
		createScenario_CoverStacking(),
		createScenario_PierceThrough(),
		createScenario_MinimumSquad(),
		createScenario_SizeAsymmetry(),
		createScenario_FullAOE(),
		createScenario_MixedRange(),
		createScenario_GoblinSwarm(),
	}
}

// createScenario_TankVsTank tests heavy armor combat with cover mechanics.
// Composition: Knight/Fighter/Fighter vs Paladin/Fighter/Fighter
// Focus: Cover effectiveness, combat duration, high armor interaction
func createScenario_TankVsTank() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Paladin", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Tank vs Tank").
		WithAttacker("Knight Squad", attackerUnits).
		WithDefender("Paladin Squad", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_DPSvsDPS tests high damage, high evasion combat.
// Composition: Warrior/Swordsman/Rogue vs Assassin/Swordsman/Rogue
// Focus: High evasion, crit rates, fast combat resolution
func createScenario_DPSvsDPS() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Warrior", GridRow: 0, GridCol: 1, IsLeader: true},
		{TemplateName: "Swordsman", GridRow: 0, GridCol: 0},
		{TemplateName: "Rogue", GridRow: 0, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Assassin", GridRow: 0, GridCol: 1},
		{TemplateName: "Swordsman", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Rogue", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("DPS vs DPS").
		WithAttacker("Striker Squad", attackerUnits).
		WithDefender("Assassin Squad", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_TankVsDPS tests defensive vs offensive compositions.
// Composition: Knight/Paladin/Fighter vs Warrior/Swordsman/Rogue
// Focus: Cover effectiveness vs high damage, tactical balance
func createScenario_TankVsDPS() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Paladin", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Warrior", GridRow: 0, GridCol: 1, IsLeader: true},
		{TemplateName: "Swordsman", GridRow: 0, GridCol: 0},
		{TemplateName: "Rogue", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Tank vs DPS").
		WithAttacker("Heavy Defense", attackerUnits).
		WithDefender("Glass Cannons", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_RangedVsMelee tests range advantage dynamics.
// Composition: Archer/Crossbowman/Marksman vs Knight/Warrior/Fighter
// Focus: Distance-based combat, range advantage
func createScenario_RangedVsMelee() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Archer", GridRow: 2, GridCol: 0},
		{TemplateName: "Crossbowman", GridRow: 2, GridCol: 1, IsLeader: true},
		{TemplateName: "Marksman", GridRow: 2, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Ranged vs Melee").
		WithAttacker("Archer Squad", attackerUnits).
		WithDefender("Melee Squad", defenderUnits).
		WithDistance(4).
		Build()
}

// createScenario_MagicVsPhysical tests magic multi-target attacks.
// Composition: Wizard/Sorcerer/Mage vs Knight/Fighter/Warrior
// Focus: Multi-target attacks, AOE patterns, magic damage
func createScenario_MagicVsPhysical() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Wizard", GridRow: 2, GridCol: 1},
		{TemplateName: "Sorcerer", GridRow: 2, GridCol: 0},
		{TemplateName: "Mage", GridRow: 2, GridCol: 2, IsLeader: true},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Magic vs Physical").
		WithAttacker("Caster Squad", attackerUnits).
		WithDefender("Heavy Armor", defenderUnits).
		WithDistance(3).
		Build()
}

// createScenario_SupportHeavy tests leadership and support effectiveness.
// Composition: Cleric/Priest/Paladin vs Warrior/Warrior/Warrior
// Focus: Leadership/morale, healing potential, support effectiveness
func createScenario_SupportHeavy() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Cleric", GridRow: 1, GridCol: 1, IsLeader: true},
		{TemplateName: "Priest", GridRow: 2, GridCol: 1},
		{TemplateName: "Paladin", GridRow: 0, GridCol: 1},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Warrior", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 1},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Support Heavy").
		WithAttacker("Holy Trinity", attackerUnits).
		WithDefender("Pure Warriors", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_BalancedMixed tests well-rounded squad composition.
// Composition: Knight/Archer/Cleric vs Fighter/Wizard/Priest
// Focus: Well-rounded squad composition balance
func createScenario_BalancedMixed() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 1, IsLeader: true},
		{TemplateName: "Archer", GridRow: 2, GridCol: 2},
		{TemplateName: "Cleric", GridRow: 1, GridCol: 1},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Wizard", GridRow: 2, GridCol: 2},
		{TemplateName: "Priest", GridRow: 1, GridCol: 1, IsLeader: true},
	}

	return combatsim.NewScenarioBuilder("Balanced Mixed").
		WithAttacker("Balanced Alpha", attackerUnits).
		WithDefender("Balanced Beta", defenderUnits).
		WithDistance(2).
		Build()
}

// createScenario_MultiCellUnits tests large unit mechanics.
// Composition: Ogre/Orc Warrior vs Fighter/Fighter/Fighter/Fighter
// Focus: Large unit targeting, 2x2 and 2x1 unit mechanics
// Note: Deliberately imbalanced (2v4) to test multi-cell durability
func createScenario_MultiCellUnits() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Ogre", GridRow: 0, GridCol: 0, IsLeader: true},       // 2x2 unit
		{TemplateName: "Orc Warrior", GridRow: 0, GridCol: 2, IsLeader: false}, // 2x1 unit
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
		{TemplateName: "Fighter", GridRow: 1, GridCol: 1},
	}

	return combatsim.NewScenarioBuilder("Multi-Cell Units").
		WithAttacker("Giant Squad", attackerUnits).
		WithDefender("Fighter Squad", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_CoverStacking tests multiple cover source mechanics.
// Composition: Knight/Knight/Archer vs Warrior/Warrior/Warrior
// Focus: Multiple cover sources, backline protection
func createScenario_CoverStacking() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Knight", GridRow: 0, GridCol: 2},
		{TemplateName: "Archer", GridRow: 2, GridCol: 1}, // Protected by both knights
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Warrior", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 1},
		{TemplateName: "Warrior", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Cover Stacking").
		WithAttacker("Protected Archer", attackerUnits).
		WithDefender("Warrior Wall", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_PierceThrough tests pierce-through targeting mechanics.
// Composition: Wizard/Sorcerer vs Fighter/Archer (sparse formation)
// Focus: Pierce-through to back row when front empty
func createScenario_PierceThrough() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Wizard", GridRow: 2, GridCol: 1},
		{TemplateName: "Sorcerer", GridRow: 2, GridCol: 0, IsLeader: true},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 0, IsLeader: true}, // Front
		{TemplateName: "Archer", GridRow: 2, GridCol: 2},                  // Back (sparse)
	}

	return combatsim.NewScenarioBuilder("Pierce Through").
		WithAttacker("Full-Grid Casters", attackerUnits).
		WithDefender("Sparse Formation", defenderUnits).
		WithDistance(3).
		Build()
}

// createScenario_MinimumSquad tests simplest 1v1 combat.
// Composition: Fighter vs Fighter
// Focus: 1v1 combat, simplest case
func createScenario_MinimumSquad() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1, IsLeader: true},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1, IsLeader: true},
	}

	return combatsim.NewScenarioBuilder("Minimum Squad (1v1)").
		WithAttacker("Solo Fighter A", attackerUnits).
		WithDefender("Solo Fighter B", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_SizeAsymmetry tests quality vs quantity dynamics.
// Composition: Knight/Paladin vs Warrior/Swordsman/Rogue/Assassin
// Focus: Quality vs quantity, outnumbered scenario
// Note: Deliberately imbalanced (2v4)
func createScenario_SizeAsymmetry() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Paladin", GridRow: 0, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Warrior", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Swordsman", GridRow: 0, GridCol: 1},
		{TemplateName: "Rogue", GridRow: 0, GridCol: 2},
		{TemplateName: "Assassin", GridRow: 1, GridCol: 1},
	}

	return combatsim.NewScenarioBuilder("Size Asymmetry (2v4)").
		WithAttacker("Elite Tanks", attackerUnits).
		WithDefender("Swarm DPS", defenderUnits).
		WithDistance(1).
		Build()
}

// createScenario_FullAOE tests maximum AOE damage potential.
// Composition: Wizard/Sorcerer/Warlock vs Knight/Fighter/Paladin/Warrior
// Focus: Maximum AOE damage, cover against magic
func createScenario_FullAOE() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Wizard", GridRow: 2, GridCol: 1},
		{TemplateName: "Sorcerer", GridRow: 2, GridCol: 0},
		{TemplateName: "Warlock", GridRow: 2, GridCol: 2, IsLeader: true},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 1},
		{TemplateName: "Paladin", GridRow: 0, GridCol: 2},
		{TemplateName: "Warrior", GridRow: 1, GridCol: 1},
	}

	return combatsim.NewScenarioBuilder("Full AOE Assault").
		WithAttacker("AOE Casters", attackerUnits).
		WithDefender("Armored Wall", defenderUnits).
		WithDistance(3).
		Build()
}

// createScenario_MixedRange tests ranged vs ranged with varied ranges.
// Composition: Archer/Scout/Marksman vs Crossbowman/Ranger/Spearman
// Focus: Various range values (2-4), ranged vs ranged
func createScenario_MixedRange() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Archer", GridRow: 2, GridCol: 0},
		{TemplateName: "Scout", GridRow: 2, GridCol: 1, IsLeader: true},
		{TemplateName: "Marksman", GridRow: 2, GridCol: 2},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Crossbowman", GridRow: 2, GridCol: 0},
		{TemplateName: "Ranger", GridRow: 2, GridCol: 1, IsLeader: true},
		{TemplateName: "Spearman", GridRow: 2, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Mixed Range Engagement").
		WithAttacker("Long Range", attackerUnits).
		WithDefender("Mid Range", defenderUnits).
		WithDistance(3).
		Build()
}

// createScenario_GoblinSwarm tests many weak units vs few strong units.
// Composition: Goblin Raider x4 vs Knight/Fighter
// Focus: Many weak units vs few strong units
// Note: Deliberately imbalanced (4v2)
func createScenario_GoblinSwarm() combatsim.CombatScenario {
	attackerUnits := []combatsim.UnitConfig{
		{TemplateName: "Goblin Raider", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Goblin Raider", GridRow: 0, GridCol: 1},
		{TemplateName: "Goblin Raider", GridRow: 0, GridCol: 2},
		{TemplateName: "Goblin Raider", GridRow: 1, GridCol: 1},
	}

	defenderUnits := []combatsim.UnitConfig{
		{TemplateName: "Knight", GridRow: 0, GridCol: 0, IsLeader: true},
		{TemplateName: "Fighter", GridRow: 0, GridCol: 2},
	}

	return combatsim.NewScenarioBuilder("Goblin Swarm (4v2)").
		WithAttacker("Goblin Horde", attackerUnits).
		WithDefender("Elite Defenders", defenderUnits).
		WithDistance(1).
		Build()
}
