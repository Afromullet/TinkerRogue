# Game Data Overview

**Last Updated:** 2026-04-21

The `assets/gamedata/` folder contains all JSON files that define game templates, tuning parameters, and configuration. Most files are loaded at startup by `templates.ReadGameData()` (called from the bootstrap layer). A handful of subsystem-specific files (raid, perks, artifacts, combat, spells, artifacts-inventory) are loaded by their owning packages through dedicated loaders.

---

## File Categories

The JSON files fall into three broad groups:

### Template / Definition Files

These define the "what" of the game — the creatures, items, spells, perks, encounters, and map nodes that can exist.

| File | Purpose |
|------|---------|
| `monsterdata.json` | Unit templates for all recruitable and enemy creatures |
| `spelldata.json` | Spell definitions including damage, AoE shapes, buffs, and debuffs |
| `unitspells.json` | Maps unit types to the spell IDs they can cast as squad leaders |
| `consumabledata.json` | Consumable item templates (potions). **Currently unused** — reserved for future use |
| `creaturemodifiers.json` | Named stat modifier templates that can be applied to creatures. **Currently unused** — reserved for future use |
| `nodeDefinitions.json` | Overworld node types — threats, settlements, and fortresses |
| `encounterdata.json` | Encounter definitions tying factions to difficulty levels and squad compositions |
| `minor_artifacts.json` | Minor artifact definitions with passive stat modifiers |
| `major_artifacts.json` | Major artifact definitions with unique activated abilities |
| `perkdata.json` | Squad perk definitions — tier, category, roles, unlock cost, exclusivity |
| `raidarchetypes.json` | Named squad compositions used by the garrison-raid generator |

### Configuration / Tuning Files

These define the "how" — numerical tuning, AI weights, and difficulty scaling.

| File | Purpose |
|------|---------|
| `gameconfig.json` | Global player/commander/combat/display tuning — attributes, starting resources, roster limits, mana pool, hit/crit caps, tile size |
| `difficultyconfig.json` | Difficulty presets controlling combat intensity, AI competence, and encounter size |
| `aiconfig.json` | AI behavior weights per role and threat-calculation parameters |
| `powerconfig.json` | Squad power scoring — role multipliers, ability values, composition bonuses |
| `overworldconfig.json` | Overworld simulation tuning — threat growth, faction AI, victory conditions, strategy bonuses |
| `influenceconfig.json` | Influence system tuning — magnitude, synergy/competition bonuses, suppression multipliers |
| `mapgenconfig.json` | Procedural generator parameters for rooms-and-corridors, cavern, overworld, and garrison-raid layouts |
| `raidconfig.json` | Raid tuning — deploy/reserve limits, floor count, recovery percentages, alert thresholds, reward scaling |
| `perkbalanceconfig.json` | Per-perk numeric tuning values (multipliers, thresholds, bonuses) |
| `artifactbalanceconfig.json` | Per-artifact numeric tuning values for activated behaviors |
| `combatbalanceconfig.json` | Combat mechanic tuning (e.g., counterattack damage multiplier and hit penalty) |

### Name Generation

| File | Purpose |
|------|---------|
| `namedata.json` | Syllable pools and format template used by `templates/namegen` to produce unit names like "Karathos the Warrior" |

---

## Per-File Summaries

### monsterdata.json

Defines every unit type in the game. Each entry specifies a unit's name, sprite, base attributes (strength, dexterity, magic, leadership, armor, weapon), physical dimensions (width/height for multi-tile units), combat role (Tank, DPS, Support), attack type (MeleeRow, MeleeColumn, Ranged, Magic), attack range, movement speed, cover values, and per-stat growth rates (S through F).

### spelldata.json

Defines all spells available in the game. Each spell has an ID, mana cost, target type (single or AoE), and effect type (damage, buff, or debuff). Damage spells specify AoE shapes (Circle, Square, Line, Rectangle, Cone) with size parameters. Buff/debuff spells specify stat modifiers and durations. All spells include visual effect type and duration.

### unitspells.json

Maps each unit type to the spell IDs its squad can cast when that unit is the squad leader. Magic-focused units get powerful caster spells as their core identity; non-caster units get cheaper, thematic spells. Spell IDs reference entries in `spelldata.json`. At squad creation, the leader's spell list is filtered against the owning player's unlocked-spell library (see `tactical/powers/progression`) before being attached as `SpellBookComponent`.

### consumabledata.json

Defines consumable items (currently potions) with stat modifications and durations. **This file is present in the asset folder but not currently loaded at startup.**

### creaturemodifiers.json

Defines named stat modifier packages (Fast, Strong, Elusive, Dire, Sturdy) that can be applied to creatures to create variants. **This file is present in the asset folder but not currently loaded at startup.**

### nodeDefinitions.json

