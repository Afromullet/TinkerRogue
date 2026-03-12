# Data Flow Patterns

**Last Updated:** 2026-03-12

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
combat.CombatActionSystem.ExecuteAttackAction()
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
`tactical/combat/combatactionsystem.go`, `tactical/combatservices/combat_service.go`

---

## Artifact System Flow

```
Player opens artifact UI (overworld: ArtifactMode, combat: ArtifactActivationHandler)
    ↓
┌─────────────────────────────────────────┐
│ EQUIP/UNEQUIP (overworld only)          │
│                                         │
│ gui/guisquads/artifactmode.go           │
│   → gear.EquipArtifact(playerID,        │
│       squadID, artifactID, manager)     │
│   → gear.UnequipArtifact(...)           │
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
│   → gear.ApplyArtifactStatEffects(      │
│       squadIDs, manager)                │
│   (applies passive stat bonuses)        │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ ACTIVATED ARTIFACTS (during combat)     │
│                                         │
│ guiartifacts.ArtifactActivationHandler  │
│   → gear.AllBehaviors() dispatch        │
│     (registered in CombatService.       │
│      setupBehaviorDispatch())           │
│   → Executes artifact-specific behavior │
│     (artifactbehaviors_activated.go /   │
│      artifactbehaviors_passive.go)      │
└─────────────────────────────────────────┘
```

Key files: `gear/system.go`, `gear/queries.go`, `gear/artifactinventory.go`,
`gear/artifactbehaviors_activated.go`, `gear/components.go`,
`gui/guiartifacts/`, `gui/guisquads/artifactmode.go`

---

## Map Generation Flow

```
Game initialization or encounter setup
    ↓
worldmap.NewGameMap(generatorName)
    ↓
worldmap.GetGeneratorOrDefault(name)
    ├─ Check ConfigOverride (if set)
    └─ Fall back to generator registry
    ↓
Generator.Generate(width, height, images)
    ├─ Initialize tile array
    ├─ Algorithm-specific generation
    │   (rooms_corridors, cavern, overworld,
    │    garrison, military_base, etc.)
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

Available generators: `gen_rooms_corridors.go`, `gen_cavern.go`, `gen_overworld.go`,
`gen_garrison.go`, `gen_military_base.go`

Key files: `world/worldmap/dungeongen.go`, `world/worldmap/generator.go`,
`world/worldmap/mapgenconfig.go`, `gamesetup/bootstrap.go`

---

## Entity/Unit Creation Flow

```
Request entity creation
    ↓
┌──────────────────────────────────────────────┐
│ Single Entity (creature/monster):            │
│                                              │
│ templates.CreateEntityFromTemplate(          │
│     manager, EntityConfig, JSONMonster)       │
│   ├─ Create entity with components:          │
│   │   ├─ PositionComponent                   │
│   │   ├─ AttributeComponent (from template)  │
│   │   ├─ NameComponent                       │
│   │   └─ Relevant tags                       │
│   └─ GlobalPositionSystem.AddEntity()        │
│                                              │
│ OR templates.CreateUnit(mgr, name, attr, pos)│
│   (bare unit entity without template)        │
└──────────────────────────────────────────────┘
    ↓
