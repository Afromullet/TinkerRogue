# Overworld Package Refactoring Recommendations

**Generated:** 2026-01-26
**Package:** `world/overworld/`
**Status:** Production-ready (improvements for maintainability)

---

## Executive Summary

The overworld package demonstrates **excellent ECS compliance** and solid architecture. These recommendations focus on **maintainability, testability, and extensibility** rather than critical fixes. The package is production-ready as-is.

### Overall Assessment
- ✅ **ECS Compliance:** Excellent (EntityID-only, value maps, query patterns)
- ✅ **Test Coverage:** Strong for resources/victory (17-18 tests each)
- ⚠️ **File Organization:** Needs improvement (monolithic components.go)
- ⚠️ **Test Gaps:** Missing threat, tick, encounter tests
- ⚠️ **Configuration:** Hardcoded parameters should be data-driven

---

## Priority 1: File Organization (1-2 hours)

### Issue: Monolithic components.go
**Current:** All 10+ components in one 300-line file
**Problem:** Hard to navigate, find related data structures

**Recommendation:** Split by subsystem feature

```
world/overworld/
├── components_core.go       # GridCell, TickData (20 lines)
├── components_faction.go    # FactionData, FactionType (80 lines)
├── components_threat.go     # ThreatData, ThreatType, ThreatParams (60 lines)
├── components_resources.go  # ResourceData, ResourceModifier (40 lines)
├── components_influence.go  # InfluenceData (30 lines)
├── components_events.go     # EventData, EventType (40 lines)
└── components_victory.go    # VictoryData, VictoryCondition (30 lines)
```

**Benefits:**
- Easier navigation for new developers
- Clear subsystem boundaries
- Smaller, focused files (project standard)

---

### Issue: Vague utils.go
**Current:** `DetermineVictoryOutcome`, `GetFactionName`, `GetThreatName` in one file
**Problem:** "Utils" is a catch-all with no clear responsibility

**Recommendation:** Move functions to proper homes

```go
// Move to victory_queries.go (already has victory checks)
func DetermineVictoryOutcome(manager *common.EntityManager) VictoryStatus

// Move to faction_queries.go (already has faction lookups)
func GetFactionName(faction FactionType) string

// Move to threat_queries.go (already has threat lookups)
func GetThreatName(threat ThreatType) string
```

**Result:** Delete `utils.go`, all functions have logical homes

---

### Issue: Mixed Constants
**Current:** `constants.go` contains enums, tuning params, and dimensions

**Recommendation:** Split by purpose

```go
// constants.go - Only true constants and enums
const (
    PlayerFaction FactionType = iota
    BanditFaction
    MonsterFaction
)

// config.go - Tuning parameters (candidates for future config files)
const (
    DefaultThreatGrowthRate = 1.5
    MinResourcesPerTile     = 1
    MaxResourcesPerTile     = 5
)

// grid_config.go - Map dimensions (future: load from level data)
const (
    DefaultGridWidth  = 100
    DefaultGridHeight = 80
)
```

---

## Priority 1: Separation of Concerns (1-2 hours)

### Issue: faction_system.go Mixing Concerns
**Current:** System logic + scoring utilities in one file

**Recommendation:** Extract `faction_scoring.go`

```go
// faction_scoring.go - Pure scoring functions
func CalculateInfluenceScore(factionID ecs.EntityID, ...) int
func CalculateResourceScore(factionID ecs.EntityID, ...) int
func CalculateThreatScore(factionID ecs.EntityID, ...) int
func CalculateTotalScore(factionID ecs.EntityID, ...) int

// faction_system.go - Only system orchestration
func UpdateFactionScores(manager *common.EntityManager)
func ProcessFactionVictory(manager *common.EntityManager)
```

**Benefits:**
- Easier testing of scoring logic
- Clear separation: calculation vs orchestration
- Reusable scoring functions

---

### Issue: victory.go Mixing Systems/Queries/Interfaces
**Current:** Interface, system functions, and queries in one file

**Recommendation:** Split responsibilities

