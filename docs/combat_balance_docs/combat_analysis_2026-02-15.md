# Combat Balance Analysis — 2026-02-15

Generated from 338 simulated battles (276 duels, 24 compositions, 15 encounters, 15 stress, 8 legacy).

---

## Changes Since Baseline (2026-02-12)

Several stat and config changes were made since the last analysis. Key diffs from `monsterdata.json`:

| Unit | Change | Old → New |
|---|---|---|
| Rogue | attackType fixed | Magic → MeleeColumn |
| Rogue | weapon buffed | 7 → 9 |
| Ogre | attackType fixed | Magic → MeleeRow |
| Ogre | weapon adjusted | 12 → 9 |
| Ogre | armor adjusted | 12 → 10 |
| Sorcerer | targetCells reduced | 9 → 3 |
| Mage | targetCells reduced | 5 → 3 |
| Priest | targetCells reduced | 9 → 2 |
| Skeleton Archer | targetCells reduced, magic buffed | 9→4 cells, Magic 3→9 |
| Warrior | weapon buffed | 9 → 11 |
| Assassin | weapon buffed | 8 → 10 |
| Archer | str/weapon buffed | Str 6→11, Weapon 7→12 |
| Crossbowman | weapon buffed | 9 → 11 |
| Marksman | weapon buffed | 8 → 11 |
| Goblin Raider | weapon buffed | 5 → 10 |
| Fighter | weapon adjusted | 10 → 8 |
| Orc Warrior | weapon adjusted | 11 → 7 |
| Warlock | str/armor buffed | Str 5→9, Armor 2→4 |

These changes addressed the baseline's top recommendations: fixing Ogre/Rogue broken attack types, nerfing magic AoE, and buffing physical ranged units.

---

## Step 1: Unit Tier List

| Unit | Role | K/D | AvgDmg | Deaths | Tier | Notes |
|---|---|---|---|---|---|---|
| Fighter | Tank | 3.94 | 11.18 | 18 | S (Overtuned) | 71 kills, 18 deaths. Way above Tank target (0.5-1.5). |
| Spearman | Tank | 3.87 | 10.45 | 15 | S (Overtuned) | 58 kills, 15 deaths. Way above Tank target. |
| Warrior | DPS | 2.51 | 17.32 | 37 | A | Within DPS target (1.5-3.0). Well balanced. |
| Assassin | DPS | 1.94 | 14.32 | 17 | A | Within DPS target. Huge improvement from 0.49 baseline. |
| Orc Warrior | Tank | 1.94 | 11.99 | 32 | A | Above Tank target. Good survivability. |
| Wizard | DPS (Magic) | 1.76 | 34.94 | 59 | A | Within DPS magic target (1.5-3.0). Highest per-hit damage. |
| Cleric | Support | 1.49 | 24.70 | 37 | A- (Overtuned) | Above Support target (0.3-1.0). Too much damage. |
| Swordsman | DPS | 1.29 | 12.64 | 56 | B | Slightly below DPS target. |
| Crossbowman | DPS (Ranged) | 1.09 | 15.85 | 43 | B | Within ranged DPS target (1.0-2.5). |
| Paladin | Support | 1.00 | 9.43 | 66 | B | At top of Support range. Acceptable. |
| Knight | Tank | 0.94 | 12.15 | 72 | B | Within Tank target. Good balance. |
| Sorcerer | DPS (Magic) | 0.89 | 34.07 | 54 | B- (Over-nerfed) | Below DPS target. targetCells nerf too aggressive. |
| Ogre | Tank | 0.85 | 16.03 | 20 | B | Within Tank target. Fixed from 0.05 baseline. |
| Battle Mage | Support | 0.84 | 7.17 | 32 | B | Within Support target. |
| Archer | DPS (Ranged) | 0.77 | 18.89 | 57 | C | Below ranged DPS target despite major buffs. |
| Goblin Raider | DPS | 0.76 | 14.88 | 46 | C | Below DPS target despite weapon buff. |
| Priest | Support | 0.75 | 22.29 | 53 | B | Within Support target. Fixed from 7.00 baseline. |
| Mage | Support | 0.68 | 31.97 | 66 | B | Within Support target. Fixed from 23.00 baseline. |
| Marksman | DPS (Ranged) | 0.54 | 14.42 | 67 | C | Below ranged target. Still weak despite buff. |
| Rogue | DPS | 0.50 | 11.41 | 38 | C | Below DPS target. Fixed from 0.00 but still weak. |
| Skeleton Archer | DPS (Magic) | 0.36 | 16.64 | 58 | C- | Below DPS target. Magic 9 too low. |
| Scout | Support | 0.34 | 6.93 | 71 | C- | Near bottom of Support range. Most deaths in roster. |
| Warlock | DPS (Magic) | 0.32 | 28.07 | 56 | F | Only 40 attacks in 338 battles. Dies before acting. |
| Ranger | Support | 0.27 | 15.55 | 45 | C- | Below Support target (0.3-1.0). |

