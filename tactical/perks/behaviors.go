package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// PerkHooks collects all hooks for a single perk.
// Attacker/Defender variants ensure hooks only fire on the correct side,
// eliminating the need for HasPerk() self-checks inside behaviors.
type PerkHooks struct {
	AttackerDamageMod DamageModHook // runs only when this squad is the attacker
	DefenderDamageMod DamageModHook // runs only when this squad is the defender
	DefenderCoverMod  CoverModHook  // runs only when this squad is the defender
	TargetOverride    TargetOverrideHook
	CounterMod        CounterModHook
	PostDamage        PostDamageHook
	TurnStart         TurnStartHook
	DamageRedirect    DamageRedirectHook
	DeathOverride     DeathOverrideHook
}

var hookRegistry = map[string]*PerkHooks{}

// RegisterPerkHooks registers a perk's hook implementations by perk ID.
func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
	hookRegistry[perkID] = hooks
}

// GetPerkHooks returns the hook implementations for a perk, or nil if not found.
func GetPerkHooks(perkID string) *PerkHooks {
	return hookRegistry[perkID]
}

func init() {
	// Tier 1: Combat Conditioning
	RegisterPerkHooks("brace_for_impact", &PerkHooks{DefenderCoverMod: braceForImpactCoverMod})
	RegisterPerkHooks("reckless_assault", &PerkHooks{
		AttackerDamageMod: recklessAssaultAttackerMod,
		DefenderDamageMod: recklessAssaultDefenderMod,
	})
	RegisterPerkHooks("stalwart", &PerkHooks{CounterMod: stalwartCounterMod})
	RegisterPerkHooks("executioners_instinct", &PerkHooks{AttackerDamageMod: executionerDamageMod})
	RegisterPerkHooks("shieldwall_discipline", &PerkHooks{DefenderDamageMod: shieldwallDamageMod})
	RegisterPerkHooks("isolated_predator", &PerkHooks{AttackerDamageMod: isolatedPredatorDamageMod})
	RegisterPerkHooks("vigilance", &PerkHooks{DefenderDamageMod: vigilanceDamageMod})
	RegisterPerkHooks("field_medic", &PerkHooks{TurnStart: fieldMedicTurnStart})
	RegisterPerkHooks("opening_salvo", &PerkHooks{AttackerDamageMod: openingSalvoDamageMod})
	RegisterPerkHooks("last_line", &PerkHooks{AttackerDamageMod: lastLineDamageMod})

	// Tier 2: Combat Specialization
	RegisterPerkHooks("cleave", &PerkHooks{
		TargetOverride:    cleaveTargetOverride,
		AttackerDamageMod: cleaveDamageMod,
	})
	RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
	RegisterPerkHooks("disruption", &PerkHooks{PostDamage: disruptionPostDamage})
	RegisterPerkHooks("guardian_protocol", &PerkHooks{DamageRedirect: guardianDamageRedirect})
	RegisterPerkHooks("overwatch", &PerkHooks{TurnStart: overwatchTurnStart})
	RegisterPerkHooks("adaptive_armor", &PerkHooks{DefenderDamageMod: adaptiveArmorDamageMod})
	RegisterPerkHooks("bloodlust", &PerkHooks{
		PostDamage:        bloodlustPostDamage,
		AttackerDamageMod: bloodlustDamageMod,
	})
	RegisterPerkHooks("fortify", &PerkHooks{
		TurnStart:        fortifyTurnStart,
		DefenderCoverMod: fortifyCoverMod,
	})
	RegisterPerkHooks("precision_strike", &PerkHooks{TargetOverride: precisionStrikeTargetOverride})
	RegisterPerkHooks("resolute", &PerkHooks{
		TurnStart:     resoluteTurnStart,
		DeathOverride: resoluteDeathOverride,
	})
	RegisterPerkHooks("grudge_bearer", &PerkHooks{
		PostDamage:        grudgeBearerPostDamage,
		AttackerDamageMod: grudgeBearerDamageMod,
	})
	RegisterPerkHooks("counterpunch", &PerkHooks{
		TurnStart:         counterpunchTurnStart,
		AttackerDamageMod: counterpunchDamageMod,
	})
	RegisterPerkHooks("marked_for_death", &PerkHooks{
		AttackerDamageMod: markedForDeathDamageMod,
	})
	RegisterPerkHooks("deadshots_patience", &PerkHooks{
		TurnStart:         deadshotTurnStart,
		AttackerDamageMod: deadshotDamageMod,
	})
}

