# TinkerRogue Project Layout

**Last Updated:** 2026-04-22

Annotated package tree with file-level descriptions for key files. The project is organized into top-level domains: `core/` (ECS primitives), `tactical/` (in-battle systems), `campaign/` (overworld/raid strategic layer), `mind/` (AI and encounters), `world/` (map generation), `visual/` (rendering), `gui/` (UI modes and panels), plus supporting `templates/`, `setup/`, `testing/`, and `tools/` trees.

```
TinkerRogue/
├── core/                   # ECS primitives and global coordinate/position infrastructure
│   ├── common/             # Core ECS utilities, shared components
│   │   ├── ecsutil.go              # EntityManager, component access helpers (GetComponentType, GetComponentTypeByID)
│   │   ├── commoncomponents.go     # PositionComponent, AttributesComponent, NameComponent
│   │   ├── positionsystem.go       # GlobalPositionSystem (O(1) spatial grid, value-based map keys)
│   │   ├── playerdata.go           # Player state
│   │   ├── randnumgen.go           # Seeded RNG
│   │   └── resources.go            # Shared resource registration / subsystem init hooks
│   ├── config/             # Centralized game constants
│   │   └── config.go               # Map dimensions, debug flags, profiling toggles
│   └── coords/             # Coordinate management (CRITICAL - always use LogicalToIndex)
│       ├── cordmanager.go          # Global CoordManager singleton
│       └── position.go             # LogicalPosition, PixelPosition types
│
├── tactical/               # In-battle tactical gameplay systems
│   ├── squads/             # Squad system (REFERENCE ECS IMPLEMENTATION)
│   │   ├── squadcore/              # Squad components, queries, creation, caches, abilities
│   │   │   ├── squadcomponents.go  # Pure data components (SquadData, SquadMemberData, ...)
│   │   │   ├── squadqueries.go     # Query functions
│   │   │   ├── squadcreation.go    # Squad/unit instantiation
│   │   │   ├── squadmanager.go     # Manager helpers
│   │   │   ├── squadcache.go       # Squad info cache
│   │   │   ├── squadabilities.go   # Leader ability definitions
│   │   │   └── units.go            # Unit helpers
│   │   ├── squadservices/          # Higher-level orchestration
│   │   │   ├── squad_deployment_service.go
│   │   │   └── unit_purchase_service.go
│   │   ├── squadcommands/          # Command pattern (undo/redo for editor operations)
│   │   │   ├── command.go, command_executor.go, command_helpers.go
│   │   │   ├── add_unit_command.go, remove_unit_command.go, move_unit_command.go
│   │   │   ├── purchase_unit_command.go, change_leader_command.go
│   │   │   ├── move_squad_command.go, rename_squad_command.go, reorder_squads_command.go
│   │   ├── roster/                 # Squad + unit rosters (persistent player-owned collections)
│   │   │   ├── squadroster.go, unitroster.go
│   │   ├── unitdefs/               # Unit template data (enums, filters, templates)
│   │   │   ├── enums.go, filters.go, templates.go
│   │   └── unitprogression/        # Per-unit experience and leveling
│   │       ├── components.go, experience.go
│   │
│   ├── combat/             # Turn-based combat management
│   │   ├── combatcore/             # Combat loop, actions, movement, balance config
│   │   │   ├── turnmanager.go, combatactionsystem.go, combatmovementsystem.go
│   │   │   ├── combatabilities.go, combatprocessing.go, balanceconfig.go
│   │   ├── combatstate/            # Combat ECS components and query cache
│   │   │   ├── combatcomponents.go, combatqueries.go, combatqueriescache.go
│   │   │   └── combatfactionmanager.go
│   │   ├── combatmath/             # Damage math, cover, targeting
│   │   │   ├── combatcalculation.go, combatcover.go, combattargeting.go
│   │   ├── combatservices/         # Combat service layer, power dispatch, AI interfaces
│   │   │   ├── combat_service.go, combat_power_dispatch.go, ai_interfaces.go
│   │   ├── combattypes/            # Shared combat types, perk callback hooks
│   │   │   ├── combattypes.go, perk_callbacks.go
│   │   └── battlelog/              # Battle recording and export
│   │       ├── battle_recorder.go, battle_summary.go, battle_export.go, combatlogging.go
│   │
│   ├── commander/          # Commander entities (player-controlled squad leaders)
│   │   ├── components.go           # CommanderData, ActionState, Roster
│   │   ├── movement.go             # Overworld movement
│   │   ├── queries.go              # Commander lookups
│   │   ├── roster.go               # Squad roster management
│   │   ├── turnstate.go            # Overworld turn tracking
│   │   ├── starters.go             # Starting commander creation
│   │   ├── system.go               # Commander system logic
│   │   └── init.go                 # Subsystem registration
│   │
│   └── powers/             # Unified powers pipeline (spells, effects, artifacts, perks)
│       ├── powercore/              # Shared pipeline, context, logger
│       │   ├── pipeline.go, context.go, logger.go
│       ├── spells/                 # Spell casting system
│       │   ├── components.go, system.go, queries.go, init.go
│       ├── effects/                # Active effects (buffs/debuffs applied by spells/artifacts)
│       │   ├── components.go, system.go, queries.go, init.go
│       ├── artifacts/              # Artifact inventory, charges, behaviors
│       │   ├── components.go, system.go, queries.go, init.go
│       │   ├── artifactinventory.go, artifactcharges.go
│       │   ├── behavior.go, behaviors.go, dispatcher.go, registry.go
│       │   ├── context.go, pending_effects.go, balanceconfig.go
│       ├── perks/                  # Perk registry, behaviors, hooks, dispatcher
│       │   ├── components.go, system.go, queries.go, init.go
│       │   ├── behaviors.go, dispatcher.go, hooks.go, registry.go
│       │   ├── perkids.go, unithelpers.go, balanceconfig.go
│       └── progression/            # Commander-level progression library (perks/spells unlock pool)
│           ├── components.go, library.go, init.go
│
├── campaign/               # Strategic (out-of-battle) campaign layer
│   ├── overworld/          # Strategic overworld layer (tick-based)
│   │   ├── core/                   # Shared components, types, registry, walkability
│   │   │   ├── components.go, types.go, events.go, resources.go
│   │   │   ├── node_registry.go, walkability.go, config.go, utils.go, init.go
│   │   ├── tick/                   # Strategic turn clock (AdvanceTick)
│   │   │   └── tickmanager.go
│   │   ├── faction/                # NPC faction AI
│   │   │   ├── system.go, archetype.go, scoring.go
│   │   ├── threat/                 # Threat node growth
│   │   │   ├── system.go, queries.go
│   │   ├── influence/              # Influence radius system
│   │   │   ├── system.go, effects.go, queries.go
│   │   ├── node/                   # Unified node management (creation, lookups, validation)
│   │   │   ├── system.go, queries.go, validation.go
│   │   ├── garrison/               # Garrison defense / raid detection
│   │   │   ├── system.go, queries.go
│   │   ├── victory/                # Win/loss conditions
│   │   │   ├── system.go, queries.go
│   │   └── overworldlog/           # Tick event recording & export
│   │       ├── overworld_recorder.go, overworld_summary.go, overworld_export.go
│   │
│   └── raid/               # Raid encounters (multi-floor garrison assaults)
│       ├── components.go, init.go, config.go, starters.go
│       ├── alert.go                # Alert level / detection
│       ├── archetypes.go           # Raid archetypes
│       ├── assignment.go           # Squad-to-floor assignment
│       ├── deployment.go           # Deployment orchestration
│       ├── floorgraph.go           # Floor graph / DAG navigation
│       ├── garrison.go             # Garrison raid linkage
│       ├── queries.go, resolvers.go, rewards.go
│       ├── raidencounter.go, raidrunner.go
│       └── recovery.go             # Post-raid cleanup
│
├── mind/                   # AI, encounter generation, combat lifecycle
│   ├── ai/                 # Utility AI decision-making
│   │   ├── ai_controller.go        # AI turn controller
│   │   └── action_evaluator.go     # Action scoring
│   ├── behavior/           # Threat analysis and danger painting
│   │   ├── threat_layers.go, threat_painting.go, threat_composite.go
│   │   ├── threat_combat.go, threat_positional.go, threat_support.go
│   │   ├── threat_queries.go, threat_gridutils.go, threat_constants.go
│   │   └── dangerlevel.go
│   ├── evaluation/         # Power and role evaluation
│   │   ├── power.go, power_config.go, roles.go
│   ├── encounter/          # Encounter system (overworld → tactical bridge)
│   │   ├── encounter_service.go    # Core service
│   │   ├── encounter_trigger.go    # Trigger conditions
│   │   ├── encounter_setup.go      # Enemy squad generation
│   │   ├── resolvers.go, rewards.go, starters.go, types.go, validators.go
│   ├── combatlifecycle/    # Combat setup, cleanup, rewards, casualties
│   │   ├── contracts.go, pipeline.go, starter.go, enrollment.go
│   │   ├── casualties.go, cleanup.go, reward.go
│   └── spawning/           # Automatic squad creation and composition
│       ├── squadscreation.go, composition.go, types.go, util.go
│
├── world/                  # Map generation and tile/biome infrastructure
│   ├── worldmapcore/       # Tile types, biome definitions, generator interface
│   │   ├── dungeongen.go, dungeontile.go, tileconfig.go
│   │   ├── biome.go, generator.go, GameMapUtil.go
│   ├── worldgen/           # Map generator algorithms and registry
│   │   ├── registry.go             # Generator registry (self-register via init())
│   │   ├── gen_cavern.go           # Cellular automata cavern generator
│   │   ├── gen_rooms_corridors.go  # BSP rooms + corridors
│   │   ├── gen_overworld.go        # Overworld map generator
│   │   └── gen_helpers.go          # Shared generator helpers
│   └── garrisongen/        # Garrison-specific map generation (multi-floor DAG)
│       ├── generator.go, dag.go, terrain.go, meta.go
│
├── visual/                 # Rendering pipeline
│   ├── graphics/           # Graphics primitives (color matrices, shapes, types)
│   │   ├── colormatrix.go, drawableshapes.go, graphictypes.go
│   ├── rendering/          # Batch rendering, viewport, caching
│   │   ├── rendering.go, quadbatch.go, renderingcache.go, viewport.go
│   ├── maprender/          # Map and tile rendering
│   │   ├── maprendering.go, tilerenderer.go
│   ├── combatrender/       # Combat overlays (squads, highlights, cover)
│   │   ├── squad_renderer.go, combatoverlays.go, squadhighlights.go, types.go
│   └── vfx/                # Visual effects (animators, renderers, factories)
│       ├── vx.go, vxfactory.go, vxhandler.go, animators.go, renderers.go
│
├── gui/                    # User interface (mode-based architecture)
│   ├── framework/          # Core mode infrastructure
│   │   ├── uimode.go               # UIMode interface
│   │   ├── basemode.go             # Common mode infrastructure
│   │   ├── modemanager.go          # Mode lifecycle & transitions
│   │   ├── modebuilder.go          # Declarative mode configuration
│   │   ├── coordinator.go          # GameModeCoordinator (two-context system)
│   │   ├── contextstate.go         # TacticalState, OverworldState (UI-only state)
│   │   ├── panelregistry.go        # Global panel type registry
│   │   ├── guiqueries.go           # ECS query abstraction (entry point)
│   │   ├── guiqueries_units.go, guiqueries_overworld.go, guiqueries_rendering.go
│   │   ├── actionmap.go            # Semantic ActionMap input
│   │   ├── inputaction.go, inputbinding.go, defaultbindings.go
│   │   ├── submenu.go              # Submenu helper
│   │   ├── squadinfo_cache.go      # Cached squad display data
│   │   └── commandhistory.go       # Undo/redo system
│   ├── builders/           # UI construction helpers
│   │   ├── panels.go, panelspecs.go, layout.go
│   │   ├── dialogs.go, lists.go, widgets.go
│   ├── widgets/            # Widget wrappers & utilities
│   │   ├── cached_list.go          # Cached list (90% CPU reduction)
│   │   ├── cached_textarea.go      # Cached text area
│   │   └── textdisplay.go          # Text display widget
│   ├── specs/              # Layout specifications
│   │   └── layout.go               # Responsive layout configuration
│   ├── widgetresources/    # Shared UI resources
│   │   ├── guiresources.go         # Fonts, images, colors
│   │   └── cachedbackground.go     # Cached background rendering
│   │
│   ├── guicombat/          # Combat + combat animation modes
│   │   ├── combatmode.go, combat_animation_mode.go
│   │   ├── combat_panels_registry.go, combat_animation_panels_registry.go
│   │   ├── combat_input_handler.go, combat_action_handler.go, combat_turn_flow.go
│   │   ├── combatvisualization.go, threatvisualizer.go, combatdeps.go
│   ├── guioverworld/       # Overworld mode
│   │   ├── overworldmode.go, overworld_panels_registry.go
│   │   ├── overworld_input_handler.go, overworld_action_handler.go
│   │   ├── overworld_renderer.go, overworld_formatters.go, overworld_deps.go
│   ├── guiexploration/     # Exploration mode
│   │   ├── explorationmode.go, exploration_panels_registry.go
│   ├── guinodeplacement/   # Node placement mode (overworld editor)
│   │   ├── nodeplacementmode.go, nodeplacement_panels_registry.go, nodeplacement_renderer.go
│   ├── guisquads/          # Squad editor, purchase, deployment, artifact modes
│   │   ├── squadeditormode.go, squadeditor_panels_registry.go
│   │   ├── squadeditor_grid.go, squadeditor_roster.go, squadeditor_perks.go, squadeditor_refresh.go
│   │   ├── unitpurchasemode.go, unitpurchase_panels_registry.go
│   │   ├── squaddeploymentmode.go, squaddeployment_panels_registry.go
│   │   ├── artifactmode.go, artifact_panels_registry.go, artifact_refresh.go
│   │   ├── commanderselector.go, squadselector.go, squadlists.go, squadcomponents.go
│   ├── guiraid/            # Raid mode
│   │   ├── raidmode.go, raid_panels_registry.go, raidstate.go
│   │   ├── deploy_panel.go, floormap_panel.go, floormap_renderer.go, summary_panel.go
│   ├── guiprogression/     # Progression mode (commander perk/spell unlocking)
│   │   ├── progressionmode.go, progression_panels_registry.go
│   │   ├── progression_controller.go, progression_refresh.go
│   ├── guiartifacts/       # Artifact management panel
│   │   ├── artifact_panel.go, artifact_handler.go, artifact_deps.go
│   ├── guispells/          # Spell casting panel
│   │   ├── spell_panel.go, spell_handler.go, spell_deps.go
│   ├── guiinspect/         # Unit inspection in combat
│   │   └── inspect_panel.go
│   ├── guiunitview/        # Unit detail view mode
│   │   ├── unitviewmode.go, unitview_panels_registry.go
│   └── guistartmenu/       # Start menu (Overworld vs Roguelike)
│       └── startmenu.go
│
├── input/                  # Input handling
│   └── cameracontroller.go         # WASD movement, diagonals, map scroll toggle
│
├── templates/              # JSON-based entity/data factory
│   ├── registry.go                 # Template registry
│   ├── entity_factory.go           # Factory functions
│   ├── readdata.go                 # JSON loading
│   ├── artifactdefinitions.go      # Artifact JSON schema
│   ├── spelldefinitions.go, unitspelldefinitions.go
│   ├── difficulty.go               # Difficulty scaling
│   ├── gameconfig.go               # Game-wide config from JSON
│   ├── jsonschema.go, validation.go
│   └── namegen.go                  # Procedural name generation
│
├── setup/                  # Game configuration and save system
│   ├── gamesetup/          # Boot orchestration
│   │   ├── bootstrap.go            # Top-level bootstrap
│   │   ├── ecsinit.go              # ECS world + subsystem init
│   │   ├── moderegistry.go         # GUI mode registration
│   │   ├── mapgenconfig.go         # Map generator configuration
│   │   ├── playerinit.go           # Player seeding
│   │   ├── initial_commanders.go   # Driven by initialsetup.json
│   │   ├── initial_squads.go       # Driven by initialsetup.json
│   │   ├── initial_factions.go     # Driven by initialsetup.json
│   │   ├── helpers.go, savehelpers.go
│   └── savesystem/         # Chunk-based save/load
│       ├── savesystem.go, idmap.go
│       └── chunks/                 # Per-domain save chunks
│           ├── commander_chunk.go, squad_chunk.go, gear_chunk.go
│           ├── map_chunk.go, player_chunk.go, progression_chunk.go, raid_chunk.go
│           └── shared_types.go
│
├── testing/                # Test fixtures and bootstrapping
│   ├── testingdata.go              # Test item creation (CreateTestItems)
│   ├── fixtures.go                 # Test fixtures (NewTestEntityManager, InitTestActionManager)
│   └── bootstrap/                  # Debug-only entity seeding (artifacts only — used under DEBUG_MODE)
│       └── initial_artifacts.go    # SeedAllArtifacts, EquipPlayerActivatedArtifacts
│
├── tools/                  # Development/analysis tools (separate binaries)
│   ├── combat_balance/             # Combat balance analysis
│   ├── combat_simulator/           # Combat simulation suites
│   ├── combat_visualizer/          # Battle visualization
│   ├── benchmark_compare/          # Benchmark diffing
│   ├── report_compressor/          # Report compression
│   └── scripts/                    # Misc build/analysis scripts
│
└── game_main/              # Entry point
    ├── main.go                     # Game struct, loop, start menu
    └── setup.go                    # Mode-specific boot wiring
```

