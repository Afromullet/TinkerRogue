# Garrison Raid Mode

**Status:** Design Document
**Parent:** `docs/garrison_attrition_design.md` (brainstorm)
**Constraint:** Uses ONLY existing systems. No new mechanics, consumables, fatigue, or equipment durability.

---

## 1. Core Premise

The garrison is an enemy stronghold. The player's **commander** and **squads** form an elite strike team — a raid party — assaulting it across a configurable number of floors (3-5). The garrison has a finite, organized defense. The raid team has finite HP, mana, morale, and living units.

Both sides burn down. The central question: **who is losing faster?**

The garrison's strength is a declining staircase — each killed squad is permanently gone. The raid team's combat readiness is a sawtooth wave — it drops during fights, partially recovers between them, then drops again. Victory goes to the player who spends resources efficiently against the garrison while preserving enough to reach the final floor.

---

## 2. The Raid Team

### What the Player Brings

- **1 Commander** — existing commander system (`tactical/commander/`)
  - `CommanderData`: name, active state
  - `ManaData`: `CurrentMana` / `MaxMana` — persists across all battles in the raid, does NOT regenerate naturally
  - `SpellBookData`: `SpellIDs []string` — the commander's prepared spells for this raid
  - `SquadRosterData`: references to the commander's squads

- **Up to N Squads** (N configurable, suggested default: 3-4)
  - Each squad uses the existing `SquadData` struct: 3x3 grid formation, morale, roles, cover
  - Equipped artifacts via `EquipmentData.EquippedArtifacts` — no changes to the artifact system
  - Units have full attributes (`Strength`, `Dexterity`, `Magic`, `Leadership`, `Armor`, `Weapon`)

### Squad Rotation

Before each encounter, the player chooses which squads to deploy. Squads held in reserve are not exposed to combat. Between encounters:

- **Deployed squads** recover a small % of missing HP (suggested: 10-15%)
- **Reserve squads** recover a larger % of missing HP (suggested: 25-30%)
- Morale adjustments apply (see Section 3)

This creates a rotation strategy: deploy Squad A while Squad B recovers, then swap. A player with 3 well-built squads sustains combat readiness longer than one relying on a single squad.

---

## 3. Attrition Axes

Four resources degrade over the course of a raid. All use existing systems — no new mechanics.

### 3.1 Hit Points

Units take damage in combat. Between encounters, partial percentage-based recovery occurs (deployed squads recover less, reserves recover more). Full HP recovery within a raid is rare. Squads accumulate wounds over multiple fights.

**Implementation:** Unit HP is tracked in `common.Attributes.Dexterity` (which functions as HP in the combat system). Recovery is a percentage of missing HP applied between encounters.

### 3.2 Unit Death

When a unit's HP reaches 0, it dies. Dead units are **permanent for the run** — they cannot be replaced mid-raid. A squad that loses its Tank loses cover for the back row. A squad that loses its healer loses sustain. Every death reshapes the squad's capability.

**Implementation:** Existing entity disposal via `manager.CleanDisposeEntity`. Dead units are removed from the squad's 3x3 grid.

### 3.3 Mana

The commander's mana pool (`ManaData.CurrentMana`) persists across battles and does **not** regenerate naturally between encounters. Mana is only recoverable via specific room rewards (e.g., clearing a Command Post grants +15 mana). This makes every spell cast a strategic decision — a Fireball spent on Floor 1 is a Fireball unavailable on Floor 4.

**Implementation:** Existing `tactical/spells/` system. `ManaData` already persists on the commander entity. The raid runner simply does not reset it between encounters.

### 3.4 Morale

Per-squad morale tracked in `SquadData.Morale` (0-100).

**What changes morale:**

| Event | Morale Change |
|-------|---------------|
| Unit killed in squad | -3 per death |
| Victory (encounter won) | +5 |
| Rest Room cleared | +5 |
| Defeat / Flee | -10 |

