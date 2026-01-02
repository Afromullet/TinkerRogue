# Implementation Plan: Overworld → Combat Transition System

**Generated:** 2026-01-02
**Feature:** Basic overworld encounter system with combat transition
**Complexity:** Medium - Foundation for future encounter systems

---

## EXECUTIVE SUMMARY

### Feature Overview
This implementation adds a basic overworld encounter system that transitions the game from exploration mode to combat mode when the player avatar collides with AI-controlled encounter entities on the overworld map. This is a foundational feature for future roguelike progression where players navigate an overworld, trigger battles, and return to exploration after combat resolves.

**What:** Spawn AI encounter entities on the overworld map. When the player moves onto a tile occupied by an encounter entity, pause exploration and trigger combat mode using existing squad placement logic.

**Why:** Creates a core gameplay loop: explore → encounter → combat → return to exploration. This is essential for roguelike progression and provides a practical foundation for future features like dynamic encounters, roaming enemies, and world events.

**Inspired By:** Classic roguelike overworld systems (Final Fantasy Tactics world map, Fire Emblem world map, XCOM strategic layer).

**Complexity:** Medium
- Requires new ECS components for encounters
- Needs collision detection integration
- Involves mode transition coordination
- Builds on existing systems (no major refactoring)

### Quick Assessment
- **Recommended Approach:** Plan 1 - Incremental Component-Based Implementation
- **Implementation Time:** 8-12 hours total
- **Risk Level:** Low
- **Blockers:** None - all dependencies exist

### Key Design Decisions

**1. Encounter Representation**
- Use ECS entities with EncounterComponent (data-driven approach)
- Store encounter configuration (AI faction definition, squad templates)
- Position encounters using GlobalPositionSystem (O(1) collision detection)

**2. Collision Detection**
- Hook into MovementController.movePlayer()
- Check GlobalPositionSystem.GetEntityIDAt(nextPosition) before movement
- Filter for encounter entities using EncounterTag
- Trigger combat transition if encounter detected

**3. Combat Initialization**
- Create encounter-specific faction setup (alternative to SetupGameplayFactions)
- Dynamically spawn squads based on encounter configuration
- Use existing combat initialization flow (CombatMode.Enter)

**4. Post-Combat Flow**
- Track encounter completion state
- Remove defeated encounter entities from overworld
- Return to ExplorationMode at original encounter position

---

## IMPLEMENTATION PLAN (RECOMMENDED)

### Plan 1: Incremental Component-Based Implementation

**Strategic Focus:** Build incrementally on existing systems with minimal disruption. Add encounter detection to movement, create encounter entities, and hook into existing combat transition flow.

**Gameplay Value:**
- Immediate playable overworld → combat loop
- Foundation for future encounter variety (different enemy types, difficulty scaling)
- Reuses combat mode and squad systems (no duplication)

**Go Standards Compliance:**
- Pure ECS component-based design (no logic in data structures)
- Query-based encounter detection (follows position system patterns)
- Value-based map keys for O(1) collision detection

**Architecture Overview:**
```
Overworld (Exploration Mode)
    ├── Player moves via MovementController
    ├── EncounterDetectionSystem checks collision
    │   └── Uses GlobalPositionSystem.GetEntityIDAt()
    ├── EncounterTrigger creates combat context
    │   ├── Spawns factions/squads from encounter template
    │   └── Stores encounter state for post-combat cleanup
    └── GameModeCoordinator transitions to Combat Mode

Combat Mode (Existing System)
    ├── Normal combat flow (turn-based, squad actions)
    └── Victory/Flee triggers return transition

Post-Combat Handler
    ├── Cleans up encounter entity (if defeated)
    ├── Updates encounter state component
    └── Returns to Exploration Mode
```

---

## PHASE 1: ENCOUNTER COMPONENTS & ENTITIES (2-3 hours)

### Step 1.1: Create Encounter ECS Components

**File:** `tactical/encounters/components.go` (NEW)

**What:** Define pure data components for encounter entities following ECS best practices.

**Code:**
```go
package encounters

import (
	"game_main/tactical/squads"
	"game_main/world/coords"
	"github.com/bytearena/ecs"
)

// Component markers
var (
	EncounterComponent *ecs.Component // Data about encounter configuration
	EncounterTag       *ecs.Component // Tag to identify encounter entities
)

// EncounterData defines an overworld encounter entity
// Pure data component - no logic
type EncounterData struct {
	EncounterID    ecs.EntityID // EntityID of this encounter entity
	Name           string       // Display name ("Goblin Patrol", "Bandit Ambush")
	Position       coords.LogicalPosition // Overworld position

	// Combat configuration
	AIFactionName  string                  // Name of AI faction to create
	SquadTemplates []SquadSpawnTemplate    // Squads to spawn in combat

	// State tracking
	IsActive       bool // Whether encounter is still on map
	IsDefeated     bool // Whether player has defeated this encounter
	CombatInProgress bool // Whether currently in combat with this encounter
}

// SquadSpawnTemplate defines how to spawn a squad for an encounter
type SquadSpawnTemplate struct {
	SquadName      string                // Squad name
	Formation      squads.FormationType  // Formation type
	RelativePos    coords.LogicalPosition // Position relative to player (for combat placement)
	Units          []squads.UnitTemplate // Units in squad
}

// EncounterState tracks active encounter during combat
// This is separate from EncounterData to maintain clean ECS separation
type EncounterState struct {
	CurrentEncounterID ecs.EntityID // EntityID of encounter currently in combat
	EncounterPosition  coords.LogicalPosition // Position to return to after combat
}
```

**Why Pure Data:**
- Follows ECS best practices from CLAUDE.md
- No methods, only fields
- Query-based access via system functions
- Matches squad system patterns

