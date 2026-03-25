# TinkerRogue Project Layout

Tactical roguelike RPG built in Go with ECS architecture, turn-based combat, squad management, procedural generation, and strategic overworld gameplay.

---

## Top-Level Structure

```
TinkerRogue/
├── common/           # ECS core, shared components, spatial grid
├── game_main/        # Application entry point
├── gui/              # UI framework, modes, and widgets
├── input/            # Camera and input control
├── mind/             # AI, threat assessment, encounters, raids
├── overworld/        # Strategic map, factions, territory
├── setup/            # Configuration, initialization, save system
├── tactical/         # Combat, squads, powers, artifacts
├── templates/        # Game data loading, entity factory
├── testing/          # Test utilities and fixtures
├── tools/            # Standalone CLI tools (simulator, visualizer)
├── visual/           # Graphics primitives and rendering
├── world/            # Coordinates, map generation
├── assets/           # Sprites, fonts, game data (JSON)
├── docs/             # Architecture and system documentation
├── simulation_logs/  # Simulation output
└── test_init.go      # Root-level test initialization
```

---

## Package Details

### common/ -- ECS Core and Shared Infrastructure

| File | Purpose |
|------|---------|
| `ecsutil.go` | EntityManager wrapper, type-safe component access |
| `commoncomponents.go` | Shared components (Name, Attributes) |
| `positionsystem.go` | O(1) spatial grid for entity lookups |
| `playerdata.go` | Player/session state |
| `resources.go` | Global resource management |
| `randnumgen.go` | Random number generation |
| `dirtycache.go` | Cache invalidation system |

### game_main/ -- Entry Point

| File | Purpose |
|------|---------|
| `main.go` | Game bootstrap and event loop |
| `setup.go` | Game initialization sequence |

Subdirs: `combat_logs/`, `saves/`, `simulation_logs/`

### gui/ -- UI Framework and Mode Implementations

