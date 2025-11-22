# Go Testing Patterns Skill

**Purpose**: Generate idiomatic Go test patterns for game code
**Trigger**: When creating test files or discussing testing strategies

## Capabilities

- Table-driven test templates
- Benchmark patterns for performance-critical code
- Mock/stub patterns for ECS systems
- Test helper function suggestions
- Coverage gap identification

## Core Testing Patterns

### 1. Table-Driven Tests

**Standard Pattern** (from `squads/squads_test.go`):
```go
func TestCreateSquad(t *testing.T) {
    tests := []struct {
        name          string
        squadName     string
        formationID   int
        expectedError bool
    }{
        {
            name:          "Valid squad creation",
            squadName:     "Alpha Squad",
            formationID:   1,
            expectedError: false,
        },
        {
            name:          "Empty squad name",
            squadName:     "",
            formationID:   1,
            expectedError: true,
        },
        {
            name:          "Invalid formation ID",
            squadName:     "Bravo Squad",
            formationID:   -1,
            expectedError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            manager := ecs.NewManager()
            squadID := CreateSquad(manager, tt.squadName, tt.formationID)

            if tt.expectedError {
                if squadID != 0 {
                    t.Errorf("Expected error but got valid squad ID: %d", squadID)
                }
            } else {
                if squadID == 0 {
                    t.Error("Expected valid squad ID but got 0")
                }
            }
        })
    }
}
```

**Key Elements**:
- Anonymous struct slice for test cases
- Descriptive `name` field for each test
- `t.Run()` for subtests (parallel execution, better error messages)
- Clear expected vs actual comparisons

### 2. Subtests with t.Run()

**Pattern**:
```go
func TestCombatSystem(t *testing.T) {
    t.Run("Hit calculation", func(t *testing.T) {
        // Test hit chance formulas
        attacker := &CombatStats{Attack: 20, CritRate: 0.25}
        defender := &CombatStats{Defense: 5, Dodge: 0.10}

        hitChance := CalculateHitChance(attacker, defender)
        if hitChance < 0 || hitChance > 1 {
            t.Errorf("Hit chance out of range: %f", hitChance)
        }
    })

    t.Run("Damage calculation", func(t *testing.T) {
        // Test damage formulas
        attacker := &CombatStats{Attack: 20}
        defender := &CombatStats{Defense: 5}

        damage := CalculateDamage(attacker, defender)
        if damage < 1 {
            t.Error("Damage should have minimum value of 1")
        }
    })

    t.Run("Critical hits", func(t *testing.T) {
        // Test crit mechanics
        // Run multiple times to verify probability
        critCount := 0
        iterations := 1000

        for i := 0; i < iterations; i++ {
            if IsCriticalHit(0.25) {
                critCount++
            }
        }

        // Should be ~25% (with tolerance)
        critRate := float64(critCount) / float64(iterations)
        if critRate < 0.20 || critRate > 0.30 {
            t.Errorf("Crit rate outside tolerance: %f (expected ~0.25)", critRate)
        }
    })
}
```

