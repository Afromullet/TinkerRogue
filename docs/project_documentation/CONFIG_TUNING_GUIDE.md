# Configuration Tuning Guide

**Last Updated:** 2026-02-01

This document explains all values in the game's JSON configuration files and how changing them impacts gameplay algorithms.

---

## Table of Contents

1. [aiconfig.json](#aiconfigjson) - AI behavior and threat assessment
2. [powerconfig.json](#powerconfigjson) - Power calculations for units and squads
3. [encounterdata.json](#encounterdatajson) - Combat encounter generation
4. [overworldconfig.json](#overworldconfigjson) - Overworld threat and faction systems

---

## aiconfig.json

Controls AI decision-making, threat assessment, and role-based behavior.

**File Location:** `assets/gamedata/aiconfig.json`

### threatCalculation

Parameters for how AI calculates danger zones and tactical positioning.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `flankingThreatRangeBonus` | 3 | Extra range added when calculating flanking threat | **Higher**: AI considers flanking danger from further away, becomes more cautious about positioning. **Lower**: AI ignores distant flanking threats, more aggressive positioning |
| `isolationSafeDistance` | 2 | Distance to ally considered "safe" (no isolation penalty) | **Higher**: Units need to stay closer together to feel safe. **Lower**: Units comfortable spreading out more |
| `isolationModerateDistance` | 3 | Distance where isolation penalty starts increasing | **Higher**: Gradual isolation penalty kicks in later. **Lower**: Penalty applies sooner, tighter formations |
| `isolationHighDistance` | 6 | Distance where maximum isolation penalty applies | **Higher**: Full penalty only at extreme distances. **Lower**: Full penalty applies sooner, much tighter formations |
| `engagementPressureMax` | 200 | Maximum combined threat used to normalize pressure (0-1) | **Higher**: Pressure values spread over larger range, less sensitivity to individual threats. **Lower**: Pressure maxes out faster, AI reacts more strongly to any threat |
| `retreatSafeThreatThreshold` | 10 | Threat level below which an adjacent tile is considered a safe escape route | **Higher**: More tiles count as escape routes, AI feels less trapped. **Lower**: Fewer escape routes, AI more likely to feel cornered |

**Formula Context:**
```
IsolationRisk = 0.0 if distance <= safeDistance
              = interpolate(0.0, 1.0) if safeDistance < distance < highDistance
              = 1.0 if distance >= highDistance

RetreatQuality = (safe_adjacent_tiles / total_adjacent_tiles)
```

---

### roleBehaviors

Defines how different unit roles weight threat factors when choosing positions. **Negative weights mean the role is attracted to that threat type.**

| Role | Parameter | Default | Description | Impact of Change |
|------|-----------|---------|-------------|------------------|
| **Tank** | `meleeWeight` | -0.5 | How much melee threat affects position scoring | **Negative**: Tanks seek melee combat (move toward enemies). **Positive**: Tanks would avoid melee (wrong behavior) |
| | `rangedWeight` | 0.3 | How much ranged threat matters | **Higher**: Tanks avoid ranged fire more. **Lower**: Tanks ignore ranged threats |
| | `supportWeight` | 0.2 | How much support value (healing) matters | **Higher**: Tanks position near healers. **Negative**: Tanks would seek wounded allies |
| | `positionalWeight` | 0.5 | How much positional risk matters | **Higher**: Tanks care more about flanking/isolation. **Lower**: Tanks ignore tactical positioning |
| **DPS** | `meleeWeight` | 0.7 | Avoidance of melee threat | **Higher**: DPS stays far from melee enemies. **Lower**: DPS willing to get closer |
| | `rangedWeight` | 0.5 | Avoidance of ranged threat | **Higher**: DPS seeks cover from ranged. **Lower**: DPS ignores ranged fire |
| | `supportWeight` | 0.1 | Attraction to support positions | Low value means DPS doesn't prioritize being near healers |
| | `positionalWeight` | 0.6 | Importance of good positioning | **Higher**: DPS very cautious about flanking. **Lower**: DPS takes risks for damage |
| **Support** | `meleeWeight` | 1.0 | Strong avoidance of melee | Highest value - supports stay far from melee |
| | `rangedWeight` | 0.8 | Strong avoidance of ranged | High value - supports seek safety |
| | `supportWeight` | -1.0 | **Negative**: Seeks wounded allies | This makes supports move toward units that need healing |
| | `positionalWeight` | 0.4 | Moderate positioning concern | Supports care about safety but not as much as DPS |

**Formula Context:**
```
PositionScore = (meleeThreat * meleeWeight) +
                (rangedThreat * rangedWeight) +
                (supportValue * supportWeight) +
                (positionalRisk * positionalWeight)

Lower score = better position (negative weights invert attraction)
```

**Tuning Tips:**
- To make Tanks more aggressive: decrease `meleeWeight` (more negative) or decrease `positionalWeight`
- To make DPS more cautious: increase all weights
- To make Support prioritize healing over safety: make `supportWeight` more negative, decrease other weights

---

### positionalRisk

Weights for the four sub-components of positional risk calculation.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `flankingWeight` | 0.4 | Importance of being attacked from multiple directions | **Higher**: AI strongly avoids positions where enemies surround them. **Lower**: AI tolerates being flanked |
| `isolationWeight` | 0.3 | Importance of staying near allies | **Higher**: AI clusters together more. **Lower**: AI spreads out more |
| `pressureWeight` | 0.2 | Importance of total threat pressure | **Higher**: AI avoids high-threat zones. **Lower**: AI ignores accumulated danger |
| `retreatWeight` | 0.1 | Importance of having escape routes | **Higher**: AI values positions with multiple escape options. **Lower**: AI doesn't plan retreats |

**Note:** These weights should sum to 1.0 for normalized risk values.

**Formula Context:**
```
PositionalRisk = (flankingRisk * flankingWeight) +
                 (isolationRisk * isolationWeight) +
                 (engagementPressure * pressureWeight) +
                 ((1.0 - retreatQuality) * retreatWeight)
```

---

### supportLayer

Parameters for how support units evaluate healing and ally proximity.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `healRadius` | 3 | Distance at which support value radiates from wounded allies | **Higher**: Support units consider healing targets from further away. **Lower**: Must be closer to recognize healing opportunities |
| `allyProximityRadius` | 2 | Distance for ally grouping bonus | **Higher**: Larger "group" detection, supports position in center of larger formations. **Lower**: Tighter grouping required |
| `buffPriorityEngagementRange` | 4 | Range at which buff targets are considered "engaged" | **Higher**: Support buffs units further from combat. **Lower**: Only buffs units actively fighting |

**Formula Context:**
```
HealPriority = 1.0 - SquadHealthPercent  // More wounded = higher priority
SupportValue at position = sum of (HealPriority * LinearFalloff(distance, healRadius))
```

---

## powerconfig.json

Controls power calculations used for encounter balancing and AI threat assessment.

**File Location:** `assets/gamedata/powerconfig.json`

### profiles

Power calculation profiles with different emphases. Used by encounter generator to match difficulty.

| Profile | offensiveWeight | defensiveWeight | utilityWeight | Use Case |
|---------|-----------------|-----------------|---------------|----------|
| **Balanced** | 0.40 | 0.40 | 0.20 | Default - equal offense/defense |
| **Offensive** | 0.60 | 0.25 | 0.15 | Glass cannon encounters |
| **Defensive** | 0.25 | 0.60 | 0.15 | Tank-heavy encounters |

**Sub-weights within each profile:**

| Category | Parameter | Balanced | Description | Impact of Change |
|----------|-----------|----------|-------------|------------------|
| **Offensive** | `damageWeight` | 0.6 | Raw damage importance | **Higher**: High-damage units valued more. **Lower**: Accuracy matters more |
| | `accuracyWeight` | 0.4 | Hit chance importance | **Higher**: Reliable hitters valued. **Lower**: Raw damage dominates |
| **Defensive** | `healthWeight` | 0.5 | HP pool importance | **Higher**: High-HP units valued. **Lower**: Resistance/avoidance matters more |
| | `resistanceWeight` | 0.3 | Damage reduction importance | **Higher**: Armored units valued. **Lower**: HP dominates |
| | `avoidanceWeight` | 0.2 | Dodge chance importance | **Higher**: Evasive units valued. **Lower**: HP/armor dominates |
| **Utility** | `roleWeight` | 0.5 | Role multiplier importance | **Higher**: Role identity matters more |
| | `abilityWeight` | 0.3 | Special ability importance | **Higher**: Leaders/casters valued more |
| | `coverWeight` | 0.2 | Cover provision importance | **Higher**: Cover-providing units valued |

**Squad modifiers:**

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `formationBonus` | 1.0 | Multiplier for formation bonuses | **Higher**: Good formations more valuable. Currently unused (1.0) |
| `moraleMultiplier` | 0.002 | How much morale affects power | **Higher**: Morale swings have larger impact. `morale * multiplier` gives 0.8x-1.2x range |
| `healthPenalty` | 2.0 | Exponent for health-based power reduction | **Higher**: Wounded squads lose power faster. `(currentHP/maxHP)^penalty` |
| `deployedWeight` | 1.0 | Weight for deployed units in roster power | Deployed units count at full value |
| `reserveWeight` | 0.3 | Weight for reserve units in roster power | Reserve units count at 30% value |

**Formula Context:**
```
UnitPower = (OffensivePower * offensiveWeight) +
            (DefensivePower * defensiveWeight) +
            (UtilityPower * utilityWeight)

SquadPower = SumOfUnitPower * MoraleBonus * HealthPenalty * CompositionBonus
```

---

### roleMultipliers

Base multiplier applied to units based on their combat role.

| Role | Multiplier | Description | Impact of Change |
|------|------------|-------------|------------------|
| Tank | 1.2 | Moderate bonus for durability role | **Higher**: Tanks count as more powerful in encounters. **Lower**: Tanks valued less |
| DPS | 1.5 | Highest bonus for damage dealers | **Higher**: DPS units very high value. **Lower**: More DPS needed for same power |
| Support | 1.0 | Baseline (no bonus) | **Higher**: Support becomes valuable. **Lower**: Support undervalued (below 1.0) |

**Tuning Tips:**
- If encounters feel too easy with DPS-heavy squads, reduce DPS multiplier
- If tanks feel weak in power calculations, increase Tank multiplier

---

### abilityValues

Power contribution from special abilities (typically for leaders).

| Ability | Power | Description | Impact of Change |
|---------|-------|-------------|------------------|
| Rally | 15.0 | Morale boost ability | **Higher**: Rally leaders valued more in encounters |
| Heal | 20.0 | Healing ability (highest value) | **Higher**: Healers become high-priority targets/assets |
| BattleCry | 12.0 | Combat buff ability | **Higher**: Buff leaders more valuable |
| Fireball | 18.0 | AoE damage ability | **Higher**: AoE casters valued more |
| None | 0.0 | No special ability | Baseline - no contribution |

---

### compositionBonuses

Bonus multiplier based on attack type diversity within a squad.

| Unique Types | Bonus | Description | Impact of Change |
|--------------|-------|-------------|------------------|
| 1 | 0.8 | **Penalty** for mono-type squads | **Higher**: Less penalty for specialization. **Lower**: Mono-type squads weaker |
| 2 | 1.1 | Small bonus for two types | **Higher**: Rewards mixed composition more |
| 3 | 1.2 | Good bonus for three types | **Higher**: Diverse squads much stronger |
| 4 | 1.3 | Best bonus for full diversity | **Higher**: Full diversity very powerful |

**Tuning Tips:**
- To encourage diverse squads: increase bonuses for 3-4 types, decrease 1-type value
- To allow specialized squads: bring all values closer to 1.0

---

### scalingConstants

Universal scaling factors used across power calculations.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `roleScaling` | 10.0 | Divisor for role power contribution | **Higher**: Roles contribute less to total power. **Lower**: Roles dominate power calculation |
| `dodgeScaling` | 100.0 | Divisor for dodge chance contribution | **Higher**: Dodge has less impact on defensive power. **Lower**: Dodge becomes very valuable |
| `coverScaling` | 100.0 | Divisor for cover contribution | **Higher**: Cover provides less power. **Lower**: Cover-providing units very valuable |
| `coverBeneficiaryMultiplier` | 2.5 | Multiplier based on units benefiting from cover | **Higher**: More allies in cover = much more value. **Lower**: Cover value less dependent on allies |
| `leaderBonus` | 1.3 | Power multiplier for leader units | **Higher**: Leaders much more valuable/powerful. **Lower**: Leaders closer to regular units |

---

## encounterdata.json

Controls combat encounter generation, difficulty scaling, and enemy composition.

**File Location:** `assets/gamedata/encounterdata.json`

### difficultyLevels

Defines difficulty tiers for encounters.

| Level | Name | powerMultiplier | minSquads | maxSquads | Description |
|-------|------|-----------------|-----------|-----------|-------------|
| 1 | Easy | 0.7 | 2 | 3 | Enemy power = 70% of player power |
| 2 | Moderate | 0.9 | 3 | 4 | Nearly matched, slight player advantage |
| 3 | Fair | 1.0 | 3 | 5 | Equal power matchup |
| 4 | Hard | **5.0** | 4 | 6 | **Currently set very high** - enemy 5x player power |
| 5 | Boss | 1.5 | 5 | 7 | 50% stronger enemies, many squads |

**Parameter Impacts:**

| Parameter | Impact of Change |
|-----------|------------------|
| `powerMultiplier` | **Higher**: Enemies have more total power budget. **Lower**: Easier encounters. This directly multiplies player roster power to get target enemy power |
| `minSquads` | **Higher**: More enemy squads (minimum). **Lower**: Potentially fewer enemies |
| `maxSquads` | **Higher**: More enemy squads (maximum). **Lower**: Caps enemy count |

**Note:** Level 4 "Hard" currently has `powerMultiplier: 5` which seems like a typo (should probably be 1.1-1.3).

**Formula Context:**
```
TargetEnemyPower = PlayerRosterPower * powerMultiplier
NumSquads = random(minSquads, maxSquads)
PowerPerSquad = TargetEnemyPower / NumSquads
```

---

### encounterTypes

Predefined encounter compositions with tactical themes.

| ID | Name | squadPreferences | defaultDifficulty | Tactical Theme |
|----|------|------------------|-------------------|----------------|
| goblin_basic | Goblin Patrol | melee, melee, ranged | 4 | Melee-heavy with ranged support |
| bandit_basic | Bandit Ambush | melee, ranged, ranged | 4 | Ranged-focused |
| beast_basic | Beast Swarm | melee, melee, melee | 4 | Pure melee rush |
| orc_basic | Orc Warband | melee, ranged, magic | 4 | Balanced composition |

**Parameter Impacts:**

| Parameter | Impact of Change |
|-----------|------------------|
| `squadPreferences` | Determines what types of enemy squads spawn. Order matters - first preferences filled first |
| `defaultDifficulty` | Which difficulty level to use if not specified. References `difficultyLevels[level]` |
| `tags` | Used for filtering/selecting encounters. Add new tags for conditional spawning |

**Tuning Tips:**
- To create harder melee encounters: add more "melee" to squadPreferences
- To create caster-heavy encounters: add "magic" entries
- To vary difficulty: change defaultDifficulty per encounter type

---

### squadTypes

Definitions for enemy squad archetypes.

| ID | Name | Description | Used For |
|----|------|-------------|----------|
| melee | Melee Squad | Close-range combat | Front-line fighters |
| ranged | Ranged Squad | Long-range attackers | Archers, crossbowmen |
| magic | Magic Squad | Spellcasters and support | Casters, healers |

These IDs are referenced by `squadPreferences` in encounter types.

---

## overworldconfig.json

Controls the overworld threat system, faction AI, and strategic gameplay.

**File Location:** `assets/gamedata/overworldconfig.json`

### threatGrowth

Parameters for how threats spread and grow on the overworld map.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `defaultGrowthRate` | 0.05 | Base rate threats grow per tick | **Higher**: Threats spread faster, more urgency. **Lower**: Slower spread, more time to respond |
| `containmentSlowdown` | 0.5 | Growth multiplier when player nearby | **Higher (closer to 1.0)**: Player presence less effective. **Lower**: Player containment very effective |
| `maxThreatIntensity` | 5 | Maximum threat level (1-5 scale) | **Higher**: Threats can become more dangerous. **Lower**: Caps threat severity |
| `childNodeSpawnThreshold` | 3 | Threat level required to spawn child nodes | **Higher**: Only high threats spread. **Lower**: Threats spread at lower levels |
| `playerContainmentRadius` | 5 | Distance player must be within to slow threat | **Higher**: Player can contain from further away. **Lower**: Must be closer to threat |
| `maxChildNodeSpawnAttempts` | 10 | Max tries to find valid spawn position | **Higher**: More likely to find spawn position. **Lower**: Constrained environments block spreading |

**Formula Context:**
```
EffectiveGrowth = baseGrowthRate * (playerNearby ? containmentSlowdown : 1.0)
ThreatIntensity += EffectiveGrowth per tick
If ThreatIntensity >= childNodeSpawnThreshold AND canSpawnChildren:
    AttemptSpawnChildNode()
```

---

### factionAI

Controls how enemy factions make strategic decisions on the overworld.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `defaultIntentTickDuration` | 10 | Ticks before faction changes intent | **Higher**: Factions commit longer to strategies. **Lower**: More reactive, changes strategy often |
| `expansionStrengthThreshold` | 5 | Minimum strength to consider expansion | **Higher**: Only strong factions expand. **Lower**: Weak factions try to expand |
| `expansionTerritoryLimit` | 20 | Territory size that triggers expansion limit | **Higher**: Factions can grow larger before slowing. **Lower**: Small factions stop expanding earlier |
| `fortificationWeakThreshold` | 3 | Strength below which faction fortifies | **Higher**: More factions choose to fortify. **Lower**: Only very weak factions fortify |
| `fortificationStrengthGain` | 1 | Strength gained per fortify tick | **Higher**: Fortifying is more effective. **Lower**: Slower recovery |
| `raidStrengthThreshold` | 7 | Minimum strength to raid | **Higher**: Only powerful factions raid. **Lower**: Weaker factions attempt raids |
| `raidProximityRange` | 5 | Distance to check for raid targets | **Higher**: Factions raid from further away. **Lower**: Must be adjacent to raid |
| `retreatCriticalStrength` | 2 | Strength that triggers retreat | **Higher**: Factions retreat sooner. **Lower**: Factions fight to near-death |
| `maxTerritorySize` | 30 | Hard cap on faction territory | **Higher**: Factions can control more area. **Lower**: Limits faction spread |

**Faction Intent Flow:**
```
If strength >= raidStrengthThreshold AND enemy_nearby: RAID
Else if strength >= expansionStrengthThreshold AND territory < limit: EXPAND
Else if strength <= fortificationWeakThreshold: FORTIFY
Else if strength <= retreatCriticalStrength: RETREAT
Else: HOLD
```

---

### spawnProbabilities

Percentage chances for various spawn events.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `expansionThreatSpawnChance` | 20 | % chance to spawn threat when expanding | **Higher**: Expansion creates more threats. **Lower**: Safer expansion |
| `fortifyThreatSpawnChance` | 30 | % chance to spawn threat when fortifying | **Higher**: Fortifying creates defenders. **Lower**: Fortifying is passive |
| `bonusItemDropChance` | 30 | % chance for bonus loot | **Higher**: More loot drops. **Lower**: Scarcer rewards |

---

### mapDimensions

Default overworld map size.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `defaultMapWidth` | 100 | Overworld width in tiles | **Higher**: Larger world, more travel time. **Lower**: Smaller, denser world |
| `defaultMapHeight` | 80 | Overworld height in tiles | **Higher**: Taller world. **Lower**: Shorter world |

---

### threatTypes

Definitions for different threat node types that appear on the overworld.

| threatType | baseGrowthRate | baseRadius | primaryEffect | canSpawnChildren | Description |
|------------|----------------|------------|---------------|------------------|-------------|
| Necromancer | 0.05 | 3 | SpawnBoost | true | Spawns undead, spreads corruption |
| BanditCamp | 0.08 | 2 | ResourceDrain | false | Fast growth, steals resources |
| Corruption | 0.03 | 5 | TerrainCorruption | true | Slow but wide spread |
| BeastNest | 0.06 | 2 | SpawnBoost | false | Spawns creatures |
| OrcWarband | 0.07 | 3 | CombatDebuff | false | Aggressive, debuffs nearby |

**Parameter Impacts:**

| Parameter | Impact of Change |
|-----------|------------------|
| `baseGrowthRate` | **Higher**: This threat type grows faster. **Lower**: Slower growth |
| `baseRadius` | **Higher**: Threat affects larger area. **Lower**: More localized threat |
| `primaryEffect` | Determines what negative effect applies to nearby tiles |
| `canSpawnChildren` | **true**: Can create satellite threats. **false**: Self-contained |
| `maxIntensity` | **Higher**: Can become more dangerous. **Lower**: Caps out sooner |

---

### factionScoring

Weights for faction AI decision scoring. Higher scores make that action more likely.

#### expansion

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `strongBonus` | 5.0 | Bonus when faction is strong | **Higher**: Strong factions much prefer expansion |
| `smallTerritoryBonus` | 3.0 | Bonus when territory is small | **Higher**: Small factions aggressively expand |
| `maxTerritoryPenalty` | -10.0 | Penalty at territory limit | **Higher (less negative)**: Factions keep trying to expand at cap |
| `cultistModifier` | 3.0 | Expansion bonus for Cultist faction | **Higher**: Cultists expand aggressively |
| `orcModifier` | 2.0 | Expansion bonus for Orc faction | **Higher**: Orcs expand more |
| `beastModifier` | -1.0 | Expansion penalty for Beast faction | **Lower (more negative)**: Beasts expand even less |

#### fortification

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `weakBonus` | 6.0 | Bonus when faction is weak | **Higher**: Weak factions always fortify |
| `baseValue` | 2.0 | Base fortification score | **Higher**: Fortification always attractive |
| `necromancerModifier` | 2.0 | Bonus for Necromancer faction | **Higher**: Necromancers fortify more |

#### raiding

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `banditModifier` | 5.0 | Raiding bonus for Bandits | **Higher**: Bandits raid constantly |
| `orcModifier` | 4.0 | Raiding bonus for Orcs | **Higher**: Orcs raid more often |
| `strongBonus` | 3.0 | Bonus when strong | **Higher**: Strong factions prefer raiding |
| `strongThreshold` | 10 | Strength needed for strong bonus | **Higher**: Must be very strong for bonus |

#### retreat

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `criticalWeakBonus` | 8.0 | Bonus when critically weak | **Higher**: Very weak factions always retreat |
| `smallTerritoryPenalty` | -5.0 | Penalty when small territory | **Higher (less negative)**: Small factions more willing to retreat |
| `minTerritorySize` | 1 | Minimum territory before forced retreat | **Higher**: Factions retreat earlier |

**Decision Formula:**
```
ExpansionScore = baseValue + (isStrong ? strongBonus : 0) +
                 (smallTerritory ? smallTerritoryBonus : 0) +
                 (atLimit ? maxTerritoryPenalty : 0) +
                 factionModifier

SelectedIntent = max(expansionScore, fortificationScore, raidingScore, retreatScore)
```

---

## Quick Reference: Common Tuning Scenarios

### "AI is too passive"
- Decrease `meleeWeight` for Tanks (more negative)
- Decrease `positionalWeight` for all roles
- Increase `raidStrengthThreshold` exceptions

### "AI is too aggressive"
- Increase `positionalWeight` for all roles
- Increase `isolationWeight`
- Decrease role approach multipliers

### "Encounters are too hard"
- Decrease `powerMultiplier` in difficultyLevels
- Decrease `maxSquads` values
- Increase player `compositionBonuses`

### "Encounters are too easy"
- Increase `powerMultiplier` in difficultyLevels
- Increase `minSquads` values
- Add more diverse `squadPreferences`

### "Threats spread too fast"
- Decrease `defaultGrowthRate`
- Decrease `baseGrowthRate` per threat type
- Increase `containmentSlowdown`
- Increase `childNodeSpawnThreshold`

### "Threats spread too slowly"
- Increase `defaultGrowthRate`
- Decrease `childNodeSpawnThreshold`
- Decrease `containmentSlowdown`

### "Factions are too aggressive"
- Decrease `raidStrengthThreshold`
- Increase `fortificationWeakThreshold`
- Decrease faction raid modifiers

### "Support units aren't healing"
- Make `supportWeight` more negative for Support role
- Increase `healRadius`
- Decrease other threat weights for Support

### "DPS units are dying too fast"
- Increase `meleeWeight` and `rangedWeight` for DPS
- Increase `positionalWeight` for DPS
- Decrease `isolationHighDistance` (tighter formations)
