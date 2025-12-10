# Performance Analysis Documentation

This directory contains performance analysis reports and optimization guides for TinkerRogue.

---

## Documents

### 1. [benchmark_1_performance_analysis.md](benchmark_1_performance_analysis.md)

**FULL TECHNICAL REPORT** (60+ pages)

Comprehensive performance analysis including:
- Executive summary with key findings
- Detailed hotspot analysis with line numbers
- ECS query performance breakdown
- Graphics/rendering optimization strategies
- Allocation analysis and reduction techniques
- Spatial grid performance evaluation
- Caching opportunities and strategies
- Complete benchmark generation templates
- Profiling command reference
- Implementation roadmap (Phases 1-3)
- Go performance best practices

**Use this when**: You need detailed technical information, code examples, or profiling commands.

---

### 2. [PERFORMANCE_SUMMARY.md](PERFORMANCE_SUMMARY.md)

**QUICK REFERENCE** (5 pages)

Executive summary with:
- Top 3 performance issues
- Quick wins (high ROI optimizations)
- Performance metrics (before/after)
- Time distribution breakdown
- Implementation roadmap overview
- Quick start commands
- Success metrics

**Use this when**: You need a quick overview or are explaining performance issues to someone else.

---

### 3. [OPTIMIZATION_CHECKLIST.md](OPTIMIZATION_CHECKLIST.md)

**ACTION PLAN** (10 pages)

Step-by-step implementation guide:
- Phase 1 tasks (4 hours, 2.8x improvement)
- Phase 2 tasks (8 hours, 3.75x total improvement)
- Phase 3 tasks (optional, profile-guided)
- Checkboxes for tracking progress
- Success criteria for each phase
- Common pitfalls to avoid
- Rollback plan if issues occur
- Quick reference commands

**Use this when**: You're ready to implement optimizations and need a structured checklist.

---

### 4. [profile_graph.svg](profile_graph.svg)

**VISUAL PROFILE** (generated from CPU profile)

Interactive call graph showing:
- Function call relationships
- Time spent in each function
- Hotspot visualization
- Call flow analysis

**Use this when**: You want a visual representation of the performance profile.

---

## Quick Navigation

**Just want to get started?**
→ Read [PERFORMANCE_SUMMARY.md](PERFORMANCE_SUMMARY.md) (5 min)
→ Follow [OPTIMIZATION_CHECKLIST.md](OPTIMIZATION_CHECKLIST.md) Phase 1 (4 hours)

**Need detailed analysis?**
→ Read [benchmark_1_performance_analysis.md](benchmark_1_performance_analysis.md) full report

**Want to understand the profile visually?**
→ Open [profile_graph.svg](profile_graph.svg) in browser

---

## Key Findings Summary

### Top 2 Performance Issues

1. **ECS Query Allocation Storm** (HIGH PRIORITY)
   - 17.03s cumulative time (5.03% of profile)
   - Solution: Query result caching
   - Expected: 15-43x improvement
   - Effort: 3 hours

2. **Tile Rendering Allocations** (MEDIUM PRIORITY)
   - 77ms + GC pressure
   - 2,000 allocations per frame
   - Solution: Reuse DrawImageOptions
   - Expected: 100% allocation elimination
   - Effort: 30 minutes

### Expected Performance Gains

**Phase 1** (4 hours):
- Frame time: 18ms → 6.5ms (2.8x faster)
- Allocations: 1.73MB → 49KB (97% reduction)
- FPS: 55 → 154

**Phase 2** (12 hours total):
- Frame time: 6.5ms → 4.8ms (3.75x total)
- Allocations: 49KB → <10KB
- FPS: 154 → 208

---

## Implementation Order

Follow this sequence for best results:

1. **Read Summary** (5 min)
   - Understand key issues
   - Review expected gains

2. **Capture Baseline** (30 min)
   - Run existing benchmarks
   - Save baseline performance

3. **Phase 1: Quick Wins** (4 hours)
   - Implement DrawImageOptions reuse (30 min)
   - Implement squad query cache (3 hours)
   - Verify 2.8x improvement

4. **Phase 2: Extended** (8 hours)
   - Extended caching
   - Frame counter system
   - Full benchmarks

5. **Phase 3: Refinement** (optional)
   - Profile-guided optimization
   - Target remaining hotspots

---

## Commands Cheat Sheet

### Benchmarking
```bash
# Baseline
go test -bench=. -benchmem ./squads/ > baseline.txt

# After optimization
go test -bench=. -benchmem ./squads/ > optimized.txt

# Compare
benchstat baseline.txt optimized.txt
```

### Profiling
```bash
# View profile
go tool pprof -http=:8080 ../benchmarking/benchmark_1/cpu_profile.pb.gz

# Compare profiles
go tool pprof -base=before.prof after.prof

# Generate graph
go tool pprof -svg cpu.prof > graph.svg
```

### Testing
```bash
# All tests
go test ./...

# Specific package
go test ./squads/...

# With coverage
go test -cover ./...
```

---

## Success Metrics

### Target Performance
- Frame time: <16.67ms (60 FPS minimum)
- Allocations: <100KB per frame
- ECS queries: <5% of frame time

### After Phase 1 (Expected)
- ✅ Frame time: 6.5ms (154 FPS) - 260% above target
- ✅ Allocations: 49KB - 51% of target
- ✅ ECS queries: 0.3% - 94% reduction

---

## Related Documents

- **ECS Best Practices**: `../ecs_best_practices.md`
- **Developer Guide**: `../../CLAUDE.md`
- **Original Profile**: `../benchmarking/benchmark_1/cpu_profile.pb.gz`

---

## Questions or Issues?

Refer to these sections in the full report:

- **Profile doesn't match expectations?** → See "Profiling Recommendations" section
- **Optimization not working?** → See "Common Pitfalls" in checklist
- **Need benchmark templates?** → See "Benchmark Generation" section
- **Cache invalidation issues?** → See "Caching Strategies" section
- **Visual regressions?** → See "Rollback Plan" in checklist

---

**Last Updated**: 2025-12-10
**Profile Version**: benchmark_1 (120.20s duration, 338.43s samples)
**Status**: Ready for Phase 1 implementation