```go
// victory_conditions.go - Interface definitions
type VictoryCondition interface {
    Check(*common.EntityManager) (bool, string)
}

// victory_system.go - System logic
func CheckVictoryConditions(manager *common.EntityManager, ...) bool
func TriggerVictory(manager *common.EntityManager, ...)

// victory_queries.go - Query functions
func GetVictoryData(manager *common.EntityManager) *VictoryData
func HasVictoryOccurred(manager *common.EntityManager) bool
func DetermineVictoryOutcome(manager *common.EntityManager) VictoryStatus
```

---

## Priority 1: Missing Test Coverage (2-3 hours)

### Gap 1: threat_test.go (Missing)
**Current:** No dedicated tests for threat system

**Recommendation:** Create comprehensive test suite

```go
// threat_test.go
func TestCreateThreat(t *testing.T)              // Creation with params
func TestThreatGrowth(t *testing.T)              // Growth rate application
func TestThreatProximity(t *testing.T)           // Distance calculations
func TestGetThreatTypeParams(t *testing.T)       // Parameter lookups
func TestGetBaseThreatUnits(t *testing.T)        // Unit generation
func TestProcessThreatsAtPosition(t *testing.T)  // Spatial queries
```

**Priority:** High - threats are core gameplay mechanic

---

### Gap 2: tick_system_test.go (Missing)
**Current:** Tick system untested despite orchestrating all systems

**Recommendation:** Integration-style tests

```go
// tick_system_test.go
func TestTickSystemInitialization(t *testing.T)  // Setup
func TestFullGameTick(t *testing.T)              // Complete cycle
func TestTickOrder(t *testing.T)                 // System execution order
func TestTickCounter(t *testing.T)               // Counter increments
```

**Priority:** High - tick orchestrates everything

---

### Gap 3: encounter_translation_test.go (Missing)
**Current:** Translation logic untested

**Recommendation:** Test combat encounter generation

```go
// encounter_translation_test.go
func TestTranslateThreatToEncounter(t *testing.T)  // Threat → Combat
func TestEmptyThreatTranslation(t *testing.T)      // Edge: no threats
func TestMultipleThreatTranslation(t *testing.T)   // Multiple threats
func TestUnitPlacement(t *testing.T)               // Position assignment
```

**Priority:** Medium - bridges overworld/combat

---

### Gap 4: Integration Tests
**Current:** Only unit tests, no full game loop tests

**Recommendation:** Add `integration_test.go`

```go
// integration_test.go
func TestFullGameLoop(t *testing.T) {
    // Create world → Run 100 ticks → Verify victory/defeat
}

func TestFactionVictoryScenario(t *testing.T) {
    // Setup dominant faction → Tick until victory
}

func TestResourceDepletionScenario(t *testing.T) {
    // Zero resources → Tick → Verify defeat
}
```

**Priority:** Medium - validates subsystem interactions

---

## Priority 2: Logging and Debugging (30 min)

### Issue: fmt.Printf Scattered Throughout
**Current:** Debug prints in `tick_system.go`, `faction_system.go`, `events.go`

**Recommendation:** Use `overworldlog` package consistently

```go
// Before
fmt.Printf("Tick %d: Victory achieved\n", tick)

// After
log := overworldlog.GetRecorder()
log.RecordVictory(tick, winningFaction)
```

**Benefits:**
- Centralized logging control
- Structured event tracking
- Easier debugging/analytics

---

## Priority 2: Package Documentation (30 min)

### Issue: Missing doc.go
**Current:** No package-level documentation

**Recommendation:** Add `doc.go`

```go
// Package overworld implements the strategic layer of TinkerRogue.
//
// The overworld simulates faction territories, resource management, threat
// propagation, and victory conditions. It uses a tick-based system to update
// the world state and generate tactical encounters for the player.
//
// # Core Subsystems
//
// Factions: Territory control and influence spreading
// Threats: Enemy units that grow and spread over time
// Resources: Consumable tiles that factions compete for
// Events: Random occurrences that affect the world state
// Victory: Win/loss condition checking
//
// # Architecture
//
// The package follows strict ECS principles:
// - components_*.go: Pure data structures
// - *_queries.go: Read-only entity searches
// - *_system.go: Logic and behavior
// - tick_system.go: Orchestrates all subsystems
//
// # Usage
//
// See example_usage_test.go for initialization and tick loop patterns.
package overworld
```

