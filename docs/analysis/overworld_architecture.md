# Overworld Architecture: Living Fronts System

**Last Updated:** 2026-01-23
**Status:** Architectural Design
**Related:** overworlddesign.md, CLAUDE.md

---

## Executive Summary

This document outlines the architecture for a pressure-driven overworld system where the world operates as a living, evolving entity independent of player actions. The system replaces traditional mission-based gameplay with emergent threat management, creating player-driven priorities through world pressure rather than scripted objectives.

### Core Principles

1. **Autonomous Evolution** - World state advances independently of player input
2. **Persistent Threats** - Encounters are ongoing systems, not discrete events
3. **Emergent Priorities** - Player urgency derives from escalating consequences, not timers
4. **Intervention-Based Play** - Players choose where to intervene, not what missions to accept
5. **Reactive World** - Combat outcomes reshape world state meaningfully

---

## System Architecture

### High-Level System Boundaries

```
┌─────────────────────────────────────────────────────┐
│             OVERWORLD SIMULATION LAYER              │
│  - Threat Node Management                           │
│  - Faction Behavior System                          │
│  - Tick Advancement Engine                          │
│  - Influence Propagation                            │
└──────────────┬──────────────────────┬───────────────┘
               │                      │
       ┌───────▼──────┐       ┌──────▼────────┐
       │   ENCOUNTER  │       │    PLAYER     │
       │   TRIGGER    │       │  MANAGEMENT   │
       │   SYSTEM     │       │    LAYER      │
       └───────┬──────┘       └──────┬────────┘
               │                      │
               └──────────┬───────────┘
                          │
                ┌─────────▼─────────┐
                │  TACTICAL COMBAT  │
                │      LAYER        │
                └─────────┬─────────┘
                          │
                ┌─────────▼─────────┐
                │   RESOLUTION &    │
                │  STATE FEEDBACK   │
                └───────────────────┘
```

### Component Responsibilities

**Overworld Simulation Layer**
- Maintains authoritative world state
- Executes periodic tick updates
- Manages threat node lifecycle and evolution
- Processes faction behaviors and territory changes
- Handles influence radius calculations and propagation

**Encounter Trigger System**
- Detects player-threat collisions on overworld
- Translates overworld entities into tactical combat scenarios
- Determines combat parameters based on threat node state

**Player Management Layer**
- Tracks player resources, squads, and positioning
- Manages recovery and reconfiguration phases
- Enforces resource constraints and squad composition rules

**Tactical Combat Layer**
- Isolated combat resolution (existing system)
- Generates combat outcomes and rewards
- No direct overworld state modification

**Resolution & State Feedback**
- Translates combat results into overworld state changes
- Applies threat node transformations (weakened, destroyed, evolved)
- Updates faction relationships and territory control
- Unlocks consequences or opportunities

---

## Core Subsystems

### 1. Overworld Tick System

**Purpose:** Drive all autonomous world evolution through discrete time steps.

**Responsibilities:**
- Advance simulation state at player-controlled or automatic intervals
- Orchestrate update order across all subsystems
- Track global tick counter for temporal dependencies
- Manage tick rate and pause states

**Key Considerations:**
- **Update Order:** Threat nodes → Factions → Environmental effects → Resource decay
- **Atomicity:** Each tick should be a complete, consistent state transition
- **Performance:** Must scale to hundreds of active entities
- **Determinism:** Same initial state + tick count = same result (for save/load integrity)

**Design Constraints:**
- Player can trigger tick advancement manually (take turn) or automatically (time-based)
- No update should depend on player input mid-tick
- Tick granularity affects gameplay pacing - too fine creates noise, too coarse loses tension

---

### 2. Threat Node System

**Purpose:** Model persistent, evolving sources of danger and pressure.

**Data Model:**

A threat node represents:
- **Spatial Position:** Overworld location
- **Threat Type:** Defines behavior patterns (Necromancer, Bandit Camp, Corruption, etc.)
- **Intensity Level:** Current strength/tier (affects rewards and difficulty)
- **State Metadata:** Type-specific data (unit counts, resource drain rate, corruption spread)
- **Evolution Parameters:** Growth rates, mutation thresholds, spawn conditions

**Behavior Patterns:**

