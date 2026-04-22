# Package Dependencies

**Generated:** 2026-04-22
**Module:** `game_main` (see `go.mod`)
**Total packages:** 77

Lists every internal Go package in the module and the other internal packages it imports. Third-party imports (`github.com/...`, `golang.org/...`) and the Go standard library are excluded. Paths are shown relative to the module root.

Sorted alphabetically by package path. Packages listed with *(no internal dependencies)* import only stdlib or third-party libraries.

> **Note on `game_main` entries:** The module itself is named `game_main`, and there is also a subdirectory `game_main/` containing the main binary. To disambiguate, the module-root package (`test_init.go`) is listed as **`game_main (module root)`** and the subdirectory is listed as **`game_main/game_main (main binary)`**.

---

## campaign/overworld/core
- campaign/overworld/overworldlog
- core/common
- core/config
- core/coords
- templates

## campaign/overworld/faction
- campaign/overworld/core
- campaign/overworld/garrison
- campaign/overworld/threat
- core/common
- core/coords
- templates

## campaign/overworld/garrison
- campaign/overworld/core
- core/common
- core/coords
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/unitdefs

## campaign/overworld/influence
- campaign/overworld/core
- core/common
- core/coords
- templates

## campaign/overworld/node
- campaign/overworld/core
- core/common
- core/coords
- templates

## campaign/overworld/overworldlog
*(no internal dependencies)*

## campaign/overworld/threat
- campaign/overworld/core
- campaign/overworld/node
- core/common
- core/coords
- templates

## campaign/overworld/tick
- campaign/overworld/core
- campaign/overworld/faction
- campaign/overworld/influence
- campaign/overworld/threat
- core/common

## campaign/overworld/victory
- campaign/overworld/core
- campaign/overworld/threat
- core/common
- templates

## campaign/raid
- core/common
- core/coords
- mind/combatlifecycle
- mind/encounter
- mind/evaluation
- tactical/combat/combatstate
- tactical/squads/squadcore
- tactical/squads/unitdefs
- world/garrisongen

## core/common
- core/config
- core/coords

## core/config
*(no internal dependencies)*

## core/coords
- core/config

## game_main (module root)
- core/common
- tactical/squads/squadcore

## game_main/game_main (main binary)
- campaign/overworld/core
- campaign/raid
- core/common
- core/config
- core/coords
- gui/framework
- gui/guistartmenu
- gui/widgetresources
- input
- mind/encounter
- setup/gamesetup
- setup/savesystem
- setup/savesystem/chunks
- tactical/commander
- tactical/squads/unitdefs
- visual/graphics
- visual/maprender
- visual/rendering
- visual/vfx
- world/worldgen
- world/worldmapcore

## gui/builders
- gui/specs
- gui/widgetresources
- gui/widgets

## gui/framework
- campaign/overworld/core
- core/common
- core/coords
- gui/builders
- gui/specs
- tactical/combat/combatstate
- tactical/squads/squadcommands
- tactical/squads/squadcore
- templates
- visual/combatrender
- world/worldmapcore

## gui/guiartifacts
- core/coords
- gui/framework
- gui/widgets
- mind/combatlifecycle
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/powers/artifacts
- tactical/powers/powercore
- visual/graphics

## gui/guicombat
- core/common
- core/config
- core/coords
- gui/builders
- gui/framework
- gui/guiartifacts
- gui/guiinspect
- gui/guispells
- gui/guisquads
- gui/specs
- gui/widgetresources
- gui/widgets
- mind/combatlifecycle
- tactical/combat/battlelog
- tactical/combat/combatcore
- tactical/combat/combatmath
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/spells
- tactical/squads/squadcommands
- tactical/squads/squadcore
- templates
- visual/combatrender
- visual/graphics
- world/worldmapcore

## gui/guiexploration
- campaign/overworld/core
- core/common
- gui/builders
- gui/framework
- gui/specs
- templates
- world/worldgen
- world/worldmapcore

## gui/guiinspect
- gui/builders
- gui/framework
- gui/specs
- tactical/squads/squadcore

