package artifacts

import (
	"encoding/json"
	"errors"
	"fmt"
	"game_main/core/config"
	"game_main/tactical/powers/powercore"
	"log"
	"os"
)

// ArtifactBalanceConfig holds all artifact balance tuning values, loaded from JSON.
type ArtifactBalanceConfig struct {
	SaboteursHourglass SaboteursHourglassBalance `json:"saboteursHourglass"`
}

type SaboteursHourglassBalance struct {
	MovementReduction int `json:"movementReduction" balance:"count"`
}

// ArtifactBalance is the global artifact balance config, loaded at startup.
var ArtifactBalance ArtifactBalanceConfig

const artifactBalancePath = "gamedata/artifactbalanceconfig.json"

// LoadArtifactBalanceConfig reads artifact balance tuning values from JSON.
// Returns an error if the file is missing, unparseable, or fails validation —
// any of these silently zero out tuning values and produce no-op artifacts
// (e.g. Saboteur's Hourglass with MovementReduction = 0).
func LoadArtifactBalanceConfig() error {
	data, err := os.ReadFile(config.AssetPath(artifactBalancePath))
	if err != nil {
		return fmt.Errorf("read %s: %w", artifactBalancePath, err)
	}

	if err := json.Unmarshal(data, &ArtifactBalance); err != nil {
		return fmt.Errorf("parse %s: %w", artifactBalancePath, err)
	}

	if err := validateArtifactBalance(&ArtifactBalance); err != nil {
		return fmt.Errorf("validate %s: %w", artifactBalancePath, err)
	}

	log.Println("Artifact balance config loaded")
	return nil
}

func validateArtifactBalance(cfg *ArtifactBalanceConfig) error {
	if errs := powercore.ValidateBalanceRanges(cfg); len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