**Low morale debuffs** (applied via the existing effects system, `tactical/effects/`):

| Morale Range | Effect |
|-------------|--------|
| 30-100 | No penalty |
| 10-29 | -2 Dexterity (reduced evasion/accuracy) |
| 0-9 | -2 Dexterity AND -2 Strength (combat effectiveness collapses) |

**Implementation:** `SquadData.Morale` already exists. Debuffs use `effects.ApplyEffectToUnits()` with `ActiveEffect{Stat: StatDexterity, Modifier: -2, RemainingTurns: -1}`. Effects are re-evaluated between encounters — if morale recovers above threshold, debuffs are removed via `effects.RemoveAllEffects()` and reapplied at the appropriate tier (or not at all).

---

## 4. The Garrison

### 4.1 Enemy Organization

The garrison is a finite, organized military presence using the 23 monster types from `assets/gamedata/monsterdata.json`. Enemy squads are pre-composed using archetypes — they are not randomly generated.

### 4.2 Monster Roster (23 Types)

All monsters from `monsterdata.json`, organized by role:

**Tanks** (frontline, high armor/cover):
| Name | Str | Dex | Armor | Weapon | Attack Type | Size | Cover |
|------|-----|-----|-------|--------|-------------|------|-------|
| Knight | 14 | 25 | 10 | 8 | MeleeRow | 1x1 | 0.40 (range 2) |
| Fighter | 15 | 30 | 8 | 8 | MeleeColumn | 1x1 | 0.25 (range 1) |
| Spearman | 13 | 32 | 6 | 8 | MeleeColumn | 1x1 | 0.30 (range 1) |
| Orc Warrior | 17 | 24 | 8 | 7 | MeleeRow | 2x1 | 0.45 (range 2) |
| Ogre | 18 | 15 | 10 | 9 | MeleeRow | 2x2 | 0.50 (range 2) |

**DPS** (damage dealers, varied attack types):
| Name | Str | Dex | Armor | Weapon | Attack Type | Size |
|------|-----|-----|-------|--------|-------------|------|
| Warrior | 12 | 28 | 7 | 11 | MeleeRow | 1x1 |
| Swordsman | 12 | 40 | 5 | 9 | MeleeColumn | 1x1 |
| Rogue | 8 | 55 | 3 | 9 | MeleeColumn | 1x1 |
| Assassin | 9 | 60 | 2 | 10 | MeleeColumn | 1x1 |
| Goblin Raider | 8 | 42 | 2 | 10 | MeleeRow | 1x1 |
| Archer | 11 | 45 | 2 | 12 | Ranged (4) | 1x1 |
| Crossbowman | 10 | 38 | 5 | 11 | Ranged (3) | 1x1 |
| Marksman | 7 | 52 | 2 | 11 | Ranged (4) | 1x1 |
| Wizard | 10 | 20 | 3 | 2 | Magic (4) | 1x1 |
| Sorcerer | 11 | 22 | 3 | 3 | Magic (3) | 1x1 |
| Warlock | 9 | 26 | 4 | 4 | Magic (4) | 1x1 |
| Skeleton Archer | 7 | 35 | 1 | 5 | Magic (3) | 1x1 |

**Support** (healing, buffs, leadership, cover):
| Name | Str | Dex | Magic | Leadership | Attack Type | Cover |
|------|-----|-----|-------|------------|-------------|-------|
| Paladin | 13 | 22 | 8 | 60 | MeleeRow | 0.35 (range 1) |
| Scout | 7 | 50 | 0 | 12 | Ranged (3) | 0.10 (range 1) |
| Ranger | 8 | 48 | 8 | 15 | Magic (3) | 0.15 (range 1) |
| Mage | 12 | 18 | 13 | 20 | Magic (3) | 0.10 (range 1) |
| Cleric | 8 | 24 | 12 | 40 | Magic (2) | 0.20 (range 2) |
| Priest | 6 | 20 | 11 | 45 | Magic (2) | 0.25 (range 2) |
| Battle Mage | 11 | 28 | 10 | 22 | MeleeColumn | 0.20 (range 1) |