---

## Priority 3: Configuration Hardcoding (3-4 hours)

### Issue: GetThreatTypeParams Switch Statement
**Current:** Hardcoded threat parameters in code

**Recommendation:** Data-driven threat config

```yaml
# config/threats.yaml
threats:
  bandits:
    base_strength: 3
    growth_rate: 1.2
    spread_range: 5
    default_units:
      - { type: bandit_archer, count: 2 }
      - { type: bandit_warrior, count: 1 }

  monsters:
    base_strength: 5
    growth_rate: 1.5
    spread_range: 3
    default_units:
      - { type: goblin, count: 3 }
```

```go
// threat_config.go
type ThreatConfig struct {
    BaseStrength int
    GrowthRate   float64
    SpreadRange  int
    DefaultUnits []UnitSpawn
}

func LoadThreatConfig(path string) (map[ThreatType]ThreatConfig, error)

// threat_system.go
var threatConfigs map[ThreatType]ThreatConfig

func init() {
    var err error
    threatConfigs, err = LoadThreatConfig("config/threats.yaml")
    if err != nil {
        log.Fatalf("Failed to load threat config: %v", err)
    }
}

func GetThreatTypeParams(threatType ThreatType) ThreatParams {
    config := threatConfigs[threatType]
    return ThreatParams{
        BaseStrength: config.BaseStrength,
        GrowthRate:   config.GrowthRate,
        SpreadRange:  config.SpreadRange,
    }
}
```

**Benefits:**
- Easy tuning without recompilation
- Modding support
- A/B testing different balance parameters

---

### Issue: GetBaseThreatUnits Hardcoded
**Current:** Unit definitions in code

**Recommendation:** Load from threat config (see above YAML)

```go
func GetBaseThreatUnits(threatType ThreatType) []squads.UnitType {
    config := threatConfigs[threatType]
    units := make([]squads.UnitType, 0, len(config.DefaultUnits))
    for _, spawn := range config.DefaultUnits {
        for i := 0; i < spawn.Count; i++ {
            units = append(units, spawn.Type)
        }
    }
    return units
}
```

---

## Priority 3: Magic Numbers (15 min)

### Issue: Hardcoded Map Dimensions
**Current:** `100x80` scattered in tests and systems

**Recommendation:** Use constants from config

```go
// grid_config.go (already created in Priority 1)
const (
    DefaultGridWidth  = 100
    DefaultGridHeight = 80
)

// Usage in tests
func TestResourceCreation(t *testing.T) {
    width := DefaultGridWidth
    height := DefaultGridHeight
    grid := make([]*GridCell, width*height)
    // ...
}
```

---

## Priority 4: Performance Optimizations (2-3 hours)

### Issue: GetFactionTerritoryMap Creates New Map Every Call
**Current:** `O(n)` every call in `faction_queries.go`

**Recommendation:** Cache territory maps in FactionData

```go
// components_faction.go
type FactionData struct {
    // ... existing fields
    TerritoryCache map[coords.LogicalPosition]bool  // NEW
    CacheDirty     bool                              // NEW
}

// faction_system.go - Invalidate on influence changes
func UpdateFactionInfluence(factionID ecs.EntityID, ...) {
    // ... update influence
    factionData.CacheDirty = true
}

// faction_queries.go - Use cache
func GetFactionTerritoryMap(factionID ecs.EntityID, ...) map[coords.LogicalPosition]bool {
    factionData := GetFactionData(factionID, manager)
    if factionData.CacheDirty {
        factionData.TerritoryCache = buildTerritoryMap(factionID, manager)
        factionData.CacheDirty = false
    }
    return factionData.TerritoryCache
}
```

**Expected Improvement:** 10-100x faster for repeated queries

---

### Recommendation: Add Benchmarks
**Current:** No performance tests

