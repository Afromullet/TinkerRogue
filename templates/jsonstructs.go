package templates

import (
	"game_main/common"
	"game_main/visual/graphics"
)

// All structs for unmarshalling JSON data

type JSONAttributes struct {
	Strength   int `json:"strength"`
	Dexterity  int `json:"dexterity"`
	Magic      int `json:"magic"`
	Leadership int `json:"leadership"`
	Armor      int `json:"armor"`
	Weapon     int `json:"weapon"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {
	return common.NewAttributes(
		attr.Strength,
		attr.Dexterity,
		attr.Magic,
		attr.Leadership,
		attr.Armor,
		attr.Weapon,
	)
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

	//Default to a 1x1 square if the area is nil
	if area == nil {
		s = graphics.NewSquare(0, 0, common.NormalQuality)
	} else if area.Type == "Rectangle" {

		s = graphics.NewRectangle(0, 0, common.NormalQuality)

	} else if area.Type == "Cone" {

		s = graphics.NewCone(0, 0, graphics.LineDown, common.NormalQuality)

	} else if area.Type == "Square" {

		s = graphics.NewSquare(0, 0, common.NormalQuality)

	} else if area.Type == "Line" {

		s = graphics.NewLine(0, 0, graphics.LineDown, common.NormalQuality)

	} else if area.Type == "Circle" {

		s = graphics.NewCircle(0, 0, common.NormalQuality)

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
	Name       string         `json:"name"`
	ImageName  string         `json:"imgname"`
	Attributes JSONAttributes `json:"attributes"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Role       string         `json:"role"`

	// Targeting fields
	AttackType  string   `json:"attackType"`  // "MeleeRow", "MeleeColumn", "Ranged", or "Magic"
	TargetCells [][2]int `json:"targetCells"` // For magic: pattern cells

	CoverValue     float64 `json:"coverValue"`     // Damage reduction provided (0.0-1.0)
	CoverRange     int     `json:"coverRange"`     // Rows behind that receive cover (1-3)
	RequiresActive bool    `json:"requiresActive"` // If true, dead/stunned units don't provide cover
	AttackRange    int     `json:"attackRange"`    // World-based attack range (Melee=1, Ranged=3, Magic=4)
	MovementSpeed  int     `json:"movementSpeed"`  // Movement speed on world map (1 tile per speed point)
}

func NewJSONMonster(m JSONMonster) JSONMonster {
	return JSONMonster{
		Name:       m.Name,
		ImageName:  m.ImageName,
		Attributes: m.Attributes,
		Width:      m.Width,
		Height:     m.Height,
		Role:       m.Role,

		AttackType:     m.AttackType,
		TargetCells:    m.TargetCells,
		CoverValue:     m.CoverValue,
		CoverRange:     m.CoverRange,
		RequiresActive: m.RequiresActive,
		AttackRange:    m.AttackRange,
		MovementSpeed:  m.MovementSpeed,
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
	Name       string `json:"name"`
	ImgName    string `json:"imgname"`
	Strength   int    `json:"strength"`
	Dexterity  int    `json:"dexterity"`
	Magic      int    `json:"magic"`
	Leadership int    `json:"leadership"`
	Armor      int    `json:"armor"`
	Weapon     int    `json:"weapon"`
	Duration   int    `json:"duration"`
}

func NewJSONAttributeModifier(a JSONAttributeModifier) JSONAttributeModifier {
	return JSONAttributeModifier{
		Name:       a.Name,
		ImgName:    a.ImgName,
		Strength:   a.Strength,
		Dexterity:  a.Dexterity,
		Magic:      a.Magic,
		Leadership: a.Leadership,
		Armor:      a.Armor,
		Weapon:     a.Weapon,
		Duration:   a.Duration,
	}
}

func CreateAttributesFromJSON(a JSONAttributeModifier) common.Attributes {
	// For consumables, create an attributes struct with modifiers only
	// Don't use NewAttributes since we don't want to initialize health
	return common.Attributes{
		Strength:   a.Strength,
		Dexterity:  a.Dexterity,
		Magic:      a.Magic,
		Leadership: a.Leadership,
		Armor:      a.Armor,
		Weapon:     a.Weapon,
		// Health fields left at zero - consumables will modify them separately
	}
}

type JSONCreatureModifier struct {
	Name              string   `json:"name"`
	AttackBonus       int      `json:"attackBonus"`
	MaxHealth         int      `json:"maxHealth"`
	CurrentHealth     int      `json:"currentHealth"`
	BaseArmorClass    int      `json:"baseArmorClass"`
	BaseProtection    int      `json:"baseProtection"`
	BaseMovementSpeed int      `json:"baseMovementSpeed"`
	BaseDodgeChance   float32  `json:"baseDodgeChance"`
	DamageBonus       int      `json:"damagebonus"`
	Width             int      `json:"width"`
	Height            int      `json:"height"`
	Role              string   `json:"role"`
	TargetCells       [][2]int `json:"targetCells"` // Cell-based targeting pattern
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
		Width:             a.Width,
		Height:            a.Height,
		Role:              a.Role,
		TargetCells:       a.TargetCells,
	}
}
