# Approach 2: Manager-Centric Squad Architecture

**Created:** 2025-10-01
**Status:** Architectural Proposal
**Complexity:** Medium (20-30 hours)
**Philosophy:** Centralized control over distributed systems

---

## 1. Architecture Philosophy

### Core Concept
**"Manager as Authority"** - A single SquadManager maintains all squad state, grid positions, and combat logic. Squads are lightweight structs, units are ECS entities, but all tactical intelligence lives in the manager.

### TRPG Inspirations
- **Jagged Alliance**: Central command managing multiple squads
- **XCOM**: Squad management with centralized turn resolution
- **Classic RTS**: Manager pattern for unit groups

### Design Rationale

**Why Manager-Centric?**
1. **Simplicity**: All squad logic in one place, easy to debug
2. **Centralized State**: No hunting for "who owns this data?"
3. **Predictable Flow**: Procedural combat resolution
4. **Easier Testing**: Mock the manager, test everything

**Tradeoffs:**
- Less ECS-pure (manager holds state, not components)
- Manager becomes large (acceptable for tactical layer)
- Harder to parallelize (single authority)
- Easier to reason about (clear ownership)

### Architectural Principles
1. **Squad is a struct**, not an ECS entity
2. **SquadManager is the single source of truth** for tactical state
3. **Units remain ECS entities** for stats/inventory/rendering
4. **Centralized ability registry** for leader customization
5. **Manager orchestrates combat**, units provide stats

---

## 2. Data Structure Design

### 2.1 Squad Struct (Lightweight Container)

```go
// squads/squad.go
package squads

import (
	"github.com/yohamta/donburi/ecs"
	"TinkerRogue/coords"
)

// GridSlot represents a position in the 3x3 internal grid
type GridSlot struct {
	Row    int // 0-2
	Col    int // 0-2
	Unit   *ecs.Entry // nil if empty
	IsSlot bool       // false for invalid slots (e.g., 2x1 units block some slots)
}

// Squad is a lightweight container for a tactical unit
type Squad struct {
	ID           int
	Name         string
	Position     coords.LogicalPosition // Tactical map tile (single tile)
	Grid         [3][3]*GridSlot        // 3x3 internal grid
	LeaderSlot   *GridSlot              // Reference to leader's slot
	UnitCount    int                    // Active units
	IsDead       bool
	Faction      string                 // "player", "enemy", "neutral"

	// Combat state (managed by SquadManager)
	ActionsTaken bool
	MovedThisTurn bool
}

// NewSquad creates an empty squad at a position
func NewSquad(id int, name string, pos coords.LogicalPosition, faction string) *Squad {
	s := &Squad{
		ID:       id,
		Name:     name,
		Position: pos,
		Faction:  faction,
	}

	// Initialize 3x3 grid with empty slots
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			s.Grid[row][col] = &GridSlot{
				Row:    row,
				Col:    col,
				Unit:   nil,
				IsSlot: true,
			}
		}
	}

	return s
}

// AddUnit places a unit in the grid
func (s *Squad) AddUnit(unit *ecs.Entry, row, col int, isLeader bool) error {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return fmt.Errorf("invalid grid position: (%d,%d)", row, col)
	}

	slot := s.Grid[row][col]
	if !slot.IsSlot {
		return fmt.Errorf("slot (%d,%d) is blocked", row, col)
	}
	if slot.Unit != nil {
		return fmt.Errorf("slot (%d,%d) is occupied", row, col)
	}

	slot.Unit = unit
	s.UnitCount++

	if isLeader {
		s.LeaderSlot = slot
	}

	// Handle multi-tile units (2x1, 1x2)
	// Check unit size component and block additional slots if needed
	// (Implementation depends on UnitSize component)

	return nil
}

// GetLiveUnits returns all active unit entities
func (s *Squad) GetLiveUnits() []*ecs.Entry {
	units := make([]*ecs.Entry, 0, s.UnitCount)
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if s.Grid[row][col].Unit != nil {
				units = append(units, s.Grid[row][col].Unit)
			}
		}
	}
	return units
}

// GetLeader returns the leader entity, or nil if dead
func (s *Squad) GetLeader() *ecs.Entry {
	if s.LeaderSlot == nil {
		return nil
	}
	return s.LeaderSlot.Unit
}
```

---

### 2.2 SquadManager (Central Authority)