Each threat type implements:
- **Per-Tick Evolution:** How the threat grows or changes when unaddressed
- **Influence Effects:** Impact on surrounding tiles (spawn modifiers, resource drain, terrain corruption)
- **Defeat Conditions:** What constitutes resolution or transformation
- **Reward Table:** Benefits granted on successful intervention

**Evolution Mechanics:**

```
On Each Tick:
    IF threat_node is in player_influence:
        Apply containment effects (slow growth)
    ELSE:
        Accumulate growth_points
        IF growth_points >= evolution_threshold:
            Upgrade intensity_level
            Apply evolution_effect (spawn child nodes, mutate type, expand territory)
            Reset growth_points
```

**Spatial Influence:**

Threat nodes project influence based on intensity:
- **Radius Calculation:** Influence tiles within distance = base_radius + (intensity * scale_factor)
- **Stacking Effects:** Multiple overlapping influences compound (multiplicative or additive based on effect type)
- **Decay Model:** Influence strength may decrease with distance from source

**Key Design Decisions:**

- **Growth Curves:** Linear vs exponential escalation (exponential creates urgency, linear creates predictability)
- **Cap Mechanisms:** Should threat nodes have maximum intensity, or unbounded growth?
- **Interaction Rules:** Can threat nodes conflict with each other, or only pressure the player?
- **Memory Requirements:** Each node carries state - optimize for hundreds of concurrent nodes

---

### 3. Faction System

**Purpose:** Simulate independent actors that expand, retreat, and conflict autonomously.

**Faction Behaviors:**

- **Expansion:** Claim new territory based on adjacent control and strength
- **Fortification:** Strengthen existing positions (increase defense, spawn reinforcements)
- **Raiding:** Target player resources or rival factions
- **Mutation:** Transform units or behaviors under certain conditions (corruption, upgrades)
- **Retreat:** Abandon territory when overpowered or resource-depleted

**Faction State:**

- **Territory Control:** Set of tiles under faction influence
- **Strength Metrics:** Military power, resource stockpiles, unit composition
- **Relationships:** Hostility/alliance values toward player and other factions
- **Strategic Intent:** Current objective (expand west, defend border, raid player, etc.)

**Decision-Making:**

Factions evaluate actions per tick:

```
FOR each faction:
    Assess current_state (territory, strength, threats)
    Evaluate available_actions (expand, raid, fortify, retreat)
    Score actions based on faction_personality + strategic_situation
    Execute highest_scored_action
    Update faction_state and world_state
```

**Design Constraints:**

- **Computational Budget:** Faction AI must be lightweight (hundreds of ticks over session)
- **Predictability vs Surprise:** Balance between telegraphed behavior and emergent chaos
- **Player Impact:** Player victories should meaningfully alter faction trajectories, not just delay them
- **Asymmetry:** Different faction types should feel mechanically distinct (swarm vs elite, expansion vs consolidation)

---

### 4. Encounter Translation System

**Purpose:** Convert overworld threat nodes into tactical combat scenarios.

**Translation Process:**

```
Player collides with threat_node on overworld:
    1. Extract threat_node.intensity and threat_node.type
    2. Generate combat_parameters:
        - Enemy composition (unit types, counts)
        - Battlefield terrain (based on overworld tile type + threat modifiers)
        - Combat modifiers (corruption effects, faction bonuses)
        - Win/loss stakes (rewards on victory, penalties on defeat)
    3. Freeze overworld state
    4. Enter tactical combat layer
    5. On combat_end:
        - Apply outcome to threat_node (reduce intensity, destroy, transform)
        - Update player_state (casualties, rewards)
        - Resume overworld state
```

**Key Considerations:**

- **State Isolation:** Tactical combat should not directly mutate overworld state (only through resolution layer)
- **Difficulty Scaling:** Threat intensity → enemy strength should be tunable and fair
- **Terrain Consistency:** Overworld tile types should map logically to tactical battlefield layouts
- **Retreat Mechanics:** Can player retreat from combat? If so, what are overworld consequences?

---

### 5. Influence Propagation System

**Purpose:** Calculate and apply area-of-effect pressure from threat nodes.

**Propagation Algorithm:**