```
gui/
├── framework/          # Core: mode manager, context state, input bindings, queries
│   ├── modemanager.go          # UI mode switching and lifecycle
│   ├── coordinator.go          # Overall GUI orchestration
│   ├── contextstate.go         # UI state management
│   ├── uimode.go               # UIMode interface
│   ├── basemode.go             # Base class for all UI modes
│   ├── modebuilder.go          # Builder pattern for modes
│   ├── inputaction.go          # Input action definitions
│   ├── inputbinding.go         # Input binding system
│   ├── actionmap.go            # Action mapping
│   ├── defaultbindings.go      # Default keybindings
│   ├── guiqueries.go           # Game state accessors
│   ├── guiqueries_overworld.go # Overworld-specific queries
│   ├── guiqueries_rendering.go # Rendering queries
│   ├── guiqueries_units.go     # Unit queries
│   ├── commandhistory.go       # Undo/redo system
│   ├── submenu.go              # Submenu handling
│   ├── panelregistry.go        # Widget/panel management
│   └── squadinfo_cache.go      # Cached squad display data
├── builders/           # Widget/panel construction helpers
│   ├── dialogs.go              # Modal dialog builders
│   ├── layout.go               # Layout helpers (grid, horizontal, vertical)
│   ├── lists.go                # List widget builders
│   ├── panels.go               # Panel construction
│   ├── panelspecs.go           # Panel specifications
│   └── widgets.go              # Generic widget helpers
├── widgets/            # Custom widgets
│   ├── cached_list.go          # Optimized list rendering
│   ├── cached_textarea.go      # Optimized text area
│   └── textdisplay.go          # Text display widget
├── widgetresources/    # Shared GUI resources
│   ├── guiresources.go         # Centralized resource management
│   └── cachedbackground.go     # Cached background loading
├── specs/              # Layout specifications
│   └── layout.go               # Layout specs and constraints
├── guicombat/          # Tactical combat UI mode
│   ├── combatmode.go                       # Main combat mode
│   ├── combat_input_handler.go             # Input processing
│   ├── combat_action_handler.go            # Action execution UI
│   ├── combat_turn_flow.go                 # Turn-based flow
│   ├── combatvisualization.go              # Grid and unit visuals
│   ├── threatvisualizer.go                 # Threat overlays
│   ├── combat_animation_mode.go            # Animation support
│   ├── combat_animation_panels_registry.go # Animation panels
│   ├── combat_panels_registry.go           # Combat panels
│   └── combatdeps.go                       # Dependency injection
├── guisquads/          # Squad editing, deployment, unit purchasing
│   ├── squadeditormode.go                  # Squad editing mode
│   ├── squaddeploymentmode.go              # Pre-battle deployment
│   ├── unitpurchasemode.go                 # Unit recruitment
│   ├── artifactmode.go                     # Artifact management
│   ├── squadcomponents.go                  # Reusable UI components
│   ├── commanderselector.go                # Commander selection
│   ├── squadselector.go                    # Squad selection
│   ├── squadeditor_grid.go                 # Editor grid
│   ├── squadeditor_roster.go               # Editor roster
│   ├── squadeditor_refresh.go              # Editor refresh logic
│   ├── artifact_panels_registry.go         # Artifact panels
│   ├── artifact_refresh.go                 # Artifact refresh
│   ├── squadeditor_panels_registry.go      # Editor panels
│   ├── squaddeployment_panels_registry.go  # Deployment panels
│   └── unitpurchase_panels_registry.go     # Purchase panels
├── guioverworld/       # Overworld exploration UI
│   ├── overworldmode.go                    # Overworld mode
│   ├── overworld_input_handler.go          # Input handling
│   ├── overworld_action_handler.go         # Action processing
│   ├── overworld_renderer.go               # Map rendering
│   ├── overworld_panels_registry.go        # UI panels
│   ├── overworld_formatters.go             # Text formatting
│   └── overworld_deps.go                   # Dependencies
├── guiraid/            # Raid preparation UI
│   ├── raidmode.go                         # Raid mode
│   ├── raidstate.go                        # Raid state tracking
│   ├── deploy_panel.go                     # Squad deployment
│   ├── floormap_panel.go                   # Floor visualization
│   ├── floormap_renderer.go                # Floor rendering
│   ├── summary_panel.go                    # Raid summary
│   └── raid_panels_registry.go             # Panels
├── guiartifacts/       # Artifact management UI
│   ├── artifact_panel.go                   # Artifact display
│   ├── artifact_handler.go                 # User interaction
│   └── artifact_deps.go                    # Dependencies
├── guispells/          # Spell management UI
│   ├── spell_panel.go                      # Spell display
│   ├── spell_handler.go                    # User interaction
│   └── spell_deps.go                       # Dependencies
├── guiinspect/         # Unit inspection
│   └── inspect_panel.go                    # Detailed unit info
├── guiunitview/        # Unit view mode
│   ├── unitviewmode.go                     # View mode
│   └── unitview_panels_registry.go         # Panels
├── guiexploration/     # Exploration mode
│   ├── explorationmode.go                  # Exploration UI
│   └── exploration_panels_registry.go      # Panels
├── guinodeplacement/   # Node placement mode
│   ├── nodeplacementmode.go                # Placement mode
│   ├── nodeplacement_renderer.go           # Visualization
│   └── nodeplacement_panels_registry.go    # Panels
└── guistartmenu/       # Main menu
    └── startmenu.go                        # Start menu
```

### input/ -- Input Handling

| File | Purpose |
|------|---------|
| `cameracontroller.go` | Camera movement and viewport control |

### mind/ -- AI and Behavior Systems