```go
// squads/squadmanager.go
package squads

import (
	"TinkerRogue/coords"
	"TinkerRogue/ecs"
)

// SquadManager is the single source of truth for tactical combat
type SquadManager struct {
	squads         map[int]*Squad              // All squads by ID
	positionIndex  map[coords.LogicalPosition]*Squad // Quick lookup by tile
	nextSquadID    int
	playerSquadIDs []int                       // Track player squads

	// Combat state
	currentTurn    int
	activeSquadID  int // Squad currently selected/acting

	// Dependencies
	entityManager  *ecs.World
	coordManager   *coords.CoordinateManager
	abilityRegistry *AbilityRegistry
}

func NewSquadManager(world *ecs.World, coordMgr *coords.CoordinateManager) *SquadManager {
	return &SquadManager{
		squads:          make(map[int]*Squad),
		positionIndex:   make(map[coords.LogicalPosition]*Squad),
		nextSquadID:     1,
		playerSquadIDs:  make([]int, 0),
		entityManager:   world,
		coordManager:    coordMgr,
		abilityRegistry: NewAbilityRegistry(),
	}
}

// CreateSquad creates and registers a new squad
func (sm *SquadManager) CreateSquad(name string, pos coords.LogicalPosition, faction string) (*Squad, error) {
	// Check position availability
	if _, occupied := sm.positionIndex[pos]; occupied {
		return nil, fmt.Errorf("position %v already occupied by squad", pos)
	}

	squad := NewSquad(sm.nextSquadID, name, pos, faction)
	sm.nextSquadID++

	sm.squads[squad.ID] = squad
	sm.positionIndex[pos] = squad

	if faction == "player" {
		sm.playerSquadIDs = append(sm.playerSquadIDs, squad.ID)
	}

	return squad, nil
}

// MoveSquad moves a squad to a new tactical position
func (sm *SquadManager) MoveSquad(squadID int, newPos coords.LogicalPosition) error {
	squad, exists := sm.squads[squadID]
	if !exists {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Check destination
	if _, occupied := sm.positionIndex[newPos]; occupied {
		return fmt.Errorf("destination %v occupied", newPos)
	}

	// Update indices
	delete(sm.positionIndex, squad.Position)
	squad.Position = newPos
	sm.positionIndex[newPos] = squad
	squad.MovedThisTurn = true

	return nil
}

// GetSquadAt returns the squad at a tactical position
func (sm *SquadManager) GetSquadAt(pos coords.LogicalPosition) *Squad {
	return sm.positionIndex[pos]
}

// GetSquad returns a squad by ID
func (sm *SquadManager) GetSquad(id int) *Squad {
	return sm.squads[id]
}

// RemoveDeadSquad removes a squad that has been eliminated
func (sm *SquadManager) RemoveDeadSquad(squadID int) {
	squad, exists := sm.squads[squadID]
	if !exists {
		return
	}

	delete(sm.positionIndex, squad.Position)
	delete(sm.squads, squadID)

	// Remove from player list if applicable
	if squad.Faction == "player" {
		for i, id := range sm.playerSquadIDs {
			if id == squadID {
				sm.playerSquadIDs = append(sm.playerSquadIDs[:i], sm.playerSquadIDs[i+1:]...)
				break
			}
		}
	}
}

// GetPlayerSquads returns all player-controlled squads
func (sm *SquadManager) GetPlayerSquads() []*Squad {
	squads := make([]*Squad, 0, len(sm.playerSquadIDs))
	for _, id := range sm.playerSquadIDs {
		if squad, exists := sm.squads[id]; exists {
			squads = append(squads, squad)
		}
	}
	return squads
}

// SetActiveSquad sets which squad is currently selected
func (sm *SquadManager) SetActiveSquad(squadID int) error {
	if _, exists := sm.squads[squadID]; !exists {
		return fmt.Errorf("squad %d not found", squadID)
	}
	sm.activeSquadID = squadID
	return nil
}

// GetActiveSquad returns the currently selected squad
func (sm *SquadManager) GetActiveSquad() *Squad {
	return sm.squads[sm.activeSquadID]
}
```

---

### 2.3 Leader vs Unit Data

```go
// squads/unitdata.go
package squads

import "github.com/yohamta/donburi/ecs"

// UnitType distinguishes leaders from regulars
type UnitType int

const (
	UnitTypeRegular UnitType = iota
	UnitTypeLeader
)

// UnitSize determines grid footprint
type UnitSize int

const (
	Size1x1 UnitSize = iota
	Size2x1
	Size1x2
)

// UnitDataComponent stores squad-specific unit information
type UnitDataComponent struct {
	Type         UnitType
	Size         UnitSize
	SquadID      int      // Which squad this unit belongs to
	GridPosition [2]int   // [row, col] in squad's 3x3 grid

	// Leader-specific
	AbilitySlots [4]string // Ability IDs (empty for regulars)
}

var UnitDataComponentTag = donburi.NewTag()

// Helper functions to add to existing entity creation
func AddUnitDataToEntity(entry *ecs.Entry, unitType UnitType, size UnitSize, squadID int) {
	data := &UnitDataComponent{
		Type:    unitType,
		Size:    size,
		SquadID: squadID,
	}

	if unitType == UnitTypeLeader {
		data.AbilitySlots = [4]string{"", "", "", ""} // 4 customizable slots
	}

	// Add component to entity
	// (Actual ECS component registration depends on your component system)
}

// GetUnitData retrieves unit data from an entity
func GetUnitData(entry *ecs.Entry) *UnitDataComponent {
	// Retrieve from ECS
	// (Implementation depends on your component access pattern)
	return nil
}
```

---

### 2.4 Centralized Ability Registry

