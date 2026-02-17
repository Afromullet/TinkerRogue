# Game Data Overview

The `assets/gamedata/` folder contains all JSON files that define game templates, tuning parameters, and configuration. Every file is loaded at startup by `templates.ReadGameData()` (called from `GameBootstrap.LoadGameData()`) and parsed into in-memory registries that the rest of the game reads from.

---

## File Categories

The 13 JSON files fall into two broad groups:

### Template / Definition Files

These define the "what" of the game — the creatures, items, encounters, and map nodes that can exist.

| File | Purpose |
|------|---------|
| `monsterdata.json` | Unit templates for all recruitable and enemy creatures |
| `spelldata.json` | Spell definitions including damage, AoE shapes, buffs, and debuffs |
| `consumabledata.json` | Consumable item templates (potions) with stat effects and durations |
| `creaturemodifiers.json` | Named stat modifier templates that can be applied to creatures |
| `nodeDefinitions.json` | Overworld node types — threats, settlements, and fortresses |
| `encounterdata.json` | Encounter definitions tying factions to difficulty levels and squad compositions |
| `minor_artifacts.json` | Minor artifact definitions with passive stat modifiers |
| `major_artifacts.json` | Major artifact definitions with unique activated abilities |

### Configuration / Tuning Files

These define the "how" — numerical tuning, AI weights, and difficulty scaling.

| File | Purpose |
|------|---------|
| `difficultyconfig.json` | Difficulty presets controlling combat intensity, AI competence, and encounter size |
| `aiconfig.json` | AI behavior weights per role and threat calculation parameters |
| `powerconfig.json` | Squad power scoring — role multipliers, ability values, and composition bonuses |
| `overworldconfig.json` | Overworld simulation tuning — threat growth, faction AI, victory conditions, and strategy bonuses |
| `influenceconfig.json` | Influence system tuning — magnitude, synergy/competition bonuses, and suppression multipliers |

---

## Per-File Summaries

### monsterdata.json

Defines every unit type in the game. Each entry specifies a unit's name, sprite, base attributes (strength, dexterity, magic, leadership, armor, weapon), physical dimensions (width/height for multi-tile units), combat role (Tank, DPS, Support), attack type (MeleeRow, MeleeColumn, Ranged, Magic), attack range, movement speed, cover values, and stat growth rates (S through F). Contains 24 unit types ranging from melee fighters to ranged archers and spellcasters.

### spelldata.json

Defines all spells available to commanders. Each spell has an ID, mana cost, target type (single or AoE), and effect type (damage, buff, or debuff). Damage spells specify AoE shapes (Circle, Square, Line, Rectangle, Cone) with size parameters. Buff/debuff spells specify stat modifiers and durations. All spells include visual effect type and duration. Contains 22 spells across fire, ice, electricity, and cloud visual themes.

### consumabledata.json

Defines consumable items (currently potions). Each entry specifies stat modifications (health, protection, movement speed, dodge chance, etc.) and a duration in turns. Contains 3 potions: healing, protection, and speed.

### creaturemodifiers.json

Defines named stat modifier packages that can be applied to creatures to create variants. Each modifier adjusts a subset of combat stats (attack bonus, health, armor class, protection, movement speed, dodge chance, damage bonus). Contains 5 modifiers: Fast, Strong, Elusive, Dire, and Sturdy.

**Note:** This file is not currently loaded at startup — it is available for future use.

### nodeDefinitions.json

Defines all overworld node types organized into three categories: threat, settlement, and fortress. Threat nodes represent enemy locations and include overworld behavior (growth rate, influence radius, ability to spawn child nodes) and a faction ID. Settlement and fortress nodes represent player-buildable structures with resource costs (iron, wood, stone). Also defines a default fallback node. Contains 9 node types: 5 threats, 3 settlements, 1 fortress.

### encounterdata.json

The largest configuration file, combining three concerns: faction-to-strategy mappings, difficulty level definitions, and encounter definitions. Factions (Necromancers, Cultists, Orcs, Bandits, Beasts) are each assigned a strategy. Five difficulty levels define power multipliers, squad counts, and unit count ranges. Encounter definitions tie a node ID to a specific encounter type with squad composition preferences (melee/ranged/magic), difficulty, tags, and faction. Contains 15 encounter definitions (3 per faction at common/elite/boss tiers).

### minor_artifacts.json

