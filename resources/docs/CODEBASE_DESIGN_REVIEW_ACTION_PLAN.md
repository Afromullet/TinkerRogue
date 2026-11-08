# Codebase Design Review: Action Plan

**Source:** [`CODEBASE_DESIGN_REVIEW.md`](CODEBASE_DESIGN_REVIEW.md) (reviewed commit `d99ffdd`, 2026-11-06)

This is the ordered list of what to do. It says what and in what order, not how. For the evidence and the direction of each fix, follow the reference at the end of each item:

- `§N` is a section of the review.
- `L1`, `C4`, `S2`, `R1`, `G1`, `W3` are entries in the review's bug index (§6).
- `D7` is a decision in Step 0 below (same numbering as review §5).

**Why this order.** Decisions first, because several of them turn "fix" into "delete". Then deletion, so later steps have less to touch. Then small fixes to the core loop, which are independent of each other. Then the design work (one owner per fact, rules into the simulation, data loading). Package moves come late because they are mechanical and easier once the dead code is gone and responsibilities have settled. GUI infrastructure and save hardening come last because earlier steps change what they need to cover.

Work top to bottom. Inside a step, items are independent unless one says "after".

---

## Step 0: Make the decisions

Nothing here needs code. Items marked **(gates deletion)** should be answered before Step 2.

- [ ] **D1** Can the overworld be won or lost? **(gates deletion)**
- [ ] **D2** Are raids fought on generated floor maps? **(gates deletion)**
- [ ] **D3** Is fleeing from combat a feature? **(gates deletion)**
- [ ] **D4** Should the squad editor have undo? **(gates deletion)**
- [ ] **D5** Keep the squad deployment mode? **(gates deletion)**
- [ ] **D6** Is "attack mode" a real combat state? **(gates deletion)**
- [ ] **D10** Are squad `Formation`, `Morale`, `SquadLevel`, `TurnCount` planned mechanics? **(gates deletion)**
- [ ] **D13** Keep the fog-of-war and stairs leftovers? **(gates deletion)**
- [ ] **D7** Can healing save a unit that took lethal damage in the same exchange? (needed for Step 4)
- [ ] **D8** What happens when a squad leader dies? (needed for Step 4)
- [ ] **D14** Do you want reproducible runs from a seed? (needed for Step 6)
- [ ] **D15** Artifact on a destroyed squad: returned or lost? Charges per behavior or per copy? (needed for Step 4)
- [ ] **D9** Do perk role restrictions restrict equipping? (needed for Step 5)
- [ ] **D12** Should Cone spells differ from Line spells? (needed for Step 3)
- [ ] **D19** Do rewards go to the squads that fought or the whole roster? (needed for Step 5)
- [ ] **D11** Are faction resources, containment and faction weakening meant to work?
- [ ] **D16** Is manual raid deployment planned?
- [ ] **D17** Should the AI's positional and support layers affect decisions?
- [ ] **D18** Who are overworld events for: the player or a debug export?

---

## Step 1: Confirm the key bugs in a running game

The review traced these in code only. Confirm them before scheduling the fixes that depend on them.

- [ ] 1.1 Win an overworld fight and check whether the threat's intensity changes. (L7)
- [ ] 1.2 Attack an NPC-garrisoned node twice. (L8)
- [ ] 1.3 Open the overworld and node placement screens and check for the six missing panels. (G1)
- [ ] 1.4 Open Unit Purchase and check whether the detail text is visible and scrolls. (G5)
- [ ] 1.5 Lose a raid battle, and separately win one that lasts more than one turn; note where the game lands. (L1, L5)
- [ ] 1.6 Save and load with equipped perks and unassigned roster units. (S1, S2, S5)

---

## Step 2: Clear the ground

### 2A. Delete what needs no decision

- [ ] 2.1 The three GUI render caches and the background pre-cache calls. (§17, G5)
- [ ] 2.2 The root `test_init.go` scratch file. (§10)
- [ ] 2.3 The unused reorder-squads command. (§12)
- [ ] 2.4 The dead methods on `CombatService` and `AICoordinator`. (§13)
- [ ] 2.5 The `WorldTags` string registry. (§10)
- [ ] 2.6 The 26 unused layout constants. (§17)
- [ ] 2.7 The two orphaned gamedata files and the dead config keys. (§11)
- [ ] 2.8 Write-only fields and unused functions from each package's "Dead code" list. (§10 to §20)
- [ ] 2.9 Encounter history and the data that exists only to feed it. (§15)

