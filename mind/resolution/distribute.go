package resolution

import (
	"fmt"
	"math/rand"
	"time"

	"game_main/common"
	"game_main/tactical/spells"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

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
	var aliveUnitIDs []ecs.EntityID
	for _, squadID := range squadIDs {
		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		for _, unitID := range unitIDs {
			unitEntity := manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attr != nil && attr.CurrentHealth > 0 {
				aliveUnitIDs = append(aliveUnitIDs, unitID)
			}
		}
	}

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
