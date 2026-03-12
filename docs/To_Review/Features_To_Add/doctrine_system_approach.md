# Doctrine System: Constraint-for-Power Tradeoffs

> A completely new design axis. Zero overlap. The only system that takes something away.

---

## Core Concept

Pre-battle commitments that impose **binding constraints** on your entire army in exchange for structural combat advantages. Before a battle begins, you choose a Doctrine. That Doctrine restricts what your squads can do for the entire fight. In exchange, it grants a powerful advantage that the enemy cannot access.

This is the only system in the game that **voluntarily limits the player**. Perks only add power. Spells only add options. Equipment (Turn/Action Economy) only grants tactical activations. Doctrines say: "You cannot do X for this battle. In exchange, Y happens."

**Player fantasy:** "I chose the Doctrine of Silence before this battle. My commander cannot cast any spells. In exchange, every enemy squad that attacks me loses their counterattack for the rest of the round. I built my army to exploit that -- pure melee rushdown that charges in without fear of retaliation."

**Feels like:** Darkest Dungeon quirk system, Into the Breach squad restrictions, Magic: The Gathering deckbuilding constraints (mono-color payoffs), XCOM 2 "Ironman mode" as a choice.

---

## Zero-Overlap Contract

Doctrines must not duplicate perks, spells, or equipment. This contract governs every Doctrine in this document.

| System | Timing | Player Agency | Effect Type | Resource |
|--------|--------|---------------|-------------|----------|
| **Perks** | Always-on, passive | None during combat | Stat mods, damage hooks, target overrides, cover mods | None (equip once) |
| **Spells** | Active, during your turn | Proactive (choose when to cast) | Damage, temp buffs (+STR/+Armor), temp debuffs (-STR/-DEX) | Mana (persists across battles) |
| **Equipment** | During combat (activation timing) | Tempo control, sequencing puzzle | Turn order, action availability, action count manipulation | Charge-based (once per battle / once per round) |
| **Doctrines** | Pre-battle (commitment) | Strategic army-building constraint | Battle-wide rule changes (constraints + powers) | None (choose once per battle) |

---

## Doctrines

### 1. Doctrine of Silence

**Constraint:** Your commander cannot cast any spell for the entire battle. `SpellBookData` is locked -- the cast button is greyed out.

**Power:** Every enemy squad that initiates an attack against one of your squads loses its counterattack capability for the remainder of the current round. Mechanically: after an enemy squad's attack resolves against you, set a flag on that enemy squad that causes `WouldSquadSurvive` checks for counterattack to return false for the rest of the round.

**Tactical decision:** You give up all 17 damage spells, War Cry, Arcane Shield, Weaken, and Frost Slow. That is an enormous sacrifice. In exchange, your melee squads can charge into combat knowing the enemy cannot counter. Build spell-independent armies that thrive on unopposed aggression.

### 2. Doctrine of Attrition

**Constraint:** No friendly squad may attack the same enemy squad in consecutive turns. After Squad A attacks Enemy B, Squad A cannot target Enemy B on its next activation. A different friendly squad may attack Enemy B, and Squad A may attack any other enemy.

**Power:** After a friendly squad completes its attack against a target it has not attacked before, that squad's `MovementRemaining` is fully restored (reset to its initial speed value). This enables hit-and-reposition tactics: strike a new target, then immediately reposition to set up the next round's attack on yet another target.

**Tactical decision:** You cannot focus-fire. Your army must spread damage across multiple enemy squads. In exchange, your squads become highly mobile -- every first strike against a new target lets you reposition. This rewards wide-front armies with many squads cycling between different targets while staying on the move.

### 3. Doctrine of Conservation

**Constraint:** Each friendly squad can only attack every other round. On even-numbered rounds (2, 4, 6...), your squads cannot use their Attack action. They may still move and cast spells (if not under Doctrine of Silence).

**Power:** Every attack your squads make (on odd rounds) applies its damage twice. The attack resolves once through `processAttackWithModifiers` (single hit/dodge/crit/resistance/cover calculation), then the final calculated damage value for each target is applied a second time. This is not a second independent attack roll -- it is a single attack whose resolved damage is recorded twice. One hit roll, one crit roll, one cover check, double the recorded damage.

