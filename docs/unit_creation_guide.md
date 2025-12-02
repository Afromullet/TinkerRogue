# Unit Creation Guide for TinkerRogue

**Purpose:** Guidelines for creating new tactical units for monsterdata.json after understanding the combat system.

**Last Updated:** 2025-12-02

---

## Core System Reference

### Attribute Formulas

```
strength (0-18) → Physical Damage = (Str/2) + (Weapon*2)
                → Physical Resistance = (Str/4) + (Armor*2)
                → Max Health = 20 + (Str*2)

dexterity (6-75) → Hit Rate = min(100 + (Dex*2), 100%)
                 → Crit Chance = min(Dex/2, 50%)
                 → Dodge Chance = min(Dex/3, 40%)

magic (0-15) → Magic Damage = Magic * 3
             → Healing Amount = Magic * 2
             → Magic Defense = Magic / 2

leadership (3-80) → Squad Capacity = min(6 + (Lead/3), 9)

armor (0-12) → Physical Resistance = (Str/4) + (Armor*2)
             → Capacity Cost = (Str + Wpn + Armor) / 5.0

weapon (1-12) → Physical Damage = (Str/2) + (Weapon*2)
              → Capacity Cost = (Str + Wpn + Armor) / 5.0
```

### Targeting Modes

**Row-Based** (`targetMode: "row"`):
- `targetRows`: [0,1,2] where 0=front, 1=mid, 2=back
- `isMultiTarget`: Hit multiple units in row?
- `maxTargets`: Limit (0 = unlimited)

**Cell-Based** (`targetMode: "cell"`):
- `targetCells`: [[row,col]] coordinates in 3x3 grid
- Common patterns:
  - Vertical line: `[[0,1], [1,1], [2,1]]`
  - Horizontal line: `[[0,0], [0,1], [0,2]]`
  - 2x2 square: `[[0,0], [0,1], [1,0], [1,1]]`
  - Diagonal: `[[0,0], [1,1], [2,2]]`
  - Cross: `[[0,1], [1,0], [1,1], [1,2], [2,1]]`
  - Corners: `[[0,0], [0,2], [2,0], [2,2]]`

### Cover System

- `coverValue`: 0.0-0.5 (damage reduction %)
- `coverRange`: 1-2 (rows behind that get protection)
- `requiresActive`: Must be alive to provide cover
- Front-line units with cover protect units behind them

---

## Unit Design Guidelines

### Stat Allocation by Archetype

**Heavy Melee (Knight, Fighter, Warrior):**
- Str: 13-16, Dex: 25-30, Magic: 0
- Armor: 7-10, Weapon: 8-11
- Leadership: 15-50

**Fast Melee (Swordsman, Rogue, Assassin):**
- Str: 8-12, Dex: 40-60, Magic: 0
- Armor: 2-5, Weapon: 7-9
- Leadership: 6-15

**Ranged Physical (Archer, Crossbowman, Marksman):**
- Str: 6-10, Dex: 38-52, Magic: 0
- Armor: 2-5, Weapon: 7-9
- Leadership: 8-15

**Pure Casters (Wizard, Sorcerer, Warlock):**
- Str: 3-5, Dex: 20-26, Magic: 13-15
- Armor: 1-2, Weapon: 2-4
- Leadership: 15-25

**Support (Cleric, Priest, Druid):**
- Str: 6-8, Dex: 18-24, Magic: 10-12
- Armor: 3-4, Weapon: 3-5
- Leadership: 20-45

**Hybrids (Paladin, Battle Mage, Ranger):**
- Balanced across categories
- Moderate stats in 2-3 attributes
- Leadership: 15-40

**Large Creatures (Orc, Ogre, Troll):**
- Str: 17-18, Dex: 15-24, Magic: 0-3
- Armor: 8-12, Weapon: 10-12
- Size: 2x1 or 2x2
- Leadership: 12-30

### Role System Guidelines

**Tank Role:**
- Cover: 0.25-0.50
- Cover Range: 1-2
- Attack Range: 1-2 (melee/reach)
- Movement Speed: 2-3
- Targeting: Row 0 or front-focused cells
- RequiresActive: true

**DPS Role:**
- Cover: 0.0
- Attack Range: 1-4
- Movement Speed: 3-5
- Targeting: Varied (precision cells, back-row focus, AOE)
- RequiresActive: false

