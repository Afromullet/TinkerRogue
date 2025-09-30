# Project Configuration for Claude Code

## Build Commands
- Build: `go build -o game_main/game_main.exe game_main/*.go`
- Run: `go run game_main/*.go`
- Test: `go test ./...`
- Clean: `go clean`

## Dependencies
- Install dependencies: `go mod tidy`

## Development Notes
- This is a Go-based roguelike game using the Ebiten engine
- Main entry point: `game_main/main.go`
- Assets directory: `../assets/` (relative to game_main)

## Common Issues
- Ensure assets directory exists with required tile images
- Run `go mod tidy` after pulling changes

## Simplification Roadmap (Priority Order)

### ✅ 1. Input System Consolidation *COMPLETED*
**Files:** `input/inputcoordinator.go`, controller files
- **Problem:** Scattered global state, tight coupling, mixed responsibilities
- **Status:** ✅ Implemented proper InputCoordinator with MovementController, CombatController, UIController
- **Achievement:** Eliminated scattered input handling and global state issues

### ✅ 2. Coordinate System Standardization *COMPLETED*
**Files:** `coords/cordmanager.go`, `coords/position.go`
- **Problem:** Multiple coordinate systems causing bugs (noted in LessonsLearned.txt)
- **Status:** ✅ Unified CoordinateManager replaces scattered CoordTransformer calls
- **Achievement:** Type-safe coordinate handling with LogicalPosition/PixelPosition

### 🔄 3. Status Effects vs Item Behaviors *70% COMPLETED*
**Files:** `gear/stateffect.go`, `gear/itemactions.go`
- **Problem:** Throwables forced into StatusEffect interface when they're actions, not effects
- **Status:** 🔄 Significant improvement - ItemAction interface created, type-safe access implemented
- **Achievement:** Conceptual separation with better spawn logic and type safety
- **Remaining:** Remove StatusEffects dependency from ItemAction interface for true separation

### 🔄 4. Entity Template System *PARTIALLY COMPLETED*
**File:** `entitytemplates/creators.go` (177 lines)
- **Problem:** Multiple `CreateXFromTemplate()` functions with identical structure
- **Status:** 🔄 Added ComponentAdder pattern and createFromTemplate()
- **Remaining:** Consolidate specialized CreateXFromTemplate() functions
- **Impact:** ~60% complete - composition pattern added but duplication remains

### ❌ 5. Graphics Shape System *NOT STARTED*
**File:** `graphics/drawableshapes.go` (390 lines)
- **Problem:** 8+ shape types with complex algorithms and code duplication
- **Status:** ❌ Reorganized but not simplified - core duplication remains
- **Impact:** Could reduce to ~200 lines with 3 basic shapes + parameters
- **Approach:** Replace with 2-3 basic shapes (Circle, Rectangle, Line) + parameters

### ❌ 6. GUI Button Factory *NOT STARTED*
**File:** `gui/playerUI.go`
- **Problem:** 6+ separate button creation functions with 90% duplicate code
- **Status:** ❌ Basic CreateButton() exists but specialized functions remain
- **Impact:** Replace duplicate functions with configurable factory pattern
- **Approach:** Generic button factory with configuration

## Completed Simplifications
- ✅ **Action Queue System Removal** - Removed complex ActionQueue/Turn system, implemented direct player actions