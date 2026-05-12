# Technical Debt Analysis: `templates/` Package

**Date:** 2026-05-12
**Scope:** 14 files, 2,348 LOC. Loads all JSON config + game data; populates ~17 global registries; consumed by 56 files across the codebase.
**Test coverage:** **10.8%** — only `difficulty.go` and `namegen.go` are tested. Validation logic (which is 99 panics across 5 files) is entirely untested.

---

## Files in Scope

| File | LOC | Purpose |
|---|---|---|
| `jsonschema.go` | 479 | JSON DTOs for every subsystem |
| `validation.go` | 427 | 15 validator functions, 99 panics |
| `difficulty.go` | 231 | DifficultyManager, presets, derivation |
| `gameconfig.go` | 165 | Player/Commander/Combat/Display config |
| `readdata.go` | 149 | 9 Read* loaders + generic `readAndUnmarshal` |
| `spelldefinitions.go` | 121 | SpellRegistry + SpellDefinition |
| `initialsetup.go` | 120 | Initial commanders/squads/factions |
| `entity_factory.go` | 116 | `CreateEntityFromTemplate` |
| `artifactdefinitions.go` | 77 | ArtifactRegistry, two-file loader |
| `namegen.go` | 54 | `GenerateName` |
| `registry.go` | 53 | Globals + `ReadGameData()` orchestrator |
| `unitspelldefinitions.go` | 41 | UnitSpellRegistry |
| `difficulty_test.go` | 163 | Tests |
| `namegen_test.go` | 152 | Tests |

---

## 1. Debt Inventory

### A. Loader Duplication — **HIGH IMPACT**
*(`readdata.go`, `gameconfig.go`, `difficulty.go`, `spelldefinitions.go`, `artifactdefinitions.go`, `unitspelldefinitions.go`)*

13 separate `Read*` / `Load*` functions all follow one of two near-identical templates:

```go
// Pattern A (panic-style) — 8 occurrences
readAndUnmarshal("gamedata/X.json", &XTemplate)
validateX(&XTemplate)
println("X loaded:", ...)

// Pattern B (warning-style) — 3 occurrences
data, err := os.ReadFile(AssetPath(path))
if err != nil { fmt.Printf("WARNING..."); return }
json.Unmarshal(...)
if err != nil { fmt.Printf("WARNING..."); return }
```

**Problems:**
- **Inconsistent failure mode**: missing `monsterdata.json` → panic; missing `spelldata.json` → silent warning + empty registry; missing `mapgenconfig.json` → "use defaults" message. A new loader has no idea which to pick. The same fault is fatal in some configs and silent in others.
- **Inconsistent path constants**: artifacts has `MinorArtifactDataPath`/`MajorArtifactDataPath`; spells has `SpellDataPath` (in `registry.go`); everything else inlines the literal `"gamedata/x.json"` at the call site.
- **Inconsistent logging**: `println()` (10×), `fmt.Printf()` (5×), `fmt.Sprintf` panics, `log.Fatal` in `entity_factory.go`. No level/prefix convention.

**Quantify:** ~80 LOC of pure boilerplate. Adding a new config file requires editing 3 files (loader, registry var, `ReadGameData` sequence).

### B. `jsonschema.go` is a God File — **HIGH IMPACT** (479 LOC, 41 struct types)

Mixes JSON shapes for completely unrelated subsystems:
- Monster/encounter DTOs
- AI config (5 nested struct types)
- Power config (5 nested struct types)
- Overworld config (10+ nested struct types)
- Influence config (4 struct types)
- Map gen config (6 nested struct types)
- Name config (2 struct types)
- Node definition + encounter definition structs

Each subsystem's JSON shape is glued into one file because they all need to be unmarshalable. The actual ownership is scattered — `RoleBehaviorConfig` belongs conceptually with `mind/ai`, `PowerProfileConfig` with `mind/evaluation`, etc. — but they all live here because the loader is here.

**Smell — `FactionAIConfig2`** (`gameconfig.go:55`):
```go
// FactionAIConfig2 avoids name collision with the existing FactionAIConfig in jsonschema.go.
type FactionAIConfig2 struct {
    StartingGold  int `json:"startingGold"`
    StartingIron  int `json:"startingIron"`
    ...
}
```
A literal `2` suffix because `gameconfig.go` and `jsonschema.go` both define a struct called `FactionAIConfig` with different fields. Both bind to the JSON key `"factionAI"` in different files. This is a naming collision papered over with a digit.

### C. `validation.go` is also a God File — **HIGH IMPACT** (427 LOC, 99 panics, 15 validate* functions)

Same single-file dumping ground problem as jsonschema.go. Validators for unrelated subsystems (Node, Encounter, AI, Power, Influence, MapGen, Overworld, Name, GarrisonRaid) all share a file.

