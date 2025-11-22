# Tactical Combat Design Skill

**Purpose**: Tactical RPG mechanics design and balance suggestions
**Trigger**: When discussing squad formations, combat formulas, or ability systems

## Capabilities

- Formation balance analysis (3x3 grid patterns)
- Combat formula suggestions (hit/dodge/crit/cover)
- Ability trigger condition recommendations
- Turn order and initiative systems
- Squad capacity balancing

## Formation Design Principles

### 3x3 Grid Tactical Positioning

```
Front Row:    [0,0] [1,0] [2,0]  ← Frontline fighters, high HP/defense
Middle Row:   [0,1] [1,1] [2,1]  ← Balanced units, flexibility
Back Row:     [0,2] [1,2] [2,2]  ← Support/ranged, lower HP, protected
```

### Formation Archetypes

**Balanced Formation**:
```
Front:  [Tank]  [DPS]   [DPS]
Middle: [DPS]   [Empty] [Support]
Back:   [Healer][Empty] [Support]

Strategy: Versatile, no weaknesses, no strengths
Win Rate: ~50% against all formations
```

**Defensive Formation**:
```
Front:  [Tank]  [Tank]  [Tank]
Middle: [Healer][Empty] [Healer]
Back:   [Support][Support][Support]

Strategy: Outlast opponent, high survivability
Strengths: Counters burst damage (Offensive)
Weaknesses: Vulnerable to sustained DPS (Ranged)
```

**Offensive Formation**:
```
Front:  [Tank]  [DPS]   [DPS]
Middle: [DPS]   [DPS]   [DPS]
Back:   [DPS]   [Empty] [Support]

Strategy: Overwhelm with damage before taking losses
Strengths: Burst damage, kills before dying
Weaknesses: Fragile, loses to sustained combat (Defensive)
```

**Ranged Formation**:
```
Front:  [Tank]  [Empty] [Tank]
Middle: [DPS]   [Empty] [DPS]
Back:   [Ranged][Ranged][Ranged]

Strategy: Damage from safety, cover mechanics
Strengths: Back row protected, sustained DPS
Weaknesses: Lower burst, vulnerable to aggressive play
```

## Combat Mechanics Design

### Hit/Miss System

**Formula**:
```go
baseHitChance := 0.75  // 75% base accuracy
dodgeReduction := defender.Dodge * 0.01
finalHitChance := baseHitChance - dodgeReduction

hit := rand.Float64() < finalHitChance
```

**Balance**:
- Base 75% hit rate feels fair (not too swingy)
- Dodge stat provides meaningful but not overwhelming evasion
- High dodge units (15%) achieve ~60% hit against them
- Low dodge units (5%) achieve ~70% hit against them

### Damage Calculation

**Formula**:
```go
baseDamage := attacker.Attack - defender.Defense
damage := max(baseDamage, 1)  // Minimum 1 damage

// Critical hits
if rand.Float64() < attacker.CritRate {
    damage *= 2  // 2x damage on crit
}

// Cover mechanics (back row)
if targetInBackRow && frontRowAlive {
    damage *= 0.5  // 50% damage reduction
}
```

**Balance**:
- Defense reduces but never negates damage (min 1)
- Crit rate ~20% provides excitement without dominance
- Cover bonus rewards tactical positioning (protect back line)

### Ability Trigger Systems

**HP Threshold Triggers**:
```go
type Ability struct {
    Name           string
    TriggerType    string  // "hp_threshold"
    TriggerValue   float64 // 0.3 = 30% HP
    Effect         func()
    Cooldown       int     // Turns between uses
}

// Trigger check
currentHPPercent := float64(unit.CurrentHP) / float64(unit.MaxHP)
if currentHPPercent <= ability.TriggerValue && ability.OffCooldown() {
    ability.Trigger()
}
```

**Trigger Type Design**:

1. **HP Threshold** (e.g., Rally at HP < 30%)
   - Good for: Comeback mechanics, desperate moves
   - Balance: 30% too late (combat over), 50% better

2. **Turn Count** (e.g., Shield Wall at Turn 1)
   - Good for: Setup abilities, defensive buffs
   - Balance: Early turns (1-3) for setup, mid-game (5-8) for escalation

3. **Enemy Count** (e.g., AOE when 3+ enemies nearby)
   - Good for: Situational tactics, positioning rewards
   - Balance: 2+ enemies (common), 4+ enemies (rare power moment)

4. **Morale/Status** (e.g., Battle Cry when morale high)
   - Good for: Momentum mechanics, snowballing
   - Balance: Requires morale system integration

5. **Combat Start** (e.g., First Strike on turn 1)
   - Good for: Consistent openers, alpha strike
   - Balance: Always triggers, design around certainty

### Stat Scaling

**Unit Stat Ranges**:
```
Tank:     HP: 150-200, Attack: 10-15, Defense: 8-12, Dodge: 3-5%,  Crit: 5-10%
DPS:      HP: 80-100,  Attack: 18-25, Defense: 2-5,  Dodge: 10-15%, Crit: 20-30%
Support:  HP: 70-90,   Attack: 8-12,  Defense: 2-4,  Dodge: 5-10%,  Crit: 10-15%
Healer:   HP: 80-100,  Attack: 6-10,  Defense: 2-4,  Dodge: 5-10%,  Crit: 3-8%
Ranged:   HP: 75-95,   Attack: 15-20, Defense: 2-4,  Dodge: 8-12%,  Crit: 15-25%
```

**Balance Guidelines**:
- Tank HP 2x DPS HP (survivability role)
- DPS Attack 1.5-2x Tank Attack (damage role)
- Support balanced across stats (versatility)
- High Dodge ↔ Low HP (glass cannon trade-off)
- High Crit ↔ Lower base Attack (RNG vs consistency)

