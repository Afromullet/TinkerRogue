# Equipment System: Major & Minor Artifacts

Equipment splits into two tiers: **Minor Artifacts** (passive stat buffs, always-on while equipped) and **Major Artifacts** (active turn/action economy manipulation, charge-based). Together they fill the gap between perks and spells without overlapping either.

---

## The Four-System Contract

Every system in the game occupies a distinct mechanical niche. This contract governs every item in this document.

| System | Timing | Player Agency | Effect Type | Resource |
|--------|--------|---------------|-------------|----------|
| **Perks** | Always-on, passive | None during combat | Behavioral hooks, targeting overrides, cover mods, damage hooks | None (equip once) |
| **Spells** | Active, during your turn | Proactive (choose when to cast) | Damage, temp buffs/debuffs (2-3 turns) | Mana (persists across battles) |
| **Minor Artifacts** | Passive while equipped | None during combat | Permanent stat buffs via `ActiveEffect` | None (equip once) |
| **Major Artifacts** | Active, during combat | Tempo control, sequencing puzzle | Turn order, action count, action availability manipulation | Charge-based (once per battle / once per round) |

### Hard Rules: Minor Artifacts

- Minor artifacts may ONLY apply `ActiveEffect` entries with `RemainingTurns = -1` (permanent for the battle)
- Minor artifacts may ONLY modify stats in the `Attributes` struct (Strength, Dexterity, Magic, Leadership, Armor, Weapon, MovementSpeed, AttackRange)
- No minor artifact may deal damage to units
- No minor artifact may heal units
- No minor artifact may consume or interact with `ManaData`
- No minor artifact may override `SelectTargetUnits` or `CanUnitAttack` logic (that is the perk pipeline)
- No minor artifact may modify `CoverBreakdown` values or `CoverData` (that is the perk pipeline)
- No minor artifact may modify `TurnStateData`, `ActionStateData`, or `TurnOrder` (that is the major artifact pipeline)

### Hard Rules: Major Artifacts

- No major artifact may modify Strength, Dexterity, Magic, Armor, Weapon, or any stat in the `Attributes` struct
- No major artifact may deal damage to units
- No major artifact may heal units
- No major artifact may apply `ActiveEffect` entries (that is the minor artifact / spell pipeline)
- No major artifact may consume or interact with `ManaData`
- No major artifact may modify `CoverBreakdown` values or `CoverData` (that is the perk pipeline)
- No major artifact may override `SelectTargetUnits` or `CanUnitAttack` logic (that is the perk pipeline)

---

## Minor Artifacts

### Core Concept

Minor artifacts are passive stat sticks. Equip one on a squad and it applies a permanent `ActiveEffect` for the duration of the battle -- no activation, no cost, no decision during combat. They are weaker than spell buffs but free and always-on.

**Player fantasy:** "My equipment defines my squad's identity before combat even starts. My vanguard is tougher, my archers shoot farther, my scouts move faster -- permanently, reliably, without spending mana."

**Feels like:** Fire Emblem stat-boosting items, FFT accessory bonuses.

### Mechanical Detail

At battle start, during `InitializeCombat`, iterate all squads with equipped minor artifacts. For each, call `effects.ApplyEffect()` on every unit in the squad with:

```go
ActiveEffect{
    Name:           "Iron Bulwark",        // artifact name
    Source:         SourceItem,            // new effect source (see below)
    Stat:           StatArmor,
    Modifier:       +2,
    RemainingTurns: -1,                    // permanent -- never ticks down
}
```

At battle end, `cleanupEffects()` in `combatservices/combat_service.go:295-309` already removes ALL effects from all squad members. No additional cleanup needed.

**New infrastructure:** Add `SourceItem` to the `EffectSource` enum in `tactical/effects/components.go:26-29`:

```go
const (
    SourceSpell   EffectSource = iota
    SourceAbility
    SourceItem    // NEW -- minor artifact stat buffs
)
```

### Design Constraint: Weaker Than Spells

Spell buffs are temporary but strong (War Cry: +4 STR for 3 turns, Arcane Shield: +3 Armor for 3 turns). Minor artifacts are permanent but weaker. This creates a clear decision: do you spend mana for a burst window, or rely on your artifact's steady baseline? They stack -- a squad with a +1 STR artifact AND War Cry gets +5 STR for the buff window, then drops back to +1 after.