```
mind/
├── ai/                 # AI decision making
│   ├── ai_controller.go       # Main AI orchestrator for factions
│   └── action_evaluator.go    # Action scoring and selection
├── behavior/           # Multi-layer threat assessment and positioning
│   ├── threat_layers.go       # Multi-layer threat evaluation
│   ├── threat_composite.go    # Threat aggregation
│   ├── threat_combat.go       # Combat-related threats
│   ├── threat_positional.go   # Positional threats (cover, distance)
│   ├── threat_painting.go     # Visual threat map generation
│   ├── threat_support.go      # Support role threats
│   ├── threat_gridutils.go    # Grid utilities
│   ├── threat_queries.go      # Query helpers
│   ├── threat_constants.go    # Configuration constants
│   └── dangerlevel.go         # Danger level assessment
├── evaluation/         # Unit power scoring, role analysis
│   ├── power.go               # Unit power scoring (damage, durability, utility)
│   ├── power_config.go        # Power calculation config and multipliers
│   └── roles.go               # Unit role definitions and analysis
├── encounter/          # Random encounter generation
│   ├── encounter_generator.go # Generates random tactical encounters
│   ├── encounter_service.go   # Service interface
│   ├── encounter_config.go    # Configuration
│   ├── encounter_setup.go     # Setup helpers
│   ├── encounter_trigger.go   # Trigger conditions
│   ├── types.go               # Type definitions
│   ├── starters.go            # Encounter startup
│   └── resolvers.go           # Encounter resolution
├── raid/               # Multi-floor raid generation and execution
│   ├── raidrunner.go          # Main raid executor
│   ├── raidencounter.go       # Encounter within raid
│   ├── components.go          # ECS components for raids
│   ├── init.go                # Initialization
│   ├── config.go              # Configuration
│   ├── deployment.go          # Unit deployment
│   ├── garrison.go            # Enemy garrison management
│   ├── archetypes.go          # Enemy unit archetypes
│   ├── assignment.go          # Squad assignment to floors
│   ├── alert.go               # Alert/alarm system
│   ├── recovery.go            # Recovery between encounters
│   ├── floorgraph.go          # Floor graph navigation
│   ├── queries.go             # Query helpers
│   ├── starters.go            # Phase handlers
│   ├── resolvers.go           # Resolution handlers
│   └── rewards.go             # Reward distribution
└── combatlifecycle/    # Combat pipeline (start, enroll, cleanup, rewards)
    ├── pipeline.go            # Main combat pipeline orchestration
    ├── starter.go             # Combat startup and initialization
    ├── enrollment.go          # Faction enrollment
    ├── cleanup.go             # Combat cleanup and teardown
    ├── casualties.go          # Casualty handling and unit death
    ├── reward.go              # Reward distribution
    └── helpers.go             # Utility functions
```

### overworld/ -- Strategic Map System

```
overworld/
├── core/               # Core infrastructure
│   ├── components.go          # ECS components for overworld entities
│   ├── init.go                # Subsystem initialization
│   ├── types.go               # Type definitions
│   ├── config.go              # Configuration
│   ├── events.go              # Event system
│   ├── resources.go           # Resource management
│   ├── node_registry.go       # Node type registry
│   ├── utils.go               # Utility functions
│   └── walkability.go         # Pathfinding walkability
├── faction/            # Faction management
│   ├── archetype.go           # Faction archetypes and templates
│   ├── system.go              # Faction simulation and updates
│   └── scoring.go             # Faction scoring
├── node/               # Location/node management
│   ├── system.go              # Node update and management
│   ├── queries.go             # Node query functions
│   └── validation.go          # Node validation
├── garrison/           # Military garrisoning
│   ├── system.go              # Garrison management
│   └── queries.go             # Garrison queries
├── influence/          # Influence spreading
│   ├── system.go              # Influence propagation
│   ├── effects.go             # Influence effects on nodes
│   └── queries.go             # Influence queries
├── threat/             # Threat assessment
│   ├── system.go              # Threat calculation
│   └── queries.go             # Threat queries
├── tick/               # Turn management
│   └── tickmanager.go         # Overworld turn/tick management
├── victory/            # Victory conditions
│   ├── system.go              # Victory condition tracking
│   └── queries.go             # Victory state queries
└── overworldlog/       # Logging and export
    ├── overworld_recorder.go  # Record overworld events
    ├── overworld_export.go    # Export functionality
    └── overworld_summary.go   # Summary generation
```

### setup/ -- Configuration and Initialization

```
setup/
├── config/
│   └── config.go                   # Global game config (debug mode, difficulty)
├── gamesetup/
│   ├── bootstrap.go                # Game bootstrap sequence
│   ├── ecsinit.go                  # ECS subsystem initialization
│   ├── moderegistry.go             # Register all UI modes
│   ├── playerinit.go               # Player initialization
│   ├── mapgenconfig.go             # Map generation config
│   ├── helpers.go                  # Setup helpers
│   ├── savehelpers.go              # Save system helpers
│   └── savesystem/
│       ├── savesystem.go           # Main save/load interface
│       ├── idmap.go                # ID mapping during deserialization
│       └── chunks/                 # Serialization by entity type
│           ├── shared_types.go     # Common serialization types
│           ├── squad_chunk.go      # Squad serialization
│           ├── commander_chunk.go  # Commander serialization
│           ├── gear_chunk.go       # Artifact/equipment serialization
│           ├── map_chunk.go        # Map/tile serialization
│           ├── player_chunk.go     # Player state serialization
│           └── raid_chunk.go       # Raid state serialization
└── savesystem/                     # Alternate save system location
    ├── savesystem.go               # Save/load interface
    ├── idmap.go                    # ID mapping
    └── chunks/                     # Same chunk structure as above
        ├── shared_types.go
        ├── squad_chunk.go
        ├── commander_chunk.go
        ├── gear_chunk.go
        ├── map_chunk.go
        ├── player_chunk.go
        └── raid_chunk.go
```