### 4.3 Squad Composition Archetypes

Eight garrison squad templates. Each archetype has a tactical identity and preferred room assignment.

#### Chokepoint Guard
**Composition:** 2 Knights + 2 Crossbowmen + 1 Priest
**Total Units:** 5
**Roles:** 2 Tank, 2 DPS, 1 Support
**Tactic:** Knights hold the front with 0.40 cover (range 2), shielding Crossbowmen who fire from behind. Priest provides healing support. Excels in narrow corridors and doorways.
**Preferred Rooms:** Guard Post, corridor intersections, stairway entrances

#### Shield Wall
**Composition:** 1 Ogre + 2 Archers + 1 Cleric
**Total Units:** 4
**Roles:** 1 Tank, 2 DPS, 1 Support
**Tactic:** The Ogre (2x2, 0.50 cover range 2) is a mobile fortress. Archers deal sustained ranged damage from behind maximum cover. Cleric keeps the Ogre alive. Slow but extremely durable.
**Preferred Rooms:** Barracks, large open rooms, armory defense

#### Ranged Battery
**Composition:** 1 Spearman + 2 Marksmen + 1 Archer + 1 Mage
**Total Units:** 5
**Roles:** 1 Tank, 3 DPS, 1 Support
**Tactic:** Spearman holds the line at range 2 while four ranged/magic attackers pour fire. High damage output but fragile if the Spearman falls. Mage provides area magic coverage.
**Preferred Rooms:** Elevated positions, rooms with long sight lines, towers

#### Fast Response
**Composition:** 2 Swordsmen + 2 Goblin Raiders + 1 Scout
**Total Units:** 5
**Roles:** 0 Tank, 4 DPS, 1 Support
**Tactic:** All units have movement speed 4-5. Closes distance immediately, overwhelming targets with melee pressure. Glass cannon — high damage, no cover, no healing. Scout provides ranged poke from movement speed 5.
**Preferred Rooms:** Patrol routes, open areas, reserve deployment

#### Mage Tower
**Composition:** 1 Battle Mage + 2 Wizards + 1 Warlock + 1 Sorcerer
**Total Units:** 5
**Roles:** 0 Tank, 3 DPS, 1 Support (Battle Mage)
**Tactic:** Massive magic damage output. Wizards hit 6-cell AoE patterns, Warlock hits corners, Sorcerer hits columns. Battle Mage provides frontline cover (0.20) and melee capability. Devastating but weak to fast melee rushes.
**Preferred Rooms:** Ritual chambers, libraries, elevated positions

#### Ambush Pack
**Composition:** 2 Assassins + 2 Rogues + 1 Ranger
**Total Units:** 5
**Roles:** 0 Tank, 4 DPS, 1 Support
**Tactic:** Maximum speed (movement 4-5 across the board). Assassins and Rogues have the highest Dex in the game (55-60). Ranger provides magic-type support from range. No cover, no healing — pure burst damage. Designed to hit hard and fast.
**Preferred Rooms:** Ambush positions, side corridors, flanking routes

#### Command Post
**Composition:** 1 Knight + 1 Paladin + 1 Crossbowman + 1 Cleric + 1 Priest
**Total Units:** 5
**Roles:** 1 Tank, 1 DPS, 3 Support
**Tactic:** Maximum leadership and healing. Paladin (Leadership 60), Priest (Leadership 45), Cleric (Leadership 40) create a heavily buffed defensive position. Knight provides cover, Crossbowman provides ranged threat. Hard to kill through healing pressure. The garrison's command node.
**Preferred Rooms:** Command rooms, officer quarters, priority target rooms