```go
// squads/abilities.go
package squads

import (
	"fmt"
	"github.com/yohamta/donburi/ecs"
)

// Ability represents a leader's customizable skill
type Ability struct {
	ID          string
	Name        string
	Description string
	APCost      int // Action points
	TargetType  string // "self", "ally", "enemy", "aoe"

	// Execution function
	Execute func(caster *ecs.Entry, target *Squad, sm *SquadManager) error
}

// AbilityRegistry manages all available abilities
type AbilityRegistry struct {
	abilities map[string]*Ability
}

func NewAbilityRegistry() *AbilityRegistry {
	reg := &AbilityRegistry{
		abilities: make(map[string]*Ability),
	}

	// Register default abilities
	reg.registerDefaultAbilities()

	return reg
}

func (ar *AbilityRegistry) registerDefaultAbilities() {
	// Example: Rally ability (buff allies)
	ar.Register(&Ability{
		ID:          "rally",
		Name:        "Rally",
		Description: "Boost squad morale, +10% accuracy for 1 turn",
		APCost:      2,
		TargetType:  "self",
		Execute: func(caster *ecs.Entry, target *Squad, sm *SquadManager) error {
			// Apply buff to all units in caster's squad
			// (Implementation depends on buff system)
			return nil
		},
	})

	// Example: Tactical Strike (bonus damage)
	ar.Register(&Ability{
		ID:          "tactical_strike",
		Name:        "Tactical Strike",
		Description: "Coordinate attack, +50% damage this turn",
		APCost:      3,
		TargetType:  "enemy",
		Execute: func(caster *ecs.Entry, target *Squad, sm *SquadManager) error {
			// Set flag for bonus damage in next attack
			return nil
		},
	})

	// Example: Heal Squad
	ar.Register(&Ability{
		ID:          "heal_squad",
		Name:        "Heal Squad",
		Description: "Restore 20 HP to all squad members",
		APCost:      4,
		TargetType:  "self",
		Execute: func(caster *ecs.Entry, target *Squad, sm *SquadManager) error {
			// Heal all units in squad
			for _, unit := range target.GetLiveUnits() {
				// Apply healing to unit
				// (Depends on Health component)
			}
			return nil
		},
	})
}

// Register adds an ability to the registry
func (ar *AbilityRegistry) Register(ability *Ability) {
	ar.abilities[ability.ID] = ability
}

// Get retrieves an ability by ID
func (ar *AbilityRegistry) Get(id string) (*Ability, error) {
	ability, exists := ar.abilities[id]
	if !exists {
		return nil, fmt.Errorf("ability %s not found", id)
	}
	return ability, nil
}

// AssignAbilityToLeader equips an ability to a leader's slot
func (ar *AbilityRegistry) AssignAbilityToLeader(leader *ecs.Entry, abilityID string, slot int) error {
	if slot < 0 || slot > 3 {
		return fmt.Errorf("invalid slot %d (must be 0-3)", slot)
	}

	// Verify ability exists
	if _, err := ar.Get(abilityID); err != nil {
		return err
	}

	// Update leader's UnitDataComponent
	data := GetUnitData(leader)
	if data.Type != UnitTypeLeader {
		return fmt.Errorf("only leaders can equip abilities")
	}

	data.AbilitySlots[slot] = abilityID
	return nil
}

// ExecuteAbility runs a leader's ability
func (ar *AbilityRegistry) ExecuteAbility(leader *ecs.Entry, slot int, target *Squad, sm *SquadManager) error {
	data := GetUnitData(leader)
	if data.Type != UnitTypeLeader {
		return fmt.Errorf("only leaders can use abilities")
	}

	abilityID := data.AbilitySlots[slot]
	if abilityID == "" {
		return fmt.Errorf("no ability in slot %d", slot)
	}

	ability, err := ar.Get(abilityID)
	if err != nil {
		return err
	}

	// Check AP cost (if you have action point system)
	// Execute ability
	return ability.Execute(leader, target, sm)
}
```

---

## 3. Combat Flow (Simultaneous Squad Combat)

### 3.1 Attack Sequence

```go
// squads/combat.go
package squads

import (
	"TinkerRogue/combat"
	"github.com/yohamta/donburi/ecs"
)

// SquadCombatResult holds the outcome of squad-vs-squad combat
type SquadCombatResult struct {
	AttackerDamageDealt int
	DefenderDamageDealt int
	AttackerCasualties  []*ecs.Entry
	DefenderCasualties  []*ecs.Entry
	AttackerWins        bool
}

// ExecuteSquadCombat resolves combat between two squads
func (sm *SquadManager) ExecuteSquadCombat(attackerID, defenderID int) (*SquadCombatResult, error) {
	attacker := sm.squads[attackerID]
	defender := sm.squads[defenderID]

	if attacker == nil || defender == nil {
		return nil, fmt.Errorf("invalid squad IDs")
	}

	result := &SquadCombatResult{}

	// PHASE 1: Pool all attacks
	attackerUnits := attacker.GetLiveUnits()
	defenderUnits := defender.GetLiveUnits()

	attackerDamage := 0
	defenderDamage := 0

	// Attacker units attack
	for _, unit := range attackerUnits {
		// Pick random defender target
		if len(defenderUnits) == 0 {
			break
		}
		target := defenderUnits[rand.Intn(len(defenderUnits))]

		// Use existing PerformAttack() but don't apply damage yet
		damage := sm.calculateUnitDamage(unit, target)
		attackerDamage += damage
	}

	// Defender units counter-attack (simultaneous)
	for _, unit := range defenderUnits {
		if len(attackerUnits) == 0 {
			break
		}
		target := attackerUnits[rand.Intn(len(attackerUnits))]

		damage := sm.calculateUnitDamage(unit, target)
		defenderDamage += damage
	}

	// PHASE 2: Apply pooled damage
	result.AttackerDamageDealt = attackerDamage
	result.DefenderDamageDealt = defenderDamage

	// Distribute damage across defender units
	result.DefenderCasualties = sm.applyPooledDamage(defender, attackerDamage)

	// Distribute damage across attacker units (counter-attack)
	result.AttackerCasualties = sm.applyPooledDamage(attacker, defenderDamage)

	// PHASE 3: Check squad death
	sm.checkSquadDeath(attacker)
	sm.checkSquadDeath(defender)

	result.AttackerWins = defender.IsDead

	return result, nil
}

// calculateUnitDamage uses existing PerformAttack logic but extracts damage only
func (sm *SquadManager) calculateUnitDamage(attacker, defender *ecs.Entry) int {
	// Wrap existing combat.PerformAttack() to get damage value
	// This preserves existing d20 mechanics, damage calculation, etc.

	// Pseudo-code (actual implementation depends on PerformAttack signature):
	// damageInfo := combat.PerformAttack(attacker, defender, applyDamage=false)
	// return damageInfo.Damage

	return 10 // Placeholder
}

// applyPooledDamage distributes damage across squad units
func (sm *SquadManager) applyPooledDamage(squad *Squad, totalDamage int) []*ecs.Entry {
	casualties := make([]*ecs.Entry, 0)
	units := squad.GetLiveUnits()

	if len(units) == 0 {
		return casualties
	}

	// Strategy: Damage from front to back (row 0 -> row 2)
	// This creates positional tactics (protect back row)

	remainingDamage := totalDamage

	for row := 0; row < 3 && remainingDamage > 0; row++ {
		for col := 0; col < 3 && remainingDamage > 0; col++ {
			slot := squad.Grid[row][col]
			if slot.Unit == nil {
				continue
			}

			unit := slot.Unit
			currentHP := getUnitHP(unit)

			if currentHP <= remainingDamage {
				// Unit dies
				remainingDamage -= currentHP
				setUnitHP(unit, 0)
				casualties = append(casualties, unit)

				// Remove from grid
				slot.Unit = nil
				squad.UnitCount--
			} else {
				// Unit survives with reduced HP
				setUnitHP(unit, currentHP-remainingDamage)
				remainingDamage = 0
			}
		}
	}

	return casualties
}

// checkSquadDeath marks squad as dead if all units eliminated
func (sm *SquadManager) checkSquadDeath(squad *Squad) {
	if squad.UnitCount == 0 {
		squad.IsDead = true
		sm.RemoveDeadSquad(squad.ID)
	}
}

// Helper functions (implementation depends on Health component)
func getUnitHP(unit *ecs.Entry) int {
	// Get HP from Health component
	return 100 // Placeholder
}

func setUnitHP(unit *ecs.Entry, hp int) {
	// Set HP on Health component
}
```

