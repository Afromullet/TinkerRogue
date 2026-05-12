# Environmental Hazards System — Design

**Status:** Proposed — design only, not implemented
**Author:** Drafted with Claude
**Date:** 2026-05-12

---

## 1. Context

TinkerRogue currently has no concept of tile-bound environmental effects. Combat happens on a squad grid where one squad occupies one world-map tile, but tiles themselves are inert — they only block or allow movement. This document proposes adding **9 hazard types** that live on world-map tiles and trigger when squads enter, occupy, or move near them.

### Why

- Hazards add tactical depth that no existing system expresses. Persistent threats independent of unit actions force route planning, not just target selection.
- They give designers a new content lever — encounter-specific terrain (volcanic battlefield, frozen plains, blighted ruins) with mechanical teeth.
- They unlock a new spell category (place-hazard spells like ice walls and fire patches), expanding the spell design space.

### Intended outcome

A new `tactical/powers/hazards/` package plus minimal edits to combat, spells, effects, and rendering. Hazards are ECS entities (one per occupied tile) carrying a `HazardComponent`. They trigger via the existing `powerPipeline` — the same dispatch layer the artifact and perk systems already use — so combat lifecycle integration costs ~5 lines in `combat_service.go`. Stat-modifier hazards (Ice / Rot / Siphon) reuse `effects.ActiveEffect` verbatim. Mechanic hazards (Fire DoT, Tentacles immobilize, Euclidian teleport, Pullers pull, Time Warp skip-turn, Spell Warp disable-spells) dispatch through a new `HazardKind`-keyed handler registry.

---

## 2. Confirmed design decisions

- **Granularity**: world-map tile only, one squad per tile, whole-squad triggering. No per-unit cell granularity inside squads.
- **Placement sources**: encounter generation, spells, and runtime conditions (artifacts / perks) — three thin wrappers around one canonical `CreateHazard`.
- **Effect model**: reuse `ActiveEffect` for stat hazards; new dispatch layer for mechanic hazards. Both can chain.
- **Time Warp** = skip affected squad's next turn (set `HasMoved=true, HasActed=true` in next `ResetSquadActions`).
- **One hazard per tile** in v1. `CreateHazard` rejects placement on an already-occupied hazard tile.
- **Refresh duration on re-entry** for stat-modifier hazards (standard TRPG convention).
- **No friendly-fire filtering** in v1 — spell-placed hazards affect all squads (preserves tactical risk). Field reserved on definitions for future opt-in.
- **Pullers chain prevention** via a `RecentlyPulled` flag on `PendingHazardActionsData` that suppresses re-pull for one move (prevents two-Puller infinite loops).

---

## 3. New package layout: `tactical/powers/hazards/`

| File | Purpose |
|---|---|
| `components.go` | `HazardKind` enum, `HazardData`, `HazardSource`, `PendingHazardActionsData`, ECS component handles |
| `system.go` | `CreateHazard` canonical factory + three placement wrappers + `OnSquadMoved` + `OnPostReset` + `ExpireHazard` |
| `dispatch.go` | `HazardContext`, `HazardHandler`, `RegisterHandler`, `Dispatch`, 9 per-kind handler implementations |
| `queries.go` | `GetHazardAt`, `GetHazardsInRadius`, `HazardIsActive` |
| `init.go` | `common.RegisterSubsystem` callback — creates components/tags and registers all handlers |
| `hazards_test.go` | 3 unit tests per hazard (trigger / expiry / re-entry semantics) |

---

## 4. Components (pure data, per ECS rules in CLAUDE.md)

