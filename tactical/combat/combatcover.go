package combat

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

func GetCoverProvidersFor(defenderID ecs.EntityID, defenderSquadID ecs.EntityID, defenderPos *squads.GridPositionData, squadmanager *common.EntityManager) []ecs.EntityID {
	var providers []ecs.EntityID

	defenderCols := make(map[int]bool)
	for c := defenderPos.AnchorCol; c < defenderPos.AnchorCol+defenderPos.Width && c < 3; c++ {
		defenderCols[c] = true
	}

	allUnitIDs := squads.GetUnitIDsInSquad(defenderSquadID, squadmanager)

	for _, unitID := range allUnitIDs {
		if unitID == defenderID {
			continue
		}
		entity := squadmanager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}
		if !entity.HasComponent(squads.CoverComponent) {
			continue
		}
		coverData := common.GetComponentType[*squads.CoverData](entity, squads.CoverComponent)
		if coverData == nil {
			continue
		}
		if !entity.HasComponent(squads.GridPositionComponent) {
			continue
		}
		unitPos := common.GetComponentType[*squads.GridPositionData](entity, squads.GridPositionComponent)
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

func CalculateCoverBreakdown(defenderID ecs.EntityID, squadmanager *common.EntityManager) CoverBreakdown {
	breakdown := CoverBreakdown{
		Providers: []CoverProvider{},
	}
	defenderEntity := squadmanager.FindEntityByID(defenderID)
	if defenderEntity == nil {
		return breakdown
	}

	if !defenderEntity.HasComponent(squads.GridPositionComponent) || !defenderEntity.HasComponent(squads.SquadMemberComponent) {
		return breakdown
	}

	defenderPos := common.GetComponentType[*squads.GridPositionData](defenderEntity, squads.GridPositionComponent)
	defenderSquadData := common.GetComponentType[*squads.SquadMemberData](defenderEntity, squads.SquadMemberComponent)
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

		if !providerEntity.HasComponent(squads.CoverComponent) {
			continue
		}

		coverData := common.GetComponentType[*squads.CoverData](providerEntity, squads.CoverComponent)
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
			providerPos := common.GetComponentType[*squads.GridPositionData](providerEntity, squads.GridPositionComponent)

			unitName := "Unknown"
			if name != nil {
				unitName = name.NameStr
			}

			provider := CoverProvider{
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