```go
// faction_bench_test.go
func BenchmarkGetFactionTerritoryMap(b *testing.B) {
    manager := setupLargeWorld(1000, 800)  // 800K cells
    factionID := createTestFaction(manager)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        GetFactionTerritoryMap(factionID, manager, influenceCache)
    }
}

// Target: <1ms for 100x100 grid
```

---

## Implementation Plan

### Phase 1: File Organization (1-2 hours)
**Risk:** Low (no logic changes)
1. Split `components.go` → `components_*.go` (7 files)
2. Move functions from `utils.go` to proper homes
3. Split `constants.go` → `constants.go`, `config.go`, `grid_config.go`
4. Update imports in all files
5. Run `go test ./world/overworld/...` to verify

### Phase 2: Test Coverage (2-3 hours)
**Risk:** Low (additive only)
1. Create `threat_test.go` (6 tests)
2. Create `tick_system_test.go` (4 tests)
3. Create `encounter_translation_test.go` (4 tests)
4. Create `integration_test.go` (3 tests)
5. Verify all tests pass and coverage improves

### Phase 3: Quality Improvements (1-2 hours)
**Risk:** Low (mostly additive)
1. Replace `fmt.Printf` with `overworldlog` calls
2. Add `doc.go` package documentation
3. Extract `faction_scoring.go` from `faction_system.go`
4. Split `victory.go` → 3 files
5. Fix magic numbers with constants

### Phase 4: Data-Driven Config (3-4 hours)
**Risk:** Medium (changes initialization flow)
1. Create `config/threats.yaml`
2. Implement `threat_config.go` loader
3. Refactor `GetThreatTypeParams` to use config
4. Refactor `GetBaseThreatUnits` to use config
5. Add tests for config loading
6. Document config file format

### Phase 5: Performance (2-3 hours)
**Risk:** Medium (cache invalidation bugs possible)
1. Add territory cache to `FactionData`
2. Implement cache invalidation in influence updates
3. Update `GetFactionTerritoryMap` to use cache
4. Add benchmark tests
5. Verify no functional changes (run full test suite)

---

## Risks and Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Import path updates break build | Low | Run tests after each file split |
| Cache invalidation bugs | Medium | Extensive testing, compare cached vs uncached results |
| Config loading failures | Medium | Fallback to hardcoded defaults, validate config on load |
| Performance regression | Low | Add benchmarks before changes, compare results |

---

## Success Metrics

**Before:**
- 1 monolithic components file (300 lines)
- 3 missing test files
- Hardcoded threat parameters
- `O(n)` territory map queries per call

**After:**
- 7 focused component files (<100 lines each)
- 100% test coverage for core systems
- Data-driven threat configuration
- Cached territory maps with benchmarks

---

## Recommendations Summary

### Immediate (Priority 1) - 4-6 hours
1. ✅ Split components.go by subsystem (1-2 hours)
2. ✅ Dissolve utils.go into proper homes (30 min)
3. ✅ Add missing test files (2-3 hours)
4. ✅ Extract scoring/victory file splits (1 hour)

### Short-term (Priority 2) - 1 hour
5. ✅ Standardize logging with overworldlog (30 min)
6. ✅ Add doc.go package documentation (30 min)

### Medium-term (Priority 3) - 4-5 hours
7. ⚠️ Data-driven threat configuration (3-4 hours)
8. ⚠️ Fix magic number constants (15 min)

### Future (Priority 4) - 2-3 hours
9. 🔄 Cache faction territory maps (1-2 hours)
10. 🔄 Add benchmark suite (1 hour)

**Total Estimated Time:** 9-14 hours

---

## Notes

- Package is **production-ready** - these are quality improvements, not bug fixes
- Start with low-risk file organization to build confidence
- Test coverage gaps are the most important functional improvement
- Configuration system enables easier game balancing and modding
- Performance optimizations can wait until profiling shows bottlenecks

**Analysis completed by:** refactoring-pro agent (agentId: ab4fa38)
**Questions?** Review specific files or start with Phase 1 implementation.