### Baseline Comparison

| Unit | Baseline K/D | New K/D | Delta | Direction |
|---|---|---|---|---|
| Sorcerer | 67.50 | 0.89 | -66.61 | Fixed (over-corrected) |
| Mage | 23.00 | 0.68 | -22.32 | Fixed |
| Priest | 7.00 | 0.75 | -6.25 | Fixed |
| Ogre | 0.05 | 0.85 | +0.80 | Fixed |
| Rogue | 0.00 | 0.50 | +0.50 | Improved (still low) |
| Assassin | 0.49 | 1.94 | +1.45 | Fixed |
| Warrior | 1.26 | 2.51 | +1.25 | Improved |
| Archer | 0.26 | 0.77 | +0.51 | Improved (still low) |
| Goblin Raider | 0.27 | 0.76 | +0.49 | Improved (still low) |
| Marksman | 0.30 | 0.54 | +0.24 | Improved (still low) |
| Warlock | 0.09 | 0.32 | +0.23 | Improved (still F tier) |
| Fighter | 2.46 | 3.94 | +1.48 | Worsened (more overtuned) |
| Spearman | 3.36 | 3.87 | +0.51 | Worsened (still overtuned) |

**Summary:** The three broken-OP magic units (Sorcerer/Mage/Priest) are now in reasonable ranges. The two broken units (Ogre/Rogue) are functional. Physical DPS got meaningful buffs. However, Fighter/Spearman became even more dominant, Sorcerer was over-nerfed, and several DPS units remain below target.

---

## Step 2: Role Balance Check

### Tanks (Target K/D: 0.5-1.5)

| Unit | K/D | Target Met? | Issue |
|---|---|---|---|
| Knight | 0.94 | Yes | Well balanced. |
| Ogre | 0.85 | Yes | Fixed. Good tank with high survivability. |
| Fighter | 3.94 | No (2.6x over) | Overperforming. MeleeColumn + decent stats = DPS-level output. |
| Spearman | 3.87 | No (2.6x over) | Same issue. MeleeColumn + range 2 + only 15 deaths. |
| Orc Warrior | 1.94 | No (1.3x over) | Slightly above target. Weapon nerf (11→7) helped but not enough. |

**Flag:** Fighter and Spearman are functioning as DPS, not Tanks. Their MeleeColumn attack pattern hitting multiple targets combined with decent damage and low death counts makes them the top performers in the roster.

### DPS Melee (Target K/D: 1.5-3.0)

| Unit | K/D | Target Met? |
|---|---|---|
| Warrior | 2.51 | Yes |
| Assassin | 1.94 | Yes |
| Swordsman | 1.29 | No (below) |
| Goblin Raider | 0.76 | No (far below) |
| Rogue | 0.50 | No (far below) |

