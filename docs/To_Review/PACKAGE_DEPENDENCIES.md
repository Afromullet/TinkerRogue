# TinkerRogue Package Dependencies

Complete internal dependency map organized by architectural layer. External/stdlib imports omitted.

---

## Metrics Key

- **Fan-out** = number of internal packages this package imports
- **Fan-in** = number of internal packages that import this package

---

## Layer 0 — Config

### `config/`
| Metric | Value |
|--------|-------|
| Fan-out | 0 |
| Fan-in | 11 |
| Imports | (none) |
| Imported by | world/coords, common, visual/graphics, templates, tactical/squads, gear, world/worldmap, overworld/core, gui/widgetresources, gui/guicombat, gui/guioverworld, gamesetup, game_main, testing/bootstrap |

---

## Layer 1 — Primitives

### `world/coords/`
| Metric | Value |
|--------|-------|
| Fan-out | 1 |
| Fan-in | 21 |
| Imports | config |
| Imported by | common, visual/graphics, visual/rendering, templates, tactical/squads, tactical/combat, tactical/commander, tactical/squadcommands, tactical/squadservices, tactical/combatservices, gear, world/worldmap, overworld/core, overworld/faction, overworld/garrison, overworld/influence, overworld/node, overworld/threat, mind/ai, mind/behavior, mind/encounter, mind/raid, gui/framework, gui/guiartifacts, gui/guicombat, gui/guioverworld, gui/guispells, gui/guisquads, gui/guinodeplacement, gamesetup, game_main, input, savesystem/chunks, testing/bootstrap |

---

## Layer 2 — Core Infrastructure

