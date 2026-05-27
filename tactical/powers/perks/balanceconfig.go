package perks

import (
	"encoding/json"
	"fmt"
	"game_main/core/config"
	"log"
	"os"
)

// PerkBalanceConfig holds all perk balance tuning values, loaded from JSON.
type PerkBalanceConfig struct {
	BraceForImpact       BraceForImpactBalance       `json:"braceForImpact"`
	ExecutionersInstinct ExecutionersInstinctBalance `json:"executionersInstinct"`
	ShieldwallDiscipline ShieldwallDisciplineBalance `json:"shieldwallDiscipline"`
	IsolatedPredator     IsolatedPredatorBalance     `json:"isolatedPredator"`
	FieldMedic           FieldMedicBalance           `json:"fieldMedic"`
	LastLine             LastLineBalance             `json:"lastLine"`
	Cleave               CleaveBalance               `json:"cleave"`
	GuardianProtocol     GuardianProtocolBalance     `json:"guardianProtocol"`
	RecklessAssault      RecklessAssaultBalance      `json:"recklessAssault"`
	Fortify              FortifyBalance              `json:"fortify"`
	Counterpunch         CounterpunchBalance         `json:"counterpunch"`
	DeadshotsPatience    DeadshotsPatienceBalance    `json:"deadshotsPatience"`
	AdaptiveArmor        AdaptiveArmorBalance        `json:"adaptiveArmor"`
	Bloodlust            BloodlustBalance            `json:"bloodlust"`
	OpeningSalvo         OpeningSalvoBalance         `json:"openingSalvo"`
	Resolute             ResoluteBalance             `json:"resolute"`
	GrudgeBearer         GrudgeBearerBalance         `json:"grudgeBearer"`
}

type BraceForImpactBalance struct {
	CoverBonus float64 `json:"coverBonus"`
}

type ExecutionersInstinctBalance struct {
	HPThreshold float64 `json:"hpThreshold"`
	CritBonus   int     `json:"critBonus"`
}

type ShieldwallDisciplineBalance struct {
	MaxTanks         int     `json:"maxTanks"`
	PerTankReduction float64 `json:"perTankReduction"`
}

type IsolatedPredatorBalance struct {
	Range      int     `json:"range"`
	DamageMult float64 `json:"damageMult"`
}

type FieldMedicBalance struct {
	HealDivisor int `json:"healDivisor"` // Max HP is divided by this value to get heal amount (e.g. 10 = heal 10% max HP)
}

type LastLineBalance struct {
	DamageMult float64 `json:"damageMult"`
	HitBonus   int     `json:"hitBonus"`
}

type CleaveBalance struct {
	DamageMult float64 `json:"damageMult"`
}

type GuardianProtocolBalance struct {
	RedirectFraction int `json:"redirectFraction"`
}

type RecklessAssaultBalance struct {
	AttackerMult float64 `json:"attackerMult"`
	DefenderMult float64 `json:"defenderMult"`
}

type FortifyBalance struct {
	MaxStationaryTurns int     `json:"maxStationaryTurns"`
	PerTurnCoverBonus  float64 `json:"perTurnCoverBonus"`
}

type CounterpunchBalance struct {
	DamageMult float64 `json:"damageMult"`
}

type DeadshotsPatienceBalance struct {
	DamageMult    float64 `json:"damageMult"`
	AccuracyBonus int     `json:"accuracyBonus"`
}

type AdaptiveArmorBalance struct {
	MaxHits         int     `json:"maxHits"`
	PerHitReduction float64 `json:"perHitReduction"`
}

type BloodlustBalance struct {
	PerKillBonus float64 `json:"perKillBonus"`
}

type OpeningSalvoBalance struct {
	DamageMult float64 `json:"damageMult"`
}

type ResoluteBalance struct {
	HPThreshold float64 `json:"hpThreshold"`
}

type GrudgeBearerBalance struct {
	MaxStacks     int     `json:"maxStacks"`
	PerStackBonus float64 `json:"perStackBonus"`
}

// PerkBalance is the global perk balance config, loaded at startup.
var PerkBalance PerkBalanceConfig

const perkBalancePath = "gamedata/perkbalanceconfig.json"

// LoadPerkBalanceConfig reads perk balance tuning values from JSON.
func LoadPerkBalanceConfig() {
	data, err := os.ReadFile(config.AssetPath(perkBalancePath))
	if err != nil {
		log.Printf("WARNING: Failed to read perk balance config: %v", err)
		return
	}

	if err := json.Unmarshal(data, &PerkBalance); err != nil {
		log.Printf("WARNING: Failed to parse perk balance config: %v", err)
		return
	}

	if errs := validatePerkBalance(&PerkBalance); len(errs) > 0 {
		for _, e := range errs {
			log.Printf("WARNING: perk balance config: %v", e)
		}
		if config.DEBUG_MODE {
			panic("perk balance config invalid — fix gamedata/perkbalanceconfig.json before running")
		}
	}
	log.Println("Perk balance config loaded")
}

// Range bounds catch typos in JSON that the previous "must be positive" checks
// would let through (e.g. DamageMult: 0.001 silently nerfing a perk, or
// MaxStationaryTurns: 1000 never capping). Fractions are bounded < 1.0,
// damage multipliers to [0.1, 10.0], and counts/bonuses to ≤ 100.
const (
	minDamageMult = 0.1
	maxDamageMult = 10.0
	maxCount      = 100
)

