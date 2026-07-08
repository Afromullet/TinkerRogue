# Package Dependencies

**Generated:** 2026-07-02
**Module:** `game_main` (see `go.mod`)
**Total packages:** 86

Lists every internal Go package in the module and the other internal packages it imports. Third-party imports (`github.com/...`, `golang.org/...`) and the Go standard library are excluded. Paths are shown relative to the module root.

Sorted alphabetically by package path. Packages listed with *(no internal dependencies)* import only stdlib or third-party libraries.

> **Note on `game_main` entries:** The module itself is named `game_main`, and there is also a subdirectory `game_main/` containing the main binary. To disambiguate, the module-root package (`test_init.go`) is listed as **`game_main (module root)`** and the subdirectory is listed as **`game_main/game_main (main binary)`**.

---

## campaign/overworld/core
- campaign/overworld/ids
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
- campaign/overworld/ids
- core/common
- core/coords
- tactical/squads/roster
- tactical/squads/squadcore
- tactical/squads/unitdefs

## campaign/overworld/ids
*(no internal dependencies)*

## campaign/overworld/influence
- campaign/overworld/core
- campaign/overworld/ids
- core/common
- core/coords
- templates

## campaign/overworld/node
- campaign/overworld/core
- campaign/overworld/ids
- core/common
- core/coords
- templates

## campaign/overworld/overworldlog
*(no internal dependencies)*

## campaign/overworld/threat
- campaign/overworld/core
- campaign/overworld/ids
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
- tactical/commander

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
- tactical/commander
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
- mind/encounter
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/powers/artifacts
- tactical/powers/powercore

## gui/guicombat
- core/common
- core/config
- gui/builders
- gui/framework
- gui/guiartifacts
- gui/guicombat/combatanimation
- gui/guicombat/combatbase
- gui/guicombat/combatinput
- gui/guicombat/combatvisualization
- gui/guiinspect
- gui/guispells
- gui/guisquads
- gui/specs
- gui/widgetresources
- gui/widgets
- mind/combatlifecycle
- mind/encounter
- tactical/combat/battlelog
- tactical/combat/combatservices
- tactical/combat/combattypes
- tactical/powers/spells
- templates

## gui/guicombat/combatanimation
- gui/builders
- gui/framework
- tactical/combat/combatcore
- tactical/combat/combatmath
- tactical/squads/squadcore
- visual/combatrender

## gui/guicombat/combatbase
- core/coords
- gui/framework
- gui/guicombat/combatanimation
- mind/encounter
- tactical/combat/combatdisposal
- tactical/combat/combatservices
- tactical/squads/squadcommands

## gui/guicombat/combatinput
- core/common
- core/coords
- gui/framework
- gui/guiartifacts
- gui/guicombat/combatbase
- gui/guicombat/combatvisualization
- gui/guiinspect
- gui/guispells
- tactical/combat/combatstate
- tactical/squads/squadcore

## gui/guicombat/combatvisualization
- core/common
- core/coords
- gui/framework
- tactical/combat/combatservices
- tactical/combat/combatstate
- visual/combatrender
- visual/graphics
- world/worldmapcore

## gui/guiexploration
- campaign/overworld/core
- core/common
- core/coords
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
- campaign/overworld/ids
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
- campaign/overworld/ids
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
- core/config
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
- mind/encounter
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
- core/config
- core/coords
- tactical/combat/combatstate
- tactical/powers/perks
- tactical/powers/progression
- tactical/powers/spells
- tactical/squads/squadcore
- tactical/squads/unitprogression

## mind/encounter
- campaign/overworld/core
- campaign/overworld/garrison
- campaign/overworld/ids
- campaign/overworld/threat
- core/common
- core/coords
- mind/combatlifecycle
- mind/spawning
- tactical/commander
- tactical/squads/roster
- tactical/squads/squadcore
- templates

## mind/evaluation
- core/common
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## mind/spawning
- campaign/overworld/core
- campaign/overworld/ids
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
- campaign/overworld/faction
- campaign/overworld/ids
- campaign/overworld/node
- campaign/overworld/tick
- core/common
- core/config
- core/coords
- gui/framework
- gui/guicombat
- gui/guicombat/combatanimation
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
- tactical/combat/combatmath
- tactical/combat/combatservices
- tactical/commander
- tactical/powers/artifacts
- tactical/powers/perks
- tactical/powers/spells
- tactical/squads/roster
- tactical/squads/squadcommands
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates
- testing/bootstrap
- world/garrisongen
- world/worldgen
- world/worldmapcore

## setup/savesystem
- core/common

## setup/savesystem/chunks
- campaign/raid
- core/common
- core/config
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
- world/worldmapcore

## tactical/combat/battlelog
- core/common
- tactical/combat/combatmath
- tactical/combat/combattypes
- tactical/squads/squadcore

## tactical/combat/combatcore
- core/common
- core/coords
- tactical/combat/battlelog
- tactical/combat/combatdisposal
- tactical/combat/combatmath
- tactical/combat/combatstate
- tactical/combat/combattypes
- tactical/powers/effects
- tactical/squads/squadcore

## tactical/combat/combatdisposal
- core/common
- core/coords
- tactical/combat/combatstate
- tactical/squads/squadcore

## tactical/combat/combatmath
- core/common
- core/config
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
- core/common
- core/coords
- tactical/powers/progression
- tactical/squads/roster
- templates

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
- tactical/combat/combatdisposal
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
- campaign/overworld/ids
- core/common
- core/config
- core/coords
- world/garrisongen/roomtypes

## testing
- core/common

## testing/bootstrap
- core/common
- tactical/commander
- tactical/powers/artifacts
- tactical/squads/roster
- templates

## tools
- tools/combat_analysis/combat_balance
- tools/combat_analysis/combat_simulator
- tools/combat_analysis/combat_visualizer
- tools/combat_analysis/report_compressor

## tools/combat_analysis/combat_balance
- tools/combat_analysis/shared

## tools/combat_analysis/combat_simulator
- core/common
- core/coords
- tactical/combat/battlelog
- tactical/combat/combatservices
- tactical/combat/combatstate
- tactical/squads/squadcore
- tactical/squads/unitdefs
- templates

## tools/combat_analysis/combat_visualizer
- tools/combat_analysis/shared

## tools/combat_analysis/report_compressor
- tools/combat_analysis/shared

## tools/combat_analysis/shared
*(no internal dependencies)*

## visual/combatrender
- core/coords
- visual/rendering

## visual/graphics
- core/common
- core/coords

## visual/maprender
- core/coords
- visual/rendering
- world/worldmapcore

## visual/rendering
- core/common
- core/coords
- world/worldmapcore

## visual/vfx
- core/common
- core/config
- core/coords

## world/garrisongen
- core/common
- core/coords
- world/garrisongen/roomtypes
- world/worldgen
- world/worldmapcore

## world/garrisongen/roomtypes
*(no internal dependencies)*

## world/worldgen
- core/common
- core/coords
- world/worldmapcore

## world/worldmapcore
- core/common
- core/config
- core/coords
- visual/graphics