### 2B. Delete or keep according to the decisions

- [ ] 2.10 Overworld victory package and everything only it uses. (D1, L9)
- [ ] 2.11 Garrison tile generator, terrain file and room-size data. (D2)
- [ ] 2.12 Flee path across GUI, lifecycle and raid. (D3, L6)
- [ ] 2.13 Squad editor undo wiring and the Ctrl+Z text. (D4, G7)
- [ ] 2.14 Squad deployment mode, its service, bindings and constants. (D5, C26)
- [ ] 2.15 Attack mode flag, button and target hotkeys. (D6)
- [ ] 2.16 Unread squad and leader fields, and the formation parameter. (D10)
- [ ] 2.17 Fog-of-war and stairs leftovers. (D13, W1)
- [ ] 2.18 Inert faction mechanics. (D11)

### 2C. Fix the guide and tooling

- [ ] 2.19 Correct `CLAUDE.md`: paths, reference implementations, method names, the two rules the code does not follow. (§3.13)
- [ ] 2.20 Add `staticcheck.conf` and install the tool.
- [ ] 2.21 Do the one-off repo-wide gofmt pass as its own change.
- [ ] 2.22 Update the `gui-architect` agent guidance to match the code, or mark it as aspirational. (§3.13)

---

## Step 3: Core-loop correctness

Small, independent fixes. Order within the step is by impact on being able to play a full session.

### 3A. Combat and raid lifecycle

- [ ] 3.1 Combat exit is an explicit operation, not a side effect of leaving the combat screen. (L4)
- [ ] 3.2 The post-combat destination survives End Turn. (L5)
- [ ] 3.3 Raid defeat is processed. (L1)
- [ ] 3.4 The final floor's stairs end the raid. (L2)
- [ ] 3.5 A second raid in one session gets post-combat processing. (L3)
- [ ] 3.6 A failed combat start cleans up after itself, and encounter entities stop leaking. (L10, L11)
- [ ] 3.7 A failed load does not leave a half-built world. (L12)

### 3B. Overworld loop

- [ ] 3.8 Victory weakens or destroys the threat. (L7)
- [ ] 3.9 A garrisoned node can be fought more than once; squads are not stranded as garrisoned; garrisoned squads are not auto-deployed. (L8, L13, L14)
- [ ] 3.10 Influence radius follows intensity; fix the nil dereference; fortification gain respects difficulty. (W5, W6)

### 3C. Save and load

- [ ] 3.11 Unassigned roster units are saved. (S1)
- [ ] 3.12 Equipped perks are saved. (S2)
- [ ] 3.13 Raid garrison data stays on its squad after load. (S3)
- [ ] 3.14 The raid's player ID is saved. (S4)
- [ ] 3.15 The roster works after a load. (S5)

### 3D. Combat rules

- [ ] 3.16 Perk state resets at the right moment. (C1, C23)
- [ ] 3.17 Guardian Protocol protects neighbours as described. (C8)
- [ ] 3.18 Buff spells can target friendly squads. (C5)
- [ ] 3.19 AoE shapes are clipped to the map; remove the panic path. (C6)
- [ ] 3.20 Rectangle sizes match the data; Cone per D12. (C7)
- [ ] 3.21 Spell effects play on the rotated tiles, and rotation has a key. (C24)
- [ ] 3.22 Combat balance config fails loudly, and the simulator loads it. (C21)
- [ ] 3.23 Generated squads keep all their units; multi-cell units cannot overlap or leave the grid. (C13, C14)
- [ ] 3.24 Combat movement respects terrain and map bounds. (C16; confirm in game first)
- [ ] 3.25 The AI's evaluator cache survives teardown correctly, and the AI finds factions without the GUI. (C25)
- [ ] 3.26 Raid deployment uses the right strength profile name. (W9)

### 3E. Visible GUI defects

- [ ] 3.27 The six missing panels exist. (G1)
- [ ] 3.28 Hotkeys do not fire while typing in a dialog; clicks on panels do not reach the map. (G2, G3)
- [ ] 3.29 List refresh keeps the panel layout. (G4)
- [ ] 3.30 ESC leaves unit view and the roguelike squad editor. (G6)
- [ ] 3.31 Undoing a purchase cannot dispose a unit that is in a squad. (R2)
- [ ] 3.32 Raid summary reflects the outcome; overworld "Age" shows an age. (G11, G12)
- [ ] 3.33 Multi-cell units are laid out correctly on the animation grid. (G9)
- [ ] 3.34 Cloud effect layers scale; pulse speed does not depend on area size. (G10)
- [ ] 3.35 The XP button in unit view is debug-only. (§18)

