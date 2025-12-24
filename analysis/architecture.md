# Package Architecture Analysis: game_main
Generated: 2025-12-24
Scope: Package game_main

---

## Executive Summary

### Overall Assessment
The game_main package has clear separation of concerns across files but suffers from coupling issues in component registration and verbose UI setup boilerplate. The initialization flow is well-structured but rigid, making testing and extension difficult.

### High-Value Refactoring Targets
1. **Component Registration** - Creates coupling between game_main and all subsystems - Subsystems should self-register
2. **UI Mode Registration** - 80+ lines of repetitive boilerplate (11 modes x 7 lines each) - Extract to registry pattern
3. **Game Struct Responsibilities** - Mixes Ebiten interface with initialization state - Extract bootstrap orchestrator
4. **Initialization Dependencies** - Hard-coded 10-step sequence - Make phases explicit and testable

### What's Working Well
- Clear file separation (main.go, gamesetup.go, gameinit.go, componentinit.go)
- Proper delegation to subsystems for their own logic
- Good use of global singletons (CoordManager, PositionSystem)
- Bootstrap vs runtime mostly separated (just Game struct issue)

---

## Package Analysis

### game_main/main.go

**Purpose**: Ebiten integration, game loop, entry point

**Cohesion**: Good
- Focused on Ebiten interface (Update, Draw, Layout)
- Delegates to systems for actual logic
- main() orchestrates initialization sequence

**Issues Identified**:

- **Mixed Responsibilities**: Game struct holds both runtime (gameModeCoordinator, inputCoordinator) and bootstrap state (gameMap, playerData)
  - **Impact**: Game struct is larger than needed, carries initialization baggage throughout runtime
  - **Recommendation**: Extract initialization orchestration to separate struct
  - **Signature Change**:
    ```go
    // Before
    type Game struct {
        em                  common.EntityManager
        gameModeCoordinator *core.GameModeCoordinator
        playerData          common.PlayerData    // Only needed during init
        gameMap             worldmap.GameMap     // Only needed during init
        inputCoordinator    *input.InputCoordinator
        renderingCache      *rendering.RenderingCache
    }

    // After
    type Game struct {
        em                  *common.EntityManager // Now pointer for consistency
        gameModeCoordinator *core.GameModeCoordinator
        inputCoordinator    *input.InputCoordinator
        renderingCache      *rendering.RenderingCache
        // playerData removed - accessed via ECS query when needed
        // gameMap removed - accessed via gameModeCoordinator.UIContext when needed
    }

    // New bootstrap orchestrator
    type GameBootstrap struct {
        em         *common.EntityManager
        playerData *common.PlayerData
        gameMap    *worldmap.GameMap
    }
    ```

- **HandleInput Function**: Top-level function instead of method
  - **Impact**: Minor - breaks method encapsulation pattern
  - **Recommendation**: Make it `(g *Game) handleInput()` for consistency
  - **Signature Change**:
    ```go
    // Before
    func HandleInput(g *Game)

    // After
    func (g *Game) handleInput()  // private, called from Update()
    ```

**Dependencies**:
- Imports: common, config, coords, graphics, gui/core, input, rendering, testing, worldmap, ebiten
- Imported by: None (main package)
- Issues: testing package imported at main level (should be build-tag guarded or moved to setup)

---

### game_main/gamesetup.go

**Purpose**: Initialization orchestration, UI setup, system configuration

**Cohesion**: Mixed
- SetupNewGame() orchestrates 10-step initialization (good)
- SetupUI() is 80+ lines of repetitive mode registration (poor cohesion)
- SetupInputCoordinator() is trivial wrapper

**Issues Identified**:

