# TinkerRogue Package Dependencies

**Last Updated:** 2026-03-20

## Dependency Hierarchy (Bottom to Top)

```
Layer 0: config (pure leaf, no internal imports)
Layer 1: world/coords -> config
Layer 2: common -> config, world/coords
Layer 3: templates, visual/graphics -> common
Layer 4: world/worldmap, visual/rendering -> common, visual/graphics
Layer 5: tactical/effects, tactical/unitprogression -> common
Layer 6: tactical/unitdefs -> common, tactical/unitprogression, templates
Layer 7: tactical/squads -> common, tactical/unitdefs
Layer 8: tactical/combat -> common, tactical/squads, tactical/effects
Layer 9: gear, tactical/spells, tactical/roster, tactical/squadservices
Layer 10: overworld/core -> common, templates; overworld/* layers build on it
Layer 11: mind/evaluation, mind/behavior, mind/combatlifecycle, mind/encounter, mind/raid, mind/ai
Layer 12: tactical/combatservices, tactical/commander
Layer 13: gui/* packages
Layer 14: gamesetup -> gui + mind + tactical
Layer 15: game_main (root) -> gamesetup + most everything
```

---

## Foundation / Leaf Packages

| Package | Notes |
|---|---|
| `config` | Pure configuration constants, no internal imports |
| `overworld/overworldlog` | Logging only, no internal imports |
| `gui/specs` | UI spec types only, no internal imports |
| `gui/widgets` | UI widget primitives only, no internal imports |
| `tools/combat_balance` | Standalone tool, no internal deps |
| `tools/combat_visualizer` | Standalone tool, no internal deps |
| `tools/report_compressor` | Standalone tool, no internal deps |

---

## Package-by-Package Internal Imports

### Core Infrastructure

**`config`** -- no internal imports

**`world/coords`**
- `config`

**`common`**
- `config`
- `world/coords`

**`templates`**
- `common`
- `config`
- `world/coords`
- `world/worldmap`

---

### Visual Layer

**`visual/graphics`**
- `common`
- `config`
- `world/coords`

**`visual/rendering`**
- `common`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

**`world/worldmap`**
- `common`
- `config`
- `visual/graphics`
- `world/coords`

---

### Testing Infrastructure

**`testing`**
- `common`
- `visual/graphics`
- `world/worldmap`

**`testing/bootstrap`**
- `common`
- `config`
- `gear`
- `overworld/core`
- `overworld/faction`
- `tactical/commander`
- `tactical/roster`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`
- `world/coords`
- `world/worldmap`

---

### Tactical Layer

**`tactical/effects`**
- `common`

**`tactical/unitprogression`**
- `common`

**`tactical/unitdefs`**
- `common`
- `config`
- `tactical/unitprogression`
- `templates`

**`tactical/squads`**
- `common`
- `config`
- `tactical/unitdefs`
- `tactical/unitprogression`
- `templates`
- `world/coords`

**`tactical/roster`**
- `common`
- `tactical/squads`

**`tactical/combat`**
- `common`
- `config`
- `tactical/effects`
- `tactical/squads`
- `tactical/unitdefs`
- `world/coords`

**`tactical/spells`**
- `common`
- `tactical/combat`
- `tactical/effects`
- `tactical/squads`
- `templates`

**`tactical/squadservices`**
- `common`
- `tactical/roster`
- `tactical/squads`
- `tactical/unitdefs`
- `world/coords`

**`tactical/squadcommands`**
- `common`
- `tactical/combat`
- `tactical/roster`
- `tactical/squads`
- `tactical/squadservices`
- `tactical/unitdefs`
- `world/coords`

**`tactical/combatservices`**
- `common`
- `gear`
- `mind/combatlifecycle`
- `tactical/combat`
- `tactical/effects`
- `tactical/squads`
- `world/coords`

**`tactical/commander`**
- `common`
- `overworld/core`
- `overworld/tick`
- `tactical/roster`
- `tactical/spells`
- `world/coords`

---

### Gear System

**`gear`**
- `common`
- `config`
- `tactical/combat`
- `tactical/effects`
- `tactical/squads`
- `templates`
- `world/coords`

---

### Overworld Layer

**`overworld/overworldlog`** -- no internal imports

**`overworld/core`**
- `common`
- `config`
- `overworld/overworldlog`
- `templates`
- `world/coords`

**`overworld/node`**
- `common`
- `overworld/core`
- `templates`
- `world/coords`

**`overworld/threat`**
- `common`
- `overworld/core`
- `overworld/node`
- `templates`
- `world/coords`

**`overworld/garrison`**
- `common`
- `overworld/core`
- `tactical/roster`
- `tactical/squads`
- `tactical/unitdefs`
- `world/coords`

**`overworld/influence`**
- `common`
- `overworld/core`
- `templates`
- `world/coords`

**`overworld/faction`**
- `common`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `templates`
- `world/coords`

**`overworld/tick`**
- `common`
- `overworld/core`
- `overworld/faction`
- `overworld/influence`
- `overworld/threat`

**`overworld/victory`**
- `common`
- `overworld/core`
- `overworld/threat`
- `templates`

---

### Mind / AI Layer

**`mind/evaluation`**
- `common`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`