**Testing Strategy:**
```go
// tactical/encounters/components_test.go
func TestEncounterDataCreation(t *testing.T) {
	encounterData := &EncounterData{
		Name: "Test Encounter",
		Position: coords.LogicalPosition{X: 10, Y: 10},
		AIFactionName: "Test Faction",
		IsActive: true,
	}

	if encounterData.Name != "Test Encounter" {
		t.Errorf("Expected name 'Test Encounter', got %s", encounterData.Name)
	}
}
```

---

### Step 1.2: Register Encounter Components

**File:** `game_main/gamesetup.go`

**What:** Register encounter components during ECS initialization.

**Code:**
```go
// Add to InitializeECS() function after squad component registration

func InitializeECS(em *common.EntityManager) {
	// ... existing component registration ...

	// Squad components (existing)
	squads.SquadComponent = em.World.NewComponent()
	squads.SquadTag = em.World.NewComponent()
	squads.SquadMemberTag = em.World.NewComponent()

	// Encounter components (NEW)
	encounters.EncounterComponent = em.World.NewComponent()
	encounters.EncounterTag = em.World.NewComponent()

	// ... rest of initialization ...
}
```

**Impact:** Minimal - follows existing registration pattern.

---

### Step 1.3: Create Encounter Query Functions

**File:** `tactical/encounters/queries.go` (NEW)

**What:** Query functions to find and filter encounter entities.

**Code:**
```go
package encounters

import (
	"game_main/common"
	"game_main/world/coords"
	"github.com/bytearena/ecs"
)

// GetAllEncounters returns all active encounter entities
// Query-based - no caching (follows ECS best practices)
func GetAllEncounters(manager *common.EntityManager) []*ecs.Entity {
	results := []*ecs.Entity{}
	for _, result := range manager.World.Query(EncounterTag) {
		results = append(results, result.Entity)
	}
	return results
}

// GetEncounterAtPosition finds an encounter at a specific position
// Uses O(1) position system lookup then validates entity type
func GetEncounterAtPosition(pos coords.LogicalPosition, manager *common.EntityManager) *ecs.Entity {
	// Use GlobalPositionSystem for O(1) lookup
	entityID := common.GlobalPositionSystem.GetEntityIDAt(pos)
	if entityID == 0 {
		return nil
	}

	// Verify entity is an encounter
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return nil
	}

	// Check if entity has EncounterTag
	if !entity.HasComponent(EncounterTag) {
		return nil
	}

	return entity
}

// GetEncounterData retrieves EncounterData component from entity
func GetEncounterData(entity *ecs.Entity) *EncounterData {
	return common.GetComponentType[*EncounterData](entity, EncounterComponent)
}

// IsActiveEncounter checks if encounter is still active and can be triggered
func IsActiveEncounter(entity *ecs.Entity) bool {
	data := GetEncounterData(entity)
	return data != nil && data.IsActive && !data.IsDefeated && !data.CombatInProgress
}
```

**Why Query-Based:**
- No caching (recalculated each time)
- Follows squad system patterns
- Leverages GlobalPositionSystem for performance

---

### Step 1.4: Create Encounter Spawning System

**File:** `tactical/encounters/system.go` (NEW)

**What:** System functions to create and manage encounter entities.

