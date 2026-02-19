# Garrison Raid — Implementation Overview

**Purpose:** High-level roadmap for implementing the Garrison Raid mode described in `garrison_raid_design.md`. Covers phases, new packages/files, modified files, systems, and implementation order.

---

## New Packages and Files

### New Package: `tactical/raid/`

The central orchestration package for garrison raid logic. Contains all raid-specific state management, flow control, and between-encounter logic.

| File | Purpose |
|------|---------|
| `components.go` | ECS components: `RaidStateData`, `FloorStateData`, `AlertData`, `GarrisonSquadData`, `RoomData` |
| `raidrunner.go` | Core raid loop — manages floor transitions, encounter sequencing, victory/defeat/retreat checks |
| `floorgraph.go` | DAG representation of rooms per floor, path traversal, room adjacency, critical path tracking |
| `alert.go` | Alert level tracking, buff application to garrison units via the effects system |
| `recovery.go` | Between-encounter and between-floor HP/morale recovery logic |
| `garrison.go` | Garrison army composition — archetype instantiation, reserve squad management, squad destruction tracking |
| `archetypes.go` | The 8 squad archetype definitions (Chokepoint Guard, Shield Wall, etc.) and their unit compositions |
| `rewards.go` | Reward distribution after room clears — spell/unit/artifact/mana grants |
| `priority_targets.go` | Priority target effects — what happens garrison-wide when Armory/Command Post/Mage Tower are cleared |
| `queries.go` | Query functions for raid state: remaining garrison strength, room status, floor progress |
| `init.go` | Subsystem registration via `common.RegisterSubsystem` |

### New Package: `gui/guiraid/`

UI for the raid mode — floor map display, squad deployment screen, between-encounter menus, room selection.

| File | Purpose |
|------|---------|
| `raidpanel.go` | Main raid UI panel — floor map visualization, room selection, path display |
| `deploypanel.go` | Pre-encounter squad deployment/rotation screen |
| `recoverypanel.go` | Between-encounter summary — HP recovered, morale changes, alert level |
| `retreatpanel.go` | Retreat confirmation and summary of retained rewards |
| `raidstate.go` | UI-only state for the raid mode (selected room, current panel, animation flags) |

### New Config File: `assets/gamedata/raidconfig.json`

Configurable values for the raid system — recovery percentages, alert buff values, morale thresholds, floor scaling parameters, archetype compositions.

### New Map Generator File: `world/worldmap/gen_garrison_raid.go`

BSP-based generator registered as `"garrison_raid"` that produces per-room tactical maps with terrain appropriate to each room type and garrison archetype.

---

## Modified Existing Files

| File | Change |
|------|--------|
| `mind/ai/action_evaluator.go` | Add support for `AIProfile` overrides (anchor point, modified approach multipliers, attack priority bonus) |
| `mind/behavior/threat_layers.go` or config | New AI profile presets for Garrison Defender, Guard Post, Reinforcement Rush |
| `assets/gamedata/aiconfig.json` | Add garrison-specific AI profile entries |
| `mind/encounter/encounter_service.go` | Extend to support raid-context encounters (garrison squad reference, room type, alert level passed in) |
| `gui/core/` (mode manager) | Register the raid GUI mode alongside existing combat/exploration/overworld modes |
| `tactical/squads/squadcombat.go` | Hook for morale changes on unit death (if not already present) |
| `tactical/effects/system.go` | Possibly extend to support bulk effect application/removal for alert buffs and morale debuffs |

---

## Systems to Implement

### 1. Raid State System
Tracks the overall raid lifecycle: which floor the player is on, which rooms have been cleared, garrison kill count, alert level per floor, and victory/defeat/retreat status.

### 2. Floor Graph System
Generates and manages the DAG of rooms per floor. Determines the critical path, branch paths, and tracks which rooms are accessible based on cleared rooms. Handles room type assignment (Barracks, Guard Post, Armory, etc.).