- **Verbose UI Mode Registration**: 11 modes with identical pattern (create mode, call RegisterMode, check error, log fatal)
  - **Impact**: High visual noise (80 lines), error-prone to add new modes, hard to see registration structure at a glance
  - **Recommendation**: Extract to registry pattern with mode constructors slice
  - **Signature Change**:
    ```go
    // Before
    func SetupUI(g *Game) {
        // ... 140 lines of setup, including 80 lines of repetitive mode registration
        explorationMode := guimodes.NewExplorationMode(battleMapManager)
        if err := g.gameModeCoordinator.RegisterBattleMapMode(explorationMode); err != nil {
            log.Fatalf("Failed to register exploration mode: %v", err)
        }
        // ... repeat 10 more times
    }

    // After
    func SetupUI(g *Game) {
        // ... 60 lines of context setup (same)

        // Compact registration (10 lines total instead of 80)
        registerBattleMapModes(g.gameModeCoordinator, battleMapManager)
        registerOverworldModes(g.gameModeCoordinator, overworldManager)

        if err := g.gameModeCoordinator.EnterBattleMap("exploration"); err != nil {
            log.Fatalf("Failed to set initial mode: %v", err)
        }
    }

    // New helper functions (in gamesetup.go)
    func registerBattleMapModes(coordinator *core.GameModeCoordinator, manager *core.ModeManager) {
        modes := []core.UIMode{
            guimodes.NewExplorationMode(manager),
            guimodes.NewInfoMode(manager),
            guicombat.NewCombatMode(manager),
            guicombat.NewCombatAnimationMode(manager),
            guisquads.NewSquadDeploymentMode(manager),
            newInventoryModeWithReturn(manager, "exploration"),
        }

        for _, mode := range modes {
            if err := coordinator.RegisterBattleMapMode(mode); err != nil {
                log.Fatalf("Failed to register battle map mode '%s': %v", mode.Name(), err)
            }
        }
    }

    func registerOverworldModes(coordinator *core.GameModeCoordinator, manager *core.ModeManager) {
        modes := []core.UIMode{
            guisquads.NewSquadManagementMode(manager),
            guisquads.NewFormationEditorMode(manager),
            guisquads.NewSquadBuilderMode(manager),
            guisquads.NewUnitPurchaseMode(manager),
            guisquads.NewSquadEditorMode(manager),
            newInventoryModeWithReturn(manager, "squad_management"),
        }

        for _, mode := range modes {
            if err := coordinator.RegisterOverworldMode(mode); err != nil {
                log.Fatalf("Failed to register overworld mode '%s': %v", mode.Name(), err)
            }
        }
    }

    func newInventoryModeWithReturn(manager *core.ModeManager, returnMode string) core.UIMode {
        mode := guimodes.NewInventoryMode(manager)
        mode.SetReturnMode(returnMode)
        return mode
    }
    ```

- **10-Step Sequential Initialization**: Hard-coded order in SetupNewGame()
  - **Impact**: Difficult to test subsystems in isolation, order dependencies implicit
  - **Recommendation**: Extract to InitializationPipeline with explicit phase dependencies
  - **Signature Change**:
    ```go
    // Before
    func SetupNewGame(g *Game) {
        // Step 1
        templates.ReadGameData()
        // Step 2
        g.gameMap = worldmap.NewGameMap("overworld")
        InitializeECS(&g.em)
        // Step 2a
        common.GlobalPositionSystem = common.NewPositionSystem(g.em.World)
        // ... 7 more steps
    }

    // After - phases are explicit, testable, and self-documenting
    func SetupNewGame(g *Game) {
        bootstrap := NewGameBootstrap()

        bootstrap.LoadGameData()           // Phase 1: Static data
        bootstrap.InitializeCoreECS(&g.em) // Phase 2: ECS world + global systems
        bootstrap.CreateWorld(&g.gameMap)  // Phase 3: Map generation
        bootstrap.CreatePlayer(&g.playerData, &g.gameMap) // Phase 4: Player entity

        if config.DEBUG_MODE {
            bootstrap.SetupDebugContent(&g.em, &g.gameMap, &g.playerData)
        }

        bootstrap.InitializeGameplay(&g.em, &g.playerData) // Phase 5: Squads, factions

        g.renderingCache = rendering.NewRenderingCache(&g.em)
    }

    // GameBootstrap encapsulates initialization logic
    type GameBootstrap struct{}

    func (gb *GameBootstrap) LoadGameData() {
        templates.ReadGameData()
    }

    func (gb *GameBootstrap) InitializeCoreECS(em *common.EntityManager) {
        InitializeECS(em)
        common.GlobalPositionSystem = common.NewPositionSystem(em.World)
        graphics.ScreenInfo.ScaleFactor = 1
        if coords.MAP_SCROLLING_ENABLED {
            graphics.ScreenInfo.ScaleFactor = 3
        }
    }

    func (gb *GameBootstrap) CreateWorld(gm *worldmap.GameMap) {
        *gm = worldmap.NewGameMap("overworld")
    }

    func (gb *GameBootstrap) CreatePlayer(pd *common.PlayerData, gm *worldmap.GameMap) {
        // Encapsulates InitializePlayerData, AddCreaturesToTracker
    }

    func (gb *GameBootstrap) SetupDebugContent(em *common.EntityManager, gm *worldmap.GameMap, pd *common.PlayerData) {
        SetupTestData(em, gm, pd)
        testing.UpdateContentsForTest(em, gm)
    }

    func (gb *GameBootstrap) InitializeGameplay(em *common.EntityManager, pd *common.PlayerData) {
        SetupSquadSystem(em)
        SetupGameplayFactions(em, pd)
    }
    ```

