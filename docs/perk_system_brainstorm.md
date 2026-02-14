# Perk/Trait System Brainstorm

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Squad + Unit + Commander | Three layers of customization |
| Acquisition | Flat unlock pool | Spend perk points to unlock any perk freely |
| Permanence | Swappable (limited slots) | Equip from unlocked pool before each battle |
| Complexity | Moderate (~40-50 perks) | Mix of stat boosts and mechanical changes |
| Scaling | Flat (single version each) | Each perk has one version. Easier to balance |
| Unlock tree | Flat pool | No branching restrictions. Player chooses freely |

## Slot Counts
- **Squad perks**: 2-3 equippable slots
- **Unit perks**: 1-2 equippable slots
- **Commander perks**: 2-3 equippable slots
- Unlocked perks go into a shared pool; equip before battle

## Perk Points Economy
- Earned from battles (more for harder fights)
- Possibly from events, shops, or campaign milestones
- Spent to unlock perks permanently; equipping is free

## Restrictions
- Some perks are role-gated (Tank/DPS/Support only)
- Some pairs are mutually exclusive (e.g., Glass Cannon + Iron Wall)
- Squad, unit, and commander perks are separate pools

---

## CATEGORY 1: Specialization (Focused Squads)

Reward committing to a specific squad identity.

### Squad-Level

| Perk | Effect | Notes |
|------|--------|-------|
| **Shield Wall** | All front-row Tanks: +30% cover value | Rewards pure tank front line |
| **Glass Cannon** | All units +35% damage dealt, +20% damage taken | All-in offense |
| **Ambush Doctrine** | +50% damage on first attack of combat | Alpha strike squads |
| **Phalanx** | +0.15 cover per Tank in squad (stacks) | More tanks = more protection |
| **Wolf Pack** | +5% damage per DPS unit in squad (stacks) | DPS-heavy reward |
| **Battle Hymn** | Support units grant +3 to primary stat to all allies | Rewards bringing supports |

### Unit-Level

| Perk | Role Gate | Effect |
|------|-----------|--------|
| **Bulwark** | Tank | +50% cover value, but can't deal damage |
| **Berserker** | DPS | +30% damage when below 50% HP |
| **Sharpshooter** | DPS (Ranged) | +1 attack range, +15% crit chance |
| **War Medic** | Support | Heals 3 HP to lowest-HP ally each turn (passive) |
| **Vanguard** | Tank | Always targeted first in squad (taunt) |

---

## CATEGORY 2: Generalization (Versatile Squads)

Reward diverse compositions and flexibility.

### Squad-Level

| Perk | Effect | Notes |
|------|--------|-------|
| **Combined Arms** | If all 3 roles present: +10% dmg, +10% def, +5 morale | Synergy for variety |
| **Flexible Doctrine** | Switch formation once per battle (free action) | Mid-fight adaptation |
| **Reserves** | +2 max unit capacity, all units -5% stats | Quantity over quality |
| **Veteran Cohesion** | +2% all stats per unique unit type in squad | Diverse roster bonus |
| **Adaptive Tactics** | After taking dmg: +10% def 1 turn. After dealing dmg: +10% atk 1 turn | Reactive playstyle |

---

## CATEGORY 3: Attack Pattern & Targeting Modifiers

Change *how* units attack, creating varied combat feel.

### Unit-Level

| Perk | Effect | System Interaction |
|------|--------|-------------------|
| **Cleave** | MeleeRow hits target row + row behind | Extends row targeting depth |
| **Impale** | MeleeColumn ignores cover entirely | Bypasses cover calculation |
| **Suppressing Fire** | Ranged attacks apply -15% hit rate to targets for 1 turn | Uses effects system |
| **Chain Cast** | Magic bounces to 1 adjacent cell beyond pattern | Extends magic cell patterns |
| **Focus Fire** | Hit single target at 2x damage instead of row/column | Trades AoE for burst |
| **Scatter Shot** | Ranged hits target row + adjacent row at 70% damage | AoE trade-off |

### Squad-Level

| Perk | Effect | Notes |
|------|--------|-------|
| **Coordinated Assault** | 2+ units hitting same row: +20% damage for all | Rewards row focus |
| **Pincer Attack** | Units in different columns: +10% dmg to center column targets | Flanking bonus |