#### Orc Vanguard
**Composition:** 1 Orc Warrior + 1 Ogre + 2 Warriors
**Total Units:** 4
**Roles:** 2 Tank, 2 DPS
**Tactic:** Raw brute force. Orc Warrior (2x1, Str 17) and Ogre (2x2, Str 18) form an immovable front. Warriors deal strong melee damage behind cover. No support or healing — relies on overwhelming physical stats. Slow (movement 2-3) but devastating in close quarters.
**Preferred Rooms:** Main halls, barracks, gate defense

#### Undead Patrol
**Composition:** 2 Skeleton Archers + 2 Fighters + 1 Ranger
**Total Units:** 5
**Roles:** 2 Tank, 2 DPS, 1 Support
**Tactic:** Skeleton Archers provide cross-pattern magic attacks from range while Fighters hold the front with MeleeColumn attacks and moderate cover (0.25). Ranger adds magic damage and scouting. A versatile patrol composition that covers both melee and ranged threats.
**Preferred Rooms:** Patrol routes, perimeter defense, connecting corridors

> **Coverage:** All 23 monster types from `monsterdata.json` appear in at least one archetype.

---

## 5. Alert System

A simple per-floor integer that **only goes up**. Represents how aware and prepared the garrison is.

### 5.1 Alert Levels

| Alert Level | Trigger | Garrison Effect |
|-------------|---------|-----------------|
| **0 — Unaware** | Default starting state | Normal compositions. No stat bonuses. Only assigned squads present. |
| **1 — Suspicious** | After 1st encounter cleared | +1 Armor to all garrison units on this floor (via effects system) |
| **2 — Alerted** | After 2nd encounter cleared | +1 Armor, +1 Strength. One reserve squad enters the encounter pool. |
| **3 — Lockdown** | After 3rd encounter cleared | +1 Armor, +2 Strength, +1 Weapon. All reserve squads active. |

### 5.2 Mechanics

- Each completed encounter raises alert by exactly 1
- Alert buffs are applied via the existing effects system: `effects.ApplyEffectToUnits(unitIDs, ActiveEffect{Stat: StatArmor, Modifier: +1, RemainingTurns: -1}, manager)`
- Reserve squads are pre-defined per floor but only enter the encounter pool at the specified alert threshold
- Alert does not decay — once raised, it stays

### 5.3 Strategic Implication

Fewer fights = less alert = weaker garrison. The player is rewarded for efficiency:
- Choosing the shortest path through a floor means fewer encounters and lower alert
- But the shortest path may skip reward rooms and priority targets
- Full-clearing a floor maximizes rewards but faces a fully alerted garrison by the end

---

## 6. Room Types

Each floor is a directed acyclic graph (DAG) of rooms with branching paths. The **critical path** (shortest route to stairs) requires 2-3 fights. A **full clear** involves 4-6 fights.

### 6.1 Room Definitions

| Room Type | Enemies? | Garrison Archetype | Reward | Notes |
|-----------|----------|-------------------|--------|-------|
| **Barracks** | Yes | Orc Vanguard, Shield Wall | Unit reward (TBD) | Heavy combat. High enemy count. |
| **Guard Post** | Yes | Chokepoint Guard, Undead Patrol | None (blocking room) | On critical path. Must clear to advance. |
| **Armory** | Yes | Shield Wall, Ranged Battery | Artifact reward (TBD) | Priority target — clearing weakens remaining garrison equipment. |
| **Command Post** | Yes | Command Post | +15 Mana, Spell reward (TBD) | Priority target — clearing removes leadership buffs from nearby rooms. |
| **Patrol Route** | Yes | Fast Response, Ambush Pack | Minor reward | Mobile enemies. May appear on flanking paths. |
| **Mage Tower** | Yes | Mage Tower | Spell reward (TBD) | High magic damage. Dangerous but valuable. |
| **Rest Room** | No | — | Heal all squads 20% missing HP, +5 Morale | Safe haven. No enemies. Off critical path. |
| **Stairs** | No | — | Floor transition | Always at the end of critical path. |

### 6.2 Priority Targets