// ========================================
// TIER 1: COMBAT CONDITIONING (10 perks)
// ========================================

// Brace for Impact: +15% cover bonus when defending
func braceForImpactCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown) {
	coverBreakdown.TotalReduction += 0.15
	if coverBreakdown.TotalReduction > 1.0 {
		coverBreakdown.TotalReduction = 1.0
	}
}

// Reckless Assault (attacker): +30% damage, sets vulnerability flag
func recklessAssaultAttackerMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	modifiers.DamageMultiplier *= 1.3
	ctx.RoundState.RecklessVulnerable = true
}

// Reckless Assault (defender): +20% incoming damage when vulnerable
func recklessAssaultDefenderMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.RecklessVulnerable {
		modifiers.DamageMultiplier *= 1.2
	}
}

// Stalwart: Full-damage counters if squad did NOT move
func stalwartCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) bool {
	if !roundState.MovedThisTurn {
		modifiers.DamageMultiplier = 1.0 // Override 0.5 default
	}
	return false // Don't skip counter
}

// Executioner's Instinct: +25% crit chance vs squads with any unit below 30% HP
func executionerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	for _, unitID := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](ctx.Manager, unitID, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		maxHP := attr.GetMaxHealth()
		if maxHP > 0 && float64(attr.CurrentHealth)/float64(maxHP) < 0.3 {
			modifiers.CritBonus += 25
			return
		}
	}
}

// Shieldwall Discipline: Per Tank in row 0, -5% damage (max 15%)
func shieldwallDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	tankCount := 0
	unitIDs := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	for _, unitID := range unitIDs {
		entity := ctx.Manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}
		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != unitdefs.RoleTank {
			continue
		}
		gridPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		if gridPos != nil && gridPos.AnchorRow == 0 {
			tankCount++
		}
	}
	if tankCount > 3 {
		tankCount = 3
	}
	if tankCount > 0 {
		reduction := float64(tankCount) * 0.05
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}
}

// Isolated Predator: +25% damage when no friendly squads within 3 tiles
func isolatedPredatorDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		ctx.Manager, ctx.AttackerSquadID, common.PositionComponent,
	)
	if squadPos == nil {
		return
	}
	attackerFaction := combatcore.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if attackerFaction == 0 {
		return
	}
	friendlySquads := combatcore.GetActiveSquadsForFaction(attackerFaction, ctx.Manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == ctx.AttackerSquadID {
			continue
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			ctx.Manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}
		if squadPos.ChebyshevDistance(friendlyPos) <= 3 {
			return
		}
	}
	modifiers.DamageMultiplier *= 1.25
}

// Vigilance: Crits become normal hits
func vigilanceDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	modifiers.SkipCrit = true
}

// Field Medic: At round start, lowest-HP unit heals 10% max HP
func fieldMedicTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	var lowestID ecs.EntityID
	lowestHP := 999999

	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestID = uid
		}
	}

	if lowestID != 0 {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, lowestID, common.AttributeComponent,
		)
		if attr != nil {
			maxHP := attr.GetMaxHealth()
			healAmount := maxHP / 10
			if healAmount < 1 {
				healAmount = 1
			}
			attr.CurrentHealth += healAmount
			if attr.CurrentHealth > maxHP {
				attr.CurrentHealth = maxHP
			}
		}
	}
}

// Opening Salvo: +35% damage on squad's first attack of the combat only
func openingSalvoDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if modifiers.IsCounterattack {
		return
	}
	if !ctx.RoundState.HasAttackedThisCombat {
		modifiers.DamageMultiplier *= 1.35
		ctx.RoundState.HasAttackedThisCombat = true
	}
}