### `common/`
| Metric | Value |
|--------|-------|
| Fan-out | 2 |
| Fan-in | 37 |
| Imports | config, world/coords |
| Imported by | (nearly every package — 37 total including all tactical/*, mind/*, overworld/*, gui/*, visual/*, savesystem/*, gamesetup, game_main, input, templates, gear, testing) |

### `visual/graphics/`
| Metric | Value |
|--------|-------|
| Fan-out | 3 |
| Fan-in | 7 |
| Imports | common, config, world/coords |
| Imported by | world/worldmap (via rendering), visual/rendering, gui/guiartifacts, gui/guicombat, gui/guispells, gui/guisquads, gamesetup, game_main, input, testing, savesystem/chunks |

### `visual/rendering/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 5 |
| Imports | common, visual/graphics, world/coords, world/worldmap |
| Imported by | gui/framework, gui/guicombat, gui/guioverworld, gui/guisquads, game_main |

### `templates/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 16 |
| Imports | common, config, world/coords, world/worldmap |
| Imported by | tactical/squads, tactical/spells, gear, mind/behavior, mind/evaluation, mind/encounter, overworld/core, overworld/faction, overworld/influence, overworld/node, overworld/threat, overworld/victory, gui/guicombat, gui/guiexploration, gui/guioverworld, gui/guispells, gui/guisquads, gui/guinodeplacement, gamesetup, testing/bootstrap |

---

## Layer 3 — Game Systems

### `tactical/effects/`
| Metric | Value |
|--------|-------|
| Fan-out | 1 |
| Fan-in | 4 |
| Imports | common |
| Imported by | tactical/squads, tactical/combat, tactical/combatservices, tactical/spells, gear |

### `tactical/squads/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 22 |
| Imports | common, config, tactical/effects, templates, world/coords |
| Imported by | tactical/combat, tactical/combat/battlelog, tactical/spells, tactical/commander, tactical/squadcommands, tactical/squadservices, tactical/combatservices, gear, mind/ai, mind/behavior, mind/combatlifecycle, mind/evaluation, mind/encounter, mind/raid, overworld/garrison, gui/builders, gui/framework, gui/guicombat, gui/guiinspect, gui/guioverworld, gui/guiraid, gui/guisquads, gui/guiunitview, gamesetup, game_main, savesystem/chunks, testing/bootstrap |

### `tactical/combat/`
| Metric | Value |
|--------|-------|
| Fan-out | 7 |
| Fan-in | 12 |
| Imports | common, config, tactical/combat/battlelog, tactical/effects, tactical/squads, testing, world/coords |
| Imported by | tactical/spells, tactical/combatservices, tactical/squadcommands, mind/ai, mind/behavior, mind/combatlifecycle, mind/encounter, mind/raid, gear, gui/framework, gui/guiartifacts, gui/guicombat, gui/guioverworld, gui/guispells, gamesetup |

### `tactical/combat/battlelog/`
| Metric | Value |
|--------|-------|
| Fan-out | 1 |
| Fan-in | 2 |
| Imports | tactical/squads |
| Imported by | tactical/combat, tactical/combatservices, gui/guicombat |

### `tactical/spells/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 3 |
| Imports | common, tactical/combat, tactical/effects, tactical/squads, templates |
| Imported by | mind/combatlifecycle, gui/guicombat, gui/guispells, savesystem/chunks |

### `tactical/commander/`
| Metric | Value |
|--------|-------|
| Fan-out | 6 |
| Fan-in | 5 |
| Imports | common, overworld/core, overworld/tick, tactical/spells, tactical/squads, world/coords |
| Imported by | gui/guioverworld, gui/guiraid, gui/guisquads, gamesetup, game_main, savesystem/chunks |

### `tactical/squadcommands/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 4 |
| Imports | common, tactical/combat, tactical/squads, tactical/squadservices, world/coords |
| Imported by | mind/ai, gui/framework, gui/guicombat, gui/guisquads, gamesetup |

### `tactical/squadservices/`
| Metric | Value |
|--------|-------|
| Fan-out | 3 |
| Fan-in | 2 |
| Imports | common, tactical/squads, world/coords |
| Imported by | tactical/squadcommands, gui/guisquads |

### `tactical/combatservices/`
| Metric | Value |
|--------|-------|
| Fan-out | 8 |
| Fan-in | 3 |
| Imports | common, gear, mind/combatlifecycle, tactical/combat, tactical/combat/battlelog, tactical/effects, tactical/squads, world/coords |
| Imported by | mind/ai, gui/guiartifacts, gui/guicombat |

### `gear/`
| Metric | Value |
|--------|-------|
| Fan-out | 6 |
| Fan-in | 4 |
| Imports | common, config, tactical/combat, tactical/effects, tactical/squads, templates |
| Imported by | tactical/combatservices, gui/guisquads, savesystem/chunks, testing/bootstrap |

### `world/worldmap/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 10 |
| Imports | common, config, visual/graphics, world/coords |
| Imported by | templates, visual/rendering, mind/raid, gui/framework, gui/guicombat, gui/guiexploration, gui/guioverworld, gui/guiraid, gui/guispells, gamesetup, game_main, input, savesystem/chunks, testing, testing/bootstrap |

---

## Layer 3 — Overworld Subsystems

### `overworld/overworldlog/`
| Metric | Value |
|--------|-------|
| Fan-out | 0 |
| Fan-in | 1 |
| Imports | (none — only ecs library) |
| Imported by | overworld/core |

### `overworld/core/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 11 |
| Imports | common, config, overworld/overworldlog, templates, world/coords |
| Imported by | overworld/faction, overworld/garrison, overworld/influence, overworld/node, overworld/threat, overworld/tick, overworld/victory, tactical/commander, mind/encounter, gui/framework, gui/guiexploration, gui/guioverworld, gui/guinodeplacement, gamesetup, game_main, testing/bootstrap |

### `overworld/faction/`
| Metric | Value |
|--------|-------|
| Fan-out | 6 |
| Fan-in | 2 |
| Imports | common, overworld/core, overworld/garrison, overworld/threat, templates, world/coords |
| Imported by | overworld/tick, testing/bootstrap |

### `overworld/garrison/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 3 |
| Imports | common, overworld/core, tactical/squads, world/coords |
| Imported by | overworld/faction, mind/encounter, gui/guioverworld |

### `overworld/influence/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 1 |
| Imports | common, overworld/core, templates, world/coords |
| Imported by | overworld/tick |

### `overworld/node/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 2 |
| Imports | common, overworld/core, templates, world/coords |
| Imported by | overworld/threat, gui/guinodeplacement |

### `overworld/threat/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 4 |
| Imports | common, overworld/core, overworld/node, templates, world/coords |
| Imported by | overworld/faction, overworld/tick, overworld/victory, mind/encounter, gui/guioverworld |

### `overworld/tick/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 2 |
| Imports | common, overworld/core, overworld/faction, overworld/influence, overworld/threat |
| Imported by | tactical/commander, gui/guioverworld, gamesetup |

### `overworld/victory/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 0 |
| Imports | common, overworld/core, overworld/threat, templates |
| Imported by | (none found — may be called at runtime only) |

---

## Layer 4 — AI & Orchestration

### `mind/evaluation/`
| Metric | Value |
|--------|-------|
| Fan-out | 3 |
| Fan-in | 3 |
| Imports | common, tactical/squads, templates |
| Imported by | mind/behavior, mind/encounter, mind/raid, gui/guicombat |

### `mind/behavior/`
| Metric | Value |
|--------|-------|
| Fan-out | 6 |
| Fan-in | 1 |
| Imports | common, mind/evaluation, tactical/combat, tactical/squads, templates, world/coords |
| Imported by | mind/ai |

### `mind/combatlifecycle/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 3 |
| Imports | common, tactical/combat, tactical/spells, tactical/squads |
| Imported by | tactical/combatservices, mind/encounter, mind/raid, gamesetup |

### `mind/ai/`
| Metric | Value |
|--------|-------|
| Fan-out | 7 |
| Fan-in | 1 |
| Imports | common, mind/behavior, tactical/combat, tactical/combatservices, tactical/squadcommands, tactical/squads, world/coords |
| Imported by | gui/guicombat |

### `mind/encounter/`
| Metric | Value |
|--------|-------|
| Fan-out | 10 |
| Fan-in | 3 |
| Imports | common, mind/combatlifecycle, mind/evaluation, overworld/core, overworld/garrison, overworld/threat, tactical/combat, tactical/squads, templates, world/coords |
| Imported by | mind/raid, gui/guioverworld, gamesetup, game_main |

### `mind/raid/`
| Metric | Value |
|--------|-------|
| Fan-out | 8 |
| Fan-in | 2 |
| Imports | common, mind/combatlifecycle, mind/encounter, mind/evaluation, tactical/combat, tactical/squads, world/coords, world/worldmap |
| Imported by | gui/guiraid, game_main, savesystem/chunks |

---

## Layer 5 — Presentation (GUI)

### `gui/specs/`
| Metric | Value |
|--------|-------|
| Fan-out | 0 |
| Fan-in | 9 |
| Imports | (none) |
| Imported by | gui/builders, gui/framework, gui/guicombat, gui/guiexploration, gui/guiinspect, gui/guioverworld, gui/guiraid, gui/guisquads, gui/guinodeplacement, gui/guiunitview |

### `gui/widgetresources/`
| Metric | Value |
|--------|-------|
| Fan-out | 1 |
| Fan-in | 3 |
| Imports | config |
| Imported by | gui/builders, gui/guicombat, gui/guiraid, gui/guinodeplacement |

### `gui/widgets/`
| Metric | Value |
|--------|-------|
| Fan-out | 0 |
| Fan-in | 5 |
| Imports | (none — only ebitenui) |
| Imported by | gui/builders, gui/guiartifacts, gui/guicombat, gui/guispells, gui/guisquads |

### `gui/builders/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 10 |
| Imports | common, gui/specs, gui/widgetresources, gui/widgets, tactical/squads |
| Imported by | gui/framework, gui/guicombat, gui/guiexploration, gui/guiinspect, gui/guioverworld, gui/guiraid, gui/guisquads, gui/guinodeplacement, gui/guiunitview, gui/guistartmenu |

### `gui/framework/`
| Metric | Value |
|--------|-------|
| Fan-out | 10 |
| Fan-in | 11 |
| Imports | common, gui/builders, gui/specs, overworld/core, tactical/combat, tactical/squadcommands, tactical/squads, visual/rendering, world/coords, world/worldmap |
| Imported by | gui/guiartifacts, gui/guicombat, gui/guiexploration, gui/guiinspect, gui/guioverworld, gui/guiraid, gui/guisquads, gui/guispells, gui/guinodeplacement, gui/guiunitview, gamesetup, game_main, input |

### `gui/guicombat/`
| Metric | Value |
|--------|-------|
| Fan-out | 25 |
| Fan-in | 1 |
| Imports | common, config, gui/builders, gui/framework, gui/guiartifacts, gui/guiinspect, gui/guispells, gui/guisquads, gui/specs, gui/widgetresources, gui/widgets, mind/ai, mind/evaluation, tactical/combat, tactical/combat/battlelog, tactical/combatservices, tactical/spells, tactical/squadcommands, tactical/squads, templates, visual/graphics, visual/rendering, world/coords, world/worldmap |
| Imported by | gamesetup |

### `gui/guioverworld/`
| Metric | Value |
|--------|-------|
| Fan-out | 17 |
| Fan-in | 2 |
| Imports | common, config, gui/builders, gui/framework, gui/specs, mind/encounter, overworld/core, overworld/garrison, overworld/threat, overworld/tick, tactical/combat, tactical/commander, tactical/squads, templates, visual/rendering, world/coords, world/worldmap |
| Imported by | gui/guinodeplacement, gamesetup |

### `gui/guisquads/`
| Metric | Value |
|--------|-------|
| Fan-out | 16 |
| Fan-in | 2 |
| Imports | common, gear, gui/builders, gui/framework, gui/guiinspect, gui/guiunitview, gui/specs, gui/widgets, tactical/commander, tactical/squadcommands, tactical/squads, tactical/squadservices, templates, visual/graphics, visual/rendering, world/coords |
| Imported by | gui/guicombat, gamesetup |

### `gui/guiraid/`
| Metric | Value |
|--------|-------|
| Fan-out | 8 |
| Fan-in | 1 |
| Imports | gui/builders, gui/framework, gui/specs, gui/widgetresources, mind/raid, tactical/commander, tactical/squads, world/worldmap |
| Imported by | gamesetup |

### `gui/guiexploration/`
| Metric | Value |
|--------|-------|
| Fan-out | 7 |
| Fan-in | 1 |
| Imports | common, gui/builders, gui/framework, gui/specs, overworld/core, templates, world/worldmap |
| Imported by | gamesetup |

### `gui/guiartifacts/`
| Metric | Value |
|--------|-------|
| Fan-out | 7 |
| Fan-in | 1 |
| Imports | gear, gui/framework, gui/widgets, tactical/combat, tactical/combatservices, visual/graphics, world/coords |
| Imported by | gui/guicombat |

### `gui/guispells/`
| Metric | Value |
|--------|-------|
| Fan-out | 9 |
| Fan-in | 1 |
| Imports | common, gui/framework, gui/widgets, tactical/combat, tactical/spells, templates, visual/graphics, world/coords, world/worldmap |
| Imported by | gui/guicombat |

### `gui/guiinspect/`
| Metric | Value |
|--------|-------|
| Fan-out | 4 |
| Fan-in | 2 |
| Imports | gui/builders, gui/framework, gui/specs, tactical/squads |
| Imported by | gui/guicombat, gui/guisquads |

### `gui/guiunitview/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 1 |
| Imports | common, gui/builders, gui/framework, gui/specs, tactical/squads |
| Imported by | gui/guisquads |

### `gui/guinodeplacement/`
| Metric | Value |
|--------|-------|
| Fan-out | 10 |
| Fan-in | 1 |
| Imports | common, gui/builders, gui/framework, gui/guioverworld, gui/specs, gui/widgetresources, overworld/core, overworld/node, templates, world/coords |
| Imported by | gamesetup |

### `gui/guistartmenu/`
| Metric | Value |
|--------|-------|
| Fan-out | 2 |
| Fan-in | 1 |
| Imports | gui/builders, savesystem |
| Imported by | game_main |

---

## Layer 6 — Bootstrap & Entry

### `gamesetup/`
| Metric | Value |
|--------|-------|
| Fan-out | 28 |
| Fan-in | 1 |
| Imports | common, config, gear, gui/framework, gui/guicombat, gui/guiexploration, gui/guinodeplacement, gui/guioverworld, gui/guiraid, gui/guisquads, gui/guiunitview, mind/combatlifecycle, mind/encounter, overworld/core, overworld/node, overworld/tick, savesystem, savesystem/chunks, tactical/combat, tactical/commander, tactical/squadcommands, tactical/squads, templates, testing, testing/bootstrap, visual/graphics, world/coords, world/worldmap |
| Imported by | game_main |

### `game_main/` (entry point)
| Metric | Value |
|--------|-------|
| Fan-out | 18 |
| Fan-in | 0 |
| Imports | common, config, gamesetup, gui/framework, gui/guistartmenu, gui/widgetresources, input, mind/encounter, mind/raid, overworld/core, savesystem, savesystem/chunks, tactical/commander, tactical/squads, visual/graphics, visual/rendering, world/coords, world/worldmap |
| Imported by | (none — entry point) |

---

## Support Packages

### `input/`
| Metric | Value |
|--------|-------|
| Fan-out | 5 |
| Fan-in | 1 |
| Imports | common, gui/framework, visual/graphics, world/coords, world/worldmap |
| Imported by | game_main |

### `savesystem/`
| Metric | Value |
|--------|-------|
| Fan-out | 1 |
| Fan-in | 3 |
| Imports | common |
| Imported by | savesystem/chunks, gui/guistartmenu, gamesetup, game_main |

### `savesystem/chunks/`
| Metric | Value |
|--------|-------|
| Fan-out | 10 |
| Fan-in | 2 |
| Imports | common, gear, mind/raid, savesystem, tactical/commander, tactical/spells, tactical/squads, visual/graphics, world/coords, world/worldmap |
| Imported by | gamesetup, game_main |

### `testing/`
| Metric | Value |
|--------|-------|
| Fan-out | 3 |
| Fan-in | 2 |
| Imports | common, visual/graphics, world/worldmap |
| Imported by | tactical/combat, gamesetup |

### `testing/bootstrap/`
| Metric | Value |
|--------|-------|
| Fan-out | 10 |
| Fan-in | 1 |
| Imports | common, config, gear, overworld/core, overworld/faction, tactical/commander, tactical/squads, templates, world/coords, world/worldmap |
| Imported by | gamesetup |

---

## Top Packages by Fan-In (most depended upon)

| Package | Fan-In |
|---------|--------|
| common | ~37 |
| world/coords | ~21 |
| tactical/squads | ~22 |
| templates | ~16 |
| gui/framework | ~11 |
| config | ~11 |
| gui/builders | ~10 |
| tactical/combat | ~12 |
| overworld/core | ~11 |
| world/worldmap | ~10 |
| gui/specs | ~9 |

## Top Packages by Fan-Out (most dependencies)

| Package | Fan-Out |
|---------|---------|
| gamesetup | 28 |
| gui/guicombat | 25 |
| game_main | 18 |
| gui/guioverworld | 17 |
| gui/guisquads | 16 |
| gui/framework | 10 |
| mind/encounter | 10 |
| savesystem/chunks | 10 |
| testing/bootstrap | 10 |
| gui/guinodeplacement | 10 |
| gui/guispells | 9 |
