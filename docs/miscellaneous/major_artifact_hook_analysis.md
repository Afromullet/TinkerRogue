# Major Artifact Hook Architecture Analysis

Can the perk system's hook architecture handle major artifacts, or do artifacts need a separate system?

**Source documents:**
- `docs/miscellaneous/perk_system_approach.md` -- Hook types, registry, runners, combat integration
- `docs/miscellaneous/equipment_system_approach.md` -- 14 major artifacts, charge model, system interactions

---

## 1. Perk Hook System Summary

The perk system (`perk_system_approach.md` sections 3-5) defines 7 typed hook function signatures:

| Hook Type | Signature Purpose | Fires Inside |
|-----------|-------------------|--------------|
| `DamageModHook` | Modify `DamageModifiers` before damage calc | `calculateDamage()` (squadcombat.go) |
| `TargetOverrideHook` | Replace target unit list | `processAttackWithModifiers()` (squadcombat.go) |
| `CounterModHook` | Suppress or modify counterattack | `ExecuteAttackAction()` (combatactionsystem.go:86-115) |
| `PostDamageHook` | React after damage recorded | `processAttackWithModifiers()` (squadcombat.go) |
| `TurnStartHook` | Fire at squad activation | `ResetSquadActions()` (turnmanager.go:72-99) |
| `CoverModHook` | Modify cover breakdown | `calculateDamage()` (squadcombat.go) |
| `DamageRedirectHook` | Intercept damage before recording | `processAttackWithModifiers()` (squadcombat.go) |

**Registry pattern:** Each perk ID maps to a `PerkHooks` struct with nil-able fields for each hook type. A global `hookRegistry` map stores these (`hook_registry.go`).

**Runner pattern:** Functions like `RunDamageModHooks()` iterate a unit's active perk IDs, look up hooks, and call non-nil entries. Runners are called from inside the combat resolution pipeline via callback injection to avoid circular imports.

**Key constraint:** All 7 hooks fire inside the **combat resolution pipeline** -- the code path from `ExecuteAttackAction` through `calculateDamage`, `processAttackWithModifiers`, and counterattack resolution. They modify values (damage multipliers, target lists, cover values) that flow through damage calculation. They are stateless, unit-level, and automatic.

---

## 2. Major Artifact Integration Point Mapping

All 14 major artifacts from `equipment_system_approach.md` mapped to the specific code locations they need to hook into.

### Group A: Active Player Abilities (8 artifacts)

These are **player-initiated actions** with targeting UI and charge tracking. The player chooses when to activate them, selects a target, and the artifact expends a charge.

| # | Artifact | Effect | Charge | Integration Point |
|---|----------|--------|--------|-------------------|
| 1 | **Double Time Drums** | One squad gets Move + Attack in one activation | Once/battle | Suppress `markSquadAsActed` (combatqueries.go:152-158) after first action |
| 2 | **Stand Down Orders** | Force enemy squad to skip Attack | Once/battle | Set target `HasActed = true` before their turn (combatqueries.go:152-158) |
| 3 | **Chain of Command Scepter** | Pass unused Attack to adjacent ally | Once/round | Toggle `HasActed` between two squads (combatqueries.go:152-158) |
| 4 | **Hourglass of Delay** | Enemy squad acts last within its faction's turn | Once/battle | Reorder faction-internal activation sequence (no existing code for this) |
| 9 | **Lockstep Banner** | Activate two adjacent squads simultaneously | Once/round | Dual activation window with shared action state consumption |
| 10 | **Saboteur's Hourglass** | Reduce ALL enemy squads' movement by 2 | Once/battle | Hook into `ResetSquadActions` (turnmanager.go:86-88) after movement init |
| 11 | **Anthem of Perseverance** | Bonus attack (no move) at end of faction turn | Once/battle | Insert between `EndTurn` factions (turnmanager.go:130-156) |
| 12 | **Deadlock Shackles** | Enemy squad skips entire next activation | Once/battle | Pre-set `HasActed = true` + `MovementRemaining = 0` in `ResetSquadActions` (turnmanager.go:83-88) |

**Common pattern:** The player makes an active decision during combat (select artifact, select target, confirm). This is a **new combat action type** -- not a hook that fires automatically during damage calculation. These artifacts need:

1. A UI flow: select artifact from available list, choose target (enemy squad, friendly squad, or self)
2. Charge tracking: once-per-battle or once-per-round state
3. Validation: is the target legal? Is the charge available?
4. Execution: modify `ActionStateData` fields or inject bonus activations

None of the 7 perk hooks support player-initiated activation with targeting UI.

### Group B: Triggered Passives (4 artifacts)

These fire automatically when a **turn-management-level event** occurs -- not during damage resolution, but during squad activation, destruction, or turn transitions.

| # | Artifact | Trigger Event | Effect | Integration Point |
|---|----------|---------------|--------|-------------------|
| 5 | **Forced Engagement Chains** | Enemy squad destroyed | Destroying squad gets 1-tile bonus move | After `RemoveSquadFromMap` (combatqueries.go:187-207) inside `ExecuteAttackAction` (combatactionsystem.go:144-151) |
| 7 | **Momentum Standard** | Enemy squad destroyed during your turn | Next squad to activate gets +1 movement | After `RemoveSquadFromMap`, then modify next squad's `ResetSquadActions` (turnmanager.go:86-88) |
| 8 | **Rallying War Horn** | Friendly squad is attacked | Grant one unacted friendly squad an out-of-turn activation | After `onAttackComplete` callback (combatactionsystem.go:168-170) when defender is player faction |
| 14 | **Echo Drums** | Friendly squad finishes full activation (moved + acted) | Squad gets second movement phase (no attack) | New event: `onSquadActivationComplete` -- does not exist yet |

**Common pattern:** These ARE hook-like, but they hook into the **turn management layer**, not the combat resolution pipeline. Their triggers are:

- Squad destroyed (`RemoveSquadFromMap` at combatqueries.go:187-207)
- Squad attacked (defender side of `onAttackComplete` at combatactionsystem.go:168-170)
- Squad activation complete (no existing callback)