// Last Line: When last friendly squad alive, +20% hit, dodge, and damage
func lastLineDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	faction := combatcore.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if faction == 0 {
		return
	}
	aliveSquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
	if len(aliveSquads) == 1 && aliveSquads[0] == ctx.AttackerSquadID {
		modifiers.DamageMultiplier *= 1.2
		modifiers.HitPenalty -= 20
	}
}

// ========================================
// TIER 2: COMBAT SPECIALIZATION (14 perks)
// ========================================

// Cleave: Hit target row + row behind, but -30% damage to ALL targets
func cleaveTargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil || targetData.AttackType != unitdefs.AttackTypeMeleeRow {
		return defaultTargets
	}
	if len(defaultTargets) == 0 {
		return defaultTargets
	}
	pos := common.GetComponentTypeByID[*squadcore.GridPositionData](
		ctx.Manager, defaultTargets[0], squadcore.GridPositionComponent,
	)
	if pos == nil {
		return defaultTargets
	}
	nextRow := pos.AnchorRow + 1
	if nextRow <= 2 {
		extraTargets := GetUnitsInRow(ctx.DefenderSquadID, nextRow, ctx.Manager)
		if len(extraTargets) > 0 {
			return append(defaultTargets, extraTargets...)
		}
	}
	return defaultTargets
}

// cleaveDamageMod applies the -30% damage penalty when Cleave is active
func cleaveDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData != nil && targetData.AttackType == unitdefs.AttackTypeMeleeRow {
		modifiers.DamageMultiplier *= 0.7
	}
}

// Riposte: Counterattacks have no hit penalty (normally -20)
func riposteCounterMod(defenderID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, roundState *PerkRoundState,
	manager *common.EntityManager) bool {
	modifiers.HitPenalty = 0
	return false
}

// Disruption: Dealing damage reduces target squad's next attack by -15% this round
func disruptionPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if damageDealt <= 0 {
		return
	}
	if ctx.RoundState.DisruptionTargets == nil {
		ctx.RoundState.DisruptionTargets = make(map[ecs.EntityID]bool)
	}
	ctx.RoundState.DisruptionTargets[ctx.DefenderSquadID] = true

	defenderState := GetRoundState(ctx.DefenderSquadID, ctx.Manager)
	if defenderState != nil {
		if defenderState.DisruptionTargets == nil {
			defenderState.DisruptionTargets = make(map[ecs.EntityID]bool)
		}
		defenderState.DisruptionTargets[ctx.AttackerSquadID] = true
	}
}

// Guardian Protocol: Redirect 25% damage to adjacent tank
func guardianDamageRedirect(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	defenderPos := common.GetComponentTypeByID[*coords.LogicalPosition](
		manager, defenderSquadID, common.PositionComponent,
	)
	if defenderPos == nil {
		return damageAmount, 0, 0
	}
	faction := combatcore.GetSquadFaction(defenderSquadID, manager)
	if faction == 0 {
		return damageAmount, 0, 0
	}
	friendlySquads := combatcore.GetActiveSquadsForFaction(faction, manager)
	for _, friendlyID := range friendlySquads {
		if friendlyID == defenderSquadID {
			continue
		}
		if !HasPerk(friendlyID, "guardian_protocol", manager) {
			continue
		}
		friendlyPos := common.GetComponentTypeByID[*coords.LogicalPosition](
			manager, friendlyID, common.PositionComponent,
		)
		if friendlyPos == nil {
			continue
		}
		if defenderPos.ChebyshevDistance(friendlyPos) > 1 {
			continue
		}
		unitIDs := squadcore.GetUnitIDsInSquad(friendlyID, manager)
		for _, unitID := range unitIDs {
			attr := squadcore.GetAliveUnitAttributes(unitID, manager)
			if attr == nil {
				continue
			}
			entity := manager.FindEntityByID(unitID)
			if entity == nil {
				continue
			}
			roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
			if roleData != nil && roleData.Role == unitdefs.RoleTank {
				guardianDmg := damageAmount / 4
				remainingDmg := damageAmount - guardianDmg
				return remainingDmg, unitID, guardianDmg
			}
		}
	}
	return damageAmount, 0, 0
}

// Overwatch: Skip attack to auto-attack at 75% damage next enemy that moves in range
func overwatchTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	// Placeholder — the actual trigger happens in the movement system (not implemented in v1).
}