```go
// HazardKind dispatches handlers via a value-keyed map (50x faster than pointer keys).
type HazardKind int

const (
    HazardIce HazardKind = iota
    HazardRot
    HazardFire
    HazardTentacles
    HazardSiphon
    HazardEuclidian
    HazardPullers
    HazardTimeWarp
    HazardSpellWarp
)

// HazardSource records the placement origin for telemetry / future filtering.
type HazardSource int

const (
    HazardSourceEncounter HazardSource = iota
    HazardSourceSpell
    HazardSourceCondition
)

// HazardData is the ECS component attached to each hazard entity.
type HazardData struct {
    Kind           HazardKind
    DefinitionID   string         // FK into templates.HazardRegistry
    Duration       int            // turns of effect on victims
    Damage         int            // for DoT hazards (Fire)
    Radius         int            // for OnProximity hazards (Pullers)
    Persistent     bool           // false = consume-on-trigger
    Charges        int            // remaining triggers if !Persistent (-1 = infinite)
    OwnerFactionID ecs.EntityID   // 0 = neutral
    SourceTag      HazardSource
    VFXType        string
}

// PendingHazardActionsData is attached to a squad entity as a queue of
// mechanic-hazard effects pending resolution by the turn manager and movement system.
type PendingHazardActionsData struct {
    SkipNextTurn   bool
    Immobilized    int  // turns remaining (cannot move; can still act)
    SpellsDisabled int  // turns remaining
    RecentlyPulled bool // suppress chain pull for one move
}

// ECS handles populated by init.go
var (
    HazardComponent               *ecs.Component
    HazardTag                     ecs.Tag
    PendingHazardActionsComponent *ecs.Component
    PendingHazardActionsTag       ecs.Tag
)
```

The `effects` package gets exactly one new constant:

```go
// in tactical/powers/effects/components.go
const (
    SourceSpell EffectSource = iota
    SourceAbility
    SourcePerk
    SourceItem
    SourceEnvironment   // NEW — hazard-induced stat modifiers
)
```

That is the **only** edit to the effects package. All stat-modifier hazards reuse `effects.ApplyEffectToUnits` with `Source: SourceEnvironment`.

---

## 5. Dispatch layer

A `HazardKind`-keyed registry of trigger-event handlers. Value-keyed map per CLAUDE.md.

```go
type TriggerEvent int
const (
    TriggerOnEnter TriggerEvent = iota
    TriggerOnOccupy
    TriggerOnProximity
    TriggerOnExpire
)

type HazardContext struct {
    Manager        *common.EntityManager
    HazardEntityID ecs.EntityID
    HazardData     *HazardData
    VictimSquadID  ecs.EntityID
    VictimUnitIDs  []ecs.EntityID   // pre-resolved via squadcore.GetUnitIDsInSquad
    Trigger        TriggerEvent
}

type HazardHandler func(ctx *HazardContext)

var handlers = map[HazardKind]map[TriggerEvent]HazardHandler{}

func RegisterHandler(kind HazardKind, trig TriggerEvent, h HazardHandler)
func Dispatch(ctx *HazardContext)  // looks up handlers[Kind][Trigger]; no-op if missing
```

Handlers are registered in `init.go` at subsystem-init time. Per-kind handler outlines:

- `handleIceOnEnter` — applies `ActiveEffect{Stat: StatMovementSpeed, Modifier: -2, RemainingTurns: 2, Source: SourceEnvironment}` to all unit IDs via `effects.ApplyEffectToUnits`.
- `handleRotOnEnter` — applies two `ActiveEffect`s (Strength −2, Dexterity −2, 3t).
- `handleSiphonOnEnter` — applies Magic debuff (−3, 2t).
- `handleFireOnEnter` / `handleFireOnOccupy` — applies HP damage directly (loop unit IDs, decrement `Attributes.CurrentHealth`, call casualty cleanup). No `ActiveEffect` — damage is instant per tick.
- `handleTentaclesOnEnter` — gets/creates `PendingHazardActionsData` on victim squad, sets `Immobilized = Duration`.
- `handleEuclidianOnEnter` — calls `teleportSquadToRandomValidTile(squadID, manager)` (queries position system for unoccupied tiles in current map). Marks hazard `Charges--`.
- `handlePullersOnProximity` — uses `posSystem.GetEntitiesInRadius(hazardPos, radius)`, filters for squads, calls `pullSquadAdjacent(squadID, hazardPos, manager)` which finds an adjacent free tile. Sets `RecentlyPulled = true` to suppress chains.
- `handleTimeWarpOnEnter` — sets `PendingHazardActionsData.SkipNextTurn = true` on victim squad. The turn manager checks this flag in `ResetSquadActions`.
- `handleSpellWarpOnEnter` — sets `PendingHazardActionsData.SpellsDisabled = Duration`. The spell-cast gate (Section 7) reads this.

