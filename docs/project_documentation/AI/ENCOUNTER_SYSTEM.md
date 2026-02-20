# Encounter System

**Last Updated:** 2026-02-17

Technical reference for TinkerRogue's encounter generation, combat lifecycle, and reward systems.

---

## Related Documents

- [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) - Overview, system diagram, performance considerations
- [AI Controller](AI_CONTROLLER.md) - AI turn orchestration and action scoring
- [Power Evaluation](POWER_EVALUATION.md) - Power calculation shared by AI threat and encounter generation
- [AI Configuration](AI_CONFIGURATION.md) - Config files, accessor patterns, tuning guide
- [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md) - Threat layer subsystems and spatial analysis

---

## Table of Contents

1. [Overview](#overview)
2. [Encounter Generation Flow](#encounter-generation-flow)
3. [Encounter Resolution and Rewards](#encounter-resolution-and-rewards)
4. [Troubleshooting](#troubleshooting)
5. [File Reference](#file-reference)

---

## Overview

The encounter system creates balanced combat scenarios by using the shared power evaluation system (documented in [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md#power-evaluation-system)) to match enemy squad power against the player's deployed forces. Encounters can be triggered by overworld threat nodes, random events, or garrison defenses.

**Integration with Power System:** `GenerateEncounterSpec()` calls `CalculateSquadPower()` to assess player strength, then uses `EstimateUnitPowerFromTemplate()` to build enemy squads within a power budget. This ensures the same power formula drives both AI threat assessment and encounter balancing.

**Integration with AI:** Once combat begins, enemy squads are controlled by the AI controller (documented in [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md#core-ai-systems)), which uses threat layers (documented in [Behavior & Threat Layers](BEHAVIOR_THREAT_LAYERS.md)) for positioning decisions.

---

## Encounter Generation Flow

```
Player Triggers Encounter
|
+- TriggerCombatFromThreat() / TriggerRandomEncounter() / TriggerGarrisonDefense()
|  +- Creates OverworldEncounterData entity (encounter ID)
|
+- EncounterService.StartEncounter() or StartGarrisonDefense()
|  |
|  +- Validate no active encounter
|  +- Hide encounter sprite during combat
|  |
|  +- SpawnCombatEntities()
|     |
|     +- Check for NPC garrison (if ThreatNodeID has garrison)
|     |  +- spawnGarrisonEncounter() - uses existing garrison squads as enemies
|     |
|     +- Standard path: GenerateEncounterSpec()
|        |
|        +- Ensure player squads deployed (auto-deploys if needed)
|        |
|        +- Calculate total player power
|        |  +- For each squad: CalculateSquadPower()
|        |
|        +- Calculate average squad power
|        |
|        +- getDifficultyModifier(level)
|        |  +- Lookup JSONEncounterDifficulty from EncounterDifficultyTemplates
|        |  +- Apply GlobalDifficulty.Encounter() overlay
|        |     +- PowerMultiplier *= diff.PowerMultiplierScale
|        |     +- SquadCount += diff.SquadCountOffset
|        |     +- MinUnitsPerSquad += diff.MinUnitsPerSquadOffset
|        |     +- MaxUnitsPerSquad += diff.MaxUnitsPerSquadOffset
|        |
|        +- targetEnemySquadPower = avgPlayerPower * difficultyMod.PowerMultiplier
|        |  (clamped to MinTargetPower / MaxTargetPower)
|        |
|        +- generateEnemySquadsByPower()
|        |  +- getSquadComposition() - encounter type preferences or random
|        |  +- Pre-compute enemy positions (circle around player)
|        |  +- For each squad (count from difficulty):
|        |     +- createSquadForPowerBudget()
|        |        +- filterUnitsBySquadType() (melee/ranged/magic)
|        |        +- Add units until power budget reached (PowerThreshold=0.95)
|        |        |  +- EstimateUnitPowerFromTemplate() for each candidate
|        |        +- Ensure MinUnitsPerSquad minimum
|        |
|        +- Return EncounterSpec
|           +- PlayerSquadIDs
|           +- EnemySquads (EnemySquadSpec with SquadID, Position, Power, Type, Name)
|           +- Difficulty
|           +- EncounterType
|
+- Create factions (CombatFactionManager.CreateFactionWithPlayer)
+- Assign player squads to faction
+- Assign enemy squads to faction + CreateActionStateForSquad
|
+- EncounterService tracks ActiveEncounter
|  (EncounterID, EnemySquadIDs, PlayerFactionID, EnemyFactionID, timing, etc.)
|
+- EnterTactical("combat") - mode transition to combat
```

**Key Insights:**
- Uses same `CalculateSquadPower()` as AI threat system
- Ensures balanced encounters (enemy power ~ player power * modifier)
- Enemy squad composition uses power estimation before spawning
- Templates converted to ECS entities only after validation
- Garrison defense (`StartGarrisonDefense`) uses garrison squads as the player-faction force

---

## Encounter Resolution and Rewards

**Location:** `mind/encounter/rewards.go`, `mind/encounter/encounter_service.go`

When combat ends, `EncounterService.ExitCombat()` is called with the exit reason (Victory/Defeat/Flee):

```go
ExitCombat(reason, result, combatCleaner):
  1. EndEncounter() or RestoreEncounterSprite() based on reason
  2. RecordEncounterCompletion() - history + restore player position
  3. combatCleaner.CleanupCombat(enemySquadIDs) - dispose combat entities
```

**Reward Calculation:**

```go
calculateRewards(intensity, encounter):
  baseGold = 100 + (intensity * 50)
  baseXP = 50 + (intensity * 25)

  // Intensity-based multiplier (1.1x-1.5x for intensity 1-5)
  typeMultiplier = 1.0 + (float64(intensity) * 0.1)

  return {
    Gold: int(baseGold * typeMultiplier),
    Experience: int(baseXP * typeMultiplier),
  }
```

**Reward Distribution:**
- Gold: Added to player's `ResourceStockpile`
- XP: Divided evenly among all alive units across all player squads via `squads.AwardExperience()`

---

## Troubleshooting

### Encounter Spawning Errors

**Symptoms:** Combat never starts, errors in encounter setup.

**Possible Causes:**
- Player has no deployed squads (auto-deploy may be needed)
- EncounterService already has active encounter
- Garrison node has no garrison data when expected

**Debug Steps:**
1. Check `EncounterService.IsEncounterActive()` - only one encounter allowed at a time
2. Verify `ensurePlayerSquadsDeployed()` auto-deployed squads
3. If garrison defense, check `garrison.GetGarrisonAtNode()` returns valid data
4. Examine `SpawnCombatEntities()` error return for specific failure reason

---

## File Reference

| File | Purpose | Key Functions |
|------|---------|---------------|
| `mind/encounter/encounter_generator.go` | Enemy creation | `GenerateEncounterSpec()` |
| `mind/encounter/encounter_setup.go` | Combat initialization | `SpawnCombatEntities()`, `spawnGarrisonEncounter()`, `generateEnemySquadsByPower()`, `createSquadForPowerBudget()` |
| `mind/encounter/encounter_service.go` | Encounter lifecycle | `EncounterService`, `StartEncounter()`, `StartGarrisonDefense()`, `ExitCombat()`, `EndEncounter()`, `RecordEncounterCompletion()` |
| `mind/encounter/encounter_trigger.go` | Encounter creation | `TriggerCombatFromThreat()`, `TriggerRandomEncounter()`, `TriggerGarrisonDefense()` |
| `mind/encounter/encounter_config.go` | Difficulty config | `getDifficultyModifier()`, `GetSquadPreferences()` |
| `mind/encounter/rewards.go` | Combat rewards | `calculateRewards()`, `grantRewards()`, `grantExperience()` |
| `mind/encounter/types.go` | Type definitions | `ActiveEncounter`, `CompletedEncounter`, `EncounterSpec`, `EnemySquadSpec`, `CombatResult`, `CombatExitReason` |

---

**End of Document**

For questions or clarifications, consult the source code or the [AI Algorithm Architecture](AI_ALGORITHM_ARCHITECTURE.md) document.
