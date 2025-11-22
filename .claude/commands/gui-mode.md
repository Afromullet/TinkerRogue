Create GUI mode: {mode-name}

Phase 1: Design & Planning
1. Layout structure (panels, positions, sizes)
2. Widget hierarchy and data flow
3. ECS queries needed (via GUIQueries)
4. Component usage (SquadListComponent, DetailPanelComponent, etc.)
5. Input handling (hotkeys, button handlers)
6. Mode transitions

Phase 2: Implementation Plan
- Create analysis/gui_{mode-name}_plan_[timestamp].md
- Ask for approval before implementation

Follow GUI_PATTERNS.md strictly.
Use BaseMode embedding pattern.
Reference similar modes for consistency.
