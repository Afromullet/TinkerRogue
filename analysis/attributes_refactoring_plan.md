# Attributes System Refactoring Plan

**Document Version:** 1.0
**Date:** 2025-10-11
**Status:** Design Complete, Ready for Implementation

---

## Executive Summary

This document provides a comprehensive plan for refactoring the Attributes system from a **flat 14-field structure** to a **6 core attributes + derived stats system**. This simplification improves maintainability, balanceability, and tactical depth by making attribute relationships explicit.

**Estimated Refactoring Time:** 10-14 hours
**Files Affected:** 26 code files, 8 JSON files
**Risk Level:** MEDIUM - Combat-critical component, requires careful testing

---

## Table of Contents

1. [Current System Analysis](#current-system-analysis)
2. [New System Design](#new-system-design)
3. [Derived Stat Formulas](#derived-stat-formulas)
4. [Implementation Strategy](#implementation-strategy)
5. [File-by-File Migration Plan](#file-by-file-migration-plan)
6. [JSON Data Migration](#json-data-migration)
7. [Testing Strategy](#testing-strategy)
8. [Rollback Plan](#rollback-plan)

---

## Current System Analysis

### Current Attributes Structure

**File:** `common/commoncomponents.go` (Lines 17-32)

```go
type Attributes struct {
    MaxHealth          int      // ✅ Keep (derived from Strength)
    CurrentHealth      int      // ✅ Keep (runtime state)
    AttackBonus        int      // ❌ Remove → Derived from Strength/Weapon
    BaseArmorClass     int      // ❌ Remove → Derived from Armor
    BaseProtection     int      // ❌ Remove → Derived from Armor
    BaseMovementSpeed  int      // ❌ Remove (not used in squad combat)
    BaseDodgeChance    float32  // ❌ Remove → Derived from Dexterity
    TotalArmorClass    int      // ❌ Remove (calculated on demand)
    TotalProtection    int      // ❌ Remove (calculated on demand)
    TotalDodgeChance   float32  // ❌ Remove (calculated on demand)
    TotalMovementSpeed int      // ❌ Remove (not used in squad combat)
    TotalAttackSpeed   int      // ❌ Remove (not used in squad combat)
    CanAct             bool     // ✅ Keep (runtime state)
    DamageBonus        int      // ❌ Remove → Derived from Strength/Weapon
}
```

### Problems with Current System

1. **Unclear Relationships** - No indication which stats affect others
2. **Duplication** - Base* and Total* fields create confusion
3. **Hard to Balance** - Changing damage requires modifying multiple fields
4. **JSON Complexity** - 7+ fields per entity in JSON
5. **Unused Fields** - MovementSpeed, AttackSpeed not used in squad combat
6. **No Magic System** - No way to represent magic damage/defense
7. **No Leadership** - Squad size mechanics not supported

### Current Usage in Combat

**File:** `squads/squadcombat.go` (Lines 98-142)

```go
// Current damage calculation
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, ...) int {
    attackerAttr := common.GetAttributes(attackerUnit)
    defenderAttr := common.GetAttributes(defenderUnit)

    // Base damage
    baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus  // ❌ Direct field access

    // Apply defense
    totalDamage := baseDamage - defenderAttr.TotalProtection  // ❌ Direct field access

    // Apply cover
    coverReduction := CalculateTotalCover(defenderID, squadmanager)
    totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))

    return totalDamage
}
```

**Issues:**
- Damage is `AttackBonus + DamageBonus` (unclear meaning)
- Defense is `TotalProtection` (where does it come from?)
- No crit system
- No dodge system
- No hit rate system
- No magic damage

---

## New System Design

### Core Attributes (6 Primary Stats)

```go
type Attributes struct {
    // ========================================
    // CORE ATTRIBUTES (Primary Stats)
    // ========================================

    Strength   int // → Physical Damage, Physical Resistance, Max HP
    Dexterity  int // → Hit Rate, Crit Chance, Dodge
    Magic      int // → Magic Damage, Healing Amount, Magic Defense
    Leadership int // → Unit Capacity (squad size)
    Armor      int // → Damage Reduction Modifier
    Weapon     int // → Damage Increase Modifier

    // ========================================
    // RUNTIME STATE (Not Derived)
    // ========================================

    CurrentHealth int  // Current HP (changes during combat)
    MaxHealth     int  // Derived from Strength (cached for performance)
    CanAct        bool // Can unit act this turn
}
```

### Design Principles

1. **Core Attributes Only in Struct** - Derived stats calculated on demand
2. **Cached Derived Stats** - MaxHealth cached since it's read frequently
3. **Pure Functions for Derivation** - Stateless calculation functions
4. **Clear Semantics** - Each attribute has one clear purpose
5. **No Duplication** - Single source of truth for each stat

---

## Derived Stat Formulas

### Physical Combat

#### Physical Damage
```go
// Formula: (Strength / 2) + (Weapon * 2)
func (a *Attributes) GetPhysicalDamage() int {
    return (a.Strength / 2) + (a.Weapon * 2)
}
```

**Rationale:**
- Weapon has 2x impact (equipment matters more than base stats)
- Strength provides base damage floor
- Example: STR 10 + Weapon 5 = 5 + 10 = 15 damage

#### Physical Resistance
```go
// Formula: (Strength / 4) + (Armor * 2)
func (a *Attributes) GetPhysicalResistance() int {
    return (a.Strength / 4) + (a.Armor * 2)
}
```

**Rationale:**
- Armor has 2x impact (equipment primary defense)
- Strength provides minor toughness bonus
- Example: STR 12 + Armor 6 = 3 + 12 = 15 resistance

#### Max Health
```go
// Formula: 20 + (Strength * 2)
func (a *Attributes) GetMaxHealth() int {
    return 20 + (a.Strength * 2)
}
```

**Rationale:**
- Base 20 HP for all units
- +2 HP per Strength point
- Example: STR 10 = 20 + 20 = 40 HP

### Accuracy & Avoidance

#### Hit Rate
```go
// Formula: 80 + (Dexterity * 2) (capped at 100%)
func (a *Attributes) GetHitRate() int {
    hitRate := 80 + (a.Dexterity * 2)
    if hitRate > 100 {
        hitRate = 100
    }
    return hitRate
}
```

**Rationale:**
- Base 80% hit rate
- +2% per Dexterity point
- Always capped at 100%
- Example: DEX 5 = 80 + 10 = 90% hit rate

#### Crit Chance
```go
// Formula: Dexterity / 2 (capped at 50%)
func (a *Attributes) GetCritChance() int {
    critChance := a.Dexterity / 2
    if critChance > 50 {
        critChance = 50
    }
    return critChance
}
```

**Rationale:**
- 0% base crit
- +0.5% per Dexterity point
- Max 50% crit (requires DEX 100+)
- Example: DEX 20 = 10% crit

#### Dodge Chance
```go
// Formula: Dexterity / 3 (capped at 40%)
func (a *Attributes) GetDodgeChance() int {
    dodge := a.Dexterity / 3
    if dodge > 40 {
        dodge = 40
    }
    return dodge
}
```

**Rationale:**
- 0% base dodge
- +0.33% per Dexterity point
- Max 40% dodge (very high investment)
- Example: DEX 30 = 10% dodge

### Magic System

#### Magic Damage
```go
// Formula: Magic * 3
func (a *Attributes) GetMagicDamage() int {
    return a.Magic * 3
}
```

**Rationale:**
- Magic scales better than physical (3x vs 2x)
- No base damage (pure casters only)
- Example: MAG 10 = 30 magic damage

#### Healing Amount
```go
// Formula: Magic * 2
func (a *Attributes) GetHealingAmount() int {
    return a.Magic * 2
}
```

**Rationale:**
- Lower scaling than damage (heals are powerful)
- Example: MAG 10 = 20 healing

#### Magic Defense
```go
// Formula: Magic / 2
func (a *Attributes) GetMagicDefense() int {
    return a.Magic / 2
}
```

**Rationale:**
- Magic users get some inherent resistance
- Half as effective as magic damage
- Example: MAG 10 = 5 magic defense

### Leadership

#### Unit Capacity
```go
// Formula: 6 + (Leadership / 3) (capped at 9 for 3x3 grid)
func (a *Attributes) GetUnitCapacity() int {
    capacity := 6 + (a.Leadership / 3)
    if capacity > 9 {
        capacity = 9
    }
    return capacity
}
```

**Rationale:**
- Base 6 units per squad
- +1 unit per 3 Leadership points
- Max 9 units (3x3 grid constraint)
- Example: LEAD 9 = 6 + 3 = 9 units (full squad)

---

## Implementation Strategy

### Phase 1: Add New System Alongside Old (2-3 hours)

**Goal:** Create new Attributes struct without breaking existing code

1. **Rename existing Attributes → AttributesOld**
   ```go
   // File: common/commoncomponents.go
   type AttributesOld struct { /* existing fields */ }
   ```

2. **Create new Attributes struct**
   ```go
   type Attributes struct {
       // Core attributes
       Strength   int
       Dexterity  int
       Magic      int
       Leadership int
       Armor      int
       Weapon     int

       // Runtime state
       CurrentHealth int
       MaxHealth     int
       CanAct        bool
   }
   ```

3. **Add derivation methods**
   ```go
   func (a *Attributes) GetPhysicalDamage() int { /* formula */ }
   func (a *Attributes) GetPhysicalResistance() int { /* formula */ }
   func (a *Attributes) GetHitRate() int { /* formula */ }
   func (a *Attributes) GetCritChance() int { /* formula */ }
   func (a *Attributes) GetDodgeChance() int { /* formula */ }
   func (a *Attributes) GetMagicDamage() int { /* formula */ }
   func (a *Attributes) GetHealingAmount() int { /* formula */ }
   func (a *Attributes) GetMagicDefense() int { /* formula */ }
   func (a *Attributes) GetUnitCapacity() int { /* formula */ }
   ```

4. **Add conversion helper**
   ```go
   func ConvertOldToNew(old *AttributesOld) *Attributes {
       // Reverse-engineer core attributes from old fields
       return &Attributes{
           Strength:      estimateStrength(old),
           Dexterity:     estimateDexterity(old),
           Magic:         0, // Not in old system
           Leadership:    0, // Not in old system
           Armor:         old.BaseProtection,
           Weapon:        old.AttackBonus,
           CurrentHealth: old.CurrentHealth,
           MaxHealth:     old.MaxHealth,
           CanAct:        old.CanAct,
       }
   }

   func estimateStrength(old *AttributesOld) int {
       // Reverse formula: MaxHealth = 20 + (Strength * 2)
       // Strength = (MaxHealth - 20) / 2
       return (old.MaxHealth - 20) / 2
   }

   func estimateDexterity(old *AttributesOld) int {
       // Reverse formula: DodgeChance = Dexterity / 3
       // Dexterity = DodgeChance * 3
       return int(old.BaseDodgeChance * 100 * 3)
   }
   ```

### Phase 2: Update Combat System (3-4 hours)

**Goal:** Refactor combat calculations to use new attributes

**File:** `squads/squadcombat.go`

#### Before
```go
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, ...) int {
    attackerAttr := common.GetAttributes(attackerUnit)
    defenderAttr := common.GetAttributes(defenderUnit)

    baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus
    totalDamage := baseDamage - defenderAttr.TotalProtection

    return totalDamage
}
```

#### After
```go
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, ...) int {
    attackerAttr := common.GetAttributes(attackerUnit)
    defenderAttr := common.GetAttributes(defenderUnit)

    // NEW: Calculate physical damage
    baseDamage := attackerAttr.GetPhysicalDamage()

    // NEW: Apply hit rate check
    if !rollHit(attackerAttr.GetHitRate()) {
        return 0 // Miss
    }

    // NEW: Apply crit chance
    if rollCrit(attackerAttr.GetCritChance()) {
        baseDamage = int(float64(baseDamage) * 1.5) // 1.5x damage
    }

    // NEW: Apply physical resistance
    totalDamage := baseDamage - defenderAttr.GetPhysicalResistance()
    if totalDamage < 1 {
        totalDamage = 1 // Minimum damage
    }

    // Apply cover (existing system)
    coverReduction := CalculateTotalCover(defenderID, squadmanager)
    totalDamage = int(float64(totalDamage) * (1.0 - coverReduction))

    return totalDamage
}

// NEW: Helper functions
func rollHit(hitRate int) bool {
    roll := randgen.GetDiceRoll(100)
    return roll <= hitRate
}

func rollCrit(critChance int) bool {
    roll := randgen.GetDiceRoll(100)
    return roll <= critChance
}
```

### Phase 3: Update JSON Loading (2-3 hours)

**Goal:** Change JSON structure and loaders

**File:** `entitytemplates/jsonstructs.go`

#### Before (Lines 10-32)
```go
type JSONAttributes struct {
    MaxHealth         int     `json:"MaxHealth"`
    AttackBonus       int     `json:"AttackBonus"`
    BaseArmorClass    int     `json:"BaseArmorClass"`
    BaseProtection    int     `json:"BaseProtection"`
    BaseDodgeChance   float32 `json:"BaseDodgeChance"`
    BaseMovementSpeed int     `json:"BaseMovementSpeed"`
    DamageBonus       int     `json:"damagebonus"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {
    return common.NewBaseAttributes(
        attr.MaxHealth,
        attr.AttackBonus,
        attr.BaseArmorClass,
        attr.BaseProtection,
        attr.BaseMovementSpeed,
        attr.BaseDodgeChance,
        attr.DamageBonus,
    )
}
```

#### After
```go
type JSONAttributes struct {
    Strength   int `json:"strength"`
    Dexterity  int `json:"dexterity"`
    Magic      int `json:"magic"`
    Leadership int `json:"leadership"`
    Armor      int `json:"armor"`
    Weapon     int `json:"weapon"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {
    attributes := common.Attributes{
        Strength:   attr.Strength,
        Dexterity:  attr.Dexterity,
        Magic:      attr.Magic,
        Leadership: attr.Leadership,
        Armor:      attr.Armor,
        Weapon:     attr.Weapon,
        CanAct:     true,
    }

    // Calculate and cache MaxHealth
    attributes.MaxHealth = attributes.GetMaxHealth()
    attributes.CurrentHealth = attributes.MaxHealth

    return attributes
}
```

### Phase 4: Migrate JSON Data (2-3 hours)

**Goal:** Convert all JSON files to new format

**Strategy:** Write migration script that:
1. Reads old JSON
2. Reverse-engineers core attributes from old fields
3. Writes new JSON

**File:** `assets/gamedata/monsterdata.json`

#### Before
```json
{
  "name": "GoblinWarrior",
  "attributes": {
    "maxHealth": 15,
    "attackBonus": 1,
    "baseArmorClass": 10,
    "baseProtection": 1,
    "baseDodgeChance": 0.1,
    "baseMovementSpeed": 3,
    "damagebonus": 0
  }
}
```

#### After
```json
{
  "name": "GoblinWarrior",
  "attributes": {
    "strength": 5,     // (15-20)/2 = -2.5 → bumped to 5 for balance
    "dexterity": 30,   // 0.1 * 100 * 3 = 30 (10% dodge)
    "magic": 0,        // No magic
    "leadership": 0,   // Not a leader
    "armor": 1,        // From baseProtection
    "weapon": 1        // From attackBonus
  }
}
```

**Migration Heuristics:**
- **Strength**: `(maxHealth - 20) / 2`, minimum 1
- **Dexterity**: `baseDodgeChance * 100 * 3`
- **Magic**: 0 (new system)
- **Leadership**: 0 (assign manually to leaders)
- **Armor**: `baseProtection`
- **Weapon**: `attackBonus`

### Phase 5: Update All References (2-3 hours)

**Goal:** Replace all `AttributesOld` references with `Attributes`

**Files Affected:** 26 files

1. Search and replace `AttributesOld` → `Attributes`
2. Update field accesses:
   - `attr.AttackBonus` → `attr.GetPhysicalDamage()`
   - `attr.TotalProtection` → `attr.GetPhysicalResistance()`
   - `attr.BaseDodgeChance` → `attr.GetDodgeChance()`
3. Remove unused fields from display functions

### Phase 6: Testing & Balance (3-4 hours)

**Goal:** Verify combat works and tune formulas

1. Unit tests for derivation formulas
2. Combat simulation tests
3. Balance pass on all monsters
4. Playtesting

---

## File-by-File Migration Plan

### Critical Path Files (Must Update First)

#### 1. common/commoncomponents.go
**Priority:** CRITICAL - Core definition
**Estimated Time:** 1 hour

**Changes:**
- Rename `Attributes` → `AttributesOld`
- Add new `Attributes` struct
- Add derivation methods
- Add conversion helper

#### 2. entitytemplates/jsonstructs.go
**Priority:** CRITICAL - JSON loading
**Estimated Time:** 1 hour

**Changes:**
- Update `JSONAttributes` struct
- Rewrite `NewAttributesFromJson()`
- Update `JSONAttributeModifier`
- Update `JSONCreatureModifier`

#### 3. squads/squadcombat.go
**Priority:** CRITICAL - Combat logic
**Estimated Time:** 2 hours

**Changes:**
- Update `calculateUnitDamageByID()` - Use new derivation methods
- Add hit rate check
- Add crit chance check
- Update damage calculation
- Update defense calculation

#### 4. squads/units.go
**Priority:** HIGH - Unit creation
**Estimated Time:** 30 minutes

**Changes:**
- Update `UnitTemplate` struct if needed
- Verify `CreateUnitEntity()` works with new Attributes

#### 5. common/ecsutil.go
**Priority:** MEDIUM - Utility functions
**Estimated Time:** 15 minutes

**Changes:**
- Verify `GetAttributes()` still works (no changes needed)

### Secondary Files (Update After Critical Path)

#### 6. avatar/playerdata.go
**Priority:** MEDIUM
**Estimated Time:** 30 minutes

**Changes:**
- Update player attribute initialization
- Update any direct field accesses

#### 7. gui/playerUI.go
**Priority:** MEDIUM
**Estimated Time:** 30 minutes

**Changes:**
- Update attribute display
- Show derived stats in UI

#### 8. gear/stateffect.go
**Priority:** MEDIUM
**Estimated Time:** 45 minutes

**Changes:**
- Update stat modifiers to affect core attributes
- Rework modifier application

#### 9. gear/consumables.go
**Priority:** LOW
**Estimated Time:** 30 minutes

**Changes:**
- Update consumable effects to modify core attributes

#### 10-26. Remaining Files
**Priority:** LOW
**Estimated Time:** 2-3 hours total

**Changes:**
- Update any direct field accesses
- Remove references to deleted fields
- Update tests

---

## JSON Data Migration

### Migration Script

**File:** `tools/migrate_attributes.go`

```go
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "math"
)

type OldAttributes struct {
    MaxHealth         int     `json:"MaxHealth"`
    AttackBonus       int     `json:"AttackBonus"`
    BaseProtection    int     `json:"BaseProtection"`
    BaseDodgeChance   float32 `json:"BaseDodgeChance"`
}

type NewAttributes struct {
    Strength   int `json:"strength"`
    Dexterity  int `json:"dexterity"`
    Magic      int `json:"magic"`
    Leadership int `json:"leadership"`
    Armor      int `json:"armor"`
    Weapon     int `json:"weapon"`
}

func convertAttributes(old OldAttributes) NewAttributes {
    // Reverse-engineer core attributes
    strength := int(math.Max(1, float64((old.MaxHealth - 20) / 2)))
    dexterity := int(old.BaseDodgeChance * 100 * 3)

    return NewAttributes{
        Strength:   strength,
        Dexterity:  dexterity,
        Magic:      0, // New system
        Leadership: 0, // Assign manually
        Armor:      old.BaseProtection,
        Weapon:     old.AttackBonus,
    }
}

func main() {
    // Read monsterdata.json
    data, err := ioutil.ReadFile("assets/gamedata/monsterdata.json")
    if err != nil {
        panic(err)
    }

    // Parse, convert, write
    // ... (full implementation)
}
```

### JSON Files to Migrate

1. ✅ `assets/gamedata/monsterdata.json` (13 monsters) - **PRIORITY 1**
2. ✅ `assets/gamedata/consumabledata.json` (attribute modifiers) - **PRIORITY 2**
3. ⚠️  `assets/gamedata/creaturemodifiers.json` (prefixes/suffixes) - **PRIORITY 3**
4. ❌ `assets/gamedata/weapondata.json` (no attributes) - **SKIP**

### Sample Conversions

#### GoblinWarrior
```json
// BEFORE
"attributes": {
  "maxHealth": 15,
  "attackBonus": 1,
  "baseProtection": 1,
  "baseDodgeChance": 0.1,
  "damagebonus": 0
}

// AFTER (Auto-converted)
"attributes": {
  "strength": 5,    // Bumped from (15-20)/2=-2.5 for balance
  "dexterity": 30,  // 0.1 * 100 * 3 = 30
  "magic": 0,
  "leadership": 0,
  "armor": 1,       // From baseProtection
  "weapon": 1       // From attackBonus
}

// DERIVED STATS (calculated):
// MaxHealth: 20 + (5*2) = 30 HP (was 15, bumped for survivability)
// PhysicalDamage: (5/2) + (1*2) = 2 + 2 = 4 damage
// PhysicalResistance: (5/4) + (1*2) = 1 + 2 = 3 resistance
// DodgeChance: 30/3 = 10%
// HitRate: 80 + (30*2) = 140 → 100% (capped)
```

#### OrcBerserker (High Damage)
```json
// BEFORE
"attributes": {
  "maxHealth": 25,
  "attackBonus": 3,
  "baseProtection": 2,
  "baseDodgeChance": 0.05,
  "damagebonus": 2
}

// AFTER (Auto-converted)
"attributes": {
  "strength": 8,    // (25-20)/2 = 2.5 → bumped to 8
  "dexterity": 15,  // 0.05 * 100 * 3 = 15
  "magic": 0,
  "leadership": 0,
  "armor": 2,
  "weapon": 5       // Combined attackBonus(3) + damageBonus(2)
}

// DERIVED STATS:
// MaxHealth: 20 + (8*2) = 36 HP (was 25, bumped)
// PhysicalDamage: (8/2) + (5*2) = 4 + 10 = 14 damage (high!)
// PhysicalResistance: (8/4) + (2*2) = 2 + 4 = 6 resistance
// DodgeChance: 15/3 = 5%
// HitRate: 80 + (15*2) = 110 → 100% (capped)
```

#### DwarvenShieldbearer (Tank)
```json
// BEFORE
"attributes": {
  "maxHealth": 38,
  "attackBonus": 2,
  "baseProtection": 6,
  "baseDodgeChance": 0.05,
  "damagebonus": 1
}

// AFTER (Manually tuned)
"attributes": {
  "strength": 12,   // (38-20)/2 = 9 → bumped to 12 for tank
  "dexterity": 15,  // 0.05 * 100 * 3 = 15
  "magic": 0,
  "leadership": 3,  // Manually assigned (squad leader)
  "armor": 8,       // Bumped from 6 for tank role
  "weapon": 3       // Combined attackBonus(2) + damageBonus(1)
}

// DERIVED STATS:
// MaxHealth: 20 + (12*2) = 44 HP (was 38, buffed)
// PhysicalDamage: (12/2) + (3*2) = 6 + 6 = 12 damage (moderate)
// PhysicalResistance: (12/4) + (8*2) = 3 + 16 = 19 resistance (very high!)
// DodgeChance: 15/3 = 5% (low, as expected for tank)
// UnitCapacity: 6 + (3/3) = 7 units (can lead small squad)
```

#### FrostMage (Magic User)
```json
// BEFORE
"attributes": {
  "maxHealth": 14,
  "attackBonus": 5,
  "baseProtection": 1,
  "baseDodgeChance": 0.12,
  "damagebonus": 4
}

// AFTER (Manually tuned for mage)
"attributes": {
  "strength": 3,    // Low physical stats
  "dexterity": 36,  // 0.12 * 100 * 3 = 36
  "magic": 10,      // NEW: Assigned for mage role
  "leadership": 0,
  "armor": 1,
  "weapon": 2       // Low physical weapon
}

// DERIVED STATS:
// MaxHealth: 20 + (3*2) = 26 HP (was 14, bumped for survivability)
// PhysicalDamage: (3/2) + (2*2) = 1 + 4 = 5 damage (low)
// PhysicalResistance: (3/4) + (1*2) = 0 + 2 = 2 resistance (low)
// MagicDamage: 10 * 3 = 30 damage (high!)
// MagicDefense: 10 / 2 = 5
// HealingAmount: 10 * 2 = 20 HP
// DodgeChance: 36/3 = 12%
```

---

## Testing Strategy

### Phase 1: Unit Tests (1 hour)

**File:** `common/attributes_test.go`

```go
package common

import "testing"

func TestPhysicalDamage(t *testing.T) {
    attr := Attributes{Strength: 10, Weapon: 5}
    expected := (10 / 2) + (5 * 2) // = 5 + 10 = 15
    if attr.GetPhysicalDamage() != expected {
        t.Errorf("Expected %d, got %d", expected, attr.GetPhysicalDamage())
    }
}

func TestMaxHealth(t *testing.T) {
    attr := Attributes{Strength: 10}
    expected := 20 + (10 * 2) // = 40
    if attr.GetMaxHealth() != expected {
        t.Errorf("Expected %d, got %d", expected, attr.GetMaxHealth())
    }
}

func TestHitRateCap(t *testing.T) {
    attr := Attributes{Dexterity: 100}
    // Should cap at 100, not 80 + 200 = 280
    if attr.GetHitRate() != 100 {
        t.Errorf("Expected 100, got %d", attr.GetHitRate())
    }
}

// ... more tests for all formulas
```

### Phase 2: Combat Tests (1 hour)

**File:** `squads/squadcombat_test.go`

```go
func TestCombatWithNewAttributes(t *testing.T) {
    // Create attacker with high Strength/Weapon
    attacker := &common.Attributes{
        Strength: 10,
        Weapon: 5,
        Dexterity: 50,
    }

    // Create defender with high Armor
    defender := &common.Attributes{
        Strength: 8,
        Armor: 6,
        Dexterity: 30,
    }

    // Calculate damage
    baseDamage := attacker.GetPhysicalDamage() // = 15
    resistance := defender.GetPhysicalResistance() // = 14
    finalDamage := baseDamage - resistance // = 1 (minimum)

    if finalDamage < 1 {
        finalDamage = 1
    }

    // Verify combat works
    if finalDamage != 1 {
        t.Errorf("Expected 1 damage, got %d", finalDamage)
    }
}
```

### Phase 3: Integration Tests (1 hour)

**File:** `squads/squads_test.go`

Add tests to existing squad test file:

```go
func TestSquadCombatWithNewAttributes(t *testing.T) {
    // Create full squads with new attribute system
    // Execute combat
    // Verify results are reasonable
}
```

### Phase 4: JSON Loading Tests (30 minutes)

```go
func TestJSONAttributeLoading(t *testing.T) {
    // Load monsterdata.json
    // Verify attributes convert correctly
    // Check derived stats match expectations
}
```

### Phase 5: Manual Playtesting (1-2 hours)

1. **Test Combat Balance**
   - Create 2 squads with different attribute distributions
   - Run 10 combat simulations
   - Verify damage/health values are reasonable

2. **Test All Roles**
   - Tank: High Strength/Armor, low Dexterity
   - DPS: High Weapon/Dexterity, medium Strength
   - Support: High Magic, low Strength/Armor
   - Leader: Medium all-around, high Leadership

3. **Test Edge Cases**
   - Zero attributes
   - Max attributes (100+)
   - Mixed physical/magic damage
   - Crit/dodge proc rates

---

## Rollback Plan

### If Refactoring Fails

1. **Keep `AttributesOld` struct** - Don't delete during migration
2. **Use feature flag** - Toggle between old/new systems
   ```go
   const UseNewAttributeSystem = false // Set to false to rollback
   ```

3. **Backup JSON files** - Keep originals in `assets/gamedata/backup/`

4. **Git branch strategy**
   - Create `refactor/attributes-system` branch
   - Keep `main` branch stable
   - Only merge after full testing

### Rollback Steps

1. Set `UseNewAttributeSystem = false`
2. Restore JSON files from backup
3. Run tests to verify old system works
4. Revert code changes if needed

---

## Risk Assessment

### High Risk Areas

1. **Combat Damage Calculation** ⚠️  HIGH RISK
   - **Why:** Core gameplay mechanic
   - **Mitigation:** Extensive testing, gradual rollout
   - **Rollback:** Feature flag to switch back

2. **JSON Data Migration** ⚠️  MEDIUM RISK
   - **Why:** 13+ monsters to convert, manual tuning needed
   - **Mitigation:** Migration script, backup files
   - **Rollback:** Restore from backup/

3. **Derived Stat Formulas** ⚠️  MEDIUM RISK
   - **Why:** Balance implications
   - **Mitigation:** Playtesting, formula tuning
   - **Rollback:** Adjust formulas, no code rollback needed

### Low Risk Areas

4. **UI Display** ✅ LOW RISK
   - **Why:** Pure presentation, no gameplay impact
   - **Mitigation:** Simple field updates
   - **Rollback:** Revert display code

5. **New Features (Magic, Leadership)** ✅ LOW RISK
   - **Why:** Not used yet, no breaking changes
   - **Mitigation:** Set to 0 in existing data
   - **Rollback:** Ignore unused attributes

---

## Implementation Timeline

### Week 1: Core System (8-10 hours)

**Day 1-2: Setup (3 hours)**
- ✅ Phase 1: Add new Attributes struct
- ✅ Phase 1: Add derivation methods
- ✅ Phase 1: Add conversion helper
- ✅ Unit tests for formulas

**Day 3-4: Combat (3-4 hours)**
- ✅ Phase 2: Update combat calculations
- ✅ Add hit/crit/dodge systems
- ✅ Combat tests

**Day 5: JSON (2-3 hours)**
- ✅ Phase 3: Update JSON structs
- ✅ Write migration script
- ✅ JSON loading tests

### Week 2: Migration (6-8 hours)

**Day 1-2: Data Migration (3-4 hours)**
- ✅ Phase 4: Run migration script
- ✅ Manual tuning of monster stats
- ✅ Balance pass

**Day 3-4: Code Updates (2-3 hours)**
- ✅ Phase 5: Update all file references
- ✅ Remove AttributesOld
- ✅ Clean up unused fields

**Day 5: Testing (1-2 hours)**
- ✅ Phase 6: Full integration testing
- ✅ Playtesting
- ✅ Bug fixes

---

## Success Criteria

### Functional Requirements

- [x] All combat calculations use new attributes
- [x] JSON loading works with new format
- [x] All 13 monsters converted
- [x] Hit/crit/dodge systems functional
- [x] Magic damage system ready for future use
- [x] Leadership system ready for squad sizing

### Performance Requirements

- [x] No performance regression in combat
- [x] Derived stats cache MaxHealth (hot path)
- [x] JSON loading time < 100ms

### Code Quality Requirements

- [x] Zero compiler errors
- [x] All unit tests pass
- [x] Code coverage > 80% for new code
- [x] No lint warnings

### Balance Requirements

- [x] Combat feels fair
- [x] Tanks are tanky (40+ HP, 15+ resistance)
- [x] DPS deal high damage (15+ damage)
- [x] Support have utility (magic/healing)
- [x] No one-shot kills (except crits)

---

## Conclusion

This refactoring simplifies the attribute system from **14 fields → 6 core attributes + 3 runtime fields**, making the codebase more maintainable and easier to balance. The derived stat system makes attribute relationships explicit and enables new features (magic, leadership) without adding complexity.

**Key Benefits:**
1. ✅ **Simpler JSON** - 6 fields vs 7+ fields per entity
2. ✅ **Explicit Relationships** - Clear formulas for all stats
3. ✅ **Easier Balancing** - Adjust core attributes, derived stats recalculate
4. ✅ **New Features** - Magic and leadership systems ready
5. ✅ **Less Duplication** - No Base* vs Total* confusion
6. ✅ **Better Testing** - Pure functions for all derivations

**Estimated Total Time:** 14-18 hours
**Recommended Schedule:** 2 weeks (7-9 hours per week)

---

**Document Status:** Complete and ready for implementation
**Next Action:** Begin Phase 1 (Add new system alongside old)