**Flag:** Goblin Raider and Rogue underperform significantly. Swordsman is borderline. Warrior and Assassin are well balanced after buffs.

### DPS Ranged (Target K/D: 1.0-2.5)

| Unit | K/D | Target Met? |
|---|---|---|
| Crossbowman | 1.09 | Yes |
| Archer | 0.77 | No (below) |
| Marksman | 0.54 | No (far below) |

**Flag:** Despite major weapon buffs, Archer and Marksman still underperform. Physical resist on defenders absorbs too much ranged damage. Crossbowman is the only ranged unit meeting targets.

### DPS Magic (Target K/D: 1.5-3.0)

| Unit | K/D | Target Met? |
|---|---|---|
| Wizard | 1.76 | Yes |
| Sorcerer | 0.89 | No (over-nerfed) |
| Skeleton Archer | 0.36 | No (far below) |
| Warlock | 0.32 | No (broken) |

**Flag:** Only Wizard meets the magic DPS target. Sorcerer went from 67.50 to 0.89 — the targetCells nerf from 9 to 3 was too aggressive. Warlock barely attacks (40 total in 338 battles), dying before it can contribute. Skeleton Archer has low Magic (9) and only 4 targetCells.

### Support (Target K/D: 0.3-1.0)

| Unit | K/D | Target Met? |
|---|---|---|
| Paladin | 1.00 | Borderline (at ceiling) |
| Battle Mage | 0.84 | Yes |
| Priest | 0.75 | Yes |
| Mage | 0.68 | Yes |
| Scout | 0.34 | Yes (barely) |
| Ranger | 0.27 | No (below) |
| Cleric | 1.49 | No (above) |

**Flag:** Cleric (1.49) still overperforms for Support with 6 targetCells and Magic 12. Ranger (0.27) underperforms. Scout (0.34) is marginally within range but dies the most of any unit (71 deaths).

---

## Step 3: Damage Formula Audit

### Physical Damage (Str/2 + Weapon*2)

| Unit | Str | Weapon | Base Dmg | Typical Net (vs 15 resist) |
|---|---|---|---|---|
| Archer | 11 | 12 | 29 | 14 |
| Warrior | 12 | 11 | 28 | 13 |
| Ogre | 18 | 9 | 27 | 12 |
| Crossbowman | 10 | 11 | 27 | 12 |
| Marksman | 7 | 11 | 25 | 10 |
| Swordsman | 12 | 9 | 24 | 9 |
| Assassin | 9 | 10 | 24 | 9 |
| Goblin Raider | 8 | 10 | 24 | 9 |
| Knight | 14 | 8 | 23 | 8 |
| Fighter | 15 | 8 | 23 | 8 |
| Rogue | 8 | 9 | 22 | 7 |
| Spearman | 13 | 8 | 22 | 7 |
| Orc Warrior | 17 | 7 | 22 | 7 |
| Paladin | 13 | 7 | 20 | 5 |
| Battle Mage | 11 | 6 | 17 | 2 |
| Scout | 7 | 6 | 15 | min 1 |

### Magic Damage (Magic*3)

| Unit | Magic | Base Dmg | Typical Net (vs 0 resist) | targetCells | Total per Attack |
|---|---|---|---|---|---|
| Wizard | 15 | 45 | 45 | 6 | 270 |
| Sorcerer | 14 | 42 | 42 | 3 | 126 |
| Mage | 13 | 39 | 39 | 3 | 117 |
| Warlock | 13 | 39 | 39 | 4 | 156 |
| Cleric | 12 | 36 | 36 | 6 | 216 |
| Priest | 11 | 33 | 33 | 2 | 66 |
| Skeleton Archer | 9 | 27 | 27 | 4 | 108 |
| Ranger | 8 | 24 | 24 | 3 | 72 |

### Physical Resistance (Str/4 + Armor*2)