- **SetupBenchmarking Global Side Effect**: Modifies global runtime settings
  - **Impact**: Can't disable profiling after startup, global state mutation
  - **Recommendation**: Keep as-is - this is appropriate for debug/profiling config

**Dependencies**:
- Imports: common, config, squads, graphics, coords, gui/*, input, templates, combat, rendering, worldmap
- Imported by: main.go
- Issues: Imports 11 GUI packages just for mode constructors (unavoidable given current design)

---

### game_main/gameinit.go

**Purpose**: Player entity initialization, creature tracking

**Cohesion**: Good
- Focused on player-specific setup
- Creature tracker registration is simple utility

**Issues Identified**:

- **Hardcoded Player Position**: Position (40, 45) set in component creation, then immediately overwritten with gameMap.StartingPosition()
  - **Impact**: Confusing code, wasted initialization
  - **Recommendation**: Remove hardcoded position, initialize with gameMap.StartingPosition() directly
  - **Signature Change**:
    ```go
    // Before
    func InitializePlayerData(ecsmanager *common.EntityManager, pl *common.PlayerData, gm *worldmap.GameMap) {
        // ...
        .AddComponent(common.PositionComponent, &coords.LogicalPosition{
            X: 40,  // Hardcoded, then overwritten below
            Y: 45,
        })
        // ...
        startPos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent)
        startPos.X = gm.StartingPosition().X
        startPos.Y = gm.StartingPosition().Y
    }

    // After
    func InitializePlayerData(ecsmanager *common.EntityManager, pl *common.PlayerData, gm *worldmap.GameMap) {
        startPos := gm.StartingPosition()
        // ...
        .AddComponent(common.PositionComponent, &coords.LogicalPosition{
            X: startPos.X,
            Y: startPos.Y,
        })
        // No need to re-assign
        pl.Pos = startPos
    }
    ```

- **AddCreaturesToTracker Function Name**: Doesn't reflect that it only adds to PositionSystem (not a general "tracker")
  - **Impact**: Minor - name is slightly misleading
  - **Recommendation**: Rename to `RegisterCreaturesInPositionSystem` or merge into InitializePlayerData
  - **Signature Change**:
    ```go
    // Before
    func AddCreaturesToTracker(ecsmanger *common.EntityManager)

    // After
    func RegisterCreaturesInPositionSystem(em *common.EntityManager)
    ```

**Dependencies**:
- Imports: common, config, gear, squads, rendering, coords, worldmap, ecs, ebitenutil
- Imported by: gamesetup.go
- Issues: None

---

### game_main/componentinit.go

**Purpose**: ECS component registration

**Cohesion**: Poor
- Registers components for 5+ subsystems (common, rendering, gear, squads, combat)
- Violates dependency inversion - bootstrap knows about all subsystems

**Issues Identified**:

- **Centralized Component Registration**: game_main reaches into gear, squads, combat packages to register their components
  - **Impact**: Every new subsystem requires changes to game_main, tight coupling, violates Open/Closed Principle
  - **Recommendation**: Self-registration pattern via init() functions in each subsystem package
  - **Signature Change**:
    ```go
    // Before - game_main/componentinit.go
    func InitializeECS(ecsmanager *common.EntityManager) {
        tags := make(map[string]ecs.Tag)
        manager := ecs.NewManager()

        registerCoreComponents(manager)
        registerItemComponents(manager, tags)
        buildCoreTags(tags)

        ecsmanager.WorldTags = tags
        ecsmanager.World = manager

        registerSquadComponents(ecsmanager)   // Reaches into squads package
        registerCombatComponents(ecsmanager)  // Reaches into combat package
    }

    func registerItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {
        gear.InitializeItemComponents(manager, tags)  // Tight coupling
    }

    // After - game_main/componentinit.go (simplified)
    func InitializeECS(ecsmanager *common.EntityManager) {
        tags := make(map[string]ecs.Tag)
        manager := ecs.NewManager()

        // Only register game_main's own components (none currently)
        registerCoreComponents(manager)
        buildCoreTags(tags)

        ecsmanager.WorldTags = tags
        ecsmanager.World = manager

        // Subsystems self-register via their init() functions
        // (gear, squads, combat packages call RegisterSubsystem in their init())
    }

    // After - common/ecsutil.go (new registry)
    var subsystemRegistrars []func(*EntityManager)

    func RegisterSubsystem(registrar func(*EntityManager)) {
        subsystemRegistrars = append(subsystemRegistrars, registrar)
    }

    func InitializeSubsystems(em *EntityManager) {
        for _, registrar := range subsystemRegistrars {
            registrar(em)
        }
    }

    // After - gear/components.go
    func init() {
        common.RegisterSubsystem(func(em *common.EntityManager) {
            InitializeItemComponents(em.World, em.WorldTags)
        })
    }

    // After - squads/components.go
    func init() {
        common.RegisterSubsystem(func(em *common.EntityManager) {
            InitSquadComponents(em)
            InitSquadTags(em)
        })
    }

    // After - combat/components.go
    func init() {
        common.RegisterSubsystem(func(em *common.EntityManager) {
            InitCombatComponents(em)
            InitCombatTags(em)
        })
    }

    // Usage in game_main/gamesetup.go
    func SetupNewGame(g *Game) {
        templates.ReadGameData()

        InitializeECS(&g.em)
        common.InitializeSubsystems(&g.em) // Auto-registers all subsystems

        // ... rest of setup
    }
    ```

- **Separate Tag Building**: Tags built separately from component registration
  - **Impact**: Tag registration split from component registration, hard to see full picture
  - **Recommendation**: Keep as-is OR combine into single registration step per subsystem (part of self-registration pattern above)

**Dependencies**:
- Imports: common, gear, combat, squads, rendering, ecs
- Imported by: gamesetup.go
- Issues: High coupling - knows about internal component details of 5 subsystems

---

## Dependency Issues

### Circular Dependencies
None detected

### Inappropriate Dependencies

| From | To | Issue | Recommendation |
|------|-----|-------|----------------|
| game_main | gear | Component registration | Self-registration pattern |
| game_main | squads | Component registration | Self-registration pattern |
| game_main | combat | Component registration | Self-registration pattern |
| game_main | testing | Imports at main level | Move to debug build tag or conditional import |
| main.go | 11 GUI packages | Mode constructor imports | Unavoidable given current design, but mode registration pattern reduces visual noise |

---

## Game Architecture Assessment

### ECS Organization
**Assessment**: Good
- Component registration properly delegated to subsystems
- Global systems (PositionSystem) initialized early
- EntityManager properly passed, not used as global (except in initialization)

**Issue**: game_main orchestrates all registration instead of subsystems self-registering

### State Management
**Assessment**: Mixed
- Game state properly stored in ECS components
- UI state properly managed by GameModeCoordinator
- PlayerData is hybrid - contains both cached state (Pos) and input state

**Issue**: Game struct holds playerData and gameMap after initialization completes (no longer needed at runtime)

### Input Flow
**Assessment**: Good
- InputCoordinator handles all input
- Clear flow: Ebiten -> Game.Update() -> HandleInput() -> InputCoordinator
- No issues with input architecture

---

## Prioritized Recommendations

### High Priority (Significant Impact)


3. **Extract GameBootstrap Orchestrator**
   - Package(s): game_main/main.go, game_main/gamesetup.go
   - Change: Create GameBootstrap type to handle initialization, remove playerData/gameMap from Game struct
   - Why: Game struct carries less baggage, initialization phases are explicit and testable
   - **Estimated Effort**: 1-2 hours



### Low Priority (Nice to Have)

5. **Remove Hardcoded Player Position**
   - Package(s): game_main/gameinit.go
   - Change: Initialize position component directly with gameMap.StartingPosition()
   - Why: Removes confusing double-initialization
   - **Estimated Effort**: 5 minutes

6. **Rename AddCreaturesToTracker**
   - Package(s): game_main/gameinit.go
   - Change: Rename to RegisterCreaturesInPositionSystem
   - Why: More accurate function name
   - **Estimated Effort**: 2 minutes

7. **Conditional Testing Import**
   - Package(s): game_main/main.go
   - Change: Use build tags or conditional compilation for testing package
   - Why: Testing code doesn't ship in production builds
   - **Estimated Effort**: 15 minutes

---

## What NOT to Change

### Game Struct as Ebiten Interface
The Game struct implementing ebiten.Game interface (Update/Draw/Layout) should stay as-is. This is the correct Ebiten pattern.

### Global Singletons (CoordManager, PositionSystem)
These are appropriate globals for true singleton systems. Don't inject these via dependency injection - they're needed everywhere and DI would add complexity without benefit.

### SetupBenchmarking Modifying Globals
Profiling configuration via global runtime settings is appropriate for debug tooling. No need to make this more "pure" - it's debug infrastructure.

### Sequential Initialization in main()
The main() function's sequential setup (NewGame -> SetupUI -> SetupInputCoordinator) is clear and appropriate for a game entry point. Don't over-engineer this.

### Testing Functions in Production Code
While testing package is imported, the actual test creation (CreateTestItems, UpdateContentsForTest) is only called in DEBUG_MODE or during explicit development. This is pragmatic for rapid game development iteration.

---

## Implementation Notes

### Self-Registration Pattern Example

The self-registration pattern is common in plugin architectures and game engines. Example flow:

1. **Subsystem declares registration in init()** (runs before main())
2. **Common package maintains registry** (slice of registration functions)
3. **game_main calls InitializeSubsystems()** after creating EntityManager
4. **Registry executes all registrations** in order

Benefits:
- Subsystems are self-contained (components, tags, initialization in one package)
- Adding new subsystems doesn't require changing game_main
- No import cycles (common imports no subsystems, subsystems import common)
- Clear ownership (squads package owns squad components, not game_main)

Tradeoffs:
- Init order is less explicit (relies on Go's init() ordering)
- Debugging registration failures is slightly harder (need to check init() functions)
- Solution: Add logging in RegisterSubsystem to show registration order

### UI Mode Registration Pattern Example

The proposed pattern uses slice iteration to reduce boilerplate. This is safe because:
- Mode constructors don't have side effects
- Registration order doesn't matter (coordinator manages mode switching)
- Error handling is preserved (loop can still log.Fatalf)

Alternative (if you want even more DRY):
```go
type modeRegistration struct {
    constructor func(*core.ModeManager) core.UIMode
    context     string // "battle" or "overworld"
}

var modeRegistrations = []modeRegistration{
    {guimodes.NewExplorationMode, "battle"},
    {guimodes.NewInfoMode, "battle"},
    // ... etc
}

func registerAllModes(coordinator *core.GameModeCoordinator, battle, overworld *core.ModeManager) {
    for _, reg := range modeRegistrations {
        var mode core.UIMode
        if reg.context == "battle" {
            mode = reg.constructor(battle)
            coordinator.RegisterBattleMapMode(mode)
        } else {
            mode = reg.constructor(overworld)
            coordinator.RegisterOverworldMode(mode)
        }
    }
}
```

This is MORE abstract but LESS readable. Recommend the helper function approach instead.

---

END OF ANALYSIS
