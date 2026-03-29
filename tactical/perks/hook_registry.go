package perks

// PerkHooks collects all hooks for a single perk.
// A perk only populates the hooks it needs -- nil slots are skipped.
type PerkHooks struct {
	DamageMod      DamageModHook
	TargetOverride TargetOverrideHook
	CounterMod     CounterModHook
	PostDamage     PostDamageHook
	TurnStart      TurnStartHook
	CoverMod       CoverModHook
	DamageRedirect DamageRedirectHook
	DeathOverride  DeathOverrideHook
}

var hookRegistry = map[string]*PerkHooks{}

// RegisterPerkHooks registers a perk's hook implementations by behavior ID.
func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
	hookRegistry[perkID] = hooks
}

// GetPerkHooks returns the hook implementations for a perk, or nil if not found.
func GetPerkHooks(perkID string) *PerkHooks {
	return hookRegistry[perkID]
}

// registerAllPerkHooks registers all perk behavior implementations.
// Called from init() in behaviors.go to ensure all hooks are available.
func registerAllPerkHooks() {
	// Tier 1: Combat Conditioning
	RegisterPerkHooks("brace_for_impact", &PerkHooks{CoverMod: braceForImpactCoverMod})
	RegisterPerkHooks("reckless_assault", &PerkHooks{DamageMod: recklessAssaultDamageMod})
	RegisterPerkHooks("stalwart", &PerkHooks{CounterMod: stalwartCounterMod})
	RegisterPerkHooks("executioners_instinct", &PerkHooks{DamageMod: executionerDamageMod})
	RegisterPerkHooks("shieldwall_discipline", &PerkHooks{DamageMod: shieldwallDamageMod})
	RegisterPerkHooks("isolated_predator", &PerkHooks{DamageMod: isolatedPredatorDamageMod})
	RegisterPerkHooks("vigilance", &PerkHooks{DamageMod: vigilanceDamageMod})
	RegisterPerkHooks("field_medic", &PerkHooks{TurnStart: fieldMedicTurnStart})
	RegisterPerkHooks("opening_salvo", &PerkHooks{DamageMod: openingSalvoDamageMod})
	RegisterPerkHooks("last_line", &PerkHooks{DamageMod: lastLineDamageMod})

	// Tier 2: Combat Specialization
	RegisterPerkHooks("cleave", &PerkHooks{
		TargetOverride: cleaveTargetOverride,
		DamageMod:      cleaveDamageMod,
	})
	RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
	RegisterPerkHooks("disruption", &PerkHooks{PostDamage: disruptionPostDamage})
	RegisterPerkHooks("guardian_protocol", &PerkHooks{DamageRedirect: guardianDamageRedirect})
	RegisterPerkHooks("overwatch", &PerkHooks{TurnStart: overwatchTurnStart})
	RegisterPerkHooks("adaptive_armor", &PerkHooks{DamageMod: adaptiveArmorDamageMod})
	RegisterPerkHooks("bloodlust", &PerkHooks{
		PostDamage: bloodlustPostDamage,
		DamageMod:  bloodlustDamageMod,
	})
	RegisterPerkHooks("fortify", &PerkHooks{
		TurnStart: fortifyTurnStart,
		CoverMod:  fortifyCoverMod,
	})
	RegisterPerkHooks("precision_strike", &PerkHooks{TargetOverride: precisionStrikeTargetOverride})
	RegisterPerkHooks("resolute", &PerkHooks{
		TurnStart:     resoluteTurnStart,
		DeathOverride: resoluteDeathOverride,
	})
	RegisterPerkHooks("grudge_bearer", &PerkHooks{
		PostDamage: grudgeBearerPostDamage,
		DamageMod:  grudgeBearerDamageMod,
	})
	RegisterPerkHooks("counterpunch", &PerkHooks{
		TurnStart: counterpunchTurnStart,
		DamageMod: counterpunchDamageMod,
	})
	RegisterPerkHooks("marked_for_death", &PerkHooks{
		DamageMod: markedForDeathDamageMod,
	})
	RegisterPerkHooks("deadshots_patience", &PerkHooks{
		TurnStart: deadshotTurnStart,
		DamageMod: deadshotDamageMod,
	})
}