| Unit | Resist | HP (20+Str*2) |
|---|---|---|
| Ogre | 24 | 56 |
| Knight | 23 | 48 |
| Orc Warrior | 20 | 54 |
| Fighter | 19 | 50 |
| Paladin | 18 | 46 |
| Warrior | 17 | 44 |
| Spearman | 15 | 46 |
| Battle Mage | 12 | 42 |
| Crossbowman | 12 | 40 |
| Swordsman | 12 | 44 |
| Warlock | 10 | 38 |
| Cleric | 10 | 36 |
| Mage | 10 | 44 |
| Goblin Raider | 8 | 36 |
| Ranger | 8 | 36 |
| Rogue | 7 | 36 |
| Scout | 7 | 34 |
| Priest | 7 | 32 |
| Archer | 5 | 42 |
| Marksman | 5 | 34 |
| Skeleton Archer | 3 | 34 |
| Assassin | 4 | 38 |

### Key Findings

**Magic is still fundamentally stronger per-hit:**
- Magic (45 base) vs physical unit (0 resist) = 45 net damage
- Physical (29 base) vs armored unit (23 resist) = 6 net damage
- Physical (29 base) vs lightly armored (5 resist) = 24 net damage

The AoE reductions have narrowed the *total output gap* significantly:
- Wizard total per action: ~270 (6 cells × 45 each, vs 0 resist)
- Sorcerer total per action: ~126 (3 cells × 42, down from 378 with 9 cells)
- MeleeColumn attacker like Fighter: ~23 × 1-3 targets ≈ 23-69

**Remaining imbalance:** Most defenders (16 of 24 units) have Magic=0, giving 0 magic resist. Physical resist ranges from 3-24. This asymmetry means magic damage almost always lands at full value while physical damage is heavily resisted by tanks.

---

## Step 4: Rate Sanity Check

| Unit | Dex | Theo Hit | Obs Hit | Theo Crit | Obs Crit | Theo Dodge | Obs Dodge |
|---|---|---|---|---|---|---|---|
| Assassin | 60 | 100% | 89.8% | 30% | 26.4% | 20% | 16.1% |
| Rogue | 55 | 100% | 89.7% | 27% | 27.2% | 18% | 18.6% |
| Marksman | 52 | 100% | 89.7% | 26% | 22.4% | 17% | 15.1% |
| Scout | 50 | 100% | 92.6% | 25% | 23.3% | 16% | 13.7% |
| Ranger | 48 | 100% | 87.5% | 24% | 24.0% | 16% | 16.8% |
| Archer | 45 | 100% | 92.9% | 22% | 20.1% | 15% | 9.8% |
| Goblin Raider | 42 | 100% | 91.7% | 21% | 18.8% | 14% | 16.2% |
| Swordsman | 40 | 100% | 90.9% | 20% | 19.9% | 13% | 9.7% |
| Crossbowman | 38 | 100% | 89.8% | 19% | 19.3% | 12% | 11.6% |
| Skeleton Archer | 35 | 100% | 87.8% | 17% | 13.5% | 11% | 7.6% |
| Spearman | 32 | 94% | 89.0% | 16% | 16.2% | 10% | 7.5% |
| Fighter | 30 | 100% | 90.4% | 15% | 13.4% | 10% | 8.2% |
| Warrior | 28 | 100% | 91.7% | 14% | 14.0% | 9% | 6.2% |
| Battle Mage | 28 | 100% | 92.0% | 14% | 11.4% | 9% | 11.6% |
| Warlock | 26 | 100% | 95.0% | 13% | 7.5% | 8% | 8.2% |
| Knight | 25 | 100% | 89.6% | 12% | 10.7% | 8% | 7.8% |
| Cleric | 24 | 100% | 94.4% | 12% | 11.2% | 8% | 9.8% |
| Orc Warrior | 24 | 100% | 89.3% | 12% | 12.1% | 8% | 10.2% |
| Sorcerer | 22 | 100% | 92.5% | 11% | 6.5% | 7% | 8.3% |
| Paladin | 22 | 100% | 90.0% | 11% | 7.4% | 7% | 7.7% |
| Wizard | 20 | 100% | 91.5% | 10% | 9.0% | 6% | 6.6% |
| Priest | 20 | 100% | 85.0% | 10% | 12.8% | 6% | 6.2% |
| Mage | 18 | 100% | 91.3% | 9% | 13.0% | 6% | 5.0% |
| Ogre | 15 | 100% | 89.9% | 7% | 2.5% | 5% | 6.2% |

