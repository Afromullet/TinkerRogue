# Developer Tools Reference

Quick reference for development commands, analysis tools, profiling, and asset preparation.

---

## Table of Contents

- [Running Unit Tests](#running-unit-tests)
- [Code Quality Checks](#code-quality-checks)
- [Combat Simulation Tools](#combat-simulation-tools)
- [WASM Build](#wasm-build)
- [Performance Profiling (pprof)](#performance-profiling-pprof)
- [Asset Preparation (GIMP)](#asset-preparation-gimp)

---

## Running Unit Tests

```bash
go test ./...                   # All tests
go test ./squads/...            # Single package
go test -v ./...                # Verbose output
go test -cover ./...            # With coverage report
```

---

## Code Quality Checks

### Dead Code

```bash
deadcode ./...                  # Check for dead code
deadcode -test ./...            # Include test files
```

### Unused Variables

```bash
golangci-lint run
```

---

## Combat Simulation Tools

### Building the Combat Simulator

Run from the TinkerRogue root directory:

```bash
go build -o combatsim.exe ./tools/combat_simulator/cmd
```

### Running the Simulator Directly

```bash
cd game_main && go run ../tools/combat_simulator/
```

### Combat Visualizer

Generates ASCII visualizations of combat battles from exported JSON logs, showing squad formations, attack flows, and statistics.

**Location:** `tools/combat_visualizer/`

First, enable combat log export in `config/config.go`:

```go
ENABLE_COMBAT_LOG_EXPORT = true
```

Then run the visualizer against a battle log:

```bash
tools\combat_visualizer\combat_visualizer.exe game_main\simulation_logs\<battle_log>.json >> all_battles.txt

tools\combat_visualizer\combat_visualizer.exe game_main\simulation_logs\battle_20260421_155214.624.json >> all_battles.txt
```

### Combat Balance Report

Generates balance analysis reports from combat simulation logs. Requires simulation logs from the combat simulator.

**Location:** `tools/combat_balance/`

```bash
cd tools/combat_balance && go run .
```

### Full Pipeline Script

Runs the complete simulation pipeline. Execute from the `game_main` directory:

```bash
scripts\run_combat_pipeline.bat

scripts\run_combat_pipeline.bat --suite duels   # Extra args forwarded to the simulator
```

---

## WASM Build

```bash
go run github.com/hajimehoshi/wasmserve@latest C:\Users\Afromullet\Desktop\TinkerRogue
```

Then open http://localhost:8080/ in a browser.

---

## Performance Profiling (pprof)

### Setup

Add the following to `main.go` before calling `ebiten.RunGame`:

```go
import (
    _ "net/http/pprof" // registers pprof HTTP handlers
    "net/http"
    "runtime"
)

// Start the pprof server in the background
go func() {
    fmt.Println("Running pprof server")
    http.ListenAndServe("localhost:6060", nil)
}()

runtime.SetCPUProfileRate(1000)
runtime.MemProfileRate = 1
```

### Collecting Profiles

```bash
# CPU profile (60 or 120 seconds)
curl -o cpu_profile.pb.gz http://localhost:6060/debug/pprof/profile?seconds=60
curl -o cpu_profile.pb.gz http://localhost:6060/debug/pprof/profile?seconds=120

# Heap profile
curl -o heap.pb.gz http://localhost:6060/debug/pprof/heap

# Benchmark profile
go test -bench . -cpuprofile=cpu.prof
```

### Analyzing a Profile

Open the interactive pprof shell:

```bash
go tool pprof cpu_profile.pb.gz
go tool pprof heap.pb.gz
```

Common commands inside the shell:

```
top                                  # Top CPU consumers
web                                  # Call graph in browser
svg > output.svg                     # Export call graph as SVG
tree                                 # Tree representation of calls
tree runtime.tracebackPCs            # Tree for a specific caller
web -node=runtime.systemstack        # Graph a specific node
list yourpkg.SomeFunction            # Annotated source for a function
```

---

# Measuring Cyclomatic Complexity

Calculate cyclomatic complexities of Go functions.
Usage:
    gocyclo [flags] <Go file or directory> ...

Flags:
    -over N               show functions with complexity > N only and
                          return exit code 1 if the set is non-empty
    -top N                show the top N most complex functions only
    -avg, -avg-short      show the average complexity over all functions;
                          the short option prints the value without a label
    -ignore REGEX         exclude files matching the given regular expression

The output fields for each line are:
<complexity> <package> <function> <file:line:column

## Examples

$ gocyclo .
$ gocyclo main.go
$ gocyclo -top 10 mind/
$ gocyclo -over 25 docker
$ gocyclo -avg .
$ gocyclo -top 20 -ignore "_test|Godeps|vendor/" .
$ gocyclo -over 3 -avg gocyclo/

---

## Asset Preparation (GIMP)

Two selection techniques cover most UI asset extraction scenarios. The choice depends on how uniform the background is.

### Extracting a UI Element — Fuzzy Select (Magic Wand)

Best when the background is mostly uniform and the element boundary is clearly defined.

1. Open the image — `File → Open`
2. Add alpha channel — `Layer → Transparency → Add Alpha Channel`
3. Select the Fuzzy Select tool — press `U`
4. Set tool options: Antialiasing on, Feather edges off, Threshold 20–30
5. Click the background; adjust threshold if the selection is too broad or too narrow
6. Invert — `Select → Invert`
7. Refine (optional) — `Select → Shrink → 1–2 px`, then `Select → Feather → 1 px`
8. Copy to new layer — `Edit → Copy`, then `Edit → Paste As → New Layer`
9. Hide or delete the original layer
10. Clean edges — Eraser tool or Quick Mask (`Shift+Q`)
11. Export — `File → Export As → PNG`

### Extracting a UI Element — Select by Color

Best when the background has slight variations or the target color appears across the whole image.

1. Open the image — `File → Open`
2. Add alpha channel — `Layer → Transparency → Add Alpha Channel`
3. Select the Select by Color tool — press `Shift+O`
4. Set tool options: Threshold 20–40, Antialiasing on, Feather edges off
5. Click the background; adjust threshold as needed
6. Invert — `Select → Invert`
7. Refine (optional) — `Select → Shrink → 1–2 px`, then `Select → Feather → 1 px`
8. Copy to new layer — `Edit → Copy`, then `Edit → Paste As → New Layer`
9. Hide or delete the original layer
10. Clean up — Quick Mask (`Shift+Q`) or a Layer Mask (right-click layer → `Add Layer Mask → Selection`)
11. Export — `File → Export As → PNG`

### Recommended Starting Values

| Setting | Fuzzy Select | Select by Color |
|---|---|---|
| Threshold | 20–30 | 20–40 |
| Shrink | 1–2 px | 1–2 px |
| Feather | 0.5–2 px | 0.5–2 px |

### Resizing an Image

1. Open the image — `File → Open`
2. `Image → Scale Image…`
3. Set Width and Height (click the chain link icon to scale independently)
4. Choose interpolation: **Cubic** (general), **NoHalo / LoHalo** (UI assets), **None** (pixel art)
5. Click **Scale**
6. Export — `File → Export As → PNG`

### Tips

- Use Quick Mask (`Shift+Q`) to paint fine-grained selection adjustments
- If a dark halo appears around the extracted element, shrink the selection by 1 px and delete the fringe
- Save a `.xcf` file to preserve layers for future edits
