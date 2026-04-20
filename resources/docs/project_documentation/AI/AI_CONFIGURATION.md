# AI Configuration & Tuning

**Last Updated:** 2026-02-20

Technical reference for TinkerRogue's AI and power configuration files, accessor patterns, and tuning guide.

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Overview, system diagram, performance considerations
- [AI Controller](AI_CONTROLLER.md) - AI turn orchestration and action scoring
- [Power Evaluation](POWER_EVALUATION.md) - Power calculation algorithms
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems and difficulty scaling
- [Encounter System](ENCOUNTER_SYSTEM.md) - Encounter generation and difficulty modifiers

---

## Table of Contents

1. [Overview](#overview)
2. [AI Configuration (aiconfig.json)](#ai-configuration-aiconfigjson)
3. [Power Configuration (powerconfig.json)](#power-configuration-powerconfigjson)
4. [Accessor Pattern](#accessor-pattern)
5. [Tuning AI Behavior](#tuning-ai-behavior)

---

## Overview

**Location:** `assets/gamedata/`

All AI weights, thresholds, and multipliers are loaded from JSON configuration files. This enables designer tuning without code changes, centralizes balance parameters, and supports playtesting iterations.

**Configuration Files:**

| File | Purpose | Consumed By |
|------|---------|-------------|
| `aiconfig.json` | AI behavior weights and thresholds | Threat layers, AI controller |
| `powerconfig.json` | Power calculation weights and multipliers | Power evaluation, encounter generation |
| `influenceconfig.json` | (Not covered in this doc) | |
| `overworldconfig.json` | (Not covered in this doc) | |

---

## AI Configuration (aiconfig.json)

**Structure:**

```json
{
  "threatCalculation": {
    "flankingThreatRangeBonus": 3,
    "isolationThreshold": 3,
    "retreatSafeThreatThreshold": 10
  },
  "roleBehaviors": [
    {
      "role": "Tank",
      "meleeWeight": -0.5,
      "supportWeight": 0.2
    },
    {
      "role": "DPS",
      "meleeWeight": 0.7,
      "supportWeight": 0.1
    },
    {
      "role": "Support",
      "meleeWeight": 1.0,
      "supportWeight": -1.0
    }
  ],
  "supportLayer": {
    "healRadius": 3
  },
  "sharedRangedWeight": 0.5,
  "sharedPositionalWeight": 0.5
}
```

### Parameter Details

| Parameter | Purpose | Range | Default |
|-----------|---------|-------|---------|
| `flankingThreatRangeBonus` | Extra range for flanking detection | 1-5 | 3 |
| `isolationThreshold` | Distance before isolation risk | 1-5 | 3 |
| `retreatSafeThreatThreshold` | Threat level for "safe" retreat | 5-20 | 10 |
| `healRadius` | Support value paint radius | 2-5 | 3 |
| `sharedRangedWeight` | Ranged threat weight (all roles) | 0.0-2.0 | 0.5 |
| `sharedPositionalWeight` | Positional risk weight (all roles) | 0.0-2.0 | 0.5 |

### Role Behaviors

| Role | Melee Weight | Support Weight | Interpretation |
|------|--------------|----------------|----------------|
| Tank | -0.5 | 0.2 | Attracted to melee, stay near support |
| DPS | 0.7 | 0.1 | Avoid melee, low support priority |
| Support | 1.0 | -1.0 | Flee all danger, seek wounded allies |

### Weight Semantics

- **Negative weight** = Attraction (lower threat score = better position)
- **Positive weight** = Avoidance (higher threat score = worse position)
- **Zero weight** = Ignore this layer

### Shared vs Role-Specific

- `sharedRangedWeight` and `sharedPositionalWeight` apply to ALL roles
- `meleeWeight` and `supportWeight` are role-specific
- Allows designers to tune role differentiation without affecting shared behaviors

For how difficulty scaling modifies these values at runtime, see [Behavior & Threat Layers - Difficulty Scaling](BEHAVIOR_THREAT_LAYERS.md#difficulty-scaling).

---

## Power Configuration (powerconfig.json)

**Structure:**

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
  ],
  "roleMultipliers": [
    {"role": "Tank", "multiplier": 1.2},
    {"role": "DPS", "multiplier": 1.5},
    {"role": "Support", "multiplier": 1.0}
  ],
  "abilityValues": [
    {"ability": "Rally", "power": 15.0},
    {"ability": "Heal", "power": 20.0},
    {"ability": "BattleCry", "power": 12.0},
    {"ability": "Fireball", "power": 18.0},
    {"ability": "None", "power": 0.0}
  ],
  "compositionBonuses": [
    {"uniqueTypes": 1, "bonus": 0.8},
    {"uniqueTypes": 2, "bonus": 1.1},
    {"uniqueTypes": 3, "bonus": 1.2},
    {"uniqueTypes": 4, "bonus": 1.3}
  ],
  "leaderBonus": 1.3
}
```

### Profile Weights

| Parameter | Purpose | Recommended Range | Notes |
|-----------|---------|-------------------|-------|
| `offensiveWeight` | Damage output importance | 0.3-0.5 | Should sum to 1.0 |
| `defensiveWeight` | Survivability importance | 0.3-0.5 | with other weights |
| `utilityWeight` | Support/role importance | 0.1-0.3 | |
| `healthPenalty` | Wounded squad penalty | 1.5-3.0 | Higher = steeper penalty |

### Role Multipliers

| Role | Multiplier | Rationale |
|------|------------|-----------|
| Tank | 1.2 | Moderate threat (damage soak) |
| DPS | 1.5 | High threat (damage dealer) |
| Support | 1.0 | Baseline (utility focus) |

**CRITICAL: Not Redundant with roleBehaviors**
- `roleMultipliers` (powerconfig.json): Combat power scaling (AI sees DPS as 1.5x more dangerous)
- `roleBehaviors` (aiconfig.json): Positioning preferences (DPS avoids melee, Support seeks wounded)
- Both needed for complete AI behavior

### Ability Power Values

| Ability | Power | Rationale |
|---------|-------|-----------|
| Heal | 20.0 | Highest (sustains squads) |
| Fireball | 18.0 | High (damage burst) |
| Rally | 15.0 | Medium (buff) |
| BattleCry | 12.0 | Medium (debuff) |
| None | 0.0 | No ability |

### Composition Bonuses

| Unique Types | Bonus | Interpretation |
|--------------|-------|----------------|
| 1 | 0.8 | Mono-type penalty (-20%) |
| 2 | 1.1 | Dual-type bonus (+10%) |
| 3 | 1.2 | Triple-type bonus (+20%) |
| 4 | 1.3 | Quad-type bonus (+30%) |

### Leader Bonus

`1.3` - Present in config for future use; current ECS-based calculation includes leader abilities via utility power rather than a direct multiplier.

---

## Accessor Pattern

### Why Data-Driven?

- Eliminates hardcoded constants
- Enables designer tuning without code changes
- Centralizes balance parameters
- Supports A/B testing and playtesting iterations
- Version control tracks balance changes separately from logic

### Implementation Pattern

```go
// Configuration accessor with fallback + difficulty scaling
func GetParameterFromConfig() ValueType {
  // Load base from JSON template
  base := defaultValue
  if templates.ConfigTemplate.Parameter != 0:
    base = templates.ConfigTemplate.Parameter

  // Apply difficulty scaling
  result := base + templates.GlobalDifficulty.AI().ParameterOffset
  if result < 1:
    return 1  // Clamp to minimum
  return result
}
```

### Examples

```go
// AI behavior parameter (with difficulty offset)
func GetIsolationThreshold() int {
  base := 3
  if templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold > 0:
    base = templates.AIConfigTemplate.ThreatCalculation.IsolationThreshold
  result := base + templates.GlobalDifficulty.AI().IsolationThresholdOffset
  if result < 1: return 1
  return result
}

// Power calculation parameter (no difficulty scaling)
func GetRoleMultiplierFromConfig(role UnitRole) float64 {
  roleStr := role.String()
  for _, rm := range templates.PowerConfigTemplate.RoleMultipliers:
    if rm.Role == roleStr:
      return rm.Multiplier

  // Fallback defaults
  switch role:
    case RoleTank:    return 1.2
    case RoleDPS:     return 1.5
    case RoleSupport: return 1.0
    default:          return 1.0
}
```

### Benefits

- JSON missing/malformed? Gracefully falls back to defaults
- Designer can experiment without code knowledge
- Multiple profiles supported (future extensibility)

---

## Tuning AI Behavior

### Common Tuning Tasks

1. **Make Role More Aggressive**
   - Increase approach multiplier in `scoreApproachEnemy()`
   - Decrease `meleeWeight` (more negative = more attraction)
   - Increase `attackBaseScore` relative to `movementBaseScore`

2. **Improve Survivability**
   - Increase `rangedWeight` and `positionalWeight`
   - Increase isolation risk weight
   - Decrease approach multiplier

3. **Focus Fire Better**
   - Increase wounded target bonus
   - Add persistence (track previous target)
   - Implement squad coordination (multiple AIs targeting same enemy)

4. **Improve Positioning**
   - Increase ally proximity bonus
   - Increase flanking risk weight
   - Add terrain awareness layer

### Configuration-Only Tuning

| Goal | Configuration Change |
|------|---------------------|
| Tanks more aggressive | Decrease Tank.meleeWeight (more negative) |
| Support stays farther back | Increase Support.meleeWeight |
| All units stick together | Increase isolationThreshold |
| Fewer flanking maneuvers | Decrease flankingThreatRangeBonus |
| DPS prioritized by AI | Increase DPS roleMultiplier |
| Healers more valuable | Increase Heal abilityValue |
| Harder encounters globally | Increase GlobalDifficulty encounter PowerMultiplierScale |
| AI more aggressive globally | Decrease GlobalDifficulty AI FlankingRangeBonusOffset |

---

**End of Document**

For questions or clarifications, consult the source code or the [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) overview.