**Analysis:** All units with Dex ≥ 10 should have 100% theoretical hit rate (80 + 20 = 100), but observed rates cluster at 85-95%. This is explained by **counterattack penalties** (-20% hit) being mixed into aggregate stats. The 5-15% gap reflects how often each unit performs counterattacks vs initiated attacks.

Crit and dodge rates are generally within ±5pp of theoretical, consistent with expected statistical variance at these sample sizes. No formula bugs detected.

**Minor anomalies:**
- Ogre: 2.5% observed crit vs 7% theoretical (only 79 attacks — small sample)
- Mage: 13.0% observed crit vs 9% theoretical (92 attacks — moderate sample, possibly high-dex targets dodging hits more, inflating crit ratio on the hits that land)

---

## Step 5: Matchup Outliers

### Extreme Damage (AvgDmg > 30, ≥ 5 attacks)

| Attacker | Defender | AvgDmg | Attacks | Battles | Notes |
|---|---|---|---|---|---|
| Wizard vs Marksman | | 40.00 | 7 | 2 | Magic 45 vs 0 resist |
| Wizard vs Scout | | 38.00 | 6 | 1 | Magic 45 vs 0 resist |
| Wizard vs Swordsman | | 37.27 | 22 | 11 | High sample, confirmed |
| Wizard vs Crossbowman | | 34.20 | 15 | 5 | Confirmed |
| Wizard vs Knight | | 32.22 | 37 | 19 | Highest sample, confirmed |
| Wizard vs Warrior | | 32.95 | 20 | 9 | Confirmed |
| Wizard vs Wizard | | 32.89 | 9 | 7 | Confirmed |
| Wizard vs Fighter | | 31.90 | 10 | 6 | Confirmed |
| Wizard vs Archer | | 30.52 | 27 | 12 | Confirmed |
| Wizard vs Mage | | 30.09 | 11 | 3 | Confirmed |
| Sorcerer vs Swordsman | | 32.86 | 14 | 6 | Confirmed |
| Sorcerer vs Orc Warrior | | 33.80 | 5 | 2 | |
| Sorcerer vs Knight | | 30.31 | 26 | 11 | High sample, confirmed |
| Cleric vs Warrior | | 36.14 | 7 | 2 | Magic 36 vs 0 resist |
| Cleric vs Crossbowman | | 31.00 | 6 | 1 | |
| Warlock vs Knight | | 32.18 | 11 | 8 | Magic 39 vs 0 resist |

Pattern: **All high-damage matchups are magic attackers vs physical-only defenders** (0 magic resist). This is consistent and expected given the formula asymmetry.

### Near-Zero Damage Matchups (< 2.0 avg, ≥ 10 attacks)

| Attacker | Defender | AvgDmg | Attacks | Notes |
|---|---|---|---|---|
| Battle Mage vs Orc Warrior | | 0.86 | 36 | Physical 17 vs 20 resist = min 1 |
| Scout vs Knight | | 0.83 | 6 | Physical 15 vs 23 resist = min 1 |
| Scout vs Orc Warrior | | 0.67 | 6 | Physical 15 vs 20 resist = min 1 |
| Paladin vs Ogre | | 1.04 | 54 | Physical 20 vs 24 resist = min 1 |
| Battle Mage vs Knight | | 1.17 | 24 | Physical 17 vs 23 resist = min 1 |
| Ogre vs Scout | | 1.19 | 21 | Physical 27 vs 7 resist... 20 expected? |
| Scout vs Warrior | | 1.20 | 10 | Physical 15 vs 17 resist = min 1 |
| Paladin vs Knight | | 1.74 | 74 | Physical 20 vs 23 resist = min 1 |

