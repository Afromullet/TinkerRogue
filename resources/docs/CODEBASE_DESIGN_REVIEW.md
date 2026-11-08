# TinkerRogue Codebase Design Review

**Date:** 2026-11-06 · **Commit reviewed:** `d99ffdd` (main) · **No code was changed by this review.**

## How to read this document

- **Scope.** All shipped game code: 76 packages, about 55,000 non-test lines. `tools/`, `testing/` and `*_test.go` files were not reviewed. Earlier tech-debt documents were deliberately not read.
- **Method.** Every non-test file in scope was read in full, package by package. `go build ./...` and `go vet ./...` were run and are clean. Nothing else was executed, so **every bug in this document comes from tracing code, not from reproducing it in a running game.** The most consequential claims were re-read against the source a second time; section 6 marks which.
- **Lenses.** Internal design (duplication, sources of truth, abstraction level, coupling, complexity), game architecture (simulation vs presentation, data-driven content, determinism, save safety, ECS discipline), and mechanics that overlap or contradict each other. Probable bugs, dead code and design-related performance issues are reported separately inside each package section.
- **Detail level.** Findings and direction. Each finding gives evidence (`file:line`), the problem, what to do, and rough effort (S/M/L) and risk. Detailed designs are left for the session that does the work.
- **Tests.** Not reviewed and not recommended. "No tests" appears only as a risk note on a refactor.
- **Structure.** Sections 1 to 8 are the synthesis: summary, verdicts, themes, target layout, decisions, bug index, roadmap, guardrails. Sections 9 to 20 are the detailed findings, one section per layer ("slice") and one subsection per package. Start with sections 1, 5 and 7; use the rest as reference when you work on a package.

---

## 1. Summary

The codebase is in better shape at the bottom than at the top. The ECS foundations, the squad query layer, the damage formula, the save envelope, the `Loader[T]` data pipeline and the input `ActionMap` are sound and should be left alone. The problems concentrate in four places, and they explain most of the individual findings:

1. **The game's control flow lives in GUI mode hooks.** The combat turn loop, combat teardown, raid progression, commander recruitment and map regeneration are driven from `gui/` (`Enter`, `Exit`, click handlers). The simulation cannot run a battle or a raid without the GUI, the combat simulator re-implements its own loop, and several lifecycle bugs follow directly from it (a battle that ends mid-animation skips combat exit; a raid battle returns to the wrong screen after the first End Turn; the last raid floor never ends the raid).

2. **The same fact is stored in two or more places.** Movement speed and attack range, unit availability in the roster, kills in an attack, garrison membership, artifact equip state, the squad spellbook, walkability, screen and map dimensions, faction identity, the overworld turn counter. Each has a traced drift path. This is the single largest source of the bugs in section 6, and it is the thing the project's own ECS rules ("query-based, don't cache relationships") are meant to prevent.

3. **Rules are enforced by whoever calls first, usually the GUI.** Spell casting (action cost, valid targets, AoE area), artifact activation, perk unlock gating, zone of control, node placement and unit placement are checked in click handlers, while the system functions they call accept anything. The planned AI spell casting would have to copy those rules.

4. **A noticeable share of the code is unreachable or inert.** A whole overworld win/lose package nothing imports, a squad deployment mode nothing transitions to, a 985-line garrison map generator only the debug menu can run, an undo system with no key binding, three render caches that cache nothing, six GUI panels that are never created, and many write-only fields. The six largest items alone are about 2,600 lines (section 3.8). Dead code that looks live is what makes the rest hard to trust.

Two further themes matter for stability rather than structure: **save/load silently loses four pieces of state** (equipped perks, unassigned roster units, raid garrison data, the raid's player ID) and leaves the roster unusable after a load, and **game data is loaded through three different conventions with five failure policies**, so a broken file may stop the game, warn, or be silently replaced by code defaults.

On package structure: the tree is over-split in three areas (ten overworld sub-packages that share one data kernel, seven combat packages, five combat-GUI packages) and has a handful of dependencies pointing the wrong way (`world → visual`, `tactical → campaign`, `mind ↔ campaign`, `setup → testing`). Section 4 proposes a layout that removes about 16 packages without adding files.

**Verdict counts (76 packages):** 15 healthy, 49 need attention, 9 need significant rework, 3 should not exist as packages.

---

## 2. Package verdicts

Lines are non-test lines. "Main issue" is the one thing to know; details are in the linked section.

### Core, setup, data, world (sections 10 and 11)

| Package | Lines | Verdict | Main issue |
|---|---:|---|---|
| `core/common` | 893 | Needs attention | Speed/range stored twice; shared RNG never seeded; `WorldTags` string registry is write-mostly |
| `core/coords` | 354 | Needs attention | Two screen-config singletons built before JSON loads, so display settings in JSON are dead knobs |
| `core/config` | 133 | Needs attention (small) | Hand-copied mirror of `templates.GameConfig`, synced by a 17-argument positional setter |
| `game_main` | 422 | Needs attention | Three setup paths repeat each other; a failed load leaves a half-built world |
| `setup/gamesetup` | 1,189 | Needs attention | `initial_squads.go` is five copies of one function; data loading split over three places |
| `setup/savesystem` | 371 | Healthy | Unmapped IDs silently become 0, which hides the chunk bugs |
| `setup/savesystem/chunks` | 1,724 | Needs attention | Hand-mirrored DTOs and factories; four pieces of state lost on save/load |
| `templates` | 2,383 | Needs attention | Loads 16 of 24 data files; uneven validation; carries an entity factory |
| `world/worldmapcore` | 651 | Needs attention | Walkability has three stores; `Tile` carries presentation state |
| `world/worldgen` | 1,790 | Needs attention | Mapgen config defined three times; generators reseed the global RNG |
| `world/garrisongen` | 1,345 | Needs significant rework | Tile generator (985 lines) is debug-only and unrelated to the raid it is named for |
| `world/garrisongen/roomtypes` | 38 | Healthy | Exists only to break one import cycle |

### Tactical (sections 12, 13, 14)

| Package | Lines | Verdict | Main issue |
|---|---:|---|---|
| `tactical/squads/squadcore` | 1,418 | Needs attention | Three diverging unit/squad construction paths; placement validated four ways |
| `tactical/squads/roster` | 398 | Needs significant rework | Three derived structures that drift from the ECS; broken after load |
| `tactical/squads/squadcommands` | 1,142 | Needs attention | `MoveSquadCommand` is a combat command; commands re-implement squadcore rules |
| `tactical/squads/squadservices` | 329 | Needs attention | Stateless wrappers; dissolve into `roster` |
| `tactical/squads/unitdefs` | 251 | Needs attention | `UnitTemplate` mixes definition and placement and stores attributes twice |
| `tactical/squads/unitprogression` | 156 | Healthy | |
| `tactical/commander` | 437 | Needs attention | Overworld package filed under `tactical/`; duplicate turn counter |
| `tactical/combat/combattypes` | 278 | Healthy | `AttackEvent` duplicates `DamageRecord` |
| `tactical/combat/combatstate` | 601 | Needs attention | Action state is a separate entity reached through a threaded cache; one ECS view leaks per combat |
| `tactical/combat/combatmath` | 744 | Needs attention | Balance config fails silently; "math" package mutates HP |
| `tactical/combat/combatcore` | 881 | Needs attention | Kill bookkeeping disagrees with the damage record |
| `tactical/combat/battlelog` | 760 | Needs attention | Two unrelated jobs; recorder lifecycle driven by the GUI |
| `tactical/combat/combatdisposal` | 41 | Should not exist | One function whose extra steps are redundant |
| `tactical/combat/combatservices` | 817 | Needs attention | Turn loop is not here; nine dead methods; duplicated teardown |
| `tactical/powers/perks` | 1,995 | Needs significant rework | State lifecycle only: per-round state is wiped right after it is set |
| `tactical/powers/artifacts` | 1,280 | Needs attention | Equip state stored twice; charges keyed by behavior name, outside ECS |
| `tactical/powers/effects` | 316 | Needs attention | Mutates base stats and undoes by inverse arithmetic with clamping |
| `tactical/powers/spells` | 376 | Needs attention | Spellbook is a stale snapshot; spell damage bypasses the combat path |
| `tactical/powers/powercore` | 270 | Healthy | Partly ceremonial (one-publisher event bus) |
| `tactical/powers/progression` | 189 | Healthy | |

### Mind and campaign (sections 15 and 16)

| Package | Lines | Verdict | Main issue |
|---|---:|---|---|
| `mind/ai` | 715 | Needs attention | Four evaluator caches (one broken by `Reset`); animation queue inside the AI |
| `mind/behavior` | 1,226 | Needs attention | Threat data is only populated because a GUI object registers factions |
| `mind/evaluation` | 465 | Needs attention | Unit power implemented twice with different results |
| `mind/combatlifecycle` | 788 | Needs attention | Tactical infrastructure filed under `mind`; casualty count cannot work |
| `mind/encounter` | 1,050 | Needs significant rework | Three responsibilities; much of it dead; no rollback on failed start |
| `mind/spawning` | 423 | Needs attention | Silently drops units; one of three copies of "build a random squad" |
| `campaign/overworld/core` | 1,177 | Needs attention | Grab-bag kernel; four faction identities; write-only fields |
| `campaign/overworld/ids` | 53 | Healthy | |
| `campaign/overworld/node` | 272 | Healthy | |
| `campaign/overworld/threat` | 290 | Needs attention | Evolution behavior hardcoded per type behind a data flag |
| `campaign/overworld/influence` | 297 | Healthy | One latent nil dereference |
| `campaign/overworld/faction` | 512 | Needs attention | Three threat-spawn paths with three rule sets; several inert mechanics |
| `campaign/overworld/garrison` | 323 | Needs attention | Membership stored on both node and squad; lockout bug |
| `campaign/overworld/tick` | 113 | Healthy | Duplicate counter; stale comments |
| `campaign/overworld/victory` | 191 | Should not exist as is | Imported by nothing; wire it or delete it |
| `campaign/overworld/overworldlog` | 393 | Needs attention | Summaries depend on event data nobody supplies |
| `campaign/raid` | 1,730 | Needs significant rework | End-of-run state machine only; the rest is sound |

### GUI, visual, input (sections 17 to 20)

| Package | Lines | Verdict | Main issue |
|---|---:|---|---|
| `gui/framework` | 2,388 | Needs attention | No input gating for dialogs or pointer-over-UI; panel registry half dead; shared `SquadInfoCache` |
| `gui/builders` | 2,104 | Needs attention | Four ways to build a panel; one silently returns nil for six live panels |
| `gui/widgetresources` | 507 | Needs attention | Background cache has no effect; load errors swallowed |
| `gui/widgets` | 408 | Needs significant rework | About 300 lines of cache wrappers that are inert or wrong |
| `gui/specs` | 290 | Healthy, needs pruning | 26 unused constants |
| `gui/guisquads` | 3,928 | Needs significant rework | Unreachable mode and undo; list-refresh helper scrambles layouts; heavy copy-paste |
| `gui/guiprogression` | 572 | Healthy | The cleanest mode; use it as the model |
| `gui/guiunitview` | 195 | Needs attention | ESC is dead; ungated XP button with a clock-seeded RNG |
| `gui/guiinspect` | 182 | Healthy | |
| `gui/guiartifacts` | 329 | Needs attention | GUI builds the artifact execution context and validates targets |
| `gui/guispells` | 533 | Needs attention | Spell rules live only here; buffs can only target enemies |
| `gui/guistartmenu` | 105 | Healthy | |
| `gui/guicombat` | 1,114 | Needs significant rework | Turn loop and teardown driven from mode hooks |
| `gui/guicombat/combatbase` | 262 | Needs attention | Cycle-breaking grab-bag; merge into `guicombat` |
| `gui/guicombat/combatinput` | 557 | Needs attention | 165-line priority ladder; merge into `guicombat` |
| `gui/guicombat/combatanimation` | 564 | Needs attention | Recomputes who hit whom after the attack already resolved |
| `gui/guicombat/combatvisualization` | 583 | Needs attention | Parallel copy of AI evaluator caches; overlays stored in map tiles |
| `gui/guioverworld` | 1,739 | Needs attention | `RecruitCommander` is a full game rule in the GUI |
| `gui/guiraid` | 1,514 | Needs attention | Raid progression rules duplicated from, and contradicting, `RaidRunner` |
| `gui/guinodeplacement` | 540 | Healthy | Fold into `guioverworld` |
| `gui/guiexploration` | 333 | Needs attention | World regeneration lives in a panel-registry file |
| `visual/graphics` | 426 | Needs significant rework | Not a graphics package: holds the AoE targeting rule and a tint type |
| `visual/rendering` | 395 | Needs attention | Vestigial double viewport wrapper with an expensive overlay path |
| `visual/maprender` | 274 | Needs attention | Cache invalidation relies on four signals; merge into `rendering` |
| `visual/combatrender` | 401 | Needs attention | Pull-style providers cost per-frame work; multi-cell layout bug |
| `visual/vfx` | 512 | Needs attention | Uses the simulation RNG every drawn frame; three clocks |
| `input` | 137 | Should not exist | One mode's input handling outside the mode system |

---

## 3. Cross-cutting themes

Each theme lists where it shows up and the direction to take. The per-package sections have the evidence.

### 3.1 Game flow lives in GUI mode hooks

| Sequence | Where it lives now | Where it should live |
|---|---|---|
| End turn → victory check → AI turn → next turn | `gui/guicombat/combat_turn_flow.go` | One headless operation on `combatservices.CombatService` |
| Combat exit (resolution, rewards, teardown) | Side effect of `CombatMode.Exit` | An explicit `finishCombat` call; `Exit` keeps only visual cleanup |
| Combat init and AI faction registration | `CombatMode.Enter`, visualization manager | Combat start path; AI discovers factions itself |
| Post-combat destination | `TacticalState.PostCombatReturnMode`, cleared every End Turn | Derived from the active encounter's combat type |
| Raid resume, start, floor advance, deploy fallback | `gui/guiraid/raidmode.go` | Verbs on `RaidRunner` |
| Recruit a commander | `gui/guioverworld/overworld_action_handler.go` | One `RecruitCommander` system function |
| Regenerate the map | `gui/guiexploration/exploration_panels_registry.go` | `gamesetup`, shared with boot |
| Battle and overworld recording start/stop | Mode `Enter`/`Exit` | Session lifecycle |

**Why it matters.** The mode manager's transition queue is a single slot that silently overwrites, and a mode's `Exit` is not guaranteed to run. Anything that must happen (teardown, rewards, raid end) cannot depend on it. The combat simulator has its own loop and its own target picker, so balance runs do not exercise the real AI or the real turn sequence.

**Direction.** Smallest first step: make combat exit an explicit call (fixes the lifecycle bugs without restructuring). Structural step: a headless turn driver beside `combatservices` that returns "what happened"; the GUI decides only what to animate. The same shape applies to `RaidRunner`.

### 3.2 The same fact stored more than once

| Fact | Copies | Consequence found |
|---|---|---|
| Movement speed, attack range | `common.Attributes` fields and `squadcore` components | Effects write the copy combat never reads: 11 gamedata effects do nothing on squads |
| Did a unit die this attack | `AttackEvent.WasKilled`, `DamageRecord.UnitsKilled`, `WouldSquadSurvive`, HP after healing | Resolute and kill hooks miss multi-hit kills; healing can revive a "killed" unit |
| Unit availability | `UnitRoster` counts, per-squad map, private index, and the ECS | Roster unusable after load; ghost entries after casualties |
| Unit attributes on a template | `UnitTemplate.Attributes` and `UnitTemplate.EntityData.Attributes` | Leader `Leadership = 20` override is dropped |
| Garrison membership | `GarrisonData.SquadIDs` on the node and `SquadData.GarrisonedAtNodeID` | Node becomes unattackable after one fight; squads stranded as "garrisoned" |
| Artifact equip state | Squad's list and inventory `EquippedOn` | Artifacts on a destroyed squad are stuck |
| Squad spellbook | Snapshot on the squad vs leader type ∩ commander unlocks | Newly unlocked spells never reach existing squads |
| Walkability | `Tile.Blocked`, `GameMap.validPositions`, global `core.WalkableGrid` | Avatar occupancy written into terrain and persisted |
| Screen and map size | `coords.ScreenInfo`, private copy in `CoordManager`, `config.Default*`, `templates.GameConfig.Display` | JSON display values never reach the coordinate system |
| Faction identity | `FactionType` enum, `ids.FactionID`, `OwnerID`, lowercase names in `initialsetup.json` | Beasts faction cannot be a starting faction |
| Overworld clock | `TickStateData.CurrentTick` and `OverworldTurnStateData.CurrentTurn` | Two singletons for one counter |
| Raid room/squad relation | Four ID lists plus fields on the squad | The code already works around its own stale lists |
| Combat round | `TurnStateData.CurrentRound` and `BattleRecorder.currentRound` | Correct only if the GUI remembers to push it |
| "Alive" in the GUI | `CanAct` (never set), `CurrentHealth > 0`, `IsSquadDestroyed` | Three definitions in one package |

**Direction.** For each row, pick the owner and derive the rest by query. The package sections name the recommended owner in every case. Where a cache is truly needed, it gets one owner and one invalidation point.

### 3.3 Rules enforced only by the caller, or implemented twice

- **Only in the GUI:** spell "already acted" check, action consumption and target side; AoE area resolution (the geometry lives in `visual/graphics`); artifact target validation and context construction; perk unlock gating; the node placement rule; the same-tile rule for engaging a threat.
- **Only when listing options, not when executing:** zone of control (`GetValidMovementTiles` applies it, `MoveSquad` does not); unit capacity (checked on one of several placement paths).
- **Twice with drift:** attack target patterns (`combatmath` real, `squadcore` preview); attack eligibility (`CanUnitAttack` and `getUnitsInRange`); squad attack range (four copies); valid movement tiles (combat system and AI); unit power (entity path and template path disagree); "build a random squad" (spawning, garrison, initial squads); teardown stripping (`combatservices` and `combatlifecycle`); threat intensity change (threat system and encounter resolver).
- **Two damage paths:** attacks record, apply, dispose and notify; spells write HP directly, do not dispose dead units in surviving squads, fire no hooks and write no battle log.

**Direction.** Each system entry point (`ExecuteSpellCast`, `ActivateArtifact`, `EquipPerk`, `MoveSquad`, `CreatePlayerNode`, `PlaceUnitInSquad`) validates its own rules. The GUI's pre-checks become feedback only. A useful test for any rule: could the AI or the simulator call this function directly and get correct behavior?

### 3.4 Game data loading

- `templates` loads 16 of the 24 JSON files. Six are loaded by hand in domain packages (`perkdata`, perk balance, artifact balance, combat balance, raid config, raid archetypes). Two files (`consumabledata.json`, `creaturemodifiers.json`) are referenced by nothing.
- Failure policy differs per loader: fatal, debug-fatal, print-and-continue (combat balance, both raid files), or log-and-skip. `combatbalanceconfig.json` failing leaves counterattack damage at zero; the combat simulator never loads it at all.
- Several validated values are shadowed by code defaults (`mind/behavior`, `mind/evaluation`, `mind/spawning`, `campaign/raid`, `world/worldgen`, `world/garrisongen`). Map generation config exists three times.
- Monsters, spells, artifacts and unit spells have no validator. Stat names in modifiers are free strings checked at cast time.
- "Keep in sync" lists are tied together by comment only (squad types, faction types).

**Direction.** One ordered `LoadGameData` that returns the first error; every file goes through `templates.Loader[T]` with a validator (domain packages can keep owning their types); delete code fallbacks for values the validator guarantees; make `mapgenconfig.json` the single source.

### 3.5 Save coverage is maintained by hand

Each persistent component has a hand-written `saved*` mirror, a field copy in `Save`, another in `Load`, and a hand-rebuilt component set that duplicates the entity factory. Unmapped entity IDs silently become 0. Four gaps exist today (section 6, S1 to S4), and the roster's private index is not rebuilt (S5). The overworld is not saved at all, yet the save callback is installed in both game modes.

**Direction.** Fix the gaps; marshal plain-data components directly instead of mirroring them; have `Load` call the same factory the game uses; make ID remapping strict for references that must resolve.

### 3.6 Randomness and determinism

The shared RNG in `core/common` starts from a fixed seed and is never seeded at boot, two map generators reseed it as a side effect, two others seed noise from the wall clock, and four places hold their own generators. Visual effects and AoE shape constructors draw from the simulation stream every frame. Several registry lookups iterate Go maps, so results differ per run even with a fixed seed (placement UI order, encounter choice, perk list order, squad role tie-breaks, raid squad creation order, AoE target order).

**Direction.** Decide the policy once (section 5, decision 14). Seed at boot and log the seed; give map generation and VFX their own streams; sort anything that came out of a map before indexing or displaying it.

### 3.7 Caches with manual invalidation

`SquadInfoCache` (shared by all modes, invalidated only by combat), `CombatQueryCache` (three live instances, one leaked ECS view per combat), four copies of the per-faction threat evaluator cache, `CachedBackground`, `CachedListWrapper`, `CachedTextAreaWrapper`, `cachedOverworldTurnStateID`, `RaidRunner.raidEntityID`, `PlayerData.Pos` (a pointer into a component). None holds data that could not be queried, three do nothing at all, and each adds a "you must call MarkDirty" obligation. The most recent commit on `main` is maintenance on one of them.

**Direction.** Delete them. The work they save is a handful of component lookups at turn-based call rates.

### 3.8 Half-finished features that look live

| Item | State | Size |
|---|---|---:|
| `campaign/overworld/victory` | Imported by nothing; `IsGameOver` never set | 191 lines + types |
| `SquadDeploymentMode` + service | Registered, never transitioned to | about 530 lines |
| Garrison tile generator and terrain | Debug menu only; raids never touch the map | 985 lines |
| Squad editor undo/redo | No binding, handler or button; dialogs advertise Ctrl+Z | 5 command types |
| Flee | `HandleFlee` has no callers; ESC swallowed | flee path in 3 packages |
| Encounter history, encounter entities | Write-only; one entity leaked per combat | |
| Fog of war (`IsRevealed`), stairs (`STAIRS_DOWN`) | Saved or placed, never read | |
| Squad `Formation`, `Morale`, `SquadLevel`, `TurnCount`; faction `Mana` | Stored and saved, never read; faction mana shown in the HUD | |
| Faction resources, containment, faction strength decline | Coded, inert with shipped data | |
| Render caches (three) | Inert or wrong | about 490 lines |
| Declarative panel registry, `StandardPanels`, `BuildPanel` options | 0 of 48 panels use it | about 300 lines |
| `reorder_squads_command.go` | No callers | 108 lines |
| `WorldTags` string registry | 30 writes, 6 reads | |

Each package section has a "Dead code" list with the smaller items. **Direction.** Section 5 lists the ones that need a product decision; everything else can simply be deleted.

### 3.9 Presentation state inside the simulation, and the reverse

- `Tile` holds an image, pixel coordinates and an overlay tint; `GameMap` carries a tint dirty flag; that is why `world/worldmapcore` imports `visual/graphics`. One tint slot per tile is shared by the threat map, the spell preview and a debug highlight, which overwrite each other.
- Map generators assign images while generating, and image selection is implemented a second time for the load path.
- Unit creation decodes a PNG inside `squadcore`/`templates`, which is why the simulator has its own squad factory. There is no image cache anywhere: about 20 `NewImageFromFile` call sites, one decode per unit created, the player sprite loaded in six places.
- The AI holds an attack queue that exists for GUI playback; the lifecycle package holds a GUI mode name; `TacticalState` (documented as UI-only) holds game-flow routing.

**Direction.** Generators emit terrain only and a single pass assigns images; overlays live in a per-owner layer in the renderer; unit creation takes an optional sprite step; one path-keyed image loader.

### 3.10 Identities as bare strings

Mode names (73 literals across gui/setup/campaign/mind, with behavior branching on them), artifact IDs and behavior keys, VFX kinds (unknown value silently becomes fire), room types (constants exist but are re-typed as strings in the GUI), POI and node types (duplicated constants), power profile (`"Balanced"` vs `"balanced"` already mismatched in `campaign/raid/deployment.go:68`), perk tier and category as raw ints in JSON, panel widgets fetched by string key (74 reads).

**Direction.** One set of typed constants per identity, defined where the concept is owned, validated at load.

### 3.11 Vocabulary collisions

- **Raid:** the roguelike run, and an NPC faction attacking a player node.
- **Garrison:** enemy squads in a raid room, squads stationed on an overworld node, and a map generator.
- **Power:** combat strength rating (`mind/evaluation`) and abilities (`tactical/powers`).
- **Squad type:** starting-squad recipes and encounter composition preferences.
- **Difficulty:** the global preset and the encounter level, both with a "Hard".
- **PanelType:** a string type in `framework` and an int type in `builders`.
- **Artifact GUI / Spell GUI:** each split across two packages with similar names.
- **Round vs turn:** perk and artifact "per round" state actually resets per faction turn.

**Direction.** Rename the overworld concept (`PendingRaid` → node attack), `evaluation`'s "power" → "strength" or "rating", and the per-turn state names, when those files are next touched.

### 3.12 Layering and package granularity

Dependencies that point the wrong way (from the import graph):

| Edge | Cause | Fix |
|---|---|---|
| `world/worldmapcore → visual/graphics` | `ColorMatrix` tint type stored on tiles | Move the type or move overlays out of `Tile` |
| `tactical/commander → campaign/overworld/core` | Walkability check; the package is an overworld package | Move to `campaign/commander` |
| `mind ↔ campaign` (both directions) | `combatlifecycle`/`encounter` are not AI; overworld rules live in `mind` | Section 4 |
| `tactical/squads/squadcommands → tactical/combat/*` | `MoveSquadCommand` | Move the command to combat |
| `templates → campaign/overworld/ids`, `→ world/garrisongen/roomtypes` | Typed IDs and one validator | Relocate `ids`; fold `roomtypes` |
| `setup/gamesetup → testing/bootstrap` | Debug artifact seeding | Move the two functions into `gamesetup` |
| `input → gui/framework` | Exploration input outside the mode system | Fold into `guiexploration` |
| `gui/guicombat → gui/guisquads` | `DetailPanelComponent` | Move it to `guicombat` |
| `gui/guistartmenu → setup/savesystem` | One yes/no question | Pass `hasSave` in |

Hubs: `setup/gamesetup` imports 38 internal packages (expected for a composition root); `gui/framework` imports six game packages directly, which is the price of `GUIQueries` living there.

Over-split areas: `campaign/overworld` (ten packages, strictly linear dependency chain, all data in `core`), `tactical/combat` (seven), `gui/guicombat` (five, with setter injection across the split), `visual` (five). A package earns its place by owning data or hiding something; where every function is exported for one sibling caller, it is a file.

### 3.13 Documentation and tooling drift

- `CLAUDE.md` points at `docs/project_documentation/…` (the real path is `resources/docs/project_documentation/…`, and `ECS_BEST_PRACTICES.md` is under `Process/`), cites reference implementations that do not exist (`squads/`, `gear/Inventory.go`, `systems/positionsystem.go`), references a deleted tech-debt file, names `GetEntitiesAtPosition` (the method is `GetAllEntityIDsAt`), and states two rules the code does not follow ("templates loads all gamedata", "`LoadGameData` is the single `log.Fatalf` point").
- `staticcheck.conf` does not exist and `staticcheck` was not installed in the review environment; 91 files are not gofmt-clean (the guide says 86).
- The `gui-architect` agent definition advertises "GUIQueries-only state access" and "PanelRegistry + Descriptor construction" as enforced principles; the code follows neither (57 `Queries.ECSManager` reach-throughs; every descriptor is an `OnCreate` closure).
- A root-level `test_init.go` is a second `package main` scratch file compiled by `go build ./...`.

---

## 4. Recommended package layout

Net effect: about 16 fewer packages, no new files beyond relocation. Every move is mechanical; do them after the dead code is gone so there is less to move (section 7, phase 5). `PROJECT_LAYOUT.md`, `PACKAGE_DEPENDENCIES.md` and the Key Systems list in `CLAUDE.md` change with each move.

| Area | Today | Recommended | Why |
|---|---|---|---|
| `campaign/overworld` | `core`, `node`, `threat`, `influence`, `faction`, `garrison`, `tick`, `victory`, `ids`, `overworldlog` | One `campaign/overworld` package (existing files keep their names), plus `ids` and `overworldlog` | The split gives no encapsulation: all data is in `core`, the chain is linear, most exports exist for one sibling |
| Overworld combat glue | `mind/encounter` (starters, resolvers, triggers, rewards) | `campaign/overworld/encounter` | Mirrors how `campaign/raid` owns its own starter and resolver; removes `mind → campaign` |
| Combat lifecycle | `mind/combatlifecycle` + session half of `mind/encounter` | `tactical/combat/combatlifecycle` | It is tactical infrastructure; two one-implementation interfaces disappear |
| `mind` | 6 packages | `ai`, `behavior`, `evaluation`, `spawning` | `spawning` takes level and squad types as parameters and stops importing `campaign` |
| `tactical/combat` | 7 packages | 5: delete `combatdisposal`, merge `combatmath` into `combatcore`, make `battlelog` a leaf | The math/systems split is not honored in either direction |
| `tactical/squads` | 6 packages | 5: dissolve `squadservices` into `roster`; move `MoveSquadCommand` to combat | Removes the upward `squads → combat` dependency |
| `tactical/commander` | under `tactical/` | `campaign/commander`; `FindCommanderForSquad` moves to `roster` | Removes `tactical → campaign` |
| `gui/guicombat` | 5 packages | 3: merge `combatbase` and `combatinput` back in; keep `combatanimation`, `combatvisualization` | The split forced setter injection and a larger public surface |
| Combat sub-panels | `gui/guispells`, `gui/guiartifacts`, `gui/guiinspect` | Rename or move under `gui/guicombat/` (optional) | They are combat-only and collide by name with the out-of-combat screens |
| `gui/guinodeplacement` | own package | Fold into `gui/guioverworld` | One consumer in each direction |
| `gui/widgets` | own package | Dissolve after the caches go (optional) | About 110 lines remain |
| `visual` | 5 packages | 3: dissolve `graphics`; merge `maprender` into `rendering` | `graphics` has no drawing code; the other two are always called as a pair |
| `input` | top-level package | Fold into `gui/guiexploration` | It is one mode's input handler |
| `world/garrisongen` | DAG + tile generator + `roomtypes` | Depends on decision 2 (section 5) | |
| `templates` | data + entity factory + formatter | Entity factory moves to `squadcore`; `templates` becomes a pure data leaf | Its only caller overrides half of what the factory does |
| `testing/bootstrap` | imported by shipping code | Two functions move to `gamesetup`; package deleted | |

Leave as they are: `core/*`, `setup/*`, `templates` as one flat package, `gui/framework`, `gui/builders`, `gui/specs`, `gui/widgetresources`, `tactical/powers/*` (six packages, clean roles), `squadcore` (do not split further).

---

## 5. Decisions needed from you

These are product decisions that determine whether code is deleted or finished. Nothing here asks for a new feature; each item is already half-present in the code. Recommendations are given where the evidence points one way.

| # | Question | If yes | If no | Recommendation |
|---:|---|---|---|---|
| 1 | Can the overworld be won or lost? | Wire `victory` into the tick and fix its four latent bugs | Delete the package, `IsGameOver`, the config block | Decide; do not leave it half-present |
| 2 | Are raids fought on generated floor maps? | `GarrisonRaidGenerator` takes the raid's DAG as input (L) | Delete `generator.go` + `terrain.go` and room-size data (S, about 985 lines) | Delete unless this is planned soon |
| 3 | Is fleeing a feature? | Add the trigger; the exit path already exists | Remove `HandleFlee`, `ExitFlee` handling, raid retreat-on-flee | Decide |
| 4 | Should the squad editor have undo? | Bind it and clear history on exit | Remove the editor's command wrappers and the Ctrl+Z text | Either; clear history on exit in both cases |
| 5 | Keep `SquadDeploymentMode`? | Rebuild on `combatlifecycle` | Delete mode, service, bindings, layout constants | Delete |
| 6 | Is "attack mode" a real state? | Gate the click path on it and show it | Remove the flag, button and 1/2/3 target hotkeys | Remove |
| 7 | Can healing save a unit that took lethal damage in the same exchange? | Survival prediction includes healing | Heal targets exclude units predicted dead | Pick one; today the code does both |
| 8 | What happens when a squad leader dies? | (define: promote successor / leaderless penalty) | | Define once in `SetSquadLeader` |
| 9 | Do perk role restrictions restrict equipping? | Enforce in `EquipPerk` | Delete the field from data and UI | Decide |
| 10 | Are `Formation`, `Morale`, `SquadLevel`, `TurnCount` planned mechanics? | Keep until built | Delete the fields, the parameter at nine call sites, the save fields | Delete; re-add with the feature |
| 11 | Are faction resources, containment and faction weakening meant to work? | Feed them (income, costs, strength loss) | Delete the inert halves | Decide per mechanic |
| 12 | Should Cone spells differ from Line spells? | Implement a widening cone | Remove the type from code and data | Either; today the data advertises a mechanic that does not exist |
| 13 | Keep fog-of-war and stairs leftovers? | (they are features to build later) | Delete `IsRevealed`, `revealAll`, `STAIRS_DOWN`, `PlaceStairs` | Delete |
| 14 | Do you want reproducible runs from a seed? | Seed stored in the save; no wall-clock seeds; sorted registries | Seed from entropy at boot; still stop VFX using the sim RNG | Decide; it sets the RNG policy |
| 15 | Artifact on a destroyed squad: returned or lost? One charge per behavior or per copy? | | | Define once |
| 16 | Is manual raid deployment planned? | Keep the deploy panel | Collapse it to a confirm step | Decide |
| 17 | Should the AI's positional and support layers affect decisions? | Normalize all layers to one range before weighting | Keep them for the debug overlay only | Decide; today they contribute under one point |
| 18 | Who are overworld events for? | Player: the GUI panel reads `EventLog` | Debug export only: delete `EventLog`, keep the recorder | Decide |
| 19 | Should rewards go to the squads that fought, or the whole roster? | Capture the deployed list at combat start | Leave as is | Probably the squads that fought |

---

## 6. Probable bugs index

All found by reading code; none reproduced in a running game. **Re-read = yes** means the central claim was checked against the source a second time during synthesis. Confidence is the original reviewer's. Details and line references are in the section named in the last column.

### Game loop and lifecycle

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| L1 | Raid defeat is never processed: the resolver sets `RaidDefeat` before the callback, whose guard then skips it | High | yes | 16 `campaign/raid` |
| L2 | Taking the final floor's stairs does not end the raid | High | | 16, 19 `gui/guiraid` |
| L3 | A second raid in one session has no post-combat processing (callback cleared, never re-registered) | High, conditional | | 16 |
| L4 | A battle that ends during the AI animation chain skips combat exit; the encounter stays active and the next combat fails to start | High | mechanism | 19 `gui/guicombat` |
| L5 | Raid battles lasting past the first End Turn return to exploration, not the raid (`Reset()` clears the return mode) | High | yes | 19 |
| L6 | The player cannot flee: ESC is swallowed, no button, `HandleFlee` has no callers | High | yes | 19 |
| L7 | Winning an overworld fight never weakens the threat: `CountDeadUnits` is always 0 because dead units are disposed mid-combat | High | yes | 15 `mind/combatlifecycle`, `mind/encounter` |
| L8 | An NPC-garrisoned node becomes unattackable after one fight (stale squad IDs in `GarrisonData`) | High / medium-high | | 16 `garrison`, 15 |
| L9 | The overworld cannot be won or lost, and is not saved | High | yes | 16 |
| L10 | A failed combat start leaves factions, action states and enemy squads in the world | Medium | | 15 `mind/encounter` |
| L11 | One encounter entity is leaked per combat | High | | 15 |
| L12 | A load that fails after `ResetWorld` leaves a half-built world | High logic, low likelihood | | 10 `game_main` |
| L13 | Player squads can be stranded as "garrisoned" after their node is captured | Medium | | 16 |
| L14 | Auto-deploy can pull garrisoned squads into a field battle | Medium | | 15 |

### Combat rules

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| C1 | Perk per-round state is wiped on every `EndTurn` right after TurnStart hooks set it: Counterpunch and Deadshot's Patience never fire after the first turn change; Reckless Assault's downside never applies on the enemy turn | High | yes | 14 `perks` |
| C2 | Kills from accumulated hits are not seen as kills (per-hit events use stale HP): Resolute and kill hooks miss them; kill counts wrong | High | yes | 13 `combatcore` |
| C3 | A unit already lethally hit is "killed" again by each later hit | High | | 13 |
| C4 | Movement-speed and attack-range effects do nothing on squad units (11 gamedata entries) | High | yes | 10 `core/common`, 12 |
| C5 | Buff spells (War Cry, Arcane Shield) can only be cast on enemies | High | yes | 18 `gui/guispells` |
| C6 | AoE shapes are not bounds-clipped: they wrap across map edges and can hit squads on the opposite edge; a negative index reaches the tile slice (panic path) | High | yes | 20 `visual/graphics` |
| C7 | Cone spells hit exactly the same tiles as Line spells; even rectangle sizes round up to odd | High | yes | 20 |
| C8 | Guardian Protocol only works if the attacked squad also has the perk | High | | 14 |
| C9 | "Next attack" perk bonuses are consumed by the first unit's first hit; Adaptive Armor and Grudge Bearer cap inside one attack | High | | 14 |
| C10 | Healing can revive a unit recorded as killed; a squad whose last unit would have been healed is disposed anyway | Medium-high | | 13 |
| C11 | Damage redirect leaves the event stale and can spend the once-per-battle Resolute charge needlessly | Medium-high | | 13 |
| C12 | Leader `Leadership = 20` override is dropped for every template-built squad (the simulator behaves differently) | High | yes | 12 `squadcore` |
| C13 | Generated enemy squads silently lose units: slot rows fall outside the 3×3 grid; multi-cell units block following slots | High | yes | 15 `mind/spawning` |
| C14 | Multi-cell units can be placed overlapping or off-grid from the squad editor | High | | 12 |
| C15 | Clamped debuffs cause permanent stat drift; effect cleanup depends on faction membership | High in code | | 14 `effects` |
| C16 | Combat movement has no terrain or map-bounds check | Medium | | 13 |
| C17 | Zone of control is not enforced in `MoveSquad`, only when listing tiles | High | | 13 |
| C18 | Newly unlocked spells never become castable for existing squads; squads created later get no spells | High | | 14 `spells` |
| C19 | Artifacts on a destroyed player squad become permanently stuck | Medium-high | | 14 `artifacts` |
| C20 | Units killed by a spell in a surviving squad are not disposed until the next physical attack | Medium | | 14 |
| C21 | `combatbalanceconfig.json` fails silently; the combat simulator never loads it, so simulated counterattacks deal 1 damage | High | yes | 13 `combatmath` |
| C22 | Every squad gets two action-state entities; one ECS view is leaked per combat | High | yes | 13 `combatstate` |
| C23 | Resolute cannot trigger during the opening enemy turn | Medium-high | | 14 |
| C24 | Rotated line spells play their visual effect on the unrotated tiles; AoE rotation is never bound to a key | High | | 20, 17 |
| C25 | AI evaluator cache splits after the first combat (`Reset` replaces the map the factory closure still holds): leak plus double computation | High | yes | 15 `mind/ai` |
| C26 | "Clear all" in deployment moves every squad in the world to (0,0) (unreachable today) | Medium-high | | 12 |

### Save and load

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| S1 | Roster units not assigned to a squad are not saved; their IDs load as 0 | High | yes | 10 `chunks` |
| S2 | Equipped perks are not saved | High | yes | 10, 14 |
| S3 | Raid garrison squad data is attached to orphan entities on load; reserves never activate and garrison squads leak | High | | 10 |
| S4 | `RaidStateData.PlayerEntityID` is not saved (reward target is 0 after load) | High | yes | 10 |
| S5 | After loading, roster units cannot be assigned or unassigned (private index not rebuilt) | High | yes | 12 `roster` |

### Roster and editing

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| R1 | Casualties leave ghost roster entries and leak roster and squad capacity | High / medium | | 12 |
| R2 | Undoing a purchase after the unit was placed in a squad disposes a live squad member | Medium-high | | 18 `gui/guisquads` |
| R3 | `AddUnitCommand` places the first unit of a type, not the unit the player picked | High | | 12 |
| R4 | Two squads with the same name: clicking the second selects the first | High | | 18 |

### GUI and rendering

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| G1 | Six panels are never created (four overworld, two node placement): spec names missing, nil result, nil-checked | High | yes | 17 `gui/builders` |
| G2 | Mode hotkeys fire while typing in a modal text dialog | High | | 17 `gui/framework` |
| G3 | Clicks on UI panels also reach map handlers | High mechanism | | 17 |
| G4 | List refresh scrambles the perk panel and squad selector layout | High | yes | 18 |
| G5 | Detail text areas built with the cached wrapper are blank or displaced and cannot scroll | Medium-high | | 17 `gui/widgets` |
| G6 | ESC does not leave unit view; ESC does nothing in the squad editor in roguelike mode | High | | 18 |
| G7 | Squad editor undo is unreachable although dialogs advertise Ctrl+Z | High | | 18 |
| G8 | Attack animation can highlight units other than those that attacked or were hit | Medium | | 19 |
| G9 | Multi-cell units are mis-rotated on the attack-animation grid | Medium-high | | 20 `visual/combatrender` |
| G10 | Cloud effect layers ignore the scale factor; pulse speed scales with AoE size | High | | 20 `visual/vfx` |
| G11 | Raid summary always says "Room Cleared", including after a defeat | High | | 19 |
| G12 | Overworld "Age: N ticks" shows the creation tick | High | | 19 |
| G13 | Spell preview and threat overlay erase each other (one tint slot per tile) | Medium | | 19, 20 |
| G14 | Stale tile tints after switching between the full-map and viewport renderers | Medium | | 20 `visual/maprender` |

### World, data and determinism

| ID | Bug | Confidence | Re-read | Section |
|---|---|---|---|---|
| W1 | One overworld POI tile is overwritten by stairs | High | | 11 `worldmapcore` |
| W2 | Cavern `StartingPosition` can land on a pillar (wall) | Medium | | 11 |
| W3 | Cavern can be disconnected: terrain changes after the connectivity pass | Medium | | 11 `worldgen` |
| W4 | `initialsetup.json` fallback position 4 is outside the map (x=120 on a 100-wide map) | High value, medium trigger | | 11 `templates` |
| W5 | `GetFortificationStrengthGain` ignores difficulty (always 1) | High | | 16 `core` |
| W6 | Influence radius not updated when a threat is weakened; nil dereference in `finalizeNetModifiers` | High / latent | | 16 |
| W7 | Gameplay rolls start from the same RNG state on every launch; VFX advances the simulation RNG per drawn frame | High | yes (VFX) | 10, 20 |
| W8 | Placement UI order, default node type and perk list order vary per run (map iteration) | High | | 16, 14 |
| W9 | Raid deployment asks for power profile `"balanced"`; the profile is `"Balanced"` (silent fallback) | High | | 15 `evaluation` |

---

## 7. Roadmap

Ordered so that each phase makes the next one safer. Phases 0 and 1 are small, independent fixes; phases 2 to 4 are the design work; phases 5 to 7 are mostly mechanical. Within a phase, items are independent unless noted.

### Phase 0: decide, confirm, clear the ground

1. Answer the decisions in section 5 (at least 1 to 6, 10 and 13, which only decide what gets deleted).
2. Confirm the two bugs most worth a quick in-game check before scheduling: L7 (victory never reduces intensity) and L8 (attack a garrisoned node twice). Also open Unit Purchase to check G5 and the overworld to check G1.
3. Delete what needs no decision: the three render caches, `test_init.go`, `reorder_squads_command.go`, the nine dead `CombatService`/`AICoordinator` methods, `WorldTags` writes (after replacing six reads), the 26 unused `specs` constants, orphaned JSON files, and the per-package "Dead code" lists.
4. Fix `CLAUDE.md` (section 3.13) so the guide matches the tree; add `staticcheck.conf` and install the tool.

### Phase 1: core-loop correctness (small, high value)

1. Combat exit as an explicit `finishCombat` call; split `TacticalState.Reset()`; derive the post-combat destination from the encounter (L4, L5).
2. Raid end-of-run: `RaidRunner` owns status transitions; final-floor stairs go through the advance path; callback registered per raid (L1, L2, L3).
3. Perk state reset moved to turn start; delete the round-end reset (C1, C23).
4. Casualties counted by snapshot-and-diff, as raid already does (L7).
5. Garrison: clear node-side IDs on teardown, or move to the single-owner model directly (L8, L13).
6. Save gaps S1 to S5.
7. AoE bounds clipping and the lower-bound check in `ApplyColorMatrix` (C6); buff target side (C5).
8. Combat balance config returns an error and is fatal at boot; the simulator loads it (C21).
9. Spawning slot table and footprint-aware placement (C13, C14).
10. Six missing panels and central input gating for dialogs and pointer-over-UI (G1, G2, G3).
11. `AIController.Reset` uses `clear`; AI discovers factions itself (C25).

### Phase 2: one source of truth

Work through the table in section 3.2. Suggested order, highest bug yield first:

1. Movement speed and attack range (C4).
2. Pending-HP model for an attack; `WasKilled`, `UnitsKilled`, survival and disposal agree (C2, C3, C10, C11). Requires decision 7.
3. `UnitRoster` reduced to owned unit IDs; rosters pruned on death (S5, R1, R3).
4. One unit builder in `squadcore`; `UnitTemplate` without placement fields (C12); the simulator uses the game's path.
5. Action state as a component on the squad entity; delete `CombatQueryCache` and `CombatFactionManager` (C22).
6. Spellbook as a query (C18); artifact equip state single-sourced (C19); effects store the applied delta (C15).
7. Walkability owned by the map; screen and map config as one object built after data load.
8. Faction identity as `ids.FactionID` only; delete `ThreatType`; one definition of "threat"; one turn counter.
9. Raid room/squad relation reduced to the squad's own component.

### Phase 3: rules into the simulation

1. Headless turn driver on `CombatService`; the simulator uses it; victory checked after each resolved attack.
2. `ExecuteSpellCast` validates acted state and targets and marks the action; AoE geometry moves out of `visual/graphics`; spell damage shares the attack path's apply-and-dispose step (C20).
3. One artifact activation entry point (context, logger, target and charge checks); `EquipPerk` checks unlock; `MoveSquad` enforces zone of control (C17); `CreatePlayerNode` validates placement.
4. `RaidRunner` verbs (`Resume`, start, room selection, floor advancement); `RecruitCommander`; `gamesetup.RegenerateMap`.
5. Attack animation built from the `CombatLog` (G8); AI returns executed attacks, with no queue inside the AI.
6. Mutators for `ActionStateData`; artifacts, AI and undo stop writing its fields directly.

### Phase 4: data loading and randomness

1. One `LoadGameData() error`; all files through `Loader[T]`; validators for monsters, spells, artifacts, unit spells, stat names, raid archetypes; one cross-file reference pass.
2. Delete code fallbacks behind validated values; `mapgenconfig.json` becomes the single source (removes `setup/gamesetup/mapgenconfig.go`).
3. RNG policy per decision 14; private streams for map generation and VFX; sorted registry outputs (W7, W8).
4. One path-keyed image loader.

### Phase 5: package layout

The moves in section 4, one area at a time, each as its own change with the three layout documents updated. Suggested order: `tactical/combat` (7 → 5), `campaign/overworld` merge, `combatlifecycle` and `encounter` relocation, `commander`, `visual` and `input`, GUI merges.

### Phase 6: GUI infrastructure

1. Delete the dead declarative half of the panel registry, `StandardPanels`, `BuildTypedPanel`, the `BuildPanel` option layer; one `CreateTextPanel` helper; fraction-to-pixel helpers on `LayoutConfig` (139 call sites).
2. Delete `SquadInfoCache`; narrow `GUIQueries` to view models.
3. Mode-name constants; one transition helper that logs on a missing mode; one `CombatInputMode` enum for the five flags.
4. `guisquads`: `SetEntries` instead of rebuilding lists (G4); one `artifactTab`; typed widget fields instead of `Custom` string keys (optional, mode by mode).
5. Shared helpers for the four sub-menu builders, eight detail panels and nine widget getters.

### Phase 7: save system hardening

Marshal plain-data components directly; `Load` goes through the game's factories; strict ID remapping; chunks restore data only and one place attaches images; decide whether the overworld is saved.

---

## 8. Keeping the design healthy

Rules distilled from the themes above. Each one would have prevented a class of findings in this review.

1. **One owner per fact.** Before adding a field, check whether the value can be queried. If a cache is truly needed, it has one owner and one invalidation point, and it is not shared through a context struct.
2. **The simulation is complete without the GUI.** Every rule is enforced at the system entry point. Test: could the AI or the combat simulator call this function directly and get correct behavior?
3. **Mode `Enter` and `Exit` do visual setup and cleanup only.** Anything that must happen is an explicit call.
4. **All game data goes through `Loader[T]` with a validator, and boot fails loudly.** No code default for a value the validator guarantees.
5. **A persistent component changes together with its save chunk.** Prefer marshalling the component itself over a mirror struct.
6. **A package owns data or hides something.** If every function is exported for one sibling, it is a file. A new package needs a stated reason (a real import cycle, or a second independent consumer).
7. **No fields or parameters for future features.** Delete a field when its last reader goes; re-add it with the feature. "Kept for compatibility" comments mark code to delete.
8. **Identities are typed constants defined once** (mode names, faction IDs, room types, effect kinds), validated at load.
9. **Presentation never draws from the simulation RNG and never writes into the world model.**
10. **One way to do each job.** When a second helper for the same job appears (panel builders, squad lookup, logging, config access), remove one in the same change.
11. **The simulator runs the game's code paths.** With almost no GUI tests, it is the cheapest regression check available; it loses that value each time it re-implements something.
12. **Tooling.** Keep `staticcheck` installed and configured; run `golang.org/x/tools/cmd/deadcode` occasionally, since it reports unreachable exported functions, which is most of what this review found by hand; do the one-off gofmt pass so `gofmt` can run repo-wide.

### What is already good and should be preserved

- `squadcore`'s view-backed, ID-based query set and `ApplyUnitComponents`.
- `templates.Loader[T]`, the ordered loader table, and the panic-free validators.
- The save envelope: checksum, atomic write, backup, pre-flight validation, two-phase load with ID remapping.
- `processAttack` and its eight named steps; the record-then-apply structure of `ExecuteAttackAction`.
- `PowerOrchestrator.WirePipeline`: one place that states subscriber order per event.
- `combattypes.PerkDispatcher` with its no-op default; the perk behavior files and `ValidateHookCoverage`.
- `Reward` / `GrantTarget` / `Grant` as the single reward path; the starter/resolver pair per combat type.
- `ActionMap` and the single bindings table; `SubMenuController`; `ModeBuilder`.
- `gui/guiprogression` as the model for a mode; `UnitPurchaseMode` delegating its transaction to the system layer.
- `visual/combatrender`'s independence from `tactical/*`; `QuadBatch`.
- The cavern generator's terrain-map-then-tiles pipeline; the generator registry.
- The faction utility-scoring shape; `ClassifyInteraction`; `DisposeRaidEntities`.
- `MoveEntity` / `MoveSquadAndMembers` / `RegisterEntityPosition` and the value-keyed `PositionSystem`.

---

## 9. Package-by-package findings: how the sections are laid out

Sections 10 to 20 follow. Each layer section opens with an overview of how its packages relate, then covers each package: role and verdict, design findings (severity, evidence, problem, direction, effort and risk), mechanics notes, probable bugs, dead code, performance, and what to leave alone. "Slice" means the group of packages covered by that section.

The layers were reviewed independently, so a few package sections name different destinations for the same move (for example, where `RecruitCommander` or `DetailPanelComponent` should live). Where a package section and section 4 disagree about a destination, section 4 is the reconciled answer.

---

## 10. Core, entry point, setup and save system

Packages: `core/common`, `core/coords`, `core/config`, `game_main` (+ root `test_init.go`), `setup/gamesetup`, `setup/savesystem`, `setup/savesystem/chunks`.

Every non-test file in these packages was read in full. "Unused" claims were checked with a repo-wide grep (including `tools/` and tests). Bug claims were traced through the call path and carry a confidence.

### Layer overview

- **Layering is sound.** `core/config` ← `core/coords` ← `core/common` ← everything; `setup/*` and `game_main` sit on top and are the only places that know about every subsystem. No import cycles or upward imports were found, with one exception: `setup/gamesetup/bootstrap.go:20` imports the non-test package `testing/bootstrap` into the shipping binary.
- **Three problems cut across the slice**, and each shows up again in the per-package sections:
  1. **Screen and map dimensions have four sources of truth** (`coords.ScreenInfo`, the private copy inside `coords.CoordManager`, `config.Default*`, `templates.GameConfig.Display`). The JSON values never reach the two that the game actually uses.
  2. **Boot and data loading are spread over three places** (`game_main.SetupSharedSystems`, `GameBootstrap.LoadGameData`, `GameBootstrap.CreatePlayer`) with five different failure policies.
  3. **Save coverage is maintained by hand** as a mirror of the component structs and of the entity factories. It has already drifted: four pieces of state are silently lost on save/load (see `setup/savesystem/chunks` → Probable bugs).
- **Layout recommendation:** keep the package tree as it is. Nothing in this slice needs to be split or merged. The work is removing duplicate state and duplicate code paths inside the existing packages.

---

### core/common (893 lines, tests: yes)

**Role:** ECS wrapper (`EntityManager`), shared components (`Attributes`, `Name`, `Renderable`, `ResourceStockpile`, `Player`), the spatial grid, and the shared RNG.
**Verdict:** Needs attention. The helpers are good, but the package carries duplicate state (speed/range, tags, player position) and an RNG whose seeding is inconsistent.

#### Design findings

##### [High] Movement speed and attack range are stored twice, and the two copies have different readers and writers
- Evidence:
  - `core/common/commoncomponents.go:31-32` (`Attributes.MovementSpeed`, `Attributes.AttackRange`), with getters at `:154-167`.
  - `tactical/squads/squadcore/units.go:105-111` adds separate `AttackRangeComponent` / `MovementSpeedComponent` to every unit.
  - Combat reads the components: `tactical/squads/squadcore/squadqueries.go:344`, `tactical/combat/combatcore/combatactionsystem.go:229-230` ("correct source for attack range").
  - Commanders read the attribute: `tactical/commander/system.go:15` ("Attributes.MovementSpeed is the canonical source of truth"), `tactical/commander/turnstate.go:68`.
  - Effects write the attribute: `tactical/powers/effects/system.go:186-189`.
  - The save file stores both for every unit: `setup/savesystem/chunks/shared_types.go:25-26` and `squad_chunk.go:85-86`.
- Problem: two fields claim to be canonical. Which one is live depends on whether the entity is a commander or a unit, and nothing in the types says so. See Probable bugs: speed/range effects on units write the copy combat never reads.
- Direction: pick one home. The simplest is to delete `MovementSpeed` and `AttackRange` from `Attributes` users on units (or delete the two squadcore components and read `Attributes` everywhere), then make effects, combat, commanders and the save chunk all use the survivor. Whichever is chosen, the save struct should carry it once.
- Effort: M. Risk: medium (touches combat, effects, save format).

##### [Medium] The shared RNG is one mutable global with inconsistent seeding
- Evidence:
  - `core/common/randnumgen.go:8` seeds with the constants `(1, 2)`; no production code path seeds it from entropy at boot (the only non-test callers of `SetRNGSeed` are the combat simulator and two generators).
  - `world/worldgen/gen_rooms_corridors.go:46-48` and `world/garrisongen/generator.go:38-40` reseed the *global* gameplay RNG as a side effect of generating a map with a non-zero seed.
  - `world/worldgen/gen_cavern.go:102-105` and `gen_overworld.go:104` use a separate wall-clock seed for noise, then draw from the shared RNG too (`gen_cavern.go:193-219`).
  - `mind/combatlifecycle/reward.go:21` builds a second `math/rand` v1 source from `NextSeed()` at package init; `tactical/squads/unitprogression/experience.go:5` and `gui/guiunitview/unitviewmode.go:29` use further sources.
  - RNG state is not part of the save file.
- Problem: the game is neither deterministic (wall-clock noise seed) nor properly random (gameplay rolls start from the same PCG state on every launch). Generating a seeded map quietly resets every later combat roll. There are 132 call sites of the `common.*` roll helpers, so the global is the de-facto RNG service, but its lifecycle is not owned by anyone.
- Direction: decide the policy once in `common`: seed from entropy at boot (log the seed), and expose a way to derive an independent stream (`common.NewStream(seed)`) for map generation so generators stop reseeding the global. Fold the stray `math/rand` v1 sources into derived streams. If reproducible runs matter, store the seed in the save envelope.
- Effort: S–M. Risk: low for the boot seed, medium for generator streams (map output changes for a given seed).

##### [Medium] Core component and tag registration is copied in three places, and `WorldTags` is a write-mostly string registry
- Evidence:
  - `common`'s own six components are assigned from outside the package in `setup/gamesetup/ecsinit.go:37-44`, `testing/fixtures.go` and `tools/combat_analysis/combat_simulator/bootstrap.go` (18 near-identical lines in total). Every other subsystem self-registers through `RegisterSubsystem`.
  - `EntityManager.WorldTags` (`core/common/ecsutil.go:38`) is written 30 times in production code and read 6 times: five reads of `"players"` and one of `"raidstate"` (`game_main/setup.go:221`), where `raid.RaidStateTag` already exists.
  - `"players"` is the only tag with no package variable, so it is built twice: `setup/gamesetup/playerinit.go:67-68` and `setup/savesystem/chunks/player_chunk.go:186-187`.
- Problem: two ways to reach a tag (typed package var vs. string lookup), and the one tag that only exists as a string must be rebuilt by whoever creates a player. Adding a core component means editing three files.
- Direction: add `common.InitCoreComponents(em)` (components + `AllEntitiesTag`, `RenderablesTag`, a new `PlayerTag`) and call it from the three sites. Replace the six string reads with typed tags, then delete `WorldTags` and the 30 writes.
- Effort: S. Risk: low.

##### [Medium] `PlayerData` caches a pointer into a component and carries write-only state
- Evidence:
  - `core/common/playerdata.go:29` (`Pos *coords.LogicalPosition`), set at `setup/gamesetup/playerinit.go:73`, re-pointed after load at `setup/gamesetup/savehelpers.go:43-45`.
  - Player movement writes through the pointer and updates the grid by hand, bypassing `EntityManager.MoveEntity`: `input/cameracontroller.go:113-122`, `gui/guiexploration/exploration_panels_registry.go:68-73`.
  - `PlayerInputStates.HasKeyInput` is set in eight places in `input/cameracontroller.go` and cleared in `game_main/main.go:65-67`; nothing reads it.
  - `PlayerData.PlayerAttributes` (`playerdata.go:36`) has no callers.
- Problem: the pointer is exactly the cached relationship the project's ECS rules forbid; it dangles across a world reset until `RestorePlayerData` runs, and it invites the two hand-rolled move paths above.
- Direction: delete `PlayerInputStates`, `HasKeyInput`, `PlayerAttributes` and `HandleInput`'s reset. Keep `PlayerEntityID`. Either keep `Pos` strictly read-only (camera centre) and route the two writers through `em.MoveEntity`, or replace it with a small accessor that looks the component up.
- Effort: S. Risk: low.

##### [Low] Entity disposal pushes a two-step protocol onto every caller
- Evidence: `core/common/ecsutil.go:248-262`. All 13 call sites first fetch the position and then pass it in (for example `tactical/squads/squadcore/squadcreation.go:435-436`, `tactical/squads/squadservices/unit_purchase_service.go:133-134`). `DisposeEntitiesByTag` (`ecsutil.go:268-277`) already does the lookup itself. Two sites still call `World.DisposeEntities` directly (`tactical/combat/combatservices/combat_service.go:325`, `campaign/overworld/node/system.go:116`).
- Problem: boilerplate, plus a documented "if pos != nil … else …" dance in CLAUDE.md that the function does not actually need (it already accepts a nil position).
- Direction: make it `em.DisposeEntity(entity)` and do the position lookup inside. Update the CLAUDE.md snippet.
- Effort: S. Risk: low.

##### [Low] Stat formulas are hard-coded in `core/common`, away from the balance data
- Evidence: `core/common/commoncomponents.go:66-147`. Coefficients such as `20 + Strength*2`, `Strength/2 + Weapon*2`, `Magic*3`, `Dexterity/3` are literals; only the caps come from `config`. Other combat tuning lives in `combatmath`'s balance JSON.
- Problem: tuning a unit's damage curve means editing a core package; tuning its cap means editing JSON. `Attributes` is also the one component that openly breaks the "components have zero logic" rule in CLAUDE.md.
- Direction: accept pure derived-stat getters as a documented exception (they are the least-bad home), but move the coefficients next to the caps so one file holds the whole formula. Do not turn this into a formula engine.
- Effort: S. Risk: low (behaviour-preserving if the defaults match the literals).

#### Mechanics notes
- `Attributes` mixes base stats with runtime state (`CurrentHealth`, `CanAct`), and effects modify the base stats in place (`tactical/powers/effects/system.go:175-189`). Max HP is derived from Strength (`commoncomponents.go:76-78`), so a Strength buff or debuff changes max HP without touching current HP. This is outside this slice, but the root is that `Attributes` has no base/modified split.
- `Leadership` exists on `Attributes` and again on `squadcore.LeaderData` (both saved: `squad_chunk.go:125-128`, `shared_types.go:22`). Only `Attributes.Leadership` feeds capacity (`commoncomponents.go:136`).

#### Probable bugs
- **Speed and range effects do nothing for squad units** (confidence: medium-high). `effects/system.go:186-189` adds the modifier to `Attributes.MovementSpeed` / `AttackRange`; squad movement and targeting read `MovementSpeedData.Speed` / `AttackRangeData.Range` (`squadqueries.go:344`, `combatactionsystem.go:230`). I did not check whether any spell or perk currently uses those two stat types.
- **Gameplay rolls start from the same RNG state on every launch** (confidence: high that the global is never entropy-seeded; medium that the repeat is visible, because map generation consumes a variable number of draws first).

#### Dead code
- `PositionSystem.manager` field and the constructor parameter (`positionsystem.go:18,23`): never read. The comment at `game_main/setup.go:150-152` ("the spatial grid caches the old manager pointer") is therefore describing something that does not happen.
- `PositionSystem.GetOccupiedPositions`, `Clear`, `ValidatePositionSync`: no callers. `GetEntityCount`: tests only.
- `PlayerData.PlayerAttributes`, `PlayerInputStates` (see above).
- `EntityManager.GetComponent` has one outside caller (`gui/guisquads/squadlists.go:88`); `GetComponentTypeByID` covers it.

#### Leave alone
- `GetComponentType` / `GetComponentTypeByID`, `FindEntityByID`.
- `RegisterSubsystem` / `InitializeSubsystems`, including the documented init-order rule.
- `MoveEntity` / `MoveSquadAndMembers` / `RegisterEntityPosition` / `UnregisterEntityPosition`: the upsert semantics and error handling are deliberate and well explained.
- `PositionSystem`'s core (value-keyed map, add/remove/query).
- `GridMovement` and `ResourceStockpile`: small, used, fine.
- `GlobalPositionSystem` as a global is a documented trade-off; not worth the churn to inject.

---

### core/coords (354 lines, tests: no)

**Role:** position types and logical/pixel/screen/index conversion.
**Verdict:** Needs attention. The conversion maths is fine, but screen configuration lives in two singletons that are snapshotted before the JSON config is loaded.

#### Design findings

##### [High] Two screen-config singletons, built at package init, so the JSON display settings are never applied
- Evidence:
  - `core/coords/cordmanager.go:12` (`var ScreenInfo = NewScreenData()`) and `:26-29` (`init()` builds `CoordManager` from a *second* `NewScreenData()` and copies the fields into private ones at `:99-106`).
  - `NewScreenData` hard-codes `DungeonWidth: 100, DungeonHeight: 80` (`:58-59`) and reads `config.DefaultTilePixels` / `DefaultScaleFactor` / `DefaultRightPadding` at init time (`:61-71`).
  - `templates/gameconfig.go:26-44` pushes the JSON values into `config` later, during `LoadGameData`. Nothing rebuilds `ScreenInfo` or `CoordManager` afterwards (the only `NewCoordinateManager` calls outside `init()` are in tests).
  - `config.DefaultMapWidth` / `DefaultMapHeight` have no readers at all.
  - The walkable grid is sized from `templates.GameConfig.Display.MapWidth/Height` (`setup/gamesetup/helpers.go:19-20`, `gui/guiexploration/exploration_panels_registry.go:61-62`), while the map, renderers and save chunk use `ScreenInfo.DungeonWidth/Height` and tile indexing uses `CoordManager.dungeonWidth`.
- Problem: `mapWidth`, `mapHeight`, `tilePixels`, `scaleFactor` and `rightPadding` in `gameconfig.json` are dead knobs for the coordinate system. It works today only because the JSON values equal the hard-coded ones (100/80/32/3/500). Changing `mapWidth` in JSON would resize the walkable grid but not the map or the index function, which is the exact class of bug the CLAUDE.md "always use `LogicalToIndex`" warning exists to prevent.
- Direction: one screen/map config object, built once after game data is loaded. Make `CoordManager` hold (or be) that object instead of copying fields, expose the dimensions through it, and delete `ScreenInfo` as a separate variable. Have the walkable grid take its size from the same place. Remove the `init()`; construct explicitly in the boot sequence (tests already construct their own).
- Effort: M. Risk: medium, no tests in this package (about 40 `ScreenInfo.` read sites to repoint, mostly mechanical).

##### [Medium] Scale factor has two copies with different update rules
- Evidence: `CoordinateManager.scaleFactor` (set once at init, used by `LogicalToScreen` only when `MAP_SCROLLING_ENABLED`, `cordmanager.go:239-242`) versus `ScreenInfo.ScaleFactor` (set to 1 or the literal `3` at `setup/gamesetup/bootstrap.go:82-85`, read by `visual/maprender/tilerenderer.go:127,161`, `visual/rendering/rendering.go:60`, `visual/vfx/renderers.go:18,49`). `MAP_SCROLLING_ENABLED` is toggled at runtime (`input/cameracontroller.go:89`) without updating `ScreenInfo.ScaleFactor`.
- Problem: sprite size and sprite position are computed from different variables. The literal `3` also duplicates `config.DefaultScaleFactor`.
- Direction: once there is a single config object (previous finding), expose one `EffectiveScale()` that already accounts for the scrolling flag, and have renderers and conversions both call it. Move the scrolling flag into that object.
- Effort: S after the previous finding. Risk: low.

##### [Low] `Viewport` is a vestigial object
- Evidence: `cordmanager.go:92-95,148-199`. `CoordinateManager` owns exactly one `Viewport`, and `LogicalToScreen` / `ScreenToLogical` overwrite its centre on every call (`:254,280`) before using it.
- Problem: hidden mutable state for what is a pure function of `(pos, centre)`. The "reusable viewport to avoid allocation storm" comment refers to a problem a plain function would not have.
- Direction: inline the two viewport methods into `CoordinateManager` as functions taking the centre; delete `Viewport`, `NewViewport`, `SetCenter`.
- Effort: S. Risk: low.

#### Probable bugs
- **Toggling map scrolling at runtime leaves sprite scale at 3 while positions switch to 1×** in any code path that still passes a centre position (confidence: low-medium; the main draw loop switches to the non-viewport draw functions, which skip scaling, so the visible effect may be limited to VFX at `visual/vfx/renderers.go:18,49`).

#### Dead code
- `LogicalPosition.IsEqual`, `LogicalPosition.InRange` (`position.go:22,44`): no callers anywhere.
- `ScreenData.LevelWidth`, `LevelHeight`: assigned, never read.
- Stale comments: "Moved from coords.LogicalPosition to maintain compatibility" on four methods in `position.go`; "These replace the scattered CoordTransformer methods" in `cordmanager.go:115`.

#### Performance
- `ManhattanDistance` / `ChebyshevDistance` convert to `float64` and call `math.Abs` / `math.Max` for integer inputs (`position.go:28-40`). `ChebyshevDistance` is on the AI and movement hot paths (14 production call sites). An integer abs/max is a three-line change. Low priority.

#### Leave alone
- `LogicalPosition` / `PixelPosition` as distinct types.
- `LogicalToIndex` / `IndexToLogical` as the single indexing functions.
- `DrawableSection`.

---

### core/config (133 lines, tests: no)

**Role:** compile-time flags, runtime-tunable defaults, asset path resolution.
**Verdict:** Needs attention (small). Half the file is a hand-copied mirror of `templates.GameConfig`.

#### Design findings

##### [Medium] `config.Default*` is a second copy of `templates.GameConfig`, kept in sync by a 17-positional-argument setter
- Evidence: `core/config/config.go:30-76`; the single caller is `templates/gameconfig.go:26-44`. The variables are named `Default…` and commented "in case JSON loading fails", but a JSON failure is fatal at boot, and after a successful load they are the live values. `mind/evaluation/power.go:66` reads `templates.GameConfig.Combat.CritDamageBonus` directly while `config.CritDamageBonus` (the mirror) has no readers.
- Problem: every tunable exists twice, and adding one means editing the JSON schema, the validator, the setter signature, the setter body and the call site in matching positional order. Two same-typed arguments swapped in that call would compile.
- Direction: the mirror exists only because `core/common` and `core/coords` cannot import `templates`. Keep that constraint, but shrink the mechanism: one small struct in `config` (for example `config.Combat` and `config.Display`) that `templates` assigns as a whole, with field names doing the matching. Drop the `Default` prefix. Remove the three mirrors nobody reads.
- Effort: S. Risk: low.

##### [Low] Debug and profiling switches are compile-time constants left on
- Evidence: `config.go:13,16` (`DEBUG_MODE = true`, `ENABLE_BENCHMARKING = true`). `setup/gamesetup/helpers.go:27-43` therefore always starts a pprof HTTP server; `bootstrap.go:124-135` always seeds every artifact; the "release builds drop to warnings" branch of `reportCoverage` (`bootstrap.go:57-71`) cannot run without editing source.
- Problem: there is no release configuration; debug content is part of normal play.
- Direction: when a non-debug build is wanted, make these build-tag or environment driven. Until then, at least note in the file that `true` is the only tested configuration. Not urgent.
- Effort: S. Risk: low.

##### [Low] Asset paths computed at package init ignore later `SetAssetRoot` calls
- Evidence: `config.go:123-126` (`PlayerImagePath`, `AssetItemsDir` are package-level `var`s evaluated before any test can call `SetAssetRoot`, `:118-120`).
- Direction: make `PlayerImagePath` a function, or resolve it at the call sites via `AssetPath`.
- Effort: S. Risk: low.

#### Dead code
- `CritDamageBonus`, `DefaultMapWidth`, `DefaultMapHeight`: written by the setter, never read.
- `AssetItemsDir`: no readers.

#### Leave alone
- `AssetPath` / asset-root resolution order.
- The log-export flags and profiling constants.

---

### game_main (422 lines + 14-line root file, tests: no)

**Role:** Ebiten `Game` struct, frame loop, and the three start paths (overworld, roguelike, load).
**Verdict:** Needs attention. The frame loop is fine; the three setup paths repeat each other and the load path has a failure mode that leaves a broken world.

#### Design findings

##### [Medium] The three setup paths duplicate their tails
- Evidence: `game_main/setup.go:125-144` (`SetupRoguelikeMode`) and `:204-231` (`SetupRoguelikeFromSave`) repeat the same sequence: `setupUICore`, select first commander from roster, `RegisterRoguelikeTacticalModes`, create and inject `RaidRunner`, `SetupInputCoordinator`, `EnterTactical("exploration")`. `SetupOverworldMode` (`:88-107`) repeats the outer shape. `gamesetup.NewGameBootstrap()` is constructed four times (`:30,89,116,154`) although the type has no fields.
- Problem: a change to roguelike wiring has to be made twice, and the load path already differs in small ways (returns errors vs `log.Fatalf`).
- Direction: extract one `finishRoguelikeSetup(g) error` used by both roguelike paths. Turn `GameBootstrap`'s methods into plain functions in `gamesetup` (or build it once and keep it on `Game`).
- Effort: S. Risk: low, no tests.

##### [Medium] Mode identity is implicit
- Evidence: there is no "current game mode" value. Save and load buttons appear when the exploration panel detects `squad_editor` registered in the tactical manager (`gui/guiexploration/exploration_panels_registry.go:181,194`); `setupUICore` wires `SaveRoguelikeGame` for both modes (`setup.go:61-67`); the load path always rebuilds a roguelike session.
- Problem: whether saving is possible is inferred from which UI modes happen to be registered, and from registration order (`setup/gamesetup/moderegistry.go:65-67`).
- Direction: store the chosen mode on `Game` (or the UI context) once at start and let the save/load wiring and the exploration panel read it. Only wire the save callbacks in roguelike mode.
- Effort: S. Risk: low.

##### [Low] Stale comments and small leaks of other layers
- Evidence: `main.go:57-59` (`HandleInput` comment describes combat, status effects and entity cleanup; the function forwards camera input and clears a write-only flag); `main.go:173-179` reaches into `campaign/overworld/core` to finalise the overworld recorder; `setup.go:33-38` loads raid config here while all other game data loads in `gamesetup`.
- Direction: fix with the boot-sequence finding in `setup/gamesetup`; move the exit-time recorder flush behind a `gamesetup.Shutdown()`.
- Effort: S. Risk: low.

#### Probable bugs
- **A load that fails after validation leaves a half-built world** (confidence: high on the logic, low on likelihood). `SetupRoguelikeFromSave` (`setup.go:166-235`) calls `ResetWorld` and then `LoadGame`, `InitUnitTemplatesFromJSON`, `RestoreRenderables`, any of which can return an error after entities exist. On the in-session path, `main.go:92-97` logs "continuing current game", but the old world is already gone and the old UI coordinator still points at the reset manager. On the start-menu path, `main.go:83-86` calls `SetupRoguelikeMode` on top of the partially loaded entities with no reset, which can leave two player entities.
  Direction: on any error after `ResetWorld`, reset again and start a fresh game (or return to the start menu); do not claim the old game continues.

#### Dead code
- `test_init.go` at the repository root: a second `package main` with its own `main()` that prints a component pointer before and after init. It is a scratch file that `go build ./...` compiles into a stray binary. Delete.

#### Leave alone
- `Update` / `Draw` / `Layout` structure, the `pendingLoad` deferral, and validating the save before tearing the world down.

---

### setup/gamesetup (1,189 lines, tests: no)

**Role:** boot orchestration: load data, initialise ECS, create world, player, commanders, starting squads, factions; register UI modes; post-load restoration.
**Verdict:** Needs attention. It is the right place for this glue, but it contains the slice's largest copy-paste block and an incoherent data-loading sequence.

#### Design findings

##### [Medium] `initial_squads.go` is five copies of one function with the content hard-coded
- Evidence: `setup/gamesetup/initial_squads.go:149-349`. `createBalancedSquad`, `createRangedSquad`, `createMagicSquad`, `createMixedSquad`, `createCavalrySquad` share the same body: pick a unit pool, pick a count, walk a literal grid-position table, mark a random leader, set `Attributes.Leadership = 20`, call `CreateSquadFromTemplate`. They differ only in pool filter (`AttackRange >= 3`, `AttackType == Magic`, `MovementSpeed >= 6`), count range, position table and formation. The valid type names are listed again in `templates/initialsetup.go:48-56` with "keep in sync" comments on both sides. The same leader rule appears a third time in `tools/combat_analysis/combat_simulator/squad_factory.go:130`.
- Problem: about 200 lines where roughly 40 carry information. Squad archetypes are content, yet they are code, while enemy squads elsewhere are data-driven.
- Direction: one `createSquad(spec)` function driven by a small table (`pool filter`, `min/max count`, `positions`, `formation`). Keep the table in Go for now; moving it to JSON is optional and only worth it if designers need it. Derive the validator's allowed names from the table's keys so the list exists once. Hoist `20` into a named constant shared with the simulator.
- Effort: S–M. Risk: low, no tests.

##### [Medium] Data loading is split across three places with five failure policies
- Evidence:
  - `bootstrap.go:36-51` (`LoadGameData`): `templates.ReadGameData` → fatal; perk definitions → `reportCoverage`; perk and artifact balance → fatal; `combatmath.LoadCombatBalanceConfig()` → prints a warning and continues with whatever is in the struct (`tactical/combat/combatmath`); `core.ValidateNodeRegistry()` → warnings only.
  - `game_main/setup.go:33-38`: raid config and archetypes → warning, "using defaults".
  - Unit templates are built lazily in `bootstrap.go:106`, again in `game_main/setup.go:192` for the load path (with a comment explaining why), and a third time in `tactical/squads/squadcore/squadmanager.go:65`.
  - `log.Fatalf` also appears in `bootstrap.go:107,112,116,154`, `playerinit.go:29`, `moderegistry.go:36,60,87` and `game_main/setup.go:105,143`, although CLAUDE.md describes `LoadGameData` as the single boot-time fatal point.
- Problem: whether a broken data file stops the game, warns, or is silently ignored depends on which loader it went through. The load-from-save path has to remember which lazy steps `CreatePlayer` would have done.
- Direction: one `gamesetup.LoadGameData() error` that loads everything static (templates, perks, artifacts, combat balance, raid config, archetypes, unit templates) and returns the first error; one fatal in `main`. Make the world-creation functions return errors instead of calling `log.Fatalf`, so an in-session failure can be handled.
- Effort: M. Risk: low-medium (changes failure behaviour for combat balance and raid config from "warn" to "stop"; flag that separately when doing it).

##### [Medium] Post-load restoration repeats work done elsewhere
- Evidence:
  - `chunks/commander_chunk.go:112-139` loads the commander image and attaches `Renderable`; `savehelpers.go:66-76` then loads the same file again once per commander and overwrites it.
  - `config.PlayerImagePath` is loaded from disk at six sites (`playerinit.go:27`, `initial_commanders.go:26`, `savehelpers.go:55,68`, `chunks/commander_chunk.go:116`, `gui/guioverworld/overworld_action_handler.go:336`).
  - `savehelpers.go:23-29` (`ConfigureMapChunk`) must be called before both save and load to inject the map pointer into a registry singleton via a type assertion.
- Problem: image reconstruction is split between chunks and `RestoreRenderables` with no rule for which owns what; forgetting `ConfigureMapChunk` yields a runtime "GameMap not set" error.
- Direction: chunks restore data only; `RestoreRenderables` is the single place that attaches images (drop the image code from `CommanderChunk.Load`). Load the commander/player sprite once. Pass the map to the save system explicitly (see `setup/savesystem`).
- Effort: S. Risk: low.

##### [Low] Production code imports `testing/bootstrap`
- Evidence: `bootstrap.go:20,132-133`.
- Direction: move `SeedAllArtifacts` / `EquipPlayerActivatedArtifacts` into `gamesetup` (they are debug start-up content, not test fixtures) and delete the `testing/bootstrap` package.
- Effort: S. Risk: low.

##### [Low] Two mode lists that must be kept in step, plus an ordering dependency
- Evidence: `moderegistry.go:24-39` vs `:69-92` share five constructors; the comment at `:65-67` says squad modes must be registered first so `ExplorationMode.Initialize()` detects them.
- Direction: build the roguelike list as "squad modes + core tactical modes + raid", and replace the order-based detection with the explicit mode value suggested under `game_main`.
- Effort: S. Risk: low.

##### [Low] `mapgenconfig.go` has a no-op builder and inconsistent defaulting
- Evidence: `buildGarrisonRaidGenerator` (`mapgenconfig.go:105-113`) returns `nil` on both branches. `buildRoomsCorridorsGenerator` starts from `DefaultRoomsCorridorsConfig()` (`:44`), while the cavern and overworld builders construct the config from scratch (`:57-73,83-101`), so a field missing from JSON becomes zero there instead of the code default.
- Direction: delete the garrison builder and its `case`; start all builders from the generator's default config. The field-by-field copy itself is the price of keeping `world/` free of `templates` and is acceptable.
- Effort: S. Risk: low.

##### [Low] Faction type names exist in three places
- Evidence: `initial_factions.go:42-54` (switch), `templates/initialsetup.go:61-66` (validator set), and the `FactionType` enum in `campaign/overworld/core`.
- Direction: one `core.ParseFactionType(string)` next to the enum; the validator and this file both use it.
- Effort: S. Risk: low.

#### Dead code
- `buildGarrisonRaidGenerator` (always nil).
- Unused parameters: `gm` in `SetupDebugContent` (`bootstrap.go:124`), `pd` in `CreateInitialFactions` (`initial_factions.go:17`).

#### Leave alone
- `InitializeECS` / the subsystem self-registration call.
- `CreateInitialCommanders` and the ordering comments in `CreateSquadsForCommander` (roster before spells).
- `newCombatServiceFactory` keeping `mind/ai` out of the GUI packages.
- The `worldgen.ConfigOverride` hook.

---

### setup/savesystem (371 lines, tests: yes)

**Role:** save envelope, chunk registry, two-phase load with entity-ID remapping.
**Verdict:** Healthy. The framework is small and well thought out (atomic write, backup, validation before world reset, per-chunk versions). A few trims.

#### Design findings

##### [Medium] Unmapped IDs silently become 0, and the strict path is unused
- Evidence: `idmap.go:36-51` (`Remap`, `RemapSlice` return 0 for unknown IDs); `RemapStrict` (`:56-65`) is called only from tests. The `Validatable` hook (`savesystem.go:53-57,272-282`) has no implementers.
- Problem: this default is what lets the coverage bugs in `chunks` pass unnoticed: a roster pointing at unsaved units loads "successfully" with zeros in it.
- Direction: make the strict behaviour the default for references that must resolve (roster → units, roster → squads, raid → squads) and return the error from `LoadGame`. Keep a lenient call for genuinely optional links. Either give `Validatable` one real implementer (a post-load roster consistency check is the obvious one) or remove the hook.
- Effort: S. Risk: low (loads that are currently lossy would start failing loudly, which is the point).

##### [Low] Chunks that need non-ECS state get it through a side channel
- Evidence: `SaveChunk` methods receive only `*EntityManager` (`savesystem.go:16-34`); `MapChunk` therefore has a public `GameMap` field set by `gamesetup.ConfigureMapChunk` via `GetChunk("map")` and a type assertion; `EntityIDMap.LoadContext map[string]interface{}` (`idmap.go:18`) carries data between phases for two chunks.
- Direction: pass a small context struct (`EM`, `GameMap`) to `SaveGame` / `LoadGame` and on to the chunks. That removes `ConfigureMapChunk`, `GetChunk`, and the stateful registry singleton. `LoadContext` can stay; it is contained.
- Effort: S. Risk: low.

#### Dead code
- `DeleteSaveFile`: no callers. `GetRegisteredChunks`, `RemapStrict`: tests only. `GetChunk`: one caller, removable with the context change.

#### Leave alone
- Envelope format, checksum, atomic write and backup, `ValidateSaveFile` pre-flight, per-chunk version check, init-time chunk registration.

---

### setup/savesystem/chunks (1,724 lines, tests: yes)

**Role:** one chunk per domain (player, commanders, squads, gear, progression, raid, map) translating ECS state to and from JSON.
**Verdict:** Needs attention. The structure is reasonable, but coverage is maintained by hand in three parallel places and has already drifted in four spots.

#### Design findings

##### [High] Nothing ties the saved shape to the component structs or to the entity factories
- Evidence:
  - Each component has a hand-written `saved*` mirror plus a field-by-field copy in `Save` and another in `Load` (for example `raid_chunk.go:35-87` mirrors six raid components; `squad_chunk.go:34-128` mirrors eleven squad/unit components).
  - Each `Load` also rebuilds the entity's component set by hand, duplicating the factory: `player_chunk.go:140-187` vs `gamesetup/playerinit.go:44-68`; `commander_chunk.go:127-158` vs `commander.CreateCommander` (the chunk says "Mirror CreateCommander"); `squad_chunk.go:313-381` vs `squadcore.CreateUnitEntity` + `ApplyUnitComponents`.
  - The four bugs listed below are all instances of this: a field added to a component, a component attached by a different system, or an entity kind the chunk never iterates.
- Problem: adding a field or a component to anything persistent requires remembering three edits in a different package, and forgetting produces a save that loads without error and is quietly wrong.
- Direction (in order of value):
  1. Fix the four gaps below.
  2. Where a component is plain data with no entity IDs (`AlertData`, `GarrisonSquadData`, `ExperienceData`, `StatGrowthData`, `CoverData`, `ManaData`, `ProgressionData`, `ResourceStockpile`, `Attributes`), marshal the component struct itself instead of a mirror. Keep explicit `saved*` structs only where IDs need remapping or fields are not serialisable. This removes a large share of the file without adding any abstraction. The mirrors currently buy no format stability, because a version mismatch is rejected rather than migrated (`savesystem.go:218-229`).
  3. Have `Load` call the same factory the game uses to build the entity skeleton, then overwrite the data, so a component added to the factory is present after load.
- Effort: M–L. Risk: medium (save format changes; bump chunk versions).

##### [Low] Small inconsistencies
- `savedAttributes.MaxHealth` is written (`shared_types.go:45`) and never read on load; it is derived.
- `MapChunk.Save` takes dimensions from `coords.ScreenInfo` (`map_chunk.go:70-71`) while `Load` validates against `coords.CoordManager` (`:131-138`): the two-singleton problem again.
- `MapChunk.Load` and `CommanderChunk.Load` load images; the other chunks leave that to `gamesetup.RestoreRenderables`.
- `CommanderChunk`'s doc comment (`commander_chunk.go:22-23`) lists mana and spellbook, which are saved by `SquadChunk`.
- `raid_chunk.go:209,221-222,237,274-275` hand the unmarshalled slices straight to components while `Save` copies them; harmless, but `copyEntityIDs` is used inconsistently.

#### Probable bugs
All four are "state lost on save/load". The repo's round-trip test file has no case for any of them.

- **Roster units that are not in a squad are not saved** (confidence: high). `SquadChunk.Save` only walks squads and their members (`squad_chunk.go:135-175`). Unassigned units have no `SquadMemberComponent` (`squadcore/squadcreation.go:245`; `CreateInitialRosterUnits` at `gamesetup/initial_squads.go:108-136` creates five of them per `initialsetup.json`). `PlayerChunk` saves their IDs in the roster (`player_chunk.go:99-115`), and on load `Remap` turns each into 0 (`:207-212`). After a load the roster still counts them (`TotalOwned`) but its `UnitEntities` hold zeros.
- **Equipped perks are not saved** (confidence: high). `perks.PerkSlotComponent` is attached to squads by `EquipPerk` (`tactical/powers/perks/system.go:28-33`); no chunk references it. I did not confirm how the UI then presents a squad with no slot component.
- **Garrison squad data is detached from its squad on load** (confidence: high). In play, `GarrisonSquadComponent` sits on the squad entity itself (`campaign/raid/garrison.go:181`). `RaidChunk.Load` creates a *new* bare entity for it (`raid_chunk.go:257-267`) and registers the squad's old ID to that entity; `SquadChunk.Load` runs later (chunks register in file-name order) and re-registers the same old ID to the real squad. Result: the real squad has no `GarrisonSquadData`, and an orphan entity carries it. After loading mid-raid, reserve activation skips every reserve (`campaign/raid/alert.go:46-48`), cleared rooms do not mark squads destroyed (`resolvers.go:44-49`), and `DisposeRaidEntities` (`garrison.go:53-62`) disposes the orphans while the real garrison squads leak.
- **`RaidStateData.PlayerEntityID` is not saved** (confidence: high that it is lost; medium on impact). The field exists on the component and feeds the reward target (`campaign/raid/rewards.go:21`), but `savedRaidState` (`raid_chunk.go:35-42`) omits it, so it is 0 after load. I did not trace what `GrantTarget` does with a zero player ID.

#### Performance
- The map is saved as one JSON object per tile with explicit `x`/`y` (`map_chunk.go:38-46,79-93`), then indented: 8,000 objects for a 100×80 map. It works; a compact per-row or parallel-array encoding would cut file size and load time if saves ever feel slow. Not a priority.

#### Leave alone
- One chunk per domain, registered via `init()`.
- The two-phase load and the use of `LoadContext` for gear and progression.
- `shared_types.go` helpers.
- Keeping all chunks in this one package rather than spreading them into the domain packages: it keeps `savesystem` imports out of gameplay code.

---

## 11. Templates and world generation

Scope: every non-test `.go` file in the five packages was read in full, plus the gamedata JSON under `resources/assets/gamedata/`. Callers outside the slice were read only to establish who depends on what. "Unused" below means a whole-repo grep (including `tools/` and tests) found no reference other than the ones listed.

### Layer overview

**How the packages relate**

- `templates` is the static-data layer: 15 loaders run by `ReadGameData()` fill package-level registries read by 28 packages. It imports `core/common`, `core/config`, `core/coords`, `campaign/overworld/ids`, `world/garrisongen/roomtypes`, `bytearena/ecs` and `ebitenutil`.
- `world/worldmapcore` owns `GameMap`, `Tile`, `Rect`, the `MapGenerator` interface and tile-image loading. It imports `visual/graphics` and ebiten.
- `world/worldgen` holds three generators (`rooms_corridors`, `cavern`, `overworld`), the generator registry and shared carving helpers.
- `world/garrisongen` holds two unrelated things: the abstract floor DAG that `campaign/raid` uses as its floor graph, and a tile-map generator (`garrison_raid`) with its terrain injectors.
- `world/garrisongen/roomtypes` is a 38-line leaf with the room-type strings, so `templates` can validate them.

**What is actually reachable in the game**

- Overworld mode generates `"overworld"` (`game_main/setup.go:90`); roguelike mode generates `"cavern"` (`game_main/setup.go:117`).
- `rooms_corridors` and `garrison_raid` are reachable only from the exploration debug menu (`gui/guiexploration/exploration_panels_registry.go:94,102`) and as the registry fallback.
- `campaign/raid` never touches `GameMap`. It calls `garrisongen.BuildGarrisonDAG` directly (`campaign/raid/garrison.go:74`) and uses only the graph.

**Themes that cut across the slice**

1. The same map-generation parameters exist three times: Go `Default*Config()` literals, `mapgenconfig.json`, and a hand-written field-by-field copy in `setup/gamesetup/mapgenconfig.go`.
2. Generators produce presentation data (images, pixel coordinates) while generating terrain, and a second copy of the image-selection logic exists for the save-load path.
3. Walkability is stored in three places that are updated by different code.
4. Gamedata is loaded through three different conventions, only one of which goes through `templates`.
5. Several vocabularies are kept in sync by comment only (squad types, faction types, POI types).

**Layout recommendations**

- Split `world/garrisongen` along its real seam. The DAG (`dag.go`) is raid gameplay data and belongs next to its only consumer, or stays as a small `garrisongen` package. The tile generator (`generator.go`, `terrain.go`, about 985 lines) is debug-only today; decide whether raids will ever be fought on generated floors. If yes, wire it to the raid's DAG. If no, delete it. See the garrisongen section.
- Fold `roomtypes` back into wherever the DAG ends up once `templates` no longer needs it (it only exists to break an import cycle for two validator lines).
- Move the entity factory out of `templates` into `tactical/squads/squadcore`, which is its only game caller. `templates` then becomes a pure data leaf with no ECS or ebiten import.
- Do not split `templates` into more packages. One flat data package is the right size; the problem is consistency, not granularity.

---

### templates (2,383 lines, tests: yes, covering difficulty and name generation only)

**Role:** Loads, validates and stores static JSON game data in package-level registries. **Verdict:** Needs attention. The `Loader[T]` core is good, but the package is not the single data layer it claims to be, validation coverage is uneven, and it still carries entity-creation and UI-formatting code.

#### Design findings

##### [High] Three conventions for loading gamedata; `templates` covers only 16 of 24 files
- Evidence:
  - Through `templates`: the 16 paths in `templates/paths.go:5-22`.
  - Domain packages with their own `os.ReadFile` + `json.Unmarshal`: `tactical/powers/perks/registry.go:130` (perkdata), `tactical/powers/perks/balanceconfig.go:126`, `tactical/powers/artifacts/balanceconfig.go:32`, `tactical/combat/combatmath/balanceconfig.go:32`, `campaign/raid/config.go:70`, `campaign/raid/archetypes.go:32`.
  - Boot wiring is split: `setup/gamesetup/bootstrap.go:36-51` and `game_main/setup.go:33-38`.
  - Failure policy differs per loader: `log.Fatalf` (templates, perk and artifact balance), debug-fatal/release-warn (`reportCoverage`), no error return at all (`combatmath.LoadCombatBalanceConfig()`, bootstrap.go:49), and print-a-warning-and-continue for both raid files (setup.go:33-38).
  - Inside `templates` there are also two shapes: "DTO here, runtime registry elsewhere" (nodes and encounters are copied into `campaign/overworld/core/node_registry.go:79-148`; monsters into `tactical/squads/unitdefs/templates.go:106-116`) versus "runtime type and registry live in templates" (spells, artifacts, unit spells, difficulty).
- Problem: `doc.go:1` and CLAUDE.md both say this package loads all static JSON. It does not, so there is no single place to see what is loaded, in what order, or what happens on failure. A broken `raidconfig.json` silently runs on code defaults while a broken `aiconfig.json` aborts.
- Direction: Keep domain-owned types where they are (perks owning `PerkDefinition` is fine), but make every loader use `templates.Loader[T]` and register in one ordered list with one failure policy. Concretely: export a small registration hook or simply call the six outside loaders from `ReadGameData`'s caller through `Loader[T]`, and move the two raid calls out of `game_main/setup.go` into `LoadGameData`. Update `doc.go` and CLAUDE.md to state the real rule: "DTOs and loading mechanics here; domain packages may own their runtime types."
- Effort: M. Risk: low.

##### [High] Map-generation config is defined three times
- Evidence:
  - Go defaults: `world/worldgen/gen_rooms_corridors.go:18-25`, `gen_cavern.go:58-76`, `gen_overworld.go:44-69`, `world/garrisongen/dag.go:43-84`.
  - JSON: `resources/assets/gamedata/mapgenconfig.json` (values are identical to the Go defaults today).
  - DTOs: `templates/schema_mapgen.go:17-85`.
  - Field-by-field copy: `setup/gamesetup/mapgenconfig.go:38-144`, installed through the `worldgen.ConfigOverride` function variable (`world/worldgen/registry.go:14`) and two package-level setters (`world/garrisongen/dag.go:87-94`).
  - `ReadMapGenConfig` is the only loader that bypasses `Loader[T]` (`templates/readdata.go:199-218`) so that a missing file can mean "use Go defaults".
- Problem: Adding one tunable means editing four files and remembering the copy step. The Go defaults and the JSON will drift the first time someone tunes one and not the other. The override hook, the nil-pointer sentinel and the registry's "default-configured" generator instances exist only to support the optional file.
- Direction: Make `mapgenconfig.json` required and the only source. Give the worldgen config structs JSON tags and unmarshal straight into them (either `templates` holds `json.RawMessage` per generator, or worldgen exposes a `Configure(cfg)` per generator). Then delete the `JSON*Config` mapgen DTOs, `setup/gamesetup/mapgenconfig.go`, `ConfigOverride`, the two `SetGarrison*` setters and the hardcoded tables in `dag.go`. Keep one small in-code default only for tests.
- Effort: M. Risk: low (smoke tests exist for worldgen).

##### [High] Validation is uneven, and several "keep in sync" lists live in the wrong place
- Evidence:
  - No validator at all: monsters (`readdata.go:32-35`), spells (`spelldefinitions.go:97-101`), artifacts (`artifactdefinitions.go:55-59`). Monster `role` and `attackType` are only checked later, in `unitdefs.CreateUnitTemplates`, during `CreatePlayer`.
  - Unit spells: an unknown spell ID is a log line, not an error (`unitspelldefinitions.go:35-38`), although `registry.go:13` describes this as validation. Unit type names are not checked against monsters.
  - `commander.startingPerks` and `startingSpells` are only checked for non-empty (`gameconfig.go:70-75`).
  - `validSquadTypeIDs` (`initialsetup.go:50-56`) must match `squadCreators` in `setup/gamesetup/initial_squads.go:21-27`. `validFactionTypeIDs` (`initialsetup.go:61-66`) must match `factionTypeFromString` in `setup/gamesetup/initial_factions.go:42-54`. Both are synced by comment. Neither includes "beasts", although `core.FactionBeasts` exists (`campaign/overworld/core/types.go:61`) and `encounterdata.json` and `nodeDefinitions.json` define a Beasts faction.
  - `requiredStrategies` (`validation.go:461`), `requiredRoles` (`validate_shared.go:9`), `"Balanced"` (`validation.go:234`) and composition sizes 1-4 (`validation.go:267`) repeat enums owned by other packages.
  - Stat names in spell and artifact modifiers (`"movementspeed"`, `"attackrange"`, ...) are free strings, checked nowhere in this package.
- Problem: The data is consistent today (I cross-checked spells, unit spells, starting perks/spells, monster roles, influence multipliers and raid archetype units with a script and found no dangling reference), but nothing keeps it that way. A typo in a spell's `targetType` or a monster's `role` surfaces far from the data file, or not at all.
- Direction: Add validators for monsters, spells, artifacts and unit spells (unknown spell ID becomes an error). For lists owned by another package, invert the dependency instead of duplicating: let the owning package register its valid keys (for example `templates.RegisterValidSquadTypes(keys)` from `gamesetup`), or run that cross-check in the owner after load, the way `core.ValidateNodeRegistry()` already does. Do one cross-file pass at the end of `ReadGameData` for references between files.
- Effort: M. Risk: low.

##### [Medium] Entity factory and UI formatting live in the data package
- Evidence:
  - `templates/entity_factory.go:34-77`: `CreateCreatureEntity` and `CreateUnit` create ECS entities, decode PNGs with `ebitenutil`, and register positions. The file's own note (lines 17-22) records that it is here only because of an import cycle with `gamesetup`.
  - The two functions are near-duplicates with different conventions: one takes `*common.EntityManager` and loads an image, the other takes `common.EntityManager` by value and does not.
  - The only game callers are in `squadcore` (`tactical/squads/squadcore/squadcreation.go:360`, `units.go:33`). After `CreateCreatureEntity` returns, the caller immediately re-adds `NameComponent` and `PositionComponent` (`squadcreation.go:365-371`), and after `CreateUnit` the caller loads the image itself (`units.go:47`).
  - `templates/format.go:12-29` (`FormatSpellDetail`) is display formatting with one caller, `gui/guiprogression/progression_controller.go:195`.
  - `templates/namegen.go:11` (`GenerateName`) is runtime logic using the shared RNG, called only from `squadcore`.
- Problem: The data layer depends on ECS, coords and ebiten for the sake of two functions whose only caller overrides half of what they do. The cycle the note describes does not apply if the factory moves to the caller rather than to `gamesetup`.
- Direction: Move unit entity creation into `squadcore` as one function (image optional), and delete both `templates` versions; `tools/combat_analysis/combat_simulator/squad_factory.go:78` already imports `squadcore`. Move `FormatSpellDetail` to `gui/guiprogression` (or `gui/guispells`). `GenerateName` can stay (it is a pure function of loaded data) or move with the factory.
- Effort: S-M. Risk: low.

##### [Medium] Map dimensions and other values have more than one home
- Evidence:
  - Map size: `gameconfig.json display.mapWidth/mapHeight` is pushed into `core/config` by a 17-positional-argument call (`templates/gameconfig.go:26-44`, `core/config/config.go:54-76`), read again directly as `templates.GameConfig.Display.MapWidth` (`setup/gamesetup/helpers.go:19`, `gui/guiexploration/exploration_panels_registry.go:61`), and held a third time in `coords.ScreenInfo.DungeonWidth`.
  - `overworldconfig.json mapDimensions` is parsed and validated (`schema_overworld.go:55-58`, `validation.go:456-459`) and read by nothing else.
  - `gameconfig.json player.limits.maxSquads` has no reader; only `commander.maxSquads` is used (`setup/gamesetup/initial_commanders.go:46`, `gui/guioverworld/overworld_action_handler.go:351`).
  - `initialsetup.json factions.fallbackPositions` contains `{x:120, y:65}` on a 100x80 map; `validateInitialSetup` does not bounds-check positions.
- Problem: Three stores for map size, joined by a positional call where swapping two ints compiles. Dead keys in JSON look tunable but do nothing.
- Direction: Replace `SetConfigFromJSON`'s positional list with a struct (or have `core/config` expose a struct that `templates` fills). Pick one reader for map size. Delete `mapDimensions` and `player.limits.maxSquads` from both JSON and DTOs. Bounds-check fallback positions against the display size in the validator.
- Effort: S. Risk: low.

##### [Low] Difficulty manager is built for a feature that does not exist
- Evidence: `templates/difficulty.go:69-86`. `SetDifficulty` has no non-test caller; `atomic.Pointer` guards a value written once at boot in a single-threaded game.
- Problem: Minor, but it is concurrency machinery with nothing concurrent, and the active difficulty is process state held in the data package (it is not part of any save chunk I saw).
- Direction: Replace the atomic pointer with a plain pointer. If a difficulty selector is added later, decide then where the choice is persisted. The four-knob derivation itself is sound and all 15 derived fields are consumed.
- Effort: S. Risk: low.

##### [Low] Stale names and comments
- Evidence: `EncounterDataWithNew` (`readdata.go:21`); "new system", "new format" (`registry.go:28`, `validation.go:86`); "all 56 downstream consumers" (`registry.go:17`); spell loader comment says unparseable data is non-fatal (`spelldefinitions.go:104`) but `Loader.Load` returns the parse error; `JSONMonster.AttackType` comment lists four values (`schema_monster.go:54`) while the data also uses `"Heal"`; `JSONSquadType.Name/Description` are documented as "for future filtering" and unread.
- Direction: Rename and correct in passing when the files are touched for the findings above.
- Effort: S. Risk: low.

#### Mechanics notes
- Two unrelated "squad type" vocabularies share the name: starting-squad recipes (`balanced, ranged, magic, mixed, cavalry` in `initialsetup.json`) and encounter composition preferences (`melee, ranged, magic` in `encounterdata.json squadTypes`). They never meet in code, but the shared term invites confusion; consider "squadRecipe" for the first.
- Two unrelated "difficulty" scales also share labels: the global preset (Easy/Medium/Hard) and encounter levels 1-5 (Easy/Moderate/Fair/Hard/Boss). The global preset multiplies the encounter level's numbers, so "Hard on Hard" is meaningful but hard to read in logs.
- `RaidIntensityScale` exists both as a difficulty multiplier and as `factionScoringControl.raidIntensityScale`; `campaign/overworld/core/config.go:89` and `campaign/overworld/faction/system.go:294` each use a different one.
- The Beasts faction has a threat node and three encounters but cannot be a starting faction (see validation finding).

#### Probable bugs
- `initialsetup.json` fallback position 4 is outside the map (x=120, map width 100). It is used when the overworld generator returns fewer than four start positions (`setup/gamesetup/initial_factions.go:27-32`), which happens when a corner sector has no walkable grassland or forest (`world/worldgen/gen_overworld.go:362`). Confidence: medium that it can trigger; high that the value is wrong.
- Registries append rather than replace: `MonsterTemplates = append(...)` (`readdata.go:91`), `SpellRegistry` and `ArtifactRegistry` are never cleared. A second `ReadGameData()` in one process (tests) duplicates monsters. Confidence: high; impact limited to tests today.

#### Dead code
- `consumabledata.json` and `creaturemodifiers.json`: no loader or reference anywhere in the repo.
- `JSONOverworldConfig.MapDimensions` and `MapDimensionsConfig` (validated only).
- `PlayerLimitsConfig.MaxSquads`.
- `JSONCavernConfig.NumChambers` and `JSONOverworldGenConfig.FactionMinSpacing`: copied through to the generators, which never read them (see worldgen).
- `DifficultyManager.SetDifficulty` and `GetCurrentDifficulty` (tests only).
- `JSONSquadType.Name`, `JSONSquadType.Description`.

#### Performance
- `CreateCreatureEntity` decodes the unit PNG from disk for every unit created (`entity_factory.go:36`), and `squadcore/units.go:47` does the same on the other path. There is no image cache anywhere in the repo (20 `NewImageFromFile` call sites). An encounter of 10 squads with up to 8 units is up to 80 decodes and 80 separate GPU images for about 30 distinct sprites. A path-keyed cache at the single factory fixes this.

#### Leave alone
- `Loader[T]` (`loader.go`): small, does one job, right level of abstraction.
- The ordered loader table in `ReadGameData` with its documented phases.
- `validate_shared.go` helpers and the "required flag in JSON" approach for nodes (`validation.go:21-29`).
- The schema file grouping (`schema_*.go`). Do not split further.
- The validators-return-error rule; the package is in fact panic-free.
- The four-knob difficulty derivation.

---

### world/worldmapcore (651 lines, tests: yes)

**Role:** Map data model (`GameMap`, `Tile`, `Rect`), the generator interface, and tile-image loading. **Verdict:** Needs attention. The accessor-based `GameMap` is a good start, but tiles mix simulation and presentation, walkability has several owners, and a vestigial stairs feature still drives the API.

#### Design findings

##### [High] Walkability has three sources of truth
- Evidence:
  - `Tile.Blocked` (`dungeontile.go:24`), which `input/cameracontroller.go:125-126` rewrites as the player moves (the player's own tile is marked blocked, the tile left behind is unblocked).
  - `GameMap.validPositions` (`dungeongen.go:55`), a list captured at generation and never updated; also written to the save file (`setup/savesystem/chunks/map_chunk.go:103`).
  - `campaign/overworld/core.WalkableGrid` (`campaign/overworld/core/walkability.go:9`), a package-level `[]bool` rebuilt from `validPositions` (`setup/gamesetup/helpers.go:17-23`), sized from `templates.GameConfig.Display` but indexed through `CoordManager`.
  - Readers disagree: commander movement, factions, threats and node placement read `WalkableGrid` (`tactical/commander/movement.go:77` and others); the camera controller reads `Tile.Blocked`.
- Problem: Entity occupancy is being written into terrain data, a derived list is persisted, and the grid every gameplay system consults is a global in an overworld package even though roguelike mode needs it too (`game_main/setup.go:123`). Any future terrain change has to update three structures.
- Direction: Make the map the single owner of terrain walkability: `GameMap.IsWalkable(pos)` backed by `Tile.Blocked`. Stop the camera controller from writing `Blocked` (occupancy is the position system's job). Build `WalkableGrid` from tiles, or replace it with the map query, and move it out of `campaign/overworld/core`. Drop `validPositions` from `GameMap` and from the save chunk; keep it only inside `GenerationResult` if generators want it during generation.
- Effort: M. Risk: medium (touches input, save format and four campaign packages).

##### [High] `Tile` and `GameMap` carry presentation state; image selection is implemented twice
- Evidence:
  - `Tile` holds `Image *ebiten.Image`, `PixelX`, `PixelY` and an overlay `ColorMatrix` (`dungeontile.go:20-31`). `PixelX/Y` are `TileCords * TileSize`; the renderer divides them back (`visual/maprender/tilerenderer.go:153-154`).
  - `GameMap.tileColorsDirty`, `ApplyColorMatrix*`, `MarkTileColorsDirty`, `ConsumeTileColorsDirty` (`dungeongen.go:139-157`, `gamemap_accessors.go:84-96`) are UI overlay plumbing used by spell targeting, threat visualization and the camera controller. This is why `worldmapcore` imports `visual/graphics`.
  - Image choice during generation: `worldgen.SelectRandomImage` / `GetBiomeImages` at about 15 sites, with `images TileImageSet` threaded through the `MapGenerator` interface and every carve and terrain helper.
  - Image choice on load: `SelectTileImage`, `selectBiomeFloorImage`, `selectBiomeWallImage` (`GameMapUtil.go:132-167`), a separate implementation of the same rule.
- Problem: Generators cannot run without an image set, about 70 function signatures in `worldgen` and `garrisongen` carry `images` only to assign a wall or floor picture, and new-game and load paths can diverge (they already do for the stairs tile, see bugs). The world layer depends on the visual layer.
- Direction: Have generators emit terrain only (`TileType`, `Blocked`, `Biome`, `POIType`). Assign images in one pass after generation using the existing `SelectTileImage`, called from `NewGameMap` and from the save loader. Remove `images` from `MapGenerator.Generate` and all helpers. Drop `PixelX/PixelY` (compute in the renderer). As a second, separable step, move the overlay color matrix into `visual/maprender` as a per-index overlay so `worldmapcore` no longer imports `visual/graphics`.
- Effort: M for the image pass, M for the overlay move. Risk: low to medium (visual only; smoke tests cover generation).

##### [Medium] Stairs and `Rooms` are leftovers that still shape the API
- Evidence:
  - `PlaceStairs` runs for every map (`dungeongen.go:73,111-136`). Nothing in the game reads `STAIRS_DOWN`; the only references outside this package's tests are the writer and `SelectTileImage`.
  - `LoadTileImages` panics if the stairs image is missing (`GameMapUtil.go:72-75`).
  - `GenerationResult.Rooms` means three different things: real rooms (`rooms_corridors`, `garrison_raid`), chamber bounding boxes (`gen_cavern.go:158-163`), and one 1x1 rect per POI "for backward compat with StartingPosition()/PlaceStairs()" (`gen_overworld.go:485-486`). Its only readers are `StartingPosition`, `PlaceStairs` and the save chunk.
  - `StartingPosition` returns `rooms[0].Center()` with no walkability check (`dungeongen.go:80-83`).
- Problem: A dead mechanic forces generators to fabricate rooms, and the player start is inferred from that fabrication.
- Direction: Delete `PlaceStairs`, `STAIRS_DOWN`, the stairs image and its panic. Add an explicit `PlayerStart coords.LogicalPosition` to `GenerationResult`, set by each generator to a tile it knows is walkable. Then `Rooms` can leave `GameMap` and the save file (generators may keep rooms as a local).
- Effort: S-M. Risk: low (save format change for the rooms array).

##### [Medium] `GenContext` promises independence from globals that the code does not have
- Evidence: `generator.go:31-39` says the context exists so generators need not read package globals, but every index computation in `worldgen` and `garrisongen` goes through `coords.CoordManager` (`gen_helpers.go:16-18`, called about 60 times), and `GameMap.TileAt`, `GetBiomeAt`, `IsOpaque` and `StartingPosition` do the same while `GameMap` also stores its own `width` and `height`. The context is built identically in two places (`setup/gamesetup/bootstrap.go:91-95`, `gui/guiexploration/exploration_panels_registry.go:51-55`).
- Problem: A context whose width differs from `CoordManager`'s produces wrong indices or a panic, so the parameter is a second copy of a value that must equal the global.
- Direction: Keep using `CoordManager` for indexing (it is the project rule) and stop pretending otherwise: build the context in one helper from `CoordManager`, or have `NewGameMap` reject a context whose dimensions differ. `TileSize` leaves the context once pixel coordinates leave `Tile`.
- Effort: S. Risk: low.

##### [Low] Biome is stored per tile and again as `biomeMap`
- Evidence: `Tile.Biome` and `GameMap.biomeMap` (`dungeongen.go:56`). `biomeMap` is read only by `GetBiomeAt`, which has no non-test caller, and it is not restored on load (`map_chunk.go:211-213`), so after a load `GetBiomeAt` would return grassland everywhere.
- Direction: Delete `biomeMap` and `GetBiomeAt` from `GameMap`; keep `BiomeMap` as a generator-local slice if useful.
- Effort: S. Risk: low.

#### Mechanics notes
- Biomes have no gameplay effect beyond passability. Swamp and mountain are impassable, the other three are identical to play on. `POIData.Biome` and `FactionStartPosition.Biome` are stored and never read by gameplay. This is fine as presentation, but the data model suggests more than exists.
- `BiomeSwamp` doubles as water: elevation below `WaterThresh` becomes swamp (`gen_overworld.go:234-236`).

#### Probable bugs
- One overworld POI tile is overwritten by stairs. POIs are appended as 1x1 rooms, `PlaceStairs` picks a random room with index >= 1 and sets `TileType = STAIRS_DOWN` and the stairs image at its center, which is the POI tile. After save and load the same tile gets the POI image back because `SelectTileImage` checks `poiType` first. Confidence: high that the tile is overwritten; visible impact depends on how much the overworld node icon covers the tile.
- `StartingPosition` can return a wall tile on cavern maps: it returns the first chamber's center, and `placePillars` (`gen_cavern.go:628-652`) places 2x2 walls within 0.6 of the radius around that center after the chamber is carved. The player entity is registered there unchecked (`setup/gamesetup/playerinit.go:65`). Confidence: medium (traced, not run).

#### Dead code
- `GameMap.IsOpaque`, `GameMap.GetBiomeAt`, `BiomeFromString`: tests only.
- `Tile.IsRevealed`: never set to true or read by rendering; only round-tripped by the save chunk.
- `FactionStartPosition.Sector`: never read.
- `MapGenerator.Description()`: implemented four times, never called.
- `STAIRS_DOWN`, `PlaceStairs`, `stairsFile` (see above).
- Package comment says "Package worldmap" (`dungeongen.go:1`).

#### Performance
- `LoadTileImages` re-reads and decodes every tile PNG on each `NewGameMap` and each save load (`dungeongen.go:67`, `map_chunk.go:141`). Not a hot path, but it is the same missing image cache noted under templates.

#### Leave alone
- Unexported `GameMap` fields with accessors and `NewGameMapFromParts`; this is the right shape.
- The `ConsumeTileColorsDirty` read-then-clear protocol (until the overlay moves).
- `Biome` enum and `tileconfig.go` path helpers.
- `Rect` and its documented inclusive `Intersect` quirk; changing it has no payoff while only debug generators use it.

---

### world/worldgen (1,790 lines, tests: yes, smoke tests)

**Role:** Generator registry, shared terrain helpers, and the three general-purpose generators. **Verdict:** Needs attention. The algorithms are clear and well commented; the problems are configuration plumbing, RNG handling and unused parameters.

#### Design findings

##### [High] Randomness is neither reproducible nor consistently varied
- Evidence:
  - The shared RNG starts at a fixed seed, `rand.NewPCG(1, 2)` (`core/common/randnumgen.go:8`). `SetRNGSeed` has no caller in game startup.
  - `cavern` and `overworld` seed their noise from the wall clock when `Seed == 0` (`gen_cavern.go:102-105`, `gen_overworld.go:102-105`), bypassing the shared RNG.
  - `rooms_corridors` and `garrison_raid` reseed the global shared RNG from inside `Generate` when `Seed != 0` (`gen_rooms_corridors.go:46-48`, `world/garrisongen/generator.go:38-40`).
- Problem: With `Seed == 0` the noise field changes per launch while every dice roll repeats per launch, so maps are not reproducible and not independently random either. With `Seed != 0` a generator resets the RNG stream for the whole game (combat, AI, loot) as a side effect.
- Direction: When `Seed == 0`, seed noise from `common.NextSeed()` (the helper already exists for this purpose). Never call `SetRNGSeed` inside a generator; if a generator needs a private stream, create a local `rand.New` from its seed. Seeding the shared RNG at startup is covered under `core/common`.
- Effort: S. Risk: low.

##### [High] Config duplication and the override hook
- See templates finding "Map-generation config is defined three times". In this package the removable pieces are the three `Default*Config()` literals as production values, `ConfigOverride` (`registry.go:10-14,29-33`), and registry entries that hold a pre-configured instance.
- Effort and risk as stated there.

##### [Medium] Adding a POI type requires edits in about nine places
- Evidence: `worldmapcore/tileconfig.go:22-35` (constants and asset map), `campaign/overworld/ids` (`NodeTypeTown` etc. duplicate the same four strings), `StrategicOverworldConfig` per-type counts (`gen_overworld.go:30-33`), `JSONOverworldGenConfig` (`templates/schema_mapgen.go:54-57`), the copy in `setup/gamesetup/mapgenconfig.go:94-97`, four hand-written blocks in `placeTypedPOIs` (`gen_overworld.go:373-407`) with literal distances 15, 10 and 20, the `isValidPOITerrain` switch (`gen_overworld.go:494-512`), `nodeDefinitions.json`, and `influenceconfig.json nodeTypeMultipliers`.
- Problem: POIs look data-driven (they become overworld nodes from `nodeDefinitions.json`) but placement is hardcoded per type. Elevation and moisture cutoffs 0.40, 0.60, 0.35, 0.55 and 0.50 sit beside configurable thresholds.
- Direction: Replace the four count fields with a list of placement rules in `mapgenconfig.json` (`nodeId`, `count`, `minDist`, allowed biomes, optional elevation bound, optional `nearNodeId` + `maxDist`). `placePOIs` already takes a rule struct, so the loop exists. Use `ids.NodeTypeID` for `POIData.NodeID` and delete the duplicate `POI*` constants.
- Effort: M. Risk: low.

##### [Medium] Neighbor counting and "set tile to floor/wall" are copied instead of shared
- Evidence: The 8-neighbor wall/floor count is written out four times in `gen_cavern.go` (lines 484-499, 527-545, 604-617, 668-681). "Make this tile a floor and record it" is written out in `gen_helpers.go:84-89`, `108-113`, `gen_overworld.go:291-298`, `garrisongen/generator.go:218-221`, `343-346`, `354-357`; the wall variant is `garrisongen/terrain.go:44-46`.
- Problem: Real copy-paste with small variations (out-of-bounds counts as wall in three of the four cavern copies but not the fourth).
- Direction: One `countNeighbors(terrain, x, y, w, h) (walls, floors)` in `gen_helpers.go` and one `setFloor` / `setWall` pair. These fall out naturally if the image-pass refactor above is done, since the helpers shrink to two assignments.
- Effort: S. Risk: low.

##### [Low] Unused parameters throughout the helper signatures
- Evidence: `width` is accepted and ignored by `CarveRoom`, `CarveTunnel`, `CarveHorizontalTunnel`, `CarveVerticalTunnel` (`gen_helpers.go:78-125`) and `connectRooms`; `elevationMap` by `placeFactionStartPositions` (`gen_overworld.go:305`); `elevationMap` and `moistureMap` by `applyConnectivityFixes` (`gen_overworld.go:274`).
- Direction: Remove with the image-parameter cleanup.
- Effort: S. Risk: low.

#### Mechanics notes
- `FactionCount` is capped at four by the hardcoded sector table (`gen_overworld.go:314-321`); a fifth faction silently falls back to `initialsetup.json` positions. Sector rectangles use literal offsets (10, 30, 25) that assume a map of about 100x80.
- Faction start order is positional: if a middle sector yields no position, later factions shift into earlier sectors' slots because results are appended, not indexed by sector.
- Cavern "faction starts" are computed and two 6-tile circles are cleared for them (`gen_cavern.go:695-708`), but nothing consumes them in roguelike mode (`FactionStartPositions()` is read only by overworld setup). The only effect is the cleared circles.

#### Probable bugs
- Cavern connectivity is guaranteed at step 9 and then terrain is changed again: an aggressive CA pass in `checkWalkableRatio` (`gen_cavern.go:590-594`), 2x2 pillars, and stalactites that can wall a 1-wide tunnel (`gen_cavern.go:656-692`). No connectivity check follows. Confidence: medium that disconnected pockets occur on some seeds.
- `checkWalkableRatio`'s comment cites 55% and 35% while the code uses the configured 0.50 and 0.42 (`gen_cavern.go:576-578`). Confidence: high; comment only.
- `CarveRoom` and `CarveTunnel` append to `ValidPositions` without checking for an existing entry, so corridors crossing rooms create duplicates (`gen_helpers.go:89,112`). Affects only the debug-only generators today. Confidence: high.

#### Dead code
- `CavernConfig.NumChambers`: never read; the chamber count is the fixed 3x2 grid (`gen_cavern.go:179-180`).
- `StrategicOverworldConfig.FactionMinSpacing`: never read.
- `RoomsAndCorridorsGenerator`: reachable only from the debug menu and as the registry fallback. Keep if the debug menu is valued; otherwise it and `Rect.Intersect` can go.
- `Description()` on all generators.

#### Performance
- `ConvertTerrainMapToTiles` allocates one `Tile` per cell on the heap (`gen_helpers.go:273-281`), discarding the contiguous block (and the random wall image per tile) that `CreateEmptyTiles` just built. One-time cost per map; fix in passing by mutating the existing tiles.

#### Leave alone
- The generator registry and `init()` self-registration.
- The cavern pipeline structure and its step comments; `EnsureTerrainConnectivity`, `FloodFillRegion`, `FindBestOpenPosition`.
- `poiPlacementRule` and `placePOIs`; this is the right abstraction, it just needs to be fed from data.
- Operating on a `[]bool` terrain map and converting to tiles at the end (cavern). This is the pattern the other generators should follow.

---

### world/garrisongen (1,345 lines, tests: no)

**Role:** Abstract floor DAG for raids, plus a tile-map generator for garrison floors. **Verdict:** Needs significant rework, because about three quarters of the package is not connected to the game and the part that is connected is duplicated in JSON.

#### Design findings

##### [High] The tile generator is debug-only and unrelated to the raid it is named for
- Evidence:
  - `GarrisonRaidGenerator` is registered with `FloorNumber: 1` (`generator.go:17-22,417-419`). `NewGarrisonRaidGenerator` has no caller. `setup/gamesetup/mapgenconfig.go:105-113` returns `nil` on both branches, so the registry instance is always used.
  - The only way to run it is the exploration debug menu (`gui/guiexploration/exploration_panels_registry.go:102`).
  - Inside `Generate` it builds its own DAG (`generator.go:56`), a different random draw from the one `campaign/raid` built for the same floor (`campaign/raid/garrison.go:74`). The DAG is discarded when `Generate` returns; only rectangles survive.
  - `campaign/raid` has no reference to `GameMap` or `worldmapcore`.
- Problem: `generator.go` (419 lines) and `terrain.go` (565 lines) implement room placement, three corridor shapes, doorways and 13 hand-authored room layouts that no raid ever uses. Room sizes and corridor widths in the DAG nodes (`FloorNode.MinWidth` etc.) matter only to this unused path. The code has no tests and reads as live gameplay.
- Direction: This is a product decision to put to the user. Option A: raids are fought on generated floors; then `Generate` must take the raid's `FloorDAG` (and floor number) as input, return room rectangles keyed by node ID, and `campaign/raid` must use them for deployment. Option B: raids stay abstract; then delete `generator.go` and `terrain.go`, the `garrison_raid` debug button, `roomSizes` from JSON, and the size fields on `FloorNode`. Either way, stop generating a second DAG.
- Effort: S for B, L for A. Risk: low for B; high for A (no tests).

##### [High] Floor scaling and room sizes are hardcoded tables that JSON then overwrites
- Evidence: `dag.go:43-84` versus `mapgenconfig.json generators.garrison_raid`, connected by `SetGarrisonRoomSizes` / `SetGarrisonFloorScaling` (`dag.go:87-94`) called from `setup/gamesetup/mapgenconfig.go:115-144`. `BuildGarrisonDAG` silently uses floor 1's entry for any floor missing from the table (`dag.go:100-103`).
- Problem: Two copies of gameplay data (how many rooms per raid floor, which room types appear) with mutable package globals in between. Raid floor structure is raid balance data, yet it lives in the map-generation config file rather than in `raidconfig.json`, which `campaign/raid` loads itself.
- Direction: One source: move `floorScaling` into `raidconfig.json` (or keep it in mapgen config but load it once), pass the table into `BuildGarrisonDAG` as an argument, and delete the hardcoded tables and setters. Validate that every floor up to the raid's floor count has an entry.
- Effort: S-M. Risk: low.

##### [Medium] Room-type knowledge is spread across the slice and the raid package
- Evidence: Constants in `roomtypes/roomtypes.go:7-16`, aliased again as `GarrisonRoom*` (`dag.go:11-20`); "combat" set (`dag.go:237-244`); "branch" set (`dag.go:259-265`); corridor width by type (`generator.go:235-247`); terrain dispatch (`terrain.go:13-27`); size table (`dag.go:43-52`); and behavior switches on the same strings in `campaign/raid/raidrunner.go:129-132`, `alert.go:33`, `rewards.go:41`, `assignment.go:16`, `floorgraph.go:96`, `gui/guiraid/raidmode.go:273`.
- Problem: Adding a room type means finding every switch. The alias layer adds a second name for each constant with no benefit.
- Direction: A single table of room-type properties (`IsCombat`, `CanBranch`, plus whatever raid needs such as `IsRest`, `IsStairs`) owned by whichever package keeps the DAG, with the raid switches reading properties. Drop the `GarrisonRoom*` aliases and use one set of names. Do this after the option A/B decision, since B removes half the list.
- Effort: S-M. Risk: low.

##### [Low] `BuildGarrisonDAG` repeats its rest-room rule and branch-type pick
- Evidence: `dag.go:162-173` and `dag.go:194-206` are the same "pick a branch type, at most one rest room" logic written twice with slightly different fallbacks.
- Direction: Extract one `pickBranchType(branchTypes, &hasRestRoom) (string, bool)`.
- Effort: S. Risk: low (no tests).

#### Mechanics notes
- "At most one rest room per floor" is enforced in the DAG builder; rest-room behavior is in `campaign/raid`. Fine, but note the rule lives in the world package.
- `findDownstreamCriticalNodes` returns at most one node, so the "pick a random downstream node" in `dag.go:184` is always a single choice; the 50% reconnect creates a diamond only to the immediate next critical node. The second loop (grandchildren, `dag.go:299-306`) cannot find anything for a critical-path parent whose direct critical child is the stairs, except through branch children. Low value; simplify to "next critical node if it is not the stairs".

#### Probable bugs
- `setTileWall` re-blocks tiles without removing them from `ValidPositions` (`terrain.go:36-48`), so the list contains walls after terrain injection and doorway placement. Debug-only path today. Confidence: high.
- Terrain injection and doorway narrowing run after the connectivity pass (`generator.go:207-229`) with no re-check, so layouts such as the armory barricade or kill box can in principle seal part of a room. Confidence: low to medium; not reproduced.
- `placeDoorway`'s comment says it narrows to one tile; the code leaves three tiles open (`generator.go:364-378`). Confidence: high; comment only.

#### Dead code
- `placePillarRow` (`terrain.go:133-141`): no caller.
- `NewGarrisonRaidGenerator`, `GarrisonFloorConfig.Seed`, `DefaultGarrisonConfig` beyond the single `init` use.
- Under option B: all of `generator.go` and `terrain.go`.

#### Leave alone
- `FloorDAG` / `FloorNode` graph shape, `TopologicalSort`, `DagDepth`.
- The terrain primitives (`placeUAlcove`, `placeLCorner`, `placeBarricadeLine`) are tidy; if option A is chosen they are worth keeping as they are.

---

### world/garrisongen/roomtypes (38 lines, tests: no)

**Role:** Leaf package holding room-type string constants so `templates` can validate them. **Verdict:** Healthy but only needed because of where validation sits.

#### Design findings

##### [Low] Exists only to break one import cycle
- Evidence: Imported by `templates/validation.go:7` (two lookups) and aliased in `garrisongen/dag.go:11-20`. `roomtypes.All` has no reader other than building `Valid`.
- Direction: If garrison floor config moves to the raid's own loader (see garrisongen), `templates` no longer needs these names and the constants can live with the DAG. Otherwise leave it.
- Effort: S. Risk: low.

#### Leave alone
- If it stays, it is fine as is.

---

## 12. Squads and commander

Scope: all non-test `.go` files in `tactical/squads/{squadcore,squadcommands,squadservices,roster,unitdefs,unitprogression}` and `tactical/commander` were read in full. Claims of "unused" were checked by grepping the whole repo including `tools/` and tests. Nothing was compiled or run; bugs are traced by reading the call paths.

### Layer overview

**How the packages relate today**

```
unitprogression  <- unitdefs <- squadcore <- roster <- squadservices <- squadcommands -> tactical/combat/{combatcore,combatstate}
                                    ^            ^
                                    |            +-- commander -> campaign/overworld/core, tactical/powers/progression
                                    +-- 30 importing packages
```

- `squadcore` is the real hub (30 importers) and is in decent shape as an ECS package. Most of the slice's problems sit *around* it: in how units get created, and in the ownership bookkeeping layered on top.
- The dominant theme of the slice is **the same fact stored in two places**, and four of the probable bugs below come directly from it:
  1. unit attributes: `UnitTemplate.Attributes` vs `UnitTemplate.EntityData.Attributes` (leader Leadership override is lost);
  2. movement speed / attack range: `common.Attributes.MovementSpeed/AttackRange` vs `squadcore.MovementSpeedData/AttackRangeData` (stat effects hit the copy nobody reads);
  3. unit availability: `UnitRosterEntry.TotalOwned/UnitsInSquads/entryByEntity` vs the ECS (`SquadMemberComponent`) (drift after casualties, broken after load);
  4. targeting rules: `combatmath.SelectTargetUnits` vs `squadcore.computeGenericTargetCells`;
  5. overworld clock: `commander.OverworldTurnStateData.CurrentTurn` vs `core.TickStateData.CurrentTick`.
- The second theme is **three parallel "build a unit / build a squad" paths** (purchase path, template path, simulator path) because the template path loads images inside simulation code.

**Layout recommendations (in priority order)**

1. Collapse `squadservices` (329 lines, 2 files). `SquadDeploymentService` is two world queries used only by one GUI mode; `UnitPurchaseService` is a stateless struct. Move purchase/refund into `roster` as plain functions (it is roster + gold logic), move or delete the deployment helpers. Net: one package and several result structs disappear.
2. Move `MoveSquadCommand` out of `squadcommands`. It is the only reason `tactical/squads/*` imports `tactical/combat/*` (an upward dependency), and the only reason `mind/ai` imports `squadcommands`. It belongs next to its one real user (`gui/guicombat/combatbase`) or in `tactical/combat/combatservices`.
3. Move `tactical/commander` under `campaign/`. It imports `campaign/overworld/core`, owns the overworld turn state, and moves commanders on the overworld map; nothing in it is tactical-combat logic. Only `FindCommanderForSquad` is needed by tactical code (`tactical/powers/spells`), and that one function reads only `roster.SquadRoster`, so it can live in `roster`.
4. Leave `squadcore`, `unitdefs`, `unitprogression`, `roster` as packages. Do not split `squadcore` further; its file sizes (max 537 lines) are fine.

---

### tactical/squads/squadcore (1418 lines, tests: yes)

**Role:** Squad and unit ECS components, creation/placement/disposal, and squad queries. **Verdict:** Needs attention. The query side is clean and consistent, but unit creation exists in three diverging copies, placement validation is inconsistent across entry points, and `SquadData`/`LeaderData` carry fields nothing reads.

#### Design findings

##### [High] Three unit/squad construction paths that have already diverged
- Evidence: `tactical/squads/squadcore/units.go:18-62` (`CreateUnitEntity`: `templates.CreateUnit` + attributes from `unit.Attributes`, image load failure is a warning); `tactical/squads/squadcore/squadcreation.go:329-384` (`placeTemplateUnit`: `templates.CreateCreatureEntity` + attributes from `template.EntityData.Attributes`, image load failure skips the unit, re-adds Name and Position components after the factory already added them); `tools/combat_analysis/combat_simulator/squad_factory.go:14-100` (`createSimSquad`, self-described "headless equivalent of CreateSquadFromTemplate ... mirrors squadcreation.go:200-355", with its own string-keyed occupancy map, literal `3` and `MaxUnits: 9`). The save loader (`setup/savesystem/chunks/squad_chunk.go:270-380`) is a fourth hand-built copy of the component set.
- Problem: The same job (template -> unit entity in a squad) is done three ways with different attribute sources, different failure behavior, and different occupancy code. They already disagree (see probable bug 1). The simulator copy exists only because the game path decodes images inside `squadcore`, so the balance tool does not exercise the code the game runs.
- Direction: One unit builder in `squadcore` that takes a template and builds the entity without touching ebiten (Name, Position, Attributes from `template.Attributes`, then `ApplyUnitComponents`). Attach the sprite in a separate, optional step (a function taking the entity and template, called by game setup and not by the simulator). `CreateSquadFromTemplate` and `PurchaseUnit` both call the one builder; `createSimSquad` is deleted in favor of `CreateSquadFromTemplate`; `templates.CreateCreatureEntity` loses its only caller.
- Effort: M. Risk: medium (touches every squad spawn path; squadcore has tests, the template path is covered by them only partly).

##### [High] Placement validation differs per entry point; add/place checks only the anchor cell
- Evidence: `squadcreation.go:189-199` (`PlaceUnitInSquad`: `ValidateGridAnchor` + occupancy of the anchor cell only); `squadcreation.go:133-148` (`AddUnitToSquad`: same); `squadcreation.go:257-284` (`MoveUnitInSquad`: full footprint bounds + per-cell occupancy); `squadcreation.go:345-358` (`placeTemplateUnit`: full footprint with a local `occupied` array); `squadcommands/command_helpers.go:32-34,57-65` and `move_unit_command.go:52-57` (commands re-validate the anchor only, before squadcore validates again).
- Problem: Four implementations of "does this unit fit here", two of them wrong for multi-cell units (see probable bug 2). Callers cannot tell which guarantees they get.
- Direction: One unexported `canPlaceFootprint(squadID, row, col, w, h, ignoreUnitID)` in squadcore used by Place, Move and template placement. Commands stop pre-validating grid rules and rely on the squadcore error (they keep only "squad exists / unit in squad" checks).
- Effort: S. Risk: low.

##### [Medium] `SquadData` and `LeaderData` carry fields no game logic reads
- Evidence: `squadcomponents.go:46-56`. Repo-wide grep: `Formation`, `Morale`, `SquadLevel`, `TurnCount`, `MaxUnits` on `SquadData` are only written at creation and copied by the save chunk (`setup/savesystem/chunks/squad_chunk.go:145-149,277-281`); nothing branches on them. `SquadData.SquadID` duplicates the entity ID. `LeaderData.Leadership` (hardcoded 10 at `squadcreation.go:19-22`) and `LeaderData.Experience` (`squadcomponents.go:185-188`) are only saved/loaded; real capacity comes from `Attributes.Leadership` (`squadqueries.go:261-274`).
- Problem: Readers assume formation, morale and leader-leadership are live mechanics. `FormationType` is threaded through nine `CreateSquadFromTemplate` call sites and the simulator even runs "Balanced vs Offensive" formation suites (`tools/combat_analysis/combat_simulator/suite_compositions.go:212-223`) on a value with no effect. `LeaderData.Leadership` reads like a second leadership stat next to `Attributes.Leadership`.
- Direction: Delete the unread fields and the `formation` parameter; make `LeaderComponent` a pure marker (empty struct). Keep `Name`, `IsDeployed`, `GarrisonedAtNodeID`. Drop `SquadID` from `SquadData` or keep it purely as a convenience, but stop using it for lookup (next finding). Save format changes accordingly (user stated no compatibility constraint).
- Effort: M (many call sites, mechanical). Risk: low.

##### [Medium] Two ways to look up a squad; the slower one is the "canonical" one
- Evidence: `squadqueries.go:118-127` (`GetSquadEntity` scans every squad comparing `SquadData.SquadID`); `core/common/ecsutil.go:102-109` (`FindEntityByID` is an O(1) map lookup). Both styles are used for the same thing: `squadcreation.go:449` uses `FindEntityByID(squadID)` "for reliability", `squadqueries.go:304-305,371` use `GetSquadEntity`.
- Problem: Two lookups that can in principle disagree (one trusts the component field, one the entity ID), and the commonly used one is O(squads) inside functions like `GetSquadDistance` and `GetSquadName` (22 call sites, several per-frame in GUI).
- Direction: Implement `GetSquadEntity` as `FindEntityByID` + `HasComponent(SquadComponent)`; callers stay unchanged.
- Effort: S. Risk: low.

##### [Medium] Attack-pattern preview re-encodes targeting rules owned by combatmath
- Evidence: `squadqueries.go:386-465` (`AttackPatternCell`, `ComputeGenericPatternFiltered`, `computeGenericTargetCells` with literal `[3][3]` and `{0,0},{0,1},{0,2}`) vs `tactical/combat/combatmath/combattargeting.go:12-110` (`SelectTargetUnits` and friends). Callers are GUI only: `gui/guisquads/squadeditor_grid.go:114,123`, `gui/guiinspect/inspect_panel.go:132`.
- Problem: Which cells each `AttackType` hits is defined twice. A change to a targeting rule must be made in both or the editor preview lies. The preview also ignores `SquadGridSize` although the file header says all bounds derive from it.
- Direction: Move the preview next to the real rules in `combatmath` and have both derive from one "base cells for attack type and attacker position" function (the real selector then adds pierce/fallback on top). Removes ~80 lines from squadcore and its only display-shaped type.
- Effort: S-M. Risk: low.

##### [Medium] Leader lifecycle is spread over three places and has no rule for a dead leader
- Evidence: promotion inside `attachUnitToSquadAtGrid` (`squadcreation.go:99-104`); swap logic inlined in `squadcommands/change_leader_command.go:53-104`; strip in `UnassignUnitFromSquad` (`squadcreation.go:239-242`); `SetSquadRenderableFromLeader` must be called by hand after each (6 call sites). `DisposeDeadUnitsInSquad` (`squadcreation.go:422-442`) disposes a dead leader with no promotion and no renderable refresh.
- Problem: "Who is leader and what sprite does the squad show" is derived state copied onto the squad entity and maintained by convention. After the leader dies the squad silently falls back to `config.DefaultBaseCapacity` (`squadqueries.go:262-266`) and keeps the dead leader's sprite; the next unit added becomes leader as a side effect of placement.
- Direction: One `SetSquadLeader(squadID, unitID)` in squadcore that swaps the marker and refreshes the squad renderable; `ChangeLeaderCommand`, attach, and dead-unit disposal call it (disposal picks a successor or explicitly clears). Decide the dead-leader rule once and encode it there.
- Effort: S. Risk: low-medium (changes behavior when a leader dies; flag as a gameplay decision).

##### [Low] Small inconsistencies worth fixing while in the file
- `GetSquadPrimaryRole` (`squadqueries.go:474-506`) breaks ties by Go map iteration order, so the AI's role for a squad with e.g. 2 Tank / 2 DPS varies run to run (`mind/ai/ai_controller.go`, `mind/behavior/threat_composite.go` consume it). Iterate roles in a fixed order.
- `placeTemplateUnit` reports problems with `fmt.Printf` and silently skips the unit (`squadcreation.go:346,353,362`); `CreateSquadFromTemplate` therefore cannot tell callers a squad came out short. Return an error or a count.
- Width/height normalisation (`0 -> 1`) is repeated in `ValidateGridPlacement`, `placeTemplateUnit` and `ApplyUnitComponents` while `CreateUnitEntity` rejects 0 (`units.go:21-27`). `unitdefs.CreateUnitTemplates` already guarantees 1-3, so the defaults can go.
- Query functions take a `*common.EntityManager` they ignore because they read package-level views (`squadqueries.go:71,100,118,131,143`). This is consistent with the repo-wide global-component pattern; no action beyond not adding more.
- Component helper methods on `GridPositionData`/`CoverData` (`squadcomponents.go:103-126,144-149`) bend the "pure data" rule in CLAUDE.md; `OccupiesCell` is a harmless geometry helper, the others are dead (below).

#### Mechanics notes
- **Capacity is enforced only on one path.** `PlaceUnitInSquad` checks capacity (`squadcreation.go:201-211`); `CreateSquadFromTemplate` does not, `ChangeLeaderCommand` does not re-check after swapping to a lower-Leadership leader, and leader death drops capacity without consequence. `IsSquadOverCapacity` (documented as "for displaying warnings") has no non-test caller. The rule exists but only gates the squad editor's add button.
- **`RequiresActive` on cover is nearly moot.** Dead units are disposed right after each attack (`tactical/combat/combatcore/combatactionsystem.go:202-207`), so a dead cover provider rarely exists to be tested; 20 of 30 monsters set it to false anyway. `combatcover.go:107-115` checks the flag and then calls `GetCoverBonus`, which checks it again.
- **`XPToNextLevel` is per-unit saved state but is always 100** (`units.go:113-117`); it is a constant stored as data.

#### Probable bugs
1. **Leader Leadership override is dropped for every template-built squad.** Confidence: high. Callers copy a template and set `unit.Attributes.Leadership = 20` (`setup/gamesetup/initial_squads.go:173,215,252,300,339`; `mind/spawning/squadscreation.go:123`), then call `CreateSquadFromTemplate`. `placeTemplateUnit` builds the entity from `template.EntityData` (`squadcreation.go:360` -> `templates/entity_factory.go:55` `m.Attributes.NewAttributesFromJson()`), never reading `template.Attributes`. Starting and spawned squad leaders keep their JSON Leadership, so `GetSquadTotalCapacity` is lower than intended. The simulator's `createSimSquad` uses `tmpl.Attributes`, so the balance tool and the game differ.
2. **Multi-cell units can be placed overlapping or off-grid from the squad editor.** Confidence: high. `PlaceUnitInSquad` validates and occupancy-checks only the anchor (`squadcreation.go:191-199`), `AddUnitCommand.Validate` likewise (`squadcommands/add_unit_command.go:61-66`). `monsterdata.json` has two width-2 units and one height-2 unit. Adding one at column 2 (or next to an occupied cell) succeeds; `GetOccupiedCells` then clips silently.
3. **Dead units' sprites/leader state:** after a leader dies the squad entity keeps the dead leader's image (no refresh in `DisposeDeadUnitsInSquad`). Confidence: medium (depends on whether the renderer reads the squad's or leader's renderable at that moment; squad renderable is a copied pointer, so it stays valid).

#### Dead code
- `RemoveUnitFromSquad` (`squadcreation.go:169-184`): no callers anywhere.
- `AddUnitToSquad` (`squadcreation.go:133-167`): test-only (24 test call sites, 0 production). Production uses `PlaceUnitInSquad` via roster.
- `InitializeSquadData` (`squadmanager.go:64-69`): test-only; ignores its parameter.
- `GridPositionData.GetRows` (no callers), `GetOccupiedCells` (tests only), `TargetRowData.String` (no explicit callers), `IsSquadOverCapacity` (tests only).
- `WorldTags["squad"|"squadmember"|"leader"]` registrations (`squadmanager.go:56-58`): never read by key.
- `SquadData.Formation/Morale/SquadLevel/TurnCount/MaxUnits`, `LeaderData.Leadership/Experience`, type `FormationType`: stored, never read (see finding above).

#### Performance
- `ebitenutil.NewImageFromFile` is called once per unit created (`units.go:45-56`, and `templates/entity_factory.go:36` on the template path) with no cache: five identical goblins decode and upload the same PNG five times. Fixing the construction finding (sprite attach as a separate step) is the natural place to add a path-keyed image cache.
- `GetUnitIDsInSquad` and `GetUnitIDsAtGridPosition` scan every squad member in the world per call (`squadqueries.go:71-114`). Callers stack them: `combatmath.getUnitsInLine` calls `GetUnitIDsAtGridPosition` three times per row/column, and melee-row targeting may do that for three rows, i.e. up to 9 full-world scans to pick one unit's targets; `MoveUnitInSquad` does one per footprint cell. With all overworld garrisons alive in the ECS this grows with world size, not squad size. Direction: no cache; add one helper that builds a squad's grid (`[SquadGridSize][SquadGridSize]ecs.EntityID`) in a single pass and have row/column/cell callers use it.

#### Leave alone
- The view-backed query set (`GetUnitIDsInSquad`, `GetLeaderID`, `ForEachUnitWithAttrs`, `CountLivingUnitsInSquad`, `IsSquadDestroyed`, `WouldSquadSurvive`, capacity queries, `GetSquadMovementSpeed[OrDefault]`, `GetSquadHealthPercent`, `GetSquadDistance`). Clear, read-only, ID-based.
- `ApplyUnitComponents` as the single list of a unit's squad components; `SquadGridSize`/`SquadMaxUnits` constants; `ValidateGridPlacement`.
- `DisposeSquadAndUnits`, `ResetSquadDeployment`, `UnassignUnitFromSquad`: correct use of `CleanDisposeEntity`/`UnregisterEntityPosition`.
- File layout (components / creation / queries / units). Do not split further.

---

### tactical/squads/roster (398 lines, tests: yes)

**Role:** Player-owned unit pool (`UnitRoster`, on the player entity) and per-commander squad list (`SquadRoster`). **Verdict:** Needs significant rework. `UnitRoster` maintains three hand-synchronised derived structures on top of facts the ECS already holds; they are wrong after save/load and after any combat casualty.

#### Design findings

##### [High] `UnitRoster` duplicates ECS state in three derived structures that drift
- Evidence: `roster/unitroster.go:17-33`: per type `TotalOwned` (always equals `len(UnitEntities)`), `UnitsInSquads map[squadID]int` (a count of units that have `SquadMemberComponent`), and the unexported index `entryByEntity`. Availability is answered two different ways in the same file: count-based `GetAvailableCount` (`:121-133`) and ECS-based `isUnitAvailable` (`:221-223`, used by `GetUnitEntityForTemplate` and `GetAvailableUnitDetails`). `MarkUnitAvailable` (`:146-161`) ignores which squad the unit was in and decrements whichever squad the map iterator yields first. To keep the copies aligned the package needs `AssignUnitToSquad`/`UnassignUnitFromSquad` wrappers with manual rollback (`:254-297`), and the GUI needs `backfillRosterWithSquadUnits` (`gui/guisquads/squadeditor_roster.go:300-345`) to repair it on entry.
- Problem: Every unit membership change must be mirrored by hand; any path that forgets (combat disposal, save load, squads created by setup) desyncs it. Probable bugs 1-3 below are all this. The per-squad breakdown in `UnitsInSquads` is never read per squad by anyone, only summed.
- Direction: Reduce `UnitRoster` to the one fact the ECS does not have: which unit entities the player owns (`UnitIDs []ecs.EntityID`, `MaxUnits`). Derive everything else on demand: type from `UnitTypeComponent`, availability from absence of `SquadMemberComponent`, counts by iterating. This deletes `UnitRosterEntry`, `TotalOwned`, `UnitsInSquads`, `entryByEntity`, `MarkUnitInSquad`, `MarkUnitAvailable`, both roster-level assign/unassign wrappers (commands call `squadcore.PlaceUnitInSquad`/`UnassignUnitFromSquad` directly), the remap code in `player_chunk.go:207-221`, and the GUI backfill's nested scan. Roster size is tens of units, so derived counts are free.
- Effort: M. Risk: medium (roster has tests that pin the current bookkeeping API and will need rewriting; save chunk changes).

##### [High] Nothing removes dead units or destroyed squads from the rosters
- Evidence: `UnitRoster.RemoveUnit` is called only from the purchase rollback/refund (`squadservices/unit_purchase_service.go:142,234`). `SquadRoster.RemoveSquad` (`squadroster.go:50-59`) has no callers. Combat disposes units and whole squads (`tactical/combat/combatcore/combatactionsystem.go:194-207`, `tactical/combat/combatdisposal/squad_removal.go:38`) and nothing in `mind/combatlifecycle` touches either roster.
- Problem: Ownership lists accumulate IDs of disposed entities (see probable bug 2).
- Direction: One "unit died / squad destroyed" cleanup step owned by the combat teardown path that prunes both rosters, or make the rosters self-healing (prune IDs whose entity no longer exists when read). With the simplified roster above, pruning is a single filter. See also `mind/combatlifecycle` and `tactical/combat/combatdisposal`.
- Effort: S after the rework. Risk: low.

##### [Low] `SquadRoster` API is larger than its use
- Evidence: `GetReserveSquads` (no callers), `RemoveSquad` (no callers), `GetSquadCount` (one GUI caller). 21 external sites read `OwnedSquads` directly.
- Direction: Fine as a plain data component; delete the unused methods once squad-destruction cleanup exists (it will need `RemoveSquad`, so keep that one).
- Effort: S. Risk: low.

#### Probable bugs
1. **After loading a save, roster units cannot be assigned or unassigned.** Confidence: high. The loader builds entries and writes `roster.Units[...] = entry` directly (`setup/savesystem/chunks/player_chunk.go:156-171`); `entryByEntity` is unexported, is only filled by `AddUnit` (`unitroster.go:77`), and no other file references it. After load `entryForUnit` returns nil, so `MarkUnitInSquad`/`MarkUnitAvailable` fail with "unit not found in roster": `AssignUnitToSquad` places then rolls back, `UnassignUnitFromSquad` fails before touching the ECS, `RemoveUnit` returns false. The GUI backfill does not repair it because it sees the IDs already in `UnitEntities`.
2. **Casualties leak roster capacity and leave ghost entries.** Confidence: high for the stale data, medium for the visible symptoms. A unit killed in combat stays in `UnitEntities`/`TotalOwned`/`UnitsInSquads`. `GetUnitCount` stays inflated, so `CanAddUnit` (and purchasing) eventually reports the roster full. `isUnitAvailable` is true for a disposed ID (`HasComponent` returns false for a missing entity, `core/common/ecsutil.go:51-66`), so `GetAvailableUnitDetails` lists the dead unit as "Unknown (Type)" in the editor, while count-based `AddUnitCommand.Validate` says none are available. Destroyed player squads likewise stay in `SquadRoster.OwnedSquads` and keep consuming `MaxSquads`; consumers that iterate `OwnedSquads` (`mind/encounter/encounter_service.go:219-221`, `gui/guiraid/raidmode.go:153`) receive IDs of disposed squads.
3. **`MarkUnitAvailable` decrements an arbitrary squad's count** (`unitroster.go:151-158`). Confidence: high that it is wrong per squad; low impact today because only the sum is read.

#### Dead code
- `UnitRoster.GetAvailableUnits` (tests only), `SquadRoster.GetReserveSquads`, `SquadRoster.RemoveSquad`.

#### Leave alone
- Storing the rosters as components on the player/commander entity, and `GetPlayerRoster`/`GetPlayerSquadRoster` accessors.
- `SquadRoster` as a simple ordered ID list with `MaxSquads`.

---

### tactical/squads/squadcommands (1142 lines, tests: no)

**Role:** Undoable squad-editing commands plus the generic undo/redo executor used by GUI modes. **Verdict:** Needs attention. The pattern is sound and worth keeping, but one command belongs to combat, one is unused, and the commands duplicate rules that squadcore owns.

#### Design findings

##### [High] `MoveSquadCommand` is a combat command living in the squad-editing package
- Evidence: `squadcommands/move_squad_command.go:6-7` imports `combatcore` and `combatstate` (the only upward dependency from `tactical/squads/*` into `tactical/combat/*`). Users: `gui/guicombat/combatbase/handler.go:123` (with undo) and `mind/ai/action_evaluator.go:35-44`, where the AI builds the command and calls `cmd.Execute()` directly, bypassing `Validate` and never undoing.
- Problem: Squad packages depend on combat, and the AI depends on a GUI undo abstraction purely to call `movementSystem.MoveSquad`.
- Direction: Move the command beside its real user (combat GUI handler package or `combatservices`). Have `MoveAction.Execute` call `movementSystem.MoveSquad` directly. `squadcommands` then imports only squads packages.
- Effort: S. Risk: low (no tests).

##### [Medium] Undo of a combat move bypasses the movement system
- Evidence: `move_squad_command.go:105-143` restores position with `EntityManager.MoveSquadAndMembers` and rewrites `ActionStateData` fields by hand; the forward path goes through `CombatMovementSystem.MoveSquad`, which fires `onMoveComplete` (`tactical/combat/combatcore/combatmovementsystem.go:101-107`).
- Problem: Anything the post-move hook does (cache invalidation, perk/artifact reactions, threat-layer dirtying) is not reversed or re-notified on undo; the command also knows the internals of the action-state component.
- Direction: Give the movement system an explicit "revert move" (or have undo call a movement-system function that restores state and fires the same hook). The subscribers are wired in `tactical/combat/combatservices/power_orchestrator.go:119-126`: perk movement tracking and the GUI cache-invalidation hook.
- Effort: S. Risk: medium (no tests; gameplay-visible).

##### [Medium] Commands re-implement validation and leader logic that squadcore owns
- Evidence: `command_helpers.go:17-28` and `:72-83` are the same check twice (`validateSquadExists` / `getSquadOrError`); `:32-34,57-65` duplicate grid checks (anchor-only, see squadcore finding); `change_leader_command.go:53-104` inlines the leader swap + renderable refresh twice (execute and undo).
- Problem: Rules live in two layers; the command-layer copies are the weaker ones.
- Direction: Commands capture undo state and call one squadcore function each; grid and leader rules live only in squadcore. Merge the two squad-exists helpers.
- Effort: S. Risk: low.

##### [Medium] `AddUnitCommand` takes a unit type, not the unit the player picked
- Evidence: the editor lists individual units with IDs (`roster.RosterUnitEntry{ID, Name, TemplateName}`, `gui/guisquads/squadeditor_refresh.go:107`) but dispatches `NewAddUnitCommand(..., entry.TemplateName, row, col)` (`gui/guisquads/squadeditor_roster.go:92-98`); `Execute` then takes the first available entity of that type (`add_unit_command.go:76`).
- Problem: Units are individuals (own name, XP, level, grown stats). Selecting "Aldric (Knight)" can place a different Knight. Redo after undo can also pick a different entity than the first execution.
- Direction: The command takes the unit entity ID. `GetUnitEntityForTemplate` and the count-based `GetAvailableCount` check become unnecessary.
- Effort: S. Risk: low.

##### [Low] Executor details
- `Redo` re-runs `Execute` without `Validate` (`command_executor.go:100-134`).
- `CommandResult{Success bool, Error string}` converts errors to strings that callers convert back with `errors.New` (`gui/guicombat/combatbase/handler.go:131-134`). Returning `error` plus a description would be simpler; low priority.
- `CommandExecutor`/`SquadCommand` are generic and used by `gui/framework/commandhistory.go`; the name and location are acceptable as long as only squad commands exist. Do not generalise now.

#### Probable bugs
- See "AddUnitCommand takes a unit type" (wrong unit placed; confidence high on the code path).
- `setup/gamesetup/playerinit.go:12` blank-imports this package "to trigger init() for command queue components"; the package has no `init()`. Harmless but stale. Confidence: high.

#### Dead code
- `reorder_squads_command.go` (108 lines): `NewReorderSquadsCommand` has no callers.
- `CommandExecutor.GetHistoryCount`, `GetRedoCount`: no callers.
- `RemoveUnitCommand.Description` uses `fmt.Sprintf` with no arguments (`remove_unit_command.go:85`); cosmetic.

#### Leave alone
- The `SquadCommand` interface and executor (validate / execute / undo / description, bounded history). It is small and earns its keep in the squad editor.
- `RenameSquadCommand`, `MoveUnitCommand`, `PurchaseUnitCommand` shape.

---

### tactical/squads/squadservices (329 lines, tests: no)

**Role:** Two "service" structs for unit purchasing and squad deployment used by `gui/guisquads`. **Verdict:** Needs attention; recommend dissolving the package. Both services are stateless wrappers around an `EntityManager`, and the result structs are mostly unread.

#### Design findings

##### [Medium] `UnitPurchaseService` is a stateless struct with over-built result types
- Evidence: `unit_purchase_service.go:15-24` (only field is the manager); `GetUnitCost` returns `template.Cost` (`:57-59`); four result structs (`:27-54,198-205`). Repo-wide, the only result fields read are `CanPurchase`, `Error`, `Success`, `CostPaid`, `UnitID` and the three `PlayerPurchaseInfo` fields; `RemainingGold`, `GoldRefunded`, `PlayerGold`, `UnitCost`, and the `RosterCount/RosterCapacity` on Purchase/Validation/Refund results are never read. `PurchaseUnitCommand` turns the string errors back into `error` (`squadcommands/purchase_unit_command.go:47-48,59-60`).
- Problem: A service object, three unread-field structs and a string-error convention for what is "check gold and roster space, create unit, add to roster, spend gold".
- Direction: Plain functions in `roster` (`CanPurchaseUnit(...) error`, `PurchaseUnit(...) (ecs.EntityID, error)`, `RefundUnit(...) error`); the GUI reads gold and roster counts directly for display. Keep the create/add/spend rollback order, which is correct.
- Effort: S. Risk: low (no tests).

##### [Medium] `SquadDeploymentService` operates on every squad in the world
- Evidence: `squad_deployment_service.go:31-59` and `:62-75` query `squadcore.SquadTag` globally; used only by `gui/guisquads/squaddeploymentmode.go:122,273` and `squaddeployment_panels_registry.go:68`.
- Problem: Not scoped to the player's squads (see probable bug), and `GetAllSquadPositions` builds a world-wide map to answer "where is this one squad" (`squaddeploymentmode.go:122-123`).
- Direction: Delete the service. The mode already has the commander's `OwnedSquads`; reading one squad's `PositionComponent` and resetting owned squads (via `squadcore.ResetSquadDeployment`, which already exists for exactly this) needs no service.
- Effort: S. Risk: low.

#### Probable bugs
1. **"Clear all" moves every squad in the world to (0,0) and leaves them flagged deployed.** Confidence: medium-high. `ClearAllSquadPositions` iterates all `SquadTag` entities (enemy, garrison and raid squads included) and calls `MoveEntity(..., (0,0))`, which also registers each of them at (0,0) in `GlobalPositionSystem`. Placement sets `IsDeployed = true` (`gui/guisquads/squaddeploymentmode.go:255-257`) but clear never resets it, so cleared squads still count as deployed for `GetDeployedSquads`.
2. `GetAllSquadPositions` reports a position for every squad because every squad is created with a `PositionComponent`; the "Not yet placed" branch in `squaddeploymentmode.go:127-131` is unreachable for a valid squad. Confidence: high.

#### Leave alone
- The transaction order and rollback in `PurchaseUnit`/`RefundUnitPurchase` (carry it over as-is).

---

### tactical/squads/unitdefs (251 lines, tests: no)

**Role:** Unit role / attack-type enums and the `UnitTemplate` registry built from monster JSON. **Verdict:** Needs attention. Small and mostly fine, but `UnitTemplate` mixes immutable definition with per-placement fields and carries the same data twice, which is the root of the Leadership bug.

#### Design findings

##### [High] `UnitTemplate` conflates definition and placement, and stores attributes twice
- Evidence: `unitdefs/templates.go:12-35`: `Attributes` (parsed) and `EntityData templates.JSONMonster` (the whole raw DTO, including the same attributes and every other field already copied out); plus `GridRow`, `GridCol`, `IsLeader`, which are not properties of a unit type. Every squad builder copies a template value and mutates it (`setup/gamesetup/initial_squads.go:167-175`, `mind/spawning/squadscreation.go:95-124`, simulator `blueprint.go:38-47`).
- Problem: Callers "configure" a placement by editing a copy of the definition; which fields the consumer honours depends on the construction path (squadcore probable bug 1). `EntityData` exists only to feed `templates.CreateCreatureEntity`.
- Direction: `UnitTemplate` = immutable definition without `EntityData`, `GridRow/GridCol`, `IsLeader`. A small placement value (template pointer, row, col, isLeader, optional leadership override or better a single `LeaderLeadership` rule applied by squadcore) is what `CreateSquadFromTemplate` takes. Pairs with the squadcore construction finding; do them together.
- Effort: M. Risk: medium (touches all squad builders; no tests in unitdefs).

##### [Low] Not the leaf its doc comment claims; grid size hardcoded
- Evidence: package comment `enums.go:1-3` vs imports of `templates`, `unitprogression`, `core/config` (`templates.go:5-8`); width/height `1-3` literals (`templates.go:46-52`) duplicate `squadcore.SquadGridSize` (unitdefs cannot import squadcore).
- Direction: Fix the comment. Leave the literal, or move `SquadGridSize` here since this is the lower package. Not worth more.
- Effort: S. Risk: low.

##### [Low] Growth grades are not validated
- Evidence: `templates.go:72-79` casts JSON strings to `GrowthGrade`; `unitprogression.GrowthChance` returns 0 for anything unknown (`experience.go:28-29`). A typo or missing grade silently yields a stat that never grows. `templates` has no monster validator at all (see the `templates` section).
- Direction: Validate in `CreateUnitTemplates` alongside role and attack type (it already returns errors).
- Effort: S. Risk: low.

#### Dead code
- `RoleError` (`enums.go:15`): no references.
- `Units` initial capacity `len(templates.MonsterTemplates)` is evaluated at package init when that slice is empty (`templates.go:38`); harmless.

#### Leave alone
- Enums, `GetRole`/`GetAttackType` parsing with errors, `FilterUnits` and the four one-line filters (all used), `GetTemplateByUnitType`.

---

### tactical/squads/unitprogression (156 lines, tests: yes)

**Role:** Unit XP, levels and growth-grade stat rolls. **Verdict:** Healthy. Small, pure-data components, RNG injected by the caller, table-driven level-up.

#### Design findings

##### [Low] `XPToNextLevel` is a constant stored per unit
- Evidence: set to 100 in `squadcore/units.go:113-117`, documented "fixed 100" (`components.go:30`), saved per unit (`setup/savesystem/chunks/squad_chunk.go:238`).
- Direction: Either a package constant, or leave if a level curve is planned. Not worth a change on its own.
- Effort: S. Risk: low.

#### Leave alone
- Everything else. `AwardExperience` taking `*rand.Rand` is the right shape for determinism; `processLevelUp`'s table is the right amount of abstraction.

---

### tactical/commander (437 lines, tests: no)

**Role:** Commander entities, the player's commander roster, commander movement on the overworld, and the overworld turn singleton. **Verdict:** Needs attention. The code is small and readable, but it is an overworld package filed under `tactical/`, it keeps a second copy of the overworld clock, and several fields are write-only.

#### Design findings

##### [Medium] Overworld concept in the tactical layer, importing campaign
- Evidence: `commander/movement.go:6` imports `campaign/overworld/core` for `IsTileWalkable`; the package owns `OverworldTurnStateData`. Importers are overworld GUI, campaign tick/raid, setup and save; the only tactical importer is `tactical/powers/spells/system.go:69` for `FindCommanderForSquad`, which drags `campaign/overworld/core` into the spell package's dependency closure. The comment in `campaign/overworld/tick/endturn.go:13-15` says commander was made independent of the overworld, which the import contradicts.
- Problem: `tactical -> campaign` is an inverted layer dependency; the directory misleads about what the package is.
- Direction: Move the package to `campaign/commander`. Move `FindCommanderForSquad` to `roster` (it only reads `SquadRoster` components; make it "find roster owner of squad") so `spells`, `raid/rewards.go:59` and `encounter/resolvers.go:176` no longer need `commander`. Update PROJECT_LAYOUT / PACKAGE_DEPENDENCIES / CLAUDE.md Key Systems as the project rules require.
- Effort: M (mechanical import churn across ~11 packages). Risk: low.

##### [Medium] Second overworld clock and a cached singleton ID
- Evidence: `OverworldTurnStateData.CurrentTurn` (`components.go:36-39`) is incremented only in `tick.EndOverworldTurn` immediately after `AdvanceTick` (`campaign/overworld/tick/endturn.go:17-24`), which is `AdvanceTick`'s only caller, so `CurrentTurn == CurrentTick + 1` always. Reads: two GUI labels (`gui/guioverworld/overworld_action_handler.go:45`, `overworldmode.go:274`). The singleton needs its own entity, tag, and a package-level `cachedOverworldTurnStateID` with stale-ID fallback (`turnstate.go:9-39`). It is not written by any save chunk (no reference under `setup/savesystem`).
- Problem: Two singletons for one counter, plus a global cache with invalidation logic, to render a label.
- Direction: Delete `OverworldTurnStateData`, its component/tag, `CreateOverworldTurnState`, `GetOverworldTurnState` and the cache; the GUI shows `core.GetCurrentTick()+1`. `StartNewTurn` stays.
- Effort: S. Risk: low.

##### [Low] Write-only state and unused plumbing
- Evidence: `CommanderData.IsActive` (set true, saved, never read); `CommanderActionStateData.HasActed` (only reset to false, never set true or read); `OverworldTurnStateData.TurnActive` (never read); `CommanderMovementSystem.posSystem` field and constructor parameter (`movement.go:16,20-25`) never used, while `GetCommanderAt` uses `common.GlobalPositionSystem` directly (`queries.go:14`); `CommanderActionTag` and the three `WorldTags[...]` entries (`init.go:32-34`) never read.
- Direction: Delete. If `CommanderData` is left with only `Name`, consider using the existing `common.NameComponent` and keeping `CommanderComponent` as a marker.
- Effort: S. Risk: low.

##### [Low] `CommanderMovementSystem` vs `CombatMovementSystem`
- Evidence: `commander/movement.go:29-72` is explicitly "modeled after" `tactical/combat/combatcore/combatmovementsystem.go:58-110`: same check-remaining / Chebyshev cost / `CanMoveTo` / move / decrement / `HasMoved` sequence. The range enumeration is already shared (`common.GridMovement`).
- Direction: Leave as is. The remaining overlap is ~30 lines over different state components; a shared abstraction would cost more than it saves. The struct could become two functions (it holds only the manager), but that is optional.

#### Mechanics notes
- Commander and squad movement are both "jump to any enterable tile within Chebyshev radius": `GridMovement.ValidTilesInRange` (`core/common/grid_movement.go:18-36`) is a bounding-box filter, not a flood fill (its comment says flood-fill), and `MoveCommander` charges straight-line distance. Unwalkable terrain blocks destinations, never paths. If that is intended, fix the comment; if not, it is one place to change for both movers.
- `CreateCommander` builds a throwaway `Attributes{MovementSpeed: n}` so commanders carry a full combat stat block (all zeros, `CurrentHealth` 0) just to hold one number (`system.go:27-43`). Works, but anything that treats "has Attributes with CurrentHealth <= 0" as dead will see commanders as dead.

#### Dead code
- Listed under "Write-only state" above. `SeedStarters`, `GetCommanderData`, `GetAllCommanders`, `GetCommanderCount`, `AddCommanderToPlayerRoster`, `CommanderView` are all used.

#### Leave alone
- `CommanderRosterData` and its helpers; `CreateCommander` (single construction point, uses `RegisterEntityPosition`); `StartNewTurn`; `GetCommanderAt` via the position system; action state stored on the commander entity.

---

## 13. Tactical combat

Packages reviewed: `combattypes`, `combatstate`, `combatmath`, `combatcore`, `battlelog`, `combatdisposal`, `combatservices`. Every non-test file was read in full. Callers outside the slice were read only to confirm usage.

### Layer overview

#### Dependency shape (actual imports)

```
combattypes   -> core/common                               (leaf; also imported by perks, artifacts, powercore, gui)
combatstate   -> core, squadcore                           (imported by 18 packages)
combatmath    -> combattypes, squadcore, unitdefs, config
battlelog     -> combattypes, combatmath, squadcore
combatdisposal-> combatstate, squadcore
combatcore    -> combatstate, combatmath, combattypes, battlelog, combatdisposal, squadcore, powers/effects
combatservices-> combatcore, combatstate, combattypes, battlelog, powers/{artifacts,perks,effects,powercore}
```

The graph is acyclic and the layering direction is sensible. The problems are in what sits inside each box, not in the arrows.

#### Where combat rules actually live

A single squad-vs-squad attack is decided by code in five packages. This is the main maintainability cost of the slice.

| Rule | Location |
|---|---|
| Can this squad attack (action available, faction, range) | `combatcore/combatactionsystem.go:302` |
| Which units take part in the attack | `squadcore/squadqueries.go:16` (`CanUnitAttack`), reached via `combatcore/combatactionsystem.go:100` |
| Which units take part in a counterattack | `combatcore/combatactionsystem.go:248` (`getUnitsInRange`, a second copy of the same eligibility rule) |
| Which units a unit targets | `combatmath/combattargeting.go:12` (real), `squadcore/squadqueries.go:443` (preview copy) |
| Hit, dodge, crit, resistance, cover | `combatmath/combatcalculation.go`, `combatmath/combatcover.go` |
| Perk hook order, redirect, death override | `combatcore/combatprocessing.go:37` |
| Counterattack rules (survival, penalties) | `combatcore/combatactionsystem.go:118`, constants in `combatmath/balanceconfig.go` |
| Would the squad survive | `squadcore/squadqueries.go:183` |
| HP mutation | `combatmath/combatcalculation.go:219,231` |
| Action economy, bonus attack, zone of control | `combatstate/combatqueries.go:153-252` |
| Totals on the result | `battlelog/combatlogging.go:86` |
| Dead unit and squad removal | `squadcore/squadcreation.go:422,447`, `combatdisposal/squad_removal.go:27` |

#### Is seven packages the right granularity?

No. Three are justified by the import graph, two are justified by size and role, and two are not pulling their weight. Recommended target is five packages, with a real boundary between them:

1. **Keep `combattypes`.** It is the leaf that lets `perks`/`artifacts`/`powercore` and `combatcore` share `DamageModifiers`, `CombatResult` and `PerkDispatcher` without a cycle. Move `UnitIdentity` out (GUI-only type).
2. **Keep `combatstate`** as the ECS data and query layer. Simplify it (see its section): it carries the most accidental machinery in the slice.
3. **Merge `combatmath` into `combatcore`.** The "pure math vs systems" split is not honoured in either direction: `combatmath` mutates HP (`ApplyRecordedDamage`, `ApplyRecordedHealing`) and loads config from disk, while `combatcore` holds rules (`getUnitsInRange`, counterattack survival, `canSquadAttackWithReason`). `processAttack` (core) and `CalculateDamage` (math) are one pipeline split across the boundary with the perk dispatcher passed through it. Every importer of `combatmath` except `battlelog` and `setup/gamesetup` already imports `combatcore`. The merge needs two tiny helpers moved first (`SumDamageMap`, `CalculateSquadStatus`, both only used by `battlelog`). If the merge is not wanted, the cheaper alternative is to make the split real: all rules and formulas in `combatmath` with no mutation, all mutation and orchestration in `combatcore`.
4. **Delete `combatdisposal`.** 41 lines wrapping `squadcore.DisposeSquadAndUnits`; its two extra steps are redundant (see its section).
5. **Keep `battlelog`, but make it a leaf.** Move `combatlogging.go` (91 lines, only called from `combatcore`) into `combatcore`; what remains (recorder, summary, export) then depends only on `combattypes`.
6. **Keep `combatservices`** as the facade and wiring point, trimmed.

Net: 7 packages to 5, no new files, fewer files overall.

#### Cross-cutting themes

- **Two parallel records of one hit.** `AttackEvent` (per hit, computed against stale HP) and `DamageRecord` (accumulated) are reconciled by hand in three places and disagree. This is the source of every correctness bug found in the slice.
- **Cached handles to ECS state held in Go structs.** `CombatQueryCache` (three live instances plus one leaked per combat), `TurnManager.turnStateEntityID`, `BattleRecorder.currentRound`. None hold data that cannot be queried.
- **Layers of delegation left over from earlier refactors.** `CombatService` has 16 one-line pass-throughs, five of them with no callers.
- **The turn loop has no headless home.** Ending a turn, checking victory and running the AI are sequenced in `gui/guicombat/combat_turn_flow.go`; the combat simulator re-implements its own loop.

---

### tactical/combat/combattypes (278 lines, tests: no)

**Role:** Leaf package of shared combat DTOs (`CombatResult`, `AttackEvent`, `DamageModifiers`) plus the `PerkDispatcher` interface. **Verdict:** Healthy. The package is the right size and in the right place; it needs pruning of fields nobody sets.

#### Design findings

##### [Medium] `AttackEvent` duplicates and contradicts `DamageRecord`
- Evidence: `combattypes/combattypes.go:8-32` (`FinalDamage`, `DefenderHPBefore`, `DefenderHPAfter`, `WasKilled`) versus `combattypes/combattypes.go:212-217` (`DamageByUnit`, `UnitsKilled`). Written by different code: `combatmath/combatcalculation.go:55-59` and `combatmath/combatcalculation.go:193-215`.
- Problem: the same facts (how much damage, did the unit die) are stored twice with different derivations. See the High finding under `combatcore` for the concrete drift.
- Direction: keep both types, but derive the event's HP and kill fields from the `DamageRecord` in one place after redirect and death override have run. `CombatLog.TotalDamage`/`UnitsKilled` (`combattypes.go:128-129`) are likewise copies of `DamageRecord` values; keep one.
- Effort: S (as part of the `combatcore` fix). Risk: low.

##### [Low] Three structs describe a battle outcome
- Evidence: `combatservices/combat_exit_controller.go:14` (`VictoryCheckResult`), `battlelog/battle_recorder.go:164` (`VictoryInfo`, comment says it exists to avoid a circular import), `mind/combatlifecycle` `EncounterOutcome`. `gui/guicombat/combatmode.go:139-155` copies fields between all three by hand.
- Direction: define the outcome once in `combattypes` (both `battlelog` and `combatservices` already import it) and delete `VictoryInfo`. `EncounterOutcome` in `mind/combatlifecycle` is the third copy; see that section.
- Effort: S. Risk: low.

##### [Low] `UnitIdentity` is a GUI list-row type living in combat types
- Evidence: `combattypes/combattypes.go:150-159`; only users are `gui/guisquads/squadlists.go:91,104` and `gui/guisquads/squadeditor_roster.go:122-137`.
- Direction: move to `gui/guisquads`. This also removes the only reason the squad editor imports a combat package.
- Effort: S. Risk: low.

#### Dead code
- `DamageModifiers.SkipCounter` (`combattypes.go:170`): read at `combatcore/combatactionsystem.go:128`, never set anywhere (repo-wide grep including tests). `CounterMod`'s bool return does the job.
- `DamageModifiers.CoverBonus` (`combattypes.go:169`): read at `combatmath/combatcalculation.go:173-178`, never set anywhere. Cover perks use the `CoverMod` hook instead (`perks/behaviors_stateless.go:48`). Two mechanisms for one job, one unused.
- `HitTypeHeal` (`combattypes.go:60`): never assigned to an event; heals use `HealEvent`. Only appears in switch arms.
- `CombatLog.DefenderStatus`, `SquadStatus` (`combattypes.go:130,144`): written once, read nowhere in Go code; only reaches the JSON export, and the value is wrong (see `battlelog` bugs).

#### Leave alone
- `PerkDispatcher` plus `NoopPerkDispatcher` (`perk_callbacks.go`). One real implementation, but the interface is what breaks the `perks -> combat -> perks` cycle, and the null object removes nil checks. This is a justified abstraction.
- `CombatResult{Status, Damage, Log}` grouping, `NewAttackModifiers`/`NewCounterattackModifiers`.

---

### tactical/combat/combatstate (601 lines, tests: no)

**Role:** Combat ECS components (factions, faction membership, turn state, action state) and the queries over them. **Verdict:** Needs attention. The data model is small and sound, but it is wrapped in a cache object and a manager object that add indirection, leak ECS views, and are instantiated three to four times.

#### Design findings

##### [High] `ActionStateData` is a separate entity found by linear scan through an injected cache
- Evidence: `combatstate/combatqueries.go:141-151` creates a free-standing entity per squad; `combatstate/combatqueriescache.go:30-49` finds it by scanning a view and comparing `SquadID`. By contrast faction membership is a component on the squad entity (`combatfactionmanager.go:69`, with the comment "ECS-idiomatic approach"). `CombatQueryCache` holds exactly one field, `ActionStateView` (`combatqueriescache.go:14-16`); its faction methods already use the package-level `factionView` (`combatqueriescache.go:56-76`).
- Instances of the cache: `combatservices/combat_service.go:62`, `gui/framework/guiqueries.go:27`, and a new one on every combat start at `mind/combatlifecycle/enrollment.go:43`. Each calls `World.CreateView`. The ECS library has no way to remove a view (`bytearena/ecs@v1.0.0/ecs.go:99-116`, views are only appended) and every component add/remove walks all views (`ecs.go:241,264`).
- Problem: (1) one ECS view is leaked per combat for the life of the process, and every later entity change pays for it; (2) the cache pointer is threaded through `CanSquadAct`, `CanSquadMove`, `MarkSquadAsActed`, `MarkSquadAsMoved`, `DecrementMovementRemaining`, `CombatFactionManager`, `TurnManager`, `CombatActionSystem`, `CombatMovementSystem`, `CombatExitController`, `MoveSquadCommand`, the artifact `BehaviorContext` and the AI, purely to reach one view; (3) those same functions also take a `manager` they never use; (4) action-state entities for destroyed squads are orphaned until teardown; (5) nothing prevents two action states for one squad, and that is what happens today (see bugs).
- Direction: make `ActionStateData` a component on the squad entity, added in `AddSquadToFaction` (or the enrollment helper) and removed in `RemoveCombatMembership`. Lookup becomes `GetComponentTypeByID`. `CombatQueryCache` then has no state and can be deleted, along with the cache parameter on every function above; `FindFactionDataByID` becomes a package function. If moving the component is too large a first step, the minimum is: make the action-state view package-level like `factionView`, and stop constructing caches outside `combatstate`.
- Effort: M (mechanical, touches ~12 files across tactical, mind, gui). Risk: medium (no tests in this package; `combatcore` tests cover the main paths).

##### [Medium] `CombatFactionManager` is a stateless struct with three live copies
- Evidence: `combatstate/combatfactionmanager.go:11-21` (fields: manager, cache). Constructed at `combat_service.go:73`, `gui/framework/guiqueries.go:31`, `mind/combatlifecycle/enrollment.go:44`.
- Problem: it holds no state, so the type only forces callers to construct or carry it. It also sits in a "state/query" package while creating entities and moving squads.
- Direction: turn `CreateFactionWithPlayer`, `AddSquadToFaction`, `CreateStandardFactions` into package functions taking `*EntityManager`; delete the type. `CombatService.FactionManager`, `GUIQueries.factionManager` and the return value of `combatlifecycle.CreateFactionPair` go with it.
- Effort: S. Risk: low.

##### [Medium] `FactionData` carries three vestigial concepts
- Evidence: `combatstate/combatcomponents.go:30-39`.
  - `Mana`/`MaxMana`: set to 100/100 at `combatfactionmanager.go:43-44`, never written again (repo-wide grep), but displayed in the combat HUD via `gui/framework/guiqueries.go:90` and `gui/guicombat/combat_mode_init.go:211`. Real mana is per squad (`spells` `ManaData`). The HUD shows a number that can never change next to the real one (`combat_mode_init.go:218`).
  - `IsPlayerControlled`, `PlayerID`, `PlayerName`: `IsPlayerControlled` is derived from `PlayerID > 0` (`combatfactionmanager.go:38`), and `PlayerName` is always `"Player 1"` or empty (`combatfactionmanager.go:28,95-96`). Three fields for one bit, in support of multiplayer that does not exist.
  - `FactionID`: the entity's own ID stored inside its component; every lookup scans the view comparing it (`combatqueriescache.go:56-76`) instead of `manager.FindEntityByID(factionID)`, which `combat_service.go:297` already does for the same data.
- Direction: delete `Mana`, `MaxMana`, `GetFactionMana`, and the HUD line. Collapse the player fields to `IsPlayerControlled`. Replace view scans with direct ID lookup.
- Effort: S. Risk: low.

##### [Medium] `HasMoved` and `MovementRemaining` are two encodings of "can still move", mutated from five packages
- Evidence: `combatstate/combatcomponents.go:48-54`. `CanSquadMove` uses `MovementRemaining > 0` (`combatqueries.go:163-169`); the AI ends a squad's activation on `HasMoved && HasActed` (`mind/ai/ai_controller.go:111`) and only generates moves when `!HasMoved` (`mind/ai/action_evaluator.go:117`); the GUI displays `HasMoved`. Direct field writes from outside: `tactical/powers/artifacts/context.go:36-47`, `artifacts/behaviors.go:134,310`, `tactical/squads/squadcommands/move_squad_command.go:136-138`, `mind/ai/action_evaluator.go:94-95`, `combatcore/turnmanager.go:87-93`.
- Problem: the player may move in several steps until `MovementRemaining` hits 0, while the AI treats one step as its whole move. Any writer that updates one field and not the other produces a state the readers interpret differently. The mutator functions in this package (`MarkSquadAsMoved` etc.) are bypassed by most writers.
- Direction: pick one meaning. Either derive "has moved" from `MovementRemaining < speed` and drop the bool, or keep the bool and document it as display-only. Add the two or three missing mutators (`LockSquad`, `ResetActions(speed)`, `RestoreMovement(remaining, moved)`) here and have artifacts, the AI wait action, and undo call them instead of writing fields.
- Effort: S-M. Risk: medium (AI behaviour may shift if the meaning changes).

##### [Low] Component, data and tag names are crossed
- Evidence: `combatcomponents.go:12-20,101-104`. `CombatFactionComponent` stores `*FactionData` and its tag is `FactionTag`; `FactionMembershipComponent` stores `*CombatFactionData` and its tag is `CombatFactionTag`. `IsSquad` (`combatqueries.go:125`) means "is enrolled in combat".
- Direction: rename to one scheme (`FactionComponent`/`FactionData`/`FactionTag`, `FactionMembershipComponent`/`FactionMembershipData`/`FactionMembershipTag`, `IsCombatSquad`).
- Effort: S. Risk: low.

##### [Low] A logger-injection mechanism exists for one unreachable log line
- Evidence: `combatstate/logging.go` (whole file), wired at `combatservices/power_orchestrator.go:61-63`, called once at `combatqueries.go:180`.
- Problem: the call is inside the `BonusAttackActive` branch, which is dead (see Dead code). If it did fire, `classifyLogPrefix("bonus_attack")` (`combatservices/combat_power_dispatch.go:39-46`) would label it `[PERK]`, not `[GEAR]`.
- Direction: delete `logging.go`, `SetGearLogger`, and the wiring.
- Effort: S. Risk: low.

#### Mechanics notes
- Zone of control is defined here (`combatqueries.go:213-252`) but only applied when listing tiles, not when moving. See the movement finding under `combatcore`.

#### Probable bugs
- **Every squad gets two action-state entities** (confidence: high that it happens; impact currently low). Enrollment creates one (`mind/combatlifecycle/enrollment.go:31`, reached from `mind/encounter/encounter_setup.go:115,121` and `campaign/raid/raidencounter.go:58,71`), then `CombatMode.Enter` (`gui/guicombat/combatmode.go:114`) calls `CombatService.InitializeCombat`, which calls `TurnManager.InitializeCombat`, which creates another for every squad (`combatcore/turnmanager.go:62-67`). Lookups return the first match in view order, and nothing removes action states mid-combat, so the same one is always found. It becomes a real bug the moment anything disposes an action state mid-combat, because the library's view removal swaps elements (`ecs.go:59-70`).

#### Dead code
- `ActionStateData.BonusAttackActive` (`combatcomponents.go:53`): never set to true outside tests. Twin Strike resets `HasActed` directly (`artifacts/behaviors.go:191`). Removes the branch at `combatqueries.go:178-182` and the reset at `turnmanager.go:89`.
- `TurnStateData.CombatActive` (`combatcomponents.go:45`): set true at creation; only `TurnManager.EndCombat` clears it and `EndCombat` has no callers. `IsCombatActive` is equivalent to "a turn-state entity exists".
- `CombatFactionTag` and the four `manager.WorldTags[...]` registrations (`combatcomponents.go:20,104-109`): no readers.
- `CombatFactionManager.GetFactionName` (`combatfactionmanager.go:110`): no callers. `CreateCombatFaction` (`:23`) is used only by the simulator and tests.
- `InitCombatComponents`/`InitCombatTags` are exported but only called from the package's own `init`.

#### Performance
- `GetSquadsForFaction` (`combatqueries.go:82-93`) iterates a view of every squad in the world (`combatSquadView` on `squadcore.SquadTag`, `combatcomponents.go:71`), including roster and garrison squads not in the battle. It is called per faction by the AI, victory check, GUI and teardown. A view on the faction-membership tag (which already exists, unused) would iterate only combat squads.
- `FindActionStateBySquadID` is an O(squads) scan called on every action check. Disappears with the High finding.

#### Leave alone
- The four-component data model itself, `GetSquadFaction`, `GetSquadMapPosition`, `GetSquadAtPosition`, `IsInZoneOfControl`, `GetFactionsForEncounter`. Pure-data components with no methods; EntityID-only references. This follows the project's ECS rules.
- `ShuffleFactionOrder` uses the shared RNG.

---

### tactical/combat/combatmath (744 lines, tests: no direct; exercised by combatcore tests)

**Role:** Damage formula, cover, target selection, heal calculation, and the combat balance config. **Verdict:** Needs attention. The formulas are readable and well factored, but the package name promises purity it does not have, and its config loader fails silently.

#### Design findings

##### [Medium] Combat balance config bypasses the gamedata conventions and fails silently
- Evidence: `combatmath/balanceconfig.go:31-45` reads and parses JSON itself, prints a warning and returns on any error, and `validateCombatBalance` (`:47-57`) only prints. The sibling loaders return errors and are fatal at boot (`setup/gamesetup/bootstrap.go:40-45`); this one is called bare at `bootstrap.go:49`. `CLAUDE.md` states validators return `error` and `LoadGameData` is the single fatal point.
- Problem: if the file is missing or malformed, `CombatBalance` stays zero-valued: counterattack damage multiplier 0 (every counter hit clamps to 1 damage at `combatcalculation.go:146-149`) and no hit penalty. Crit has an inline fallback of 1.5 (`combatcalculation.go:131-134`) while counterattack has none, so the two knobs behave differently on failure.
- Concrete consequence today: the combat simulator never calls the loader (`tools/combat_analysis/combat_simulator/bootstrap.go:14-26`; repo-wide grep shows `bootstrap.go:49` in gamesetup as the only caller), so all simulated battles used for balance analysis run with 1-damage counterattacks and no counter hit penalty. `combatcore` tests run with the same zero config.
- Direction: load through the `templates` `Loader[T]` path (or at minimum return `error` and make boot fatal), validate into errors, delete the inline 1.5 fallback, and have the simulator bootstrap load it. The simulator itself was out of scope for this review.
- Effort: S. Risk: low (simulator results will change, which is the point).

##### [Medium] A "math" package that mutates game state
- Evidence: `combatcalculation.go:219-243` (`ApplyRecordedDamage`, `ApplyRecordedHealing` write `CurrentHealth`); `combatcalculation.go:193-215` (`RecordDamageToUnit` mutates the result and re-derives the kill list).
- Problem: readers looking for "where does HP change in an attack" have to know it is in the calculation package, called from `combatcore/combatactionsystem.go:182-183`. Spell damage mutates HP by a third route (`tactical/powers/spells/system.go:181-184`).
- Direction: resolved by merging into `combatcore` (slice-level recommendation), or by moving the three functions to `combatcore`.
- Effort: S. Risk: low.

##### [Low] Target-selection rule exists twice
- Evidence: `combatmath/combattargeting.go:27-40` (used to resolve attacks) and `squadcore/squadqueries.go:443-465` `computeGenericTargetCells` (used for the attack-pattern preview). Both switch on the same five attack types.
- Problem: the preview ignores pierce-through (`combattargeting.go:44-72`) and the ranged fallback to lowest armour (`:88`), so it can show cells the attack will not hit. A new attack type must be added in both.
- Direction: express each attack type's base cell pattern once (a function from attack type and attacker grid position to cells), and have both the resolver and the preview start from it. The same finding appears under `tactical/squads/squadcore`.
- Effort: S-M. Risk: low.

#### Mechanics notes
- **Resolution is simultaneous within a phase, sequential between phases, and the code is not consistent about it.** All main-phase hits are recorded against pre-attack HP and applied at the end. Consequence: units already "dead" from earlier hits in the same attack are still targeted, so overkill damage is wasted. Counterattackers, by contrast, are filtered by predicted survival (`combatcore/combatactionsystem.go:135`). Worth deciding deliberately.
- **Healers act during counterattacks** (`combatactionsystem.go:139-141`), so attacking a squad with a healer gives it a free heal, but only for wounds that existed before the attack: heal targets and amounts are computed from pre-attack HP (`combattargeting.go:216`, `combatcalculation.go:295-298`). A unit at full HP that just took 8 damage is not a heal target.
- **Healers must be in attack range to heal their own squad.** Both phases gate on `Range >= squadDistance` (`squadcore/squadqueries.go:32`, `combatactionsystem.go:281`) before checking whether the unit is a healer.
- **Ranged attacks have two unrelated behaviours**: all units in the attacker's own row index, or, if that row is empty, exactly one unit chosen by lowest armour (`combattargeting.go:75-89`).
- Cover has two caps at 1.0 (`combatcover.go:140`, `combatcalculation.go:175`), so 100% cover still deals 1 damage because of the floor at `combatcalculation.go:184`.

#### Dead code
- `selectMeleeRowTargets`' `attackerID` parameter is unused (`combattargeting.go:44`).
- `CalculateSquadStatus` (`combatcalculation.go:255`) feeds only the unread `DefenderStatus`.
- `GetCoverProvidersFor` is exported but used only inside the package and in tests.

#### Leave alone
- `CalculateDamage` and its three helpers (`combatcalculation.go:21-190`). Clear staging, one roll helper, shared RNG, modifiers passed by value. Do not abstract further.
- `getUnitsInLine` with the row/column wrappers.
- `CalculateCoverBreakdown` is long but linear; leave it.

---

### tactical/combat/combatcore (881 lines, tests: yes, 2,472 lines)

**Role:** The three combat systems: attack execution (with perk hook pipeline), squad movement, and the turn manager. **Verdict:** Needs attention. The orchestration reads well, but kill bookkeeping is wrong in common situations and movement legality is only partly enforced at the point of execution.

#### Design findings

##### [High] Per-hit events are computed against stale HP, so kill detection disagrees with the damage record
- Evidence: `combatmath/combatcalculation.go:35-37,55-59` computes `DefenderHPBefore/After` and `WasKilled` from `CurrentHealth`, which does not change until `combatactionsystem.go:182`. Accumulated damage lives separately in `DamageByUnit` (`combatcalculation.go:195`). `processAttack` then patches the two records by hand in `applyDamageWithRedirect` (`combatprocessing.go:116-128`) and `applyDeathOverride` (`combatprocessing.go:134-168`). Survival is predicted from damage only (`squadcore/squadqueries.go:183-205`), while healing is applied afterwards (`combatactionsystem.go:183`).
- Problem: one fact (did this unit die) has four derivations: `event.WasKilled`, `DamageRecord.UnitsKilled`, `WouldSquadSurvive`, and actual HP after damage and healing. They diverge; see Probable bugs 1 to 4. Perks, the battle log, the balance tools (`tools/combat_analysis/combat_balance/aggregator.go:113`) and squad removal each consume a different one.
- Direction: give the pipeline one notion of "pending HP" for the engagement: current HP minus recorded damage (and, if healing is meant to offset, plus recorded healing). Compute `HPBefore`, `HPAfter` and `WasKilled` from it in `processAttack` after redirect, and run death override and post-damage hooks off that single value. Decide the healing rule explicitly (either healing cannot save a unit that took lethal damage, or survival prediction includes it) and make `WouldSquadSurvive`, `UnitsKilled` and disposal agree.
- Effort: M. Risk: medium (combat outcomes change in the affected cases; test suite is strong here).

##### [Medium] Movement legality is split between "list valid tiles" and "execute move"
- Evidence: zone of control is applied only in `GetValidMovementTiles` (`combatmovementsystem.go:126`) and in a second copy of that function in `mind/ai/action_evaluator.go:160-171`. `MoveSquad` (`combatmovementsystem.go:58-110`) checks only movement points and occupancy. The GUI pre-filters clicks against the tile list (`gui/guicombat/combatinput/handler.go:250-262`).
- Problem: the rule holds only because every current caller pre-filters. `MoveSquadCommand`, the AI, or any new caller can move a squad three tiles out of an enemy's zone of control. The AI's tile listing is a line-for-line duplicate.
- Direction: have `MoveSquad` validate against the same rule (compute the effective range and reject moves beyond it), and have the AI call `GetValidMovementTiles` instead of rebuilding it.
- Effort: S. Risk: low.

##### [Medium] "Which units can attack" is implemented twice, and the GUI builds a throwaway system to ask
- Evidence: `executeMainAttack` uses `squadcore.CanUnitAttack` (`combatactionsystem.go:100`); `getUnitsInRange` (`combatactionsystem.go:248-287`) re-implements alive + `CanAct` + range, and backs both `getCounterattackingUnits` and the exported `GetAttackingUnits`. The only caller of `GetAttackingUnits` is `gui/guicombat/combatanimation/colors.go:46-47`, which constructs a new `CombatActionSystem` to call it. `battlelog.SnapshotAttackingUnits` (`battlelog/combatlogging.go:30`) is a third use of the rule.
- Problem: two definitions of attack eligibility that must stay in step; the animation colours can disagree with what the attack actually did.
- Direction: one package-level function `UnitsAbleToAttack(squadID, distance, requireCanAct)` built on `CanUnitAttack`; use it in all three places; the GUI calls it directly (or better, reads `result.Log.AttackingUnits`, which already records who attacked).
- Effort: S. Risk: low.

##### [Medium] The attack system reads rule inputs from the log object, and the log package writes result totals
- Evidence: squad distance used for eligibility comes from `combatLog.SquadDistance` (`combatactionsystem.go:73-74,82,100`); `result.Damage.TotalDamage` is set inside `battlelog.FinalizeCombatLog` (`battlelog/combatlogging.go:87`).
- Problem: display data and execution data are entangled, so the attack cannot run without building the log, and a logging helper owns a field of the execution record.
- Direction: compute distance once in `ExecuteAttackAction` and pass it to both the rules and the log; set `TotalDamage` in `combatcore`. Move `battlelog/combatlogging.go` into `combatcore` (it has no other callers). Record engagements by subscribing the recorder to the existing attack-complete pipeline in `combatservices/power_orchestrator.go:95-105` instead of injecting it into the action system (`combatactionsystem.go:19,37-39,189-191`); `combatcore` then no longer imports `battlelog`.
- Effort: S. Risk: low.

##### [Low] `TurnManager` owns a second `CombatMovementSystem` for a one-line delegation
- Evidence: `turnmanager.go:17,30,92`; `combatmovementsystem.go:38-40` just calls `squadcore.GetSquadMovementSpeedOrDefault`. `CombatService` constructs its own movement system at `combat_service.go:65`.
- Direction: call `squadcore.GetSquadMovementSpeedOrDefault` directly; delete the field and the wrapper method.
- Effort: S. Risk: low.

##### [Low] Thin wrappers and dead branches
- `ProcessAttackOnTargets`/`ProcessCounterattackOnTargets` (`combatprocessing.go:180-194`) wrap `processAttack` to supply modifiers; `counterattackDamageMultiplier`/`counterattackHitPenalty` (`:13-20`) wrap two config reads.
- `CanMoveTo` (`combatmovementsystem.go:42-56`): both branches after the occupancy check return false, and the `squadID` parameter is unused. It is `occupyingID == 0`.
- `canSquadAttackWithReason` recomputes distance by hand (`combatactionsystem.go:309-332`) while the rest uses `squadcore.GetSquadDistance`.
- Direction: inline or simplify when touching these files. Not worth a dedicated pass.

#### Mechanics notes
- `TurnManager.ResetSquadActions` also ticks effect durations (`turnmanager.go:95-97`). Artifacts and perks hook turn start through the pipeline (`power_orchestrator.go:88-93`); effects are hard-wired. It works and `effects` is a lower layer, so leave it, but it is the one place `combatcore` imports `powers`.
- Eliminated factions still receive turns (`turnmanager.go:137-163`). Harmless with two factions.

#### Probable bugs
1. **Kills from accumulated hits are not seen as kills** (confidence: high). Two units each deal 6 to a 10 HP unit: neither event has `WasKilled` (each is computed as 10 - 6), but `DamageByUnit` is 12 and the unit dies. Consequences: `applyDeathOverride` returns early (`combatprocessing.go:137`) so the Resolute perk never triggers; post-damage hooks get `wasKill=false` (`:57`); per-unit kill counts in the battle summary and balance aggregator are 0. Multiple units hitting the same target is the normal case for row attacks.
2. **A unit already lethally hit is "killed" again by each later hit** (confidence: high). If the first hit is lethal on its own, every later hit on that unit in the same attack also reports `WasKilled=true`, firing kill hooks and incrementing summary kill counts once per hit.
3. **Damage redirect leaves the event stale** (confidence: medium-high). After Guardian Protocol reduces the damage (`combatprocessing.go:120-123`), `event.FinalDamage`, `DefenderHPAfter` and `WasKilled` keep the pre-redirect values. A hit that was lethal before redirect still calls `DeathOverride` (`:137-145`), which can spend the once-per-battle Resolute charge on a unit that was not going to die. The redirect target gets no event and no death-override check.
4. **Healing can revive a unit recorded as killed, or be ignored when it would save the squad** (confidence: medium-high on the code path; whether intended is unclear). Heal targets are not filtered by predicted death. Example: a pre-wounded defender takes lethal damage in the main phase, a surviving healer in its squad heals it in the counter phase, damage then healing are applied (`combatactionsystem.go:182-183`), and it ends above 0 HP. It is in `UnitsKilled`, perks were told it died, yet `DisposeDeadUnitsInSquad` keeps it. In the opposite direction, `attackerDestroyed`/`defenderDestroyed` are computed from damage alone (`:175-176`) before healing, so a squad whose last unit would have been healed above 0 is disposed anyway.
5. **`CounterMod` runs even when no counterattack will happen** (confidence: medium; depends on perk side effects). `combatactionsystem.go:126` calls the hook before checking `defenderWouldSurvive` at `:128`. Any perk that tracks state in `CounterMod` is affected. Checked afterwards: the only two implementers, Riposte and Stalwart, only change modifiers and write a log line, so today the effect is a spurious log entry.
6. **Combat movement has no terrain or map-bounds check** (confidence: medium; not verified in play). `CanMoveTo` consults only the position system (`combatmovementsystem.go:42-56`), which holds entities, not tiles; the comment "Occupied by terrain/obstacle" assumes otherwise. `GridMovement.ValidTilesInRange` (`core/common/grid_movement.go:23-34`) generates coordinates without bounds, so tiles off the map edge pass. Commander movement does check walkability (`tactical/commander/movement.go:77`). The AI picks from this tile list by score.
7. `TurnManager.InitializeCombat` indexes `turnOrder[0]` with no empty check (`turnmanager.go:70`); the GUI guards with `len(factionIDs) > 0`, the simulator always passes two (confidence: high, low impact).

#### Dead code
- `TurnManager.EndCombat` (`turnmanager.go:173-185`): no callers anywhere, including tests.
- `GetAttackingUnits`: one caller, via a throwaway system (see finding above).

#### Performance
- `squadcore.GetUnitIDsInSquad` scans every squad member in the world and is called many times per attack (targeting per cell, cover per defender, survival checks). `getUnitsInLine` calls `GetUnitIDsAtGridPosition` three times per line, each a full member scan (`combattargeting.go:117-123`). Fine at current sizes; it is the first thing to look at if AI turns get slow, since the AI evaluates many candidate attacks. The fix would live in `squadcore`.

#### Leave alone
- `processAttack` and its eight named steps (`combatprocessing.go:22-63`). This is the best-structured function in the slice; the step list in the comment matches the code. Fix the data it works on, not its shape.
- `ExecuteAttackAction`'s three-phase structure (main, counter, apply) and the record-then-apply approach. The idea is right; it needs the single pending-HP source.
- The single-callback setters on each system feeding `powercore.PowerPipeline`.
- `TurnManager.EndTurn`/`GetCurrentFaction`/`GetCurrentRound`.

---

### tactical/combat/battlelog (760 lines, tests: no)

**Role:** Builds the per-attack `CombatLog` snapshot, and records, summarizes and exports whole battles to JSON for the analysis tools. **Verdict:** Needs attention, mainly placement: 91 lines are part of attack execution and belong in `combatcore`; the remaining 669 are a self-contained export feature that should not be on the attack hot path's import list.

#### Design findings

##### [Medium] Two unrelated responsibilities in one package
- Evidence: `combatlogging.go` (called only from `combatcore/combatactionsystem.go:73,82,83,171`; imports `squadcore` and `combatmath`) versus `battle_recorder.go`, `battle_summary.go`, `battle_export.go` (import only `combattypes`).
- Direction: move `combatlogging.go` into `combatcore`. `battlelog` becomes a leaf that depends only on `combattypes`, and `combatcore` stops importing it (with the recorder subscribed via the pipeline as described under `combatcore`).
- Effort: S. Risk: low.

##### [Medium] The recorder's lifecycle and round counter are driven by the GUI
- Evidence: `gui/guicombat/combatmode.go:105-109` (enable, start, set round 1), `gui/guicombat/combat_turn_flow.go:212-214` (set round each turn), `combatmode.go:150-162` (finalize, export, clear, disable). `BattleRecorder.currentRound` (`battle_recorder.go:102`) duplicates `TurnStateData.CurrentRound`; its comment says it is "updated by CombatService", which is not true. The simulator repeats the same sequence (`tools/combat_analysis/combat_simulator/combat_runner.go:51-58,102-108`).
- Problem: a second source of truth for the round that is only correct if the presentation layer remembers to push it; recording start/stop is duplicated by every driver.
- Direction: pass the round into `RecordEngagement` from whoever subscribes it (the turn manager is available in `WirePipeline`), delete `SetCurrentRound`. Give `CombatService` `StartRecording()`/`FinishRecording(outcome)` so the GUI and simulator each make one call. Collapse `enabled` + `Start` + `Clear` into start/finish.
- Effort: S. Risk: low.

##### [Low] `battle_summary.go` round-trips an enum through strings
- Evidence: `formatOutcome` then `parseOutcome` (`battle_summary.go:244-266`) to compare ranks at `:105`.
- Direction: keep the `HitType` on a local struct while aggregating and stringify once at the end; `parseOutcome` goes away. Only worth doing if this file is touched.
- Effort: S. Risk: low.

#### Probable bugs
- **`DefenderStatus` is the pre-attack status** (confidence: high; impact low). `FinalizeCombatLog` runs before damage is applied (`combatcore/combatactionsystem.go:171` versus `:182`), and `CalculateSquadStatus` reads current HP. The field has no Go reader, so it only affects exported JSON. Delete it rather than fix it.
- `log.TotalHealing` and the summary's healing totals count healing that is later clamped at max HP (`combatprocessing.go:209-213` versus `combatcalculation.go:237-241`) when two healers target the same unit (confidence: medium; reporting only).
- Per-unit `UnitsKilled` in summaries inherits the `WasKilled` errors described under `combatcore`.

#### Dead code
- `BattleRecorder.EngagementCount` (`battle_recorder.go:198`): no callers.
- `HitTypeHeal` arms at `battle_summary.go:147,261`.
- `getTargetMode` (`battle_summary.go:225`) says "most common" and returns the first.

#### Leave alone
- `ExportBattleJSON`, the record/summary struct shapes (the tools parse this JSON, and `tools/` is out of scope), `buildUnitSummary`'s aggregation.

---

### tactical/combat/combatdisposal (41 lines, tests: no)

**Role:** One function, `RemoveSquadFromMap`, that removes a destroyed squad and its units. **Verdict:** Should not be a package.

#### Design findings

##### [Medium] A package for one function whose extra steps are redundant
- Evidence: `combatdisposal/squad_removal.go:27-41`. Step 1 removes the squad from the position system, but `squadcore.DisposeSquadAndUnits` does that again through `CleanDisposeEntity` (`tactical/squads/squadcore/squadcreation.go:466-468`, `core/common/ecsutil.go:256-258`); the second removal's error is discarded. Step 2 strips faction membership from an entity that is disposed on the next line.
- Problem: the function is `squadcore.DisposeSquadAndUnits` plus an existence check, behind a package whose doc comment is about import cycles that would not exist if callers used `squadcore` directly. Three callers: `combatcore/combatactionsystem.go:195,198`, `tactical/powers/spells/system.go:196`, `gui/guicombat/combatbase/handler.go:211`; all already import `squadcore`.
- Direction: replace the three calls with `squadcore.DisposeSquadAndUnits` and delete the package. (If the action-state component moves onto the squad entity, disposal handles that too with no extra step.)
- Effort: S. Risk: low.

---

### tactical/combat/combatservices (817 lines, tests: yes, small)

**Role:** Facade that constructs and wires the combat systems, the artifact/perk pipeline, AI holders, exit state, and teardown. **Verdict:** Needs attention. The wiring (`PowerOrchestrator`) is good; the facade around it has accumulated pass-throughs, an AI holder it does not use, and duplicated teardown.

#### Design findings

##### [Medium] The turn loop lives in the GUI; the service exposes parts, not the operation
- Evidence: `gui/guicombat/combat_turn_flow.go:116-137,191-220` sequences `TurnManager.EndTurn`, `CheckVictoryCondition`, `CacheVictoryResult`, the player-or-AI decision, `AI.DecideFactionTurn` and recorder round updates. `tools/combat_analysis/combat_simulator/combat_runner.go:57-95` writes a different loop with its own target picker because there is nothing headless to call. Callers reach through the facade into its fields: `combatService.TurnManager.*` 15 times, `BattleRecorder.*` 12, `CombatActSystem`, `MovementSystem`, `FactionManager`, `CombatCache` (tally from grep over gui, mind, tools).
- Problem: simulation rules (when victory is checked, when the AI acts, what ends a turn) are owned by presentation code, so they cannot be tested or simulated without the GUI, and the simulator's battles do not exercise the real AI. `CombatService` holds an `AITurnController` it never calls.
- Direction: add one headless operation on `CombatService`, roughly "end the current turn; report whether the battle is over; if the next faction is AI, run its decisions and return the queued attacks". The GUI keeps animation chaining and panel refresh and calls that. The simulator can then use it. This is moving existing code, not adding a feature.
- Effort: M. Risk: medium (GUI package has no tests).

##### [Medium] Sixteen one-line delegations, five with no callers
- Evidence: `combat_service.go:139-196`. No callers anywhere (grep including tests): `Exit()` (`:146`), `SetThreatProvider` (`:181`), `SetThreatEvaluatorFactory` (`:184`), `UpdateThreatLayers` (`:189`), `SetAIController` (`:196`); and behind them `AICoordinator.SetController`, `SetThreatProvider`, `SetThreatEvaluatorFactory`, `UpdateThreatLayers` (`ai_coordinator.go:43-56,97-104`). The comments say "kept for caller compatibility" and "prefer cs.AI.Install for new code"; the only installer is `setup/gamesetup/moderegistry.go:103`.
- Problem: three ways to reach the same thing (`cs.X()`, `cs.AI.X()`, `cs.Exit().X()`) with comments steering between them. `cs.logger` duplicates `cs.Powers.Logger()` by its own admission (`combat_service.go:51-54`).
- Direction: delete the five unused delegates and the four unused coordinator methods. For the rest pick one access style: since `AI` and `Powers` are already exported fields, export the exit controller the same way and drop the remaining pass-throughs, or keep the pass-throughs and unexport the fields. Drop `cs.logger`.
- Effort: S. Risk: low.

##### [Medium] Teardown re-implements disposal and stripping that exist elsewhere
- Evidence: the player-squad loop at `combat_service.go:218-226` is the body of `combatlifecycle.StripCombatComponents` (`mind/combatlifecycle/teardown.go:48-63`; its comment acknowledges the duplicate). `disposeEnemySquads` plus the `SquadMemberTag` predicate pass (`combat_service.go:229-242,334-342`) is `squadcore.DisposeSquadAndUnits` per enemy squad. `cleanupEffects` and `collectPlayerSquadIDs` each run a full squad or member query, and `collectPlayerSquadIDs` runs twice (`:218,259`).
- Problem: the list of "combat-only state on a squad" is maintained in two packages; adding a new combat-only component means editing both.
- Direction: one `StripCombatState(squad)` owned next to the components it strips (it cannot live in `combatstate` because of `perks`, so `combatservices` is the natural owner and `combatlifecycle` calls it). Use `squadcore.DisposeSquadAndUnits` for enemies. Compute the player squad list once.
- Effort: S. Risk: low-medium (teardown bugs show up as leaked entities; verify with a before/after entity count).

##### [Low] AI vocabulary and a service-locator role in the combat facade
- Evidence: `ai_interfaces.go:17-55` defines `AITurnController`, `ThreatProvider`, `ThreatLayerEvaluator` and `ThreatSnapshot` (seven named AI threat layers) so that `gui` can reach `mind/behavior` data through `CombatService` (`gui/guicombat/combatvisualization/manager.go:52,62,135`). Nothing in `tactical/combat` calls these except `AICoordinator.Reset` at teardown.
- Problem: adding an AI threat layer means editing a struct in the combat service package. It is dependency inversion used to route around a GUI-to-AI import, with combat as the middleman.
- Direction: acceptable as is if the turn-loop finding is implemented (then the service genuinely uses the controller). Otherwise move the threat interfaces and `AICoordinator` to where they are produced (`mind/`), and pass them to the GUI through its dependency struct. The `mind/ai` section recommends `AICoordinator` as the single owner of the evaluator cache, which fits keeping these here.
- Effort: M. Risk: low.

##### [Low] Construction order encoded in side effects
- Evidence: `NewCombatService` calls `setupPowerDispatch(cs)` (`combat_power_dispatch.go:22-34`), a free function that mutates the half-built service; `InstallLogger` creates the perk dispatcher as a side effect (`power_orchestrator.go:58-68`) and "must be called before WirePipeline". The four `if o.perkDispatcher != nil` checks in `WirePipeline` (`:90,97,109,120`) guard a state that cannot occur.
- Direction: `NewPowerOrchestrator(manager, logger)` builds the tracker, both dispatchers and the logger in one step; `WirePipeline` loses the nil checks; `setupPowerDispatch` reduces to constructing the logger.
- Effort: S. Risk: low.

#### Dead code
- Listed in the delegation finding above (nine methods).
- `combat_service.go:92-94` section header "Power System Dispatch" sits above GUI setters.

#### Leave alone
- `PowerOrchestrator.WirePipeline` (`power_orchestrator.go:81-135`): one place that states the artifact, perk, GUI order per event. Keep this exactly as the model for adding the battle recorder as a subscriber.
- `CombatExitController.CheckVictoryCondition`.
- `AICoordinator.Reset` and the per-combat cleanup it performs.
- `classifyLogPrefix` (fine once the dead gear logger is gone).

---

## 14. Tactical powers (perks, artifacts, spells, effects)

Packages reviewed in full (all non-test files): `powercore`, `perks`, `artifacts`, `spells`, `effects`, `progression`. Callers read for context: `tactical/combat/combatservices` (power orchestrator, combat service), `tactical/combat/combatcore` (turn manager, attack pipeline), `tactical/combat/combatmath`, `tactical/combat/combattypes`, `gui/guiartifacts`, `gui/guispells`, `gui/guisquads/squadeditor_perks.go`, `gui/guiprogression`, `setup/savesystem/chunks`, `setup/gamesetup/bootstrap.go`, `templates` (artifact/spell definitions, `Loader[T]`).

### Layer overview

**Are the four layers clean?** As gameplay roles, mostly yes. Each layer turns a different knob, and I found no case of two layers implementing the same rule:

| Layer | Trigger | Knob it turns | State |
|---|---|---|---|
| effects | applied by spells / artifacts | unit base stats, timed or battle-long | ECS component on unit |
| spells | player-cast, mana-gated | direct HP damage, or applies effects | ECS components on squad (mana, spellbook) |
| artifacts (minor) | passive | applies battle-long effects at combat start | none |
| artifacts (major) | player-activated or reactive | squad action economy (`ActionStateData`) | Go object on `PowerOrchestrator` |
| perks | passive hooks | damage pipeline (modifiers, cover, targets, death) | ECS component on squad |

The layering is sound and should be kept. The problems are in three other places:

1. **Lifecycle wiring between the layers and the turn manager is wrong in ways that disable perks** (see perks "Probable bugs"). "Round", "turn" and "battle" are used loosely across perks, artifacts and `TurnManager`, and the code that resets state follows the names rather than the actual event timing.
2. **Several pieces of state are stored twice or cached**: artifact equip state (squad list + inventory instance), the squad spellbook (snapshot of leader spells ∩ commander unlocks), effects (base stats mutated in place with the delta remembered separately). Each one has a concrete drift path.
3. **Rules that belong to the simulation are enforced only in the GUI**: perk unlock gating, artifact target/equip validation, "casting a spell uses the squad's action", AoE target resolution. `ExecuteSpellCast`, `ActivateArtifact` and `EquipPerk` accept inputs the GUI would have rejected. The planned AI spell casting (`mind/ai/ai_controller.go:86` TODO) would have to re-implement these.

**Plumbing duplicated between perks and artifacts.** Both have: a behavior interface with a no-op base struct, a registry map keyed by ID, a JSON definition registry, a "coverage" validator, a balance-config loader, a dispatcher with `SetLogger`, and a context struct embedding `powercore.PowerContext`. I do **not** recommend unifying the behavior interfaces or building a generic registry: the hook sets are different, there are 6 artifact behaviors and 21 perks, and a shared abstraction would save roughly 40 lines while coupling two things that change independently. What is worth aligning is the conventions around them (below).

**Inconsistent conventions for the same job across the slice:**

- *Loading gamedata.* Spell and artifact definitions use `templates.Loader[T]`. Perk definitions (`perks/registry.go:127`), perk balance (`perks/balanceconfig.go:125`) and artifact balance (`artifacts/balanceconfig.go:31`) each hand-roll `os.ReadFile` + `json.Unmarshal`, and `setup/gamesetup/bootstrap.go:41-46` adds two extra `log.Fatalf` points that CLAUDE.md says should not exist. `templates.Loader[T]` is exported and generic; all three can use it without moving the types.
- *IDs.* Perks use a typed `PerkID` with Go constants; spells use typed `templates.SpellID`; artifacts use bare `string` for both artifact IDs and behavior keys.
- *Enums in JSON.* Spells and artifacts use strings (`"damage"`, `"major"`). Perks use raw ints (`"tier": 1, "category": 3` in `perkdata.json`), which are unreadable in data and silently change meaning if the Go `iota` order changes.
- *Logging.* Perks/artifacts go through `powercore.PowerLogger`; spells and effects call stdlib `log.Printf` unconditionally (`spells/system.go:197,202,231`, `effects/system.go:42,74,128`); the logger itself ends in `fmt.Printf` (`combatservices/combat_power_dispatch.go:26-29`). None of it reaches the in-game UI.
- *Per-battle state.* Perks keep it in an ECS component (`PerkRoundState`); artifacts keep it in a Go object (`ArtifactChargeTracker`) owned by a service.

**Layout.** Keep the six packages. No merges or splits are justified. `progression` (commander currencies and unlock libraries) is not really a "power" and is easy to confuse with `tactical/squads/unitprogression`, but moving it buys little; a rename is optional.

**Suggested order of work for this slice:** (1) fix the perk lifecycle and save bugs, (2) make spellbook and artifact equip state single-source, (3) fix effects apply/reverse, (4) move rule validation out of the GUI into the three system entry points, (5) convention cleanup and dead code.

---

### tactical/powers/perks (1995 lines, tests: yes)

**Role:** Passive squad perks implemented as hook behaviors on the damage pipeline, with per-squad combat state, JSON definitions and balance values.
**Verdict:** Needs significant rework of state lifecycle (not of structure). The file layout, registry and behavior pattern are good, but turn/round reset timing disables or weakens at least six of the 21 perks, and equipped perks are not saved.

#### Design findings

##### [High] "Per-round" perk state is reset on every faction turn, immediately after TurnStart hooks populate it
- Evidence: `tactical/combat/combatcore/turnmanager.go:137-163` (`EndTurn` calls `ResetSquadActions(newFaction)`, which fires the post-reset hook, and only then fires `onTurnEnd`); `tactical/combat/combatservices/power_orchestrator.go:88-93` (post-reset → `DispatchTurnStart`) and `:107-112` (turn end → `DispatchRoundEnd`); `perks/dispatcher.go:192-200` (`DispatchRoundEnd` sets `PerkState = nil` on every squad with perk slots); `perks/system.go:129-131`.
- Problem: `DispatchRoundEnd` is subscribed to the *turn*-end event, and it runs *after* the next faction's TurnStart hooks. Everything a TurnStart hook writes into `PerkState` is erased before the faction can act, and everything written during a faction's own turn is erased before the opponent acts. The name "round" hides this; the lifecycle table in `perks/components.go:23-32` documents an intent the wiring does not implement.
- Direction: Drop the round-end reset entirely. In `DispatchTurnStart`, clear the squad's per-turn perk state first, then run TurnStart hooks. State then lives from a squad's turn start to its next turn start, which is what every perk description expects ("until next turn", "next attack", "resets per round"). Delete `DispatchRoundEnd`, `ResetPerkRoundStateRound`, and the perk `OnTurnEnd` subscription.
- Effort: S. Risk: medium (changes live combat behavior; existing perk tests call the dispatcher directly and will not catch the wiring, so verify with one end-to-end turn sequence).

##### [High] Squad-level "next attack" perks are implemented in a per-unit, per-target hook that also mutates state
- Evidence: `tactical/combat/combatcore/combatprocessing.go:49-57` (`applyDamageMods` runs once per attacking unit per target); `perks/behaviors_per_round.go:126-133` (Counterpunch clears `Ready` on first call), `:152-168` (Deadshot, same), `perks/behaviors_per_battle.go:32-43` (Opening Salvo sets `HasAttackedThisCombat` on first call), `perks/behaviors_per_round.go:179-193` (Adaptive Armor increments per call), `perks/behaviors_per_battle.go:105-116` (Grudge Bearer stacks per damaging hit).
- Problem: The perks are described at squad-attack granularity ("squad's first attack", "next attack"), but the hook fires for every unit-vs-target calculation, and the `*DamageMod` hooks change state as a side effect. Result: only the first unit's first hit gets the Counterpunch / Deadshot / Opening Salvo bonus; Adaptive Armor and Grudge Bearer reach their caps inside a single squad attack. Mutating state inside "modifier" hooks also means the hooks can never be reused for a damage preview or AI estimate without corrupting state.
- Direction: Make `*DamageMod` / `CoverMod` / `TargetOverride` hooks read-only. Move state changes to a squad-level point that already exists: `DispatchAttackTracking` (fired once per attack action). Counterpunch, Deadshot and Opening Salvo can then be pure functions of shared flags (`AttackedThisTurn`, a new battle-level "has attacked" flag) and their three one-field state structs disappear. Decide explicitly whether Adaptive Armor / Grudge Bearer count hits or attacks, and count in the matching hook.
- Effort: M. Risk: medium (balance-visible).

##### [Medium] `HookContext` is populated differently by each dispatch method, and one field carries the wrong kind of ID
- Evidence: `perks/hooks.go:18-29` ("some fields may be zero-valued depending on the hook type"); `perks/dispatcher.go:112-127` (`CounterMod` puts the attacker *squad* ID into `AttackerID`, which everywhere else is a unit ID, and leaves `AttackerSquadID` zero; call site `combatcore/combatactionsystem.go:126` passes two squad IDs); `perks/dispatcher.go:98-110` (`TargetOverride` leaves `DefenderID` zero); `perks/behaviors_stateless.go:262` (Precision Strike re-derives the squad from `AttackerID` although `AttackerSquadID` is set).
- Problem: A perk author must know which of seven ID fields is valid in which hook. It works today because each perk happens to read the right ones.
- Direction: Always fill the owner squad and both squad IDs; leave unit IDs zero only where no unit exists, and say so per hook in the interface comment. Rename the `CounterMod` parameter to `attackerSquadID`. Consider passing one small struct of IDs through `combattypes.PerkDispatcher` instead of four positional `ecs.EntityID`s, which removes the argument-order risk at nine call sites.
- Effort: S. Risk: low.

##### [Medium] Perk rules that are defined in data are not enforced by the system layer
- Evidence: `perks/registry.go:67` (`Roles`), validated at load (`:157-166`) and displayed (`perks/format.go:23-26`), but `EquipPerk` (`perks/system.go:17-58`) never checks it and `GetPerksByRole` has no callers; unlock gating exists only in `gui/guisquads/squadeditor_perks.go:82-90` (and is skipped when the squad has no commander); `EquipPerk` takes a `maxSlots` parameter whose only caller passes `perks.MaxPerkSlots` (`gui/guisquads/squadeditor_perks.go:165`).
- Problem: Role restrictions are advertised to the player and do nothing. Unlock gating can be bypassed by any non-GUI caller. The slot cap is a constant that is also a parameter.
- Direction: Decide whether roles restrict equipping; either enforce in `EquipPerk` or delete the field from data and UI. Move the unlock check to one system-level function (it needs `progression`, which already imports `perks`, so put a `CanEquipPerk`-style check in `progression` or pass an `isUnlocked` predicate). Drop the `maxSlots` parameter.
- Effort: S. Risk: low.

##### [Medium] Perk numbers live in three places
- Evidence: `resources/assets/gamedata/perkbalanceconfig.json` (authoritative), `perkdata.json` descriptions ("+30% damage dealt, but +20% damage received", "+35% damage on squad's first attack"), and Go comments above each behavior (`perks/behaviors_stateless.go:41,55,68,86,146,170,224`).
- Problem: Changing a balance value leaves the player-facing description and the code comment wrong. `guardianProtocol.redirectFraction` and `fieldMedic.healDivisor` are divisors tagged as `count`, described as percentages.
- Direction: Either format descriptions from `PerkBalance` (placeholders in the description string) or accept the duplication and remove the numbers from Go comments. Rename `RedirectFraction` to a divisor name or make it a real fraction.
- Effort: S-M. Risk: low.

##### [Low] Naming that misdescribes lifetime
- Evidence: `PerkRoundState` holds per-turn flags, per-round map and per-battle map (`perks/components.go:33-59`); `ResoluteState.RoundStartHP` is snapshotted at the owning faction's turn start (`perks/behaviors_per_battle.go:53-66`); `CleanupRoundState` (`perks/system.go:90`) and `RemovePerkRoundState` (`perks/init.go:26`) do the same thing, and only the second is used in production.
- Direction: Rename to `PerkCombatState`; keep one removal function; fix the comments in `components.go:57` and `behaviors_per_battle.go:4` that point at the unused one.
- Effort: S. Risk: low.

##### [Low] Perk definition loading does not follow the templates conventions
- Evidence: `perks/registry.go:127-196` (own file read; logs each problem *and* returns it; skips bad entries and continues; integer enums for tier/category); `perks/registry.go:12-18` describes `PerkTier` as "implementation complexity", which is not a gameplay concept.
- Direction: Use `templates.Loader[T]` with a `Validate` func; return errors without also logging; switch tier/category to strings in JSON. Same for `LoadPerkBalanceConfig`.
- Effort: S. Risk: low.

#### Mechanics notes
- Deadshot's Patience and Counterpunch read "last turn" snapshots that are computed from a zero-valued state on a squad's first turn, so `WasIdleLastTurn` is true at battle start (`perks/system.go:116-125`): Deadshot is armed on turn one for everyone.
- Riposte sets `HitModifier = 0` and Stalwart sets `DamageMultiplier = 1.0` by assignment (`perks/behaviors_stateless.go:219`, `perks/behaviors_per_round.go:73`), overwriting anything an earlier perk in slot order did to the same field. Other perks multiply/add.
- Cleave and `GetUnitsInRow` hardcode the grid bound `2` (`perks/behaviors_stateless.go:193`, `perks/unithelpers.go:132`) instead of `squadcore.SquadGridSize`.
- Enemy squads never receive perks (no assignment in spawning/encounter code), so the system is currently player-only.

#### Probable bugs
- **Counterpunch and Deadshot's Patience never fire after the first turn change** (confidence: high). TurnStart arms them via `SetPerkState`, then `DispatchRoundEnd` wipes `PerkState` in the same `EndTurn` call. Trace: `turnmanager.go:153` → `power_orchestrator.go:91` → `perks/dispatcher.go:186-187`, then `turnmanager.go:158-159` → `power_orchestrator.go:110` → `perks/dispatcher.go:197`.
- **Reckless Assault's downside never applies on the enemy turn** (confidence: high). `Vulnerable` is set during the owner's turn and wiped at that turn's end, same path as above. It only affects the immediate counterattack.
- **Guardian Protocol only works if the attacked squad also has the perk** (confidence: high). `DamageRedirect` is dispatched over the *defender* squad's equipped perks (`perks/dispatcher.go:155-173`), then the behavior additionally requires the neighbour to have it (`perks/behaviors_stateless.go:237-240`). The description says the guardian protects adjacent squads.
- **Equipped perks are lost on save/load** (confidence: high). No file under `setup/savesystem` references `PerkSlotComponent` / `PerkSlotData` (grep over the repo); only the commander's unlocked list is saved.
- **Resolute cannot trigger during the opening enemy turn** (confidence: medium-high). The HP snapshot is taken in the owner's TurnStart; `TurnManager.InitializeCombat` only resets the first faction (`turnmanager.go:69-71`), so if the enemy acts first there is no snapshot and `DeathOverride` returns false (`perks/behaviors_per_battle.go:84-87`).
- **Fortify's `MovedThisTurn` branch is unreachable** (confidence: high, harmless). `ResetPerkRoundStateTurn` clears the flag before TurnStart hooks run (`perks/dispatcher.go:186-187`), so `perks/behaviors_per_round.go:88-89` never executes; the reset actually happens in `DispatchMoveTracking`.
- **Available-perk list order changes on every refresh** (confidence: high, cosmetic). `GetAllPerkIDs` returns map order (`perks/registry.go:84-90`) and `gui/guisquads/squadeditor_perks.go:83` does not sort. Sort inside `GetAllPerkIDs`, as `artifacts.AllBehaviors` does.

#### Dead code
- `GetPerksByTier`, `GetPerksByRole` (`perks/registry.go:93,104`): no callers anywhere, including tests.
- `CleanupRoundState` (`perks/system.go:90`): tests only.
- `em.WorldTags["perkslot"]` (`perks/init.go:17`): never read.
- `combattypes.DamageModifiers.CoverBonus` and `.SkipCounter` are documented as perk outputs but never written by any perk (see `tactical/combat/combattypes`).

#### Performance
- Every hook call builds its log string with `fmt.Sprintf` before the nil-logger check (e.g. `perks/behaviors_stateless.go:82`, `perks/behaviors_per_round.go:50,105,190`), once per unit-vs-target hit. Irrelevant in the game, measurable in `tools/combat_analysis` batch runs. Pass format + args, or check for a logger first.
- Squad-level conditions (`HasWoundedUnit`, `CountTanksInRow`, friendly-squad range scans) are recomputed for every unit-vs-target hit. Fine at current squad sizes; goes away naturally if conditions are evaluated once per attack action.

#### Leave alone
- One file per state category (`behaviors_stateless.go`, `behaviors_per_round.go`, `behaviors_per_battle.go`), the `BasePerkBehavior` embedding, `perkids.go`, `ValidateHookCoverage`, `unithelpers.go`, and the `SquadPerkDispatcher.run` primitive. These are the right size and read well.
- The `combattypes.PerkDispatcher` interface with a no-op default. It keeps the combat core free of a perks import. Adding a hook touches five declarations, which is acceptable at the current rate of change.
- The tagged balance structs in `balanceconfig.go`.

---

### tactical/powers/artifacts (1280 lines, tests: yes)

**Role:** Artifact inventory and equipment, minor-artifact stat effects, and major-artifact behaviors with charges and deferred effects.
**Verdict:** Needs attention. The behaviors are small and clear, but equip state has two sources of truth with a real drift path, and per-battle state sits outside ECS keyed in a way that cannot distinguish squads.

#### Design findings

##### [High] Equip state is stored twice and kept in sync by hand
- Evidence: `artifacts/components.go:11-20` (`EquipmentData.EquippedArtifacts` on the squad; `ArtifactInstance.EquippedOn` in the player inventory); `artifacts/system.go:83-177` (equip/unequip write both, with manual rollback); both are saved (`setup/savesystem/chunks/gear_chunk.go:72,87`). Squad destruction does not touch either: `tactical/combat/combatdisposal/squad_removal.go:27-41` → `squadcore.DisposeSquadAndUnits`; the only caller of `UnequipArtifact` is `gui/guisquads/artifact_refresh.go:370`.
- Problem: When a player squad carrying artifacts is destroyed, the inventory instances keep `EquippedOn` pointing at a disposed entity. They are neither available nor removable (`RemoveArtifactFromInventory` only removes unequipped instances), and the GUI resolves a squad name for a dead ID.
- Direction: Keep one source. The squad's `EquippedArtifacts` list is the one combat reads, so keep that and reduce the inventory to owned counts per artifact ID; "available" is owned minus the number equipped across squads (one query). This deletes `ArtifactInstance`, `MarkArtifactEquipped`, `MarkArtifactAvailable`, both rollback branches and the `EquippedOn` remap in the gear chunk. Then decide the rule for a destroyed squad (artifact returns, or is lost) and implement it in one place.
- Effort: M. Risk: medium (save format change).

##### [Medium] Charges and pending effects are combat state kept in a service object, keyed only by behavior name
- Evidence: `artifacts/artifactcharges.go:15-19` (maps keyed by behavior string); `tactical/combat/combatservices/power_orchestrator.go:33-52` (tracker owned by the orchestrator); `artifacts/pending_effects.go:34-112`; `artifacts/dispatcher.go:54-79`; `artifacts/behaviors.go:81-114`.
- Problem: CLAUDE.md puts game state in ECS components; this is the one piece of combat state in the slice that is not. Because the key is the behavior, two squads with a copy of the same artifact share one charge (the inventory explicitly supports multiple copies), and the tracker cannot tell factions apart. The pending queue needs `Add`/`AddBroadcast`/`Consume`/`Keys`, a broadcast flag, and special dispatch for "behaviors with pending effects", and `applyPendingEffects` consumes every queued entry for a behavior even when its target is not in the faction being reset (harmless only because every battle currently has exactly two factions).
- Direction: A deferred effect is "something happens to squad X at its next reset". Store that on the target squad's `ActionStateData` (for example a skip-next-turn flag and a next-turn movement penalty) and let the reset consume it; the pending queue, its dispatch branch and the broadcast mode go away. Store charge usage on the squad that holds the artifact (a small component created at combat init, like `PerkRoundState`). If "one charge per behavior per battle regardless of copies" is the intended rule, write it down and block equipping duplicates.
- Effort: M. Risk: medium.

##### [Medium] Activation is validated in the GUI, and the GUI assembles the simulation context
- Evidence: `gui/guiartifacts/artifact_handler.go:150-161` (GUI builds `BehaviorContext` from `CombatService` internals and passes a nil logger); `:71-87` (friendly/enemy target check lives here); `artifacts/registry.go:48-57` (`ActivateArtifact` checks only "is player-activated"); `artifacts/behaviors.go:179-194,297-314` (Twin Strike and Echo Drums never check that the target is friendly, or that any squad has the artifact equipped).
- Problem: The simulation entry point accepts activations the rules forbid, and the same dispatcher that owns manager, cache, tracker and logger is bypassed, which is why activation log lines are silently dropped.
- Direction: Add one activation method on `ArtifactDispatcher` (or `CombatService`) that takes behavior key + target, builds the context itself, checks target type against `TargetType()`, checks the artifact is equipped by the acting faction, then calls `Activate`. The GUI handler shrinks to input handling.
- Effort: S. Risk: low.

##### [Low] Slack in the behavior contract
- Evidence: `ArtifactBehavior.OnTurnEnd` (`artifacts/behavior.go:59`) has no overriding implementation, yet `DispatchOnTurnEnd` sorts and iterates all behaviors every turn (`artifacts/dispatcher.go:100-106`); `ChargeType()` is declared per behavior (`behaviors.go:231,295`) but each `Activate` hardcodes the charge type again in `UseCharge` (`:274,309`) and `ChargeType()` is read only by the GUI (`gui/guiartifacts/artifact_panel.go:81`); `requireCharge` always says "already used this battle" (`behaviors.go:49`); `ArtifactDefinition.Behavior` duplicates the ID for every major artifact (`templates/artifactdefinitions.go:67-70`); `TierMinor`/`TierMajor` are unused while the GUI compares against the literal `"major"` (`gui/guiartifacts/artifact_handler.go:113`).
- Direction: Remove `OnTurnEnd` from the interface; add one `consumeCharge(ctx, b)` that uses `b.ChargeType()`; make "behavior key == artifact ID" the rule and drop the separate field; use the tier constants or delete them.
- Effort: S. Risk: low.

##### [Low] Speculative `MaxSlots` on every squad
- Evidence: `artifacts/components.go:5-14` ("so future ... slot variants can be supported"); `artifacts/system.go:17-19,111-119` (zero means "read config"); duplicated fallback in `gui/guisquads/artifact_refresh.go:267-270`; the config key is `MaxArtifactsPerCommander` but it is applied per squad.
- Direction: Read the limit from config in one query function; drop the stored field until a feature needs it; rename the config key.
- Effort: S. Risk: low (save format).

##### [Low] A JSON file, loader and validator for a single integer
- Evidence: `artifacts/balanceconfig.go` (54 lines), `gamedata/artifactbalanceconfig.json` (one value), and the same value restated in the artifact description ("reduce all enemy squads' movement by 2").
- Direction: Keep the mechanism only if more artifact tuning values are expected soon; otherwise put the value on the artifact's definition entry.
- Effort: S. Risk: low.

#### Mechanics notes
- "Once per round" charges refresh at every faction turn end (`artifacts/dispatcher.go:101`), not at round wrap. With two factions and player-only activation the result is the same; the name is misleading in the same way as the perk reset.
- Twin Strike's comment says it "grants bonus attack flag" (`behaviors.go:170`) but it resets `HasActed`; the actual `BonusAttackActive` flag in `combatstate` is never set to true anywhere (see `tactical/combat/combatstate`).
- No production code adds artifacts to the inventory; `AddArtifactToInventory` is called only from `testing/bootstrap/initial_artifacts.go:31`.

#### Probable bugs
- **Artifacts on a destroyed player squad become permanently stuck** (confidence: medium-high; traced as described in the first finding, not reproduced in-game).
- **Activation log lines are never emitted** (confidence: high). `gui/guiartifacts/artifact_handler.go:155` passes `nil` as logger, so every `ctx.Log(..., "activated")` in `behaviors.go` is a no-op for the only activation path.
- **Deferred effects aimed at a faction that is not next in turn order would be consumed and discarded** (confidence: high in code, latent: all encounters use `CreateFactionPair`).

#### Dead code
- `TierMinor`, `TierMajor` (`artifacts/components.go:37-40`): no references.
- `RemoveArtifactFromInventory` (`artifacts/system.go:34`), `PendingEffectQueue.Has` and `.Count` (`artifacts/pending_effects.go:87,105`): tests only.
- `ArtifactBehavior.OnTurnEnd`: never overridden.

#### Leave alone
- `behaviors.go` as a single file with one small type per artifact, and the three charge/pending helpers at its top.
- `ValidateBehaviorCoverage`, `AllBehaviors` (deterministic order), `BaseBehavior`.
- `ApplyArtifactStatEffects` delegating to `effects`: minor artifacts correctly reuse the effects layer instead of inventing a second stat-bonus mechanism.

---

### tactical/powers/effects (316 lines, tests: yes)

**Role:** Timed or battle-long stat modifiers attached to units, used by spells and minor artifacts.
**Verdict:** Needs attention. It is small and well scoped, but its core technique (mutate base stats, undo by applying the inverse) is not reversible in all cases and depends on every combat exit path remembering to clean up.

#### Design findings

##### [High] Effects mutate base `Attributes` in place and undo by inverse arithmetic
- Evidence: `effects/system.go:29` (apply), `:127,166` (reverse), `:172-196` (each stat clamped with `max(0, …)` on both apply and reverse); cleanup is one call site, `tactical/combat/combatservices/combat_service.go:257-279`, filtered by faction membership; `mind/encounter/exit_hooks.go:56-64` strips faction membership from garrison squads *before* `TeardownCombat` runs.
- Problem: (a) Apply and reverse are not inverses when the clamp engages: a −2 armor debuff on a unit with 1 armor lowers it by 1 and later raises it by 2. Stacked debuffs make this reachable with current spell data (`weaken`, armor −2 spells against low-armor units). (b) The unit's real stats are only correct if cleanup runs; any exit path that skips it bakes battle-long artifact bonuses into base stats, and they are applied again next battle. (c) Strength changes move `GetMaxHealth()` (`core/common/commoncomponents.go:76-78`) without touching `CurrentHealth`, so a unit can sit above max HP after a buff expires or a debuff lands.
- Direction: Minimum fix: store the delta actually applied in `ActiveEffect` and reverse exactly that; make cleanup independent of faction membership (remove effects for every unit of the squads in the combat, or do it inside the same function that strips combat components). Cleaner alternative if stat reads can be funnelled: never mutate base stats and compute effective stats from base + active effects; that is a larger change touching every `Attributes` reader, so only do it if another reason appears.
- Effort: S for the minimum fix, L for effective-stat computation. Risk: low / high.

##### [Medium] Stat names are parsed at apply time, and the modifier shape is declared three times
- Evidence: `effects/system.go:71-76` (unknown stat → log line, modifier skipped, at cast time); identical structs `effects.StatModifier` (`effects/components.go:53-56`), `templates.SpellStatModifier` (`templates/spelldefinitions.go:28`), `templates.ArtifactStatModifier` (`templates/artifactdefinitions.go:6-9`); two copy loops to convert between them (`spells/system.go:213-216`, `artifacts/system.go:201-204`).
- Problem: A typo in `spelldata.json` or `minor_artifacts.json` is discovered when the spell is cast, as a log line. The three types exist only so `effects` does not import `templates`, which every other package in the slice already imports.
- Direction: One `StatModifier` type in `templates` used by both definitions; validate stat names in the loaders' `Validate` step (the stat vocabulary is eight strings); `effects.ApplyStatModifiers` takes that type. Removes two structs and two loops.
- Effort: S. Risk: low.

##### [Low] `EffectSource` is written but never read
- Evidence: `effects/components.go:24-32` (two of four values marked "reserved ... no production writer yet"); no reader of `ActiveEffect.Source` in the repo.
- Direction: Delete the field and enum until something filters by source.
- Effort: S. Risk: low.

#### Mechanics notes
- Durations tick at the start of the *affected* unit's faction turn (`combatcore/turnmanager.go:95-97`). A 2-turn debuff cast on an enemy is active for one enemy turn; a 2-turn buff cast on your own squad covers the rest of this turn plus one full turn. Same number, different effective length.
- In `ResetSquadActions` the squad's movement for the turn is computed before effects are ticked (`turnmanager.go:92-97`), so a movement modifier that expires this turn still applies to this turn's movement.

#### Probable bugs
- **Permanent stat drift from clamped debuffs** (confidence: high in code; needs a debuff larger than the current stat, reachable by stacking).
- **Garrison squads keep battle effects after a successful garrison defense** (confidence: medium). Order traced: `OnBeforeTeardown` strips membership (`mind/encounter/exit_hooks.go:63`), then `cleanupEffects` selects units by player-controlled faction membership and no longer finds them. Depends on whether garrison squads belong to the player-controlled faction in that combat type, which I did not trace.

#### Dead code
- `ActiveEffectsTag` (`effects/init.go:12`): never queried.
- `SourceAbility`, `SourcePerk`: unused; `GetActiveEffects` and `ApplyEffectToUnits` are used only inside the package and could be unexported.

#### Leave alone
- The package boundary (imports only `core/common`), the single `ActiveEffectsData` component, and `ApplyStatModifiers` as the one entry point shared by spells and artifacts.

---

### tactical/powers/spells (376 lines, tests: no)

**Role:** Squad mana and spellbook components plus spell cast execution (direct damage or applying effects).
**Verdict:** Needs attention. The spellbook is a cached relationship that goes stale, and spell damage is a second, thinner copy of the combat damage path.

#### Design findings

##### [High] `SpellBookData` is a snapshot of data that changes
- Evidence: `spells/system.go:45-63` (spellbook = leader unit type's spells ∩ commander's unlocked spells, computed once); only caller is `setup/gamesetup/initial_squads.go:83`; the only other writer is save load (`setup/savesystem/chunks/squad_chunk.go:297`); `progression.UnlockSpell` (`progression/library.go:126-132`) updates only the commander.
- Problem: This is the "cached relationship" the project's ECS rules forbid. Unlocking a spell does not add it to any existing squad; changing a squad's leader does not change its spells; squads created after game start never get spells or mana at all. The snapshot is also saved, so stale data persists.
- Direction: Delete `SpellBookData`. Make `GetAllSpells(squadID)` a query: leader unit type → `templates.GetSpellsForUnitType` → filter by the owning commander's library. Keep `ManaData`, and attach it wherever squads are created (or lazily on first query). This also removes the creation-order dependency documented at `spells/system.go:38-44`.
- Effort: S-M. Risk: medium (save format; no tests in this package).

##### [Medium] Spell damage bypasses the combat damage path, and casting rules live in the GUI
- Evidence: `spells/system.go:152-204` writes `CurrentHealth` directly and handles squad destruction itself; the physical path records damage, applies it, disposes dead units and fires the attack-complete event (`combatcore/combatactionsystem.go:181-213`); "casting uses the squad's action" is applied by the GUI after the call (`gui/guispells/spell_handler.go:310`); AoE target resolution and the enemy-only filter are in `gui/guispells/spell_handler.go:203-232`; `ExecuteSpellCast` checks mana and spellbook but not whether the squad has already acted, whose turn it is, or whether targets are legal.
- Problem: Two implementations of "apply damage to units, then handle a wiped squad" that already differ: spell kills do not dispose dead units in surviving squads, are not recorded by the battle recorder, and fire no completion event (so GUI refresh and perk tracking depend on the caller). Any second caller of `ExecuteSpellCast` (AI) must re-implement the GUI's rules.
- Direction: Have `ExecuteSpellCast` validate and mark the caster's action itself, and route spell damage through the same "record damage → apply → dispose → notify" steps the attack path uses (a shared function in the combat packages that takes a damage-by-unit map would serve both). Keeping spells out of perk hooks is a stated design decision and can stay.
- Effort: M. Risk: medium (no tests).

##### [Low] Small inconsistencies in the cast path
- Evidence: mana is checked before spellbook membership (`spells/system.go:124-134`); an unrecognized `EffectType` spends mana and reports success (`:140-148`); mana defaults come from `templates.GameConfig.Commander` although mana is per squad (`:61-62`).
- Direction: Validate effect type at load; reorder checks; move the config keys when convenient.
- Effort: S. Risk: low.

#### Mechanics notes
- There are two mana concepts: per-squad `spells.ManaData`, and a per-faction `Mana`/`MaxMana` on `combatstate.FactionData` that the combat HUD also displays (`gui/guicombat/combat_mode_init.go:211-218`). Spells only use the squad pool.
- Enemy squads never receive a spellbook, and the AI cannot cast (`mind/ai/ai_controller.go:86` TODO).

#### Probable bugs
- **Newly unlocked spells never become castable for existing squads** (confidence: high; by grep there is no writer of `SpellBookComponent` after creation/load).
- **Units killed by a spell in a surviving squad are not disposed until that squad is next in a physical attack** (confidence: medium; `DisposeDeadUnitsInSquad` is called only from `combatactionsystem.go:203,206`. Impact depends on how many readers filter on alive units).

#### Dead code
- `ManaTag`, `SpellBookTag` and their `WorldTags` entries (`spells/init.go:15-19`): never queried or read.

#### Leave alone
- `queries.go` and the component definitions other than the spellbook. Spell definitions living in `templates` is fine.

---

### tactical/powers/powercore (270 lines, tests: yes)

**Role:** Shared context struct, logger interface, lifecycle event pipeline and balance-range validator for perks and artifacts.
**Verdict:** Healthy but partly ceremonial. Nothing here causes bugs; about half of it is indirection with a single user.

#### Design findings

##### [Low] `PowerContext` is less shared than it claims
- Evidence: `powercore/context.go:25-30`; `RoundNumber` is written (`perks/system.go:145`, `artifacts/dispatcher.go:46`) and never read; `Cache` is used only by artifacts, and perks leave it nil (`perks/dispatcher.go:51`); the package doc says both contexts embed `*PowerContext` (`context.go:5`) but both embed by value.
- Direction: Delete `RoundNumber`. Either accept that the shared part is `Manager` + `Logger`, or move `Cache` to `BehaviorContext`. Fix the doc comment.
- Effort: S. Risk: low.

##### [Low] `PowerPipeline` is an event bus with one publisher and a fixed subscriber list
- Evidence: `powercore/pipeline.go` (four handler types, four `On*`, four `Fire*`); sole user `tactical/combat/combatservices/power_orchestrator.go:81-135`, which registers the same two or three closures per event at construction, each with a nil check on a dispatcher that is never nil at that point.
- Problem: Subscriber order is the important fact, and it is expressed as registration order of closures rather than as four plain functions that call artifacts, then perks, then the GUI hook.
- Direction: Optional. Four methods on `PowerOrchestrator` would say the same thing in fewer lines and delete `pipeline.go`. Not urgent; do it if this code is being edited anyway (for example when fixing the perk reset).
- Effort: S. Risk: low.

##### [Low] The logger abstraction only prints to stdout, and classifies messages by asking the artifact registry
- Evidence: `powercore/logger.go` (interface + func adapter, one implementation); `combatservices/combat_power_dispatch.go:23-46` (`fmt.Printf`, prefix chosen by `artifacts.IsRegisteredBehavior(source)`).
- Direction: Leave it unless power activations are meant to appear in the in-game combat log; if so, that is the moment to give the logger a real second implementation and pass the category explicitly instead of inferring it from the source string.
- Effort: S. Risk: low.

#### Leave alone
- `balancevalidate.go`. Tag-driven range checks are a good fit for the 17 perk balance structs, fail loudly on untagged fields, and are tested.

---

### tactical/powers/progression (189 lines, tests: yes)

**Role:** Per-commander currencies (Arcana, Skill) and unlocked perk/spell libraries.
**Verdict:** Healthy. Small, tested, one clear responsibility.

#### Design findings

##### [Low] Stale justification and a slightly clever internal helper
- Evidence: `progression/components.go:20-21` says IDs are plain strings "to avoid a common <-> perks dependency cycle", but the package already imports `perks` and `templates` (`progression/library.go:7-8`); `library` (`progression/library.go:30-47`) uses closures returning `*[]string` / `*int` to share ~25 lines between two axes.
- Direction: Use typed ID slices, or correct the comment. If the save chunk is being touched anyway, giving `ProgressionData` two fields of one small `{Points, Unlocked}` struct would let the shared logic be plain methods with no closures. Not worth a change on its own.
- Effort: S. Risk: low.

##### [Low] Unlock checks are consumed inconsistently
- Evidence: spells apply the library at squad creation inside the system layer (`spells/system.go:68-84`); perks apply it only in the GUI list filter (`gui/guisquads/squadeditor_perks.go:89`).
- Direction: Covered by the perks and spells findings; after those, both layers should ask `progression` at the point of use (equip / cast-list query).
- Effort: n/a.

#### Dead code
- `ProgressionTag` and `WorldTags["progression"]` (`progression/init.go:12-13`): never read.

#### Leave alone
- The public API (`IsPerkUnlocked`, `UnlockPerk`, `AddSkillPoints` and the spell equivalents) and the sentinel errors.

---

## 15. Mind (AI, combat lifecycle, encounters, spawning)

All non-test `.go` files in the six packages were read in full (34 files, ~4,670 lines). Claims marked "unused" were grepped across the whole repo including `tools/` and tests. Bugs were traced statically through the call path; nothing was executed.

### Layer overview

**`mind` holds two unrelated things.** Only three of the six packages are "the AI":

| Group | Packages | What it really is |
|---|---|---|
| AI brain | `ai`, `behavior`, `evaluation` | Enemy decision-making, threat maps, squad power rating |
| Combat plumbing | `combatlifecycle`, `encounter`, `spawning` | Combat start/exit pipeline, rewards, overworld encounter orchestration, enemy squad generation |

The second group has nothing to do with AI and is where most of the problems are.

**Layer direction between `mind` and `campaign` runs both ways.**
- `mind/encounter` imports `campaign/overworld/{core,garrison,ids,threat}`; `mind/spawning` imports `campaign/overworld/{core,ids}`.
- `campaign/raid` imports `mind/encounter`, `mind/combatlifecycle`, `mind/evaluation`.

No Go import cycle exists, but there is no answer to "which layer is above the other".

**Recommended layout (three moves, no new packages beyond relocation):**
1. `mind/combatlifecycle` → `tactical/combat/combatlifecycle`. It imports only `core/*` and `tactical/*` and is consumed by `campaign/raid`, `gui`, `setup`, the combat simulator and `mind/encounter`. It is tactical infrastructure.
2. Split `mind/encounter` along the line that already exists inside it:
   - The generic session part (`EncounterService`, `ModeCoordinator`, `EncounterController`, `ActiveEncounter`) joins `combatlifecycle`. It is the lifecycle's state holder; raid uses it too. Two one-implementation interfaces disappear as a result (see combatlifecycle findings).
   - The overworld-specific part (starters, resolvers, triggers, rewards, `SpawnCombatEntities`) moves to `campaign/overworld/encounter`, mirroring how `campaign/raid` already owns its own starter/resolver.
3. `mind/spawning` drops its `campaign` import by taking `(level int, encounterType)` or resolved squad-type preferences as parameters instead of `*core.OverworldEncounterData`. It then becomes the single enemy-squad generator that `campaign/overworld/garrison` and `setup/gamesetup` can call (both currently carry their own copy, see spawning findings).

After this, `mind` = `ai` + `behavior` + `evaluation` + `spawning`, it imports only `core`, `tactical`, `templates`, and `campaign` depends on `mind` one-way.

**ai / behavior / evaluation split is reasonable and should stay.** `evaluation` (power rating, no combat state) ← `behavior` (threat maps over combat state) ← `ai` (action choice + execution). The dependency direction is clean. The problems are inside the packages and in how they are wired to the GUI, not in the split.

**encounter vs spawning do not really overlap.** `spawning` builds squads and positions; `encounter` assembles factions and orchestrates. The only literal overlap is one duplicated positions call (`mind/encounter/encounter_setup.go:58` vs `mind/spawning/squadscreation.go:34`). Do not merge them; fix the dependency direction instead.

---

### mind/ai (715 lines, tests: no)

**Role:** Utility-style enemy AI: for each squad, generate move/attack/wait actions, score them, execute the best until the squad is out of actions.
**Verdict:** Needs attention. The decision loop is small and readable, but it carries presentation state, shares four evaluator caches with other layers (one of which is broken by `Reset`), and its tuning lives in two unrelated places.

#### Design findings

##### [High] Four caches of per-faction threat evaluators; `AIController.Reset` breaks one of them
- Evidence: `mind/ai/ai_controller.go:24,49,62-69,230-244,273-276`; `tactical/combat/combatservices/ai_coordinator.go:15,68-78,92`; `gui/guicombat/combatvisualization/manager.go:61-67,131-139,197`.
- Problem: `map[factionID]evaluator` exists in (1) `AIController.layerEvaluators`, (2) the local map captured by the `evalFactory` closure in `SetupCombatAI`, (3) `AICoordinator.layerEvaluators`, (4) `CombatVisualizationManager.layerEvaluators`. (1) and (2) start as the same map. `Reset()` at line 274 assigns a *new* map to the field, so the closure keeps writing to the old one. See "Probable bugs" for the consequence. Each cache needs its own reset, and the comments in three files explain to each other who resets what.
- Direction: One owner. `AICoordinator` already has `GetThreatEvaluator` with lazy creation; make the AI controller ask it (or hand the controller the same getter) and delete `AIController.layerEvaluators`, `getThreatEvaluator`, and the closure-captured map. The GUI manager should query the coordinator on demand instead of holding a parallel map.
- Effort: S. Risk: low (no tests).

##### [High] Animation queue lives inside the AI
- Evidence: `mind/ai/action_evaluator.go:48-52,71-75`; `mind/ai/ai_controller.go:26-27,82,246-267`; interface `tactical/combat/combatservices/ai_interfaces.go:17-24`; consumer `gui/guicombat/combat_turn_flow.go:131-185`.
- Problem: `AttackAction` holds an `*AIController` back-pointer solely so it can push onto an "attack queue for animations". Four of the five `AITurnController` methods (`HasQueuedAttacks`, `GetQueuedAttacks`, `ClearAttackQueue`, plus the queue part of `Reset`) exist for the GUI's playback. The simulation is holding presentation state and the GUI must remember to clear it.
- Direction: `DecideFactionTurn` returns what happened (e.g. the list of executed attacks). The GUI keeps its own playback queue. `AITurnController` shrinks to `DecideFactionTurn` + `Reset`; `AttackAction.aiController` and `ActionContext.AIController` go away.
- Effort: S-M. Risk: low (no tests).

##### [Medium] Scoring constants are hardcoded next to a JSON-driven weight system
- Evidence: `mind/ai/action_evaluator.go:180` (base 50), `:186` (ally ×3), `:213-222` (ZoC 0/-5/-15/-3), `:251-259` (approach 15/8/-5/5), `:266-270` (+20/+10 range), `:408-432` (attack 100, +20, +15, +10, +10, +10). JSON-driven side: `mind/behavior/threat_constants.go`, `resources/assets/gamedata/aiconfig.json`.
- Problem: Two tuning surfaces for one score. The role table in `aiconfig.json` (`meleeWeight`, `supportWeight`) and a second, hidden per-role table in three `switch` statements here. The comment at `:405-407` states an invariant ("attacks start at 100 to always beat movement") that the code does not enforce: a tank closing 3+ tiles scores 50 + 45 + 20 before threat attraction.
- Direction: Move the role-indexed numbers into the existing `roleBehaviors` entries and the scalar ones into one `aiconfig.json` section; or, if they are deliberately code-only, put them in one named const block at the top of the file. Either way, one place.
- Effort: S. Risk: low.

##### [Medium] Per-tile recomputation of values that do not depend on the tile
- Evidence: `mind/ai/action_evaluator.go:175-197` calls, for every candidate tile, `scoreApproachEnemy` → `findNearestEnemy` (`:276-304`, all factions × all squads, depends only on `CurrentPos`) and `getMaxAttackRange` (`:370-390`, all units); `mind/behavior/threat_composite.go:64-65` re-derives the squad role and re-scans the role-weights config for every tile.
- Problem: Up to (2r+1)² tiles × (enemy squads + own units + config scan) per action, per squad. Also obscures which inputs are per-squad and which are per-tile.
- Direction: Compute nearest enemy, max attack range and role weights once in `NewActionContext` (it already caches `SquadRole` and `CurrentPos`); `GetRoleWeightedThreat` takes the weights (or role) instead of a squad ID.
- Effort: S. Risk: low.

##### [Medium] Squad attack range is implemented three times
- Evidence: `mind/ai/action_evaluator.go:370-390` is a line-for-line copy of the private `tactical/combat/combatcore/combatactionsystem.go:218-243`; a third variant is `mind/behavior/threat_queries.go:124-137`; `mind/evaluation/power.go:232-235` reads the same component inline.
- Problem: The AI decides "can I attack" with its own copy of the rule the combat system enforces. If one changes, AI attacks start failing silently (a failed action just ends the squad's turn at `ai_controller.go:120-122`).
- Direction: Export one `squadcore.GetSquadAttackRange(squadID, manager)` and use it from all four sites.
- Effort: S. Risk: low.

##### [Low] Enemy-squad enumeration is repeated
- Evidence: `mind/ai/action_evaluator.go:280-301` and `:334-349`; `mind/behavior/threat_layers.go:33-44` with loops at `threat_combat.go:50-53`, `threat_positional.go:84-87`.
- Direction: One `combatstate` query returning active enemy squad IDs for a faction; four loops collapse to one call.
- Effort: S. Risk: low.

##### [Low] `WaitAction` writes `ActionStateData` fields directly; unconditional `[AI]` prints
- Evidence: `mind/ai/action_evaluator.go:91-96` (bypasses the `combatstate` mark-as-acted helpers used elsewhere); `:66-77` prints on every attack regardless of `config.DEBUG_MODE`.
- Direction: Use the `combatstate` helper; gate prints like `combatlifecycle/reward.go` does.
- Effort: S. Risk: low.

#### Mechanics notes
- **No "stay here" baseline.** Candidate tiles exclude the current tile (`core/common/grid_movement.go:27`) and the only alternative to moving is `WaitAction` at a fixed 0 (`action_evaluator.go:127-130`). Tile scores sit on an absolute scale around 50, so the AI moves whenever any reachable tile scores above 0, even when its current tile would score higher.
- **Threat maps are computed once per faction turn** (`ai_controller.go:89`) and not refreshed as squads act (`:101-126`). Later squads decide against ally positions and enemy squads that earlier squads in the same turn have already moved or destroyed.
- The `TODO` at `ai_controller.go:84-86` (AI spell casting) is a missing feature, not debt; noted only because the player-side spell system has no AI counterpart.

#### Probable bugs
- **Evaluator cache leak and split after the first combat (confidence: high).** `CombatService` is created once (`gui/guicombat/combat_mode_init.go:24`). After the first `TeardownCombat` → `AICoordinator.Reset` → `AIController.Reset` (`ai_controller.go:273-276`), the closure in `SetupCombatAI` (`:62-69`) still points at the original map. From the second combat on: evaluators requested through the factory (GUI path) are stored in the orphaned map and never cleared (one `CompositeThreatEvaluator` with eight position maps per faction per combat, for the whole session, which is the leak commit d99ffdd set out to fix), and the AI builds a second, separate evaluator per faction in the new map, so every layer is computed twice per update. Fix is `clear(aic.layerEvaluators)` instead of `make`, or better the single-owner change above.

#### Dead code
- `mind/ai/ai_controller.go:278-280`: comment about functions that moved long ago.
- `tactical/combat/combatservices/ai_coordinator.go:97-104` + `combat_service.go:189` (`UpdateThreatLayers`): defined, never called (outside this slice; noted because it is a fifth "update the layers" entry point).

#### Performance
- Covered by the per-tile finding and the double-evaluator bug above.

#### Leave alone
- The generate → score → pick → execute loop and the `ScoredAction` shape. It is simple and correct for the current feature set; do not replace it with a behaviour tree or planner.
- The `SquadAction` interface with three implementations is fine (a closed, real set).
- `SetupCombatAI` returning interfaces for `gamesetup` to install keeps `mind` out of `tactical` and `gui`; keep that wiring point.

---

### mind/behavior (1,226 lines, tests: yes)

**Role:** Per-faction threat maps (melee, ranged, support, positional risk) and the per-squad base threat table they are built from.
**Verdict:** Needs attention. The layer structure is clear, but the AI's inputs are only populated because a GUI object registers factions, two of the three layers cannot affect decisions at their current scale, and there is a fair amount of dead surface.

#### Design findings

##### [High] The AI's base threat data is populated only by the GUI
- Evidence: `mind/behavior/dangerlevel.go:26-33` (`AddFaction`) is called only from `gui/guicombat/combatvisualization/manager.go:56,127`. The AI path (`mind/ai/ai_controller.go:132-135`) calls `UpdateAllFactions`, which iterates only already-registered factions (`dangerlevel.go:35-39`). `mind/behavior/threat_combat.go:62-65` silently skips squads whose faction is unknown.
- Problem: If `CombatMode.Enter` did not call `RefreshFactions` (`gui/guicombat/combatmode.go:102`), the melee and ranged layers would be empty and the AI would move on approach bonuses alone, with no error. Simulation correctness depends on a visualization object, and the AI cannot run headless. `ThreatProvider.AddFaction` is in the tactical-layer interface purely to support this.
- Direction: `UpdateAllFactions` discovers factions itself from `combatstate.GetAllFactions` each update (it already re-queries squads per faction every time). Remove `AddFaction` from `combatservices.ThreatProvider` and from the GUI manager.
- Effort: S. Risk: low.

##### [Medium] Three stateful types for a table that is rebuilt from scratch every update
- Evidence: `mind/behavior/dangerlevel.go:12-141`: `FactionThreatLevelManager` → `FactionThreatLevel` → `SquadThreatLevel`, each copying `manager` and `cache`, to hold `map[squadID]map[range]float64`. `CalculateThreatLevels` (`:137-141`) replaces the whole map on each call. Destroyed squads stay in `squadThreatLevels` until `Reset`.
- Problem: Cached state with lifecycle management (add, update, reset, stale entries) where a recomputed value would do. This is the structure that made the `AddFaction` coupling and the reset commit necessary.
- Direction: One struct holding `map[squadID]map[int]float64`, with `Update()` (rebuild for all active squads of all factions) and the two getters. No per-faction or per-squad objects.
- Effort: S-M. Risk: low (tests exist in `dangerlevel_test.go` and will need adjusting).

##### [Medium] Code-side defaults duplicate values the loader already guarantees
- Evidence: `mind/behavior/threat_constants.go:9-12,51,65,80,94,105,133-138,150,159-161` against `templates/validation.go:196-229` (every role required; all `threatCalculation` values and `healRadius` must be positive).
- Problem: A second source of truth for every AI knob, reachable only when config is not loaded (tests). The file itself says at `:114-116` that mirrored fallbacks "silently rot", then keeps eight of them.
- Direction: Read the validated config directly; delete the `if > 0` fallbacks. Tests that need values load the fixture config (or set the template struct).
- Effort: S. Risk: low.

##### [Low] Dead constructor plumbing and oversized helper struct
- Evidence: `cache *combatstate.CombatQueryCache` is stored in `ThreatLayerBase` (`threat_layers.go:14`), `CompositeThreatEvaluator` (`threat_composite.go:17`), `FactionThreatLevel` and `SquadThreatLevel` (`dangerlevel.go:80,121`) and threaded through six constructors, but never read (only passed on). `UnitCombatData` (`threat_queries.go:15-22`) has six fields; only `AttackType` and `AttackRange` are read, and `GetUnitCombatData` returns nil for a unit lacking `Attributes` it never uses. Its doc comment claims it is used by `evaluation`, which cannot import `behavior`.
- Direction: Drop the `cache` parameter everywhere in this package; reduce `GetUnitCombatData` to the two values used.
- Effort: S. Risk: low.

#### Mechanics notes
- **Support and positional layers are numerically inert in AI scoring.** `GetRoleWeightedThreat` (`threat_composite.go:74-77`) sums four terms on very different scales: melee threat and ranged pressure are sums of squad power (the shipped config normalizes engagement pressure at 200 and sets the retreat-safe threshold at 10, `resources/assets/gamedata/aiconfig.json`), while positional risk is a 0-1 blend × 0.5 (`threat_positional.go:322-333`) and support value is 0-1 per wounded ally × at most 1 (`threat_support.go:52,75`). Against hardcoded bonuses of 5-20 in `mind/ai`, the 333-line positional layer contributes under one point. Consequence: three of the five `aiCompetence` difficulty knobs (`templates/difficulty.go:128-132`: flanking range, isolation threshold, retreat threshold) and the positional weight change essentially nothing. (Confidence: medium-high; derived from the formulas' ranges, not measured in play.) Decide one of: normalize all layers to a common range before weighting, or keep these layers for the threat overlay only and stop presenting them as AI inputs.
- **"Melee" and "ranged" layers both carry roughly whole-squad power for mixed squads.** `evaluation.CalculateSquadPowerByRange` counts every unit at range 1 and counts melee units at any range up to move+1 (`mind/evaluation/power.go:267-284`); `threat_combat.go:91,107` then paint `ThreatByRange[1]` as melee and `ThreatByRange[maxRange]` as ranged. A squad with one archer paints its full power twice, and `engagementPressure` (`threat_positional.go:224`) adds the two together.
- Heal-type units count as a ranged threat (`threat_queries.go:72`).

#### Dead code
- `CompositeThreatEvaluator.GetOptimalPositionForRole` (`threat_composite.go:82-104`): test-only callers.
- `GetCombatLayer`, `GetPositionalLayer` (`threat_composite.go:106-109,116-119`): no callers.
- "backward compatible" section banners in `threat_combat.go:119-131` refer to a compatibility need that no longer exists.

#### Performance
- `GetRoleBehaviorWeights` and `getPositionalRiskWeights` are looked up per tile per squad (config scans); `hasUnitsWithAttackType` + `getMaxRangeForAttackTypes` walk each enemy squad's units twice per layer per update. Minor individually; they disappear with the ai-package hoisting change.

#### Leave alone
- `PaintThreatToMap` with falloff functions, and the layer-per-file organization.
- Sparse storage with accessor-supplied defaults in `threat_positional.go` is well documented and correct.
- `ThreatSnapshot` / `EvaluateAt` as the single GUI-facing read (defined in `combatservices`, implemented here) is a good inversion.

---

### mind/evaluation (465 lines, tests: yes)

**Role:** Rates unit and squad combat strength ("power") for AI threat tables, encounter budgets and raid deployment ordering.
**Verdict:** Needs attention (small package, small fixes). The same formula is written twice with a divergence, and there is a "profile" abstraction with exactly one profile.

#### Design findings

##### [High] Unit power is implemented twice and the two copies disagree
- Evidence: entity path `mind/evaluation/power.go:17-53` + `:57-122`; template path `:297-344`. The template path re-implements defensive power inline (`:305-317`) and adds a heal term (`:330-334`) that the entity path lacks. Comments at `:242` and in `threat_queries.go:27` mention a leader bonus that neither path applies; `leaderBonus` exists in `powerconfig.json` and is validated (`templates/validation.go:279`) but never read.
- Problem: The same unit rates differently before and after it is spawned. Spawning budgets enemies with the template formula and measures the player with the entity formula (see mechanics note).
- Direction: One pure function taking the inputs both paths have (attributes, role, attack type, cover value). The entity path extracts those from components; the template path from template fields. Decide whether healing counts and apply it in the one place. Remove `leaderBonus` from JSON/schema or use it.
- Effort: S. Risk: medium (changes enemy budgets slightly; tests exist).

##### [Medium] "Power profile" is a one-entry abstraction with three spellings
- Evidence: one profile in `resources/assets/gamedata/powerconfig.json`; `GetPowerConfigByProfile("Balanced")` at `mind/behavior/dangerlevel.go:139`, `DefaultPowerProfile` const at `mind/spawning/types.go:35`, and `"balanced"` (lowercase) at `campaign/raid/deployment.go:68`. The lookup is case-sensitive (`power_config.go:40-41`) and falls back silently to hardcoded weights (`:50-56`).
- Problem: The raid call already misses the JSON profile and uses the code defaults; today they are equal, so behaviour is unchanged, but the first JSON tuning pass will make raid deployment diverge without any error. `PowerConfig` also mirrors the template profile struct field for field, and a new one is allocated per squad per threat update.
- Direction: Replace with a single accessor returning the (validated) config; no name parameter. If multiple profiles are ever needed, reintroduce with typed constants and an error on miss.
- Effort: S. Risk: low.

##### [Low] Unused parameters, duplicated bonus code, hardcoded fallbacks, oversized exports
- Evidence: `config` unused in `CalculateOffensivePower` (`power.go:57`), `CalculateDefensivePower` (`:73`); `attr` and `config` unused in `CalculateUtilityPower` (`:97-102`). Composition bonus computed in `CalculateSquadCompositionBonus` (`:169-191`) and again inline in `CalculateSquadPowerByRange` (`:221,238-240,287`). `healthRatio` arithmetic at `:75-78` reduces to `CurrentHealth`. Fallback tables at `roles.go:30-40,51-63` and `power_config.go:24-29` duplicate validated JSON. Ten exported identifiers are used only inside the package.
- Direction: Trim signatures, reuse the composition helper, delete fallbacks, unexport.
- Effort: S. Risk: low.

#### Mechanics notes
- `CalculateSquadPower` applies composition bonus and a health exponent (`power.go:153-166`); `CalculateSquadPowerByRange` applies composition only (`:286-290`). The AI therefore rates a half-dead squad almost as dangerous as a fresh one (only per-unit current HP in the defensive term reduces it), while encounter budgeting squares the health ratio.
- Naming: "power" here means combat strength; `tactical/powers/*` uses "power" for abilities. `PowerConfig`, `powerconfig.json` and `powercore` sit one directory apart in meaning. Worth a rename to "strength"/"rating" if this package is touched anyway.

#### Leave alone
- The offensive × hit × crit and effective-HP formulas themselves, and the package's position below `behavior` and `spawning`.

---

### mind/combatlifecycle (788 lines, tests: yes, rewards only)

**Role:** Shared combat start/exit pipeline: starter and resolver contracts, faction enrollment helpers, reward granting.
**Verdict:** Needs attention. The core idea (every combat type supplies a starter and a resolver; all rewards go through `Grant`) is the right abstraction and is used by three combat types. It sits in the wrong layer, has accumulated one-implementation interfaces and dead fields, and its casualty helper cannot work.

#### Design findings

##### [High] Package is tactical infrastructure filed under `mind`
- Evidence: imports are `core/{common,config,coords}` and `tactical/{combat/combatstate, powers/perks, powers/progression, powers/spells, squads/squadcore, squads/unitprogression}` only. Importers: `campaign/raid` (6 files), `gui/guicombat`, `gui/guioverworld`, `setup/gamesetup`, `tools/combat_analysis/combat_simulator`, `mind/encounter`.
- Problem: Nothing here is AI. `campaign/raid` and the simulator reach into `mind` for reward and enrollment code.
- Direction: Move to `tactical/combat/combatlifecycle` (path change only), and absorb `EncounterService` as described in the slice-level section.
- Effort: M (mechanical, ~20 import sites, plus `PROJECT_LAYOUT.md` / `PACKAGE_DEPENDENCIES.md` / CLAUDE.md). Risk: low.

##### [Medium] Two interfaces exist only because the service lives in another package
- Evidence: `CombatTransitioner` (`start.go:92-96`), sole implementation `EncounterService.TransitionToCombat`; `ExitHooks` (`exit.go:127-143`), sole implementation `encounter.EncounterExitHooks`. Of the three hook methods, `OnAfterResolution` operates on a sprite that never exists and a flag nobody reads, and `OnBeforeTeardown` repeats what teardown does anyway (both detailed under mind/encounter); only `OnRestorePlayer` does real work.
- Problem: `ExecuteCombatStart` is three lines of forwarding; `ExecuteCombatExit` takes five parameters (`ctx`, `hooks`, `teardown`, `onHistory`, manager) to sequence steps whose only caller is `EncounterService.ExitCombat`. A reader has to hop between two packages and three types to follow one exit.
- Direction: With the service in this package, `StartCombat(starter)` and `ExitCombat(...)` become two methods that read top to bottom. Delete `CombatTransitioner`, `ExitHooks`, the `onHistory` callback and `EncounterExitHooks`. Keep `CombatStarter`, `CombatResolver` (three real implementations each) and `CombatTeardown` (one method, real inversion).
- Effort: M. Risk: medium (no tests on the exit path).

##### [Medium] `CombatSetup` / `ResolutionContext` / `EncounterOutcome` carry fields nobody reads
- Evidence (all grepped repo-wide):
  - `CombatSetup.PlayerFactionID`, `.EnemyFactionID` (`start.go:56-57`): written by three constructors, never read.
  - `CombatSetup.ThreatID`, `.ThreatName` (`start.go:62-63`): read only into `CompletedEncounter`, which is write-only (see encounter).
  - `CombatTypeDebug` (`start.go:34`): never assigned; debug encounters are built with `NewOverworldSetup` and a nil resolver.
  - `EncounterOutcome.DefeatedFactions` (`exit.go:54`): never read. `.VictorFaction`, `.VictorName`, `.RoundsCompleted`: read only into the write-only history. The struct is a field-for-field copy of `combatservices.VictoryCheckResult`, made at `gui/guicombat/combatmode.go:139-145`.
  - `ResolutionResult.Description` (`exit.go:110`): set by every resolver, never read.
- Problem: The three constructors in `setup_constructors.go` take 8-9 positional arguments, five of them `ecs.EntityID`; the garrison starter passes `TargetNodeID` for both `threatID` and `defendedNodeID` (`mind/encounter/starters.go:133-135`). Most of that surface feeds nothing.
- Direction: Delete the dead fields; `EncounterOutcome` reduces to what resolvers use (`IsPlayerVictory`) or is replaced by passing `VictoryCheckResult`. With five or six live fields, build `CombatSetup` with a struct literal and drop `setup_constructors.go`.
- Effort: S-M. Risk: low.

##### [Medium] Teardown stripping is written twice
- Evidence: `mind/combatlifecycle/teardown.go:48-63` (`StripCombatComponents`) and `tactical/combat/combatservices/combat_service.go:218-226` perform the same three calls. The only caller of the exported one is `mind/encounter/exit_hooks.go:63`, which is redundant (see encounter).
- Direction: Delete `StripCombatComponents` with the hook. If a second caller ever appears, `CombatService` exposes its loop.
- Effort: S. Risk: low.

##### [Low] GUI mode key in the lifecycle package
- Evidence: `start.go:75-90` (`PostCombatReturnRaid = "raid"`), relayed by `EncounterService.TransitionToCombat` (`mind/encounter/encounter_service.go:190-192`) through `ModeCoordinator.SetPostCombatReturnMode` into `TacticalState`, read at `gui/guicombat/combat_turn_flow.go:225-229`.
- Problem: A string that must match a GUI mode registration name is defined in a package described in its own comments as "domain-free".
- Direction: The GUI maps `CombatType` to a mode name itself; the lifecycle only exposes the type.
- Effort: S. Risk: low.

##### [Low] Satellite RNG for XP rolls
- Evidence: `reward.go:17-24`. `xpRNG` is seeded once at package initialization from the shared generator and is not affected by later `common.SetRNGSeed` calls (`world/worldgen/gen_rooms_corridors.go:47`, `world/garrisongen/generator.go:39`); the simulator has to call `SetXPRNG` separately (`tools/combat_analysis/combat_simulator/main.go:28-29`).
- Problem: A second random stream with its own seeding rule, existing only because `unitprogression.AwardExperience` takes a `*math/rand.Rand`.
- Direction: `AwardExperience` draws from the shared `common` RNG (or takes a small interface the shared RNG satisfies); delete `xpRNG` and `SetXPRNG`.
- Effort: S. Risk: low.

##### [Low] `ApplyHPRecovery` is in the wrong file and package
- Evidence: `teardown.go:26-38`; all three callers are in `campaign/raid` (`recovery.go:27,40`, `raidrunner.go:194`).
- Direction: Move next to the other squad health helpers in `squadcore` (or into `raid`).
- Effort: S. Risk: low.

#### Probable bugs
- **`CountDeadUnits` always returns 0 at combat exit (confidence: high).** `casualties.go:22-32` counts units with `CurrentHealth <= 0` in the given squads. But a destroyed squad and all its units are disposed during combat (`tactical/combat/combatcore/combatactionsystem.go:194-199` → `tactical/combat/combatdisposal/squad_removal.go:37-38` → `squadcore.DisposeSquadAndUnits`), and dead units in surviving squads are disposed after every attack (`combatactionsystem.go:202-207`). By the time a resolver runs there are no dead unit entities left to count. Effect is described under mind/encounter (threats can never be weakened by winning). Raid already uses the workable approach: snapshot alive counts before combat and diff afterwards (`campaign/raid/raidrunner.go:222-233`).
- **Arcana/Skill-only rewards are dropped (confidence: high, impact currently nil).** `exit.go:121` calls `Grant` only if gold, XP or mana is positive, but `Reward` also has `ArcanaPts` and `SkillPts`. `Grant` already skips zero fields, so the guard is both redundant and wrong; delete it. No current resolver produces such a reward.

#### Dead code
- Fields and enum value listed in the dead-fields finding; `PostCombatReturnDefault` is used only inside `PostCombatReturnMode`.

#### Leave alone
- `Reward` / `GrantTarget` / `Grant` as the one path for all rewards. This is the best abstraction in the slice.
- `CombatStarter` returning a rollback closure, `CombatResolver`, `DetermineExitReason`.
- `CreateFactionPair` / `EnrollSquadsAtPositions` / `MarkSquadsDeployed`: used by overworld, garrison and raid; correct level of sharing.

---

### mind/encounter (1,050 lines, tests: no)

**Role:** Tracks the active combat session, drives the mode switch into and out of combat, and implements overworld and garrison-defense starters, resolvers and rewards.
**Verdict:** Needs significant rework. It mixes a generic session service, overworld campaign rules and GUI mode driving; a meaningful share of it is dead; and it contains the slice's three gameplay-affecting bugs.

#### Design findings

##### [High] Three responsibilities in one package, in the wrong layer
- Evidence: generic session (`encounter_service.go`, `types.go:18-45,73-81`), used by raid via `campaign/raid/raidrunner.go:30,55,186`; overworld rules (`resolvers.go`, `starters.go`, `encounter_trigger.go`, `encounter_setup.go`, `rewards.go`, importing `campaign/overworld/{core,garrison,ids,threat}`); GUI driving (`ModeCoordinator`, `types.go:20-27`, six methods implemented by `gui/framework/coordinator.go:175-204`).
- Problem: `campaign/raid` imports `mind/encounter` for the service while `mind/encounter` imports `campaign/overworld` for the rules. Overworld mutations (node intensity, influence radius, node ownership) are written in `mind`.
- Direction: The split described in the slice-level section. No new files are needed: `encounter_service.go` + the generic half of `types.go` move to `combatlifecycle`; the rest moves as-is to `campaign/overworld/encounter`.
- Effort: M. Risk: medium (no tests).

##### [High] Encounter history is write-only, and a chain of data exists only to feed it
- Evidence: `encounter_service.go:36-37,59-60,137-154`; `types.go:47-64`. `es.history` is appended to and trimmed and never read (grep: three hits, all in the append block). The type comment (`:22`) advertises "Analytics (win rate, last encounter)" that do not exist.
- Problem: Feeding it are `CombatSetup.ThreatID` / `ThreatName`, `ActiveEncounter.StartTime`, three `EncounterOutcome` fields, the `onHistory` callback in `ExecuteCombatExit`, `OverworldEncounterData.Name` with `getEncounterDisplayName` (`encounter_trigger.go:21-33,65-67`), and a second copy of the same name formatting in `gui/guioverworld/overworld_action_handler.go:152-157`.
- Direction: Delete the history, `CompletedEncounter`, and everything listed that has no other reader. `ExitCombat` then only needs to clear `activeEncounter`.
- Effort: S. Risk: low.

##### [High] Encounter entities are parameter objects stored in ECS and never disposed
- Evidence: `encounter_trigger.go:15-19` creates an entity with only `OverworldEncounterComponent`; the caller passes its ID to a starter (`gui/guioverworld/overworld_action_handler.go:145-165,180-195,226-237`), whose `Prepare` immediately looks it up again and validates it (`validators.go:14-33`). Repo-wide, `OverworldEncounterComponent` is added once and read twice; nothing disposes the entity, including on failed starts.
- Problem: One leaked entity per combat for the life of the session. The entity never has a `Renderable` (the only `AddComponent(RenderableComponent…)` sites are commanders, units, squads, the player and the template factory), so the sprite hide/rollback in `starters.go:31-45` and `restoreEncounterSprite` / the visibility half of `markEncounterDefeated` in `exit_hooks.go:66-86` never do anything. `IsDefeated` is read only by that dead restore path. All three `Trigger*` functions return an `error` that is always nil.
- Direction: Starters take the four values they need (level, encounter type, threat node ID, attacking faction) as fields. If a unique encounter ID is still wanted for `FactionData.EncounterID` (used by `IsEnemySquadInEncounter`), use the threat node ID or a counter, as raid already uses its raid entity ID. Delete `encounter_trigger.go`'s entity creation, `validators.go`, the sprite code, `IsDefeated`, and `OnAfterResolution`.
- Effort: M. Risk: medium (no tests).

##### [High] Combat start has no rollback of ECS side effects
- Evidence: `encounter_setup.go:113-123` creates two faction entities, enrolls player squads (membership component, action states, unit positions) and then may fail enrolling enemies; `spawning.GenerateAttackerSquads` has already created enemy squads by then (`encounter_setup.go:60`). `starters.go:47-51` rolls back only the (non-existent) sprite. `encounter_service.go:185-200` sets tactical state and overwrites the player position before `EnterCombatMode` can fail, and does not restore either.
- Problem: A failed start leaves faction entities, action-state entities, enemy squads and player-squad combat membership in the world. The next combat's `CombatMode.Enter` initializes from every faction it finds (`gui/guicombat/combatmode.go:112-117`), so stale factions would join the turn order. The garrison bug below produces exactly this failure repeatedly.
- Direction: Make `Prepare`'s rollback closure real: on any failure after faction creation, run the same teardown used at exit for the IDs created so far. Reorder `TransitionToCombat` so nothing is mutated until `EnterCombatMode` has succeeded, or restore on failure.
- Effort: M. Risk: medium.

##### [Medium] Global single-subscriber post-combat callback
- Evidence: `encounter_service.go:39-47,96-106,160-162`; subscriber `campaign/raid/raidrunner.go:50-59,354`. The comments on both sides document the hazards: last-write-wins, and the raid must filter out overworld results by checking its own state.
- Problem: A session-wide listener that receives every combat and must decide whether the event is its own.
- Direction: Per-encounter completion: the starter attaches an optional `OnComplete` to `CombatSetup` (next to `Resolver`); `ExitCombat` calls it after teardown. No registration, no filtering, no clear-on-finish.
- Effort: S. Risk: low.

##### [Medium] Overworld resolver re-implements threat rules owned by `campaign/overworld/threat`
- Evidence: `resolvers.go:136-147` (intensity +1, recompute influence radius and magnitude) duplicates `campaign/overworld/threat/system.go:125-131`. The victory path (`resolvers.go:72-74`) changes intensity without recomputing influence, and the defeat path ignores the max-intensity cap applied at `threat/system.go:57`.
- Direction: One `threat` function for "change intensity by delta" that owns radius, magnitude and caps; the resolver calls it for both outcomes.
- Effort: S. Risk: low.

##### [Medium] `commandersForSquads` duplicated verbatim, and unnecessary in both places
- Evidence: `mind/encounter/resolvers.go:165-187` and `campaign/raid/rewards.go:48-70` are identical. In the overworld case the squads come from `RosterOwnerID`'s own roster (`encounter_service.go:131,218-224`), so the result is always that one commander; raid likewise knows its `CommanderID` (`campaign/raid/starters.go:21`).
- Problem: A scan of every commander's roster per squad to rediscover an ID already in hand.
- Direction: Pass the known commander ID into `GrantTarget.CommanderIDs`; delete both copies.
- Effort: S. Risk: low.

##### [Low] Player position is mutated through a shared pointer
- Evidence: `encounter_service.go:178-195` and `exit_hooks.go:47-49` write through `ModeCoordinator.GetPlayerPosition() *coords.LogicalPosition` (which returns `PlayerData.Pos`, `gui/framework/coordinator.go:199-204`).
- Problem: A simulation-side service repositions the camera anchor by aliasing GUI-held state; the interface exposes a mutable pointer.
- Direction: `SetPlayerPosition(pos)` / `GetPlayerPosition() pos` by value on the interface. `SetTriggeredEncounterID` can be removed from the interface outright (see dead code).
- Effort: S. Risk: low.

#### Mechanics notes
- **Rewards are split across the whole roster, not the squads that fought.** `collectPlayerSquadIDs` returns `roster.OwnedSquads` (`encounter_service.go:218-224`), while combat uses only deployed squads (`encounter_setup.go:79-100`). XP is divided evenly over every living unit in every owned squad (`combatlifecycle/reward.go:122-136`), including reserves and garrisoned squads. If unintended, capture the deployed list at start and use it at exit.
- **Garrisoned threat nodes are easier than ungarrisoned ones.** With an NPC garrison, the enemy force is the garrison's single squad of 3-4 random units (`campaign/overworld/garrison/system.go:194-211`); without one, enemies are power-budgeted and scale to 5-10 squads (`resources/assets/gamedata/encounterdata.json`). Encounter level and type are ignored in the garrison branch (`encounter_setup.go:55-58`).
- **Kill-based intensity damage is circular even once the count is fixed.** On victory every enemy squad is dead by definition, so "units killed / 5" (`resolvers.go:72`, `types.go:14`) is just a function of how many enemies were generated, which is itself a function of intensity.
- Overworld combat results reach the player only as `fmt.Printf` (nine unconditional prints in `resolvers.go`); `RewardText` is surfaced only by raid (`gui/guiraid/summary_panel.go:69-71`).

#### Probable bugs
- **Winning an overworld fight never weakens or destroys the threat (confidence: high).** `resolvers.go:44` uses `CountDeadUnits`, which is always 0 at exit (traced under combatlifecycle). So `damageDealt` (`:72`) is 0, intensity is unchanged, the `Intensity <= 0` destroy branch (`:85-105`) is unreachable from a normal win, and every victory takes the "weakened" branch: half rewards (`:108`), growth progress reset, and a log line reporting the same intensity as before.
- **Fighting an NPC-garrisoned node disposes the garrison and leaves the node unattackable (confidence: medium-high).** `encounter_setup.go:55-58` uses `garrisonData.SquadIDs` as the enemy list. Teardown disposes all enemy squads on every exit reason (`combatlifecycle/exit.go:189-191` → `combatservices/combat_service.go:238,334-340`), and destroyed squads are disposed mid-combat anyway. The node's `GarrisonData.SquadIDs` still lists them, `IsNodeGarrisoned` stays true so the faction never re-garrisons (`campaign/overworld/faction/system.go:243-245`), and the next attack fails in `AddSquadToFaction` with "squad N not found" (`tactical/combat/combatstate/combatfactionmanager.go:62-65`), leaking factions as described above. Since victory does not destroy the node (previous bug), this is reachable after any outcome.
- **Auto-deploy can pull garrisoned squads into a field battle (confidence: medium).** When no squad is deployed, `encounter_setup.go:86-93` marks every squad in `roster.OwnedSquads` deployed without checking `GarrisonedAtNodeID`; garrisoned squads remain in `OwnedSquads` (`campaign/overworld/garrison/queries.go:33-41` filters them by that field).
- `exit_hooks.go:36` marks the encounter defeated on `IsPlayerVictory` even when the exit reason is flee (flee takes precedence in `DetermineExitReason`). Harmless today because the flag is dead.

#### Dead code
- `EncounterService.GetEnemySquadIDs` (`encounter_service.go:88-94`): no callers.
- `EncounterService.GetRosterOwnerID` (`:79-86`) and its `EncounterController` entry (`types.go:79`): no callers.
- `ModeCoordinator.SetTriggeredEncounterID` (`types.go:22`) → `TacticalState.TriggeredEncounterID` (`gui/framework/contextstate.go:40`): written, never read.
- `EncounterService.maxHistory` field duplicating the `maxHistoryEntries` const; `IsEncounterActive` used only internally.
- `OnBeforeTeardown` (`exit_hooks.go:55-64`): in garrison defense the garrison squads are enrolled in the player-controlled faction (`starters.go:111-117`), so `CombatService.TeardownCombat` strips them itself (`combat_service.go:218-226`) and they are not in the disposed enemy list. The hook's stated purpose (surviving enemy-squad disposal) does not apply.
- Stale names in comments: "EndEncounter", "generateRandomComposition" (`encounter_trigger.go:36-37`), "SetupBalancedEncounter" (`encounter_service.go:26`), "combat.CombatTransitioner" (`:167`).

#### Leave alone
- The starter/resolver pair per combat type and `assembleCombatFactions` as the shared tail of both overworld starters.
- `CalculateIntensityReward` reading all tunables from `templates.GameConfig.Encounter`.
- `EncounterController` as a narrow GUI port (minus the dead method).

---

### mind/spawning (423 lines, tests: no)

**Role:** Generates enemy squads to a power budget and computes spawn positions around a point.
**Verdict:** Needs attention. Small and focused, but squad assembly silently drops units, archetypes are hardcoded behind JSON-declared names, and it is one of three copies of "build a random squad".

#### Design findings

##### [High] "Random squad from a template pool" exists in three packages
- Evidence: `mind/spawning/squadscreation.go:66-133`; `campaign/overworld/garrison/system.go:189-211` (documented as "power-budget generation" at `:122`, actually 3-4 random units); `setup/gamesetup/initial_squads.go:149-339` (five functions). Each has its own hardcoded grid-slot table, leader pick and `Leadership = 20`.
- Problem: Three slot tables, three leader rules, and the placement bug below exists because none of them asks the squad grid where a unit fits.
- Direction: One helper in `squadcore` that takes a list of unit templates and assigns non-overlapping grid cells honouring width/height (the occupancy logic already exists in `placeTemplateUnit`, `tactical/squads/squadcore/squadcreation.go:329-358`), marks the leader and sets leadership. `spawning` owns "which units, to what budget" and, once it no longer imports `campaign`, the garrison and initial-squad code call it.
- Effort: M. Risk: medium (no tests here; `squadcore` has tests).

##### [Medium] Depends on `campaign` for two integers and a lookup
- Evidence: `squadscreation.go:158,166-168` reads `encounterData.Level`; `composition.go:11-25,40-47` reads `EncounterType` and calls `core.GetNodeRegistry()`. Imports `campaign/overworld/core` and `ids`.
- Problem: This is what stops `garrison` and other campaign code from using the generator, and it is half of the mind→campaign back-edge.
- Direction: `GenerateAttackerSquads(manager, center, sourceSquadIDs, level, squadTypes)`; the caller (overworld encounter code) resolves preferences from the node registry.
- Effort: S. Risk: low.

##### [Medium] Squad archetypes are declared in JSON but defined in a Go switch
- Evidence: `types.go:14-19` constants; `squadscreation.go:136-149` (`melee` = range ≤ 2, `ranged` = range ≥ 3, by literal); JSON declares `squadTypes` and validates preferences against them (`resources/assets/gamedata/encounterdata.json:61`, `templates/validation.go:117-121`). `composition.go:42` special-cases the encounter ID `"default"`.
- Problem: Two lists of type names. A type added to JSON passes validation and then falls through the switch to "all units".
- Direction: Either the JSON squad type carries its filter (attack type / range bounds), or the Go constants are the only list and validation checks against them.
- Effort: S. Risk: low.

##### [Low] Staging struct with unused fields; hardcoded fallbacks; duplicated positions call
- Evidence: `enemySquadSpec.Power`, `.Type`, `.Name` (`types.go:39-45`) are never read; `util.go:96-107` hardcodes a difficulty row and `squadscreation.go:165` a default level 3, both shadowing JSON; `encounter_setup.go:58` repeats the enemy-ring call from `squadscreation.go:34`; the package doc comment still says "Package generation" (`types.go:1`).
- Direction: Return IDs and positions directly; drop fallbacks (validated at load); export one `GenerateEnemyPositions(center, n)`.
- Effort: S. Risk: low.

#### Mechanics notes
- **Enemy budget and player strength are measured on different scales.** Player squads are rated with `CalculateSquadPower` (composition multiplier 0.8-1.3 and a health exponent; `util.go:138-143`); enemy squads are filled by summing `EstimateUnitPowerFromTemplate` (no composition multiplier, plus a heal term; `squadscreation.go:94-109`). The target is the player's *average squad* power, applied to each of N enemy squads where N comes from the difficulty row, not from how many squads the player fields.
- Squads are created at their position and then enrolled at the same position again (`squadscreation.go:126-132` then `encounter_setup.go:121`).

#### Probable bugs
- **Generated squads silently lose units (confidence: high).** The slot table at `squadscreation.go:87` includes `{3,0}` and `{3,1}`, but the squad grid is 3×3 (`tactical/squads/squadcore/squadcomponents.go:19`); `placeTemplateUnit` rejects them with a console warning and skips the unit (`squadcreation.go:345-348`). Level-5 encounters allow 8 units per squad (`encounterdata.json:56`), so units 7 and 8 never spawn. Separately, slots are assigned one cell apart regardless of unit size; `monsterdata.json` has 2-wide and 2×2 units, so a 2×2 unit placed first blocks the next three slots and those units are skipped too (`squadcreation.go:350-358`). The power budget was computed for units that do not exist.
- **Spawn positions can coincide or be unwalkable (confidence: medium).** `util.go:16-41` clamps to map bounds only; near an edge several squads clamp to the same tile, and no walkability or occupancy check is made before `AddSquadToFaction`.

#### Leave alone
- `GeneratePositionsAroundPoint` as a small pure helper (apart from the missing validity check).
- `getDifficultyModifier` applying the global difficulty overlay in one place.

---

## 16. Campaign (overworld and raid)

All paths are repo-relative. "Unused" means the identifier was grepped across the whole repo (including `tools/` and tests) and has no reader/caller outside its own definition. Bugs carry a confidence level and the trace that supports it.

### Layer overview

**Two unrelated game modes live here.** `SetupOverworldMode` (game_main/setup.go:87) boots the overworld simulation; `SetupRoguelikeMode` (game_main/setup.go:116) boots the raid. They never run together, share no campaign code, and meet only in `mind/combatlifecycle`. `campaign/raid` imports nothing from `campaign/overworld` and vice versa. That separation is clean and should be kept.

**The vocabulary collides across the two modes.** "Raid" means the player's dungeon run (`campaign/raid`) and also an NPC faction attacking a player node (`core.PendingRaid`, `IntentRaid`, `faction.ExecuteRaid`, `EventFactionRaid`, `OverworldActionHandler.HandleRaid`). "Garrison" means the enemy squads in a raid room (`raid.GarrisonSquadData`, `raid.GenerateGarrison`), squads stationed on an overworld node (`overworld/garrison`, `core.GarrisonData`, `SquadData.GarrisonedAtNodeID`), and the map generator (`world/garrisongen`). `gui/guioverworld/overworld_action_handler.go:177` even names a `*core.PendingRaid` parameter `raid`. Direction: rename the overworld concept (for example `PendingRaid` → `PendingNodeAttack`, `IntentRaid` → `IntentAttack`) so "raid" only means the roguelike run. Effort S, risk low.

**Ten overworld sub-packages is the wrong granularity.** 3,300 lines are split into ten packages of 50–510 lines each (apart from `core` at 1,177). The split does not give encapsulation:

- Every component type, tag, view, config accessor, event type, and the registry live in `core`, because every sub-package and five outside packages need them. The sub-packages own functions but not their data. `core.GarrisonData` is mutated by `garrison`, read by `faction` and `mind/encounter`; `core.OverworldNodeData` is mutated by `node`, `threat`, `garrison`, and `mind/encounter/resolvers.go`.
- The dependency order is strictly linear (`ids` ← `core` ← `node`/`influence`/`garrison` ← `threat` ← `faction` ← `tick`), so there is no cycle the split is protecting against. `tick/tickmanager.go:3-4` says the package exists to "break the circular dependency", but that cycle only exists because of the split.
- Functions are exported only because their sole caller is in a sibling package: `faction.ExpandTerritory`, `FortifyTerritory`, `ExecuteRaid`, `AbandonTerritory`, `EvaluateFactionIntent`, `threat.EvolveThreatNode`, `ExecuteThreatEvolutionEffect`, `SpreadCorruption`, the four `core.Get*ScoringParams` getters (used only by `faction/scoring.go`).

Direction: collapse `core`, `node`, `threat`, `influence`, `faction`, `garrison`, `tick` (and `victory` if it is revived, see below) into one `campaign/overworld` package of roughly 2,700 lines, keeping the existing file-per-subsystem names (`faction_system.go`, `threat_system.go`, ...). Keep `ids` (it has a real reason, below) and `overworldlog` (dependency-free, self-contained). This removes the `core` dumping ground, lets most of the above functions become unexported, and deletes the tuple-returning config getters. Cost: an import-path rewrite in about 35 files outside the slice (`gui/guioverworld`, `gui/guinodeplacement`, `mind/encounter`, `mind/spawning`, `setup/gamesetup`). Effort M (mechanical), risk low. If the merge is not wanted, the minimum is: move faction-only config getters into `faction`, and stop adding to `core`.

**What `ids` works around.** `templates` needs typed IDs in its JSON DTOs (templates/schema_overworld.go:153-207) and `core` imports `templates`, so the types cannot live in `core`. A dependency-free leaf package is a reasonable answer and should stay. The oddity is direction: the generic data loader (`templates`) imports a `campaign/...` path. If that bothers the dependency doc, move the package to `core/ids` or similar; no code change beyond the path.

**What `core` works around.** Nothing structural; it is the shared kernel that the sub-package split forced into existence. It also absorbed things that are not shared at all (faction scoring getters, the event recorder session API, walkability grid).

**The overworld cannot be saved, won, or lost.** There is no overworld save chunk (`setup/savesystem/chunks/` has commander, gear, map, player, progression, raid, squad). The `victory` package is imported by nothing, so `CheckVictoryCondition` never runs and `TickStateData.IsGameOver` is never set to true. These are facts about the current state, not necessarily defects, but they decide how much of the overworld code is live. See the `victory` and cross-package sections.

---

### campaign/overworld/core (1,177 lines, tests: no)

**Role:** Shared kernel for the overworld: all component types and tags, the node/encounter registry, event log, logger, config accessors, walkability grid, and assorted helpers. **Verdict:** Needs attention. It is a grab-bag whose data model carries several parallel identity schemes and a noticeable amount of write-only state.

#### Design findings

##### [High] Faction identity has four representations that must be kept in sync by hand
- Evidence: `FactionType` int enum + `String()` (core/types.go:56-94); `ids.FactionID` string documented as "matches FactionType.String() at runtime" (ids/ids.go:24-26); `ids.OwnerID` built from the same string (core/types.go:99-106); lowercase names in `initialsetup.json` mapped by a hand-written switch (setup/gamesetup/initial_factions.go:42-54, which has no case for `FactionBeasts`); `templates.FactionArchetypeTemplates` keyed by `FactionType.String()` (faction/archetype.go:23-24). `OverworldFactionData.FactionID` (core/components.go:42) is a fifth thing: the entity's own ID, never read.
- Problem: Adding or renaming a faction means editing an enum, a `String()` switch, a second switch in gamesetup, and three JSON files, with only string equality tying them together. The comment on `ThreatType` says new types can be added "via JSON without code changes" (core/types.go:15), which is not true for factions.
- Direction: Make `ids.FactionID` the only faction identity. Delete the `FactionType` enum and its `String()`; `OverworldFactionData` stores a `FactionID`; `OwnerID(factionID)` is the conversion. `initialsetup.json` uses the same IDs as `nodeDefinitions.json`, validated against the registry at load. This also removes the zero-value sentinel bug in `victory` (below).
- Effort: M. Risk: medium (touches `mind/encounter`, `mind/spawning`, `gui/guioverworld`, gamesetup; no tests in `core`).

##### [High] `ThreatType` duplicates `ids.NodeTypeID`
- Evidence: `type ThreatType string` (core/types.go:16) holds the same value as `NodeTypeID`. Conversions back and forth at threat/system.go:62, 120, 145, 151; mind/encounter/resolvers.go:144; `AsNodeTypeID` (core/types.go:42); `GetNodeByType` is documented as "the same as GetNodeByID, kept for API compatibility" (core/node_registry.go:167-172).
- Problem: Two names for one string, with a second set of registry methods (`GetNodeByType`, `GetDisplayName`, `GetOverworldParams`, `GetEncounterTypeID`, `GetEncounterForThreatType`) and wrappers (`ThreatType.String`, `ThreatType.EncounterTypeID`, `MapFactionToThreatType`, `GetThreatTypeParamsFromConfig`, `ThreatTypeParams`) that exist only to bridge them.
- Direction: Delete `ThreatType` and `ThreatTypeParams`; callers use `NodeTypeID` and read `BaseRadius`/`CanSpawnChildren` straight off `*NodeDefinition`. The registry shrinks to `GetNodeByID`, `GetPlaceableNodeTypes`, `GetEncounterByID`, `GetEncountersByFaction`, `GetEncounterByTypeID`, and one "threat node type for faction" lookup.
- Effort: S–M. Risk: low.

##### [High] "Threat" is defined two different ways
- Evidence: `Category == NodeCategoryThreat` drives growth and counting (threat/system.go:88, threat/queries.go:14/27/42, victory/queries.go:39). `ids.IsHostileOwner(OwnerID)` drives attackability, placement blocking, and influence classification (core/utils.go:83-95, influence/effects.go:22-23). `garrison.TransferNodeOwnership` changes `OwnerID` only (garrison/system.go:226).
- Problem: After an NPC faction captures a player settlement, the node is hostile-owned but still `Category = settlement`: `GetThreatNodeAt` returns it (so the player can attack it and `OverworldCombatResolver` treats it as a threat with intensity 0 and destroys it), while `CountThreatNodes`, growth, and the threat stats ignore it. `OverworldNodeData.Category` is commented "Cached" (core/components.go:141); it is a copy of `NodeDefinition.Category`.
- Direction: Pick one definition and name it. Either "threat = hostile-owned node" everywhere (and growth applies only when the node definition has a growth rate), or keep category and make capture an explicit conversion. Then expose a single `IsThreat(nodeData)` and use it in every filter above.
- Effort: S. Risk: medium (changes what captured nodes do; no tests).

##### [Medium] Components carry write-only and duplicated fields
- Evidence (all grepped, no readers): `OverworldFactionData.FactionID`, `.Disposition`, `.GrowthRate`, `.CurrentIntent` (duplicate of `StrategicIntentData.Intent`, written together at faction/system.go:132-134); `StrategicIntentData.TargetPosition`, `.Priority` (always 0.5); `InfluenceData.BaseMagnitude` (written at node/system.go:68, threat/system.go:131, mind/encounter/resolvers.go:146; never read, so `CalculateBaseMagnitude` and `BaseMagnitudeMultiplier` are inert); `OverworldNodeData.NodeID` (entity's own ID); `EventLog.Unread`. `OverworldFactionData.TerritorySize` duplicates `len(TerritoryData.OwnedTiles)` and is hand-maintained with `++`/`--` (faction/system.go:196, 318). `OverworldNodeData.IsContained` is read (threat/system.go:94, gui/guioverworld/overworld_formatters.go:37) but never set to true.
- Problem: Fields that look meaningful but do nothing make the data model hard to trust, and the duplicated ones can drift.
- Direction: Delete the write-only fields. Replace `TerritorySize` with `len(OwnedTiles)`. Decide whether containment is a mechanic (see Mechanics notes); if not, delete `IsContained`, `GetContainmentSlowdown`, and the config/difficulty keys behind it.
- Effort: S. Risk: low.

##### [Medium] Influence radius is derived state recomputed in three places, and one path forgets
- Evidence: `radius = BaseRadius + Intensity` at node/system.go:59-63, threat/system.go:130, mind/encounter/resolvers.go:145. The weakened-but-not-destroyed victory path lowers `Intensity` without touching `InfluenceData` (mind/encounter/resolvers.go:72-74, 107-128).
- Problem: Cached derived value with three writers. See Probable bugs.
- Direction: One `SetNodeIntensity(entity, nodeData, n)` in the overworld package that clamps to max and updates `InfluenceData`; all three sites call it. Or drop the cached radius and compute it in `collectAllNodes` (influence/queries.go:49) and the renderer.
- Effort: S. Risk: low.

##### [Medium] The event system has no in-game consumer
- Evidence: `LogEvent` (core/events.go:85) appends to `EventLog.Events` and forwards to the recorder. Nothing reads `EventLog.Events` (the GUI event panel is fed by its own `logEvent(string)`, gui/guioverworld/overworldmode.go:300-314). The recorder is gated on `config.ENABLE_OVERWORLD_LOG_EXPORT = false` (core/config/config.go:25). `overworldlog` summaries key off `Data["threat_type"]`, `"intensity"`, `"tiles_gained"`, but every `LogEvent` call in the slice passes `nil` data; only `mind/encounter/resolvers.go` passes data.
- Problem: About 25 call sites build `fmt.Sprintf` descriptions for a ring buffer nobody reads and an exporter that is off, while the player-visible log shows none of it (threat spawned, faction expanded, node captured are invisible in the UI). `core` importing `overworldlog` and `core/config` for the recorder session API (`StartRecordingSession`, `FinalizeRecording`, `ClearRecording`, events.go:106-136) puts file export in the simulation kernel, and `tick.CreateTickStateEntity` starts a recording session as a side effect (tick/tickmanager.go:33).
- Direction: Decide what events are for. If they are meant for the player, have the GUI panel read `EventLog` and drop its private text log. If they are a debug export, keep `LogEvent` → recorder and delete `EventLog`. Either way move session start/finalize out of `core` to the caller that owns the session (gui or game_main), and delete the `Data` map parameter or populate it where the summaries expect it.
- Effort: M. Risk: low.

##### [Medium] Registry lookups iterate Go maps, so results are order-random
- Evidence: `GetPlaceableNodeTypes` (core/node_registry.go:175-183) feeds the placement UI list and its default selection (gui/guinodeplacement/nodeplacementmode.go:89-93) unsorted. `GetEncountersByFaction` (213-221) is indexed with `common.RandomInt` (threat/system.go:32). `GetEncounterForThreatType` returns `encounters[0]` (207). `GetThreatTypeForFaction` returns the first map hit (237-245).
- Problem: UI order changes between runs, and seeded RNG does not give reproducible encounters because the slice being indexed is in random order.
- Direction: Build sorted slices once in `newNodeRegistry` (by ID or JSON order) and have these methods return them.
- Effort: S. Risk: low.

##### [Low] Config accessor layer is uneven
- Evidence: core/config.go wraps some template values (tuple-returning scoring getters, difficulty-adjusted values) while sibling code reads `templates.OverworldConfigTemplate` directly (faction/scoring.go:17/22/27, faction/system.go:128/169/294, threat/system.go:93/154).
- Problem: Two ways to read the same config in the same function. The tuple getters add indirection without adding meaning.
- Direction: Keep only the difficulty-adjusted accessors (they compute something). Delete the four `Get*ScoringParams` getters and read the struct directly.
- Effort: S. Risk: low.

##### [Low] Package-level mutable state
- Evidence: `WalkableGrid` (core/walkability.go:9), `defaultContext` (core/events.go:72), `globalNodeRegistry` lazy init (core/node_registry.go:64-73), `logger` (core/logger.go:31).
- Problem: Acceptable for a single-instance game, but `WalkableGrid` is real map state held outside both the ECS and `GameMap` and must be rebuilt by hand in four places (game_main/setup.go:123, 202; setup/gamesetup/bootstrap.go:150; gui/guiexploration/exploration_panels_registry.go:60).
- Direction: Leave the registry and logger. For walkability, have callers ask the `GameMap` (or pass a walkability func in) instead of maintaining a second grid. See `world/worldmapcore` (walkability has three sources of truth).
- Effort: M. Risk: medium.

#### Probable bugs
- **Influence radius not updated when a threat is weakened** (high). mind/encounter/resolvers.go:72-128 lowers `nodeData.Intensity` but leaves `InfluenceData.Radius` at the old value; the defeat path at 141-147 does update it. Radius then stays too large until the next evolution.
- **`GetFortificationStrengthGain` ignores difficulty** (high). core/config.go:78-84 does `int(scale)` with no base value; scales are 0.6 / 1.0 / 1.4 (templates/difficulty_test.go:103, 134), which truncate to 0→1, 1, 1. The result is always 1.
- **Placement UI order and default node type vary per run** (high). Map iteration in `GetPlaceableNodeTypes`, no sort in `gui/guinodeplacement`.
- **`GetNodeByID` never returns nil when a default exists**, so the `nodeDef == nil` "unknown node type" checks at node/system.go:29, 81 and node/validation.go:83 never fire; an unknown ID silently creates a "default" node (medium; depends on `DefaultNodeTemplate` being present, which the shipped data has).

#### Dead code
- `TravelStateComponent`, `TravelStateTag`, `TravelStateData` (core/components.go:18, 25, 76-85; init.go:34, 46, 55).
- `VictoryStateComponent`/`VictoryStateTag`/`VictoryStateData`/`VictoryCondition` are only used by the dead `victory` package.
- `NodeTypeDisplayName` (core/types.go:47), `InitWalkableGrid` as a public entry (only called by `InitWalkableGridFromPositions`), `SetLogger` (core/logger.go:34, no caller), `ThreatBeastNest`, `ThreatOrcWarband` constants (core/types.go:24-25).
- `EventFactionDefeated` is never emitted. `EventLog.Unread` never read.
- All seven `manager.WorldTags[...]` registrations in core/init.go:52-58 have no reader (only `"players"` and `"raidstate"` are read anywhere in the repo).
- `InfluenceData.BaseMagnitude`, `CalculateBaseMagnitude` (core/utils.go:38): written, never read.

#### Leave alone
- `OverworldNodeView` / `OverworldFactionView` and the unified `OverworldNodeData` (one component for threats, settlements, POIs). This is the right model.
- `OverworldLogger` with `Debugf`/`Warnf`. Small and does its job; just remove the unused `SetLogger` or use it.
- `ResourceCost` + `CanAfford`/`SpendResources` (core/resources.go).
- Using `GlobalPositionSystem` for node-at-position lookups (core/utils.go:70-107).

---

### campaign/overworld/ids (53 lines, tests: no)

**Role:** Typed string IDs shared by `templates` and the overworld. **Verdict:** Healthy. It exists for a real import-cycle reason and is tiny.

#### Design findings
##### [Low] Unused constants
- Evidence: `NodeTypeTown`, `NodeTypeGuildHall`, `NodeTypeTemple`, `NodeTypeWatchtower` (ids/ids.go:18-21) have no references.
- Direction: Delete. Node types are data.
- Effort: S. Risk: low.

#### Leave alone
- The typed IDs and `IsHostileOwner`/`IsFriendlyOwner`. If `FactionType` is removed (see `core`), `FactionID` becomes the primary faction key and this package gains importance.

---

### campaign/overworld/node (272 lines, tests: no)

**Role:** Create/destroy overworld nodes and validate player placement. **Verdict:** Healthy. Small, coherent, correct use of `RegisterEntityPosition` and `CleanDisposeEntity`.

#### Design findings
##### [Low] Cost is checked in validation and spent in creation by separate lookups
- Evidence: `ValidatePlayerPlacementWithCost` (node/validation.go:74-98) and `CreatePlayerNode` (node/system.go:77-90) each fetch the definition and stockpile; `CreatePlayerNode` skips the cost silently when `stockpile == nil` or `playerEntityID == 0`.
- Problem: Minor duplication, and the "free if no stockpile" branch is a quiet rule.
- Direction: Leave the split (preview vs commit is legitimate); make `CreatePlayerNode` return an error when the stockpile is missing.
- Effort: S. Risk: low.

##### [Low] Distance metric differs from the rest of the overworld
- Evidence: Euclidean in node/queries.go:48-50 and node/validation.go:48-50; Manhattan in influence/queries.go:33.
- Direction: Note only. Use `coords` distance helpers so the choice is visible.

#### Leave alone
- `CreateNode` / `CreateNodeParams` / `DestroyNode`. Good single creation path that `threat.CreateThreatNode` and POI conversion both use.

---

### campaign/overworld/threat (290 lines, tests: no)

**Role:** Threat-node creation, per-tick growth, evolution effects, destruction. **Verdict:** Needs attention, mainly for hardcoded type behaviour and the type-conversion noise inherited from `core`.

#### Design findings
##### [Medium] Evolution behaviour is hardcoded per type behind a data flag
- Evidence: `ExecuteThreatEvolutionEffect` checks the data-driven `CanSpawnChildren` and then switches on `ThreatNecromancer` / `ThreatCorruption` with literal IDs `"necromancer"`, `"corruption"` (threat/system.go:144-167, 157, 211). Spawn radius 3 and 10 attempts are literals (173-176).
- Problem: `CanSpawnChildren: true` on any other node type in JSON does nothing, silently. The data says one thing and the code another.
- Direction: Replace the bool with a small enum in `nodeDefinitions.json` (`"evolution": "none" | "spawn_child" | "spread_adjacent"`) and switch on that. Two behaviours do not need an interface.
- Effort: S. Risk: low.

##### [Low] `currentTick` is passed in and also re-queried
- Evidence: `UpdateThreatNodes(manager, currentTick)` passes the tick, but `EvolveThreatNode`, `SpawnChildThreatNode`, `SpreadCorruption`, `DestroyThreatNode` call `core.GetCurrentTick(manager)` (a `World.Query`) again (threat/system.go:135, 158, 184, 211, 233). Same pattern in `faction`.
- Direction: Thread the parameter through.
- Effort: S. Risk: low.

#### Dead code
- `UpdateThreatNodes` always returns `nil`; the `error` return and the wrapping in tick/tickmanager.go:67-69 are noise. Same for `faction.UpdateFactions`.

#### Leave alone
- `CreateThreatNode` delegating to `node.CreateNode`; the growth loop itself.

---

### campaign/overworld/influence (297 lines, tests: no)

**Role:** Per-tick pairwise influence interactions (synergy, competition, suppression) producing a growth modifier per node. **Verdict:** Healthy in structure; one latent nil dereference and noisy logging.

#### Design findings
##### [Low] Every overlapping pair logs an event every tick
- Evidence: `logInteractionEvent` is called for each pair on each tick (influence/system.go:48, 124-139); the comment says "significant new interactions" but nothing tracks newness.
- Problem: With N overlapping nodes this pushes O(N²) identical events per turn into a 100-entry ring, evicting everything else, and allocates a `Sprintf` string and a map per event (core/events.go:88-90).
- Direction: Delete the logging, or log only when a pair appears that was not present last tick.
- Effort: S. Risk: low.

##### [Low] `InteractionComponent` is fully derived per-tick state stored as a component
- Evidence: cleared and rebuilt every tick (influence/system.go:19, 57-64); only `NetModifier` is consumed (threat/system.go:99-103). `NodeInteraction.Distance`/`TargetID` are read by nothing outside the package except possibly GUI display.
- Direction: Fine as is. If it is never displayed, it could be a `map[EntityID]float64` returned from `UpdateInfluenceInteractions` and passed to `UpdateThreatNodes`. Not worth doing on its own.

#### Probable bugs
- **Nil dereference in `finalizeNetModifiers`** (high as written, low likelihood): `if data == nil || len(data.Interactions) == 0 { data.NetModifier = 1.0; continue }` at influence/system.go:92-95 writes through `data` when it is nil.

#### Leave alone
- `ClassifyInteraction` and the top-2-per-type rule. Clear, data-driven, cheap at the node counts involved. O(N²) pair scan is fine and is documented.

---

### campaign/overworld/faction (512 lines, tests: yes)

**Role:** NPC faction AI: score four intents, pick one, execute it each tick. **Verdict:** Needs attention. Structure is fine; the rules have overlapping spawn paths and several inert pieces.

#### Design findings
##### [Medium] Three threat-spawn paths with three different rule sets
- Evidence: `SpawnThreatForFaction` checks/deducts resource cost, random intensity 1–3, no occupancy check (faction/system.go:338-360). `ExecuteRaid` calls `threat.CreateThreatNode` directly: no cost, strength-scaled intensity, no occupancy check (292-298). `threat.SpawnChildThreatNode`/`SpreadCorruption`: no cost, but check walkability and existing threat (threat/system.go:183, 206).
- Problem: Whether a spawn costs resources, respects occupancy, or respects walkability depends on which code path asked. Fortify and raid can stack several threat nodes on one tile; `GetThreatNodeAt` then only ever returns the first.
- Direction: One `spawnThreat(faction, pos, intensity)` that applies the same occupancy/walkability check and the same cost rule; the three callers differ only in position and intensity.
- Effort: S. Risk: low (tests exist for scoring, not for spawning).

##### [Medium] Intent executes every tick, not once per decision
- Evidence: `UpdateFactions` re-evaluates every `DefaultIntentTickDuration` (10) ticks but calls `ExecuteFactionIntent` on every tick (faction/system.go:77-90).
- Problem: A faction in `IntentRaid` spawns a high-intensity threat (or returns a `PendingRaid` against the same garrisoned node) on each of 10 consecutive turns. Probably intended for Expand/Fortify, questionable for Raid. See Mechanics notes.
- Direction: Decide per intent whether it is "per tick" or "once"; encode that in `ExecuteFactionIntent`.
- Effort: S. Risk: medium (balance change).

##### [Low] `FactionArchetype` / `FactionBonuses` are pass-through copies of template structs
- Evidence: faction/archetype.go:9-46 copies `templates` fields one-for-one into local structs; `GetFactionBonuses` is called four times per evaluation, each doing two map lookups.
- Direction: Return the template struct directly; call once per `EvaluateFactionIntent`. Remove `FactionArchetype` (one field).
- Effort: S. Risk: low.

##### [Low] Hardcoded tuning in otherwise data-driven code
- Evidence: 30% garrison chance (faction/system.go:243), intensity `1 + RandomInt(3)` (358), disposition −50 and growth 1.0 (30-32, both unused), `squadLevel := 1 + strength/20` (garrison/system.go:146, unused downstream).
- Direction: Move the two live numbers into `overworldconfig.json`; delete the dead ones.

#### Mechanics notes
- **Faction strength only ever goes up.** The only write is `Strength += gain` in Fortify (faction/system.go:221). Nothing reduces it when the player destroys that faction's threats or wins a garrison defence. Consequently `ScoreRetreat`'s critical-weak branch can only trigger from a low starting roll, factions cannot be weakened or eliminated, and no code ever disposes a faction entity.
- **Faction resources are inert.** Factions get a stockpile (faction/system.go:49-55) and `SpawnThreatForFaction` checks cost, but every threat node costs 0/0/0 in `nodeDefinitions.json` and there is no faction income. The mechanic exists in code only.
- **Territory has almost no gameplay effect.** `OwnedTiles` decides where a faction's threats spawn and which player nodes are "adjacent" for raids. Destroying a faction's threats does not shrink territory; `AbandonTerritory` drops the most recently added tile regardless of position.
- **Only one faction raid per tick is honoured** (faction/system.go:88-90), but each raiding faction still logs `EventGarrisonAttacked` before its result is dropped.
- **Containment vs suppression.** `IsContained` + `GetContainmentSlowdown` ("player presence slows growth") and influence suppression do the same job. Only suppression is live.

#### Probable bugs
- **`SpawnThreatForFaction` ignores the error from `core.SpendResources`** (faction/system.go:354). Harmless today because `CanAfford` is checked first (low).

#### Leave alone
- The utility-scoring shape (`ScoreExpansion` etc. in scoring.go) and its config. Simple, tested, readable. The "TODO: consider using an interface for intent" (system.go:137) should not be done; a five-case switch is the right size.

---

### campaign/overworld/garrison (323 lines, tests: no)

**Role:** Assign/remove player squads on nodes, create NPC garrisons, transfer node ownership. **Verdict:** Needs attention. The garrison relation is stored on both ends and goes stale, with a confirmed lockout bug.

#### Design findings
##### [High] Garrison membership is stored in two places and neither is cleaned up by combat
- Evidence: `core.GarrisonData.SquadIDs` on the node and `squadcore.SquadData.GarrisonedAtNodeID` on the squad (garrison/system.go:52-67, 105-109). NPC garrisons set only the node side (172-175). `TransferNodeOwnership` removes the node side only (229-231). Combat teardown disposes enemy squads on every exit (tactical/combat/combatservices/combat_service.go:238) and nothing updates `GarrisonData` afterwards.
- Problem: Bidirectional cached relation, contrary to the project's "query-based, don't cache relationships" rule, and it drifts in both directions. See the two bugs below.
- Direction: Keep one side. `SquadData.GarrisonedAtNodeID` is enough: "garrison at node" becomes a query over squads (set it for NPC garrison squads too), and `GarrisonComponent` can be deleted. If the list is kept instead, add one `ClearGarrison(nodeID)` that fixes both sides and call it from ownership transfer and from the encounter resolvers.
- Effort: M. Risk: medium (touches `mind/encounter`, `gui/guioverworld`, squad save chunk; no tests).

##### [Medium] `generateGarrisonUnits` ignores both of its parameters
- Evidence: garrison/system.go:186-210 takes `factionType` and `level`, uses neither, and picks 3–4 units uniformly from the entire `unitdefs.Units` pool. The doc comment says "filtered by faction type preferences"; `CreateNPCGarrison`'s says "using power-budget generation".
- Problem: NPC garrison composition is unrelated to the faction or its strength, while `mind/spawning` already builds faction-appropriate, power-budgeted squads from `EncounterDefinition.SquadPreferences`. Two squad generators for the overworld, one of them a stub.
- Direction: Have `CreateNPCGarrison` call the `mind/spawning` generator (or have faction AI request a garrison through the same path encounters use), and delete `generateGarrisonUnits`. This also removes `garrison`'s direct dependency on `unitdefs`.
- Effort: M. Risk: medium.

##### [Low] `TransferNodeOwnership` is not a garrison concern
- Evidence: garrison/system.go:212-238 mutates `OverworldNodeData.OwnerID`; its only caller is mind/encounter/resolvers.go:212.
- Direction: Belongs with node code (moot if the packages merge).

#### Probable bugs
- **NPC-garrisoned threat node becomes un-attackable after one fight that does not destroy it** (high). Trace: `FortifyTerritory` → `CreateNPCGarrison` stores the squad ID in `GarrisonData` (garrison/system.go:172-175). Player attacks; `SpawnCombatEntities` uses `garrisonData.SquadIDs` as the enemy (mind/encounter/encounter_setup.go:55-57). On any exit `TeardownCombat` disposes those squads. If the node survives (weakened, defeat, or flee), `GarrisonData.SquadIDs` still lists the disposed ID, `IsNodeGarrisoned` stays true so Fortify never re-garrisons (faction/system.go:243), and the next attack fails in `AddSquadToFaction` with "squad N not found" (tactical/combat/combatstate/combatfactionmanager.go:62-65).
- **Player squads can be stranded as "garrisoned" after a node is captured** (medium). `GarrisonDefenseResolver` transfers ownership on any non-victory including flee (mind/encounter/resolvers.go:199-212); `TransferNodeOwnership` removes `GarrisonComponent` but does not clear `GarrisonedAtNodeID` on surviving squads. They are then excluded by `GetAvailableSquadsForGarrison` (garrison/queries.go:39) and `RemoveSquadFromNode` fails with "node has no garrison" (garrison/system.go:86-88). Not verified end to end whether defeated defender squads are disposed elsewhere; the flee case leaves survivors for certain.
- **NPC garrison squads leak when their node is destroyed without combat teardown covering them** (low). `node.DestroyNode` disposes only the node entity.

#### Dead code
- `seen` map in `FindPlayerNodesNearFaction` (garrison/queries.go:55, 70-72): each node is visited once, so the check can never hit.

#### Leave alone
- `AssignSquadToNode` / `RemoveSquadFromNode` validation and error returns.

---

### campaign/overworld/tick (113 lines, tests: no)

**Role:** Orders the per-turn subsystem updates and ends the overworld turn. **Verdict:** Healthy but thin; carries stale comments and a duplicated counter.

#### Design findings
##### [Medium] Two counters for the same thing
- Evidence: `EndOverworldTurn` increments `TickStateData.CurrentTick` (via `AdvanceTick`, tick/tickmanager.go:60) and `OverworldTurnState.CurrentTurn` (tick/endturn.go:22-24) together. `AdvanceTick` has no other caller. The GUI prints both (gui/guioverworld/overworldmode.go:270-282).
- Problem: Two sources of truth for "what turn is it" that are always equal, owned by different packages (`core` and `tactical/commander`).
- Direction: Keep one. The tick is the simulation clock; `commander`'s turn state can drop `CurrentTurn` or read the tick.
- Effort: S. Risk: low.

##### [Low] Stale comments and unused parameters
- Evidence: comment references a non-existent `InfluenceCache`/`influence_cache.go` (tick/tickmanager.go:77-78); file header says `manager.go` and lists "travel, victory" as orchestrated subsystems (1-4); doc says result contains "travel completion status" (41); `AdvanceTick`'s `playerData` parameter is unused; error returns from threat/faction updates are always nil.
- Direction: Clean up. `AdvanceTick` should either call the victory check or the `IsGameOver` guard at 55-57 should go.
- Effort: S. Risk: low.

#### Leave alone
- The explicit update order (influence → threat → faction) in one short function. Keep it that visible.

---

### campaign/overworld/victory (191 lines, tests: no)

**Role:** Overworld win/lose evaluation. **Verdict:** Needs significant rework or deletion: the whole package is unreachable.

#### Design findings
##### [High] The package is dead; the overworld has no end condition
- Evidence: No file in the repo imports `game_main/campaign/overworld/victory`. `CheckVictoryCondition`, `CreateVictoryStateEntity`, `CheckPlayerDefeat`, `GetTotalThreatInfluence`, `HasPlayerEliminatedAllThreats`, `HasPlayerDefeatedFactionType`, `GetVictoryState` have no callers. `TickStateData.IsGameOver` is read (tick/tickmanager.go:55, gui/guioverworld/overworldmode.go:267, 335) but never assigned true. `OverworldConfigTemplate.VictoryConditions` is read only here.
- Problem: 191 lines plus the `VictoryState*`/`VictoryCondition` types, three `IsGameOver` branches, `EventVictory`/`EventDefeat`, and a JSON config block describe a feature that does not run. Anyone reading `tick` or the GUI would assume the game can end.
- Direction: This is a product decision. Either wire it (call from `tick.AdvanceTick`, set `IsGameOver`, have the GUI react), fixing the issues below first, or delete the package and everything that only it uses. Do not leave it half-present.
- Effort: S to delete, M to wire. Risk: low.

##### [Medium] If kept: four copies of the same end-of-game block, and a rule mixed with file export
- Evidence: log event + `core.FinalizeRecording` + `core.ClearRecording` repeated at victory/system.go:26-33, 46-53, 69-76, 88-95.
- Direction: `CheckVictoryCondition` should return `(condition, reason)` and do nothing else; the caller logs and exports once.

#### Probable bugs (latent, since the code does not run)
- **Cannot target Necromancers for faction victory** (high): `TargetFactionType != core.FactionType(0)` is the "is configured" test (victory/system.go:82) and `FactionNecromancers` is 0.
- **Instant victory when no threats exist** (medium): `HasPlayerEliminatedAllThreats` is true on any tick with zero threat nodes (victory/system.go:62, 123-127), including before the first spawn.
- **Faction-defeat victory is unreachable**: no code ever removes a faction entity (see `faction` mechanics notes).
- `VictoryAchieved = true` is set on defeat (victory/system.go:22).

---

### campaign/overworld/overworldlog (393 lines, tests: yes)

**Role:** Session recorder and JSON exporter for overworld events, with summary statistics. **Verdict:** Needs attention. Self-contained and dependency-free, but its summaries depend on event data nobody supplies.

#### Design findings
##### [Medium] Summaries are coupled to event display strings and to data keys that are not populated
- Evidence: switches on `"Threat Spawned"`, `"Faction Expanded"`, `"Combat Resolved"` etc. (overworldlog/overworld_summary.go:55-113, 135-161, 172-190), which must match `core.EventType.String()` (core/types.go:152-193). Reads `Data["threat_type"]`, `["intensity"]`, `["new_intensity"]`, `["tiles_gained"]`; every producer in the slice passes `nil`. Only `"victory"` / `"intensity_reduced"` / `"new_intensity"` from mind/encounter/resolvers.go are ever present.
- Problem: `ThreatTypeStats`, `MaxIntensity`, and `TerritoryGains` are always empty/zero in a real export. Renaming an event's display string silently breaks the summary. The `.(float64)` branches handle "JSON unmarshaling" that never happens (records are only ever written).
- Direction: If the export is worth keeping, pass a stable event key (the enum's identifier, not its display name) and populate the data at the `LogEvent` call sites. Otherwise cut the summaries to the counts that work.
- Effort: S–M. Risk: low (tests exist).

#### Probable bugs
- **`FactionStats` is keyed by every event's entity ID** (high): overworldlog/overworld_summary.go:127-133 creates a `FactionStats` entry for any non-zero `EntityID`, so threat nodes, player nodes, and garrisoned nodes all appear as "factions" with zero stats.
- **`FinalTick` is the tick of the last event, not the session's last tick** (low): overworld_recorder.go:103-108.

#### Leave alone
- `OverworldRecorder` lifecycle (`Start`/`RecordEvent`/`Finalize`/`Clear`) and `ExportOverworldJSON`. Keeping it free of `core` imports (strings in, record out) is the right boundary.

---

### campaign/raid (1,730 lines, tests: yes, disposal only)

**Role:** The roguelike run: generates a multi-floor garrison as ECS entities, tracks room/floor/alert state, starts room combats and resolves them. **Verdict:** Needs significant rework in one area: the raid's end-of-run state machine is spread across four places and does not work for defeat or for normal victory. The rest (generation, alert, deployment, rewards, disposal) is sound.

#### Design findings

##### [High] Raid status transitions are split across resolver, callback guard, runner, and GUI
- Evidence: `RaidEncounterResolver.Resolve` sets `Status = RaidDefeat` (raid/resolvers.go:22-26). The post-combat callback only proceeds when `Status == RaidActive` (raid/raidrunner.go:55-60). `PostEncounterProcessing` has a `Status != RaidActive → finishRaid` branch (280-283) and calls `CheckRaidEndConditions` (292). `AdvanceFloor` has a victory branch (306-312). `gui/guiraid/raidmode.go:341-361` decides whether to call `AdvanceFloor`. `finishRaid` takes a `status` it never uses (352) and clears the callback registered once in the constructor (354).
- Problem: No single function owns "the raid ended". Each piece assumes another will act, and the combination produces the three bugs listed below. `FloorStateData.IsComplete` vs `IsFloorComplete()` (resolvers.go:54-58, 93; raidrunner.go:207; floorgraph.go:93) is the same split for floors.
- Direction: `RaidRunner` owns all transitions through one `endRaid(status)`; the resolver reports the outcome and does not write `Status`; the callback guard tests for "not retreated" rather than "active"; the callback is registered in `StartRaid`/on restore (or never cleared and guarded by `IsActive`); final-floor stairs go through the same advance path as other floors so victory is reached in the runner, not inferred by the GUI.
- Effort: M. Risk: medium (tests cover disposal only).

##### [Medium] Room/floor/squad relations and progress are stored redundantly
- Evidence: A garrison squad's room is recorded in `RoomData.GarrisonSquadIDs`, `FloorStateData.GarrisonSquadIDs`, `FloorStateData.ReserveSquadIDs`, and `GarrisonSquadData.{FloorNumber, RoomNodeID, IsReserve}` (raid/components.go:61-97; written together at garrison.go:95-132 and alert.go:53-56). `FloorStateData.GarrisonSquadIDs` has no reader outside the save chunk; `RoomsCleared`/`RoomsTotal` are written (floorgraph.go:58, garrison.go:127) and never read; `GarrisonSquadData.IsDestroyed` is written (resolvers.go:47) and never read; `ArchetypeName` is only saved. The doc on `DisposeRaidEntities` (garrison.go:48-52) already notes the ID lists go stale after combat teardown and works around it by querying the tag.
- Problem: Multiple sources of truth that the code itself no longer trusts, all of which are also serialized and remapped in `setup/savesystem/chunks/raid_chunk.go`.
- Direction: `GarrisonSquadData` on the squad is the single record; "squads in room N" is a tag query. Delete `FloorStateData.GarrisonSquadIDs`, `RoomsCleared`, `RoomsTotal`, `GarrisonSquadData.IsDestroyed`; keep one of `IsComplete` / `IsFloorComplete`. `FloorStateData` may reduce to the reserve list or disappear. The save chunk shrinks with it.
- Effort: M. Risk: medium (save format changes; allowed per the brief).

##### [Medium] Config loading bypasses the templates pipeline and duplicates its defaults in Go
- Evidence: `LoadRaidConfig`/`LoadArchetypeData` do their own `os.ReadFile` + `json.Unmarshal` (raid/config.go:69-82, archetypes.go:31-47), are called from game_main/setup.go:33-38 outside `LoadGameData`, and on failure the game prints a warning and continues. Eleven accessors return hardcoded fallbacks that mirror `raidconfig.json` value for value (config.go:98-185, rewards.go:73-86). Other code nil-checks `RaidConfig` and silently does nothing (raidrunner.go:192, recovery.go:13, 35, rewards.go:15, alert.go:80). `archetypes.go:45` uses bare `println`. There is no validation that archetype names in `archetypeAssignment` exist, that `monsterType` values resolve, or that the lists are non-empty.
- Problem: Two sources of truth for every tuning value, a boot path that contradicts the project rule that `LoadGameData` is the single fatal point, and behaviour that differs by which nil check you hit (no recovery, zero rewards, alert frozen). A bad archetype name yields `GetArchetype → nil → InstantiateGarrisonSquad → 0` (garrison.go:96-97, 140-142), leaving a combat room with no squads that `TriggerRaidEncounter` refuses to start (raidrunner.go:153-155) and that can therefore never be cleared.
- Direction: Move both files to `templates` with `Loader[T]` and validators (names resolve, monster types resolve, alert levels ordered, pools non-empty); fail at boot. Delete every fallback and every `RaidConfig == nil` check; accessors that only added a fallback go away.
- Effort: M. Risk: low.

##### [Low] `RaidRunner` keeps state that the ECS already has
- Evidence: `raidEntityID` (raidrunner.go:31) is "does a `RaidStateTag` entity exist" and needs `RestoreFromSave` plus a `WorldTags["raidstate"]` lookup in game_main/setup.go:220-224 to stay in sync after load. `preCombatAliveCounts` and `currentRoomNodeID` are per-encounter scratch.
- Direction: `IsActive()` returns `GetRaidState(manager) != nil`; delete `raidEntityID` and `RestoreFromSave`. Put the pre-combat counts and room ID on the `RaidCombatStarter`/resolver that already travels with the encounter.
- Effort: S. Risk: low.

##### [Low] Small inconsistencies
- `RaidEncounterResolver` holds a `*RaidStateData` captured at `Prepare` and also re-queries `GetRaidState` in the same method (resolvers.go:17, 23, 32).
- `commandersForSquads` (rewards.go:52-70) is a line-for-line duplicate of mind/encounter/resolvers.go:169-187.
- Warnings use `fmt.Printf("WARNING: ...")` / `println` (floorgraph.go:48, garrison.go:149, 168, archetypes.go:45) where the overworld has a logger.
- Stale docs: `processStairsRoom` "advances floor" (it does not, raidrunner.go:201); `IsFloorComplete` "cleared or accessible" (checks cleared only, floorgraph.go:91); `RaidRunner` doc "post-combat combatlifecycle" (raidrunner.go:26); `Retreat` "ends the raid" (320).
- `generateFloor` iterates the `assignments` map (garrison.go:95), so squad creation order and entity IDs vary per run even with a fixed seed. Iterate `dag.Nodes` instead.

#### Mechanics notes
- **Floor depth does not change which enemies you meet.** Every combat room type in `garrisongen` has at least one archetype listing it in `preferredRooms` (raidarchetypes.json), so `pickArchetypeForRoom` always returns from the preferred list (assignment.go:44-46). The fallback pools (`criticalPathArchetypes`, `branchArchetypes`, `eliteArchetypes`, `eliteFloorThreshold`) and the `onCriticalPath`/`floorNumber` parameters are unreachable with shipped data. Floors scale only reward amounts, reserve count, and DAG size.
- **Any non-victory exit, including flee, is a raid defeat** (resolvers.go:22). There is no "lost this room, try another" state, which is consistent with teardown disposing the room's garrison squads either way.
- **Between-floor recovery uses `DeployedHPPercent`** for everyone (recovery.go:40); there is no separate setting.
- **The raid's room graph and the `garrison_raid` map are generated independently.** `raid.generateFloor` and `garrisongen.GarrisonRaidGenerator.Generate` each call `BuildGarrisonDAG` (raid/garrison.go:74, world/garrisongen/generator.go:56), producing two unrelated random DAGs, and every room fight happens at one fixed config position (config.go:122-127, raidencounter.go:15-21).

#### Probable bugs
- **Raid defeat is never processed** (high). Order is resolver → callback (mind/encounter/encounter_service.go:157-162). The resolver sets `RaidDefeat` (resolvers.go:25); the callback guard then sees `Status != RaidActive` and skips `ResolveEncounter` (raidrunner.go:57). Result: no `finishRaid`, no entity disposal, no summary (`LastEncounterResult` was nil'd at 175), `IsActive()` stays true, and `RaidMode.Enter` shows the floor map of a raid whose status is `RaidDefeat` (gui/guiraid/raidmode.go:112-130). Every later combat in that raid also skips post-processing (no recovery, no alert, no end check). The `Status != RaidActive` branch in `PostEncounterProcessing` (280-283) is unreachable.
- **Clearing the final floor's stairs does not end the raid** (high). `processStairsRoom` sets `IsComplete` (raidrunner.go:202-209); `checkFloorAdvancement` only calls `AdvanceFloor` when `CurrentFloor < TotalFloors` and otherwise just sets status text (gui/guiraid/raidmode.go:351-360). `AdvanceFloor`'s victory branch is therefore unreachable from the GUI, and `CheckRaidEndConditions` runs only after a combat. The raid stays `RaidActive` with all entities alive unless the player then wins another leftover room on that floor.
- **Second raid in a session has no post-combat processing** (high, conditional on a raid ever finishing). The callback is registered only in `NewRaidRunner` (raidrunner.go:55), which runs once per session (game_main/setup.go:137, 215). `finishRaid` clears it (354). `StartRaid` does not re-register. A subsequent `autoStartRaid` runs with no callback.
- **Rest rooms do nothing if config failed to load** (low): raidrunner.go:192.

#### Dead code
- `RaidNone` and `RaidStatus.String()` (components.go:26, 33-48): no references outside the definition.
- `RaidEncounterResult.IsVictory` (raidrunner.go:22, 268): never read.
- `finishRaid`'s `status` parameter.
- `FloorStateData.RoomsCleared`, `RoomsTotal`, `GarrisonSquadIDs`; `GarrisonSquadData.IsDestroyed` (write-only apart from save/load).
- `SelectRoom`'s combat-room path returns nil without doing anything (raidrunner.go:137); the GUI only calls it for rest/stairs rooms.
- Five of the six `manager.WorldTags[...]` registrations in init.go:33-38 (only `"raidstate"` is read, and that read goes away with the `raidEntityID` change).

#### Performance
- `GetRoomData` / `GetFloorState` / `GetAlertData` each run `World.Query` and scan all floors' entities (queries.go:16-46). Fine at 3 floors × ~10 rooms and turn-based call rates; only worth a view if the floor-map panel calls them per frame (`gui/guiraid` does call `GetAccessibleRooms` every frame; see that section).

#### Leave alone
- `DisposeRaidEntities` (garrison.go:53-69): tag-based, well reasoned, tested.
- `buildFloorGraph` / `MarkRoomCleared` / accessibility propagation (floorgraph.go). `IsAccessible` is cached derived state, but it is updated in exactly one place and is cheap to reason about.
- `IncrementAlert` / `ActivateReserves`: data-driven and short.
- `AutoDeploy` / `SetDeployment`.
- `RaidCombatStarter` + `RaidEncounterResolver` plugging into `combatlifecycle.ExecuteCombatStart`/`ExecuteResolution`. This is the right seam; only the status write inside the resolver needs to move.
- `RaidRunner` being a plain service rather than an ECS system.
- `calculateRoomReward` returning a reward for the pipeline to grant.

---

## 17. GUI infrastructure

All claims below were checked by reading every non-test file in the slice and grepping the whole repo (including `tools/` and tests) for callers. Nothing was executed; items that depend on runtime behaviour are marked as such. ebitenui behaviour was checked against the vendored source (`ebitenui@v0.5.8` in the Go module cache).

### Layer overview

**Dependency shape is clean.** `specs` (no imports) ← `widgetresources` / `widgets` ← `builders` ← `framework` ← all `gui/gui*` mode packages. No cycles, and the direction is right. No package needs to be split.

**What the modes actually use** (12 modes, 48 registered panels):

| Used by every mode | Used by 1 to 4 modes | Bypassed or dead |
|---|---|---|
| `BaseMode`, `ModeBuilder`, `UIModeManager`, `ActionMap` | `SubMenuController` (4), `CommandHistory` (2), `BuildSelectionPanel` (2 panels), `BuildGridEditor` (2 files), `TextDisplayComponent` (1), `PanelBuilders.BuildPanel` (1) | Declarative half of the panel registry (0 of 48 panels) |
| `RegisterPanel` + `OnCreate` + `Custom` map | `BuildTypedPanel` (8 call sites, 6 of them broken) | `StandardPanels` (11 of 13 entries unused) |
| `CreateButtonWithConfig` (47 ext. uses), `CreateSmallLabel` (35), `CreateListWithConfig`, `CreateStaticPanel` / `CreatePanelWithConfig`, `Anchor*` helpers, padding constants | | `CachedBackground` pools (results discarded), `CachedListWrapper` (never in the widget tree), 26 `specs` constants |

**The main theme of this slice is caching that does not do what it says.** There are four caches. One is a no-op (`CachedBackground`), one is never attached to the widget tree (`CachedListWrapper`), one is attached and very likely draws in the wrong place (`CachedTextAreaWrapper`), and one is a process-wide cache of game state that only combat mode invalidates (`SquadInfoCache`). Each one adds "you MUST call MarkDirty / MarkSquadDirty" obligations to callers. Removing all four deletes roughly 650 lines and a class of stale-display bugs.

**The second theme is abstraction that the modes outgrew.** The panel registry's declarative path, `StandardPanels`, `BuildTypedPanel` and the functional-option `BuildPanel` were designed for simple "container + text" panels. Every real panel is built by hand inside an `OnCreate` closure, so those layers are now either dead or actively hiding failures (six panels silently do not exist).

**Layout recommendation.** Keep `framework`, `builders`, `specs`, `widgetresources`. After the cache wrappers go, `gui/widgets` is about 110 lines (`SelectionPanelCore`, `TextDisplayComponent`); fold `SelectionPanelCore` into `builders` next to `BuildSelectionPanel` and delete the package. This is optional and low priority.

**Suggested order of work for this slice:** (1) fix the six missing panels and the input-gating bug, (2) delete the three widget/background caches, (3) remove or scope `SquadInfoCache`, (4) strip the dead declarative registry path, (5) mode-name constants and transition helper, (6) the rest.

---

### gui/framework (2388 lines, tests: no)

**Role:** Mode lifecycle (`UIMode`, `BaseMode`, `UIModeManager`, `GameModeCoordinator`), semantic input (`ActionMap`), panel registry, shared GUI state (`TacticalState`, `OverworldState`) and the `GUIQueries` read facade.
**Verdict:** Needs attention. The mode/input core is small and sound, but the panel registry is half dead, `GUIQueries` carries a drifting cache, and input dispatch has no concept of modal dialogs or pointer-over-UI.

#### Design findings

##### [High] Mode input is dispatched with no awareness of dialogs, text focus, or the pointer being over a widget
- Evidence: `gui/framework/modemanager.go:104-114` calls `currentMode.HandleInput` unconditionally, before `GetEbitenUI().Update()`. No file under `gui/`, `input/` or `game_main/` references `input.UIHovered`, `UI.HasFocus`, `UI.GetFocusedWidget` or `UI.IsWindowOpen` (grep returns nothing). Dialogs are opened with `GetEbitenUI().AddWindow(dialog)` at `gui/guisquads/squadeditor_roster.go:72,192,235,297` and `gui/guioverworld/overworld_input_handler.go:348`.
- Problem: Each mode must defend itself against input meant for widgets, and none do. This is the root cause of two bugs listed under "Probable bugs" (hotkeys firing while typing a squad name; clicks on panels also hitting the map). Fixing it per mode would mean the same guard in 12 places.
- Direction: Gate centrally in `UIModeManager.Update`. ebitenui v0.5.8 already exposes what is needed: `input.UIHovered` (pointer over a widget), `UI.GetFocusedWidget()` (a `*widget.TextInput` has focus) and `UI.IsWindowOpen`. When a text input is focused or a modal window is open, clear key-derived actions before calling `HandleInput`; when the pointer is over a widget, clear mouse-derived actions. A `BaseMode.ShowDialog(window)` helper that tracks open modals would replace the five raw `AddWindow` calls.
- Effort: S/M. Risk: medium (touches every mode's input; no tests).

##### [High] The panel registry is an indirection with no remaining benefit, and half of it is dead
- Evidence: 48 `framework.RegisterPanel` calls across 12 files; all 48 supply `OnCreate` (48 `OnCreate:` matches). `BuildRegisteredPanel` only uses `SpecName`, `Position`, `Width`, `Height` and `Content` when `OnCreate == nil` (`gui/framework/panelregistry.go:83-111,118-123`), so that branch never runs. `Content` is set 48 times (44 `ContentCustom`, 4 `ContentText`) and read by nothing reachable. Widgets are returned through `PanelResult.Custom map[string]interface{}` (78 writes) and pulled back out with `GetPanelWidget[T](panels, type, "key")` (74 reads), for example `gui/guisquads/squadeditormode.go:154-185`. Every `OnCreate` starts with a type assertion such as `mode.(*SquadEditorMode)`, which is the only reason `BaseMode.self` / `SetSelf` exist (`gui/framework/basemode.go:35,55`). Modes also add widgets outside the registry (`gui/guisquads/squadeditormode.go:146-147`).
- Problem: The flow is "register a closure in `init()`, look it up by string type, call it with the mode as an interface, cast it back, stash widgets in a string-keyed map, read them back by string key into typed fields". A typo in a key or a type parameter yields a silent zero value, not a compile error. Each panel is built by exactly one mode, so the global registry does not enable reuse. There are also two different types called `PanelType` (`framework.PanelType` string, `builders.PanelType` int) and both a global `panelRegistry` map and a `PanelRegistry` struct, which makes the code hard to read.
- Direction: Two steps, the first worth doing regardless.
  1. Delete the dead declarative half: `PanelContentType` and its three constants, `PanelDescriptor.SpecName/Content/Position/Width/Height`, and the non-`OnCreate` branch. The descriptor becomes a single build function.
  2. Optional, mode by mode: replace `Custom` + string keys with a typed struct per panel returned from a plain method on the mode (`func (m *SquadEditorMode) buildGridPanel() gridPanelWidgets`). The `*_panels_registry.go` files stay as the place panels are defined; only the global map, `SetSelf`, `GetPanelWidget` and the string keys go away.
- Effort: step 1 S, step 2 L (mechanical, 48 panels). Risk: step 1 low, step 2 medium (no tests).

##### [High] `SquadInfoCache` is shared by every mode but only combat invalidates it
- Evidence: One `GUIQueries` is created for the whole UI (`NewGUIQueries`, 1 external caller) and holds `squadInfoCache` (`gui/framework/guiqueries.go:22,35`). Invalidation calls exist only in combat-related GUI code: `gui/guicombat/combatmode.go:95`, `gui/guicombat/combat_mode_init.go:243-258`, `gui/guicombat/combatbase/handler.go:146,156,215`, `gui/guispells/spell_handler.go:320-323`, `gui/guiartifacts/artifact_handler.go:170`. A non-combat reader exists: `gui/guisquads/squaddeploymentmode.go:146` filters the deployable list through cached `IsDestroyed`. Squad editor commands never mark anything dirty.
- Problem: This is derived state with a manual invalidation contract spread across ten call sites in four packages, which is exactly the "cached relationships" pattern the project's ECS rules forbid. What it saves is small: `buildSquadInfo` (`gui/framework/squadinfo_cache.go:42-117`) is a handful of component lookups for at most nine units. `GetSquadRenderInfo` allocates a fresh struct per squad per frame on top of the cache anyway (`gui/framework/guiqueries_rendering.go:19-32`).
- Direction: Delete `SquadInfoCache` and the four `Mark*/Invalidate*/Clear*` methods on `GUIQueries`; build `SquadInfo` on demand. If profiling later shows a need, memoise per frame inside combat mode, owned and cleared by `CombatMode`, not shared through `UIContext`.
- Effort: S/M. Risk: low to medium (no tests; behaviour becomes "always fresh").

##### [Medium] `SquadInfo` defines "alive" differently from the rest of the code
- Evidence: `gui/framework/squadinfo_cache.go:68` counts a unit as alive when `attrs.CanAct` is true. `Attributes.CanAct` is never assigned after construction (grep for `CanAct =` in non-test code finds only the default at `core/common/commoncomponents.go:53` and save/load in `setup/savesystem/chunks/shared_types.go`). The canonical rule is `CurrentHealth > 0` (`tactical/squads/squadcore/squadqueries.go:165-180`), and the same package uses that rule in `gui/framework/guiqueries_units.go:50` and `guiqueries_rendering.go:59`.
- Problem: `AliveUnits` is really "units that have an Attributes component" and `IsDestroyed` is really "has zero units". It works today only because dead units are disposed after each attack (`combatactionsystem.go:203-206`). Three definitions of one concept in one package is a source-of-truth problem.
- Direction: Compute `IsDestroyed` with `squadcore.IsSquadDestroyed` and `AliveUnits` with `CurrentHealth > 0`. `Attributes.CanAct` itself is a dead field in `core/common`.
- Effort: S. Risk: low.

##### [Medium] `GUIQueries` is neither a boundary nor a facade
- Evidence: Modes reach through it to the raw manager 57 times (`Queries.ECSManager`) and call domain packages directly about 170 times (for example `squadcore.GetSquadName` is called directly at 15 GUI sites while the wrapper `GUIQueries.GetSquadName` has 1 caller, `gui/guiraid/deploy_panel.go:89`). Nine methods are one-line pass-throughs with 1 to 5 callers each (`guiqueries_overworld.go:12-45`, `guiqueries_units.go:57-77`). It constructs its own `CombatQueryCache` and `CombatFactionManager` (`guiqueries.go:27-32`) although `CombatService` already owns both (`tactical/combat/combatservices/combat_service.go:62,73`); `gui/guicombat/combatbase/handler.go` uses `deps.Queries.CombatCache` on line 183 and `deps.CombatService.CombatCache` on lines 121 and 173.
- Problem: The stated rule ("GUIQueries-only state access") is not what the code does, so the type gives a false sense of a boundary while adding a second copy of combat helpers.
- Direction: Narrow it to what it is good at, which is building view models: `SquadInfo`, `FactionInfo`, `UnitGridInfo`, and the two `combatrender` provider implementations. Delete the one-line component getters and let modes call the domain query functions they already call. Take the combat cache and faction manager from `CombatService` instead of constructing duplicates.
- Effort: M. Risk: low.

##### [Medium] Mode identity and transitions are stringly typed and inconsistent
- Evidence: Each of the 12 modes sets its name twice, once in the constructor (`SetModeName("squad_editor")`, needed because `RegisterMode` reads the name before `Initialize`, `modemanager.go:42`) and again in `ModeConfig.ModeName`. About 20 further sites look modes up by literal (`GetMode("combat")`, `GetMode("unit_purchase")`, `fromMode.GetModeName() == "combat_animation"`), and `GameModeCoordinator.EnterCombatMode` hardcodes `"combat"` (`coordinator.go:190`). Transitions are written as `if m, ok := GetMode(name); ok { RequestTransition(m, reason) }`, which does nothing when the name is wrong. Five sites call the immediate `SetMode` from inside callbacks (`gui/guioverworld/overworld_panels_registry.go:90`, `overworld_input_handler.go:52,95`, `gui/guiraid/floormap_panel.go:53`, `raidmode.go:332`) while 22 use the deferred `RequestTransition`. All 12 constructors also assign `mode.ModeManager = modeManager` and call `SetSelf` by hand.
- Problem: Two sources of truth per mode name, silent failure on a typo, and two transition semantics (mid-frame versus end-of-frame) chosen ad hoc.
- Direction: One file of mode-name constants in `framework`; a `UIModeManager.RequestTransitionTo(name, reason)` that logs when the mode is missing; make `ModeBuilder` read the name from the mode instead of taking it again; use the deferred transition everywhere a mode switches itself.
- Effort: S/M. Risk: low.

##### [Medium] Context switching skips `Exit`, and the camera reads another manager's input state
- Evidence: `GameModeCoordinator.switchToContext` (`coordinator.go:93-114`) swaps `activeManager` without calling `Exit` on the outgoing manager's current mode. When the context is re-entered with `SetMode`, that stale mode receives `Exit(toMode)` followed by `Enter(fromMode)` with itself as both (`modemanager.go:72-88`). `game_main/main.go:62-63` passes the tactical manager's `InputState` to the camera controller every frame, including while the overworld manager is active and the tactical one is not being updated. `UIModeManager.Update` resolves actions with the old mode's map and then applies the pending transition, so the new mode's first `HandleInput` sees actions resolved by the previous mode's bindings (`modemanager.go:93-106`, `143-196`).
- Problem: `Enter`/`Exit` are not a reliable pair, so cleanup placed in `Exit` runs late or not at all (see "Probable bugs"). The camera path keeps three otherwise unnecessary things alive: `UIModeManager.GetInputState`, `InputState.PlayerInputStates` ("bridge to existing system", never read) and `PlayerInputStates.HasKeyInput` (set in `input/cameracontroller.go`, cleared in `main.go:65-66`, read by nothing else).
- Direction: Have `switchToContext` call `Exit` on the outgoing mode. Move camera handling into `ExplorationMode.HandleInput` (it is the only mode whose `ActionMap` produces camera actions) and delete `GetInputState`, the `PlayerInputStates` bridge, and the unused `coordinator` parameter of `NewCameraController`. Resolve actions after the pending transition.
- Effort: S/M. Risk: medium (no tests).

##### [Medium] `TacticalState` / `OverworldState` mix UI selection with flow state and model one mode as five booleans
- Evidence: `gui/framework/contextstate.go:24-37` has `InAttackMode`, `InMoveMode`, `InSpellMode`, `InArtifactMode`, `InInspectMode`, which are mutually exclusive in practice and kept consistent by hand in `gui/guicombat/combatbase`, `combatinput`, `gui/guispells` and `gui/guiartifacts`. `TriggeredEncounterID` is written through `SetTriggeredEncounterID` and read nowhere (0 external reads). `PostCombatReturnMode` is game-flow routing stored in a struct whose comment says "UI STATE ONLY". `OverworldState.SelectedCommanderID` doubles as the game-wide "active commander" for roster ownership, including in roguelike mode where there is no overworld (`uimode.go:62-74`, `game_main/setup.go` sets it directly). `OverworldState.CameraX/CameraY` are read 14 times in `gui/guioverworld/overworld_renderer.go` and never written.
- Problem: Five flags allow 32 states where about six are valid; each handler must remember to clear the others. The state structs no longer match their own documentation.
- Direction: Replace the five flags with one `CombatInputMode` enum. Delete `TriggeredEncounterID` and its interface method on `encounter.ModeCoordinator`. Move "active commander" to an explicitly named field on the coordinator or `UIContext`. Delete `CameraX/CameraY` or implement overworld scrolling (a feature, so delete for now).
- Effort: M. Risk: medium (enum change touches four packages; no tests).

##### [Low] Small API leftovers
- `UIMode.HandleInput` returns a "consumed" bool that `UIModeManager` ignores (`modemanager.go:106`).
- `InputState` exports `KeysPressed`, `KeysJustPressed`, `KeysJustReleased`, `MousePressed`, `MouseJustPressed`, `MouseButton`; no code outside `framework` reads them, so modes cannot bypass the action map by accident only by convention. Unexport them.
- `BaseMode.CommandHistory` exists for 2 of 12 modes and is the only reason `framework` imports `tactical/squads/squadcommands`. Acceptable; see dead code for the unused methods.
- `ModeBuilder` adds the status label to the root anchor container with no layout data (`modebuilder.go:72-75`), so each mode repositions it.
- Direction: tidy when touching these files. Effort: S. Risk: low.

#### Probable bugs

1. **Mode hotkeys fire while typing in a modal text dialog. Confidence: high (traced, not run).** `SquadEditorMode.onNewSquad` and rename open `CreateTextInputDialog` (`gui/guisquads/squadeditor_roster.go:38,272`). `UIModeManager` still calls `SquadEditorMode.HandleInput` every frame, and `DefaultSquadEditorBindings` binds plain `U`, `R`, `N`, `V`, `B`, `P`, `K` and `Escape` (`gui/framework/defaultbindings.go:61-74`). Typing a lowercase name containing `p` requests a transition to unit purchase; `n` opens a second new-squad dialog; `Escape` leaves the mode with the dialog open.
2. **Clicks on UI panels also reach map handlers. Confidence: high on mechanism, medium on visible impact.** `ActionMouseClick` is resolved for every left click. For example in spell targeting, clicking the spell panel's own list or Cast button also runs `HandleSingleTargetClick(mouseX, mouseY)` (`gui/guicombat/combatinput/handler.go:100-106`), and outside sub-modes any click runs `handleSquadClick` / `handleMovementClick` (`:148-160`). Impact depends on what tile lies under the widget.
3. **AoE shape rotation cannot be triggered. Confidence: high.** `ActionAoERotateLeft/Right` are checked at `gui/guicombat/combatinput/handler.go:91,95` but no `ActionMap` binds them (only references are the enum at `inputaction.go:37-38`).
4. **Deployment list can hide or show the wrong squads between combats. Confidence: medium.** Create an empty squad, open squad deployment (the squad is cached with `IsDestroyed = true`, `squadinfo_cache.go:112`), return to the editor and add units, reopen deployment: the cached entry is not dirty, so `FilterSquadsAlive` still drops the squad until the next `CombatMode.Enter` clears the cache. The reverse (emptied squad still listed) follows from the same cause.
5. **`OverworldMode.Exit` does not run when leaving for the tactical context. Confidence: high on mechanism, low on impact.** `Exit` exports the overworld recording and clears selection (`gui/guioverworld/overworldmode.go:177-184`); it runs only later, when `ReturnToOverworld("overworld")` re-enters the same mode.
6. **Camera controller reads a stale `InputState` while the overworld context is active. Confidence: low** that it is observable (requires a camera action to be active on the last tactical frame), high that the state is stale.

#### Dead code
- `panelregistry.go`: `ContentEmpty`, `ContentText`, `ContentCustom`, `PanelDescriptor.SpecName/Position/Width/Height`, lines 83-111 and 118-123 (unreachable because every descriptor has `OnCreate`).
- `commandhistory.go`: `CommandHistory.HandleInput` and `CommandHistory.Clear` (0 callers).
- `actionmap.go` / `inputbinding.go`: `BindHeld`, `TriggerHeld`, `ModAlt` (no bindings use them). Harmless; remove or keep as deliberate API.
- `contextstate.go`: `TacticalState.TriggeredEncounterID`; `OverworldState.CameraX/CameraY` (never written).
- `coordinator.go`: `GetPlayerData` (0 callers); `SetTriggeredEncounterID` (value never read).
- `uimode.go`: `InputState.PlayerInputStates` (never read through `InputState`).
- `guiqueries.go`: `GetFactionsForEncounter` and `SquadFilter` have no external users; `FilterSquadsAlive` / `ApplyFilterToSquads` have exactly one (`squaddeploymentmode.go:146`).

#### Performance
- Removing `SquadInfoCache` makes the two combat renderers rebuild `SquadInfo` per squad per frame. That is tens of component lookups per frame and is not a concern; `GetSquadRenderInfo` already allocates per call today.

#### Leave alone
- `UIMode`, `UIModeManager`'s register/transition core, and the two-manager `GameModeCoordinator` concept. Small and understandable.
- `ActionMap` and the single `defaultbindings.go` table. Keeping every mode's bindings in one file is the right call for future rebinding; the central `InputAction` enum is a fair price.
- `SubMenuController`, including the remove-from-tree behaviour and its comment explaining why.
- `ModeBuilder` / `ModeConfig`. Thin, and used by all 12 modes.
- The `framework` package boundary. It mixes generic mode machinery with game-specific state and queries, but splitting it would add a package without removing coupling.

---

### gui/builders (2104 lines, tests: yes, `layout_test.go` only)

**Role:** Factory functions for ebitenui widgets, panels, dialogs and layout data.
**Verdict:** Needs attention. The config-struct widget factories are heavily used and fine; the panel-building layer has four overlapping paths, one of which silently returns nil for six live panels.

#### Design findings

##### [High] `BuildTypedPanel` + `StandardPanels` fail silently, and most of both is unused
- Evidence: `BuildTypedPanel` returns `TypedPanelResult{Panel: nil}` when the spec name is unknown (`gui/builders/panels.go:396-399`). `StandardPanels` (`panelspecs.go:33-157`) has 13 keys. The 8 external call sites use `message_log` and `player_resources` (exist) and `node_list`, `placement_info`, `threat_info`, `tick_status`, `event_log`, `threat_stats` (do not exist; `StandardPanels` is never written to elsewhere). The other 11 keys have no live reader (`framework/panelregistry.go:85` is on the dead path). The file's header comment lists `UIComponentFactory`, `InfoMode` and `InventoryMode`, none of which exist.
- Problem: A "single source of truth for panel layouts" that most panels ignore, with a lookup that hides missing entries. `PanelTypeDetail` with an empty `SpecName` would also compute a negative minimum size from a zero spec (`panels.go:441-447`).
- Direction: Delete `StandardPanels`, `PanelLayoutSpec`, `BuildTypedPanel`, `TypedPanelConfig`, `TypedPanelResult` and `builders.PanelType`. Replace the eight call sites with one small helper, for example `CreateTextPanel(layout, anchor, widthFrac, heightFrac, initialText)`, which cannot fail on a string key. That also fixes the six missing panels (they need sizes and positions chosen).
- Effort: S/M. Risk: low.

##### [Medium] Four ways to build a panel container
- Evidence: `PanelBuilders.BuildPanel(opts...)` with 16 functional options (`panels.go:14-320`) has one live external caller (`gui/guiinspect/inspect_panel.go:22`); of its ten position options only `RightCenter` is used outside the package. `CreateStaticPanel` (18 ext. uses) and `CreatePanelWithConfig` (13) differ only by the `EnableCaching` flag, which has no effect (see `widgetresources`). `BuildTypedPanel` is the fourth. Sizing conventions differ: screen fractions (`Size`), pixels (`ContainerConfig.MinWidth`), and "screen size plus percent" (`SimpleStringListConfig`). `PanelBuilders` is a struct holding only `Layout`, while every other builder is a free function taking `layout`.
- Problem: A new panel author has to choose between equivalent APIs, and the least-used one is the largest (about 300 lines for one caller).
- Direction: Keep `CreatePanelWithConfig` as the one container builder. Delete `CreateStaticPanel` and `EnableCaching` (replace 18 calls mechanically). Convert the single `BuildPanel` caller and delete the functional-option layer; `BuildGridEditor` becomes a free function taking `layout`, which lets `PanelBuilders` and `BaseMode.PanelBuilders` go away.
- Effort: M. Risk: low.

##### [Medium] Screen-fraction arithmetic is repeated 139 times
- Evidence: `int(float64(layout.ScreenWidth) * X)` or the height equivalent appears 139 times under `gui/` (44 in `guisquads`, 39 in `builders`, 17 in `guicombat`). `specs.LayoutConfig` has two methods and neither is called.
- Problem: Noise in every panel definition, and an easy place to mix up width and height. `BuildPanel` uses screen width for all four padding sides (`panels.go:221,247`) while `NewResponsiveRowPadding` uses width for horizontal and height for vertical (`layout.go:24-34`).
- Direction: Add two methods on `specs.LayoutConfig` (width fraction to pixels, height fraction to pixels) and use them. This is the single cheapest readability win in the GUI.
- Effort: S (mechanical). Risk: low.

##### [Low] Zero-value defaults make some configurations impossible
- Evidence: `CreateButtonWithConfig` replaces `MinWidth == 0` with 100 and `MinHeight == 0` with 40 (`widgets.go:178-183`), yet `BuildGridEditor` passes 0 with the comment "Let grid layout handle sizing" (`panels.go:522-523`). `Padding.Left == 0` triggers a full default in three factories (`widgets.go:195,318,405`), so zero left padding cannot be requested.
- Direction: Accept as is unless it bites; fix the misleading comment. Effort: S. Risk: low.

##### [Low] Four dialog constructors repeat the same button and close wiring
- Evidence: `dialogs.go:116-394`; the "call handler, then close window" closure appears eight times.
- Direction: Leave alone unless a fifth dialog is added; the shared helpers at the top already cover container, header and window. Not worth an abstraction.

#### Probable bugs
1. **Six panels are never created. Confidence: high (traced, not run).** `overworld_threat_info`, `overworld_tick_status`, `overworld_event_log`, `overworld_threat_stats` (`gui/guioverworld/overworld_panels_registry.go:173-246`) and `node_placement_node_list`, `node_placement_info` (`gui/guinodeplacement/nodeplacement_panels_registry.go:22-58`) request spec names that are not in `StandardPanels`. `BuildTypedPanel` returns a nil panel and nil text area, `BaseMode.BuildPanels` skips nil containers (`framework/basemode.go:173`), and the modes nil-check the text areas before every update (`overworldmode.go:242,256,286,301`; `nodeplacementmode.go:217,253`). Result: no error, no panel, and the refresh code for threat info, tick status, event log, threat stats and node placement text runs against nothing.

#### Dead code
- `widgets.go`: `SliderConfig`, `CheckboxConfig` (placeholders), `ContainerConfig.Title` (never set), `TextConfig.LayoutData`.
- `dialogs.go`: `dialogBaseConfig` (never used), `CreateMessageDialog` / `MessageDialogConfig` (0 callers).
- `lists.go`: `CreateInventoryList`, `InventoryListConfig` (0 callers).
- `layout.go`: `PaddingSide` and its eight constants (refer to a `NewResponsivePaddingSingle` that does not exist).
- `panels.go`: `PanelTypeSimple`, `PanelTypeList`, `PanelTypeGrid`, `TypedPanelConfig.Position/Size/Layout/Padding/ListConfig` (no caller uses them); position options `TopCenter`, `LeftCenter`, `LeftBottom`, `BottomCenter`, `BottomLeft`, `Center` and `HorizontalRowLayout`, `AnchorLayout`, `CustomPadding` are reachable only through unused `StandardPanels` entries.
- `panelspecs.go`: 11 of 13 entries.

#### Leave alone
- `CreateButtonWithConfig`, `CreateListWithConfig`, `CreateTextAreaWithConfig`, `CreateSmallLabel` / `CreateLargeLabel`, `CreateButtonGroup`, the `Anchor*` helpers and the three action-bar helpers. These are the real shared vocabulary and are used consistently.
- `BuildSelectionPanel`. It removed a real duplicate, and its doc comment stating when not to use it is exactly the right guard against over-generalising.
- `BuildGridEditor` (shared by squad editor and inspect panel).

---

### gui/widgetresources (507 lines, tests: no)

**Role:** Loads fonts, nine-slice images and colours used by all widgets; also hosts a background image cache.
**Verdict:** Needs attention. The resource half is fine apart from swallowed load errors; the cache half (190 lines) has no effect.

#### Design findings

##### [High] `CachedBackground` renders images that nothing ever draws
- Evidence: Every call discards the result: `_ = widgetresources.GetPanelBackground(...)` at `gui/builders/panels.go:307` and `widgets.go:329`; `PreCacheScrollContainerBackgrounds` and `PreCacheScrollContainerSizes` (called from `game_main/setup.go:47,59`) do the same internally (`cachedbackground.go:166-168,186-188`). Containers are still given the `NineSlice` itself as background (`widgets.go:333`).
- Problem: The cache allocates offscreen images (30 at startup for scroll containers, plus one per distinct panel size) and saves no draw work. It also props up the meaningless `EnableCaching` / `CreateStaticPanel` distinction in `builders`.
- Direction: Delete `cachedbackground.go`, the two `PreCache*` calls in `game_main/setup.go`, and `EnableCaching`.
- Effort: S. Risk: low.

##### [Medium] Resource loading swallows errors and duplicates work
- Evidence: `newPanelResources`, `newListResources` and `newTextAreaResources` return `nil` on any load error (`guiresources.go:49-56,81-115,178-212`), and the package-level `PanelRes`, `ListRes`, `TextAreaRes` are then dereferenced everywhere, so a missing asset surfaces as a nil-pointer panic far from the cause. `loadFont` and `loadButtonImage` errors are discarded with `_` (`:18-20`). `HexToColor` panics (`:303`). `newTextAreaResources` reloads the same seven image files as `newListResources`, and only its `Face` and `EntryPadding` fields are read (`builders/widgets.go:138-139`).
- Problem: The project's stated convention is a single boot-time fatal point with clear errors; this package fails late and silently instead. It also loads at package-init time, so anything that imports `gui/builders` needs the asset files on disk.
- Direction: Replace the package-level initialisers with one `Load() error` called from game setup, returning the failing path. Reduce `textAreaResources` to the two fields used, or reuse `ListRes`.
- Effort: S/M. Risk: low.

#### Dead code
- `cachedbackground.go` in full (see above); `titleBarBackgroundPool`, `buttonBackgroundPool` are declared and never used.
- `guiresources.go`: `panelResources.TitleBar` and `.Padding`, `listResources.EntryPadding`, all `textAreaResources` fields except `Face` and `EntryPadding`; the unexported `hexToColor` wrapper "for backward compatibility".

#### Leave alone
- The exported fonts, `ButtonImage`, `PanelRes.Image`, `ListRes` as the shared theme. One place for look and feel is correct.

---

### gui/widgets (408 lines, tests: no)

**Role:** Render-caching wrappers for lists and text areas, plus two small shared widget controllers.
**Verdict:** Needs significant rework. About 300 of the 408 lines are caching wrappers that are either inert or, where active, very likely wrong.

#### Design findings

##### [High] `CachedListWrapper` is never part of the widget tree
- Evidence: At all five creation sites the base list is added to the container and the wrapper is only stored: `gui/builders/selection_panel.go:103-104`, `gui/guisquads/squaddeployment_panels_registry.go:87,96`, `gui/guisquads/unitpurchase_panels_registry.go:99,109`, `gui/guiprogression/progression_panels_registry.go:147-148,160-161`.
- Problem: `Render` on the wrapper is never called, so there is no cache. The wrapper is a handle with a dirty flag nobody reads, and callers are told they "MUST call MarkDirty()".
- Direction: Delete `cached_list.go`; hold `*widget.List` directly and drop the `MarkDirty` calls and the duplicate `baseList` entries in the `Custom` maps.
- Effort: S. Risk: low (no behaviour change).

##### [High] `CachedTextAreaWrapper` is inert in two places and probably mis-renders in six
- Evidence: In `BuildSelectionPanel` the raw text area is added and the wrapper only stored (`selection_panel.go:114-115`), so it is inert there. The wrapper itself is added as the child at `gui/guisquads/squaddeployment_panels_registry.go:136`, `unitpurchase_panels_registry.go:148,167`, `squadeditor_panels_registry.go:442` and `gui/guiprogression/progression_panels_registry.go:169`. When active, `Render` draws the inner text area into an offscreen image sized to the widget (`cached_textarea.go:93-99`) and then blits that image at the widget's screen position (`:78-80`). ebitenui widgets draw at absolute screen coordinates (`widget.go:457-459`, `scrollcontainer.go:216-219` in v0.5.8), so the content lands in the buffer at the widget's screen offset and is offset again on blit; anything beyond the buffer is clipped. The wrapper also does not implement `SetupInputLayer` or `GetFocusers`, which `Container` forwards to children by type assertion (`container.go:248-256,269-291`), and skips `Widget.Render` (which fires hover and scroll events) on cached frames.
- Problem: For a text area whose top-left is further from the origin than its own size, nothing is visible; otherwise it is displaced and cropped. Scrolling cannot work on these text areas either way.
- Direction: Delete `cached_textarea.go` and `builders.CreateCachedTextArea`; use `CreateTextAreaWithConfig`. If profiling ever shows text areas to be a cost, cache at the text level, not by re-rendering a positioned widget into a local buffer.
- Effort: S. Risk: low. Not verified at runtime: open Unit Purchase or Squad Deployment and check whether the detail text is visible.

##### [Low] `TextDisplayComponent` overlaps `guisquads.DetailPanelComponent`
- Evidence: `textdisplay.go` (40 lines, 1 user at `gui/guicombat/combat_mode_init.go:169`) and `gui/guisquads/squadcomponents.go:14-80`, which combat also uses for two labels. Both are "a `*widget.Text` plus a formatter".
- Direction: Low value. If `DetailPanelComponent` moves into `guicombat` (see `gui/guisquads`), fold this into it.

#### Probable bugs
1. **Detail text areas built with `CreateCachedTextArea` are blank or displaced, and cannot scroll. Confidence: medium-high (traced through the ebitenui source, not run).** Affected: squad deployment detail, unit purchase detail and stats, squad editor perk detail, progression perk and spell detail.

#### Leave alone
- `SelectionPanelCore`. Small, has two real users (`gui/guispells/spell_panel.go`, `gui/guiartifacts/artifact_panel.go`), and its scope is documented. Move it into `builders` if `gui/widgets` is dissolved.

---

### gui/specs (290 lines, tests: no)

**Role:** Screen-fraction layout constants and the `LayoutConfig` screen-size holder.
**Verdict:** Healthy in concept, needs pruning. About a third of the constants and both methods are unused.

#### Design findings

##### [Medium] Stale constants describe layouts that no longer exist
- Evidence: 26 exported constants have no reference outside `layout.go`: `CombatActionButtonHeight/Width`, `CombatLogWidth`, `CombatSquadListHeight/Width`, `ExplorationMapWidth`, all six `Formation*`, `InfoContentWidth`, `InventoryListHeight/Width`, `PaddingLoose`, `PaddingStackedWidget`, `PanelHeightExtraTall`, `PanelHeightFull`, `PanelHeightThird`, `PanelWidthStandard`, `SquadDeployInstructHeight/Width`, `UnitPurchaseDetailHeight/Width`, `UnitPurchaseResourceWidth`. Section comments refer to inventory mode, info mode and the formation editor, none of which exist. `PanelHeightExtraTall` (0.6) is smaller than `PanelHeightTall` (0.75).
- Problem: The file reads as the authoritative layout table, but a third of it is fiction, so nobody can tell which numbers matter.
- Direction: Delete the 26 constants and the stale section comments. Keep the convention that panel sizes live here.
- Effort: S. Risk: low.

##### [Low] `LayoutConfig` has the wrong methods
- Evidence: `CenterWindow` and `GridLayoutArea` have 0 callers; `LayoutConfig.TileSize` is never read through a `LayoutConfig`. Meanwhile the fraction-to-pixel expression is hand-written 139 times (see `builders`).
- Direction: Delete the two methods and `TileSize`; add the two fraction-to-pixel helpers.
- Effort: S. Risk: low.

#### Dead code
- The 26 constants above, `LayoutConfig.CenterWindow`, `LayoutConfig.GridLayoutArea`, `LayoutConfig.TileSize`, and the "DEPRECATED METHODS REMOVED" comment block.

#### Leave alone
- The padding scale (`PaddingExtraSmall`, `PaddingTight`, `PaddingStandard`: 28 to 30 uses each) and the per-mode size constants that are in use.

---

## 18. GUI: squad management and menus

Packages: `gui/guisquads`, `gui/guiprogression`, `gui/guiunitview`, `gui/guiinspect`, `gui/guiartifacts`, `gui/guispells`, `gui/guistartmenu`.
All files were read in full. Nothing was run; every "bug" below is from tracing code paths, and the confidence stated reflects that.

### Layer overview

**The slice is two different things.** `guisquads`, `guiprogression`, `guiunitview` and `guistartmenu` are out-of-combat screens. `guispells`, `guiartifacts` and `guiinspect` are combat sub-panels: they are constructed only by `gui/guicombat/combat_mode_init.go:71-131`, depend on `framework.TacticalState`, `combatservices.CombatService` and `encounter.EncounterController`, and have no caller outside `guicombat`. They sit as top-level siblings of `guicombat` while `guicombat` already has a `combat*` subpackage convention (`combatanimation`, `combatbase`, `combatinput`, `combatvisualization`).

**Names collide across that split.** "Artifact GUI" is in two places: `guisquads.ArtifactMode` (inventory/equip screen, 790 lines across `artifactmode.go`, `artifact_refresh.go`, `artifact_panels_registry.go`) and package `guiartifacts` (in-combat activation). "Spell GUI" is likewise split between `guiprogression` (unlocking) and `guispells` (casting). A reader looking for one lands in the other.

**Dependency direction is mostly clean, with three warts.**
- `guicombat` imports `guisquads` only for `DetailPanelComponent` (`gui/guisquads/squadcomponents.go`), which `guisquads` itself never uses.
- `guisquads` imports `guiprogression` and `guiunitview` only to type-assert a mode and call a setter before a transition (`squadeditormode.go:227`, `squadeditor_roster.go:256`).
- `guisquads` imports `guiinspect` for two 3x3-grid helpers (`squadeditor_grid.go:81-124`).

**Recommended layout (small moves, no new packages beyond renames):**
1. Move `guispells`, `guiartifacts`, `guiinspect` under `gui/guicombat/` as `combatspells`, `combatartifacts`, `combatinspect` (or leave the paths and only fix the names; the point is that the name says "combat").
2. Move `DetailPanelComponent` into `guicombat`. That removes the `guicombat -> guisquads` import entirely.
3. Delete `SquadDeploymentMode` (unreachable, see below).
4. Leave `guiprogression`, `guiunitview`, `guistartmenu` as packages. They are small and coherent.

**One pattern costs the most across the slice:** panels are registered in a global map from `init()`, built through `OnCreate(result, mode)` closures that immediately type-assert the concrete mode, write widgets into `result.Custom[string]`, and are then read back by the same mode through `framework.GetPanelWidget[T](..., "key")`. In this slice that is 54 `result.Custom[...]` writes and 53 string-keyed reads, for panels that each have exactly one registering package and one consuming mode. All 21 registrations here are `ContentCustom` + `OnCreate`; none uses the descriptor's declarative fields. See the panel registry finding under `gui/framework`.

---

### gui/guisquads (3,928 lines, tests: no)

**Role:** Squad editor, unit purchase, artifact management and squad deployment modes, plus shared squad/commander selector widgets.
**Verdict:** Needs significant rework. The package works, but it carries one unreachable mode, an unreachable undo system, a list-refresh helper that scrambles panel layout, heavy copy-paste between sibling panels, and several game rules that only the GUI enforces.

#### Design findings

##### [High] Lists are refreshed by rebuilding the widget, through a duplicated helper that assumes a layout it does not have
- Evidence: `gui/guisquads/squadeditor_refresh.go:19-37` and `gui/guisquads/artifact_refresh.go:19-37` (identical bodies, different receivers, neither uses its receiver). Callers: `squadeditor_refresh.go:66,88,118`, `squadeditor_perks.go:99,114`, `artifact_refresh.go:121,298`.
- Problem: `replaceListInContainer` removes the old list, then re-adds `children[0]`, the new list, then everything else. That is only correct when the list is the second child. It is wrong for the squad selector panel (children are commander row, title, list, two buttons) and for the perk panel (two lists in one container). Because every list is rebuilt, each list's configuration is written twice, once in the registry and once in the refresh: squad list `squadeditor_panels_registry.go:147-157` vs `squadeditor_refresh.go:67-77`; unit list `:266-273` vs `:89-96`; roster `:324-330` vs `:119-130`; perk lists `:378-391`, `:408-421` vs `squadeditor_perks.go:100-111`, `:115-126`; artifact lists `artifact_panels_registry.go:120-126`, `:156-162` vs `artifact_refresh.go:122-151`, `:299-317`. The two copies have already drifted (perk list width is `panelWidth - 20` at build, `250` on refresh; the available-perk label gains a `[category]` suffix only after the first refresh).
- The same package already does it the simple way elsewhere: `squaddeploymentmode.go:152`, `unitpurchasemode.go:198` and `guiprogression/progression_refresh.go:44-47` call `list.SetEntries(...)`.
- Direction: use `SetEntries` on the existing `*widget.List` everywhere and delete both `replaceListInContainer` copies and every "rebuild" closure. Make list entries carry identity (squad `ecs.EntityID`, a small artifact-instance struct) with the label computed in `EntryLabelFunc`, which also removes the select-by-display-string lookups in `squadlists.go:50-59` and `artifact_refresh.go:128-149`, `:305-315`.
- Effort: M. Risk: low (no tests; behavior is visual and easy to check by hand).

##### [High] `SquadDeploymentMode` is registered but unreachable, and would misbehave if reached
- Evidence: `gui/guisquads/squaddeploymentmode.go` (288 lines), `squaddeployment_panels_registry.go` (167 lines). Registered at `setup/gamesetup/moderegistry.go:31,81`. A repo-wide grep for `squad_deployment` finds only the mode's own name, its `ModeConfig`, and the action-map name; no `GetMode`, `SetMode`, `EnterTactical` or `RequestTransition` targets it.
- Problem: about 455 lines in this package plus `tactical/squads/squadservices/squad_deployment_service.go` (75 lines, no other caller), `framework.DefaultSquadDeploymentBindings`, and `specs.SquadDeploy*` constants exist only for this mode. The logic also contradicts how deployment actually works now (`mind/combatlifecycle/enrollment.go:71`, `mind/encounter/encounter_setup.go:90` own `IsDeployed`): `placeSquadAt` sets `squadData.IsDeployed = true` from GUI code (`squaddeploymentmode.go:255-257`) and moves only the squad entity, not its members; `ClearAllSquadPositions` moves every squad in the world, enemies included, to (0,0) and never clears `IsDeployed`; "placed" is inferred from having a position component, which every squad has.
- Direction: delete the mode, its registry file, `SquadDeploymentService`, the bindings function, the layout constants and the two registration lines. If pre-combat manual placement is wanted later, build it on `combatlifecycle` rather than reviving this.
- Effort: S. Risk: low.

##### [High] The squad editor's undo/redo system cannot be triggered, and no mode ever clears command history
- Evidence: `squadeditormode.go:105-111` enables `Commands: true`; every unit operation goes through `sem.CommandHistory.Execute` (`squadeditor_roster.go:99,185,228,290`, `squadeditor_grid.go:75`); dialogs tell the player "You can undo with Ctrl+Z" (`squadeditor_roster.go:172,216`). But `framework.DefaultSquadEditorBindings` (`gui/framework/defaultbindings.go:61-74`) binds no undo/redo action, `SquadEditorMode.HandleInput` (`squadeditormode.go:284-363`) never checks `ActionUndo`/`ActionRedo`, there is no undo button, and `CommandHistory.HandleInput` (`gui/framework/commandhistory.go:90`) has no caller anywhere.
- Evidence (clearing): no call to `CommandHistory.Clear()` exists in the repo. `UnitPurchaseMode` keeps its history across exits (`unitpurchasemode.go:238`).
- Problem: five command types and an executor are maintained for a feature the player cannot reach in the editor. In the purchase screen, where undo does work, the history outlives the context that made it valid (see Probable bugs).
- Direction: decide whether editor undo is wanted. If yes, merge `DefaultUndoRedoBindings()` into the editor bindings and call `sem.CommandHistory.HandleInput(inputState)`. Either way, clear history in `Exit` for both modes, or make them share one executor so interleaved operations stay ordered.
- Effort: S. Risk: low to medium (no tests; stale-undo interactions need a manual pass).

##### [Medium] Game rules and data repair live in GUI code
- Evidence and problem, per item:
  - `squadeditor_roster.go:302-344` `backfillRosterWithSquadUnits` runs on every `Enter` (`squadeditormode.go:252`). It is roster data repair ("units created before roster tracking"), implemented as four nested loops in a mode file. `setup/gamesetup/initial_squads.go:141` already registers squad units at creation.
  - `squadeditor_roster.go:50-54` creates a squad entity and then registers it in the roster as two GUI steps. If `AddSquad` fails the entity is left unowned. Unlike every other editor operation it is not a command.
  - `squadeditor_perks.go:82-91` is the only place the "perk must be unlocked in the commander's library" rule is applied. `perks.EquipPerk` (`tactical/powers/perks/system.go:17`) does not check it, and the GUI filter is skipped when the commander resolves to 0. `EquipPerk` also takes `maxSlots` as a parameter although `MaxPerkSlots` is a constant in its own package and the GUI is its only production caller (`squadeditor_perks.go:165`).
  - `artifact_refresh.go:267-270` re-implements the max-slot fallback that `tactical/powers/artifacts/system.go:17-19,115-119` already owns.
- Direction: move the backfill into `tactical/squads/roster` (it disappears entirely with the roster rework described under `tactical/squads/roster`). Add a `CreateSquadCommand` (or a single roster function) for new squads. Give perks/progression one "equip for squad" entry point that checks unlock and slots, and drop the `maxSlots` parameter. Export an `artifacts` query for a squad's slot capacity and use it in the GUI.
- Effort: M. Risk: medium (no tests in this package; touches save-relevant roster state).

##### [Medium] `ArtifactMode` duplicates its two tabs end to end
- Evidence: widget fields `artifactmode.go:24-36`; panel construction `artifact_panels_registry.go:110-143` vs `:146-180`; refresh `artifact_refresh.go:58-153` vs `:253-319`; detail text `:187-249` vs `:322-357`; action handlers `:156-184` vs `:360-384`; the tab is tracked as the strings `"inventory"`/`"equipment"` in six places.
- Problem: any change to how an artifact is listed or described must be made twice, and the copies already disagree (equipment detail shows `Behavior`, inventory detail does not; inventory shows ownership counts, equipment does not).
- Direction: one small `artifactTab` struct (content, list, title, detail, button, selected ID) instantiated twice, and one artifact detail formatter placed beside the existing `templates.FormatSpellDetail` / `perks.FormatPerkDetail` so all three power types follow the same convention.
- Effort: M. Risk: low.

##### [Medium] Widgets take a string-keyed round trip through the panel registry
- Evidence: `squadeditormode.go:154-186`, `squadeditor_perks.go:37-55`, `artifactmode.go:95-116`, `unitpurchasemode.go:87-102`, `squaddeploymentmode.go:96-100`, matched by `result.Custom[...]` blocks at the end of every `OnCreate` in the four `*_panels_registry.go` files.
- Problem: each `OnCreate` already holds the typed mode (`sem := mode.(*SquadEditorMode)`), stores widgets in a `map[string]interface{}`, and the mode then fetches them back with a string and a type parameter. A typo or wrong type yields a silent nil that panics later (`Exit` dereferences `sem.attackLabel` unconditionally at `squadeditormode.go:267`). `unitpurchase_panels_registry.go:194-200` goes further and finds the Buy button by asserting `children[0]` of the action bar.
- Direction: assign to the mode's typed fields directly inside `OnCreate` and delete the `initializeWidgetReferences` functions. Whether the global registry itself stays is covered under `gui/framework`.
- Effort: M. Risk: low.

##### [Medium] Three ways to say "current commander", two near-identical selectors, and selection by display name
- Evidence: `ctx.GetSquadRosterOwnerID()` (`squadeditor_roster.go:24,57,309`, `squadeditor_refresh.go:138`); `commander.FindCommanderForSquad(currentSquad)` (`squadeditormode.go:225`, `squadeditor_perks.go:82`); `sem.commanderSelector.CurrentID()` (defined `commanderselector.go:45`, never used by the mode for these lookups). `CommanderSelector` (`commanderselector.go`) and `SquadSelector` (`squadselector.go`) are the same struct and methods with different `Load`. `CreateSquadList` resolves a click by comparing squad names (`squadlists.go:50-59`).
- Problem: the three commander lookups agree only by construction and diverge when a commander has no squads. In the squad editor `SquadSelector` is built with three nil widgets (`squadeditormode.go:83`) and used purely as an index holder while a separate `*widget.List` field named `squadSelector` shows the list, so the type's label/button half is dead weight there and the names mislead.
- Direction: resolve the commander once from the selector (or `GetSquadRosterOwnerID`) and pass it down. Collapse the two selectors into one small cycling helper parameterised by a label function, or keep `SquadSelector` as plain index state and let the two modes own their widgets. Key the squad list by entity ID.
- Effort: S to M. Risk: low.

##### [Low] `DetailPanelComponent` is combat code living in the squad package
- Evidence: `gui/guisquads/squadcomponents.go` (114 lines). Only users: `gui/guicombat/combat_mode_init.go:197,226`, `combat_turn_flow.go:26-27,54-55`, `combatmode.go:39-40`. It formats `HasActed`/`HasMoved` and faction mana.
- Problem: it is the only reason `guicombat` imports `guisquads`. Its `DetailFormatter func(interface{}) string` forces type assertions for two fixed types; `DefaultFactionFormatter` is unreachable because the only `ShowFaction` caller supplies its own formatter.
- Direction: move to `guicombat`; replace with two small typed helpers (or two typed formatter fields).
- Effort: S. Risk: low.

##### [Low] Mode wiring by string literal, type assertion and registration order
- Evidence: mode names such as `"squad_editor"`, `"unit_purchase"`, `"artifact_manager"`, `"progression_manager"`, `"unit_view"` are repeated as literals in constructors, `ModeConfig`, and every `GetMode` call (`squadeditormode.go:207,212,217,356`, `unitpurchase_panels_registry.go:188`, `squadeditor_roster.go:250`). Each mode sets name and return mode twice (constructor and `ModeConfig`, e.g. `artifactmode.go:53-54` and `:67-68`). `squadeditormode.go:100` decides its return mode by probing whether `"overworld"` is already registered, so it depends on the order in `moderegistry.go`. `squadeditor_roster.go:256` is an unchecked type assertion.
- Direction: mode-name constants in `framework`; set name/return mode in one place; pass the return mode (or a "has overworld" flag) into `NewSquadEditorMode` from `gamesetup`, which already knows which game mode it is building.
- Effort: S. Risk: low.

##### [Low] Repeated panel boilerplate and hardcoded pixels
- Evidence: the right-side panel block (size from `specs`, `CreateStaticPanel`, `AnchorEndStart`) is repeated at `squadeditor_panels_registry.go:248-262`, `:306-320`, `:355-369`. Pixel sizes such as 400x200, 380x100, 250, 120, 150 are scattered through `squadeditor_panels_registry.go:326-327,381,411,438`, `artifact_panels_registry.go:122-123,130-131`, `squadeditor_refresh.go:121-122`, next to otherwise percentage-based layout. `CreateSimpleStringList` is called with `ScreenWidth: 400` used as a pixel width.
- Direction: one local `newRightPanel(layout, wFrac, hFrac)` helper; derive list sizes from the panel size.
- Effort: S. Risk: low.

#### Probable bugs

| # | Bug | Evidence | Confidence |
|---|-----|----------|------------|
| 1 | Perk panel layout is scrambled after its first refresh: order becomes slot label, Available list, Equipped list, "Equipped:" label, Unequip, "Available:" label, Equip, detail. | `replaceListInContainer` always inserts at index 1 (`squadeditor_refresh.go:31-35`); called twice on one container at `squadeditor_perks.go:99,114`. | High |
| 2 | "Select Squad:" title renders below the squad list after the first refresh (every `Enter`). | Same helper; container order is commander row, title, list (`squadeditor_panels_registry.go:140-158`). | High |
| 3 | Undo/redo is unreachable in the squad editor although the UI advertises Ctrl+Z. | See High finding above. | High |
| 4 | Undoing a purchase after the unit was placed in a squad disposes a live squad member. Buy unit, return to editor, add it to a squad, re-enter purchase, Ctrl+Z. | History never cleared; `AddUnitCommand.Execute` assigns the roster's existing entity (`tactical/squads/squadcommands/add_unit_command.go:76-82`); `RefundUnitPurchase` removes and disposes without checking squad membership (`tactical/squads/squadservices/unit_purchase_service.go:233-244`). | Medium-high |
| 5 | Two squads with the same name: clicking the second selects the first. Nothing prevents duplicate names on create or rename. | `squadlists.go:50-59`. | High |
| 6 | In roguelike mode ESC does nothing in the squad editor when no sub-menu is open, though the button reads "Exploration (ESC)". | Return mode is `""` (`squadeditormode.go:99-102`), so `HandleCommonInput` finds no mode (`gui/framework/basemode.go:113-118`). | High |
| 7 | "Progression" opened for a commander with no squads edits commander 0 (shows zeros, unlocks fail). | `squadeditormode.go:223-229`. | Medium |
| 8 | Each list refresh re-adds every sibling widget, and ebitenui's `AddChild` appends three event handlers per child each time, so handler lists grow for the life of the mode. | `squadeditor_refresh.go:29-35`; `ebitenui@v0.5.8/widget/container.go:93-104`, `event/event.go:92-94`. | High on mechanism, low on visible impact |
| 9 | Deployment mode (unreachable today): a map click made before selecting a squad stays pending and places the squad as soon as one is picked; clicks on the action buttons or detail panel also place the squad under the cursor. | `squaddeploymentmode.go:173-184,206-220`; only the list rect is excluded. | High, moot if the mode is deleted |

#### Dead code
- `squaddeploymentmode.go`, `squaddeployment_panels_registry.go` in full, plus `tactical/squads/squadservices/squad_deployment_service.go`, `framework.DefaultSquadDeploymentBindings`, `specs.SquadDeploy*`, and `moderegistry.go:31,81`.
- `result.Custom["baseList"]` at `squaddeployment_panels_registry.go:100` and `unitpurchase_panels_registry.go:112`: written, never read.
- `DefaultFactionFormatter` (`squadcomponents.go:98`): unreachable in practice.
- `CommanderSelector.ShowPrevious` / `ShowNext` (`commanderselector.go:65-72`): pass-throughs to `Cycle`, one caller each.
- `widgets.CachedListWrapper` in `squaddeployment_panels_registry.go:87` and `unitpurchase_panels_registry.go:99`: the raw list is what gets added to the container (`:97`, `:109`), so the wrapper never renders and the `MarkDirty()` calls (`unitpurchasemode.go:199,226`, `squaddeploymentmode.go:153`) do nothing.
- `UnitPurchaseMode.Enter` (`unitpurchasemode.go:221-226`) repeats `refreshUnitList` (`:192-200`) instead of calling it.

#### Performance
- Nothing per-frame. All refresh work is event-driven. The only growth concern is bug 8. `refreshAfterCommand` rebuilds every list after every command, which is fine for a menu and disappears with `SetEntries`.

#### Leave alone
- The file split of the squad editor (`squadeditormode.go` lifecycle, `_grid`, `_roster`, `_perks`, `_refresh`) is sensible; do not split further.
- `UnitPurchaseMode` delegating cost, validation and the transaction to `UnitPurchaseService` and `PurchaseUnitCommand` is the right shape and is the model the editor's ad hoc operations should follow.
- `perkPanelController` as a small controller inside the editor is a reasonable size and boundary once its lists use `SetEntries`.
- Unit operations going through `squadcommands` is good; keep it.
- Selection and toggle state on the mode structs (`selectedGridCell`, `showAttackPattern`, `activeTab`) is UI state in the right place.

---

### gui/guiprogression (572 lines, tests: no)

**Role:** Mode for spending a commander's Skill and Arcana points to unlock perks and spells.
**Verdict:** Healthy. It is the cleanest mode in the slice: one generic library-panel controller drives both panels, lists are updated with `SetEntries`, and all mutation goes through `tactical/powers/progression`.

#### Design findings

##### [Low] The caller must remember to set the commander before transitioning
- Evidence: `progressionmode.go:33-37`; only caller `gui/guisquads/squadeditormode.go:216-230`.
- Problem: forgetting `SetCommanderID`, or passing 0, silently shows an empty library (see guisquads bug 7). `activeCommanderID()` (`progressionmode.go:55-57`) is a getter that adds nothing over the field.
- Direction: in `Enter`, if the commander is 0, fall back to `Context.GetSquadRosterOwnerID()` and set a status message. Drop the getter.
- Effort: S. Risk: low.

##### [Low] Affordability is re-derived in the GUI
- Evidence: `progression_controller.go:69-71` compares points to cost; `progression.UnlockPerk`/`UnlockSpell` enforce the same rule on click.
- Direction: only worth changing if unlock rules grow (prerequisites, tiers). Then add a `progression.CanUnlock...` query and use it for the button state.
- Effort: S. Risk: low.

#### Dead code
- `widgets.CachedListWrapper` at `progression_panels_registry.go:147,160`: the base lists are added to the container (`:148,161`), so the wrapper and the `MarkDirty()` calls in `progression_refresh.go:45,47` do nothing. Hold `*widget.List` directly.

#### Leave alone
- `librarySource` / `libraryPanelController` (`progression_controller.go:27-85`). This is an abstraction that earns its keep: two real instances, no speculative hooks.
- `buildLibraryPanel` returning the `OnCreate` closure for both panels (`progression_panels_registry.go:111-184`).
- Debug buttons gated on `config.DEBUG_MODE` (`progression_panels_registry.go:91`).

---

### gui/guiunitview (195 lines, tests: no)

**Role:** Read-only screen showing one unit's attributes, level and XP.
**Verdict:** Needs attention. Tiny and simple, but its only keyboard exit is broken and it ships an ungated cheat button with a private time-seeded RNG.

#### Design findings

##### [Medium] Private wall-clock RNG and an ungated debug button
- Evidence: `unitviewmode.go:23,29` creates `rand.New(rand.NewSource(time.Now().UnixNano()))`; `:73-76` awards 100 XP; the button is added unconditionally at `unitview_panels_registry.go:55-59`. The sibling mode gates its debug buttons (`gui/guiprogression/progression_panels_registry.go:91`). The project's shared RNG is `core/common/randnumgen.go`, and real XP awards use `combatlifecycle`'s seeded RNG (`mind/combatlifecycle/reward.go:21`).
- Problem: a GUI package owns a second, non-reproducible random source that feeds level-up stat rolls, and any player can grant XP from a normal screen.
- Direction: wrap the button in `config.DEBUG_MODE`, drop the `rng` field, and use the shared RNG source (see the RNG theme in section 3).
- Effort: S. Risk: low.

##### [Low] Attribute formatting duplicated with the purchase screen
- Evidence: `unitviewmode.go:95-104` and `gui/guisquads/unitpurchasemode.go:142-168` both lay out the same `common.Attributes` fields by hand.
- Direction: one `formatAttributes(attr)` helper if either is touched again; not worth a change on its own.
- Effort: S. Risk: low.

#### Probable bugs
- ESC does not leave the mode although the button says "Back (ESC)". `UnitViewMode` has no `GetActionMap`, so it is not an `ActionMapProvider`; `UIModeManager` clears `ActionsActive` each frame and only fills it for providers (`gui/framework/modemanager.go:188-196`); `HandleCommonInput` therefore never sees `ActionCancel` (`gui/framework/basemode.go:113`). No other code reads `KeyEscape`. Confidence: high. Fix: give the mode `framework.CommonBindings()`.

#### Leave alone
- The single-panel structure and `refreshUnitDisplay`. Do not add a controller or split files.

---

### gui/guiinspect (182 lines, tests: no)

**Role:** Controller and builder for the in-combat squad formation / attack-pattern inspect panel, plus two 3x3 grid helpers reused by the squad editor.
**Verdict:** Healthy. Small, single-purpose, no rules in it. The issues are placement and one duplicated loop.

#### Design findings

##### [Low] Formation-grid population is written twice
- Evidence: `inspect_panel.go:97-122` and `gui/guisquads/squadeditor_grid.go:86-103` both iterate `GetUnitIDsInSquad`, call `Queries.GetUnitGridInfo`, prefix leaders and bounds-check the anchor cell. They differ only in the HP/dead suffix and in the leader marker.
- Direction: export one `PopulateFormationCells(cells, squadID, queries, showHP)` here beside `PopulateAttackGridCells` and use it from the editor.
- Effort: S. Risk: low.

##### [Low] Combat panel outside the combat tree; half-split construction
- Evidence: `InspectPanelType = "combat_inspect_grid"` (`inspect_panel.go:16`); registered in `gui/guicombat/combat_panels_registry.go:289-293`; widgets wired in by `SetWidgets` from `gui/guicombat/combat_mode_init.go:125-131`.
- Problem: the panel type lives here, the registration lives in `guicombat`, and construction is `BuildPanel` here plus a `Custom` round trip there.
- Direction: when the combat sub-panels are regrouped, have `BuildPanel` return the controller already wired (it creates all four widgets itself) and drop `SetWidgets`.
- Effort: S. Risk: low.

#### Leave alone
- `PopulateAttackGridCells` and `ClearGridCells` as shared free functions.
- The controller's `Show`/`Hide`/`PopulateGrid` surface.

---

### gui/guiartifacts (329 lines, tests: no)

**Role:** In-combat artifact activation: option list, targeting click, execution.
**Verdict:** Needs attention. The panel half is fine; the handler half builds the power-pipeline context and enforces targeting rules that belong in `tactical/powers/artifacts`.

#### Design findings

##### [Medium] The GUI assembles the artifact execution context and validates targets
- Evidence: `artifact_handler.go:150-159` builds `artifacts.NewBehaviorContext(powercore.NewPowerContext(manager, cache, 0, nil), tracker)` by hand. The canonical builder is `ArtifactDispatcher.makeBehaviorContext` (`tactical/powers/artifacts/dispatcher.go:44-49`), which carries the injected logger. Friendly/enemy target validation is at `artifact_handler.go:72-85`; `artifacts.ActivateArtifact` (`tactical/powers/artifacts/registry.go:48-57`) checks neither target side nor charge.
- Problem: player activations run with a nil logger, so the `ctx.Log(...)` calls in `tactical/powers/artifacts/behaviors.go:165,192,210,282,312` produce nothing for exactly the actions a player most wants logged. Any second caller (AI, tests, simulator) has to re-implement target validation, charge check and context construction.
- Direction: add one activation entry point on the dispatcher or `CombatService` (validate charge, validate target side, build context, activate) and have the handler call it and then do only UI cleanup.
- Effort: S to M. Risk: medium (no tests here; `artifacts` has tests to extend).

##### [Low] Small inconsistencies with the sibling spell panel
- Evidence: `artifact_handler.go:113` uses the literal `"major"` although `artifacts.TierMajor` exists (`tactical/powers/artifacts/components.go:39`). Player position arrives by setter (`:36`, called at `gui/guicombat/combat_mode_init.go:106`) while `guispells` takes it in its deps struct. `ArtifactPanelController.Show` sets `BattleState.InArtifactMode` directly (`artifact_panel.go:112`) while the handler clears it in two places (`artifact_handler.go:92-93,166-167`); the spell side routes all transitions through its handler. Activation failure is only `fmt.Printf` (`:162`).
- Direction: use the constant; put `PlayerPos` in `ArtifactActivationDeps`; add `EnterArtifactMode()` on the handler and call `CancelArtifactMode()` from `executeArtifact`.
- Effort: S. Risk: low.

#### Leave alone
- `ArtifactPanelController` on top of `widgets.SelectionPanelCore`. The shared core removed the real duplication with the spell panel; the remaining parallel methods (`Toggle`, `OnCancelClicked`, `Handler`) are a few lines each and not worth another abstraction.
- `ArtifactOption` as the list entry type.

---

### gui/guispells (533 lines, tests: no)

**Role:** In-combat spell casting: spell list, single-target and AoE targeting, execution, VFX trigger.
**Verdict:** Needs attention. The turn-economy and targeting rules for spells exist only in this GUI package, and one of them is wrong for buff spells.

#### Design findings

##### [High] Spell casting rules are enforced by the GUI, not by `spells.ExecuteSpellCast`
- Evidence: "a squad that has acted cannot cast" is checked only when the panel opens (`spell_handler.go:116-132`, used at `spell_panel.go:110`). "casting consumes the squad's action" is applied by the GUI after the cast (`spell_handler.go:312`). "targets must be enemies" is applied by the GUI (`:168`, `:224`). `ExecuteSpellCast` (`tactical/powers/spells/system.go:96-148`) checks combat-active, mana and spellbook only. `mind/ai/ai_controller.go:86` has a placeholder comment for AI casting that would call `ExecuteSpellCast` directly.
- Problem: the simulation's spell rules are incomplete without the GUI. An AI or simulator caller would cast without spending an action, after already acting, at any target. This is the clearest simulation/presentation leak in the slice.
- Direction: move the acted check, `MarkSquadAsActed`, and target-side validation (driven by `spell.EffectType`) into `ExecuteSpellCast` or a thin `CastSpell` wrapper in `tactical/powers/spells`. The handler keeps click-to-squad resolution, overlay, VFX and cache invalidation.
- Effort: M. Risk: medium (no tests in this package or in `tactical/powers/spells`).

##### [Medium] Duplicated rule checks and a third spell formatter
- Evidence: `CanSelectedSquadCast` (`spell_handler.go:116-132`) and `CastBlockReason` (`:136-150`) are the same three checks; one is `CastBlockReason() == ""`. Mana affordability is tested by `spells.HasEnoughMana` at `:44` and re-derived as `currentMana >= spell.ManaCost` at `spell_panel.go:74-75`. `spell_panel.go:77-83` formats spell details by hand while `templates.FormatSpellDetail` (`templates/format.go:12`) exists and is used by `guiprogression`; the hand-rolled one prints "Damage: 0" for buff spells and omits their modifiers and duration.
- Direction: keep `CastBlockReason` only; use `HasEnoughMana` in the panel; extend `FormatSpellDetail` to cover buff fields and call it here.
- Effort: S. Risk: low.

##### [Low] Targeting state is spread over three places; pixel math done two ways
- Evidence: the casting state is `BattleState.InSpellMode`, `BattleState.SelectedSpellID` and `handler.activeShape` together (`spell_handler.go:34,53,57,79-80,315-316`). `spell_handler.go:337-338` multiplies by `coords.ScreenInfo.TileSize` by hand while `:189,214` use `coords.CoordManager.LogicalToPixel`. AoE targets are collected from a map (`:219-236`), so effects are applied in random order.
- Direction: use `CoordManager.LogicalToPixel` in `triggerSpellVX`. Collect targets in tile order (slice plus seen-set) so results and logs are reproducible. The three-variable state is tolerable at this size; do not build a state machine for it.
- Effort: S. Risk: low.

#### Mechanics notes
- Buff spells can only be cast on enemies. `war_cry` and `arcane_shield` are `targetType: single`, `effectType: buff` (`resources/assets/gamedata/spelldata.json:234-262`), and the only single-target path rejects any non-enemy squad (`spell_handler.go:167-170`). `SpellDefinition.IsBuff()`/`IsDebuff()` (`templates/spelldefinitions.go:83-90`) exist for this decision and have no callers.
- Spells have no range and no line-of-sight as coded: `SpellDefinition` has no range field and neither the handler nor `ExecuteSpellCast` checks distance. A single-target spell can hit any enemy squad on the map. Flagging in case this is not intended; no change proposed.
- Casting uses the squad's attack action but not its movement, and is blocked after attacking; this is consistent, just located in the GUI (see High finding).

#### Probable bugs
- Buff spells target enemies only, so War Cry strengthens an enemy squad and cannot be cast on your own. Confidence: high (input routing confirmed at `gui/guicombat/combatinput/handler.go:100-106`).

#### Dead code
- `SpellCastingHandler.GetAvailableSpells` (`spell_handler.go:84-90`): no callers. Its only use of `spells.GetCastableSpells` makes that function dead too outside tests.
- `SpellPanelDeps.BattleState` (`spell_panel.go:15`): set at `gui/guicombat/combat_mode_init.go:82`, never read.

#### Leave alone
- The handler / panel-controller split and the narrow `SpellCastingDeps` struct.
- AoE overlay handling (`HandleAoETargetingFrame`, `ClearOverlay`).

---

### gui/guistartmenu (105 lines, tests: no)

**Role:** Pre-game menu that returns the chosen game mode to `game_main`.
**Verdict:** Healthy. Self-contained, polled by `game_main/main.go:74-88`, no game state.

#### Design findings

##### [Low] GUI package imports the save system to ask one yes/no question
- Evidence: `startmenu.go:7,71` calls `savesystem.HasSaveFile()`. Elsewhere save/load reaches the GUI by injection (`UIContext.SaveGameCallback`, `gui/framework/uimode.go:56-57`).
- Direction: `NewStartMenu(hasSave bool)`, with `game_main` asking the save system.
- Effort: S. Risk: low.

#### Leave alone
- Everything else. It rightly does not use the mode framework; it runs before the framework exists.

---

## 19. GUI: combat, overworld, raid, node placement, exploration

Scope: `gui/guicombat` (+ `combatbase`, `combatinput`, `combatanimation`, `combatvisualization`), `gui/guioverworld`, `gui/guiraid`, `gui/guinodeplacement`, `gui/guiexploration`. About 7,200 non-test lines, no tests in any of the nine packages. Every file was read in full. Bugs were traced through the code but not reproduced by running the game.

### Layer overview

**How the packages relate.** Each mode embeds `framework.BaseMode`, registers panels in an `init()` via `framework.RegisterPanel`, and is constructed in `setup/gamesetup/moderegistry.go`. The only cross-mode import inside the slice is `guinodeplacement -> guioverworld` (it reuses `OverworldRenderer`). `guicombat` additionally imports `guispells`, `guiartifacts`, `guiinspect` and `guisquads`.

**The modes follow the framework for panels and input, and bypass it for everything else.** `PanelRegistry`, `ActionMap` and `SubMenuController` are used consistently. What is not shared is the code around them, so the same 20-line helpers are copied per mode (sub-menu builder x4, text-area getter x9, see below).

**The recurring structural problem is that turn flow, campaign progression and lifecycle live in GUI code.** This is the main finding of the slice and it is the direct cause of four of the probable bugs:

| Rule or sequence | Where it lives now | Where the sim-side counterpart is |
|---|---|---|
| End turn -> victory check -> run AI -> chain animations -> next turn | `gui/guicombat/combat_turn_flow.go` | nowhere; `tools/combat_analysis/combat_simulator/combat_runner.go:57-95` has its own second loop |
| Combat exit (resolution, rewards, teardown) | triggered from `CombatMode.Exit`, `gui/guicombat/combatmode.go:129-171` | `combatlifecycle.ExecuteCombatExit` (fine, but only reachable through a mode hook) |
| Post-combat destination | `TacticalState.PostCombatReturnMode`, a GUI struct written by `mind/encounter` | `CombatSetup.PostCombatReturnMode()` already derives it |
| Recruit a commander (capacity, location, cost, creation, refund) | `gui/guioverworld/overworld_action_handler.go:273-370` | `setup/gamesetup/initial_commanders.go:26-56` repeats the creation half |
| Raid: resume, auto-start, floor advancement, auto-deploy fallback, combat-vs-non-combat room | `gui/guiraid/raidmode.go` | `campaign/raid/raidrunner.go` has overlapping versions of most of these |
| Regenerate the world map | `gui/guiexploration/exploration_panels_registry.go:49-79` | `setup/gamesetup/bootstrap.go:96` + `helpers.go:18` |

**Layout recommendations (net effect: nine packages become six, no new files).**
1. Fold `combatbase` and `combatinput` back into `guicombat`. They are not independent units; `combatbase` exists only to break the import cycle that the split created. Details under `gui/guicombat/combatbase`.
2. Keep `combatanimation` and `combatvisualization` as sub-packages. Both have a narrow, one-directional interface.
3. Fold `guinodeplacement` into `guioverworld`. It shares `OverworldState`, the renderer, the overworld mode manager and the camera, and it is the only package that needs `OverworldRenderer` exported.
4. Leave `guiraid` and `guiexploration` as packages.

**Cross-mode duplication to remove once, in the framework/builders layer.**
- Sub-menu builder, four near-identical copies: `gui/guicombat/combat_panels_registry.go:19-38`, `gui/guioverworld/overworld_panels_registry.go:26-45`, `gui/guiexploration/exploration_panels_registry.go:27-46`, `gui/guisquads/squadeditor_panels_registry.go:28`. The only difference is `specs.CombatSubMenuOffset` versus the literal `0.15`, which is the same value (`gui/specs/layout.go:181`). Direction: one `builders.CreateSubMenu(layout, subMenus, name, buttons)`.
- Hand-written widget getters, nine of them (`GetOverworld*` x5, `GetNodeListText`, `GetPlacementInfoText`, `GetExplorationMessageLog`, `GetCombatAnimationPromptLabel`), all reimplementing `framework.GetPanelWidget[T]` (`gui/framework/panelregistry.go:152`), which `guicombat` and `guiraid` already use. Direction: delete the getters, call the generic.
- "Detail text area" panel registration, eight copies that differ only in `SpecName`, custom key and placeholder: `overworld_panels_registry.go:154-246` (5), `nodeplacement_panels_registry.go:22-57` (2), `exploration_panels_registry.go:117-134` (1). Direction: one helper `registerDetailPanel(panelType, specName, key, placeholder)` in builders or framework.
- Mode names are bare strings: 44 literals in this slice (73 across gui/setup/campaign/mind), and behaviour branches on them (`combatmode.go:90,130`, `raidmode.go:103`). There are no constants. Direction: `framework.ModeCombat` etc.; this is a prerequisite for safely touching the transition logic.
- Node display-name lookup with fallback, five copies: `overworld_action_handler.go:151-155`, `overworld_formatters.go:29-33`, `nodeplacementmode.go:190-194`, `nodeplacement_renderer.go:64-67` and `:82-88`. Direction: `NodeRegistry.DisplayName(id)` in `campaign/overworld/core`.

---

### gui/guicombat (1,114 lines, tests: no)

**Role:** The combat UI mode: panel wiring, per-frame refresh, and the turn/AI/victory sequencing. **Verdict:** Needs significant rework, because the battle's control flow and its teardown are driven from mode hooks and a single-slot transition queue, which produces at least two traced lifecycle bugs.

#### Design findings

##### [High] Combat teardown is a side effect of `CombatMode.Exit`, and the turn loop lives in the GUI
- Evidence: `gui/guicombat/combatmode.go:129-171` (Exit runs `ExitCombat`, battle-log export, `ClearExitState`, `ResetThreatState`, guarded by a mode-name string check at `:130`); `gui/guicombat/combat_turn_flow.go:63-74,116-137,152-196,200-220` (end turn, AI turn, animation chaining, victory); `gui/framework/modemanager.go:64-69` (`RequestTransition` holds one pending transition and silently overwrites it).
- Problem: Ending a battle is not an operation anyone calls; it happens only if the mode manager happens to call `Exit` on the right mode with the right destination. The flow reaches that point through recursive callbacks (`playNextAIAttack` -> `SetOnComplete` -> `advanceAfterAITurn` -> `executeAITurnIfNeeded`) that issue several `RequestTransition` calls in one frame. Because the queue is a single slot, the result depends on call order. The headless simulator cannot reuse any of it and has its own loop.
- Direction: (1) Introduce one explicit `finishCombat(result)` in the turn flow that performs exit, export and state reset and then requests the transition; `Exit` keeps only visual cleanup. (2) Move the sequencing itself (end turn, victory check, run AI faction, report queued attacks) into a non-GUI turn driver next to `combatservices`, returning "what happened" so the GUI only decides what to animate; the simulator then uses the same driver. Step 1 is small and fixes the bugs; step 2 is the structural fix.
- Effort: S for step 1, L for step 2. Risk: medium (no tests).

##### [High] `TacticalState` mixes per-click UI flags with per-battle routing state, and one `Reset()` clears both
- Evidence: `gui/framework/contextstate.go:17-44,75-101`; `gui/guicombat/combat_turn_flow.go:70` (reset on every player End Turn); `mind/encounter/encounter_service.go:185-192` (sim layer writes `TriggeredEncounterID` and `PostCombatReturnMode` into it); `gui/guicombat/combat_turn_flow.go:225-233`.
- Problem: The struct's own comment says it is transient UI state, but it carries where the game goes after the battle. `Reset()` is documented as "for starting a new battle" and is called each turn. `TriggeredEncounterID` is set and then zeroed by the very next line (`encounter_service.go:185-186`) and is never read anywhere in the repo.
- Direction: Remove `PostCombatReturnMode` and `TriggeredEncounterID` from `TacticalState`. The return mode is already derivable from the active encounter (`CombatSetup.PostCombatReturnMode()`); expose it on the `EncounterController` port and read it at combat end. Split `Reset()` into a selection reset (per turn) and nothing else. This also shrinks `encounter.ModeCoordinator` by two methods.
- Effort: S. Risk: low.

##### [Medium] Three dependency-wiring styles inside one mode
- Evidence: `combatbase/deps.go:10-32` (deps struct); `combatinput/handler.go:49-72` (six setters: `SetPlayerPosition`, `SetCurrentFactionID`, `SetSpellPanel`, `SetArtifactPanel`, `SetInspectPanel`, `SetVisualization`); `combat_turn_flow.go:31-60` (seven-argument constructor plus `SetUIComponents`); `combatmode.go:219-220` (two of the setters are called every frame).
- Problem: `Initialize` is 140 lines of ordered wiring (`combat_mode_init.go:22-160`) whose order matters ("before initializeUpdateComponents which sets UI refs on it"). `CombatMode` and `CombatTurnFlow` each hold their own copies of the three refresh components. A reader cannot tell what a handler depends on without reading all of `Initialize`.
- Direction: After merging the sub-packages (see `combatbase`), let the handlers and turn flow hold a pointer to `CombatMode` (or one deps struct that includes the context) and read `PlayerData.Pos` and the current faction when needed. The per-frame setters, `SetUIComponents`, and the duplicate component fields go away.
- Effort: M. Risk: low.

##### [Medium] Panel registry file repeats the same panel four times
- Evidence: `combat_panels_registry.go:77-106,108-137,139-168,170-202`: turn order, faction info, squad detail and layer status differ only in size constants, anchor, label size and initial text.
- Problem: About 120 lines for four labelled boxes; every tweak to padding is made four times.
- Direction: One local helper `textPanel(widthFrac, heightFrac, anchor, label)` or four entries in a table. Do not add a framework abstraction for it.
- Effort: S. Risk: low.

##### [Low] Thin pass-through click handlers
- Evidence: `combat_mode_panels.go:51-77`: seven one-line methods that forward to `actionHandler`, `spellPanel`, `inputHandler`, `turnFlow`.
- Direction: Pass the target methods directly in the `ButtonSpec`s; these disappear naturally with the wiring cleanup.
- Effort: S. Risk: low.

#### Mechanics notes
- **Victory is only evaluated when a turn ends** (`combat_turn_flow.go:200-207`). A player who destroys the last enemy squad stays in a finished battle until they press End Turn. The debug kill paths (`combatinput/debug.go`, `combatbase/handler.go:206-216`) have the same property. If a turn driver is introduced, check victory after every resolved attack.
- **Attack mode is vestigial.** Clicking an enemy with a squad selected attacks immediately without attack mode (`combatinput/handler.go:320-324`). `InAttackMode` only gates the 1/2/3 hotkeys (`combatinput/handler.go:209-222`), which index into `GetEnemySquads`, a list the player never sees and whose order is not shown anywhere. Nothing renders differently in attack mode (`combatvisualization/manager.go:76-95` never reads the flag). `contextstate.go:23` says game logic must not read these flags, yet `combatbase/handler.go:66` does. Decide whether attack mode is a real state (then gate the click path on it and show it) or remove the flag, the button and the three target hotkeys.
- **Health-bar toggle resets every turn** because `TacticalState.Reset()` clears `ShowHealthBars` (`contextstate.go:83`, called from `combat_turn_flow.go:70`).

#### Probable bugs
- **A battle that ends during the AI's animation chain never runs combat exit.** Confidence: high by trace, not reproduced. In `playNextAIAttack`'s terminal branch (`combat_turn_flow.go:153-161`) the code requests a transition to `combat` and then calls `advanceAfterAITurn`; if `CheckAndHandleVictory` (`:96-113`) finds the battle over it requests the return mode, overwriting the pending transition (`modemanager.go:64-69`). The current mode is still `combat_animation`, so the manager runs `animation.Exit` then `exploration.Enter` (`modemanager.go:72-88`); `CombatMode.Exit` is skipped. Consequence: no `ExitCombat`, no resolution or rewards, no teardown, and `EncounterService.activeEncounter` stays set so the next `TransitionToCombat` fails with "encounter already in progress" (`mind/encounter/encounter_service.go:169-171`). The typical trigger is the player being wiped out by AI attacks. `RaidMode.Enter` also checks `fromMode == "combat"` (`gui/guiraid/raidmode.go:103`), which would be `combat_animation` here.
- **Raid battles that last past the first End Turn return to exploration instead of raid.** Confidence: high by trace, not reproduced. `HandleEndTurn` calls `TacticalState.Reset()` (`combat_turn_flow.go:70`), which sets `PostCombatReturnMode = ""` (`contextstate.go:100`). It is only ever set once, in `TransitionToCombat` (`encounter_service.go:190-192`). Any later victory resolves to `"exploration"` (`combat_turn_flow.go:225-233`), so `RaidMode.Enter` never sees `fromMode == "combat"`, the encounter summary is never shown, and `LastEncounterResult` stays stale. A battle won on turn one (for example with the debug kill-all key) hides this.
- **The player cannot flee.** Confidence: high. `HandleInput` swallows ESC with the comment "use the Flee button instead" (`combatmode.go:225-228`); there is no Flee button in `combat_mode_panels.go:32-48` or anywhere under `gui/`; `HandleFlee` (`combat_turn_flow.go:77-91`) has no callers in the repo. The whole `ExitFlee` path (`combatlifecycle.DetermineExitReason`, `EncounterExitHooks.restoreEncounterSprite`, raid retreat-on-flee) is unreachable from the game.
- **Two AI factions in a row: the second faction's first animation loses auto-play and colours.** Confidence: medium, latent (standard encounters spawn two factions). The chain requests `combat_animation` while already in it; the self-transition runs `Exit`, which nils the colour maps and clears `autoPlay` (`combatanimation/mode.go:173-180`) after `SetCombatants`/`SetAutoPlay(true)` were already applied (`combat_turn_flow.go:169-178`).

#### Dead code
- `CombatTurnFlow.HandleFlee` (`combat_turn_flow.go:77-91`): no callers (see bug above; either wire it or remove it together with the flee path).
- `viewportSize := 30` (`combatmode.go:197`) is a hard-coded duplicate of `config.DefaultZoomNumberOfSquare`.
- Import alias `combatpanels` for `combatbase` in `combat_panels_registry.go:9` only; every other file uses `combatbase`.

#### Performance
- `CombatMode.Exit` calls `ClearAllVisualizations` on every hand-off to the animation mode (`combatmode.go:173`), which rewrites the colour matrix of every map tile and marks tile colours dirty (`combatvisualization/threat.go:178-183`) even when the threat overlay was never turned on. Guard on `IsActive`.

#### Leave alone
- `Update`'s dirty checks on faction and selected squad (`combatmode.go:177-193`) are the right level of refresh logic.
- Injecting the combat service through a factory so that `mind/ai` stays out of the GUI (`setup/gamesetup/moderegistry.go:96-106`) is a good boundary.
- `registerCombatCallbacks` (`combat_mode_init.go:241-265`): cache invalidation driven by sim events rather than by the GUI guessing is the right direction; the turn-flow rework should extend this pattern, not replace it.

---

### gui/guicombat/combatbase (262 lines, tests: no)

**Role:** Holds the deps struct, the panel-type constants and `CombatActionHandler` so that `guicombat` and `combatinput` can both import them. **Verdict:** Needs attention; it is a cycle-breaking grab-bag rather than a unit with its own responsibility.

#### Design findings

##### [Medium] `combatbase` and `combatinput` are not independent packages; merge both into `guicombat`
- Evidence: `combatbase` contains three unrelated things (`deps.go`, `panels.go`, `handler.go`). `combatinput` is used only by `guicombat` and needs six setters to receive `guicombat`'s objects. `combatbase/handler.go:7,106-114` imports `combatanimation` for a concrete type assertion, so "base" is not a leaf. Stale comments in `gui/guioverworld/overworld_action_handler.go:23`, `overworld_input_handler.go:18` and `overworld_deps.go:12` still cite `gui/guicombat/combat_action_handler.go`, `combat_input_handler.go` and `combatdeps.go`, which shows these were files of one package before the split.
- Problem: The split buys no encapsulation (everything the pieces share had to be exported: all `CombatModeDeps` fields, every handler method, every setter) and costs a larger public surface, setter injection, and an extra package to navigate. For comparison `gui/guisquads` is 3,900 lines in one package; the combat mode is 3,080 lines in five.
- Direction: Move the seven files of `combatbase` and `combatinput` into `gui/guicombat` (same file count, prefixed names as before). Unexport what no longer crosses a boundary. Keep `combatanimation` and `combatvisualization` separate.
- Effort: S-M (mechanical). Risk: low.

##### [Medium] The action handler reaches into the animation mode by name and concrete type
- Evidence: `combatbase/handler.go:105-116`; the same lookup-and-assert block is in `gui/guicombat/combat_turn_flow.go:167-187`.
- Problem: Two places know the mode's registered name, its concrete type and its setup protocol (`SetCombatants`, `SetOnComplete`, optional `SetAutoPlay`, then `RequestTransition`). The player path installs a no-op completion callback purely to satisfy the protocol.
- Direction: One `playAttackAnimation(attackerID, defenderID, autoPlay, onDone)` helper owned by the turn flow, used by both paths.
- Effort: S. Risk: low.

##### [Low] Two sources for "squads of a faction"
- Evidence: `handler.go:79` uses `Queries.GetEnemySquads`; `handler.go:188` uses `CombatService.GetAliveSquadsInFaction`; `handler.go:183` reads `Queries.CombatCache` while `handler.go:121,173` read `CombatService.CombatCache`.
- Direction: Pick one access path per kind of data inside the handler (the service for combat state).
- Effort: S. Risk: low.

#### Dead code
- `RedoLastMove` and `CanRedoMove` (`handler.go:150-166`): no callers outside the pair; no redo binding exists in combat.

#### Leave alone
- `MoveSquad` going through `squadcommands.MoveSquadCommand` and the command executor for undo (`handler.go:119-138`) is the right use of the command pattern, and `isSelectedSquadActed` blocking undo after an attack is a sensible rule to keep here.

---

### gui/guicombat/combatinput (557 lines, tests: no)

**Role:** Translates semantic input actions into calls on the action handler, the spell/artifact/inspect controllers and the visualizer. **Verdict:** Needs attention; one long priority chain, plus some rule checks that the sim already performs.

#### Design findings

##### [Medium] `HandleInput` is a 165-line priority ladder whose order is the behaviour
- Evidence: `handler.go:75-241`. Sub-mode handling (debug kill, spell, artifact, inspect, move) is inlined, each with its own cancel and click rules; a `defer` resets a flag mid-function (`:151`); `ActionEndTurn` is detected only to return false (`:164-166`).
- Problem: The implicit rule "which sub-mode owns the click" is spread through the function, and inspect mode has to cancel spell and artifact mode by hand (`:188-195`), although the earlier branches make those lines unreachable while either mode is active.
- Direction: Split into a small dispatcher plus one function per sub-mode (`handleSpellModeInput`, `handleArtifactModeInput`, `handleInspectModeInput`, `handleDefaultInput`), in the same file. No new types.
- Effort: S. Risk: low.

##### [Low] Movement validity is checked in the GUI and again in the sim
- Evidence: `handler.go:250-264` recomputes `GetValidMovementTiles` and searches it; `MoveSquad` then validates again through `CombatMovementSystem.MoveSquad` -> `CanMoveTo` (`tactical/combat/combatcore/combatmovementsystem.go:58-84`). The returned error is discarded (`handler.go:271`).
- Direction: Call `MoveSquad` and surface its error; drop the pre-check.
- Effort: S. Risk: low.

##### [Low] "Squad under the cursor" is written three times
- Evidence: `handler.go:279-280`, `debug.go:22-23`, `inspect.go:27-28` (and three more in `guispells`/`guiartifacts`).
- Direction: One `squadAtCursor(mouseX, mouseY)` method.
- Effort: S. Risk: low.

#### Mechanics notes
- Debug "kill all" sets unit health to zero directly on components (`debug.go:62-82`) while debug "kill squad" disposes the squad through `combatdisposal` (`combatbase/handler.go:211`). Two different meanings of "dead" for the same debug intent; the first leaves squads on the map until something else notices.

#### Dead code
- `handler.go:188-195`: the spell/artifact cancellation inside the inspect hotkey branch cannot run while those modes are active, because `:81-128` return first.
- `println` debug output in `debug.go:35-59` and `handler.go:235`.

#### Leave alone
- The use of semantic `InputAction`s throughout (no raw key checks) is consistent and should stay.
- `viz.go` (`updateLayerStatusWidget`) is a reasonable place for the overlay status text.

---

### gui/guicombat/combatanimation (564 lines, tests: no)

**Role:** A separate full-screen UI mode that shows attacker and defender grids with colour flashes for one attack. **Verdict:** Needs attention; the package boundary is good, but it re-derives what happened instead of being told.

#### Design findings

##### [High] The animation recomputes attackers and targets after the attack has already been resolved
- Evidence: `colors.go:46-62` constructs a fresh `combatcore.NewCombatActionSystem` and calls `GetAttackingUnits`, then `combatmath.SelectTargetUnits` and `SelectHealTargets` (`colors.go:77,105`). `ExecuteAttackAction` has already run (`combatbase/handler.go:95`; for AI, every queued attack of the turn has run before the first animation, `combat_turn_flow.go:129-133`). `GetAttackingUnits` filters on `CurrentHealth > 0` and `CanAct` (`tactical/combat/combatcore/combatactionsystem.go:258-272`), and row/column targeting pierces past empty lines (`tactical/combat/combatmath/combattargeting.go:43-71`). The real record of the attack exists: `CombatResult.Log` carries `AttackingUnits`, `AttackEvents` and `HealEvents` (`tactical/combat/combattypes/combattypes.go:108-124,195-199`), and is dropped by both callers.
- Problem: Two sources of truth for "who hit whom". The displayed one is computed from post-attack state, so units that died, units whose `CanAct` changed, and targets reached only after the front line fell can be shown wrongly. It also puts targeting rules (heal versus attack, target selection) into presentation code, and forces the GUI to build a second action system.
- Direction: Pass the `CombatLog` (or a small view of it) to `SetCombatants` and build the colour maps from its events. Add the log to `QueuedAttack` for the AI path. Remove the `combatcore`/`combatmath` imports from this package.
- Effort: M. Risk: low.

##### [Medium] Whether this should be a separate `UIMode` at all
- Evidence: `combatmode.go:90,130` and `gui/guiraid/raidmode.go:103` special-case the mode name; both traced lifecycle bugs in `guicombat` come from the combat/animation mode boundary; the animation mode holds no state that outlives one attack.
- Problem: Making a transient presentation a peer mode turns every attack into an `Exit`/`Enter` pair on the combat mode, which then has to tell "really leaving" from "showing an animation" by string comparison.
- Direction: Not urgent once combat exit is explicit (see `guicombat` High #1). If the boundary keeps causing trouble, run the animation as a sub-state of `CombatMode` (the package and its renderer stay; only the `UIMode` registration goes).
- Effort: M. Risk: medium.

#### Probable bugs
- **Highlighted units can differ from the units that actually attacked or were hit.** Confidence: medium (follows from the evidence above; the visible effect depends on whether `CanAct` and deaths change during the attack).

#### Dead code
- `drawSquadNames` (`render.go:41-44`) fetches two names and discards them; it draws nothing.
- `colors.go:72-75` fetches `targetRowData` only to nil-check it; `SelectTargetUnits` does the same check itself.
- Replay reset is written twice: `mode.go:104-112` (`ResetForNextAttack`) and inline at `mode.go:267-273`.
- `healColorScale` is `(1,1,1)`, i.e. no tint, so heal targets are visually indistinguishable from idle units (`colors.go:34`); probably unintended rather than dead.

#### Leave alone
- The phase state machine (`mode.go:183-232`) and the cached grid image (`render.go:13-39`) are simple and adequate.
- The narrow entry API (`SetCombatants`, `SetOnComplete`, `SetAutoPlay`) is a good shape; only its input should change.

---

### gui/guicombat/combatvisualization (583 lines, tests: no)

**Role:** Owns the combat overlays (selection highlight, move tiles, health bars) and the threat/layer heat-map debug view. **Verdict:** Needs attention; mostly sound, but it keeps a parallel copy of sim-side caches and writes presentation state into the world map.

#### Design findings

##### [Medium] A third (and fourth) copy of the per-faction threat-evaluator cache, updated from the GUI
- Evidence: `manager.go:26,60-66,130-140,197-203` (`layerEvaluators`), `threat.go:88` (visualizer's reference to the same map); the sim side already caches them in `combatservices.AICoordinator.layerEvaluators` (`tactical/combat/combatservices/ai_coordinator.go:14,68-80`) and again in `mind/ai/ai_controller.go:24,230-244`. `UpdateThreatManagers` + `UpdateThreatEvaluator` (`manager.go:151-164`, called from `gui/guicombat/combat_mode_init.go:259-260`) duplicate `AICoordinator.UpdateThreatLayers` (`ai_coordinator.go:97-104`), which has no callers in the repo, and `AIController.updateThreatLayers` (`ai_controller.go:132-140`).
- Problem: The GUI refreshes AI data structures on every turn end, and must be told to reset its copy at teardown (`ResetThreatState`, with a comment in `combat_service.go:244-246` acknowledging the parallel cache). The most recent commit ("Resetting threat layers upon combat teardown") is maintenance on exactly this duplication.
- Direction: Drop `layerEvaluators` from the manager and the visualizer; ask `combatService.GetThreatEvaluator(factionID)` when drawing. Replace the two GUI update methods with one call to `combatService.UpdateThreatLayers()`, made only when the overlay is active. `ResetThreatState` then reduces to clearing the faction list.
- Effort: S. Risk: low.

##### [Medium] Overlays are stored as per-tile colour matrices on the world map, with three independent writers
- Evidence: `threat.go:178-183,231-232` write and clear via `GameMap.ApplyColorMatrixToIndex`; `gui/guispells/spell_handler.go:183,199,258` and `input/cameracontroller.go:136` write the same slot.
- Problem: The map tile has one colour-matrix slot. `ClearVisualization` wipes every tile, including an active spell AoE preview; the spell handler's clear punches holes in the heat map. Presentation state is stored in the world model, and each writer assumes it is the only one.
- Direction: Minimum: have each writer clear only the indices it wrote (the spell handler already tracks `prevIndices`; the visualizer should do the same). Better, if overlays grow: a separate overlay layer in `visual/` that the tile renderer composes, so `GameMap` carries no UI state.
- Effort: S for the minimum, M for the layer. Risk: low.

##### [Low] `CombatVisualizationManager` construction and `RefreshFactions` are the same code twice; redundant parameters
- Evidence: `manager.go:52-70` versus `manager.go:120-148`; `RenderAll` takes a `combatService` although the manager stores one (`manager.go:32,81`); `GetValidMoveTiles` takes `inMoveMode` and re-checks what its only caller already checked (`manager.go:85-86,98-105`); `ecsManager` duplicates `queries.ECSManager`.
- Direction: Constructor calls `RefreshFactions`; drop the redundant parameters and field.
- Effort: S. Risk: low.

##### [Low] Layer metadata is split between a map and two switches
- Evidence: `threat.go:44-80` (name, description, colour key), `threat.go:307-324` (value selector), `threat.go:348-367` (gradient). The "Retreat Quality" colour key says "Red-Green" but the gradient is plain green, the same as "Support Value" (`threat.go:75-79,355,362-363`).
- Direction: One table row per layer holding name, colour key, value function and gradient function.
- Effort: S. Risk: low.

#### Dead code
- `ThreatVisualizer.GetLayerMode` (`threat.go:289-291`): no callers.
- `ThreatVisualizer.currentFactionID` (`threat.go:98,199`): written, never read; the `currentFactionID` parameter of `Update` and `UpdateThreatVisualization` is therefore unused too.
- `LayerModeInfo.Description` (`threat.go:39`): populated for all seven layers, never displayed.

#### Performance
- `RenderAll` calls `MovementSystem.GetValidMovementTiles` every frame while move mode is on (`manager.go:85-90`); that is a `(2r+1)^2` sweep of `CanMoveTo` plus a zone-of-control check and a fresh slice per frame (`tactical/combat/combatcore/combatmovementsystem.go:112-135`). The overworld caches its equivalent on entering move mode. Compute on entering move mode and after each move; invalidate on turn end.
- `ThreatVisualizer.Update` calls `GetSquadFaction` for every squad for every tile in the viewport (`threat.go:245-260`) although all `relevantSquads` belong to `viewFactionID` by construction (`threat.go:211`). It only runs when dirty, so the cost is bounded; pass the faction in.

#### Leave alone
- Delegating the actual drawing to `visual/combatrender` renderers keeps this package a thin coordinator; that split is right.
- The threat view consuming `combatservices.ThreatProvider`/`ThreatLayerEvaluator` interfaces instead of importing `mind/behavior` is a good boundary.
- `iterateViewport` and the dirty/round gating in `Update` are fine.

---

### gui/guioverworld (1,739 lines, tests: no)

**Role:** The strategic-map mode: commander selection and movement, engaging threats, garrison dialog, recruiting, turn end, and drawing nodes and commanders. **Verdict:** Needs attention; it is structured sensibly (mode / action handler / input handler / renderer), but it holds a full game rule and its panel code is repetitive.

#### Design findings

##### [High] `RecruitCommander` is a complete game rule implemented in the GUI
- Evidence: `overworld_action_handler.go:273-370`: roster capacity, friendly settlement/fortress check, gold cost, sprite loading from disk, `commander.CreateCommander`, `commander.SeedStarters`, roster add, refund on failure. `setup/gamesetup/initial_commanders.go:26-56` repeats image load + create + seed + roster add; the sprite load appears a third and fourth time in `setup/gamesetup/savehelpers.go:55,68` and `setup/savesystem/chunks/commander_chunk.go:116`.
- Problem: The rule cannot be exercised or reused without the GUI, and the two creation paths can drift (the GUI path calls `roster.AddCommander`, the setup path calls `commander.AddCommanderToPlayerRoster`). If `AddCommander` fails after creation the gold is refunded but the new commander entity is left alive (`:346-362`).
- Direction: A single `commander.RecruitCommander(manager, playerID, pos) (ecs.EntityID, error)` (validation + cost + creation + roster) in `tactical/commander`, with one shared "create commander with starters and sprite" helper used by both recruitment and initial setup. The GUI keeps selection changes and log lines.
- Effort: M. Risk: low.

##### [Medium] The overworld has a private coordinate system built on a camera that never moves
- Evidence: `OverworldState.CameraX/CameraY` (`gui/framework/contextstate.go:110-111`) are never assigned anywhere in the repo. `(pos.X - CameraX) * tileSize` is written out seven times: `overworld_renderer.go:90-91,136-137,197-198,224-225,248-249`, `gui/guinodeplacement/nodeplacement_renderer.go:24-25`, and inverted in `ScreenToLogical` (`overworld_renderer.go:277-281`). The map itself is drawn without any camera (`maprender.DrawMap`, `overworld_renderer.go:74`).
- Problem: Dead state that looks live, and a second screen/tile conversion beside `coords.CoordManager`. If anyone ever sets the camera, nodes and clicks shift but the map does not.
- Direction: Delete `CameraX/CameraY`. Add one `logicalToScreen(pos)` beside `ScreenToLogical` on the renderer (or use `CoordManager` if the full-map view is covered there) and use it at all seven sites.
- Effort: S. Risk: low.

##### [Medium] Recording session lifecycle is tied to mode Enter/Exit
- Evidence: `overworldmode.go:160-161,177-178,318-348`.
- Problem: `Exit` finalizes, exports and clears the overworld recording on every mode switch, including opening the squad editor or node placement (`overworld_input_handler.go:52,95`), so one play session is exported as a series of "Session Paused" fragments. Whether the simulation log is exported is a campaign-session concern, not a property of which screen is showing. Same pattern as the battle recorder in `gui/guicombat/combatmode.go:104-109,150-162`.
- Direction: Start and finalize the recording where the campaign session starts and ends (game start/load, save/quit, game over), outside the mode.
- Effort: S. Risk: low.

##### [Medium] The input/action handler split is blurred
- Evidence: `overworld_action_handler.go:22` says the action handler "handles all game-state-changing logic", but `toggleMoveMode`, `cycleCommander`, `handleGarrison` and the 65-line dialog builder `showGarrisonDialog` live in the input handler (`overworld_input_handler.go:190-349`) and are invoked from button callbacks in the panel registry (`overworld_panels_registry.go:94,124`), while the pure UI toggle `ToggleInfluence` sits in the action handler (`overworld_action_handler.go:202-210`). "Return to tactical" is written twice (`overworld_input_handler.go:42-46`, `overworld_panels_registry.go:132-138`).
- Problem: Buttons and hotkeys for the same action take different routes, and there is no rule for where a new action goes.
- Direction: Make the rule explicit and cheap: the action handler owns every user intent (toggle move, cycle commander, open garrison dialog, return to tactical); the input handler only maps actions and clicks to those methods; the panel registry only calls action-handler methods.
- Effort: S. Risk: low.

##### [Low] `OverworldModeDeps` carries unused and redundant members
- Evidence: `overworld_deps.go:18` `EncounterService` is never read (and so `NewOverworldMode`'s `encounterService` parameter and field, `overworldmode.go:31,59,127`, are write-only). `OverworldRenderer` stores both `manager` and `context` and reaches the same ECS through each (`overworld_renderer.go:22-27`).
- Direction: Remove the field and constructor parameter; keep one of `manager`/`context` on the renderer.
- Effort: S. Risk: low.

##### [Low] The event log's only storage is the widget's text
- Evidence: `overworldmode.go:300-314` prepends to `eventLogText.GetText()` and truncates at 2,000 bytes.
- Problem: UI-originated messages exist nowhere else, the truncation can cut a multi-byte character, and the simulation's own event stream (`campaign/overworld/core.LogEvent`) is not shown here at all, so the panel titled "event log" shows only GUI messages.
- Direction: Keep a bounded `[]string` on the mode and render from it; consider feeding sim events into the same list. Not urgent.
- Effort: S. Risk: low.

#### Mechanics notes
- `EngageThreat` re-checks in the GUI that the commander stands on the threat's tile (`overworld_action_handler.go:107-138`) and builds the threat's display name for the combat starter. The same-tile rule belongs with the encounter trigger; verify whether `encounter.TriggerCombatFromThreat` enforces it, otherwise the rule exists only in the GUI.
- Overworld move mode caches valid tiles in UI state (`OverworldState.ValidMoveTiles`, contrary to the "do not cache computed game data here" comment at `contextstate.go:106`), while combat move mode recomputes every frame. Same job, two approaches; the cached form is the better one, but it should be owned by the handler rather than the shared state struct.

#### Probable bugs
- **"Age: N ticks" shows the creation tick, not the age.** Confidence: high. `overworld_formatters.go:48,56` prints `data.CreatedTick`.
- **Garrison dialog acts on the wrong squad when two entries have identical labels.** Confidence: medium. `overworld_input_handler.go:326-341` maps the selected string back to an entry by first label match; two squads with the same name, unit count and status produce the same label.
- **`FormatThreatInfo` dereferences a nil position.** Confidence: low (requires a node without a position component). `overworld_formatters.go:23,52,81`.

#### Dead code
- `OverworldMode.initialized` (`overworldmode.go:56,146`): written, never read.
- `OverworldModeDeps.EncounterService` and the matching mode field/parameter (above).
- `result.Custom["tickControls"]` (`overworld_panels_registry.go:147`): never read.
- Stale file references in comments (`overworld_action_handler.go:23`, `overworld_input_handler.go:18`, `overworld_deps.go:12`).

#### Leave alone
- The four-way file split (mode, action handler, input handler, renderer) and the `OverworldModeDeps` struct are the right size for this mode.
- `Update`'s tick and selection dirty checks (`overworldmode.go:186-203`).
- `EndTurn` delegating to `tick.EndOverworldTurn` and combat starting through the injected `StartCombat` closure are good examples of the GUI calling a sim entry point rather than doing the work.
- Rendering iterates ECS views each frame without caching; at overworld entity counts that is fine.

---

### gui/guiraid (1,514 lines, tests: no)

**Role:** The garrison-raid mode: floor map of room cards, a deploy panel, and a post-encounter summary. **Verdict:** Needs attention; the rendering side is solid, but raid progression decisions are made in the mode and have drifted from `RaidRunner`.

#### Design findings

##### [High] Raid progression rules live in `RaidMode` and duplicate or contradict `RaidRunner`
- Evidence:
  - Resume: `raidmode.go:112-118` flips `raidState.Status` from `RaidRetreated` to `RaidActive` directly on the component; `RaidRunner` has `Retreat()` but no resume (`campaign/raid/raidrunner.go:322-330`).
  - Start: `raidmode.go:135-170` picks the first commander, truncates the squad list to `raid.MaxPlayerSquads()`, starts the raid and enters floor 1.
  - Room kind: `raidmode.go:272-281` switches on rest room/stairs; `raidrunner.go:128-135` has the same switch. For combat rooms the GUI never calls `SelectRoom`, so its accessibility/cleared validation (`raidrunner.go:120-126`) is bypassed and re-implemented in the click handler (`floormap_panel.go:119`).
  - Deployment fallback: `raidmode.go:295-303` auto-deploys when nothing is deployed; `TriggerRaidEncounter` has its own fallback to all squads (`raidrunner.go:164-169`).
  - Floor advancement: `raidmode.go:342-361` decides when to call `AdvanceFloor`, from two call sites (`:279,338`).
- Problem: The GUI is the orchestrator of the raid, so the sequence cannot run headless, and the two halves disagree (see the final-floor bug below). The mode also mutates ECS game state directly.
- Direction: Give `RaidRunner` the missing verbs and let the mode call them and redraw: `Resume()`, `StartDefaultRaid(playerID)` (or move the debug auto-start out of `Enter`), `SelectRoom` returning whether deployment is needed, and floor advancement performed inside the runner when a room clear completes a floor. `checkFloorAdvancement` and the room-type switch then leave the GUI.
- Effort: M. Risk: medium (no tests in `guiraid`; `campaign/raid` has some).

##### [Medium] `RaidUIState` holds redundant and unused state, and selection is stored twice
- Evidence: `raidstate.go:15-21`. `ShowingSummary` duplicates `CurrentPanel == PanelSummary`; `SummaryData`, `ShowingSummary` and `HoveredRoomID` are written (`raidmode.go:317-318,326-327,366-367`, `floormap_panel.go:114`) and never read. The selected and hovered room are also kept in the renderer (`floormap_renderer.go:62-63`), synced only on `Refresh` (`floormap_panel.go:102`).
- Direction: Reduce `RaidUIState` to `CurrentPanel` and `SelectedRoomID` (or inline them into `RaidMode`); let the renderer own hover.
- Effort: S. Risk: low.

##### [Medium] Buttons are created with empty handlers and re-wired later
- Evidence: `raid_panels_registry.go:67-70,120-136,184-187` create five buttons with `OnClick: func() {}`; `floormap_panel.go:44-58`, `deploy_panel.go:46-70`, `summary_panel.go:40-48` then add the real handler with `Configure(ClickedHandler(...))`.
- Problem: Two-phase wiring that no other mode uses (combat and overworld close over the mode in the registry callback, which runs lazily at click time). It needs five extra custom-widget keys and three `wireButtons` methods.
- Direction: Put the real callbacks in the registry closures (`bm.OnDeployConfirmed`, `bm.deployPanel.autoDeploy`, ...). The three panel registrations are also the same container built three times (`raid_panels_registry.go:27-43,87-104,159-176`); one local `centeredPanel(widthFrac, heightFrac, hidden)` helper removes that.
- Effort: S. Risk: low.

##### [Low] Room presentation data is hard-coded and keyed by raw strings
- Evidence: `floormap_renderer.go:41-50` keys display names, icons and colours by `"guard_post"`, `"barracks"`, ... while the canonical constants are in `world/garrisongen/roomtypes` (`world/garrisongen/dag.go:12-19`).
- Problem: A renamed or added room type silently falls back to the grey "?" card.
- Direction: Key the map by the `garrisongen` constants. Moving the visuals to JSON is optional and not needed now.
- Effort: S. Risk: low.

##### [Low] The floor-map renderer recomputes DAG depth that generation already computed
- Evidence: `floormap_renderer.go:455-510` (Kahn's algorithm, longest path) versus `garrisongen.DagDepth` (`world/garrisongen/dag.go:347-360`, used at `generator.go:60`).
- Direction: Store depth on `raid.RoomData` when the floor graph is built and read it here; `computeDepths` and `roomByNode` go away.
- Effort: S. Risk: low (touches the raid save chunk if the field is persisted).

#### Mechanics notes
- The deploy panel is described as "pre-encounter squad selection" (`raid_panels_registry.go:80`) but offers only Auto Deploy, Start and Back; there is no way to choose squads. Not a request for a feature, but the panel, `raid.SetDeployment` and the two fallbacks above are three mechanisms around a choice the player cannot make. If manual deployment is not planned soon, the panel can be collapsed into a confirm step.
- "Floor complete" has two definitions in `campaign/raid`: the stored flag `FloorStateData.IsComplete` (read by the GUI, `raidmode.go:348`) and the computed `IsFloorComplete()` (used for victory, `campaign/raid/resolvers.go:93`). See `campaign/raid`.

#### Probable bugs
- **Taking the stairs on the final floor never finishes the raid.** Confidence: medium-high by trace. Every floor's critical path ends in a non-combat stairs room (`world/garrisongen/dag.go:132-137`). Clearing it goes through `OnRoomSelected` -> `SelectRoom` -> `processStairsRoom` (`campaign/raid/raidrunner.go:202-210`), then `checkFloorAdvancement`. On the last floor that takes the `else` branch and only sets a status string (`raidmode.go:358-360`). The victory transition lives in `AdvanceFloor` (`raidrunner.go:306-311`) and in `CheckRaidEndConditions`, which only runs after a combat encounter (`raidrunner.go:274-298`). The raid stays active with its entities until the player retreats or fights another room.
- **The summary always says "Room Cleared", including after a defeat.** Confidence: high. `summary_panel.go:58` is unconditional; `RaidEncounterResult.IsVictory` (`campaign/raid/raidrunner.go:22,268`) is never read anywhere.
- See also the `guicombat` bug that returns raid battles to exploration; `RaidMode.Enter`'s summary path (`raidmode.go:103-109`) depends on arriving from `"combat"`.

#### Dead code
- `RaidUIState.ShowingSummary`, `SummaryData`, `HoveredRoomID` (write-only, above).
- `FloorMapRenderer.rooms`, `areaX/areaY/areaW/areaH` (`floormap_renderer.go:55,59,87-91`) and `CardLayout.Depth` (`:31,188`): written, never read.
- `RaidMode.Render`, `DeployPanel.Render`, `SummaryPanel.Render` (`raidmode.go:199-200`, `deploy_panel.go:155-157`, `summary_panel.go:89-91`) are empty; `BaseMode` already supplies a no-op `Render`.
- `fmt.Println`/`Printf` progress output (`raidmode.go:116,168`).

#### Performance
- `FloorMapPanel.HandleInput` runs `raid.GetAccessibleRooms` (an ECS query) every frame to service number-key shortcuts (`floormap_panel.go:126-127`), even when no key is pressed. Check the nine actions first, or reuse the cards already laid out.
- `renderEdges` builds a `cardByNode` map every frame (`floormap_renderer.go:209-212`); build it once in `ComputeLayout`.

#### Leave alone
- `FloorMapRenderer`'s layout and card drawing (`floormap_renderer.go:86-344`) are self-contained and readable; do not abstract them.
- The three-panel controller structure (`FloorMapPanel`, `DeployPanel`, `SummaryPanel`) with one mode switching between them, and per-panel action maps via `GetActionMap` (`raidmode.go:54-65`), is a good fit.
- Using `OverlayRenderer` to draw cards over the ebitenui panel is the intended framework hook.

---

### gui/guinodeplacement (540 lines, tests: no)

**Role:** A sub-mode of the overworld for placing player nodes with a hover preview. **Verdict:** Healthy, with small cleanups; it should become part of `guioverworld`.

#### Design findings

##### [Medium] It is an overworld sub-mode living in its own package
- Evidence: imports `guioverworld` for `OverworldRenderer` (`nodeplacementmode.go:12,24,82`), shares `framework.OverworldState` (`:23,55`) and its camera fields (`nodeplacement_renderer.go:24-25`), returns only to `"overworld"`, and is registered only in the overworld mode manager (`setup/gamesetup/moderegistry.go:50`).
- Problem: A package boundary with one consumer in each direction; it keeps `OverworldRenderer` and its hit-test helpers exported and makes the camera/coordinate duplication cross-package.
- Direction: Move its three files into `gui/guioverworld`. No code changes beyond the package clause and dropping the import.
- Effort: S. Risk: low.

##### [Low] Placement is validated in three places with two different functions
- Evidence: every frame in `HandleInput` (`nodeplacementmode.go:155-158`, `ValidatePlayerPlacementWithCost`), again on click (`:176-181`), and a fallback in the renderer using `ValidatePlayerPlacement` without the cost check (`nodeplacement_renderer.go:42-46`). `node.CreatePlayerNode` itself does not validate placement, only cost (`campaign/overworld/node/system.go:77-107`).
- Problem: The renderer fallback is unreachable (the validation result is always set when the cursor position is set, `nodeplacementmode.go:153,158`) and would disagree with the real rule if it ran. More importantly, the placement rule is enforced only because the GUI remembers to call the validator first.
- Direction: Delete the renderer fallback. Make `CreatePlayerNode` validate internally so the GUI's hover check is purely for feedback. Validate on hover only when the hovered tile or selected type changes.
- Effort: S. Risk: low.

##### [Low] Info panel doubles as transient feedback
- Evidence: `setInfo` overwrites the whole placement-info text with a one-line message (`nodeplacementmode.go:296-301`); on success `refreshPlacementInfo` immediately overwrites it again (`:196-198`), so "Placed X at (..)" is never visible; failures replace the whole info panel until the next selection change.
- Direction: Use `BaseMode.SetStatus` (a status label) for feedback and keep the info panel for info.
- Effort: S. Risk: low.

#### Dead code
- `nodeplacement_renderer.go:42-46`: unreachable re-validation branch.
- `result.Custom["controls"]` (`nodeplacement_panels_registry.go:85`): never read. The comment "Place + Cancel buttons" (`:59`) describes a Place button that does not exist.
- `Update` (`nodeplacementmode.go:108-110`) is an empty override of `BaseMode.Update`.

#### Leave alone
- The mode is small and linear; `handlePlaceNode` delegating creation and cost to `node.CreatePlayerNode` is the right split.
- The custom HUD bar and ghost preview drawing are fine as they are.

---

### gui/guiexploration (333 lines, tests: no)

**Role:** The default tactical-context mode: a bottom action bar (overworld, squad, debug, save/load) and a debug menu for regenerating the map. **Verdict:** Needs attention; it is mostly a shell, and its two real behaviours (map regeneration and game-mode detection) are in the wrong place.

#### Design findings

##### [Medium] World regeneration is implemented inside a panel-registry file
- Evidence: `exploration_panels_registry.go:49-79`: builds a `GenContext`, replaces `*GameMap`, rebuilds the overworld walkable grid, moves the player, updates the position system, marks tiles dirty. Boot does the first two steps separately in `setup/gamesetup/bootstrap.go:96` and `setup/gamesetup/helpers.go:18`.
- Problem: A world-state operation written against five packages lives in GUI wiring. It takes map size from two sources in one function (`coords.ScreenInfo.DungeonWidth/Height` for generation at `:52-53`, `templates.GameConfig.Display.MapWidth/Height` for the walkable grid at `:61-62`), moves the player by writing `PlayerData.Pos` and calling `GlobalPositionSystem.MoveEntity` by hand instead of `manager.MoveEntity` (`:68-73`), and leaves every other positioned entity (commanders, nodes, squads) where it was on the old map.
- Direction: One `gamesetup.RegenerateMap(generatorName)` (or equivalent beside the existing bootstrap code) that boot and the debug menu both call; the GUI button passes the generator name only. If this is purely a developer tool, mark it as such and keep it behind the debug flag.
- Effort: S-M. Risk: low.

##### [Medium] Game mode is inferred by probing which modes are registered
- Evidence: `exploration_panels_registry.go:142-200`: `_, hasSquadInTactical := em.ModeManager.GetMode("squad_editor")` decides whether the Overworld, Debug, Save and Load buttons exist; `setup/gamesetup/moderegistry.go:65-67` documents that registration order is arranged so this probe works.
- Problem: An implicit flag encoded in registration order across two packages; reordering the slice in `RegisterRoguelikeTacticalModes` changes which buttons appear.
- Direction: Put an explicit game-mode value (roguelike/overworld) on `UIContext` and branch on that.
- Effort: S. Risk: low.

##### [Low] The mode shell carries placeholders
- Evidence: `explorationmode.go:16,76` (`initialized`, write-only); `:23,82` (`messageLog` is fetched and never written, so the message-log panel is always empty); `:85-94` (`Enter`/`Exit` only print); `:96-106` (`Update`/`Render` are empty overrides with aspirational comments).
- Direction: Remove the field, the empty overrides and the prints. Either remove the message-log panel or give it a real source.
- Effort: S. Risk: low.

#### Dead code
- `ExplorationMode.initialized`, `messageLog`, `GetExplorationMessageLog`, and the `ExplorationPanelMessageLog` panel (built, never written to).
- Empty `Update`, `Render`, and print-only `Enter`/`Exit`.
- Hard-coded generator names in the debug menu (`:89-104`) duplicate the worldgen registry keys; a typo fails silently at click time.

#### Leave alone
- Save/load going through callbacks on `UIContext` (`:181-199`) keeps `setup/savesystem` out of the GUI; keep that.
- The mode is correctly minimal otherwise; do not grow handlers or a deps struct for it.

---

## 20. Visual layer and input

Scope: `visual/graphics`, `visual/rendering`, `visual/maprender`, `visual/combatrender`, `visual/vfx`, `input`. 2,145 non-test lines, no test files in any of the six packages. Every file was read in full; callers in `game_main`, `gui/guispells`, `gui/guicombat`, `gui/framework`, `gui/guiexploration`, `world/worldmapcore` and `core/coords` were read to trace usage.

### Layer overview

**Dependency picture (verified by grep of imports):**

| Package | Imports (project) | Imported by |
|---|---|---|
| `visual/graphics` | `core/common`, `core/coords` | `world/worldmapcore`, `gui/guispells`, `gui/guicombat/combatvisualization`, `input` |
| `visual/rendering` | `core/common`, `core/coords`, `world/worldmapcore` | `game_main`, `visual/maprender`, `visual/combatrender` |
| `visual/maprender` | `visual/rendering`, `core/coords`, `world/worldmapcore` | `game_main`, `gui/guioverworld` |
| `visual/combatrender` | `visual/rendering`, `core/coords` | `gui/framework`, `gui/guicombat/*`, `gui/guisquads` |
| `visual/vfx` | `core/common`, `core/config`, `core/coords` | `game_main`, `gui/guispells` |
| `input` | `core/common`, `core/coords`, `gui/framework`, `visual/graphics`, `world/worldmapcore` | `game_main` |

**Is five visual packages the right granularity? No; three is.**

1. `visual/graphics` is not a graphics package. Neither of its two files imports ebiten or draws anything. It holds (a) `BaseShape`, which is the spell area-of-effect rule (which tiles a spell covers), and (b) `ColorMatrix`, a four-float tint stored on every map tile. The first is game rules; the second is a value type owned in practice by `worldmapcore.Tile`. Because of (b), the world model imports a `visual/` package (`world/worldmapcore/dungeontile.go:28`), which inverts the intended layering. Recommendation: dissolve the package (details under its section).
2. `visual/maprender` and the sprite half of `visual/rendering` are one job split in two. `game_main/main.go:128-134` always calls them as a pair with the same arguments (`DrawMapCentered` + `ProcessRenderablesInSquare`, or `DrawMap` + `ProcessRenderables`), and each re-implements the same "viewport vs full map" branch and destination-rectangle maths (`rendering.go:52-76`, `tilerenderer.go:111-130`). `rendering` already imports `worldmapcore`, so merging adds no new dependency. Recommendation: fold `maprender` into `rendering` (about 670 lines, 6 files).
3. `visual/combatrender` and `visual/vfx` are coherent and independent; keep both as packages.
4. `input` is one 137-line file that is not an input system (the real one, `InputState` and `ActionMap`, lives in `gui/framework`). It should be dissolved into the exploration mode (details under its section).

Net: `visual/rendering`, `visual/combatrender`, `visual/vfx`. Two packages and one top-level directory disappear; no new files are needed.

**Does rendering reach into game rules or state? Mostly no, with three real exceptions.**

- Good: `combatrender` never imports `tactical/*`; it sees squads only through two small provider interfaces. `rendering` reads only `Renderable` and position components. `vfx` knows nothing about spells.
- Exception 1: the AoE targeting rule lives in `visual/graphics` and is evaluated by GUI code (`gui/guispells/spell_handler.go:206-239`), so the simulation has no way to compute or validate an AoE on its own.
- Exception 2: presentation code draws from the shared simulation RNG. `vfx/animators.go:31-57` calls `common.RandomFloat()` up to six times per effect per drawn frame, and the shape constructors call `common.RandomInt` (`drawableshapes.go:181-231`). `core/common/randnumgen.go:8` is a single seeded PCG that the "Using shared RNG" commit made the reproducibility source for the whole game. Any spell effect on screen now advances that stream by a frame-rate-dependent amount, so a seeded run is not reproducible once a visual effect plays.
- Exception 3: transient UI overlays (threat heat map, spell AoE preview, a debug highlight) are stored inside the world model's tiles as `Tile.cm`, one slot per tile shared by three unrelated writers.

**Per-frame work that reflects a design flaw** is listed under each package's Performance heading; the two that matter are `ViewportRenderer.DrawTileOverlay` (a render-target fill per tile per frame) and the pull-style squad providers in `combatrender` (every squad queried and copied twice per frame).

**Scale factor has three sources.** Code in this slice reads `coords.ScreenInfo.ScaleFactor` directly in five places (`rendering/rendering.go:60`, `maprender/tilerenderer.go:127` and `:161`, `vfx/renderers.go:18` and `:49`) while `rendering/viewport.go:68` asks `CoordManager.GetScaledTileSize()`, which uses the manager's private `scaleFactor` gated on `MAP_SCROLLING_ENABLED` (`core/coords/cordmanager.go:293-298`). `setup/gamesetup/bootstrap.go:82-85` mutates only `ScreenInfo.ScaleFactor`. The values agree today (3) only by coincidence of defaults. Everything in `visual/` should ask `CoordManager` and never read `ScreenInfo` for scale; the underlying duplication belongs to the `core/coords` review.

---

### visual/graphics (426 lines, tests: no)

**Role:** Tile-area shapes for spell targeting plus the tile tint value type. **Verdict:** Needs significant rework: the package is misnamed, sits in the wrong layer, and `BaseShape` carries several layers of leftover design with real edge bugs.

#### Design findings

##### [High] The AoE targeting rule lives in a `visual/` package and is only ever evaluated by the GUI
- Evidence: `visual/graphics/drawableshapes.go:137-163` (`BaseShape`, `GetIndices`); sole consumer `gui/guispells/spell_handler.go:57`, `:193`, `:218`, `:339`. Target squads are gathered from shape tiles inside the click handler (`spell_handler.go:217-227`) and then passed to `spells.ExecuteSpellCast` as a plain ID list (`:299`). A grep for `shape` across `tactical/`, `mind/` and `campaign/` finds no other use of spell shapes.
- Problem: "which tiles does this spell cover" and "which squads are in that area" are combat rules, but the simulation cannot compute them. Anything that is not the mouse handler (AI casting, a simulator in `tools/`, a replay) has no way to resolve an AoE, and the cast function cannot check that the targets it is handed are actually inside the area.
- Direction: move the geometry out of `visual/` into the spell system (for example a `targetarea.go` beside the spell code, or `core/coords` if it should stay pure geometry) as a function from (area definition, centre `LogicalPosition`, direction) to a bounds-clipped `[]LogicalPosition`. Move "squads inside area" next to it. The GUI then only calls it for the preview and the confirm.
- Effort: M. Risk: medium (no tests; behaviour at map edges changes for the better, see bugs).

##### [High] `BaseShape` is three abandoned designs layered on each other
- Evidence:
  - Three copies of the same six fields: `templates.JSONTargetArea` (`templates/schema_monster.go:90-97`) → `graphics.ShapeConfig` (`drawableshapes.go:76-83`) → `BaseShape` (`:137-145`), copied field by field in `spell_handler.go:269-276`.
  - Two type systems: five string names in data (`Circle`, `Square`, `Rectangle`, `Line`, `Cone`) collapse to three `BasicShapeType` values (`:129-135`).
  - Constructors roll random sizes from the shared RNG (`:181`, `:192`, `:205-206`, `:219`, `:231`) which `CreateShapeFromConfig` then overwrites (`:96-117`). Every shape in `spelldata.json` supplies its dimensions, so the random values are never the final ones, yet each construction still consumes RNG draws.
  - Position is stored in pixels (`:138`) and converted straight back to logical in `GetIndices` (`:152`). Both callers start with a logical position and convert it to pixels only to satisfy the API (`spell_handler.go:188-192`, `:213-215`, `:337-340`).
  - `ShapeSize`/`SizeCategory` has one value and is never read (`:12-16`, `:144`). `ShapeConfig.Radius` is never read (`:82`).
  - `UpdateSize` decides "is this a square" by testing `Width == Height` (`:248`), so a rectangle that happens to be square is silently treated differently.
  - Section banners still say "NEW SIMPLIFIED SHAPE SYSTEM", "preserved from original", "Replace ShapeUpdater pattern" (`:19`, `:126`, `:243`).
- Problem: roughly 340 lines, a mutable struct with cross-field invariants, pointer-typed direction, and a factory layer, to answer one pure question. The leftovers hide the real bugs below.
- Direction: when moving the logic (finding above), replace the struct and its factories with one pure function over the data-file definition plus a direction enum with `RotateLeft/RotateRight`. Drop `ShapeConfig`, `ShapeSize`, the random constructors, `UpdatePosition`, `UpdateSize`, `UpdateDimensions`, `GetDirection`, `Rotate`. The UI keeps only "current direction" as state.
- Effort: M. Risk: medium (no tests).

##### [Medium] `ColorMatrix` makes the world model depend on the visual layer, and the tint slot has three unrelated owners
- Evidence: `world/worldmapcore/dungeontile.go:28`, `:48`, `:53` and `dungeongen.go:139-157` import `graphics.ColorMatrix`. Writers of the single per-tile slot: threat heat map (`gui/guicombat/combatvisualization/threat.go:180`, `:238`), spell AoE preview (`gui/guispells/spell_handler.go:183`, `:199`, `:258`), debug highlight (`input/cameracontroller.go:136`).
- Problem: transient UI overlay state is stored in the map model, and there is one slot per tile with no notion of layers. Each writer clears by writing the empty matrix, which erases whatever another writer put there (see Probable bugs). The nine `Create*Gradient` helpers (`colormatrix.go:31-83`) are one-line struct literals whose doc comments describe the caller's intent ("Used for melee threat zone visualization"), which is the caller's knowledge leaking into a value type.
- Direction: smallest fix is to move the `ColorMatrix` type into `worldmapcore` (it is the only package that stores it) so `world` stops importing `visual`. Better fix, same effort class: keep tint overlays out of `Tile` entirely, as a per-tile tint layer owned by the map renderer and keyed by owner ("threat", "spell preview"), so clearing one overlay cannot wipe another and no dirty flag has to travel through `GameMap`. Replace the nine gradient functions with a small colour table in `combatvisualization`, where their meaning lives.
- Effort: S (type move) to M (overlay layer). Risk: low to medium.

#### Mechanics notes
- **Cone is a Line.** `NewCone` (`drawableshapes.go:230-240`) is identical to `NewLine` (`:218-228`); both are `Linear` and both go through `calculateLine`. The three `Cone` spells in `spelldata.json` therefore hit exactly the same tiles as a `Line` of the same length. Either implement a widening cone or delete the type from code and data; today the data advertises a mechanic that does not exist.
- **Even sizes round up to the next odd size.** `calculateRectangle` uses `Width/2` and `Height/2` and iterates `-half..+half` (`:288-291`). A Square of size 2 covers 3×3, the same as size 3; the Rectangle defined as 3×2 covers 3×3; the one defined as 5×4 covers 5×5. Data values in `spelldata.json`: squares of size 1, 2, 3; rectangles 3×2 and 5×4. Two of the five distinct rectangular sizes do not do what the data says.
- **Lines start on the clicked tile** (`:312-316`, `i` from 0), while circles and rectangles are centred on it. Likely intended, but worth stating in the data documentation.

#### Probable bugs
- **AoE shapes wrap around the map edge and can produce negative indices (confidence: high).** `calculateCircle/Rectangle/Line` call `CoordManager.LogicalToIndex` on positions with no bounds check (`:279`, `:292`, `:315`), and `LogicalToIndex` is plain `y*width + x` (`core/coords/cordmanager.go:118-120`). With the centre on column 0, `x = -1` yields the last tile of the previous row. `HandleAoEConfirmClick` converts those indices back to positions and collects squads there (`spell_handler.go:221-227`), so a shape touching the left or right edge can hit a squad on the opposite edge, one row away. With the centre on row 0, `y = -1` yields a negative index.
- **Consequent panic path (confidence: high, in `worldmapcore` but caused here).** `HandleAoETargetingFrame` stores the raw indices in `prevIndices` (`spell_handler.go:202`) and next frame passes them to `GameMap.ApplyColorMatrix`, which guards only `ind < len(tiles)` (`world/worldmapcore/dungeongen.go:143`). A negative index passes that guard and indexes the slice. Hovering an AoE preview so that the shape extends above row 0 should panic on the following frame. The single-index variant at `:153` does check `>= 0`.
- **Untyped direction constants (confidence: n/a, latent).** `LineUp = iota` (`:25`) has no type, so the direction constants are untyped ints rather than `ShapeDirection`; nothing misbehaves today.

#### Dead code
All confirmed by repo-wide grep (including `tools/` and tests):
- `RedColorMatrix` (`colormatrix.go:21`), `CreateBlueGradient` (`:39`), `CreateMagentaGradient` (`:51`): no references.
- `BaseShape.GetDirection` and `NoDirection` (`drawableshapes.go:169-174`, `:33`): no callers.
- `BaseShape.Rotate` (`:263-267`): no callers; the GUI rotates through the pointer directly (`spell_handler.go:244`, `:251`).
- `ShapeSize`, `SmallShape`, `BaseShape.SizeCategory`: written, never read.
- `ShapeConfig.Radius` (and therefore `JSONTargetArea.Radius`): never read.
- `DirectionToCoords`, `UpdateDimensions`, `UpdateSize`, `NewCircle/Square/Rectangle/Line/Cone`: exported but only used inside this file.

#### Leave alone
- `ColorMatrix` as a small value type with `IsTransparent` is fine; only its home and its helper zoo need attention.
- `RotateLeft/RotateRight` over an ordered direction list is a fine, simple approach.

---

### visual/rendering (395 lines, tests: no)

**Role:** Quad batching, world-sprite drawing, and viewport-relative tile overlay/border helpers. **Verdict:** Needs attention: the batching core is sound, but `ViewportRenderer`/`CachedViewport` is a vestigial double wrapper with an expensive overlay path.

#### Design findings

##### [Medium] `CachedViewport` and `ViewportRenderer` are two wrapper layers around `CoordManager` that no longer cache anything meaningful
- Evidence: `viewport.go:40-59` (comment: "Now a thin wrapper around CoordinateManager"; `NewViewportRenderer` ignores its `screen` parameter). `CachedViewport.GetRenderer` (`:148-162`) rebuilds the renderer when screen size changes, but the renderer does not depend on screen size, so the only effect of a rebuild is to throw away the cached border and overlay images. Three separate `CachedViewport` instances exist in combat (`combatrender/combatoverlays.go:15`, `:42`, `squadhighlights.go:31`), plus one in `gui/guisquads/squaddeploymentmode.go:41` via the highlight renderer, each with its own image caches.
- Problem: two types, a constructor and an invalidation protocol to hold one `LogicalPosition`. The centre is passed on every `Render` call anyway, so storing it is redundant state that must be kept in sync (`lastCenter`, `UpdateCenter`).
- Direction: collapse to a single small helper (or package-level functions) `DrawTileOverlay(screen, pos, center, color)` and `DrawTileBorder(screen, pos, center, color, thickness)` backed by one shared 1×1 white image scaled with `GeoM` and tinted with `ColorScale`. That removes `CachedViewport`, `BorderImageCache`, `overlayCache`, the four border images, and the per-renderer duplicates in `combatrender`.
- Effort: S. Risk: low.

##### [Medium] Two different mechanisms tint a tile, chosen by history rather than need
- Evidence: mechanism A writes a `ColorMatrix` into the tile so the map batch draws it tinted (threat map, spell preview; `maprender/tilerenderer.go:132-137`). Mechanism B draws a translucent quad on top via `ViewportRenderer.DrawTileOverlay` (movement range; `combatrender/combatoverlays.go:26-32`). Within mechanism B, borders tint a cached white image with `ColorScale` (`viewport.go:104-134`) while fills re-`Fill` an offscreen image per tile (`:89`).
- Problem: three ways to "colour a tile" with different costs, different layering behaviour, and different owners. A new overlay has no obvious right choice, and mechanism A is the one that pollutes the world model (see `visual/graphics`).
- Direction: standardise on one overlay path in this package (batched tinted quads per overlay layer). Movement range, threat and spell preview then become three layers of the same thing.
- Effort: M. Risk: low to medium.

##### [Low] Sprite draw order is map-iteration order
- Evidence: `renderingcache.go:53-57` iterates `map[*ebiten.Image]*QuadBatch`; Go randomises map order per iteration.
- Problem: when two visible sprites with different images share or overlap a tile (for example the player avatar and a squad or commander marker), which one is on top can change from frame to frame.
- Direction: keep batches in a slice in first-seen order (or sort by a layer value) and use the map only for lookup.
- Effort: S. Risk: low.

##### [Low] `QuadBatch` silently overflows at 16,383 quads
- Evidence: `quadbatch.go:31` casts `len(vertices)` to `uint16`.
- Problem: not reachable today (100×80 map = 8,000 tiles, at most 32,000 vertices in one batch), but a larger map sharing one tile image would wrap indices with no error.
- Direction: flush and continue, or guard, when the batch reaches the limit.
- Effort: S. Risk: low.

#### Probable bugs
- Sprite z-order flicker described above (confidence: low to medium; depends on two differently-imaged visible sprites overlapping, which I did not confirm occurs in play).

#### Dead code
- `QuadBatch.Count` (`quadbatch.go:77-79`): no callers anywhere.
- `NewViewportRenderer`'s `screen` parameter (`viewport.go:55`): unused.
- `CachedViewport.lastScreenX/lastScreenY` logic: serves no purpose once the renderer stopped depending on screen size.

#### Performance
- **`DrawTileOverlay` fills an offscreen image for every tile every frame** (`viewport.go:89-93`). Each `Fill` is a draw to a different render target followed by a draw to the screen, so N movement tiles cost 2N draw commands with a target switch between each, which defeats ebiten's batching. The border path in the same file already shows the cheap pattern (white image + `ColorScale`). Fixed by the collapse recommended above.
- `processRenderablesCore` walks every renderable including hidden unit entities (units are `Visible: false`, nine per squad) and does two component lookups before the visibility check (`rendering.go:35-42`). Acceptable at current scale; reorder the check if it ever shows in a profile.

#### Leave alone
- `QuadBatch` itself (`quadbatch.go`): small, correct, reused well by both sprites and tiles.
- `RenderingCache` using an ECS `View` for renderables: appropriate use of the library.
- The single `processRenderablesCore` with a nil-able viewport parameter is a reasonable way to share the two modes.

---

### visual/maprender (274 lines, tests: no)

**Role:** Batched drawing of map tiles, full-map or viewport-centred. **Verdict:** Needs attention: correct in the common path, but cache invalidation is spread over four signals and package globals, and part of its API is dead.

#### Design findings

##### [Medium] Cache invalidation relies on four independent signals, one of which can be consumed by the wrong renderer
- Evidence:
  1. Slice identity check `tilesChanged` (`maprendering.go:11-21`).
  2. Manual `ResetMapRenderers()` (`:31-35`), called from `game_main/setup.go:160`.
  3. Consume-once flag `gameMap.ConsumeTileColorsDirty()` (`:49`, `:69`; defined at `world/worldmapcore/gamemap_accessors.go:92-96`).
  4. Centre/viewport-size comparison (`tilerenderer.go:56-62`).
  
  State is held in three package-level variables (`maprendering.go:23-27`) shared by two independent renderers (`cachedFullRenderer`, `cachedViewportRenderer`) and by two callers that may pass different intent (`game_main/main.go:129`, `:132`; `gui/guioverworld/overworld_renderer.go:74`).
- Problem: signal 3 is read-and-clear, but there are two renderers. Whichever draws first after a tint change consumes the flag; the other keeps its stale batch, because the full-map renderer rebuilds only when it sees the flag (`tilerenderer.go:55`, `:63-68`). The overworld view and the scrolling tactical view both draw the same `GameMap` through different renderers, so this is reachable. More generally, a reader has to hold four mechanisms in their head to know when tiles redraw.
- Direction: replace the consume-once flag with a monotonically increasing version number on the map (bumped on tint change and on tile replacement). Each renderer remembers the last version it built from. That removes `tilesChanged`, the consume protocol, and the need for `ResetMapRenderers` to exist as a public call. If the overlay-layer change under `visual/graphics` is adopted, tints leave the map entirely and only the map-replaced signal remains.
- Effort: S to M. Risk: low to medium (no tests).

##### [Low] Tile bounds use `ScreenInfo` while indexing uses `CoordManager`
- Evidence: `tilerenderer.go:193-194`, `:200-201` read `coords.ScreenInfo.DungeonWidth/Height`; `:102` indexes with `CoordManager.LogicalToIndex`. `:152-154` recomputes a tile's logical position from `PixelX / TileSize` although `addTileToBatch` already has `x, y` (`:100-101`).
- Problem: CLAUDE.md's critical warning exists precisely because those two width sources can differ. This file mixes them.
- Direction: take width and height from `CoordManager.GetDungeonWidth/Height` (or from the map), and pass the known logical position down instead of re-deriving it.
- Effort: S. Risk: low.

#### Probable bugs
- **Stale tints after switching between the full-map and viewport renderers** (confidence: medium). Trace: tint changes while the viewport renderer is active → `DrawMapCentered` consumes the flag (`maprendering.go:69`) → later `DrawMap` runs with `TileColorsDirty: false` and `batchesBuilt == true` → no rebuild (`tilerenderer.go:55-68`). Reachable through the overworld view or the scroll toggle (`M`). I traced the code path but did not reproduce it in a running game.
- **`RenderedBounds.RightEdgeX/TopEdgeY` are wrong on every cached frame** (confidence: high, currently harmless). They are reset to 0 at `tilerenderer.go:51-52` and only filled during a rebuild (`:165-174`). `TopEdgeY` also starts at 0 and only decreases, so it can never report a positive top edge. Nobody reads the result (see Dead code), so delete rather than fix.

#### Dead code
- `RenderOptions.RevealAll` (`tilerenderer.go:33`) and the `revealAll` parameters of `DrawMap`/`DrawMapCentered` (`maprendering.go:38`, `:57`): threaded through from `config.DEBUG_MODE` but never read. No renderer reads `Tile.IsRevealed` either; it is only saved and loaded (`setup/savesystem/chunks/map_chunk.go:90`, `:183`). There is no fog of war in the renderer.
- `RenderedBounds` as a return value of `DrawMapCentered`: the only caller discards it (`game_main/main.go:129`). `RightEdgeX` and `TopEdgeY` have no readers anywhere.

#### Performance
- While an AoE spell is being aimed, `gui/guispells/spell_handler.go:176-203` rewrites tile tints every frame whether or not the hovered tile changed, which sets the dirty flag and forces a full rebuild of the viewport batch every frame. The cost is modest (one viewport of tiles) but it is a direct result of tints travelling through the map's dirty flag.

#### Leave alone
- `TileRenderer.Render`'s batch-once-and-redraw approach is the right design for a static tile map.
- Grouping tiles by image into `QuadBatch`es.

---

### visual/combatrender (401 lines, tests: no)

**Role:** Combat-specific drawing: squad highlight borders, health bars, movement range, and the 3×3 unit grid on the attack-animation screen. **Verdict:** Needs attention: clean boundaries, but an inconsistent data-access pattern that costs per-frame work, and a layout bug for multi-cell units.

#### Design findings

##### [Medium] Provider interfaces pull data per squad per frame; pushing a slice would be simpler and cheaper
- Evidence: `types.go:32-43` defines `SquadInfoProvider` and `UnitInfoProvider`; the only implementation is `gui/framework.GUIQueries` (`gui/framework/guiqueries_rendering.go:13-69`). `SquadHighlightRenderer.Render` (`squadhighlights.go:75-79`) and `HealthBarRenderer.Render` (`combatoverlays.go:77-81`) each call `GetAllSquadIDs()` (a full ECS query, `squadcore.FindAllSquads`) and then `GetSquadRenderInfo` per squad, which allocates a `SquadRenderInfo` that is a six-field copy of the framework's `SquadInfo` (`guiqueries_rendering.go:19-32`). `MovementTileRenderer.Render` in the same package takes its data as a slice argument instead (`combatoverlays.go:26`).
- Problem: two patterns for the same job in one 400-line package. The pull pattern exists only to avoid an import cycle with `gui/framework`, at the price of a one-implementation interface, an adapter file in `framework`, and every squad being queried and copied twice per frame. It also means the renderer decides which squads to show (all squads in the world), not the mode that knows the encounter.
- Direction: make the renderers take plain data: `Render(screen, center, squads []SquadRenderInfo, selectedID)`. The combat mode builds the slice once per frame (it already knows the encounter's squads) and hands it to both renderers. `SquadInfoProvider` disappears; `UnitInfoProvider` can follow the same route for the animation screen.
- Effort: S to M. Risk: low.

##### [Low] Faction colours are assigned by first-seen order
- Evidence: `squadhighlights.go:46-58`: colour index is `len(factionColors) % palette`, filled lazily in whatever order `GetAllSquadIDs` returns squads. `ResetFactionColors` is called per combat (`gui/guicombat/combatvisualization/manager.go:181`).
- Problem: the player's faction is not guaranteed the same colour from one battle to the next, and the mapping is hidden mutable state inside a renderer.
- Direction: have the caller state which faction is the player's (or pass the colour in the pushed data) so the player is always the same colour. This is presentation consistency, not a new feature.
- Effort: S. Risk: low.

##### [Low] Tunables are hardcoded in constructors
- Evidence: colours, bar height, width ratio, offsets, border thickness at `combatoverlays.go:21`, `:52-56`; `squadhighlights.go:13-22`, `:38-41`.
- Problem: minor; acceptable for now. Mentioned only so it is a conscious choice.
- Direction: leave unless a theme/config pass is planned.
- Effort: S. Risk: low.

#### Probable bugs
- **Multi-cell units are laid out wrongly on the attack-animation grid** (confidence: medium-high). `RenderUnitWithColor` rotates only the anchor cell (`squad_renderer.go:90-98`) and then sizes the sprite with unrotated `CellWidth`/`CellHeight` (`:108-109`). For a unit two cells wide at row `r`, columns `c..c+1`:
  - Attacker side: the footprint after rotation should be one column wide and two rows tall starting at display row `c`; the code draws it two columns wide starting at display column `2 - r`, which for `r = 0` runs off the right edge of the 3×3 grid.
  - Defender side: the rotated footprint's top row should be `2 - c - (CellWidth - 1)`; the code uses `2 - c`, one row too low, and again does not swap the dimensions.
  
  `monsterdata.json` contains two width-2 units and one height-2 unit, so this is reachable. Arithmetic traced; not observed in a running game.

#### Dead code
- `SquadCombatRenderer.RenderSquadWithHighlight` (`squad_renderer.go:42-70`): no callers in the repo. The animation mode has its own per-unit colour loop that calls `RenderUnitWithColor` directly (`gui/guicombat/combatanimation/render.go:46-63`).
- `SquadHighlightRenderer.Render`'s `currentFactionID` parameter (`squadhighlights.go:70`): never read; callers still compute and pass it.
- `SquadRenderInfo.ID` (`types.go:12`): set by the adapter, never read by a renderer.

#### Performance
- Squad list queried and `SquadRenderInfo` allocated twice per squad per frame (see first finding).
- `drawHealthBar` calls `SubImage` per squad per frame (`combatoverlays.go:115`), which allocates a new `*ebiten.Image` wrapper each time; scaling the fill image horizontally with `GeoM` avoids it. Small.

#### Leave alone
- The package's independence from `tactical/*` is the right boundary; keep data flowing in as plain structs.
- `SquadCombatRenderer.RenderUnitWithColor`'s fit-and-centre logic is fine apart from the rotation bug.
- `MovementTileRenderer` is as simple as it should be.

---

### visual/vfx (512 lines, tests: no)

**Role:** Short-lived image effects (fire, ice, cloud, electricity) played at spell targets. **Verdict:** Needs attention: the right size and well isolated, but the abstraction is wider than its one implementation, it works in screen pixels, it uses the simulation RNG, and animation state is advanced in the wrong place.

#### Design findings

##### [High] Animators consume the shared simulation RNG every drawn frame
- Evidence: `animators.go:31`, `:35`, `:39`, `:43`, `:56-57` call `common.RandomFloat()`; `core/common/randnumgen.go:8` is the single seeded generator also used for dice rolls (`:22-27`).
- Problem: cosmetic flicker advances the same random stream that decides combat outcomes, by an amount that depends on frame rate and on how long an effect is on screen. Determinism from a seed (the purpose of the shared RNG) is lost the first time a fire effect plays.
- Direction: give `vfx` its own private `rand.Rand` (seeded from anything; it does not need to be reproducible). One line in the package, no API change.
- Effort: S. Risk: low.

##### [Medium] Animation state advances inside `Draw`, per copy, on a frame counter, while lifetime uses the wall clock
- Evidence: `BaseEffect.DrawVisualEffect` calls `animator.Update` (`vx.go:70-74`); `UpdateVisualEffect` only checks expiry (`:53-63`). Both animators ignore the `elapsed` argument and step internal counters instead (`animators.go:23`, `:79`). `Copy()` shares the animator pointer between all tiles of an area (`vx.go:114-117`).
- Problem: three clocks (wall-clock lifetime, draw-call counters, and the game's fixed 1/60 tick that is not used at all). Because every copy's `Draw` calls `Update` on the shared animator, an N-tile cloud advances its pulse phase N times per frame, so pulse speed scales with area size, and "lock-step" random animators actually give each tile different values. The doc comment on `Copy` (`vx.go:104-113`) states the opposite ("advances once per frame and is sampled by every copy").
- Direction: compute animation state as a pure function of `elapsed` (which is already passed in), and compute it once per effect per frame rather than per tile. That removes animator mutable state, `Reset`, and the sharing question.
- Effort: S to M. Risk: low (visual only).

##### [Medium] The API works in screen pixels and has two parallel effect kinds
- Evidence: `NewEffect(startX, startY, …)` takes screen coordinates (`vxfactory.go:19`); `NewVisualEffectArea` takes a logical centre plus tile indices and converts to screen itself (`vxhandler.go:22-40`); the handler keeps two lists and duplicates every loop (`:64-126`). The caller does the logical-to-screen conversion for single targets and rebuilds a shape for areas (`gui/guispells/spell_handler.go:336-357`).
- Problem: positions are frozen at spawn time, so an effect does not follow the map if the view centre changes while it plays; callers need to know about camera centre and scale; and `VisualEffectArea` is not a `VisualEffect`, which is why there are two of everything.
- Direction: one entry point taking logical tile positions (`Spawn(kind, positions []LogicalPosition, duration)`), with conversion to screen done at draw time against the current centre. One effect type holding a list of positions replaces `VisualEffect` + `VisualEffectArea` + the two lists.
- Effort: M. Risk: low.

##### [Medium] `VisualEffect` is a seven-method interface with one implementation
- Evidence: `vx.go:10-18`; only `BaseEffect` implements it. `SetVXCommon`, `VXImg` and `Copy` exist solely so `NewVisualEffectArea` can clone through the interface (`vxhandler.go:31-34`); `ResetVX` has no caller.
- Problem: speculative abstraction; the interface is wider than any consumer needs and forces accessor methods for what are private fields of the only implementation.
- Direction: use the concrete type. `Animator` and `Renderer` (two implementations each) can stay as the extension points, or shrink to plain functions if the previous finding is taken.
- Effort: S. Risk: low.

##### [Low] Effect kinds are string-keyed with a silent fallback and no load-time validation
- Evidence: `CreateVisualEffectByType` defaults to fire for unknown strings (`vxfactory.go:95-108`); `templates/spelldefinitions.go:43` declares `VXType` and a grep of `templates/` finds no validation of it. Current data uses only the four known values (`cloud` 12, `electricity` 6, `fire` 16, `ice` 13).
- Problem: a typo in `spelldata.json` shows fire with no error, contrary to the templates package's validate-at-boot convention.
- Direction: export the set of known kinds (or an `IsKnownKind`) so the spell validator can reject unknown `vxType` at load.
- Effort: S. Risk: low.

##### [Low] Global handler plus duplicate free functions
- Evidence: `var VXHandler` (`vxhandler.go:9`), `AddVX`/`AddVXArea` wrappers (`:70-76`) next to the methods they forward to (`:78-84`); `game_main/setup.go:159` resets it by assignment.
- Problem: two API surfaces for one thing. A package-level singleton is acceptable here, but pick one surface.
- Direction: keep the package-level functions (`Add`, `Update`, `Draw`, `Reset`) and make the handler type unexported, or the reverse.
- Effort: S. Risk: low.

#### Probable bugs
- **Pulse speed scales with AoE size; random effects are not in lock-step** (confidence: high). Trace given in the second finding.
- **Cloud "fluff" layers ignore the scale factor** (confidence: high). The main image is scaled by `state.Scale * ScaleFactor` (`renderers.go:49`), the two extra layers by `layerScale` only (`:62-63`), and the centring offsets (`:52-53`, `:65-66`) ignore the scale factor as well. With the default scale factor of 3 the layers are drawn at one third size in the tile's top-left corner. Twelve spells use the cloud effect.
- **Effects are frozen to the screen position at spawn** (confidence: high as code behaviour, low impact today since the view centre does not move during combat).
- Image load errors are discarded (`vxfactory.go:22`); a missing asset yields an effect that silently draws nothing (`renderers.go:13-15`).

#### Dead code
- `VisualEffect.ResetVX` / `BaseEffect.ResetVX` (`vx.go:16`, `:95-102`), and with it `originalDuration` and both `Animator.Reset` methods (`animators.go:64-66`, `:87-89`): no callers.
- `EffectConfig` and `NewEffect` are exported but used only inside the package.

#### Performance
- **Every effect spawn decodes its PNG from disk and creates a new GPU texture** (`vxfactory.go:20-23`). A single-target spell on several squads loads the same file once per squad (`spell_handler.go:348-357`). Four images total are involved; load each once (lazily, into a package-level map) and reuse. This also makes missing-asset errors visible in one place.
- `opts.ColorM` (`renderers.go:23-29`, `:55`, `:69`) is the deprecated colour path in ebiten 2.7; the rest of the slice uses `ColorScale`. For plain opacity/brightness scaling `ColorScale` is equivalent and stays on the fast batch path.

#### Leave alone
- The package's isolation: it imports only `core/*` and knows nothing about spells. Keep that.
- Animator/Renderer composition is a reasonable level of flexibility for four effects.
- Effect definitions in Go rather than JSON are fine at this count; moving them to data would be a feature, not a cleanup.

---

### input (137 lines, tests: no)

**Role:** Moves the player avatar on the exploration map and handles two debug keys. **Verdict:** Needs significant rework in the sense that the package should not exist; the code itself is small and easy to relocate.

#### Design findings

##### [High] The package is misnamed, sits outside the mode system, and should be folded into the exploration mode
- Evidence: single file `cameracontroller.go`. It depends on `gui/framework` for `InputState` and the action enum (`:5`, `:28-93`). `game_main/main.go:60-68` calls it every tick, in every mode, by reaching into the tactical mode manager through `GetInputState()`, an accessor whose doc comment says it exists for this controller (`gui/framework/modemanager.go:210-214`; that is its only caller). The actions it listens for are bound only by the exploration mode (`gui/guiexploration/explorationmode.go:56`), whose own `HandleInput` does nothing but common input (`:108-115`).
- Problem: one mode's input handling lives in a top-level package named `input`, is invoked from `main` instead of from the mode, and requires a special escape hatch in the framework. It is called a camera controller but moves an ECS entity and mutates map tiles. `CLAUDE.md` lists `input/cameracontroller.go` as the "Camera and input control" system, which misdirects readers away from the real input system in `gui/framework`.
- Direction: move `movePlayer` and the two debug actions into `gui/guiexploration` and call them from `ExplorationMode.HandleInput`. Delete the `input` package, `Game.cameraController`, `SetupInputCoordinator`, `main.HandleInput`, and `UIModeManager.GetInputState`. Update the Key Systems list in `CLAUDE.md` and the two layout docs.
- Effort: S. Risk: low.

##### [Medium] Player movement is hand-rolled here and again in exploration mode, bypassing `EntityManager.MoveEntity`
- Evidence: `cameracontroller.go:112-123` updates `GlobalPositionSystem` and then writes the position component through the `playerData.Pos` pointer. `gui/guiexploration/exploration_panels_registry.go:66-73` does the same two steps in the opposite order. `core/common/ecsutil.go:133-155` is the documented atomic helper for exactly this and `CLAUDE.md` tells callers to use it.
- Problem: two copies of a two-step update that the project has a single function for.
- Direction: both sites call `manager.MoveEntity` (the controller already holds an unused `ecsManager`).
- Effort: S. Risk: low.

##### [Medium] The avatar marks its tile as blocked terrain
- Evidence: `cameracontroller.go:125-126` sets `nextTile.Blocked = true` and `oldTile.Blocked = false`. A repo-wide grep shows this is the only runtime writer of `Tile.Blocked`; every other write is in map generation.
- Problem: occupancy is being recorded in terrain data, a second source of truth beside `GlobalPositionSystem`. Other code that repositions the avatar does not maintain it (`exploration_panels_registry.go:66-73`), nor does player creation, so the flag is already inconsistent with the avatar's real position at start. The empty `else` branch with a "Melee combat removed" comment (`:127-130`) shows this is a leftover from an earlier game design.
- Direction: delete both lines and the empty branch after confirming nothing reads `Blocked` expecting the avatar's tile (checked afterwards: outside map generation the only reader is this controller, and the map save chunk persists the flag).
- Effort: S. Risk: low to medium.

##### [Low] Eight copy-pasted direction blocks
- Evidence: `cameracontroller.go:32-79`.
- Problem: 48 lines for a table of eight (action, dx, dy) entries.
- Direction: loop over a small table when the code is moved.
- Effort: S. Risk: low.

#### Mechanics notes
- "Player position" doubles as the camera centre for every tactical view: `PlayerData.Pos` is passed as the viewport centre by the map renderer (`game_main/main.go:129-130`), combat (`gui/guicombat/combatmode.go:196`, `:211`), deployment (`gui/guisquads/squaddeploymentmode.go:191`) and spell targeting. There is no camera concept separate from the avatar entity. That is a workable simplification; it only needs honest naming. No change recommended beyond the rename implied by removing `CameraController`.

#### Dead code
- `CameraController.ecsManager` (`:12`, `:20`): stored, never read.
- `NewCameraController`'s `coordinator` parameter (`:18`): unused.
- `PlayerInputStates.HasKeyInput`: set eight times here (`:34-77`), cleared immediately in `game_main/main.go:65-67`, and read nowhere else. `InputState.PlayerInputStates` (`gui/framework/uimode.go:86`, assigned at `modemanager.go:32`, `:186`) is likewise only ever assigned. The whole "bridge to existing system" can be removed, including the `PlayerInputStates` type in `core/common/playerdata.go:18-22`.
- Empty `else` branch (`:127-130`).

#### Leave alone
- Using semantic actions from the `ActionMap` rather than raw keys is the right pattern; keep it when the code moves.
