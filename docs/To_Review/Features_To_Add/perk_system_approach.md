# Perk System: Table-Driven Hook Functions (Approach A)

Implementation plan for TinkerRogue's perk/trait system using table-driven hook functions.

**Goal:** Minimize changes to existing code, leverage existing patterns, follow Go idioms.

**Design Principle:** Perks are **permanent conditional combat rules**. They do not give flat stat bonuses (minor artifacts do that), temporary buffs/debuffs (spells do that), or limited-use activated abilities (major artifacts do that). Every perk has an "if/when" clause that changes HOW combat works.

---

## Table of Contents

1. [Three-System Identity](#1-three-system-identity)
2. [Perk Design Philosophy](#2-perk-design-philosophy)
3. [Perk Tier Classification](#3-perk-tier-classification)
4. [Foundation](#4-foundation)
5. [Hook Types](#5-hook-types)
6. [Hook Registry](#6-hook-registry)
7. [Hook Runner Functions](#7-hook-runner-functions)
8. [Perk ID Collection](#8-perk-id-collection)
9. [Example Perk Behaviors](#9-example-perk-behaviors)
10. [Existing Code Changes](#10-existing-code-changes)
11. [Combat Integration Strategy](#11-combat-integration-strategy)
12. [Minimal Invasion Accounting](#12-minimal-invasion-accounting)
13. [Data-Driven JSON Schema](#13-data-driven-json-schema)
14. [Implementation Order](#14-implementation-order)
15. [Relationship to Artifact System](#15-relationship-to-artifact-system)
16. [Go Design Pattern Analysis](#16-go-design-pattern-analysis)
17. [Implementation Landmines](#17-implementation-landmines)
18. [Stacking Rules](#18-stacking-rules)
19. [Testing Strategy](#19-testing-strategy)

---

## 1. Three-System Identity

Each of TinkerRogue's three power systems answers a different tactical question:

| System | Question It Answers | Acquisition | Duration | Examples |
|--------|-------------------|-------------|----------|----------|
| **Spells** | "What do I actively do this turn?" | Squad leader type | Temporary (2-4 turns) | Fireball (damage), War Cry (+4 str for 3 turns), Weaken (-3 str for 2 turns) |
| **Artifacts** | "What are my numbers / what limited ability do I have?" | Found/purchased, transferable | Permanent stats / limited charges | Iron Bulwark (+2 armor), Twin Strike Banner (attack twice, once/battle) |
| **Perks** | "What special rules apply to how I fight?" | Earned through progression, permanent | Always-on, conditional | Reckless Assault (+30% damage but +20% damage taken), Vigilance (crits become normal hits) |

### Why This Separation Matters

The old perk design had ~50 perks, many of which overlapped with spells and artifacts:

- **Tier 1 stat perks** (Iron Constitution +2 HP, Hardened Armor +4 armor) overlapped with **minor artifacts** (Iron Bulwark +2 armor, Keen Edge Whetstone +2 weapon)
- **Temporary buff perks** overlapped with **buff spells** (both give temporary stat boosts)
- **Commander spell-modifying perks** (Efficient Casting, Spell Mastery) overlapped with spell system scaling

The redesigned perks occupy a unique design space: **conditional combat rule changes** that are always-on but only activate in specific tactical situations.

---

## 2. Perk Design Philosophy

### Core Principles

1. **Conditional, not flat** -- Every perk has an "if/when" clause. No unconditional stat bonuses.
2. **Rule-changing, not number-changing** -- Perks alter combat mechanics, not stat sheets.
3. **Always-on but situational** -- No activation cost, no charges, but only relevant in specific tactical contexts.
4. **Identity-defining** -- Perks make squads feel fundamentally different in how they fight.
5. **Earned, not equipped** -- Tied to squad progression. Cannot be transferred. Represents training/experience.

### What Perks Do NOT Do

- Give flat +X to any stat (that's what minor artifacts do)
- Apply temporary duration-based buffs or debuffs (that's what spells do)
- Have charges, mana costs, or activation buttons (that's what major artifacts do)
- Provide one-time burst effects (that's what spells do)

### What Perks Do NOT Reward

- **Normal good play.** Moving, focusing fire, flanking, and ganging up on enemies are basic tactics every player already does. Perks must not reward actions players would take regardless. A perk condition must create a NEW tactical decision or reward a COSTLY/UNUSUAL choice.

### What Perks DO

- Change targeting rules (Cleave, Precision Strike, Marked for Death)
- Modify damage conditionally with genuine trade-offs (Reckless Assault, Isolated Predator, Executioner's Instinct)
- Alter counterattack mechanics (Stalwart, Riposte)
- Create reactive combat events (Guardian Protocol, Disruption, Bloodlust, Grudge Bearer)
- Provide passive turn-based effects with conditions (Field Medic, Fortify)
- Override death/crit rules (Resolute, Vigilance)
- Reward costly sacrifices -- skipping turns, operating alone, taking damage first (Counterpunch, Deadshot's Patience, Isolated Predator)

---

## 3. Perk Tier Classification

24 perks across 2 tiers, classified by implementation complexity.

**Important design rule:** Perks must not reward actions players would already take as part of normal good play. Moving, focusing fire, flanking, and ganging up on enemies are basic tactics. A perk condition must create a NEW tactical decision or reward a COSTLY/UNUSUAL choice.

### Tier 1: Combat Conditioning (10 perks) -- Simple Conditionals

One "if" clause, one effect. Easy to understand, low implementation complexity.

| # | Perk | Roles | Mechanic | Hook Type |
|---|------|-------|----------|-----------|
| 1 | **Brace for Impact** | Tank | When defending (not attacking), all units gain +0.15 cover bonus | CoverModHook |
| 2 | **Reckless Assault** | DPS | +30% damage dealt, but +20% damage received until next turn | DamageModHook + state flag |
| 3 | **Stalwart** | Tank, Support | If squad did NOT move this turn, counterattacks deal 100% instead of 50% | CounterModHook |
| 4 | **Executioner's Instinct** | DPS | +25% crit chance vs squads with any unit below 30% HP | DamageModHook |
| 5 | **Shieldwall Discipline** | Tank | Per Tank unit in row 0: all squad units take 5% less damage (max 15%) | DamageModHook (defender) |
| 6 | **Isolated Predator** | DPS | +25% damage when no friendly squads within 3 tiles | DamageModHook |
| 7 | **Vigilance** | Tank, Support | Critical hits against this squad become normal hits | DamageModHook (defender) |
| 8 | **Field Medic** | Support | At round start, lowest-HP unit heals 10% max HP (no mana, no action) | TurnStartHook |
| 9 | **Opening Salvo** | DPS | +35% damage on squad's first attack of the combat only | DamageModHook + per-battle flag |
| 10 | **Last Line** | Tank, Support | When last friendly squad alive: +20% hit, dodge, and damage | DamageModHook |

**Why none overlap with spells/artifacts:**
- No flat stat bonuses (artifacts)
- No temporary durations (spells)
- Every entry has a conditional trigger that represents a genuine trade-off or unusual situation

### Tier 2: Combat Specialization (14 perks) -- Event Reactions & Targeting Changes

Multiple conditions, event reactions, state tracking, or targeting modifications.

| # | Perk | Roles | Mechanic | Hook Type |
|---|------|-------|----------|-----------|
| 11 | **Cleave** | DPS (melee) | Melee attacks also hit one unit in the row behind the target, but ALL damage is reduced by 30% | TargetOverrideHook + DamageModHook |
| 12 | **Riposte** | Tank, DPS | Counterattacks have no hit penalty (normally -20) | CounterModHook |
| 13 | **Disruption** | DPS, Support | Dealing damage reduces target squad's next attack by -15% this round | PostDamageHook |
| 14 | **Guardian Protocol** | Tank | When adjacent friendly squad is attacked, one Tank absorbs 25% of damage | DamageRedirectHook |
| 15 | **Overwatch** | DPS (ranged) | Skip attack to auto-attack at 75% damage next enemy that moves in range | TurnStartHook + MovementHook |
| 16 | **Adaptive Armor** | Tank | -10% damage from same attacker per hit (stacks to 30%, resets per round) | DamageModHook (defender) + state tracking |
| 17 | **Bloodlust** | DPS | Each unit kill this round: +15% damage on next attack (stacks, resets per round) | PostDamageHook + DamageModHook |
| 18 | **Fortify** | Tank, Support | +0.05 cover per consecutive turn not moving (max +0.15 after 3 turns, moving resets) | TurnStartHook + CoverModHook |
| 19 | **Precision Strike** | DPS | Highest-dex unit targets the lowest-HP enemy instead of normal targeting | TargetOverrideHook |
| 20 | **Resolute** | All | Unit survives at 1 HP if it had >50% HP at round start (once per unit per battle) | DeathOverrideHook |
| 21 | **Grudge Bearer** | DPS, Tank | +20% damage vs enemy squads that have damaged this squad (stacks to +40%). Must take hits first. | PostDamageHook (defender) + DamageModHook |
| 22 | **Counterpunch** | DPS, Tank | If attacked last turn AND did not attack last turn, next attack deals +40% damage | TurnStartHook + DamageModHook |
| 23 | **Marked for Death** | DPS, Support | Spend attack action to Mark an enemy instead of dealing damage. Marked enemy takes +25% from next friendly attack (then mark is consumed). | New action type + DamageModHook |
| 24 | **Deadshot's Patience** | DPS (ranged) | If squad did NOT move AND did NOT attack on its previous turn, next ranged attack gains +50% damage and +20 accuracy | TurnStartHook + DamageModHook |

---

## 3.1 Role Availability Summary

| Perk | Tank | DPS | Support |
|------|------|-----|---------|
| Brace for Impact | Y | | |
| Reckless Assault | | Y | |
| Stalwart | Y | | Y |
| Executioner's Instinct | | Y | |
| Shieldwall Discipline | Y | | |
| Isolated Predator | | Y | |
| Vigilance | Y | | Y |
| Field Medic | | | Y |
| Opening Salvo | | Y | |
| Last Line | Y | | Y |
| Cleave | | Y | |
| Riposte | Y | Y | |
| Disruption | | Y | Y |
| Guardian Protocol | Y | | |
| Overwatch | | Y | |
| Adaptive Armor | Y | | |
| Bloodlust | | Y | |
| Fortify | Y | | Y |
| Precision Strike | | Y | |
| Resolute | Y | Y | Y |
| Grudge Bearer | Y | Y | |
| Counterpunch | Y | Y | |
| Marked for Death | | Y | Y |
| Deadshot's Patience | | Y | |

**Tank**: 10 perks available (Tier 1: 4, Tier 2: 6)
**DPS**: 16 perks available (Tier 1: 5, Tier 2: 11)
**Support**: 7 perks available (Tier 1: 4, Tier 2: 3)

Perks are assigned at the **squad** level. Role restrictions mean the perk only activates for (or requires) units of the specified roles. A squad with mixed roles can equip any perk where at least one unit matches the required role.

---

## 3.2 Detailed Perk Descriptions

### Tier 1 Details

**1. Brace for Impact** (Tank)
- **Condition:** Squad is the defender in a combat exchange (being attacked, not initiating)
- **Effect:** +0.15 cover bonus to all units
- **Tactical choice:** Encourages letting enemies attack you rather than rushing in. Pairs with positioning to bait attacks.
- **Hook:** `CoverModHook` -- add cover bonus when squad is defending, not attacking.

**2. Reckless Assault** (DPS)
- **Condition:** Always active when attacking (trade-off perk)
- **Effect:** +30% damage dealt, but +20% damage received until this squad's next turn
- **Tactical choice:** Every attack makes the squad vulnerable. Against dangerous enemies, the defense penalty could get units killed. You might deliberately hold back attacks to avoid the vulnerability window. Creates a genuine aggression-vs-survival decision each turn.
- **Hook:** `DamageModHook` -- apply +30% on outgoing attacks. Set a state flag that applies +20% incoming damage modifier until next turn via `DamageModHook` (defender).

**3. Stalwart** (Tank, Support)
- **Condition:** Squad did NOT move this turn
- **Effect:** Counterattacks deal 100% damage instead of 50%
- **Tactical choice:** Hold position for full-power counters, or reposition and lose the bonus?
- **Hook:** `CounterModHook` -- override `counterattackDamageMultiplier` to 1.0 if squad did not move.

**4. Executioner's Instinct** (DPS)
- **Condition:** Target squad has any unit below 30% HP
- **Effect:** +25% crit chance on all attacks
- **Tactical choice:** Focus weakened squads to exploit the bonus, or spread damage?
- **Hook:** `DamageModHook` -- scan target squad units' HP before crit roll.

**5. Shieldwall Discipline** (Tank)
- **Condition:** Tank units present in front row (row 0)
- **Effect:** Per Tank in row 0, all squad units take 5% less damage (max 15%)
- **Tactical choice:** Stack tanks in front for max defense, or spread roles for flexibility?
- **Hook:** `DamageModHook` (defender) -- count alive tanks in row 0, apply percentage reduction.

**6. Isolated Predator** (DPS)
- **Condition:** No friendly squads within 3 tiles of this squad
- **Effect:** +25% damage on all attacks
- **Tactical choice:** Normal good play is keeping squads close for mutual support, combined attacks, and overlapping threat zones. This perk rewards the opposite -- operating alone, cut off from help. An isolated squad can be surrounded and focused down with no defensive support. You give up safety and coordination for raw damage.
- **Hook:** `DamageModHook` -- spatial query for friendly squads within radius 3 at time of attack.

**7. Vigilance** (Tank, Support)
- **Condition:** Always active (rule negation)
- **Effect:** Critical hits against this squad become normal hits
- **Tactical choice:** Protects against spike damage. Valuable against high-dex enemies.
- **Hook:** `DamageModHook` (defender) -- set `SkipCrit = true`.

**8. Field Medic** (Support)
- **Condition:** Round start (automatic)
- **Effect:** Lowest-HP unit heals 10% of its max HP
- **Tactical choice:** Extends squad endurance over long fights without spending mana.
- **Hook:** `TurnStartHook` -- find lowest HP unit in squad, apply heal.

**9. Opening Salvo** (DPS)
- **Condition:** Squad's very first attack of the entire combat (once per battle)
- **Effect:** +35% damage on that attack only. No bonus on any subsequent attacks.
- **Tactical choice:** Players must decide WHAT to spend their one big hit on. Use it early on a high-value target and risk overkill? Wait for a critical moment, but that means delaying your first attack (a real cost). Makes target selection on turn 1 a high-stakes decision. The bonus is gone forever after the first attack.
- **Hook:** `DamageModHook` -- check per-battle flag `HasAttackedThisCombat`; bonus applies only when false, then set to true permanently.

**10. Last Line** (Tank, Support)
- **Condition:** This is the last surviving friendly squad
- **Effect:** +20% hit, +20% dodge, +20% damage
- **Tactical choice:** Safety net that makes your last squad dangerous rather than helpless.
- **Hook:** `DamageModHook` -- check if only one friendly squad remains.

### Tier 2 Details

**11. Cleave** (DPS, melee only)
- **Condition:** Attacker uses MeleeRow attack type
- **Effect:** Hits the primary target AND one unit in the row behind the target, but ALL damage from this attack (to both targets) is reduced by 30%
- **Tactical choice:** Without Cleave: 100% damage to 1 target. With Cleave: 70% damage to 2 targets. Against high-armor targets this is WORSE because armor applies twice and each hit is weaker. Against clusters of low-armor units, it's better. The player may wish they didn't have Cleave in some fights.
- **Hook:** `TargetOverrideHook` -- add spillover targets from adjacent row. `DamageModHook` -- apply 0.7 multiplier when Cleave is active.

**12. Riposte** (Tank, DPS)
- **Condition:** During counterattack phase
- **Effect:** Counterattacks have no hit penalty (normally -20)
- **Tactical choice:** Makes this squad dangerous to attack. Enemies might avoid engaging.
- **Note:** Different from Stalwart (which removes the *damage* penalty when stationary). Riposte is about *accuracy*.
- **Hook:** `CounterModHook` -- set `HitPenalty` to 0.

**13. Disruption** (DPS, Support)
- **Condition:** This squad successfully deals damage
- **Effect:** Target squad's next attack this round deals -15% damage
- **Tactical choice:** Attack dangerous enemies first to weaken their return fire.
- **Hook:** `PostDamageHook` -- apply round-scoped damage reduction flag to target squad.

**14. Guardian Protocol** (Tank)
- **Condition:** Adjacent friendly squad is attacked
- **Effect:** One Tank unit absorbs 25% of the damage
- **Tactical choice:** Position tank squads adjacent to fragile squads. The tank takes wear over time.
- **Hook:** `DamageRedirectHook` -- intercept before `recordDamageToUnit`, split damage.

**15. Overwatch** (DPS, ranged only)
- **Condition:** Squad does not attack this turn
- **Effect:** Next enemy that moves within attack range takes a free attack at 75% damage
- **Tactical choice:** Attack now for guaranteed damage, or set up overwatch to punish movement?
- **Hook:** New state flag in `ActionStateData` + check in `CombatMovementSystem` when enemy moves.

**16. Adaptive Armor** (Tank)
- **Condition:** Hit by the same squad repeatedly
- **Effect:** -10% damage per hit from the same attacker (stacks to 30%, resets each round)
- **Tactical choice:** Enemies should spread attacks; players should funnel attacks into the adaptive tank.
- **Hook:** `DamageModHook` (defender) + `PerkRoundState.AttackedBy` tracking.

**17. Bloodlust** (DPS)
- **Condition:** Unit kills in this round
- **Effect:** +15% damage on next attack per kill (stacks, resets each round)
- **Tactical choice:** Sequence attacks to maximize kill chains. Finish off wounded units first.
- **Hook:** `PostDamageHook` on kill + `DamageModHook` reads `PerkRoundState.KillsThisRound`.

**18. Fortify** (Tank, Support)
- **Condition:** Squad has not moved for consecutive turns
- **Effect:** +0.05 cover per turn stationary (max +0.15 after 3 turns, moving resets)
- **Tactical choice:** Hold a chokepoint for increasing defense, or reposition and lose the bonus?
- **Hook:** `TurnStartHook` updates `PerkRoundState.TurnsStationary` + `CoverModHook` reads it.

**19. Precision Strike** (DPS)
- **Condition:** During target selection
- **Effect:** Highest-dex DPS unit targets the lowest-HP enemy instead of normal targeting
- **Tactical choice:** Guarantees finishing blows on weakened units.
- **Hook:** `TargetOverrideHook` -- for one unit, replace target with lowest-HP enemy.

**20. Resolute** (All roles)
- **Condition:** Unit would be killed AND had >50% HP at round start
- **Effect:** Survives at 1 HP (once per unit per battle)
- **Tactical choice:** Keeps squads intact against burst damage. One-time use prevents abuse.
- **Hook:** `DeathOverrideHook` -- check `PerkRoundState.RoundStartHP` snapshot, set HP to 1.

**21. Grudge Bearer** (DPS, Tank)
- **Condition:** This squad has been damaged by the target enemy squad previously in this combat
- **Effect:** +20% damage against that specific enemy squad (stacks to +40% from multiple hits received). Only triggers from damage received, not from attacking.
- **Tactical choice:** The bonus only applies against enemies who have already hit you. You cannot alpha-strike a fresh target with the bonus -- you must absorb punishment first. It rewards staying engaged with a dangerous enemy rather than retreating. Also, your ideal target (the one with grudge stacks) might not be the strategically optimal target -- creating a tension between following the bonus and making the best tactical move.
- **Hook:** `PostDamageHook` (defender) -- record source squad ID and increment grudge counter. `DamageModHook` -- check grudge tracker against current target's squad ID.

**22. Counterpunch** (DPS, Tank)
- **Condition:** Squad was attacked during the previous enemy turn AND did not attack on its own previous turn
- **Effect:** Next attack deals +40% damage
- **Tactical choice:** You must spend an entire turn NOT attacking. In a tactical RPG, skipping your offensive turn is a major cost -- the enemy gets a free round to act without pressure. You are banking on a future big hit. The player must judge whether losing a full turn of attacks is worth one enhanced strike. Against fast enemies or in close fights, the tempo loss could be fatal.
- **Hook:** `TurnStartHook` -- check previous turn state: was attacked AND did not use attack action; set flag. `DamageModHook` -- consume flag for damage bonus.

**23. Marked for Death** (DPS, Support)
- **Condition:** Squad spends its attack action to Mark an enemy instead of dealing damage
- **Effect:** Marked enemy takes +25% damage from the next friendly attack (any squad). The mark is then consumed. The marking squad deals NO damage this turn.
- **Tactical choice:** The marking squad sacrifices its entire attack. If no other squad follows up, the mark is wasted. If the marked target dies before the follow-up, the mark is wasted. Requires coordination and gives up guaranteed damage for potential amplified damage from an ally. Is it better to just attack normally, or sacrifice your turn to boost someone else?
- **Hook:** New action type "Mark" that applies a component to target enemy squad (no damage). `DamageModHook` -- check if target has Mark component, apply bonus, remove Mark.

**24. Deadshot's Patience** (DPS, ranged only)
- **Condition:** Squad did NOT move AND did NOT attack on its previous turn (completely idle for one full turn)
- **Effect:** Next ranged attack gains +50% damage and +20 accuracy
- **Tactical choice:** Skipping an ENTIRE turn is an enormous cost. No movement, no attacks, no actions. The squad is a sitting duck for one full round. Enemy squads can reposition, close distance, or focus fire on you. In exchange, the next shot is devastating. This is a sniper's patience fantasy -- but the tempo loss is real and dangerous.
- **Hook:** `TurnStartHook` -- check previous turn: no move AND no attack; set "patient" flag. `DamageModHook` -- consume flag for damage/accuracy bonus (require ranged weapon check).

---

## 4. Foundation

### 4.1 Package Structure

```
tactical/perks/
    init.go              -- Subsystem registration (RegisterSubsystem pattern)
    components.go        -- ECS component definitions (pure data)
    perkdefinition.go    -- PerkDefinition struct, JSON loading
    registry.go          -- PerkRegistry global map, LoadPerkDefinitions()
    hooks.go             -- Hook function type definitions
    hook_registry.go     -- PerkHooks struct, RegisterPerkHooks, GetPerkHooks
    behaviors.go         -- Individual perk behavior implementations
    system.go            -- ApplyPerks, RemovePerks, equip/unequip
    queries.go           -- HasPerk, GetEquippedPerks, getActivePerkIDs, hook runners
```

### 4.2 PerkDefinition (Data-Driven, mirrors SpellDefinition)

Follows the exact pattern from `templates/spelldefinitions.go:34-47`:

```go
// tactical/perks/perkdefinition.go

type PerkTier int
const (
    PerkTierConditioning   PerkTier = iota // Tier 1: Simple conditionals
    PerkTierSpecialization                  // Tier 2: Event reactions, targeting, state tracking
)

type PerkCategory int
const (
    CategoryOffense   PerkCategory = iota // Damage-oriented perks
    CategoryDefense                        // Damage reduction, cover perks
    CategoryTactical                       // Targeting, positioning perks
    CategoryReactive                       // Event-triggered perks
    CategoryDoctrine                       // Squad-wide behavioral changes
)

// PerkDefinition is a static blueprint loaded from JSON.
// Analogous to SpellDefinition in templates/spelldefinitions.go.
type PerkDefinition struct {
    ID            string       `json:"id"`
    Name          string       `json:"name"`
    Description   string       `json:"description"`
    Tier          PerkTier     `json:"tier"`
    Category      PerkCategory `json:"category"`
    Roles         []string     `json:"roles"`          // ["Tank"], ["DPS", "Support"], etc.
    ExclusiveWith []string     `json:"exclusiveWith"`
    UnlockCost    int          `json:"unlockCost"`

    // Behavioral hook key (all perks use hooks, no stat-only perks)
    BehaviorID string         `json:"behaviorId"`
    Params     map[string]any `json:"params,omitempty"`
}
```

### 4.3 ECS Components

```go
// tactical/perks/components.go

// PerkSlotData stores equipped perks on a squad entity.
// Number of available slots scales with squad progression.
type PerkSlotData struct {
    PerkIDs []string // Equipped perk IDs (max based on squad level)
}

// PerkRoundState tracks per-round state needed by conditional perks.
// Reset at round start, except fields marked as per-battle.
type PerkRoundState struct {
    // Per-turn state (resets each turn)
    MovedThisTurn          bool                       // For Stalwart, Fortify
    AttackedThisTurn       bool                       // For Reckless Assault vulnerability window
    RecklessVulnerable     bool                       // For Reckless Assault (+20% damage received)

    // Per-round state (resets each round)
    AttackedBy             map[ecs.EntityID]int        // For Adaptive Armor (attacker -> hit count)
    KillsThisRound         int                        // For Bloodlust
    DisruptionTargets      map[ecs.EntityID]bool       // For Disruption (squads debuffed this round)
    OverwatchActive        bool                       // For Overwatch

    // Per-round but persists across rounds
    TurnsStationary        int                        // For Fortify (resets on movement)

    // Per-battle state (persists entire combat)
    HasAttackedThisCombat  bool                       // For Opening Salvo (one-time bonus)
    ResoluteUsed           map[ecs.EntityID]bool       // For Resolute (unit -> used flag)
    RoundStartHP           map[ecs.EntityID]int        // For Resolute (updated each round, not reset)
    GrudgeStacks           map[ecs.EntityID]int        // For Grudge Bearer (enemy squad -> damage count, persists)
    WasAttackedLastTurn    bool                       // For Counterpunch (was attacked previous turn)
    DidNotAttackLastTurn   bool                       // For Counterpunch (did not attack previous turn)
    CounterpunchReady      bool                       // For Counterpunch (both conditions met)
    WasIdleLastTurn        bool                       // For Deadshot's Patience (no move AND no attack last turn)
    DeadshotReady          bool                       // For Deadshot's Patience (ready to fire)
    MarkedSquad            ecs.EntityID               // For Marked for Death (which enemy is marked, 0 = none)
}

// PerkUnlockData tracks which perks have been unlocked for a commander/roster.
type PerkUnlockData struct {
    UnlockedPerks map[string]bool // Perk IDs that have been unlocked
    PerkPoints    int             // Available points to spend
}

var (
    PerkSlotComponent      *ecs.Component
    PerkRoundStateComponent *ecs.Component
    PerkUnlockComponent    *ecs.Component
)
```

### 4.4 Subsystem Registration

Follows the `init()` pattern used by all existing subsystems:

```go
// tactical/perks/init.go
func init() {
    common.RegisterSubsystem(func(em *common.EntityManager) {
        PerkSlotComponent = em.World.NewComponent()
        PerkRoundStateComponent = em.World.NewComponent()
        PerkUnlockComponent = em.World.NewComponent()
    })
}
```

### 4.5 Perk Registry

Mirrors `templates/spelldefinitions.go:50-55`:

```go
// tactical/perks/registry.go

var PerkRegistry = make(map[string]*PerkDefinition)

func GetPerkDefinition(id string) *PerkDefinition {
    return PerkRegistry[id]
}

func LoadPerkDefinitions() {
    // Read from assets/gamedata/perkdata.json
    // Parse JSON, populate PerkRegistry
    // Validate: no duplicate IDs, exclusive pairs symmetric, roles valid
}
```

### 4.6 Required Change to Effects System (1 line)

```go
// In tactical/effects/components.go:26-29, add SourcePerk:
const (
    SourceSpell   EffectSource = iota
    SourceAbility
    SourcePerk    // NEW -- 1 line addition
    SourceItem
)
```

---

## 5. Hook Types

Eight typed function signatures covering all 24 perks.

```go
// tactical/perks/hooks.go

// DamageModHook modifies damage modifiers before calculation.
// Called inside calculateDamage() for the attacking unit.
type DamageModHook func(
    attackerID, defenderID ecs.EntityID,
    attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers,
    roundState *PerkRoundState,
    manager *common.EntityManager,
)

// TargetOverrideHook overrides target selection.
// Returns nil to use default targeting.
type TargetOverrideHook func(
    attackerID, defenderSquadID ecs.EntityID,
    defaultTargets []ecs.EntityID,
    manager *common.EntityManager,
) []ecs.EntityID

// CounterModHook modifies counterattack behavior.
// Return skipCounter=true to suppress counterattack entirely.
type CounterModHook func(
    defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers,
    roundState *PerkRoundState,
    manager *common.EntityManager,
) (skipCounter bool)

// PostDamageHook runs after damage is recorded for a single attack.
type PostDamageHook func(
    attackerID, defenderID ecs.EntityID,
    attackerSquadID, defenderSquadID ecs.EntityID,
    damageDealt int, wasKill bool,
    roundState *PerkRoundState,
    manager *common.EntityManager,
)

// TurnStartHook runs at the start of a squad's turn.
type TurnStartHook func(
    squadID ecs.EntityID,
    roundNumber int,
    roundState *PerkRoundState,
    manager *common.EntityManager,
)

// CoverModHook modifies cover calculation for a defender.
type CoverModHook func(
    attackerID, defenderID ecs.EntityID,
    coverBreakdown *squads.CoverBreakdown,
    roundState *PerkRoundState,
    manager *common.EntityManager,
)

// DamageRedirectHook intercepts damage before recordDamageToUnit.
// Returns reduced damage for original target, plus a redirect target and amount.
// Required for Guardian Protocol perk.
type DamageRedirectHook func(
    defenderID ecs.EntityID,
    defenderSquadID ecs.EntityID,
    damageAmount int,
    manager *common.EntityManager,
) (reducedDamage int, redirectTargetID ecs.EntityID, redirectAmount int)

// DeathOverrideHook intercepts lethal damage.
// Returns true if death should be prevented (unit survives at 1 HP).
// Required for Resolute perk.
type DeathOverrideHook func(
    unitID ecs.EntityID,
    squadID ecs.EntityID,
    roundState *PerkRoundState,
    manager *common.EntityManager,
) (preventDeath bool)
```

---

## 6. Hook Registry

```go
// tactical/perks/hook_registry.go

// PerkHooks collects all hooks for a single perk.
// A perk only populates the hooks it needs -- nil slots are skipped.
type PerkHooks struct {
    DamageMod       DamageModHook
    TargetOverride  TargetOverrideHook
    CounterMod      CounterModHook
    PostDamage      PostDamageHook
    TurnStart       TurnStartHook
    CoverMod        CoverModHook
    DamageRedirect  DamageRedirectHook
    DeathOverride   DeathOverrideHook
}

var hookRegistry = map[string]*PerkHooks{}

func RegisterPerkHooks(perkID string, hooks *PerkHooks) {
    hookRegistry[perkID] = hooks
}

func GetPerkHooks(perkID string) *PerkHooks {
    return hookRegistry[perkID]
}

func init() {
    // Tier 1: Combat Conditioning
    RegisterPerkHooks("brace_for_impact", &PerkHooks{CoverMod: braceForImpactCoverMod})
    RegisterPerkHooks("reckless_assault", &PerkHooks{DamageMod: recklessAssaultDamageMod})
    RegisterPerkHooks("stalwart", &PerkHooks{CounterMod: stalwartCounterMod})
    RegisterPerkHooks("executioners_instinct", &PerkHooks{DamageMod: executionerDamageMod})
    RegisterPerkHooks("shieldwall_discipline", &PerkHooks{DamageMod: shieldwallDamageMod})
    RegisterPerkHooks("isolated_predator", &PerkHooks{DamageMod: isolatedPredatorDamageMod})
    RegisterPerkHooks("vigilance", &PerkHooks{DamageMod: vigilanceDamageMod})
    RegisterPerkHooks("field_medic", &PerkHooks{TurnStart: fieldMedicTurnStart})
    RegisterPerkHooks("opening_salvo", &PerkHooks{DamageMod: openingSalvoDamageMod})
    RegisterPerkHooks("last_line", &PerkHooks{DamageMod: lastLineDamageMod})

    // Tier 2: Combat Specialization
    RegisterPerkHooks("cleave", &PerkHooks{
        TargetOverride: cleaveTargetOverride,
        DamageMod:      cleaveDamageMod, // -30% damage penalty
    })
    RegisterPerkHooks("riposte", &PerkHooks{CounterMod: riposteCounterMod})
    RegisterPerkHooks("disruption", &PerkHooks{PostDamage: disruptionPostDamage})
    RegisterPerkHooks("guardian_protocol", &PerkHooks{DamageRedirect: guardianDamageRedirect})
    RegisterPerkHooks("overwatch", &PerkHooks{TurnStart: overwatchTurnStart})
    RegisterPerkHooks("adaptive_armor", &PerkHooks{DamageMod: adaptiveArmorDamageMod})
    RegisterPerkHooks("bloodlust", &PerkHooks{
        PostDamage: bloodlustPostDamage,
        DamageMod:  bloodlustDamageMod,
    })
    RegisterPerkHooks("fortify", &PerkHooks{
        TurnStart: fortifyTurnStart,
        CoverMod:  fortifyCoverMod,
    })
    RegisterPerkHooks("precision_strike", &PerkHooks{TargetOverride: precisionStrikeTargetOverride})
    RegisterPerkHooks("resolute", &PerkHooks{
        TurnStart:     resoluteTurnStart,
        DeathOverride: resoluteDeathOverride,
    })
    RegisterPerkHooks("grudge_bearer", &PerkHooks{
        PostDamage: grudgeBearerPostDamage,  // Track damage received from enemy squads
        DamageMod:  grudgeBearerDamageMod,   // Apply bonus vs grudge targets
    })
    RegisterPerkHooks("counterpunch", &PerkHooks{
        TurnStart: counterpunchTurnStart,    // Check if conditions met last turn
        DamageMod: counterpunchDamageMod,    // Apply +40% bonus on next attack
    })
    RegisterPerkHooks("marked_for_death", &PerkHooks{
        DamageMod: markedForDeathDamageMod,  // Apply +25% bonus to marked target
    })
    RegisterPerkHooks("deadshots_patience", &PerkHooks{
        TurnStart: deadshotTurnStart,        // Check if completely idle last turn
        DamageMod: deadshotDamageMod,        // Apply +50% damage and +20 accuracy
    })
}
```

---

## 7. Hook Runner Functions

```go
// tactical/perks/queries.go

// RunDamageModHooks runs all DamageMod hooks for an attacker's perks.
func RunDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.DamageMod != nil {
            hooks.DamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, manager)
        }
    }
}

// RunDefenderDamageModHooks runs hooks for the DEFENDER's perks
// (e.g., Shieldwall Discipline, Vigilance, Adaptive Armor).
func RunDefenderDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.DamageMod != nil {
            hooks.DamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, manager)
        }
    }
}

// RunTargetOverrideHooks applies target overrides from attacker perks.
func RunTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
    targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    attackerSquadID := getSquadIDForUnit(attackerID, manager)
    for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.TargetOverride != nil {
            targets = hooks.TargetOverride(attackerID, defenderSquadID, targets, manager)
        }
    }
    return targets
}

// RunDefenderTargetOverrideHooks applies target overrides from defender perks
// (e.g., Rearguard Doctrine protecting row 2).
func RunDefenderTargetOverrideHooks(attackerID, defenderSquadID ecs.EntityID,
    targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.TargetOverride != nil {
            targets = hooks.TargetOverride(attackerID, defenderSquadID, targets, manager)
        }
    }
    return targets
}

// RunCounterModHooks checks if counterattack should be suppressed or modified.
func RunCounterModHooks(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) bool {
    defenderSquadID := getSquadIDForUnit(defenderID, manager)
    for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.CounterMod != nil {
            if hooks.CounterMod(defenderID, attackerID, modifiers, roundState, manager) {
                return true // Skip counter
            }
        }
    }
    return false
}

// RunPostDamageHooks runs post-damage hooks for the attacker.
func RunPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    damageDealt int, wasKill bool, roundState *PerkRoundState, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.PostDamage != nil {
            hooks.PostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damageDealt, wasKill, roundState, manager)
        }
    }
}

// RunTurnStartHooks runs turn-start hooks for a squad.
func RunTurnStartHooks(squadID ecs.EntityID, roundNumber int,
    roundState *PerkRoundState, manager *common.EntityManager) {
    for _, perkID := range getActivePerkIDs(squadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.TurnStart != nil {
            hooks.TurnStart(squadID, roundNumber, roundState, manager)
        }
    }
}

// RunCoverModHooks runs cover modification hooks.
func RunCoverModHooks(attackerID, defenderID ecs.EntityID,
    coverBreakdown *squads.CoverBreakdown, roundState *PerkRoundState, manager *common.EntityManager) {
    attackerSquadID := getSquadIDForUnit(attackerID, manager)
    defenderSquadID := getSquadIDForUnit(defenderID, manager)
    // Check attacker perks
    for _, perkID := range getActivePerkIDs(attackerSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.CoverMod != nil {
            hooks.CoverMod(attackerID, defenderID, coverBreakdown, roundState, manager)
        }
    }
    // Check defender perks (Brace for Impact, Fortify)
    for _, perkID := range getActivePerkIDs(defenderSquadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.CoverMod != nil {
            hooks.CoverMod(attackerID, defenderID, coverBreakdown, roundState, manager)
        }
    }
}

// RunDeathOverrideHooks checks if lethal damage should be prevented.
func RunDeathOverrideHooks(unitID, squadID ecs.EntityID,
    roundState *PerkRoundState, manager *common.EntityManager) bool {
    for _, perkID := range getActivePerkIDs(squadID, manager) {
        hooks := GetPerkHooks(perkID)
        if hooks != nil && hooks.DeathOverride != nil {
            if hooks.DeathOverride(unitID, squadID, roundState, manager) {
                return true // Prevent death
            }
        }
    }
    return false
}
```

---

## 8. Perk ID Collection

```go
// getActivePerkIDs returns all perk IDs equipped on a squad.
// Since perks are now squad-level only, this is simpler than the old unit+squad lookup.
func getActivePerkIDs(squadID ecs.EntityID, manager *common.EntityManager) []string {
    if data := common.GetComponentTypeByID[*PerkSlotData](
        manager, squadID, PerkSlotComponent,
    ); data != nil {
        return data.PerkIDs
    }
    return nil
}

// HasPerk checks if a squad has a specific perk equipped.
func HasPerk(squadID ecs.EntityID, perkID string, manager *common.EntityManager) bool {
    for _, id := range getActivePerkIDs(squadID, manager) {
        if id == perkID {
            return true
        }
    }
    return false
}

// getSquadIDForUnit returns the parent squad ID for a unit.
func getSquadIDForUnit(unitID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
    if memberData := common.GetComponentTypeByID[*squads.SquadMemberData](
        manager, unitID, squads.SquadMemberComponent,
    ); memberData != nil {
        return memberData.SquadID
    }
    return 0
}
```

---

## 9. Example Perk Behaviors

```go
// tactical/perks/behaviors.go

// --- Tier 1 Examples ---

// Reckless Assault: +30% damage dealt, +20% damage received until next turn
func recklessAssaultDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if modifiers.IsCounterattack {
        return // Only applies to initiating attacks
    }
    // Outgoing: +30% damage
    modifiers.DamageMultiplier *= 1.3
    // Set vulnerability flag -- incoming damage +20% until next turn
    roundState.RecklessVulnerable = true
}

// Stalwart: Full-damage counters if squad did NOT move
func stalwartCounterMod(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) bool {
    if !roundState.MovedThisTurn {
        modifiers.DamageMultiplier = 1.0 // Override 0.5 default
    }
    return false // Don't skip counter
}

// Isolated Predator: +25% damage when no friendly squads within 3 tiles
func isolatedPredatorDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    // Check spatial query for friendly squads within radius 3
    squadPos := common.GetComponentTypeByID[*coords.LogicalPosition](
        manager, attackerSquadID, common.PositionComponent,
    )
    if squadPos == nil { return }

    // Query friendly squads and check distance
    friendlyNearby := false
    // ... iterate friendly squads, check Manhattan/Euclidean distance <= 3 ...
    // (Full implementation depends on combat cache for squad positions)

    if !friendlyNearby {
        modifiers.DamageMultiplier *= 1.25
    }
}

// Opening Salvo: +35% damage on first attack of combat only
func openingSalvoDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if !roundState.HasAttackedThisCombat {
        modifiers.DamageMultiplier *= 1.35
        roundState.HasAttackedThisCombat = true
    }
}

// Vigilance: Crits become normal hits
func vigilanceDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    modifiers.SkipCrit = true
}

// Field Medic: Heal lowest-HP unit at turn start
func fieldMedicTurnStart(squadID ecs.EntityID, roundNumber int,
    roundState *PerkRoundState, manager *common.EntityManager) {
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
    var lowestID ecs.EntityID
    lowestHP := 999999
    for _, uid := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](
            manager, uid, common.AttributeComponent,
        )
        if attr != nil && attr.CurrentHealth > 0 && attr.CurrentHealth < lowestHP {
            lowestHP = attr.CurrentHealth
            lowestID = uid
        }
    }
    if lowestID != 0 {
        attr := common.GetComponentTypeByID[*common.Attributes](
            manager, lowestID, common.AttributeComponent,
        )
        if attr != nil {
            healAmount := attr.MaxHealth / 10 // 10% max HP
            if healAmount < 1 { healAmount = 1 }
            attr.CurrentHealth += healAmount
            if attr.CurrentHealth > attr.MaxHealth {
                attr.CurrentHealth = attr.MaxHealth
            }
        }
    }
}

// --- Tier 2 Examples ---

// Riposte: Counterattacks have no hit penalty
func riposteCounterMod(defenderID, attackerID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) bool {
    modifiers.HitPenalty = 0 // Override -20 default
    return false             // Don't skip counter
}

// Cleave: Hit target row + row behind, but -30% damage to ALL targets
func cleaveTargetOverride(attackerID, defenderSquadID ecs.EntityID,
    defaultTargets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
    targetData := common.GetComponentTypeByID[*squads.TargetRowData](
        manager, attackerID, squads.TargetRowComponent,
    )
    if targetData == nil || targetData.AttackType != squads.AttackTypeMeleeRow {
        return defaultTargets
    }

    if len(defaultTargets) == 0 { return defaultTargets }
    pos := common.GetComponentTypeByID[*squads.GridPositionData](
        manager, defaultTargets[0], squads.GridPositionComponent,
    )
    if pos == nil { return defaultTargets }

    nextRow := pos.AnchorRow + 1
    if nextRow <= 2 {
        extraTargets := squads.GetUnitsInRow(defenderSquadID, nextRow, manager)
        return append(defaultTargets, extraTargets...)
    }
    return defaultTargets
}

func cleaveDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    // Apply -30% damage penalty when Cleave is active
    // This makes Cleave a trade-off: more targets but less damage per target
    modifiers.DamageMultiplier *= 0.7
}

// Bloodlust: Track kills and boost damage
func bloodlustPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    damageDealt int, wasKill bool, roundState *PerkRoundState, manager *common.EntityManager) {
    if wasKill {
        roundState.KillsThisRound++
    }
}

func bloodlustDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.KillsThisRound > 0 {
        bonus := 1.0 + float64(roundState.KillsThisRound)*0.15
        modifiers.DamageMultiplier *= bonus
    }
}

// Grudge Bearer: Track damage received, bonus vs enemies who hurt you
func grudgeBearerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    damageDealt int, wasKill bool, roundState *PerkRoundState, manager *common.EntityManager) {
    // This runs as a DEFENDER hook -- track who attacked us
    if roundState.GrudgeStacks == nil {
        roundState.GrudgeStacks = make(map[ecs.EntityID]int)
    }
    current := roundState.GrudgeStacks[attackerSquadID]
    if current < 2 { // Cap at +40% (2 stacks * 20%)
        roundState.GrudgeStacks[attackerSquadID] = current + 1
    }
}

func grudgeBearerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.GrudgeStacks != nil {
        stacks := roundState.GrudgeStacks[defenderSquadID]
        if stacks > 0 {
            bonus := 1.0 + float64(stacks)*0.20
            modifiers.DamageMultiplier *= bonus
        }
    }
}

// Counterpunch: +40% damage if attacked last turn AND did not attack last turn
func counterpunchTurnStart(squadID ecs.EntityID, roundNumber int,
    roundState *PerkRoundState, manager *common.EntityManager) {
    // Check previous turn state and set ready flag
    if roundState.WasAttackedLastTurn && roundState.DidNotAttackLastTurn {
        roundState.CounterpunchReady = true
    } else {
        roundState.CounterpunchReady = false
    }
}

func counterpunchDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.CounterpunchReady {
        modifiers.DamageMultiplier *= 1.4
        roundState.CounterpunchReady = false // Consumed on first attack
    }
}

// Deadshot's Patience: +50% damage and +20 accuracy if completely idle last turn
func deadshotTurnStart(squadID ecs.EntityID, roundNumber int,
    roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.WasIdleLastTurn {
        roundState.DeadshotReady = true
    } else {
        roundState.DeadshotReady = false
    }
}

func deadshotDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
    modifiers *squads.DamageModifiers, roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.DeadshotReady {
        // Verify ranged weapon type
        targetData := common.GetComponentTypeByID[*squads.TargetRowData](
            manager, attackerID, squads.TargetRowComponent,
        )
        if targetData == nil { return }
        if targetData.AttackType == squads.AttackTypeRangedRow ||
           targetData.AttackType == squads.AttackTypeMagicColumn {
            modifiers.DamageMultiplier *= 1.5
            modifiers.HitPenalty -= 20 // +20 accuracy (reduce penalty)
            roundState.DeadshotReady = false // Consumed
        }
    }
}

// Resolute: Survive lethal damage at 1 HP (once per unit per battle)
func resoluteTurnStart(squadID ecs.EntityID, roundNumber int,
    roundState *PerkRoundState, manager *common.EntityManager) {
    if roundState.RoundStartHP == nil {
        roundState.RoundStartHP = make(map[ecs.EntityID]int)
    }
    unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
    for _, uid := range unitIDs {
        attr := common.GetComponentTypeByID[*common.Attributes](
            manager, uid, common.AttributeComponent,
        )
        if attr != nil && attr.CurrentHealth > 0 {
            roundState.RoundStartHP[uid] = attr.CurrentHealth
        }
    }
}

func resoluteDeathOverride(unitID, squadID ecs.EntityID,
    roundState *PerkRoundState, manager *common.EntityManager) bool {
    if roundState.ResoluteUsed == nil {
        roundState.ResoluteUsed = make(map[ecs.EntityID]bool)
    }
    if roundState.ResoluteUsed[unitID] {
        return false
    }

    attr := common.GetComponentTypeByID[*common.Attributes](
        manager, unitID, common.AttributeComponent,
    )
    if attr == nil { return false }

    roundStartHP, ok := roundState.RoundStartHP[unitID]
    if !ok { return false }

    if float64(roundStartHP)/float64(attr.MaxHealth) > 0.5 {
        roundState.ResoluteUsed[unitID] = true
        return true // Prevent death, caller sets HP to 1
    }
    return false
}

// Guardian Protocol: Redirect 25% damage to adjacent tank
func guardianDamageRedirect(defenderID, defenderSquadID ecs.EntityID,
    damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
    // Find adjacent friendly squads with Guardian Protocol
    // Check if they have a Tank unit to absorb
    // This is a simplified example; full implementation needs adjacency check on tactical map
    guardianDmg := damageAmount / 4
    remainingDmg := damageAmount - guardianDmg
    return remainingDmg, 0, guardianDmg // 0 = no redirect found in this stub
}
```

---

## 10. Existing Code Changes

### `tactical/squads/squadcombat.go` -- DamageModifiers struct:
```go
type DamageModifiers struct {
    HitPenalty       int
    DamageMultiplier float64
    IsCounterattack  bool
    // New for perks:
    CritBonus        int     // Added to crit threshold (Executioner's Instinct)
    CoverBonus       float64 // Added to cover calculation (Brace, Fortify)
    SkipCounter      bool    // If true, no counterattack phase (Glass Cannon)
    SkipCrit         bool    // If true, crits become normal hits (Vigilance)
}
```

### `tactical/squads/squadcombat.go` -- calculateDamage():

> **Landmine note:** These hook calls cannot directly reference `perks.Run*` from `squadcombat.go` due to circular imports (see [Section 15.1](#151-circular-import-prevention)). Use **callback injection**: define hook runner function types in the `squads` package, pass implementations from `combat/combatactionsystem.go`.

```go
// CRITICAL: Hooks must run BEFORE multiplier application line,
// not after resistance calc. Perks modify the multiplier,
// so they must execute before it is applied.

// Before multiplier application:
damageModRunner(attackerID, defenderID, attackerSquadID, defenderSquadID, &modifiers, roundState, manager)
defenderDamageModRunner(attackerID, defenderID, attackerSquadID, defenderSquadID, &modifiers, roundState, manager)

// Multiplier application (existing):
baseDamage = int(float64(baseDamage) * modifiers.DamageMultiplier)

// Crit check (modified):
if modifiers.SkipCrit {
    // Skip crit entirely (Vigilance)
} else {
    critThreshold += modifiers.CritBonus // Executioner's Instinct
    // ... existing crit logic
}

// After cover calculation:
coverBreakdown := CalculateCoverBreakdown(defenderID, squadmanager)
coverModRunner(attackerID, defenderID, &coverBreakdown, roundState, manager)
```

### `tactical/squads/squadcombat.go` -- processAttackWithModifiers():

```go
// Signature change: add defenderSquadID and roundState parameters
func processAttackWithModifiers(attackerID ecs.EntityID,
    defenderSquadID ecs.EntityID, // NEW
    targetIDs []ecs.EntityID, ...) int {

    // NEW: Run target override hooks (attacker perks like Cleave)
    targetIDs = targetOverrideRunner(attackerID, defenderSquadID, targetIDs, manager)
    // NEW: Run defender target override hooks (Rearguard Doctrine)
    targetIDs = defenderTargetOverrideRunner(attackerID, defenderSquadID, targetIDs, manager)

    for _, defenderID := range targetIDs {
        damage, event := calculateDamage(attackerID, defenderID, modifiers, manager)
        recordDamageToUnit(defenderID, damage, result, manager)

        // NEW: Post-damage hooks
        wasKill := event.WasKilled
        postDamageRunner(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, wasKill, roundState, manager)

        // NEW: Death override check
        if wasKill {
            preventDeath := deathOverrideRunner(defenderID, defenderSquadID, roundState, manager)
            if preventDeath {
                // Set HP to 1, mark event as non-lethal
                event.WasKilled = false
            }
        }

        log.AttackEvents = append(log.AttackEvents, *event)
    }
    return attackIndex
}
```

### `tactical/combat/combatactionsystem.go` -- ExecuteAttackAction():
```go
// Before counterattack section:

// NEW: Check if counter should be suppressed by perks
counterModifiers := squads.DamageModifiers{
    HitPenalty: 20, DamageMultiplier: 0.5, IsCounterattack: true,
}
skipCounter := perks.RunCounterModHooks(defenderID, attackerID, &counterModifiers, roundState, cas.manager)

if defenderWouldSurvive && !skipCounter && !counterModifiers.SkipCounter {
    // ... existing counterattack code, using counterModifiers ...
}
```

---

## 11. Combat Integration Strategy

How the hardest perks integrate with the existing combat pipeline.

### 11.1 Guardian Protocol (Redirect 25% damage to adjacent Tank)

**Integration point:** `DamageRedirectHook` before `recordDamageToUnit()`.

This runs inside `processAttackWithModifiers` after `calculateDamage` returns but before `recordDamageToUnit`. If `redirectTargetID != 0`, record `redirectAmount` as additional damage to the guardian tank.

**Note:** Guardian operates at the *inter-squad* level (the guardian is in a different squad than the defender). The hook must check tactical map adjacency between squads.

### 11.2 Overwatch (Skip attack for reactive fire)

**Integration point:** New state flag + movement system hook.

This requires:
1. A flag in `PerkRoundState.OverwatchActive` set when the squad skips its attack
2. A check in `CombatMovementSystem` when an enemy moves: if any friendly squad with Overwatch has the enemy in range, trigger a 75% damage attack

**Recommendation:** Implement in a later phase after the core hook system is stable.

### 11.3 Resolute (Survive lethal damage)

**Integration point:** `DeathOverrideHook` in `ApplyRecordedDamage`.

When HP would drop to 0, check:
1. Does the squad have the Resolute perk?
2. Was the unit's HP > 50% at round start? (from `PerkRoundState.RoundStartHP` snapshot)
3. Has this unit already used Resolute this battle? (from `PerkRoundState.ResoluteUsed`)

If all pass, set HP to 1 instead of 0.

### 11.4 Marked for Death (New action type)

**Integration point:** New action type in `CombatActionSystem`.

This perk introduces a "Mark" action that costs the squad's attack action but deals no damage. Implementation:
1. Add a `MarkComponent` that can be attached to enemy squad entities
2. In the combat action system, add a "Mark" action option when a squad has the Marked for Death perk
3. When any squad attacks a marked target, check for the Mark component, apply +25% damage, and remove the Mark
4. If the marked target dies before the mark is consumed, the mark is simply lost

**Note:** This is the only perk that introduces a new action type. It may be deferred if combat action system changes are too invasive for v1.

### 11.5 Reckless Assault (Vulnerability window)

**Integration point:** `DamageModHook` (attacker) + `DamageModHook` (defender).

When this squad attacks, set `RecklessVulnerable = true`. When this squad is ATTACKED and `RecklessVulnerable` is true, apply +20% incoming damage multiplier. The flag resets at the start of this squad's next turn.

**Note:** The vulnerability window persists across the enemy's turn, making the timing meaningful -- attack early in the round and you're exposed to more enemy attacks.

---

## 12. Minimal Invasion Accounting

Exact existing file changes required.

| File | Change Description | Lines Added | Lines Modified |
|------|--------------------|-------------|----------------|
| `tactical/effects/components.go` | Add `SourcePerk` to EffectSource enum | +1 | 0 |
| `tactical/squads/squadcombat.go` | Add `CritBonus`, `CoverBonus`, `SkipCounter`, `SkipCrit` fields to `DamageModifiers` | +4 | 0 |
| `tactical/squads/squadcombat.go` | Hook calls in `calculateDamage()` (via callback injection) | +12 | 0 |
| `tactical/squads/squadcombat.go` | Hook calls in `processAttackWithModifiers()` | +8 | 0 |
| `tactical/squads/squadcombat.go` | Add `defenderSquadID` param to `processAttackWithModifiers` | 0 | 3 |
| `tactical/squads/squadcombat.go` | Export `getUnitsInRow` (capitalize) | 0 | 1 |
| `tactical/combat/combatactionsystem.go` | Counter mod hook + modified counter construction | +12 | 2 |
| `tactical/combat/combatactionsystem.go` | Death override hook in damage application | +6 | 0 |
| **Total** | | **~43** | **~6** |

---

## 13. Data-Driven JSON Schema

### perkdata.json (goes in `assets/gamedata/`)

```json
{
  "perks": [
    {
      "id": "reckless_assault",
      "name": "Reckless Assault",
      "description": "+30% damage dealt, but +20% damage received until next turn.",
      "tier": 0,
      "category": 0,
      "roles": ["DPS"],
      "exclusiveWith": [],
      "unlockCost": 2,
      "behaviorId": "reckless_assault"
    },
    {
      "id": "stalwart",
      "name": "Stalwart",
      "description": "If this squad did not move, counterattacks deal 100% damage instead of 50%.",
      "tier": 0,
      "category": 1,
      "roles": ["Tank", "Support"],
      "exclusiveWith": [],
      "unlockCost": 2,
      "behaviorId": "stalwart"
    },
    {
      "id": "isolated_predator",
      "name": "Isolated Predator",
      "description": "+25% damage when no friendly squads within 3 tiles.",
      "tier": 0,
      "category": 0,
      "roles": ["DPS"],
      "exclusiveWith": [],
      "unlockCost": 2,
      "behaviorId": "isolated_predator"
    },
    {
      "id": "opening_salvo",
      "name": "Opening Salvo",
      "description": "+35% damage on squad's first attack of the combat only.",
      "tier": 0,
      "category": 0,
      "roles": ["DPS"],
      "exclusiveWith": [],
      "unlockCost": 2,
      "behaviorId": "opening_salvo"
    },
    {
      "id": "cleave",
      "name": "Cleave",
      "description": "Melee attacks also hit one unit in the row behind the target, but all damage reduced by 30%.",
      "tier": 1,
      "category": 2,
      "roles": ["DPS"],
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "cleave"
    },
    {
      "id": "grudge_bearer",
      "name": "Grudge Bearer",
      "description": "+20% damage vs enemy squads that have damaged this squad (stacks to +40%).",
      "tier": 1,
      "category": 3,
      "roles": ["DPS", "Tank"],
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "grudge_bearer"
    },
    {
      "id": "counterpunch",
      "name": "Counterpunch",
      "description": "If attacked last turn AND did not attack last turn, next attack deals +40% damage.",
      "tier": 1,
      "category": 3,
      "roles": ["DPS", "Tank"],
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "counterpunch"
    },
    {
      "id": "marked_for_death",
      "name": "Marked for Death",
      "description": "Spend attack action to Mark enemy. Marked enemy takes +25% from next friendly attack.",
      "tier": 1,
      "category": 2,
      "roles": ["DPS", "Support"],
      "exclusiveWith": [],
      "unlockCost": 3,
      "behaviorId": "marked_for_death"
    },
    {
      "id": "deadshots_patience",
      "name": "Deadshot's Patience",
      "description": "If completely idle last turn (no move, no attack), next ranged attack gains +50% damage and +20 accuracy.",
      "tier": 1,
      "category": 0,
      "roles": ["DPS"],
      "exclusiveWith": [],
      "unlockCost": 4,
      "behaviorId": "deadshots_patience"
    },
    {
      "id": "resolute",
      "name": "Resolute",
      "description": "A unit that would die survives at 1 HP if it had >50% HP at round start (once per unit per battle).",
      "tier": 1,
      "category": 1,
      "roles": ["Tank", "DPS", "Support"],
      "exclusiveWith": [],
      "unlockCost": 4,
      "behaviorId": "resolute"
    }
  ]
}
```

### JSON Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (snake_case) |
| `name` | string | Display name |
| `description` | string | Tooltip text (should include the condition) |
| `tier` | int | 0=Conditioning, 1=Specialization |
| `category` | int | 0=Offense, 1=Defense, 2=Tactical, 3=Reactive |
| `roles` | []string | Which roles can use: "Tank", "DPS", "Support" |
| `exclusiveWith` | []string | Mutually exclusive perk IDs |
| `unlockCost` | int | Perk points to unlock |
| `behaviorId` | string | Key into hook registry |
| `params` | map | Per-behavior tuning parameters (optional) |

---

## 14. Implementation Order

### Phase 1: Foundation (~2 hours)

Create `tactical/perks/` package with all shared infrastructure.

1. `init.go` -- subsystem registration
2. `components.go` -- PerkSlotData, PerkRoundState, PerkUnlockData, ECS components
3. `perkdefinition.go` -- PerkDefinition struct, enums
4. `registry.go` -- PerkRegistry, LoadPerkDefinitions(), validation
5. `queries.go` -- getActivePerkIDs, HasPerk helpers
6. Add `SourcePerk` to `tactical/effects/components.go`
7. Create `assets/gamedata/perkdata.json` with all 24 perks

### Phase 2: Hook Infrastructure (~2 hours)

Build the hook system.

1. `hooks.go` -- 8 hook function type definitions
2. `hook_registry.go` -- PerkHooks struct, RegisterPerkHooks, GetPerkHooks
3. Hook runner functions in `queries.go`
4. Add new fields to `DamageModifiers` in `squadcombat.go`
5. Define hook runner callback types in `squads` package (circular import prevention)

### Phase 3: Existing Code Integration (~2 hours)

Insert hook calls into existing combat pipeline.

1. `calculateDamage()` -- inject damage mod + cover mod callbacks
2. `processAttackWithModifiers()` -- add `defenderSquadID` parameter, update callers
3. `ExecuteAttackAction()` -- move counter modifier construction up, add counter mod hooks
4. Add death override check in damage application
5. Export `getUnitsInRow` if needed

### Phase 4: Tier 1 Perks (~3 hours)

Implement all 10 Tier 1 perks -- simplest conditionals.

1. `behaviors.go` -- Reckless Assault, Stalwart, Vigilance (one per hook type)
2. Brace for Impact, Field Medic (cover mod, turn start)
3. Executioner's Instinct, Shieldwall, Isolated Predator, Opening Salvo, Last Line (damage mods)
4. Unit tests for each behavior

### Phase 5: Tier 2 Perks (~6 hours)

Implement 14 Tier 2 perks -- event reactions, state tracking, new action types.

1. Cleave (target override + damage penalty), Precision Strike (target override)
2. Riposte (counter mod)
3. Disruption, Bloodlust (post-damage + state tracking)
4. Adaptive Armor, Fortify (state tracking + mods)
5. Resolute (death override + turn start snapshot)
6. Grudge Bearer (post-damage tracking + damage mod)
7. Counterpunch, Deadshot's Patience (turn tracking + conditional damage)
8. Guardian Protocol (damage redirect -- may defer if too complex)
9. Overwatch (movement system hook -- may defer)
10. Marked for Death (new action type -- may defer if too invasive)
11. Unit tests

### Phase 6: GUI Equip Screen (~4-6 hours)

Pre-battle perk management.

1. New mode in `gui/guisquads/`
2. Unlocked perk pool display with role filtering
3. Squad perk slot management
4. Role gate and mutual exclusivity validation
5. Perk point spending UI
6. Perks cannot be changed mid-battle

### Phase 7: Polish + Testing (~4-6 hours)

1. **Interaction tests** -- verify stacking rules, exclusive perk enforcement
2. **Edge case tests** -- dead unit perks, PerkRoundState reset
3. Integration tests with full combat scenarios
4. Performance benchmarks: combat with 0 perks vs full loadout
5. AI consideration: how do enemy squads evaluate perk threats?

---

## 15. Relationship to Artifact System

The artifact system already has combat hooks wired through `CombatService`. This section clarifies why **no artifact code needs to be extracted** for perks, and how the two systems coexist.

### 15.1 Existing Artifact Hooks (post-event callbacks)

Registered via `combat_events.go` on `CombatService`. These fire AFTER events complete.

| Existing Hook | Signature | Where Fired | File |
|--------------|-----------|-------------|------|
| `OnPostReset` | `(factionID, squadIDs)` | After squad action reset | `turnmanager.go:108` |
| `OnAttackComplete` | `(attackerID, defenderID, result)` | After full combat resolution | `combatactionsystem.go:178` |
| `OnTurnEnd` | `(round)` | After turn advance | `turnmanager.go:165` |
| `OnMoveComplete` | `(squadID)` | After squad movement | Movement system |

Artifacts use these to trigger behaviors like "grant movement after kill" (OnAttackComplete) or "lock enemy squad next turn" (OnPostReset via pending effects).

### 15.2 New Perk Hooks (damage pipeline injection)

Perks need hooks that fire INSIDE the damage calculation pipeline, which artifacts never touch:

| New Hook | Where It Goes | Purpose |
|----------|--------------|---------|
| `DamageModHook` | Inside `calculateDamage()` BEFORE multiplier application | Modify damage multiplier conditionally |
| `TargetOverrideHook` | Inside `processAttackWithModifiers()` before target loop | Change who gets hit |
| `CounterModHook` | In `ExecuteAttackAction()` before counter phase | Modify counter accuracy/damage |
| `PostDamageHook` | Inside `processAttackWithModifiers()` after `recordDamageToUnit()` | React to damage events (kills, etc.) |
| `CoverModHook` | Inside `calculateDamage()` after cover calc | Modify cover values |
| `DeathOverrideHook` | In `ApplyRecordedDamage()` when HP would hit 0 | Prevent lethal damage |

### 15.3 Why No Extraction Is Needed

The two systems hook in at **different levels** with **different signatures** and **different state tracking**:

1. **Different hook signatures** -- Artifact hooks receive `BehaviorContext` + `CombatResult`. Perk hooks receive `DamageModifiers` + `PerkRoundState` + individual unit IDs.

2. **Different timing** -- Artifacts fire after events complete (post-event). Perks fire inside the calculation to modify it (mid-pipeline).

3. **Different state tracking** -- Artifacts use `ArtifactChargeTracker` (per-battle/per-round boolean charges + pending effect queue). Perks use `PerkRoundState` (per-turn/per-round/per-battle mixed state with stacks, counters, HP snapshots).

4. **The existing callback registration system already supports multiple consumers** -- `CombatService.RegisterPostResetHook()`, `RegisterOnTurnEnd()`, etc. all append to slices. Perks register alongside artifacts without any changes.

### 15.4 What Perks Reuse (Without Extraction)

Perks register additional callbacks on the SAME `CombatService` hooks that artifacts use:

- **`RegisterPostResetHook()`** -- For perk TurnStart logic (Field Medic healing, Resolute HP snapshots, Counterpunch/Deadshot readiness checks)
- **`RegisterOnTurnEnd()`** -- For PerkRoundState per-round field resets
- **`RegisterOnAttackComplete()`** -- For Grudge Bearer tracking (which squads attacked which)
- **`RegisterOnMoveComplete()`** -- For Overwatch trigger checks

Implementation: add a `setupPerkDispatch()` function in `combat_service.go` alongside existing `setupBehaviorDispatch()`:

```go
// In combat_service.go, called from NewCombatService()
setupBehaviorDispatch(cs, manager, cache)  // Existing: artifact behaviors
setupPerkDispatch(cs, manager, cache)      // New: perk hooks
```

### 15.5 What Perks Need New (Damage Pipeline)

New callback injection points in `combatcore/` using function-type parameters to avoid circular imports:

```go
// In combatcore/ - new callback types for perk injection
type DamageHookRunner func(attackerID, defenderID ecs.EntityID, mods *DamageModifiers)
type CoverHookRunner func(attackerID, defenderID ecs.EntityID, cover *CoverBreakdown)
type TargetHookRunner func(attackerID, defenderSquadID ecs.EntityID, targets []ecs.EntityID) []ecs.EntityID
type PostDamageRunner func(attackerID, defenderID ecs.EntityID, damage int, wasKill bool)
```

`CombatActionSystem` stores these runners (set at construction) and passes them into the calculation functions. The `combatcore` package defines the function types; the `perks` package provides the implementations; `combatservices` connects them. No circular import.

### 15.6 Summary

```
CombatService (combatservices/)
    ├─ setupBehaviorDispatch()     -- artifacts: post-event hooks
    │   ├─ RegisterPostResetHook     (artifact OnPostReset)
    │   ├─ RegisterOnAttackComplete  (artifact OnAttackComplete)
    │   └─ RegisterOnTurnEnd         (artifact OnTurnEnd + charge refresh)
    │
    ├─ setupPerkDispatch()         -- perks: post-event hooks (REUSES same API)
    │   ├─ RegisterPostResetHook     (perk TurnStart: Field Medic, Resolute snapshot)
    │   ├─ RegisterOnAttackComplete  (perk round tracking: Grudge Bearer)
    │   ├─ RegisterOnTurnEnd         (perk PerkRoundState reset)
    │   └─ RegisterOnMoveComplete    (perk Overwatch trigger)
    │
    └─ CombatActionSystem          -- perks: damage pipeline hooks (NEW)
        ├─ DamageHookRunner          (injected into calculateDamage)
        ├─ CoverHookRunner           (injected into calculateDamage)
        ├─ TargetHookRunner          (injected into processAttackWithModifiers)
        └─ PostDamageRunner          (injected into processAttackWithModifiers)
```

**Artifact package: no changes needed.** Both systems coexist through the shared callback registration API.

---

## 16. Go Design Pattern Analysis

### 16.1 Circular Import Prevention

**Risk:** `tactical/perks` needs types from `tactical/squads` (DamageModifiers, CoverBreakdown) and `tactical/effects` (ActiveEffect). Meanwhile `tactical/squads/squadcombat.go` will call perk hook runners.

**Solution -- Dependency direction:**

```
tactical/effects/   <-- no dependencies on perks or squads
tactical/squads/    <-- imports effects (already does via squadabilities.go)
tactical/perks/     <-- imports effects + squads (one-way)
tactical/combat/    <-- imports perks + squads (one-way)
```

The key insight: `tactical/squads/squadcombat.go` does NOT import `tactical/perks`. Instead, `tactical/combat/combatactionsystem.go` is the integration layer that imports both `squads` and `perks`. The hook calls in `calculateDamage()` and `processAttackWithModifiers()` are passed as **function parameters or called from the combat action system level**.

**Simplest approach:** Have `combatactionsystem.go` call perk runners directly. The `squadcombat.go` functions receive already-modified `DamageModifiers` and `targetIDs` -- they don't need to know about perks at all.

### 16.2 Function Types vs Interfaces

Go convention favors function types for single-method abstractions:

```go
// Preferred (Go-idiomatic):
type DamageModHook func(attackerID, defenderID ecs.EntityID,
    modifiers *squads.DamageModifiers, manager *common.EntityManager)

// Less Go-idiomatic:
type DamagePolicy interface {
    ModifyDamage(attackerID, defenderID ecs.EntityID,
        modifiers *squads.DamageModifiers, manager *common.EntityManager)
}
```

The function type approach has advantages:
- No struct needed for simple perks (just a named function)
- `nil` check is natural for optional hooks
- Closures capture parameters when needed

### 16.3 Composition Pattern

When a perk needs multiple hooks (Cleave: target override + damage penalty), the `PerkHooks` struct naturally composes them:

```go
RegisterPerkHooks("cleave", &PerkHooks{
    TargetOverride: cleaveTargetOverride,
    DamageMod:      cleaveDamageMod,
})
```

Each hook runs independently at its own call site. No complex middleware chain or decorator wrapping needed.

---

## 17. Implementation Landmines

Issues identified during code review that are not immediately obvious from the design above.

### Landmine 1: Hook Placement Precision

**Problem:** `DamageModifiers.DamageMultiplier` is consumed at a specific line in `calculateDamage()`. DamageModHooks **must run BEFORE** that line so that perks like Reckless Assault (+30%) and Bloodlust (+15%/kill) modify the multiplier before it's applied.

**Additionally:** `squadcombat.go` cannot import `tactical/perks` (circular import). But hook calls need to run inside `calculateDamage()`.

**Resolution:** Use **callback injection**. Define hook runner function types in the `squads` package. `combatactionsystem.go` wires the actual implementations.

### Landmine 2: Missing defenderSquadID Parameter

**Problem:** `processAttackWithModifiers()` does not include the defender's squad ID. `TargetOverrideHook` needs it to find adjacent rows (Cleave) or alternative targets.

**Impact:** Requires a signature change and updating all callers.

### Landmine 3: Counterattack Modifier Refactoring

**Problem:** Counterattack modifiers are currently constructed inside `ProcessCounterattackOnTargets`. For perk hooks (Riposte, Stalwart) to modify these values, the construction must move **up** to `ExecuteAttackAction()`.

### Landmine 4: PostDamageHook Timing

**Problem:** `recordDamageToUnit` only records damage -- actual HP changes happen later in `ApplyRecordedDamage`. PostDamageHooks run with **pre-damage HP** and should rely on the `damageDealt`/`wasKill` parameters, not entity HP values.

### Landmine 5: PerkRoundState Lifecycle

**Problem:** `PerkRoundState` has fields that reset per-round and fields that persist per-battle. Must clearly separate these in the reset logic.

| Reset Per Turn | Reset Per Round | Persist Per Battle |
|---------------|----------------|-------------------|
| MovedThisTurn | AttackedBy | HasAttackedThisCombat |
| AttackedThisTurn | KillsThisRound | ResoluteUsed |
| RecklessVulnerable | DisruptionTargets | GrudgeStacks |
| | OverwatchActive | MarkedSquad |

`TurnsStationary` persists across rounds but resets on movement.
`WasAttackedLastTurn`, `DidNotAttackLastTurn`, `WasIdleLastTurn` are updated at turn boundaries (previous turn state snapshots).
`CounterpunchReady` and `DeadshotReady` are set at turn start, consumed on first attack.

### Landmine 6: Guardian Protocol Complexity

**Problem:** Guardian operates at the inter-squad level (damage redirect between squads). Needs tactical map adjacency checks.

**Resolution:** Defer Guardian Protocol to a later phase if it proves too complex for v1.

### Landmine 7: getActivePerkIDs Performance

**Problem:** Every hook runner calls `getActivePerkIDs()` which does component lookups. For a squad with multiple perks, a single `calculateDamage()` call runs 3+ hook runners.

**Resolution:** Cache active perk IDs and resolved hooks per squad at **battle start**:

```go
type CachedSquadPerks struct {
    PerkIDs  []string
    HasHooks map[string]bool
}
var squadPerkCache map[ecs.EntityID]*CachedSquadPerks
```

Cache is built in `InitializeCombat()` and never invalidated (perks don't change mid-battle).

### Landmine Summary

| # | Landmine | Severity | When It Bites |
|---|----------|----------|---------------|
| 1 | Hook placement before multiplier application | High | First perk test |
| 2 | Missing defenderSquadID in processAttackWithModifiers | Medium | First target override perk |
| 3 | Counterattack modifiers hardcoded in wrong location | Medium | Riposte/Stalwart implementation |
| 4 | PostDamageHook sees pre-damage HP | Low | Disruption debuff on dead units |
| 5 | PerkRoundState lifecycle (per-round vs per-battle) | High | Fury stacks or Resolute reset incorrectly |
| 6 | Guardian Protocol inter-squad complexity | Medium | Guardian implementation |
| 7 | getActivePerkIDs performance | Low | Full perk loadouts in large battles |

---

## 18. Stacking Rules

When multiple perks modify the same values, stacking behavior must be defined explicitly.

- **DamageMultiplier: Multiplicative stacking** (each perk multiplies the current value)
  ```
  Reckless Assault: modifiers.DamageMultiplier *= 1.3   // Now 1.3
  Bloodlust(2):     modifiers.DamageMultiplier *= 1.30  // Now 1.69
  ```
- **CritBonus: Additive** (Executioner's Instinct +25% stacks with other crit sources)
- **CoverBonus: Additive** (Brace +0.15 + Fortify +0.15 = +0.30, but each has its own condition)
- **HitPenalty: Additive** (Deadshot's Patience -20 stacks with other hit modifiers)
- **SkipCrit: OR logic** (any perk setting true wins)

---

## 19. Testing Strategy

### Unit Tests (per hook function)

```go
func TestRecklessAssaultDamageMod(t *testing.T) {
    manager := setupTestManager()
    roundState := &PerkRoundState{}
    modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}

    recklessAssaultDamageMod(attacker, defender, attackerSquad, defenderSquad, &modifiers, roundState, manager)

    if modifiers.DamageMultiplier != 1.3 {
        t.Errorf("expected 1.3, got %f", modifiers.DamageMultiplier)
    }
    if !roundState.RecklessVulnerable {
        t.Error("expected RecklessVulnerable to be set")
    }
}

func TestOpeningSalvo_OnlyFirstAttack(t *testing.T) {
    manager := setupTestManager()
    roundState := &PerkRoundState{HasAttackedThisCombat: false}
    modifiers := squads.DamageModifiers{DamageMultiplier: 1.0}

    // First attack: should get bonus
    openingSalvoDamageMod(attacker, defender, attackerSquad, defenderSquad, &modifiers, roundState, manager)
    if modifiers.DamageMultiplier != 1.35 {
        t.Errorf("expected 1.35, got %f", modifiers.DamageMultiplier)
    }

    // Second attack: no bonus
    modifiers2 := squads.DamageModifiers{DamageMultiplier: 1.0}
    openingSalvoDamageMod(attacker, defender, attackerSquad, defenderSquad, &modifiers2, roundState, manager)
    if modifiers2.DamageMultiplier != 1.0 {
        t.Errorf("expected 1.0 (no bonus on second attack), got %f", modifiers2.DamageMultiplier)
    }
}
```

### Integration Tests (full pipeline)

```go
func TestRiposte_NoHitPenaltyOnCounter(t *testing.T) {
    manager := setupTestManager()
    attacker := createTestSquad(manager)
    defender := createTestSquadWithPerk(manager, "riposte")

    result := combat.ExecuteAttackAction(attacker, defender)
    for _, event := range result.CombatLog.AttackEvents {
        if event.IsCounterattack {
            // Verify counter hit penalty is 0, not -20
        }
    }
}
```

### Interaction Tests (critical)

```go
func TestStalwart_Plus_Riposte(t *testing.T) {
    // Stalwart: full damage counter if stationary
    // Riposte: no hit penalty on counter
    // Both should apply: 100% damage, 0 hit penalty
}

func TestResolute_OnlyOncePerUnit(t *testing.T) {
    // First lethal hit: survive at 1 HP
    // Second lethal hit: actually die
}

func TestRecklessAssault_VulnerabilityWindow(t *testing.T) {
    // After attacking: squad takes +20% damage
    // After next turn starts: vulnerability resets
}

func TestGrudgeBearer_StackCap(t *testing.T) {
    // Verify grudge stacks cap at +40% (2 stacks)
    // Verify bonus only applies vs specific enemy squad that hurt you
}

func TestCounterpunch_RequiresBothConditions(t *testing.T) {
    // Was attacked last turn + did NOT attack = +40% bonus
    // Was attacked last turn + DID attack = no bonus
    // Was NOT attacked + did not attack = no bonus
}
```

### Edge Case Tests

```go
func TestPerkRoundStateReset(t *testing.T) {
    // Verify per-round fields reset but per-battle fields persist
}

func TestPerkOnDeadUnit(t *testing.T) {
    // Dead unit's perks should not trigger
}

func TestRoleGateEnforcement(t *testing.T) {
    // Verify Tank-only perk cannot be equipped on DPS-only squad
}

func TestExclusivePerkEnforcement(t *testing.T) {
    // Verify mutually exclusive perks cannot both be equipped
}
```

---

## Implementation Estimate

| Phase | Work | Time |
|-------|------|------|
| Foundation (package, components, registry, JSON) | Phase 1 | ~2 hours |
| Hook infrastructure | Phase 2 | ~2 hours |
| Existing code integration | Phase 3 | ~2 hours |
| Tier 1 perks (10 perks) | Phase 4 | ~3 hours |
| Tier 2 perks (14 perks) | Phase 5 | ~6 hours |
| GUI equip screen | Phase 6 | ~4-6 hours |
| Polish + testing | Phase 7 | ~4-6 hours |
| **Total** | | **~23-27 hours** |
