
### Approach 1: Incremental Service Layer Extraction

**Strategic Focus**: Gradual extraction with minimal architectural disruption - pragmatic path to testability

**Problem Statement**:
Current code embeds game logic directly in UI handlers, making it impossible to test combat mechanics, movement validation, or squad operations without instantiating the entire GUI stack. Example: CombatActionHandler.executeAttack() (combat_action_handler.go lines 167-204) mixes combat system creation, validation checks, attack execution, destruction checking, AND UI logging in a single method that's untestable in isolation.

**Solution Overview**:
Create dedicated service classes (CombatService, SquadService, MovementService) that encapsulate game logic and are injected into UI handlers. Services own game systems and expose clean APIs. UI handlers become thin coordinators that call services and update UI based on results. This provides immediate testability without requiring full architectural restructure.

**Code Example**:

*Before (combat_action_handler.go lines 167-204):*
```go
func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Create combat action system (BUSINESS LOGIC IN UI)
    combatSys := combat.NewCombatActionSystem(cah.entityManager)

    // Check if attack is valid (BUSINESS LOGIC IN UI)
    reason, canAttack := combatSys.CanSquadAttackWithReason(selectedSquad, selectedTarget)
    if !canAttack {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", reason))
        cah.battleMapState.InAttackMode = false
        return
    }

    // Execute attack (BUSINESS LOGIC IN UI)
    attackerName := cah.queries.GetSquadName(selectedSquad)
    targetName := cah.queries.GetSquadName(selectedTarget)

    err := combatSys.ExecuteAttackAction(selectedSquad, selectedTarget)
    if err != nil {
        cah.addLog(fmt.Sprintf("Attack failed: %v", err))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", attackerName, targetName))

        // Check if target destroyed (BUSINESS LOGIC IN UI)
        if squads.IsSquadDestroyed(selectedTarget, cah.entityManager) {
            cah.addLog(fmt.Sprintf("%s was destroyed!", targetName))
        }
    }

    // Reset attack mode (UI STATE UPDATE)
    cah.battleMapState.InAttackMode = false
}
```

*After - Service Layer (combat/combat_service.go):*
```go
// CombatService encapsulates all combat game logic
type CombatService struct {
    entityManager  *common.EntityManager
    turnManager    *TurnManager
    factionManager *FactionManager
}

func NewCombatService(manager *common.EntityManager) *CombatService {
    return &CombatService{
        entityManager:  manager,
        turnManager:    NewTurnManager(manager),
        factionManager: NewFactionManager(manager),
    }
}

// AttackResult contains all information about an attack
type AttackResult struct {
    Success         bool
    ErrorReason     string
    AttackerName    string
    TargetName      string
    TargetDestroyed bool
    DamageDealt     int
}

// ExecuteSquadAttack performs a squad attack and returns result
func (cs *CombatService) ExecuteSquadAttack(attackerID, targetID ecs.EntityID) *AttackResult {
    result := &AttackResult{}

    // Create combat action system
    combatSys := NewCombatActionSystem(cs.entityManager)

    // Validate attack
    reason, canAttack := combatSys.CanSquadAttackWithReason(attackerID, targetID)
    if !canAttack {
        result.Success = false
        result.ErrorReason = reason
        return result
    }

    // Get names for result
    result.AttackerName = getSquadNameByID(attackerID, cs.entityManager)
    result.TargetName = getSquadNameByID(targetID, cs.entityManager)

    // Execute attack
    err := combatSys.ExecuteAttackAction(attackerID, targetID)
    if err != nil {
        result.Success = false
        result.ErrorReason = err.Error()
        return result
    }

    result.Success = true
    result.TargetDestroyed = squads.IsSquadDestroyed(targetID, cs.entityManager)

    return result
}

// GetTurnManager exposes turn manager for UI queries
func (cs *CombatService) GetTurnManager() *TurnManager {
    return cs.turnManager
}

// GetFactionManager exposes faction manager for UI queries
func (cs *CombatService) GetFactionManager() *FactionManager {
    return cs.factionManager
}
```

*After - UI Handler (combat_action_handler.go):*
```go
type CombatActionHandler struct {
    battleMapState *core.BattleMapState
    logManager     *CombatLogManager
    queries        *guicomponents.GUIQueries
    combatService  *combat.CombatService  // Service injection
    combatLogArea  *widget.TextArea
}

func NewCombatActionHandler(
    battleMapState *core.BattleMapState,
    logManager *CombatLogManager,
    queries *guicomponents.GUIQueries,
    combatService *combat.CombatService,  // Inject service
    combatLogArea *widget.TextArea,
) *CombatActionHandler {
    return &CombatActionHandler{
        battleMapState: battleMapState,
        logManager:     logManager,
        queries:        queries,
        combatService:  combatService,
        combatLogArea:  combatLogArea,
    }
}

func (cah *CombatActionHandler) executeAttack() {
    selectedSquad := cah.battleMapState.SelectedSquadID
    selectedTarget := cah.battleMapState.SelectedTargetID

    if selectedSquad == 0 || selectedTarget == 0 {
        return
    }

    // Call service for all game logic
    result := cah.combatService.ExecuteSquadAttack(selectedSquad, selectedTarget)

    // Handle result - UI ONLY
    if !result.Success {
        cah.addLog(fmt.Sprintf("Cannot attack: %s", result.ErrorReason))
    } else {
        cah.addLog(fmt.Sprintf("%s attacked %s!", result.AttackerName, result.TargetName))
        if result.TargetDestroyed {
            cah.addLog(fmt.Sprintf("%s was destroyed!", result.TargetName))
        }
    }

    // Reset UI state
    cah.battleMapState.InAttackMode = false
}
```