### Example Minor Artifacts

#### 1. Iron Bulwark
**Effect:** +2 Armor (permanent)
**Tactical note:** Turns a squishy ranged squad into one that survives an extra hit. Weaker than Arcane Shield's +3, but doesn't cost 12 mana and lasts the whole battle.

#### 2. Keen Edge Whetstone
**Effect:** +2 Weapon (permanent)
**Tactical note:** Flat damage increase for every attack, every turn. Pairs well with Chain of Command Scepter -- extra attacks multiply the value of flat weapon bonuses.

#### 3. Fleet Runner's Sandals
**Effect:** +1 MovementSpeed (permanent)
**Tactical note:** One extra tile of movement every turn adds up. Flanking squads reach their targets a round earlier. Melee squads close the gap before ranged squads whittle them down.

#### 4. Marksman's Scope
**Effect:** +1 AttackRange (permanent)
**Tactical note:** Ranged squads can engage from one tile farther back. The extra range tile is a significant positioning advantage -- it can mean shooting without being in melee threat range.

#### 5. Berserker's Torc
**Effect:** +2 Strength, -1 Armor (permanent)
**Tactical note:** A tradeoff item. Your squad hits harder but takes more damage. Ideal on glass cannon squads that plan to kill before being hit. Terrible on your frontline that absorbs attacks.

#### 6. Sentinel's Plate
**Effect:** +2 Armor, -1 MovementSpeed (permanent)
**Tactical note:** Inverse of Berserker's Torc. Your squad becomes an immovable wall but loses repositioning flexibility. Perfect for chokepoint defenders, terrible for flankers.

#### 7. Duelist's Gloves
**Effect:** +1 Dexterity, +1 Strength (permanent)
**Tactical note:** Small dual buff, no downside. A safe generalist pick. Neither stat is as high as a dedicated artifact, but the breadth covers more situations.

---

## Major Artifacts

### Core Concept

Major artifacts change **when and how often things happen** in the turn sequence. Not what damage is dealt (perks), not what active ability is cast (spells), but the structure of who acts when, how many actions they get, and what the turn order looks like.

This operates on `TurnManager` and `ActionStateData` -- the structural layer that perks and spells are built on top of. Perks modify the damage formula. Spells are actions within a turn. Major artifacts control the turns themselves.

**Player fantasy:** "I don't hit harder or cast better spells. I control the tempo of the entire battle. My squads act when I want, skip when I say, and chain momentum that the enemy can never match."

**Feels like:** Advance Wars bonus turns, Into the Breach timeline manipulation, Final Fantasy Tactics speed/CT system.

### Items

#### 1. Double Time Drums

**Effect:** Once per battle, one friendly squad may take its Move action AND its Attack action on the same activation, ignoring the normal action economy.

**Mechanical detail:** When activated on a squad that has not yet acted this turn, set a flag that prevents `markSquadAsActed` from firing after the first action. The squad completes both Move and Attack before `HasActed` and `MovementRemaining` are consumed. Both actions still resolve normally through the existing damage/movement pipeline -- no stat changes, no damage bonuses.

**Tactical decision:** Do you use this on turn 1 for a devastating alpha strike, or save it for a critical repositioning moment when your ranged squad needs to flee AND shoot?

#### 2. Stand Down Orders

**Effect:** Once per battle, force one enemy squad to skip its Attack action this turn. The squad can still move.

**Mechanical detail:** When activated, target one enemy squad. Set its `HasActed = true` before its turn resolves. The squad's `MovementRemaining` is unaffected -- it can reposition but cannot attack or cast spells. Does not apply any debuff, modify any stat, or deal damage. Pure action denial.

**Tactical decision:** Do you silence their strongest squad early to prevent an alpha strike, or hold it for the critical moment when they're about to deliver a killing blow to your weakened vanguard?

#### 3. Chain of Command Scepter

**Effect:** Once per round, one friendly squad may pass its unused Attack action to an adjacent friendly squad. The receiving squad gets a second Attack this round. The passing squad loses its Attack.