### tactical/ -- Turn-Based Combat System

```
tactical/
├── squads/
│   ├── squadcore/              # Core squad system
│   │   ├── squadcomponents.go      # Squad/unit ECS components
│   │   ├── squadqueries.go         # Squad query functions
│   │   ├── squadmanager.go         # Squad creation and lifecycle
│   │   ├── squadcreation.go        # Squad creation helpers
│   │   ├── squadcache.go           # Squad caching
│   │   ├── units.go                # Unit helper functions
│   │   └── squadabilities.go       # Squad special abilities
│   ├── squadcommands/          # Command pattern for squad operations
│   │   ├── command.go              # Command base interface
│   │   ├── command_executor.go     # Command execution
│   │   ├── command_helpers.go      # Shared helpers
│   │   ├── add_unit_command.go     # Add unit to squad
│   │   ├── remove_unit_command.go  # Remove unit from squad
│   │   ├── move_unit_command.go    # Move unit in formation
│   │   ├── move_squad_command.go   # Move entire squad
│   │   ├── purchase_unit_command.go    # Purchase/recruit unit
│   │   ├── change_leader_command.go    # Change squad leader
│   │   ├── rename_squad_command.go     # Rename squad
│   │   └── reorder_squads_command.go   # Reorder squads
│   ├── squadservices/          # Service layer
│   │   ├── squad_deployment_service.go # Pre-combat deployment
│   │   └── unit_purchase_service.go    # Unit purchasing
│   ├── unitdefs/               # Unit definitions
│   │   ├── enums.go                # Unit type enumerations and roles
│   │   ├── templates.go            # Unit templates and stat arrays
│   │   └── filters.go              # Unit filtering utilities
│   ├── unitprogression/        # Unit advancement
│   │   ├── components.go           # Progression ECS components
│   │   └── experience.go           # Experience and leveling
│   └── roster/                 # Roster management
│       ├── init.go                 # Initialization
│       ├── squadroster.go          # Squad roster tracking
│       └── unitroster.go           # Unit roster management
├── combat/
│   ├── combatcore/             # Core combat system
│   │   ├── combatcomponents.go     # Combat ECS components
│   │   ├── combatqueries.go        # Combat queries
│   │   ├── combatqueriescache.go   # Cached queries for performance
│   │   ├── turnmanager.go          # Turn ordering and management
│   │   ├── combatactionsystem.go   # Action execution
│   │   ├── combatmovementsystem.go # Movement mechanics
│   │   ├── combatcalculation.go    # Damage and hit calculations
│   │   ├── combattargeting.go      # Target selection and validation
│   │   ├── combatcover.go          # Cover and defense system
│   │   ├── combatfactionmanager.go # Faction management in combat
│   │   ├── combatlogging.go        # Combat event logging
│   │   ├── combatprocessing.go     # Turn processing and resolution
│   │   ├── combatabilities.go      # Combat abilities
│   │   ├── combattypes.go          # Type definitions and enums
│   │   ├── combat_contracts.go     # Interfaces/contracts
│   │   ├── battle_recorder.go      # Battle recording
│   │   ├── battle_export.go        # Export battle data
│   │   ├── battle_summary.go       # Battle summary generation
│   │   └── combattestfx/           # Test fixtures
│   ├── combatservices/         # Combat service layer
│   │   ├── combat_service.go       # Main combat service interface
│   │   ├── ai_interfaces.go        # AI-facing interfaces
│   │   └── combat_events.go        # Event system for combat
│   └── combattypes/            # Combat type definitions
├── commander/                  # Commander/leader system
│   ├── components.go               # Commander ECS components
│   ├── system.go                   # Commander mechanics
│   ├── init.go                     # Initialization
│   ├── queries.go                  # Commander queries
│   ├── turnstate.go                # Turn state
│   ├── movement.go                 # Commander movement
│   └── roster.go                   # Commander roster
└── powers/
    ├── spells/                 # Spell system
    │   ├── components.go           # Spell ECS components
    │   ├── system.go               # Spell casting and mechanics
    │   ├── init.go                 # Initialization
    │   └── queries.go              # Spell queries
    ├── artifacts/              # Artifact/gear system
    │   ├── components.go           # Artifact ECS components
    │   ├── artifactinventory.go    # Inventory management
    │   ├── artifactcharges.go      # Charge tracking
    │   ├── artifactbehavior.go     # Behavior system
    │   ├── artifactbehaviors_passive.go    # Passive effects
    │   ├── artifactbehaviors_activated.go  # Activated abilities
    │   ├── system.go               # Artifact system
    │   ├── queries.go              # Artifact queries
    │   ├── init.go                 # Initialization
    │   └── effects/                # Artifact-specific effects
    │       ├── components.go       # Effect ECS components
    │       ├── system.go           # Effect mechanics
    │       ├── init.go             # Initialization
    │       └── queries.go          # Effect queries
    └── effects/                # General effect system (buffs, debuffs, DoT)
        ├── components.go           # Effect ECS components
        ├── system.go               # Effect mechanics
        ├── init.go                 # Initialization
        └── queries.go              # Effect queries
```