**Hardcoded valid-room-types map** at `validation.go:337`:
```go
validRoomTypes := map[string]bool{
    "barracks": true, "guard_post": true, "armory": true,
    "command_post": true, "patrol_route": true, "mage_tower": true,
    "rest_room": true, "stairs": true,
}
```
Duplicates a list owned by `world/garrisongen/` — shotgun surgery if a room type is renamed.

**Hardcoded required-IDs lists** at `validation.go:28-34` and `82-88`:
```go
requiredNodes := map[string]bool{
    "necromancer": false, "banditcamp": false, "corruption": false,
    "beastnest": false, "orcwarband": false,
}
```
Same five strings duplicated at lines 28 and 82. If you add a 6th threat, you must edit two literal maps in the same file. The whole point of `nodeDefinitions.json` was to centralize this — yet the validator hardcodes the IDs it expects to find there.

### D. Global Mutable State — **MEDIUM IMPACT** (`registry.go`)

13 package-level `var`s, all mutated at load time, all read across 56 files. Plus `GlobalDifficulty`, `SpellRegistry`, `UnitSpellRegistry`, `ArtifactRegistry`, `NameConfigTemplate`.

**Consequences:**
- Test isolation is hard — `namegen_test.go:11-30` reaches directly into `NameConfigTemplate`.
- Tests cannot run in parallel.
- No init ordering protection: `ReadGameData()`'s function order at `registry.go:36-52` is the only thing preventing crashes (e.g., `ReadEncounterData` calls `validateNodeEncounterLinks` which reads `EncounterDefinitionTemplates` set by the same function — works only because the order in `ReadGameData` puts `ReadNodeDefinitions` first).

### E. Validation: panic-only — **HIGH IMPACT**

99 raw `panic()` calls. No structured error returns. A bad JSON file crashes the whole game with a stack trace ending at `panic`. There is no path to recover, log to a file, or fall back. Validation messages are string-concatenated with `+ key +` — not localized, not formatted, no field context (line/column from JSON).

### F. `entity_factory.go` uses `any` for type dispatch — **LOW IMPACT**

```go
func CreateEntityFromTemplate(manager *common.EntityManager, config EntityConfig, data any) *ecs.Entity {
    switch config.Type {
    case EntityCreature:
        m, ok := data.(JSONMonster)
        ...
```

Only one entity type exists (`EntityCreature`). The switch, type assertion, and `any` parameter exist for a future case that never arrived. `entity_factory.go:46-61` could be a single typed function `CreateCreatureEntity(mgr, cfg, JSONMonster)`. The "unreachable return" comment at line 60 is a tell.

### G. Architecture leak: `templates` imports `worldmapcore` — **MEDIUM IMPACT**

`entity_factory.go` imports `game_main/world/worldmapcore` to block a tile (`config.GameMap.Tiles[ind].Blocked = true` at line 94). This is the only consumer in templates. A `templates` package logically should be a leaf (data + DTOs) — instead it pulls in map internals. The `GameMap` field on `EntityConfig` exists solely for this side effect; out of ~10 call sites of `CreateEntityFromTemplate`, most pass `GameMap: nil`.

### H. Documentation gaps — **MEDIUM IMPACT**

- `registry.go` has no header documentation explaining the registry pattern, mutation rules, or load order.
- `ReadGameData()` (the entry point for 14 loaders) has zero comments on ordering constraints — and there *are* constraints (Game config before others; Difficulty before others; Nodes before Encounters; Spells before UnitSpells).
- No package-level doc comment for `templates`. The closest is `entity_factory.go:1-3` which is misnamed (`Package entitytemplates`) — the package is actually called `templates`. Doc comment lies.

### I. Testing debt — **CRITICAL**

- **No tests** for: `validation.go` (99 panic paths), `readdata.go` (file I/O, error paths), `gameconfig.go`, `initialsetup.go`, `spelldefinitions.go`, `artifactdefinitions.go`, `unitspelldefinitions.go`, `entity_factory.go`, `jsonschema.go`.
- Existing tests (`difficulty_test.go`, `namegen_test.go`) modify global state (`NameConfigTemplate`, `GlobalDifficulty`) — not parallelizable.
- No table-driven tests for the 13 loaders — each one repeats panic / success / missing-file logic that nothing exercises.

---

## 2. Impact Assessment

| Item | Severity | Why |
|---|---|---|
| Loader duplication + inconsistent failure modes | **High** | Every new config file forces edits in 3 places and a coin-flip on error semantics |
| `jsonschema.go` 479 LOC god file | **High** | Slows orientation; any subsystem JSON change creates a merge magnet |
| `validation.go` 427 LOC + hardcoded required lists | **High** | Defeats the data-driven intent of `nodeDefinitions.json`; adding a threat = edit JSON + 2 hardcoded maps |
| Untested validation (99 panic paths) | **Critical** | A single bad JSON character can hard-crash the game with no test catching the regression |
| Global mutable registries | **Medium** | Tests can't run in parallel; load order is implicit, encoded only in `ReadGameData()` body |
| `FactionAIConfig2` naming collision | **Low** | Cosmetic but a clear "we gave up" smell |
| `worldmapcore` leak in entity_factory | **Medium** | Stops `templates` from being a leaf data package; complicates dependency graph |
| `entity_factory.go` over-abstracted dispatch | **Low** | Dead generality; cheap to delete |

