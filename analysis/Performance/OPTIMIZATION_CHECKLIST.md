# Performance Optimization Checklist

**Profile**: benchmark_1/cpu_profile.pb.gz (2025-12-10)
**Target**: 3.75x performance improvement (55 FPS → 208 FPS)

---

## Phase 1: Critical Optimizations (4 hours) ⚠️ START HERE

### Task 1.1: Capture Baseline Benchmarks (30 minutes)

- [ ] Create `squads/squadsystem_bench_test.go`
- [ ] Implement BenchmarkGetSquadEntityNoCache
- [ ] Implement BenchmarkGetUnitIDsInSquadNoCache
- [ ] Run: `go test -bench=. -benchmem ./squads/ > baseline.txt`
- [ ] Save baseline.txt to docs/benchmarking/benchmark_1/

**Deliverable**: Baseline performance metrics

---

### Task 1.2: DrawImageOptions Reuse (30 minutes)

**File**: `worldmap/tilerenderer.go`

- [ ] Add field to TileRenderer struct:
  ```go
  drawOpts ebiten.DrawImageOptions
  ```

- [ ] Modify renderTile() function:
  - [ ] Replace `drawOpts := &ebiten.DrawImageOptions{}`
  - [ ] With `r.drawOpts = ebiten.DrawImageOptions{}`
  - [ ] Update all references to use `&r.drawOpts`

- [ ] Test changes:
  - [ ] `go test ./worldmap/...`
  - [ ] Run game visually (verify rendering)
  - [ ] Check for visual glitches

**Expected**: 97% allocation reduction (2,000 allocs → 0)

**Deliverable**: Zero allocation tile rendering

---

### Task 1.3: Squad Query Cache Implementation (3 hours)

#### Step 1: Create Cache Structure (30 minutes)

**Create**: `squads/squadcache.go`

- [ ] Implement SquadQueryCache struct:
  ```go
  type SquadQueryCache struct {
      allSquads      []*ecs.QueryResult
      allMembers     []*ecs.QueryResult
      squadByID      map[ecs.EntityID]*ecs.Entity
      membersBySquad map[ecs.EntityID][]ecs.EntityID
      dirtySquads    bool
      dirtyMembers   bool
      lastFrameCount int
  }
  ```

- [ ] Implement NewSquadQueryCache()
- [ ] Implement RefreshIfNeeded(manager)

#### Step 2: Cached Query Functions (1 hour)

- [ ] Implement GetSquadEntity (cached)
- [ ] Implement GetUnitIDsInSquad (cached)
- [ ] Implement GetLeaderID (cached)
- [ ] Add manual invalidation methods:
  - [ ] InvalidateSquads()
  - [ ] InvalidateMembers()

#### Step 3: Integration (1 hour)

**File**: Create `squads/squadsystem.go` or modify existing

- [ ] Add SquadSystem struct:
  ```go
  type SquadSystem struct {
      manager *common.EntityManager
      cache   *SquadQueryCache
  }
  ```

- [ ] Create NewSquadSystem()
- [ ] Wrap all query functions to use cache
- [ ] Add cache invalidation to:
  - [ ] CreateSquad()
  - [ ] AddMemberToSquad()
  - [ ] RemoveMemberFromSquad()
  - [ ] DeleteSquad()

#### Step 4: Testing (30 minutes)

- [ ] Run existing tests: `go test ./squads/...`
- [ ] Create benchmark with cache:
  - [ ] BenchmarkGetSquadEntityWithCache
  - [ ] BenchmarkGetUnitIDsInSquadWithCache
- [ ] Run benchmarks: `go test -bench=. -benchmem ./squads/ > optimized.txt`
- [ ] Compare: `benchstat baseline.txt optimized.txt`

**Expected**: 15-43x improvement in squad queries

**Deliverable**: Cached squad system with benchmarks

---

### Phase 1 Completion Checklist

- [ ] All tasks completed
- [ ] Tests passing: `go test ./...`
- [ ] Benchmarks captured (before/after)
- [ ] Performance improvement verified (>2x)
- [ ] No visual regressions in game
- [ ] Code committed with benchmark results