After each handler runs, dispatch decrements `Charges` on non-persistent hazards and disposes the hazard entity when `Charges == 0`.

---

## 6. Trigger lifecycle (integration sites)

| Trigger | Existing hook | File:line | New subscriber |
|---|---|---|---|
| **OnEnter** | `powerPipeline.OnMoveComplete(squadID)` | `tactical/combat/combatservices/combat_service.go:134-143` | `hazards.OnSquadMoved(squadID, manager)` — query position system for hazard at squad's new position, call `Dispatch(TriggerOnEnter)` |
| **OnOccupy** | `powerPipeline.OnPostReset(factionID, squadIDs)` | `combat_service.go:103-108` (fires `TurnManager.postResetHook` from `turnmanager.go:108-111`) | `hazards.OnPostReset(factionID, squadIDs, manager)` — for each squad, check if standing on a hazard with `onOccupy` trigger, dispatch |
| **OnProximity** | `powerPipeline.OnMoveComplete(squadID)` | same as OnEnter | After OnEnter dispatch, also scan `posSystem.GetEntitiesInRadius(squadPos, MaxPullerRadius)` for Pullers hazards. Single pass keeps integration thin |
| **OnExpire** | Internal — invoked by dispatch when `Charges == 0` or by `OnPostReset` when `Duration` ticks to zero on a persistent hazard | `hazards/system.go` | `hazards.ExpireHazard(entityID, manager)` — calls `manager.CleanDisposeEntity` |

The hazards package subscribes during `init.go` by registering callbacks into `powerPipeline` via the same pattern used by `perkDispatcher` (see `combat_service.go:103-143`). **Two new subscriber lines** in `combat_service.go`:

```go
cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) {
    hazards.OnSquadMoved(squadID, cs.EntityManager)
})
cs.powerPipeline.OnPostReset(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
    hazards.OnPostReset(factionID, squadIDs, cs.EntityManager)
})
```

---

## 7. Edits to existing files

| File | Edit |
|---|---|
| `tactical/powers/effects/components.go` | Add `SourceEnvironment` to the `EffectSource` const block |
| `tactical/combat/combatservices/combat_service.go` (~line 143) | Add two `powerPipeline` subscriptions (above). Mirrors existing artifact / perk dispatch wiring at lines 103-143 |
| `tactical/combat/combatcore/turnmanager.go` (`ResetSquadActions`, lines 81-114) | After `effects.TickEffectsForUnits` (line 102), read each squad's `PendingHazardActionsData`. If `SkipNextTurn`, set `actionState.HasMoved=true, HasActed=true` and clear flag. Decrement `Immobilized` and `SpellsDisabled` counters. Clear `RecentlyPulled` |
| `tactical/combat/combatcore/combatmovementsystem.go` (`MoveSquad` entry, ~line 58-83) | If squad's `PendingHazardActionsData.Immobilized > 0`, reject move with "squad is immobilized" |
| `tactical/powers/spells/system.go` (`CastSpell` dispatch) | (a) Add `EffectPlaceHazard` case that calls `hazards.PlaceHazardFromSpell` for each target tile. (b) Gate cast: if caster squad has `SpellsDisabled > 0`, reject with error |
| `templates/spelldefinitions.go` | Add `EffectPlaceHazard SpellEffectType = "place_hazard"` constant and `HazardToPlace HazardID` field on `SpellDefinition` |
| `setup/gamesetup/` (wherever `LoadSpellDefinitions()` is called) | Add `LoadHazardDefinitions()` call alongside |
| `mind/encounter/` | Designer-driven calls to `PlaceHazardFromEncounter` during encounter setup (specific call sites depend on encounter content) |

