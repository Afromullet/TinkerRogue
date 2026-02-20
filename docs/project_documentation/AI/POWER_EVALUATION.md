# Power Evaluation System

**Last Updated:** 2026-02-20

Technical reference for TinkerRogue's unified combat power assessment, used by both AI threat evaluation and encounter generation.

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Overview, system diagram, performance considerations
- [AI Controller](AI_CONTROLLER.md) - AI turn orchestration and action scoring
- [AI Configuration](AI_CONFIGURATION.md) - Config files, accessor patterns, tuning guide
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems consuming power data
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation using power budgets

---

## Table of Contents

1. [Architecture](#architecture)
2. [Power Configuration](#power-configuration)
3. [Unit Power Calculation](#unit-power-calculation)
4. [Squad Power Calculation](#squad-power-calculation)
5. [Squad Power By Range](#squad-power-by-range)
6. [Role Multipliers and Ability Values](#role-multipliers-and-ability-values)
7. [DirtyCache](#dirtycache)
8. [Performance Considerations](#performance-considerations)
9. [Extension Points](#extension-points)
10. [File Reference](#file-reference)

---

## Architecture

**Location:** `mind/evaluation/`

**Purpose:** Unified combat power assessment shared by AI threat evaluation and encounter generation.

```
Power Calculation (Shared)
|
+- Unit Power
|  +- Offensive Power (damage output)
|  +- Defensive Power (survivability)
|  +- Utility Power (role, abilities, cover)
|
+- Squad Power
|  +- Sum unit powers
|  +- Composition bonus (attack type diversity)
|  +- Health multiplier (wounded penalty)
|
+- Squad Power By Range
   +- Map of range -> power (for threat assessment)
```

**Consumers:**
- **AI Threat System** - `FactionThreatLevelManager` calls `CalculateSquadPowerByRange()` to build threat-by-range maps
- **Encounter Generation** - `GenerateEncounterSpec()` calls `CalculateSquadPower()` to assess player strength and `EstimateUnitPowerFromTemplate()` to build enemy squads within power budgets

---

## Power Configuration

**Location:** `mind/evaluation/power_config.go`

**Data Structure:**

```go
type PowerConfig struct {
  OffensiveWeight float64  // Weight for damage output (0.0-1.0)
  DefensiveWeight float64  // Weight for survivability (0.0-1.0)
  UtilityWeight   float64  // Weight for utility (0.0-1.0)
  HealthPenalty   float64  // Exponent for health scaling (e.g., 2.0)
}
```

**Configuration (powerconfig.json):**

```json
{
  "profiles": [
    {
      "name": "Balanced",
      "offensiveWeight": 0.4,
      "defensiveWeight": 0.4,
      "utilityWeight": 0.2,
      "healthPenalty": 2.0
    }
  ]
}
```

**Accessor Pattern:**

```go
GetPowerConfigByProfile("Balanced"):
  // Try to find in loaded config
  for each profile in templates.PowerConfigTemplate.Profiles:
    if profile.Name == profileName:
      return PowerConfig from profile

  // Fallback to defaults (DefaultOffensiveWeight=0.4, DefaultDefensiveWeight=0.4, DefaultUtilityWeight=0.2)
  return default Balanced config
```

For the full powerconfig.json structure and parameter tables, see [AI Configuration - Power Configuration](AI_CONFIGURATION.md#power-configuration-powerconfigjson).

---

## Unit Power Calculation

**Function:** `calculateUnitPower()` in `mind/evaluation/power.go`

**Algorithm:**

```go
calculateUnitPower(unitID, manager, config):
  // Get components
  attr = GetAttributes(unitID)
  roleData = GetUnitRole(unitID)

  // Calculate category powers
  offensivePower = CalculateOffensivePower(attr, config)
  defensivePower = CalculateDefensivePower(attr, config)
  utilityPower = CalculateUtilityPower(entity, attr, roleData, config)

  // Weighted sum
  totalPower = (offensivePower * config.OffensiveWeight) +
               (defensivePower * config.DefensiveWeight) +
               (utilityPower * config.UtilityWeight)

  return totalPower
```

**Offensive Power:**

```go
CalculateOffensivePower(attr, config):
  avgDamage = (PhysicalDamage + MagicDamage) / 2.0
  hitRate = HitRate / 100.0
  critMultiplier = 1.0 + (CritChance / 100.0 * CritDamageBonus)

  // Expected damage per attack
  return avgDamage * hitRate * critMultiplier
```

**Defensive Power:**

```go
CalculateDefensivePower(attr, config):
  // Effective health based on current HP
  healthRatio = CurrentHealth / MaxHealth
  effectiveHealth = MaxHealth * healthRatio

  // Resistance provides damage reduction
  avgResistance = (PhysicalResistance + MagicDefense) / 2.0

  // Dodge multiplier: HP / (1 - dodgeChance)
  dodgeChance = DodgeChance / 100.0
  dodgeMultiplier = 1.0 / max(1.0 - dodgeChance, 0.5)  // Cap at 2x

  return (effectiveHealth * dodgeMultiplier) + avgResistance
```

**Utility Power:**

```go
CalculateUtilityPower(entity, attr, roleData, config):
  // Sum all utility components (no sub-weights, just add together)
  return calculateRoleValue(roleData) + calculateAbilityValue(entity) + calculateCoverValue(entity)

calculateRoleValue(roleData):
  roleMultiplier = GetRoleMultiplierFromConfig(role)  // From powerconfig.json
  return roleMultiplier * RoleScalingFactor  // 10.0

calculateAbilityValue(entity):
  if !IsLeader(entity):
    return 0.0

  // Iterate AbilitySlotData.Slots (IsEquipped = true)
  totalPower = 0.0
  for each equipped ability slot:
    totalPower += GetAbilityPowerValue(slot.AbilityType)  // From powerconfig.json

  return totalPower

calculateCoverValue(entity):
  coverData = GetCoverData(entity)
  if coverData == nil:
    return 0.0

  return coverData.CoverValue * CoverScalingFactor * CoverBeneficiaryMultiplier
  // CoverScalingFactor=100.0, CoverBeneficiaryMultiplier=2.5
```

---

## Squad Power Calculation

**Function:** `CalculateSquadPower()` in `mind/evaluation/power.go`

**Algorithm:**

```go
CalculateSquadPower(squadID, manager, config):
  unitIDs = GetUnitIDsInSquad(squadID)
  if len(unitIDs) == 0:
    return 0.0

  // Sum unit powers
  totalUnitPower = 0.0
  for each unitID:
    unitPower = calculateUnitPower(unitID, manager, config)
    totalUnitPower += unitPower

  basePower = totalUnitPower

  // Apply squad-level modifiers
  compositionMod = CalculateSquadCompositionBonus(squadID, manager)
  basePower *= compositionMod

  healthPercent = GetSquadHealthPercent(squadID, manager)
  basePower *= CalculateHealthMultiplier(healthPercent, config.HealthPenalty)

  return basePower
```

**Composition Bonus:**

```go
CalculateSquadCompositionBonus(squadID, manager):
  attackTypes = set()

  for each unit in squad:
    attackTypes.add(unit.AttackType)  // Via TargetRowData.AttackType

  uniqueTypes = len(attackTypes)
  return GetCompositionBonusFromConfig(uniqueTypes)
```

**Configuration (powerconfig.json):**

```json
{
  "compositionBonuses": [
    {"uniqueTypes": 1, "bonus": 0.8},
    {"uniqueTypes": 2, "bonus": 1.1},
    {"uniqueTypes": 3, "bonus": 1.2},
    {"uniqueTypes": 4, "bonus": 1.3}
  ]
}
```

**Health Multiplier:**

```go
CalculateHealthMultiplier(healthPercent, healthPenalty):
  // Health penalty as exponent
  // e.g., 50% health with penalty 2.0 = 0.5^2 = 0.25 power
  return pow(healthPercent, healthPenalty)
```

**Why Composition Matters:**
- Encourages diverse unit types (melee row, melee column, ranged, magic)
- Penalizes mono-composition squads (0.8x)
- Rewards mixed squads (up to 1.3x for 4 types)
- Applies to both player and AI squads

---

## Squad Power By Range

**Function:** `CalculateSquadPowerByRange()` in `mind/evaluation/power.go`

**Purpose:** Computes how threatening a squad is at each distance. Used by AI threat assessment.

**Algorithm:**

```go
CalculateSquadPowerByRange(squadID, manager, config):
  unitIDs = GetUnitIDsInSquad(squadID)
  movementRange = GetSquadMovementSpeed(squadID)

  // Collect unit data
  units = []
  attackTypeCount = map[AttackType]int

  for each unitID:
    // Get attack range (defaults to 1 if AttackRangeComponent missing)
    attackRange = rangeData.Range  // From AttackRangeComponent

    // Track attack types for composition bonus
    attackTypeCount[attackType]++

    // Use FULL unit power calculation (same as CalculateSquadPower)
    unitPower = calculateUnitPower(unitID, manager, config)

    units.append({power: unitPower, attackRange: attackRange})

  // Find maximum threat range
  maxThreatRange = max(movementRange + unit.attackRange) for all units

  // Calculate power at each range
  powerByRange = map[int]float64

  for currentRange in [1, maxThreatRange]:
    rangePower = 0.0

    for each unit:
      effectiveThreatRange = movementRange + unit.attackRange

      if effectiveThreatRange >= currentRange:
        rangePower += unit.power  // Full unit power already includes role multiplier

    powerByRange[currentRange] = rangePower

  // Apply composition bonus
  compositionBonus = GetCompositionBonusFromConfig(len(attackTypeCount))
  for each range:
    powerByRange[range] *= compositionBonus

  return powerByRange
```

**Important Implementation Note:**
This function uses the **full** `calculateUnitPower()` calculation (including offensive, defensive, and utility components). Earlier documentation described a simplified formula (weapon + dex/2), which was a prior implementation. The current code uses the same unified power calculation as `CalculateSquadPower()`.

**Example Output:**

```go
// Squad with move=2, melee units (range 1), ranged units (range 3)
powerByRange = {
  1: 150.0,  // All units threaten at range 1 (move 2 + attack 1 >= 1)
  2: 150.0,  // All units threaten at range 2
  3: 150.0,  // All units threaten at range 3
  4: 80.0,   // Only ranged units threaten (move 2 + range 3 >= 4)
  5: 80.0    // Only ranged units threaten
}
```

**Usage in Threat System:**

```go
// CombatThreatLayer uses this data:
meleeThreat = squadThreat.ThreatByRange[1]  // Close-range power
rangedThreat = squadThreat.ThreatByRange[maxRange]  // Long-range power
```

---

## Role Multipliers and Ability Values

**Location:** `mind/evaluation/roles.go`

**Scaling Constants:**

```go
const (
    RoleScalingFactor          = 10.0   // Base multiplier for role value
    CoverScalingFactor         = 100.0  // Scale cover value (0.0-0.5) to comparable range (0-50)
    CoverBeneficiaryMultiplier = 2.5    // Average units protected per cover provider
)
```

**Role Multipliers (powerconfig.json):**

```json
{
  "roleMultipliers": [
    {"role": "Tank", "multiplier": 1.2},
    {"role": "DPS", "multiplier": 1.5},
    {"role": "Support", "multiplier": 1.0}
  ]
}
```

**Purpose:**
- DPS squads are inherently more threatening (1.5x)
- Tanks are moderately threatening (1.2x)
- Support provides utility but lower threat (1.0x)

**CRITICAL NOTE:**
- `roleMultipliers` (powerconfig.json) controls POWER SCALING (single positive scalar)
- `roleBehaviors` (aiconfig.json) controls AI POSITIONING WEIGHTS (4 floats, can be negative)
- These are NOT redundant - they serve orthogonal purposes

**Ability Power Values (powerconfig.json):**

```json
{
  "abilityValues": [
    {"ability": "Rally", "power": 15.0},
    {"ability": "Heal", "power": 20.0},
    {"ability": "BattleCry", "power": 12.0},
    {"ability": "Fireball", "power": 18.0},
    {"ability": "None", "power": 0.0}
  ]
}
```

**Usage:**
- Adds to unit's utility power
- Only leaders have abilities (checked via `LeaderComponent`)
- Multiple abilities stack via `AbilitySlotData.Slots` (sum of all equipped slots)

**Leader Bonus (powerconfig.json):**

```json
{
  "leaderBonus": 1.3
}
```

**Note:** The `leaderBonus` field exists in `powerconfig.json` and is accessible via `GetLeaderBonusFromConfig()` in `roles.go`. However, the current `CalculateSquadPowerByRange()` implementation uses `calculateUnitPower()` which bundles leader ability values into utility power rather than applying a separate multiplier. The `leaderBonus` value may be used in future systems or in template-based estimation (`EstimateUnitPowerFromTemplate`).

---

## DirtyCache

**Location:** `mind/evaluation/cache.go`

**Purpose:** Lazy evaluation with round-based invalidation. Embedded in all threat layers and `ThreatVisualizer`.

**Data Structure:**

```go
type DirtyCache struct {
  lastUpdateRound int
  isDirty         bool
  isInitialized   bool
}
```

**API:**

```go
NewDirtyCache() *DirtyCache  // Created in dirty state; first access triggers computation

IsValid(currentRound int) bool  // True if initialized, not dirty, and round matches
MarkDirty()                     // Invalidate; forces recomputation on next access
MarkClean(currentRound int)     // Mark as valid for given round
IsDirty() bool                  // Whether recomputation is needed
IsInitialized() bool            // Whether computed at least once
GetLastUpdateRound() int        // Round number of last update
```

**Usage Pattern:**

```go
// In a layer's Compute() method:
func (layer *SomeLayer) Compute(currentRound int) {
    // ... compute data ...
    layer.markClean(currentRound)  // Calls DirtyCache.MarkClean internally
}

// In CompositeThreatEvaluator.Update():
if !cte.isDirty && cte.lastUpdateRound == currentRound {
    return  // Use IsValid() pattern
}
```

---

## Performance Considerations

**Complexity:** O(units) per squad

**Optimization Techniques:**

1. **Shared Calculation**
   - Single implementation for AI and encounter generation
   - Eliminates code duplication and maintenance
   - Ensures consistent threat/power assessment

2. **Component Batching**
   - `GetUnitCombatData()` bundles all components in one query
   - Reduces component access overhead
   - Pre-calculates derived values (role, attack type, range)

3. **Config Caching**
   - PowerConfig loaded once per profile
   - Role/ability lookups iterate small arrays (<10 entries)
   - Could use maps for O(1) lookup if needed

**Bottlenecks:**

- **CalculateSquadPowerByRange()**: O(units x ranges)
  - Now uses full `calculateUnitPower()` per unit (vs. old simplified formula)
  - Called once per squad during threat update
  - Typical cost: 4 units x 5 ranges = 20 iterations

---

## Extension Points

### Adding New Power Factors

**Steps:**

1. **Define Component** (if not existing)
   ```go
   type NewPowerData struct {
     PowerValue float64
   }
   var NewPowerComponent *ecs.Component
   ```

2. **Update CalculateUtilityPower()**
   ```go
   func CalculateUtilityPower(entity, attr, roleData, config) float64 {
     roleValue := calculateRoleValue(roleData)
     abilityValue := calculateAbilityValue(entity)
     coverValue := calculateCoverValue(entity)

     // New power factor
     newValue := calculateNewPowerFactor(entity)

     return roleValue + abilityValue + coverValue + newValue
   }
   ```

3. **Implement Calculation**
   ```go
   func calculateNewPowerFactor(entity *ecs.Entity) float64 {
     data := GetComponentType[*NewPowerData](entity, NewPowerComponent)
     if data == nil:
       return 0.0

     return data.PowerValue * scalingConstant
   }
   ```

4. **Update powerconfig.json (if configurable)**
   ```json
   {
     "newPowerFactorScaling": 25.0
   }
   ```

5. **Update EstimateUnitPowerFromTemplate()**
   ```go
   // If new factor applies to templates
   func EstimateUnitPowerFromTemplate(unit UnitTemplate, config) {
     // Existing calculations

     // New factor (if applicable to templates)
     newValue := unit.NewPowerValue * scalingConstant

     utilityPower += newValue
   }
   ```

---

## File Reference

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/evaluation/power.go` | Power calculation | `CalculateSquadPower()`, `CalculateSquadPowerByRange()`, `CalculateOffensivePower()`, `CalculateDefensivePower()`, `CalculateUtilityPower()`, `EstimateUnitPowerFromTemplate()` |
| `mind/evaluation/power_config.go` | Config loading | `GetPowerConfigByProfile()` |
| `mind/evaluation/roles.go` | Role/ability config | `GetRoleMultiplierFromConfig()`, `GetAbilityPowerValue()`, `GetCompositionBonusFromConfig()` |
| `mind/evaluation/cache.go` | Dirty flag system | `DirtyCache`, `MarkDirty()`, `MarkClean()`, `IsValid()`, `IsDirty()`, `IsInitialized()` |

---

**End of Document**

For questions or clarifications, consult the source code or the [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) overview.