**Code:**
```go
package encounters

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"github.com/bytearena/ecs"
)

// CreateEncounter spawns an encounter entity on the overworld
// This is a system function - logic outside components
func CreateEncounter(
	manager *common.EntityManager,
	name string,
	position coords.LogicalPosition,
	aiFactionName string,
	squadTemplates []SquadSpawnTemplate,
) (ecs.EntityID, error) {
	// Create encounter entity
	encounterEntity := manager.World.NewEntity()
	encounterID := encounterEntity.GetID()

	// Add encounter data component
	encounterData := &EncounterData{
		EncounterID:      encounterID,
		Name:             name,
		Position:         position,
		AIFactionName:    aiFactionName,
		SquadTemplates:   squadTemplates,
		IsActive:         true,
		IsDefeated:       false,
		CombatInProgress: false,
	}
	encounterEntity.AddComponent(EncounterComponent, encounterData)

	// Add encounter tag
	encounterEntity.AddComponent(EncounterTag, &struct{}{})

	// Add position component (for consistency with other entities)
	positionComp := &common.PositionData{LogicalPos: position}
	encounterEntity.AddComponent(common.PositionComponent, positionComp)

	// Add to GlobalPositionSystem for O(1) collision detection
	if err := common.GlobalPositionSystem.AddEntity(encounterID, position); err != nil {
		return 0, fmt.Errorf("failed to add encounter to position system: %w", err)
	}

	// Add renderable component (visual indicator on overworld)
	// Use red '!' symbol to indicate encounter
	renderable := &rendering.Renderable{
		Visible: true,
		Char:    '!',
		FGColor: rendering.ColorRed,
		BGColor: rendering.ColorBlack,
	}
	encounterEntity.AddComponent(rendering.RenderableComponent, renderable)

	fmt.Printf("Created encounter '%s' at position (%d, %d)\n", name, position.X, position.Y)
	return encounterID, nil
}

// RemoveEncounter cleans up an encounter entity
// Called after combat completes (if encounter is defeated)
func RemoveEncounter(encounterID ecs.EntityID, manager *common.EntityManager) error {
	entity := manager.FindEntityByID(encounterID)
	if entity == nil {
		return fmt.Errorf("encounter entity %d not found", encounterID)
	}

	data := GetEncounterData(entity)
	if data == nil {
		return fmt.Errorf("encounter %d has no EncounterData component", encounterID)
	}

	// Remove from position system
	if err := common.GlobalPositionSystem.RemoveEntity(encounterID, data.Position); err != nil {
		return fmt.Errorf("failed to remove encounter from position system: %w", err)
	}

	// Mark as inactive (don't dispose yet - might need for encounter history)
	data.IsActive = false

	fmt.Printf("Removed encounter '%s' from overworld\n", data.Name)
	return nil
}

// SpawnTestEncounters creates a few test encounters near player spawn
// Temporary function for testing - will be replaced by encounter generation system
func SpawnTestEncounters(manager *common.EntityManager, playerPos coords.LogicalPosition) error {
	// Encounter 1: Simple goblin patrol (2 squads)
	encounter1Templates := []SquadSpawnTemplate{
		{
			SquadName:   "Goblin Squad 1",
			Formation:   squads.FormationBalanced,
			RelativePos: coords.LogicalPosition{X: -5, Y: -3},
			Units:       createSimpleGoblinUnits(3), // Helper function below
		},
		{
			SquadName:   "Goblin Squad 2",
			Formation:   squads.FormationBalanced,
			RelativePos: coords.LogicalPosition{X: 5, Y: -3},
			Units:       createSimpleGoblinUnits(3),
		},
	}

	encounterPos1 := coords.LogicalPosition{
		X: playerPos.X + 10,
		Y: playerPos.Y - 5,
	}

	if _, err := CreateEncounter(
		manager,
		"Goblin Patrol",
		encounterPos1,
		"Goblin Raiders",
		encounter1Templates,
	); err != nil {
		return fmt.Errorf("failed to create encounter 1: %w", err)
	}

	// Encounter 2: Larger bandit group (3 squads)
	encounter2Templates := []SquadSpawnTemplate{
		{
			SquadName:   "Bandit Squad 1",
			Formation:   squads.FormationBalanced,
			RelativePos: coords.LogicalPosition{X: -7, Y: 0},
			Units:       createSimpleGoblinUnits(4),
		},
		{
			SquadName:   "Bandit Squad 2",
			Formation:   squads.FormationBalanced,
			RelativePos: coords.LogicalPosition{X: 0, Y: 0},
			Units:       createSimpleGoblinUnits(4),
		},
		{
			SquadName:   "Bandit Squad 3",
			Formation:   squads.FormationBalanced,
			RelativePos: coords.LogicalPosition{X: 7, Y: 0},
			Units:       createSimpleGoblinUnits(4),
		},
	}

	encounterPos2 := coords.LogicalPosition{
		X: playerPos.X - 15,
		Y: playerPos.Y + 8,
	}

	if _, err := CreateEncounter(
		manager,
		"Bandit Ambush",
		encounterPos2,
		"Bandit Horde",
		encounter2Templates,
	); err != nil {
		return fmt.Errorf("failed to create encounter 2: %w", err)
	}

	fmt.Printf("Spawned 2 test encounters on overworld\n")
	return nil
}

// createSimpleGoblinUnits creates a simple squad of goblins for testing
// Uses existing unit templates from squads.Units
func createSimpleGoblinUnits(count int) []squads.UnitTemplate {
	units := []squads.UnitTemplate{}

	// Grid positions for units
	gridPositions := [][2]int{
		{0, 0}, {0, 1}, {0, 2}, // Front row
		{1, 0}, {1, 1}, {1, 2}, // Middle row
		{2, 0}, {2, 1}, {2, 2}, // Back row
	}

	for i := 0; i < count && i < len(gridPositions); i++ {
		// Use first available unit template (or cycle through templates)
		unitTemplate := squads.Units[i%len(squads.Units)]
		unitTemplate.GridRow = gridPositions[i][0]
		unitTemplate.GridCol = gridPositions[i][1]
		unitTemplate.IsLeader = (i == 0) // First unit is leader

		if unitTemplate.IsLeader {
			unitTemplate.Attributes.Leadership = 20
		}

		units = append(units, unitTemplate)
	}

	return units
}
```

**Key Design Choices:**
- System functions (CreateEncounter, RemoveEncounter) not component methods
- Uses GlobalPositionSystem for collision detection
- Renderable component added for visual indicator
- SpawnTestEncounters is temporary scaffolding

---

## PHASE 2: ENCOUNTER DETECTION (2-3 hours)

### Step 2.1: Add Collision Detection to MovementController

**File:** `input/movementcontroller.go`

**What:** Hook encounter detection into player movement flow.

**Code:**
```go
// Add at top of file
import (
	"game_main/tactical/encounters"
)

// Modify movePlayer function to check for encounters BEFORE movement
func (mc *MovementController) movePlayer(xOffset, yOffset int) {
	nextPosition := coords.LogicalPosition{
		X: mc.playerData.Pos.X + xOffset,
		Y: mc.playerData.Pos.Y + yOffset,
	}

	nextLogicalPos := coords.LogicalPosition{X: nextPosition.X, Y: nextPosition.Y}
	index := coords.CoordManager.LogicalToIndex(nextLogicalPos)
	nextTile := mc.gameMap.Tiles[index]

	currentLogicalPos := coords.LogicalPosition{X: mc.playerData.Pos.X, Y: mc.playerData.Pos.Y}
	index = coords.CoordManager.LogicalToIndex(currentLogicalPos)
	oldTile := mc.gameMap.Tiles[index]

	// NEW: Check for encounter at next position BEFORE checking blocked tiles
	encounterEntity := encounters.GetEncounterAtPosition(nextLogicalPos, mc.ecsManager)
	if encounterEntity != nil && encounters.IsActiveEncounter(encounterEntity) {
		// Trigger encounter
		mc.triggerEncounter(encounterEntity, nextLogicalPos)
		return // Don't proceed with movement - entering combat
	}

	// Existing movement logic (unchanged)
	if !nextTile.Blocked {
		// Update PositionSystem before moving player
		if common.GlobalPositionSystem != nil {
			common.GlobalPositionSystem.MoveEntity(
				mc.playerData.PlayerEntityID,
				currentLogicalPos,
				nextLogicalPos,
			)
		}

		mc.playerData.Pos.X = nextPosition.X
		mc.playerData.Pos.Y = nextPosition.Y
		nextTile.Blocked = true
		oldTile.Blocked = false
	}
}

// triggerEncounter initiates combat with an encounter entity
func (mc *MovementController) triggerEncounter(encounterEntity *ecs.Entity, encounterPos coords.LogicalPosition) {
	data := encounters.GetEncounterData(encounterEntity)
	if data == nil {
		fmt.Println("ERROR: Encounter entity has no EncounterData")
		return
	}

	fmt.Printf("=== ENCOUNTER: %s ===\n", data.Name)

	// Mark encounter as in combat
	data.CombatInProgress = true

	// Store encounter state for post-combat cleanup
	// This will be used by a new EncounterManager (Phase 3)
	mc.sharedState.CurrentEncounterID = encounterEntity.GetID()
	mc.sharedState.EncounterPosition = encounterPos

	// Trigger combat mode transition
	// This will be handled by a new EncounterTransitionHandler (Phase 3)
	mc.sharedState.EncounterTriggered = true
}
```