Defines all overworld node types organized into three categories: threat, settlement, and fortress. Threat nodes carry overworld behavior data (growth rate, influence radius, ability to spawn child nodes) and a faction ID. Settlement and fortress nodes represent player-buildable structures with resource costs (iron, wood, stone). Also defines a default fallback node.

### encounterdata.json

The largest configuration file, combining three concerns: faction-to-strategy mappings, difficulty level definitions, and encounter definitions. Factions are each assigned a strategy. Difficulty levels define power multipliers, squad counts, and unit count ranges. Encounter definitions tie a node ID to a specific encounter type with squad composition preferences (melee/ranged/magic), difficulty, tags, and faction.

### minor_artifacts.json

Defines minor artifacts — passive items that grant simple stat modifiers when equipped. Each artifact has an ID, name, description, and one or more stat modifiers (which can be positive or negative for trade-off effects).

### major_artifacts.json

Defines major artifacts — powerful items with unique activated abilities rather than passive stat bonuses. Each artifact has an ID, name, description, and tier. The actual behavior logic is implemented in code (`tactical/powers/artifacts/behaviors.go`), keyed by the artifact ID.

### perkdata.json

Defines all squad perks. Each perk has an `id`, `name`, `description`, `tier`, `category`, a list of `roles` it is eligible for, a list of `exclusiveWith` perks it cannot be equipped alongside, and an `unlockCost` (skill points). Behavior logic is implemented in `tactical/powers/perks/behaviors.go`, dispatched via the perk registry. Loaded by `perks.LoadPerks` (path `gamedata/perkdata.json`).

### raidarchetypes.json

Named squad compositions used by the garrison-raid generator when populating rooms and reserves. Each archetype specifies a list of units with monster type, 3×3 grid position, size, and leader flag, plus a list of `preferredRooms`. Loaded by `raid.LoadArchetypeData` in the game_main bootstrap.

### gameconfig.json

Global tuning for the player and core combat. Covers player starting attributes and resources, roster size limits, commander movement/cost/starting-mana, faction-AI starting resources, combat constants (default movement, attack range, hit rate caps, crit/dodge caps, capacity, magic resist, crit damage bonus), and display constants (tile pixels, scale factor, padding, zoom, map dimensions). Exposed at runtime as `templates.GameConfig`.

### difficultyconfig.json

Defines global difficulty presets. Each preset specifies multipliers for combat intensity, overworld pressure, and AI competence, plus an encounter-size offset. Also specifies the default difficulty.

### aiconfig.json

Controls AI decision-making during tactical combat. Defines threat-calculation parameters (flanking range, isolation threshold, retreat threshold), per-role behavior weights (meleeWeight, supportWeight) that control how the AI positions units, support-layer configuration (heal radius), and shared weights for ranged and positional scoring.

### powerconfig.json

Defines the squad power rating formula used for encounter balancing and army composition scoring. Includes evaluation profiles (offensive/defensive/utility weight distribution), per-role power multipliers, ability power values, composition diversity bonuses, and a leader bonus multiplier.

### overworldconfig.json

The primary tuning file for the overworld simulation layer. Covers threat growth (containment, max intensity, child-node spawning), faction AI tick duration and territory limits, strength thresholds, victory/loss conditions, faction intent scoring formulas, threat spawn probabilities, map dimensions, player node placement limits, and per-strategy scoring bonuses.

### influenceconfig.json

Tunes the influence propagation system on the overworld map. Controls base magnitude, default player-node influence values, and three interaction types: synergy (friendly overlap bonus), competition (enemy overlap penalty), and suppression (player nodes countering threat influence, with per-node-type multipliers).

### mapgenconfig.json

Optional — if missing, generators fall back to code defaults. Provides parameters for the `rooms_corridors`, `cavern`, `overworld`, and `garrison_raid` generators (room sizes, density thresholds, noise scales, POI counts, garrison room-size bounds, etc.).

### raidconfig.json

All raid tuning parameters: max deployable squads per encounter, default floor count, reserves per floor, recovery percentages (deployed / reserve / rest-room), alert-level thresholds and reserve-activation flags, and per-room reward scaling (gold, XP, arcana, skill). Loaded by `raid.LoadRaidConfig`.

### perkbalanceconfig.json

Per-perk tuning values keyed by camelCase perk name. Each entry carries only the numeric fields that specific perk needs (e.g., `braceForImpact.coverBonus`, `executionersInstinct.hpThreshold` + `critBonus`). Loaded into `perks.PerkBalance` at startup; behaviors reference `PerkBalance.*` instead of hardcoded constants.

### artifactbalanceconfig.json

Per-artifact tuning values for activated behaviors (e.g., `saboteursHourglass.movementReduction`). Loaded into `artifacts.ArtifactBalance`.