┌──────────────────────────────────────────────┐
│ Full Squad (with units in 3x3 grid):         │
│                                              │
│ squads.CreateSquadFromTemplate(              │
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
│ encounter_setup.go                           │
│   → createSquadForPowerBudget()              │
│   → Uses squads.Units (from JSON templates)  │
│   → squads.CreateSquadFromTemplate()         │
└──────────────────────────────────────────────┘
```

Templates loaded by `templates.ReadGameData()` from JSON files. Unit templates
initialized via `squads.InitUnitTemplatesFromJSON()`.

Key files: `templates/entity_factory.go`, `templates/registry.go`, `templates/readdata.go`,
`tactical/squads/squadcreation.go`, `mind/encounter/encounter_setup.go`

---

## Game Initialization Flow

```
1. main() starts
   ↓
2. SetupSharedSystems() (setup_shared.go)
   ├─ LoadGameData() → JSON templates
   ├─ InitializeCoreECS() → ECS manager, 50+ components, GlobalPositionSystem
   └─ Configure graphics
   ↓
3. Show StartMenu (Overworld vs Roguelike selection)
   ↓
4. Mode-specific setup
   ├─ SetupOverworldMode() (setup_overworld.go)
   │   OR SetupRoguelikeMode() (setup_roguelike.go)
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
- **Overworld-only:** `overworld`, `node_placement`, `unit_purchase`, `squad_editor`, `artifact`, `unit_view`
- **Roguelike-only:** `raid`

Key files: `game_main/setup.go`, `gamesetup/bootstrap.go`, `gamesetup/moderegistry.go`

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
│ (Tiles)        │          │ Rendering      │
│ [Tactical only]│          │ [Tactical only]│
└───────┬────────┘          └──────┬─────────┘
        │                          │
        └──────────┬───────────────┘
                   │
           ┌───────▼────────┐
           │ Visual Effects │
           │ Rendering      │
           └────────┬───────┘
                    │
           ┌────────▼─────────────┐
           │ GameModeCoordinator  │
           │ .Render()            │
           │ (EbitenUI overlay)   │
           └──────────────────────┘
```

Key files: `game_main/main.go`

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

Key files: `overworld/tick/tickmanager.go`, `gui/guioverworld/overworld_action_handler.go`

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
combatpipeline.ExecuteCombatStart(encounterService, manager, &OverworldCombatStarter{})
    ↓
OverworldCombatStarter.Prepare(manager)
    → encounter.SpawnCombatEntities(...)
        ├─ Generate enemy squads from power budget
        ├─ Create faction entities
        └─ Assign squads to factions
    ↓
EncounterService.TransitionToCombat(setup)
    → beginCombatTransition()
    → modeCoordinator.EnterCombatMode()
    (enters "combat" mode directly)
    ↓
CombatMode (turn-based tactical combat)
    ↓
Combat ends (victory/defeat/retreat)
    ↓
CombatMode.Exit()
    → encounterService.ExitCombat(reason, result, combatCleaner)
    ↓
EndEncounter()
    → combatpipeline.ExecuteResolution(manager, &OverworldCombatResolver{})
        ├─ Grant rewards (XP, artifacts)
        └─ Mark threat node as defeated (or retreat)
    → RecordEncounterCompletion()
        (restores player position)
    → combatCleaner.CleanupCombat(enemySquadIDs)
    → PostCombatCallback (if set, e.g., RaidRunner)
    ↓
Returns to "exploration" mode (or PostCombatReturnMode)
```

Key files: `gui/guioverworld/overworld_action_handler.go`, `mind/encounter/encounter_trigger.go`,
`mind/encounter/encounter_setup.go`, `mind/encounter/encounter_service.go`,
`mind/combatpipeline/pipeline.go`, `mind/combatpipeline/starter.go`

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
        │   ├─ squads.GetUnitIDsInSquad() per target
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

Key files: `tactical/spells/system.go`, `tactical/effects/system.go`,
`tactical/effects/components.go`, `gui/guispells/`

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
│   │   → squads.PlaceUnitInSquad()               │
│   ├─ Remove unit:                               │
│   │   → squadcommands.RemoveUnitCommand         │
│   │   → squads.UnassignUnitFromSquad()          │
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
│   → Player clicks tiles to place squads         │
│   → squadservices.SquadDeploymentService        │
│   → Sets SquadData.IsDeployed = true            │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│ SQUAD CREATION (code-level)                     │
│                                                 │
│ squads.CreateEmptySquad(manager, name)          │
│   → Squad entity with SquadComponent            │
│                                                 │
│ squads.AddUnitToSquad(squadID, manager,          │
│     unit, row, col)                             │
│   → Unit entity with SquadMemberComponent       │
│                                                 │
│ squads.CreateSquadFromTemplate(manager, name,    │
│     formation, pos, unitTemplates)              │
│   → Full squad with units in 3x3 grid           │
│                                                 │
│ Roster: squads.GetPlayerSquadRoster(ownerID, mgr)│
│   → Tracked on commander entities               │
└─────────────────────────────────────────────────┘
```

Key files: `tactical/squads/squadcreation.go`, `tactical/squadcommands/`,
`tactical/squadservices/`, `gui/guisquads/squadeditormode.go`,
`gui/guisquads/squaddeploymentmode.go`