**Why Hook Here:**
- MovementController already handles player movement
- Collision detection happens before movement validation
- Minimal disruption to existing flow

**Alternative Approach (Considered but Rejected):**
- Separate EncounterDetectionSystem running each frame
- Rejected because: Movement-based detection is simpler and avoids redundant position checks

---

### Step 2.2: Add Encounter State to SharedInputState

**File:** `input/inputcoordinator.go`

**What:** Track encounter state across input system.

**Code:**
```go
// Modify SharedInputState struct
type SharedInputState struct {
	PrevCursor         coords.PixelPosition
	PrevThrowInds      []int
	PrevRangedAttInds  []int
	PrevTargetLineInds []int
	TurnTaken          bool

	// NEW: Encounter state
	EncounterTriggered  bool             // Whether encounter was just triggered
	CurrentEncounterID  ecs.EntityID     // EntityID of current encounter
	EncounterPosition   coords.LogicalPosition // Position of encounter
}

// Update NewSharedInputState
func NewSharedInputState() *SharedInputState {
	return &SharedInputState{
		PrevCursor:         coords.PixelPosition{X: -1, Y: -1},
		PrevThrowInds:      make([]int, 0),
		PrevRangedAttInds:  make([]int, 0),
		PrevTargetLineInds: make([]int, 0),
		TurnTaken:          false,

		// NEW
		EncounterTriggered: false,
		CurrentEncounterID: 0,
		EncounterPosition:  coords.LogicalPosition{},
	}
}
```

**Impact:** Minimal - adds state tracking for encounter transition.

---

## PHASE 3: COMBAT TRANSITION HANDLER (3-4 hours)

### Step 3.1: Create EncounterCombatInitializer

**File:** `tactical/encounters/combat_init.go` (NEW)

**What:** Initialize combat from encounter configuration (alternative to SetupGameplayFactions).

**Code:**
```go
package encounters

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"github.com/bytearena/ecs"
)

// InitializeCombatFromEncounter spawns factions and squads for an encounter
// This replaces SetupGameplayFactions for encounter-based combat
func InitializeCombatFromEncounter(
	manager *common.EntityManager,
	encounterID ecs.EntityID,
	playerFactionID ecs.EntityID, // Existing player faction (if persistent)
) error {
	// Get encounter data
	encounterEntity := manager.FindEntityByID(encounterID)
	if encounterEntity == nil {
		return fmt.Errorf("encounter entity %d not found", encounterID)
	}

	data := GetEncounterData(encounterEntity)
	if data == nil {
		return fmt.Errorf("encounter %d has no EncounterData", encounterID)
	}

	fmt.Printf("Initializing combat for encounter: %s\n", data.Name)

	// Create faction manager
	fm := combat.NewFactionManager(manager)

	// Create AI faction for encounter
	aiFactionID := fm.CreateFactionWithPlayer(data.AIFactionName, 0, "")

	// Spawn squads from encounter templates
	for _, squadTemplate := range data.SquadTemplates {
		// Calculate world position (encounter templates use relative positions)
		// In combat, we position squads relative to player faction
		// This is a simplified version - real implementation would handle positioning better
		squadPos := coords.LogicalPosition{
			X: squadTemplate.RelativePos.X + 50, // Center of combat map
			Y: squadTemplate.RelativePos.Y + 40,
		}

		// Create squad entity
		squadID := squads.CreateSquadFromTemplate(
			manager,
			squadTemplate.SquadName,
			squadTemplate.Formation,
			squadPos,
			squadTemplate.Units,
		)

		// Add squad to AI faction
		if err := fm.AddSquadToFaction(aiFactionID, squadID, squadPos); err != nil {
			return fmt.Errorf("failed to add squad to AI faction: %w", err)
		}

		// Create action state for squad
		if err := createActionStateForSquad(manager, squadID); err != nil {
			return fmt.Errorf("failed to create action state: %w", err)
		}
	}

	fmt.Printf("Spawned %d AI squads for encounter\n", len(data.SquadTemplates))

	// If using persistent player faction, we're done
	// If creating fresh player squads for each encounter, spawn them here
	// For now, we'll use the existing player faction from SetupGameplayFactions

	return nil
}

// createActionStateForSquad creates ActionStateData for a squad
// Copied from gameplayfactions.go - should be moved to shared location
func createActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) error {
	actionEntity := manager.World.NewEntity()

	movementSpeed := squads.GetSquadMovementSpeed(squadID, manager)
	if movementSpeed == 0 {
		movementSpeed = 3
	}

	actionEntity.AddComponent(combat.ActionStateComponent, &combat.ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})

	return nil
}

// CleanupEncounterCombat removes AI faction and squads after combat
// Called when returning to overworld
func CleanupEncounterCombat(
	manager *common.EntityManager,
	encounterID ecs.EntityID,
	playerVictory bool,
) error {
	// Get encounter data
	encounterEntity := manager.FindEntityByID(encounterID)
	if encounterEntity == nil {
		return fmt.Errorf("encounter entity %d not found", encounterID)
	}

	data := GetEncounterData(encounterEntity)
	if data == nil {
		return fmt.Errorf("encounter %d has no EncounterData", encounterID)
	}

	// Mark combat as complete
	data.CombatInProgress = false

	if playerVictory {
		// Player won - mark encounter as defeated and remove from map
		data.IsDefeated = true
		if err := RemoveEncounter(encounterID, manager); err != nil {
			fmt.Printf("Warning: failed to remove encounter: %v\n", err)
		}
	} else {
		// Player fled or lost - encounter remains active
		fmt.Printf("Encounter '%s' remains active on overworld\n", data.Name)
	}

	// Clean up AI faction and squads
	// TODO: Implement faction cleanup (remove all squads, dispose faction entity)
	// For now, squads will be cleaned up by existing combat exit logic

	return nil
}
```

