package artifacts

import (
	"encoding/json"
	"fmt"
	"game_main/setup/config"
	"os"
)

// ArtifactBalanceConfig holds all artifact balance tuning values, loaded from JSON.
type ArtifactBalanceConfig struct {
	SaboteursHourglass SaboteursHourglassBalance `json:"saboteursHourglass"`
}

type SaboteursHourglassBalance struct {
	MovementReduction int `json:"movementReduction"`
}

// ArtifactBalance is the global artifact balance config, loaded at startup.
var ArtifactBalance ArtifactBalanceConfig

const artifactBalancePath = "gamedata/artifactbalanceconfig.json"

// LoadArtifactBalanceConfig reads artifact balance tuning values from JSON.
func LoadArtifactBalanceConfig() {
	data, err := os.ReadFile(config.AssetPath(artifactBalancePath))
	if err != nil {
		fmt.Printf("WARNING: Failed to read artifact balance config: %v\n", err)
		return
	}

	if err := json.Unmarshal(data, &ArtifactBalance); err != nil {
		fmt.Printf("WARNING: Failed to parse artifact balance config: %v\n", err)
		return
	}

	validateArtifactBalance(&ArtifactBalance)
	fmt.Println("Artifact balance config loaded")
}

func validateArtifactBalance(cfg *ArtifactBalanceConfig) {
	if cfg.SaboteursHourglass.MovementReduction <= 0 {
		fmt.Println("WARNING: saboteursHourglass.movementReduction should be positive")
	}
}
