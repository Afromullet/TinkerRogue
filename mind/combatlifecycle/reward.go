package combatlifecycle

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/powers/progression"
	"game_main/tactical/powers/spells"
	"game_main/tactical/squads/unitprogression"
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
	ArcanaPts  int
	SkillPts   int
}

// Scale returns a new Reward with all values multiplied by factor.
// Used for partial rewards (e.g., Scale(0.5) for weakened threats).
func (r Reward) Scale(factor float64) Reward {
	return Reward{
		Gold:       int(math.Round(float64(r.Gold) * factor)),
		Experience: int(math.Round(float64(r.Experience) * factor)),
		Mana:       int(math.Round(float64(r.Mana) * factor)),
		ArcanaPts:  int(math.Round(float64(r.ArcanaPts) * factor)),
		SkillPts:   int(math.Round(float64(r.SkillPts) * factor)),
	}
}

// GrantTarget identifies who receives the reward.
// Callers construct this from their own context (combatOutcome, raidState, etc).
type GrantTarget struct {
	PlayerEntityID ecs.EntityID   // For gold (ResourceStockpile owner)
	SquadIDs       []ecs.EntityID // For XP and mana distribution
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

	if r.Mana > 0 && len(target.SquadIDs) > 0 {
		if desc := grantManaToSquads(manager, target.SquadIDs, r.Mana); desc != "" {
			parts = append(parts, desc)
		}
	}

	if r.ArcanaPts > 0 && target.PlayerEntityID != 0 {
		if desc := grantProgressionPoints(manager, target.PlayerEntityID, r.ArcanaPts, "Arcana", progression.AddArcanaPoints); desc != "" {
			parts = append(parts, desc)
		}
	}

	if r.SkillPts > 0 && target.PlayerEntityID != 0 {
		if desc := grantProgressionPoints(manager, target.PlayerEntityID, r.SkillPts, "Skill", progression.AddSkillPoints); desc != "" {
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
		unitprogression.AwardExperience(unitID, xpPerUnit, manager, rng)
	}

	fmt.Printf("Granted %d XP each to %d alive units (total %d XP)\n",
		xpPerUnit, len(aliveUnitIDs), totalXP)
	return fmt.Sprintf("%d XP", totalXP)
}

// grantManaToSquads restores mana to each surviving squad that has a mana pool.
func grantManaToSquads(manager *common.EntityManager, squadIDs []ecs.EntityID, amount int) string {
	restored := 0
	for _, squadID := range squadIDs {
		manaData := common.GetComponentTypeByID[*spells.ManaData](manager, squadID, spells.ManaComponent)
		if manaData == nil {
			continue
		}
		manaData.CurrentMana += amount
		if manaData.CurrentMana > manaData.MaxMana {
			manaData.CurrentMana = manaData.MaxMana
		}
		restored++
	}
	if restored == 0 {
		return ""
	}
	desc := fmt.Sprintf("%d mana to %d squads", amount, restored)
	fmt.Printf("Granted %s\n", desc)
	return desc
}

// grantProgressionPoints adds progression points of a given currency to the player's
// ProgressionData. Returns "" if the player has no progression component.
func grantProgressionPoints(
	manager *common.EntityManager,
	playerEntityID ecs.EntityID,
	amount int,
	label string,
	add func(ecs.EntityID, int, *common.EntityManager),
) string {
	if progression.GetProgression(playerEntityID, manager) == nil {
		return ""
	}
	add(playerEntityID, amount, manager)
	fmt.Printf("Granted %d %s to player %d\n", amount, label, playerEntityID)
	return fmt.Sprintf("%d %s", amount, label)
}

// FormatDescription joins non-empty reward description parts into a single string.
// Example: ["150 gold", "75 XP"] -> "150 gold, 75 XP"
func FormatDescription(parts []string) string {
	return strings.Join(parts, ", ")
}
