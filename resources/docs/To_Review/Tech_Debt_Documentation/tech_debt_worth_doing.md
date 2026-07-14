# TinkerRogue — What's Actually Worth Doing

**Date:** 2026-07-09
**Scope:** Reality-check of `tech_debt_codebase.md` (2026-07-08) against the actual constraints of this project — solo developer, hobby tactical roguelike, pre-release, no users, no team, no deadline.
**Verdict count:** 21 do / 15 eventually / 10 skip (46 verdicts covering the doc's 45 roadmap items — Phase 3 item 12 is split into a CI half and a font half, each ruled separately).

## The frame

Every item below was judged against exactly three questions. If an item doesn't clearly pass one of them, it's noise no matter what severity label the original report gave it.

1. Does it protect the solo dev's own playtesting/iteration loop (crashes, corrupted saves, misleading state during testing)?
2. Does it prevent losing work (save data, git history, reproducibility of balance analysis)?
3. Does it remove friction the developer actually hits — not friction a hypothetical team or hypothetical future user would hit?

Two corrections to the source report, verified against the working tree before writing anything below:

- **The CRLF "202 phantom files" crisis is not currently real.** `.gitattributes` already contains `* text=auto` (present since the initial commit), and `git status` right now shows a clean tree (one real, non-CRLF diff — the tech-debt doc itself mid-edit). The 202-file figure was a snapshot of a transient uncommitted state at analysis time, not a persistent problem. The fix is still free, so it stays on the do-list, just demoted from "quick win" to "15-minute verification."
- **`game_main/setup.go:202` and `main.go:92-98` are exactly as described.** Read directly: `SetupRoguelikeFromSave` does `log.Fatalf` on `EnterTactical` failure at line 202, and `main.go`'s `pendingLoad` branch (lines 92-98) calls `SetupRoguelikeFromSave(g)` on the already-running game with no reset. `finishRaid` (`campaign/raid/raidrunner.go:339-343`) is confirmed to do exactly two things — clear a callback and zero an ID — and dispose nothing. `core/config/config.go:13,16` confirms `DEBUG_MODE = true` and `ENABLE_BENCHMARKING = true` as committed consts. These are real.

---

## Do these

Ordered by payoff-per-hour. Grouped where several tiny fixes live in the same debugging session.

**1. Knock out the six trivial crash/correctness fixes in one sitting (~3-4h total).** These are all 10-30 minute mechanical changes with an outsized "stops a real crash or a real silent bug" payoff:
   - Open `game_main/setup.go:202` — replace `log.Fatalf` with a returned wrapped error so the load-game fallback in `main.go` actually works.
   - Open `templates/entity_factory.go:37` — replace `log.Fatal` with a returned error (restores the templates package's own panic-free rule).
   - Open `gui/framework/contextstate.go:75-98` — add `PostCombatReturnMode` to `TacticalState.Reset()`.
   - Open `gui/guicombat/combatbase/handler.go:133` — fix `fmt.Errorf(result.Error)` to `errors.New(result.Error)`; delete the 4 confirmed-dead exported methods (`BaseShape.CanRotate/StartPositionPixels/SetDirection`, `CommanderRosterData.RemoveCommander`).
   - Open `gui/guiexploration/explorationmode.go:59` and `gui/guioverworld/overworldmode.go:94` — pass `RootContainer` instead of nil to `SubMenuController`, matching the ebitenui hidden-`ScrollContainer` input-layer gotcha already documented in your own memory notes.
  

**2. Guard the in-game Load button today (~1h).** This is one click away from duplicating every entity in your current session. First step: `gui/guiexploration/exploration_panels_registry.go:194-199` — disable or hide the Load action while a game is already running, until item 5 below lands.

**3. Seed the RNGs so balance runs are reproducible (~1-2h).** First step: `mind/combatlifecycle/reward.go:21` — seed `xpRNG` from `core/common/randnumgen.go`'s seeded PCG instead of `time.Now().UnixNano()`. Then call `SetRNGSeed`/`SetXPRNG` at the top of the `tools/combat_analysis` simulator entry point. Without this, every Monte-Carlo balance run you do produces different numbers for reasons that have nothing to do with the balance change you're testing — that's an hour lost to "wait, did I actually change anything?" every time you use the tool.

**4. Do the real fix for in-game Load: reset the world before loading (~1-2 days).** First step: in `main.go`, before the `SetupRoguelikeFromSave(g)` calls at both line 83 and line 94, add a `ResetWorld(g)` step that disposes all existing entities and calls `common.GlobalPositionSystem.Clear()`. This turns item 2's guard into an actual working feature instead of a permanently-disabled button.

**5. Fix the save-chunk versioning gap, then add round-trip tests (~3-4 days combined, do in this order).** First step: wire the already-implemented-but-unused `ChunkVersion()` into `SaveEnvelope` so it's actually serialized and checked on load (`setup/savesystem/savesystem.go`) — this is not hypothetical, `progression_chunk.go:29` already returns version 2 while every other chunk returns 1, meaning a real schema change already shipped with zero detection. Then write the round-trip test per chunk (build via `testing/` fixtures → save → load → assert equality) for all 8 serializers in `setup/savesystem/chunks/`. Do the version wiring first — testing round-trips without version detection tests the wrong invariant.

**6. Dispose raid garrison entities in `finishRaid` (~1 day).** First step: `campaign/raid/raidrunner.go:339` — walk floors → rooms → garrison/reserve squads → units → room/alert/floor-state/raid-state entities and call `CleanDisposeEntity` on each, mirroring the pattern already correct in `combat_service.go:208-337`. Every raid you run today (finished or abandoned) permanently leaks entities into the session.

**7. Reset the threat/AI/visualization faction maps at combat teardown (~half day).** First step: `mind/behavior/dangerlevel.go:15` — add a `Reset()` to `FactionThreatLevelManager`, call it (and clear the two `layerEvaluators` maps in `mind/ai/ai_controller.go` and `gui/guicombat/combatvisualization/manager.go`) from wherever combat teardown already runs. If you run many test combats in one sitting for balance work, this is the one that quietly corrupts your own AI/threat state without a crash to tell you.

**8. Add validators for `monsterdata.json`/`unitspells.json`, promote dangling spell refs to fatal (~1 day).** First step: `templates/readdata.go:32-35` (`monsterDataLoader` has no validator at all). The fatal-on-bad-reference pattern already exists for nodes/encounters/factions (`validation.go:132-166`) — copy it. Right now a unit referencing a deleted spell boots clean and only fails silently at cast time, mid-playtest.

**9. Convert `GameMapUtil.go`'s missing-asset panics to returned errors (~half day).** First step: `world/worldmapcore/GameMapUtil.go:52,68,74`. A missing tile asset during New Game hard-crashes today instead of surfacing a message — this bites exactly when you're iterating on map art.

**10. Make the balance-config loaders fail loud instead of silently zeroing (~2-3h, scoped down from the doc's full 2-day "unify four loaders" item).** First step: `tactical/combat/combatmath/balanceconfig.go:31-45` — `validateCombatBalance` currently only prints a `WARNING` and leaves the global at zero values. A typo in a balance JSON file while you're mid-tuning silently zeroes your attack math and the game "just runs wrong" with no error — that's the exact silent-failure-eats-an-evening scenario this report should be optimizing against. You don't need the full four-loader unification to fix this one.

**11. Make `testing/fixtures.go` delegate to production's `registerCoreComponents`/`buildCoreTags` (~half day).** First step: `testing/fixtures.go:28-33` vs `setup/gamesetup/ecsinit.go:38-56` — two independently maintained lists mean your test suite (57.5% test:source ratio in `tactical/` — real coverage you rely on) can silently diverge from what production actually does. This protects the value of tests you've already written, not new ones.

**12. Replace the 4.25MB generated font `.go` file with `//go:embed` on the `.ttf` (~1-2h).** First step: `resources/assets/fonts/mplus1pregular.go`. Independent of any CI work — this is pure "stop dragging 4.25MB through every clone, log, and diff." Do this without setting up CI.

**13. Quick-align the overworld event-key drift (~2-3h, not the full typed-struct refactor).** First step: `campaign/overworld/overworldlog/overworld_summary.go` lines 60/84/94/142/145 read `threat_type`/`intensity`/`tiles_gained` keys that `mind/encounter/resolvers.go` never writes. Just rename the reads to match what's actually produced (`new_intensity`, `intensity_reduced`, `victory`). This revives a summary feature you presumably built because you wanted it, without the bigger typed-event-struct project.

**14. Verify the CRLF fix is already effective, then stop worrying about it (~15 min).** First step: `git add --renormalize . --dry-run` at repo root. If it comes back empty (it should, per the fact-check above), the Phase 1 `.gitattributes` item is already done — mark it closed instead of re-doing work.

**15. Migrate the deprecated ebiten APIs (~half day, deferred out of item 1).** Item 1's `staticcheck` sweep surfaced 28 `SA1019` deprecation hits, which are currently suppressed via `staticcheck.conf` so the tool reports a clean tree. They are a real migration, not noise, and they will eventually break on an ebiten upgrade:
   - `ebiten/v2/text` → `text/v2`, plus `text.BoundString` → font metrics (`gui/guinodeplacement/nodeplacement_renderer.go`, `gui/guiraid/floormap_renderer.go`)
   - `opts.ColorM` → `ColorScale` / the `colorm` package (`visual/vfx/renderers.go`, 10 hits)
   - `Image.Dispose` → `Deallocate` (`gui/widgetresources/cachedbackground.go`, `gui/widgets/cached_list.go`, `gui/widgets/cached_textarea.go`)
   - `strings.Title` → `golang.org/x/text/cases` (`gui/guisquads/artifact_refresh.go`, 6 hits)
   - `ebiten.DeviceScaleFactor` / `ebiten.SetWindowResizable` (`game_main/main.go`)

   Do this when you next touch rendering, and drop `-SA1019` from `staticcheck.conf` once it's done. Not urgent: deprecated ≠ broken, and none of these is a crash today.

**16. Run a dedicated repo-wide `go fmt ./...` pass (~15 min, as its own commit).** 86 `.go` files are not gofmt-clean — misaligned comment blocks and struct fields, plus a stray trailing blank line at EOF. Verified semantically inert: for each file, `gofmt(committed)` and `gofmt(working)` are byte-identical, so reformatting changes nothing but whitespace.

   The catch is that a repo-wide `go fmt` rewrites all 86 at once. Done casually mid-feature, it buries the real change in formatting noise (this happened during item 1 — the six-fix diff ballooned from 27 files to 113 and had to be unwound). So:
   - Do it as a **standalone commit that touches nothing else**, so the blame churn is isolated and reviewable as "formatting only."
   - Until then, format only the files you actually edited: `gofmt -w <file>`. Never `go fmt ./...` while you have functional changes in the tree.

---

## Do it eventually

Worth doing, but only when you're already touching that area — not worth carving out dedicated time for now.

| Item | Trigger |
|---|---|
| Generic typed panel-registration helper (Phase 2 #2) | Next time you're adding 2-3 new panels and feel the copy-paste pain of the `mode.(*XxxMode)` assertion dance — migrate `squadeditor`/`combat` first since they're worst. |
| Core-path tests for `campaign/raid` (Phase 2 #3) | Next time you touch raid-runner state transitions or deployment logic for a feature reason — write the test alongside the change, don't do a dedicated test sprint. |
| Split `squadcore/squadqueries.go` into 3 files (Phase 2 #4) | Next time you're already editing that file for an unrelated reason — it's a pure file move, fold it into that visit. |
| Cache per-frame combat render info / flood-fill (Phase 2 #7) | Only if you actually notice stutter or input lag while in move mode with a large squad/map. No profiling evidence exists that this is currently a problem — don't chase it speculatively. |
| Table-driven tests for `combatmath`/`combatstate` (Phase 3 #1) | Next time you change a combat formula — write the test with the change so a regression is loud instead of "combat feels different" three weeks later. |
| Consolidate `gui/builders` option plumbing (Phase 3 #2) | Next time you're adding a widget constructor and about to copy-paste from `panels.go`/`widgets.go` — refactor the piece you're touching, not the whole 1,700-line surface. |
| Extract shared overworld `Execute*` scaffolding (Phase 3 #3) | Next time you add a 7th overworld action — extract the helper then, when you have 7 examples instead of 6 to generalize from. |
| Ship the AI-commander-spellcasting feature (Phase 3 #7) | This is a real feature gap, not debt — schedule it like any other feature when you're next in `mind/ai`. |
| Finish unifying the three RNGs (Phase 3 #8) | Item 3 above (seed `xpRNG` + the simulator) already closes the part that matters for reproducibility. The remaining UI-local RNG in `guiunitview` doesn't affect determinism — fold it in next time you're in that file. |
| Mode-name typed constants (Phase 3 #13) | You haven't actually been bitten by this yet (no observed bug, just a latent risk from the double-declaration). Next time you add a new mode, at minimum sanity-check the name string matches; do the full typed refactor only if a real mode-name typo actually costs you a debugging session. |
| Unify sub-mode enter/exit API + add artifact `BlockReason` (Phase 3 #14) | The missing artifact "why not" feedback is the one concrete piece worth doing — do it next time you're in `artifact_handler.go`. The full three-way API unification can wait for a 4th sub-mode type to force the issue. |
| `bytearena/ecs` insulation/fork to fix entity-slice growth (Phase 4 #1) | No observed memory/perf problem yet — the Phase 2 leak fixes (raid dispose, in-game load reset, threat-map reset) remove the biggest practical contributors already. Revisit only if a long play session actually shows slowdown or memory growth. |
| GUI logic extraction out of `OnCreate`/`OnClick` closures (Phase 4 #3) | Literally an "as you touch it" habit per the source doc itself — no dedicated project needed. |

---

## Skip — consciously

| Item | Why |
|---|---|
| Gate `DEBUG_MODE`/`ENABLE_BENCHMARKING` behind build tags (Phase 1 #2) | There's no release build to protect yet, and a localhost pprof server is not a threat to the one machine you develop on. Do this when you actually cut a build to share with someone. |
| Delete empty dirs, regenerate `complexity_report.txt`, fix a doc-header date (Phase 1 #5) | A stale timestamp in a doc header costs nothing. Solving a problem you don't have. |
| Early-return in `vfx.clearVisualEffects` for the always-empty case (Phase 1 #8) | Two small slice allocations at 60fps in a turn-based tactical game is not GC pressure you will ever feel. No profiling evidence this matters; premature optimization dressed as a bug fix. |
| Deleting orphaned `consumabledata.json`/`creaturemodifiers.json`, gitignoring extensionless binaries (Phase 1 #10) | Moved to "eventually" bucket for the gitignore part; the JSON deletion specifically is inert dead weight that isn't confusing anyone but a hypothetical future reader — you know they're dead. |
| Decide/document balance-config placement across 4 files (Phase 3 #4) | You already know where your own 4 balance files are — you wrote them. Consolidating "one tuning story" serves a team-handoff scenario that doesn't exist. |
| Extract spawning magic numbers into JSON config (Phase 3 #5) | "Tunable without recompiling" solves a problem a solo dev with a 5-second Go build doesn't have. You already have the fastest possible iteration loop: edit the constant, rebuild. |
| Doc comments for the undocumented half of `combatcore` (Phase 3 #6) | Doc comments protect future collaborators and onboarding. There is no team to onboard. You wrote the code; you know what it does. |
| Keybindings JSON loader on top of `ActionMap` (Phase 3 #11) | No users means no one but you needs to rebind keys, and you can edit `defaultbindings.go` and recompile faster than you could build and test a JSON loader. |
| Minimal CI (GitHub Actions) — the CI half of Phase 3 #12 | No team, no PRs to gate, nothing pushing to this repo but you. CI enforces process discipline for a process (code review, merge gates) that doesn't exist here. `go vet`/`staticcheck` run locally (item 1 above) get you the actual value for free. |
| Consolidate 6 logging styles onto one seam (Phase 3 #15) | You're the only reader of your own terminal output. Unifying logging serves multi-service observability and ops handoff — neither applies. Gating the noisy debug prints (already in the do-list) gets you the only benefit that matters: less scroll. |
| Reduce direct `CoordManager`/position-system global access via params (Phase 4 #2) | Singletons are fine in a single-threaded ebiten game with one developer. This trades simplicity for parallel-test/headless-simulation capability you don't need yet. |

---

## Coverage tables — every roadmap item, one verdict each

### Phase 1 — Quick wins (15 items)

| # | Item | Verdict | Why |
|---|---|---|---|
| 1 | `.gitattributes` eol fix + renormalize | **Do** | Already effectively in place (`* text=auto` since initial commit, tree currently clean) — downgraded from "fix" to a 15-min verification, but still worth confirming and closing the loop. |
| 2 | Gate `DEBUG_MODE`/`ENABLE_BENCHMARKING` behind build tags | Skip | No release build exists yet to protect; localhost pprof isn't a risk on a solo dev's own machine. |
| 3 | `staticcheck`/`go vet` in standard workflow | **Do** | An hour of setup, mechanically catches real bugs (it would have caught the `fmt.Errorf` non-constant-format bug in this same repo) — cheap and directly useful, not ceremony. |
| 4 | Fix `time.Sleep` flakiness in `overworld_recorder_test.go` | Eventually | One test, low observed pain (zero other flaky tests in the suite) — fold into the next visit to that file. |
| 5 | Delete empty dirs, regenerate stale report, fix doc-header date | Skip | Pure hygiene with zero functional cost from leaving it alone. |
| 6 | `game_main/setup.go:202` `log.Fatalf` → error | **Do** | Verified: this is the one unrecoverable step in an otherwise-recoverable load path. 30-minute fix for a real crash. |
| 7 | `templates/entity_factory.go:37` `log.Fatal` → error | **Do** | Reachable at entity-creation time (not just boot); crashes mid-playtest on a missing creature image. Cheap, real. |
| 8 | Early-return in `vfx.clearVisualEffects` | Skip | No evidence of GC pressure or frame drops; two tiny slice allocs/frame at 60fps in a turn-based game is not a felt problem. |
| 9 | Seed `xpRNG`; seed the combat-analysis simulator | **Do** | Directly protects reproducibility of the one tool whose entire purpose is repeatable comparisons — high value, cheap. |
| 10 | Delete orphaned gamedata JSON; gitignore extensionless binaries | Eventually | Harmless to leave; batch with the next repo-hygiene pass rather than a dedicated trip. |
| 11 | Guard the in-game Load button until reset exists | **Do** | Verified: `main.go:92-98` calls `SetupRoguelikeFromSave` on the running game with no reset. A real, one-click-away corruption bug — stopgap now, real fix in Phase 2 #9. |
| 12 | Add `PostCombatReturnMode` to `TacticalState.Reset()` | **Do** | 10-minute fix for a real stale-state bug that would manifest as "why did combat return me to the wrong screen" during testing. |
| 13 | Gate AI/encounter debug prints behind `DEBUG_MODE`/injectable seam | **Do** | Printing on every AI attack floods your own console during the exact sessions (combat playtesting) where you need to read your own debug output. Cheap. |
| 14 | Fix `fmt.Errorf(result.Error)`; delete 4 dead exported methods | **Do** | The `go vet`-flagged format-string bug is a real (if rare) correctness risk; deleting confirmed-dead code costs nothing and declutters search. |
| 15 | Pass `RootContainer` to the two nil-parent `SubMenuController`s | **Do** | This is the exact ebitenui hidden-`ScrollContainer` input-layer gotcha already in your own memory notes — you've been bitten by this class of bug before. |

### Phase 2 — Short term (11 items)

| # | Item | Verdict | Why |
|---|---|---|---|
| 1 | Round-trip serialization tests for `setup/savesystem/chunks` | **Do** | The Critical item: no test, no version check, no decode-error path protects your actual save data. This is "prevent losing work," the sharpest test in the frame. |
| 2 | Generic typed panel-registration helper | Eventually | Real payoff (kills ~2,381 lines of boilerplate) but a 3-4 day investment — do it the next time you're adding a batch of new panels and feeling the pain directly, not speculatively. |
| 3 | Core-path tests for `campaign/raid` | Eventually | Raid is a real gameplay pillar but nothing is currently reported broken; write tests alongside the next raid-logic change rather than a dedicated 2-3 day sprint. |
| 4 | Split `squadqueries.go` into 3 files | Eventually | Pure navigability win, zero risk, but not blocking anything — fold into the next unrelated edit of that file. |
| 5 | Wire `ChunkVersion()` into the save envelope + decide `DisallowUnknownFields` | **Do** | Not hypothetical: `progression_chunk.go` already returns version 2 while every other chunk returns 1 — a real schema change already shipped undetected. Pairs with item 1. |
| 6 | Validators for `monsterdata.json`/`unitspells.json`; promote dangling refs to fatal | **Do** | Converts a silent runtime failure (missing spell reference discovered at cast time) into a boot-time error — exactly the "misleading state during testing" the frame flags. |
| 7 | Cache per-frame combat render info / flood-fill re-run | Eventually | No profiling evidence of actual stutter; do this only if move-mode input lag is ever actually felt, not preemptively. |
| 8 | Dispose raid garrison entities in `finishRaid` | **Do** | Verified: `finishRaid` disposes nothing. Every completed or abandoned raid leaks squads/units/room entities permanently into the session. |
| 9 | World-reset for in-game Load | **Do** | The real fix behind Phase 1 #11's stopgap — turns a guarded-off feature back into a working one. |
| 10 | Reset threat/AI/CVM faction maps at combat teardown | **Do** | Cheap (half day) and directly protects long solo balance-testing sessions from accumulating stale AI/threat state across many test combats. |
| 11 | Fix overworld event-data key drift | **Do** | Scoped to the quick key-rename (not the full typed-event-struct rewrite) — cheap, revives a summary feature that's currently silently dead. |

### Phase 3 — Medium term (16 items, one split into two)

| # | Item | Verdict | Why |
|---|---|---|---|
| 1 | Tests for `combatmath`/`combatstate` | Eventually | Cheap per-change (table-driven), but no current bug — write alongside the next formula change rather than a dedicated pass. |
| 2 | Consolidate `gui/builders` option plumbing | Eventually | Real friction reduction but a 3-4 day refactor with no current blocker — chip away at it opportunistically. |
| 3 | Extract shared overworld `Execute*` scaffolding | Eventually | DRY win, but 6 copies isn't yet painful enough to justify a dedicated 1-2 days; do it when a 7th action makes the pattern obvious. |
| 4 | Decide balance-config placement (4 files vs. centralized) | Skip | Solves a "which of my 4 files is it in" confusion that a solo dev who wrote all 4 files doesn't have. |
| 5 | Extract spawning magic numbers into config | Skip | Values that require a rebuild to change are fine when the rebuild takes 5 seconds and you're the only one rebuilding. |
| 6 | Doc comments for undocumented half of `combatcore` | Skip | Protects onboarding for collaborators that don't exist. |
| 7 | Ship AI-commander spellcasting (the TODO) | Eventually | Legitimate feature gap, not tech debt — schedule as a feature next time you're in `mind/ai`. |
| 8 | Unify the three RNGs fully | Eventually | The part that matters (balance reproducibility) is already covered by Phase 1 #9; the remaining UI-local RNG doesn't affect determinism. |
| 9 | Unify four gamedata loaders onto fail-fast semantics | **Do** | Scoped down: making balance-config validators fail loud instead of silently zeroing (Phase 2-adjacent, see do-list #10) is the part with real payoff; the full four-loader unification isn't needed to get it. |
| 10 | Convert `GameMapUtil.go` asset panics to errors | **Do** | Cheap (half day), removes a hard crash reachable during New Game asset iteration. |
| 11 | Keybindings JSON loader | Skip | No users need to rebind keys; editing `defaultbindings.go` and recompiling is faster than building a config loader. |
| 12a | Minimal CI (GitHub Actions) | Skip | No team, no PRs — nothing to gate. Local `go vet`/`staticcheck` (already in do-list) gets the real value for free. |
| 12b | Replace 4.25MB font `.go` with `//go:embed` | **Do** | Independent of CI; cheap, permanently shrinks every clone/diff/log involving that file. |
| 13 | Mode-name typed constants | Eventually | Real latent risk (silent double-declaration, order-coupling) but zero observed incidents yet — do the full refactor only after it actually costs a debugging session. |
| 14 | Unify sub-mode enter/exit API + artifact `BlockReason` | Eventually | The missing artifact "why not" feedback is worth doing on its own the next time you're in that file; the 3-way API unification can wait for a forcing 4th case. |
| 15 | Consolidate 6 logging styles into 1 seam | Skip | Solo dev reading their own terminal doesn't need a unified logging seam; gating the noisy prints (already in do-list) is the only piece with real payoff. |
| 16 | Fixtures delegate to production bootstrap (`registerCoreComponents`/`buildCoreTags`) | **Do** | Protects the value of the 57.5% test coverage `tactical/` already has — without this, passing tests can silently diverge from what production actually does. |

### Phase 4 — Long term (3 items)

| # | Item | Verdict | Why |
|---|---|---|---|
| 1 | `bytearena/ecs` insulation/fork to fix never-pruned entity slice | Eventually | Real structural risk, but no observed memory/perf problem today, and the Phase 2 leak fixes (raid dispose, load reset, threat-map reset) remove the biggest practical contributors already. Revisit only if a long session actually shows growth or slowdown. |
| 2 | Reduce direct global access via params/context | Skip | Singletons are fine in a single-threaded ebiten game with one developer; this buys parallel-test/headless-sim capability nobody's asking for. |
| 3 | GUI logic extraction as panels are touched | Eventually | Exactly as the source doc frames it — an ongoing habit while touching panels, not a project. |

---

## What I'd do this week

A concrete sequence, roughly 4-5 hours total, all pulled from the top of the "Do these" list — the highest payoff-per-hour items, in dependency order:

1. **(30 min)** `game_main/setup.go:202` — replace `log.Fatalf` with a wrapped error so the load-game fallback works.
2. **(1h)** `gui/guiexploration/exploration_panels_registry.go:194-199` — disable/guard the in-game Load button until the world-reset fix lands.
3. **(30 min)** `templates/entity_factory.go:37` — replace `log.Fatal` with a returned error.
4. **(1-2h)** `mind/combatlifecycle/reward.go:21` — seed `xpRNG` from the common seeded RNG; wire `SetRNGSeed`/`SetXPRNG` into the `tools/combat_analysis` simulator entry point.
5. **(1h combined)** Small correctness sweep: add `PostCombatReturnMode` to `TacticalState.Reset()` (`gui/framework/contextstate.go:75-98`), fix `fmt.Errorf(result.Error)` → `errors.New` (`combatbase/handler.go:133`), and pass `RootContainer` instead of nil to the two `SubMenuController`s (`explorationmode.go:59`, `overworldmode.go:94`).

That closes both Critical items from the source report (or guards one until the full fix, item 4 in "Do these," gets its own dedicated day) and removes five real, verified crash/correctness bugs — all before touching anything that requires a multi-day investment.