## Notes on Reorganization

- `common/`, `config/`, `coords/` now live under `core/`.
- `overworld/` and `raid/` are now under `campaign/` (strategic layer split from tactical).
- `tactical/spells/`, `tactical/effects/` moved under `tactical/powers/` alongside `artifacts/`, `perks/`, `progression/`, and the shared `powercore/` pipeline.
- `tactical/combat/` was split into domain subpackages (`combatcore`, `combatstate`, `combatmath`, `combatservices`, `combattypes`, `battlelog`).
- `tactical/squads/` was split into `squadcore`, `squadservices`, `squadcommands`, `roster`, `unitdefs`, `unitprogression`.
- `world/worldmap/` was split into `worldmapcore/` (types), `worldgen/` (generators), and `garrisongen/` (garrison-specific pipelines).
- `visual/` gained `maprender/`, `combatrender/`, and `vfx/` subpackages.
- `gui/` added `guiprogression/` (commander progression mode) and reorganized around per-mode `*_panels_registry.go` files driven by the shared `PanelRegistry`.
- `mind/` gained `combatlifecycle/` (setup/cleanup pipeline) and `spawning/` (auto squad creation) alongside existing `ai/`, `behavior/`, `evaluation/`, `encounter/`.
- `savesystem/` moved under `setup/` and is fully chunk-based.
- `gamesetup/` moved under `setup/`.