**Key Changes**:
- Created CombatService that owns TurnManager, FactionManager, CombatActionSystem
- Extracted ExecuteSquadAttack() method that returns rich AttackResult struct
- UI handler becomes thin - calls service, interprets result, updates UI
- Business logic is now 100% testable without GUI dependencies
- Service exposes TurnManager/FactionManager for UI queries (read-only access)

**Value Proposition**:
- **Maintainability**: Game logic centralized in services, not scattered across UI handlers
- **Readability**: Clear separation - services = logic, handlers = coordination + UI
- **Extensibility**: Easy to add new game actions (just add service methods)
- **Complexity Impact**:
  - Reduced cyclomatic complexity in UI handlers (from 15+ to <5)
  - New service layer adds ~500 lines but removes ~800 from UI
  - Net -300 lines with better organization

**Implementation Strategy**:
1. **Phase 1 - Create Services** (2 days):
   - Create combat/combat_service.go with CombatService struct
   - Move TurnManager/FactionManager/MovementSystem ownership from CombatMode to CombatService
   - Create squads/squad_service.go with SquadService struct
   - Implement 5-10 core service methods (ExecuteSquadAttack, MoveSquad, CreateSquad, etc.)

2. **Phase 2 - Refactor CombatMode** (1 day):
   - Inject CombatService into CombatMode instead of creating managers directly
   - Update CombatActionHandler to use CombatService
   - Update UI code to call service methods instead of direct system calls
   - Test combat flow end-to-end

3. **Phase 3 - Refactor SquadBuilder** (1 day):
   - Create SquadBuilderService with PlaceUnit, RemoveUnit, CreateSquad methods
   - Inject service into SquadBuilderMode
   - Remove direct ECS manipulation from UI callbacks
   - Test squad builder flow

4. **Phase 4 - Add Tests** (1 day):
   - Write unit tests for CombatService.ExecuteSquadAttack()
   - Write tests for SquadService.CreateSquad() with capacity validation
   - Write tests for MovementService.ValidateMove()
   - Achieve >80% coverage of business logic

**Advantages**:
- **Immediate testability**: Can write unit tests for combat logic TODAY (just test service methods)
- **Incremental migration**: Refactor one UI handler at a time, no big-bang rewrite
- **Low risk**: Services wrap existing ECS code, minimal changes to combat/squads packages
- **Familiar pattern**: Service layer is well-understood, team can adapt quickly
- **Performance neutral**: No event dispatching overhead, direct method calls

**Drawbacks & Risks**:
- **Not complete separation**: Services still directly manipulate ECS, not a true "model" layer
  - *Mitigation*: This is acceptable for Phase 1. Services ARE the model layer for now.
- **Potential for fat services**: Services could grow large with many methods
  - *Mitigation*: Split into domain services (CombatService, SquadService, MovementService)
- **Still tightly coupled to ECS**: Services depend on EntityManager
  - *Mitigation*: This is by design. ECS IS our data model. Services orchestrate ECS systems.
- **Doesn't enable undo/replay**: Service methods mutate state immediately
  - *Mitigation*: Add command layer later if needed. Don't over-engineer now.

**Effort Estimate**:
- **Time**: 5 days (1 week with testing)
- **Complexity**: Low-Medium
- **Risk**: Low (wrapping existing code, not rewriting)
- **Files Impacted**:
  - New: 3 files (combat_service.go, squad_service.go, movement_service.go)
  - Modified: 8 files (combatmode.go, combat_action_handler.go, squadbuilder.go, squad_builder_grid_manager.go, formationeditormode.go, squaddeploymentmode.go, squadmanagementmode.go, gamemodecoordinator.go)

**Critical Assessment** (from refactoring-critic):
This is the **most pragmatic approach**. It provides immediate value (testability) without architectural astronautics. The service layer is a well-understood pattern that team members can grasp in 5 minutes. Risk is low because we're wrapping existing code, not rewriting it. The only danger is treating this as "done" when it's really Phase 1 - services are not a complete separation, but they're a huge step forward. Recommended as **first step** before considering more complex patterns.