**Key Design:**
- Spawns AI faction dynamically from encounter template
- Reuses existing squad creation logic
- Handles cleanup after combat
- Supports both victory and flee outcomes

---

### Step 3.2: Create EncounterTransitionCoordinator

**File:** `tactical/encounters/transition.go` (NEW)

**What:** Coordinate mode transitions between exploration and combat for encounters.

**Code:**
```go
package encounters

import (
	"fmt"
	"game_main/common"
	"game_main/gui/core"
	"github.com/bytearena/ecs"
)

// EncounterTransitionCoordinator manages encounter → combat → exploration flow
type EncounterTransitionCoordinator struct {
	manager           *common.EntityManager
	modeCoordinator   *core.GameModeCoordinator

	// Encounter state
	activeEncounterID ecs.EntityID
	inEncounterCombat bool
}

// NewEncounterTransitionCoordinator creates a new transition coordinator
func NewEncounterTransitionCoordinator(
	manager *common.EntityManager,
	modeCoordinator *core.GameModeCoordinator,
) *EncounterTransitionCoordinator {
	return &EncounterTransitionCoordinator{
		manager:           manager,
		modeCoordinator:   modeCoordinator,
		activeEncounterID: 0,
		inEncounterCombat: false,
	}
}

// StartEncounterCombat transitions from exploration to combat mode
func (etc *EncounterTransitionCoordinator) StartEncounterCombat(encounterID ecs.EntityID) error {
	fmt.Printf("Starting encounter combat (encounter ID: %d)\n", encounterID)

	// Store encounter ID
	etc.activeEncounterID = encounterID
	etc.inEncounterCombat = true

	// Initialize combat from encounter configuration
	// Note: Player faction should already exist from SetupGameplayFactions
	// We only need to spawn AI faction and squads
	if err := InitializeCombatFromEncounter(etc.manager, encounterID, 0); err != nil {
		return fmt.Errorf("failed to initialize encounter combat: %w", err)
	}

	// Transition to combat mode
	if err := etc.modeCoordinator.EnterBattleMap("combat"); err != nil {
		return fmt.Errorf("failed to enter combat mode: %w", err)
	}

	return nil
}

// EndEncounterCombat transitions from combat back to exploration
func (etc *EncounterTransitionCoordinator) EndEncounterCombat(playerVictory bool) error {
	if !etc.inEncounterCombat {
		return fmt.Errorf("not in encounter combat")
	}

	fmt.Printf("Ending encounter combat (victory: %t)\n", playerVictory)

	// Clean up encounter combat state
	if err := CleanupEncounterCombat(etc.manager, etc.activeEncounterID, playerVictory); err != nil {
		fmt.Printf("Warning: encounter cleanup failed: %v\n", err)
	}

	// Reset encounter state
	etc.activeEncounterID = 0
	etc.inEncounterCombat = false

	// Return to exploration mode
	if err := etc.modeCoordinator.ReturnToOverworld("exploration"); err != nil {
		return fmt.Errorf("failed to return to exploration: %w", err)
	}

	return nil
}

// IsInEncounterCombat returns whether currently in encounter-based combat
func (etc *EncounterTransitionCoordinator) IsInEncounterCombat() bool {
	return etc.inEncounterCombat
}

// GetActiveEncounterID returns the current encounter ID (0 if none)
func (etc *EncounterTransitionCoordinator) GetActiveEncounterID() ecs.EntityID {
	return etc.activeEncounterID
}
```

**Purpose:**
- Centralized coordination for encounter transitions
- Handles combat initialization and cleanup
- Tracks encounter state across mode transitions

---

### Step 3.3: Integrate EncounterTransitionCoordinator into Game

**File:** `game_main/game.go` (assuming this is the main Game struct)

**What:** Add EncounterTransitionCoordinator to Game struct and check for encounter triggers each frame.

**Code:**
```go
// Add to Game struct
type Game struct {
	// ... existing fields ...
	encounterCoordinator *encounters.EncounterTransitionCoordinator
}

// Modify game initialization (in SetupNewGame or similar)
func (g *Game) InitializeEncounterSystem() {
	g.encounterCoordinator = encounters.NewEncounterTransitionCoordinator(
		&g.em,
		g.modeCoordinator, // Assuming modeCoordinator is accessible
	)

	// Spawn test encounters
	if err := encounters.SpawnTestEncounters(&g.em, *g.playerData.Pos); err != nil {
		fmt.Printf("Warning: failed to spawn test encounters: %v\n", err)
	}
}

// Modify Update loop (in game Update method)
func (g *Game) Update() error {
	// ... existing update logic ...

	// Check if encounter was triggered this frame
	if g.inputCoordinator.GetSharedState().EncounterTriggered {
		encounterID := g.inputCoordinator.GetSharedState().CurrentEncounterID

		// Start encounter combat
		if err := g.encounterCoordinator.StartEncounterCombat(encounterID); err != nil {
			fmt.Printf("ERROR: Failed to start encounter combat: %v\n", err)
		}

		// Reset trigger flag
		g.inputCoordinator.GetSharedState().EncounterTriggered = false
	}

	// ... rest of update logic ...
}
```