**Anomaly: Ogre vs Scout** averages only 1.19 damage despite Ogre having 27 base physical and Scout having 7 resist (expected ~20 damage). This warrants investigation — Ogre may have targeting issues or the attacks counted are mostly counterattacks (which deal 50% damage).

### Warlock Activity Problem

Warlock made only 40 attacks across 338 battles, with only 12 unique matchups. Most matchups have just 1-2 attacks:
- Warlock vs Knight: 11 attacks (biggest matchup)
- Warlock vs Orc Warrior: 23 attacks
- All others: 1-2 attacks each

This means Warlock dies very quickly in almost every battle, getting at most 1-2 attacks before being killed. With HP 38 and Armor 4 (resist 10), it's one of the most fragile magic users.

---

## Step 6: Balance Alert Triage

### Alert Summary

| AlertType | Count | Severity |
|---|---|---|
| SkewedKD | 0 | Critical |
| HighDamage | 25 | High |
| HighCrit | 3 | Medium |
| PerfectHitRate | 66 | Low |

**No SkewedKD alerts** — a major improvement from baseline (which had 3). No unit exceeds K/D 5.0.

### High Priority — HighDamage (25 alerts)

Most-flagged attackers:
- **Wizard:** 12 alerts (Wizard vs Archer, Crossbowman, Fighter, Knight, Mage, Marksman, Orc Warrior, Rogue, Scout, Skeleton Archer, Spearman, Swordsman, Warrior, Wizard)
- **Sorcerer:** 4 alerts (vs Archer, Knight, Orc Warrior, Swordsman)
- **Warlock:** 1 alert (vs Knight)
- **Cleric:** 2 alerts (vs Crossbowman, Warrior)
- **Mage:** 2 alerts (vs Fighter, Scout)
- **Priest:** 1 alert (vs Swordsman)

All 25 HighDamage alerts involve magic attackers hitting low/zero magic-resist defenders. This is structural, not a bug.

### Medium Priority — HighCrit (3 alerts)

- Archer vs Skeleton Archer: 66.7% (6 attacks) — small sample artifact
- Goblin Raider vs Sorcerer: 53.8% (13 attacks) — moderate sample but still plausibly variance (theoretical 21%)
- Ranger vs Fighter: 66.7% (6 attacks) — small sample artifact

### Low Priority — PerfectHitRate (66 alerts)

Up from 46 in baseline, but most have < 15 attacks. Expected with 547 matchup pairs — some will naturally have 100% hit rate at small sample sizes.

---

## Step 7: Stat Correlation

### What Predicts Winning?

**Magic stat remains the strongest per-hit predictor** — the top 4 AvgDmgDealt are all magic users (Wizard 34.94, Sorcerer 34.07, Mage 31.97, Warlock 28.07). However, the AoE reductions have broken the link between magic and K/D dominance. Wizard (K/D 1.76) is now balanced because it dies enough to offset its damage.

**MeleeColumn attack type is the strongest K/D predictor** — Fighter (3.94), Spearman (3.87), Assassin (1.94), and Swordsman (1.29) all use MeleeColumn. Column attacks hit all units in one column (1-3 targets), giving multi-target damage with physical stats that also provide survivability.

**Weapon stat is the primary physical damage driver** — it contributes Weapon×2 (vs Strength/2). Buffing Weapon from 7→12 for Archer gave +10 base damage, explaining most of its improvement.

**Armor/Strength dominate survivability** — physical resist (Str/4 + Armor×2) determines whether a unit can survive physical attacks. This creates the "tank wall" where units like Knight (23 resist) reduce most physical attacks to minimum damage.