**Option A: Flood Fill (Accurate but Expensive)**
- Start from threat_node position
- Expand outward tile-by-tile up to max_radius
- Track distance and apply decay function
- Handles obstacles and irregular shapes naturally

**Option B: Radius Check (Fast but Crude)**
- For each tile in bounding_box around threat_node:
    - IF distance(tile, threat_node) <= influence_radius:
        - Apply influence_effect with decay_function(distance)
- Ignores obstacles, assumes circular spread

**Option C: Cached Influence Maps (Optimized)**
- Pre-calculate influence zones per threat type/intensity
- On threat_node spawn, stamp pre-computed influence pattern
- Update only dirty regions when nodes change
- Trade memory for computation time

**Effect Application:**

Influenced tiles accumulate modifiers:
- **Spawn Weights:** Increase probability of enemy encounters
- **Resource Penalties:** Reduce gold/supply yields
- **Terrain Mutations:** Corruption spreads, hazards appear
- **Combat Modifiers:** Enemies gain buffs, player units suffer debuffs

**Performance Considerations:**

- Recalculating influence every tick for every node is expensive
- **Dirty Flagging:** Only recalculate when threat nodes spawn/die/evolve
- **Spatial Partitioning:** Use quadtree or grid sectors to limit range queries
- **Incremental Updates:** Update only changed regions, not entire map

---

## Data Flow Diagrams

### Overworld Tick Cycle

```
START TICK
    ↓
[1. Threat Node Updates]
    - Accumulate growth
    - Check evolution thresholds
    - Spawn child nodes if triggered
    ↓
[2. Faction Behavior]
    - Evaluate strategic situation
    - Execute expansion/raid/fortify actions
    - Update territory control
    ↓
[3. Influence Recalculation]
    - Recompute affected regions (dirty flags)
    - Apply environmental effects
    ↓
[4. Resource & Economy]
    - Apply drain effects from active threats
    - Replenish player resources (if in safe zones)
    - Update shop availability/pricing
    ↓
[5. Event Generation]
    - Spawn new threat nodes (probabilistic)
    - Trigger faction conflicts
    - Generate world events (optional narrative color)
    ↓
END TICK → Present updated overworld to player
```

### Combat Resolution Flow

```
[Tactical Combat Ends]
    ↓
[Outcome Analysis]
    - Player victory/defeat/retreat?
    - Casualties on both sides
    - Loot/rewards earned
    ↓
[Threat Node Resolution]
    IF player_victory:
        threat_intensity -= damage_dealt
        IF threat_intensity <= 0:
            Destroy threat_node
            Grant completion_rewards
            Unlock consequences (faction response, new opportunities)
        ELSE:
            Weaken threat_node
            Grant partial_rewards
    ELSE IF player_defeat:
        threat_intensity += growth_boost (won battle)
        Player suffers penalties (resource loss, squad casualties)
    ELSE IF player_retreat:
        threat_intensity unchanged
        Player escapes with reduced casualties
    ↓
[World State Update]
    - Update overworld tile states
    - Recalculate influence maps (if threat destroyed)
    - Trigger faction reactions (if territory changed)
    - Advance partial tick (time passed during combat)
    ↓
[Return to Overworld]
    - Player can reposition, reconfigure squads
    - World continues ticking
```

---

## Design Considerations

### 1. Escalation vs Degeneracy

**Problem:** Unbounded growth leads to unwinnable states; too much containment trivializes threats.