---

## CATEGORY 4: Attribute Modifiers

Straightforward stat boosts, meaningful but not game-breaking.

### Unit-Level

| Perk | Effect |
|------|--------|
| **Iron Constitution** | +20% max HP |
| **Quick Reflexes** | +15% dodge chance |
| **Precise Strikes** | +10% hit rate, +10% crit chance |
| **Hardened Armor** | +4 armor |
| **Arcane Attunement** | +5 magic stat |
| **Fleet Footed** | +1 movement speed |

### Squad-Level

| Perk | Effect |
|------|--------|
| **Forced March** | +1 squad movement speed |
| **Fortified Position** | +0.1 cover to all units when squad hasn't moved this turn |

---

## CATEGORY 5: Attack & Counterattack Modifiers

The existing counterattack system (50% damage, -20% hit) is rich design space.

### Counterattack Modifiers (Unit-Level)

| Perk | Effect | Notes |
|------|--------|-------|
| **Riposte** | Counterattacks deal 100% damage (not 50%) | Major offensive counter |
| **Parry** | 25% chance to negate an incoming attack | Defensive RNG |
| **Vengeful Strike** | Counterattacks have +20% crit chance | Punish attackers with crits |
| **Stone Wall** | Can't counterattack, but take 30% less damage | Pure defense trade-off |
| **Preemptive Strike** | When attacked, deal damage BEFORE attacker resolves | Reverses attack order |
| **Retribution** | When an ally in your row dies, next attack +50% damage | Vengeance mechanic |

### Attack Modifiers (Unit-Level)

| Perk | Effect | Notes |
|------|--------|-------|
| **Double Strike** | 25% chance to attack twice | High variance |
| **Armor Piercing** | Ignore 50% of target's armor | Anti-tank |
| **Reckless Assault** | +30% damage dealt, can't counterattack this turn | Aggressive trade-off |
| **Executioner** | +50% damage vs targets below 30% HP | Finisher |

---

## CATEGORY 6: Interesting Depth Modifiers

Create emergent strategies and unique squad identities.

### Unit-Level

| Perk | Effect | Notes |
|------|--------|-------|
| **Lifesteal** | Heal 25% of damage dealt | Sustain through aggression |
| **Last Stand** | Last unit alive: +50% all combat stats | Dramatic comeback |
| **Inspiration** | On kill: all allies +2 strength for 2 turns | Kill snowball |
| **Guardian** | Takes 50% of damage dealt to adjacent allies | Bodyguard |
| **Overwatch** | Skip attack this turn: counterattack at 150% damage next turn | Patience reward |
| **Adrenaline** | +5% damage per turn of combat (stacks, resets per battle) | Long fight reward |

### Squad-Level

| Perk | Effect | Notes |
|------|--------|-------|
| **Esprit de Corps** | Morale floor at 40. Would-be drops become +5% damage | Morale stability |
| **Scavengers** | Win combat: restore 15% HP to all survivors | Cross-battle sustain |
| **Tactical Withdrawal** | Below 30% squad HP: +2 movement, can disengage | Escape mechanism |
| **Shock and Awe** | Turn 1: +25% damage, +15% hit rate | Blitz squads |
| **Dig In** | Each turn without moving: +5% defense (max +25%) | Hold position reward |
| **Underdog** | Facing higher-power squad: +15% all stats | Comeback mechanic |

---

## CATEGORY 7: Commander Perks

Modify how the commander uses spells and interacts with combat.

### Spell Enhancement

| Perk | Effect | Notes |
|------|--------|-------|
| **Efficient Casting** | -20% mana cost on all spells | More casts per battle |
| **Spell Mastery** | +25% spell damage | Raw power boost |
| **Overcharge** | Spells cost +50% mana but deal +75% damage | High risk/reward |
| **Wide Spread** | AoE spells: +1 size in all dimensions | Bigger blast radius |
| **Focused Intent** | Single-target spells: +40% damage | Single target specialist |

### Buff/Debuff Enhancement