### templates/ -- Game Data and Entity Factory

| File | Purpose |
|------|---------|
| `gameconfig.go` | Game configuration loader |
| `registry.go` | Registry system for game data |
| `readdata.go` | JSON data reading utilities |
| `jsonschema.go` | JSON schema definitions |
| `entity_factory.go` | Create entities from templates |
| `artifactdefinitions.go` | Artifact definition loading |
| `spelldefinitions.go` | Spell definition loading |
| `difficulty.go` | Difficulty system |
| `validation.go` | Data validation |
| `namegen.go` | Name generation |

### testing/ -- Test Utilities

```
testing/
├── fixtures.go                 # Common test fixtures and helpers
├── testingdata.go              # Test data constants
└── bootstrap/
    ├── initial_squads.go       # Pre-configured test squads
    ├── initial_commanders.go   # Pre-configured test commanders
    ├── initial_artifacts.go    # Pre-configured test artifacts
    └── initial_factions.go     # Pre-configured test factions
```

**Note:** Functions from `testing/` must only be called when `config.DEBUG_MODE` is `true` in production code.

### tools/ -- Standalone CLI Tools

```
tools/
├── combat_simulator/       # Automated combat simulation
│   ├── main.go                 # Entry point
│   ├── bootstrap.go            # Setup and initialization
│   ├── combat_runner.go        # Simulation execution
│   ├── squad_factory.go        # Squad generation for scenarios
│   ├── scenarios.go            # Test scenario definitions
│   ├── blueprint.go            # Squad blueprints
│   ├── unit_pool.go            # Unit pool for generation
│   ├── suite_duels.go          # Duel test suite
│   ├── suite_compositions.go   # Composition test suite
│   ├── suite_encounters.go     # Encounter test suite
│   └── suite_stress.go         # Stress test suite
├── combat_balance/         # Balance analysis with CSV output
│   ├── main.go                 # Entry point
│   ├── aggregator.go           # Data aggregation
│   ├── csv_writer.go           # CSV output
│   ├── loader.go               # Data loading
│   └── types.go                # Data types
├── combat_visualizer/      # Battle replay visualizer
│   ├── main.go                 # Entry point
│   ├── visualizer.go           # Main visualizer logic
│   ├── loader.go               # Log file loading
│   ├── grid_renderer.go        # Combat grid rendering
│   ├── summary_renderer.go     # Battle summary rendering
│   └── types.go                # Type definitions
├── report_compressor/      # Report compression utility
│   ├── main.go                 # Entry point
│   ├── aggregator.go           # Report aggregation
│   ├── loader.go               # Loading reports
│   ├── writer.go               # Writing compressed reports
│   └── types.go                # Data types
├── benchmark_compare/      # Performance benchmark comparison
└── scripts/                # Utility scripts
```

