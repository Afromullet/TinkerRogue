package influence

import (
	"game_main/templates"
)

// --- Synergy Config ---

func getSynergyGrowthBonus() float64 {
	return templates.InfluenceConfigTemplate.Synergy.GrowthBonus
}

// --- Competition Config ---

func getCompetitionGrowthPenalty() float64 {
	return templates.InfluenceConfigTemplate.Competition.GrowthPenalty
}

// --- Suppression Config ---

func getSuppressionGrowthPenalty() float64 {
	return templates.InfluenceConfigTemplate.Suppression.GrowthPenalty
}

func getSuppressionNodeTypeMultiplier(nodeTypeID string) float64 {
	if mult, ok := templates.InfluenceConfigTemplate.Suppression.NodeTypeMultipliers[nodeTypeID]; ok {
		return mult
	}
	return 1.0
}

// --- Player Synergy Config ---

func getPlayerSynergyBaseBonus() float64 {
	return templates.InfluenceConfigTemplate.PlayerSynergy.BaseBonus
}

func getPlayerSynergyComplementaryBonus() float64 {
	return templates.InfluenceConfigTemplate.PlayerSynergy.ComplementaryBonus
}

func getComplementaryPairs() [][]string {
	return templates.InfluenceConfigTemplate.PlayerSynergy.ComplementaryPairs
}

// --- Diminishing Returns Config ---

func getDiminishingFactor() float64 {
	return templates.InfluenceConfigTemplate.DiminishingFactor
}
