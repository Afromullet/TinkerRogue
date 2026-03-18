# Package Dependency Reference

**Last Updated:** 2026-03-15

Complete internal dependency map for every package in TinkerRogue. Only project-internal imports are listed — standard library and third-party dependencies are omitted.

---

## Dependency Flow (High-Level)

```
config, assets/fonts, gui/specs, gui/widgets, overworld/overworldlog
    ↑ (leaf packages — no internal dependencies)

world/coords → config
common → config, world/coords

templates → common, config, visual/rendering, world/coords, world/worldmap
visual/graphics → common, config, world/coords
visual/rendering → common, visual/graphics, world/coords, world/worldmap
world/worldmap → common, config, visual/graphics, world/coords

tactical/effects → common
tactical/squads → common, config, tactical/effects, templates, visual/rendering, world/coords
tactical/combat → common, config, tactical/combat/battlelog, tactical/effects, tactical/squads, world/coords
tactical/squadservices → common, tactical/squads, world/coords
tactical/squadcommands → common, tactical/combat, tactical/squads, tactical/squadservices, world/coords
tactical/spells → common, tactical/combat, tactical/effects, tactical/squads, templates
tactical/commander → common, overworld/core, overworld/tick, tactical/spells, tactical/squads, visual/rendering, world/coords

gear → common, config, tactical/combat, tactical/effects, tactical/squads, templates, world/coords

mind/evaluation → common, tactical/squads, templates
mind/combatpipeline → common, tactical/combat, tactical/spells, tactical/squads, world/coords
mind/behavior → common, mind/evaluation, tactical/combat, tactical/squads, templates, visual/graphics, world/coords, world/worldmap
mind/ai → common, mind/behavior, tactical/combat, tactical/combatservices, tactical/squadcommands, tactical/squads, world/coords
mind/encounter → common, mind/combatpipeline, mind/evaluation, overworld/core, overworld/garrison, overworld/threat, tactical/combat, tactical/squads, templates, visual/rendering, world/coords
mind/raid → common, mind/combatpipeline, mind/encounter, mind/evaluation, tactical/combat, tactical/squads, world/coords, world/worldmap
tactical/combatservices → common, gear, mind/combatlifecycle, tactical/combat, tactical/combat/battlelog, tactical/effects, tactical/squads, world/coords

overworld/core → common, config, overworld/overworldlog, templates, world/coords
overworld/garrison → common, overworld/core, tactical/squads, world/coords
overworld/node → common, overworld/core, templates, world/coords
overworld/threat → common, overworld/core, overworld/node, templates, world/coords
overworld/influence → common, overworld/core, templates, world/coords
overworld/faction → common, overworld/core, overworld/garrison, overworld/threat, templates, world/coords
overworld/tick → common, overworld/core, overworld/faction, overworld/influence, overworld/threat
overworld/victory → common, overworld/core, overworld/threat, templates

gui/* → (see GUI section below)
input → common, gui/framework, visual/graphics, world/coords, world/worldmap
gamesetup → (see below — highest fan-out in the project)
```

---

## Foundation Layer (No Internal Dependencies)

| Package | Notes |
|---|---|
| `config` | Build flags, constants |
| `assets/fonts` | Embedded font data |
| `gui/specs` | GUI specification types |
| `gui/widgets` | Reusable widget primitives |
| `overworld/overworldlog` | Overworld log buffer |
| `tools/combat_balance` | Standalone CLI tool |
| `tools/combat_visualizer` | Standalone CLI tool |
| `tools/report_compressor` | Standalone CLI tool |

---

## Core Layer

### `world/coords`
- `config`

### `common`
- `config`
- `world/coords`

---

## Data / Visual Layer

### `templates`
- `common`
- `config`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

### `visual/graphics`
- `common`
- `config`
- `world/coords`

### `visual/rendering`
- `common`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

### `world/worldmap`
- `common`
- `config`
- `visual/graphics`
- `world/coords`

---

## Tactical Layer

### `tactical/effects`
- `common`

### `tactical/squads`
- `common`
- `config`
- `tactical/effects`
- `templates`
- `testing` *(debug-only)*
- `visual/rendering`
- `world/coords`

### `tactical/combat/battlelog`
- `tactical/squads`

### `tactical/combat`
- `common`
- `config`
- `tactical/combat/battlelog`
- `tactical/effects`
- `tactical/squads`
- `testing` *(debug-only)*
- `world/coords`

