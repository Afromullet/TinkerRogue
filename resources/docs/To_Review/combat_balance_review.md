# Combat Balance Recommendations — 2026-03-12

Based on 495 battles, 501 scenarios, 30 unit types.

---

## Priority 1: Fix Overtuned Tanks

### 1a. Nerf Lancer (K/D 3.75 — target 0.5-1.5)

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Reduce Lancer weapon from 9 to 7, or reduce Dex from 35 to 28
- **Impact:** Base damage drops from 25 to 22, bringing K/D closer to 2.0-2.5. Further reduction may be needed.

### 1b. Nerf Cataphract (K/D 3.57 — target 0.5-1.5)

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Reduce Cataphract weapon from 8 to 6
- **Impact:** Base damage drops from 24 to 20. High armor (9) keeps it tanky but with less kill pressure.

### 1c. Nerf Fighter (K/D 2.84 — target 0.5-1.5)

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Reduce Fighter weapon from 8 to 6
- **Impact:** Base damage drops from 23 to 19, bringing K/D toward ~1.5-2.0.

---

## Priority 2: Buff Underperforming Magic DPS

### 2a. Buff Sorcerer (K/D 0.84 — target 1.5-3.0)

The targetCells nerf from 9 to 3 overshot. Sorcerer went from 67.50 to 0.84.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Sorcerer targetCells from 3 to 5, or increase armor from 3 to 5 for survivability
- **Impact:** More targets per attack should raise K/D toward 1.5-2.0 without returning to the broken 67.50 state.

### 2b. Buff Mage (K/D 0.65 — target 1.5-3.0)

Same issue — targetCells reduction overshot.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Mage targetCells from 3 to 4-5
- **Impact:** Should bring K/D toward 1.0-1.5.

### 2c. Fix Warlock (K/D 0.14 — target 1.5-3.0)

Only 32 attacks across all battles. Dies before contributing.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Warlock strength from 9 to 12 (HP 38->44), armor from 4 to 6 (resist +4)
- **Impact:** More HP/armor = more attacks made = higher K/D. Per-hit damage (24.03) is already good.

---

## Priority 3: Buff Weak Physical DPS

### 3a. Buff Scout (K/D 0.33, AvgDmg 7.25)

Extremely low damage for DPS.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Scout weapon from 6 to 9
- **Impact:** Damage goes from 15 to 21. Should roughly double effective output.

### 3b. Buff Outrider (K/D 0.56, AvgDmg 9.45)

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Outrider weapon from 7 to 10
- **Impact:** Damage goes from 18 to 24.

### 3c. Buff Battle Mage (K/D 1.05, AvgDmg 6.91)

Very low damage for DPS role. Uses MeleeColumn but has moderate stats.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Battle Mage weapon from 6 to 9
- **Impact:** Damage goes from 17 to 23.

---

## Priority 4: Address Survivability Issues

### 4a. Priest dies too much (79 deaths vs Cleric's 38)

Both are healers, but Priest has Str 6 (HP=32) vs Cleric Str 8 (HP=36). Priest also has only 2 targetCells for healing vs Cleric's 6.

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Priest strength from 6 to 9 (HP 32->38), increase coverValue from 0.25 to 0.3
- **Impact:** More survivable healer, closer to Cleric's death rate.

### 4b. Marksman dies excessively (83 deaths, K/D 0.47)

- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Increase Marksman armor from 2 to 4
- **Impact:** Resist goes from 5 to 9. Should reduce deaths by ~15-20%.

---

## Priority 5: Monitor / Low Priority

### 5a. Ranger (K/D 0.31)

Has Magic 8 with Magic attackType and only 3 targetCells. Consider increasing Magic to 10 or targetCells to 4.

### 5b. Skeleton Archer (K/D 0.50)

Magic 9 gives 27 base damage but Str 7, Armor 1 makes it extremely fragile. Consider armor 1->3.

### 5c. Goblin Raider (K/D 0.80)

Improved from 0.27 baseline. Monitor after other changes before further tuning.

### 5d. Rogue (K/D 0.51)

Fixed from 0.00 baseline (attack type was broken). Now functional but still below DPS target. Consider weapon 9->11 or slight Str buff.

### 5e. Assassin (K/D 1.27)

Below DPS target (1.5-3.0) despite 60 Dex. High Dex gives crit/dodge but no damage bonus. Consider weapon 10->12.

---

## Baseline Comparison

| Unit | Baseline K/D | Current K/D | Target K/D | Status |
|---|---|---|---|---|
| Sorcerer | 67.50 | 0.84 | 1.5-3.0 | Fixed but overshot |
| Mage | 23.00 | 0.65 | 1.5-3.0 | Fixed but overshot |
| Priest | 7.00 | 0.00 (healer) | Support | Fixed |
| Rogue | 0.00 | 0.51 | 1.5-3.0 | Fixed, still weak |
| Ogre | 0.05 | 1.17 | 0.5-1.5 | Perfectly balanced |
| Warlock | 0.09 | 0.14 | 1.5-3.0 | Still broken |
| Goblin Raider | 0.27 | 0.80 | 1.5-3.0 | Improved, still below |
| Archer | 0.26 | 0.96 | 1.0-2.5 | Major improvement |
| Marksman | 0.30 | 0.47 | 1.0-2.5 | Slight improvement |
| Assassin | 0.49 | 1.27 | 1.5-3.0 | Improved, still below |
| Wizard | 1.84 | 2.54 | 1.5-3.0 | Well balanced |
| Warrior | 1.26 | 1.89 | 1.5-3.0 | Within target |
| Lancer | N/A | 3.75 | 0.5-1.5 | New, overtuned |
| Cataphract | N/A | 3.57 | 0.5-1.5 | New, overtuned |
| Dragoon | N/A | 2.35 | 1.5-3.0 | New, within target |
| Hussar | N/A | 2.83 | 1.5-3.0 | New, within target (small sample) |
| Horse Archer | N/A | 0.85 | 1.0-2.5 | New, below target |
| Outrider | N/A | 0.56 | 1.0-2.5 | New, below target |
