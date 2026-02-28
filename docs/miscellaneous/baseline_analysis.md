# CPU Profile Analysis: benchmark_baseline.pb.gz

**Profile Duration**: 120.13s | **Total Samples**: 130,723ms (108.82% — multiple cores sampled)
**Binary**: `__debug_bin1702592596.exe` (debug build from VS Code debugger)
**Date**: 2026-02-28

---

## Summary

| Category | Flat Time | % of Total | Real Problem? |
|----------|-----------|------------|---------------|
| `runtime/pprof.lostProfileEvent` | 40,360ms | 30.87% | No — profiler artifact |
| `runtime.cgocall` (via syscall) | 34,554ms | 26.43% | Partially — VSync wait dominates |
| `runtime.mallocgc` + profiling overhead | 33,924ms | 25.95% | Yes — real allocation problem amplified by debug profiling |

Over 80% of sampled time falls into these three categories. Two are artifacts of profiling a debug build; the third reveals a real allocation hotspot.

---

## Finding 1: lostProfileEvent (30.87%) — Profiler Artifact

`runtime/pprof.lostProfileEvent` at 40,360ms is the single largest entry. This is **not game code doing work**. It means the Go profiler was unable to record stack traces fast enough and dropped samples.

**Cause**: The profile was captured from a debug binary launched from VS Code's debugger. Debug builds disable optimizations, add instrumentation, and run the Delve debugger alongside the game. This overloads the profiler's sampling capacity, causing roughly 1 in 3 samples to be lost.

**Action**: This will disappear when profiling a release build outside the debugger.

---

## Finding 2: cgocall / syscall (26.43%) — Mostly VSync Wait

The 34,554ms in `runtime.cgocall` breaks down into two categories.

### 2a. IDXGISwapChain.Present — 29,733ms (85.9% of cgocall)

Call chain:

```
ebiten graphicscommand.(*commandQueue).flush
  -> directx.(*graphics11).End
    -> directx.(*graphicsInfra).present
      -> directx.(*_IDXGISwapChain).Present
        -> syscall.Syscall → runtime.cgocall
```

**Root cause**: `IDXGISwapChain.Present()` is the DirectX 11 call that submits the rendered frame to the display. With VSync enabled (Ebiten default), this call **blocks the CPU until the next monitor refresh**. At 60Hz, that is up to 16.67ms of idle wait per frame.

Over 120s at 60 FPS: `120 * 60 * ~4ms avg wait = ~28,800ms` — matches the observed 29,733ms almost exactly.

**This is normal and expected.** The CPU is sleeping, waiting for the monitor. It means the game finishes its work fast enough to have time left over for VSync.

### 2b. DirectX 11 Draw Calls + GLFW — ~4,800ms (13.9% of cgocall)

| DirectX / GLFW Call | Time |
|---------------------|------|
| `Proc.Call` (GLFW window management) | 3,513ms |
| `ID3D11DeviceContext.Map` | 322ms |
| `IDirectInput8W.EnumDevices` (gamepad) | 316ms |
| `ID3D11DeviceContext.OMSetRenderTargets` | 205ms |
| `PSSetShaderResources` | 84ms |
| `DrawIndexed` | 44ms |
| Various other DX11 state changes | ~315ms |

Within the 3,513ms `Proc.Call`:

- `_PeekMessageW` (Windows message pump): 2,357ms
- `_CallWindowProcW` (gamepad callbacks): 1,251ms
- `_GetKeyState`, `_SetCursor`, `_DispatchMessageW`, etc.: ~900ms

**These are mandatory costs of running a windowed game on Windows.** Not actionable.

---

## Finding 3: Memory Allocation Overhead (25.95%) — The Real Problem

`runtime.mallocgc` consumed 33,924ms, but **97% of that (33,028ms) was spent inside `runtime.profilealloc`** — profiler instrumentation, not actual allocation work.

```
runtime.mallocgc [33,924ms total]
  -> runtime.profilealloc [33,028ms — 97%]
    -> runtime.mProf_Malloc [32,986ms]
      -> runtime.callers [16,948ms]       (capturing stack traces)
        -> runtime.tracebackPCs [16,074ms]
      -> runtime.setprofilebucket [14,997ms]
```

**In a release build, actual allocation cost would be ~896ms** (33,924 - 33,028). However, the high allocation *rate* is real and worth investigating.

### Top Allocation Sources (runtime.newobject — 20,325ms cumulative)