### `tactical/squadservices`
- `common`
- `tactical/squads`
- `world/coords`

### `tactical/squadcommands`
- `common`
- `tactical/combat`
- `tactical/squads`
- `tactical/squadservices`
- `world/coords`

### `tactical/spells`
- `common`
- `tactical/combat`
- `tactical/effects`
- `tactical/squads`
- `templates`

### `tactical/commander`
- `common`
- `overworld/core`
- `overworld/tick`
- `tactical/spells`
- `tactical/squads`
- `visual/rendering`
- `world/coords`

### `tactical/combatservices`
- `common`
- `gear`
- `mind/combatlifecycle`
- `tactical/combat`
- `tactical/combat/battlelog`
- `tactical/effects`
- `tactical/squads`
- `world/coords`

---

## Gear Layer

### `gear`
- `common`
- `config`
- `tactical/combat`
- `tactical/effects`
- `tactical/squads`
- `templates`
- `testing` *(debug-only)*
- `world/coords`

---

## Mind Layer

### `mind/evaluation`
- `common`
- `tactical/squads`
- `templates`

### `mind/combatpipeline`
- `common`
- `tactical/combat`
- `tactical/spells`
- `tactical/squads`
- `world/coords`

### `mind/behavior`
- `common`
- `mind/evaluation`
- `tactical/combat`
- `tactical/squads`
- `templates`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

### `mind/ai`
- `common`
- `mind/behavior`
- `tactical/combat`
- `tactical/combatservices`
- `tactical/squadcommands`
- `tactical/squads`
- `world/coords`

### `mind/encounter`
- `common`
- `mind/combatpipeline`
- `mind/evaluation`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `tactical/combat`
- `tactical/squads`
- `templates`
- `visual/rendering`
- `world/coords`

### `mind/raid`
- `common`
- `mind/combatpipeline`
- `mind/encounter`
- `mind/evaluation`
- `tactical/combat`
- `tactical/squads`
- `world/coords`
- `world/worldmap`

---

## Overworld Layer

### `overworld/core`
- `common`
- `config`
- `overworld/overworldlog`
- `templates`
- `world/coords`

### `overworld/garrison`
- `common`
- `overworld/core`
- `tactical/squads`
- `world/coords`

### `overworld/node`
- `common`
- `overworld/core`
- `templates`
- `world/coords`

### `overworld/threat`
- `common`
- `overworld/core`
- `overworld/node`
- `templates`
- `world/coords`

### `overworld/influence`
- `common`
- `overworld/core`
- `templates`
- `world/coords`

### `overworld/faction`
- `common`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `templates`
- `world/coords`

### `overworld/tick`
- `common`
- `overworld/core`
- `overworld/faction`
- `overworld/influence`
- `overworld/threat`

### `overworld/victory`
- `common`
- `overworld/core`
- `overworld/threat`
- `templates`

---

## GUI Layer

### `gui/widgetresources`
- `config`

### `gui/builders`
- `common`
- `gui/specs`
- `gui/widgetresources`
- `gui/widgets`
- `tactical/squads`

### `gui/framework`
- `common`
- `gui/builders`
- `gui/specs`
- `overworld/core`
- `tactical/combat`
- `tactical/squadcommands`
- `tactical/squads`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

### `gui/guiinspect`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `tactical/squads`

### `gui/guiunitview`
- `common`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `tactical/squads`

### `gui/guistartmenu`
- `gui/builders`
- `savesystem`

### `gui/guiartifacts`
- `gear`
- `gui/framework`
- `gui/widgets`
- `mind/encounter`
- `tactical/combat`
- `tactical/combatservices`
- `visual/graphics`
- `world/coords`

### `gui/guispells`
- `common`
- `gui/framework`
- `gui/widgets`
- `mind/encounter`
- `tactical/combat`
- `tactical/spells`
- `templates`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

### `gui/guiexploration`
- `common`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `overworld/core`
- `templates`
- `world/worldmap`

### `gui/guisquads`
- `common`
- `gear`
- `gui/builders`
- `gui/framework`
- `gui/guiinspect`
- `gui/guiunitview`
- `gui/specs`
- `gui/widgets`
- `tactical/commander`
- `tactical/squadcommands`
- `tactical/squads`
- `tactical/squadservices`
- `templates`
- `visual/graphics`
- `visual/rendering`
- `world/coords`

