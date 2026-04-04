package combatmath

import (
	"game_main/common"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

func GetCoverProvidersFor(defenderID ecs.EntityID, defenderSquadID ecs.EntityID, defenderPos *squadcore.GridPositionData, squadmanager *common.EntityManager) []ecs.EntityID {
	var providers []ecs.EntityID

	defenderCols := make(map[int]bool)
	for c := defenderPos.AnchorCol; c < defenderPos.AnchorCol+defenderPos.Width && c < 3; c++ {
		defenderCols[c] = true
	}

	allUnitIDs := squadcore.GetUnitIDsInSquad(defenderSquadID, squadmanager)

	for _, unitID := range allUnitIDs {
		if unitID == defenderID {
			continue
		}
		entity := squadmanager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}
		if !entity.HasComponent(squadcore.CoverComponent) {
			continue
		}
		coverData := common.GetComponentType[*squadcore.CoverData](entity, squadcore.CoverComponent)
		if coverData == nil {
			continue
		}
		if !entity.HasComponent(squadcore.GridPositionComponent) {
			continue
		}
		unitPos := common.GetComponentType[*squadcore.GridPositionData](entity, squadcore.GridPositionComponent)
		if unitPos == nil {
			continue
		}
		if unitPos.AnchorRow >= defenderPos.AnchorRow {
			continue
		}
		rowDistance := defenderPos.AnchorRow - unitPos.AnchorRow
		if rowDistance > coverData.CoverRange {
			continue
		}
		unitCols := make(map[int]bool)
		for c := unitPos.AnchorCol; c < unitPos.AnchorCol+unitPos.Width && c < 3; c++ {
			unitCols[c] = true
		}
		hasOverlap := false
		for col := range defenderCols {
			if unitCols[col] {
				hasOverlap = true
				break
			}
		}
		if hasOverlap {
			providers = append(providers, unitID)
		}
	}

	return providers
}

func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) combattypes.CoverBreakdown {
	breakdown := combattypes.CoverBreakdown{
		Providers: []combattypes.CoverProvider{},
	}
	defenderEntity := squadmanager.FindEntityByID(defenderID)
	if defenderEntity == nil {
		return breakdown
	}

	if !defenderEntity.HasComponent(squadcore.GridPositionComponent) || !defenderEntity.HasComponent(squadcore.SquadMemberComponent) {
		return breakdown
	}

	defenderPos := common.GetComponentType[*squadcore.GridPositionData](defenderEntity, squadcore.GridPositionComponent)
	defenderSquadData := common.GetComponentType[*squadcore.SquadMemberData](defenderEntity, squadcore.SquadMemberComponent)
	if defenderPos == nil || defenderSquadData == nil {
		return breakdown
	}
	defenderSquadID := defenderSquadData.SquadID

	providerIDs := GetCoverProvidersFor(defenderID, defenderSquadID, defenderPos, squadmanager)

	totalCover := 0.0
	for _, providerID := range providerIDs {
		providerEntity := squadmanager.FindEntityByID(providerID)
		if providerEntity == nil {
			continue
		}

		if !providerEntity.HasComponent(squadcore.CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*squadcore.CoverData](providerEntity, squadcore.CoverComponent)
		if coverData == nil {
			continue
		}

		isActive := true
		if coverData.RequiresActive {
			attr := common.GetComponentType[*common.Attributes](providerEntity, common.AttributeComponent)
			if attr != nil {
				isActive = attr.CurrentHealth > 0
			}
		}

		coverValue := coverData.GetCoverBonus(isActive)
		if coverValue > 0 {
			name := common.GetComponentType[*common.Name](providerEntity, common.NameComponent)
			providerPos := common.GetComponentType[*squadcore.GridPositionData](providerEntity, squadcore.GridPositionComponent)

			unitName := "Unknown"
			if name != nil {
				unitName = name.NameStr
			}

			provider := combattypes.CoverProvider{
				UnitID:     providerID,
				UnitName:   unitName,
				CoverValue: coverValue,
			}
			if providerPos != nil {
				provider.GridRow = providerPos.AnchorRow
				provider.GridCol = providerPos.AnchorCol
			}

			breakdown.Providers = append(breakdown.Providers, provider)
			totalCover += coverValue
		}
	}

	if totalCover > 1.0 {
		totalCover = 1.0
	}
	breakdown.TotalReduction = totalCover

	return breakdown
}
