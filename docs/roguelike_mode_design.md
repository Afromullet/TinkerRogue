# TinkerRogue: Roguelike Mode Design Document

**Version:** 1.0
**Date:** 2026-02-17

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Design Philosophy](#2-design-philosophy)
3. [Run Structure](#3-run-structure)
4. [The Node Map](#4-the-node-map)
5. [Node Types](#5-node-types)
6. [Combat Encounters](#6-combat-encounters)
7. [Enemy Scaling and Composition](#7-enemy-scaling-and-composition)
8. [Reward Systems](#8-reward-systems)
9. [Between-Battle Activities](#9-between-battle-activities)
10. [Spell and Mana Economy](#10-spell-and-mana-economy)
11. [Artifact Strategy Layer](#11-artifact-strategy-layer)
12. [Run Identity and Replayability](#12-run-identity-and-replayability)
13. [Difficulty and Balance Philosophy](#13-difficulty-and-balance-philosophy)
14. [Leveraging Existing Systems](#14-leveraging-existing-systems)
15. [Player Progression Within a Run](#15-player-progression-within-a-run)
16. [Meta-Progression (Future)](#16-meta-progression-future)
17. [Future Expansion Possibilities](#17-future-expansion-possibilities)

---

## 1. Executive Summary

TinkerRogue's Roguelike Mode is a self-contained campaign experience built around the game's deep tactical combat, squad-building, and spell systems. Each run takes the player through a series of escalating battles connected by a node-based map, where between-battle decisions around army composition, artifact acquisition, spell management, and risk-taking shape the trajectory of the run as much as the battles themselves.

The mode draws inspiration from the structural clarity of Slay the Spire's branching maps, the tactical weight of Into the Breach's combat puzzles, and the army-building satisfaction of games like Symphony of War and Battle Brothers. The goal is not merely to string battles together, but to create a complete strategic arc where the player builds, adapts, and ultimately tests their army against a final boss encounter.

The entire mode operates without an overworld map. Navigation happens through a node map -- a visual web of interconnected encounter nodes that the player progresses through floor by floor. This keeps the pacing tight and the focus squarely on the decisions that matter: what to fight, what to buy, when to rest, and how to build.

---

## 2. Design Philosophy

### Core Pillars

**Tactical Combat is King.** Every design decision serves the combat system. Between-battle activities exist to make battles more interesting, not to replace them. The 3x3 formation grid, cover mechanics, attack types, and counterattack system are the foundation everything else stands on.

**Meaningful Scarcity.** Gold, mana, unit health, and artifact slots are all limited. The player can never have everything they want. This scarcity is what transforms simple decisions ("buy this unit") into interesting ones ("buy this unit now and miss the artifact shop later, or save gold and risk the next fight understrength").

**Builds, Not Builds.** Unlike a traditional RPG where the player follows a single character build, roguelike mode builds are emergent. The player starts with a basic army and adapts based on what the run offers them. A run that offers early magic artifacts plays differently than one that offers early tank units. The player's job is to recognize what the run is giving them and lean into it.

**No Wasted Fights.** Every combat encounter should feel like it matters. This means every fight offers something -- experience for units, gold for purchasing, sometimes artifacts or spell scrolls -- and every fight costs something (unit health, mana spent, time not spent at a rest node). The player should never feel like they are grinding.

**Permadeath with Persistence.** Units that die in combat are gone for the run. Squads that lose their leader must be reorganized. This creates tension in every battle and makes the decision to retreat or sacrifice a squad genuinely weighty. However, between-battle healing and recruitment prevent a single bad fight from ending the run.

---

## 3. Run Structure

### Acts and Floors

A complete run consists of **three Acts**, each representing a thematic escalation in challenge and complexity.

**Act 1: Mustering (Floors 1-3)**
The player begins with a starter army and limited resources. Encounters are drawn from a single enemy faction (randomly selected at run start). The act establishes the baseline challenge and gives the player enough gold and experience to start forming a real strategy. Act 1 ends with an **Elite Boss** -- a harder-than-normal encounter that tests whether the player's army can handle focused pressure.

- Encounter difficulty: Easy to Moderate (levels 1-2 from encounterdata)
- Enemy squad count: 4-6 squads per battle
- Player expected to have 2-3 squads by act end
- Gold rewards: Enough to recruit one commander and buy a meaningful number of units

**Act 2: Campaigning (Floors 4-7)**
A second enemy faction is introduced alongside the first. Encounters become more varied and demanding. The player should be making build-defining decisions: committing to a particular army composition, choosing artifacts that synergize with their squads, and managing mana as a strategic resource across multiple battles. Act 2 ends with a **Faction Boss** -- a full-strength encounter against the toughest configuration of one of the two factions.

- Encounter difficulty: Moderate to Hard (levels 2-4 from encounterdata)
- Enemy squad count: 6-8 squads per battle
- Player expected to have 3-4 squads by act end
- Artifact economy becomes important: first major artifacts appear

**Act 3: Siege (Floors 8-10)**
All active factions field their strongest compositions. The player faces the most demanding tactical puzzles of the run. Resources are scarce relative to the challenge. The player must choose battles carefully and play to their army's strengths. Act 3 culminates in the **Final Boss** -- a multi-phase encounter that tests the full breadth of the player's army.

- Encounter difficulty: Hard to Boss (levels 4-5 from encounterdata)
- Enemy squad count: 8-10 squads per battle
- Player expected to have 4-5 squads at peak strength
- Mana conservation and artifact charge management become critical

### Floor Structure

Each floor contains **5-8 nodes** arranged in a branching path. The player starts at the left side and must reach the right side. Each column of nodes represents a "layer" -- the player moves left to right, choosing one node per layer. Branching means some layers have 2-3 nodes to choose from, while others have only 1 (forcing the encounter).

A typical floor layout:

```
Layer 1     Layer 2     Layer 3     Layer 4     Layer 5
[Battle] -- [Event] --- [Shop] ---- [Battle] -- [Boss]
         \           /           \
          [Rest] ---              [Elite]
```

The player always has at least one choice per floor, but some paths are clearly riskier (elite encounters) or safer (rest nodes) than others. This creates the Slay-the-Spire-style strategic path planning where the player looks ahead and decides whether the rewards of the harder path justify the costs.

---

## 4. The Node Map

### Visual Presentation

The node map displays the current floor's nodes as icons connected by lines. The player can see all nodes on the current floor (and their types) but not future floors. Completed nodes are grayed out, the current node is highlighted, and available next nodes are selectable.

Each node type has a distinct icon and color:
- **Battle:** Red crossed swords
- **Elite Battle:** Gold crossed swords with a star
- **Boss:** Large red skull
- **Shop:** Yellow coin
- **Rest:** Green campfire
- **Event:** Blue question mark
- **Treasure:** Orange chest
- **Recruitment:** Purple banner

### Path Rules

- The player must visit exactly one node per layer (no skipping layers)
- Each floor begins with a guaranteed battle or event node (to prevent empty starts)
- Each floor ends with either a boss node (end-of-act floors) or a battle node
- Shop and rest nodes never appear in the first or last layer of a floor
- At least one path through each floor must include a shop node
- Elite battles are always optional (there is always a non-elite alternative in the same layer)

### Map Generation

Floor maps are generated at the start of each floor using weighted randomization:

- **Act 1 floors** favor rest and shop nodes (40% battle, 20% shop, 15% rest, 15% event, 10% treasure)
- **Act 2 floors** shift toward battles and events (45% battle, 15% shop, 10% rest, 20% event, 5% treasure, 5% elite)
- **Act 3 floors** are combat-heavy with scarce resources (50% battle, 10% shop, 5% rest, 15% event, 5% treasure, 15% elite)

The weights ensure the player always has meaningful choices without the map becoming too predictable.

---

## 5. Node Types

### 5.1 Battle Node

The standard combat encounter. The player enters tactical combat against an enemy force on a generated battle map.

**What happens:**
1. Enemy composition is revealed (faction, squad count, approximate strength)
2. Player chooses which squads to deploy (up to their available squads, max depends on the encounter)
3. A tactical map is generated using the appropriate biome generator
4. Standard turn-based combat plays out using the existing combat system
5. After victory: gold, XP, and possible item rewards are distributed

**Pre-battle information shown to the player:**
- Enemy faction name and icon
- Number of enemy squads
- Relative strength indicator (Weak / Balanced / Strong / Overwhelming)
- Biome of the battle map
- Special conditions, if any

This preview lets the player make informed decisions about which path to take on the node map.

### 5.2 Elite Battle Node

A harder variant of the standard battle with better rewards. Elite encounters feature enemy compositions from the "elite" tier of encounter definitions (difficulty level 4), and may include one or more **battle modifiers** that change the tactical landscape.

**Battle Modifiers (pick 1-2 per elite):**
- **Reinforcements:** Enemy receives an additional squad after round 3
- **Entrenched:** Enemy squads start in favorable positions with cover bonuses
- **Fog of War:** Player cannot see enemy squad compositions until adjacent
- **Narrow Front:** Battle map has heavy choke points, limiting deployment width
- **High Ground:** Enemy squads have elevation advantage (bonus damage)

**Rewards:** 1.5x gold, guaranteed minor artifact or spell scroll, bonus XP.

### 5.3 Boss Node

Boss encounters appear at the end of each act. They are the most challenging encounters in the run, featuring "boss" tier encounter definitions (difficulty level 5) with unique mechanics.

**Act 1 Boss -- "The Vanguard":**
A single faction's elite force. Tests basic army composition and tactical fundamentals.
- 6-8 enemy squads with strong compositions
- One enemy squad has a unique leader ability (higher stats, special trigger)
- No special mechanics -- a "pure" tactical test

**Act 2 Boss -- "The Warlord":**
A faction commander with tactical tricks.
- 8-10 enemy squads
- Boss modifier: one of the enemy squads regenerates a unit each round (representing reserves)
- The boss squad itself has artifact-like abilities (e.g., Twin Strike equivalent)

**Act 3 Final Boss -- "The Overlord":**
A multi-phase encounter.
- **Phase 1:** 8 enemy squads, standard composition. On defeat, a brief narrative interlude.
- **Phase 2:** 6 fresh elite squads join. The player's squads carry over their current HP and positions from Phase 1 -- no reset. Mana carries over. This tests endurance and resource management.
- Defeating Phase 2 wins the run.

**Boss rewards:** Large gold payout, choice of one major artifact (pick 1 of 3), full heal for all surviving units.

### 5.4 Shop Node

The shop is the primary economic activity between battles. It is where gold translates into army power.

**Shop Inventory (refreshed each visit):**

*Unit Market (always available):*
- 4-6 unit types available for purchase, drawn from the full 23-unit roster
- Prices scale with unit power: cheap units (Goblin Raider, Skeleton Archer) at 50-100g, mid-range units (Knight, Archer, Wizard) at 150-250g, premium units (Ogre, Assassin, Paladin) at 300-500g
- Units purchased go into the player's unit roster for placement into squads

*Artifact Stall (sometimes available):*
- 0-2 artifacts for sale, mix of minor and major
- Minor artifacts: 200-400g
- Major artifacts: 600-1000g
- Availability increases in later acts

*Spell Vendor (sometimes available):*
- 0-2 spell scrolls for sale (adds spell to commander's spellbook)
- Cheap spells (Spark, Singe, Frost Snap): 100-150g
- Mid spells (Fireball, Chain Lightning, War Cry): 200-350g
- Expensive spells (Obliterate, Firestorm, Absolute Zero): 500-800g
- The player starts with a small spellbook; scrolls expand it

*Consumable Shelf (always available):*
- Healing Salves: Restore flat HP to all units in a squad (100-200g)
- Mana Crystals: Restore mana (150-300g)
- Supplies: Restore flat HP to all units across all squads, smaller amount (250g)

**Shop design philosophy:** The shop should feel like a place of agonizing trade-offs. There is always more to buy than the player can afford. The player must prioritize between immediate power (units), strategic power (artifacts, spells), and sustain (healing, mana). This is where the "builds emerge from scarcity" principle comes alive.

### 5.5 Rest Node

Rest nodes provide recovery and reorganization between battles.

**Options at a rest node (choose one):**

*Heal:*
- Restore 50% of missing HP to all units across all squads
- This is the primary way to recover from a tough fight without spending gold

*Reorganize:*
- Enter the squad editor: reassign units between squads, change formations, swap artifacts
- Recruit one free basic unit (from a limited selection) to replace losses
- This does NOT heal, but it lets the player optimize their army structure

*Train:*
- Award 50 bonus XP to all surviving units
- Useful for pushing units toward level-up thresholds
- No healing, no reorganization

The choice between heal, reorganize, and train creates different strategic value at different points in a run. After a brutal fight, heal is essential. When the army composition feels wrong, reorganize lets the player fix it. When the army is healthy but a few key units are close to leveling, train provides a power spike.

### 5.6 Event Node

Events are narrative encounters that present the player with a choice. Each event has 2-3 options with different risk/reward profiles. Events are the primary source of run variety and surprise.

**Event Categories:**

*Mercenary Camp:*
- A wandering mercenary offers to join. The player can:
  - **Hire** (costs gold): Add a specific unit to the roster. The unit type is revealed upfront.
  - **Challenge** (free, risky): Fight the mercenary's squad in a 1v1 mini-battle. Win to recruit them for free. Lose and take damage.
  - **Pass:** No effect.

*Abandoned Armory:*
- The player discovers an old cache. Options:
  - **Search Thoroughly** (costs time -- skip next rest bonus): Find 1-2 artifacts (could be minor or major).
  - **Grab and Go** (free): Find one guaranteed minor artifact.
  - **Sell for Scrap:** Gain 150-250 gold.

*Mysterious Shrine:*
- A magical shrine offers power at a price:
  - **Pray for Strength:** All units gain +1 Strength permanently, but all units lose 20% current HP.
  - **Pray for Protection:** All units gain +1 Armor permanently, but lose 10 max mana.
  - **Pray for Wisdom:** Gain 2 random spells in spellbook, but one random equipped artifact is destroyed.
  - **Leave:** No effect.

*Scouting Report:*
- Scouts return with intelligence:
  - **Detailed Map:** Reveal the next floor's node map completely (normally partially hidden in later acts).
  - **Enemy Weakness:** The next battle encounter has one enemy squad start with a debuff (Weaken applied).
  - **Supply Cache:** Gain 100-200 gold.

*Ambush:*
- The player's army is ambushed. This is a forced combat encounter, but smaller scale:
  - 2-3 enemy squads attack
  - The player can only deploy 2 squads (rushed response)
  - Rewards are small but the player gains a "veteran" buff: +10 XP to all surviving units
  - If the player has a Scout or Ranger in their roster, the ambush is detected and becomes a normal battle instead (full deployment)

*Trader's Gamble:*
- A traveling merchant offers a deal:
  - **Buy a Mystery Box** (200g): Could contain a major artifact (30%), a minor artifact (50%), or nothing useful (20%).
  - **Trade an Artifact:** Exchange one owned artifact for a random different one. Could be upgrade or downgrade.
  - **Decline:** No effect.

*Wounded Soldiers:*
- The player encounters a damaged squad of allied soldiers:
  - **Take Them In:** Gain 2-3 free units, but they start at 50% HP and level 1.
  - **Share Supplies:** Lose 100g but gain a free minor artifact as thanks.
  - **Send Them Away:** No cost, no benefit.

**Event Design Philosophy:** Events should never feel like random noise. Each event should present a genuine decision where the "right" choice depends on the current state of the player's run. A player with full HP loves the Shrine's strength prayer. A player who just lost units loves the Wounded Soldiers event. This context-sensitivity is what makes events interesting rather than just loot dispensers.

### 5.7 Treasure Node

A simple reward node with no combat or choices. The player receives one of:
- 150-400 gold (scaled by act)
- A random minor artifact
- A spell scroll
- A bonus 75 XP to all units

Treasure nodes are rare and serve as "breather" nodes that reduce tension after hard battles. They also provide catch-up resources for players who took a hard path through the floor.

### 5.8 Recruitment Node

Recruitment nodes let the player hire a new commander and start a new squad. This is the primary way to expand the player's army beyond their initial squads.

**What happens:**
- The player is offered 2-3 commander candidates, each with different Leadership stats and starting abilities
- Hiring a commander costs 3000-5000 gold (cheaper than overworld mode but still significant)
- The new commander comes with an empty squad that the player must fill with units
- The player can also promote an existing high-level unit to commander status (if level 5+), which is free but removes them from their current squad

Recruitment nodes appear 1-2 times per act, making commander expansion a rare and valuable opportunity.

---

## 6. Combat Encounters

### Map Generation Per Encounter

Each battle generates its tactical map using the existing map generation systems. The biome is selected based on the current act and thematic context:

- **Act 1:** Grassland and Forest biomes (open and partially obstructed terrain)
- **Act 2:** Desert, Forest, and Mountain biomes (more varied tactical challenges)
- **Act 3:** Mountain, Swamp, and all biomes (full tactical variety)

The tactical_biome generator's profiles (obstacle density, cover, choke points, open space, rough terrain) create naturally varied battlefields. Forest maps force close-quarters engagements that favor melee, while Desert maps create long sightlines that reward ranged compositions. Mountain maps with choke points favor defensive play, and Swamp maps punish low-movement squads.

This biome-combat interaction is one of the strongest existing tactical systems and should be heavily leveraged. Players who see an upcoming Forest battle should consider their melee options. Players heading into a Desert battle should prepare ranged squads.

### Deployment

Before each battle, the player enters a deployment phase:
- The player sees the battle map and the enemy's approximate starting positions
- The player chooses which squads to deploy (limited by encounter's deployment cap)
- The player places their squads on valid deployment tiles (one side of the map)
- Squads not deployed serve as reserves and cannot participate

The existing squad deployment GUI and system handle this flow.

### Victory and Defeat

**Victory:** All enemy squads are destroyed. The player receives rewards (gold, XP, and encounter-specific drops). Surviving units retain their current HP -- they do NOT auto-heal. This makes HP a persistent resource between battles.

**Defeat:** All player squads are destroyed. The run ends. There is no retry mechanic for individual battles. This is a roguelike.

**Retreat (Future Feature):** The player could be given the option to retreat from a battle after 3 rounds, preserving surviving squads but forfeiting rewards and taking a morale penalty. This is noted as a future feature.

---

## 7. Enemy Scaling and Composition

### Scaling Model

Enemy difficulty scales along two axes: **act progression** and **encounter tier**.

**Act Progression** controls the base power level:
- Act 1: Enemies use powerMultiplier 0.7-0.9 (encounterdata levels 1-2)
- Act 2: Enemies use powerMultiplier 0.9-1.2 (encounterdata levels 2-4)
- Act 3: Enemies use powerMultiplier 1.2-1.5 (encounterdata levels 4-5)

**Encounter Tier** controls the specific encounter strength within an act:
- Standard battles: Base power for the act
- Elite battles: +1 difficulty level from base
- Boss battles: +2 difficulty levels from base (capped at level 5)

### Faction Composition

Each enemy faction has a distinct tactical identity drawn from the existing encounter definitions:

**Necromancers** (Defensive strategy)
- Squad preferences: Heavy melee front with magic support
- Tactical identity: Slow but durable. Their squads have high cover values and tanky front lines. Magic support units target the player's back row.
- Player counter: Ranged-heavy compositions that can bypass the front line, or column-attack units that pierce through.

**Cultists** (Expansionist strategy)
- Squad preferences: Magic-heavy compositions
- Tactical identity: High damage, low durability. Their squads are glass cannons that rely on killing the player's units before taking return damage.
- Player counter: Fast melee squads that close distance before magic can do its work, or Arcane Shield buff to absorb magic damage.

**Orcs** (Aggressor strategy)
- Squad preferences: Balanced melee/ranged with some magic
- Tactical identity: Aggressive and well-rounded. Their squads push forward and have good all-around stats. No obvious weakness but no overwhelming strength.
- Player counter: Requires good tactical play rather than counter-composition. Cover use and counterattack positioning are key.

**Bandits** (Raider strategy)
- Squad preferences: Ranged-heavy with melee support
- Tactical identity: High dexterity, lots of ranged units. They try to kite and pick off player units at distance.
- Player counter: High-movement squads that can close distance, or tank-heavy front lines with cover to absorb ranged fire while advancing.

**Beasts** (Territorial strategy)
- Squad preferences: All melee, all the time
- Tactical identity: High stats, pure melee swarm. They rush the player with overwhelming numbers of melee units. Individual units are weaker but there are many.
- Player counter: AoE spells to thin the swarm, ranged squads to damage before engagement, choke point positioning.

### Enemy Composition Generation

For each battle, the enemy force is generated using the following process:
1. Select faction (determined by act and run parameters)
2. Select difficulty level (determined by act and encounter tier)
3. Use the difficulty level's squadCount, minUnitsPerSquad, and maxUnitsPerSquad to determine army size
4. Use the encounter definition's squadPreferences to determine squad composition types (melee/ranged/magic)
5. Fill each squad with appropriate units from the monster roster, scaled by powerMultiplier
6. Assign formations based on squad type (defensive for tanky squads, ranged for archer squads, balanced for mixed)

This leverages the existing encounter data system almost entirely. The roguelike mode simply needs to select the right encounter definition and difficulty level for each battle.

---

## 8. Reward Systems

### Gold Economy

Gold is the universal currency. It flows in from combat and events and flows out at shops and recruitment nodes.

**Gold Sources:**
- Standard battle victory: 100 + (difficulty_level * 50) gold -- matching the existing formula
- Elite battle victory: 1.5x standard gold
- Boss victory: 3x standard gold + bonus based on remaining squad HP
- Events: Variable (0-250g depending on choices)
- Treasure nodes: 150-400g

**Gold Sinks:**
- Unit purchasing: 50-500g per unit (varies by unit power)
- Artifact purchasing: 200-1000g
- Spell scroll purchasing: 100-800g
- Commander recruitment: 3000-5000g
- Healing consumables: 100-300g
- Mana crystals: 150-300g
- Event costs: Variable

**Balance Target:** A player who wins every battle on a standard path should earn enough gold to make meaningful purchases each act but not enough to buy everything. Elite and boss paths should provide noticeably more gold, rewarding the player for taking harder fights.

### Experience Distribution

Experience works exactly as the existing system: XP is distributed equally among alive units in a squad after combat. The per-unit leveling with stat growth grades (S through F) creates natural unit specialization over the course of a run.

**XP Sources:**
- Combat victory: Base 50 + (difficulty_level * 25) XP per surviving unit
- Rest node training option: 50 XP to all units
- Event bonuses: Variable (10-75 XP)

**XP Pacing Target:** Units should reach level 2-3 by end of Act 1, level 4-6 by end of Act 2, and level 7-10 by end of Act 3. This ensures stat growth has a meaningful impact on combat performance throughout the run.

### Artifact Distribution

Artifacts are rare, powerful, and define the player's strategy for the run.

**Minor Artifacts** appear:
- As shop purchases (200-400g)
- As elite battle rewards (guaranteed 1 per elite)
- As event rewards (variable)
- From treasure nodes (uncommon)

**Major Artifacts** appear:
- As shop purchases (600-1000g, rare availability)
- As boss rewards (pick 1 of 3)
- As event rewards (rare, usually with significant trade-off)

**Artifact Limit:** The player has a limited number of artifact slots across all squads. Each squad can equip a limited number of artifacts (e.g., 2 per squad). This prevents artifact stacking and forces the player to distribute them strategically.

### Spell Acquisition

The player's commander starts with 3-4 basic spells in their spellbook. Additional spells are acquired through:
- Spell scroll purchases at shops
- Event rewards
- Boss rewards (choice of spell)

Spells are permanent additions to the spellbook. The constraint on spell usage is mana, not spell count.

---

## 9. Between-Battle Activities

### Army Management Hub

Between any node that is not a forced encounter, the player has access to the Army Management Hub, a persistent menu that allows:

**Squad Editor:**
- Full access to the existing squad editor with undo/redo
- Assign units to squads, rearrange formations
- Set squad formations (Balanced, Defensive, Offensive, Ranged)
- Review unit stats, levels, and growth grades

**Artifact Management:**
- Equip and unequip artifacts on squads
- View all owned artifacts and their effects
- Compare artifact options side by side

**Spellbook Review:**
- View available spells, mana costs, and effects
- Plan mana usage for upcoming battles
- No spell equipping needed -- all spells in the book are always available

**Roster Overview:**
- View all owned units, their squads, health, and levels
- See unit counts and roster capacity
- Identify units close to level-up

This hub uses the existing GUI systems (squad editor mode, artifact mode, unit purchase mode) and simply makes them accessible from the roguelike mode's between-node interface.

### Persistent State Between Battles

The following state persists throughout the entire run:
- **Unit HP:** Units retain their current health. There is no free heal between battles.
- **Mana:** The commander's mana pool carries over. Mana spent in battle is gone until restored.
- **Gold:** Accumulates across the run.
- **Artifacts:** Once acquired, artifacts remain until the run ends (or a shrine destroys one).
- **Spellbook:** Spells are permanent additions.
- **Unit levels and XP:** Progress is cumulative.
- **Dead units:** Gone permanently. No resurrection.

This persistence is what creates the strategic tension of the roguelike mode. Every battle has lasting consequences, and every purchase is an investment that must pay off across multiple future battles.

---

## 10. Spell and Mana Economy

### Mana as a Strategic Resource

Mana does not regenerate between battles (except through items or events). This transforms spells from "use whenever optimal in combat" to "use only when the strategic value justifies the mana cost across the remainder of the run."

**Starting Mana:** The commander begins each run with 60/60 mana (the existing ManaData MaxMana).

**Mana Recovery:**
- Mana Crystals (shop purchase): Restore 15-30 mana per crystal
- Rest node: No mana recovery (mana is more scarce than HP)
- Boss victory: Restore 20 mana
- Rare events: Variable mana restoration
- Future: Mana regeneration per floor (small amount, e.g., 5 mana per floor completed)

**Mana Spending Philosophy:**
- Cheap spells (5-10 mana: Spark, Singe, Frost Snap, Weaken) should be usable in most battles without guilt
- Mid spells (12-20 mana: Fireball, Chain Lightning, War Cry, Arcane Shield) should be reserved for tough encounters
- Expensive spells (25-40 mana: Blizzard, Obliterate, Firestorm, Absolute Zero) should be reserved for bosses and dire situations

This creates a natural tension: the player has powerful spells but using them means having fewer options later. The buff/debuff spells (War Cry, Arcane Shield, Weaken, Frost Slow) become particularly interesting because their lower mana cost makes them sustainable choices, while their tactical impact (stat modification for multiple turns) can be battle-defining.

### Spell Progression

**Starting Spellbook:** The commander begins with 3 spells: one cheap damage spell (Spark or Singe), one medium damage spell (Fireball or Lightning Bolt), and one buff/debuff spell (War Cry or Weaken). The specific starting spells are randomly selected.

**Spellbook Growth:** By the end of a run, the player should have 6-10 spells, providing a broad toolkit. The 22 available spells mean that no two runs will have the same full spellbook, contributing to run variety.

---

## 11. Artifact Strategy Layer

### Artifacts as Build-Defining Items

The existing artifact system is already designed for strategic depth. Minor artifacts provide incremental power (stat bonuses), while major artifacts provide game-changing active abilities. In roguelike mode, artifacts become the primary way a run's identity crystallizes.

**Example Build Archetypes (Emergent, Not Prescribed):**

*The Blitz Army:*
- Fleet Runner's Sandals on a fast melee squad
- Engagement Chains major artifact (bonus move on kill)
- Echo Drums major artifact (bonus movement after move+attack)
- Strategy: Rush enemies, kill one squad, chain into the next

*The Fortress:*
- Iron Bulwark and Sentinel's Plate on tank squads
- Arcane Shield spell on rotation
- Deadlock Shackles major artifact (skip enemy activation)
- Strategy: Outlast enemies through superior defense and counterattacks

*The Glass Cannon:*
- Berserker's Torc and Keen Edge Whetstone on DPS squads
- Twin Strike Banner major artifact (double attack)
- War Cry spell to stack damage buffs
- Strategy: Kill before being killed, alpha-strike critical targets

*The Controller:*
- Saboteur's Hourglass (reduce enemy movement)
- Frost Slow and Weaken debuff spells
- Chain of Command Scepter (pass turns between squads)
- Strategy: Dictate the pace of battle, deny enemy options

These builds are not chosen from a menu. They emerge from which artifacts, spells, and units the run offers and which the player prioritizes acquiring.

### Artifact Charge Management

Major artifact charges are per-battle in the current system. This remains unchanged for roguelike mode. The strategic layer comes from choosing WHICH battle to use a limited charge like Twin Strike or Deadlock Shackles. Using Deadlock Shackles in a standard battle might make that fight trivial, but then it is unavailable for the next elite or boss encounter.

---

## 12. Run Identity and Replayability

### What Makes Each Run Different

**Faction Matchup:** The enemy faction(s) assigned to each act change every run. Fighting Necromancers in Act 1 requires a very different opening strategy than fighting Beasts. The second faction introduced in Act 2 adds another layer of adaptation.

**Node Map Layout:** The branching path structure means the player faces different node sequences each run. One run might offer an early shop followed by a rest, while another might force two consecutive battles before any recovery.

**Shop Inventory:** Randomized unit, artifact, and spell availability means the player cannot plan a fixed build before starting. They must adapt to what is offered.

**Event Variance:** The 7+ event types with multiple options create different narratives each run. Getting the Mysterious Shrine early is a very different experience than getting it late.

**Starting Conditions:** The starting army (2 squads with basic units) varies between runs. Different starting units push the player toward different early strategies.

**Biome Variation:** The randomly selected biomes for each battle create different tactical puzzles even for the same enemy composition. Fighting Orcs in a Forest plays very differently than fighting them on Grassland.

### Replayability Hooks

**Run Scoring:** Each run receives a final score based on:
- Floors completed
- Battles won
- Boss battles completed
- Total gold earned
- Total enemies defeated
- Remaining army strength
- Difficulty level

**Run Statistics:** After each run (win or loss), the player sees detailed statistics: favorite unit (most kills), most used spell, total damage dealt, artifacts collected, gold spent. This encourages players to try different approaches to see different statistics.

**Challenge Modifiers (Future):** Additional optional constraints that modify the run for experienced players:
- Single Commander: Cannot recruit additional commanders
- Pacifist Mage: Cannot attack physically, only spells
- Iron Man: No rest nodes available
- Gauntlet: All battles become elite

---

## 13. Difficulty and Balance Philosophy

### Difficulty Selection

The existing difficulty system (Easy, Medium, Hard) applies to roguelike mode. It modifies the same master knobs:

**Easy:**
- CombatIntensity 0.8: Enemy squads are weaker
- EncounterSizeOffset -1: Fewer enemy squads per battle
- AICompetence 0.8: Enemy AI makes suboptimal positioning choices
- Gold rewards +20%: More purchasing power
- Rest healing 75% of missing HP instead of 50%

**Medium:**
- All values at 1.0: The balanced experience as designed in this document

**Hard:**
- CombatIntensity 1.2: Enemy squads are stronger
- EncounterSizeOffset +1: More enemy squads per battle
- AICompetence 1.2: Enemy AI makes better tactical decisions
- Gold rewards -15%: Tighter economy
- Rest healing 40% of missing HP

### Balance Principles

**The player should win roughly 60-70% of runs on Medium difficulty.** This means some runs will end in defeat even with good play. Bad luck (poor shop offerings, tough encounter sequences) should end maybe 10% of runs, while poor strategic decisions should account for the other 20-30% of losses.

**No single unit type or artifact should dominate.** If every winning run uses the same artifact, that artifact is too strong. If every winning run avoids a particular unit, that unit is too weak. Balance should reward diverse army compositions.

**Elite battles should feel optional but tempting.** The rewards for elite battles should be good enough that a strong player wants to take them, but the difficulty should be high enough that a weak army should skip them. This creates a natural difficulty-reward curve that the player controls.

**Bosses should require preparation, not luck.** A well-built army with good tactical play should beat bosses consistently. Bosses should test whether the player built well, not whether they got lucky with random rolls.

**HP persistence prevents steamrolling.** Because units do not auto-heal, a player who wins battles by brute force (losing lots of HP each fight) will eventually run out of resources. Efficient play -- winning battles with minimal damage -- is rewarded with greater longevity.

---

## 14. Leveraging Existing Systems

### Direct Usage (No Modification Needed)

The following existing systems can be used directly in roguelike mode:

- **Combat System:** Turn-based combat, attack resolution, counterattacks, damage calculation, simultaneous resolution -- all work as-is
- **Squad System:** 3x3 formation grid, multi-cell units, formation types, cover mechanics, leader abilities -- all work as-is
- **Unit Templates:** All 23 unit types with their stats, roles, attack types, growth rates -- used directly for both player purchasing and enemy generation
- **Spell System:** All 22 spells with mana costs, targeting, damage/buff/debuff effects -- used directly
- **Artifact System:** All 7 minor and 6 major artifacts with their behaviors -- used directly
- **Experience System:** Per-unit XP with stat growth grades -- used directly
- **Map Generation:** tactical_biome generator with 5 biomes -- used directly for battle map creation
- **Encounter Data:** 15 encounter definitions across 5 factions at 3 tiers -- used directly for enemy composition
- **Difficulty System:** 3 difficulty presets with master knobs -- used directly
- **Squad Editor GUI:** Full squad editing with undo/redo -- used directly in between-battle hub
- **Unit Purchase GUI:** Unit purchasing interface -- used directly at shop nodes
- **Artifact Management GUI:** Artifact equipping/viewing -- used directly in between-battle hub
- **Spell Casting GUI:** Spell selection and targeting in combat -- used directly

### Extensions Needed

The following existing systems need modest extensions:

- **Unit Roster:** Exists and tracks owned units. Needs a "unit shop" layer that generates available-for-purchase unit lists with prices.
- **Gold Economy:** The gold reward formula exists. Needs a persistent gold tracker across the run (not just per-battle).
- **Mana System:** ManaData already supports current/max mana. Needs to persist across battles (the comment in the code already says it should).
- **Encounter Generation:** The encounter definitions exist. Needs a selection layer that picks appropriate encounters based on act, floor, and difficulty.
- **Artifact Inventory:** The inventory system exists. Needs a "loot drop" layer that generates artifact rewards.

### New Systems Required

The following are genuinely new systems for roguelike mode:

- **Node Map:** The branching floor map with node types, path connections, and visual display
- **Run State:** Persistent state tracker for the current run (act, floor, gold, mana, roster, artifacts, spellbook, statistics)
- **Event System:** Event definitions, choice presentation, outcome resolution
- **Shop Generation:** Inventory generation logic for unit/artifact/spell/consumable shops
- **Enemy Scaling:** Logic to select and scale encounter definitions based on run progress
- **Reward Distribution:** Post-battle reward calculation and presentation
- **Run Summary:** End-of-run statistics and scoring display

---

## 15. Player Progression Within a Run

### Power Curve

The player's power should grow throughout the run, but never outpace the enemy scaling. The goal is for the player to feel stronger while the challenges remain engaging.

**Early Run (Act 1):**
- 2 squads, basic units, 3 spells, no artifacts
- Player is learning the current run's faction matchup
- Decisions feel small but impactful (which units to buy first)

**Mid Run (Act 2):**
- 3-4 squads, mixed unit quality, 5-7 spells, 2-4 artifacts
- Build identity is forming (melee-focused? magic-focused? balanced?)
- Decisions feel significant (which artifact synergizes best, spend mana or save it)

**Late Run (Act 3):**
- 4-5 squads, high-level units, 7-10 spells, 4-6 artifacts
- Build is defined, execution matters more than decisions
- Decisions are about resource conservation (save mana for boss, save artifact charges)

### Army Growth Path

**Squads:** Start with 2, recruit commander for 3rd in Act 1-2, possibly 4th in Act 2-3. More squads means more tactical options but more units to maintain.

**Units:** Start with basic units, progressively replace with stronger purchased units. Dead units create holes that must be filled. The 23-unit roster provides enough variety that unit choices remain interesting throughout.

**Levels:** Units gain 1-2 levels per act through combat XP. High-growth-grade stats (+S, +A) become noticeably stronger, making growth grades a factor in which units to invest in (keep alive, give XP).

**Artifacts:** Acquired gradually, each one opening new tactical possibilities. By mid-run, the player should have enough artifacts to have made meaningful equipping decisions.

**Spells:** The spellbook grows from 3 to 8-10 spells, providing a wider tactical toolkit. Mana management becomes more interesting with more spell options.

---

## 16. Meta-Progression (Future)

This section describes future features that would add persistence between runs. These are NOT part of the initial implementation but inform design decisions.

**Unlock System:** Completing runs (or achieving milestones) could unlock new starting conditions:
- New starting squads with different unit compositions
- New starting spells
- New artifacts added to the global pool
- New challenge modifiers

**Commander Roster:** A persistent roster of commanders that the player can choose from at run start, each with different Leadership stats and starting abilities.

**Bestiary:** A catalog of encountered enemies with tactical notes, encouraging the player to engage with all five factions.

These features would extend the game's lifespan significantly but depend on a save/load system that may not exist yet.

---

## 17. Future Expansion Possibilities

### Near-Term Extensions

**Consumable Integration:** The consumable data file already defines healing potions, protection potions, and speed potions. These could be integrated as battle-use items: before or during combat, apply a consumable to a squad for a temporary effect. This adds another resource management layer.

**Formation Bonuses:** Specific formation configurations (e.g., all tanks in front row, all ranged in back row) could provide passive bonuses. This rewards thoughtful squad building beyond just placing strong units.

**Terrain Hazards:** Certain tile types on battle maps could have effects: lava tiles deal damage to units that stop on them, mud tiles cost extra movement, high ground tiles provide accuracy bonuses. The biome system already differentiates terrain profiles; this extends them with mechanical impact.

### Medium-Term Extensions

**Spell Crafting:** Instead of finding complete spells, the player finds spell components (element + shape + effect) and combines them. This multiplies spell variety dramatically and adds a between-battle crafting activity.

**Unit Promotion:** At certain level thresholds (5, 10), units could be promoted to advanced classes. A Knight could become a Crusader (higher stats) or a Guardian (better cover). This adds another layer of progression and build variety.

**Rival Commander:** A persistent enemy commander that appears at fixed points in the run, growing stronger each time. Creates a recurring antagonist and tests different aspects of the player's army at different stages.

### Long-Term Vision

**Daily Challenge Runs:** A fixed-seed run that all players attempt on the same day, with a shared leaderboard. The fixed seed means optimal strategies can be discussed and compared.

**Multiplayer Roguelike:** Two players progress through parallel node maps and fight a boss together at the end of each act. Complementary army builds are rewarded.

**Endless Mode:** After defeating the Act 3 boss, the player can continue into endlessly scaling floors. How far can the army go?

---

## Appendix A: Existing Data Reference

### Unit Roster (23 units)

| Unit | Role | Attack Type | Size | Notes |
|------|------|-------------|------|-------|
| Knight | Tank | MeleeRow | 1x1 | High cover (0.4), slow |
| Paladin | Support | MeleeRow | 1x1 | Balanced stats, good leader candidate |
| Fighter | Tank | MeleeColumn | 1x1 | Fast tank, column pierce |
| Warrior | DPS | MeleeRow | 1x1 | Pure damage, no cover |
| Swordsman | DPS | MeleeColumn | 1x1 | Highest dexterity standard unit |
| Spearman | Tank | MeleeColumn | 1x1 | Extended attack range (2) |
| Rogue | DPS | MeleeColumn | 1x1 | Very fast (speed 5), glass cannon |
| Assassin | DPS | MeleeColumn | 1x1 | Highest dexterity in game (60) |
| Scout | Support | Ranged | 1x1 | Fast ranged support |
| Archer | DPS | Ranged | 1x1 | Long range (4), high weapon |
| Ranger | Support | Magic | 1x1 | Diagonal magic pattern, versatile |
| Crossbowman | DPS | Ranged | 1x1 | Mid-range (3), balanced |
| Marksman | DPS | Ranged | 1x1 | Long range (4), extreme dexterity |
| Wizard | DPS | Magic | 1x1 | Wide 2-row magic pattern |
| Sorcerer | DPS | Magic | 1x1 | 1-row magic pattern, mid-range |
| Mage | Support | Magic | 1x1 | Support mage, some cover |
| Cleric | Support | Magic | 1x1 | Back-row magic, high leadership (40) |
| Priest | Support | Magic | 1x1 | Narrow magic, highest leadership (45) |
| Warlock | DPS | Magic | 1x1 | Corner-targeting pattern, long range |
| Battle Mage | Support | MeleeColumn | 1x1 | Hybrid melee/magic, balanced stats |
| Orc Warrior | Tank | MeleeRow | 2x1 | Multi-cell, massive cover (0.45) |
| Goblin Raider | DPS | MeleeRow | 1x1 | Fast (4), high leadership, cheap |
| Ogre | Tank | MeleeRow | 2x2 | Largest unit, highest cover (0.5) |
| Skeleton Archer | DPS | Magic | 1x1 | Cross magic pattern, fragile |

### Spell List (22 spells)

| Spell | Cost | Type | Effect | Shape |
|-------|------|------|--------|-------|
| Spark | 5 | Single | 15 dmg | - |
| Singe | 6 | Single | 18 dmg | - |
| Frost Snap | 8 | AoE | 12 dmg | Square 1 |
| Lightning Bolt | 10 | Single | 30 dmg | - |
| Ice Lance | 12 | AoE | 25 dmg | Line 4 |
| Chain Lightning | 14 | AoE | 22 dmg | Line 3 |
| Miasma | 14 | AoE | 10 dmg | Rect 5x2 |
| Fireball | 15 | AoE | 20 dmg | Circle 1 |
| Scalding Gust | 16 | AoE | 16 dmg | Cone 3 |
| Wall of Flame | 18 | AoE | 18 dmg | Line 5 |
| Fog of Ruin | 20 | AoE | 12 dmg | Rect 3x4 |
| Thunder Cone | 20 | AoE | 20 dmg | Cone 4 |
| Blizzard | 25 | AoE | 15 dmg | Square 2 |
| Noxious Eruption | 25 | AoE | 18 dmg | Cone 5 |
| Immolation | 30 | AoE | 25 dmg | Circle 2 |
| Obliterate | 30 | Single | 55 dmg | - |
| Firestorm | 35 | AoE | 22 dmg | Circle 3 |
| Absolute Zero | 40 | AoE | 20 dmg | Square 3 |
| War Cry | 10 | Buff | +4 Str/3t | Single |
| Arcane Shield | 12 | Buff | +3 Armor/3t | Single |
| Weaken | 8 | Debuff | -3 Str/2t | Single |
| Frost Slow | 10 | Debuff | -2 Dex, -1 Move/2t | Single |

### Artifact List (13 artifacts)

**Minor (7):** Iron Bulwark (+2 Armor), Keen Edge Whetstone (+2 Weapon), Fleet Runner's Sandals (+1 Move), Marksman's Scope (+1 Range), Berserker's Torc (+2 Str/-1 Armor), Sentinel's Plate (+2 Armor/-1 Move), Duelist's Gloves (+1 Dex/+1 Str)

**Major (6):** Twin Strike Banner (extra attack), Chain of Command Scepter (pass action), Forced Engagement Chains (bonus move on kill), Saboteur's Hourglass (reduce enemy movement), Deadlock Shackles (skip enemy activation), Echo Drums (bonus movement after move+attack)

### Encounter Definitions (15 encounters)

5 factions x 3 tiers (common, elite, boss) = 15 pre-defined encounter configurations in encounterdata.json.
