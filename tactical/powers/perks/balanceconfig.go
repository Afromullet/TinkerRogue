package perks

import (
	"encoding/json"
	"fmt"
	"game_main/setup/config"
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
		fmt.Printf("WARNING: Failed to read perk balance config: %v\n", err)
		return
	}

	if err := json.Unmarshal(data, &PerkBalance); err != nil {
		fmt.Printf("WARNING: Failed to parse perk balance config: %v\n", err)
		return
	}

	validatePerkBalance(&PerkBalance)
	fmt.Println("Perk balance config loaded")
}

func validatePerkBalance(cfg *PerkBalanceConfig) {
	// Existing checks
	if cfg.BraceForImpact.CoverBonus <= 0 {
		fmt.Println("WARNING: braceForImpact.coverBonus should be positive")
	}
	if cfg.ExecutionersInstinct.HPThreshold <= 0 || cfg.ExecutionersInstinct.HPThreshold >= 1.0 {
		fmt.Println("WARNING: executionersInstinct.hpThreshold should be between 0 and 1")
	}
	if cfg.ShieldwallDiscipline.MaxTanks <= 0 {
		fmt.Println("WARNING: shieldwallDiscipline.maxTanks should be positive")
	}
	if cfg.FieldMedic.HealDivisor <= 0 {
		fmt.Println("WARNING: fieldMedic.healDivisor should be positive")
	}
	if cfg.GuardianProtocol.RedirectFraction <= 0 {
		fmt.Println("WARNING: guardianProtocol.redirectFraction should be positive")
	}
	if cfg.Resolute.HPThreshold <= 0 || cfg.Resolute.HPThreshold >= 1.0 {
		fmt.Println("WARNING: resolute.hpThreshold should be between 0 and 1")
	}
	if cfg.GrudgeBearer.MaxStacks <= 0 {
		fmt.Println("WARNING: grudgeBearer.maxStacks should be positive")
	}
	// Previously missing checks — multipliers that would zero damage if missing from JSON
	if cfg.IsolatedPredator.DamageMult <= 0 {
		fmt.Println("WARNING: isolatedPredator.damageMult should be positive")
	}
	if cfg.IsolatedPredator.Range <= 0 {
		fmt.Println("WARNING: isolatedPredator.range should be positive")
	}
	if cfg.LastLine.DamageMult <= 0 {
		fmt.Println("WARNING: lastLine.damageMult should be positive")
	}
	if cfg.Cleave.DamageMult <= 0 {
		fmt.Println("WARNING: cleave.damageMult should be positive")
	}
	if cfg.RecklessAssault.AttackerMult <= 0 {
		fmt.Println("WARNING: recklessAssault.attackerMult should be positive")
	}
	if cfg.RecklessAssault.DefenderMult <= 0 {
		fmt.Println("WARNING: recklessAssault.defenderMult should be positive")
	}
	if cfg.Fortify.MaxStationaryTurns <= 0 {
		fmt.Println("WARNING: fortify.maxStationaryTurns should be positive")
	}
	if cfg.Fortify.PerTurnCoverBonus <= 0 {
		fmt.Println("WARNING: fortify.perTurnCoverBonus should be positive")
	}
	if cfg.Counterpunch.DamageMult <= 0 {
		fmt.Println("WARNING: counterpunch.damageMult should be positive")
	}
	if cfg.DeadshotsPatience.DamageMult <= 0 {
		fmt.Println("WARNING: deadshotsPatience.damageMult should be positive")
	}
	if cfg.AdaptiveArmor.MaxHits <= 0 {
		fmt.Println("WARNING: adaptiveArmor.maxHits should be positive")
	}
	if cfg.AdaptiveArmor.PerHitReduction <= 0 {
		fmt.Println("WARNING: adaptiveArmor.perHitReduction should be positive")
	}
	if cfg.Bloodlust.PerKillBonus <= 0 {
		fmt.Println("WARNING: bloodlust.perKillBonus should be positive")
	}
	if cfg.OpeningSalvo.DamageMult <= 0 {
		fmt.Println("WARNING: openingSalvo.damageMult should be positive")
	}
}