### 3. Garrison Composition System
Instantiates the garrison's defense for each floor. Assigns archetype squads to rooms based on room type and floor difficulty. Manages reserve squads that activate at higher alert levels. Tracks which garrison squads are alive across the entire raid.

### 4. Alert System
Integer counter per floor that increments after each encounter. Applies stat buffs to remaining garrison units via the effects system. Activates reserve squads at threshold levels. Resets on retreat.

### 5. Recovery System
Applies between-encounter and between-floor recovery. Differentiates between deployed squads (low recovery) and reserve squads (high recovery). Applies morale adjustments. Does NOT recover mana (mana only from Command Post rewards).

### 6. Squad Rotation System
Manages which squads are deployed vs. held in reserve for the next encounter. Enforces deployment limits. Feeds into the recovery system (reserve squads heal more).

### 7. Morale Debuff System
Monitors per-squad morale thresholds and applies/removes stat debuffs accordingly. Uses the existing effects system. Re-evaluates after every morale change (combat casualties, victories, rest rooms).

### 8. AI Profile System
Extends the existing utility AI with configurable profile overrides. Adds positional anchoring (penalize movement away from a point) and per-archetype scoring adjustments. No new AI architecture — just parameterization of existing `ActionEvaluator` scoring.

### 9. Priority Target System
Tracks which priority rooms (Armory, Command Post, Mage Tower) have been cleared on each floor. Applies garrison-wide debuffs to remaining squads when a priority target falls.

### 10. Reward System
Distributes rewards after room clears — spells added to the commander's spellbook, units recruited into squads, artifacts added to inventory, mana restored. Ties into existing spell, squad creation, and gear systems.

### 11. Raid Map Generator
BSP-based map generator that produces tactical combat maps per room. Room geometry and terrain placement vary by room type and garrison archetype (chokepoints for Guard Posts, open areas for Barracks, etc.).

### 12. Raid GUI
Floor map visualization, room selection, squad deployment screen, between-encounter summaries, retreat confirmation. Integrates with the existing mode manager for context switching between raid navigation and tactical combat.

---

## Implementation Order

### Phase 1: Core Data and State

**Goal:** Establish the raid's data model and state machine without any gameplay yet.

1. **Raid config file** — Create `raidconfig.json` with all tunable values (recovery %, alert buffs, morale thresholds, floor counts, archetype compositions)
2. **Raid components** — Define ECS components for raid state, floor state, room data, alert data, garrison squad data
3. **Archetype definitions** — Define the 8 garrison squad archetypes as data (unit compositions, roles, preferred room types)
4. **Floor graph data structure** — DAG representation of rooms with room types, connections, critical path marking
5. **Subsystem registration** — `init.go` with `common.RegisterSubsystem`

**Outcome:** All data types exist. Nothing runs yet but the shape of the system is defined.

### Phase 2: Garrison Setup and Floor Generation

**Goal:** Generate a garrison's defense for a full raid.

1. **Garrison composition** — Given floor count and difficulty, generate the full garrison: which archetypes on which floors, which rooms, which squads are reserves
2. **Floor graph generation** — Generate DAG room layouts per floor with room type assignment based on floor number and available archetypes
3. **Garrison squad instantiation** — Create actual ECS entities for garrison squads and their units using existing monster data and squad creation systems

**Outcome:** A complete garrison exists as ECS entities distributed across a multi-floor room graph. No combat yet.

### Phase 3: Raid Runner and Encounter Flow

**Goal:** The raid loop works end-to-end with sequential encounters.

1. **Raid runner** — Core loop: floor entry → room selection → encounter trigger → combat resolution → post-encounter processing → next room or floor transition
2. **Squad deployment/rotation** — Pre-encounter selection of which player squads to deploy vs. reserve
3. **Encounter integration** — Connect room encounters to the existing encounter service, passing garrison squad data and room context
4. **Victory/defeat/retreat checks** — After each encounter, evaluate raid end conditions
5. **Floor transitions** — When stairs are reached, advance to next floor, apply between-floor recovery

**Outcome:** A playable (but bare) raid loop. Player can fight through floors sequentially. No alert, no recovery, no rewards yet — just fight after fight.