Their effects alter **control flow** (grant bonus moves, inject activations, modify future squads' movement) rather than modifying values in a damage formula.

### Group C: Passive Always-On (2 artifacts)

These apply structural modifications at battle start or turn start, with no player decision during combat.

| # | Artifact | When Applied | Effect | Integration Point |
|---|----------|-------------|--------|-------------------|
| 4 | **Commander's Initiative Badge** | Battle start | Player faction always acts first | `InitializeCombat` (turnmanager.go:39): after `shuffleFactionOrder`, move player faction to index 0 of `TurnOrder` |
| 13 | **Vanguard's Oath** | Each round start | First friendly squad to activate gets +2 movement | `ResetSquadActions` (turnmanager.go:86-88): add +2 to first squad's `MovementRemaining` after normal init |

**Common pattern:** Simple modifications to existing initialization code. No hooks needed -- just conditional logic in `InitializeCombat` and `ResetSquadActions`.

---

## 3. Why Perk Hooks Are Not Sufficient

Five fundamental mismatches between the perk hook architecture and major artifact requirements:

### Mismatch 1: Layer

Perk hooks operate inside the **combat resolution pipeline** -- the code path that calculates damage, selects targets, applies cover, and resolves counterattacks. All 7 hook types fire within `calculateDamage()`, `processAttackWithModifiers()`, or `ExecuteAttackAction()`'s counterattack section.

Major artifacts operate on the **turn management layer** -- `TurnManager`, `ActionStateData`, activation sequencing, and turn transitions. The code locations are:

- `InitializeCombat` (turnmanager.go:35-70)
- `ResetSquadActions` (turnmanager.go:72-99)
- `EndTurn` (turnmanager.go:130-156)
- `markSquadAsActed` / `markSquadAsMoved` / `decrementMovementRemaining` (combatqueries.go:152-179)
- `RemoveSquadFromMap` (combatqueries.go:187-207)

These two layers share no code. A `DamageModHook` firing inside `calculateDamage()` cannot manipulate `ActionStateData` in a meaningful way. The hook fires per-unit during a single attack; artifacts need to manipulate squad-level action state across turns.

### Mismatch 2: Activation Model

Perk hooks fire **automatically** as part of normal combat flow. When a unit attacks, its damage mod hooks run. When damage is recorded, post-damage hooks run. The player has no control over when hooks execute.

8 of 14 major artifacts (Group A) are **player-activated abilities** with explicit targeting and timing decisions. The player chooses WHEN to activate Double Time Drums, WHO to target with Stand Down Orders, and WHICH squad receives Chain of Command's passed action. This requires:

- An activation UI (button in combat HUD)
- Target selection flow (click on squad)
- Confirmation and charge deduction
- Integration with the existing action selection system (Move / Attack / Cast Spell / **Use Artifact**)

The perk hook system has no concept of player activation. Adding one would fundamentally change what "hook" means in that architecture.

### Mismatch 3: Granularity

Perk hooks are **unit-level**. The `getActivePerkIDs()` function (perk_system_approach.md section 6) collects perk IDs from a specific unit's `UnitPerkData` plus its parent squad's `SquadPerkData`. Hook runners iterate per-unit:

```
RunDamageModHooks(attackerUnitID, defenderUnitID, &modifiers, manager)
```

Major artifacts are **squad-level or faction-level**. They operate on:

- Entire squads: "this squad gets a bonus attack" (Anthem of Perseverance)
- Enemy squads: "all enemy squads lose 2 movement" (Saboteur's Hourglass)
- Faction turn order: "your faction goes first" (Commander's Initiative Badge)
- Cross-squad relationships: "adjacent squad gets simultaneous activation" (Lockstep Banner)

Forcing artifacts through the unit-level hook runner would require collecting artifact IDs at the faction level, then somehow routing them through a per-unit iteration loop. This inverts the natural data flow.

### Mismatch 4: Response Type

Perk hooks **modify values in-place**. A `DamageModHook` adjusts `DamageModifiers.DamageMultiplier`. A `CoverModHook` adjusts `CoverBreakdown.TotalReduction`. A `TargetOverrideHook` returns a filtered target list. All hooks return to the caller, which continues with the modified values.

Major artifacts **alter control flow**:

- Rallying War Horn: interrupts the enemy turn, yields control to the player for a bonus activation, then resumes the enemy turn
- Lockstep Banner: two squads share a single activation window with both resolved before moving to the next activation
- Echo Drums: inserts a bonus movement phase after a squad's normal activation completes
- Anthem of Perseverance: inserts a bonus attack window between faction turns in `EndTurn`

These require suspending normal turn flow, injecting new activation sequences, and resuming -- a fundamentally different pattern from modifying a float and returning.

### Mismatch 5: Lifecycle

Perk hooks are **stateless**. `berserkerDamageMod` checks the attacker's current HP and modifies the multiplier. It stores nothing between calls. The hook registry is built once and never changes.

Major artifacts need **charge tracking**:

- Once-per-battle: Double Time Drums, Stand Down Orders, Saboteur's Hourglass, Anthem of Perseverance, Deadlock Shackles, Rallying War Horn
- Once-per-round: Chain of Command Scepter, Lockstep Banner, Echo Drums
- Passive (no charges): Commander's Initiative Badge, Vanguard's Oath
- Triggered (conditional): Forced Engagement Chains, Momentum Standard

Charge state must persist across turns, reset at round boundaries (for once-per-round), and be queryable by the UI (to grey out used artifacts). This is a component-level state management problem that the perk hook system explicitly avoids.

---

## 4. What CAN Be Reused

The perk hook system's architecture is sound and Go-idiomatic. Several patterns apply directly to artifacts:

### Pattern: Typed Function Signatures

The perk system defines each hook as a specific Go function type:

```go
type DamageModHook func(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager)
```

The same approach works for artifact triggers:

```go
type OnSquadDestroyedTrigger func(destroyedSquadID, destroyerSquadID ecs.EntityID,
    manager *common.EntityManager)

type OnTurnEndTrigger func(factionID ecs.EntityID, round int,
    manager *common.EntityManager)
```

### Pattern: Registry Map

The perk system uses `hookRegistry = map[string]*PerkHooks{}` to map perk IDs to their hook implementations. An artifact system can use the same pattern:

```go
var artifactRegistry = map[string]*ArtifactDefinition{}
```

### Pattern: Callback Injection

The perk system solves circular imports by defining function types in the `squads` package and passing implementations from `combat`. Artifacts face the same import challenge (artifact logic needs access to `TurnManager` and `ActionStateData`, but those are in `combat`). The same callback injection approach applies.

### Pattern: Caching at Battle Start

The perk system builds a cache of active perk IDs per unit at battle start (`CachedUnitPerks`). An artifact system should similarly resolve equipped artifacts at battle start and cache the results, avoiding per-turn component lookups.

---

## 5. Recommended Architecture

Three-part system, one for each artifact group.

### Part 1: Artifact Action System (Group A -- Active Artifacts)

A new combat action type alongside Move / Attack / Cast Spell: **Use Artifact**.

**Model:** Each artifact has an `Execute` function that takes the activating squad, a target, and the entity manager. The function validates the action, performs the effect, and deducts the charge.

**Components:**

```go
// On squad entity -- tracks equipped major artifact and charge state
type ArtifactChargeData struct {
    ArtifactID     string
    MaxCharges     int   // From artifact definition
    ChargesUsed    int
    ChargeType     int   // 0=per-battle, 1=per-round
}
```

**Execute pattern** (mirrors `ExecuteAttackAction` at combatactionsystem.go:38-173):

```go
func (as *ArtifactSystem) ExecuteArtifact(userSquadID, targetSquadID ecs.EntityID) error {
    // 1. Validate: artifact equipped, charge available, target legal
    // 2. Execute artifact-specific effect (switch on artifact ID)
    // 3. Deduct charge
    // 4. Fire post-artifact callback (for UI animation)
    return nil
}
```

**Per-artifact execution examples:**

- **Double Time Drums:** Set a flag on the squad's `ActionStateData` that prevents `markSquadAsActed` from setting `HasActed = true` after the next action. After both Move and Attack resolve, clear the flag and set `HasActed = true`.

- **Stand Down Orders:** Find target enemy squad's `ActionStateData` via `CombatQueryCache.FindActionStateBySquadID()`. Set `HasActed = true`. The squad can still move but cannot attack.

- **Chain of Command Scepter:** Source squad `HasActed = false` -> set to `true`. Target adjacent squad `HasActed = true` -> set to `false`. Target then takes a normal attack through the standard pipeline.

- **Deadlock Shackles:** Set a `SkipNextActivation` flag (new field on `ActionStateData` or tracked in artifact charge system). When `ResetSquadActions` fires for that squad, leave `HasActed = true` and `MovementRemaining = 0`.

- **Saboteur's Hourglass:** Iterate enemy squads via `GetSquadsForFaction()`, find each squad's `ActionStateData`, reduce `MovementRemaining` by 2 (min 0).

- **Lockstep Banner:** Flag the selected adjacent squad for simultaneous activation. The combat UI resolves both squads in a shared activation window before marking both `HasActed = true` and `HasMoved = true`.

- **Anthem of Perseverance:** At end of faction turn (before `EndTurn` advances to next faction), select a squad that has `HasActed = true`, set `HasActed = false`, open a bonus attack window (no movement -- `HasMoved` stays true, `MovementRemaining` stays 0). After the bonus attack, set `HasActed = true` and proceed with `EndTurn`.

- **Hourglass of Delay:** Reorder the faction-internal activation queue. Currently squads within a faction have no enforced activation order -- the player/AI picks freely. This artifact constrains enemy AI to activate the delayed squad last.

**UI integration:** Add an "Artifact" button alongside Move/Attack/Cast in the combat HUD (`gui/guicombat/`). When clicked, show available artifacts with charge state. Player selects artifact, then selects target. Follows the same selection flow as Attack (click enemy squad) or Cast (click target).

### Part 2: Turn Management Callbacks (Group B -- Triggered Artifacts)

Extend the existing callback system to support artifact triggers.

**Current callbacks** (single function each):

| Callback | Location | Signature |
|----------|----------|-----------|
| `onTurnEnd` | turnmanager.go:19 | `func(round int)` |
| `onAttackComplete` | combatactionsystem.go:18 | `func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)` |
| `onMoveComplete` | combatmovementsystem.go:19 | `func(squadID ecs.EntityID)` |

**Problem:** Each callback is a single function field. Setting a second callback overwrites the first. The GUI already uses these callbacks for animation updates.

**Option A: Extend to slices**

Change each callback field from a single function to a slice of functions:

```go
// Before
onAttackComplete func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)

// After
onAttackComplete []func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult)
```

Add `AddOnAttackComplete(fn)` alongside or replacing `SetOnAttackComplete(fn)`. Fire loop calls all registered functions.

**Option B: Dispatcher pattern**

Create an `ArtifactTriggerManager` that registers as the single callback and dispatches to artifact-specific handlers:

```go
type ArtifactTriggerManager struct {
    onSquadDestroyed []func(destroyedID, destroyerID ecs.EntityID)
    onSquadAttacked  []func(attackerID, defenderID ecs.EntityID)
    onActivationDone []func(squadID ecs.EntityID)
}
```

The trigger manager registers itself as `onAttackComplete` callback and routes events to artifact handlers.

**Recommendation:** Option A is simpler and more consistent with existing patterns. The existing `Set*` functions become `Add*` functions.

**New callback needed:** `onSquadActivationComplete` -- fires when a squad finishes its full activation (both `HasMoved = true` and `HasActed = true`). Required for Echo Drums. Could be fired from the combat UI layer after the player completes all actions for a squad.

**Per-artifact trigger wiring:**

- **Forced Engagement Chains:** Register on `onAttackComplete`. Check if `result.TargetDestroyed` and destroying squad has this artifact. If yes, set destroying squad's `MovementRemaining = 1`, `HasMoved = false`. After one move, restore to 0.

- **Momentum Standard:** Register on `onAttackComplete`. Check if `result.TargetDestroyed` during player faction's turn. Set a momentum flag. Next squad to begin activation gets `MovementRemaining += 1`.

- **Rallying War Horn:** Register on `onAttackComplete`. Check if defender is player faction. Player selects one unacted friendly squad. That squad gets an immediate bonus activation (move + attack), then is marked acted. Enemy turn resumes.

- **Echo Drums:** Register on `onSquadActivationComplete` (new callback). When a friendly squad has `HasMoved = true && HasActed = true`, offer the player the option to trigger Echo Drums. Reset `HasMoved = false`, set `MovementRemaining` to squad speed. `HasActed` stays true (no second attack). After bonus movement, set `HasMoved = true` and consume the round's charge.

### Part 3: Combat Initialization Hooks (Group C -- Passive Artifacts)

Simple conditional logic in existing functions. No new infrastructure needed.

**Commander's Initiative Badge:**

In `InitializeCombat` (turnmanager.go:35-70), after `shuffleFactionOrder(turnOrder)` at line 39:

```go
shuffleFactionOrder(turnOrder)

// If player has Commander's Initiative Badge, ensure they go first
if hasArtifact(playerFactionID, "commanders_initiative_badge", manager) {
    for i, fid := range turnOrder {
        if fid == playerFactionID {
            turnOrder[0], turnOrder[i] = turnOrder[i], turnOrder[0]
            break
        }
    }
}
```

**Vanguard's Oath:**

In `ResetSquadActions` (turnmanager.go:72-99), after the normal reset loop, apply +2 movement to the first squad:

```go
// After the for loop at lines 75-96
if isPlayerFaction && hasArtifact(factionID, "vanguards_oath", manager) {
    if len(factionSquads) > 0 {
        firstActionState := cache.FindActionStateBySquadID(factionSquads[0])
        if firstActionState != nil {
            firstActionState.MovementRemaining += 2
        }
    }
}
```

---

## 6. Comparison Table

| Dimension | Perk Hooks | Major Artifact System |
|-----------|-----------|----------------------|
| **Layer** | Combat resolution (damage calc, targeting, cover) | Turn management (action state, activation order, turn transitions) |
| **Activation** | Automatic (fires during combat flow) | Player-initiated (Group A) or event-triggered (Group B) |
| **Granularity** | Unit-level (iterate unit perk IDs) | Squad-level or faction-level |
| **Response** | Modify values in-place (multipliers, target lists) | Alter control flow (inject activations, skip turns, grant bonus actions) |
| **State** | Stateless (read entity state, return modified values) | Charge-based (once/battle, once/round, track usage) |
| **Pattern** | Registry + runners | Same registry pattern, different hook points |
| **Integration points** | `calculateDamage`, `processAttackWithModifiers`, counterattack section | `TurnManager`, `ActionStateData`, `RemoveSquadFromMap`, activation loop |
| **Code locations** | squadcombat.go, combatactionsystem.go:86-115 | turnmanager.go:35-156, combatqueries.go:117-207, new artifact system |
| **UI involvement** | None (invisible to player) | Full action UI for Group A (target selection, charge display) |

---

## 7. Implementation Complexity Assessment

### New Code

| Component | Estimate | Notes |
|-----------|----------|-------|
| `ArtifactChargeData` component + init | Small | New ECS component, follows existing patterns |
| Artifact definition loading (JSON) | Small | Mirrors perk/spell definition loading |
| `ExecuteArtifact` system function | Medium | One function with per-artifact switch, ~200 lines |
| Charge tracking (per-battle + per-round reset) | Small | Reset per-round charges in `ResetSquadActions` |
| Callback slice conversion (3 callbacks) | Small | Change field type, add `Add*` methods, update fire loops |
| New `onSquadActivationComplete` callback | Small | Fire from combat UI after squad finishes actions |
| GUI: artifact action button + selection | Medium-High | New mode in guicombat, target selection flow |
| Group C initialization hooks | Small | ~20 lines in turnmanager.go |

### Reused Patterns

- Registry map (from perk system)
- Callback injection (from perk system)
- JSON definition loading (from spell/perk systems)
- Action state manipulation (existing `markSquadAsActed`, `decrementMovementRemaining`)
- ECS component access patterns (project-wide)

### Risk Areas

**Highest risk: TurnManager modification.** `TurnManager` controls the core combat loop. Artifacts that inject bonus activations (Rallying War Horn, Anthem of Perseverance, Echo Drums) or reorder activation (Hourglass of Delay) require the turn manager to support interruptible, resumable activation sequences. Currently, the turn manager has no concept of "bonus activations" or "faction-internal ordering" -- it just tracks which faction's turn it is.

**Medium risk: Simultaneous activation (Lockstep Banner).** Two squads sharing an activation window means the combat UI must handle two active squads simultaneously. Currently the UI assumes one squad is active at a time.

**Lower risk: Everything else.** Stand Down Orders, Deadlock Shackles, Saboteur's Hourglass, and the Group C artifacts are straightforward `ActionStateData` modifications. Forced Engagement Chains and Momentum Standard are simple post-attack callbacks.

### Recommended Implementation Order

1. **Group C first** (Commander's Initiative Badge, Vanguard's Oath) -- 2 artifacts, minimal code, validates the artifact-equipped detection path
2. **Simple Group A next** (Stand Down Orders, Deadlock Shackles, Saboteur's Hourglass) -- straightforward `ActionStateData` manipulation, no bonus activations
3. **Simple Group B** (Forced Engagement Chains, Momentum Standard) -- callback-based, no turn interruption
4. **Complex Group A** (Double Time Drums, Chain of Command Scepter, Anthem of Perseverance) -- require suppressing/toggling action flags, bonus attack windows
5. **Control-flow Group B** (Rallying War Horn, Echo Drums) -- require interrupting normal turn flow
6. **Hardest** (Lockstep Banner, Hourglass of Delay) -- simultaneous activation and faction-internal ordering

---

## 8. Conclusion

The perk hook system's architecture is the right **pattern** for major artifacts -- typed functions, a registry map, callback injection -- but the wrong **layer**. Perk hooks modify combat resolution values (damage, targeting, cover) automatically at the unit level. Major artifacts manipulate turn management state (action availability, activation order, movement budget) at the squad/faction level, with player-initiated timing.

The recommendation is a **separate artifact system** that reuses the perk system's Go-idiomatic patterns (registry, typed callbacks, caching) but hooks into the turn management layer (`TurnManager`, `ActionStateData`, `ResetSquadActions`, `EndTurn`) rather than the combat resolution layer (`calculateDamage`, `processAttackWithModifiers`).

The two systems are complementary, not competing. Perks modify WHAT happens when combat resolves. Artifacts modify WHEN and HOW OFTEN combat happens. They share no integration points and cannot interfere with each other.