**Dexterity provides diminishing returns** — hit rate caps at 100% (Dex ≥ 10), crit and dodge provide marginal benefits. The weapon buffs (Assassin 8→10, Rogue 7→9) had more impact on K/D than any amount of Dex.

**Leadership has no combat effect** — confirmed. High-leadership units (Paladin 60, Knight 50, Skeleton Archer 50) get no combat benefit from the stat.

### New Insight: The "Physical Damage Wall"

Many physical matchups produce near-minimum damage (1-3 per hit). For example:
- Paladin (20 base) vs Knight (23 resist) = 1 damage
- Battle Mage (17 base) vs Orc Warrior (20 resist) = 1 damage
- Scout (15 base) vs Knight (23 resist) = 1 damage

This creates a binary outcome: physical attackers either deal meaningful damage (if base > resist) or deal nothing (if base ≤ resist). Magic completely bypasses this wall because 16/24 units have 0 magic resist.

---

## Step 8: Recommendations

### Priority 1: Fix Fighter and Spearman (S Tier Tanks)

**Problem:** Both are Tanks with K/D 3.87-3.94, performing as the top DPS in the roster.

**Root cause:** MeleeColumn hits all units in a column (up to 3 targets) while they have good survivability. They effectively have AoE physical damage + tank stats.

**Option A — Reduce damage output:**
- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Fighter weapon 8→6, Spearman weapon 8→6
- **Impact:** Fighter base damage 23→19, Spearman 22→18. Reduces net damage by ~4 per target, which is significant against moderate-armor defenders.
- **Expected K/D:** ~2.0-2.5 (still above Tank target but much closer)

**Option B — Change attack type:**
- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Switch one or both to MeleeRow instead of MeleeColumn
- **Impact:** MeleeRow hits front row (typically 1-3 units in row 0 only), while MeleeColumn pierces through all rows. This would reduce multi-target damage significantly.

**Option C — Reclassify as DPS:**
- If the design intent is for these units to deal high damage, reclassify them as DPS and reduce their armor/cover to match.

**Recommended:** Option A first — it's the smallest change with measurable impact.

### Priority 2: Un-nerf Sorcerer (Over-corrected)

**Problem:** K/D dropped from 67.50 to 0.89. targetCells went from 9 (entire grid) to 3 — too aggressive.

**File:** `assets/gamedata/monsterdata.json`
**Change:** Increase Sorcerer targetCells from 3 to 5:
```json
"targetCells": [[0, 0], [0, 1], [0, 2], [1, 0], [1, 2]]
```
This hits front row + flanks of middle row (5 cells instead of 3).

**Expected impact:** Roughly 1.7x current damage output, projecting K/D from 0.89 to ~1.5-2.0, within DPS magic target.

### Priority 3: Fix Warlock (40 attacks in 338 battles)

**Problem:** Dies before it can act. HP 38, resist 10. Only 40 attacks, 56 deaths.

**File:** `assets/gamedata/monsterdata.json`
**Option A — Increase survivability:**
- Strength 9→13 (HP 38→46, physical resist 10→11)
- Armor 4→6 (physical resist 10→14)

**Option B — Increase magic to compensate for early death:**
- Magic 13→16 (base damage 39→48, hitting harder in fewer attacks)

**Recommended:** Option A — Warlock needs to survive long enough to use its 4-cell AoE. Survivability is the bottleneck, not damage.

### Priority 4: Buff Remaining Weak Physical DPS

**Rogue (K/D 0.50, target 1.5-3.0):**
- Base damage 22 hits into 15-24 resist, netting 0-7 damage against tanks
- **File:** `assets/gamedata/monsterdata.json`
- **Change:** Weapon 9→12 (base damage 22→28)
- **Impact:** Net damage vs medium armor (15 resist) goes from 7 to 13

**Goblin Raider (K/D 0.76, target 1.5-3.0):**
- Base damage 24, HP 36, resist 8. Too fragile.
- **Change:** Strength 8→11 (HP 36→42, resist 8→10, base damage 24→25)
- **Impact:** +17% HP and slightly more damage