**Support Role:**
- Cover: 0.10-0.25
- Attack Range: 2-4
- Movement Speed: 2-4
- Targeting: Multi-target rows or AOE cells
- RequiresActive: true (if providing cover)

### Targeting Pattern Guidelines

**Tanks:**
- Row 0 single-target (challenge enemy tanks)
- Cell [[0,1]] or [[0,1], [1,1]] (center focus)
- Row 0 multi-target if large unit (maxTargets: 2)

**Melee DPS:**
- Front row sweep: `[[0,0], [0,1], [0,2]]`
- Precision: `[[0,1]]` or `[[2,1]]` (backline access)
- Corners: `[[2,0], [2,2]]` (flankers)

**Ranged DPS:**
- Row 2 single-target (sniper)
- Row targeting any row (flexible)
- Vertical line: `[[0,1], [1,1], [2,1]]`

**Mages:**
- 4-6 cell AOE patterns
- Row multi-target (maxTargets: 2-3)
- Cross or diagonal patterns

**Support:**
- Row 1-2 (mid/back healing)
- Multi-target rows (maxTargets: 2-3)
- Diagonal or vertical for utility

### Size Guidelines

**1x1 (Standard):**
- 90% of units
- All standard classes

**2x1 or 1x2 (Large):**
- Special creatures (Orcs, large beasts)
- Elite units
- Moderate stat boost

**2x2 (Colossus):**
- Rare mega-units (Ogre, Dragon)
- Very high stats
- High capacity cost to balance

---

## Validation Checklist

### Stat Validation

**Range Check:**
- [ ] Strength: 0-18
- [ ] Dexterity: 6-75
- [ ] Magic: 0-15
- [ ] Leadership: 3-80
- [ ] Armor: 0-12
- [ ] Weapon: 1-12

**Derived Stat Reasonability:**
- [ ] Max Health: 20-56 HP
- [ ] Physical Damage: 1-37
- [ ] Crit Chance: 3-37.5%
- [ ] Dodge Chance: 2-25%
- [ ] Magic Damage: 0-45
- [ ] Squad Capacity: 6-9 units

### Archetype Consistency

**Red Flags to Avoid:**
- Unrealistic Stats, such as 
- ❌ Fighter with Magic 15
- ❌ Wizard with Strength 15
- ❌ Tank with Armor 1
- ❌ Rogue with Dexterity 20
- ❌ Melee unit targeting row 2 with range 1

**Good Patterns:**
- ✅ Tank with high Armor and Cover 0.3+
- ✅ Assassin with Dex 55+ and backline targeting
- ✅ Mage with Magic 13+ and AOE pattern
- ✅ Healer with Magic 10+ and multi-target

### Targeting Pattern Validation

**Match targeting to role:**
- [ ] Melee units (range 1) target front/mid rows or accessible cells
- [ ] Long-range units (range 4) can target any row
- [ ] Tanks target front-line threats
- [ ] DPS units have offensive patterns
- [ ] Support units have area/multi-target options

**Pattern Diversity:**
- [ ] Mix row-based (~60%) and cell-based (~40%)
- [ ] Mix single-target and multi-target
- [ ] Include various AOE shapes

### Balance Checks

**Capacity Economics:**
```
capacityCost = (strength + weapon + armor) / 5.0
```
- [ ] Elite units (cost 4-5): ~5-6 per squad
- [ ] Standard units (cost 3-4): ~6-7 per squad
- [ ] Light units (cost 2-3): ~7-8 per squad

**Role Distribution (20-24 unit roster):**
- [ ] Tanks: 25-30% with Cover 0.25-0.50
- [ ] DPS: 40-45% with varied ranges
- [ ] Support: 25-30% with utility/healing

**Power Level:**
- [ ] High damage → Low defense or high capacity cost
- [ ] High defense → Lower damage output
- [ ] Large size → High capacity cost
- [ ] AOE damage → Lower single-target damage

---

## Quick Reference: Common Archetypes

### Heavy Knight
```json
{
  "name": "Knight",
  "attributes": {"strength": 14, "dexterity": 25, "magic": 0, "leadership": 50, "armor": 10, "weapon": 8},
  "width": 1, "height": 1,
  "role": "Tank",
  "targetMode": "row", "targetRows": [0], "isMultiTarget": false,
  "coverValue": 0.40, "coverRange": 2, "requiresActive": true,
  "attackRange": 1, "movementSpeed": 2
}
```