**Benefits**:
- Isolates test failures (one subtest failing doesn't skip others)
- Better test output (shows which subtest failed)
- Can run specific subtests: `go test -run TestCombatSystem/Hit`

### 3. Edge Case Testing

**Common Edge Cases for Game Code**:
```go
func TestInventoryEdgeCases(t *testing.T) {
    manager := ecs.NewManager()

    t.Run("nil checks", func(t *testing.T) {
        // Nil manager
        result := AddItem(nil, &Inventory{}, 123)
        if result != nil {
            t.Error("Should handle nil manager gracefully")
        }

        // Nil inventory
        result = AddItem(manager, nil, 123)
        if result != nil {
            t.Error("Should handle nil inventory gracefully")
        }
    })

    t.Run("empty inputs", func(t *testing.T) {
        inv := &Inventory{ItemEntityIDs: []ecs.EntityID{}}

        // Remove from empty inventory
        err := RemoveItem(manager, inv, 999)
        if err == nil {
            t.Error("Should error when removing from empty inventory")
        }
    })

    t.Run("boundary values", func(t *testing.T) {
        inv := &Inventory{ItemEntityIDs: make([]ecs.EntityID, 0, 100)}

        // Fill to capacity
        for i := 0; i < 100; i++ {
            AddItem(manager, inv, ecs.EntityID(i))
        }

        // Try adding beyond capacity
        err := AddItem(manager, inv, 999)
        if err == nil {
            t.Error("Should error when inventory full")
        }
    })

    t.Run("invalid IDs", func(t *testing.T) {
        inv := &Inventory{}

        // Invalid entity ID
        err := AddItem(manager, inv, ecs.EntityID(0))
        if err == nil {
            t.Error("Should reject invalid entity ID")
        }

        // Non-existent entity
        err = AddItem(manager, inv, ecs.EntityID(99999))
        if err == nil {
            t.Error("Should error for non-existent entity")
        }
    })
}
```

### 4. Error Path Testing

**Pattern**:
```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        setup       func(*ecs.Manager) (squadID ecs.EntityID)
        operation   func(*ecs.Manager, ecs.EntityID) error
        expectError bool
        errorMsg    string
    }{
        {
            name: "Delete non-existent squad",
            setup: func(m *ecs.Manager) ecs.EntityID {
                return 0  // Invalid ID
            },
            operation: func(m *ecs.Manager, id ecs.EntityID) error {
                return DeleteSquad(m, id)
            },
            expectError: true,
            errorMsg:    "squad not found",
        },
        {
            name: "Delete squad with members",
            setup: func(m *ecs.Manager) ecs.EntityID {
                squadID := CreateSquad(m, "Test", 1)
                CreateSquadMember(m, squadID)
                return squadID
            },
            operation: func(m *ecs.Manager, id ecs.EntityID) error {
                return DeleteSquad(m, id)
            },
            expectError: false,  // Should cascade delete members
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            manager := ecs.NewManager()
            squadID := tt.setup(manager)

            err := tt.operation(manager, squadID)

            if tt.expectError {
                if err == nil {
                    t.Error("Expected error but got nil")
                }
                if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
                    t.Errorf("Expected error message '%s' but got '%s'", tt.errorMsg, err.Error())
                }
            } else {
                if err != nil {
                    t.Errorf("Expected no error but got: %v", err)
                }
            }
        })
    }
}
```

### 5. Benchmarks for Performance-Critical Code

**Pattern**:
```go
func BenchmarkGetSquadMembers(b *testing.B) {
    manager := ecs.NewManager()

    // Setup: Create squads with members
    for i := 0; i < 10; i++ {
        squadID := CreateSquad(manager, fmt.Sprintf("Squad%d", i), 1)
        for j := 0; j < 5; j++ {
            CreateSquadMember(manager, squadID)
        }
    }

    // Get first squad ID for benchmark
    squads := manager.FilterByTag(SquadTag)
    squadID := squads[0].GetID()

    // Reset timer (exclude setup time)
    b.ResetTimer()

    // Benchmark target
    for i := 0; i < b.N; i++ {
        GetSquadMembers(manager, squadID)
    }
}
```

**Comparison Benchmarks**:
```go
// Before optimization
func BenchmarkSpatialGridPointerKeys(b *testing.B) {
    grid := NewSpatialGridPointerKeys()

    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            grid.Set(&Position{X: x, Y: y}, ecs.EntityID(x*100+y))
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        grid.Get(&Position{X: 50, Y: 50})
    }
}

// After optimization
func BenchmarkSpatialGridValueKeys(b *testing.B) {
    grid := NewSpatialGridValueKeys()

    for x := 0; x < 100; x++ {
        for y := 0; y < 100; y++ {
            grid.Set(Position{X: x, Y: y}, ecs.EntityID(x*100+y))
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        grid.Get(Position{X: 50, Y: 50})
    }
}

// Run: go test -bench=BenchmarkSpatialGrid -benchmem
// Compare results: pointer keys vs value keys
```

**Benchmark Options**:
```bash
# Run all benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkGetSquadMembers

# Include memory allocation stats
go test -bench=. -benchmem

# Run for longer (more accurate)
go test -bench=. -benchtime=10s

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### 6. Test Helpers

**Pattern**:
```go
// Test helper functions
func createTestManager(t *testing.T) *ecs.Manager {
    t.Helper()  // Marks as helper (stack traces skip this)
    return ecs.NewManager()
}

func createTestSquad(t *testing.T, manager *ecs.Manager, name string) ecs.EntityID {
    t.Helper()
    squadID := CreateSquad(manager, name, 1)
    if squadID == 0 {
        t.Fatal("Failed to create test squad")
    }
    return squadID
}

func assertSquadExists(t *testing.T, manager *ecs.Manager, squadID ecs.EntityID) {
    t.Helper()
    entity := manager.GetEntity(squadID)
    if entity == nil {
        t.Fatalf("Squad %d does not exist", squadID)
    }
}

func assertSquadMemberCount(t *testing.T, manager *ecs.Manager, squadID ecs.EntityID, expected int) {
    t.Helper()
    members := GetSquadMembers(manager, squadID)
    if len(members) != expected {
        t.Errorf("Expected %d members but got %d", expected, len(members))
    }
}

// Usage
func TestSquadOperations(t *testing.T) {
    manager := createTestManager(t)
    squadID := createTestSquad(t, manager, "Test Squad")

    assertSquadExists(t, manager, squadID)
    assertSquadMemberCount(t, manager, squadID, 0)

    // Add members
    CreateSquadMember(manager, squadID)
    assertSquadMemberCount(t, manager, squadID, 1)
}
```

### 7. TestMain for Setup/Teardown

**Pattern**:
```go
func TestMain(m *testing.M) {
    // Global setup
    fmt.Println("Setting up test environment...")

    // Initialize test data
    setupTestData()

    // Run tests
    exitCode := m.Run()

    // Global teardown
    fmt.Println("Cleaning up test environment...")
    cleanupTestData()

    os.Exit(exitCode)
}

func setupTestData() {
    // Load test fixtures
    // Initialize test database
    // Create temp directories
}

func cleanupTestData() {
    // Remove temp files
    // Close connections
    // Reset state
}
```

### 8. Mock/Stub Patterns for ECS Systems

**Pattern**:
```go
// Mock ECS Manager for isolated testing
type MockManager struct {
    entities map[ecs.EntityID]*ecs.Entity
    nextID   ecs.EntityID
}

func NewMockManager() *MockManager {
    return &MockManager{
        entities: make(map[ecs.EntityID]*ecs.Entity),
        nextID:   1,
    }
}

func (m *MockManager) NewEntity() *ecs.Entity {
    entity := &ecs.Entity{ID: m.nextID}
    m.entities[m.nextID] = entity
    m.nextID++
    return entity
}

func (m *MockManager) GetEntity(id ecs.EntityID) *ecs.Entity {
    return m.entities[id]
}

// Usage in tests
func TestWithMock(t *testing.T) {
    mock := NewMockManager()
    entity := mock.NewEntity()

    // Test behavior with controlled mock
    if entity.ID != 1 {
        t.Errorf("Expected ID 1 but got %d", entity.ID)
    }
}
```

### 9. Coverage Analysis

**Commands**:
```bash
# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Coverage per package
go test ./squads -coverprofile=squads.out
go test ./gui -coverprofile=gui.out
go test ./gear -coverprofile=gear.out

# Combine coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Coverage Targets**:
- Core systems (squad, combat, inventory): >80%
- ECS components: >60% (pure data, less logic)
- GUI code: >50% (harder to test, more visual)
- Utility functions: >90%

### 10. Integration Tests

**Pattern** (from project):
```go
func TestSquadCombatIntegration(t *testing.T) {
    // Setup: Create full game state
    manager := ecs.NewManager()

    // Create attacker squad
    attackerID := CreateSquad(manager, "Attacker", 1)
    for i := 0; i < 3; i++ {
        memberID := CreateSquadMember(manager, attackerID)
        SetCombatStats(manager, memberID, &CombatStats{
            Health: 100,
            Attack: 20,
            Defense: 5,
        })
    }

    // Create defender squad
    defenderID := CreateSquad(manager, "Defender", 2)
    for i := 0; i < 3; i++ {
        memberID := CreateSquadMember(manager, defenderID)
        SetCombatStats(manager, memberID, &CombatStats{
            Health: 100,
            Attack: 15,
            Defense: 8,
        })
    }

    // Execute combat (integration point)
    result := ExecuteSquadAttack(manager, attackerID, defenderID)

    // Verify integration
    if !result.Hit {
        t.Error("Expected combat to execute")
    }

    // Verify defender took damage
    defenderMembers := GetSquadMembers(manager, defenderID)
    damaged := false
    for _, member := range defenderMembers {
        stats := GetCombatStats(manager, member.GetID())
        if stats.Health < 100 {
            damaged = true
            break
        }
    }

    if !damaged {
        t.Error("Expected defender to take damage")
    }
}
```

## Testing Strategies

### Unit Tests
- Test individual functions in isolation
- Mock dependencies (ECS manager, etc.)
- Focus on logic correctness
- Fast execution (<100ms per test)

### Integration Tests
- Test system interactions (squad ↔ combat ↔ position)
- Use real ECS manager
- Verify data flow between components
- Slower but more realistic

### Combat Scenario Tests
```go
func TestCombatScenarios(t *testing.T) {
    scenarios := []struct {
        name           string
        attackerStats  CombatStats
        defenderStats  CombatStats
        expectedResult string
    }{
        {
            name:          "High attack vs low defense",
            attackerStats: CombatStats{Attack: 30, Defense: 5},
            defenderStats: CombatStats{Attack: 10, Defense: 3, Health: 50},
            expectedResult: "attacker_wins",
        },
        {
            name:          "Tank vs tank",
            attackerStats: CombatStats{Attack: 15, Defense: 10, Health: 200},
            defenderStats: CombatStats{Attack: 15, Defense: 10, Health: 200},
            expectedResult: "long_combat",
        },
    }

    for _, scenario := range scenarios {
        t.Run(scenario.name, func(t *testing.T) {
            // Simulate combat scenario
            // Verify expected outcome
        })
    }
}
```

## Reference Tests

**Squad System Tests** (`squads/squads_test.go`):
- Table-driven tests for squad creation
- Integration tests for combat
- Benchmark tests for query performance

**Combat System Tests** (`squads/squadcombat_test.go`):
- Damage calculation tests
- Hit/dodge/crit mechanics
- Cover mechanics validation

**Inventory System Tests** (`gear/inventory_test.go`):
- Item addition/removal
- Capacity limits
- Edge cases (nil, empty, invalid IDs)

## Quick Test Generation

For new code, follow this template:

```go
package mypackage

import (
    "testing"
    "github.com/bytearena/ecs"
)

func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    invalidInput,
            expected: nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)

            if tt.wantErr {
                if err == nil {
                    t.Error("Expected error but got nil")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }

            // Compare result to expected
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("Expected %v but got %v", tt.expected, result)
            }
        })
    }
}
```

## Testing Checklist

Before committing:
- [ ] All exported functions have tests
- [ ] Edge cases covered (nil, empty, boundaries)
- [ ] Error paths tested
- [ ] Integration points verified
- [ ] Coverage >60% for package
- [ ] Benchmarks for performance-critical code
- [ ] All tests pass: `go test ./...`

## Usage Example

```markdown
User: "Write tests for the combat damage calculation"

Response:
Here's a table-driven test for CalculateDamage:

```go
func TestCalculateDamage(t *testing.T) {
    tests := []struct {
        name      string
        attack    int
        defense   int
        minDamage int
        maxDamage int
    }{
        {
            name:      "Normal damage",
            attack:    20,
            defense:   5,
            minDamage: 15,
            maxDamage: 30,  // With crit
        },
        {
            name:      "High defense",
            attack:    10,
            defense:   15,
            minDamage: 1,   // Minimum damage
            maxDamage: 2,   // With crit
        },
        {
            name:      "Zero defense",
            attack:    25,
            defense:   0,
            minDamage: 25,
            maxDamage: 50,  // With crit
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            attacker := &CombatStats{Attack: tt.attack}
            defender := &CombatStats{Defense: tt.defense}

            // Run multiple times to check range
            for i := 0; i < 100; i++ {
                damage := CalculateDamage(attacker, defender)

                if damage < tt.minDamage {
                    t.Errorf("Damage %d below minimum %d", damage, tt.minDamage)
                }
                if damage > tt.maxDamage {
                    t.Errorf("Damage %d above maximum %d", damage, tt.maxDamage)
                }
            }
        })
    }
}
```

Reference: squads/squadcombat_test.go for similar combat testing patterns
```