**Swordsman (K/D 1.29, target 1.5-3.0):**
- Borderline. Base damage 24 is adequate but could use a small bump.
- **Change:** Weapon 9→10 (base damage 24→26)
- **Impact:** +2 net damage per target, should push K/D to ~1.5

### Priority 5: Address Ranged DPS Underperformance

**Problem:** Archer (0.77) and Marksman (0.54) are below target despite major weapon buffs. The fundamental issue is physical resist — their damage is absorbed by armored targets.

**Short-term fix:**
- **File:** `assets/gamedata/monsterdata.json`
- Archer: Weapon 12→13, Strength 11→12 (base damage 29→30, HP 42→44)
- Marksman: Weapon 11→13 (base damage 25→29), Armor 2→3 (resist 5→7)
- This provides marginal improvement but doesn't solve the structural issue.

**Structural fix (medium-term):** Consider an armor penetration mechanic for Ranged attacks:
- **File:** `config/config.go` or damage calculation code
- Add: `RangedArmorPenetration = 0.30` — ranged attacks ignore 30% of physical resist
- **Impact:** Archer (29 base) vs Knight (23 resist → 16 effective resist) = 13 damage instead of 6. This would make ranged units competitive against armored targets without affecting magic balance.

### Priority 6: Reduce Cleric Overperformance

**Problem:** K/D 1.49 with Support role (target 0.3-1.0). Has 6 targetCells and Magic 12.

**File:** `assets/gamedata/monsterdata.json`
**Change:** Reduce targetCells from 6 to 4:
```json
"targetCells": [[1, 0], [1, 1], [1, 2], [2, 1]]
```
This hits middle row + center back row (4 cells instead of 6).

**Impact:** ~33% reduction in AoE, projecting K/D from 1.49 to ~1.0, near Support ceiling.

### Priority 7: Improve Skeleton Archer and Ranger

**Skeleton Archer (K/D 0.36, DPS target 1.5-3.0):**
- Magic 9 gives 27 base damage. With 4 targetCells, total damage per action is ~108.
- **Change:** Magic 9→12 (base damage 27→36, total per action 108→144)
- **Impact:** ~33% damage increase, projecting K/D toward 0.6-0.8. May need further targetCells increase.

**Ranger (K/D 0.27, Support target 0.3-1.0):**
- Magic 8 gives 24 base damage with 3 targetCells. Fragile (HP 36, resist 8).
- **Change:** Strength 8→10 (HP 36→40, resist 8→10)
- **Impact:** ~11% HP increase and slightly better resist. Should push K/D above 0.3.

---

## Summary of All Recommended Changes

| Unit | Stat | Old | New | Priority |
|---|---|---|---|---|
| Fighter | weapon | 8 | 6 | 1 |
| Spearman | weapon | 8 | 6 | 1 |
| Sorcerer | targetCells | 3 | 5 | 2 |
| Warlock | strength | 9 | 13 | 3 |
| Warlock | armor | 4 | 6 | 3 |
| Rogue | weapon | 9 | 12 | 4 |
| Goblin Raider | strength | 8 | 11 | 4 |
| Swordsman | weapon | 9 | 10 | 4 |
| Cleric | targetCells | 6 | 4 | 6 |
| Skeleton Archer | magic | 9 | 12 | 7 |
| Ranger | strength | 8 | 10 | 7 |

### Structural Issue for Future Consideration

The **magic vs physical resist asymmetry** remains the core balance challenge:
- 16/24 units have Magic=0 → 0 magic resistance
- Physical resist ranges from 3-24

Consider adding a `BaseMagicResist` constant (e.g., 5) in `config/config.go` that all units receive regardless of Magic stat. This would:
- Reduce magic damage by ~5 per hit across the board
- Narrow the gap between magic and physical damage
- Allow further tuning of individual unit stats
