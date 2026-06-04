package behavior

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"
	"game_main/core/coords"
	"math"

	"github.com/bytearena/ecs"
)

// PositionalRiskLayer computes tactical risk based on positioning
// Identifies flanking exposure, isolation, and retreat path quality
type PositionalRiskLayer struct {
	*ThreatLayerBase

	// Core risk data. These maps are sparse: only positions whose value differs from the
	// dimension's default are stored. The Get*At accessors below supply the default for
	// missing keys, so consumers must read through them rather than indexing the maps.
	flankingRisk       map[coords.LogicalPosition]float64 // Position -> flank exposure (default 0)
	isolationRisk      map[coords.LogicalPosition]float64 // Position -> isolation penalty (default 1.0 when allies exist)
	engagementPressure map[coords.LogicalPosition]float64 // Position -> net damage exposure (default 0)
	retreatQuality     map[coords.LogicalPosition]float64 // Position -> escape route quality (default 1.0)

	// hasAllies records whether the faction had any positioned allies on the last Compute.
	// When false, isolation risk is undefined and GetIsolationRiskAt returns 0.
	hasAllies bool

	// Dependencies
	baseThreatMgr *FactionThreatLevelManager
	combatLayer   *CombatThreatLayer
}

// NewPositionalRiskLayer creates a new positional risk layer
func NewPositionalRiskLayer(
	factionID ecs.EntityID,
	manager *common.EntityManager,
	cache *combatstate.CombatQueryCache,
	baseThreatMgr *FactionThreatLevelManager,
	combatLayer *CombatThreatLayer,
) *PositionalRiskLayer {
	return &PositionalRiskLayer{
		ThreatLayerBase:    NewThreatLayerBase(factionID, manager, cache),
		flankingRisk:       make(map[coords.LogicalPosition]float64),
		isolationRisk:      make(map[coords.LogicalPosition]float64),
		engagementPressure: make(map[coords.LogicalPosition]float64),
		retreatQuality:     make(map[coords.LogicalPosition]float64),
		baseThreatMgr:      baseThreatMgr,
		combatLayer:        combatLayer,
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
	alliedSquads := combatstate.GetActiveSquadsForFaction(prl.factionID, prl.manager)
	enemyFactions := prl.getEnemyFactions()

	// Compute flanking risk based on enemy positions
	prl.computeFlankingRisk(enemyFactions)

	// Compute isolation risk based on ally positions
	prl.computeIsolationRisk(alliedSquads)

	// Compute engagement pressure using threat layers
	prl.computeEngagementPressure()

	// Compute retreat path quality
	prl.computeRetreatQuality()
}

// computeFlankingRisk identifies positions that can be attacked from multiple directions
func (prl *PositionalRiskLayer) computeFlankingRisk(enemyFactions []ecs.EntityID) {
	// For each position, count enemy threat directions
	threatDirections := make(map[coords.LogicalPosition]map[int]bool) // pos -> set of attack angles

	for _, enemyFactionID := range enemyFactions {
		squadIDs := combatstate.GetActiveSquadsForFaction(enemyFactionID, prl.manager)

		for _, squadID := range squadIDs {
			enemyPos, err := combatstate.GetSquadMapPosition(squadID, prl.manager)
			if err != nil {
				continue
			}

			// Get threat range for this squad
			moveSpeed := squadcore.GetSquadMovementSpeed(squadID, prl.manager)
			threatRange := moveSpeed + GetFlankingThreatRangeBonus()

			// Paint threat directions
			for dx := -threatRange; dx <= threatRange; dx++ {
				for dy := -threatRange; dy <= threatRange; dy++ {
					pos := coords.LogicalPosition{X: enemyPos.X + dx, Y: enemyPos.Y + dy}
					distance := enemyPos.ChebyshevDistance(&pos)

					if distance > 0 && distance <= threatRange {
						// Calculate attack angle (simplified to 8 directions)
						angle := getDirection(dx, dy)

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

// directionTable maps (sign(dx)+1, sign(dy)+1) to one of 8 compass directions
// (N=0, NE=1, E=2, SE=3, S=4, SW=5, W=6, NW=7). The center cell [1][1] (dx==dy==0)
// is never read — callers only classify offsets at distance > 0.
var directionTable = [3][3]int{
	{5, 6, 7},  // dx<0:  dy<0 SW, dy==0 W,  dy>0 NW
	{4, -1, 0}, // dx==0: dy<0 S,  center,  dy>0 N
	{3, 2, 1},  // dx>0:  dy<0 SE, dy==0 E,  dy>0 NE
}

// getDirection returns simplified direction (0-7) based on dx, dy.
func getDirection(dx, dy int) int {
	return directionTable[sign(dx)+1][sign(dy)+1]
}

// sign returns -1, 0, or 1 for negative, zero, or positive n.
func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}

// computeIsolationRisk identifies positions far from allied support
func (prl *PositionalRiskLayer) computeIsolationRisk(alliedSquads []ecs.EntityID) {
	// Get ally positions
	allyPositions := []coords.LogicalPosition{}
	for _, squadID := range alliedSquads {
		pos, err := combatstate.GetSquadMapPosition(squadID, prl.manager)
		if err != nil {
			continue
		}
		allyPositions = append(allyPositions, pos)
	}

	prl.hasAllies = len(allyPositions) > 0
	if !prl.hasAllies {
		return
	}

	threshold := GetIsolationThreshold()
	maxDist := GetIsolationMaxDistance()

	// Only positions within maxDist of an ally can fall below the 1.0 isolation default
	// (see GetIsolationRiskAt). Walk the Chebyshev box around each ally instead of the whole
	// grid, deduping shared positions into a candidate set.
	candidates := make(map[coords.LogicalPosition]struct{})
	for _, allyPos := range allyPositions {
		for dx := -maxDist; dx <= maxDist; dx++ {
			for dy := -maxDist; dy <= maxDist; dy++ {
				candidates[coords.LogicalPosition{X: allyPos.X + dx, Y: allyPos.Y + dy}] = struct{}{}
			}
		}
	}

	for pos := range candidates {
		minDistance := math.MaxInt32
		for _, allyPos := range allyPositions {
			distance := pos.ChebyshevDistance(&allyPos)
			if distance < minDistance {
				minDistance = distance
			}
		}

		// Isolation risk grows linearly with distance from the nearest ally.
		// >= maxDist defaults to 1.0 (skip). Within the threshold we store 0 explicitly,
		// because a missing key now reads as "fully isolated".
		switch {
		case minDistance >= maxDist:
			// default 1.0 via GetIsolationRiskAt
		case minDistance > threshold:
			prl.isolationRisk[pos] = float64(minDistance-threshold) / float64(maxDist-threshold)
		default:
			prl.isolationRisk[pos] = 0
		}
	}
}

// computeEngagementPressure combines melee and ranged threat for net pressure
func (prl *PositionalRiskLayer) computeEngagementPressure() {
	maxPressure := float64(GetEngagementPressureMax())

	// Engagement pressure is non-zero only where the combat layer painted threat. Iterate
	// those positions instead of the whole grid; everything else defaults to 0 via
	// GetEngagementPressureAt, which equals math.Min(0/max, 1).
	set := func(pos coords.LogicalPosition) {
		if _, done := prl.engagementPressure[pos]; done {
			return
		}
		totalPressure := prl.combatLayer.GetMeleeThreatAt(pos) + prl.combatLayer.GetRangedPressureAt(pos)
		prl.engagementPressure[pos] = math.Min(totalPressure/maxPressure, 1.0)
	}
	for pos := range prl.combatLayer.meleeThreatByPos {
		set(pos)
	}
	for pos := range prl.combatLayer.rangedPressureByPos {
		set(pos)
	}
}

// computeRetreatQuality evaluates escape route quality
func (prl *PositionalRiskLayer) computeRetreatQuality() {
	retreatThreshold := float64(GetRetreatSafeThreatThreshold())

	// A position's retreat quality is below the 1.0 default (GetRetreatQuality) only when at
	// least one of its 8 neighbours is "unsafe" (threat >= retreatThreshold). Collect the
	// neighbours of every unsafe tile, then score only those candidates.
	candidates := make(map[coords.LogicalPosition]struct{})
	collect := func(threatMap map[coords.LogicalPosition]float64) {
		for tp, v := range threatMap {
			if v < retreatThreshold {
				continue
			}
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					candidates[coords.LogicalPosition{X: tp.X + dx, Y: tp.Y + dy}] = struct{}{}
				}
			}
		}
	}
	collect(prl.combatLayer.meleeThreatByPos)
	collect(prl.combatLayer.rangedPressureByPos)

	for pos := range candidates {
		// Count adjacent positions that are low-threat (good retreat routes).
		retreatScore := 0.0
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				adjacentPos := coords.LogicalPosition{X: pos.X + dx, Y: pos.Y + dy}
				meleeThreat := prl.combatLayer.GetMeleeThreatAt(adjacentPos)
				rangedThreat := prl.combatLayer.GetRangedPressureAt(adjacentPos)
				if meleeThreat < retreatThreshold && rangedThreat < retreatThreshold {
					retreatScore += 1.0
				}
			}
		}

		// 8 directions are always checked (out-of-bounds neighbours read as 0 = safe).
		prl.retreatQuality[pos] = retreatScore / 8.0
	}
}

// Query API methods

// GetFlankingRiskAt returns flanking exposure at a position (0-1)
func (prl *PositionalRiskLayer) GetFlankingRiskAt(pos coords.LogicalPosition) float64 {
	return prl.flankingRisk[pos]
}

// GetIsolationRiskAt returns isolation penalty at a position (0-1).
// With no positioned allies isolation is undefined and reads 0; otherwise a position not
// stored by the sparse computation is beyond maxDist of every ally, i.e. fully isolated (1.0).
func (prl *PositionalRiskLayer) GetIsolationRiskAt(pos coords.LogicalPosition) float64 {
	if !prl.hasAllies {
		return 0
	}
	if v, ok := prl.isolationRisk[pos]; ok {
		return v
	}
	return 1.0
}

// GetEngagementPressureAt returns net damage exposure at a position (0-1).
// Positions without painted threat default to 0.
func (prl *PositionalRiskLayer) GetEngagementPressureAt(pos coords.LogicalPosition) float64 {
	return prl.engagementPressure[pos]
}

// GetRetreatQuality returns escape route quality at a position (0-1, higher = better).
// Positions not adjacent to any threat have perfect retreat (1.0), the default for missing keys.
func (prl *PositionalRiskLayer) GetRetreatQuality(pos coords.LogicalPosition) float64 {
	if v, ok := prl.retreatQuality[pos]; ok {
		return v
	}
	return 1.0
}

// GetTotalRiskAt returns combined positional risk as a weighted, normalized blend of the four
// risk dimensions. Weights come from aiconfig.json (positionalRiskWeights); the default equal
// weights reproduce a simple average. Reads go through the Get*At accessors so sparse-storage
// defaults are applied consistently.
func (prl *PositionalRiskLayer) GetTotalRiskAt(pos coords.LogicalPosition) float64 {
	w := getPositionalRiskWeights()
	totalWeight := w.Flanking + w.Isolation + w.EngagementPressure + w.Retreat
	if totalWeight == 0 {
		return 0
	}
	flanking := prl.GetFlankingRiskAt(pos)
	isolation := prl.GetIsolationRiskAt(pos)
	pressure := prl.GetEngagementPressureAt(pos)
	retreatPenalty := 1.0 - prl.GetRetreatQuality(pos)
	return (flanking*w.Flanking + isolation*w.Isolation + pressure*w.EngagementPressure + retreatPenalty*w.Retreat) / totalWeight
}