## gui/guinodeplacement
- campaign/overworld/core
- campaign/overworld/node
- core/common
- core/coords
- gui/builders
- gui/framework
- gui/guioverworld
- gui/specs
- gui/widgetresources
- templates

## gui/guioverworld
- campaign/overworld/core
- campaign/overworld/garrison
- campaign/overworld/threat
- campaign/overworld/tick
- core/common
- core/config
- core/coords
- gui/builders
- gui/framework
- gui/specs
- mind/combatlifecycle
- mind/encounter
- tactical/commander
- tactical/squads/squadcore
- templates
- visual/maprender
- world/worldmapcore

## gui/guiprogression
- core/common
- gui/builders
- gui/framework
- gui/specs
- gui/widgets
- tactical/powers/perks
- tactical/powers/progression
- templates

## gui/guiraid
- campaign/raid
- gui/builders
- gui/framework
- gui/specs
- gui/widgetresources
- tactical/commander
- tactical/squads/roster
- tactical/squads/squadcore
- world/garrisongen

## gui/guispells
- core/common
- core/coords
- gui/framework
- gui/widgets
- mind/combatlifecycle
- tactical/combat/combatstate
- tactical/powers/spells
- templates
- visual/graphics
- visual/vfx
- world/worldmapcore

## gui/guisquads
- core/common
- core/coords
- gui/builders
- gui/framework
- gui/guiinspect
- gui/guiprogression
- gui/guiunitview
- gui/specs
- gui/widgets
- tactical/combat/combattypes
- tactical/commander
- tactical/powers/artifacts
- tactical/powers/perks
- tactical/powers/progression
- tactical/squads/roster
- tactical/squads/squadcommands
- tactical/squads/squadcore
- tactical/squads/squadservices
- tactical/squads/unitdefs
- templates
- visual/combatrender
- visual/graphics

## gui/guistartmenu
- gui/builders
- setup/savesystem

## gui/guiunitview
- core/common
- gui/builders
- gui/framework
- gui/specs
- tactical/squads/unitprogression

## gui/specs
*(no internal dependencies)*

## gui/widgetresources
- core/config

## gui/widgets
*(no internal dependencies)*

## input
- core/common
- core/coords
- gui/framework
- visual/graphics
- world/worldmapcore

## mind/ai
- core/common
- core/coords
- mind/behavior
- tactical/combat/combatcore
- tactical/combat/combatmath
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/squads/squadcommands
- tactical/squads/squadcore
- tactical/squads/unitdefs

## mind/behavior
- core/common
- core/coords
- mind/evaluation
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## mind/combatlifecycle
- core/common
- core/coords
- tactical/combat/combatstate
- tactical/commander
- tactical/powers/perks
- tactical/powers/progression
- tactical/powers/spells
- tactical/squads/squadcore
- tactical/squads/unitprogression

## mind/encounter
- campaign/overworld/core
- campaign/overworld/garrison
- campaign/overworld/threat
- core/common
- core/coords
- mind/combatlifecycle
- mind/spawning
- tactical/squads/roster
- tactical/squads/squadcore

## mind/evaluation
- core/common
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## mind/spawning
- campaign/overworld/core
- core/common
- core/coords
- mind/evaluation
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## resources/assets/fonts
*(no internal dependencies)*

## setup/gamesetup
- campaign/overworld/core
- campaign/overworld/node
- campaign/overworld/tick
- core/common
- core/config
- core/coords
- gui/framework
- gui/guicombat
- gui/guiexploration
- gui/guinodeplacement
- gui/guioverworld
- gui/guiprogression
- gui/guiraid
- gui/guisquads
- gui/guiunitview
- mind/ai
- mind/combatlifecycle
- mind/encounter
- setup/savesystem
- setup/savesystem/chunks
- tactical/combat/combatcore
- tactical/combat/combatservices
- tactical/commander
- tactical/powers/artifacts
- tactical/powers/perks
- tactical/squads/roster
- tactical/squads/squadcommands
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates
- testing
- testing/bootstrap
- visual/graphics
- world/garrisongen
- world/worldgen
- world/worldmapcore