// Adaptive Armor: -10% damage from same attacker per hit (stacks to 30%, resets per round)
func adaptiveArmorDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.AttackedBy == nil {
		ctx.RoundState.AttackedBy = make(map[ecs.EntityID]int)
	}
	hits := ctx.RoundState.AttackedBy[ctx.AttackerSquadID]
	if hits > 3 {
		hits = 3
	}
	if hits > 0 {
		reduction := float64(hits) * 0.10
		modifiers.DamageMultiplier *= (1.0 - reduction)
	}
	ctx.RoundState.AttackedBy[ctx.AttackerSquadID]++
}

// Bloodlust: Track kills and boost damage
func bloodlustPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if wasKill {
		ctx.RoundState.KillsThisRound++
	}
}

func bloodlustDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.KillsThisRound > 0 {
		bonus := 1.0 + float64(ctx.RoundState.KillsThisRound)*0.15
		modifiers.DamageMultiplier *= bonus
	}
}

// Fortify: +0.05 cover per consecutive turn not moving (max +0.15 after 3 turns)
func fortifyTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.MovedThisTurn {
		roundState.TurnsStationary = 0
	} else {
		if roundState.TurnsStationary < 3 {
			roundState.TurnsStationary++
		}
	}
}

func fortifyCoverMod(ctx *HookContext, coverBreakdown *combatcore.CoverBreakdown) {
	if ctx.RoundState.TurnsStationary > 0 {
		bonus := float64(ctx.RoundState.TurnsStationary) * 0.05
		coverBreakdown.TotalReduction += bonus
		if coverBreakdown.TotalReduction > 1.0 {
			coverBreakdown.TotalReduction = 1.0
		}
	}
}

// Precision Strike: Highest-dex DPS unit targets the lowest-HP enemy
func precisionStrikeTargetOverride(ctx *HookContext, defaultTargets []ecs.EntityID) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(ctx.AttackerID, ctx.Manager)
	if attackerSquadID == 0 {
		return defaultTargets
	}

	unitIDs := squadcore.GetUnitIDsInSquad(attackerSquadID, ctx.Manager)
	var highestDexID ecs.EntityID
	highestDex := 0
	for _, uid := range unitIDs {
		entity := ctx.Manager.FindEntityByID(uid)
		if entity == nil {
			continue
		}
		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}
		roleData := common.GetComponentType[*squadcore.UnitRoleData](entity, squadcore.UnitRoleComponent)
		if roleData == nil || roleData.Role != unitdefs.RoleDPS {
			continue
		}
		if attr.Dexterity > highestDex {
			highestDex = attr.Dexterity
			highestDexID = uid
		}
	}

	if highestDexID != ctx.AttackerID {
		return defaultTargets
	}

	enemyUnits := squadcore.GetUnitIDsInSquad(ctx.DefenderSquadID, ctx.Manager)
	var lowestHPID ecs.EntityID
	lowestHP := 999999
	for _, uid := range enemyUnits {
		attr := squadcore.GetAliveUnitAttributes(uid, ctx.Manager)
		if attr == nil {
			continue
		}
		if attr.CurrentHealth < lowestHP {
			lowestHP = attr.CurrentHealth
			lowestHPID = uid
		}
	}

	if lowestHPID != 0 {
		return []ecs.EntityID{lowestHPID}
	}
	return defaultTargets
}

// Resolute: Survive lethal damage at 1 HP (once per unit per battle)
func resoluteTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.RoundStartHP == nil {
		roundState.RoundStartHP = make(map[ecs.EntityID]int)
	}
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	for _, uid := range unitIDs {
		attr := common.GetComponentTypeByID[*common.Attributes](
			manager, uid, common.AttributeComponent,
		)
		if attr != nil && attr.CurrentHealth > 0 {
			roundState.RoundStartHP[uid] = attr.CurrentHealth
		}
	}
}

