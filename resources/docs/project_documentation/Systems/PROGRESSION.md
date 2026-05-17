# Progression Package

**Last Updated:** 2026-05-16
**Package:** `tactical/powers/progression`
**Related:** `tactical/powers/perks`, `templates` (SpellRegistry), `tactical/commander`, `mind/combatlifecycle`, `setup/savesystem/chunks`, `gui/guiprogression`, `gui/guisquads`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Purpose and Context](#2-purpose-and-context)
3. [Architecture](#3-architecture)
4. [File-by-File Breakdown](#4-file-by-file-breakdown)
5. [Data Model](#5-data-model)
6. [Public API](#6-public-api)
7. [Integration Points](#7-integration-points)
8. [Key Flows](#8-key-flows)
9. [Invariants and Gotchas](#9-invariants-and-gotchas)
10. [Test Coverage](#10-test-coverage)
11. [Relationship to Design Documents](#11-relationship-to-design-documents)

---

## 1. Executive Summary

The `progression` package manages **per-commander** permanent progression state: two independent point currencies (Arcana Points and Skill Points) and two growing libraries (unlocked spells and unlocked perks). It answers one question at every call site: "has this commander unlocked this spell or perk, and do they have enough points to unlock more?"

The component (`ProgressionData`) is attached to each Commander entity by `commander.CreateCommander` (`tactical/commander/system.go:46`). The player can own multiple commanders (capped by `templates.GameConfig.Commander.MaxCommanders`); each maintains its own currencies and unlock library. Points flow in from combat rewards (overworld encounters and raid rooms), routed to the commanders whose squads participated. Spending is a deliberate choice in the Progression Mode UI. The resulting libraries gate what *that commander's* squads can equip and what *that commander's* spellcasting squads can learn — libraries are permanent and never shrink.

---

## 2. Purpose and Context

### Where It Sits

TinkerRogue organizes its power systems under `tactical/powers/`:

```
tactical/powers/
├── perks/          Squad-level passive abilities (hook-based, ECS dispatch)
├── spells/         Mana-gated active abilities (cast during combat)
├── effects/        Duration-based stat modifiers applied by spells/artifacts
├── artifacts/      Per-player inventory items with charge mechanics
└── progression/    Per-commander permanent library: currency + unlock state
```

The `progression` package is the meta-layer above perks and spells. It does not run any hooks or execute combat logic itself. It stores the record of what each commander has earned and paid for.

### The Problem It Solves

Squads and their units are intentionally expendable, but the player's *commanders* are not — they are scarce, paid for at a high gold cost, and bounded by `MaxCommanders` (configured in `templates.GameConfig.Commander`). Anchoring progression on commanders aligns the player's strategic identity with the units they actually invest in: one commander can be specialized into a perk-heavy bruiser line while another grows an arcana-focused spellcasting line, without either's unlock pool draining the other's. The 3-commander cap is what bounds the "protect-your-commander" pressure — losing one is painful but recoverable.

Squads and units remain expendable; perks attached to a squad are no longer relevant once that squad is destroyed, but the *library* they came from persists on the commander.

### Split Currency Design

Two separate currencies exist so that the perk track and the spell track can be balanced independently. A player who focuses on combat positioning unlocks perks with Skill Points without accidentally draining the Arcana pool needed to access new spells, and vice versa. The design note in memory explicitly rejects a single combined currency for this reason.

---

## 3. Architecture

### ECS Patterns Used

The package follows the project's standard ECS conventions:

1. **Pure data component.** `ProgressionData` is a plain struct with no methods and no ECS imports. All logic lives in system functions in `library.go`.

2. **EntityID-based access.** Every exported function takes `ownerID ecs.EntityID` (the entity carrying `ProgressionComponent`, which today is a Commander) rather than a pointer to the entity. The `GetProgression` helper wraps `common.GetComponentTypeByID` for this. The package is deliberately entity-agnostic — see `library.go:46`.

3. **Self-registration.** `init.go` registers the component and tag allocation with `common.RegisterSubsystem`, following the same pattern as `perks/init.go`, `spells/init.go`, and others. No manual wiring in `main.go` is required.

4. **Typed string IDs to avoid import cycles.** Perk IDs and spell IDs are stored as plain `string` values in `ProgressionData` rather than as typed `perks.PerkID` or `templates.SpellID`. This prevents a dependency cycle where `progression` would need to import `perks` and `templates` just to store data, while `perks` and `templates` might also import `progression`. Conversion to typed IDs happens at consumption sites (`IsPerkUnlocked`, `IsSpellUnlocked`).

### Internal Structure: The `library` Type

The central architectural decision in `library.go` is the unexported `library` struct:

```go
type library struct {
    unlocked     func(*ProgressionData) *[]string
    currency     func(*ProgressionData) *int
    currencyName string
}
```

This struct captures two accessor closures that point into `ProgressionData` — one for the relevant `[]string` slice and one for the relevant `int` currency field. Two package-level values, `perkLib` and `spellLib`, are constructed once and used by all perk-related and spell-related functions respectively.

The result is that `isUnlocked`, `unlock`, and `addPoints` are written once on the `library` receiver rather than being duplicated for perks and spells. The public functions (`IsPerkUnlocked`, `UnlockPerk`, `AddSkillPoints`, etc.) are thin typed wrappers that delegate to these shared receivers. If a third progression axis were added in the future, only a new `library` value would need to be declared.

---

## 4. File-by-File Breakdown

### `components.go` — Data Definitions

Defines the single component data struct, the ECS component variable, and the ECS tag variable.

- `ProgressionData` — the component's data type (see Section 5 for fields)
- `ProgressionComponent *ecs.Component` — component handle, allocated in `init.go`
- `ProgressionTag ecs.Tag` — tag for querying all entities that have this component

Callers looking up progression state by entity ID use `ProgressionComponent` directly via `common.GetComponentTypeByID`. The tag exists for bulk iteration (e.g., in the save system) but is not used by most callers.

### Starter sets — NOT in this package

`ProgressionData` has no constructor. New commanders are given a zero-valued component by `commander.CreateCommander` (`tactical/commander/system.go:46`); the optional starter library is layered on by `commander.SeedStarters` (`tactical/commander/init.go:42`), which reads from `templates.GameConfig.Commander.StartingPerks` and `.StartingSpells`. The JSON source is `resources/gamedata/gameconfig.json` under `commander.starting{Perks,Spells}`.

Skipping `SeedStarters` is intentional and supported: commanders created for tests or for an "empty library" play style simply do not call it.

There is no `defaults.go` in this package. There never was a `NewProgressionData()` constructor in the current (Commander-scoped) implementation — the prior Player-scoped version had one, but the migration removed it in favor of the explicit zero-value + SeedStarters split.

### `init.go` — ECS Registration

Registers the subsystem with `common.RegisterSubsystem`. When `common.InitializeSubsystems(manager)` is called during startup or test setup, this callback allocates `ProgressionComponent` and builds `ProgressionTag`, then stores the tag in `em.WorldTags["progression"]` for external reference.

This file contains no game logic and is only four lines of substance.

### `library.go` — All Logic

Contains the `library` struct definition, the two library instances (`perkLib`, `spellLib`), the three shared receiver methods (`isUnlocked`, `unlock`, `addPoints`), and the seven exported public functions (`GetProgression`, `IsPerkUnlocked`, `IsSpellUnlocked`, `UnlockPerk`, `UnlockSpell`, `AddArcanaPoints`, `AddSkillPoints`). Also defines the four sentinel error values.

---

## 5. Data Model

### `ProgressionData` (`components.go:8`)

```go
type ProgressionData struct {
    ArcanaPoints     int
    SkillPoints      int
    UnlockedSpellIDs []string
    UnlockedPerkIDs  []string
}
```

| Field | Type | Purpose |
|---|---|---|
| `ArcanaPoints` | `int` | Spendable currency for spell unlocks. Earned from overworld encounters and raid rooms. |
| `SkillPoints` | `int` | Spendable currency for perk unlocks. Earned from the same combat sources. |
| `UnlockedSpellIDs` | `[]string` | Permanently unlocked spells. Grows monotonically — IDs are never removed. Used to filter squad spellbooks. |
| `UnlockedPerkIDs` | `[]string` | Permanently unlocked perks. Grows monotonically. Used to gate the "available" list in the squad editor perk panel. |

**Attachment point.** One `ProgressionData` exists per commander. It is attached as a zero value by `commander.CreateCommander` (`tactical/commander/system.go:46`); starter perks/spells are then added by `commander.SeedStarters` (`tactical/commander/init.go:42`). On save load it is re-attached by `ProgressionChunk.RemapIDs` after `CommanderChunk` has recreated the commander entities.

**String IDs, not typed IDs.** The slices hold plain strings rather than `perks.PerkID` or `templates.SpellID`. This is documented in the component's godoc comment at `components.go:7`:

> "Perk and spell IDs are stored as plain strings to avoid a common <-> perks dependency cycle; conversion to typed IDs happens at consumption sites."

The actual conversion happens at `library.go:100` (`IsPerkUnlocked` casts to `perks.PerkID`) and `library.go:104` (`IsSpellUnlocked` casts to `templates.SpellID`).

### ECS Component and Tag

```go
var (
    ProgressionComponent *ecs.Component
    ProgressionTag       ecs.Tag
)
```

Both are nil until `common.InitializeSubsystems` runs. Any code that calls progression functions before subsystem initialization will panic. In practice this is safe: the game always calls `InitializeSubsystems` before any entity is created.

---

## 6. Public API

### Constructors — none

There is no constructor in this package. `commander.CreateCommander` attaches a zero-valued `ProgressionData`; `commander.SeedStarters(commanderID, manager)` populates the starter unlock set from `templates.GameConfig.Commander`. See **§4 Starter sets — NOT in this package** above.

---

### Queries

#### `GetProgression(ownerID ecs.EntityID, manager *common.EntityManager) *ProgressionData` (`library.go:46`)

Returns the `*ProgressionData` for the entity carrying `ProgressionComponent` (typically a Commander), or nil if the component is absent. This is the single entry point for reading progression state — all other public query functions call through this internally.

Callers outside the package (e.g., `guiprogression/progression_refresh.go:15`, `guiprogression/progression_controller.go:69`) use this to read current point totals for display.

#### `IsPerkUnlocked(ownerID ecs.EntityID, perkID perks.PerkID, manager *common.EntityManager) bool` (`library.go:99`)

Reports whether `perkID` is in the owner's `UnlockedPerkIDs` list. Returns false if the entity has no `ProgressionData`. Performs a linear scan of the slice (expected to remain small — the current registry has 21 perks).

**Callers:**
- `gui/guisquads/squadeditor_perks.go:90` — filters the "available" perk list to exclude perks locked in the squad's owning commander's library
- `gui/guiprogression/progression_controller.go` — divides perks into unlocked/locked panels for the active commander

#### `IsSpellUnlocked(ownerID ecs.EntityID, spellID templates.SpellID, manager *common.EntityManager) bool` (`library.go:106`)

Same semantics as `IsPerkUnlocked` but for the spell library. Returns false on missing data.

**Callers:**
- `tactical/powers/spells/system.go:67` (`filterSpellsByCommanderLibrary`) — filters a leader's spell list against the squad's owning commander before attaching to `SpellBookData`
- `gui/guiprogression/progression_controller.go` — divides spells into unlocked/locked panels for the active commander

---

### Systems (Mutations)

#### `UnlockPerk(ownerID ecs.EntityID, perkID perks.PerkID, manager *common.EntityManager) error` (`library.go:109`)

Spends `SkillPoints` equal to `perks.GetPerkDefinition(perkID).UnlockCost` and appends `perkID` (as string) to `UnlockedPerkIDs`. Returns:

- `ErrUnknownPerk` (wrapped) if the perk ID is not in `perks.PerkRegistry`
- `ErrNoProgressionData` if the owner entity has no `ProgressionData`
- `ErrNotEnoughPoints` (wrapped with amount detail) if `SkillPoints < unlockCost`
- `nil` if already unlocked (idempotent — no double-spend)

The idempotency guarantee means UI code can call this on repeated button presses without needing to guard against double-unlock.

**Callers:**
- `gui/guiprogression/progression_controller.go:79` (`onUnlockClicked`, perk path)

#### `UnlockSpell(ownerID ecs.EntityID, spellID templates.SpellID, manager *common.EntityManager) error` (`library.go:118`)

Same semantics as `UnlockPerk` but for spells, spending `ArcanaPoints` equal to `templates.GetSpellDefinition(spellID).UnlockCost`. Returns the same error set (with `ErrUnknownSpell` in place of `ErrUnknownPerk`).

**Callers:**
- `gui/guiprogression/progression_controller.go:79` (`onUnlockClicked`, spell path)

#### `AddArcanaPoints(ownerID ecs.EntityID, amount int, manager *common.EntityManager)` (`library.go:127`)

Adds `amount` to `ProgressionData.ArcanaPoints`. No-op if `amount <= 0` or if the owner has no `ProgressionData`. The non-positive guard prevents callers from needing to check before calling.

**Callers:**
- `mind/combatlifecycle/reward.go` — called indirectly through `grantProgressionPoints` with this function as a parameter, once per commander in `GrantTarget.CommanderIDs`

#### `AddSkillPoints(ownerID ecs.EntityID, amount int, manager *common.EntityManager)` (`library.go:134`)

Same semantics as `AddArcanaPoints` but for `SkillPoints`.

**Callers:**
- `mind/combatlifecycle/reward.go` — same indirect path

---

### Error Sentinels (`library.go:13–18`)

```go
var (
    ErrNotEnoughPoints   = errors.New("not enough points")
    ErrUnknownPerk       = errors.New("unknown perk")
    ErrUnknownSpell      = errors.New("unknown spell")
    ErrNoProgressionData = errors.New("no progression data")
)
```

All four are package-level variables suitable for `errors.Is` checks. `ErrNotEnoughPoints` is always wrapped with context (amount needed, amount available, currency name) via `fmt.Errorf("%w: ...")`. The others may also be wrapped when an ID is available.

---

## 7. Integration Points

### 7.1 Commander Entity Initialization (`tactical/commander/system.go:46`)

`commander.CreateCommander` constructs a Commander entity and attaches an empty `ProgressionData` (zero points, empty unlock slices) alongside its other components:

```go
.AddComponent(progression.ProgressionComponent, &progression.ProgressionData{})
```

Callers that want a starter library then call `commander.SeedStarters(commanderID, manager)` (`commander/init.go:42`), which appends `templates.GameConfig.Commander.StartingPerks` and `.StartingSpells` to the unlock slices. Callers that want an empty library (some tests, certain debug flows) simply skip the seed step.

There is no single "Player entity has progression" attachment point — each commander created during play (initial + any purchased through the roster) carries its own `ProgressionData`.

### 7.2 Combat Reward Granting (`mind/combatlifecycle/reward.go`)

The `Grant` function in `combatlifecycle` is the universal reward distributor. `Reward.ArcanaPts` and `Reward.SkillPts` are the two progression-relevant fields:

```go
type Reward struct {
    Gold       int
    Experience int
    Mana       int
    ArcanaPts  int
    SkillPts   int
}
```

When either field is non-zero, `Grant` iterates `target.CommanderIDs` (deduplicated) and calls the private `grantProgressionPoints` helper for each commander, which delegates to `progression.AddArcanaPoints` or `progression.AddSkillPoints` by passing them as function parameters. This keeps the reward pipeline decoupled from the specific currency names and supports multi-commander encounters without re-deriving squad ownership at award time.

```go
type GrantTarget struct {
    PlayerEntityID ecs.EntityID   // For gold (ResourceStockpile owner)
    SquadIDs       []ecs.EntityID // For XP and mana distribution
    CommanderIDs   []ecs.EntityID // Receives Arcana/Skill progression points
}
```

Each caller is responsible for populating `CommanderIDs` at construction time. The existing helper `commander.FindCommanderForSquad(squadID, manager)` (`commander/queries.go:55`) maps a squad to its owning commander; both `mind/encounter/resolvers.go` and `campaign/raid/rewards.go` define a local `commandersForSquads` helper that walks the squad list through this helper.

**Where `Reward` values come from:**

- **Overworld encounters** (`mind/encounter/rewards.go`). `CalculateIntensityReward(intensity int)` computes both `ArcanaPts` and `SkillPts` as `int(float64(1 + intensity) * (1.0 + float64(intensity) * 0.1))`. An intensity-5 encounter yields `int(6 * 1.5) = 9` points of each currency. Both currencies always scale identically in this formula.

- **Raid rooms** (`campaign/raid/rewards.go`). `calculateRoomReward` reads from `RaidConfig.Rewards.BaseArcanaPerRoom` and `RaidConfig.Rewards.BaseSkillPerRoom`, then scales by floor: `1.0 + (floor - 1) * FloorScalePercent / 100`. Falls back to 1 point per currency if the config fields are zero. Command post rooms additionally restore mana.

The full chain is: combat resolves → `CombatResolver.Resolve()` returns a `ResolutionPlan` → `ExecuteResolution` calls `Grant` → `Grant` calls `AddArcanaPoints`/`AddSkillPoints` once per commander in `target.CommanderIDs`.

### 7.3 Spell System Filtering (`tactical/powers/spells/system.go:67`)

`InitSquadSpellsFromLeader` is called after a squad is created. It looks up the leader unit's spell list from `templates.UnitSpellRegistry`, then calls `filterSpellsByCommanderLibrary` before attaching those spells to the squad's `SpellBookData`.

`filterSpellsByCommanderLibrary` (`system.go:67`) resolves the squad's owning commander via `commander.FindCommanderForSquad(squadID, manager)`, calls `progression.GetProgression`, and then calls `progression.IsSpellUnlocked` for each candidate spell. Spells not in the library are silently filtered out. If no owning commander is found (enemy squads, test fixtures without a commander) or the commander has no progression data, the full spell list passes through unchanged.

This is the key enforcement point for the spell library: a squad can only cast spells whose IDs are in the *owning commander's* `UnlockedSpellIDs` at the moment the squad is created. Unlocking a spell after squad creation does not retroactively update that squad's spellbook.

### 7.4 Perk Equip Gate (`gui/guisquads/squadeditor_perks.go:83-92`)

The squad editor's perk panel (`refreshPerkPanel`) resolves the squad's owning commander via `commander.FindCommanderForSquad`, then iterates all perk IDs from `perks.GetAllPerkIDs()` and calls `progression.IsPerkUnlocked(ownerCommanderID, id, manager)` for each. Perks not in the commander's library are excluded from the "available" list entirely — the player cannot see or equip them. This is a UI-side gate, not an ECS-side gate; `perks.EquipPerk` itself has no knowledge of the progression library.

If the squad has no resolvable commander (an orphaned or detached squad), no library gating is applied — all perks pass through. This is an intentional permissive fallback for edge cases; in normal play every squad has a commander.

### 7.5 Progression UI Mode (`gui/guiprogression/`)

`ProgressionMode` is a dedicated UI mode accessible from the squad editor. It has three panels:

- **Header** (`ProgressionPanelHeader`) — shows the active commander's current Arcana and Skill point totals.
- **Perks panel** (`ProgressionPanelPerks`) — two lists (unlocked, locked) for perks, with an unlock button.
- **Spells panel** (`ProgressionPanelSpells`) — same layout for spells.

The controller reads `ProgressionData` for the active commander (`pm.activeCommanderID()`) via `GetProgression` for display and calls `UnlockPerk`/`UnlockSpell` on user action. The "Unlock" button is disabled when the commander cannot afford the selected item (checked at `progression_controller.go:70-71` by comparing `currentPoints(data) >= entry.item.unlockCost`).

Both panels are driven by the same `libraryPanelController` type with a `librarySource` configuration struct. The perk and spell sources are package-level variables (`perkLibrarySource`, `spellLibrarySource`) defined at `progression_controller.go:132`. This mirrors the `library` struct in the `progression` package — both use the same two-instance pattern to avoid duplicating code across the two currency axes.

### 7.6 Save System (`setup/savesystem/chunks/progression_chunk.go`)

`ProgressionChunk` serializes and deserializes `ProgressionData` for every commander to JSON. It follows the project's standard chunk pattern (save, load, remap):

- **Save**: queries `commander.CommanderTag`, reads the `ProgressionData` component from each commander, and marshals one `savedCommanderProgression` entry per commander into `savedProgressionChunk.Commanders`.
- **Load**: unmarshals into `savedProgressionChunk` and stores it in `idMap.LoadContext` under a private key. No ECS mutations happen here because the commander entities may not yet exist at load time — `CommanderChunk` recreates them in a separate pass.
- **RemapIDs**: after entity IDs are remapped, looks up the stored data, finds each commander entity by its new ID, and calls `entity.AddComponent(progression.ProgressionComponent, data)` to restore that commander's progression state.

**The chunk version is 2.** Format was bumped when progression moved from Player-scoped (v1, single entry) to Commander-scoped (v2, one entry per commander) on 2026-04-21. Old v1 saves are no longer readable.

---

## 8. Key Flows

### Flow 1: Awarding Points After Combat

1. Combat ends; the appropriate `CombatResolver` calls `resolver.Resolve(manager)` and returns a `ResolutionPlan` with non-zero `ArcanaPts` and `SkillPts` in its `Reward`.
2. The resolver builds `GrantTarget`, populating `CommanderIDs` from the participating squads via its local `commandersForSquads` helper. The helper walks each squad through `commander.FindCommanderForSquad`, skipping zero results and deduplicating so each commander appears at most once — important because one commander typically leads multiple squads in an encounter and would otherwise receive N× rewards.
3. `ExecuteResolution` calls `Grant(manager, plan.Rewards, plan.Target)`.
4. `Grant` trusts `target.CommanderIDs` as already deduplicated. For each commander, when `r.ArcanaPts > 0` it calls `grantProgressionPoints(manager, commanderID, r.ArcanaPts, "Arcana", progression.AddArcanaPoints)`; same shape for `SkillPts` via `AddSkillPoints`.
5. `grantProgressionPoints` calls `progression.GetProgression(commanderID, manager)` as a nil-guard, then calls the supplied `add` function.
6. `AddArcanaPoints` delegates to `spellLib.addPoints`, which calls `lib.currency(data)` to get `&data.ArcanaPoints` and increments it by `amount`.
7. `Grant` returns a reward description string such as `"150 gold, 75 XP, 4 Arcana, 4 Skill"` for display (or with the per-commander grant repeated when multiple commanders share the encounter).

### Flow 2: Commander Unlocks a Perk

1. Player opens `ProgressionMode` from the squad editor; the mode's active commander has already been selected.
2. `progressionController.refresh()` calls `GetProgression(activeCommanderID, …)` and sets the skill label to `"Skill: N"`.
3. `libraryPanelController.refresh()` (perk panel) iterates `allPerkItems()`, calls `perkLibrarySource.isUnlocked(activeCommanderID, …)` for each, and puts items into the locked or unlocked list accordingly. Locked items display `"Perk Name (cost)"`.
4. Player selects a locked perk. `onLockedSelected` is called; the unlock button becomes enabled if `data.SkillPoints >= entry.item.unlockCost`.
5. Player clicks "Unlock Perk". `onUnlockClicked` calls `perkLibrarySource.unlock(activeCommanderID, …)`, which calls `progression.UnlockPerk(commanderID, perks.PerkID(itemID), manager)`.
6. `UnlockPerk` looks up `perks.GetPerkDefinition(perkID)` for the cost, then calls `perkLib.unlock`.
7. `perkLib.unlock` checks the ID is not already in `UnlockedPerkIDs`, checks `SkillPoints >= unlockCost`, deducts the cost, and appends the string ID to the slice.
8. `onUnlockClicked` calls `pc.mode.controller.refresh()`, which re-reads `ProgressionData` and rebuilds both lists. The newly unlocked perk moves from the locked list to the unlocked list.

### Flow 3: Commander Unlocks a Spell

Identical to Flow 2 with `spellLibrarySource` and `ArcanaPoints`. The cost comes from `templates.GetSpellDefinition(spellID).UnlockCost` rather than `perks.GetPerkDefinition`.

### Flow 4: Equipping an Unlocked Perk on a Squad

This flow does not involve the `progression` package directly, but progression gates it:

1. Player opens the squad editor perk panel for a given squad.
2. `refreshPerkPanel` (`squadeditor_perks.go`) resolves the squad's owning commander via `commander.FindCommanderForSquad(squadID, manager)`, then calls `progression.IsPerkUnlocked(ownerCommanderID, id, manager)` for each perk ID in `perks.GetAllPerkIDs()`. Only perks unlocked in that commander's library appear in the "available" list.
3. Player selects an available perk and clicks "Equip".
4. `onEquipClicked` calls `perks.EquipPerk(squadID, def.ID, perks.MaxPerkSlots, manager)`. This function has no knowledge of the progression library — the gatekeeping was done at step 2. The equip call checks slot capacity and mutual exclusivity, then appends the perk to `PerkSlotData.PerkIDs`.

### Flow 5: Initializing a Squad's Spellbook

1. A new squad is created (e.g., when the player purchases units or a raid deploys squads).
2. `spells.InitSquadSpellsFromLeader(squadID, manager)` is called.
3. The leader unit type is found; `templates.GetSpellsForUnitType(leaderUnitType)` returns the candidate spell list.
4. `filterSpellsByCommanderLibrary` resolves the squad's owning commander via `commander.FindCommanderForSquad`. If found, it calls `progression.GetProgression` and then `progression.IsSpellUnlocked` for each candidate. Unlocked spells are kept; locked spells are dropped silently. If no owning commander is resolved, the full list passes through.
5. The filtered list is passed to `AddSpellCapabilityToSquad`, which attaches `ManaData` and `SpellBookData` to the squad entity.

**Implication:** a spell unlocked after this squad was created will not appear in that squad's spellbook until the squad is re-created or a refresh mechanism is added. No such mechanism currently exists.

### Flow 6: Saving and Loading Progression State

**Save:**
1. `ProgressionChunk.Save` queries `commander.CommanderTag`, reads `ProgressionData` from each commander, and marshals one `savedCommanderProgression` entry per commander into `savedProgressionChunk.Commanders`.

**Load:**
1. `ProgressionChunk.Load` parses the JSON into `savedProgressionChunk` and stores it in `idMap.LoadContext`.
2. After all chunks load, `ProgressionChunk.RemapIDs` runs. For each saved commander entry it maps the saved entity ID to the new entity ID (commander entities were re-created by `CommanderChunk`), then calls `entity.AddComponent(progression.ProgressionComponent, data)` with a freshly allocated `ProgressionData` reconstructed from the saved values.

---

## 9. Invariants and Gotchas

### The 3-Perk-Slot Cap

The hard limit of 3 equipped perk slots per squad is defined as `perks.MaxPerkSlots = 3` in `tactical/powers/perks/system.go:11`. The `progression` package does not define this constant and has no knowledge of it. The cap is enforced by `perks.EquipPerk`, not by the progression library.

The design explicitly decided against a 4th slot at high veterancy ranks. Every perk is balanced around at most 2 other perks being active simultaneously. Adding a rank-gated slot would require rebalancing all perk interactions.

The progression library is unbounded — the player can unlock all 21 perks and all available spells if they earn enough points. The tension in the system is unlock order, not a permanent ceiling.

### Idempotent Unlock

`UnlockPerk` and `UnlockSpell` (and the underlying `library.unlock`) return nil without deducting points if the item is already in the unlocked list. This is checked by a linear scan before the affordability check. UI code does not need to guard against double-unlocks.

### Points Cannot Be Negative

`addPoints` silently returns if `amount <= 0` (`library.go:87`). There is no decrement operation for points other than the deduction inside `unlock`. Points are never negative after construction.

### No Respec / No Point Refund

Once points are spent and a perk or spell is unlocked, there is no refund mechanism. The design document states explicitly: "No respec needed. With unlimited ranks, everything eventually unlocks. The tension is unlock ORDER." No respec function exists in this package or its callers.

### Spell Library Does Not Retroactively Update Squads

`filterSpellsByCommanderLibrary` in `spells/system.go:67` runs only at squad creation time (`InitSquadSpellsFromLeader`). Unlocking a spell after that squad was created does not update the squad's `SpellBookData`. Squads formed before a spell unlock will never have that spell unless they are destroyed and re-created.

This is a known design characteristic, not a bug. The design note says "Spell library supplements leader spells" — the filtering at creation time is the integration point.

### Currency Is Commander-Scoped

`ProgressionData` attaches to each Commander entity (`commander.CreateCommander` at `commander/system.go:46`); the Player entity does **not** carry it. `AddArcanaPoints`, `AddSkillPoints`, `IsPerkUnlocked`, `UnlockPerk`, and their spell counterparts all take a commander entity ID. Passing the player's entity ID would silently do nothing (the component would not be found).

This means:

- Two commanders earn separate point pools; one cannot spend the other's Arcana.
- Their unlock libraries are independent — a perk unlocked by commander A is not visible to a squad owned by commander B.
- Combat rewards are routed by `combatlifecycle.Grant` to every commander listed in `GrantTarget.CommanderIDs` (deduplicated). Callers (`mind/encounter/resolvers.go`, `campaign/raid/rewards.go`) populate this list at construction time by mapping participating squads through `commander.FindCommanderForSquad`. The earlier "first squad's commander wins" derivation was removed because it silently dropped rewards in multi-commander encounters.

### ECS Dependency-Cycle Avoidance

Storing IDs as `string` rather than `perks.PerkID` or `templates.SpellID` in `ProgressionData` is load-bearing. If the fields were typed, `progression` would import `perks` and `templates`; but `spells/system.go` imports `progression` and is in the same `tactical/powers/` subtree as `perks`. Go's import cycle detection would reject a circular dependency. The string storage is the package's primary structural constraint.

### No Mana in Progression

Mana is a per-squad resource managed by `spells.ManaData`, not by the progression system. `Reward.Mana` exists in `combatlifecycle` but is distributed directly to squad components, not to `ProgressionData`. The progression package has no mana field or mana-related API.

### `ProgressionTag` Is Stored but Unused for Iteration

The tag (`em.WorldTags["progression"]`) is registered, but the save chunk now iterates `commander.CommanderTag` and then reads `ProgressionComponent` per commander — there is no consumer that queries `ProgressionTag` directly. Code that needs to iterate all entities carrying progression state should iterate `CommanderTag` and check / read the component.

---

## 10. Test Coverage

### `progression/library_test.go`

Tests live in `package progression` (white-box; access to exported symbols only) and use `testfx.NewTestEntityManager()` from the `testing` package.

The test helper `newTestManagerWithPerkData` (`library_test.go:12`) manually seeds `perks.PerkRegistry` and `templates.SpellRegistry` with a small fixture set and attaches a fresh `ProgressionData` to a test entity. This avoids loading JSON data files in tests while still exercising the registry lookups in `UnlockPerk` and `UnlockSpell`.

| Test | What It Verifies |
|---|---|
| `TestUnlockPerkDeductsAndIsIdempotent` | Cost deducted correctly; second call is no-op with no extra deduction |
| `TestUnlockPerkInsufficientPoints` | `ErrNotEnoughPoints` returned; points unchanged; perk remains locked |
| `TestUnlockSpellDeductsAndIsIdempotent` | Mirror of perk test for arcana / spell path |
| `TestAddPoints` | Both currencies accumulate correctly; zero and negative amounts are ignored |

**Gaps:**

- `ErrNoProgressionData` path is not exercised against the entity-with-no-component case.
- `ErrUnknownPerk` / `ErrUnknownSpell` paths (ID not in registry) are not tested.
- `AddArcanaPoints` / `AddSkillPoints` called on an entity with no `ProgressionData` (the nil-guard branch in `addPoints`) is not tested.
- No integration test for the reward path (`combatlifecycle.Grant` → `commander.FindCommanderForSquad` → `progression.AddArcanaPoints`).
- No integration test for the save round-trip (`ProgressionChunk` save / load / remap preserving per-commander data).
- No integration test for `commander.SeedStarters` populating from `GameConfig`.

### `perks/perks_test.go` — Related but Separate

The perks package test file exercises `EquipPerk`/`UnequipPerk`, perk round state lifecycle, behavior implementations, and multi-perk interaction tests. It does not test the `progression` package. The progression-perk integration (the UI gate in `squadeditor_perks.go`) has no dedicated test.

### Save System Tests

The save system chunk for progression (`setup/savesystem/chunks/progression_chunk.go`) has no dedicated test file. Save-system integration tests, if they exist, would live in `setup/savesystem/`.

---

## 11. Relationship to Design Documents

The design notes in `project_progression_system.md` (memory file) describe decisions made before and during implementation. Most are faithfully implemented:

| Design Decision | Implementation Status |
|---|---|
| Max 3 perk slots, no 4th slot from high ranks | Implemented. `perks.MaxPerkSlots = 3` is a constant, not computed from any rank. |
| Split currency immediately (SkillPoints for perks, ArcanaPoints for spells) | Implemented as two separate fields on `ProgressionData`. |
| Progression is per-commander (revised) | Implemented. `ProgressionData` attaches to each commander in `commander.CreateCommander` (`commander/system.go:46`); starters seeded via `commander.SeedStarters`. The original design referenced a player-scoped library; that was migrated to commander-scoped on 2026-04-21 (save format bumped to v2). |
| Perk library follows artifact inventory pattern | Implemented. Each commander owns `UnlockedPerkIDs`; squads under that commander equip from it. |
| Spell library filters spellbook at squad creation | Implemented in `spells/system.go:filterSpellsByCommanderLibrary`. |
| No respec | Implemented by omission — no refund function exists. |
| Both currencies earned from combat, scaled by intensity | Implemented. Encounter rewards (`mind/encounter/rewards.go`) use intensity scaling; raid rewards (`campaign/raid/rewards.go`) use floor scaling. |
| Raids grant points too | Implemented in `campaign/raid/rewards.go`. |

**Discrepancy — "Commander Veterancy Ranks":** The design document's title and framing reference "Veterancy Ranks" as an intermediate layer that grants points. The implementation has no rank system. There is no `VeterancyRank` field, no rank thresholds, and no function that computes or increments a rank. Points are awarded directly by combat rewards without any rank intermediary. The design may have intended ranks as a future layer on top of the point system, or the rank concept was simplified away during implementation.