## setup/savesystem
- core/common

## setup/savesystem/chunks
- campaign/raid
- core/common
- core/coords
- setup/savesystem
- tactical/commander
- tactical/powers/artifacts
- tactical/powers/progression
- tactical/powers/spells
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/unitdefs
- tactical/squads/unitprogression
- templates
- visual/graphics
- world/worldmapcore

## tactical/combat/battlelog
- core/common
- tactical/combat/combatmath
- tactical/combat/combattypes
- tactical/squads/squadcore

## tactical/combat/combatcore
- core/common
- core/config
- core/coords
- tactical/combat/battlelog
- tactical/combat/combatmath
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/effects
- tactical/squads/squadcore

## tactical/combat/combatmath
- core/common
- tactical/combat/combattypes
- tactical/squads/squadcore
- tactical/squads/unitdefs

## tactical/combat/combatservices
- core/common
- core/coords
- tactical/combat/battlelog
- tactical/combat/combatcore
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/artifacts
- tactical/powers/effects
- tactical/powers/perks
- tactical/powers/powercore
- tactical/squads/squadcore

## tactical/combat/combatstate
- core/common
- core/coords
- tactical/squads/squadcore

## tactical/combat/combattypes
- core/common

## tactical/commander
- campaign/overworld/core
- campaign/overworld/tick
- core/common
- core/coords
- tactical/powers/progression
- tactical/squads/roster

## tactical/powers/artifacts
- core/common
- core/config
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/effects
- tactical/powers/powercore
- tactical/squads/squadcore
- templates

## tactical/powers/effects
- core/common

## tactical/powers/perks
- core/common
- core/config
- core/coords
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/powercore
- tactical/squads/squadcore
- tactical/squads/unitdefs

## tactical/powers/powercore
- core/common
- tactical/combat/combatstate
- tactical/combat/combattypes

## tactical/powers/progression
- core/common
- tactical/powers/perks
- templates

## tactical/powers/spells
- core/common
- tactical/combat/combatstate
- tactical/commander
- tactical/powers/effects
- tactical/powers/progression
- tactical/squads/squadcore
- templates

## tactical/squads/roster
- core/common
- tactical/squads/squadcore

## tactical/squads/squadcommands
- core/common
- core/coords
- tactical/combat/combatcore
- tactical/combat/combatstate
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/squadservices
- tactical/squads/unitdefs

## tactical/squads/squadcore
- core/common
- core/config
- core/coords
- tactical/squads/unitdefs
- tactical/squads/unitprogression
- templates

## tactical/squads/squadservices
- core/common
- core/coords
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/unitdefs

## tactical/squads/unitdefs
- core/common
- core/config
- tactical/squads/unitprogression
- templates

## tactical/squads/unitprogression
- core/common

## templates
- core/common
- core/config
- core/coords
- world/worldmapcore

## testing
- core/common
- visual/graphics
- visual/vfx
- world/worldmapcore

## testing/bootstrap
- campaign/overworld/core
- campaign/overworld/faction
- core/common
- core/config
- core/coords
- tactical/commander
- tactical/powers/artifacts
- tactical/powers/spells
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates
- world/worldmapcore

## tools/combat_balance
*(no internal dependencies)*

## tools/combat_simulator
- core/common
- core/coords
- tactical/combat/battlelog
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## tools/combat_visualizer
*(no internal dependencies)*

## tools/report_compressor
*(no internal dependencies)*

## visual/combatrender
- core/coords
- visual/rendering

## visual/graphics
- core/common
- core/coords

## visual/maprender
- core/coords
- visual/graphics
- visual/rendering
- world/worldmapcore

## visual/rendering
- core/common
- core/coords
- visual/graphics
- world/worldmapcore

## visual/vfx
- core/common
- core/config
- core/coords
- visual/graphics

## world/garrisongen
- core/common
- core/coords
- world/worldgen
- world/worldmapcore

## world/worldgen
- core/common
- core/coords
- visual/graphics
- world/worldmapcore

## world/worldmapcore
- core/common
- core/config
- core/coords
- visual/graphics