func resoluteDeathOverride(unitID, squadID ecs.EntityID,
	roundState *PerkRoundState, manager *common.EntityManager) bool {
	if roundState.ResoluteUsed == nil {
		roundState.ResoluteUsed = make(map[ecs.EntityID]bool)
	}
	if roundState.ResoluteUsed[unitID] {
		return false
	}
	attr := common.GetComponentTypeByID[*common.Attributes](
		manager, unitID, common.AttributeComponent,
	)
	if attr == nil {
		return false
	}
	roundStartHP, ok := roundState.RoundStartHP[unitID]
	if !ok {
		return false
	}
	maxHP := attr.GetMaxHealth()
	if maxHP > 0 && float64(roundStartHP)/float64(maxHP) > 0.5 {
		roundState.ResoluteUsed[unitID] = true
		return true
	}
	return false
}

// Grudge Bearer: +20% damage vs enemy squads that have damaged this squad (stacks to +40%)
func grudgeBearerPostDamage(ctx *HookContext, damageDealt int, wasKill bool) {
	if damageDealt <= 0 {
		return
	}
	if ctx.RoundState.GrudgeStacks == nil {
		ctx.RoundState.GrudgeStacks = make(map[ecs.EntityID]int)
	}
	current := ctx.RoundState.GrudgeStacks[ctx.AttackerSquadID]
	if current < 2 {
		ctx.RoundState.GrudgeStacks[ctx.AttackerSquadID] = current + 1
	}
}

func grudgeBearerDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.GrudgeStacks != nil {
		stacks := ctx.RoundState.GrudgeStacks[ctx.DefenderSquadID]
		if stacks > 0 {
			bonus := 1.0 + float64(stacks)*0.20
			modifiers.DamageMultiplier *= bonus
		}
	}
}

// Counterpunch: +40% damage if attacked last turn AND did not attack last turn
func counterpunchTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.WasAttackedLastTurn && roundState.DidNotAttackLastTurn {
		roundState.CounterpunchReady = true
	} else {
		roundState.CounterpunchReady = false
	}
}

func counterpunchDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if ctx.RoundState.CounterpunchReady {
		modifiers.DamageMultiplier *= 1.4
		ctx.RoundState.CounterpunchReady = false
	}
}

// Marked for Death: +25% damage to marked target
func markedForDeathDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	faction := combatcore.GetSquadFaction(ctx.AttackerSquadID, ctx.Manager)
	if faction == 0 {
		return
	}
	friendlySquads := combatcore.GetActiveSquadsForFaction(faction, ctx.Manager)
	for _, friendlyID := range friendlySquads {
		friendlyState := GetRoundState(friendlyID, ctx.Manager)
		if friendlyState != nil && friendlyState.MarkedSquad == ctx.DefenderSquadID {
			modifiers.DamageMultiplier *= 1.25
			friendlyState.MarkedSquad = 0
			return
		}
	}
}

// Deadshot's Patience: +50% damage and +20 accuracy if completely idle last turn
func deadshotTurnStart(squadID ecs.EntityID, roundNumber int,
	roundState *PerkRoundState, manager *common.EntityManager) {
	if roundState.WasIdleLastTurn {
		roundState.DeadshotReady = true
	} else {
		roundState.DeadshotReady = false
	}
}

func deadshotDamageMod(ctx *HookContext, modifiers *combatcore.DamageModifiers) {
	if !ctx.RoundState.DeadshotReady {
		return
	}
	targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](
		ctx.Manager, ctx.AttackerID, squadcore.TargetRowComponent,
	)
	if targetData == nil {
		return
	}
	if targetData.AttackType == unitdefs.AttackTypeRanged || targetData.AttackType == unitdefs.AttackTypeMagic {
		modifiers.DamageMultiplier *= 1.5
		modifiers.HitPenalty -= 20
		ctx.RoundState.DeadshotReady = false
	}
}

// ========================================
// HELPER: GetUnitsInRow (exported for Cleave)
// ========================================

// GetUnitsInRow returns all alive units in a specific row of a squad.
func GetUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for col := 0; col <= 2; col++ {
		cellUnits := squadcore.GetUnitIDsAtGridPosition(squadID, row, col, manager)
		for _, unitID := range cellUnits {
			if !seen[unitID] {
				if squadcore.GetAliveUnitAttributes(unitID, manager) != nil {
					units = append(units, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return units
}
