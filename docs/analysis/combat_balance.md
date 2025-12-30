# Phase 1: Critical Fixes - Implementation Plan

**Goal:** Make all unit classes functional
**Duration:** Week 1 (8-15 hours total)
**Status:** Not Started

---

## Overview

Phase 1 addresses the four **game-breaking** issues identified in the balance analysis:

1. **Magic units are non-functional** (0.05-0.06 efficiency)
2. **Warrior is catastrophically overpowered** (10.09 efficiency)
3. **Ranged units have no range advantage** (1.3 round survival)
4. **Cover system completely inactive** (0% activation)

These issues prevent any meaningful tactical gameplay. All must be fixed before proceeding to Phase 2 balance tuning.

---

## Exit Criteria

Before Phase 1 is complete, ALL of the following must be true:

- ✓ All unit classes have 0.5-2.0 efficiency in their primary role matchups
- ✓ No unit class has <20% survival rate in role-appropriate matchups
- ✓ Combat duration >2.0 rounds for all scenarios (no instant losses)
- ✓ Magic users deal meaningful damage (20-30 damage range)
- ✓ Warrior efficiency drops below 3.0 in DPS matchups
- ✓ Ranged units survive at least 3 rounds vs melee
- ✓ Cover system activates 25-35% of the time

---

## Task 1: Fix Magic Damage Calculation

**Priority:** P0 (Critical)
**Estimated Time:** 2 hours
**Dependencies:** None
**Validation:** Magic efficiency 0.06 → 0.8+

### Problem Statement