**Integration Points:**
- Game struct holds EncounterTransitionCoordinator
- Update loop checks for encounter triggers
- SpawnTestEncounters called during initialization

---

## PHASE 4: POST-COMBAT FLOW (2-3 hours)

### Step 4.1: Add Victory/Flee Handlers to CombatMode

**File:** `gui/guicombat/combatmode.go`

**What:** Detect combat completion and trigger return to exploration.

**Code:**
```go
// Modify handleFlee function to check for encounter combat
func (cm *CombatMode) handleFlee() {
	// Check if this is encounter-based combat
	// This requires access to EncounterTransitionCoordinator
	// For now, we'll add a flag to BattleMapState to track this

	battleState := cm.Context.ModeCoordinator.GetBattleMapState()

	// If in encounter combat, notify coordinator
	if battleState.IsEncounterCombat {
		// Trigger encounter end with player flee
		// This will be called via a callback to EncounterTransitionCoordinator
		// For now, just return to exploration mode (existing behavior)
		if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
			cm.ModeManager.RequestTransition(exploreMode, "Fled from encounter")
		}
	} else {
		// Normal combat flee (non-encounter)
		if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
			cm.ModeManager.RequestTransition(exploreMode, "Fled from combat")
		}
	}
}

// Add victory detection (simplified - full implementation needs victory condition checking)
func (cm *CombatMode) checkEncounterVictory() {
	// Check if AI faction is defeated
	currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()

	// Get all factions
	allFactions := cm.Queries.GetAllFactions()

	// Count alive factions
	aliveFactions := 0
	for _, factionID := range allFactions {
		factionData := cm.Queries.CombatCache.FindFactionDataByID(factionID, cm.Queries.ECSManager)
		if factionData != nil && factionData.IsPlayerControlled {
			continue // Skip player faction
		}

		// Check if faction has any alive squads
		hasAliveSquads := false
		for _, squadID := range factionData.SquadIDs {
			squadEntity := cm.Queries.ECSManager.FindEntityByID(squadID)
			if squadEntity != nil {
				squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
				if squadData != nil && !squadData.IsDestroyed {
					hasAliveSquads = true
					break
				}
			}
		}

		if hasAliveSquads {
			aliveFactions++
		}
	}

	// If no AI factions remain, player wins
	if aliveFactions == 0 {
		cm.triggerEncounterVictory()
	}
}

// triggerEncounterVictory handles encounter victory
func (cm *CombatMode) triggerEncounterVictory() {
	battleState := cm.Context.ModeCoordinator.GetBattleMapState()

	if battleState.IsEncounterCombat {
		// Trigger encounter end with player victory
		// This will be called via callback
		cm.logManager.UpdateTextArea(cm.combatLogArea, "=== VICTORY ===")

		// Return to exploration after short delay
		if exploreMode, exists := cm.ModeManager.GetMode("exploration"); exists {
			cm.ModeManager.RequestTransition(exploreMode, "Encounter victory")
		}
	}
}
```

**Note:** Full victory detection requires integration with existing combat service victory condition checking. This is a simplified version for encounter flow.

---

### Step 4.2: Add IsEncounterCombat Flag to BattleMapState

**File:** `gui/core/contextstate.go`

**What:** Track whether current combat is encounter-based.

**Code:**
```go
// Modify BattleMapState struct
type BattleMapState struct {
	// UI Selection State
	SelectedSquadID  ecs.EntityID
	SelectedTargetID ecs.EntityID

	// UI Mode Flags
	InAttackMode bool
	InMoveMode   bool

	// NEW: Encounter tracking
	IsEncounterCombat bool          // Whether this combat is from an encounter
	EncounterID       ecs.EntityID  // EntityID of encounter (if applicable)
}

// Update Reset method
func (bms *BattleMapState) Reset() {
	bms.SelectedSquadID = ecs.EntityID(0)
	bms.SelectedTargetID = ecs.EntityID(0)
	bms.InAttackMode = false
	bms.InMoveMode = false

	// NEW
	bms.IsEncounterCombat = false
	bms.EncounterID = 0
}
```

**Impact:** Minimal - adds state tracking for encounter vs normal combat.

---

### Step 4.3: Connect Post-Combat Cleanup

**File:** `tactical/encounters/transition.go`

**What:** Ensure cleanup happens when combat mode exits.

**Code:**
```go
// Modify EndEncounterCombat to be called from CombatMode.Exit
// Alternative: Add callback to CombatMode that gets called on exit

// In CombatMode.Exit, check if this is encounter combat
func (cm *CombatMode) Exit(toMode core.UIMode) error {
	fmt.Println("Exiting Combat Mode")

	battleState := cm.Context.ModeCoordinator.GetBattleMapState()

	// NEW: Check if this is encounter combat
	if battleState.IsEncounterCombat && battleState.EncounterID != 0 {
		// Determine if player won or fled
		playerVictory := cm.checkPlayerVictory() // Simplified - use existing victory checking

		// Call encounter cleanup via coordinator
		// This requires access to EncounterTransitionCoordinator
		// For now, we'll add it to UIContext
		if cm.Context.EncounterCoordinator != nil {
			if err := cm.Context.EncounterCoordinator.EndEncounterCombat(playerVictory); err != nil {
				fmt.Printf("Warning: encounter cleanup failed: %v\n", err)
			}
		}

		// Reset encounter state
		battleState.IsEncounterCombat = false
		battleState.EncounterID = 0
	}

	// ... existing exit logic ...
}

// Helper function to check if player won
func (cm *CombatMode) checkPlayerVictory() bool {
	// Simplified victory check - use existing victory condition logic
	victor := cm.combatService.CheckVictoryCondition()
	return victor.VictorFaction != 0 && victor.VictorName != "" // Player won if victor exists
}
```