### 3F. World generation

- [ ] 3.36 The player never starts on a wall. (W2)
- [ ] 3.37 Caverns stay connected after the late terrain passes. (W3)
- [ ] 3.38 Starting-faction fallback positions are inside the map and validated. (W4)

---

## Step 4: One owner per fact

The design core of the review (§3.2). Order is by how many bugs each item removes.

- [ ] 4.1 Movement speed and attack range have one home. (C4)
- [ ] 4.2 One notion of "did this unit die" for an attack. After D7. (C2, C3, C10, C11)
- [ ] 4.3 The unit roster stores only ownership; rosters are pruned when units and squads die. (R1, R3)
- [ ] 4.4 One way to build a unit and a squad; unit templates stop carrying placement and duplicate attributes. (C12)
- [ ] 4.5 One way to pick a leader and one rule for a dead leader. After D8. (§12)
- [ ] 4.6 Action state lives on the squad; the combat query cache and faction manager objects go away. (C22)
- [ ] 4.7 The spellbook is derived, not stored. (C18)
- [ ] 4.8 Artifact equip state has one owner; charges and deferred effects live in the ECS. After D15. (C19)
- [ ] 4.9 Effects are reversible and cleaned up on every combat exit. (C15)
- [ ] 4.10 Garrison membership has one owner. After 3.9. (§16)
- [ ] 4.11 Walkability is owned by the map. (§11)
- [ ] 4.12 Screen and map dimensions come from one object built after data load. (§10)
- [ ] 4.13 Faction identity has one representation; `ThreatType` goes; "threat" has one definition. (§16)
- [ ] 4.14 One overworld turn counter; one combat round counter. (§12, §13)
- [ ] 4.15 Raid room and squad relations are stored once. (§16)
- [ ] 4.16 "Has moved" and "movement remaining" mean one thing. (§13)
- [ ] 4.17 One definition of "alive" in the GUI. (§17)

---

## Step 5: Rules into the simulation, presentation out of it

### 5A. Rules

- [ ] 5.1 The combat turn sequence runs without the GUI, and the simulator uses it. (§3.1)
- [ ] 5.2 Spell casting validates and charges its own cost; AoE targeting is a simulation rule. (§18, §20)
- [ ] 5.3 Spell damage and attack damage share one apply-and-resolve path. (C20)
- [ ] 5.4 Artifact activation has one validated entry point. (§14)
- [ ] 5.5 Perk equipping checks unlocks, and roles per D9. (§14)
- [ ] 5.6 Perk modifier hooks are read-only; "next attack" bonuses apply per squad attack. (C9)
- [ ] 5.7 Moving a squad enforces zone of control. (C17)
- [ ] 5.8 Placing a player node validates placement. (§19)
- [ ] 5.9 Raid progression is owned by the raid runner. (§16, §19)
- [ ] 5.10 Recruiting a commander is one system function shared with initial setup. (§19)
- [ ] 5.11 Map regeneration is shared between boot and the debug menu. (§19)
- [ ] 5.12 Target patterns, attack eligibility, squad attack range, valid move tiles and unit strength each exist once. (§3.3)
- [ ] 5.13 One squad generator for encounters, garrisons and starting squads. (§15, §10)
- [ ] 5.14 Teardown stripping exists once; one outcome type for a battle. (§13, §15)
- [ ] 5.15 Rewards per D19. (§15)

### 5B. Presentation

- [ ] 5.16 The attack animation shows what the combat log recorded. (G8)
- [ ] 5.17 The AI reports what it did; the GUI owns playback. (§15)
- [ ] 5.18 Tile overlays are owned by the renderer, per owner, not stored on map tiles. (G13, G14)
- [ ] 5.19 Map generators produce terrain; images are assigned in one pass shared with load. (§11)
- [ ] 5.20 Battle and overworld recording follow the session, not the screen. (§13, §19)
- [ ] 5.21 Game-flow state leaves the UI state structs; game mode is explicit, not inferred from registered screens. (§17, §10)

---

## Step 6: Data loading and randomness

