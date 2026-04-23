# TinkerRogue Complexity Analysis — Combined Report

**Source report:** `resources/docs/complexity_report.txt` (generated 2026-04-22, commit `7fd505e`)
**Inputs:** Three independent specialist analyses
- `complexity_analysis_refactoring_pro.md` — refactoring expert lens
- `complexity_analysis_game_dev.md` — game-dev / performance lens
- `complexity_analysis_tactical_simplifier.md` — tactical systems lens

---

## Executive Summary

The codebase is healthy. Global averages (cyc 3.34, cog 5.18) are low and 83.8% of functions are cyc ≤ 5. Hotspots cluster in four known categories: procedural generation, GUI panel registration, input handling, and config validation. Most high numbers are inherent to the domain, not pain points.

### Unanimous HIGH (all three agents flagged)

1. **Split `CombatInputHandler.HandleInput`** at `gui/guicombat/combat_input_handler.go:112` (cyc 47, cog 76). Extract `handleSpellModeInput`, `handleArtifactModeInput`, `handleInspectModeInput`, `handleNormalInput` — by sub-mode, not by key. Per-frame during combat; real readability cost; no tactical depth loss. **Effort: small (1-2 hr).**
2. **De-duplicate bootstrap squad factories** in `testing/bootstrap/initial_squads.go`. `dupl` caught `createRangedSquad:168` ↔ `createCavalrySquad:324`; `createMagicSquad`, `createMixedSquad`, `createBalancedSquad` share the same shape. One `createSquadFromFilter` helper collapses ~150-200 LOC. **Effort: small (1-2 hr).**
3. **Extract `applyDeathOverride`** from `processAttack` at `tactical/combat/combatcore/combatprocessing.go:88-116` (nestif 13). Isolates Resolute logic without reordering the perk-hook pipeline. Accounts for the bulk of `processAttack`'s cog=57. **Effort: small (<1 hr).**

### Majority HIGH (2 of 3 flagged)

- **Decompose oversized panel-registry `init()` functions** (refactoring-pro and game-dev flagged HIGH; tactical-simplifier agreed at MEDIUM). Seven files: `combat_panels_registry.go:53` (341 LOC), `squadeditor_panels_registry.go:49` (384 LOC), `raid_panels_registry.go:19`, `artifact_panels_registry.go:19`, `squaddeployment_panels_registry.go:24`, `unitpurchase_panels_registry.go:24`, `exploration_panels_registry.go:74`. Extract each inline `OnCreate` closure to a named `buildXPanel(...) framework.PanelDescriptor` helper. **Effort: medium (~2-3 hr total).**
  - *Dissent:* refactoring-pro's Part C explicitly calls these "declarative registration tables" and lists them under Leave Alone, arguing splitting adds indirection without reducing volume. Game-dev and tactical-simplifier recommend extraction. The disagreement is aesthetic: table readability vs. init() line count.

### Unique Highest-Value Findings (do not drop)

