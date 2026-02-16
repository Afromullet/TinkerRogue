# Artifact Implementation Status & Difficulty Assessment

Overview of all 14 major artifacts and 7 minor artifacts: what's done, what's remaining, and how hard each remaining piece is.

---

## Minor Artifacts (7 total) — FULLY COMPLETE

All 7 minor artifacts are passive stat sticks. They work end-to-end:
- JSON definitions in `artifactdata.json`
- `ApplyArtifactStatEffects()` applies `ActiveEffect` entries at battle start
- `RemoveAllEffects()` cleans them up at battle end
- Equip/unequip GUI exists in `gui/guisquads/artifactmode.go`
- Full test coverage

| Artifact | Effect | Status |
|----------|--------|--------|
| Iron Bulwark | +2 Armor | Done |
| Keen Edge Whetstone | +2 Weapon | Done |
| Fleet Runner's Sandals | +1 MovementSpeed | Done |
| Marksman's Scope | +1 AttackRange | Done |
| Berserker's Torc | +2 Strength, -1 Armor | Done |
| Sentinel's Plate | +2 Armor, -1 MovementSpeed | Done |
| Duelist's Gloves | +1 Dexterity, +1 Strength | Done |

No remaining work.

---

## Major Artifacts — Status Summary

### Fully Integrated (behavior + combat wiring + tests)

These artifacts have behavior code, are wired into combat callbacks, and work end-to-end at the logic layer. The only missing piece for all of them is the **combat GUI activation button** (for player-activated ones).

| # | Artifact | Behavior Key | Behavior Code | Combat Wiring | Tests | GUI |
|---|----------|-------------|---------------|---------------|-------|-----|
| 1 | Commander's Initiative Badge | `initiative_first` | Directly in `combat_service.go:142-152` | `forceFirstFactionID` param in `InitializeCombat` | Via combat tests | N/A (passive) |
| 2 | Vanguard's Oath | `vanguard_movement` | `VanguardMovementBehavior` | `postResetHook` in `combat_service.go:305-308` | Yes | N/A (passive) |
| 3 | Double Time Drums | `double_time` | `DoubleTimeBehavior` | `DoubleTimeActive` flag in `ActionStateData`, consumed in `markSquadAsActed` | Yes | **Missing** |
| 4 | Stand Down Orders | `stand_down` | `StandDownBehavior` | Pending effects via `OnPostReset` hook | Yes | **Missing** |
| 5 | Chain of Command Scepter | `chain_of_command` | `ChainOfCommandBehavior` | Adjacency validation, action state swap | Yes | **Missing** |
| 6 | Forced Engagement Chains | `engagement_chains` | `EngagementChainsBehavior` | `OnAttackComplete` via `combat_service.go:311-316` | Yes | N/A (triggered) |
| 7 | Momentum Standard | `momentum_standard` | `MomentumStandardBehavior` | `OnAttackComplete` via `combat_service.go:311-316` | Yes | N/A (triggered) |
| 8 | Saboteur's Hourglass | `saboteurs_hourglass` | `SaboteursHourglassBehavior` | Pending effects via `OnPostReset` hook | Yes | **Missing** |
| 9 | Anthem of Perseverance | `anthem_perseverance` | `AnthemPerseveranceBehavior` | Direct `ActionStateData` manipulation | Yes | **Missing** |
| 10 | Deadlock Shackles | `deadlock_shackles` | `DeadlockShacklesBehavior` | Pending effects via `OnPostReset` hook | Yes | **Missing** |
| 11 | Rallying War Horn | `rallying_horn` | `RallyingHornBehavior` | `OnAttackComplete` via `combat_service.go:311-316` | Yes | N/A (triggered) |
| 12 | Echo Drums | `echo_drums` | `EchoDrumsBehavior` | `OnAttackComplete` via `combat_service.go:311-316` | Yes | N/A (triggered) |

### NOT Implemented

| # | Artifact | Behavior Key | Status |
|---|----------|-------------|--------|
| 13 | **Hourglass of Delay** | `hourglass_delay` | No behavior constant, no struct, no code |
| 14 | **Lockstep Banner** | `lockstep_banner` | No behavior constant, no struct, no code |

---

## Remaining Work — Detailed Breakdown

### 1. Hourglass of Delay — HARD

**Effect:** Once per battle, force one enemy squad to act last in its faction's turn.