### `gui/guioverworld`
- `common`
- `config`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `mind/combatpipeline`
- `mind/encounter`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `overworld/tick`
- `tactical/commander`
- `tactical/squads`
- `templates`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

### `gui/guinodeplacement`
- `common`
- `gui/builders`
- `gui/framework`
- `gui/guioverworld`
- `gui/specs`
- `gui/widgetresources`
- `overworld/core`
- `overworld/node`
- `templates`
- `world/coords`

### `gui/guiraid`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `gui/widgetresources`
- `mind/combatpipeline`
- `mind/raid`
- `tactical/commander`
- `tactical/squads`
- `world/worldmap`

### `gui/guicombat`
- `common`
- `config`
- `gui/builders`
- `gui/framework`
- `gui/guiartifacts`
- `gui/guiinspect`
- `gui/guispells`
- `gui/guisquads`
- `gui/specs`
- `gui/widgetresources`
- `gui/widgets`
- `mind/ai`
- `mind/evaluation`
- `tactical/combat`
- `tactical/combat/battlelog`
- `tactical/combatservices`
- `tactical/spells`
- `tactical/squadcommands`
- `tactical/squads`
- `templates`
- `visual/graphics`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

---

## Input Layer

### `input`
- `common`
- `gui/framework`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

---

## Save System

### `savesystem`
- `common`

### `savesystem/chunks`
- `common`
- `gear`
- `mind/raid`
- `savesystem`
- `tactical/commander`
- `tactical/spells`
- `tactical/squads`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

---

## Game Setup (Highest Fan-Out)

### `gamesetup`
- `common`
- `config`
- `gear`
- `gui/framework`
- `gui/guicombat`
- `gui/guiexploration`
- `gui/guinodeplacement`
- `gui/guioverworld`
- `gui/guiraid`
- `gui/guisquads`
- `gui/guiunitview`
- `mind/encounter`
- `overworld/core`
- `overworld/node`
- `overworld/tick`
- `savesystem`
- `savesystem/chunks`
- `tactical/commander`
- `tactical/squadcommands` *(blank import for init)*
- `tactical/squads`
- `templates`
- `testing`
- `testing/bootstrap`
- `visual/graphics`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

---

## Entry Point

### `game_main`
- `common`
- `config`
- `gamesetup`
- `gui/framework`
- `gui/guistartmenu`
- `gui/widgetresources`
- `input`
- `mind/encounter`
- `mind/raid`
- `overworld/core`
- `savesystem`
- `savesystem/chunks` *(blank import for init)*
- `tactical/commander`
- `tactical/squads`
- `visual/graphics`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

---

## Testing

### `testing`
- `common`
- `visual/graphics`
- `world/worldmap`

### `testing/bootstrap`
- `common`
- `config`
- `gear`
- `overworld/core`
- `overworld/faction`
- `tactical/commander`
- `tactical/squads`
- `templates`
- `world/coords`
- `world/worldmap`

### `tools/combat_simulator`
- `common`
- `tactical/combat`
- `tactical/combat/battlelog`
- `tactical/combatservices`
- `tactical/squads`
- `templates`
- `world/coords`

---

## Most Depended-On Packages

Packages imported by the most other packages, in rough order:

1. **`common`** — imported by nearly every package
2. **`world/coords`** — used wherever positions matter
3. **`tactical/squads`** — central game data
4. **`config`** — constants and flags
5. **`templates`** — unit/spell definitions
6. **`tactical/combat`** — combat state and logic
7. **`world/worldmap`** — map data
8. **`visual/rendering`** — draw operations
9. **`visual/graphics`** — sprite/tile data
10. **`gui/framework`** — GUI mode management

## Notable Cross-Layer Dependencies

- `tactical/combatservices` → `mind/combatlifecycle` (tactical layer reaching into mind layer for combat lifecycle; `mind/ai` and `mind/behavior` decoupled via `AITurnController`, `ThreatProvider`, and `ThreatLayerEvaluator` interfaces)
- `mind/encounter` → `overworld/garrison`, `overworld/threat` (mind layer reading overworld state)
- `tactical/commander` → `overworld/core`, `overworld/tick` (commander bridges tactical and overworld)
- `gui/guicombat` has the widest import list of any GUI package (24 dependencies)
- `gamesetup` has the widest import list overall (26 dependencies) — expected for a bootstrap package