Certain rooms have **garrison-wide effects** when cleared. The exact impact mechanics are **TBD**, but the design intent:

- **Armory:** Remaining garrison squads on this floor lose an equipment-related buff (specifics TBD)
- **Command Post:** Remaining garrison squads on this floor lose a leadership-related buff (specifics TBD)
- **Mage Tower:** Remaining magic-using garrison squads on this floor lose a magic-related buff (specifics TBD)

Priority targets create the core strategic decision: spend resources now assaulting a high-value target to weaken everything else, or conserve resources and brute-force the remaining encounters?

### 6.3 Floor Graph Structure

```
Floor Example (5 rooms, 2 branches):

    [Guard Post] ─── [Barracks] ──┐
         │                         ├── [Stairs]
         └─── [Rest Room] ── [Armory] ──┘

Critical path: Guard Post → Barracks → Stairs (2 fights)
Full clear: Guard Post → Rest Room → Armory → Barracks → Stairs (3 fights + rest)
```

Branching paths let the player choose between:
- **Speed:** Take the critical path, fewer fights, lower alert, but fewer rewards
- **Resources:** Detour to Rest Rooms for healing and morale recovery
- **Investment:** Clear priority targets to weaken the remaining garrison

---

## 7. Floor Progression

Configurable 3-5 floors. Each floor escalates both garrison strength and attrition pressure.

### 7.1 Floor Scaling

| Floor | Garrison Character | Encounter Count | Archetypes Available | Alert Cap |
|-------|-------------------|-----------------|---------------------|-----------|
| **1** | Outer defenses. Light patrols, basic compositions. | 3-4 rooms | Chokepoint Guard, Undead Patrol, Fast Response | 2 |
| **2** | Inner perimeter. Specialists appear. | 4-5 rooms | + Shield Wall, Ranged Battery, Ambush Pack | 3 |
| **3** | Core garrison. Priority targets appear. | 5-6 rooms | + Command Post, Mage Tower, Orc Vanguard | 3 |
| **4** | Elite zone. All archetypes, stronger compositions. | 4-5 rooms | All archetypes with +1 bonus stat levels | 3 |
| **5** | Commander's sanctum. Boss encounter. | 3-4 rooms | Elite compositions + boss squad | 3 |

### 7.2 Between-Floor Recovery

When the raid team descends to the next floor:

| Resource | Recovery |
|----------|----------|
| HP (deployed squads) | +10-15% of missing HP |
| HP (reserve squads) | +25-30% of missing HP |
| Mana | No natural recovery (only via Command Post reward) |
| Morale | +3 per squad (minor stabilization) |
| Dead units | No recovery (permanent for the run) |

### 7.3 Escalation Design

- **Early floors** (1-2): Light patrols, basic compositions. Teach the system. Player learns squad rotation, alert management, room selection. Attrition pressure is gentle — mistakes are survivable.
- **Mid floors** (2-3): Specialists appear. Priority targets become available. The player must choose between safe paths and high-value targets. Attrition pressure builds — every saved resource matters more.
- **Final floor** (4-5): Elites only. Boss encounter at the end. Every HP, every mana point, every living unit saved from earlier floors pays dividends. A well-managed raid arrives in fighting shape. A sloppy raid arrives crippled.

---

## 8. AI Behavior

Garrison defenders use the existing utility AI system (`mind/ai/`) with tuned scoring parameters. No new AI code — only configuration of existing scoring weights and threat layer evaluations.

### 8.1 AI Profile Concept

An `AIProfile` is a set of overrides to the existing `ActionEvaluator` scoring constants. Each garrison archetype uses a profile that matches its tactical identity.

```
AIProfile {
    ApproachMultiplier    map[UnitRole]float64  // Override default role approach weights
    AllyProximityWeight   float64               // Override +3 per adjacent ally
    AttackPriorityBonus   float64               // Added to base attack score (100.0)
    PositionalAnchor      *LogicalPosition       // If set, penalize movement away from this point
    AnchorWeight          float64               // Strength of anchor penalty
}
```

