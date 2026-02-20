package effects

import (
	"strings"

	"github.com/bytearena/ecs"
)

// StatType identifies which attribute a modifier targets.
type StatType int

const (
	StatStrength StatType = iota
	StatDexterity
	StatMagic
	StatLeadership
	StatArmor
	StatWeapon
	StatMovementSpeed
	StatAttackRange
)

// EffectSource identifies what created the effect (for filtering/removal).
type EffectSource int

const (
	SourceSpell EffectSource = iota
	SourceAbility
	SourceItem
	SourcePerk
)

// ActiveEffect is a single stat modifier with a duration.
type ActiveEffect struct {
	Name           string
	Source         EffectSource
	Stat           StatType
	Modifier       int // positive = buff, negative = debuff
	RemainingTurns int // -1 = permanent (equipment/perks), 0 = expired
}

// ActiveEffectsData is the ECS component attached to entities with active effects.
type ActiveEffectsData struct {
	Effects []ActiveEffect
}

// ParseStatType converts a JSON stat name string to a StatType.
func ParseStatType(stat string) StatType {
	switch strings.ToLower(stat) {
	case "strength":
		return StatStrength
	case "dexterity":
		return StatDexterity
	case "magic":
		return StatMagic
	case "leadership":
		return StatLeadership
	case "armor":
		return StatArmor
	case "weapon":
		return StatWeapon
	case "movementspeed":
		return StatMovementSpeed
	case "attackrange":
		return StatAttackRange
	default:
		return StatStrength
	}
}

// ECS variables
var (
	ActiveEffectsComponent *ecs.Component
	ActiveEffectsTag       ecs.Tag
)