- **[tactical-simplifier] Delete the dormant Leader Abilities subsystem.** `EquipAbilityToLeader` at `tactical/squads/squadcore/squadabilities.go:12` has zero production callers (grep confirms: one hit, its own definition). `CheckAndTriggerAbilities` runs every turn from **4 call paths** (`combatactionsystem.go:205, 209`, `turnmanager.go:70`, `turnmanager.go:105` — the `ResetSquadActions` path is distinct and scales with faction turns). Because `AbilitySlotComponent` is added unconditionally in `AddLeaderComponents` (`squadcreation.go:25-32`) but no slot is ever equipped, each call terminates cheaply on the first slot's `!IsEquipped` check — so the per-turn cost is small, not a full slot iteration. The case for deletion rests on mental-model weight, not performance. Additional evidence the subsystem was never production-ready: `combatabilities.go` contains four bare `fmt.Printf` debug statements, and `applyFireballEffect` directly mutates `common.Attributes.CurrentHealth` (`combatabilities.go:228`), bypassing combat math, perk hooks, and kill recording. Power evaluation (`power.go:104, 113-134` + `roles.go:44-46`) is safe to remove because `GetAbilityPowerValue` always returns 0 on unequipped slots. Removing eliminates ~300 LOC and collapses the power model from four layers to three. **Effort: medium (~1.5-2 hr). Risk: low.** Full checklist under action item 4.
- **[game-dev] Bitmask refactor of `CalculateCoverBreakdown` / `GetCoverProvidersFor`** at `tactical/combat/combatmath/combatcover.go:11, 69`. Per-attack `map[int]bool{}` allocations for 3-column grids → replace with `uint8` bitmask. Consolidates 9 nil/HasComponent guards into one loop. Complexity + perf win. **Effort: medium (~2-3 hr).**
- **[refactoring-pro] Extract `incrementThreatTypeStat`** in `campaign/overworld/overworldlog/overworld_summary.go:49, 120`. Three near-identical int/float64 JSON-unmarshaling blocks in `GenerateThreatSummary` (cog=52). One helper drops cog to ~15 and removes 30-40 lines. **Effort: small (~1 hr).**

### Unanimous SKIP

All three agents reached the same verdict independently:

