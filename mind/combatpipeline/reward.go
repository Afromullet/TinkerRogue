package combatpipeline

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/spells"
	"game_main/tactical/squads"
	"math"
	"math/rand"
	"strings"
	"time"

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

// grantGold finds the player's ResourceStockpile and adds gold.
func grantGold(manager *common.EntityManager, playerEntityID ecs.EntityID, amount int) string {
	resources := common.GetResourceStockpile(playerEntityID, manager)
	if resources == nil {
		return ""
	}
	common.AddGold(resources, amount)
	fmt.Printf("Granted %d gold to player %d\n", amount, playerEntityID)
	return fmt.Sprintf("%d gold", amount)
}

// grantExperience distributes XP evenly across all alive units in the given squads.
func grantExperience(manager *common.EntityManager, squadIDs []ecs.EntityID, totalXP int) string {
	aliveUnitIDs := GetLivingUnitIDs(manager, squadIDs)

	if len(aliveUnitIDs) == 0 {
		return ""
	}

	xpPerUnit := totalXP / len(aliveUnitIDs)
	if xpPerUnit <= 0 {
		xpPerUnit = 1
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, unitID := range aliveUnitIDs {
		squads.AwardExperience(unitID, xpPerUnit, manager, rng)
	}

	fmt.Printf("Granted %d XP each to %d alive units (total %d XP)\n",
		xpPerUnit, len(aliveUnitIDs), totalXP)
	return fmt.Sprintf("%d XP", totalXP)
}

// grantMana restores mana to a commander, capped at max.
func grantMana(manager *common.EntityManager, commanderID ecs.EntityID, amount int) string {
	manaData := common.GetComponentTypeByID[*spells.ManaData](manager, commanderID, spells.ManaComponent)
	if manaData == nil {
		return ""
	}

	manaData.CurrentMana += amount
	if manaData.CurrentMana > manaData.MaxMana {
		manaData.CurrentMana = manaData.MaxMana
	}

	desc := fmt.Sprintf("%d mana (%d/%d)", amount, manaData.CurrentMana, manaData.MaxMana)
	fmt.Printf("Granted %s to commander %d\n", desc, commanderID)
	return desc
}

// FormatDescription joins non-empty reward description parts into a single string.
// Example: ["150 gold", "75 XP"] -> "150 gold, 75 XP"
func FormatDescription(parts []string) string {
	return strings.Join(parts, ", ")
}
