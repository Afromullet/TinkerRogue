package resolution

import (
	"game_main/common"
	"math"

	"github.com/bytearena/ecs"
)

// Reward represents what a player receives. Zero-valued fields are skipped.
type Reward struct {
	Gold       int
	Experience int
	Mana       int
}

// Scale returns a new Reward with all values multiplied by factor.
// Used for partial rewards (e.g., Scale(0.5) for weakened threats).
func (r Reward) Scale(factor float64) Reward {
	return Reward{
		Gold:       int(math.Round(float64(r.Gold) * factor)),
		Experience: int(math.Round(float64(r.Experience) * factor)),
		Mana:       int(math.Round(float64(r.Mana) * factor)),
	}
}

// GrantTarget identifies who receives the reward.
// Callers construct this from their own context (combatOutcome, raidState, etc).
type GrantTarget struct {
	PlayerEntityID ecs.EntityID   // For gold (ResourceStockpile owner)
	SquadIDs       []ecs.EntityID // For XP distribution (alive units across squads)
	CommanderID    ecs.EntityID   // For mana restoration
}

// CalculateIntensityReward determines loot from defeating a threat.
// Reward multiplier is derived from intensity: 1.0 + (intensity x 0.1) gives 1.1x-1.5x for intensity 1-5.
func CalculateIntensityReward(intensity int) Reward {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	return Reward{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
	}
}

// Grant distributes a Reward to the target. Returns a description string.
// Skips any reward field that is zero.
func Grant(manager *common.EntityManager, r Reward, target GrantTarget) string {
	var parts []string

	if r.Gold > 0 && target.PlayerEntityID != 0 {
		if desc := grantGold(manager, target.PlayerEntityID, r.Gold); desc != "" {
			parts = append(parts, desc)
		}
	}

	if r.Experience > 0 && len(target.SquadIDs) > 0 {
		if desc := grantExperience(manager, target.SquadIDs, r.Experience); desc != "" {
			parts = append(parts, desc)
		}
	}

	if r.Mana > 0 && target.CommanderID != 0 {
		if desc := grantMana(manager, target.CommanderID, r.Mana); desc != "" {
			parts = append(parts, desc)
		}
	}

	return FormatDescription(parts)
}
