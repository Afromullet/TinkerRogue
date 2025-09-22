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

### 1. Graphics Shape System 🎯 *Highest Impact*
**File:** `graphics/drawableshapes.go` (766 lines)
- **Problem:** 8+ shape types with massive code duplication
- **Impact:** Could reduce from 766 lines to ~200 lines
- **Approach:** Replace with 2-3 basic shapes (Circle, Rectangle, Line) + parameters
- **Status:** Not started

### 2. Status Effects vs Item Behaviors 🎯 *High Impact*
**Files:** `gear/stateffect.go` (457 lines), throwable system
- **Problem:** Throwables forced into StatusEffect interface when they're actions, not effects
- **Impact:** Clear separation of concerns, remove forced abstractions
- **Note:** Already identified in Todos - "Make throwable something that's not an item effect"
- **Status:** Not started

### 3. Input System Consolidation 🎯 *High Impact*
**Files:** `input/avatarmovement.go`, `input/avataractions.go`, `input/inputdrawing.go`
- **Problem:** Scattered global state, tight coupling, mixed responsibilities
- **Impact:** Single `InputManager` with clean state management
- **Approach:** Consolidate into one input handler class
- **Status:** Partially done (action queue system removed)

### 4. GUI Button Factory 🎯 *Medium Impact*
**File:** `gui/playerUI.go`
- **Problem:** Duplicate button creation functions (90% same code)
- **Impact:** Replace 6+ creation functions with 1 generic factory
- **Approach:** Generic button factory with configuration
- **Status:** Not started

### 5. Entity Template System 🎯 *Medium Impact*
**File:** `entitytemplates/creators.go`
- **Problem:** Multiple `CreateXFromTemplate()` functions with identical structure
- **Impact:** Single generic template creator
- **Approach:** Use composition over specialized functions
- **Status:** Not started

### 6. Coordinate System Standardization 🎯 *Medium Impact*
**Files:** Multiple files with coordinate confusion
- **Problem:** Multiple coordinate systems causing bugs (noted in LessonsLearned.txt)
- **Impact:** Standardize on one system, remove conversion complexity
- **Approach:** Pick logical coordinates as the standard
- **Status:** Not started

## Completed Simplifications
- ✅ **Action Queue System Removal** - Removed complex ActionQueue/Turn system, implemented direct player actions