---

## PHASE 5: INTEGRATION & TESTING (2-3 hours)

### Step 5.1: Add EncounterCoordinator to UIContext

**File:** `gui/core/uimode.go`

**What:** Make EncounterTransitionCoordinator accessible to UI modes.

**Code:**
```go
// Modify UIContext struct
type UIContext struct {
	ECSManager           *common.EntityManager
	PlayerData           *common.PlayerData
	GameMap              interface{}
	ScreenWidth          int
	ScreenHeight         int
	TileSize             int
	ModeCoordinator      *GameModeCoordinator
	Queries              interface{}

	// NEW
	EncounterCoordinator interface{} // *encounters.EncounterTransitionCoordinator (interface{} to avoid circular import)
}
```

**Impact:** Allows CombatMode and other UI modes to access encounter coordinator for cleanup.

---

### Step 5.2: Wire Everything Together in gamesetup.go

**File:** `game_main/gamesetup.go`

**What:** Initialize encounter system during game setup.

**Code:**
```go
// Add new phase to SetupNewGame
func SetupNewGame(g *Game) {
	bootstrap := NewGameBootstrap()

	// ... existing phases ...

	// Phase 5: Initialize gameplay systems
	bootstrap.InitializeGameplay(&g.em, &g.playerData)

	// NEW Phase 6: Initialize encounter system
	bootstrap.InitializeEncounterSystem(&g.em, &g.playerData, g.modeCoordinator)
}

// Add new bootstrap method
func (gb *GameBootstrap) InitializeEncounterSystem(
	em *common.EntityManager,
	pd *common.PlayerData,
	modeCoordinator *core.GameModeCoordinator,
) *encounters.EncounterTransitionCoordinator {
	// Create encounter coordinator
	encounterCoordinator := encounters.NewEncounterTransitionCoordinator(em, modeCoordinator)

	// Spawn test encounters
	if err := encounters.SpawnTestEncounters(em, *pd.Pos); err != nil {
		log.Printf("Warning: failed to spawn test encounters: %v", err)
	}

	fmt.Println("Encounter system initialized")
	return encounterCoordinator
}
```

---

### Step 5.3: Manual Testing Checklist

**Test Scenario 1: Basic Encounter Trigger**
1. Build and run: `go build -o game_main/game_main.exe game_main/*.go && ./game_main/game_main.exe`
2. Move player to encounter position (should see red '!' on map)
3. Verify combat mode transitions
4. Expected: Combat mode loads with AI faction squads

**Test Scenario 2: Combat Victory Flow**
1. Trigger encounter
2. Defeat all AI squads
3. Verify return to exploration mode
4. Expected: Encounter '!' disappears from map

**Test Scenario 3: Flee from Encounter**
1. Trigger encounter
2. Press ESC to flee
3. Verify return to exploration mode
4. Expected: Encounter '!' remains on map (can be re-triggered)

**Test Scenario 4: Multiple Encounters**
1. Trigger first encounter, flee
2. Move to second encounter, defeat enemies
3. Return to first encounter
4. Expected: All encounters work independently

**Performance Tests:**
```bash
go test ./tactical/encounters/... -v
go test ./tactical/encounters/... -bench=.
```

**Build Verification:**
```bash
go build -o game_main/game_main.exe game_main/*.go
go test ./...
```

---

## ARCHITECTURAL CONSIDERATIONS

### ECS Compliance Checklist
- ✅ Pure data components (EncounterData, SquadSpawnTemplate)
- ✅ No logic in components
- ✅ Query-based entity access (GetEncounterAtPosition, GetAllEncounters)
- ✅ System functions in separate files (system.go, queries.go)
- ✅ EntityID only (no *ecs.Entity storage)
- ✅ Value-based map keys (GlobalPositionSystem.GetEntityIDAt)

### Performance Considerations
- **O(1) collision detection:** Uses GlobalPositionSystem.GetEntityIDAt (hash map lookup)
- **No caching:** Encounter queries recalculate each time (follows ECS best practices)
- **Minimal allocations:** Encounter entities are sparse (2-5 per overworld map)
- **Hot path analysis:** Movement collision check adds ~1 hash lookup per move (negligible)

### Integration Points
- **MovementController:** Hooks into movePlayer() for detection
- **GlobalPositionSystem:** Used for O(1) position lookups
- **GameModeCoordinator:** Handles context switching
- **CombatMode:** Victory/flee detection for post-combat flow
- **SetupGameplayFactions:** Alternative spawning for encounters

---

## FUTURE ENHANCEMENTS (NOT IN THIS ITERATION)

### Phase 2 Features (Future Roadmap)
1. **Dynamic Encounter Generation**
   - Procedural encounter templates based on player level
   - Difficulty scaling
   - Loot tables

2. **Roaming Encounters**
   - AI-controlled movement on overworld
   - Chase/flee behaviors
   - Patrol patterns

3. **Encounter Variety**
   - Multi-stage encounters (reinforcements)
   - Environmental effects (terrain bonuses)
   - Special conditions (ambush, surprise)

4. **Persistent World State**
   - Encounter respawn timers
   - Faction territory control
   - Dynamic encounter spawning

5. **Player Faction Persistence**
   - Carry squads between encounters
   - Permanent squad losses
   - Experience/leveling

