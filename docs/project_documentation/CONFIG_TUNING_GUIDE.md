# Configuration Tuning Guide

**Last Updated:** 2026-02-06

This document explains all values in the game's JSON configuration files and how changing them impacts gameplay algorithms.

---

## Table of Contents

1. [aiconfig.json](#aiconfigjson) - AI behavior and threat assessment
2. [powerconfig.json](#powerconfigjson) - Power calculations for units and squads
3. [encounterdata.json](#encounterdatajson) - Combat encounter generation and faction archetypes
4. [overworldconfig.json](#overworldconfigjson) - Overworld threat and faction systems
5. [nodeDefinitions.json](#nodedefinitionsjson) - Overworld node types (threats, settlements, fortresses)
6. [config.go](#configgo) - Compile-time game constants

---

## aiconfig.json

Controls AI decision-making, threat assessment, and role-based behavior.

**File Location:** `assets/gamedata/aiconfig.json`

### threatCalculation

Parameters for how AI calculates danger zones and tactical positioning.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `flankingThreatRangeBonus` | 3 | Extra range added when calculating flanking threat | **Higher**: AI considers flanking danger from further away, becomes more cautious about positioning. **Lower**: AI ignores distant flanking threats, more aggressive positioning |
| `isolationThreshold` | 3 | Distance to nearest ally before isolation risk starts | **Higher**: Units can spread out more before feeling isolated. **Lower**: Units must stay closer to avoid isolation penalty |
| `retreatSafeThreatThreshold` | 10 | Threat level below which an adjacent tile is considered a safe escape route | **Higher**: More tiles count as escape routes, AI feels less trapped. **Lower**: Fewer escape routes, AI more likely to feel cornered |

**Note:** `engagementPressureMax` (200) and `isolationMaxDistance` (8) are hardcoded constants in `mind/behavior/threat_constants.go`, not configurable via JSON.

**Formula Context:**
```
IsolationRisk = 0.0 if distance <= isolationThreshold
              = linear gradient (0.0 to 1.0) from isolationThreshold to isolationMaxDistance (8)
              = 1.0 if distance >= isolationMaxDistance

RetreatQuality = (safe_adjacent_tiles / total_adjacent_tiles)
```

---

### roleBehaviors

Defines how different unit roles weight threat factors when choosing positions. Each role configures only `meleeWeight` and `supportWeight`. **Negative weights mean the role is attracted to that threat type.**

`rangedWeight` and `positionalWeight` are shared across all roles (see [Shared Weights](#shared-weights) below).

| Role | Parameter | Default | Description | Impact of Change |
|------|-----------|---------|-------------|------------------|
| **Tank** | `meleeWeight` | -0.5 | How much melee threat affects position scoring | **Negative**: Tanks seek melee combat (move toward enemies). **Positive**: Tanks would avoid melee (wrong behavior) |
| | `supportWeight` | 0.2 | How much support value (healing) matters | **Higher**: Tanks position near healers. **Negative**: Tanks would seek wounded allies |
| **DPS** | `meleeWeight` | 0.7 | Avoidance of melee threat | **Higher**: DPS stays far from melee enemies. **Lower**: DPS willing to get closer |
| | `supportWeight` | 0.1 | Attraction to support positions | Low value means DPS doesn't prioritize being near healers |
| **Support** | `meleeWeight` | 1.0 | Strong avoidance of melee | Highest value - supports stay far from melee |
| | `supportWeight` | -1.0 | **Negative**: Seeks wounded allies | This makes supports move toward units that need healing |

### Shared Weights

These top-level fields apply the same weight to all roles for ranged threat and positional awareness:

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `sharedRangedWeight` | 0.5 | How much ranged threat matters (all roles) | **Higher**: All roles avoid ranged fire more. **Lower**: All roles ignore ranged threats |
| `sharedPositionalWeight` | 0.5 | How much positional risk matters (all roles) | **Higher**: All roles care more about flanking/isolation. **Lower**: All roles ignore tactical positioning |

**Formula Context:**
```
PositionScore = (meleeThreat * meleeWeight) +
                (rangedThreat * sharedRangedWeight) +
                (supportValue * supportWeight) +
                (positionalRisk * sharedPositionalWeight)

Lower score = better position (negative weights invert attraction)
```

**Tuning Tips:**
- To make Tanks more aggressive: decrease `meleeWeight` (more negative)
- To make DPS more cautious: increase `meleeWeight` for DPS
- To make Support prioritize healing over safety: make `supportWeight` more negative
- To make all roles more cautious: increase `sharedPositionalWeight`

---

### Positional Risk (Hardcoded)

Positional risk sub-weights are **no longer configurable**. They use equal weights (0.25 each for flanking, isolation, pressure, retreat) and are hardcoded in `mind/behavior/threat_constants.go`. The role's `sharedPositionalWeight` controls overall importance of positional risk.

---

### supportLayer

Parameters for how support units evaluate healing and ally proximity.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `healRadius` | 3 | Distance at which support value radiates from wounded allies | **Higher**: Support units consider healing targets from further away. **Lower**: Must be closer to recognize healing opportunities |
| `buffPriorityEngagementRange` | 4 | Range at which buff targets are considered "engaged" | **Higher**: Support buffs units further from combat. **Lower**: Only buffs units actively fighting |

**Note:** `allyProximityRadius` is derived automatically as `healRadius - 1`.

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

Power calculation profile with top-level category weights. Currently only the **Balanced** profile is used. Sub-calculations within each category (offensive, defensive, utility) use fixed formulas internally -- only the category weights are configurable.

| Profile | offensiveWeight | defensiveWeight | utilityWeight | healthPenalty | Use Case |
|---------|-----------------|-----------------|---------------|---------------|----------|
| **Balanced** | 0.40 | 0.40 | 0.20 | 2.0 | Default -- equal offense/defense |

**Note:** Category weights must sum to 1.0.

**Parameter Details:**

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `offensiveWeight` | 0.4 | Weight for offensive stats (damage output) | **Higher**: High-damage units valued more in power calculations |
| `defensiveWeight` | 0.4 | Weight for defensive stats (survivability) | **Higher**: Tanky units valued more |
| `utilityWeight` | 0.2 | Weight for utility (role, abilities, cover) | **Higher**: Leaders, support units, cover providers valued more |
| `healthPenalty` | 2.0 | Exponent for health-based power scaling | **Higher**: Wounded squads lose power faster. `(currentHP/maxHP)^penalty` |

**Internal Formulas (not configurable):**
```
OffensivePower = avgDamage * hitRate * critMultiplier
DefensivePower = (effectiveHP * dodgeMultiplier) + avgResistance
UtilityPower   = roleValue + abilityValue + coverValue

UnitPower  = (OffensivePower * offensiveWeight) +
             (DefensivePower * defensiveWeight) +
             (UtilityPower * utilityWeight)

SquadPower = SumOfUnitPower * HealthMultiplier * CompositionBonus
HealthMultiplier = (currentHP/maxHP)^healthPenalty
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

### leaderBonus

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `leaderBonus` | 1.3 | Power multiplier for leader units | **Higher**: Leaders much more valuable/powerful. **Lower**: Leaders closer to regular units |

---

### Scaling Constants (Hardcoded)

The following scaling factors are **no longer configurable via JSON**. They are internal implementation constants in `mind/evaluation/roles.go`:

| Constant | Value | Purpose |
|----------|-------|---------|
| `RoleScalingFactor` | 10.0 | Converts role multiplier to comparable power range |
| `DodgeScalingFactor` | 100.0 | Converts dodge chance to comparable power range |
| `CoverScalingFactor` | 100.0 | Converts cover value to comparable power range |
| `CoverBeneficiaryMultiplier` | 2.5 | Average units benefiting from cover |

---

## encounterdata.json

Controls combat encounter generation, difficulty scaling, faction archetypes, and enemy composition.

**File Location:** `assets/gamedata/encounterdata.json`

### factions

Defines strategic archetypes for each enemy faction. These control overworld AI behavior via the strategy bonus system.

| Faction | Strategy | Aggression | Description |
|---------|----------|------------|-------------|
| Necromancers | Defensive | 0.3 | Turtles and fortifies, low aggression |
| Cultists | Expansionist | 0.7 | Aggressively expands territory |
| Orcs | Aggressor | 0.9 | Most aggressive faction, raids constantly |
| Bandits | Raider | 0.8 | Focus on raiding over expanding |
| Beasts | Territorial | 0.5 | Holds territory, moderate aggression |

**Parameter Impacts:**

| Parameter | Impact of Change |
|-----------|------------------|
| `strategy` | Which strategy bonus set from `overworldconfig.json` `strategyBonuses` applies. Must be one of: Expansionist, Aggressor, Raider, Defensive, Territorial |
| `aggression` | 0.0-1.0 scale. Modifies scoring: high aggression boosts expansion/raiding, low aggression boosts fortification/retreat |

---

### difficultyLevels

Defines difficulty tiers for encounters. Each level controls enemy power budget and squad composition limits.

| Level | Name | powerMultiplier | squadCount | minUnits | maxUnits | minPower | maxPower |
|-------|------|-----------------|------------|----------|----------|----------|----------|
| 1 | Easy | 0.7 | 5 | 2 | 4 | 50 | 1000 |
| 2 | Moderate | 0.9 | 6 | 2 | 4 | 50 | 1500 |
| 3 | Fair | 1.0 | 7 | 3 | 5 | 50 | 2000 |
| 4 | Hard | 1.2 | 8 | 3 | 6 | 100 | 2500 |
| 5 | Boss | 1.5 | 10 | 4 | 8 | 200 | 3000 |

**Parameter Details:**

| Parameter | Description | Impact of Change |
|-----------|-------------|------------------|
| `powerMultiplier` | Multiplies average player squad power to get target enemy squad power | **Higher**: Enemies have more power per squad. **Lower**: Easier encounters |
| `squadCount` | Number of enemy squads generated | **Higher**: More enemy squads. **Lower**: Fewer squads |
| `minUnitsPerSquad` | Minimum units in each enemy squad | **Higher**: Denser squads. **Lower**: Potentially thin squads |
| `maxUnitsPerSquad` | Maximum units in each enemy squad | **Higher**: Larger squads possible. **Lower**: Caps squad size |
| `minTargetPower` | Floor for target enemy squad power | Prevents trivially weak encounters when player power is very low |
| `maxTargetPower` | Ceiling for target enemy squad power | Prevents impossibly strong encounters when player power is very high |

**Formula Context:**
```
AvgPlayerSquadPower = TotalDeployedPower / NumDeployedSquads
TargetEnemySquadPower = AvgPlayerSquadPower * powerMultiplier
TargetEnemySquadPower = clamp(TargetEnemySquadPower, minTargetPower, maxTargetPower)
NumEnemySquads = squadCount
```

---

### encounterDefinitions

Each encounter definition ties a node type to combat configuration. Multiple encounters per faction are supported (basic/elite/boss variants). Encounters are selected based on the threat node's faction and intensity level.

| ID | encounterTypeId | Name | squadPreferences | difficulty | Tags |
|----|-----------------|------|------------------|------------|------|
| necromancer | undead_basic | Undead Horde | melee, melee, magic | 3 | common, undead |
| necromancer_crypt | undead_crypt | Crypt Guardians | melee, magic, magic | 4 | elite, undead |
| necromancer_lich | lich_master | Lich Master | magic, magic, melee | 5 | boss, undead |
| banditcamp | bandit_basic | Bandit Ambush | melee, ranged, ranged | 3 | common, ranged-focused |
| banditcamp_raiders | bandit_raiders | Bandit Raiders | ranged, ranged, melee | 4 | elite, ranged-focused |
| banditcamp_fortress | bandit_fortress | Bandit Fortress | melee, ranged, ranged | 5 | boss, fortified |
| corruption | corruption_basic | Corrupted Forces | magic, ranged, melee | 3 | common, corruption |
| corruption_ritual | corruption_ritual | Ritual Cultists | magic, magic, ranged | 4 | elite, corruption |
| corruption_temple | corruption_temple | Corrupted Temple | magic, magic, magic | 5 | boss, corruption |
| beastnest | beast_basic | Beast Pack | melee, melee, melee | 3 | common, swarm |
| beastnest_alpha | beast_alpha | Alpha Beast Pack | melee, melee, melee | 4 | elite, swarm |
| beastnest_primal | beast_primal | Primal Beast Lord | melee, melee, melee | 5 | boss, swarm |
| orcwarband | orc_basic | Orc Warband | melee, ranged, magic | 3 | common, balanced |
| orcwarband_raiders | orc_raiders | Orc Raiders | melee, melee, ranged | 4 | elite, balanced |
| orcwarband_warlord | orc_warlord | Orc Warlord | melee, melee, magic | 5 | boss, balanced |

**Parameter Details:**

| Parameter | Description | Impact of Change |
|-----------|-------------|------------------|
| `id` | Unique identifier. Must match a node definition ID for the base encounter (e.g., "necromancer") | Required for backwards compatibility |
| `encounterTypeId` | Combat type identifier (e.g., "undead_basic") | Must be unique across all encounters |
| `squadPreferences` | Determines what types of enemy squads spawn: "melee", "ranged", or "magic" | More melee = aggressive encounters, more magic = caster-heavy |
| `defaultDifficulty` | Which difficulty level to use (1-5) | Higher = harder encounters |
| `tags` | Used for filtering/selecting encounters | Add new tags for conditional spawning |
| `basicDrops` | Normal item drops from this encounter | Add/remove items to adjust loot tables |
| `highTierDrops` | Drops at high threat intensity | Rare/powerful items as rewards |
| `factionId` | Links encounter to a faction (must match a faction in `factions`) | Required for faction-based encounter selection |

**Tuning Tips:**
- To create harder melee encounters: add more "melee" to squadPreferences
- To create caster-heavy encounters: add "magic" entries
- To vary difficulty: change defaultDifficulty per encounter type
- To add new encounter variants: add new entries with same factionId but different id/difficulty

---

### squadTypes

Definitions for enemy squad archetypes. Referenced by `squadPreferences` in encounter definitions.

| ID | Name | Description | Used For |
|----|------|-------------|----------|
| melee | Melee Squad | Close-range combat units | Front-line fighters |
| ranged | Ranged Squad | Long-range attackers | Archers, crossbowmen |
| magic | Magic Squad | Spellcasters and support | Casters, healers |

---

### defaultEncounter

Fallback configuration when an encounter definition is not found.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `basicDrops` | ["Unknown Item"] | Generic drop for unrecognized encounters |
| `highTierDrops` | [] | No high-tier drops for unknown encounters |

---

## overworldconfig.json

Controls the overworld threat system, faction AI, and strategic gameplay.

**File Location:** `assets/gamedata/overworldconfig.json`

### threatGrowth

Parameters for how threats spread and grow on the overworld map.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `containmentSlowdown` | 0.5 | Growth multiplier when player nearby | **Higher (closer to 1.0)**: Player presence less effective. **Lower**: Player containment very effective |
| `maxThreatIntensity` | 5 | Maximum threat level (1-5 scale) | **Higher**: Threats can become more dangerous. **Lower**: Caps threat severity |
| `childNodeSpawnThreshold` | 3 | Threat level required to spawn child nodes | **Higher**: Only high threats spread. **Lower**: Threats spread at lower levels |
| `maxChildNodeSpawnAttempts` | 10 | Max tries to find valid spawn position | **Higher**: More likely to find spawn position. **Lower**: Constrained environments block spreading |

**Note:** Per-node `baseGrowthRate` is configured in `nodeDefinitions.json`, not here.

**Formula Context:**
```
EffectiveGrowth = node.baseGrowthRate * (playerNearby ? containmentSlowdown : 1.0)
ThreatIntensity += EffectiveGrowth per tick
If ThreatIntensity >= childNodeSpawnThreshold AND node.canSpawnChildren:
    AttemptSpawnChildNode()
```

---

### factionAI

Controls how enemy factions make strategic decisions on the overworld.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `defaultIntentTickDuration` | 10 | Ticks before faction changes intent | **Higher**: Factions commit longer to strategies. **Lower**: More reactive, changes strategy often |
| `expansionTerritoryLimit` | 20 | Territory size that triggers expansion limit | **Higher**: Factions can grow larger before slowing. **Lower**: Small factions stop expanding earlier |
| `fortificationStrengthGain` | 1 | Strength gained per fortify tick | **Higher**: Fortifying is more effective. **Lower**: Slower recovery |
| `maxTerritorySize` | 30 | Hard cap on faction territory | **Higher**: Factions can control more area. **Lower**: Limits faction spread |

---

### strengthThresholds

Unified strength thresholds used across all faction AI decisions. Replaces the old per-action thresholds.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `weak` | 3 | Strength at or below which faction is "weak" | **Higher**: More factions choose to fortify. **Lower**: Only very weak factions fortify |
| `strong` | 7 | Strength at or above which faction is "strong" | **Higher**: Only powerful factions expand/raid. **Lower**: Weaker factions attempt expansion/raids |
| `critical` | 2 | Strength at or below which faction must retreat | **Higher**: Factions retreat sooner. **Lower**: Factions fight to near-death |

**Constraint:** Must satisfy `critical <= weak < strong`.

**Faction Intent Flow:**
```
Scores are calculated for each action (expansion, fortification, raiding, retreat)
Each score factors in: strength vs thresholds, territory size, faction archetype bonuses, aggression

SelectedIntent = highest scoring action
If all scores < idleScoreThreshold: IDLE (do nothing)
```

---

### victoryConditions

Thresholds that determine game-ending conditions.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `highIntensityThreshold` | 4 | Intensity level considered "high" for threats | **Higher**: More tolerant of strong threats. **Lower**: Earlier defeat trigger |
| `maxHighIntensityThreats` | 10 | Max number of high-intensity threats before defeat | **Higher**: More room for threats before losing. **Lower**: Tighter defeat condition |
| `maxThreatInfluence` | 100.0 | Total threat influence threshold for defeat | **Higher**: More total threat tolerated. **Lower**: Stricter defeat condition |

---

### factionScoringControl

Controls for the faction scoring system behavior.

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `idleScoreThreshold` | 2.0 | Minimum score for any action to be taken (below = idle) | **Higher**: Factions idle more often. **Lower**: Factions always take action |
| `raidBaseIntensity` | 3 | Base intensity for raid-spawned threats | **Higher**: Raids create stronger threats. **Lower**: Raids create weaker threats |
| `raidIntensityScale` | 0.33 | How much faction strength scales raid intensity | **Higher**: Strong factions create much stronger raid threats. **Lower**: Raid intensity less dependent on strength |

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

### factionScoring

Base scoring parameters for faction AI decisions. These are combined with faction archetype bonuses and aggression modifiers.

#### expansion

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `strongBonus` | 5.0 | Bonus when faction is strong | **Higher**: Strong factions much prefer expansion |
| `smallTerritoryBonus` | 3.0 | Bonus when territory is small | **Higher**: Small factions aggressively expand |
| `maxTerritoryPenalty` | -10.0 | Penalty at territory limit | **Higher (less negative)**: Factions keep trying to expand at cap |

#### fortification

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `weakBonus` | 6.0 | Bonus when faction is weak | **Higher**: Weak factions always fortify |
| `baseValue` | 2.0 | Base fortification score | **Higher**: Fortification always attractive |

#### raiding

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `strongBonus` | 3.0 | Bonus when very strong (strong threshold + 3) | **Higher**: Strong factions prefer raiding |

#### retreat

| Parameter | Default | Description | Impact of Change |
|-----------|---------|-------------|------------------|
| `criticalWeakBonus` | 8.0 | Bonus when critically weak | **Higher**: Very weak factions always retreat |
| `smallTerritoryPenalty` | -5.0 | Penalty when small territory | **Higher (less negative)**: Small factions more willing to retreat |
| `minTerritorySize` | 1 | Minimum territory before forced retreat | **Higher**: Factions retreat earlier |

**Decision Formula:**
```
ExpansionScore = (isStrong ? strongBonus : 0) +
                 (smallTerritory ? smallTerritoryBonus : 0) +
                 (atLimit ? maxTerritoryPenalty : 0) +
                 strategyBonus.expansionBonus
ExpansionScore *= (0.7 + aggression * 0.3)

FortificationScore = (isWeak ? weakBonus : 0) + baseValue +
                     strategyBonus.fortificationBonus
FortificationScore *= (1.3 - aggression * 0.3)

RaidingScore = (notStrong ? 0) +
               strategyBonus.raidingBonus +
               (veryStrong ? strongBonus : 0)
RaidingScore *= (0.5 + aggression * 0.5)

RetreatScore = (isCritical ? criticalWeakBonus : 0) +
               (tinyTerritory ? smallTerritoryPenalty : 0) -
               strategyBonus.retreatPenalty
RetreatScore *= (1.2 - aggression * 0.4)

SelectedIntent = max(all scores)
If all < idleScoreThreshold: IDLE
```

---

### strategyBonuses

Per-strategy scoring modifiers. Each faction's archetype (from `encounterdata.json`) maps to one of these strategies.

| Strategy | expansionBonus | fortificationBonus | raidingBonus | retreatPenalty |
|----------|----------------|--------------------|--------------|----------------|
| Expansionist | 3.0 | 0.0 | 1.0 | 0.0 |
| Aggressor | 2.0 | 0.0 | 4.0 | 0.0 |
| Raider | 0.0 | 0.0 | 5.0 | -2.0 |
| Defensive | 0.0 | 2.0 | 0.0 | 2.0 |
| Territorial | -1.0 | 1.0 | 0.0 | -3.0 |

**How it works:** A faction's strategy (e.g., Orcs = "Aggressor") selects a row from this table. The bonuses are added to the base scoring parameters when that faction evaluates its options. `retreatPenalty` is *subtracted* from retreat score (positive = less likely to retreat).

**Tuning Tips:**
- To make a faction more aggressive: increase its `raidingBonus` or switch to "Aggressor" strategy
- To make a faction turtle: increase `fortificationBonus` and set `retreatPenalty` positive
- Negative `retreatPenalty` (e.g., Territorial at -3.0) actually *increases* retreat score, making retreat more likely

---

## nodeDefinitions.json

Defines all overworld node types: threats, settlements, and fortresses. This replaces the old `threatTypes` section of `overworldconfig.json`.

**File Location:** `assets/gamedata/nodeDefinitions.json`

### Node Categories

Valid categories: `threat`, `settlement`, `fortress`

### Threat Nodes

| ID | Display Name | Growth Rate | Radius | Effect | Spawns Children | Faction |
|----|-------------|-------------|--------|--------|-----------------|---------|
| necromancer | Necromancer | 0.05 | 3 | SpawnBoost | Yes | Necromancers |
| banditcamp | Bandit Camp | 0.08 | 2 | ResourceDrain | No | Bandits |
| corruption | Corruption | 0.03 | 5 | TerrainCorruption | Yes | Cultists |
| beastnest | Beast Nest | 0.06 | 2 | SpawnBoost | No | Beasts |
| orcwarband | Orc Warband | 0.07 | 3 | CombatDebuff | No | Orcs |

### Settlement Nodes

| ID | Display Name | Radius | Services |
|----|-------------|--------|----------|
| marketplace | Marketplace | 1 | trade, repair |
| guild_hall | Guild Hall | 2 | recruit, train |

### Fortress Nodes

| ID | Display Name | Radius | Effect |
|----|-------------|--------|--------|
| watchtower | Watchtower | 4 | SpawnBoost |

### Parameter Details

| Parameter | Description | Impact of Change |
|-----------|-------------|------------------|
| `id` | Unique identifier (also used as the type key) | Must be unique. Base threat node IDs must match encounter definition IDs |
| `category` | Node category: "threat", "settlement", "fortress" | Determines behavior and validation rules |
| `displayName` | Human-readable name | Shown in UI |
| `color` | RGBA color `{r, g, b, a}` | Display color on overworld map. Alpha 0 = invisible |
| `overworld.baseGrowthRate` | Growth rate per tick (threat nodes only) | **Higher**: This threat type grows faster. **Lower**: Slower growth |
| `overworld.baseRadius` | Influence radius in tiles | **Higher**: Threat affects larger area. **Lower**: More localized |
| `overworld.primaryEffect` | Effect type: SpawnBoost, ResourceDrain, TerrainCorruption, CombatDebuff | Determines what negative effect applies to nearby tiles |
| `overworld.canSpawnChildren` | Whether node can create satellite threats | **true**: Can create child nodes. **false**: Self-contained |
| `factionId` | Faction this node belongs to (required for threats) | Links to faction in `encounterdata.json` |
| `services` | Available services (settlements only) | Determines what player can do at this location |

### defaultNode

Fallback configuration for unknown node types:
- Display Name: "Unknown Location"
- Color: Gray (128, 128, 128)
- Radius: 1

---

## config.go

Compile-time constants defined in `config/config.go`. These require recompilation to change.

**File Location:** `config/config.go`

### Debug and Profiling Flags

| Constant | Default | Description |
|----------|---------|-------------|
| `DISPLAY_THREAT_MAP_LOG_OUTPUT` | false | Show threat map debug logging |
| `DISPLAY_DEATAILED_COMBAT_OUTPUT` | false | Show detailed combat debug logging |
| `DEBUG_MODE` | true | Enable debug visualization and logging |
| `ENABLE_BENCHMARKING` | false | Enable pprof profiling server on localhost:6060 |
| `ENABLE_COMBAT_LOG` | false | Enable combat log UI panel and message recording |
| `ENABLE_COMBAT_LOG_EXPORT` | false | Export battle logs as JSON after each battle |
| `COMBAT_LOG_EXPORT_DIR` | "./combat_logs" | Directory for exported combat logs |
| `ENABLE_OVERWORLD_LOG_EXPORT` | false | Export overworld session logs as JSON |
| `OVERWORLD_LOG_EXPORT_DIR` | "./overworld_logs" | Directory for exported overworld logs |

### Player Starting Attributes

| Constant | Default | Derived Effect |
|----------|---------|----------------|
| `DefaultPlayerStrength` | 15 | 50 HP (20 + 15*2) |
| `DefaultPlayerDexterity` | 20 | 100% hit, 10% crit, 6% dodge |
| `DefaultPlayerMagic` | 0 | No magic abilities at start |
| `DefaultPlayerLeadership` | 0 | No squad leadership at start |
| `DefaultPlayerArmor` | 2 | 4 physical resistance (2*2) |
| `DefaultPlayerWeapon` | 3 | 6 bonus damage (3*2) |

### Player Resources

| Constant | Default | Description |
|----------|---------|-------------|
| `DefaultPlayerStartingGold` | 100000 | Starting gold for purchasing units |
| `DefaultPlayerMaxUnits` | 50 | Maximum units player can own |
| `DefaultPlayerMaxSquads` | 10 | Maximum squads player can own |

### Unit Defaults

| Constant | Default | Description |
|----------|---------|-------------|
| `DefaultMovementSpeed` | 3 | Base movement speed |
| `DefaultAttackRange` | 1 | Base attack range (melee) |
| `DefaultBaseHitChance` | 80 | Base % chance to hit |
| `DefaultMaxHitRate` | 100 | Maximum hit rate cap |
| `DefaultMaxCritChance` | 50 | Maximum crit chance cap |
| `DefaultMaxDodgeChance` | 30 | Maximum dodge chance cap |
| `DefaultBaseCapacity` | 6 | Base squad unit capacity |
| `DefaultMaxCapacity` | 9 | Maximum squad unit capacity |

### Combat Constants

| Constant | Default | Description |
|----------|---------|-------------|
| `CritDamageBonus` | 0.5 | Extra damage multiplier on crits (1.5x total) |

### Display Constants

| Constant | Default | Description |
|----------|---------|-------------|
| `DefaultMapWidth` | 100 | Map width in tiles |
| `DefaultMapHeight` | 80 | Map height in tiles |
| `DefaultTilePixels` | 32 | Tile size in pixels |
| `DefaultScaleFactor` | 3 | Display scale factor |
| `DefaultRightPadding` | 500 | Right UI panel padding |
| `DefaultZoomNumberOfSquare` | 30 | Squares visible when zoomed |
| `DefaultStaticUIOffset` | 1000 | Static UI position offset |

---

## Quick Reference: Common Tuning Scenarios

### "AI is too passive"
- Decrease `meleeWeight` for Tanks (more negative) in aiconfig.json
- Decrease `sharedPositionalWeight` in aiconfig.json
- Increase faction `aggression` in encounterdata.json

### "AI is too aggressive"
- Increase `sharedPositionalWeight` in aiconfig.json
- Increase `isolationThreshold` in aiconfig.json
- Decrease faction `aggression` in encounterdata.json

### "Encounters are too hard"
- Decrease `powerMultiplier` in difficultyLevels
- Decrease `squadCount` values
- Decrease `maxUnitsPerSquad` values
- Increase player `compositionBonuses`

### "Encounters are too easy"
- Increase `powerMultiplier` in difficultyLevels
- Increase `squadCount` values
- Increase `minUnitsPerSquad` values
- Add more diverse `squadPreferences`

### "Threats spread too fast"
- Decrease per-node `baseGrowthRate` in nodeDefinitions.json
- Increase `containmentSlowdown` in overworldconfig.json
- Increase `childNodeSpawnThreshold` in overworldconfig.json

### "Threats spread too slowly"
- Increase per-node `baseGrowthRate` in nodeDefinitions.json
- Decrease `childNodeSpawnThreshold` in overworldconfig.json
- Decrease `containmentSlowdown` in overworldconfig.json

### "Factions are too aggressive"
- Increase `strong` threshold in overworldconfig.json (harder to qualify for raids)
- Decrease faction `aggression` values in encounterdata.json
- Decrease `raidingBonus` in strategy bonuses

### "Factions are too passive"
- Decrease `strong` threshold in overworldconfig.json
- Increase faction `aggression` values in encounterdata.json
- Increase `raidingBonus` and `expansionBonus` in strategy bonuses
- Decrease `idleScoreThreshold` in factionScoringControl

### "Support units aren't healing"
- Make `supportWeight` more negative for Support role in aiconfig.json
- Increase `healRadius` in aiconfig.json
- Decrease `sharedRangedWeight` to reduce Support's tendency to flee ranged fire

### "DPS units are dying too fast"
- Increase `meleeWeight` for DPS in aiconfig.json
- Increase `sharedPositionalWeight` in aiconfig.json
- Decrease `isolationThreshold` (tighter formations)
