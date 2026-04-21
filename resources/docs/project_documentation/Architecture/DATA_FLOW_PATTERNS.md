# Data Flow Patterns

**Last Updated:** 2026-04-21

Understanding how data flows through the system is critical for debugging and extending functionality.

---

## Player Action Flow

```
User Input (Keyboard/Mouse)
    ↓
┌─────────────────────────────────────┐
│ GameModeCoordinator.Update()        │
│   → Active UIModeManager.Update()   │
│     → Resolve ActionMap             │
│       (InputActions → ActionsActive)│
│     → UIMode.HandleInput(state)     │
│   (handles UI clicks, combat,       │
│    overworld commands, etc.)        │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│ HandleInput(g *Game) [main.go]      │
│   → Gets InputState from            │
│     TacticalManager                 │
│   → CameraController.HandleInput()  │
│     (WASD via ActionMap actions,    │
│      diagonals, map scroll toggle)  │
└──────────────┬──────────────────────┘
               ↓
ECS components modified
    ↓
System functions process changes
    ↓
Rendering reads ECS state
    ↓
Display updated to screen
```

### ActionMap Pattern

Each `UIMode` can implement `GetActionMap()` returning an `ActionMap` that maps
keyboard/mouse inputs to semantic `InputAction` values. The `UIModeManager` resolves
these each frame into `inputState.ActionsActive` before calling `HandleInput`.
The `CameraController` checks actions like `ActionCameraMoveUp` via `inputState.ActionActive()`.

Key files: `gui/framework/modemanager.go`, `gui/framework/actionmap.go`, `input/cameracontroller.go`

---

## Combat Action Flow

```
Player presses attack key or clicks attack button
    ↓
CombatActionHandler.ToggleAttackMode()
    → Sets BattleState.InAttackMode = true
    ↓
Player clicks target tile
    ↓
CombatInputHandler.HandleInput()
    → Dispatches to CombatActionHandler.SelectTarget(targetSquadID)
    ↓
CombatActionHandler.ExecuteAttack()
    ↓
CombatService.CombatActSystem.ExecuteAttackAction(attacker, defender)
    ↓
combatcore.CombatActionSystem.ExecuteAttackAction()
    ├─ Query attacker units
    ├─ Query defender units
    ├─ Calculate damage per unit
    ├─ Apply damage to components
    └─ Generate CombatResult
    ↓
ModeManager.RequestTransition("combat_animation")
    ↓
Combat animation plays (visual feedback)
    ↓
Animation complete callback
    ↓
Combat log updated
    ↓
Turn advancement check
```

Key files: `gui/guicombat/combat_action_handler.go`, `gui/guicombat/combatmode.go`,
`tactical/combat/combatcore/combatactionsystem.go`, `tactical/combat/combatservices/combat_service.go`

---

## Artifact System Flow

```
Player opens artifact UI (overworld: ArtifactMode, combat: ArtifactActivationHandler)
    ↓
┌─────────────────────────────────────────┐
│ EQUIP/UNEQUIP (overworld only)          │
│                                         │
│ gui/guisquads/artifactmode.go           │
│   → artifacts.EquipArtifact(playerID,   │
│       squadID, artifactID, manager)     │
│   → artifacts.UnequipArtifact(...)      │
│                                         │
│ Artifacts stored in:                    │
│   Player: ArtifactInventoryData         │
│     (ArtifactInventoryComponent)        │
│   Squad: EquipmentData                  │
│     (EquipmentComponent)                │
│     → EquippedArtifacts []string        │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ COMBAT START                            │
│                                         │
│ CombatService.InitializeCombat()        │
│   → artifacts.ApplyArtifactStatEffects( │
│       squadIDs, manager)                │
│   (applies passive stat bonuses as      │
│    permanent effects via effects pkg)   │
│   → ArtifactDispatcher already created  │
│     in NewCombatService; its            │
│     ChargeTracker is reset here         │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ ACTIVATED ARTIFACTS (during combat)     │
│                                         │
│ guiartifacts.ArtifactActivationHandler  │
│   → artifacts.ActivateArtifact(         │
│       behavior, targetSquadID, ctx)     │
│   → Dispatches to registered behavior's │
│     Activate() method                   │
│                                         │
│ ArtifactDispatcher lifecycle hooks      │
│ (wired on the shared PowerPipeline):    │
│   → DispatchPostReset(factionID, squads)│
│     (fires OnPostReset for all behaviors│
│      e.g. Deadlock Shackles lock,       │
│      Saboteur's Hourglass slow)         │
│   → DispatchOnAttackComplete(attacker,  │
│       defender, result)                 │
│     (fires OnAttackComplete for         │
│      equipped behaviors on attacker,    │
│      e.g. Engagement Chains)            │
│   → DispatchOnTurnEnd(round)            │
│     (fires OnTurnEnd + refreshes round  │
│      charges via ChargeTracker)         │
└─────────────────────────────────────────┘
```

