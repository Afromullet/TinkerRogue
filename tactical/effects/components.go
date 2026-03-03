package effects

import (
	"fmt"
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
// Returns an error if the stat name is not recognized.
func ParseStatType(stat string) (StatType, error) {
	switch strings.ToLower(stat) {
	case "strength":
		return StatStrength, nil
	case "dexterity":
		return StatDexterity, nil
	case "magic":
		return StatMagic, nil
	case "leadership":
		return StatLeadership, nil
	case "armor":
		return StatArmor, nil
	case "weapon":
		return StatWeapon, nil
	case "movementspeed":
		return StatMovementSpeed, nil
	case "attackrange":
		return StatAttackRange, nil
	default:
		return StatStrength, fmt.Errorf("unrecognized stat type %q; valid values: strength, dexterity, magic, leadership, armor, weapon, movementspeed, attackrange", stat)
	}
}

// ECS variables
var (
	ActiveEffectsComponent *ecs.Component
	ActiveEffectsTag       ecs.Tag
)