---

## RISK MITIGATION

### Identified Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Combat initialization conflicts with SetupGameplayFactions | Medium | Medium | Use separate InitializeCombatFromEncounter, isolate faction creation |
| Mode transition timing issues | Medium | Low | Use existing transition queuing (ModeManager.RequestTransition) |
| Encounter cleanup leaves orphaned entities | Medium | Medium | Implement thorough cleanup in CleanupEncounterCombat, add disposal verification |
| GlobalPositionSystem collision detection misses encounters | High | Low | Use HasComponent(EncounterTag) validation after position lookup |
| Player faction squads lost between encounters | Low | Low | This iteration uses persistent squads from SetupGameplayFactions |

### Rollback Plan
If encounter system causes issues:
1. Remove encounter detection hook from MovementController.movePlayer()
2. Remove EncounterTriggered check from game Update loop
3. Game returns to normal exploration/combat flow
4. Encounter entities remain harmless (just visual indicators)

---

## SUCCESS CRITERIA

**Minimum Viable Implementation:**
- ✅ Encounter entities spawn on overworld map
- ✅ Player collision triggers combat mode
- ✅ AI squads spawn from encounter template
- ✅ Combat completes normally
- ✅ Return to exploration after victory/flee
- ✅ Defeated encounters removed from map

**Quality Gates:**
- ✅ Build compiles: `go build -o game_main/game_main.exe game_main/*.go`
- ✅ All tests pass: `go test ./...`
- ✅ No regressions in existing exploration/combat
- ✅ ECS patterns followed (pure data components, query-based access)
- ✅ No memory leaks (encounter entities properly disposed)

**Performance Targets:**
- Collision detection: <1ms per movement
- Encounter spawn: <10ms per encounter
- Combat initialization: <50ms
- Memory: <100KB per encounter entity

---

## IMPLEMENTATION TIMELINE

### Day 1 (4 hours)
- **Morning (2h):** Phase 1 - Encounter Components
  - Create components.go, queries.go, system.go
  - Register components in gamesetup.go
  - Write basic unit tests

- **Afternoon (2h):** Phase 1 cont. - Encounter Spawning
  - Implement CreateEncounter, SpawnTestEncounters
  - Test encounter entity creation
  - Verify GlobalPositionSystem integration

### Day 2 (4 hours)
- **Morning (2h):** Phase 2 - Encounter Detection
  - Modify MovementController.movePlayer()
  - Add SharedInputState encounter tracking
  - Test collision detection

- **Afternoon (2h):** Phase 3 - Combat Transition
  - Create combat_init.go with InitializeCombatFromEncounter
  - Create transition.go with EncounterTransitionCoordinator
  - Wire into game Update loop

### Day 3 (4 hours)
- **Morning (2h):** Phase 4 - Post-Combat Flow
  - Modify CombatMode flee/victory handlers
  - Add IsEncounterCombat to BattleMapState
  - Implement cleanup logic

- **Afternoon (2h):** Phase 5 - Integration & Testing
  - Wire EncounterCoordinator into UIContext
  - Run manual test scenarios
  - Fix any integration issues

**Total Estimated Time:** 12 hours (3 half-days)

---

## APPENDIX: CODE PATTERNS REFERENCE

### ECS Component Pattern
```go
// Pure data component
type EncounterData struct {
	EncounterID ecs.EntityID
	Name        string
	// ... only fields, no methods
}
```

### ECS Query Pattern
```go
// Query function (no caching)
func GetAllEncounters(manager *common.EntityManager) []*ecs.Entity {
	results := []*ecs.Entity{}
	for _, result := range manager.World.Query(EncounterTag) {
		results = append(results, result.Entity)
	}
	return results
}
```

### ECS System Function Pattern
```go
// System function (logic outside component)
func CreateEncounter(manager *common.EntityManager, name string, pos coords.LogicalPosition) (ecs.EntityID, error) {
	entity := manager.World.NewEntity()
	data := &EncounterData{Name: name, Position: pos}
	entity.AddComponent(EncounterComponent, data)
	return entity.GetID(), nil
}
```

### GlobalPositionSystem Usage
```go
// O(1) position lookup
entityID := common.GlobalPositionSystem.GetEntityIDAt(position)

// Add entity to position system
common.GlobalPositionSystem.AddEntity(entityID, position)

// Remove entity from position system
common.GlobalPositionSystem.RemoveEntity(entityID, position)

// Move entity
common.GlobalPositionSystem.MoveEntity(entityID, oldPos, newPos)
```

### Mode Transition Pattern
```go
// Request mode transition (queued, happens at end of frame)
if mode, exists := modeManager.GetMode("combat"); exists {
	modeManager.RequestTransition(mode, "Encounter triggered")
}

// Context switch (battle map vs overworld)
modeCoordinator.EnterBattleMap("combat")
modeCoordinator.ReturnToOverworld("exploration")
```

---

## RESOURCES

### Codebase References
- **ECS Patterns:** `docs/ecs_best_practices.md`, `CLAUDE.md`
- **Squad System:** `tactical/squads/` (reference implementation)
- **Combat System:** `tactical/combat/`, `tactical/combatservices/`
- **Position System:** `common/positionsystem.go`
- **Mode Management:** `gui/core/modemanager.go`, `gui/core/gamemodecoordinator.go`
- **Movement Input:** `input/movementcontroller.go`

### Go Best Practices
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Testing Commands
```bash
# Run all tests
go test ./...

# Run encounter package tests
go test ./tactical/encounters/... -v

# Build game
go build -o game_main/game_main.exe game_main/*.go

# Run game
./game_main/game_main.exe

# Format code
go fmt ./...

# Check for issues
go vet ./...
```

---

END OF IMPLEMENTATION PLAN