### Assassin
```json
{
  "name": "Assassin",
  "attributes": {"strength": 9, "dexterity": 60, "magic": 0, "leadership": 6, "armor": 2, "weapon": 8},
  "width": 1, "height": 1,
  "role": "DPS",
  "targetMode": "cell", "targetCells": [[2, 1]],
  "coverValue": 0.0, "coverRange": 0, "requiresActive": false,
  "attackRange": 1, "movementSpeed": 5
}
```

### Archer
```json
{
  "name": "Archer",
  "attributes": {"strength": 6, "dexterity": 45, "magic": 0, "leadership": 10, "armor": 2, "weapon": 7},
  "width": 1, "height": 1,
  "role": "DPS",
  "targetMode": "row", "targetRows": [2],
  "coverValue": 0.0, "coverRange": 0, "requiresActive": false,
  "attackRange": 4, "movementSpeed": 3
}
```

### Wizard (AOE)
```json
{
  "name": "Wizard",
  "attributes": {"strength": 3, "dexterity": 20, "magic": 15, "leadership": 25, "armor": 1, "weapon": 2},
  "width": 1, "height": 1,
  "role": "DPS",
  "targetMode": "cell", "targetCells": [[0,0], [0,1], [0,2], [1,0], [1,1], [1,2]],
  "coverValue": 0.0, "coverRange": 0, "requiresActive": false,
  "attackRange": 4, "movementSpeed": 2
}
```

### Cleric (Healer)
```json
{
  "name": "Cleric",
  "attributes": {"strength": 8, "dexterity": 24, "magic": 12, "leadership": 40, "armor": 4, "weapon": 4},
  "width": 1, "height": 1,
  "role": "Support",
  "targetMode": "row", "targetRows": [1, 2], "isMultiTarget": false,
  "coverValue": 0.20, "coverRange": 1, "requiresActive": true,
  "attackRange": 2, "movementSpeed": 3
}
```

### Battle Mage (Hybrid)
```json
{
  "name": "BattleMage",
  "attributes": {"strength": 11, "dexterity": 28, "magic": 10, "leadership": 22, "armor": 5, "weapon": 6},
  "width": 1, "height": 1,
  "role": "Support",
  "targetMode": "cell", "targetCells": [[0, 1], [1, 1]],
  "coverValue": 0.20, "coverRange": 1, "requiresActive": true,
  "attackRange": 2, "movementSpeed": 3
}
```

### Ogre (Colossus)
```json
{
  "name": "Ogre",
  "attributes": {"strength": 18, "dexterity": 15, "magic": 0, "leadership": 12, "armor": 12, "weapon": 12},
  "width": 2, "height": 2,
  "role": "Tank",
  "targetMode": "cell", "targetCells": [[0,0], [0,1], [0,2], [1,0], [1,1], [1,2]],
  "coverValue": 0.50, "coverRange": 2, "requiresActive": true,
  "attackRange": 1, "movementSpeed": 2
}
```

---

## Design Principles Summary

1. **Stats match archetype:** Fighters ≠ high magic, Wizards ≠ high strength
2. **Targeting makes tactical sense:** Match patterns to unit role and range
3. **Cover values match role:** Tanks 0.25-0.5, Support 0.1-0.25, DPS 0.0
4. **Balance through trade-offs:** High damage → Low defense or high cost
5. **Distinct tactical niches:** Each unit should encourage different squad builds
6. **Variety in all dimensions:** Stats, targeting, range, speed, size
7. **Capacity economics:** Most units allow 6+ per squad for tactical choices

---

## Using the TRPG-Creator Agent

When creating new units, use the `trpg-creator` agent with this template:

```
I need you to design [X] tactical RPG units for TinkerRogue.

## Context: Monster Data System

[Provide JSON structure example]

**Attribute Formulas:**
[Copy formulas from "Core System Reference" above]

**Targeting Modes:**
[Explain row-based vs cell-based with examples]

**Role System:**
Tank: High armor/strength, cover 0.25-0.5, melee, slow
DPS: High weapon/dex, low armor, ranged, fast
Support: Balanced stats, magic, moderate cover, mid-range

## Requirements:

1. Unit Archetypes: [List specific units wanted]
2. Stats must match archetype
3. Variety in targeting patterns
4. Cover values match role
5. Squad building variety

Provide complete JSON ready for monsterdata.json.
```

The agent will generate balanced units following tactical RPG design principles.