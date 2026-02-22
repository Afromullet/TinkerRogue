package chunks

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// --- Shared serialization structs ---
// These are used across multiple chunks (player, commander, squad).

type savedPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type savedAttributes struct {
	Strength      int  `json:"strength"`
	Dexterity     int  `json:"dexterity"`
	Magic         int  `json:"magic"`
	Leadership    int  `json:"leadership"`
	Armor         int  `json:"armor"`
	Weapon        int  `json:"weapon"`
	MovementSpeed int  `json:"movementSpeed"`
	AttackRange   int  `json:"attackRange"`
	CurrentHealth int  `json:"currentHealth"`
	MaxHealth     int  `json:"maxHealth"`
	CanAct        bool `json:"canAct"`
}

// --- Conversion helpers ---

func attributesToSaved(attr *common.Attributes) savedAttributes {
	return savedAttributes{
		Strength:      attr.Strength,
		Dexterity:     attr.Dexterity,
		Magic:         attr.Magic,
		Leadership:    attr.Leadership,
		Armor:         attr.Armor,
		Weapon:        attr.Weapon,
		MovementSpeed: attr.MovementSpeed,
		AttackRange:   attr.AttackRange,
		CurrentHealth: attr.CurrentHealth,
		MaxHealth:     attr.MaxHealth,
		CanAct:        attr.CanAct,
	}
}

func savedToAttributes(sa savedAttributes) common.Attributes {
	return common.Attributes{
		Strength:      sa.Strength,
		Dexterity:     sa.Dexterity,
		Magic:         sa.Magic,
		Leadership:    sa.Leadership,
		Armor:         sa.Armor,
		Weapon:        sa.Weapon,
		MovementSpeed: sa.MovementSpeed,
		AttackRange:   sa.AttackRange,
		CurrentHealth: sa.CurrentHealth,
		MaxHealth:     sa.MaxHealth,
		CanAct:        sa.CanAct,
	}
}

func positionToSaved(pos *coords.LogicalPosition) savedPosition {
	return savedPosition{X: pos.X, Y: pos.Y}
}

func savedToPosition(sp savedPosition) coords.LogicalPosition {
	return coords.LogicalPosition{X: sp.X, Y: sp.Y}
}

// --- Slice copy helpers ---
// Used across chunks to safely copy slices during save/load.

func copyEntityIDs(ids []ecs.EntityID) []ecs.EntityID {
	if ids == nil {
		return nil
	}
	result := make([]ecs.EntityID, len(ids))
	copy(result, ids)
	return result
}

func copyInts(ints []int) []int {
	if ints == nil {
		return nil
	}
	result := make([]int, len(ints))
	copy(result, ints)
	return result
}