---

## 3. Quick Wins (1–2 days, do now)

1. **Remove `FactionAIConfig2` rename** — rename to `GameConfigFactionResources` (it's faction starting resources, not AI). 5 min.
2. **Promote `gamedata/...` paths to named constants** in `registry.go`, like `SpellDataPath` already is. Eliminates 11 inline string literals. 15 min.
3. **Pick one logging style** — either `log.Printf("[templates] X loaded: ...")` or kill the load chatter entirely. Standardize across the package. 30 min.
4. **Delete `entity_factory.go`'s type switch** — collapse `CreateEntityFromTemplate` → `CreateCreatureEntity` directly, drop `EntityType` enum, drop `any`. ~20 LOC removed. 30 min.
5. **Fix the lying package comment** at `entity_factory.go:1-3` (says `Package entitytemplates`, package is `templates`). 1 min.
6. **De-duplicate the hardcoded required-IDs lists** in `validation.go:28-34` and `82-88` into one package-level `requiredNodeIDs` slice used by both validators. 15 min.

**Total ~2 hours, removes ~100 LOC of inconsistency.**

---

## 4. Medium-Term (1–2 weeks)

### 1. Extract a `Loader[T]` helper
```go
type Loader[T any] struct {
    Path     string
    Optional bool
    Validate func(*T)
}

func (l Loader[T]) Load(target *T) error { ... }
```
Each `Read*` shrinks to a one-line declaration. Resolves the panic-vs-warning inconsistency by making `Optional` explicit.

### 2. Split `jsonschema.go` by subsystem
- `schema_ai.go` (AI/threat/support)
- `schema_power.go` (Power profiles, role multipliers)
- `schema_overworld.go` (faction/threat/victory)
- `schema_mapgen.go` (rooms, caverns, garrison raid)
- `schema_node_encounter.go`
- `schema_name.go`
- `schema_game.go` (move `gameconfig.go` types here)

Same package, file-level concerns separated. Re-locate companion validators alongside.

### 3. Split `validation.go` the same way
Each subsystem owns its validators next to its schema.

### 4. Add validation tests
At minimum one happy-path + 2 panic-path tests per validate function. ~30 tests. Bring coverage from 10.8% → ~60%.

### 5. Replace panic with error returns in validators
Keep panic only in `ReadGameData()` for boot-time fatal errors, with a wrapping message identifying which loader failed.

---

## 5. Long-Term (1+ month, only if pain justifies)

### 1. Inject a `*Templates` value instead of globals
`func NewTemplates(assetDir string) (*Templates, error)` returns a struct holding all registries. Tests construct their own; production constructs one in `main`. Removes 17 globals, makes tests parallelizable. **Significant churn** across 56 consumer files — only worth it if there's a real reason (e.g., supporting hot-reload, alternate config sets, modding).

### 2. Move `entity_factory.go` out of `templates`
Into a `setup/entityfactory/` package. It uses `worldmapcore` and is the only file in `templates` that does anything ECS-y. Removing it lets `templates` become a true leaf package.

### 3. Schema-driven validation
For the node/encounter/threat case (where the same five IDs are hardcoded in 2 maps), generate the required list from a single declarative source so adding a 6th threat is a one-line JSON change.

---

## 6. Prevention

- **Add a `templates_test.go`** that runs every `Read*`/`Load*` function against the production JSON files, asserting no panics and non-empty registries. Catches every bad-JSON regression with one test.
- **Lint rule (or `go vet` script):** flag new `panic(` inside `templates/` and ask author to use error return.
- **CLAUDE.md note:** "When adding a new gamedata JSON file, use the `Loader[T]` helper — do not add a new bespoke `Read*` function."

---

## 7. ROI Summary

| Action | Effort | Payoff |
|---|---|---|
| Quick wins (1–6 above) | 2 hours | Removes ~100 LOC inconsistency, kills `FactionAIConfig2` smell, deletes dead dispatch |
| `Loader[T]` extraction | 1 day | 13 functions → 13 one-liners; one place to change error semantics |
| Split god files | 0.5 day | 906 LOC across 2 files → ~150 LOC across 10 files; merge conflicts drop |
| Validation tests | 1–2 days | 10.8% → ~60% coverage; catches regressions on the hardest-to-debug failure mode (boot panic) |
| Replace panic → error | 0.5 day | Game can degrade gracefully on bad JSON instead of hard-crashing |

**Highest ROI:** the validation tests. The package has 99 panic paths and zero tests covering them — a single bad commit to any gamedata JSON file can take down the game.