**Why it's hard:** The current turn system has no concept of faction-internal activation order. Squads within a faction can be freely activated in any order by the player/AI. This artifact requires:

- A new constraint system that forces a specific squad to be activated last within its faction's turn
- For AI factions: modify AI squad selection to defer the delayed squad until all others have acted
- For player factions: either enforce activation order in the GUI (grey out the delayed squad until others are done) or warn/block the player from activating it early
- A new `BehaviorHourglassDelay` constant and struct
- Integration with `ActionStateData` or a new field to track "delayed" status
- The activation order constraint needs to survive the full turn cycle (not just one moment)

**Estimated scope:** New behavior struct + new field on `ActionStateData` (e.g., `DelayedUntilLast bool`) + AI awareness + GUI enforcement. Touches `gear/`, `tactical/combat/`, `tactical/ai/`, and `gui/guicombat/`.

**Difficulty: 7/10** — The AI and GUI enforcement of activation ordering is the hard part. The behavior logic itself is straightforward.

---

### 2. Lockstep Banner — VERY HARD

**Effect:** Once per round, activate two adjacent friendly squads simultaneously.

**Why it's hard:** This is the hardest artifact in the entire set. The combat system fundamentally assumes one squad is active at a time:

- The combat GUI (`gui/guicombat/`) manages a single "selected squad" with move/attack actions
- The AI turn system processes one squad at a time
- `ActionStateData` is per-squad and actions are resolved sequentially
- "Simultaneous activation" means the UI needs to handle two squads sharing an activation window: the player can move/attack with either squad in any order, and both are marked as done when the window closes

**What's needed:**
- A new `BehaviorLockstepBanner` constant and struct
- A "dual activation" mode in the combat UI where two squads are highlighted and the player can switch between them
- Logic to track which of the two linked squads have completed their actions
- Both squads marked acted/moved only when the shared window closes
- AI handling for simultaneous dual-squad activation (if AI faction has this artifact)

**Estimated scope:** New behavior struct + significant GUI combat mode changes + dual-squad state tracking + AI support. This is the only artifact that requires a fundamentally new UI interaction paradigm.

**Difficulty: 9/10** — Requires rethinking the single-active-squad assumption in both the GUI and AI layers.

---

### 3. Combat GUI — Artifact Activation Button — MEDIUM

**What's needed:** 5 artifacts are player-activated (`double_time`, `stand_down`, `chain_of_command`, `saboteurs_hourglass`, `anthem_perseverance`, `deadlock_shackles`) and their behavior code works, but there's no way for the player to trigger them during combat.

**Required work:**
- Add an "Artifact" button to the combat HUD alongside Move/Attack/Cast
- When clicked, show available artifacts with charge state (greyed out if used)
- Target selection flow: some target enemy squads (Stand Down, Deadlock, Saboteur's), some target friendly squads (Double Time, Anthem, Chain of Command)
- Call `gear.ActivateArtifact()` with the selected behavior and target
- The `BehaviorContext` needs to be constructed with the `CombatService`'s `chargeTracker`

**Where it fits:** New mode in `gui/guicombat/` following the same pattern as the existing attack/move/cast selection flows.

**Difficulty: 5/10** — The combat UI already has the selection pattern (click squad to target). This is mostly following existing patterns. The `ActivateArtifact()` function and charge checking already exist.

---

## Difficulty Summary

| Remaining Work | Difficulty | Dependencies |
|----------------|-----------|--------------|
| Combat GUI artifact activation button | 5/10 (Medium) | None — all behavior code is ready |
| Hourglass of Delay behavior | 7/10 (Hard) | Needs GUI button + AI activation order awareness |
| Lockstep Banner behavior | 9/10 (Very Hard) | Needs GUI button + dual-squad UI mode + AI support |

### Recommended Implementation Order

1. **Combat GUI artifact button** (5/10) — Unlocks all 6 player-activated artifacts that already have working behavior code. Biggest value-to-effort ratio.
2. **Hourglass of Delay** (7/10) — Hard but scoped. The activation order constraint is a new concept but doesn't break existing assumptions.
3. **Lockstep Banner** (9/10) — Save for last. Requires the most architectural change. Consider whether this artifact's gameplay value justifies the implementation cost, or if the design could be simplified (e.g., "give adjacent squad +2 movement and a bonus attack" instead of true simultaneous activation).