### 8.2 Garrison AI Profiles

#### Garrison Defender (default)
**Used by:** Chokepoint Guard, Shield Wall, Ranged Battery, Command Post
**Behavior:** Hold position, defend the room. Tanks anchor to their starting positions. Ranged units maintain distance. Support stays behind tanks.

| Override | Value | vs Default |
|----------|-------|------------|
| Tank ApproachMultiplier | 5.0 | 15.0 (less aggressive) |
| DPS ApproachMultiplier | 3.0 | 8.0 (less aggressive) |
| Support ApproachMultiplier | -8.0 | -5.0 (more retreating) |
| AllyProximityWeight | +5.0 | +3.0 (tighter formation) |
| PositionalAnchor | Room center | — |
| AnchorWeight | 10.0 | — |

**Result:** Garrison defenders cluster near their post. They engage targets that come to them but don't chase fleeing enemies far. Tanks hold the doorway while ranged units fire from safety.

#### Guard Post (stationary)
**Used by:** Undead Patrol on guard duty
**Behavior:** Minimal movement. Attack anything in range. Never leave the room.

| Override | Value | vs Default |
|----------|-------|------------|
| All ApproachMultipliers | 0.0 | Varied |
| PositionalAnchor | Exact spawn position | — |
| AnchorWeight | 25.0 | — (very strong) |

**Result:** Guards essentially don't move unless an enemy is adjacent. They attack if they can, otherwise they wait. Breaking through a guard post requires the player to come to the guards.

#### Reinforcement Rush (aggressive)
**Used by:** Fast Response, Ambush Pack, Orc Vanguard
**Behavior:** Charge the nearest enemy. Close distance immediately. Prioritize isolated targets.

| Override | Value | vs Default |
|----------|-------|------------|
| Tank ApproachMultiplier | 20.0 | 15.0 (more aggressive) |
| DPS ApproachMultiplier | 15.0 | 8.0 (much more aggressive) |
| Support ApproachMultiplier | 5.0 | -5.0 (pushes forward) |
| AllyProximityWeight | +1.0 | +3.0 (looser formation) |
| AttackPriorityBonus | +15.0 | 0 (attacks valued higher) |
| PositionalAnchor | nil | — (no anchor) |

**Result:** These squads rush the player. They don't hold formation — they spread out to surround and overwhelm. DPS units dive past tanks to reach squishy targets. Dangerous but exploitable with good positioning.

### 8.3 Implementation Notes

All profile overrides feed into the existing scoring pipeline in `mind/ai/action_evaluator.go`:
- `ApproachMultiplier` modifies the role-specific approach bonus in movement scoring
- `AllyProximityWeight` modifies the `+3 per adjacent ally` bonus
- `AttackPriorityBonus` adds to the base 100.0 attack score
- `PositionalAnchor` + `AnchorWeight` subtract from movement scores based on distance from anchor

The existing `RoleThreatWeights` from `mind/behavior/` (loaded from `aiconfig.json`) continue to control how threat layers influence movement. Profiles override the approach/positioning logic, not the threat evaluation.

---

## 9. The Double Resource Curve

The core design tension of the raid mode.

### 9.1 Player Curve (Sawtooth)

```
Strength
  100% ┤\
       │ \    /\
       │  \  /  \    /\
       │   \/    \  /  \
       │         \/    \
       │                \
    0% ┤─────────────────\──
       Floor 1  Floor 2  Floor 3  ...

       \ = combat (drops)
       / = between-encounter recovery (partial rise)
       Overall trend: downward
```

- **HP** drops in combat, partially recovers between fights (more for reserves)
- **Mana** drops with each spell, recovers only from Command Post rewards
- **Morale** drops from casualties, rises from victories and rest rooms — but casualties accumulate
- **Living units** only goes down — every death is permanent