**Target**: 2.8x overall improvement (18ms → 6.5ms per frame)

---

## Phase 2: Extended Optimization (8 hours)

### Task 2.1: Extend Cache to All Squad Operations (3 hours)

**File**: `squads/squadcache.go`

- [ ] Add leader index cache
- [ ] Add grid position cache
- [ ] Implement GetUnitIDsAtGridPosition (cached)
- [ ] Implement GetUnitIDsInRow (cached)
- [ ] Implement GetSquadMovementSpeed (cached)

**Expected**: Additional 5-10s improvement

---

### Task 2.2: Frame Counter System (2 hours)

**File**: `common/EntityManager.go`

- [ ] Add FrameCount field to EntityManager
- [ ] Implement IncrementFrame() method
- [ ] Integrate with game loop Update()

**File**: `squads/squadcache.go`

- [ ] Update RefreshIfNeeded() to use FrameCount
- [ ] Auto-invalidate cache once per frame

**Expected**: Simplified cache invalidation

---

### Task 2.3: Slice Pre-Allocation (1 hour)

**Files**: All functions in `squads/squadqueries.go`

- [ ] FindAllSquads - add capacity hint
- [ ] GetUnitIDsInSquad - add capacity hint
- [ ] GetUnitIDsInRow - add capacity hint
- [ ] GetUnitIDsAtGridPosition - add capacity hint

**Pattern**:
```go
// Before
units := make([]ecs.EntityID, 0)

// After
results := manager.World.Query(tag)
units := make([]ecs.EntityID, 0, len(results))
```

**Expected**: 10-20ms reduction in slice reallocations

---

### Task 2.4: Full Frame Benchmarks (2 hours)

**Create**: `game_main/game_bench_test.go`

- [ ] Implement SetupTestGame() helper
- [ ] Implement BenchmarkFullFrameRender
- [ ] Implement BenchmarkFullFrameUpdate
- [ ] Run benchmarks with CPU profile:
  ```bash
  go test -bench=BenchmarkFullFrame -cpuprofile=optimized.prof -benchmem ./game_main/
  ```
- [ ] Compare with original profile:
  ```bash
  go tool pprof -base=cpu_profile.pb.gz optimized.prof
  ```
- [ ] Document results in benchmark notes

**Expected**: Full frame performance metrics

---

### Phase 2 Completion Checklist

- [ ] All extended caching implemented
- [ ] Frame counter system integrated
- [ ] Slice pre-allocation applied
- [ ] Full benchmarks captured
- [ ] Profile comparison generated
- [ ] Tests passing: `go test ./...`
- [ ] Code committed with full analysis

**Target**: Additional 1.4x improvement (cumulative 3.75x total)

---

## Phase 3: Profile-Guided Refinement (8+ hours, optional)

### Task 3.1: Generate New Profile (1 hour)

- [ ] Run optimized game with profiling enabled
- [ ] Capture 2-minute CPU profile
- [ ] Generate profile graph: `go tool pprof -svg -nodecount=50`
- [ ] Compare with baseline profile

---

### Task 3.2: Identify Remaining Hotspots (2 hours)

- [ ] Analyze new profile top 20 functions
- [ ] Look for functions consuming >2% CPU
- [ ] Check allocation sites with memory profile
- [ ] Prioritize remaining optimizations

---

### Task 3.3: Target-Specific Optimizations (5+ hours)

**IF GUI is bottleneck (>10% CPU)**:
- [ ] Profile ebitenui rendering
- [ ] Cache text rendering
- [ ] Implement dirty rectangle optimization
- [ ] Reduce update frequency

**IF Combat is bottleneck (>5% CPU)**:
- [ ] Profile combat calculations
- [ ] Cache combat query results
- [ ] Optimize damage formulas
- [ ] Reduce nested loops

**IF Pathfinding is bottleneck (>5% CPU)**:
- [ ] Profile pathfinding algorithm
- [ ] Implement A* caching
- [ ] Reduce search space
- [ ] Use incremental pathfinding

---

### Phase 3 Completion Checklist

- [ ] New profile analyzed
- [ ] Remaining hotspots addressed
- [ ] Additional benchmarks captured
- [ ] Final profile comparison
- [ ] Performance targets achieved
- [ ] Documentation updated

