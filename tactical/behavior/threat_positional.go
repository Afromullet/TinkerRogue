package behavior

import (
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"math"

	"github.com/bytearena/ecs"
)

// PositionalRiskLayer computes tactical risk based on positioning
// Identifies flanking exposure, isolation, and retreat path quality
type PositionalRiskLayer struct {
	*ThreatLayerBase

	// Core risk data
	flankingRisk       map[coords.LogicalPosition]float64 // Position -> flank exposure
	isolationRisk      map[coords.LogicalPosition]float64 // Position -> isolation penalty
	engagementPressure map[coords.LogicalPosition]float64 // Position -> net damage exposure
	retreatQuality     map[coords.LogicalPosition]float64 // Position -> escape route quality

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
	meleeLayer    *MeleeThreatLayer
	rangedLayer   *RangedThreatLayer
}

// NewPositionalRiskLayer creates a new positional risk layer
func NewPositionalRiskLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combat.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
	meleeLayer *MeleeThreatLayer,
	rangedLayer *RangedThreatLayer,
) *PositionalRiskLayer {
	return &PositionalRiskLayer{
		ThreatLayerBase:    NewThreatLayerBase(factionID, manager, cache),
		flankingRisk:       make(map[coords.LogicalPosition]float64),
		isolationRisk:      make(map[coords.LogicalPosition]float64),
		engagementPressure: make(map[coords.LogicalPosition]float64),
		retreatQuality:     make(map[coords.LogicalPosition]float64),
		baseThreatMgr:      baseThreatMgr,
		meleeLayer:         meleeLayer,
		rangedLayer:        rangedLayer,
	}
}

// Compute recalculates positional risks
func (prl *PositionalRiskLayer) Compute() {
	// Clear existing data (reuse maps to reduce GC pressure)
	clear(prl.flankingRisk)
	clear(prl.isolationRisk)
	clear(prl.engagementPressure)
	clear(prl.retreatQuality)

	// Get all squads (allies and enemies)
	alliedSquads := combat.GetSquadsForFaction(prl.factionID, prl.manager)
	enemyFactions := prl.getEnemyFactions()

	// Compute flanking risk based on enemy positions
	prl.computeFlankingRisk(enemyFactions)

	// Compute isolation risk based on ally positions
	prl.computeIsolationRisk(alliedSquads)

	// Compute engagement pressure using threat layers
	prl.computeEngagementPressure()

	// Compute retreat path quality
	prl.computeRetreatQuality(alliedSquads)

	prl.markClean(0)
}

// computeFlankingRisk identifies positions that can be attacked from multiple directions
func (prl *PositionalRiskLayer) computeFlankingRisk(enemyFactions []ecs.EntityID) {
	// For each position, count enemy threat directions
	threatDirections := make(map[coords.LogicalPosition]map[int]bool) // pos -> set of attack angles

	for _, enemyFactionID := range enemyFactions {
		squadIDs := combat.GetSquadsForFaction(enemyFactionID, prl.manager)

		for _, squadID := range squadIDs {
			if squads.IsSquadDestroyed(squadID, prl.manager) {
				continue
			}

			enemyPos, err := combat.GetSquadMapPosition(squadID, prl.manager)
			if err != nil {
				continue
			}

			// Get threat range for this squad
			moveSpeed := squads.GetSquadMovementSpeed(squadID, prl.manager)
			threatRange := moveSpeed + FlankingThreatRangeBonus

			// Paint threat directions
			for dx := -threatRange; dx <= threatRange; dx++ {
				for dy := -threatRange; dy <= threatRange; dy++ {
					pos := coords.LogicalPosition{X: enemyPos.X + dx, Y: enemyPos.Y + dy}
					distance := enemyPos.ChebyshevDistance(&pos)

					if distance > 0 && distance <= threatRange {
						// Calculate attack angle (simplified to 8 directions)
						angle := prl.getDirection(dx, dy)

						if threatDirections[pos] == nil {
							threatDirections[pos] = make(map[int]bool)
						}
						threatDirections[pos][angle] = true
					}
				}
			}
		}
	}

	// Calculate flanking risk based on number of threat directions
	for pos, directions := range threatDirections {
		numDirections := len(directions)

		// 1 direction = 0 risk
		// 2 directions = moderate risk
		// 3+ directions = high flanking risk
		if numDirections >= 3 {
			prl.flankingRisk[pos] = 1.0
		} else if numDirections == 2 {
			prl.flankingRisk[pos] = 0.5
		}
	}
}

// getDirection returns simplified direction (0-7) based on dx, dy
func (prl *PositionalRiskLayer) getDirection(dx, dy int) int {
	if dx == 0 && dy > 0 {
		return 0 // North
	} else if dx > 0 && dy > 0 {
		return 1 // NE
	} else if dx > 0 && dy == 0 {
		return 2 // East
	} else if dx > 0 && dy < 0 {
		return 3 // SE
	} else if dx == 0 && dy < 0 {
		return 4 // South
	} else if dx < 0 && dy < 0 {
		return 5 // SW
	} else if dx < 0 && dy == 0 {
		return 6 // West
	} else {
		return 7 // NW
	}
}

