# combat_analysis — Refactoring Notes

## 1. Overview

`tools/combat_analysis/` houses four standalone CLI tools used to generate,
analyze, and inspect tactical combat data. They are intentionally headless
(no rendering, no input) and share a small `shared/` package for the data
formats and helpers they all need.

| Tool | Role |
| --- | --- |
| `combat_simulator` | Runs headless combat scenarios and writes per-battle JSON logs |
| `combat_balance` | Reads battle JSON logs and aggregates matchup statistics into a CSV report |
| `combat_visualizer` | Reads battle JSON logs and renders ASCII visualizations of each engagement |
| `report_compressor` | Reads the balance CSV and produces a compressed unit-overview / matchup / alerts report |

Data flow:

```
combat_simulator
      │
      ▼
 battle JSON logs
      │
      ├──────────────────────────────┐
      ▼                              ▼
 combat_balance ─► balance CSV   combat_visualizer ─► ASCII output
                       │
                       ▼
                report_compressor ─► compressed CSV
```

## 2. Duplication Eliminated

| Type / Function | Was In | Now In |
| --- | --- | --- |
| `BattleRecord` | `combat_balance/types.go`, `combat_visualizer/types.go` | `shared/battletypes.go` (visualizer keeps a local wrapper because its `EngagementRecord` carries an extra `Summary` field) |
| `EngagementRecord` | `combat_balance/types.go`, `combat_visualizer/types.go` | `shared/battletypes.go` (visualizer defines a Summary-augmented variant locally) |
| `CombatLog` | `combat_balance/types.go`, `combat_visualizer/types.go` | `shared/battletypes.go` |
| `UnitSnapshot` | `combat_balance/types.go`, `combat_visualizer/types.go` | `shared/battletypes.go` |
| `HealEvent` | `combat_balance/types.go`, `combat_visualizer/types.go` | `shared/battletypes.go` |
| `AttackEvent` | `combat_balance/types.go` | `shared/battletypes.go` |
| `TargetInfo` | `combat_balance/types.go` | `shared/battletypes.go` |
| `HitResult` | `combat_balance/types.go` | `shared/battletypes.go` |
| `LoadBattleRecord` | `combat_balance/loader.go`, `combat_visualizer/loader.go` | `shared/loader.go` (the visualizer keeps a tiny local loader because it parses into its Summary-aware struct) |
| `FindAllBattles` | `combat_balance/loader.go`, `combat_visualizer/loader.go` | `shared/loader.go` |
| `FindLatestBattle` | `combat_visualizer/loader.go` | `shared/loader.go` |
| `SafeDiv` (formerly `safeDiv` / `safeRate` / `safeAvg`) | `combat_balance/csv_writer.go`, `report_compressor/writer.go` | `shared/mathutil.go` |

In `combat_balance/types.go`, the moved types are re-exposed via type aliases
(`type BattleRecord = sharedtypes.BattleRecord`, etc.) so existing balance
code referring to them by their bare names compiles unchanged.

## 3. `shared/` Package Reference

Import path: `game_main/tools/combat_analysis/shared`

### `battletypes.go`

The canonical battle-log data types serialized by the simulator and consumed
by the analyzers.

| Type | Purpose |
| --- | --- |
| `BattleRecord` | Root JSON document for a single battle: ID, timestamps, victor, engagements |
| `EngagementRecord` | One squad-vs-squad exchange inside a battle (index, round, combat log) |
| `CombatLog` | The detailed combat data for an engagement (units, attack events, heal events) |
| `UnitSnapshot` | Per-unit state captured at the moment of combat (ID, name, grid position, role) |
| `HealEvent` | One healer→target heal action with HP delta |
| `AttackEvent` | One attacker→defender attack with target info, hit roll outcome, damage, and kill flag |
| `TargetInfo` | Targeting data for an attack (mode and target row/column) |
| `HitResult` | Resolution of an attack roll (hit/dodge/crit thresholds and rolls) |

Note: `CombatLog` carries `AttackEvents` even though the visualizer ignores
them. Keeping a single canonical shape is simpler than two near-duplicate
structs; downstream consumers just don't read the field they don't need.

### `loader.go`

Battle-file I/O helpers shared by every analyzer.

| Function | When to use |
| --- | --- |
| `LoadBattleRecord(path)` | Parse a single JSON file into `*BattleRecord`. Use when you have a specific path. |
| `FindAllBattles(dir)` | List all `*.json` files in `dir`, sorted by name (which sorts chronologically given the `battle_YYYYMMDD_HHMMSS.json` naming). Use for batch processing. |
| `FindLatestBattle(dir)` | Return the most recent battle file path. Use for "show me the last run" CLI flags. |