---

### 3.2 Combat Flow Diagram

```
Player selects attacker squad
          ↓
Player selects target squad
          ↓
SquadManager.ExecuteSquadCombat()
          ↓
┌─────────────────────────────────┐
│ PHASE 1: Pool Attacks           │
│  - Each attacker unit attacks   │
│  - Each defender unit counters  │
│  - Track total damage pools     │
└─────────────────────────────────┘
          ↓
┌─────────────────────────────────┐
│ PHASE 2: Apply Pooled Damage    │
│  - Distribute to defender front │
│  - Distribute to attacker front │
│  - Track casualties             │
└─────────────────────────────────┘
          ↓
┌─────────────────────────────────┐
│ PHASE 3: Check Squad Death      │
│  - Remove dead squads           │
│  - Update position index        │
└─────────────────────────────────┘
          ↓
Return SquadCombatResult
```

---

## 4. 3x3 Grid Mechanics

### 4.1 Grid Position System

**Coordinate Layers:**
1. **Tactical Map**: Squad occupies 1 tile (e.g., LogicalPosition{10, 5})
2. **Internal Grid**: 3x3 grid for unit placement (row 0-2, col 0-2)
3. **Unit Entity**: Still has LogicalPosition component (for pathfinding, rendering)

**Position Synchronization:**
```go
// When squad moves, update all unit entity positions
func (sm *SquadManager) updateUnitPositions(squad *Squad) {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			slot := squad.Grid[row][col]
			if slot.Unit == nil {
				continue
			}

			// Update unit's LogicalPosition component to match squad
			// (Units render at squad position, grid is internal only)
			updateEntityPosition(slot.Unit, squad.Position)
		}
	}
}
```

### 4.2 Multi-Size Unit Placement

```go
// PlaceUnitInGrid handles 1x1, 2x1, 1x2 sizes
func (s *Squad) PlaceUnitInGrid(unit *ecs.Entry, row, col int) error {
	unitData := GetUnitData(unit)

	switch unitData.Size {
	case Size1x1:
		// Single slot
		if err := s.AddUnit(unit, row, col, unitData.Type == UnitTypeLeader); err != nil {
			return err
		}

	case Size2x1:
		// Horizontal 2-tile
		if col+1 > 2 {
			return fmt.Errorf("2x1 unit doesn't fit at col %d", col)
		}

		// Occupy both slots
		s.Grid[row][col].Unit = unit
		s.Grid[row][col].IsSlot = true

		s.Grid[row][col+1].Unit = unit // Same unit reference
		s.Grid[row][col+1].IsSlot = false // Blocked slot

		s.UnitCount++

	case Size1x2:
		// Vertical 2-tile
		if row+1 > 2 {
			return fmt.Errorf("1x2 unit doesn't fit at row %d", row)
		}

		s.Grid[row][col].Unit = unit
		s.Grid[row][col].IsSlot = true

		s.Grid[row+1][col].Unit = unit
		s.Grid[row+1][col].IsSlot = false

		s.UnitCount++
	}

	unitData.GridPosition = [2]int{row, col}
	return nil
}
```

### 4.3 Front-to-Back Damage Distribution

**Tactical Implication:** Row position matters for survival.

```
Row 0 (Front):  [Tank] [Fighter] [Empty]   <- Takes damage first
Row 1 (Mid):    [Archer] [Empty] [Mage]    <- Protected by front
Row 2 (Back):   [Healer] [Empty] [Leader]  <- Safest position
```

**Combat Example:**
- Enemy squad deals 150 damage
- Front Tank (100 HP) dies, 50 damage remaining
- Front Fighter (80 HP) takes 50 damage, survives with 30 HP
- Mid/Back rows untouched

This creates **positional tactics**: place fragile units in back, tanks in front.

---

## 5. Leader Ability System

### 5.1 Ability Swap Flow