**Target**: Situational improvements based on new profile

---

## Continuous Monitoring Setup

### Add Performance Metrics to Game

**File**: `game_main/performance.go`

- [ ] Implement PerformanceMonitor struct
- [ ] Add FPS counter to debug overlay
- [ ] Track frame time histogram
- [ ] Log performance warnings (frame >16.67ms)

### Enable pprof HTTP Endpoint

**File**: `game_main/main.go`

- [ ] Import `_ "net/http/pprof"`
- [ ] Start pprof server: `http.ListenAndServe("localhost:6060", nil)`
- [ ] Document access URL in README

### Create Benchmark Suite

- [ ] Add benchmark targets to Makefile
- [ ] Document benchmark commands
- [ ] Set up benchmark history tracking
- [ ] Create benchstat comparison script

---

## Success Criteria

### Phase 1 Success (REQUIRED)

- ✅ Frame time: <10ms (100+ FPS)
- ✅ Allocations: <100KB per frame
- ✅ ECS query time: <1s cumulative
- ✅ Benchmark improvement: >2x

### Phase 2 Success (REQUIRED)

- ✅ Frame time: <7ms (142+ FPS)
- ✅ Allocations: <50KB per frame
- ✅ ECS query time: <0.5s cumulative
- ✅ Benchmark improvement: >3x cumulative

### Phase 3 Success (OPTIONAL)

- ✅ Frame time: <5ms (200+ FPS)
- ✅ No single function consuming >2% CPU
- ✅ Allocation rate <10KB per frame
- ✅ All systems optimized

---

## Common Pitfalls to Avoid

### During Implementation

- ❌ Modifying external libraries (bytearena/ecs, ebitenui)
- ❌ Optimizing without profiling first
- ❌ Breaking existing functionality
- ❌ Introducing visual regressions
- ❌ Complicating cache invalidation

### During Testing

- ❌ Testing with unrealistic data (too few entities)
- ❌ Ignoring allocation benchmarks
- ❌ Skipping visual verification
- ❌ Not comparing with baseline
- ❌ Assuming improvements without measurement

### During Deployment

- ❌ Committing without benchmarks
- ❌ Missing cache invalidation calls
- ❌ Forgetting to update documentation
- ❌ Not testing in release build
- ❌ Skipping integration tests

---

## Quick Reference Commands

### Benchmarking
```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific package
go test -bench=. -benchmem ./squads/

# Compare before/after
benchstat baseline.txt optimized.txt

# With CPU profile
go test -bench=. -cpuprofile=cpu.prof -benchmem ./squads/
```

### Profiling
```bash
# View profile in browser
go tool pprof -http=:8080 cpu_profile.pb.gz

# Compare profiles
go tool pprof -base=before.prof after.prof

# Generate SVG graph
go tool pprof -svg cpu_profile.pb.gz > graph.svg

# List specific function
go tool pprof -list=renderTile cpu_profile.pb.gz
```

### Testing
```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./squads/...

# Race detector
go test -race ./...
```

---

## Rollback Plan (If Issues Occur)

### If Phase 1 Breaks Game

1. Revert DrawImageOptions changes:
   - [ ] Restore original `drawOpts := &ebiten.DrawImageOptions{}`
   - [ ] Test rendering

2. Disable squad cache:
   - [ ] Bypass cache in SquadSystem
   - [ ] Use original query functions
   - [ ] Investigate cache invalidation issues

### If Performance Degrades

1. Compare profiles:
   - [ ] Check new hotspots introduced
   - [ ] Verify cache overhead not excessive
   - [ ] Review allocation patterns

2. Rollback changes:
   - [ ] Revert to last working commit
   - [ ] Re-analyze profile
   - [ ] Adjust optimization strategy

---

## Notes Section (Record Issues/Decisions)

### Issues Encountered

(Space for recording problems during implementation)

---

### Performance Notes

(Space for recording actual benchmark results)

---

### Future Optimization Ideas

(Space for noting additional optimization opportunities discovered)

---

**Ready to start Phase 1! Begin with Task 1.1 (Baseline Benchmarks)**