**Mechanical detail:** Source squad must have `HasActed = false`. Target squad must be adjacent (Chebyshev distance 1) and must have already attacked this turn (`HasActed = true`). On activation: set source `HasActed = true`, set target `HasActed = false`. The target then takes a normal attack through the standard combat pipeline. No stat modification occurs -- the attack resolves with the target squad's existing attributes.

**Tactical decision:** Your support squad has weak attacks. Your DPS squad already attacked. Do you sacrifice the support squad's action to give your heavy hitters a second swing? The "battery" dynamic creates a new army archetype: squads built to feed actions rather than fight.

#### 4. Commander's Initiative Badge

**Effect:** Your faction always acts first in every round. Overrides the random faction ordering in `TurnStateData.TurnOrder`.

**Mechanical detail:** During `InitializeCombat`, after Fisher-Yates shuffle of `TurnOrder`, move the player's faction ID to index 0. This is a permanent structural change for the entire battle. No stats are modified, no damage is changed -- but first-mover advantage compounds over many rounds.

**Tactical decision:** Consistent tempo control versus an equipment slot spent on something more situational. Going first every round means you always set the board state the enemy reacts to. But if you're defensive, first-mover matters less.

#### 5. Forced Engagement Chains

**Effect:** When one of your squads destroys an enemy squad, the destroying squad immediately gets a free Move action (1 tile, ignoring `MovementRemaining`).

**Mechanical detail:** After `RemoveSquadFromMap` fires for a destroyed enemy squad, check if the destroying squad has this equipment. If yes, grant a bonus move: set `MovementRemaining = 1` temporarily, allow one move, then restore `MovementRemaining` to 0. The bonus move does not refresh the attack. No stats change, no damage bonus for the kill -- only the reward of repositioning.

**Tactical decision:** Build your army around kill-chaining. Aggressive compositions that can destroy squads quickly get free repositioning for the next engagement. Defensive turtling ignores this bonus entirely. Do you optimize for snowball or stability?

#### 6. Hourglass of Delay

**Effect:** Once per battle, designate one enemy squad to act last within its faction's turn.

**Mechanical detail:** When activated, move the target enemy squad's action to the end of the enemy faction's resolution order. The enemy faction still takes its full turn with all squads, but the internal ordering changes. The delayed squad faces a battlefield that has already shifted from its allies' actions and your responses.

**Tactical decision:** Delay their strongest squad to let your counterattacks soften it first, or delay their fastest squad to prevent it from flanking before you can reposition? Timing and target selection create a deep decision.

#### 7. Momentum Standard

**Effect:** If your faction destroys an enemy squad during your turn, your next squad to activate this turn gets +1 `MovementRemaining`.

**Mechanical detail:** Track a "momentum" flag on the faction's turn state. When an enemy squad is destroyed during your faction's turn, set the flag. The next friendly squad to have `ResetSquadActions` called (or the next squad to begin its activation) receives `MovementRemaining += 1`. The flag resets after one use per turn. No stat modification -- only movement range for one squad.

**Tactical decision:** Sequence your activations to kill first with a squad that doesn't need to move, then activate a squad that benefits from the bonus movement. Turn order within your own faction becomes a puzzle.

#### 8. Rallying War Horn

**Effect:** Once per battle, when one of your squads is attacked, immediately grant one other friendly squad (that has not yet acted this round) a bonus activation out of normal turn order.

**Mechanical detail:** Hooks into the post-attack-resolution callback in `CombatActionSystem`. When the defender belongs to the player's faction, check if the artifact is equipped and uncharged. If triggered, the player selects one friendly squad with `ActionStateData.HasActed == false && ActionStateData.HasMoved == false`. That squad is flagged for immediate activation: the combat loop yields control to the player to resolve that squad's Move and Attack before the enemy faction's remaining squads continue. After the bonus activation, the squad's `HasActed` and `HasMoved` are set to `true` as normal. The interrupted enemy turn then resumes. No stats, damage, or effects are modified -- only the sequencing of when a friendly squad gets to act.

**Tactical decision:** You must decide WHICH incoming attack triggers the horn. Blow it on the first attack and you react early, but the enemy might not have committed their dangerous squads yet. Hold it and you risk losing the squad you wanted to protect. And you must choose which ally gets the bonus activation -- your support squad repositioning to safety, or your flanker striking the exposed attacker?