```go
// Example: Player customizes leader abilities
func ExampleLeaderCustomization(sm *SquadManager) {
	// Get player's first squad
	squad := sm.GetPlayerSquads()[0]
	leader := squad.GetLeader()

	if leader == nil {
		fmt.Println("No leader in squad")
		return
	}

	// Assign abilities to slots
	sm.abilityRegistry.AssignAbilityToLeader(leader, "rally", 0)
	sm.abilityRegistry.AssignAbilityToLeader(leader, "tactical_strike", 1)
	sm.abilityRegistry.AssignAbilityToLeader(leader, "heal_squad", 2)
	// Slot 3 left empty

	fmt.Println("Leader abilities configured:")
	data := GetUnitData(leader)
	for i, abilityID := range data.AbilitySlots {
		if abilityID == "" {
			fmt.Printf("Slot %d: Empty\n", i)
		} else {
			ability, _ := sm.abilityRegistry.Get(abilityID)
			fmt.Printf("Slot %d: %s (%s)\n", i, ability.Name, ability.Description)
		}
	}
}
```

### 5.2 Ability Execution in Combat

```go
// Player uses leader ability mid-combat
func ExampleUseAbility(sm *SquadManager) {
	squad := sm.GetActiveSquad()
	leader := squad.GetLeader()

	// Use ability in slot 1 (Tactical Strike)
	targetSquad := sm.GetSquadAt(coords.LogicalPosition{X: 10, Y: 5})

	err := sm.abilityRegistry.ExecuteAbility(leader, 1, targetSquad, sm)
	if err != nil {
		fmt.Printf("Ability failed: %v\n", err)
		return
	}

	fmt.Println("Tactical Strike activated! Next attack deals +50% damage")

	// Proceed with normal squad combat
	result, _ := sm.ExecuteSquadCombat(squad.ID, targetSquad.ID)
	fmt.Printf("Combat result: %d damage dealt\n", result.AttackerDamageDealt)
}
```

### 5.3 Regular Units (No Abilities)

```go
// Regular units have fixed behavior, no customization
func CreateRegularUnit(template string, squadID int) *ecs.Entry {
	// Use existing CreateEntityFromTemplate
	unit := entitytemplates.CreateEntityFromTemplate(template)

	// Add unit data (no ability slots for regulars)
	AddUnitDataToEntity(unit, UnitTypeRegular, Size1x1, squadID)

	return unit
}
```

---

## 6. Integration Points

### 6.1 EntityManager Integration

**Principle:** SquadManager tracks squads, EntityManager tracks entities. Clear ownership.

```go
// squads/integration.go

// CreateSquadWithUnits combines squad creation with entity spawning
func (sm *SquadManager) CreateSquadWithUnits(
	name string,
	pos coords.LogicalPosition,
	faction string,
	leaderTemplate string,
	regularTemplates []string,
) (*Squad, error) {
	// Create squad struct
	squad, err := sm.CreateSquad(name, pos, faction)
	if err != nil {
		return nil, err
	}

	// Spawn leader entity
	leaderEntity := entitytemplates.CreateEntityFromTemplate(leaderTemplate)
	AddUnitDataToEntity(leaderEntity, UnitTypeLeader, Size1x1, squad.ID)

	// Place leader at back center (row 2, col 1)
	squad.PlaceUnitInGrid(leaderEntity, 2, 1)

	// Spawn regular units
	positions := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}} // Front/mid positions
	for i, template := range regularTemplates {
		if i >= len(positions) {
			break
		}

		unit := entitytemplates.CreateEntityFromTemplate(template)
		AddUnitDataToEntity(unit, UnitTypeRegular, Size1x1, squad.ID)

		pos := positions[i]
		squad.PlaceUnitInGrid(unit, pos[0], pos[1])
	}

	return squad, nil
}

// CleanupDeadSquads removes entities when squad dies
func (sm *SquadManager) CleanupDeadSquads() {
	for id, squad := range sm.squads {
		if squad.IsDead {
			// Remove all unit entities from ECS
			for row := 0; row < 3; row++ {
				for col := 0; col < 3; col++ {
					if unit := squad.Grid[row][col].Unit; unit != nil {
						sm.entityManager.RemoveEntity(unit)
					}
				}
			}

			// Remove squad from manager
			sm.RemoveDeadSquad(id)
		}
	}
}
```

---

### 6.2 Input System Integration

**Pattern:** Input controllers call SquadManager methods.

```go
// controller/combatcontroller.go (existing file)

type CombatController struct {
	squadManager *squads.SquadManager
	// ... existing fields
}

func (cc *CombatController) HandleSquadAttack(attackerID, targetPos coords.LogicalPosition) error {
	// Get target squad at position
	targetSquad := cc.squadManager.GetSquadAt(targetPos)
	if targetSquad == nil {
		return fmt.Errorf("no squad at target position")
	}

	// Execute squad combat
	result, err := cc.squadManager.ExecuteSquadCombat(attackerID, targetSquad.ID)
	if err != nil {
		return err
	}

	// Display combat result in UI
	cc.displayCombatResult(result)

	return nil
}

func (cc *CombatController) HandleSquadMove(squadID int, newPos coords.LogicalPosition) error {
	return cc.squadManager.MoveSquad(squadID, newPos)
}
```

---

### 6.3 PerformAttack() Preservation

**Strategy:** Wrap existing combat logic, don't replace it.

```go
// combat/attacking.go (existing file)

// Existing function signature (UNCHANGED)
func PerformAttack(attacker, defender *ecs.Entry) {
	// ... existing d20 logic, damage calculation, etc.
}

// New wrapper for squad combat (extracts damage without applying)
func CalculateAttackDamage(attacker, defender *ecs.Entry) int {
	// Clone defender HP to prevent modification
	originalHP := getHP(defender)

	// Run attack
	PerformAttack(attacker, defender)

	// Calculate damage dealt
	newHP := getHP(defender)
	damage := originalHP - newHP

	// Restore HP (we're only calculating, not applying)
	setHP(defender, originalHP)

	return damage
}
```