**`mind/behavior`**
- `common`
- `mind/evaluation`
- `tactical/combat`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`
- `world/coords`

**`mind/combatlifecycle`**
- `common`
- `overworld/core`
- `tactical/combat`
- `tactical/spells`
- `tactical/squads`
- `tactical/unitprogression`
- `templates`
- `world/coords`

**`mind/ai`**
- `common`
- `mind/behavior`
- `tactical/combat`
- `tactical/combatservices`
- `tactical/squadcommands`
- `tactical/squads`
- `tactical/unitdefs`
- `world/coords`

**`mind/encounter`**
- `common`
- `mind/combatlifecycle`
- `mind/evaluation`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `tactical/combat`
- `tactical/roster`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`
- `world/coords`

**`mind/raid`**
- `common`
- `mind/combatlifecycle`
- `mind/encounter`
- `mind/evaluation`
- `tactical/combat`
- `tactical/squads`
- `tactical/unitdefs`
- `world/coords`
- `world/worldmap`

---

### Save System

**`savesystem`**
- `common`

**`savesystem/chunks`**
- `common`
- `gear`
- `mind/raid`
- `savesystem`
- `tactical/commander`
- `tactical/roster`
- `tactical/spells`
- `tactical/squads`
- `tactical/unitdefs`
- `tactical/unitprogression`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

---

### GUI Layer

**`gui/widgetresources`**
- `config`

**`gui/builders`**
- `common`
- `gui/specs`
- `gui/widgetresources`
- `gui/widgets`
- `tactical/combat`
- `tactical/squads`

**`gui/framework`**
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

**`gui/guiinspect`**
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `tactical/squads`

**`gui/guiunitview`**
- `common`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `tactical/unitprogression`

**`gui/guiartifacts`**
- `gear`
- `gui/framework`
- `gui/widgets`
- `tactical/combat`
- `tactical/combatservices`
- `visual/graphics`
- `world/coords`

**`gui/guispells`**
- `common`
- `gui/framework`
- `gui/widgets`
- `tactical/combat`
- `tactical/spells`
- `templates`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

**`gui/guiexploration`**
- `common`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `overworld/core`
- `templates`
- `world/worldmap`

**`gui/guisquads`**
- `common`
- `gear`
- `gui/builders`
- `gui/framework`
- `gui/guiinspect`
- `gui/guiunitview`
- `gui/specs`
- `gui/widgets`
- `tactical/combat`
- `tactical/commander`
- `tactical/roster`
- `tactical/squadcommands`
- `tactical/squads`
- `tactical/squadservices`
- `tactical/unitdefs`
- `templates`
- `visual/graphics`
- `visual/rendering`
- `world/coords`

**`gui/guiraid`**
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `gui/widgetresources`
- `mind/raid`
- `tactical/commander`
- `tactical/roster`
- `tactical/squads`
- `world/worldmap`

**`gui/guioverworld`**
- `common`
- `config`
- `gui/builders`
- `gui/framework`
- `gui/specs`
- `mind/encounter`
- `overworld/core`
- `overworld/garrison`
- `overworld/threat`
- `overworld/tick`
- `tactical/combat`
- `tactical/commander`
- `tactical/squads`
- `templates`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

**`gui/guinodeplacement`**
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

**`gui/guicombat`**
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
- `tactical/combatservices`
- `tactical/spells`
- `tactical/squadcommands`
- `tactical/squads`
- `templates`
- `visual/graphics`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

**`gui/guistartmenu`**
- `gui/builders`
- `savesystem`

---

### Input

**`input`**
- `common`
- `gui/framework`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

---

### Game Setup

**`gamesetup`**
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
- `mind/ai`
- `mind/combatlifecycle`
- `mind/encounter`
- `overworld/core`
- `overworld/node`
- `overworld/tick`
- `savesystem`
- `savesystem/chunks`
- `tactical/combat`
- `tactical/combatservices`
- `tactical/commander`
- `tactical/roster`
- `tactical/squadcommands`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`
- `testing`
- `testing/bootstrap`
- `visual/graphics`
- `world/coords`
- `world/worldmap`

---

### Entry Points

**`game_main` (root, `main` package)**
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
- `savesystem/chunks`
- `tactical/commander`
- `tactical/unitdefs`
- `visual/graphics`
- `visual/rendering`
- `world/coords`
- `world/worldmap`

**`tools/combat_simulator`**
- `common`
- `tactical/combat`
- `tactical/combatservices`
- `tactical/squads`
- `tactical/unitdefs`
- `templates`
- `world/coords`

---

## Notes

- **No circular dependencies** exist. `tactical/combatservices` imports `mind/combatlifecycle`, and `mind/combatlifecycle` imports `tactical/combat` (not `tactical/combatservices`). `gear` imports `tactical/combat` and `tactical/combatservices` imports `gear` -- also acyclic.
- Test-only imports (from `testing` and `testing/bootstrap`) are used only in `_test.go` files and don't affect the production dependency graph.
- The `overworld` package was split into sub-packages: `core`, `node`, `threat`, `garrison`, `influence`, `faction`, `tick`, `victory`, and `overworldlog`.
- The `tactical` package was split into: `combat`, `combatservices`, `commander`, `effects`, `roster`, `spells`, `squadcommands`, `squadservices`, `squads`, `unitdefs`, and `unitprogression`.
- The `mind` package was split into: `ai`, `behavior`, `combatlifecycle`, `encounter`, `evaluation`, and `raid`.
