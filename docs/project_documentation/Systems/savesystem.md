# Save System Technical Documentation

**Last Updated:** 2026-02-27
**Package:** `game_main/savesystem` and `game_main/savesystem/chunks`

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Package Structure](#2-package-structure)
3. [Architecture Overview](#3-architecture-overview)
4. [Core Interfaces and Types](#4-core-interfaces-and-types)
5. [The EntityIDMap: Solving the ECS Serialization Problem](#5-the-entityidmap-solving-the-ecs-serialization-problem)
6. [The Three-Phase Load Protocol](#6-the-three-phase-load-protocol)
7. [Chunk Implementations](#7-chunk-implementations)
   - 7.1 [PlayerChunk](#71-playerchunk)
   - 7.2 [SquadChunk](#72-squadchunk)
   - 7.3 [CommanderChunk](#73-commanderchunk)
   - 7.4 [GearChunk](#74-gearchunk)
   - 7.5 [MapChunk](#75-mapchunk)
   - 7.6 [RaidChunk](#76-raidchunk)
8. [Save File Format](#8-save-file-format)
9. [Data Flow: Saving](#9-data-flow-saving)
10. [Data Flow: Loading](#10-data-flow-loading)
11. [Integration with the Rest of the Codebase](#11-integration-with-the-rest-of-the-codebase)
12. [Design Decisions and Rationale](#12-design-decisions-and-rationale)
13. [Non-Serialized Data and Post-Load Reconstruction](#13-non-serialized-data-and-post-load-reconstruction)
14. [Backward Compatibility Strategy](#14-backward-compatibility-strategy)
15. [Adding a New Chunk](#15-adding-a-new-chunk)
16. [API Reference](#16-api-reference)

---

## 1. Executive Summary

The save system serializes and deserializes the entire ECS (Entity Component System) game state to and from a single JSON file on disk. It solves the fundamental ECS serialization challenge — that entity IDs are runtime-assigned integers that change between sessions — through a two-phase load protocol using an `EntityIDMap` to translate saved IDs into newly assigned ones.

The system is organized around the `SaveChunk` interface. Each chunk owns one domain of game state (player, squads, commanders, gear, map, raid). Chunks register themselves via Go's `init()` mechanism, so the core save/load loop is unaware of the details of any specific domain. Chunks are serialized into a top-level JSON envelope with checksum verification and atomic file writes to prevent data corruption.

Non-serializable data (ECS component pointers, rendered images, Ebiten sprite objects) is never written to disk. Instead, post-load reconstruction steps rebuild these from serialized metadata (unit type strings, image paths, biome enums) after the ECS is restored.

---

## 2. Package Structure

```
savesystem/
├── savesystem.go          # Core framework: interfaces, SaveGame/LoadGame, chunk registry
├── idmap.go               # EntityIDMap: old-to-new ID translation
├── savesystem_test.go     # Unit tests for envelope marshaling, ID map behavior
└── chunks/
    ├── shared_types.go    # Shared serialization structs and conversion helpers
    ├── player_chunk.go    # PlayerChunk: player entity, resources, rosters
    ├── squad_chunk.go     # SquadChunk: squads and their unit members
    ├── commander_chunk.go # CommanderChunk: commanders, action state, spells
    ├── gear_chunk.go      # GearChunk: artifact inventory and equipment
    ├── map_chunk.go       # MapChunk: tile data, rooms, POIs
    └── raid_chunk.go      # RaidChunk: raid progression state
```

The two-package split (core framework vs. chunk implementations) is intentional. The `savesystem` package defines only the abstract contract (`SaveChunk` interface, `EntityIDMap`, `SaveEnvelope`) and the orchestration logic. The `chunks` package contains the domain-specific implementations that depend on game packages like `tactical/squads`, `mind/raid`, and `gear`. This prevents the core framework from importing any game-domain code.

---

## 3. Architecture Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                        main.go                                      │
│  _ "game_main/savesystem/chunks"  ← blank import triggers init()  │
└────────────────────────────────────────────────────────────────────┘
                              │ (via init())
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                 savesystem.registeredChunks []SaveChunk             │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────┐ ┌───┐ ┌──────┐ │
│  │  Player  │ │  Squads  │ │Commanders │ │ Gear │ │Map│ │ Raid │ │
│  └──────────┘ └──────────┘ └───────────┘ └──────┘ └───┘ └──────┘ │
└────────────────────────────────────────────────────────────────────┘
                              │
               ┌──────────────┴──────────────┐
               ▼                             ▼
       SaveGame(em)                   LoadGame(em)
               │                             │
               ▼                             ▼
        SaveEnvelope                  SaveEnvelope
        (JSON on disk)          ← reads from disk
               │                             │
               │                    ┌────────┴────────┐
               │                    ▼                 ▼
               │              Phase 1:           Phase 2:
               │             chunk.Load()     chunk.RemapIDs()
               │                    │                 │
               │               EntityIDMap            │
               │            (old→new mappings)        │
               │                              Phase 3 (optional):
               │                              chunk.Validate()
               ▼
         saves/roguelike_save.json
```

### Key Architectural Properties

**Chunk isolation.** Each chunk handles exactly one domain. The orchestrator (`SaveGame`/`LoadGame`) does not know what data any chunk serializes. It only calls the chunk's interface methods and collects results.

**Self-registration via `init()`.** Every chunk file calls `savesystem.RegisterChunk(&MyChunk{})` inside its `init()` function. The `chunks` package is imported with a blank import in `main.go`, which causes all `init()` functions in that package to run. The core framework never needs to be updated when a new chunk is added.

**Stateless chunk structs.** Most chunk structs hold no fields (e.g., `type PlayerChunk struct{}`). They are pure logic containers. The exception is `MapChunk`, which holds a `GameMap` pointer that must be set before calling `Save` or `Load` because `GameMap` exists outside the ECS world and cannot be discovered by querying entities.

---

## 4. Core Interfaces and Types

### `SaveChunk` Interface

Defined in `savesystem/savesystem.go:16`.

```go
type SaveChunk interface {
    ChunkID() string
    ChunkVersion() int
    Save(em *common.EntityManager) (json.RawMessage, error)
    Load(em *common.EntityManager, data json.RawMessage, idMap *EntityIDMap) error
    RemapIDs(em *common.EntityManager, idMap *EntityIDMap) error
}
```

- `ChunkID()` returns the string key used to store this chunk's data in the `SaveEnvelope.Chunks` map (e.g., `"player"`, `"squads"`). This key is stable across versions because it appears in saved files on disk.
- `ChunkVersion()` returns the serialization version for the chunk. Embedded in the chunk data and intended for future migration logic.
- `Save()` extracts state from the ECS and returns a JSON blob. Returning `nil, nil` signals that there is nothing to save (e.g., no player entity exists).
- `Load()` deserializes the JSON blob and creates new ECS entities, recording `old→new` ID mappings in the `EntityIDMap`. Cross-entity references should not be resolved here — they are stored with old IDs and fixed in `RemapIDs`.
- `RemapIDs()` is called after all chunks have completed `Load`. At this point, the `EntityIDMap` is fully populated and cross-entity references can be translated to the new IDs.

### `Validatable` Interface

Defined in `savesystem/savesystem.go:44`. Optional extension for chunks that need post-load integrity checks.

```go
type Validatable interface {
    Validate(em *common.EntityManager) error
}
```

If a chunk struct implements `Validatable`, its `Validate` method is called in Phase 3 after all IDs have been remapped. No existing chunk implements this yet; it is provided as an extension point.

### `SaveEnvelope`

Defined in `savesystem/savesystem.go:37`.

```go
type SaveEnvelope struct {
    Version   int                        `json:"version"`
    Timestamp string                     `json:"timestamp"`
    Checksum  string                     `json:"checksum,omitempty"`
    Chunks    map[string]json.RawMessage `json:"chunks"`
}
```

The envelope is the entire on-disk save file. `Chunks` maps chunk IDs to their raw JSON bytes. Using `json.RawMessage` for chunk data means the chunks map can hold heterogeneous JSON structures without the envelope needing to know the shape of any individual chunk's data. Unknown keys in `Chunks` are preserved as raw bytes when the file is read, supporting forward compatibility (a new chunk in a save file will simply be ignored by older code that does not have that chunk registered).

---

## 5. The EntityIDMap: Solving the ECS Serialization Problem

Defined in `savesystem/idmap.go`.

The fundamental problem with serializing an ECS is that entity IDs are runtime-assigned integers. The ECS library (`github.com/bytearena/ecs`) assigns IDs sequentially or by some internal scheme. When the game loads, it creates a fresh ECS world with no entities. If you recreate entities in any order, the new IDs will differ from the saved IDs. But many components hold references to other entities by ID (e.g., `SquadMemberData.SquadID`, `CommanderRosterData.CommanderIDs`, `RaidStateData.PlayerSquadIDs`). These cross-references would all be wrong after a naive load.

`EntityIDMap` solves this with a translation table:

```go
type EntityIDMap struct {
    oldToNew    map[ecs.EntityID]ecs.EntityID
    LoadContext map[string]interface{}
}
```

#### How It Works

During `Load` (Phase 1), each chunk:
1. Creates a new ECS entity.
2. Calls `idMap.Register(savedEntityID, newEntity.GetID())` to record the mapping.
3. Copies cross-entity references verbatim (with old IDs) into the component data.

During `RemapIDs` (Phase 2), each chunk:
1. Queries the ECS for the entities it created.
2. Calls `idMap.Remap(oldID)` to translate each stored ID to its new equivalent.
3. Overwrites the component fields with the translated IDs.

The key insight is that Phase 2 runs only after all chunks have completed Phase 1. This means when `SquadChunk.RemapIDs` runs, the mapping for squad IDs, unit IDs, commander IDs, and all other entity types already exists in the map.

#### The `LoadContext` Field

`LoadContext` is a `map[string]interface{}` used to pass state from a chunk's `Load` to its `RemapIDs` without storing mutable state on the chunk struct itself. This keeps chunk structs stateless (safe to reuse) while allowing data to survive across the two phases.

The `GearChunk` demonstrates the pattern: during `Load`, it deserializes the gear data and stores it in `LoadContext` under a known key. During `RemapIDs`, it retrieves it, resolves the entity IDs, and attaches the gear components to the now-existing player and squad entities.

#### Remap Variants

| Method | Behavior |
|---|---|
| `Remap(oldID)` | Returns new ID, or 0 if not found. Use for optional references. |
| `RemapSlice(ids)` | Remaps a slice; unknown IDs become 0. |
| `RemapStrict(oldID)` | Returns error if a non-zero ID has no mapping. Use for required references (e.g., squad-to-commander links). |

Zero is the zero value for `ecs.EntityID` and represents "no entity" throughout the codebase. `Remap(0)` always returns 0 by convention.

---

## 6. The Three-Phase Load Protocol

```
Phase 1 — Create Entities (Load)
  For each registered chunk, in registration order:
    chunk.Load(em, savedData, idMap)
      → Deserializes JSON
      → Creates new ECS entities
      → Registers oldID → newID in idMap
      → Stores old cross-entity IDs in components (unresolved)

Phase 2 — Fix Cross-Entity References (RemapIDs)
  For each registered chunk, in registration order:
    chunk.RemapIDs(em, idMap)
      → Queries ECS for entities this chunk created
      → Translates all stored old IDs to new IDs via idMap

Phase 3 — Optional Post-Load Validation (Validate)
  For each registered chunk that implements Validatable:
    chunk.Validate(em)
      → Checks integrity of restored data
      → Returns error if any invariant is violated
```

Phase separation is critical because cross-chunk references (e.g., a commander that references a squad by ID) require that the referenced entity exists and is registered in the ID map before the reference can be resolved. By loading all entities first and then remapping all references, the system avoids ordering dependencies between chunks.

---

## 7. Chunk Implementations

### 7.1 PlayerChunk

**File:** `savesystem/chunks/player_chunk.go`
**Chunk ID:** `"player"`

**What it serializes:**

| Field | Component | Notes |
|---|---|---|
| Entity ID | (implicit) | Saved for ID mapping registration |
| Position | `common.PositionComponent` | X, Y tile coordinates |
| Attributes | `common.AttributeComponent` | All 11 attribute fields |
| Resources | `common.ResourceStockpileComponent` | Gold, Iron, Wood, Stone |
| Unit Roster | `squads.UnitRosterComponent` | Per-type unit counts and entity IDs |
| Commander Roster | `commander.CommanderRosterComponent` | List of commander entity IDs |

**Key behaviors:**

- Finds the player entity by querying `em.WorldTags["players"]`. If the tag is not registered, `Save` returns `nil, nil`.
- During `Load`, after creating the new player entity, it manually rebuilds the `"players"` tag in `em.WorldTags`. This is necessary because the normal game initialization code in `gameinit.go` does not run on the load path; the chunk must reproduce any ECS tags that other systems depend on.
- Also registers the player entity with `common.GlobalPositionSystem` during `Load`.
- `RemapIDs` translates all unit entity IDs stored in `UnitRosterEntry.UnitEntities` and all squad IDs in `UnitRosterEntry.UnitsInSquads`. Commander roster IDs are similarly remapped via `idMap.RemapSlice`.

**Serialization struct:**

```go
type savedPlayer struct {
    EntityID  ecs.EntityID     `json:"entityID"`
    Position  savedPosition    `json:"position"`
    Attrs     savedAttributes  `json:"attributes"`
    Resources savedResources   `json:"resources"`
    UnitRos   *savedUnitRoster `json:"unitRoster,omitempty"`
    CmdRos    *savedCmdRoster  `json:"commanderRoster,omitempty"`
}
```

---

### 7.2 SquadChunk

**File:** `savesystem/chunks/squad_chunk.go`
**Chunk ID:** `"squads"`

**What it serializes:**

Each squad entity and its member units in a single nested structure. The squad itself holds squad-level data; its members are serialized inline as a slice.

**Squad-level fields:**

| Field | Component |
|---|---|
| Name, Formation, Morale, SquadLevel, TurnCount, MaxUnits | `squads.SquadComponent` (`SquadData`) |
| IsDeployed, GarrisonedAtNodeID | `squads.SquadComponent` (`SquadData`) |
| Position | `common.PositionComponent` |

**Member-level fields (all optional, added only when component exists):**

| Field | Component |
|---|---|
| Name | `common.NameComponent` |
| UnitType | `squads.UnitTypeComponent` |
| Attributes | `common.AttributeComponent` |
| GridPosition | `squads.GridPositionComponent` |
| Role | `squads.UnitRoleComponent` |
| TargetRow | `squads.TargetRowComponent` |
| Cover | `squads.CoverComponent` |
| AttackRange, MovementSpeed | `squads.AttackRangeComponent`, `squads.MovementSpeedComponent` |
| Experience | `squads.ExperienceComponent` |
| StatGrowth | `squads.StatGrowthComponent` |
| Leader | `squads.LeaderComponent` |
| AbilitySlots | `squads.AbilitySlotComponent` |
| Cooldowns | `squads.CooldownTrackerComponent` |

**Key behaviors:**

- Uses `squads.GetUnitIDsInSquad(squadID, em)` to discover member units at save time; does not rely on member ordering.
- During `Load`, member units are created with `SquadMemberData.SquadID` set directly to the new squad ID (already resolved within the same `Load` call). This means `RemapIDs` does not need to remap the member's `SquadID` field — it is one of the few cross-entity references that can be set correctly in Phase 1.
- `GarrisonedAtNodeID` is a reference to an overworld node entity. It is stored with the old ID during `Load` and remapped in `RemapIDs`.
- Optional components (Cover, TargetRow, Experience, etc.) use `omitempty` in JSON tags and are only attached during `Load` if they were present at save time.
- Both squad and unit entities are registered with `GlobalPositionSystem` during `Load`.

---

### 7.3 CommanderChunk

**File:** `savesystem/chunks/commander_chunk.go`
**Chunk ID:** `"commanders"`

**What it serializes:**

| Field | Component |
|---|---|
| Name, IsActive | `commander.CommanderComponent` (`CommanderData`) |
| Position | `common.PositionComponent` |
| Attributes | `common.AttributeComponent` |
| Action State | `commander.CommanderActionStateComponent` |
| Squad Roster | `squads.SquadRosterComponent` |
| Mana | `spells.ManaComponent` |
| SpellBook | `spells.SpellBookComponent` |

**Key behaviors:**

- Uses `commander.CommanderTag` to discover all commander entities at save time.
- The `SquadRoster.OwnedSquads` slice holds entity IDs that are stored verbatim during `Load` and remapped via `idMap.RemapSlice` in `RemapIDs`. This is the primary cross-chunk reference for the commander system.
- SpellBook spell IDs are string identifiers (not entity IDs), so they do not require remapping.
- Commander entities are registered with `GlobalPositionSystem` during `Load`.

---

### 7.4 GearChunk

**File:** `savesystem/chunks/gear_chunk.go`
**Chunk ID:** `"gear"`

The `GearChunk` demonstrates the most advanced use of the `LoadContext` pattern. Gear components (`ArtifactInventoryData`, `EquipmentData`) attach to entities created by other chunks (player and squad entities respectively). During Phase 1, those entities do not yet exist in the ECS. The chunk cannot attach components to nonexistent entities.

**Solution:** During `Load`, the chunk deserializes the JSON and stores the raw `savedGearChunkData` pointer in `idMap.LoadContext["gear_pending_data"]`. It does not create any entities. During `RemapIDs`, it retrieves the pending data, resolves entity IDs via `idMap.Remap`, looks up the actual entity pointers with `em.FindEntityByID`, and attaches the components.

This means `GearChunk` is the only chunk that does meaningful entity modification work in `RemapIDs` rather than in `Load`.

**What it serializes:**

| Section | Target entity | Fields |
|---|---|---|
| Artifact Inventory | Player entity | `MaxArtifacts`, per-artifact instance list (with `EquippedOn` entity IDs) |
| Equipment | Each squad entity | `EquippedArtifacts` (list of artifact definition IDs as strings) |

**Key behaviors:**

- Artifact definition IDs (`defID` in `OwnedArtifacts`) are string identifiers from the game data configuration, not entity IDs. They survive serialization unchanged.
- `ArtifactInstance.EquippedOn` is an entity ID (the squad entity wearing the artifact) and must be remapped.
- After attaching the inventory, the `LoadContext` key is deleted to allow garbage collection.

---

### 7.5 MapChunk

**File:** `savesystem/chunks/map_chunk.go`
**Chunk ID:** `"map"`

The `MapChunk` is the only chunk with a struct field, because the `GameMap` is not an ECS entity — it is a plain Go struct stored on the `Game` object. The chunk must be given a pointer to it before `Save` or `Load` is called. This pointer is injected via `gamesetup.ConfigureMapChunk(gm)`, which is called immediately before every `SaveGame` or `LoadGame` call.

**What it serializes:**

| Field | Notes |
|---|---|
| Width, Height | Validated against `coords.CoordManager` on load |
| Tiles (X, Y, TileType, Blocked, Biome, POIType, IsRevealed) | Per-tile data; tile images are NOT saved |
| Rooms (X1, X2, Y1, Y2) | Room rectangles |
| ValidPositions | Walkable tile coordinates |
| POIs (Position, NodeID, Biome) | Points of interest |

**Key behaviors:**

- Tile images are deliberately omitted from serialization. On load, `worldmap.LoadTileImages()` reads all image assets from disk, and `worldmap.SelectTileImage(images, tileType, biome, poiType)` reconstructs the correct image reference from the saved `TileType`, `Biome`, and `POIType` fields.
- The chunk validates that the saved map dimensions match the current `coords.CoordManager` dimensions on load. A mismatch (e.g., from a configuration change) produces a clear error rather than silently producing wrong tile indices.
- Tile initialization during load follows the same pattern used by world generators: first allocate all tiles as default walls, then apply saved overrides. This uses `coords.CoordManager.LogicalToIndex(pos)` for safe array indexing per the project-wide rule documented in `CLAUDE.md`.
- `RemapIDs` is a no-op: the map contains no entity ID references.

---

### 7.6 RaidChunk

**File:** `savesystem/chunks/raid_chunk.go`
**Chunk ID:** `"raid"`

Saves the full raid progression state when a raid is in progress. If there is no active raid (`chunkData.RaidState == nil`), only an empty JSON object is written for this chunk.

**Entity types serialized:**

| Entity Type | Component | Tag |
|---|---|---|
| Raid State (singleton) | `raid.RaidStateComponent` | `raid.RaidStateTag` |
| Floor States | `raid.FloorStateComponent` | `raid.FloorStateTag` |
| Room Data | `raid.RoomDataComponent` | `raid.RoomDataTag` |
| Alert Data | `raid.AlertDataComponent` | `raid.AlertDataTag` |
| Garrison Squads | `raid.GarrisonSquadComponent` | `raid.GarrisonSquadTag` |
| Deployment | `raid.DeploymentComponent` | `raid.DeploymentTag` |

**Cross-entity references remapped in `RemapIDs`:**

| Entity | Field | References |
|---|---|---|
| RaidState | `CommanderID`, `PlayerSquadIDs` | Commander entities, player squad entities |
| FloorState | `GarrisonSquadIDs`, `ReserveSquadIDs` | Garrison squad entities |
| RoomData | `GarrisonSquadIDs` | Garrison squad entities |
| Deployment | `DeployedSquadIDs`, `ReserveSquadIDs` | Player squad entities |

**Key behaviors:**

- After loading a save that includes raid state, the `RaidRunner.RestoreFromSave(raidEntityID)` method is called in `setup.go:SetupRoguelikeFromSave`. This gives the runner the new entity ID of the `RaidState` entity so that `IsActive()` returns `true` and the automatic raid start logic does not create a duplicate garrison.
- `GarrisonSquadData.ArchetypeName`, `FloorNumber`, and `RoomNodeID` are primitives that require no remapping.
- `ChildNodeIDs` and `ParentNodeIDs` in `RoomData` are integer node graph IDs, not entity IDs, so they are not remapped.

---

## 8. Save File Format

The save file is located at `saves/roguelike_save.json` relative to the game's working directory. A backup is kept at `saves/roguelike_save.json.bak`.

**Top-level structure:**

```json
{
  "version": 1,
  "timestamp": "2026-02-27T14:30:00Z",
  "checksum": "abc123...",
  "chunks": {
    "player":      { ... },
    "squads":      { ... },
    "commanders":  { ... },
    "gear":        { ... },
    "map":         { ... },
    "raid":        { ... }
  }
}
```

**Constants:**

| Constant | Value | Purpose |
|---|---|---|
| `CurrentSaveVersion` | `1` | Envelope version; incremented on breaking format changes |
| `SaveDirectory` | `"saves"` | Relative directory for save files |
| `SaveFileName` | `"roguelike_save.json"` | Primary save file name |

**Checksum:** A SHA-256 hash of the marshaled `chunks` map. Computed on save and verified on load. If the checksum field is absent (empty string), verification is skipped (for saves that predate checksum support).

---

## 9. Data Flow: Saving

```
SaveRoguelikeGame(g)                     [game_main/setup.go]
  │
  ├─ gamesetup.ConfigureMapChunk(&g.gameMap)
  │    └─ savesystem.GetChunk("map").(*MapChunk).GameMap = &g.gameMap
  │
  └─ savesystem.SaveGame(&g.em)           [savesystem/savesystem.go]
       │
       ├─ Create SaveEnvelope{version, timestamp, chunks: {}}
       │
       ├─ For each registeredChunk:
       │    data, err := chunk.Save(&g.em)
       │    envelope.Chunks[chunk.ChunkID()] = data
       │
       ├─ Compute SHA-256 checksum over marshaled chunks map
       │
       ├─ json.MarshalIndent(envelope)  → bytes
       │
       ├─ os.WriteFile(savePath + ".tmp", bytes, 0644)
       │
       ├─ os.Rename(savePath, savePath + ".bak")    ← backup old save
       │
       └─ os.Rename(savePath + ".tmp", savePath)    ← atomic replace
```

The atomic write pattern (write to `.tmp`, backup old, rename `.tmp` to final) ensures that a crash mid-save does not leave a partial file in place. Either the old save or the new save exists; never a corrupted half-write.

---

## 10. Data Flow: Loading

```
SetupRoguelikeFromSave(g)                [game_main/setup.go]
  │
  ├─ gamesetup.ConfigureMapChunk(&g.gameMap)
  │
  ├─ savesystem.LoadGame(&g.em)           [savesystem/savesystem.go]
  │    │
  │    ├─ Read saves/roguelike_save.json
  │    ├─ Unmarshal into SaveEnvelope
  │    ├─ Version check (error if envelope.Version > CurrentSaveVersion)
  │    ├─ SHA-256 checksum verification (if checksum field present)
  │    ├─ idMap := NewEntityIDMap()
  │    │
  │    ├─ Phase 1: For each registeredChunk where chunk is in envelope:
  │    │    chunk.Load(&g.em, chunkData, idMap)
  │    │      → Creates new ECS entities
  │    │      → Registers old→new ID mappings
  │    │
  │    ├─ Phase 2: For each registeredChunk where chunk is in envelope:
  │    │    chunk.RemapIDs(&g.em, idMap)
  │    │      → Resolves cross-entity references
  │    │
  │    └─ Phase 3: For each chunk implementing Validatable:
  │         chunk.Validate(&g.em)
  │
  ├─ rendering.NewRenderingCache(&g.em)
  │
  ├─ gamesetup.RestorePlayerData(&g.em, &g.playerData)
  │    └─ Queries player entity, populates g.playerData.PlayerEntityID and Pos
  │
  ├─ squads.InitUnitTemplatesFromJSON()   ← must be done explicitly on load path
  │
  ├─ gamesetup.RestoreRenderables(&g.em)
  │    ├─ Attach Renderable components to player entity (load image from disk)
  │    ├─ Attach Renderable components to commander entities
  │    ├─ Attach Renderable components to squad member entities (by UnitType template)
  │    └─ Set squad entity Renderable from leader's sprite
  │
  ├─ gamesetup.InitWalkableGridFromMap(&g.gameMap)
  │
  ├─ Wire up UI coordinator, modes, and input
  │
  └─ If raid was in progress: raidRunner.RestoreFromSave(raidEntityID)
```

---

## 11. Integration with the Rest of the Codebase

### Chunk Registration (blank import)

The `chunks` package is imported in `game_main/main.go:28` with a blank import:

```go
_ "game_main/savesystem/chunks" // Blank import to register SaveChunks via init()
```

This is the sole coupling point between `main` and the chunk implementations. Every chunk file in the `chunks` package has an `init()` that calls `savesystem.RegisterChunk(...)`. The blank import ensures all these `init()` functions run before `main()` starts.

### UIContext Callbacks

The save and load operations are exposed to the UI layer via callback functions stored on `framework.UIContext`:

```go
// Defined in gui/framework/uimode.go:56
uiContext.SaveGameCallback = func() error {
    return SaveRoguelikeGame(g)
}

uiContext.LoadGameCallback = func() {
    g.pendingLoad = true
}
```

The exploration mode UI (`gui/guiexploration/exploration_panels_registry.go`) shows Save and Load buttons only when these callbacks are non-nil. This means save/load buttons are automatically absent in overworld mode, where the callbacks are not set.

The `pendingLoad` flag is used for the load operation because loading is destructive (it resets the entire ECS world) and must happen at the start of the next `Update` tick rather than mid-frame from a UI callback.

### Start Menu Integration

`gui/guistartmenu/startmenu.go` calls `savesystem.HasSaveFile()` when building the start menu. The "Load Roguelike Save" button is only added to the UI if a save file exists on disk:

```go
if savesystem.HasSaveFile() {
    // Add "Load Roguelike Save" button
}
```

### Position System Integration

Each chunk that creates entities with positions must also register those entities with `common.GlobalPositionSystem`. This is done during `Load` rather than `RemapIDs` because the position data does not contain cross-entity references:

```go
if common.GlobalPositionSystem != nil {
    common.GlobalPositionSystem.AddEntity(newID, pos)
}
```

The `nil` check handles the test environment where `GlobalPositionSystem` may not be initialized.

### Non-ECS Data: The MapChunk Special Case

The `GameMap` is not an ECS entity. It is a value stored on the `Game` struct. The `MapChunk` bridges this gap by holding a pointer to it. The pointer must be injected before each save or load operation via `gamesetup.ConfigureMapChunk(&g.gameMap)`. If the pointer is not set when `Save` or `Load` is called, the function returns a clear error.

---

## 12. Design Decisions and Rationale

### Why JSON and not a binary format?

JSON is human-readable, making save files inspectable and debuggable without special tooling. The game is turn-based and single-player, so save file size and write latency are not performance concerns. Readability during development was prioritized over compactness.

### Why separate save structs from ECS component structs?

Each chunk defines its own serialization structs (e.g., `savedSquadMember`, `savedAttributes`) that are separate from the actual ECS component types (`squads.SquadMemberData`, `common.Attributes`). This is a deliberate decoupling decision:

1. **Version stability.** The ECS components can evolve (fields added, renamed) without changing the JSON format, as long as the conversion functions in `shared_types.go` are updated.
2. **Explicit mapping.** The conversion helpers (`attributesToSaved`, `savedToAttributes`) make the field mapping between in-memory and on-disk formats explicit and auditable.
3. **No JSON tags on domain types.** ECS components do not carry `json:` struct tags. Components are pure data that are not coupled to any serialization concern.

### Why the two-phase load (Load + RemapIDs) instead of ordering chunks?

An ordering-based approach would require defining a dependency graph between chunks (e.g., "squads must load before commanders because commanders reference squads"). As the number of chunks grows and cross-references become more complex, maintaining such an ordering becomes error-prone.

The two-phase approach has no ordering requirements between chunks in Phase 1. Every chunk creates its own entities independently. Cross-chunk references are resolved uniformly in Phase 2 once all entities exist. Adding a new chunk with new cross-references never requires reordering existing chunks.

### Why does GearChunk defer all work to RemapIDs?

The artifact inventory attaches to the player entity, and equipment attaches to squad entities. Both of those entities are created by other chunks (`PlayerChunk` and `SquadChunk`). If `GearChunk.Load` ran first, those entities would not exist yet. If it ran last, it would still not have entity references because the ID map would not be fully populated.

The `LoadContext` deferral pattern solves this cleanly: `GearChunk` stores its pending data during Phase 1 and applies it during Phase 2 when both the entities and the ID map are available.

### Why is `MapChunk.GameMap` a pointer field on the struct?

The game map is not managed by the ECS. It exists as a plain struct on the `Game` object. The `SaveChunk` interface only provides an `EntityManager`, which is sufficient for ECS-backed chunks. For the map, the chunk needs a direct reference to the `GameMap` struct.

The alternative — adding `GameMap` to `EntityManager` or putting it into the ECS as an entity — was rejected because the map is fundamental to game initialization (it is created before the ECS entities that populate it) and treating it as an ECS entity would complicate that initialization order.

### Why atomic writes and backups?

A game save that is interrupted mid-write (power loss, crash) would produce an unreadable or partially valid JSON file. The atomic write pattern (write to `.tmp`, rename to final) ensures the final file is either the complete new save or the complete old save — never partial. The `.bak` file provides an additional recovery option if the rename itself fails.

---

## 13. Non-Serialized Data and Post-Load Reconstruction

Several categories of data are never written to disk and must be reconstructed after loading.

### Ebiten Images and Renderables

Ebiten `*ebiten.Image` objects are GPU-resident resources. They cannot be serialized to JSON. The `Renderable` component (which holds an image pointer) is entirely absent from all chunk serializations.

After loading, `gamesetup.RestoreRenderables(em)` in `savesystem/savehelpers.go` rebuilds `Renderable` components for all entities that need them:

1. **Player entity:** Loads the image from `config.PlayerImagePath`.
2. **Commander entities:** Loads the same player image (commanders use the player sprite).
3. **Squad member entities:** Looks up the unit's type string (`UnitTypeData.UnitType`), finds the template definition, and loads the image from the template's asset path.
4. **Squad entities:** Calls `squads.SetSquadRenderableFromLeader(...)` which copies the leader unit's image to the squad entity (squads are rendered using the leader's sprite on the overworld map).

Unit image restoration depends on `squads.InitUnitTemplatesFromJSON()` having been called first, because template lookup requires the template registry to be populated. The load path in `setup.go:SetupRoguelikeFromSave` calls this explicitly before `RestoreRenderables`.

### ECS Tags

ECS tags (which are `ecs.Tag` values used for querying) are not serialized. Most tags are recreated by the ECS subsystem `init()` functions (called via `common.InitializeSubsystems(em)`), which run as part of `SetupSharedSystems` before any load or save operation.

The one exception is the `"players"` tag, which is rebuilt inside `PlayerChunk.Load`:

```go
playersTag := ecs.BuildTag(common.PlayerComponent, common.PositionComponent)
em.WorldTags["players"] = playersTag
```

This is necessary because, on the load path, the normal game startup code that would build this tag (`gameinit.go` and related setup functions) does not run. The chunk must reproduce any tags it depends on.

### `PlayerData` Struct

`common.PlayerData` is a convenience struct (not ECS-managed) that caches the player entity ID and a pointer to the player's position component. After loading, `gamesetup.RestorePlayerData(em, pd)` repopulates it by querying the `"players"` tag and reading the entity.

### Walkable Grid

The pathfinding walkable grid is rebuilt from the loaded `GameMap` tiles by `gamesetup.InitWalkableGridFromMap(gm)`.

### Overworld Systems

The full overworld system (ticks, factions, node influence) is not initialized on the roguelike load path. `SetupRoguelikeFromSave` intentionally mirrors `SetupRoguelikeMode` in its system initialization, not `SetupOverworldMode`. The save system only supports roguelike mode.

---

## 14. Backward Compatibility Strategy

The save system implements several forward and backward compatibility mechanisms:

**Missing chunks are skipped.** If a registered chunk has no corresponding entry in the save file's `Chunks` map, the chunk's `Load` and `RemapIDs` are not called. This means loading a save file that was created before a new chunk was added will simply leave that domain at its zero state. No error is produced.

**Unknown chunks are preserved.** The `SaveEnvelope.Chunks` field is `map[string]json.RawMessage`. If a save file contains a chunk key that no registered chunk handles, the raw bytes are preserved in memory (and would be written back to disk if re-saved). This allows future versions of the game to add chunks without corrupting saves from that future version when loaded in an older build.

**Version gating.** The envelope `Version` field exists for future use. Currently, the only check is that `envelope.Version > CurrentSaveVersion` produces an error (cannot load a save from a newer version of the game). Equal or older versions are accepted.

**`ChunkVersion()` is recorded.** Each chunk's version integer is stored on the chunk struct but is not yet included in the per-chunk JSON (it is accessible via the interface but not serialized into the chunk data). The field is provided as an extension point for implementing per-chunk migration logic in the future.

**`omitempty` fields.** Optional components use `omitempty` JSON tags on their containing structs. If a component was not present on the entity at save time, the field is absent from JSON entirely (not `null`). On load, a nil pointer from JSON deserialization indicates the component was not present and should not be added.

---

## 15. Adding a New Chunk

To add a new chunk for a new game domain:

1. **Create a file** in `savesystem/chunks/` (e.g., `overworld_chunk.go`).

2. **Define the chunk struct and serialization types:**
   ```go
   package chunks

   import (
       "encoding/json"
       "game_main/common"
       "game_main/savesystem"
       "game_main/mypackage"
   )

   func init() {
       savesystem.RegisterChunk(&OverworldChunk{})
   }

   type OverworldChunk struct{}

   func (c *OverworldChunk) ChunkID() string  { return "overworld" }
   func (c *OverworldChunk) ChunkVersion() int { return 1 }

   type savedOverworldData struct {
       // ... fields with json tags
   }
   ```

3. **Implement `Save`:** Query the ECS for your entities, extract data into saved structs, and call `json.Marshal`.

4. **Implement `Load`:** Unmarshal the JSON, create new ECS entities with `em.World.NewEntity()`, call `idMap.Register(savedID, newID)` for each entity, and store old cross-entity IDs verbatim in components.

5. **Implement `RemapIDs`:** Query the ECS for your entities, and call `idMap.Remap(oldID)` or `idMap.RemapStrict(oldID)` on every stored entity ID reference.

6. **Use `LoadContext` if needed:** If your chunk attaches components to entities created by another chunk (like `GearChunk` does), store a pointer to your deserialized data in `idMap.LoadContext["my_chunk_key"]` during `Load` and retrieve it during `RemapIDs`.

7. **Register position system entries** if your entities have positions:
   ```go
   if common.GlobalPositionSystem != nil {
       common.GlobalPositionSystem.AddEntity(newID, pos)
   }
   ```

8. **The blank import in `main.go` covers all files in the `chunks` package.** No change to `main.go` is needed.

---

## 16. API Reference

### Package `savesystem`

#### Interfaces

```go
// SaveChunk is implemented by each domain's chunk struct.
type SaveChunk interface {
    ChunkID() string
    ChunkVersion() int
    Save(em *common.EntityManager) (json.RawMessage, error)
    Load(em *common.EntityManager, data json.RawMessage, idMap *EntityIDMap) error
    RemapIDs(em *common.EntityManager, idMap *EntityIDMap) error
}

// Validatable may optionally be implemented for post-load validation.
type Validatable interface {
    Validate(em *common.EntityManager) error
}
```

#### Types

```go
// SaveEnvelope is the top-level save file structure.
type SaveEnvelope struct {
    Version   int                        `json:"version"`
    Timestamp string                     `json:"timestamp"`
    Checksum  string                     `json:"checksum,omitempty"`
    Chunks    map[string]json.RawMessage `json:"chunks"`
}

// EntityIDMap tracks old (saved) -> new (loaded) entity ID mappings.
type EntityIDMap struct {
    oldToNew    map[ecs.EntityID]ecs.EntityID
    LoadContext map[string]interface{}
}
```

#### Functions

```go
// SaveGame serializes the current game state to disk using atomic write + backup.
func SaveGame(em *common.EntityManager) error

// LoadGame deserializes a save file and rebuilds ECS state via the three-phase protocol.
func LoadGame(em *common.EntityManager) error

// HasSaveFile returns true if a save file exists at the standard path.
func HasSaveFile() bool

// DeleteSaveFile removes the save file (for use on permadeath or explicit reset).
func DeleteSaveFile() error

// RegisterChunk adds a SaveChunk to the global registry. Call from init().
func RegisterChunk(chunk SaveChunk)

// GetRegisteredChunks returns all registered chunks (for testing).
func GetRegisteredChunks() []SaveChunk

// GetChunk returns a registered chunk by its ID, or nil if not found.
// Used by ConfigureMapChunk to inject the GameMap pointer into MapChunk.
func GetChunk(chunkID string) SaveChunk
```

#### `EntityIDMap` Methods

```go
// NewEntityIDMap creates an empty ID mapping.
func NewEntityIDMap() *EntityIDMap

// Register records an old -> new entity ID mapping.
func (m *EntityIDMap) Register(oldID, newID ecs.EntityID)

// Remap returns the new ID for an old (saved) ID. Returns 0 if not found.
func (m *EntityIDMap) Remap(oldID ecs.EntityID) ecs.EntityID

// RemapSlice remaps a slice of old IDs to new IDs. Unknown IDs become 0.
func (m *EntityIDMap) RemapSlice(ids []ecs.EntityID) []ecs.EntityID

// RemapStrict returns the new ID or an error if a non-zero ID is unmapped.
func (m *EntityIDMap) RemapStrict(oldID ecs.EntityID) (ecs.EntityID, error)

// Count returns the number of registered mappings.
func (m *EntityIDMap) Count() int
```

### Package `savesystem/chunks`

#### Exported Types

```go
// MapChunk is the only chunk with a public field; set GameMap before Save/Load.
type MapChunk struct {
    GameMap *worldmap.GameMap
}
```

All other chunk types (`PlayerChunk`, `SquadChunk`, `CommanderChunk`, `GearChunk`, `RaidChunk`) are exported only so they can be accessed via the `SaveChunk` interface. Their structs have no public fields.

#### Helper Functions (from `shared_types.go`, package-private)

```go
// Attribute conversion (shared across player, squad, commander chunks)
func attributesToSaved(attr *common.Attributes) savedAttributes
func savedToAttributes(sa savedAttributes) common.Attributes

// Position conversion (shared across player, squad, commander chunks)
func positionToSaved(pos *coords.LogicalPosition) savedPosition
func savedToPosition(sp savedPosition) coords.LogicalPosition

// Slice copy helpers (safe nil handling)
func copyEntityIDs(ids []ecs.EntityID) []ecs.EntityID
func copyInts(ints []int) []int
```

### Integration Functions (package `gamesetup`)

These functions in `gamesetup/savehelpers.go` are called by `setup.go` to complete the load process after the ECS is restored:

```go
// ConfigureMapChunk injects the GameMap pointer into the MapChunk before save/load.
func ConfigureMapChunk(gm *worldmap.GameMap)

// RestorePlayerData repopulates the PlayerData struct from the loaded player entity.
func RestorePlayerData(em *common.EntityManager, pd *common.PlayerData)

// RestoreRenderables reconstructs Renderable components from saved metadata.
// Must be called after InitUnitTemplatesFromJSON().
func RestoreRenderables(em *common.EntityManager) error
```
