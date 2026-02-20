package perks

// PerkHooks collects all hooks for a single perk.
// A perk only populates the hooks it needs -- nil slots are skipped by runners.
type PerkHooks struct {
	DamageMod      DamageModHook
	TargetOverride TargetOverrideHook
	CounterMod     CounterModHook
	PostDamage     PostDamageHook
	TurnStart      TurnStartHook
	CoverMod       CoverModHook
	DamageRedirect DamageRedirectHook
}

var hookRegistry = map[string]*PerkHooks{}

// RegisterPerkHooks registers hook functions for a behavioral perk.
func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
	hookRegistry[perkID] = hooks
}

// GetPerkHooks returns the hooks for a perk. Returns nil if no hooks registered.
func GetPerkHooks(perkID string) *PerkHooks {
	return hookRegistry[perkID]
}

// registerAllPerkHooks registers all behavioral perk hooks.
// Called from init() to wire up perk behaviors to the hook system.
func registerAllPerkHooks() {
	RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
	RegisterPerkHooks("stone_wall", &PerkHooks{
		CounterMod: stoneWallCounterMod,
		DamageMod:  stoneWallDamageMod,
	})
	RegisterPerkHooks("berserker", &PerkHooks{DamageMod: berserkerDamageMod})
	RegisterPerkHooks("armor_piercing", &PerkHooks{DamageMod: armorPiercingDamageMod})
	RegisterPerkHooks("glass_cannon", &PerkHooks{DamageMod: glassCannonDamageMod})
	RegisterPerkHooks("focus_fire", &PerkHooks{
		TargetOverride: focusFireTargetOverride,
		DamageMod:      focusFireDamageMod,
	})
	RegisterPerkHooks("cleave", &PerkHooks{TargetOverride: cleaveTargetOverride})
	RegisterPerkHooks("lifesteal", &PerkHooks{PostDamage: lifestealPostDamage})
	RegisterPerkHooks("inspiration", &PerkHooks{PostDamage: inspirationPostDamage})
	RegisterPerkHooks("impale", &PerkHooks{CoverMod: impaleCoverMod})
	RegisterPerkHooks("war_medic", &PerkHooks{TurnStart: warMedicTurnStart})
}

func init() {
	registerAllPerkHooks()
}