The visualizer keeps its own `LoadBattleRecord` — it is the one analyzer
whose root type (`visualizer.BattleRecord`) carries an extra Summary field
on each engagement, so it cannot be parsed via the shared loader.

### `mathutil.go`

| Function | Purpose |
| --- | --- |
| `SafeDiv(num, denom)` | Returns `num / denom`, or `0.0` when `denom == 0`. Replaces the per-tool `safeDiv` / `safeRate` / `safeAvg` copies that all did the same thing. |

## 4. Squad Factory Pattern

`combat_simulator/squad_factory.go` previously held five near-duplicate
factory functions (`createBalancedSquad`, `createRangedSquad`,
`createMagicSquad`, `createMixedSquad`, `createMeleeSquad`) and three
near-identical filter helpers
(`filterUnitsByAttackRange`, `filterUnitsByAttackType`, `filterUnitsByRole`).

The shared scaffolding now consists of:

```go
type squadConfig struct {
    formation  squadcore.FormationType
    positions  [][2]int
    unitCount  int
    selectUnit func(i int) unitdefs.UnitTemplate
}

func buildSquad(manager *common.EntityManager, name string, pos coords.LogicalPosition, cfg squadConfig) ecs.EntityID
func filterUnits(predicate func(unitdefs.UnitTemplate) bool) []unitdefs.UnitTemplate
```

`buildSquad` handles the loop that copies each template, sets its grid
position, randomly assigns the leader, applies the leadership bonus, and
calls `createSimSquad`. `filterUnits` replaces all three per-attribute
filters with a single predicate-driven helper.

To add a new squad type, write a thin wrapper that builds the config:

```go
func createScoutSquad(manager *common.EntityManager, name string, worldPos coords.LogicalPosition) ecs.EntityID {
    pool := filterUnits(func(u unitdefs.UnitTemplate) bool {
        return u.Role == unitdefs.RoleScout
    })
    if len(pool) == 0 {
        return createBalancedSquad(manager, name, worldPos)
    }
    return buildSquad(manager, name, worldPos, squadConfig{
        formation: squadcore.FormationRanged,
        positions: [][2]int{{0, 0}, {1, 1}, {2, 2}},
        unitCount: 3,
        selectUnit: func(i int) unitdefs.UnitTemplate {
            return pool[common.RandomInt(len(pool))]
        },
    })
}
```

The new factory is ~10 lines instead of ~30 and shares all leader/position
logic with the existing factories.

## 5. Adding New Tools

When you add a new analyzer under `tools/combat_analysis/`:

1. Import `game_main/tools/combat_analysis/shared` for battle-file I/O
   (`shared.LoadBattleRecord`, `shared.FindAllBattles`,
   `shared.FindLatestBattle`) and the canonical battle types.
2. Define your own aggregation/output types (matchup keys, summary structs,
   CSV row shapes, etc.) inside your tool's package — those are
   tool-specific and don't belong in `shared`.
3. Use `shared.SafeDiv` for any rate/average calculation that needs to be
   safe against zero denominators.
4. If your analyzer needs a wider view of an engagement than the
   shared `EngagementRecord` provides (e.g., the visualizer's `Summary`
   field), define a local `BattleRecord` / `EngagementRecord` pair that
   still uses `shared.CombatLog` and the other unchanged shared types,
   plus a small local loader that parses into your local type.

## 6. Architecture

```
                                    ┌───────────────────────────────┐
                                    │     tools/combat_analysis     │
                                    └───────────────────────────────┘
                                                   │
                  ┌────────────────┬──────── shared package ───────┬────────────────┐
                  │                │   battletypes / loader / math │                │
                  │                │                               │                │
                  ▼                ▼                               ▼                ▼
         combat_simulator   combat_balance                 combat_visualizer   report_compressor
                  │                │                               │                │
                  │                ▲                               │                ▲
                  │                │                               │                │
                  ▼                │                               ▼                │
           battle JSON logs ──────┘                          ASCII output           │
                                   │                                                │
                                   └─► balance CSV ────────────────────────────────┘
                                                                  │
                                                                  ▼
                                                          compressed CSV
```

- `combat_simulator` writes the only inputs to the analyzers (the JSON
  logs).
- `combat_balance` and `combat_visualizer` read those logs independently;
  they do not share state at runtime.
- `report_compressor` consumes the CSV produced by `combat_balance` — it
  never reads battle JSON directly.
- All four tools depend on `shared/` for the canonical battle types, the
  file I/O helpers, and the safe-division helper.
