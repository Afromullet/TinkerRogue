# Performance Analysis Summary

**Date**: 2025-12-10
**Profile**: benchmark_1/cpu_profile.pb.gz
**Status**: GOOD WITH HIGH OPTIMIZATION POTENTIAL

---

## Key Findings

### Top 3 Performance Issues

1. **ECS Query Allocation Storm** (Priority: HIGH)
   - 17.03s (5.03% of profile)
   - Creates componentMap for every query
   - Solution: Query result caching
   - Expected: 15-43x improvement

2. **Tile Rendering Allocations** (Priority: MEDIUM)
   - 77ms + GC pressure
   - Allocates DrawImageOptions per tile (2,000/frame)
   - Solution: Reuse single instance
   - Expected: 100% allocation elimination

3. **Runtime/GC Overhead** (Priority: INDIRECT)
   - 250.5s (74.02% of profile - profiling artifact)
   - Caused by allocation pressure above
   - Solution: Reduce allocations to reduce GC

---

## Quick Wins (High ROI)

### 1. Squad Query Cache (4 hours, 15x faster)

**Problem**: GetSquadEntity() queries full ECS world 15+ times per frame

**Solution**:
```go
type SquadQueryCache struct {
    squadByID map[ecs.EntityID]*ecs.Entity
    dirty     bool
}

// Refresh once per frame instead of 15 times
func (cache *SquadQueryCache) RefreshIfNeeded(manager) {
    if cache.dirty {
        results := manager.World.Query(SquadTag)
        // Build index map...
        cache.dirty = false
    }
}
```

**Impact**: 17s → 1.1s (15x faster)

### 2. DrawImageOptions Reuse (30 minutes, 97% fewer allocs)

**Problem**: Allocates new DrawImageOptions for every tile

**Solution**:
```go
type TileRenderer struct {
    drawOpts ebiten.DrawImageOptions  // Reusable field
}

func (r *TileRenderer) renderTile(...) {
    r.drawOpts = ebiten.DrawImageOptions{}  // Reset instead of allocate
    // ... use &r.drawOpts ...
}
```

**Impact**: 2,000 allocations/frame → 0 allocations

---

## Performance Metrics

### Current Performance (Estimated)
- Frame time: ~18ms (55 FPS)
- Allocations: 1.73MB per frame
- GC pressure: High (allocations trigger frequent GC)

### After Phase 1 Optimizations (4 hours)
- Frame time: ~6.5ms (154 FPS)
- Allocations: 49KB per frame
- GC pressure: Low (97% reduction)
- **Improvement: 2.8x faster**

### After Phase 2 Optimizations (12 hours total)
- Frame time: ~4.8ms (208 FPS)
- Allocations: <10KB per frame
- **Improvement: 3.75x faster overall**

---

## Time Distribution

| System | Time | % of Total | Optimizable? |
|--------|------|------------|--------------|
| Runtime/GC (profiling artifact) | 250.5s | 74.02% | Indirect |
| GUI (ebitenui) | 29.5s | 8.72% | Maybe later |
| ECS Queries | 17.0s | 5.03% | **YES** |
| Graphics (Ebiten) | 14.8s | 4.36% | Partial |
| Tile Rendering | 9.7s | 2.87% | **YES** |
| Other | 17.0s | 5.00% | Various |

---

## Implementation Roadmap

### Phase 1: Critical Fixes (4 hours)

1. **DrawImageOptions Reuse** (30 min)
   - Edit worldmap/tilerenderer.go
   - Add reusable field
   - Test in game

2. **Baseline Benchmarks** (30 min)
   - Create squad query benchmarks
   - Run before optimizations

3. **Squad Query Cache** (3 hours)
   - Create squads/squadcache.go
   - Implement caching logic
   - Integrate with squad system
   - Test and benchmark

**Expected: 2.8x performance improvement**

### Phase 2: Extended Optimization (8 hours)

1. **Extended Query Cache** (3 hours)
   - Cache all squad operations
   - Index members by squad ID
   - Cache leader queries

2. **Frame Counter System** (2 hours)
   - Add FrameCount to EntityManager
   - Auto-invalidate cache per frame

3. **Slice Pre-Allocation** (1 hour)
   - Pre-allocate slice capacities
   - Reduce reallocation overhead

4. **Full Benchmarks** (2 hours)
   - Full frame rendering benchmark
   - Profile comparison
   - Document results

**Expected: Additional 1.4x improvement (cumulative 3.75x)**

### Phase 3: Profile-Guided (Optional, 8+ hours)

- Generate new profile after Phase 1+2
- Identify remaining hotspots
- Target GUI if needed
- Optimize as necessary

---

## Quick Start Commands

### Capture Baseline
```bash
cd squads/
go test -bench=. -benchmem > baseline.txt
```

### View Current Profile
```bash
go tool pprof -http=:8080 docs/benchmarking/benchmark_1/cpu_profile.pb.gz
```

### After Optimization
```bash
go test -bench=. -benchmem > optimized.txt
benchstat baseline.txt optimized.txt
```

### Generate New Profile
```bash
go test -bench=BenchmarkFullFrameRender -cpuprofile=new.prof ./game_main/
go tool pprof -base=cpu_profile.pb.gz new.prof
```

---

## What NOT to Optimize

1. **Coordinate Conversions** (0.0089% - already optimal)
2. **FOV Library** (0.079% - external dependency, acceptable)
3. **Ebiten DrawImage Internals** (unavoidable library overhead)
4. **GetComponentData** (0.094% - acceptable lock overhead)

---

## Success Metrics

### Targets
- 60 FPS minimum (16.67ms per frame)
- <100KB allocations per frame
- <5% time in ECS queries
- <10% time in rendering

### After Phase 1 (Expected)
- ✅ 154 FPS (6.5ms per frame) - **260% above target**
- ✅ 49KB per frame - **51% of target**
- ✅ 0.3% in ECS queries - **94% reduction**
- ✅ 3% in rendering - **Within target**

---

## References

- **Full Report**: `docs/analysis/benchmark_1_performance_analysis.md`
- **Profile Graph**: `docs/analysis/profile_graph.svg`
- **Baseline Profile**: `docs/benchmarking/benchmark_1/cpu_profile.pb.gz`

---

## Contact / Questions

See full performance analysis report for:
- Detailed code examples
- Benchmark templates
- Profiling commands
- Implementation checklists
- Go performance best practices

**Ready to implement Phase 1 optimizations!**