### Artifact Behavior Types

- **Player-activated** (`IsPlayerActivated() = true`): Triggered via GUI (e.g., Chain of Command, Echo Drums, Twin Strike, Deadlock Shackles, Saboteur's Hourglass)
- **Passive/event-driven**: Fire on combat events only (e.g., Engagement Chains on kill)
- **Charge tracking**: `ArtifactChargeTracker` manages `ChargeOncePerBattle` and `ChargeOncePerRound` limits, plus pending effects for deferred behaviors

### Balance Configuration

Artifact tuning values loaded from `resources/assets/gamedata/artifactbalanceconfig.json` into `artifacts.ArtifactBalance`.

Key files: `tactical/powers/artifacts/system.go`, `tactical/powers/artifacts/queries.go`,
`tactical/powers/artifacts/artifactinventory.go`, `tactical/powers/artifacts/components.go`,
`tactical/powers/artifacts/behavior.go` (interface + constants),
`tactical/powers/artifacts/behaviors.go` (concrete behaviors),
`tactical/powers/artifacts/dispatcher.go`, `tactical/powers/artifacts/artifactcharges.go`,
`tactical/powers/artifacts/pending_effects.go`, `tactical/powers/artifacts/balanceconfig.go`,
`tactical/powers/artifacts/registry.go`, `tactical/powers/artifacts/context.go`,
`gui/guiartifacts/`, `gui/guisquads/artifactmode.go`

---

## Perk System Flow

```
┌─────────────────────────────────────────┐
│ EQUIP/UNEQUIP (overworld)               │
│                                         │
│ perks.EquipPerk(squadID, perkID,        │
│     maxSlots, manager)                  │
│ perks.UnequipPerk(squadID, perkID,      │
│     manager)                            │
│                                         │
│ Storage: PerkSlotData on squad entity   │
│   → PerkIDs []PerkID (max 3 slots)      │
│                                         │
│ Validation:                             │
│   → Slot capacity check                 │
│   → Duplicate check                     │
│   → Mutual exclusivity check            │
│     (PerkDefinition.ExclusiveWith)      │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ COMBAT START                            │
│                                         │
│ perks.InitializePerkRoundStatesForFaction│
│   (factionSquadIDs, manager)            │
│   → Creates PerkRoundState component    │
│     on each squad that has perks        │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ COMBAT — DAMAGE PIPELINE HOOKS          │
│                                         │
│ SquadPerkDispatcher (implements         │
│   combattypes.PerkDispatcher interface) │
│                                         │
│ Per-attack hook methods (called by      │
│ CombatActionSystem via the injected     │
│ PerkDispatcher interface):              │
│   → AttackerDamageMod(...modifiers)     │
│   → DefenderDamageMod(...modifiers)     │
│   → CoverMod(...coverBreakdown)         │
│   → TargetOverride(...targets)          │
│   → CounterMod(...modifiers) bool       │
│   → AttackerPostDamage(...dmg, kill)    │
│   → DefenderPostDamage(...dmg, kill)    │
│   → DeathOverride(unitID, squadID)      │
│   → DamageRedirect(...dmgAmount)        │
│                                         │
│ Each method iterates active perks on    │
│ the relevant squad and calls the        │
│ matching PerkBehavior hook.             │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ COMBAT — LIFECYCLE DISPATCH             │
│                                         │
│ SquadPerkDispatcher lifecycle methods   │
│ (wired as subscribers on the shared     │
│ powercore.PowerPipeline):               │
│                                         │
│   → DispatchTurnStart(squadIDs, round)  │
│     1. ResetPerkRoundStateTurn()        │
│        (snapshot previous turn state:   │
│         WasAttackedLastTurn,            │
│         DidNotAttackLastTurn,           │
│         WasIdleLastTurn)                │
│     2. Iterate perks, calling each      │
│        behavior's TurnStart hook        │
│                                         │
│   → DispatchRoundEnd(manager)           │
│     Clears per-round PerkState map      │
│     (per-battle PerkBattleState kept)   │
│                                         │
│   → DispatchAttackTracking(atk, def)    │
│     Sets AttackedThisTurn / WasAttacked │
│                                         │
│   → DispatchMoveTracking(squadID)       │
│     Sets MovedThisTurn, resets          │
│     TurnsStationary                     │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ COMBAT END                              │
│                                         │
│ perks.CleanupRoundState(squadID, mgr)   │
│   → Removes PerkRoundStateComponent     │
│     from each squad entity              │
└─────────────────────────────────────────┘
```

### Perk State Architecture

Perks use a two-level state system on `PerkRoundState`:

- **Shared tracking fields** (e.g., `MovedThisTurn`, `WasAttackedLastTurn`): Set by the dispatch layer, read by multiple perks. Reset each turn by `ResetPerkRoundStateTurn`.
- **Per-perk round state** (`PerkState map[PerkID]any`): Isolated state per perk (e.g., `RecklessAssaultState`, `BloodlustState`). Cleared each round by `ResetPerkRoundStateRound`. Accessed via generic helpers `GetPerkState[T]` / `SetPerkState`.
- **Per-perk battle state** (`PerkBattleState map[PerkID]any`): Persists the entire combat (e.g., `OpeningSalvoState`, `ResoluteState`, `GrudgeBearerState`). Only cleaned up at combat end. Accessed via `GetBattleState[T]` / `SetBattleState`.

### Perk Behavior Organization

All 21 perk behavior implementations live in a single file, `behaviors.go`, grouped by state requirement via section comments:

- **Stateless** (11 perks) — pure functions of `HookContext`, no state tracking.
- **Per-round stateful** (7 perks) — read shared tracking or use per-round `PerkState`.
- **Per-battle stateful** (3 perks) — use per-battle `PerkBattleState`.

Each section's `init()` registers its behaviors via `RegisterPerkBehavior`.

### Perk Definitions and Balance

- **Definitions**: Loaded from `resources/assets/gamedata/perkdata.json` into `perks.PerkRegistry`. Each perk has tier, category, roles, exclusivity rules, and unlock cost.
- **Balance values**: Loaded from `resources/assets/gamedata/perkbalanceconfig.json` into `perks.PerkBalance`. All numeric tuning values are data-driven.
- **Behavior registration**: Behaviors register via `init()` → `RegisterPerkBehavior()`. Startup validation (`validateHookCoverage`) ensures JSON definitions and behavior registrations are in sync.

### Perk Hook Ordering in Damage Pipeline

```
Attacker initiates attack
    ↓
TargetOverride (Cleave, Precision Strike)
    → Modify which units are targeted
    ↓
AttackerDamageMod (all attacker perks)
    → Modify outgoing damage multiplier, crit bonus, hit penalty
    ↓
DefenderDamageMod (all defender perks)
    → Modify incoming damage multiplier, skip crit
    ↓
CoverMod (Brace for Impact, Fortify)
    → Modify cover reduction
    ↓
DamageRedirect (Guardian Protocol)
    → Redirect portion of damage to another unit
    ↓
Damage applied to units
    ↓
DeathOverride (Resolute)
    → Prevent lethal damage (once per battle per unit)
    ↓
AttackerPostDamage / DefenderPostDamage
    → Track kills (Bloodlust), grudge stacks (Grudge Bearer)
    ↓
CounterMod (Riposte, Stalwart)
    → Modify or skip counterattack
```

Key files: `tactical/powers/perks/system.go`, `tactical/powers/perks/dispatcher.go`,
`tactical/powers/perks/hooks.go`, `tactical/powers/perks/components.go`,
`tactical/powers/perks/perkids.go`, `tactical/powers/perks/registry.go`,
`tactical/powers/perks/queries.go`, `tactical/powers/perks/behaviors.go`,
`tactical/powers/perks/balanceconfig.go`, `tactical/powers/perks/init.go`,
`tactical/combat/combattypes/perk_callbacks.go` (PerkDispatcher interface)

---

## Power Pipeline Flow (Artifacts + Perks Lifecycle)

```
CombatService holds powerPipeline *powercore.PowerPipeline
    ↓
NewCombatService registers subscribers in declared order:
    ├─ PostReset        : ArtifactDispatcher → PerkDispatcher TurnStart
    ├─ AttackComplete   : ArtifactDispatcher → perk attack tracking → GUI
    ├─ TurnEnd          : ArtifactDispatcher (refresh charges + OnTurnEnd)
    │                     → PerkDispatcher round reset → GUI
    └─ MoveComplete     : perk movement tracking → GUI
    ↓
Combat internals (CombatActionSystem, CombatMovementSystem, TurnManager)
  hold a single-subscriber callback each. Those callbacks are wired to the
  pipeline's Fire* methods in NewCombatService:
    ├─ combatActSystem.SetOnAttackComplete(pipeline.FireAttackComplete)
    ├─ movementSystem.SetOnMoveComplete(pipeline.FireMoveComplete)
    ├─ turnManager.SetOnTurnEnd(pipeline.FireTurnEnd)
    └─ turnManager.SetPostResetHook(pipeline.FirePostReset)
    ↓
When combat fires one of these events, the single-subscriber hook
  invokes the pipeline's Fire*, which invokes every registered
  handler in registration order.
```

Both `artifacts.BehaviorContext` and `perks.HookContext` embed `powercore.PowerContext`,
sharing the entity manager, query cache, round number, and a single `PowerLogger`.
The logger writes `[GEAR] <source>` for artifact messages and `[PERK] <source>` for perk
messages, distinguishing via `artifacts.IsRegisteredBehavior(source)`.

Key files: `tactical/powers/powercore/pipeline.go`, `tactical/powers/powercore/context.go`,
`tactical/powers/powercore/logger.go`, `tactical/combat/combatservices/combat_service.go`,
`tactical/combat/combatservices/combat_power_dispatch.go`

---

## Map Generation Flow

```
Game initialization or encounter setup
    ↓
worldmapcore.NewGameMap(generatorName)
    ↓
worldgen.GetGeneratorOrDefault(name)
    ├─ Check ConfigOverride (if set)
    └─ Fall back to generator registry
    ↓
Generator.Generate(width, height, images)
    ├─ Initialize tile array
    ├─ Algorithm-specific generation
    │   (rooms_corridors, cavern,
    │    strategic_overworld, garrison_raid)
    ├─ Place doors/features
    ├─ Collect valid positions
    └─ Populate GenerationResult:
        ├─ Tiles
        ├─ POIs (points of interest)
        ├─ FactionStartPositions
        ├─ BiomeMap
        └─ GarrisonData
    ↓
GameMap created from GenerationResult
    ↓
[Overworld only] bootstrap.InitializeGameplay()
    → ConvertPOIsToNodes()
    (converts POIs into overworld node entities)
    ↓
Spawn player at valid position
    → GlobalPositionSystem.AddEntity(playerID, startPos)
    ↓
Spawn entities (squads, threats, resources)
    ↓
Rendering displays map
```

Registered generators:
- `rooms_corridors` (`world/worldgen/gen_rooms_corridors.go`)
- `cavern` (`world/worldgen/gen_cavern.go`)
- `strategic_overworld` (`world/worldgen/gen_overworld.go`)
- `garrison_raid` (`world/garrisongen/generator.go`)

Key files: `world/worldmapcore/generator.go` (MapGenerator interface, GameMap),
`world/worldmapcore/dungeongen.go`, `world/worldmapcore/dungeontile.go`,
`world/worldgen/registry.go` (RegisterGenerator / GetGeneratorOrDefault / ConfigOverride),
`world/garrisongen/generator.go`, `world/garrisongen/dag.go`,
`setup/gamesetup/mapgenconfig.go`, `setup/gamesetup/bootstrap.go`

---

## Entity/Unit Creation Flow

```
Request entity creation
    ↓
┌──────────────────────────────────────────────┐
│ Single Entity (creature/monster):            │
│                                              │
│ templates.CreateEntityFromTemplate(          │
│     manager, EntityConfig, JSONMonster)      │
│   ├─ Create entity with components:          │
│   │   ├─ PositionComponent                   │
│   │   ├─ AttributeComponent (from template)  │
│   │   ├─ NameComponent                       │
│   │   └─ Relevant tags                       │
│   └─ GlobalPositionSystem.AddEntity()        │
│                                              │
│ OR squadcore.CreateUnit(mgr, name, attr, pos)│
│   (bare unit entity without template)        │
└──────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────┐
│ Full Squad (with units in 3x3 grid):         │
│                                              │
│ squadcore.CreateSquadFromTemplate(           │
│     manager, name, formation,                │
│     worldPos, unitTemplates)                 │
│   ├─ Create squad entity (SquadComponent)    │
│   ├─ For each UnitTemplate:                  │
│   │   ├─ Create unit entity                  │
│   │   ├─ Add SquadMemberComponent            │
│   │   └─ Place in formation grid             │
│   └─ Return squadID                          │
└──────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────┐
│ Enemy Squads (encounter generation):         │
│                                              │
│ mind/spawning/squadscreation.go              │
│   → createSquadForPowerBudget()              │
│   → Uses squadcore.Units (from JSON          │
│     templates) + spawning.Composition        │
│   → squadcore.CreateSquadFromTemplate()      │
└──────────────────────────────────────────────┘
```

Templates loaded by `templates.ReadGameData()` from JSON files. Unit templates
initialized via `squadcore.InitUnitTemplatesFromJSON()`.

Key files: `templates/entity_factory.go`, `templates/registry.go`, `templates/readdata.go`,
`tactical/squads/squadcore/squadcreation.go`, `tactical/squads/squadcore/units.go`,
`mind/spawning/squadscreation.go`, `mind/spawning/composition.go`,
`mind/spawning/types.go`, `mind/spawning/util.go`,
`mind/encounter/encounter_setup.go`

---

## Game Initialization Flow

```
1. main() starts
   ↓
2. SetupSharedSystems() (game_main/setup.go)
   ├─ LoadGameData() → JSON templates
   ├─ InitializeCoreECS() → ECS manager, 50+ components, GlobalPositionSystem
   └─ Configure graphics
   ↓
3. Show StartMenu (Overworld vs Roguelike selection)
   ↓
4. Mode-specific setup
   ├─ SetupOverworldMode()
   │   OR SetupRoguelikeMode()
   │   OR SetupRoguelikeFromSave() (load saved game)
   │
   ├─ CreateWorld() → Generate map (overworld or cavern)
   ├─ CreatePlayer() → Player entity, initial commander, starting squads
   ├─ Set SelectedCommanderID on state
   ├─ SetupDebugContent() → Test data (if DEBUG_MODE)
   │
   ├─ [Overworld only] InitializeGameplay()
   │   ├─ tick.CreateTickStateEntity()
   │   ├─ commander.CreateOverworldTurnState()
   │   ├─ InitWalkableGrid()
   │   ├─ bootstrap.InitializeOverworldFactions()
   │   └─ ConvertPOIsToNodes()
   │
   ├─ [Roguelike only] Create RaidRunner → inject into RaidMode
   │
   ├─ setupUICore() → GameModeCoordinator + EncounterService
   ├─ Register modes via moderegistry.go:
   │   ├─ RegisterTacticalModes()
   │   ├─ RegisterOverworldModes() [overworld only]
   │   └─ RegisterRoguelikeTacticalModes() [roguelike only]
   └─ SetupInputCoordinator() → CameraController
   ↓
5. coordinator.EnterTactical("exploration")
   ↓
6. Run Game Loop (ebiten.RunGame)
   ├─ Update() @ 60 FPS
   └─ Draw() @ 60 FPS
```

### Registered UI Modes

- **Tactical (shared):** `exploration`, `combat`, `combat_animation`, `squad_deployment`
- **Overworld-only:** `overworld`, `node_placement`, `unit_purchase`, `squad_editor`, `artifact`, `progression`, `unit_view`
- **Roguelike-only:** `raid` (plus the shared tactical modes and `progression`)

Key files: `game_main/setup.go`, `setup/gamesetup/bootstrap.go`, `setup/gamesetup/moderegistry.go`

---

## Game Loop Flow

```
┌─────────────────────────────────────────────────┐
│          Update() - 60 FPS                      │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼──────────────┐     ┌─────▼──────────┐
│ GameModeCoordinator  │     │ Visual Effects │
│ .Update()            │     │ Update         │
│                      │     └─────┬──────────┘
│ Active UIModeManager │           │
│   → Resolve ActionMap│     ┌─────▼──────────┐
│   → UIMode.Update()  │     │ CameraController│
│   (UI + game input)  │     │ .HandleInput() │
└───────┬──────────────┘     │ (WASD movement)│
        │                    └─────┬──────────┘
        └──────────┬───────────────┘
                   │
           ECS state updated
                   │
                   ▼

┌─────────────────────────────────────────────────┐
│          Draw() - 60 FPS                        │
└─────────────────────┬───────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼────────┐          ┌──────▼─────────┐
│ Map Rendering  │          │ Entity         │
│ (maprender)    │          │ Rendering      │
│ [Tactical only]│          │ (combatrender) │
│                │          │ [Tactical only]│
└───────┬────────┘          └──────┬─────────┘
        │                          │
        └──────────┬───────────────┘
                   │
           ┌───────▼────────┐
           │ Visual Effects │
           │ Rendering (vfx)│
           └────────┬───────┘
                    │
           ┌────────▼─────────────┐
           │ GameModeCoordinator  │
           │ .Render()            │
           │ (EbitenUI overlay)   │
           └──────────────────────┘
```

Key files: `game_main/main.go`, `visual/graphics/`, `visual/rendering/`,
`visual/maprender/`, `visual/combatrender/`, `visual/vfx/`

---

## Overworld Tick Flow

```
Player clicks "End Turn" (OverworldMode)
    ↓
OverworldActionHandler.EndTurn()
    → commander.EndTurn(manager)
        (resets commander action states)
    ↓
tick.AdvanceTick(manager)
    ├─ Increment TickStateData.CurrentTick
    ├─ influence.UpdateInfluenceInteractions(manager, tick)
    │   (synergy/competition/suppression between factions)
    ├─ threat.UpdateThreatNodes(manager, tick)
    │   (grow threat nodes by influence)
    └─ faction.UpdateFactions(manager, tick)
        (evaluate intents, execute actions)
        → Returns *PendingRaid (if a faction raids a garrisoned node)
    ↓
TickResult (events, pending raid, flags)
    ↓
GUI refreshes overworld panels
    ↓
If pending raid → trigger garrison defense encounter
```

Key files: `campaign/overworld/tick/tickmanager.go`, `campaign/overworld/influence/`,
`campaign/overworld/threat/`, `campaign/overworld/faction/`,
`gui/guioverworld/overworld_action_handler.go`

---

## Encounter Flow

```
Commander approaches threat node (overworld movement)
    ↓
OverworldActionHandler.EngageThreat(nodeID)
    ↓
encounter.TriggerCombatFromThreat(manager, threatEntity)
    → Creates encounter entity with OverworldEncounterData
    → Returns encounterID
    ↓
combatlifecycle.ExecuteCombatStart(encounterService, manager, &OverworldCombatStarter{})
    ↓
OverworldCombatStarter.Prepare(manager)
    → encounter.SpawnCombatEntities(...)
        ├─ Generate enemy squads from power budget
        │   (via mind/spawning/squadscreation.go)
        ├─ Create faction entities
        └─ Assign squads to factions
    ↓
EncounterService.TransitionToCombat(setup)
    → Save OriginalPlayerPosition
    → Move camera to CombatPosition
    → modeCoordinator.EnterCombatMode()
    (enters "combat" mode directly)
    ↓
CombatMode (turn-based tactical combat)
    ↓
Combat ends (victory/defeat/retreat)
    ↓
CombatMode.Exit()
    → encounterService.ExitCombat(reason, result, combatService)
    ↓
Inside ExitCombat:
    → combatlifecycle.ExecuteResolution(manager, &OverworldCombatResolver{})
        ├─ Grant rewards (XP, artifacts)
        └─ Mark threat node as defeated (or retreat)
    → RecordEncounterCompletion()
        (restores player position)
    → combatService.CleanupCombat(enemySquadIDs) → returns player squad IDs
    → PostCombatCallback (if set, e.g., RaidRunner)
    ↓
Returns to "exploration" mode (or PostCombatReturnMode)
```

Key files: `gui/guioverworld/overworld_action_handler.go`, `mind/encounter/encounter_trigger.go`,
`mind/encounter/encounter_setup.go`, `mind/encounter/encounter_service.go`,
`mind/encounter/resolvers.go`,
`mind/combatlifecycle/starter.go`, `mind/combatlifecycle/pipeline.go`,
`mind/combatlifecycle/contracts.go`, `mind/combatlifecycle/cleanup.go`

---

## Combat Pipeline Flow

```
Trigger (GUI button, tick event, or debug menu)
    ↓
Construct type-specific CombatStarter
    ↓
combatlifecycle.ExecuteCombatStart()           ← single shared entry point
    ├─ starter.Prepare(manager)
    │   ├─ CreateFactionPair() (player + enemy)
    │   ├─ EnrollSquadsAtPositions()
    │   └─ Returns CombatSetup (factions, positions, type)
    │
    └─ EncounterService.TransitionToCombat(setup)
        ├─ Save player's OriginalPlayerPosition
        ├─ Move camera to CombatPosition
        └─ GameModeCoordinator.EnterCombatMode()
            ↓
CombatMode.Enter()
    ├─ registerCombatCallbacks()
    ├─ CombatService.InitializeCombat(factionIDs)
    └─ TurnManager: randomize turn order, CombatActive = true
            ↓
Turn-based combat loop
    (player/AI actions, spells, artifacts per turn)
            ↓
Victory / Defeat / Flee detected
            ↓
CombatMode.Exit()
    → EncounterService.ExitCombat(reason, outcome, combatService)
        ├─ resolveEncounterOutcome()
        │   (dispatches to type-specific CombatResolver)
        ├─ RecordEncounterCompletion()
        │   (restore player position, clear ActiveEncounter)
        ├─ CombatService.CleanupCombat(enemySquadIDs)
        │   ├─ Clear callbacks + effects
        │   ├─ Collect player squad IDs (returned to caller)
        │   └─ Dispose enemy squads, factions, action state,
        │     turn state entities
        ├─ StripCombatComponents(playerSquadIDs)
        │   (strip player squads: position, faction membership,
        │    IsDeployed=false — squads NOT disposed)
        └─ postCombatCallback (e.g. RaidRunner)
            ↓
Return to previous mode (exploration / raid / overworld)
```

### Entry Pathways

Five triggers all funnel into `ExecuteCombatStart`:

1. **Overworld threat** — player engages a threat node
2. **Garrison defense** — NPC faction raids a player-garrisoned node
3. **Raid room** — player selects a combat room in raid mode
4. **Debug raid** — debug menu starts a raid (roguelike only)
5. **Debug random encounter** — debug menu spawns a random fight (overworld only)

Each pathway constructs a type-specific `CombatStarter` whose `Prepare()` method handles faction creation and squad positioning. After that, the shared pipeline takes over.

### Exit Pathways

Three exit reasons all funnel into `ExitCombat`:

- **Victory** — all enemy squads destroyed
- **Defeat** — all player squads destroyed
- **Flee** — player clicks retreat

`ExitCombat` dispatches to a type-specific `CombatResolver` based on `CombatType` (overworld, garrison defense, raid, debug). Raid resolution is handled separately via the post-combat callback registered by `RaidRunner`.

*For the full combat lifecycle including all 5 entry pathways, type-specific resolution, cleanup ordering, and edge cases, see [COMBAT_PIPELINES.md](COMBAT_PIPELINES.md).*

Key files: `mind/combatlifecycle/starter.go`, `mind/combatlifecycle/pipeline.go`,
`mind/combatlifecycle/contracts.go`, `mind/combatlifecycle/enrollment.go`,
`mind/combatlifecycle/cleanup.go`, `mind/combatlifecycle/casualties.go`,
`mind/combatlifecycle/reward.go`,
`mind/encounter/encounter_service.go`, `mind/encounter/resolvers.go`,
`tactical/combat/combatservices/combat_service.go`,
`tactical/combat/combatstate/combatfactionmanager.go`,
`gui/guicombat/combatmode.go`

---

## Spell Casting Flow

```
Player opens spell panel in CombatMode
    → Selects spell from list
    → Selects target(s) or AoE area
    → Clicks cast
    ↓
guispells.SpellCastingHandler.CastSpell(spellID, targetSquadIDs)
    ↓
spells.ExecuteSpellCast(casterEntityID, spellID, targetSquadIDs, manager)
    ├─ templates.GetSpellDefinition(spellID)
    ├─ spells.GetManaData(casterEntityID, manager)
    │   → Validate sufficient mana
    │   → mana.CurrentMana -= spell.ManaCost
    └─ Switch on spell.EffectType:
        │
        ├─ EffectDamage → applyDamageSpell()
        │   ├─ squadcore.GetUnitIDsInSquad() per target
        │   ├─ attr.CurrentHealth -= damage per unit
        │   └─ if squad destroyed: combat.RemoveSquadFromMap()
        │
        └─ EffectBuff/EffectDebuff → applyBuffDebuffSpell()
            ├─ effects.ParseStatType(mod.Stat)
            └─ effects.ApplyEffectToUnits(unitIDs, ActiveEffect, manager)
                ├─ applyModifierToStat() (immediate stat change)
                └─ Append to ActiveEffectsData.Effects
                    (with RemainingTurns for expiry tracking)
    ↓
Returns SpellCastResult
    ↓
GUI updates (mana bar, health bars, combat log)
```

Note: `ExecuteSpellCast` is the direct entry point for casting. `tactical/powers/powercore/pipeline.go`
is **not** used for spell casting — it orchestrates combat *lifecycle* events (post-reset,
attack-complete, turn-end, move-complete) for artifacts and perks. Spells bypass the pipeline
and mutate state directly.

### Effect Lifecycle

```
Effect applied (spell cast or artifact activation)
    → effects.ApplyEffect() [immediate stat modification]
    → Stored in ActiveEffectsData.Effects with RemainingTurns
    ↓
Each turn: effects.TickEffects(entityID, manager)
    / effects.TickEffectsForUnits()
    → Decrement RemainingTurns
    → Remove expired effects (revert stat changes)
    ↓
Combat end: CombatService.cleanupEffects()
    → effects.RemoveAllEffects() (clean slate)
```

Key files: `tactical/powers/spells/system.go`, `tactical/powers/spells/components.go`,
`tactical/powers/effects/system.go`, `tactical/powers/effects/components.go`,
`gui/guispells/`

---

## AI Decision Flow

```
CombatTurnFlow.executeAITurnIfNeeded()
    → Check: currentFaction.IsPlayerControlled == false
    → combatService.GetAIController() (lazy init)
    ↓
aiController.DecideFactionTurn(factionID)
    ├─ updateThreatLayers(currentRound)
    │   ├─ behavior.FactionThreatLevelManager.UpdateAllFactions()
    │   └─ behavior.CompositeThreatEvaluator.Update() per faction
    │       (positional threat + support layer + role multipliers)
    │
    └─ For each alive squad in faction:
        ├─ NewActionContext(squadID, aic)
        └─ executeSquadAction(ctx)
            ↓
        NewActionEvaluator(ctx).EvaluateAllActions()
            ├─ evaluateMovement()
            │   → MoveActions scored by:
            │     threat proximity + ally support + enemy approach
            │     + role-based weights (from aiconfig.json roleBehaviors)
            │
            ├─ evaluateAttacks()
            │   → AttackActions scored by:
            │     target health + role + counter matchups
            │     + power scaling (from powerconfig.json roleMultipliers)
            │
            └─ WaitAction (fallback, score 0)
            ↓
        SelectBestAction(actions) → highest score
            ↓
        bestAction.Execute(manager, movementSystem, combatActSystem, cache)
            ├─ MoveAction → squadcommands.NewMoveSquadCommand().Execute()
            ├─ AttackAction → combatActSystem.ExecuteAttackAction()
            │                 + aiController.QueueAttack()
            └─ WaitAction → sets HasMoved=true, HasActed=true
    ↓
If aiController.HasQueuedAttacks():
    → CombatTurnFlow.playAIAttackAnimations() (chain animations)
    ↓
advanceAfterAITurn() → TurnManager.EndTurn()
```

Key files: `mind/ai/ai_controller.go`, `mind/ai/action_evaluator.go`,
`mind/behavior/`, `mind/evaluation/`, `gui/guicombat/combat_turn_flow.go`

---

## Squad Management Flow

```
┌─────────────────────────────────────────────────┐
│ SQUAD EDITING (overworld: "squad_editor" mode)  │
│                                                 │
│ gui/guisquads/squadeditormode.go                │
│   ├─ View squads: 3x3 grid with unit placements │
│   ├─ Add unit from roster:                      │
│   │   → squadcommands.AddUnitCommand            │
│   │   → squadcore.PlaceUnitInSquad()            │
│   ├─ Remove unit:                               │
│   │   → squadcommands.RemoveUnitCommand         │
│   │   → squadcore.UnassignUnitFromSquad()       │
│   ├─ Change leader:                             │
│   │   → squadcommands.ChangeLeaderCommand       │
│   └─ All via CommandExecutor (undo/redo)        │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│ UNIT PURCHASE (overworld: "unit_purchase" mode) │
│                                                 │
│ gui/guisquads/unitpurchasemode.go               │
│   → squadservices.UnitPurchaseService           │
│   → Adds purchased unit to commander roster     │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│ SQUAD DEPLOYMENT (pre-combat: "squad_deployment")│
│                                                 │
│ gui/guisquads/squaddeploymentmode.go            │
│   → squadservices.SquadDeploymentService        │
│   → Sets SquadData.IsDeployed = true            │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│ SQUAD CREATION (code-level)                     │
│                                                 │
│ squadcore.CreateEmptySquad(manager, name)       │
│   → Squad entity with SquadComponent            │
│                                                 │
│ squadcore.AddUnitToSquad(squadID, manager,      │
│     unit, row, col)                             │
│   → Unit entity with SquadMemberComponent       │
│                                                 │
│ squadcore.CreateSquadFromTemplate(manager, name,│
│     formation, pos, unitTemplates)              │
│   → Full squad with units in 3x3 grid           │
│                                                 │
│ Roster: roster.GetPlayerSquadRoster(ownerID,    │
│     manager)                                    │
│   → Tracked on commander entities               │
└─────────────────────────────────────────────────┘
```

Key files: `tactical/squads/squadcore/squadcreation.go`, `tactical/squads/squadcore/squadmanager.go`,
`tactical/squads/squadcore/units.go`, `tactical/squads/squadcommands/`,
`tactical/squads/squadservices/squad_deployment_service.go`,
`tactical/squads/squadservices/unit_purchase_service.go`,
`tactical/squads/roster/squadroster.go`, `tactical/squads/roster/unitroster.go`,
`gui/guisquads/squadeditormode.go`, `gui/guisquads/squaddeploymentmode.go`

---

## Progression Flow

```
Commander earns XP from combat victories
    ↓
Progression Library tracks available unlocks:
    ├─ Perk unlocks (by tier + category)
    ├─ Spell unlocks
    └─ Artifact unlocks
    ↓
Player opens ProgressionMode (overworld or roguelike)
    ↓
guiprogression.ProgressionMode
    → Shows currency balances (commander XP, battle honors)
    → Lists available unlocks with costs
    → Player spends currency to unlock
    ↓
Unlocked perks/spells/artifacts become equippable
via SquadEditor / ArtifactMode / SpellMode
```

Key files: `tactical/powers/progression/components.go`, `tactical/powers/progression/defaults.go`,
`tactical/powers/progression/library.go`, `tactical/powers/progression/init.go`,
`gui/guiprogression/progressionmode.go`, `gui/guiprogression/progression_controller.go`,
`gui/guiprogression/progression_panels_registry.go`,
`gui/guiprogression/progression_refresh.go`