#### 9. Lockstep Banner

**Effect:** Once per round, when you activate a squad, you may immediately activate one adjacent friendly squad as well. Both squads share a single activation window and both are marked as acted afterward.

**Mechanical detail:** When the player selects a squad to activate, if the Lockstep Banner is equipped and charged, the player may designate one adjacent friendly squad (Chebyshev distance 1) with `HasActed == false`. Both squads' `ActionStateData` are consumed together: both get their Move and Attack actions, but both are marked `HasActed = true` and `HasMoved = true` at the end of the shared window regardless of what each individually did. The second squad's `MovementRemaining` is set from its own speed as normal. This does not grant extra actions -- it grants simultaneous timing.

**Tactical decision:** Paired activations let you coordinate a pincer attack before the opponent can respond to either squad. But you burn two squads' actions in one window. If either squad would have benefited from waiting to see how the battlefield develops after other activations, you lose that flexibility. The adjacency requirement also forces you to keep squads close, which may conflict with optimal positioning.

#### 10. Saboteur's Hourglass

**Effect:** Once per battle, at the start of the enemy faction's turn, reduce `MovementRemaining` for ALL enemy squads by 2 (minimum 0) for that turn.

**Mechanical detail:** Hooks into `TurnManager.ResetSquadActions()`. After the normal reset loop sets each enemy squad's `MovementRemaining` from squad speed, apply a modifier: `actionState.MovementRemaining = max(0, actionState.MovementRemaining - 2)`. The `HasMoved` and `HasActed` flags are unaffected -- enemy squads can still move and attack, they just have fewer tiles to work with. This is a one-time modification during the reset phase, not a persistent debuff or `ActiveEffect`.

**Tactical decision:** Timing is everything. Use it early when enemy melee squads are still closing distance and the reduced movement might prevent them from reaching your line entirely. Or save it for the round when the enemy is trying to reposition after you break their formation -- two fewer tiles might strand a squad in a terrible position. Against ranged-heavy enemies who don't need to move, the value drops significantly.

#### 11. Anthem of Perseverance

**Effect:** Once per battle, at the end of your faction's turn, choose one friendly squad that attacked this turn. That squad immediately gets a bonus attack action (no movement) before the turn passes to the next faction.

**Mechanical detail:** Fires during the transition in `TurnManager.EndTurn()`, after the current faction finishes but before the next faction begins. The selected squad must have `HasActed == true`. Set `HasActed = false` and open a bonus attack window for that squad only (no movement -- `HasMoved` stays `true`, `MovementRemaining` stays 0). After the bonus attack resolves, `HasActed` is set back to `true` and the turn ends normally. No stat modification occurs -- the attack resolves with the squad's existing attributes.

**Tactical decision:** You get a guaranteed second attack with one squad, but only at the END of your turn -- the battlefield has already shifted from all your other activations. The squad cannot move before this bonus attack, so it must attack from wherever it ended up. This rewards planning your activation order so the right squad ends up adjacent to a valuable target when all other activations are done.

#### 12. Deadlock Shackles

**Effect:** Once per battle, designate one enemy squad. That squad's entire next activation is skipped -- no move, no attack.

**Mechanical detail:** When activated, target one enemy squad. Set a `SkipNextActivation` flag (tracked in the artifact charge system or on `ActionStateData`). When that squad's faction turn begins and `ResetSquadActions` fires, check the flag. If set: leave `HasActed = true` and set `MovementRemaining = 0`, then clear the flag. The squad's entire activation is consumed. No `ActiveEffect` is applied, no stat is modified -- the action state fields are simply pre-set to "already done."

**Tactical decision:** Stronger than Stand Down Orders (which only denies attack, not movement) but the same once-per-battle charge. Stand Down Orders lets the enemy reposition, which might be fine if they're melee and you want them to walk into a bad position. Deadlock Shackles freezes them completely -- but if the target was going to skip its turn anyway (already in position, no good targets), you wasted the charge. Target selection is the entire decision.

#### 13. Vanguard's Oath

**Effect:** Passive (no charge). The first friendly squad to activate each round gets +2 `MovementRemaining`.