func validatePerkBalance(cfg *PerkBalanceConfig) []error {
	var errs []error
	if cfg.BraceForImpact.CoverBonus <= 0 || cfg.BraceForImpact.CoverBonus >= 1.0 {
		errs = append(errs, fmt.Errorf("braceForImpact.coverBonus must be in (0, 1)"))
	}
	if cfg.ExecutionersInstinct.HPThreshold <= 0 || cfg.ExecutionersInstinct.HPThreshold >= 1.0 {
		errs = append(errs, fmt.Errorf("executionersInstinct.hpThreshold must be between 0 and 1"))
	}
	if cfg.ExecutionersInstinct.CritBonus < 0 || cfg.ExecutionersInstinct.CritBonus > maxCount {
		errs = append(errs, fmt.Errorf("executionersInstinct.critBonus must be in [0, %d]", maxCount))
	}
	if cfg.ShieldwallDiscipline.MaxTanks <= 0 || cfg.ShieldwallDiscipline.MaxTanks > maxCount {
		errs = append(errs, fmt.Errorf("shieldwallDiscipline.maxTanks must be in (0, %d]", maxCount))
	}
	if cfg.ShieldwallDiscipline.PerTankReduction <= 0 || cfg.ShieldwallDiscipline.PerTankReduction >= 1.0 {
		errs = append(errs, fmt.Errorf("shieldwallDiscipline.perTankReduction must be in (0, 1)"))
	}
	if cfg.FieldMedic.HealDivisor <= 0 || cfg.FieldMedic.HealDivisor > maxCount {
		errs = append(errs, fmt.Errorf("fieldMedic.healDivisor must be in (0, %d]", maxCount))
	}
	if cfg.GuardianProtocol.RedirectFraction <= 0 || cfg.GuardianProtocol.RedirectFraction > maxCount {
		errs = append(errs, fmt.Errorf("guardianProtocol.redirectFraction must be in (0, %d]", maxCount))
	}
	if cfg.Resolute.HPThreshold <= 0 || cfg.Resolute.HPThreshold >= 1.0 {
		errs = append(errs, fmt.Errorf("resolute.hpThreshold must be between 0 and 1"))
	}
	if cfg.GrudgeBearer.MaxStacks <= 0 || cfg.GrudgeBearer.MaxStacks > maxCount {
		errs = append(errs, fmt.Errorf("grudgeBearer.maxStacks must be in (0, %d]", maxCount))
	}
	if cfg.GrudgeBearer.PerStackBonus <= 0 || cfg.GrudgeBearer.PerStackBonus >= 1.0 {
		errs = append(errs, fmt.Errorf("grudgeBearer.perStackBonus must be in (0, 1)"))
	}
	if cfg.IsolatedPredator.DamageMult < minDamageMult || cfg.IsolatedPredator.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("isolatedPredator.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.IsolatedPredator.Range <= 0 || cfg.IsolatedPredator.Range > maxCount {
		errs = append(errs, fmt.Errorf("isolatedPredator.range must be in (0, %d]", maxCount))
	}
	if cfg.LastLine.DamageMult < minDamageMult || cfg.LastLine.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("lastLine.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.LastLine.HitBonus < 0 || cfg.LastLine.HitBonus > maxCount {
		errs = append(errs, fmt.Errorf("lastLine.hitBonus must be in [0, %d]", maxCount))
	}
	if cfg.Cleave.DamageMult < minDamageMult || cfg.Cleave.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("cleave.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.RecklessAssault.AttackerMult < minDamageMult || cfg.RecklessAssault.AttackerMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("recklessAssault.attackerMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.RecklessAssault.DefenderMult < minDamageMult || cfg.RecklessAssault.DefenderMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("recklessAssault.defenderMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.Fortify.MaxStationaryTurns <= 0 || cfg.Fortify.MaxStationaryTurns > maxCount {
		errs = append(errs, fmt.Errorf("fortify.maxStationaryTurns must be in (0, %d]", maxCount))
	}
	if cfg.Fortify.PerTurnCoverBonus <= 0 || cfg.Fortify.PerTurnCoverBonus >= 1.0 {
		errs = append(errs, fmt.Errorf("fortify.perTurnCoverBonus must be in (0, 1)"))
	}
	if cfg.Counterpunch.DamageMult < minDamageMult || cfg.Counterpunch.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("counterpunch.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.DeadshotsPatience.DamageMult < minDamageMult || cfg.DeadshotsPatience.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("deadshotsPatience.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	if cfg.DeadshotsPatience.AccuracyBonus < 0 || cfg.DeadshotsPatience.AccuracyBonus > maxCount {
		errs = append(errs, fmt.Errorf("deadshotsPatience.accuracyBonus must be in [0, %d]", maxCount))
	}
	if cfg.AdaptiveArmor.MaxHits <= 0 || cfg.AdaptiveArmor.MaxHits > maxCount {
		errs = append(errs, fmt.Errorf("adaptiveArmor.maxHits must be in (0, %d]", maxCount))
	}
	if cfg.AdaptiveArmor.PerHitReduction <= 0 || cfg.AdaptiveArmor.PerHitReduction >= 1.0 {
		errs = append(errs, fmt.Errorf("adaptiveArmor.perHitReduction must be in (0, 1)"))
	}
	if cfg.Bloodlust.PerKillBonus <= 0 || cfg.Bloodlust.PerKillBonus >= 1.0 {
		errs = append(errs, fmt.Errorf("bloodlust.perKillBonus must be in (0, 1)"))
	}
	if cfg.OpeningSalvo.DamageMult < minDamageMult || cfg.OpeningSalvo.DamageMult > maxDamageMult {
		errs = append(errs, fmt.Errorf("openingSalvo.damageMult must be in [%v, %v]", minDamageMult, maxDamageMult))
	}
	return errs
}