**Alternative (Less Invasive):** Add optional parameter to PerformAttack:
```go
func PerformAttack(attacker, defender *ecs.Entry, applyDamage bool) int {
	// ... existing logic

	damage := calculateDamage(attacker, defender)

	if applyDamage {
		applyDamageToEntity(defender, damage)
	}

	return damage
}
```

---

### 6.4 Spawning System Integration

**Pattern:** Spawning creates squads instead of individual entities.

```go
// spawning/spawnmonsters.go (existing file)

func SpawnEnemySquad(pos coords.LogicalPosition, difficulty int, sm *squads.SquadManager) {
	// Choose squad composition based on difficulty
	leaderTemplate := "enemy_captain"
	regularTemplates := []string{
		"enemy_soldier",
		"enemy_soldier",
		"enemy_archer",
	}

	// Create squad with units
	squad, err := sm.CreateSquadWithUnits(
		"Enemy Patrol",
		pos,
		"enemy",
		leaderTemplate,
		regularTemplates,
	)

	if err != nil {
		fmt.Printf("Failed to spawn squad: %v\n", err)
		return
	}

	// Customize leader abilities
	leader := squad.GetLeader()
	sm.abilityRegistry.AssignAbilityToLeader(leader, "rally", 0)
	sm.abilityRegistry.AssignAbilityToLeader(leader, "tactical_strike", 1)
}
```

---

## 7. Pros & Cons Analysis

### 7.1 Advantages

**1. Simplicity and Clarity**
- All tactical logic in one place (SquadManager)
- Easy to debug: check manager state
- Clear ownership: manager owns squads, EntityManager owns entities

**2. Centralized Control**
- Single source of truth for combat state
- No "which component holds this?" questions
- Predictable execution flow

**3. Easy Testing**
- Mock SquadManager, test combat logic
- No complex component interactions to mock
- Clear input/output for functions

**4. Familiar Pattern**
- Classic game architecture (RTS-style)
- Lower learning curve for new developers
- Aligns with procedural combat resolution

**5. Performance (for small-medium scale)**
- No component lookup overhead
- Direct struct access
- Efficient position indexing

---

### 7.2 Disadvantages

**1. Manager Becomes Large**
- SquadManager holds all tactical logic
- ~500-800 LOC once fully implemented
- Risk of "god object" anti-pattern

**Mitigation:** Split into multiple files:
```
squads/
  squadmanager.go       (core state)
  combat.go             (combat logic)
  movement.go           (movement/positioning)
  abilities.go          (ability system)
  integration.go        (ECS integration)
```

**2. Less ECS-Pure**
- Squads are structs, not entities
- Manager holds state outside component system
- Breaks "composition over inheritance" dogma

**Mitigation:** Accept pragmatic tradeoff. Tactical layer needs different architecture than entity simulation.

**3. Harder to Parallelize**
- Manager is single authority
- Concurrent squad combat requires locking
- No component-based parallelism

**Mitigation:** Not a concern for turn-based game (sequential by design).

**4. Testing Manager Requires Full Setup**
- Can't test combat in isolation easily
- Need to create manager, squads, units
- More setup code for tests

**Mitigation:** Provide test helpers:
```go
func CreateTestSquadManager() *SquadManager { ... }
func CreateTestSquad(name string) *Squad { ... }
```

---

### 7.3 Complexity Assessment

**Implementation Effort:** 20-30 hours

**Breakdown:**
- Core structs (Squad, SquadManager): 4 hours
- Combat system (pooled damage): 6 hours
- Grid mechanics (3x3 positioning): 4 hours
- Ability registry: 4 hours
- Integration (Input, spawning): 6 hours
- Testing and debugging: 6 hours

**Lines of Code Estimate:** ~1200 LOC
- squadmanager.go: 300 LOC
- squad.go: 200 LOC
- combat.go: 250 LOC
- abilities.go: 200 LOC
- integration.go: 150 LOC
- unitdata.go: 100 LOC

**Maintainability:** Medium
- Manager size risk (mitigated by file splitting)
- Clear flow aids debugging
- Centralized changes (good for features, bad for flexibility)

**Performance:** Good for <100 squads
- O(1) position lookup (map indexing)
- O(n) squad iteration (acceptable)
- No component query overhead

---

## 8. Code Examples

### 8.1 Complete Example: Creating and Fighting with Squads