## Formation Balance Matrix

**Ideal Rock-Paper-Scissors**:
```
Offensive > Defensive (60% win rate) - Burst overwhelms tanks
Defensive > Ranged    (60% win rate) - Tanks absorb ranged poke
Ranged    > Offensive (60% win rate) - Kites aggressive play
Balanced  ≈ All       (50% win rate) - Jack of all trades
```

**Balancing Process**:
1. Simulate 1000 combats per matchup (use combat-simulator agent)
2. Identify win rates outside 45-55% range
3. Adjust stats/abilities to create counter-relationships
4. Re-simulate and validate

## Tactical Depth Principles

**1. Positional Advantage (Cover)**:
- Back row gets damage reduction when front row alive
- Encourages protecting squishy units
- Rewards killing front row first

**2. Role Diversity**:
- Tank: High HP, draws aggro, protects back line
- DPS: High damage, fragile, kills priority targets
- Support: Buffs/debuffs, utility, force multiplier
- Healer: Sustains squad, extends combat duration

**3. Ability Timing**:
- Setup abilities (turn 1-3): Shield Wall, Battle Cry
- Mid-game abilities (turn 4-8): Rally, Heal
- Desperation abilities (HP < 30%): Last Stand, Frenzy

**4. Counter-Play**:
- Offensive beats Defensive (burst before sustain)
- Defensive beats Ranged (outlasts poke)
- Ranged beats Offensive (kites aggression)

## Common Balance Issues

**Issue: Defensive Too Strong**
- Symptom: Wins 60%+ against all formations
- Cause: Tank HP too high, sustain too strong
- Fix: Reduce Tank HP by 10-15% OR reduce heal effectiveness

**Issue: Offensive Too Weak**
- Symptom: Wins <40% against most formations
- Cause: DPS dies before dealing enough damage
- Fix: Increase DPS attack by 10% OR add front-row tank

**Issue: Abilities Trigger Too Rarely**
- Symptom: Rally triggers <20% of combats (HP < 30%)
- Cause: Threshold too low, combat ends first
- Fix: Increase trigger threshold to 40-50% HP

**Issue: No Tactical Decisions**
- Symptom: Optimal formation always wins
- Cause: No rock-paper-scissors relationship
- Fix: Create counter-relationships via stat adjustments

## Turn Order & Initiative

**Simple Initiative System**:
```go
// Speed stat determines turn order
type Unit struct {
    Speed int  // 5-15 range
}

// Sort units by speed (highest first)
sort.Slice(allUnits, func(i, j int) bool {
    return allUnits[i].Speed > allUnits[j].Speed
})

// Execute turns in order
for _, unit := range allUnits {
    unit.ExecuteTurn()
}
```

**Initiative Balance**:
- Speed range: 5-15 (reasonable variance)
- Fast units (13-15): DPS, Ranged (offensive advantage)
- Medium units (8-12): Support, Balanced (flexibility)
- Slow units (5-7): Tank, Healer (defensive, react to threats)

## Squad Capacity System

**Capacity Cost Balance**:
```
Tank:    Cost: 3 (expensive, high impact)
DPS:     Cost: 2 (medium, damage output)
Support: Cost: 1.5 (cheap, utility)
Healer:  Cost: 2 (medium, sustain)
Ranged:  Cost: 2 (medium, safety)

Max Squad Capacity: 12-15 points
```

**Capacity Design**:
- Encourages diverse squads (can't stack all tanks)
- Tank-heavy = fewer units (defensive trade-off)
- DPS-heavy = fragile (offensive trade-off)
- Balanced = more units (flexibility)

## Fire Emblem / FFT Inspiration

**Fire Emblem Principles**:
- Weapon triangle (sword > axe > lance > sword)
- Terrain bonuses (forest = +dodge, mountain = +defense)
- Support bonuses (adjacent allies boost stats)

**FFT Principles**:
- Height advantage (+damage from higher ground)
- Charge time (powerful abilities need setup turns)
- Job system (unit roles with clear strengths/weaknesses)

**Adaptation for This Project**:
- Formation = weapon triangle (Offensive/Defensive/Ranged)
- Cover = terrain (back row gets bonus)
- Abilities = charge time (cooldowns)

## Usage Example

```markdown
User: "How should I balance the Offensive formation?"

Response:
Offensive formation should be high-risk, high-reward. Here's the design:

**Unit Distribution**:
- Front: 1 Tank, 2 DPS (protect glass cannons)
- Middle: 3 DPS (maximize damage output)
- Back: 1 DPS, 1 Support (sustained damage + utility)

**Stat Profile**:
- Tank: 150 HP, 12 Attack, 8 Defense
- DPS: 90 HP, 20 Attack, 3 Defense, 25% Crit

**Strategy**: Burst down enemy front line before your fragile DPS die

**Counter**: Defensive formation (tanks outlast your burst)

**Counter-Strategy**: Focus fire single tank to break front line faster

Reference: FFT aggressive job compositions (Ninja, Monk, Thief for burst)
```

## Formation Testing Workflow

1. Design formation (unit placement + stats)
2. Identify intended counter-relationships
3. Use `/squad-balance [formation]` to analyze
4. Use `combat-simulator` agent to simulate matchups
5. Adjust stats based on win rate data (target: 45-55%)
6. Re-simulate and validate balance

## Key Takeaway

**Tactical depth comes from meaningful choices**:
- Formation selection (counter-pick opponent)
- Positioning (protect back line, focus fire)
- Ability timing (use Rally when it matters)
- Squad composition (balance damage, survivability, utility)

**Balance is about trade-offs, not equality**:
- Offensive = high damage, low survivability
- Defensive = high survivability, low damage
- Ranged = safe damage, lower burst
- Balanced = no weakness, no strength