| Perk | Effect | Notes |
|------|--------|-------|
| **Lingering Magic** | Buff/debuff durations +2 turns | Longer lasting effects |
| **Potent Enchantment** | Buff/debuff stat modifiers +50% stronger | Stronger effects |
| **Curse Specialist** | Debuffs also reduce target hit rate by 10% | Extra debuff penalty |
| **Inspiring Presence** | Buffs also grant +5 morale to target squad | Morale synergy |

### Combat Influence

| Perk | Effect | Notes |
|------|--------|-------|
| **Battle Sense** | Commander can see enemy squad compositions before combat | Information advantage |
| **Rally Commander** | Once per battle: reset one squad's action (attack again) | Powerful but limited |
| **Mana Regeneration** | Restore 10% max mana at end of each battle | Cross-battle sustain |
| **War Sage** | Spells cast on allied squads also grant +2 armor for 1 turn | Defensive spell bonus |

---

## Build Archetypes (Synergy Examples)

### "The Fortress" (Defensive)
- **Squad**: Shield Wall + Dig In
- **Front Tanks**: Bulwark + Stone Wall
- **Back DPS**: Sharpshooter + Focus Fire
- **Commander**: War Sage + Inspiring Presence
- *Strategy*: Park, absorb, pick off from behind impenetrable front line

### "Blitz Squad" (Alpha Strike)
- **Squad**: Ambush Doctrine + Shock and Awe
- **DPS units**: Reckless Assault + Double Strike
- **Commander**: Overcharge + Spell Mastery
- *Strategy*: Devastate turn 1, vulnerable after. Win fast or lose.

### "The Grinder" (Attrition)
- **Squad**: Combined Arms + Scavengers
- **Tank**: Riposte + Iron Constitution
- **DPS**: Lifesteal + Adrenaline
- **Support**: War Medic
- **Commander**: Mana Regeneration + Lingering Magic
- *Strategy*: Outlast, heal, get stronger over time

### "Glass Cannons" (Max Damage)
- **Squad**: Glass Cannon + Wolf Pack
- **All DPS**: Berserker + Armor Piercing
- **Commander**: Spell Mastery + Focused Intent
- *Strategy*: Annihilate before being annihilated

### "The Anvil & Hammer"
- **Squad**: Coordinated Assault + Phalanx
- **Tanks**: Vanguard + Guardian
- **DPS**: Executioner + Precise Strikes
- **Commander**: Rally Commander + Efficient Casting
- *Strategy*: Tanks hold the line, DPS finishes weakened targets

---

## Implementation Approach (High-Level)

### Leveraging Existing Systems
- **Stat perks**: Use `ActiveEffect` with `RemainingTurns = -1` (permanent). Already supported.
- **Perk definitions**: Data-driven JSON (`perkdata.json`) like `spelldata.json`
- **Triggered perks**: Extend ability trigger system from `squadabilities.go`

### New Components Needed
- `PerkDefinition` - Data structure for perk templates (loaded from JSON)
- `SquadPerkSlots` - Component on squad entities (2-3 equipped perk IDs)
- `UnitPerkSlots` - Component on unit entities (1-2 equipped perk IDs)
- `CommanderPerkSlots` - Component on commander entity
- `PerkUnlockData` - Player-level tracking of unlocked perks + perk points

### Combat Pipeline Hooks (in `squadcombat.go`)
Perks that modify combat need hook points in the damage pipeline:
1. **Pre-attack** - Ambush Doctrine, Shock and Awe, Preemptive Strike
2. **Damage calculation** - Glass Cannon, Armor Piercing, Focus Fire, Berserker
3. **Post-hit** - Lifesteal, Suppressing Fire, Inspiration
4. **Pre-counterattack** - Riposte, Stone Wall, Overwatch
5. **On-death** - Retribution, Last Stand
6. **Turn-start** - Adrenaline, Dig In, Adaptive Tactics, War Medic

### Key Files to Modify
- `tactical/squads/squadcombat.go` - Damage pipeline hooks
- `tactical/combat/combatactionsystem.go` - Attack/counter flow
- `tactical/effects/system.go` - Permanent effect support
- `tactical/commander/system.go` - Commander perk integration
- `assets/gamedata/` - New `perkdata.json`
