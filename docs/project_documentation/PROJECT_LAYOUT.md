# TinkerRogue Project Layout

```
TinkerRogue/                        # package main (root)
├── game_main/                      # Starting point of game
├── common/                         # Used by multiple packages. Important ECS components
├── config/                         # Default values. Gobal configuration settings
├── gamesetup/                      # Handles setup prior to game launch
├── gear/                           # Artifacts. Items used in combat. 
├── input/                          # User input which is not handled by the GUI
├── savesystem/                     # Save System
│   └── chunks/                     # Handles saving specific portions of the game
├── templates/                      # Reads JSON files for game data
├── testing/                        # Creates testing data for game
│   └── bootstrap/                  # package bootstrap
├── gui/
│   ├── builders/                   # High level widget builders
│   ├── framework/                  # Core gui operations. Used in different parts of the GUI
│   ├── specs/                      # Basic Gi layout information
│   ├── widgetresources/            # Widget Resources
│   ├── widgets/                    # Widgets
│   ├── guiartifacts/               # Artifact handling in combat
│   ├── guicombat/                  # Handles main combat operations
│   ├── guiexploration/             # package guiexploration
│   ├── guiinspect/                 # Inspects squads in combat
│   ├── guinodeplacement/           # GUI handling for placing nodes in Overworld
│   ├── guioverworld/               # GUI handling for overworld oeprations
│   ├── guiraid/                    # GUI for raids
│   ├── guispells/                  # GUI handling of spells in combat
│   ├── guisquads/                  # GUI for editing and creating squads. Also allows unit purchase
│   ├── guistartmenu/               # package guistartmenu
│   └── guiunitview/                # GUI for unit detail
├── mind/
│   ├── ai/                         # High level handling of AI operations
│   ├── behavior/                   # Handles combat in behavior
│   ├── combatpipeline/             # Interface for combat rewards. Basic combat setup.
│   ├── encounter/                  # Random encounter evaluation
│   ├── evaluation/                 # package evaluation
│   └── raid/                       # Handles raid system
├── tactical/
│   ├── combat/                     # Handles game combat logic
│   │   └── battlelog/              # package battlelog
│   ├── combatservices/             # Handles certain combat information
│   ├── commander/                  # package commander
│   ├── effects/                    # Effects are used by artifact and spells
│   ├── spells/                     # Spell system
│   ├── squadcommands/              # Allows undo and redo of squad related commands
│   ├── squads/                     # Handles squad logic
│   └── squadservices/              # Handles complex squad operations
├── overworld/
│   ├── core/                       # package core
│   ├── faction/                    # package faction
│   ├── garrison/                   # package garrison
│   ├── influence/                  # package influence
│   ├── node/                       # package node
│   ├── overworldlog/               # package overworldlog
│   ├── threat/                     # package threat
│   ├── tick/                       # package tick
│   └── victory/                    # package victory
├── visual/
│   ├── graphics/                   # package graphics
│   └── rendering/                  # package rendering
├── world/
│   ├── coords/                     # package coords
│   └── worldmap/                   # package worldmap
└── tools/
    ├── combat_balance/             # package main
    ├── combat_simulator/           # package main
    ├── combat_visualizer/          # package main
    └── report_compressor/          # package main
```