### New asset files

- `templates/hazarddefinitions.go` — `HazardDefinition`, `HazardRegistry`, `LoadHazardDefinitions` (mirrors `spelldefinitions.go`)
- `gamedata/hazarddata.json` — 9 hazard definitions (see appendix)
- `visual/combatrender/hazardoverlays.go` — `HazardTileRenderer`, mirrors `MovementTileRenderer` at `visual/combatrender/combatoverlays.go:13-31`. Wired into the combat render pipeline (`gui/guicombat/`) **before** unit rendering so units draw on top

---

## 8. Existing utilities reused (no new infrastructure)

| Utility | File:line | Used for |
|---|---|---|
| `effects.ApplyEffectToUnits` | `tactical/powers/effects/system.go:46-52` | Ice / Rot / Siphon stat modifiers |
| `effects.TickEffectsForUnits` | `tactical/powers/effects/system.go:100-106` | Already runs at turn boundary; no change |
| `powerPipeline.OnMoveComplete` | `tactical/combat/combatservices/combat_service.go:134-143` | OnEnter and OnProximity trigger hook |
| `powerPipeline.OnPostReset` | `tactical/combat/combatservices/combat_service.go:103-108` (fires from `turnmanager.go:108-111`) | OnOccupy and per-turn hazard decrement |
| `GlobalPositionSystem.GetEntityIDAt` / `GetEntitiesInRadius` | `core/common/positionsystem.go` | Hazard lookup at squad position; Pullers radius scan |
| `squadcore.GetUnitIDsInSquad` | (used at `turnmanager.go:101`) | Resolve victim units for effect application |
| `manager.CleanDisposeEntity` | per CLAUDE.md | Hazard cleanup on expiry |
| `common.RegisterSubsystem` | per CLAUDE.md (see `tactical/powers/effects/init.go:9-14`) | Self-register hazard components at startup |
| `JSONTargetArea` shape pattern | `templates/spelldefinitions.go` | Reference for spell-placed multi-tile hazards (caster targets a shape, spell loops over cells calling `PlaceHazardFromSpell`) |

---

## 9. Per-hazard behavior

| Hazard | Triggers | Stat modifiers (ActiveEffect, `SourceEnvironment`) | Mechanic action | N | Damage | Consumed? |
|---|---|---|---|---|---|---|
| **Ice** | OnEnter | MovementSpeed −2, 2t | — | 2 | — | Persistent |
| **Rot** | OnEnter | Strength −2, Dexterity −2, 3t | — | 3 | — | Persistent |
| **Fire** | OnEnter, OnOccupy | — | Direct HP damage to all units in squad | 3 (tile duration) | 4/tick | Persistent (ticks down each round) |
| **Tentacles** | OnEnter | — | Set `Immobilized = 2` on victim squad | 2 | — | Persistent |
| **Siphon** | OnEnter | Magic −3, 2t | — | 2 | — | Persistent |
| **Euclidian** | OnEnter | — | Teleport squad to random valid tile (in-bounds, traversable, no squad, no hazard) | 0 | — | **Consumed** (1 charge, consumed even on no-op) |
| **Pullers** | OnProximity (R=3) | — | Pull squad adjacent to puller (find free tile via posSystem) | 0 | — | Persistent |
| **Time Warp** | OnEnter | — | Set `SkipNextTurn = true` on victim squad | 0 | — | **Consumed** (1 charge) |
| **Spell Warp** | OnEnter | — | Set `SpellsDisabled = 3` on victim squad | 3 | — | Persistent |