- [ ] 6.1 One boot-time load with one failure policy. (§3.4)
- [ ] 6.2 Every gamedata file goes through the shared loader. (§11)
- [ ] 6.3 Validators for monsters, spells, artifacts, unit spells, stat names and raid archetypes, plus one cross-file reference check. (§11)
- [ ] 6.4 Lists kept in sync by comment are defined once. (§11)
- [ ] 6.5 No code defaults behind validated values. (§15, §16)
- [ ] 6.6 Map generation config has one source. (§11)
- [ ] 6.7 Config mirror between `templates` and `core/config` is replaced by one assignment. (§10)
- [ ] 6.8 RNG policy per D14: boot seeding, no reseeding by generators, no stray generators. (W7)
- [ ] 6.9 Visual effects stop using the simulation RNG. (W7)
- [ ] 6.10 Anything read out of a map is sorted before it is displayed or indexed. (W8)
- [ ] 6.11 One shared image loader. (§3.9)
- [ ] 6.12 GUI resources load with one error-returning call at boot. (§17)

---

## Step 7: Package layout

One area per change. Update `PROJECT_LAYOUT.md`, `PACKAGE_DEPENDENCIES.md` and the Key Systems list in `CLAUDE.md` in the same change. Target layout is in review §4.

- [ ] 7.1 `tactical/combat`: seven packages to five.
- [ ] 7.2 `tactical/squads`: dissolve `squadservices`; move the combat move command to combat.
- [ ] 7.3 `campaign/overworld`: merge the sub-packages; keep `ids` and `overworldlog`.
- [ ] 7.4 Combat lifecycle and the encounter session move to `tactical/combat`; overworld encounter rules move to `campaign`.
- [ ] 7.5 `mind/spawning` stops importing `campaign`.
- [ ] 7.6 `tactical/commander` moves to `campaign`.
- [ ] 7.7 `visual`: dissolve `graphics`; merge `maprender` into `rendering`.
- [ ] 7.8 `input` folds into the exploration mode.
- [ ] 7.9 `gui/guicombat`: merge `combatbase` and `combatinput` back in.
- [ ] 7.10 `gui/guinodeplacement` folds into `gui/guioverworld`.
- [ ] 7.11 Entity factory leaves `templates`; `testing/bootstrap` leaves the shipping binary.
- [ ] 7.12 Remaining wrong-way imports from review §3.12 are gone.
- [ ] 7.13 Optional: combat sub-panels renamed or moved under `guicombat`; `gui/widgets` dissolved; `roomtypes` folded.

---

## Step 8: GUI infrastructure

- [ ] 8.1 Remove the dead declarative panel path, `StandardPanels` and the unused panel builder layer. (§17)
- [ ] 8.2 One way to build a panel container; shared helpers for sub-menus, detail panels and widget lookup. (§17, §19)
- [ ] 8.3 Screen-fraction arithmetic goes through two helpers. (§17)
- [ ] 8.4 Remove the shared squad info cache; narrow `GUIQueries` to view models. (§17)
- [ ] 8.5 Mode names are constants; one transition helper; context switching calls `Exit`. (§17)
- [ ] 8.6 The five combat input flags become one mode value. (§17)
- [ ] 8.7 Squad editor: lists update in place and are keyed by ID; the two artifact tabs share one implementation. (§18, R4)
- [ ] 8.8 Combat mode: one wiring style; the input handler is split by sub-mode. (§19)
- [ ] 8.9 Combat visualization stops keeping its own copy of AI data. (§19)
- [ ] 8.10 Optional, mode by mode: typed widget fields instead of string-keyed lookups. (§17)

---

## Step 9: Save system hardening

- [ ] 9.1 Decide whether the overworld is saved; stop offering save where it cannot work. (L9)
- [ ] 9.2 Plain-data components are saved directly, without mirror structs. (§10)
- [ ] 9.3 Loading builds entities through the same factories the game uses. (§10)
- [ ] 9.4 References that must resolve fail the load when they do not. (§10)
- [ ] 9.5 Chunks restore data only; images are attached in one place. (§10)
- [ ] 9.6 Derived data leaves the map save. (§11)

---

## Ongoing: rules for new work

Apply from now on, independent of the steps above (review §8).

1. One owner per fact; derive the rest.
2. The simulation is complete without the GUI.
3. Screen `Enter` and `Exit` do visual setup and cleanup only.
4. All game data goes through the shared loader with a validator; boot fails loudly.
5. A persistent component changes together with its save chunk.
6. A new package needs a stated reason.
7. No fields or parameters for future features.
8. Identities are typed constants defined once.
9. Presentation never uses the simulation RNG or writes into the world model.
10. One way to do each job.
11. The simulator runs the game's code paths.
12. Keep `staticcheck` and a dead-code check in regular use.