### 9.2 Garrison Curve (Staircase)

```
Strength
  100% ┤────┐
       │    │
       │    └────┐
       │         │
       │         └────┐
       │              │
    0% ┤──────────────└──
       Enc 1  Enc 2  Enc 3  ...

       Each step = one garrison squad destroyed (permanent)
```

- Each killed squad is permanently gone
- Priority targets create **disproportionate drops** (clearing the Command Post weakens multiple remaining squads)
- Alert buffs partially offset the staircase — surviving squads get stronger as alert rises
- But no new squads appear — the garrison is finite

### 9.3 The Intersection

The raid succeeds when the garrison curve hits zero before the player curve does. The player's job is to:

1. **Steepen the garrison curve** — hit priority targets, fight efficiently, maximize kills per engagement
2. **Flatten the player curve** — rotate squads, use rest rooms, conserve mana, minimize casualties
3. **Read the curves** — know when to push (garrison is weak, player is strong) and when to rest or retreat (player is degrading faster)

---

## 10. Rewards

Players can gain three types of rewards during a raid. Exact placement, trigger mechanics, and specific rewards are **TBD**.

### 10.1 Reward Types

| Reward | System | How It Works |
|--------|--------|-------------|
| **Spells** | `tactical/spells/` | New spell ID added to `SpellBookData.SpellIDs` on the commander entity |
| **Units** | `tactical/squads/` | New unit entity created and added to a squad's 3x3 grid via `squadcreation.go` |
| **Artifacts** | `gear/` | New artifact added to `ArtifactInventoryData.OwnedArtifacts` on the player entity |

### 10.2 Reward Sources (TBD)

| Room Type | Possible Reward |
|-----------|----------------|
| Barracks | Unit (recruit a captured/freed soldier) |
| Armory | Artifact (loot from the armory) |
| Command Post | +15 Mana + Spell (captured intelligence / grimoire) |
| Mage Tower | Spell (arcane research materials) |
| Boss Room (final floor) | Major artifact + spell |

### 10.3 Design Intent

Rewards serve two purposes:
1. **Immediate:** Offset attrition. A new unit replaces a casualty. A new spell gives options. Mana recovery extends the commander's reach.
2. **Persistent:** Rewards kept after the raid (even on retreat) provide permanent progression for the player's overworld campaign.

---

## 11. Victory / Defeat / Retreat

### Victory
Clear the boss room on the final floor. All surviving squads, gained rewards (spells, units, artifacts), and any other loot are preserved.

### Defeat
Triggered when:
- All deployed squads are destroyed in combat, OR
- All squads reach morale 0 (combat ineffective)

On defeat, the raid ends. Killed garrison enemies **stay dead** — the garrison is permanently weakened for future attempts. Rewards gained before the final encounter are lost (TBD — may allow partial reward retention).

### Retreat
Available **between encounters** (not mid-combat). The player voluntarily ends the raid.

- Surviving squads are preserved with their current HP, morale, and unit roster
- Killed garrison enemies stay dead
- Alert resets to 0 on all floors
- Rewards gained so far are kept
- The player can attempt the raid again — with fewer garrison enemies but also fewer resources if squads were damaged

**Retreat has value.** A partial run that kills 60% of the garrison and retreats sets up a much easier second attempt. This prevents the "all or nothing" feel and gives struggling players a viable path forward.

---

## 12. Map Generation (Deferred)

Map generation design is deferred to the implementation phase. The following are architectural recommendations for when it is built.

### 12.1 Recommended Approach

- **Algorithm:** BSP (Binary Space Partitioning) via a new `gen_garrison.go` registered in the worldmap generator registry
- **Room assignment:** Based on room size, position in the BSP tree, and connectivity to other rooms
- **Per-room tactical terrain:** Cover objects, chokepoints, barricades placed based on the room's garrison archetype
- **Floor-specific parameters:** Early floors have more open rooms (easier), later floors have tighter corridors and more defensive terrain