```go
package main

import (
	"fmt"
	"TinkerRogue/squads"
	"TinkerRogue/coords"
)

func main() {
	// Initialize manager
	world := ecs.NewWorld()
	coordMgr := coords.NewCoordinateManager(tileSize, mapWidth, mapHeight)
	squadMgr := squads.NewSquadManager(world, coordMgr)

	// === CREATE PLAYER SQUAD ===
	playerSquad, _ := squadMgr.CreateSquadWithUnits(
		"Alpha Squad",
		coords.LogicalPosition{X: 5, Y: 5},
		"player",
		"player_leader", // Leader template
		[]string{
			"player_soldier",
			"player_soldier",
			"player_archer",
			"player_mage",
		},
	)

	// Customize leader abilities
	leader := playerSquad.GetLeader()
	squadMgr.abilityRegistry.AssignAbilityToLeader(leader, "rally", 0)
	squadMgr.abilityRegistry.AssignAbilityToLeader(leader, "heal_squad", 1)
	squadMgr.abilityRegistry.AssignAbilityToLeader(leader, "tactical_strike", 2)

	// === CREATE ENEMY SQUAD ===
	enemySquad, _ := squadMgr.CreateSquadWithUnits(
		"Orc Warband",
		coords.LogicalPosition{X: 10, Y: 5},
		"enemy",
		"orc_chieftain",
		[]string{
			"orc_warrior",
			"orc_warrior",
			"orc_warrior",
		},
	)

	// === PLAYER TURN: USE ABILITY ===
	fmt.Println("Player uses Rally ability...")
	squadMgr.abilityRegistry.ExecuteAbility(leader, 0, playerSquad, squadMgr)

	// === EXECUTE COMBAT ===
	fmt.Println("\nAlpha Squad attacks Orc Warband!")
	result, err := squadMgr.ExecuteSquadCombat(playerSquad.ID, enemySquad.ID)
	if err != nil {
		fmt.Printf("Combat failed: %v\n", err)
		return
	}

	// === DISPLAY RESULTS ===
	fmt.Printf("\nCombat Results:\n")
	fmt.Printf("  Alpha Squad dealt %d damage\n", result.AttackerDamageDealt)
	fmt.Printf("  Orc Warband dealt %d damage\n", result.DefenderDamageDealt)
	fmt.Printf("  Alpha Squad casualties: %d\n", len(result.AttackerCasualties))
	fmt.Printf("  Orc Warband casualties: %d\n", len(result.DefenderCasualties))

	if result.AttackerWins {
		fmt.Println("\n  VICTORY! Orc Warband eliminated!")
	} else if enemySquad.UnitCount == 0 {
		fmt.Println("\n  DEFEAT! Alpha Squad eliminated!")
	} else {
		fmt.Println("\n  Both squads survived!")
	}

	// === MOVE SQUAD ===
	if !playerSquad.IsDead {
		fmt.Println("\nAlpha Squad advances...")
		squadMgr.MoveSquad(playerSquad.ID, coords.LogicalPosition{X: 6, Y: 5})
	}
}
```

**Output:**
```
Player uses Rally ability...
Rally activated! Squad gains +10% accuracy for 1 turn.

Alpha Squad attacks Orc Warband!

Combat Results:
  Alpha Squad dealt 185 damage
  Orc Warband dealt 120 damage
  Alpha Squad casualties: 1
  Orc Warband casualties: 2

  Both squads survived!

Alpha Squad advances...
```

---

### 8.2 Complete Example: Grid Visualization

```go
func PrintSquadGrid(squad *squads.Squad) {
	fmt.Printf("\n=== %s ===\n", squad.Name)
	fmt.Printf("Position: %v, Faction: %s\n", squad.Position, squad.Faction)
	fmt.Printf("Unit Count: %d\n\n", squad.UnitCount)

	for row := 0; row < 3; row++ {
		fmt.Printf("Row %d: ", row)
		for col := 0; col < 3; col++ {
			slot := squad.Grid[row][col]

			if !slot.IsSlot {
				fmt.Print("[BLOCKED] ")
				continue
			}

			if slot.Unit == nil {
				fmt.Print("[  EMPTY  ] ")
				continue
			}

			// Get unit info
			data := squads.GetUnitData(slot.Unit)
			unitType := "Regular"
			if data.Type == squads.UnitTypeLeader {
				unitType = "LEADER"
			}

			hp := getUnitHP(slot.Unit)
			fmt.Printf("[ %s %dHP ] ", unitType, hp)
		}
		fmt.Println()
	}
}

// Example output:
/*
=== Alpha Squad ===
Position: {5 5}, Faction: player
Unit Count: 5

Row 0: [ Regular 80HP ] [ Regular 80HP ] [ Regular 60HP ]
Row 1: [  EMPTY  ] [  EMPTY  ] [ Regular 50HP ]
Row 2: [  EMPTY  ] [ LEADER 100HP ] [  EMPTY  ]
*/
```

---

### 8.3 Complete Example: Leader Ability Swap UI

```go
// GUI function for ability customization
func ShowLeaderAbilityMenu(leader *ecs.Entry, registry *squads.AbilityRegistry) {
	data := squads.GetUnitData(leader)

	fmt.Println("\n=== Leader Ability Customization ===")
	fmt.Println("Current Abilities:")
	for i, abilityID := range data.AbilitySlots {
		if abilityID == "" {
			fmt.Printf("  Slot %d: Empty\n", i)
		} else {
			ability, _ := registry.Get(abilityID)
			fmt.Printf("  Slot %d: %s (AP: %d)\n", i, ability.Name, ability.APCost)
		}
	}

	fmt.Println("\nAvailable Abilities:")
	allAbilities := []string{"rally", "tactical_strike", "heal_squad"}
	for i, id := range allAbilities {
		ability, _ := registry.Get(id)
		fmt.Printf("  %d. %s - %s (AP: %d)\n", i+1, ability.Name, ability.Description, ability.APCost)
	}

	// Simulate player input
	fmt.Print("\nSelect slot to modify (0-3): ")
	var slot int
	fmt.Scanln(&slot)

	fmt.Print("Select ability (1-3, 0 to remove): ")
	var choice int
	fmt.Scanln(&choice)

	if choice == 0 {
		// Remove ability
		data.AbilitySlots[slot] = ""
		fmt.Println("Ability removed.")
	} else if choice >= 1 && choice <= len(allAbilities) {
		// Assign ability
		abilityID := allAbilities[choice-1]
		registry.AssignAbilityToLeader(leader, abilityID, slot)
		ability, _ := registry.Get(abilityID)
		fmt.Printf("Equipped %s to slot %d.\n", ability.Name, slot)
	}
}
```

---

## 9. Migration Strategy

### 9.1 Phased Implementation

