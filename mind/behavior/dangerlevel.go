package behavior

import (
	"game_main/common"
	"game_main/config"
	"game_main/mind/evaluation"
	"game_main/tactical/combat"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

var avgCombatStats = &averageCombatStats{}

// Keeps track of Each Factions Danger Level.
type FactionThreatLevelManager struct {
	manager  *common.EntityManager
	cache    *combat.CombatQueryCache
	factions map[ecs.EntityID]*FactionThreatLevel
}

func NewFactionThreatLevelManager(manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevelManager {

	calculateAverageAttributes()
	return &FactionThreatLevelManager{
		manager:  manager,
		cache:    cache,
		factions: make(map[ecs.EntityID]*FactionThreatLevel),
	}
}

func (ftlm *FactionThreatLevelManager) AddFaction(factionID ecs.EntityID) {

	if _, exists := ftlm.factions[factionID]; !exists {
		ftlm.factions[factionID] = NewFactionThreatLevel(factionID, ftlm.manager, ftlm.cache)
	}

	ftlm.factions[factionID].UpdateThreatRatings()
}

func (ftlm *FactionThreatLevelManager) UpdateFaction(factionID ecs.EntityID) {
	if faction, exists := ftlm.factions[factionID]; exists {
		faction.UpdateThreatRatings()
	}
}

func (ftlm *FactionThreatLevelManager) UpdateAllFactions() {
	for _, faction := range ftlm.factions {
		faction.UpdateThreatRatings()
	}
}

type FactionThreatLevel struct {
	manager          *common.EntityManager
	cache            *combat.CombatQueryCache
	factionID        ecs.EntityID
	squadDangerLevel map[ecs.EntityID]*SquadThreatLevel //Key is the squad ID. Value is the danger level
}

func NewFactionThreatLevel(factionID ecs.EntityID, manager *common.EntityManager, cache *combat.CombatQueryCache) *FactionThreatLevel {

	squadIDs := combat.GetSquadsForFaction(factionID, manager)

	ftl := &FactionThreatLevel{

		factionID:        factionID,
		squadDangerLevel: make(map[ecs.EntityID]*SquadThreatLevel, len(squadIDs)),
		manager:          manager,
		cache:            cache,
	}

	for _, ID := range squadIDs {
		ftl.squadDangerLevel[ID] = NewSquadThreatLevel(ftl.manager, ftl.cache, ID)
	}

	return ftl
}

func (ftr *FactionThreatLevel) UpdateThreatRatings() {

	squadIDs := combat.GetSquadsForFaction(ftr.factionID, ftr.manager)

	for _, squadID := range squadIDs {
		// Update threat calculations
		ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()

	}

}

func (ftr *FactionThreatLevel) UpdateThreatRatingForSquad(squadID ecs.EntityID) {
	ftr.squadDangerLevel[squadID].CalculateSquadDangerLevel()

}

// GetSquadDistanceTracker retrieves distance tracker for specific squad
// Returns nil if squad not found in this faction
func (ftr *FactionThreatLevel) GetSquadDistanceTracker(squadID ecs.EntityID) *SquadDistanceTracker {
	if stl, exists := ftr.squadDangerLevel[squadID]; exists {
		return stl.SquadDistances
	}
	return nil
}

// MarkAllSquadDistancesDirty marks all squad distance trackers as needing recalculation
// Call this when squad positions change (e.g., after movement phase)
func (ftr *FactionThreatLevel) MarkAllSquadDistancesDirty() {
	for _, squadThreatLevel := range ftr.squadDangerLevel {
		squadThreatLevel.SquadDistances.isDirty = true
	}
}

// UpdateSquadDistancesIfNeeded updates distances only if needed (lazy evaluation)
// Pass current round to enable round-based caching
func (ftr *FactionThreatLevel) UpdateSquadDistancesIfNeeded(squadID ecs.EntityID, currentRound int) {
	if stl, exists := ftr.squadDangerLevel[squadID]; exists {
		stl.SquadDistances.UpdateSquadDistances(ftr.manager, currentRound)
	}
}

// ========================================
// SQUAD DISTANCE TRACKING STRUCTURES
// ========================================

// SquadDistanceInfo stores distance data for a single squad
type SquadDistanceInfo struct {
	SquadID  ecs.EntityID // Target squad ID
	Distance int          // Chebyshev distance from source squad
}

// FactionSquadDistances groups all squads in a faction by distance
type FactionSquadDistances struct {
	FactionID        ecs.EntityID
	Squads           []SquadDistanceInfo
	SquadsByDistance map[int][]ecs.EntityID // Quick lookup by distance
}

// SquadDistanceTracker tracks distances from one squad to all other squads
type SquadDistanceTracker struct {
	SourceSquadID     ecs.EntityID
	ByFaction         map[ecs.EntityID]*FactionSquadDistances
	AllSquads         []SquadDistanceInfo
	AlliesByDistance  map[int][]ecs.EntityID
	EnemiesByDistance map[int][]ecs.EntityID

	// Optimization: Cache to avoid unnecessary recalculations
	lastUpdateRound int  // Last combat round when distances were calculated
	isDirty         bool // Mark as dirty when squad moves
	isInitialized   bool // Track if distances have been calculated at least once
}

type SquadThreatLevel struct {
	manager       *common.EntityManager
	cache         *combat.CombatQueryCache
	squadID       ecs.EntityID
	DangerByRange map[int]float64 //Key is the range. Value is the danger level. How dangerous the squad is at each range

	// Expected damage output per range using actual damage formulas
	ExpectedDamageByRange map[int]float64 //Key is the range. Value is expected damage

	// Distance tracking to all other squads in the game grouped by faction
	SquadDistances *SquadDistanceTracker
}

func NewSquadThreatLevel(manager *common.EntityManager, cache *combat.CombatQueryCache, squadID ecs.EntityID) *SquadThreatLevel {

	return &SquadThreatLevel{
		manager: manager,
		cache:   cache,
		squadID: squadID,
		SquadDistances: &SquadDistanceTracker{
			SourceSquadID:     squadID,
			ByFaction:         make(map[ecs.EntityID]*FactionSquadDistances),
			AllSquads:         make([]SquadDistanceInfo, 0),
			AlliesByDistance:  make(map[int][]ecs.EntityID),
			EnemiesByDistance: make(map[int][]ecs.EntityID),
			lastUpdateRound:   -1,
			isDirty:           true,  // Start as dirty so first access calculates
			isInitialized:     false, // Not initialized until first calculation
		},
	}
}


// CalculateSquadDangerLevel computes threat ratings for the squad.
// Uses shared power calculation from evaluation package for DangerByRange,
// and local damage formulas for ExpectedDamageByRange.
func (stl *SquadThreatLevel) CalculateSquadDangerLevel() {
	// Use shared power calculation for DangerByRange
	config := evaluation.GetDefaultConfig()
	stl.DangerByRange = evaluation.CalculateSquadPowerByRange(stl.squadID, stl.manager, config)

	// Collect unit combat data for expected damage calculation
	unitIDs := squads.GetUnitIDsInSquad(stl.squadID, stl.manager)
	var units []*UnitCombatData

	for _, unitID := range unitIDs {
		data := GetUnitCombatData(unitID, stl.manager)
		if data == nil {
			continue
		}
		units = append(units, data)
	}

	movementRange := squads.GetSquadMovementSpeed(stl.squadID, stl.manager)

	// Find maximum threat range for expected damage calculation
	maxThreatRange := 0
	for _, ud := range units {
		threatRange := movementRange + ud.AttackRange
		if threatRange > maxThreatRange {
			maxThreatRange = threatRange
		}
	}

	// Calculate expected damage by range using actual damage formulas
	stl.ExpectedDamageByRange = stl.calculateSquadExpectedDamageByRange(
		units,
		movementRange,
		maxThreatRange,
	)
}

// calculateExpectedDamageForUnit computes expected damage for one attacker unit.
// Uses actual damage formulas from squadcombat.go with probability weighting.
//
// Parameters:
//   - attackerAttr: Attacker's attributes
//   - defenderAttr: Reference target attributes
//
// Returns: Expected damage (float64) accounting for hit/dodge/crit probabilities
func (stl *SquadThreatLevel) calculateExpectedDamageForUnit(
	attackerAttr *common.Attributes,
	defenderAttr *common.Attributes,
) float64 {

	// 1. Base damage (same as GetPhysicalDamage)
	baseDamage := float64(attackerAttr.GetPhysicalDamage())

	// 2. Hit probability
	hitRate := float64(attackerAttr.GetHitRate()) / 100.0
	if hitRate > 1.0 {
		hitRate = 1.0
	}

	dodgeChance := float64(defenderAttr.GetDodgeChance()) / 100.0
	pHit := hitRate * (1.0 - dodgeChance)

	// 3. Expected crit multiplier
	// Expected value: P(normal) × 1.0 + P(crit) × 1.5
	critChance := float64(attackerAttr.GetCritChance()) / 100.0
	expectedCritMult := 1.0 + (critChance * config.CritDamageBonus)

	// 4. Apply resistance (subtraction, not percentage)
	resistance := float64(defenderAttr.GetPhysicalResistance())
	damageAfterResist := baseDamage - resistance
	if damageAfterResist < 1.0 {
		damageAfterResist = 1.0 // Minimum damage
	}

	// 5. Apply cover (initially 0, no cover assumed - conservative)
	coverReduction := 0.0
	coverMultiplier := 1.0 - coverReduction

	// 6. Combine all factors
	expectedDamage := damageAfterResist * pHit * expectedCritMult * coverMultiplier

	return expectedDamage
}

// calculateSquadExpectedDamageByRange computes expected damage output for the squad
// at each range. Similar to DangerByRange but uses actual damage formulas.
//
// This reuses UnitCombatData from CalculateSquadDangerLevel to avoid redundant queries.
func (stl *SquadThreatLevel) calculateSquadExpectedDamageByRange(
	units []*UnitCombatData,
	movementRange int,
	maxThreatRange int,
) map[int]float64 {

	// Calculate expected damage at each range
	expectedDamageByRange := make(map[int]float64, maxThreatRange)

	for currentRange := 1; currentRange <= maxThreatRange; currentRange++ {
		var rangeDamage float64 = 0.0

		// Sum expected damage from units that can threaten this range
		for _, ud := range units {
			effectiveThreatRange := movementRange + ud.AttackRange

			if effectiveThreatRange >= currentRange {
				// Calculate expected damage for this unit using pre-fetched attributes
				unitExpectedDamage := stl.calculateExpectedDamageForUnit(ud.Attributes, &avgCombatStats.avgAttributes)

				// Apply leader bonus (same as DangerByRange)
				if ud.IsLeader {
					unitExpectedDamage *= evaluation.LeaderBonus
				}

				rangeDamage += unitExpectedDamage
			}
		}

		expectedDamageByRange[currentRange] = rangeDamage
	}

	// Note: We do NOT apply composition bonus to damage (only to threat)
	// Damage is raw output; composition affects tactical value

	return expectedDamageByRange
}

// ========================================
// SQUAD DISTANCE TRACKING SYSTEM FUNCTIONS
// ========================================

// UpdateSquadDistances calculates distances from source squad to all other squads
// Organizes results by faction for easy querying

func (tracker *SquadDistanceTracker) UpdateSquadDistances(manager *common.EntityManager, currentRound int) {
	//Check if we need to recalculate
	if tracker.isInitialized && !tracker.isDirty && tracker.lastUpdateRound == currentRound {
		return // Data is still valid, skip recalculation
	}

	// Clear existing data
	tracker.ByFaction = make(map[ecs.EntityID]*FactionSquadDistances)

	// Get source squad's faction for ally/enemy classification
	sourceFactionID := combat.GetSquadFaction(tracker.SourceSquadID, manager)

	// Iterate all factions in the game
	allFactionIDs := combat.GetAllFactions(manager)

	for _, factionID := range allFactionIDs {
		// Get all squads in this faction
		squadIDs := combat.GetSquadsForFaction(factionID, manager)

		// Initialize faction distance data
		factionDistances := &FactionSquadDistances{
			FactionID:        factionID,
			Squads:           make([]SquadDistanceInfo, 0),
			SquadsByDistance: make(map[int][]ecs.EntityID),
		}

		// Calculate distance to each squad in this faction
		for _, targetSquadID := range squadIDs {
			// Skip self
			if targetSquadID == tracker.SourceSquadID {
				continue
			}

			// Calculate distance
			distance := squads.GetSquadDistance(tracker.SourceSquadID, targetSquadID, manager)

			if distance < 0 {
				continue // Skip invalid distances
			}

			squadInfo := SquadDistanceInfo{
				SquadID:  targetSquadID,
				Distance: distance,
			}

			factionDistances.Squads = append(factionDistances.Squads, squadInfo)
			factionDistances.SquadsByDistance[distance] = append(
				factionDistances.SquadsByDistance[distance],
				targetSquadID,
			)

		}

		tracker.ByFaction[factionID] = factionDistances
	}

	// Build convenience caches
	tracker.buildDistanceCaches(sourceFactionID)

	// OPTIMIZATION: Mark as clean and initialized
	tracker.isDirty = false
	tracker.isInitialized = true
	tracker.lastUpdateRound = currentRound
}

// buildDistanceCaches rebuilds the cached lookup maps in SquadDistanceTracker
// Called internally after UpdateSquadDistances completes
func (tracker *SquadDistanceTracker) buildDistanceCaches(sourceFactionID ecs.EntityID) {
	// Clear caches
	tracker.AllSquads = make([]SquadDistanceInfo, 0)
	tracker.AlliesByDistance = make(map[int][]ecs.EntityID)
	tracker.EnemiesByDistance = make(map[int][]ecs.EntityID)

	// Iterate all factions
	for factionID, factionDistances := range tracker.ByFaction {
		isAlly := (factionID == sourceFactionID)

		// Add all squads to AllSquads
		tracker.AllSquads = append(tracker.AllSquads, factionDistances.Squads...)

		// Categorize by ally/enemy
		for _, squadInfo := range factionDistances.Squads {
			if isAlly {
				tracker.AlliesByDistance[squadInfo.Distance] = append(
					tracker.AlliesByDistance[squadInfo.Distance],
					squadInfo.SquadID,
				)
			} else {
				tracker.EnemiesByDistance[squadInfo.Distance] = append(
					tracker.EnemiesByDistance[squadInfo.Distance],
					squadInfo.SquadID,
				)
			}
		}
	}
}

// Used for predicting damage for danger level calculations
// Average attributes of all creatures

type averageCombatStats struct {
	avgAttributes     common.Attributes
	avgPhyslDamage    int
	avgPhysResistance int
	avgMaxHealth      int
	avgHitRate        int
	avgCritChance     int
	avgDodgeChance    int
	avgMagicDamage    int
	avgMMagicDefense  int
	avgHealingAmount  int
}

func calculateAverageAttributes() {

	num_units := len(squads.Units)

	// Avoid divide by zero when no units exist (e.g., in tests)
	if num_units == 0 {
		return
	}

	for _, unit := range squads.Units {

		avgCombatStats.avgAttributes.Strength += unit.Attributes.Strength
		avgCombatStats.avgAttributes.Dexterity += unit.Attributes.Dexterity
		avgCombatStats.avgAttributes.Magic += unit.Attributes.Magic
		avgCombatStats.avgAttributes.Leadership += unit.Attributes.Leadership
		avgCombatStats.avgAttributes.Armor += unit.Attributes.Armor
		avgCombatStats.avgAttributes.Weapon += unit.Attributes.Weapon
		avgCombatStats.avgAttributes.MovementSpeed += unit.Attributes.MovementSpeed
		avgCombatStats.avgAttributes.AttackRange += unit.Attributes.AttackRange
		avgCombatStats.avgAttributes.CurrentHealth += unit.Attributes.CurrentHealth
		avgCombatStats.avgMaxHealth += unit.Attributes.GetMaxHealth()
		avgCombatStats.avgHitRate += unit.Attributes.GetHitRate()
		avgCombatStats.avgCritChance += unit.Attributes.GetCritChance()
		avgCombatStats.avgDodgeChance += unit.Attributes.GetDodgeChance()
		avgCombatStats.avgMagicDamage += unit.Attributes.GetMagicDamage()
		avgCombatStats.avgMMagicDefense += unit.Attributes.GetMagicDefense()
		avgCombatStats.avgHealingAmount += unit.Attributes.GetHealingAmount()

	}

	avgCombatStats.avgAttributes.Strength /= num_units
	avgCombatStats.avgAttributes.Dexterity /= num_units
	avgCombatStats.avgAttributes.Magic /= num_units
	avgCombatStats.avgAttributes.Leadership /= num_units
	avgCombatStats.avgAttributes.Armor /= num_units
	avgCombatStats.avgAttributes.Weapon /= num_units
	avgCombatStats.avgAttributes.MovementSpeed /= num_units
	avgCombatStats.avgAttributes.AttackRange /= num_units
	avgCombatStats.avgAttributes.CurrentHealth /= num_units
	avgCombatStats.avgMaxHealth /= num_units
	avgCombatStats.avgHitRate /= num_units
	avgCombatStats.avgCritChance /= num_units
	avgCombatStats.avgDodgeChance /= num_units
	avgCombatStats.avgMagicDamage /= num_units
	avgCombatStats.avgMMagicDefense /= num_units
	avgCombatStats.avgHealingAmount /= num_units

}
