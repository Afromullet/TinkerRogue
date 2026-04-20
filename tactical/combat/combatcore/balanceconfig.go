package combatcore

import (
	"encoding/json"
	"fmt"
	"game_main/core/config"
	"os"
)

// CombatBalanceConfig holds combat balance tuning values, loaded from JSON.
type CombatBalanceConfig struct {
	Counterattack CounterattackBalance `json:"counterattack"`
}

type CounterattackBalance struct {
	DamageMultiplier float64 `json:"damageMultiplier"` // Fraction of normal damage on counterattack (e.g. 0.5 = 50%)
	HitPenalty       int     `json:"hitPenalty"`       // Hit chance penalty on counterattack (e.g. 20 = -20%)
}

// CombatBalance is the global combat balance config, loaded at startup.
var CombatBalance CombatBalanceConfig

const combatBalancePath = "gamedata/combatbalanceconfig.json"

// LoadCombatBalanceConfig reads combat balance tuning values from JSON.
func LoadCombatBalanceConfig() {
	data, err := os.ReadFile(config.AssetPath(combatBalancePath))
	if err != nil {
		fmt.Printf("WARNING: Failed to read combat balance config: %v\n", err)
		return
	}

	if err := json.Unmarshal(data, &CombatBalance); err != nil {
		fmt.Printf("WARNING: Failed to parse combat balance config: %v\n", err)
		return
	}

	validateCombatBalance(&CombatBalance)
	fmt.Println("Combat balance config loaded")
}

func validateCombatBalance(cfg *CombatBalanceConfig) {
	if cfg.Counterattack.DamageMultiplier <= 0 || cfg.Counterattack.DamageMultiplier > 1.0 {
		fmt.Println("WARNING: counterattack.damageMultiplier should be between 0 and 1")
	}
	if cfg.Counterattack.HitPenalty < 0 {
		fmt.Println("WARNING: counterattack.hitPenalty should be non-negative")
	}
}
