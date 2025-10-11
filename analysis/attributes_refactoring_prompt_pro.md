# Refactoring-Pro Analysis: Attributes System Refactoring

## Agent Role
You are **refactoring-pro**, focused on pragmatic architectural improvements for maintainability, extensibility, and complexity reduction.

## Current System Analysis

### Attributes Struct (common/commoncomponents.go)
```go
type Attributes struct {
    MaxHealth          int
    CurrentHealth      int
    AttackBonus        int
    BaseArmorClass     int
    BaseProtection     int
    BaseMovementSpeed  int
    BaseDodgeChance    float32
    TotalArmorClass    int
    TotalProtection    int
    TotalDodgeChance   float32
    TotalMovementSpeed int
    TotalAttackSpeed   int
    CanAct             bool
    DamageBonus        int
}
```

### Current Usage Patterns

**1. JSON Loading (entitytemplates/jsonstructs.go)**
```go
type JSONAttributes struct {
    MaxHealth         int     `json:"MaxHealth"`
    AttackBonus       int     `json:"AttackBonus"`
    BaseArmorClass    int     `json:"BaseArmorClass"`
    BaseProtection    int     `json:"BaseProtection"`
    BaseDodgeChance   float32 `json:"BaseDodgeChance"`
    BaseMovementSpeed int     `json:"BaseMovementSpeed"`
    DamageBonus       int     `json:"damagebonus"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {
    return common.NewBaseAttributes(
        attr.MaxHealth,
        attr.AttackBonus,
        attr.BaseArmorClass,
        attr.BaseProtection,
        attr.BaseMovementSpeed,
        attr.BaseDodgeChance,
        attr.DamageBonus,
    )
}
```

**2. Combat Calculations (squads/squadcombat.go)**
```go
func calculateUnitDamageByID(attackerID, defenderID ecs.EntityID, squadmanager *SquadECSManager) int {
    attackerAttr := common.GetAttributes(attackerUnit)
    defenderAttr := common.GetAttributes(defenderUnit)

    // Base damage calculation
    baseDamage := attackerAttr.AttackBonus + attackerAttr.DamageBonus

    // Apply defense
    totalDamage := baseDamage - defenderAttr.TotalProtection
    if totalDamage < 1 {
        totalDamage = 1
    }

    return totalDamage
}
```

**3. Consumable Effects (gear/consumables.go)**
```go
func (c *Consumable) ApplyEffect(baseAttr *common.Attributes) {
    baseAttr.AttackBonus += c.AttrModifier.AttackBonus
    baseAttr.BaseArmorClass += c.AttrModifier.BaseArmorClass
    baseAttr.BaseDodgeChance += c.AttrModifier.BaseDodgeChance
    baseAttr.BaseProtection += c.AttrModifier.BaseProtection

    if c.AttrModifier.BaseMovementSpeed != 0 {
        baseAttr.BaseMovementSpeed = c.AttrModifier.BaseMovementSpeed
    }
}

func UpdateEntityAttributes(e *ecs.Entity) {
    attr := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)

    // Recalculate totals
    attr.TotalArmorClass = attr.BaseArmorClass
    attr.TotalProtection = attr.BaseProtection
    attr.TotalDodgeChance = attr.BaseDodgeChance
    attr.TotalMovementSpeed = attr.BaseMovementSpeed
}
```

**4. JSON Data (assets/gamedata/monsterdata.json)**
```json
{
  "name": "GoblinWarrior",
  "attributes": {
    "maxHealth": 15,
    "attackBonus": 1,
    "baseArmorClass": 10,
    "baseProtection": 1,
    "baseDodgeChance": 0.1,
    "baseMovementSpeed": 3,
    "damagebonus": 0
  }
}
```

## Desired New System

### Six Core Attributes
1. **Strength** → Physical Damage, Physical Resistance
2. **Dexterity** → Hit Rate, Crit Chance, Dodge
3. **Magic** → Magic Damage, Healing Amount, Magic Defense
4. **Leadership** → Unit Capacity (larger/smaller squads)
5. **Armor** → Damage Reduction Modifier
6. **Weapon** → Damage Increase Modifier

### Design Goals
- Derive combat stats from core attributes (e.g., Strength 10 → +5 Physical Damage)
- Support tactical depth through attribute tradeoffs
- Enable squad size mechanics via Leadership
- Maintain ECS architectural purity (pure data components)
- Support JSON-driven configuration
- Preserve existing combat system functionality

## Current Architectural Issues

1. **Mixed Responsibilities**: Base stats + derived stats in same struct
2. **Manual Recalculation**: Total stats updated manually in multiple places
3. **No Derivation System**: No formulas connecting attributes to combat stats
4. **Flat Design**: Doesn't support rich tactical attribute interactions
5. **No Leadership Mechanic**: Can't vary squad sizes based on attributes
6. **Scattered Updates**: UpdateEntityAttributes() called from consumables, but pattern not enforced

## Files Heavily Using Attributes (Impact Analysis)

- `common/commoncomponents.go` - Struct definition
- `entitytemplates/jsonstructs.go` - JSON loading
- `entitytemplates/creators.go` - Entity creation
- `squads/squadcombat.go` - Combat calculations
- `squads/units.go` - Unit templates
- `gear/consumables.go` - Temporary modifications
- `monsters/creatures.go` - Display and usage
- `assets/gamedata/monsterdata.json` - Data definitions

## Required Output

Provide **EXACTLY 3 distinct refactoring approaches** with:

### For Each Approach:

1. **Name**: Clear, descriptive name
2. **Problem Statement**: What specific architectural issue does this solve?
3. **Code Examples**: Before/After comparisons in Go
4. **Key Structural Changes**: List of major modifications
5. **Derivation Formulas**: How core attributes → combat stats (if applicable)
6. **Pros**: Advantages with specific examples
7. **Cons**: Disadvantages and risks with mitigation strategies
8. **Effort Estimate**: Hours to implement
9. **File Impact**: Which files need changes and what kind

### Approach Diversity Requirements

Ensure your 3 approaches differ along these dimensions:
- **Incremental vs Revolutionary**: Some approaches should be evolutionary, others transformative
- **Complexity**: Range from simple (low risk) to sophisticated (high value)
- **Derivation Strategy**: Different ways to calculate combat stats from attributes
- **ECS Integration**: Different levels of ECS pattern sophistication

### Quality Standards

- **Concrete Code**: Show actual Go structs, functions, formulas
- **Tactical Game Context**: Remember this is squad-based tactical combat, NOT web/CRUD
- **ECS Compliance**: Maintain pure data components, no logic methods
- **Practical Balance**: Theory (DRY, SOLID, KISS, YAGNI, SLAP, SOC) + practice
- **JSON Compatibility**: Show how JSON data structure changes

## Example Formula Patterns (for inspiration)

```go
// Example: Strength → Physical Damage
PhysicalDamage = WeaponBaseDamage + (Strength / 2)

// Example: Dexterity → Dodge Chance
DodgeChance = BaseDodge + (Dexterity * 0.02)

// Example: Leadership → Squad Size
MaxSquadSize = 6 + (Leadership / 3)
```

---

**Remember**: You are refactoring-pro. Focus on **pragmatic architectural improvements** that balance clean design with implementation reality. Avoid over-engineering. Provide actionable recommendations with clear migration paths.