### Phase 4: Attrition Systems

**Goal:** The resource curves that make the raid strategic.

1. **Recovery system** — Between-encounter HP and morale recovery with deployed/reserve differentiation
2. **Alert system** — Per-floor alert counter, garrison buff application, reserve squad activation
3. **Morale debuff system** — Threshold monitoring and debuff application/removal
4. **Permanent death tracking** — Ensure dead units stay dead across the entire raid, update squad grids accordingly

**Outcome:** The "double resource curve" is functional. Both sides degrade over time. Squad rotation matters. Alert makes efficiency matter.

### Phase 5: AI Profiles

**Goal:** Garrison squads fight according to their archetype identity.

1. **AI profile data structure** — Define the profile override format (approach multipliers, anchor point, attack priority)
2. **Action evaluator integration** — Modify the scoring pipeline to accept and apply profile overrides
3. **Three garrison profiles** — Implement Garrison Defender, Guard Post, and Reinforcement Rush profiles
4. **Profile assignment** — Assign profiles to garrison squads based on their archetype

**Outcome:** Chokepoint Guards hold doorways. Fast Response squads rush. Guard Posts barely move. Each archetype feels distinct in combat.

### Phase 6: Priority Targets and Rewards

**Goal:** Strategic room choices matter beyond just "fight or skip."

1. **Priority target effects** — Clearing Armory/Command Post/Mage Tower applies garrison-wide debuffs on the current floor
2. **Reward distribution** — Room-specific rewards granted on clear (spells, units, artifacts, mana)
3. **Reward persistence** — Rewards retained on retreat, handled correctly on defeat

**Outcome:** The strategic layer is complete. Players choose between safe paths, resource recovery, and high-value targets.

### Phase 7: Map Generation

**Goal:** Tactical combat maps that match room types.

1. **BSP garrison generator** — Registered as `"garrison_raid"` in the worldmap generator registry
2. **Room-type terrain** — Guard Posts get chokepoints, Barracks get open space, Mage Towers get elevated positions
3. **Floor-scaled parameters** — Early floors are more open, later floors have tighter defensive terrain
4. **Per-room generation** — Each room in the DAG generates its own tactical map when the player enters it

**Outcome:** Combat encounters have terrain that matches their narrative and tactical purpose.

### Phase 8: Raid GUI

**Goal:** The player can see and interact with the raid.

1. **Raid mode registration** — Register with the existing mode manager
2. **Floor map panel** — Visualize the room DAG, show cleared/available/locked rooms, highlight critical path
3. **Squad deployment panel** — Pre-encounter screen for choosing deployed vs. reserve squads
4. **Between-encounter summary** — Show recovery applied, alert level changes, garrison status
5. **Retreat and end-of-raid screens** — Retreat confirmation, victory summary, defeat summary

**Outcome:** The raid is fully playable with proper UI.

### Phase 9: Tuning and Polish

**Goal:** The raid feels good to play.

1. **Balance pass** — Tune recovery percentages, alert buff values, morale thresholds, archetype compositions
2. **Floor scaling** — Adjust per-floor difficulty curves
3. **Reward balancing** — Ensure rewards offset attrition at the right rate
4. **AI tuning** — Adjust profile weights so archetypes feel distinct but fair

**Outcome:** A complete, balanced garrison raid mode.

---

## Dependencies Between Phases

```
Phase 1 (Data) ──→ Phase 2 (Garrison Setup) ──→ Phase 3 (Raid Runner)
                                                       │
                                                       ├──→ Phase 4 (Attrition)
                                                       ├──→ Phase 5 (AI Profiles)
                                                       └──→ Phase 6 (Rewards)
                                                                │
                                            Phase 7 (Map Gen) ←─┤
                                            Phase 8 (GUI) ←─────┘
                                                       │
                                                       └──→ Phase 9 (Tuning)
```

Phases 4, 5, and 6 can be worked on in parallel after Phase 3 is complete. Phase 7 and 8 depend on the gameplay systems being functional. Phase 9 comes last.