Defines minor artifacts — passive items that grant simple stat modifiers when equipped. Each artifact has an ID, name, description, and one or more stat modifiers (which can be positive or negative for trade-off effects). Contains 7 minor artifacts affecting armor, weapon, movement speed, attack range, strength, and dexterity.

### major_artifacts.json

Defines major artifacts — powerful items with unique activated abilities rather than passive stat bonuses. Each artifact has an ID, name, description, and tier. The actual behavior logic is implemented in code, keyed by the artifact ID. Contains 6 major artifacts providing abilities like double attacks, action passing, bonus movement, and enemy disruption.

### difficultyconfig.json

Defines the global difficulty presets. Each preset specifies multipliers for combat intensity, overworld pressure, and AI competence, plus an encounter size offset. Also specifies the default difficulty. Contains 3 presets: Easy, Medium, Hard.

### aiconfig.json

Controls AI decision-making during tactical combat. Defines threat calculation parameters (flanking range, isolation threshold, retreat threshold), per-role behavior weights that control how the AI positions units (melee weight, support weight), support layer configuration (heal radius), and shared weights for ranged and positional scoring. Contains behaviors for 3 roles: Tank, DPS, Support.

### powerconfig.json

Defines the squad power rating formula used for encounter balancing and army composition scoring. Includes evaluation profiles (offensive/defensive/utility weight distribution), per-role power multipliers, ability power values, composition diversity bonuses (rewarding squads with varied unit types), and a leader bonus multiplier.

### overworldconfig.json

The primary tuning file for the overworld simulation layer. Covers threat growth (containment, max intensity, child node spawning), faction AI tick duration and territory limits, strength thresholds, victory/loss conditions, faction intent scoring formulas, threat spawn probabilities, map dimensions, player node placement limits, and per-strategy scoring bonuses (Expansionist, Aggressor, Raider, Defensive, Territorial).

### influenceconfig.json

Tunes the influence propagation system on the overworld map. Controls base magnitude, default player node influence values, and three interaction types: synergy (bonus when friendly influences overlap), competition (penalty when enemy factions overlap), and suppression (how player nodes counter threat influence, with per-node-type multipliers).

---

## Cross-File Relationships

Several files reference shared identifiers that must stay in sync:

- **Faction IDs** — `nodeDefinitions.json` assigns a `factionId` to each threat node (e.g., "Necromancers", "Bandits"). `encounterdata.json` maps these same faction IDs to strategies and encounter definitions. `overworldconfig.json` defines strategy scoring bonuses referenced by the strategy names in `encounterdata.json`.

- **Role names** — `monsterdata.json` assigns each unit a `role` (Tank, DPS, Support). Both `aiconfig.json` (behavior weights) and `powerconfig.json` (power multipliers) define entries keyed by these same role names.

- **Node IDs** — `nodeDefinitions.json` defines node IDs (e.g., "necromancer", "banditcamp", "watchtower"). `encounterdata.json` uses these as encounter definition IDs. `influenceconfig.json` references player node type IDs in its suppression multipliers.

- **Strategy names** — `encounterdata.json` assigns each faction a strategy (e.g., "Defensive", "Expansionist"). `overworldconfig.json` defines scoring bonuses for each of these strategy names.

- **Spell IDs** — `spelldata.json` defines spell IDs (e.g., "fireball", "war_cry"). These are referenced by the spell system in code when commanders cast spells.

- **Artifact IDs** — `major_artifacts.json` defines artifact IDs (e.g., "twin_strike", "echo_drums"). The code in `gear/artifactbehaviors_activated.go` implements behavior logic keyed by these IDs.

- **Ability names** — `powerconfig.json` defines ability values (e.g., "Rally", "Heal", "Fireball"). These are referenced by the power calculation system when evaluating squad strength.

---

## Loading Order

Files are loaded in a specific order during startup because some validations depend on data from earlier files:

1. `difficultyconfig.json` — loaded first since other systems reference difficulty settings
2. `monsterdata.json` — unit templates
3. `consumabledata.json` — consumable templates
4. `nodeDefinitions.json` — loaded before encounters so encounter validation can check node IDs
5. `encounterdata.json` — validates against node definitions
6. `aiconfig.json` — AI configuration
7. `powerconfig.json` — power scoring configuration
8. `overworldconfig.json` — overworld simulation tuning
9. `influenceconfig.json` — influence system tuning
10. `spelldata.json` — spell definitions
11. `minor_artifacts.json` and `major_artifacts.json` — artifact definitions

Loading failures cause the game to panic at startup rather than proceeding with missing data.