- **`processAttack`'s perk-hook ordering** — load-bearing for Bloodlust/Resolute; comment at `combatprocessing.go:86-87` documents this. Extracting `applyDeathOverride` is fine; reordering the chain is a bug.
- **Worldgen internals** — `world/worldgen/gen_cavern.go` (cellular automata, drunkard's walk, erosion), `world/garrisongen/generator.go` + `dag.go`. Nested loops are the algorithms. Called once per floor.
- **`validate*` config checkers** — `templates/validation.go`, `templates/gameconfig.go`, `tactical/powers/perks/balanceconfig.go`. One-if-per-field sanity checks; table-driving loses field-named grep-ability.
- **Save/load chunks** — `setup/savesystem/chunks/*_chunk.go`. 1:1 schema mapping is the point.
- **`NewCombatService`** at `tactical/combat/combatservices/combat_service.go:63`. Single declared site of pipeline order (artifacts → perks → GUI).
- **`EventType.String()`** at `campaign/overworld/core/types.go:160`. Idiomatic Go enum→string switch.
  - *Dissent:* game-dev suggests `go:generate stringer` or `[]string` lookup. Low-stakes; either form is defensible.
- **`CombatAnimationMode.Update`** at `gui/guicombat/combat_animation_mode.go:341`. Legitimate 4-phase FSM.
- **All `tools/` code** — offline dev utilities, not gameplay.
- **`PositionalRiskLayer.getDirection`** at `mind/behavior/threat_positional.go:127`. 8-way direction classifier.

---

## Part A — Cyclomatic Complexity

### Worth refactoring

| File:Line | Function | Cyc | Cog | Agents | Verdict |
|---|---|---|---|---|---|
| `gui/guicombat/combat_input_handler.go:112` | `HandleInput` | 47 | 76 | all 3 | Split by sub-mode (HIGH) |
| `tactical/combat/combatcore/combatprocessing.go:24` | `processAttack` | 24 | 57 | all 3 | Extract `applyDeathOverride` only; keep pipeline intact (HIGH) |
| `tactical/combat/combatmath/combatcover.go:11` | `GetCoverProvidersFor` | 17 | 29 | game-dev | Bitmask + guard consolidation (MEDIUM, perf win) |
| `tactical/combat/combatmath/combatcover.go:69` | `CalculateCoverBreakdown` | 16 | 26 | game-dev | Same refactor as above |
| `tactical/combat/battlelog/battle_summary.go:284` | `generateSummaryText` | 19 | — | refactoring-pro | Extract `formatTargeting/Engagement/Outcome/Healing` (LOW) |
| `campaign/overworld/core/types.go:160` | `EventType.String()` | 19 | — | game-dev (for); refactoring-pro (against) | `stringer` if convenient; don't mandate |

### Complex but leave alone (unanimous)

- `CavernGenerator.carveDrunkardTunnel` (cyc 26), `cellularAutomataStep` (16), `erosionAccretionPass`, `checkWalkableRatio` (19), `placeStalactites` (16), `buildMST` (16) — `world/worldgen/gen_cavern.go`
- `GarrisonRaidGenerator.Generate` (cyc 37, cog 74) — `world/garrisongen/generator.go:37`. Optional: extract `placeRoomWithFallback` for lines 112-171 (refactoring-pro MEDIUM, optional).
- `BuildGarrisonDAG` (cyc 23), `carveZShapeCorridor` (26), `placeDoorway` (25) — `world/garrisongen/*.go`
- `validateOverworldConfig` (25), `validateGameConfig` (19), `validatePerkBalance` (24), all other `validate*` — `templates/*.go`, `tactical/powers/perks/balanceconfig.go`
- `EquipArtifact` (cyc 11) — `tactical/powers/artifacts/system.go:75`
- `ActionMap.ResolveInto` (cyc 19) — `gui/framework/actionmap.go:84`. Optional: extract `bindingActiveFor` helper (game-dev LOW).
- `NewCombatService` (cyc 14, 74 lines) — `tactical/combat/combatservices/combat_service.go:63`
- `LoadGame` (17), `SaveGame` (11), `MapChunk.Load` (14), `RaidChunk.Save` (14), `saveSquadMember` (17), `GearChunk.RemapIDs` (21) — `setup/savesystem/`
- `Grant` (17) — `mind/combatlifecycle/reward.go:48`
- `ExitCombat` (16) — `mind/encounter/encounter_service.go:137`
- `CreateSquadFromTemplate` (16) — `tactical/squads/squadcore/squadcreation.go:294`. Tactical-simplifier notes `map[string]bool{"row,col"}` at line 319 should become `[3][3]bool` (trivial cleanup).
- `ComputeGenericPatternFiltered` (15) — `tactical/squads/squadcore/squadqueries.go:345`
- `CheckAndTriggerAbilities` (14) — `tactical/combat/combatcore/combatabilities.go:14`. Will be deleted if Leader Abilities removed. Note: early-exits cheaply today (first slot's `!IsEquipped` check), so the per-turn runtime cost is small — delete for mental-model reasons, not performance.
- `applyDamageSpell` (11) — `tactical/powers/spells/system.go:150`
- `CombatAnimationMode.Update` (13), `ThreatVisualizer.Update` (11), `TileRenderer.Render` (13), `processRenderablesCore` (11), `RandomAnimator.Update` (13) — per-frame renderers, all already gated

---

## Part B — Cognitive Complexity

### Worth refactoring

| File:Line | Function | Cog | Agents | Action |
|---|---|---|---|---|
| `gui/guicombat/combat_input_handler.go:112` | `HandleInput` (spell-mode block at :120, nestif 16) | 76 | all 3 | Covered by Part A HIGH |
| `tactical/combat/combatcore/combatprocessing.go:88` | Death-override block in `processAttack` (nestif 13) | — | all 3 | Extract `applyDeathOverride` |
| `gui/guisquads/artifact_refresh.go:59` | `refreshInventory` | 46 | refactoring-pro, game-dev | Extract `buildInventoryEntries` + `onInventorySelect`. Siblings `refreshEquipment` (cog 21) and `refreshInventoryDetail` (cog 21) share shape; tactical-simplifier suggests unified `buildInstanceList` helper. **Effort: small-medium.** |
| `campaign/overworld/overworldlog/overworld_summary.go:49` | `GenerateThreatSummary` | 52 | refactoring-pro | Extract `incrementThreatTypeStat` for int/float64 JSON dance (also applies to `GenerateFactionSummary:120`, cog=25) |
| `world/worldgen/gen_cavern.go:579` | `checkWalkableRatio` | 49 | refactoring-pro | Extract only the "too-closed" 30-line branch into `relaxWallsForMinWalkableRatio`. Leave the "too-open" path inline. |

### Nesting that IS earned (unanimous skip)

- CA and erosion-accretion `for y { for x { for dy { for dx {` neighborhood scans
- `getDirection` 8-way ladder at `threat_positional.go:127`
- `computeFlankingRisk` nested enemy-squad × range scan — per AI turn, not per frame
- Tunnel-bias block in `gen_cavern.go:395` — drunkard's-walk step
- Panel-registry `init()` closures — cognitive load is from count, not nesting

---

## Part C — Structural Complexity

### C1. Duplication — `testing/bootstrap/initial_squads.go` (unanimous HIGH)

`dupl` flagged `createRangedSquad:168` ↔ `createCavalrySquad:324`. On inspection, `createMagicSquad:216`, `createMixedSquad:262`, `createBalancedSquad` all share the same filter→pick→position→leader→create shape. Extract:

```go
func buildSquadFromFilter(manager, name, formation, filter func() []UnitTemplate,
    gridPositions [][2]int, unitCount int, leadership int) (ecs.EntityID, error)
```

Collapses each variant to 5-10 lines; file shrinks from ~370 to ~150.

### C2. Panel-registry `init()` sprawl (majority HIGH, refactoring-pro dissents)

Seven files with 113-384 LOC `init()` functions. Each is a linear sequence of `framework.RegisterPanel(type, descriptor{...})` calls. Game-dev and tactical-simplifier recommend extracting each panel's `OnCreate` closure to a named `buildXPanel` helper. Refactoring-pro's Part C explicitly lists these under "Leave Alone," arguing they are declarative tables and splitting adds indirection without reducing volume.

**Resolution:** Treat as MEDIUM — majority view wins, refactoring-pro's Part C verdict is overridden. Extraction targets individual-panel grep-ability and reduces init() cognitive load from inline closures; file count stays constant. One helper per panel is enough — do not build further framework.

### C3. Leader Abilities — delete (tactical-simplifier unique HIGH)

Dormant subsystem with no production callers. Full deletion surface:

- `combatabilities.go` (entire file) — executor + four `applyXEffect` helpers with bare `fmt.Printf` debug lines; `applyFireballEffect:228` directly mutates `attr.CurrentHealth`, bypassing combat math / perk hooks / kill recording
- `squadabilities.go` (entire file) — `EquipAbilityToLeader`, never called
- `squadcomponents.go:185-289` — `AbilitySlotData`, `AbilityType` (Rally/Heal/BattleCry/Fireball), `TriggerType`, `CooldownTrackerData`, `AbilityParams`
- `squadmanager.go:37` — component registration line for `AbilitySlotComponent` + cooldown tracker
- `squadcreation.go:25-32` — `AddLeaderComponents` and `RemoveLeaderComponents` must stop adding/removing the ability slot + cooldown components
- `squad_chunk.go:275-291, 417-434` — save/load serialization of `AbilitySlotData` and `CooldownTrackerData`
- `power.go:104, 113-134` — `calculateAbilityValue` call and definition (always returns 0 today; safe to remove)
- `roles.go:44-46` — `GetAbilityPowerValue` definition
- `turnmanager.go:70`, `turnmanager.go:105`, `combatactionsystem.go:205, 209` — 4 call sites of `CheckAndTriggerAbilities`
- `squads_test.go:112` — test fixture that calls `AddLeaderComponents`; verify after deletion
- `savesystem.go:51` — bump `CurrentSaveVersion` from 1 to 2. No migration function needed: Go's `json.Unmarshal` ignores unknown fields, so old saves containing default-zero ability data round-trip cleanly

Rally/BattleCry can convert to perk `TurnStart` hooks later if desired; Heal/Fireball already live in spells. Deleting collapses the power model from four layers (perks/artifacts/spells/abilities) to three.

### C4. Cover calculation perf + complexity (game-dev unique, MEDIUM)

`tactical/combat/combatmath/combatcover.go:11, 69` — both invoked per damage calculation from `combatcalculation.go:160`. Fresh `map[int]bool{}` allocations per defender per attack. A 3-bit bitmask (`uint8`) replaces the map; guard consolidation reduces cyc from 17 → ~10. Drops allocations and GC pressure.

### C5. `CombatMode.Initialize` — split (refactoring-pro MEDIUM)

`gui/guicombat/combatmode.go:89`, 113 lines. Does service wiring, action-map setup, sub-menu construction, panel building, panel wiring, and visualization wiring. Extract `wireControllers`, `wirePanels`, `wireVisualization`. Keep `Initialize` as a 3-line orchestrator.

### C6. `gui/builders/panels.go` — optional MEDIUM (refactoring-pro)

`BuildTypedPanel`'s spec-vs-manual branch (nestif-8 at line 394) → extract `buildPanelOptsFromConfig(config) []PanelOption`. `BuildPanel`'s responsive-padding → `applyResponsivePadding(config)`. Small readability win.

### C7. Clean separations that MUST NOT collapse (unanimous)

- `tactical/powers/effects/` as stat-modifier primitive consumed by 3 layers
- `tactical/powers/powercore/` as crosscutting runtime for artifacts/perks (do NOT absorb spells — different lifecycle shape)
- `combatstate` / `combatcore` / `combatservices` split (state / behavior / orchestration)
- `squads/squadcore/` vs `combat/*` (squad-structural vs combat-turn)
- `mind/behavior/threat_*` layer split (independently-tunable strategic axes)
- Perks ≠ Artifacts ≠ Spells — different hook surfaces by design; `powercore` already unifies what should unify

### C8. Structural smells rejected by all three

- Reflection-based save systems — hides schema, breaks migrations
- Unifying `PerkBalance` / `artifacts.BalanceConfig` / `PowerConfig` — per-system balance is a designer feature, not duplication
- Collapsing `Grant`'s reward-type blocks into a loop — flat list is the designer-readable spec
- Merging `threat_*` layers in `mind/behavior/` — grid iteration duplication is the cost of decoupling

---

## Performance Considerations (game-dev lens)

1. **`HandleInput`** — per-frame; `ActionMap.ResolveInto` is O(1). Verify during profiling that `ResolveInto` is called once per frame, not per `ActionActive`.
2. **`CalculateCoverBreakdown` + `GetCoverProvidersFor`** — per-attack map allocations. Bitmask refactor is the single highest-ROI perf + complexity win.
3. **`PositionalRiskLayer.Compute`** — full-grid passes per AI turn, not per frame. Not urgent unless many factions or larger maps arrive.
4. **`TileRenderer.Render`** — already dirty-cache gated.
5. **`processRenderablesCore`** — per frame, per entity; viewport-mode dispatch is cheap.
6. **VFX `RandomAnimator.Update`** — per visible VFX; 5 zero-gated channels; fine.
7. **Save/load** — high complexity, very low frequency. Skip.

---

## Mental Model Assessment (tactical-simplifier lens)

| Subsystem | Clarity | Justification |
|---|---|---|
| Combat core | Clear | Only muddy spot: 6 nil-dispatcher guards in `processAttack` |
| Squads | Clear | Pure-data components, query/system split intact; dormant abilities is the only smell |
| Powers overall | Mostly clear | `powercore` + `progression.library` dedupe well; phantom 4th layer (abilities) is the remaining smell |
| Perks internal | Clear | Behaviors taxonomy documented; shared `run()` helper |
| Artifacts internal | Clear | Small documented interface; explicit charge tracker |
| Progression | Clear | `library` pattern at `library.go:24` is textbook |
| AI | Muddled | Layer abstractions clean; `threat_positional.go`'s grid-iterating methods could tighten but content is tactical depth |
| Combat lifecycle | Clear | Data-driven reward/grant resolver |
| Combat GUI | Tangled at input layer | `HandleInput` cog 76 is the single worst mental bottleneck; rest is fine |
| Encounter/Spawning | Clear | Clean trigger/setup/resolve phase split |

---

## Unified Prioritized Action List

### HIGH

1. **Split `CombatInputHandler.HandleInput`** (`gui/guicombat/combat_input_handler.go:112`) into `handleSpellModeInput` / `handleArtifactModeInput` / `handleInspectModeInput` / `handleNormalInput`. Method extraction on `*CombatInputHandler` receiver — no new parameter passing needed. Verify with `go build` + manual combat test (no automated GUI tests exist). **Effort: small (1-2 hr). Risk: low.** [all 3]
2. **Extract `buildSquadFromFilter` helper** in `testing/bootstrap/initial_squads.go`. Test-only code (`testing/bootstrap/`) so a seasoned engineer may reasonably defer until the file is touched for another reason. Run `go test ./testing/...`. **Effort: small (1-2 hr). Risk: low.** [all 3]
3. **Extract `applyDeathOverride`** from `processAttack` lines 88-116. Signature: `applyDeathOverride(result *combattypes.CombatResult, manager *common.EntityManager, defenderID, defenderSquadID ecs.EntityID, event *combattypes.AttackEvent)`. The ordering comment at `combatprocessing.go:86-87` is load-bearing and must move into the extracted function's doc comment, not remain at the call site. Expected cog reduction on `processAttack`: 57 → 35-45 (still elevated; intentional — the perk-hook rhythm stays). Run `go test ./tactical/combat/...` before and after (existing coverage in `combatexecution_test.go`, `combat_service_test.go`). **Effort: small (<1 hr). Risk: medium.** [all 3]
4. **Delete Leader Abilities subsystem.** Full checklist in C3 above — 11 deletion sites plus save version bump in `savesystem.go:51` (1 → 2). No migration function needed (JSON unmarshal ignores unknown fields). Run `go build ./...` to surface compile errors reactively; fix `squads_test.go:112` fixture. **Effort: medium (~1.5-2 hr). Risk: low.** [tactical-simplifier]

### MEDIUM

5. **Bitmask refactor of `CalculateCoverBreakdown` / `GetCoverProvidersFor`** in `tactical/combat/combatmath/combatcover.go:11, 69`. Complexity + perf win. **Pre-requisite:** `tactical/combat/combatmath` currently has **no `_test.go` file**. Write at minimum two table-driven tests covering both functions before touching the implementation. Budget ~1 hour for test writing before the 2-3 hr refactor. **Effort: medium (3-4 hr total). Risk: medium.** [game-dev]
6. **Refactor `refreshInventory` / `refreshEquipment` / `refreshInventoryDetail`** in `gui/guisquads/artifact_refresh.go:59`. Extract `buildInventoryEntries` / `onInventorySelect` / shared `buildInstanceList`. **Effort: small-medium. Risk: low.** [refactoring-pro, game-dev, tactical-simplifier]
7. **Extract `incrementThreatTypeStat`** in `campaign/overworld/overworldlog/overworld_summary.go:49, 120`. Applies to both `GenerateThreatSummary` and `GenerateFactionSummary`. **Effort: small (~1 hr). Risk: low.** [refactoring-pro]
8. **Decompose panel-registry `init()` blocks** — 7 files. Extract each `OnCreate` closure to a named helper. **Effort: medium (~2-3 hr total). Risk: low** (init-only). [game-dev, tactical-simplifier]
9. **Split `CombatMode.Initialize`** at `gui/guicombat/combatmode.go:89` into `wireControllers`, `wirePanels`, `wireVisualization`. **Effort: small (~45 min). Risk: low.** [refactoring-pro]
10. **Extract `relaxWallsForMinWalkableRatio`** from `world/worldgen/gen_cavern.go:579` (too-closed branch only). **Effort: small (~30 min). Risk: low.** [refactoring-pro]
11. **Extract `placeRoomWithFallback`** from `world/garrisongen/generator.go:37` lines 112-171. Cuts 60 lines without breaking phase sequencing. **Effort: small. Risk: low.** [refactoring-pro]
12. **Replace `map[string]bool{"row,col"}`** in `tactical/squads/squadcore/squadcreation.go:319` with `[3][3]bool`. **Effort: trivial. Risk: low.** [tactical-simplifier]

### LOW

13. **Split `BuildTypedPanel` spec-vs-manual branch** and extract `applyResponsivePadding` in `gui/builders/panels.go:394`. **Effort: small. Risk: low.** [refactoring-pro]
14. **Split `generateSummaryText`** in `tactical/combat/battlelog/battle_summary.go:284`. **Effort: small. Risk: low.** [refactoring-pro]
15. **Table-drive `CameraController.HandleInput`** at `input/cameracontroller.go:28` — 8 directional bindings. **Effort: small. Risk: low.** [game-dev]
16. **Extract `bindingActiveFor` helper** from `ActionMap.ResolveInto` at `gui/framework/actionmap.go:84`. **Effort: small. Risk: low.** [game-dev]
17. **Introduce `NoopPerkDispatcher`** to eliminate `if dispatcher != nil` guards in `processAttack`. **Effort: small. Risk: low-medium** — every hook call-site uses the noop. [tactical-simplifier]
18. **Grid-occupancy validation extraction** from `CreateSquadFromTemplate`. Only if touched for other reasons. [refactoring-pro]
19. **`EventType.String()`** → `go:generate stringer` or `[]string` lookup. **Effort: trivial.** [game-dev] (refactoring-pro explicitly opposes; low-stakes either way)

### SKIP (unanimous)

- All `gen_cavern.go` procgen internals — earned
- All `garrisongen/` internals (DAG, corridor, doorway) — earned
- `templates/validation.go`, `gameconfig.go`, `difficulty.go`, `perks/balanceconfig.go` — flat per-field checks
- `setup/savesystem/chunks/*` — 1:1 schema mapping
- `tools/combat_*` — offline dev utilities
- `NewCombatService` — pipeline ordering is the contract
- `CombatAnimationMode.Update` — legitimate 4-phase FSM
- `PositionalRiskLayer.getDirection` — 8-way classifier
- `Grant` — flat reward-type list is designer-readable spec
- `ExitCombat` — inherent domain dispatch
- `EquipArtifact` — 11 branches = 11 distinct failure modes
- `newListResources` / `newTextAreaResources` — ebitenui theming, declarative
- Threat layers in `mind/behavior/threat_*` — independently-tunable axes

---

## Anti-Recommendations

Union of the three agents' explicit warnings against changes that look appealing but should not happen.

**AR1. Do NOT flatten `processAttack`'s perk-hook chain.** The ordering (target override → attacker mod → defender mod → damage redirect → apply → death override → post-damage) is load-bearing for Bloodlust/Resolute interaction; `combatprocessing.go:86-87` documents this. A chain-of-responsibility abstraction just relocates and hides the contract.

**AR2. Do NOT extract inner loops of `cellularAutomataStep`, `erosionAccretionPass`, `carveDrunkardTunnel`, `carveZShapeCorridor`.** Classical PCG kernels. Extraction imposes per-cell call overhead and hides the algorithm.

**AR3. Do NOT refactor `CombatAnimationMode.Update` into a generic state-machine framework.** Four phases; infrastructure-for-nothing.

**AR4. Do NOT collapse `validate*` per-field checks into a loop or DSL.** Each check carries a specific field-named warning — that's the value. Grep-ability beats abstraction here.

**AR5. Do NOT refactor `EquipArtifact` / `LoadPerkDefinitions` / `CreateSquadFromTemplate` for cyclomatic reduction alone.** Branches map to distinct failure modes surfaced to the player / loader / editor.

**AR6. Do NOT "optimize" `TileRenderer.Render` or `processRenderablesCore`.** Already dirty-cache gated, per-image batched. Per-frame cost is low.

**AR7. Do NOT refactor `BuildGarrisonDAG` or `Generate`.** The DAG stages share mutable state (`dag`, `nextID`, `hasRestRoom`, `currentTotal`, `targetTotal`). Extracting to builder-struct methods relocates those 5 variables to fields for zero win.

**AR8. Do NOT introduce reflection-based save/load.** Hides on-disk schema from diffs and makes migrations harder. `funlen` is wrong to complain about 91-line `saveSquadMember`.

**AR9. Do NOT unify perks + artifacts + spells into a single "power behavior" interface.** `powers/artifacts/behavior.go:43` (charge + Activate) vs `powers/perks/hooks.go:33` (9 damage-pipeline hooks) vs spells (mana + library + intentionally bypass perk hooks, documented at `spells/system.go:88-93`) encode different tactical axes. `powercore` already shares crosscutting concerns correctly.

**AR10. Do NOT merge `mind/behavior/threat_*` layers.** Independently-tunable strategic axes. Apparent grid-iteration duplication is the cost of decoupling.

**AR11. Do NOT flatten `combatstate` / `combatcore` / `combatservices` into one package.** The split enforces ECS rules.

**AR12. Do NOT hide `NewCombatService` pipeline order behind config.** The explicit calls at `combat_service.go:103-143` are the only statement of artifacts→perks→GUI ordering.

**AR13. Do NOT unify `PerkBalance` / `artifacts.BalanceConfig` / `PowerConfig`.** Per-system balance structs let designers tune one axis without touching others.

**AR14. Do NOT collapse `Grant`'s reward-type blocks into a loop.** The flat list is a designer-readable spec of currencies.

**AR15. Do NOT split `HandleInput` by key.** Split by sub-mode (spell, artifact, inspect, normal) — each has its own input contract. Per-key splitting produces methods that each re-query mode state.

---

## Closing Note

The distribution (83.8% of functions at cyc ≤ 5) says the project has discipline. The HIGH list is small, tightly scoped, and high-signal. Do those four HIGH items, then the MEDIUM cover-calculation refactor for the perf win, then reassess. Resist any urge to "fix" procgen, perk pipelines, state machines, or validators — those earn their complexity.

**Realistic time budget:** The HIGH list is ~6-8 hours of real work (not 4), driven mostly by the Leader Abilities deletion surface and the `processAttack` test-run-before-and-after discipline. The combatmath bitmask refactor (MEDIUM item 5) adds ~1 hour of test-writing before the refactor itself because no tests currently exist for that package. Plan accordingly.

---

## Changelog

- **2026-04-23 — Karen review pass.** Expanded Leader Abilities deletion surface from ~60% to full checklist (11 sites + save version bump); corrected `CheckAndTriggerAbilities` call count (3 → 4 paths); softened per-turn performance framing (early-exits are cheap — delete for mental-model reasons); flagged missing `tactical/combat/combatmath` test file as pre-requisite for cover bitmask refactor; added explicit `applyDeathOverride` signature and `go test ./tactical/combat/...` verification step; clarified refactoring-pro's Part C dissent on panel registries was overridden by majority. Review: `complexity_analysis_karen_review.md`.

---

## Source Documents

- `resources/docs/complexity_analysis_refactoring_pro.md` — refactoring expert lens, detailed per-function verdicts
- `resources/docs/complexity_analysis_game_dev.md` — game-dev / performance lens, hot-path vs cold-path breakdown
- `resources/docs/complexity_analysis_tactical_simplifier.md` — tactical systems lens, mental model assessment, Leader Abilities deep-dive
- `resources/docs/complexity_analysis_karen_review.md` — reality check; claim verification via grep/read; executability gaps
- `resources/docs/complexity_report.txt` — raw metrics