| Caller | Alloc Time | Category |
|--------|-----------|----------|
| `ebitenui/image.NineSlice.drawTile` | 4,698ms | GUI background rendering |
| `ebitenui/widget.Text.draw` | 1,590ms | GUI text rendering |
| `ebiten EnqueueDrawTrianglesCommand` | 1,477ms | Ebiten internals |
| `ebiten ui.UserInterface.updateGame` | 1,158ms | Ebiten internals |
| `ebiten commandQueue.flush` | 1,111ms | Ebiten internals |
| `ebiten ui._GetMonitorInfoW` | 956ms | Monitor info (per-frame) |
| `ebiten ui._MonitorFromWindow` | 947ms | Monitor info (per-frame) |
| `gui/framework.GetSquadRenderInfo` | 453ms | Game code |
| `ebitenui/widget.Slider.SetupInputLayer` | 410ms | GUI input layers |
| `ebitenui/widget.ScrollContainer.SetupInputLayer` | 407ms | GUI input layers |

### Top Slice Growth Sources (runtime.growslice — 9,967ms cumulative)

| Caller | Time | Category |
|--------|------|----------|
| `directx.adjustUniforms` | 4,387ms | Ebiten internal |
| `ebitenui datastructures.Stack.Push` | 2,883ms | GUI color stack |
| `ui.monitors.append` | 945ms | Ebiten internal |
| `squads.SquadQueryCache.FindAllSquads` | 286ms | Game code |
| `squads.GetUnitIDsInSquad` | 59ms | Game code |

### Allocation Source Groups

- **ebitenui (GUI library)**: ~8,000ms+ — `NineSlice.drawTile`, `Text.draw`, `SetupInputLayer` all allocate heavily per frame rather than caching. Library design issue.
- **Ebiten engine internals**: ~5,000ms+ — `EnqueueDrawTrianglesCommand`, `commandQueue.flush`, monitor queries. Not directly controllable.
- **Game code**: ~500ms — `GetSquadRenderInfo` (453ms), `FindAllSquads` (286ms). Actionable.

---

## Finding 4: Rendering Call Distribution

`main.(*Game).Draw` total: 33,055ms

| Subsystem | Time | % of Draw |
|-----------|------|-----------|
| GUI (ebitenui) via `GameModeCoordinator.Render` | 28,566ms | 86.4% |
| Tile map (`DrawMapCentered`) | 3,029ms | 9.2% |
| Entity sprites (`ProcessRenderablesInSquare`) | 846ms | 2.6% |
| Start menu | 586ms | 1.8% |
| Visual effects | 9ms | 0.03% |

Within the GUI (28,566ms):

| GUI Widget | Time |
|------------|------|
| `Container.Render` | 22,549ms |
| `Text.Render` | 12,795ms |
| `Button.Render` | 12,119ms |
| `ScrollContainer.Render` | 6,165ms |
| `List.Render` | 5,292ms |

Game logic (`Update`) was only 1,351ms total — very fast.

---

## Conclusions

### Why are syscall and cgocall consuming so much time?

1. **VSync blocking (29,733ms / 85.9% of cgocall)**: `IDXGISwapChain.Present` blocks until monitor refresh. Normal, expected, desired. Not a problem.

2. **Windows message pump + GLFW (3,513ms / 10.2%)**: `PeekMessage`, `DispatchMessage`, input polling. Mandatory cost of a windowed application. Not actionable.

3. **DX11 state changes and draw submissions (~1,300ms / 3.8%)**: Buffer mapping, render targets, shader binding, draw calls. Scales with draw call count. ebitenui GUI is the primary source through `NineSlice.drawTile` and `Text.draw`.

### What is actually slow?

- **GUI rendering dominates** at 86.4% of draw time. ebitenui allocates heavily per frame.
- **Game logic is fast** at only 1,351ms over the entire 120s profile.
- **Debug profiling overhead** inflated `mallocgc` by 97% and caused 30.87% sample loss.

---

## Recommendations

### Immediate: Fix Profiling Methodology

Profile a release build outside the debugger to get accurate numbers:

```bash
go build -o game_main/game_main.exe game_main/*.go
```

Use `runtime/pprof` or `net/http/pprof` to capture a profile while the release binary runs. This eliminates `lostProfileEvent` (30.87%) and `profilealloc` overhead (25.27%).

### Investigate: Game Code Allocations

- `GetSquadRenderInfo` (453ms) — consider caching or pre-allocating return structs
- `FindAllSquads` (286ms) — pre-allocate slice capacity to avoid `growslice`
- `GetUnitIDsInSquad` (59ms) — pre-allocate slice capacity

### Monitor: ebitenui Allocation Patterns

The ebitenui library allocates heavily per frame (~8,000ms+ of alloc time). While this is largely a library design issue, reducing the number of visible widgets (especially `NineSlice` backgrounds and `Text` widgets) will reduce both allocation pressure and draw call count.
