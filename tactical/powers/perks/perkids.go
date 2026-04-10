package perks

// PerkID is a typed string for perk identifiers, providing compile-time safety.
type PerkID string

// Perk ID constants — single source of truth for perk string keys.
// Used in hook registration, state access, and HasPerk checks.
const (
	PerkBraceForImpact       PerkID = "brace_for_impact"
	PerkExecutionersInstinct PerkID = "executioners_instinct"
	PerkShieldwallDiscipline PerkID = "shieldwall_discipline"
	PerkIsolatedPredator     PerkID = "isolated_predator"
	PerkVigilance            PerkID = "vigilance"
	PerkFieldMedic           PerkID = "field_medic"
	PerkLastLine             PerkID = "last_line"
	PerkCleave               PerkID = "cleave"
	PerkRiposte              PerkID = "riposte"
	PerkGuardianProtocol     PerkID = "guardian_protocol"
	PerkPrecisionStrike      PerkID = "precision_strike"
	PerkRecklessAssault      PerkID = "reckless_assault"
	PerkStalwart             PerkID = "stalwart"
	PerkFortify              PerkID = "fortify"
	PerkCounterpunch         PerkID = "counterpunch"
	PerkDeadshotsPatience    PerkID = "deadshots_patience"
	PerkAdaptiveArmor        PerkID = "adaptive_armor"
	PerkBloodlust            PerkID = "bloodlust"
	PerkOpeningSalvo         PerkID = "opening_salvo"
	PerkResolute             PerkID = "resolute"
	PerkGrudgeBearer         PerkID = "grudge_bearer"
)