**Tactical decision:** Half as many attack opportunities, but each one is devastating. Position carefully on your "off" rounds, then unleash double attacks on your "on" rounds. Pairs terribly with Forced Engagement Chains (fewer kills per round = fewer chain opportunities) but pairs well with Chain of Command Scepter (pass the off-round squad's useless attack to an on-round squad).

### 4. Doctrine of Fortification

**Constraint:** No friendly squad may move more than 1 tile per turn, regardless of their `MovementRemaining` value. After moving 1 tile, `MovementRemaining` is set to 0.

**Power:** Any enemy squad that moves to a tile adjacent to (Chebyshev distance 1) a friendly squad has its `MovementRemaining` immediately set to 0. The enemy squad is locked in place for the rest of its movement phase. This creates a defensive web: approaching your army means getting stuck next to it, unable to reposition or retreat. Multiple friendly squads create overlapping no-move zones.

**Tactical decision:** Your army barely moves. Deployment becomes critical because you are nearly locked to your starting positions. But enemies that approach you get trapped -- they commit to the engagement with no option to disengage. Build tanky armies that punish enemies for coming to you.

### 5. Doctrine of Sacrifice

**Constraint:** At the start of each battle, your lowest-level unit across all squads is immediately removed from the fight. That unit does not participate, freeing its grid cell. If multiple units share the lowest level, one is chosen randomly.

**Power:** All remaining units in the army have their attacks resolve before counterattacks are calculated. Mechanically: the counterattack phase in `ExecuteAttackAction` is moved to the end of the full attack resolution. All your attacking units deal damage, THEN the surviving defenders counterattack. Normally, counterattacks happen per-attacker, meaning later attackers face a defender that has already counterattacked earlier ones.

**Tactical decision:** You permanently lose a unit every fight. Over a campaign, this is a heavy attrition cost. But your attacks land without interruption from counterattacks, making focus-fire and alpha strikes dramatically stronger. Recruit deep, expendable armies.

### 6. Doctrine of Commitment

**Constraint:** Once a friendly squad attacks an enemy squad, that friendly squad cannot attack any other enemy squad for the rest of the battle. It is "committed" to that target until the target is destroyed.

**Power:** Committed squads gain +1 `MovementRemaining` per turn (tracking their committed target) and their attacks against their committed target bypass the `WouldSquadSurvive` prediction check -- counterattacks happen regardless of predicted lethality, but committed squads' counterattacks also always fire (even if they would normally be suppressed by predicted death).

**Tactical decision:** You must choose your engagements carefully. Once committed, a squad cannot switch targets. If you commit your DPS to a low-value target by mistake, that DPS is locked in. But committed squads are faster and more tenacious fighters. Rewards careful target assessment and long-term planning within a battle.

### 7. Doctrine of Asymmetry

**Constraint:** You may only bring squads with odd-numbered unit counts (1, 3, 5, 7, 9). Squads with even numbers of living units are not deployed. This is checked at deployment, not during combat -- if units die mid-battle and the count becomes even, no penalty applies.

**Power:** After all factions have completed their turns each round, every friendly squad gets a bonus movement-only activation. During this bonus phase, each squad may move up to 2 tiles but cannot attack or cast spells. This is a separate activation added to the end of the round, not a modification to normal turns.

**Tactical decision:** Building odd-count squads constrains your army composition. You cannot have neat 2x3 or 3x3 formations. But every squad gets a free repositioning phase at the end of each round -- enabling hit-and-run tactics, flanking setups, and retreat from overextended positions. Build lean, mobile assault squads.

---

## Zero-Overlap Proof

**vs Perks:** Perks modify stats (`Attributes`), damage calculations (`DamageModifiers`), cover values (`CoverBreakdown`), and targeting logic (`SelectTargetUnits`). No Doctrine modifies any of these. Doctrine powers operate on: action availability (`HasActed` lockouts), movement economy (`MovementRemaining` resets/caps), attack resolution ordering (counterattack sequencing), enemy movement (`MovementRemaining` on enemy squads), and spell access (`SpellBookData` lockout). These are structural combat flow changes, not damage/stat pipeline modifications.

**vs Spells:** Spells are active, targeted, mana-consuming combat abilities. No Doctrine can be "cast." No Doctrine targets a specific squad with an effect during a turn. No Doctrine uses mana. Doctrine of Silence explicitly disables spells, making the relationship oppositional rather than overlapping.

**vs Equipment (Turn/Action Economy):** Equipment items are one-shot tactical activations that grant bonus actions or deny specific actions at a chosen moment during combat. Doctrines impose persistent, battle-wide rules that constrain or restructure how combat systems work for the entire fight. The distinction is **tactical activation** (Equipment) vs **strategic commitment** (Doctrines). Both touch `ActionStateData`, but in fundamentally different ways.

---

## Existing System Interactions

| System | Interaction |
|--------|------------|
| `TurnManager.InitializeCombat` | Doctrine constraints checked and applied at battle start |
| `ActionStateData.HasActed` | Conservation locks attacks on even rounds |
| `ActionStateData.MovementRemaining` | Fortification caps at 1; Commitment adds +1; Attrition restores on first-hit |
| `ActionStateData.MovementRemaining` (enemy) | Fortification sets enemy movement to 0 when adjacent |
| `ExecuteAttackAction` counterattack phase | Silence disables enemy counters; Sacrifice reorders resolution |
| `ExecuteAttackAction` attack resolution | Conservation triggers second attack; Commitment tracks target lock |
| `SpellBookData` / `ExecuteSpellCast` | Silence locks spell access |
| Squad deployment screen | Asymmetry validates odd unit counts; Sacrifice removes lowest-level unit |
| `TurnManager.EndTurn` | Asymmetry adds bonus movement-only phase after all factions |

---

## Implementation Complexity: Medium

**Systems touched:** Pre-battle UI (Doctrine selection), `TurnManager`, `ActionStateData`, `CombatActionSystem`, squad deployment validation

**New infrastructure needed:**
- Doctrine selection screen (pre-battle, one choice)
- Active Doctrine state tracked on a combat state entity (not on individual squads)
- Constraint enforcement hooks (prevent illegal actions based on active Doctrine)
- Power application hooks (modify combat flow based on active Doctrine)
- Target tracking per squad (Attrition: which targets have been attacked; Commitment: locked target)

**Risk:** Lower than Equipment. Most Doctrines add simple if-checks to existing code paths ("is Doctrine of Conservation active AND is this an even round? Then block the attack"). The constraint side is actually easier to implement than the power side -- blocking an action is simpler than granting one.

---

## Interaction with the Equipment System

Doctrines set battle-wide rules. Equipment provides tactical activations within those rules. The two systems multiply each other's value:

### Complementary Examples

1. **Doctrine of Silence + Chain of Command Scepter:** You gave up spells. Your support squad passes its attack to your DPS squad (Chain of Command). Three systems working together -- Doctrine sets the constraint, Equipment exploits the tempo, Perks amplify the damage.

2. **Doctrine of Conservation + Forced Engagement Chains:** You attack every other round with double damage. On your attack rounds, if you destroy a squad, Chains gives you a free move to reposition for your next devastating strike. On off-rounds, you reposition without equipment cost.

3. **Doctrine of Fortification + Stand Down Orders:** You barely move, but enemies that approach you get stuck adjacent. When the enemy's strongest squad is about to attack your position, Stand Down Orders cancels its attack entirely. Layered defense from two different systems.

4. **Doctrine of Conservation + Chain of Command Scepter:** On off-rounds where your squads can't attack, Chain of Command Scepter lets you pass those useless attack actions to a squad that benefits. The Doctrine's downside becomes manageable through smart equipment use.

5. **Doctrine of Commitment + Commander's Initiative Badge:** Your squads are locked to their targets but faster. Going first every round (Initiative Badge) means your committed squads always reach their targets before the enemy can reposition. Tempo control reinforces target commitment.