### visual/ -- Graphics and Rendering

```
visual/
├── graphics/               # Graphics primitives
│   ├── graphictypes.go         # Type definitions
│   ├── vx.go                   # Vertex system
│   ├── vxfactory.go            # Vertex factory
│   ├── vxhandler.go            # Vertex handling
│   ├── drawableshapes.go       # Shape drawing utilities
│   ├── renderers.go            # Renderer implementations
│   ├── animators.go            # Animation system
│   └── colormatrix.go          # Color transformation matrices
└── rendering/              # Game rendering
    ├── rendering.go            # Main rendering orchestrator
    ├── maprendering.go         # Map/dungeon rendering
    ├── tilerenderer.go         # Individual tile rendering
    ├── squad_renderer.go       # Squad and unit rendering
    ├── quadbatch.go            # Batch quad rendering optimization
    ├── renderdata.go           # Render data structures
    ├── renderingcache.go       # Rendering cache
    ├── viewport.go             # Camera viewport
    ├── combatoverlays.go       # Combat UI overlays and effects
    └── squadhighlights.go      # Squad highlight visualization
```

### world/ -- World Data and Coordinates

```
world/
├── coords/
│   ├── cordmanager.go          # CRITICAL: Coordinate conversion, tile indexing
│   └── position.go             # LogicalPosition type
└── worldmap/
    ├── dungeongen.go           # Main dungeon generator
    ├── generator.go            # Generator registry and interface
    ├── dungeontile.go          # Tile definitions and properties
    ├── biome.go                # Biome definitions
    ├── tileconfig.go           # Tile configuration
    ├── GameMapUtil.go          # General map utilities
    ├── gen_helpers.go          # Generation helper functions
    ├── gen_cavern.go           # Cavern generation algorithm
    ├── gen_rooms_corridors.go  # Room and corridor generation
    ├── gen_military_base.go    # Military base layout
    ├── gen_garrison.go         # Garrison layout generation
    ├── gen_garrison_meta.go    # Garrison metadata
    ├── gen_garrison_dag.go     # Garrison DAG structure
    ├── gen_garrison_terrain.go # Garrison terrain generation
    ├── gen_overworld.go        # Overworld map generation
    └── garrison/               # Garrison management subdirectory
```

### docs/ -- Documentation

```
docs/
├── project_documentation/      # Main documentation
│   ├── ECS_BEST_PRACTICES.md       # ECS design patterns and conventions
│   ├── ARCHITECTURE_LAYERS.md      # Overall architecture
│   ├── COMBAT_PIPELINES.md         # Combat system architecture
│   ├── CACHING_OVERVIEW.md         # Caching strategies
│   ├── DATA_FLOW_PATTERNS.md       # Data flow architecture
│   ├── ENTITY_REFERENCE.md         # Entity type reference
│   ├── GAMEDATA_OVERVIEW.md        # Game data format reference
│   ├── PERFORMANCE_GUIDE.md        # Performance optimization
│   ├── AI/                         # AI system documentation
│   ├── Architecture/               # Architecture documents
│   ├── Process/                    # Development workflows
│   ├── Systems/                    # Individual system docs
│   └── UI/                         # GUI system documentation
├── notes/                      # Development notes
└── To_Review/                  # Pending documentation
    ├── Review_First/               # Priority items
    └── Features_To_Add/            # Planned features
```

### assets/ -- Game Assets

```
assets/
├── creatures/              # Creature sprites
├── effects/                # Visual effect sprites
├── items/                  # Item/artifact sprites
├── tiles/
│   ├── floors/             # Floor textures
│   │   └── desert/, forest/, grassland/, limestone/, mountain/, swamp/
│   ├── walls/              # Wall textures
│   │   └── desert/, forest/, grassland/, marble/, mountain/, swamp/
│   ├── decorations/        # Decoration sprites
│   └── maptiles/           # Location tiles
│       └── guild_hall/, temple/, town/, watchtower/
├── guiassets/              # GUI graphics
│   ├── buttons/            # Button sprites
│   ├── panels/             # Panel backgrounds
│   └── gui_png_sheets/     # Sprite sheets
├── fonts/                  # Game fonts (includes embedded mplus1pregular.go)
└── gamedata/               # JSON configs and balance data
```