Stat-modifier rows are mechanically identical to spell buff/debuff resolution. Fire, Tentacles, Euclidian, Pullers, Time Warp, Spell Warp are the dispatch-layer mechanics.

---

## 10. Placement APIs

One canonical factory, three thin wrappers tagging source for telemetry / filtering:

```go
// CreateHazard is the canonical hazard-entity factory.
// Validates pos is in-bounds and not already occupied by another hazard.
// Creates an ECS entity with PositionComponent + HazardComponent.
// Registers in GlobalPositionSystem.AddEntity.
// Returns the new hazard entity ID, or 0 on failure.
func CreateHazard(
    def *templates.HazardDefinition,
    pos coords.LogicalPosition,
    source HazardSource,
    ownerFactionID ecs.EntityID,
    manager *common.EntityManager,
) ecs.EntityID

// (1) Encounter generation — called from mind/encounter/ or mind/spawning/ during map setup
func PlaceHazardFromEncounter(defID templates.HazardID,
    pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID

// (2) Spell effect — invoked by spell system on EffectPlaceHazard
func PlaceHazardFromSpell(defID templates.HazardID,
    pos coords.LogicalPosition, casterFactionID ecs.EntityID,
    manager *common.EntityManager) ecs.EntityID

// (3) Runtime condition (artifact / perk hook)
func PlaceHazardFromCondition(defID templates.HazardID,
    pos coords.LogicalPosition, ownerFactionID ecs.EntityID,
    manager *common.EntityManager) ecs.EntityID
```

### Spell-effect extension

Add new constant in `templates/spelldefinitions.go`:

```go
EffectPlaceHazard SpellEffectType = "place_hazard"
```

Extend `SpellDefinition` with one optional field:

```go
HazardToPlace HazardID `json:"hazardToPlace,omitempty"`
```

In `tactical/powers/spells/system.go` add a new branch alongside existing buff/debuff and damage paths:

```go
func applyPlaceHazardSpell(spell *templates.SpellDefinition,
    targetTiles []coords.LogicalPosition, casterFactionID ecs.EntityID,
    result *SpellCastResult, manager *common.EntityManager) {
    for _, tile := range targetTiles {
        hazards.PlaceHazardFromSpell(spell.HazardToPlace, tile, casterFactionID, manager)
    }
}
```

Dispatched from the existing `CastSpell` switch on `spell.EffectType`.

### Condition-driven placement

