# TinkerRogue Project Layout

Annotated package tree with file-level descriptions for key files.

```
TinkerRogue/
├── common/              # Core ECS utilities, shared components
│   ├── ecsutil.go      # Type-safe component access helpers
│   ├── commoncomponents.go  # Position, Attributes, Name
│   ├── positionsystem.go    # O(1) spatial grid (GlobalPositionSystem)
│   └── playerdata.go   # Player state
│
├── config/              # Centralized game constants
│   └── config.go            # Map dimensions, debug flags, profiling
│
├── world/               # World systems
│   ├── coords/         # Coordinate management (CRITICAL)
│   │   ├── cordmanager.go  # Global CoordManager singleton
│   │   └── position.go     # LogicalPosition, PixelPosition types
│   └── worldmap/       # Procedural generation
│       ├── generator.go         # Generator registry
│       ├── gen_rooms_corridors.go
│       ├── gen_tactical_biome.go
│       └── gen_overworld.go
│
├── tactical/            # Tactical gameplay systems
│   ├── squads/         # Squad system (REFERENCE IMPLEMENTATION)
│   │   ├── squadcomponents.go   # 8 pure data components
│   │   ├── squadqueries.go      # Query functions
│   │   ├── squadcombat.go       # Combat logic
│   │   ├── squadabilities.go    # Leader abilities
│   │   └── squadmanager.go      # Initialization
│   ├── combat/         # Turn-based combat management
│   │   ├── turnmanager.go       # Turn order, round tracking
│   │   ├── combatfactionmanager.go  # Faction system
│   │   └── battlelog/           # Battle recording and export
│   ├── combatservices/ # Combat service layer
│   ├── squadservices/  # Squad service layer
│   ├── squadcommands/  # Squad command pattern (undo/redo)
│   ├── commander/      # Commander entities
│   │   ├── components.go       # CommanderData, ActionState, Roster
│   │   ├── movement.go         # Overworld movement
│   │   ├── queries.go          # Commander lookups
│   │   ├── roster.go           # Squad roster management
│   │   ├── turnstate.go        # Overworld turn tracking
│   │   └── system.go           # Commander system logic
│   ├── spells/         # Spell system
│   │   ├── components.go       # ManaData, SpellBookData
│   │   ├── system.go           # Spell casting logic
│   │   └── queries.go          # Spell lookups
│   └── effects/        # Active effects system
│       ├── components.go       # ActiveEffectsData, StatType
│       ├── system.go           # Effect application/tick-down
│       └── queries.go          # Effect lookups
│
├── mind/                # AI and encounter systems
│   ├── ai/             # Utility AI decision-making
│   │   ├── ai_controller.go    # AI turn controller
│   │   └── action_evaluator.go # Action scoring
│   ├── behavior/       # Threat analysis
│   │   ├── threat_layers.go    # Multi-layer threat maps
│   │   ├── threat_painting.go  # Threat projection
│   │   └── dangerlevel.go      # Danger level queries
│   ├── evaluation/     # Power and role evaluation
│   │   ├── power.go            # Squad power scoring
│   │   ├── roles.go            # Role-based multipliers
│   │   └── cache.go            # Evaluation caching
│   ├── encounter/      # Encounter system
│   │   ├── encounter_service.go   # Core service (start/resolve)
│   │   ├── encounter_trigger.go   # Trigger conditions
│   │   ├── encounter_setup.go     # Enemy squad generation
│   │   ├── encounter_resolution.go # Post-combat cleanup
│   │   ├── encounter_generator.go # Config-driven generation
│   │   └── rewards.go             # XP and loot
│   ├── combatpipeline/ # Interface for combat rewards, setup
│   └── raid/           # Raid system
│
├── overworld/           # Strategic overworld layer
│   ├── core/           # Shared components and types
│   │   ├── components.go      # All overworld ECS components
│   │   ├── types.go           # FactionType, NodeCategory, enums
│   │   ├── node_registry.go   # Node type definitions (JSON)
│   │   ├── walkability.go     # Overworld walkable grid
│   │   ├── events.go          # Event types for tick results
│   │   └── resources.go       # Global overworld context
│   ├── tick/           # Strategic turn clock
│   │   └── tickmanager.go     # AdvanceTick, tick processing
│   ├── faction/        # NPC faction AI
│   │   ├── system.go          # Faction tick processing
│   │   ├── archetype.go       # Faction behavior archetypes
│   │   └── scoring.go         # Intent scoring functions
│   ├── threat/         # Threat node growth
│   │   ├── system.go          # Threat growth logic
│   │   └── queries.go         # Threat lookups
│   ├── influence/      # Influence radius system
│   │   ├── system.go          # Influence recalculation
│   │   ├── effects.go         # Interaction effects
│   │   └── queries.go         # Influence lookups
│   ├── node/           # Unified node management
│   │   ├── system.go          # Node creation
│   │   ├── queries.go         # Node lookups
│   │   └── validation.go      # Placement validation
│   ├── garrison/       # Garrison defense
│   │   ├── system.go          # Garrison logic, raid detection
│   │   └── queries.go         # Garrison lookups
│   ├── victory/        # Win/loss conditions
│   │   ├── system.go          # Victory checks
│   │   └── queries.go         # Victory state lookups
│   └── overworldlog/   # Event recording
│       ├── overworld_recorder.go  # Event capture
│       ├── overworld_summary.go   # Summary generation
│       └── overworld_export.go    # File export
│
├── gear/                # Inventory and items
│   ├── Inventory.go         # Pure ECS inventory (REFERENCE)
│   ├── items.go             # Item components
│   └── inventory_service.go # Service layer
│
├── gui/                 # User interface
│   ├── framework/      # Core mode infrastructure
│   │   ├── uimode.go          # UIMode interface, UIContext
│   │   ├── basemode.go        # Common mode infrastructure
│   │   ├── modemanager.go     # Mode lifecycle & transitions
│   │   ├── coordinator.go     # GameModeCoordinator (two-context system)
│   │   ├── contextstate.go    # TacticalState, OverworldState
│   │   ├── modebuilder.go     # Declarative mode configuration
│   │   ├── panelregistry.go   # Global panel type registry
│   │   ├── guiqueries.go      # ECS query abstraction
│   │   └── commandhistory.go  # Undo/redo system
│   ├── builders/       # UI construction helpers
│   │   ├── panels.go          # Panel building with functional options
│   │   ├── layout.go          # Layout calculations
│   │   ├── dialogs.go         # Modal dialog builders
│   │   └── panelspecs.go      # Standard panel specifications
│   ├── widgets/        # Widget wrappers & utilities
│   │   ├── cached_list.go     # Cached list (90% CPU reduction)
│   │   ├── cached_textarea.go # Cached text area
│   │   └── textdisplay.go     # Text display widget
│   ├── specs/          # Layout specifications
│   │   └── layout.go          # Responsive layout configuration
│   ├── widgetresources/ # Shared UI resources
│   │   ├── guiresources.go     # UI resource loading
│   │   └── cachedbackground.go # Cached background rendering
│   ├── guicombat/      # Combat mode implementation
│   ├── guisquads/      # Squad management modes (editor, purchase, deployment, artifacts)
│   ├── guioverworld/   # Overworld mode
│   ├── guinodeplacement/ # Node placement mode
│   ├── guiexploration/ # Exploration mode
│   ├── guispells/      # Spell casting UI
│   ├── guiartifacts/   # Artifact management UI
│   ├── guiinspect/     # Squad inspection in combat
│   ├── guiraid/        # Raid UI
│   ├── guiunitview/    # Unit detail view
│   └── guistartmenu/   # Start menu (Overworld vs Roguelike)
│
├── visual/              # Rendering systems
│   ├── graphics/       # Graphics utilities
│   └── rendering/      # Batch rendering, sprite management
│
├── input/               # Input handling
│   └── cameracontroller.go  # WASD movement, diagonals, map scroll toggle
│
├── templates/           # JSON-based entity creation
│   ├── templatelib.go       # Template registry
│   ├── creators.go          # Factory functions
│   └── readdata.go          # JSON loading
│
├── savesystem/          # Save system
│   └── chunks/              # Handles saving specific portions of the game
│
├── testing/             # Test data and bootstrapping
│   ├── testingdata.go       # Test item creation
│   ├── fixtures.go          # Test fixtures
│   └── bootstrap/           # Initial game entity seeding
│       ├── initial_squads.go     # Starting squads
│       ├── initial_commanders.go # Starting commanders
│       ├── initial_factions.go   # Starting factions
│       └── initial_artifacts.go  # Starting artifacts
│
├── tools/               # Development tools
│   ├── combat_balance/        # Combat balance analysis
│   ├── combat_simulator/      # Combat simulation suites
│   ├── combat_visualizer/     # Battle visualization
│   └── report_compressor/     # Report compression
│
├── gamesetup/           # Game configuration and booting
│
└── game_main/           # Entry point and initialization
    ├── main.go              # Game struct, loop, start menu
    ├── gameinit.go          # ECS initialization
    ├── componentinit.go     # Component registration
    ├── setup_shared.go      # Shared bootstrap (data, ECS, player)
    ├── setup_overworld.go   # Overworld mode setup
    └── setup_roguelike.go   # Roguelike mode setup
```