**Mechanical detail:** In `TurnManager.ResetSquadActions()`, after the normal reset loop, track a `FirstActivationThisRound` flag on the turn state. When the first friendly squad begins its activation, add +2 to its `ActionStateData.MovementRemaining`. After that squad completes its activation, clear the flag so no subsequent squads receive the bonus. This modifies `MovementRemaining` directly -- not stats, speed, or any `Attributes` field.

**Tactical decision:** You must decide which squad activates first each round, because that squad gets the movement bonus. A scout activated first covers enormous ground for flanking. A melee vanguard activated first closes distance before enemies can react. But activating your strongest squad first might not always be correct -- sometimes you want information from a safer activation before committing. The bonus creates a tension between "who benefits most from extra movement" and "who should act first for tactical information."

#### 14. Echo Drums

**Effect:** Once per round, after one of your squads finishes its full activation (both move and attack resolved), that squad's `MovementRemaining` resets to its squad speed for a second movement phase. It cannot attack again.

**Mechanical detail:** After a friendly squad completes its activation (both `HasMoved = true` and `HasActed = true`), if Echo Drums is equipped and charged this round, the player may trigger the artifact. Reset `HasMoved = false` and set `MovementRemaining` to the squad's speed value (via `GetSquadMovementSpeed`). `HasActed` stays `true` -- no second attack. The squad gets a full second movement phase. After the bonus movement completes, `HasMoved = true` is set again and the charge is consumed for this round (refreshes next round).

**Tactical decision:** Attack-then-retreat becomes a core tactic. A melee squad charges in, attacks, then falls back behind your line. A ranged squad repositions to a firing position, shoots, then retreats to cover. But the once-per-round constraint means only one squad gets this each round. Which squad needs the escape route most? And using Echo Drums on a squad that doesn't need to retreat wastes the round's charge.

---

## Zero-Overlap Proof

### Minor Artifacts

**vs Perks:** Perks operate on behavioral hooks -- they modify the damage formula (`DamageModifiers`), override targeting (`SelectTargetUnits`, `CanUnitAttack`), and adjust cover (`CoverBreakdown`). Minor artifacts never touch any of these. They only apply `ActiveEffect` entries that modify raw stats. A perk that says "ignore 2 armor on counterattacks" and an artifact that says "+2 Armor" operate on completely different systems.

**vs Spells:** Spells are active, cost mana, are temporary (2-3 turns), and can be strong (War Cry: +4 STR, Arcane Shield: +3 Armor). Minor artifacts are passive, free, permanent, and weaker (+1 to +2 per stat). No minor artifact reaches the stat values of a spell buff. The two layer naturally: artifact provides baseline, spell provides burst.

**vs Major Artifacts:** Minor artifacts modify stats via `ActiveEffect`. Major artifacts modify `TurnStateData`, `ActionStateData`, and `TurnOrder`. They touch entirely different ECS components. A minor artifact never manipulates action economy; a major artifact never touches a stat.

### Major Artifacts

**vs Perks:** Perks modify the damage formula (`DamageModifiers`), stats (`Attributes`), cover (`CoverBreakdown`), and targeting (`SelectTargetUnits`). No major artifact touches any of these. Every major artifact operates on `TurnStateData`, `ActionStateData`, or `TurnOrder` -- structures perks never interact with.

**vs Spells:** Spells are actions within a turn that deal damage, apply buffs, or apply debuffs to targeted squads using mana. No major artifact deals damage, applies any `ActiveEffect`, targets a squad with a stat change, or uses mana. These items change the structural rules of when actions happen, not what those actions do.

**vs Minor Artifacts:** Major artifacts never apply `ActiveEffect` or modify stats. Minor artifacts never modify `TurnStateData`, `ActionStateData`, or turn order. The two tiers are mechanically disjoint.

---

## Existing System Interactions