// computeIsolationRisk identifies positions far from allied support
func (prl *PositionalRiskLayer) computeIsolationRisk(alliedSquads []ecs.EntityID) {
	// Get ally positions
	allyPositions := []coords.LogicalPosition{}
	for _, squadID := range alliedSquads {
		if squads.IsSquadDestroyed(squadID, prl.manager) {
			continue
		}

		pos, err := combat.GetSquadMapPosition(squadID, prl.manager)
		if err != nil {
			continue
		}
		allyPositions = append(allyPositions, pos)
	}

	if len(allyPositions) == 0 {
		return
	}

	// For each potential position, find distance to nearest ally
	// Sample a grid of positions (optimization - don't check every tile)
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			pos := coords.LogicalPosition{X: x, Y: y}

			minDistance := math.MaxInt32
			for _, allyPos := range allyPositions {
				distance := pos.ChebyshevDistance(&allyPos)
				if distance < minDistance {
					minDistance = distance
				}
			}

			// Isolation risk increases with distance from nearest ally
			// See threat_constants.go for threshold definitions
			if minDistance >= IsolationHighDistance {
				prl.isolationRisk[pos] = 1.0
			} else if minDistance >= IsolationModerateDistance {
				prl.isolationRisk[pos] = float64(minDistance-IsolationSafeDistance) / 4.0 // 0.25 to 0.75
			}
		}
	}
}

// computeEngagementPressure combines melee and ranged threat for net pressure
func (prl *PositionalRiskLayer) computeEngagementPressure() {
	// Sample grid positions
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			pos := coords.LogicalPosition{X: x, Y: y}

			meleeThreat := prl.meleeLayer.GetMeleeThreatAt(pos)
			rangedThreat := prl.rangedLayer.GetRangedPressureAt(pos)

			// Total engagement pressure
			totalPressure := meleeThreat + rangedThreat

			// Normalize to 0-1 range using configured max pressure
			prl.engagementPressure[pos] = math.Min(totalPressure/EngagementPressureMax, 1.0)
		}
	}
}

// computeRetreatQuality evaluates escape route quality
func (prl *PositionalRiskLayer) computeRetreatQuality(alliedSquads []ecs.EntityID) {
	// Get allied positions (safe zones)
	allyPositions := []coords.LogicalPosition{}
	for _, squadID := range alliedSquads {
		if squads.IsSquadDestroyed(squadID, prl.manager) {
			continue
		}

		pos, err := combat.GetSquadMapPosition(squadID, prl.manager)
		if err != nil {
			continue
		}
		allyPositions = append(allyPositions, pos)
	}

	// For each position, check if there's a low-threat path to allies
	mapWidth := coords.CoordManager.GetDungeonWidth()
	mapHeight := coords.CoordManager.GetDungeonHeight()

	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			pos := coords.LogicalPosition{X: x, Y: y}

			// Check adjacent positions for low-threat retreat paths
			retreatScore := 0.0
			checkedDirs := 0

			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}

					adjacentPos := coords.LogicalPosition{X: pos.X + dx, Y: pos.Y + dy}

					meleeThreat := prl.meleeLayer.GetMeleeThreatAt(adjacentPos)
					rangedThreat := prl.rangedLayer.GetRangedPressureAt(adjacentPos)

					// Low threat path = good retreat route
					if meleeThreat < RetreatSafeThreatThreshold && rangedThreat < RetreatSafeThreatThreshold {
						retreatScore += 1.0
					}
					checkedDirs++
				}
			}

			// Retreat quality = percentage of low-threat adjacent positions
			if checkedDirs > 0 {
				prl.retreatQuality[pos] = retreatScore / float64(checkedDirs)
			}
		}
	}
}

// Query API methods

// GetFlankingRiskAt returns flanking exposure at a position (0-1)
func (prl *PositionalRiskLayer) GetFlankingRiskAt(pos coords.LogicalPosition) float64 {
	return prl.flankingRisk[pos]
}

// GetIsolationRiskAt returns isolation penalty at a position (0-1)
func (prl *PositionalRiskLayer) GetIsolationRiskAt(pos coords.LogicalPosition) float64 {
	return prl.isolationRisk[pos]
}

// GetEngagementPressureAt returns net damage exposure at a position (0-1)
func (prl *PositionalRiskLayer) GetEngagementPressureAt(pos coords.LogicalPosition) float64 {
	return prl.engagementPressure[pos]
}

// GetRetreatQuality returns escape route quality at a position (0-1, higher = better)
func (prl *PositionalRiskLayer) GetRetreatQuality(pos coords.LogicalPosition) float64 {
	return prl.retreatQuality[pos]
}

// GetTotalRiskAt returns combined positional risk
func (prl *PositionalRiskLayer) GetTotalRiskAt(pos coords.LogicalPosition) float64 {
	flanking := prl.flankingRisk[pos]
	isolation := prl.isolationRisk[pos]
	pressure := prl.engagementPressure[pos]
	retreatPenalty := 1.0 - prl.retreatQuality[pos] // Invert: low quality = high risk

	// Weighted combination
	return (flanking*0.4 + isolation*0.3 + pressure*0.2 + retreatPenalty*0.1)
}

// IsFlankingPosition checks if attacking from pos would flank target
func (prl *PositionalRiskLayer) IsFlankingPosition(pos, targetPos coords.LogicalPosition) bool {
	// Simple heuristic: flanking if attacking from side or behind relative to target's allies
	// For now, just check if position has low flanking risk (safe to attack from)
	return prl.flankingRisk[pos] < 0.3
}