Magic units (Wizard, Sorcerer, Mage) are completely non-functional:
- Deal only 2.7 damage per attack (vs Fighter's 65.0 damage)
- Die in 1 round with 0% survival rate
- Efficiency: 0.05-0.06 (400x worse than physical units)

**Root Cause:** Magic attribute is not being included in damage formula OR magic users have zero weapon stats.

### Investigation Steps

1. **Locate damage calculation function**
   - Search for: `CalculateDamage`, `ApplyDamage`, or similar
   - Check: `combat/` package, likely in `combat_system.go` or `damage.go`

2. **Verify damage formula**
   ```go
   // Expected formula
   baseDamage = attacker.Weapon + attacker.Strength + attacker.Magic

   // If current formula is missing Magic:
   baseDamage = attacker.Weapon + attacker.Strength  // ❌ Wrong
   ```

3. **Check magic user weapon stats**
   - Search for unit templates: `Wizard`, `Sorcerer`, `Mage`
   - Verify they have Weapon stat >0 (should be 5-8 minimum)

### Implementation

**Option A: Fix Damage Formula (if Magic is missing)**

```go
// combat/damage.go
func CalculateDamage(attacker, defender *UnitData) int {
    // Add Magic to damage calculation
    baseDamage := attacker.Weapon + attacker.Strength + attacker.Magic

    // Apply armor reduction
    reduction := defender.Armor + defender.Toughness
    finalDamage := baseDamage - reduction

    return max(1, finalDamage) // Minimum 1 damage
}
```

**Option B: Add Weapon Stats to Magic Users (if Weapon=0)**

```go
// units/templates.go or similar
Wizard: {
    Weapon: 5 → 12,  // Add base weapon damage
    Magic: 15,       // Keep high magic
    // ... other stats
}
```

### Validation Testing

Run simulation and verify:
1. Wizard damage output: 20-30 damage range
2. Magic efficiency: 0.8-1.2 in Magic vs Physical scenario
3. Magic survival: >15% in balanced matchups
4. Combat duration: Magic vs Physical increases from 1.0 → 2.5+ rounds

### Expected Changes

**Before:**
```
Wizard: 2.7 damage dealt, 47.2 damage taken, 0.05 efficiency
Sorcerer: 2.6 damage dealt, 46.3 damage taken, 0.06 efficiency
Mage: 2.8 damage dealt, 48.1 damage taken, 0.06 efficiency
```

**After:**
```
Wizard: 24-30 damage dealt, 30-40 damage taken, 0.8-1.2 efficiency
Sorcerer: 22-28 damage dealt, 32-42 damage taken, 0.7-1.1 efficiency
Mage: 20-26 damage dealt, 34-44 damage taken, 0.7-1.0 efficiency
```

---

## Task 2: Nerf Warrior Stats

**Priority:** P0 (Critical)
**Estimated Time:** 30 minutes
**Dependencies:** None
**Validation:** Warrior efficiency 10.09 → 1.5-2.0

### Problem Statement

Warrior is catastrophically overpowered:
- Deals 82.7 damage (11x more than Assassin)
- Takes only 8.2 damage in DPS matchups
- 100% survival rate, 10.09 efficiency
- Makes all other DPS units obsolete

**Root Cause:** Warrior has either massive stat values OR a stat multiplier bug.

### Investigation Steps

1. **Locate Warrior template**
   - Search for: `Warrior` in unit definitions
   - Check stats: Weapon, Strength, Dexterity

2. **Compare to other DPS units**
   ```
   Warrior: Weapon ?, Strength ?, HP ?
   Swordsman: Weapon 13, Strength 11, HP ~45
   Rogue: Weapon 10, Strength 8, HP ~35
   ```

3. **Check for multiplier bugs**
   - Search for any special Warrior bonuses in damage calculation
   - Verify no accidental double-damage applications

### Implementation

**Stat Adjustments** (adjust values based on actual stats found):

```go
// units/templates.go
Warrior: {
    Weapon: 18 → 13,    // Reduce by ~28%
    Strength: 15 → 10,  // Reduce by ~33%
    HP: Keep current,
    Armor: Keep current,
    // ... other stats unchanged
}
```

**Verification Formula:**
```
Expected Warrior damage after nerf:
baseDamage = 13 (weapon) + 10 (strength) = 23
After crit/variance: 25-35 damage range
Target efficiency vs other DPS: 1.5-2.0
```

### Validation Testing

Run DPS vs DPS simulation:
1. Warrior efficiency: Should be 1.5-2.0 (down from 10.09)
2. Warrior survival: Should be 60-80% (down from 100%)
3. Other DPS should survive: Swordsman >40%, Assassin >20%
4. Combat duration: Should increase from 1.9 → 3.5+ rounds

### Expected Changes

**Before:**
```
Warrior: 82.7 damage, 8.2 taken, 10.09 efficiency, 100% survival
Swordsman: 21.3 damage, 19.3 taken, 1.10 efficiency, 50% survival
```

**After:**
```
Warrior: 30-35 damage, 15-20 taken, 1.8 efficiency, 75% survival
Swordsman: 21-25 damage, 18-22 taken, 1.1 efficiency, 55% survival
```

---

## Task 3: Implement Range Mechanics

**Priority:** P0 (Critical)
**Estimated Time:** 4-8 hours
**Dependencies:** None (can work in parallel with Tasks 1-2)
**Validation:** Ranged efficiency 0.12 → 0.8+

### Problem Statement

Ranged units have zero range advantage:
- Die in 1.3 rounds with 100% mortality
- Deal minimal damage (5.8-7.8) before dying
- Efficiency: 0.12-0.40 (completely non-functional)
- Range stat exists in templates but has no effect

**Root Cause:** No positional combat system. Ranged units fight in melee range.

### Implementation Options

Choose **ONE** of these three approaches:

#### Option A: Front/Back Row System (Recommended)

**Pros:** Most tactical depth, enables formations
**Cons:** Requires new position tracking system

```go
// combat/position.go
type CombatPosition struct {
    Row int // 0=front, 1=back, 2=far back
    Column int
}

// Range advantage: Back row units attack front row without retaliation
func CanRetaliate(attacker, defender *UnitData) bool {
    if attacker.Range > 0 && defender.Range == 0 {
        // Ranged attacking melee from back row
        if attacker.Position.Row > defender.Position.Row {
            return false // Melee cannot retaliate
        }
    }
    return true
}
```

**Implementation Steps:**
1. Add `Position CombatPosition` to unit combat data
2. Modify attack logic to check `CanRetaliate()`
3. Force ranged units to back row (Row 1-2)
4. Melee must move forward to attack back row (costs action)

#### Option B: Range Stat Bonus (Simpler)

**Pros:** Easy to implement, no position system needed
**Cons:** Less tactical depth

```go
// combat/damage.go
func ApplyRangeBonus(attacker, defender *UnitData, damage int) int {
    if attacker.Range > 0 && defender.Range == 0 {
        damage = int(float64(damage) * 1.5) // +50% damage
    }
    return damage
}

// In combat resolution, first strike advantage
if attacker.Range > defender.Range {
    // Attacker strikes first, defender cannot counter this round
    defendingUnit.CanActThisRound = false
}
```

#### Option C: HP Compensation (Fallback)

**Pros:** No system changes needed
**Cons:** Doesn't feel like ranged advantage

```go
// units/templates.go - Just buff HP
Archer:       HP: 30 → 55
Crossbowman:  HP: 32 → 60
Marksman:     HP: 28 → 50
```

### Recommended: Option A (Front/Back Row)

**Files to Modify:**
- `combat/combat_state.go` - Add position tracking
- `combat/attack_resolution.go` - Add retaliation checks
- `squads/formation.go` - Enforce ranged in back row
- Unit templates - Set default positions by role

**Testing:**
1. Verify ranged units start in Row 1-2
2. Verify melee units cannot immediately attack back row
3. Ranged efficiency should reach 0.8-1.2
4. Combat duration: 3-5 rounds (up from 1.3)

### Expected Changes

**Before (No Range Mechanics):**
```
Archer: 5.8 damage, 46.2 taken, 0.12 efficiency, 0% survival, 1.3 rounds
Crossbowman: 7.8 damage, 19.5 taken, 0.40 efficiency, 0% survival
```

**After (With Range Mechanics):**
```
Archer: 25-30 damage, 28-35 taken, 0.9 efficiency, 40% survival, 4.0 rounds
Crossbowman: 28-32 damage, 25-30 taken, 1.1 efficiency, 50% survival
```

---

## Task 4: Debug Cover System

**Priority:** P0 (Critical)
**Estimated Time:** 2-4 hours
**Dependencies:** None
**Validation:** Cover activation 0% → 30%+

### Problem Statement

Cover system is completely inactive:
- 0% activation across 1,400 simulations
- Defensive mechanic completely unused
- Reduces tactical depth

**Root Cause:** Unknown - requires investigation.

### Investigation Steps

1. **Locate cover calculation**
   ```bash
   # Search for cover-related functions
   grep -r "Cover" combat/
   grep -r "CalculateCover" .
   ```

2. **Check if cover is called**
   ```go
   // Add debug logging
   fmt.Printf("Cover check for %s: conditions met? %v\n",
              unit.Name, coverConditions)
   ```

3. **Verify cover conditions**
   - Check if units have `CoverValue` stat in templates
   - Check if position logic exists (unit behind ally)
   - Check if cover calculation happens before damage

4. **Common bugs to check:**
   - Cover function exists but never called
   - CoverValue = 0 in all templates
   - Position check always fails (no allies in front)
   - Cover applied AFTER damage instead of before

### Implementation

**Likely Fix: Ensure cover is calculated and applied**

```go
// combat/damage.go
func ApplyDamage(attacker, defender *UnitData, manager *EntityManager) int {
    baseDamage := CalculateDamage(attacker, defender)

    // Calculate cover BEFORE applying damage
    coverReduction := CalculateCoverReduction(defender, manager)

    finalDamage := baseDamage - coverReduction
    return max(1, finalDamage)
}

func CalculateCoverReduction(defender *UnitData, manager *EntityManager) int {
    // Check if ally is in front row providing cover
    if HasAllyInFront(defender, manager) && defender.CoverValue > 0 {
        return int(float64(baseDamage) * defender.CoverValue)
    }
    return 0
}
```

**Add Cover Stats to Templates:**

```go
// units/templates.go
Knight:  CoverValue: 0 → 0.20  // 20% damage reduction
Fighter: CoverValue: 0 → 0.15  // 15% damage reduction
Paladin: CoverValue: 0 → 0.25  // 25% damage reduction (best tank)
```

### Validation Testing

After fixes, verify:
1. Cover activation rate: 25-35% of attacks
2. Damage reduction visible in logs
3. Tank survival rate increases by ~10-15%
4. Cover activates more for back-row units

**Add logging for testing:**
```go
if coverApplied {
    fmt.Printf("[Cover] %s protected by ally: %d damage reduced\n",
               defender.Name, coverReduction)
}
```

### Expected Changes

**Before:**
```
Cover activations: 0/1400 (0%)
Tank survival without cover: 75%
```

**After:**
```
Cover activations: 420/1400 (30%)
Tank survival with cover: 85-90%
Back-row units take 15-25% less damage
```

---

## Implementation Order

Execute tasks in this sequence to minimize conflicts:

### Day 1 (4-5 hours)
1. **Task 1: Fix Magic** (2 hours)
   - Investigate damage formula
   - Apply fix (formula or stats)
   - Run quick validation test

2. **Task 2: Nerf Warrior** (30 min)
   - Locate template
   - Reduce stats
   - Quick validation test

3. **Task 4: Debug Cover** (2 hours)
   - Investigate cover system
   - Apply fix
   - Add debug logging

### Day 2-3 (4-8 hours)
4. **Task 3: Range Mechanics** (4-8 hours)
   - Choose implementation approach (front/back row recommended)
   - Implement position system
   - Add retaliation logic
   - Update unit templates
   - Comprehensive testing

### Day 4 (2-3 hours)
5. **Full Validation Testing**
   - Run all 7 scenarios (200 iterations each)
   - Verify exit criteria met
   - Document changes
   - Compare before/after metrics

---

## Testing & Validation

### Quick Test After Each Task

```bash
# Run single scenario to verify fix
go test ./combat/... -run TestBalanceScenario -scenario "Magic vs Physical"
```

### Full Phase 1 Validation

After all tasks complete, run comprehensive analysis:

```bash
# Run full balance report
go run combatsim_test.go -iterations 200 -all-scenarios
```

**Success Criteria Checklist:**
- [ ] Magic efficiency: 0.8-1.2 (was 0.06)
- [ ] Warrior efficiency: 1.5-2.0 (was 10.09)
- [ ] Ranged efficiency: 0.8-1.2 (was 0.12)
- [ ] Cover activation: 25-35% (was 0%)
- [ ] Combat duration: All scenarios >2.0 rounds
- [ ] No unit class <20% survival in role matchup
- [ ] All unit classes: 0.5-2.0 efficiency range

### Regression Testing

Ensure fixes don't break existing functionality:
- [ ] Tank vs Tank balance preserved (Knight ~1.6 efficiency)
- [ ] DPS vs Tank matchups still favor tanks
- [ ] Balanced Mixed scenario remains most competitive
- [ ] No new units drop below 0.3 efficiency

---

## Rollback Plan

If a fix causes unexpected issues:

### Git Safety
```bash
# Before starting Phase 1
git checkout -b phase1-critical-fixes
git commit -m "Phase 1: Starting point"

# After each task
git commit -m "Phase 1: Task X complete"
```

### Task-Level Rollback
```bash
# Revert specific task
git revert <commit-hash>

# Or reset to before task
git reset --hard <previous-commit>
```

### Full Rollback
```bash
# Return to pre-Phase 1 state
git checkout main
git branch -D phase1-critical-fixes
```

---

## Success Metrics

Phase 1 is **complete** when balance report shows:

| Metric | Before | Target | Status |
|--------|--------|--------|--------|
| Magic efficiency | 0.06 | 0.8-1.2 | ⏳ |
| Warrior efficiency | 10.09 | 1.5-2.0 | ⏳ |
| Ranged efficiency | 0.12 | 0.8-1.2 | ⏳ |
| Cover activation | 0% | 25-35% | ⏳ |
| Magic combat duration | 1.0 rounds | 2.5+ rounds | ⏳ |
| Ranged combat duration | 1.3 rounds | 3.5+ rounds | ⏳ |
| DPS combat duration | 1.9 rounds | 3.5+ rounds | ⏳ |

**Generate comparison report:**
```bash
go run combatsim_test.go -compare phase1-before.json phase1-after.json
```

---

## Next Steps

After Phase 1 completion:
1. Generate Phase 1 completion report
2. Review Phase 2: Balance Tuning plan
3. Prioritize remaining issues (Paladin, Assassin, Support mechanics)
4. Update project timeline

**Phase 2 Preview:**
- Buff weak units (Paladin, Assassin, Rogue, Swordsman)
- Implement support mechanics (healing, buffs)
- Add defensive mechanics (block, counter)
- Fine-tune attacker/defender balance

---

## Notes

- Keep code changes minimal and focused
- Document all stat changes in commit messages
- Run tests after each task (don't batch)
- If investigation reveals different root cause, update plan accordingly
- Magic formula fix may cascade into other calculations - watch for side effects

**Report Generated:** 2025-12-30
**Status:** Not Started
**Next Review:** After Task 1 completion