| System | Interaction |
|--------|------------|
| `effects.ApplyEffect` | Minor artifacts apply permanent effects at battle start |
| `combatservices.cleanupEffects` | Removes all effects (including minor artifact effects) at battle end |
| `TurnManager.InitializeCombat` | Minor artifacts hook here to apply effects; Initiative Badge modifies `TurnOrder` after shuffle |
| `TurnManager.ResetSquadActions` | Momentum Standard (+1 movement on kill), Saboteur's Hourglass (-2 enemy movement), Vanguard's Oath (+2 first squad movement), Deadlock Shackles (pre-set skip flag) |
| `TurnManager.EndTurn` | Hourglass of Delay reorders faction-internal resolution; Anthem of Perseverance grants bonus end-of-turn attack |
| `ActionStateData.HasActed` | Stand Down Orders sets preemptively; Chain of Command toggles between squads; Deadlock Shackles pre-sets to skip; Anthem of Perseverance resets for bonus attack |
| `ActionStateData.HasMoved` | Double Time Drums delays flag consumption; Echo Drums resets for bonus movement phase |
| `ActionStateData.MovementRemaining` | Saboteur's Hourglass reduces enemy movement; Vanguard's Oath adds to first squad; Echo Drums resets to full speed |
| `CombatActionSystem` (post-attack) | Forced Engagement Chains hooks into post-destruction; Rallying War Horn triggers on defender being attacked |
| `RemoveSquadFromMap` | Forced Engagement Chains and Momentum Standard trigger on squad destruction |
| Squad activation selection | Lockstep Banner allows paired simultaneous activation of adjacent squads |

---

## Implementation Complexity

### Minor Artifacts: Low

**Systems touched:** `effects/components.go`, `combat/turnmanager.go` (InitializeCombat hook)

**New infrastructure needed:**
- `SourceItem` effect source in `tactical/effects/components.go`
- Equipment slot on squad entities (new component, e.g., `EquippedArtifactData`)
- Apply-at-battle-start loop in `InitializeCombat` (iterate squads → check equipped artifact → apply `ActiveEffect` to each unit)
- No new cleanup needed -- `cleanupEffects()` already removes all effects at battle end

**Risk:** Low. Uses the existing `ActiveEffect` pipeline end-to-end. The only new code is the apply-at-start hook and the `SourceItem` constant.

### Major Artifacts: Medium-High

**Systems touched:** `TurnManager`, `ActionStateData`, `CombatActionSystem`

**New infrastructure needed:**
- Equipment activation UI (select item, select target squad)
- "Once per battle" / "once per round" charge tracking per item
- Post-destruction hook in `ExecuteAttackAction` (after `RemoveSquadFromMap`)
- Faction-internal turn ordering (currently all squads in a faction act in undefined order)

**Risk:** Modifying `TurnManager` is the highest-risk change because it controls the core combat loop. Each item needs careful testing to ensure the turn sequence remains consistent.

---

## Relationship with Other Systems

### Perks + Equipment

Perks modify the damage formula -- equipment controls stats (minor) and tempo (major). A perk that adds +2 damage on every attack becomes more valuable when a minor artifact boosts Weapon by +2 (stacking raw damage) or when a major artifact grants extra attacks (Chain of Command Scepter). The three systems multiply each other's value without overlapping.

### Spells + Equipment

Spells are active abilities cast during a turn. Minor artifacts provide a permanent stat baseline that spell buffs spike above -- a squad with +1 STR from an artifact and War Cry (+4 STR) gets +5 during the buff window, dropping to +1 after. Major artifacts control the turn structure itself: Double Time Drums lets a squad move AND attack in one activation, and the squad can still cast spells during that activation.

### Minor Artifacts + Major Artifacts

Minor artifacts define what a squad is good at (stat identity). Major artifacts define when and how often the squad gets to act. A squad with Keen Edge Whetstone (+2 Weapon) paired with Chain of Command Scepter (extra attack via action pass) gets to use that +2 Weapon twice in one round. A squad with Fleet Runner's Sandals (+1 Movement) paired with Momentum Standard (+1 Movement on kill) can cover enormous ground in aggressive plays. The two tiers are mechanically independent but strategically synergistic.

### Equipment + Doctrines

If Doctrines are implemented as a separate system, they set battle-wide rules while equipment provides tactical activations within those rules. Doctrine of Conservation (attack every other round) makes Chain of Command Scepter more valuable -- pass your off-round squad's useless action to an on-round squad. Doctrine of Fortification (limited movement) makes Forced Engagement Chains less useful but Momentum Standard more valuable for the rare extra movement.
