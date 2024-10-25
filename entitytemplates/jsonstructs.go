package entitytemplates

import (
	"game_main/common"
	"game_main/graphics"
)

// All structs for unmarshalling JSON data

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

type JSONArmor struct {
	ArmorClass  int     `json:"armorClass"`
	Protection  int     `json:"protection"`
	DodgeChance float32 `json:"dodgeChance"`
}

type JSONMeleeWeapon struct {
	Name        string `json:"name,omitempty"`
	ImgName     string `json:"imgname,omitempty"`
	MinDamage   int    `json:"minDamage"`
	MaxDamage   int    `json:"maxDamage"`
	AttackSpeed int    `json:"attackSpeed"`
}

func NewJSONMeleeWeapon(w JSONWeapon) JSONMeleeWeapon {
	return JSONMeleeWeapon{
		ImgName:     w.ImgName,
		Name:        w.Name,
		MinDamage:   w.MinDamage,
		MaxDamage:   w.MaxDamage,
		AttackSpeed: w.AttackSpeed,
	}
}

// Different TileShapes require different parameters
// The JSONTargetArea struct contains optional fields for all of the options
type JSONTargetArea struct {
	Type   string `json:"type"`
	Size   int    `json:"size,omitempty"`
	Length int    `json:"length,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Radius int    `json:"radius,omitempty"`
}

// For creating the TileBasedShape from JSON data
func CreateTargetArea(area *JSONTargetArea) graphics.TileBasedShape {

	var s graphics.TileBasedShape

	//Default to a 1x1 square if the area is nill
	if area == nil {
		s = graphics.NewTileSquare(0, 0, 1)
	} else if area.Type == "Rectangle" {

		s = graphics.NewTileRectangle(0, 0, area.Width, area.Height)

	} else if area.Type == "Cone" {

		s = graphics.NewTileCone(0, 0, area.Length, graphics.LineDown)

	} else if area.Type == "Square" {

		s = graphics.NewTileSquare(0, 0, area.Size)

	} else if area.Type == "Line" {

		s = graphics.NewTileLine(0, 0, area.Length, graphics.LineDown)

	} else if area.Type == "Circle" {

		s = graphics.NewTileCircle(0, 0, area.Radius)

	}

	return s

}

type JSONRangedWeapon struct {
	Name            string          `json:"name,omitempty"`
	ShootingVXXName string          `json:"shootingVX"`
	ImgName         string          `json:"imgname,omitempty"`
	MinDamage       int             `json:"minDamage"`
	MaxDamage       int             `json:"maxDamage"`
	ShootingRange   int             `json:"shootingRange"`
	AttackSpeed     int             `json:"attackSpeed"`
	TargetArea      *JSONTargetArea `json:"targetArea"`
}

func NewJSONRangedWeapon(r JSONWeapon) JSONRangedWeapon {

	return JSONRangedWeapon{

		Name:            r.Name,
		ShootingVXXName: r.ShootingVX,
		ImgName:         r.ImgName,
		MinDamage:       r.MinDamage,
		MaxDamage:       r.MaxDamage,
		ShootingRange:   r.ShootingRange,
		AttackSpeed:     r.AttackSpeed,
		TargetArea:      r.TargetArea,
	}

}

type JSONMonster struct {
	Name         string            `json:"name"`
	ImageName    string            `json:"imgname"`
	Attributes   JSONAttributes    `json:"attributes"`
	Armor        *JSONArmor        `json:"armor"`       // Use pointer to allow null values
	MeleeWeapon  *JSONMeleeWeapon  `json:"meleeWeapon"` // Use pointer to allow null values
	RangedWeapon *JSONRangedWeapon `json:"rangedWeapon"`
}

func NewJSONMonster(m JSONMonster) JSONMonster {
	return JSONMonster{

		Name:         m.Name,
		ImageName:    m.ImageName,
		Attributes:   m.Attributes,
		Armor:        m.Armor,
		MeleeWeapon:  m.MeleeWeapon,
		RangedWeapon: m.RangedWeapon,
	}
}

// Intermediate struct for reading data from weapondata.json
// The json file contains both melee and ranged weapons, which
// Is why we have optional fields.
type JSONWeapon struct {
	Type          string          `json:"type"` // Can be "MeleeWeapon" or "RangedWeapon"
	Name          string          `json:"name"` // Weapon name
	ImgName       string          `json:"imgname"`
	MinDamage     int             `json:"minDamage"`               // Minimum damage
	MaxDamage     int             `json:"maxDamage"`               // Maximum damage
	AttackSpeed   int             `json:"attackSpeed"`             // Attack speed
	ShootingRange int             `json:"shootingRange,omitempty"` // For ranged weapons only
	AmmoType      string          `json:"ammoType,omitempty"`      // For ranged weapons only
	ShootingVX    string          `json:"shootingvx,omitempty"`
	TargetArea    *JSONTargetArea `json:"targetArea"`
}

type JSONAttributeModifier struct {
	Name              string  `json:"name"`
	ImgName           string  `json:"imgname"`
	AttackBonus       int     `json:"attackBonus"`
	MaxHealth         int     `json:"maxHealth"`
	CurrentHealth     int     `json:"currentHealth"`
	BaseArmorClass    int     `json:"baseArmorClass"`
	BaseProtection    int     `json:"baseProtection"`
	BaseMovementSpeed int     `json:"baseMovementSpeed"`
	BaseDodgeChance   float32 `json:"baseDodgeChance"`
	Duration          int     `json:"duration"`
}

func NewJSONAttributeModifier(a JSONAttributeModifier) JSONAttributeModifier {
	return JSONAttributeModifier{
		Name:              a.Name,
		ImgName:           a.ImgName,
		MaxHealth:         a.MaxHealth,
		CurrentHealth:     a.CurrentHealth,
		BaseArmorClass:    a.BaseArmorClass,
		BaseProtection:    a.BaseProtection,
		BaseMovementSpeed: a.BaseMovementSpeed,
		BaseDodgeChance:   a.BaseDodgeChance,
		Duration:          a.Duration,
	}
}

func CreateAttributesFromJSON(a JSONAttributeModifier) common.Attributes {
	return common.Attributes{
		MaxHealth:         a.MaxHealth,
		CurrentHealth:     a.CurrentHealth,
		AttackBonus:       a.AttackBonus,
		BaseArmorClass:    a.BaseArmorClass,
		BaseProtection:    a.BaseProtection,
		BaseDodgeChance:   a.BaseDodgeChance,
		BaseMovementSpeed: a.BaseMovementSpeed,
	}
}

type JSONCreatureModifier struct {
	Name              string  `json:"name"`
	AttackBonus       int     `json:"attackBonus"`
	MaxHealth         int     `json:"maxHealth"`
	CurrentHealth     int     `json:"currentHealth"`
	BaseArmorClass    int     `json:"baseArmorClass"`
	BaseProtection    int     `json:"baseProtection"`
	BaseMovementSpeed int     `json:"baseMovementSpeed"`
	BaseDodgeChance   float32 `json:"baseDodgeChance"`
	DamageBonus       int     `json:"damagebonus"`
}

func CreatureModifierFromJSON(a JSONCreatureModifier) JSONCreatureModifier {
	return JSONCreatureModifier{
		Name:              a.Name,
		MaxHealth:         a.MaxHealth,
		CurrentHealth:     a.CurrentHealth,
		AttackBonus:       a.AttackBonus,
		BaseArmorClass:    a.BaseArmorClass,
		BaseProtection:    a.BaseProtection,
		BaseDodgeChance:   a.BaseDodgeChance,
		BaseMovementSpeed: a.BaseMovementSpeed,
		DamageBonus:       a.DamageBonus,
	}
}