### 12.2 Floor Graph Generation

Each floor generates a DAG of rooms:
1. BSP generates room geometry
2. Rooms are classified by size and position (large rooms = Barracks, small rooms = Guard Posts, etc.)
3. Connections form the DAG — critical path is the shortest route from entry to stairs
4. Branch paths lead to optional rooms (Rest Rooms, priority targets, reward rooms)
5. Each room is populated with a garrison archetype matching its type

### 12.3 Generator Registration

```go
// In gen_garrison.go
func init() {
    RegisterGenerator("garrison_raid", &GarrisonRaidGenerator{})
}
```

---

## Existing Systems Referenced

No changes needed to any of these systems. The raid mode orchestrates them.

| System | Package | Key Files | What It Provides |
|--------|---------|-----------|-----------------|
| Squad | `tactical/squads/` | `squadcomponents.go`, `squadcombat.go`, `squadcreation.go` | 3x3 grid, formations, morale (`SquadData.Morale`), roles, cover, combat, abilities |
| Combat | `tactical/combat/` | `turnmanager.go`, `combatfactionmanager.go`, `combatactionsystem.go` | Turn management, factions (`FactionData`), action states (`ActionStateData`), movement |
| Commander | `tactical/commander/` | `components.go`, `movement.go`, `roster.go`, `system.go` | Commander entity (`CommanderData`), mana (`ManaData`), squad roster, overworld turn state |
| Spells | `tactical/spells/` | `components.go`, `system.go` | Spell registry (loaded from JSON), mana costs, `ExecuteSpellCast`, damage + buff/debuff routing |
| Effects | `tactical/effects/` | `components.go`, `system.go` | `ActiveEffect` with duration, 8 stat types (`StatStrength` through `StatAttackRange`), `ApplyEffect`, `TickEffects` |
| Artifacts | `gear/` | `components.go`, `artifactcharges.go`, `artifactbehavior.go`, `artifactinventory.go` | 14 artifacts (8 minor, 6 major), charge system (`ArtifactChargeTracker`), behavior interface |
| AI | `mind/ai/` | `ai_controller.go`, `action_evaluator.go` | Utility AI, role-weighted scoring, `ActionContext`, `ScoredAction` pipeline |
| Threat | `mind/behavior/` | `threat_layers.go`, `threat_composite.go`, `threat_combat.go` | 3 threat layers (combat, support, positional), `CompositeThreatEvaluator`, `RoleThreatWeights` |
| Encounter | `mind/encounter/` | `encounter_service.go`, `types.go`, `encounter_resolution.go` | Encounter lifecycle (`ActiveEncounter`), spawn, resolution (`CombatExitReason`), history |
| Monsters | `assets/gamedata/` | `monsterdata.json` | 23 monster types with full attributes, roles, attack types, growth grades |

---

## TBD Items

Items marked for future design decisions:

| Item | Section | Notes |
|------|---------|-------|
| Priority target exact impact mechanics | 6.2 | What specific buffs/debuffs are applied when Armory/Command Post/Mage Tower are cleared |
| Reward specifics | 10.2 | Which spells, units, and artifacts are awarded at each room type |
| Defeat reward retention | 11 | Whether rewards gained before a defeat are kept or lost |
| HP recovery percentages | 2, 7.2 | Exact values for between-encounter and between-floor recovery |
| Morale thresholds and values | 3.4 | Exact morale change amounts may need tuning |
| Alert stat buff values | 5.1 | Exact +Armor/+Str/+Weapon values per alert tier |
| Floor room counts | 7.1 | Exact number of rooms per floor |
| Boss encounter design | 7.1 | What the final floor boss looks like (enhanced archetype? unique composition?) |
| Map generation | 12 | Full BSP generator implementation |
| Between-floor squad selection | 7.2 | Can the player leave damaged squads behind? What happens to them? |
| Retreat garrison recovery | 11 | Does the garrison recover any strength between raid attempts? |