**Mitigation Strategies:**
- **Soft Caps:** Threat growth slows asymptotically (diminishing returns)
- **Containment Mechanics:** Player presence nearby slows (but doesn't halt) threat evolution
- **Lateral Escalation:** Instead of existing threats growing infinitely, new threat types emerge
- **Resource Drains:** Strong threats consume resources to sustain, creating natural ceilings

**Tuning Levers:**
- Growth rates per threat type
- Evolution thresholds (how many ticks to next tier)
- Maximum intensity caps (if enforced)
- Containment effectiveness radius

---

### 2. Player Agency vs Overwhelming Complexity

**Problem:** Too many simultaneous threats paralyze decision-making; too few removes tension.

**Design Approaches:**
- **Information Hierarchy:** Highlight most urgent threats (proximity + intensity)
- **Threat Clustering:** Nearby nodes visually/mechanically group (tackle region, not individual nodes)
- **Natural Priorities:** Some threat types are objectively more dangerous (corruption > bandit raids)
- **Player Tools:** Scouting, intelligence systems to preview threat evolution trajectories

**UI Considerations:**
- Overworld must clearly communicate threat intensity (visual scaling, color coding)
- Prediction system: "If ignored for N ticks, this node will evolve to tier X"
- Risk/reward comparison: "Defeating this node grants Y resources vs Z combat difficulty"

---

### 3. Determinism and Save/Load Integrity

**Problem:** Emergent systems with randomness can diverge across save/load cycles.

**Requirements:**
- **Seeded RNG:** All random decisions use deterministic seed derived from tick count + world state
- **State Completeness:** Save files must capture all threat node state, faction positions, influence maps
- **Replay Consistency:** Loading a save at tick T and advancing to tick T+10 should produce identical state every time

**Implementation Considerations:**
- Avoid using system time or non-deterministic inputs
- Random events use tick-based seed: `seed = world_seed XOR tick_count XOR entity_id`
- Store RNG state in save files if using stateful generators

---

### 4. Integration with Existing ECS Architecture

**Alignment with TinkerRogue Patterns:**

**Threat Nodes as ECS Entities:**
- **ThreatNodeComponent:** Pure data (type, intensity, evolution progress)
- **ThreatNodeTag:** Query target for tick updates
- **InfluenceComponent:** Cached influence radius and effect modifiers
- **PositionComponent:** Overworld logical position (reuse existing system)

**Faction State:**
- **FactionComponent:** Faction ID, relationships, strength metrics
- **TerritoryComponent:** Set of tile indices under control
- **StrategicIntentComponent:** Current objective and action queue

**Queries and Systems:**
- `GetActiveThreatNodes()` → Returns all entities with ThreatNodeTag
- `UpdateThreatNode(entity, tick_delta)` → System function for evolution logic
- `CalculateInfluenceMap(threat_nodes)` → System function for propagation
- `ExecuteFactionTurn(faction_entity)` → System function for faction AI

**Lifecycle Management:**
- Use `manager.CleanDisposeEntity()` when threat nodes destroyed (removes from PositionSystem)
- Use `manager.MoveEntity()` if threat nodes can relocate (preserves position system integrity)

**Separation of Concerns:**
- Overworld state = ECS components
- Overworld UI state = Separate structure (similar to BattleMapState pattern)
- Tactical combat state = Separate ECS context (existing combat system)

---

### 5. Temporal Granularity and Pacing

**Tick Advancement Models:**

**Option A: Player-Driven Ticks**
- Player explicitly ends turn (button press, movement, combat completion)
- Explicit control, no pressure from real-time clock
- Risk: Players optimize-to-death, removing urgency

**Option B: Hybrid (Recommended)**
- Player actions consume "action points"
- World ticks when action_points exhausted OR player manually advances
- Creates soft time pressure without real-time stress
- Example: Movement costs 1 AP, combat costs 3 AP, tick advances at 10 AP

**Option C: Automatic Ticks**
- World advances every N seconds regardless of player input
- Creates real-time pressure
- Risk: Punishes deliberate thinking, excludes accessibility needs

**Recommendation:** Hybrid model balances agency and urgency. Allow players to configure tick rate (accessibility).

---

### 6. Threat Node Diversity and Mechanical Depth

**Avoiding Homogeneity:**

Different threat types should create distinct strategic problems:

**Expansion Threats (Necromancer, Corruption):**
- Grow outward geographically
- Priority: Contain before spread becomes unmanageable
- Defeat strategy: Strike the core node to collapse network

**Resource Threats (Bandit Camps, Locusts):**
- Drain player economy over time
- Priority: Balance cost of ignoring vs cost of fighting
- Defeat strategy: Quick surgical strikes to preserve resources

**Mutation Threats (Corruption Shrines, Mad Wizard):**
- Alter combat rules or unit stats in radius
- Priority: Prevent permanent world state corruption
- Defeat strategy: Purge before effects become irreversible

**Mobile Threats (Rival Armies, Nomadic Raiders):**
- Reposition dynamically, unpredictable
- Priority: Intercept before they reach vulnerable targets
- Defeat strategy: Prediction and ambush

**Implementation:**
- Each threat type has unique `EvolveBehavior()` and `InfluenceEffect()` functions
- Shared ECS component structure, type-specific logic in system functions
- Easily extensible (add new types without refactoring core systems)

---

### 7. Victory and Loss Conditions

**Open Question:** Does this system have win/loss states, or is it infinite sandbox?

**Option A: Survival Sandbox**
- No formal victory, player persists until defeated
- Score/prestige metrics track performance
- Escalation eventually overwhelms player (roguelike permadeath)

**Option B: Campaign Objectives**
- Hidden "boss" threat nodes (defeat ancient evil, destroy corruption source)
- Victory = eliminate specific endgame threats
- Contradicts "no scripted objectives" design goal

**Option C: Dynamic Equilibrium**
- Player success reduces threat spawn rates
- Cleared regions provide safe havens and resource generation
- Victory = stabilizing entire overworld into peaceful state
- Loss = world becomes entirely hostile (no safe zones remain)

**Recommendation:** Option C aligns with emergent design. Track global threat/stability ratio. Victory = ratio below threshold for N ticks.

---

## Technical Constraints

### Performance Targets

- **Tick Update Time:** < 100ms for 500 active entities (threat nodes + factions)
- **Influence Recalculation:** < 50ms for dirty region updates
- **Save/Load Time:** < 2 seconds for complete world state

### Scalability Limits

- **Maximum Threat Nodes:** 500 concurrent (based on ECS query performance)
- **Overworld Size:** 256x256 tiles (65,536 tiles total, aligns with existing CoordinateManager)
- **Faction Count:** 10-20 active factions (AI budget constraint)

### Memory Considerations

- **Influence Maps:** If cached, 256x256 grid of modifiers = ~256KB per effect type
- **Threat Node State:** Average 1KB per node × 500 nodes = 500KB
- **Faction State:** ~10KB per faction × 20 factions = 200KB
- **Total Overworld State:** ~1-2MB (acceptable for modern systems)

---

## Open Design Questions

1. **Threat Node Respawn:** Should defeated nodes respawn eventually, or are they permanent removals?
2. **Player Death:** Permadeath, or respawn with penalties? How does this interact with world state advancement?
3. **Multi-Front Combat:** Can player engage multiple threat nodes simultaneously (split squads)?
4. **Faction Diplomacy:** Can player ally with factions, or are all factions hostile?
5. **Retreat Consequences:** Should player retreat from combat trigger threat evolution or faction responses?
6. **Containment vs Elimination:** Should player presence slow threats, or does it require active combat to contain?
7. **Endgame Scaling:** Should max threat intensity cap, or scale infinitely with player progression?

---

## Implementation Phases (Suggested)

**Phase 1: Foundation**
- Overworld tick system (basic time advancement)
- Threat node ECS components and lifecycle
- Simple linear evolution (intensity grows per tick)
- Placeholder influence (flat radius, no propagation)

**Phase 2: Behavior Depth**
- Threat type diversity (3-5 distinct types with unique behaviors)
- Influence propagation system (choose algorithm based on profiling)
- Combat-to-overworld resolution pipeline

**Phase 3: Faction Dynamics**
- Basic faction AI (expand, fortify, retreat)
- Territory control visualization
- Faction-threat node interactions (conflicts, alliances)

**Phase 4: Tuning and Emergence**
- Balance escalation curves
- Add player containment mechanics
- Implement victory/loss conditions
- Performance optimization (caching, spatial partitioning)

**Phase 5: Polish**
- Overworld UI enhancements (threat prediction, priority highlighting)
- Save/load integrity testing
- Accessibility options (configurable tick rates, pause states)

---

## Conclusion

The Living Fronts overworld system replaces traditional mission structures with autonomous world evolution, creating emergent gameplay through pressure and intervention. Success depends on:

1. **Clear Separation:** Overworld simulation, tactical combat, and UI state must remain distinct layers
2. **Performance:** Tick updates must be fast and scalable to support hundreds of active entities
3. **Tuning:** Escalation curves and containment mechanics require careful balancing to avoid runaway or trivial states
4. **Integration:** Align with existing ECS patterns for maintainability

The architecture prioritizes emergent complexity over scripted content, allowing meaningful player agency within a living, reactive world.
