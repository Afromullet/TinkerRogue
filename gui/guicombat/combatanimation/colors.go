package combatanimation

import (
	"math"

	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatmath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

func createColorScale(r, g, b float32) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.SetR(r)
	cs.SetG(g)
	cs.SetB(b)
	cs.SetA(1.0)
	return cs
}

var attackColorPalette = []ebiten.ColorScale{
	createColorScale(2.0, 0.3, 0.3),
	createColorScale(0.3, 2.0, 0.3),
	createColorScale(0.3, 0.3, 2.0),
	createColorScale(2.0, 2.0, 0.0),
	createColorScale(2.0, 0.0, 2.0),
	createColorScale(0.0, 2.0, 2.0),
	createColorScale(2.0, 1.0, 0.0),
	createColorScale(1.0, 0.0, 2.0),
	createColorScale(2.0, 0.3, 1.0),
}

var healColorScale = createColorScale(1, 1, 1)

// SetCombatants sets the attacker and defender squads for the animation
func (cam *CombatAnimationMode) SetCombatants(attackerID, defenderID ecs.EntityID) {
	cam.attackerSquadID = attackerID
	cam.defenderSquadID = defenderID

	cam.attackerColors = make(map[ecs.EntityID]ebiten.ColorScale)
	cam.defenderFlashIndex = make(map[ecs.EntityID]int)
	cam.defenderColorList = make(map[ecs.EntityID][]ebiten.ColorScale)
	cam.flashTimer = 0

	combatSys := combatcore.NewCombatActionSystem(cam.Queries.ECSManager, cam.Queries.CombatCache)
	attackingUnits := combatSys.GetAttackingUnits(attackerID, defenderID)

	var nonHealAttackers []ecs.EntityID
	colorIdx := 0
	for _, attackerUnitID := range attackingUnits {
		if combatmath.IsHealUnit(attackerUnitID, cam.Queries.ECSManager) {
			cam.attackerColors[attackerUnitID] = healColorScale
		} else {
			cam.attackerColors[attackerUnitID] = attackColorPalette[colorIdx%len(attackColorPalette)]
			colorIdx++
			nonHealAttackers = append(nonHealAttackers, attackerUnitID)
		}
	}

	cam.computeDefenderColorLists(nonHealAttackers, defenderID)
	cam.computeHealTargetColors(attackingUnits, attackerID)
}

func (cam *CombatAnimationMode) computeDefenderColorLists(
	attackingUnits []ecs.EntityID,
	defenderSquadID ecs.EntityID,
) {
	defenderToAttackers := make(map[ecs.EntityID][]ecs.EntityID)

	for _, attackerID := range attackingUnits {
		targetRowData := cam.Queries.GetTargetRowData(attackerID)
		if targetRowData == nil {
			continue
		}

		targets := combatmath.SelectTargetUnits(attackerID, defenderSquadID, cam.Queries.ECSManager)

		for _, defenderID := range targets {
			defenderToAttackers[defenderID] = append(defenderToAttackers[defenderID], attackerID)
		}
	}

	for defenderID, attackerList := range defenderToAttackers {
		var colorList []ebiten.ColorScale
		for _, attackerID := range attackerList {
			if color, exists := cam.attackerColors[attackerID]; exists {
				colorList = append(colorList, color)
			}
		}
		cam.defenderColorList[defenderID] = colorList
		cam.defenderFlashIndex[defenderID] = 0
	}
}

func (cam *CombatAnimationMode) computeHealTargetColors(
	attackingUnits []ecs.EntityID,
	attackerSquadID ecs.EntityID,
) {
	for _, attackerID := range attackingUnits {
		if !combatmath.IsHealUnit(attackerID, cam.Queries.ECSManager) {
			continue
		}

		healTargets := combatmath.SelectHealTargets(attackerID, attackerSquadID, cam.Queries.ECSManager)

		for _, targetID := range healTargets {
			if _, exists := cam.defenderColorList[targetID]; !exists {
				cam.defenderColorList[targetID] = []ebiten.ColorScale{healColorScale}
				cam.defenderFlashIndex[targetID] = 0
			}
			if _, exists := cam.attackerColors[targetID]; !exists {
				cam.attackerColors[targetID] = healColorScale
			}
		}
	}
}

func (cam *CombatAnimationMode) getAttackHighlightColor(unitID ecs.EntityID) *ebiten.ColorScale {
	baseColor, exists := cam.attackerColors[unitID]
	if !exists {
		return cam.getDefaultAttackPulse()
	}

	pulse := float32(0.5 + 0.5*math.Sin(cam.animationTimer*2.0))

	colorScale := ebiten.ColorScale{}
	colorScale.SetR(baseColor.R() + pulse*0.3)
	colorScale.SetG(baseColor.G() + pulse*0.3)
	colorScale.SetB(baseColor.B() + pulse*0.3)
	colorScale.SetA(1.0)

	return &colorScale
}

func (cam *CombatAnimationMode) getDefaultAttackPulse() *ebiten.ColorScale {
	pulse := float32(0.5 + 0.5*float64(cam.animationTimer/AttackingDuration))
	colorScale := &ebiten.ColorScale{}
	colorScale.SetR(1.0 + pulse*0.5)
	colorScale.SetG(1.0 - pulse*0.3)
	colorScale.SetB(1.0 - pulse*0.5)
	colorScale.SetA(1.0)
	return colorScale
}

func (cam *CombatAnimationMode) getDefenderHighlightColor(unitID ecs.EntityID) *ebiten.ColorScale {
	colorList, exists := cam.defenderColorList[unitID]
	if !exists || len(colorList) == 0 {
		return nil
	}

	currentIndex := cam.defenderFlashIndex[unitID]
	if currentIndex >= len(colorList) {
		currentIndex = 0
	}

	return &colorList[currentIndex]
}