### combatbalanceconfig.json

Combat mechanic tuning — currently contains the counterattack `damageMultiplier` and `hitPenalty`. Loaded into `combatcore.CombatBalance`.

### namedata.json

Configures the procedural name generator: a format template (`{name} the {type}`), syllable-count bounds, and one or more named pools (currently just `default`) of prefix / middle / suffix syllables. Used wherever a unit is created with a display name.

---

## Cross-File Relationships

Several files reference shared identifiers that must stay in sync:

- **Faction IDs** — `nodeDefinitions.json` assigns a `factionId` to each threat node. `encounterdata.json` maps these same faction IDs to strategies and encounter definitions. `overworldconfig.json` defines strategy scoring bonuses referenced by the strategy names in `encounterdata.json`.

- **Role names** — `monsterdata.json` assigns each unit a `role` (Tank, DPS, Support). Both `aiconfig.json` (behavior weights) and `powerconfig.json` (power multipliers) define entries keyed by these same role names. `perkdata.json` also keys perk eligibility by these role names.

- **Node IDs** — `nodeDefinitions.json` defines node IDs (e.g., `necromancer`, `banditcamp`, `watchtower`). `encounterdata.json` uses these as encounter definition IDs. `influenceconfig.json` references player node type IDs in its suppression multipliers.

- **Strategy names** — `encounterdata.json` assigns each faction a strategy (e.g., `Defensive`, `Expansionist`). `overworldconfig.json` defines scoring bonuses for each of these strategy names.

- **Unit type names** — `monsterdata.json` defines unit type names (e.g., `Knight`, `Wizard`). `unitspells.json` and `raidarchetypes.json` use these same names to assign spells and populate garrison squads.

- **Spell IDs** — `spelldata.json` defines spell IDs (e.g., `fireball`, `war_cry`). `unitspells.json` references these IDs to define which spells each unit type can cast as a squad leader. Spells unlocked in `perkdata.json` / progression gate whether a spell actually makes it into a squad's `SpellBookComponent`.

- **Artifact IDs** — `major_artifacts.json` / `minor_artifacts.json` define artifact IDs. `artifactbalanceconfig.json` keys tuning values by artifact-specific names. The behavior registry in `tactical/powers/artifacts/behaviors.go` keys handlers by artifact ID.

- **Perk IDs** — `perkdata.json` defines perk IDs. `perkbalanceconfig.json` keys tuning values by camelCase perk name. The behavior registry in `tactical/powers/perks/behaviors.go` keys handlers by perk ID and registers hook entries into the dispatcher.

- **Archetype room types** — `raidarchetypes.json` uses `preferredRooms` strings that must match garrison room-type constants in `world/garrisongen` (e.g., `barracks`, `guard_post`, `mage_tower`). `mapgenconfig.json` also keys per-room size bounds by the same names.

- **Ability names** — `powerconfig.json` defines ability values (e.g., `Rally`, `Heal`, `Fireball`). These are referenced by the power calculation system when evaluating squad strength.

---

## Loading Order

Files loaded by `templates.ReadGameData()` during startup, in this order:

1. `gameconfig.json` — global tuning, loaded first so other systems can reference it
2. `difficultyconfig.json` — difficulty presets
3. `monsterdata.json` — unit templates
4. `namedata.json` — name generator configuration
5. `nodeDefinitions.json` — overworld node types
6. `encounterdata.json` — validates links against node definitions
7. `aiconfig.json` — AI configuration
8. `powerconfig.json` — power-scoring configuration
9. `overworldconfig.json` — overworld simulation tuning
10. `influenceconfig.json` — influence system tuning
11. `mapgenconfig.json` — map-generator parameters (optional, falls back to code defaults if missing)
12. `spelldata.json` — spell definitions (via `LoadSpellDefinitions`)
13. `unitspells.json` — unit-type → spell mappings (via `LoadUnitSpellDefinitions`; loaded after spell data so it can validate spell IDs)
14. `minor_artifacts.json` and `major_artifacts.json` — artifact definitions (via `LoadArtifactDefinitions`)

Files loaded by subsystem-specific loaders, outside `ReadGameData`:

- `raidconfig.json` — `raid.LoadRaidConfig`, called from `game_main/setup.go`
- `raidarchetypes.json` — `raid.LoadArchetypeData`, called from `game_main/setup.go`
- `perkdata.json` — `perks.LoadPerks`, called from the perks init path
- `perkbalanceconfig.json` — `perks.LoadPerkBalance`
- `artifactbalanceconfig.json` — `artifacts.LoadArtifactBalanceConfig`
- `combatbalanceconfig.json` — `combatcore.LoadCombatBalanceConfig`

Loading failures cause the game to panic at startup rather than proceeding with missing data, except for the optional `mapgenconfig.json`.