**Phase 1: Foundation (8 hours)**
- Create Squad struct, SquadManager
- Implement CreateSquad, MoveSquad, position indexing
- Basic 3x3 grid with 1x1 units only
- No combat yet, just squad creation and movement

**Phase 2: Combat (8 hours)**
- Implement ExecuteSquadCombat with pooled damage
- Integrate with PerformAttack() wrapper
- Front-to-back damage distribution
- Test squad-vs-squad combat

**Phase 3: Abilities (6 hours)**
- Create AbilityRegistry
- Implement 3-4 basic abilities
- Leader ability assignment and execution
- Test ability effects in combat

**Phase 4: Integration (8 hours)**
- Connect to Input controllers
- Update spawning system
- Add UI for squad selection and combat
- Polish and balance testing

---

### 9.2 Backward Compatibility

**Existing 1v1 combat can coexist:**
```go
// Old code (still works)
combat.PerformAttack(playerEntity, enemyEntity)

// New code (squad combat)
squadMgr.ExecuteSquadCombat(playerSquadID, enemySquadID)
```

**Migration path:**
1. Implement squad system alongside existing combat
2. Test thoroughly in isolated maps
3. Gradually convert spawning to use squads
4. Eventually deprecate direct entity combat
5. Remove old combat once squads are stable

---

## 10. Testing Strategy

### 10.1 Unit Tests

```go
// squads/squadmanager_test.go

func TestCreateSquad(t *testing.T) {
	sm := NewSquadManager(mockWorld, mockCoordMgr)

	squad, err := sm.CreateSquad("Test Squad", LogicalPosition{5, 5}, "player")
	if err != nil {
		t.Fatalf("Failed to create squad: %v", err)
	}

	if squad.Name != "Test Squad" {
		t.Errorf("Expected name 'Test Squad', got '%s'", squad.Name)
	}

	if squad.UnitCount != 0 {
		t.Errorf("New squad should have 0 units, got %d", squad.UnitCount)
	}
}

func TestSquadCombat(t *testing.T) {
	sm := createTestSquadManager()

	// Create two squads with units
	attacker := createTestSquadWithUnits(sm, "Attacker", 3)
	defender := createTestSquadWithUnits(sm, "Defender", 3)

	// Execute combat
	result, err := sm.ExecuteSquadCombat(attacker.ID, defender.ID)
	if err != nil {
		t.Fatalf("Combat failed: %v", err)
	}

	// Verify damage was dealt
	if result.AttackerDamageDealt == 0 {
		t.Error("Attacker should have dealt damage")
	}

	// Verify casualties
	totalUnits := attacker.UnitCount + defender.UnitCount
	if totalUnits == 6 {
		t.Error("Combat should have caused casualties")
	}
}
```

---

### 10.2 Integration Tests

```go
func TestSpawningIntegration(t *testing.T) {
	sm := NewSquadManager(realWorld, realCoordMgr)

	// Use real spawning system
	SpawnEnemySquad(LogicalPosition{10, 5}, 5, sm)

	squad := sm.GetSquadAt(LogicalPosition{10, 5})
	if squad == nil {
		t.Fatal("Spawning failed to create squad")
	}

	if squad.UnitCount == 0 {
		t.Error("Spawned squad has no units")
	}

	if squad.GetLeader() == nil {
		t.Error("Spawned squad has no leader")
	}
}

func TestInputIntegration(t *testing.T) {
	sm := NewSquadManager(realWorld, realCoordMgr)
	controller := NewCombatController(sm)

	// Create squads
	attacker, _ := sm.CreateSquadWithUnits("Player", LogicalPosition{5, 5}, "player", "leader", []string{"soldier"})
	defender, _ := sm.CreateSquadWithUnits("Enemy", LogicalPosition{6, 5}, "enemy", "orc_chief", []string{"orc"})

	// Simulate player input
	err := controller.HandleSquadAttack(attacker.ID, LogicalPosition{6, 5})
	if err != nil {
		t.Errorf("Input handling failed: %v", err)
	}
}
```

---

### 10.3 Balance Testing

```go
func TestCombatBalance(t *testing.T) {
	sm := createTestSquadManager()

	// Run 100 combats with equal squads
	attackerWins := 0
	for i := 0; i < 100; i++ {
		attacker := createBalancedSquad(sm, "Attacker")
		defender := createBalancedSquad(sm, "Defender")

		result, _ := sm.ExecuteSquadCombat(attacker.ID, defender.ID)
		if result.AttackerWins {
			attackerWins++
		}
	}

	// Equal squads should have ~50% win rate (allow 40-60% variance)
	if attackerWins < 40 || attackerWins > 60 {
		t.Errorf("Balance issue: attacker won %d/100 times (expected ~50)", attackerWins)
	}
}
```

---

## 11. Summary

### Quick Reference

**Architecture:** Manager-Centric
**Squad Entity:** Struct (not ECS entity)
**Units:** ECS entities
**Authority:** SquadManager
**Combat:** Simultaneous pooled damage
**Grid:** 3x3 internal positioning
**Abilities:** Centralized registry

### When to Use This Approach

**Best For:**
- Teams familiar with OOP/procedural design
- Projects prioritizing simplicity over ECS purity
- Turn-based games (no parallelism needed)
- Small-medium scale (< 100 squads)

**Avoid If:**
- Need real-time squad updates
- Want pure ECS architecture
- Require heavy parallelization
- Manager size concerns outweigh simplicity benefits

### Next Steps

1. **Review this document** with team/stakeholders
2. **Decide on approach** (compare with Approach 1 if available)
3. **Create implementation plan** with detailed task breakdown
4. **Set up test environment** (mock dependencies)
5. **Begin Phase 1** (foundation: squads, manager, movement)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-01
**Ready for Implementation:** Yes
