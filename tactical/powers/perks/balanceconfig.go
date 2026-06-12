package perks

import (
	"encoding/json"
	"errors"
	"fmt"
	"game_main/core/config"
	"game_main/tactical/powers/powercore"
	"log"
	"os"
)

// PerkBalanceConfig holds all perk balance tuning values, loaded from JSON.
//
// Every numeric field declares its valid range via a `balance` tag
// (fraction/mult/count/bonus — see powercore.ValidateBalanceRanges). Untagged
// numeric fields fail validation at load, so adding a tuning value here is
// the only step needed for it to be range-checked.
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
	CoverBonus float64 `json:"coverBonus" balance:"fraction"`
}

type ExecutionersInstinctBalance struct {
	HPThreshold float64 `json:"hpThreshold" balance:"fraction"`
	CritBonus   int     `json:"critBonus" balance:"bonus"`
}

type ShieldwallDisciplineBalance struct {
	MaxTanks         int     `json:"maxTanks" balance:"count"`
	PerTankReduction float64 `json:"perTankReduction" balance:"fraction"`
}

type IsolatedPredatorBalance struct {
	Range      int     `json:"range" balance:"count"`
	DamageMult float64 `json:"damageMult" balance:"mult"`
}

type FieldMedicBalance struct {
	HealDivisor int `json:"healDivisor" balance:"count"` // Max HP is divided by this value to get heal amount (e.g. 10 = heal 10% max HP)
}

type LastLineBalance struct {
	DamageMult float64 `json:"damageMult" balance:"mult"`
	HitBonus   int     `json:"hitBonus" balance:"bonus"`
}

type CleaveBalance struct {
	DamageMult float64 `json:"damageMult" balance:"mult"`
}

type GuardianProtocolBalance struct {
	RedirectFraction int `json:"redirectFraction" balance:"count"`
}

type RecklessAssaultBalance struct {
	AttackerMult float64 `json:"attackerMult" balance:"mult"`
	DefenderMult float64 `json:"defenderMult" balance:"mult"`
}

type FortifyBalance struct {
	MaxStationaryTurns int     `json:"maxStationaryTurns" balance:"count"`
	PerTurnCoverBonus  float64 `json:"perTurnCoverBonus" balance:"fraction"`
}

type CounterpunchBalance struct {
	DamageMult float64 `json:"damageMult" balance:"mult"`
}

type DeadshotsPatienceBalance struct {
	DamageMult    float64 `json:"damageMult" balance:"mult"`
	AccuracyBonus int     `json:"accuracyBonus" balance:"bonus"`
}

type AdaptiveArmorBalance struct {
	MaxHits         int     `json:"maxHits" balance:"count"`
	PerHitReduction float64 `json:"perHitReduction" balance:"fraction"`
}

type BloodlustBalance struct {
	PerKillBonus float64 `json:"perKillBonus" balance:"fraction"`
}

type OpeningSalvoBalance struct {
	DamageMult float64 `json:"damageMult" balance:"mult"`
}

type ResoluteBalance struct {
	HPThreshold float64 `json:"hpThreshold" balance:"fraction"`
}

type GrudgeBearerBalance struct {
	MaxStacks     int     `json:"maxStacks" balance:"count"`
	PerStackBonus float64 `json:"perStackBonus" balance:"fraction"`
}

// PerkBalance is the global perk balance config, loaded at startup.
var PerkBalance PerkBalanceConfig

const perkBalancePath = "gamedata/perkbalanceconfig.json"

// LoadPerkBalanceConfig reads perk balance tuning values from JSON.
// Returns an error if the file is missing, unparseable, or fails validation —
// any of these would leave PerkBalance zero-valued, silently no-oping every
// perk (and dividing by zero in FieldMedic).
func LoadPerkBalanceConfig() error {
	data, err := os.ReadFile(config.AssetPath(perkBalancePath))
	if err != nil {
		return fmt.Errorf("read %s: %w", perkBalancePath, err)
	}

	if err := json.Unmarshal(data, &PerkBalance); err != nil {
		return fmt.Errorf("parse %s: %w", perkBalancePath, err)
	}

	if errs := powercore.ValidateBalanceRanges(&PerkBalance); len(errs) > 0 {
		return fmt.Errorf("validate %s: %w", perkBalancePath, errors.Join(errs...))
	}

	log.Println("Perk balance config loaded")
	return nil
}