Artifacts / perks that drop hazards call `PlaceHazardFromCondition` directly from their handler functions (e.g. an "Ice Trail" perk that runs on `OnMoveComplete` and drops `haz_ice` on the squad's previous tile). No new infrastructure — just call the API.

---

## 11. Rendering integration

### New: `visual/combatrender/hazardoverlays.go`

Mirror of `MovementTileRenderer` (`combatoverlays.go:13-31`):

```go
type HazardTileRenderer struct {
    viewport   rendering.CachedViewport
    iconCache  map[string]*ebiten.Image    // keyed by VFXType
    fillColors map[HazardKind]color.Color  // fallback per-kind tint
}

func NewHazardTileRenderer() *HazardTileRenderer

// Render queries all entities with HazardComponent, draws a tile overlay
// (color tint based on Kind) plus an optional VFX icon from iconCache.
func (htr *HazardTileRenderer) Render(screen *ebiten.Image,
    centerPos coords.LogicalPosition, manager *common.EntityManager)
```

### Wire-in

In the combat-mode render pipeline (`gui/guicombat/`), wherever `MovementTileRenderer.Render` is called, add a `HazardTileRenderer.Render` call **before** unit rendering so units draw on top.

Animated effects (flame for Fire, frost crystals for Ice) can be added later via the existing `vfx` package — attach a `BaseShape` VFX to the hazard entity at creation time in `CreateHazard`.

---

## 12. Testing strategy

### Fixtures (`testing/`)

- `CreateTestHazard(kind HazardKind, pos coords.LogicalPosition, manager) ecs.EntityID` — hand-crafted `HazardData` for unit tests, bypasses JSON.
- `InitTestHazardRegistry()` — seeds `templates.HazardRegistry` with all 9 defaults so dispatch-handler tests can resolve definitions without disk I/O.

Per existing memory rule: these helpers gated on `config.DEBUG_MODE` in production code; freely callable from `_test.go`.

### Unit tests (per hazard)

In `tactical/powers/hazards/hazards_test.go`, three tests per hazard:

1. `TestHazard_<Kind>_OnTrigger` — place hazard, move test squad onto tile (or within radius for Pullers), assert expected state (stat modifier present / HP reduced / `PendingHazardActionsData` flag set / position changed for Euclidian / Pullers).
2. `TestHazard_<Kind>_Expiry` — tick `Duration` rounds via `TurnManager.ResetSquadActions`; assert hazard disposed (non-persistent) or `ActiveEffect` reversed (stat-modifier hazards).
3. `TestHazard_<Kind>_NoDoubleApply` — re-enter while effect active; assert refresh semantics (replace duration, don't stack).

### Integration tests

- `TestHazardPlacement_FromSpell` — cast a `place_hazard` spell with `HazardToPlace = "haz_fire"`; assert hazard entity created at target tile with `SourceTag = HazardSourceSpell`.
- `TestHazardPlacement_FromEncounter` — encounter generates hazards via `PlaceHazardFromEncounter`; assert positions match.
- `TestHazardChain_PullersOntoFire` — Pullers drags squad onto Fire; assert both resolve (pull then OnEnter Fire).
- `TestHazardChain_TwoPullersNoInfiniteLoop` — two Pullers within range of each other; assert `RecentlyPulled` flag prevents bounce.
- `TestSpellWarp_BlocksCast` — apply Spell Warp, attempt cast; assert rejection.
- `TestTimeWarp_SkipsNextTurn` — trigger Time Warp; call `ResetSquadActions`; assert `HasMoved && HasActed` set, flag cleared.
- `TestTentacles_BlocksMove` — trigger Tentacles; attempt move; assert rejection. Tick to expiry; assert move succeeds.

### Build verification

```
go build -o game_main/game_main.exe game_main/*.go
go test ./tactical/powers/hazards/...
go test ./tactical/powers/effects/...
go test ./tactical/combat/...
go test ./tactical/powers/spells/...
```

### Manual playtest

1. Boot `./game_main/game_main.exe`, enter a debug encounter, place each hazard manually via debug content (gated on `config.DEBUG_MODE`).
2. Verify `HazardTileRenderer` draws overlays correctly per `VFXType`; verify overlays disappear when hazards expire.
3. Verify Pullers chain prevention by placing two Pullers within range of each other and stepping into one.

---

## 13. Open questions / risks

1. **Stacking semantics.** If a squad enters a tile already occupied by a hazard *of the same kind it already suffers from* — refresh, stack, or no-op? **Recommendation:** refresh duration on re-entry (simplest, matches typical TRPG conventions). Marked explicit in design.
2. **Multiple hazards per tile.** Decision: **one hazard entity per tile.** `CreateHazard` rejects placement on an already-occupied tile. Allows OnOccupy to be unambiguous. Alternative (stack via `GetAllEntityIDsAt`) deferred until needed.
3. **Euclidian valid-tile definition.** Proposed: tile that is (a) in-bounds, (b) traversable terrain per worldmap, (c) no squad present, (d) no hazard present. Falls back to no-op (and consumes charge) if no valid tile within map bounds after N retries.
4. **Pullers chain reaction.** If Pullers drags squad onto another hazard, run OnEnter for the new tile? **Yes** — same `OnSquadMoved` callback path, since post-pull we synthesize a `MoveComplete` event. Risk: infinite loop if two Pullers face each other. **Mitigation:** `RecentlyPulled` flag on `PendingHazardActionsData` suppresses re-pull for one move.
5. **Spell Warp vs queued spells.** Spell-cast is synchronous per-action in this codebase (no queue), so `SpellsDisabled` simply gates the cast at command time. Confirmed acceptable.
6. **Two hazards in one move.** Movement is single-tile resolution (squad teleports atomically to destination, no path-walk), so a single move triggers at most one OnEnter — clean. Time Warp + Tentacles on same tile applies both (each via its own handler).
7. **Friendly-fire on owner-placed hazards.** `OwnerFactionID` filtering: should a spell-placed Fire hurt the caster's own squad that steps on it? **Recommendation:** yes (no faction filtering) — keeps tactical risk meaningful and simplifies dispatch. Can be added as `def.AffectsOwner bool` later.
8. **Time Warp interaction with EndTurn.** `SkipNextTurn` is consumed in `ResetSquadActions` (sets HasMoved / HasActed). If the squad's faction has multiple squads, only this squad skips — others act normally. Confirm with playtest.
9. **Order of OnPostReset events.** `effects.TickEffects` runs first (line 102), then `PendingHazardActionsData` decrement, then ability checks (line 105). Hazard ticking must run between effects and abilities so abilities see the correct skip-turn state.

---

## 14. Effort estimate

- **Lines of code:** ~600-800 (new package ~500, integration edits ~100, JSON ~50, tests ~250)
- **Time:** 8-12 hours
- **Complexity:** Medium — leverages existing pipeline, no new infrastructure
- **Risk level:** Low-Medium. Highest risk is Pullers chain reactions and stacking edge cases (mitigated by `RecentlyPulled` flag and one-hazard-per-tile rule).

---

## Appendix A — hazard JSON definitions

`gamedata/hazarddata.json`:

```json
{"hazards":[
 {"id":"haz_ice","kind":"ice","name":"Ice Patch","duration":2,"persistent":true,"charges":-1,
  "statModifiers":[{"stat":"movementspeed","modifier":-2}],"vfxType":"frost","triggers":["onEnter"]},

 {"id":"haz_rot","kind":"rot","name":"Rot Bloom","duration":3,"persistent":true,"charges":-1,
  "statModifiers":[{"stat":"strength","modifier":-2},{"stat":"dexterity","modifier":-2}],"vfxType":"blight","triggers":["onEnter"]},

 {"id":"haz_fire","kind":"fire","name":"Inferno","duration":3,"damage":4,"persistent":true,"charges":-1,
  "vfxType":"flame","triggers":["onEnter","onOccupy"]},

 {"id":"haz_tentacles","kind":"tentacles","name":"Tentacle Pit","duration":2,"persistent":true,"charges":-1,
  "vfxType":"tendril","triggers":["onEnter"]},

 {"id":"haz_siphon","kind":"siphon","name":"Mana Siphon","duration":2,"persistent":true,"charges":-1,
  "statModifiers":[{"stat":"magic","modifier":-3}],"vfxType":"drain","triggers":["onEnter"]},

 {"id":"haz_euclid","kind":"euclidian","name":"Euclidian Rift","duration":0,"persistent":false,"charges":1,
  "vfxType":"warp","triggers":["onEnter"]},

 {"id":"haz_pullers","kind":"pullers","name":"Pull Anchor","duration":0,"radius":3,"persistent":true,"charges":-1,
  "vfxType":"chain","triggers":["onProximity"]},

 {"id":"haz_timewarp","kind":"timewarp","name":"Time Warp","duration":0,"persistent":false,"charges":1,
  "vfxType":"chrono","triggers":["onEnter"]},

 {"id":"haz_spellwarp","kind":"spellwarp","name":"Spell Warp","duration":3,"persistent":true,"charges":-1,
  "vfxType":"null","triggers":["onEnter"]}
]}
```
