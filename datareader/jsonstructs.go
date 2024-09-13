package entitytemplates

import "game_main/common"

type JSONAttributes struct {
	MaxHealth         int     `json:"MaxHealth"`
	AttackBonus       int     `json:"AttackBonus"`
	BaseArmorClass    int     `json:"BaseArmorClass"`
	BaseProtection    int     `json:"BaseProtection"`
	BaseDodgeChance   float32 `json:"BaseDodgeChance"`
	BaseMovementSpeed int     `json:"BaseMovementSpeed"`
}

func (attr JSONAttributes) NewAttributesFromJson() common.Attributes {

	return common.NewBaseAttributes(
		attr.MaxHealth,
		attr.AttackBonus,
		attr.BaseArmorClass,
		attr.BaseProtection,
		attr.BaseMovementSpeed,
		attr.BaseDodgeChance)

}

type JSONArmor struct {
	ArmorClass  int     `json:"ArmorClass"`
	Protection  int     `json:"Protection"`
	DodgeChance float32 `json:"DodgeChance"`
}

type JSONMeleeWeapon struct {
	MinDamage   int `json:"MinDamage"`
	MaxDamage   int `json:"MaxDamage"`
	AttackSpeed int `json:"AttackSpeed"`
}

type JSONMonster struct {
	Name        string           `json:"name"`
	ImageName   string           `json:"imgname"`
	Attributes  JSONAttributes   `json:"attributes"`
	Armor       *JSONArmor       `json:"armor"`       // Use pointer to allow null values
	MeleeWeapon *JSONMeleeWeapon `json:"meleeWeapon"` // Use pointer to allow null values

}

func NewJSONMonster(name string, imgname string, attr JSONAttributes, armor *JSONArmor, meleeWeapon *JSONMeleeWeapon) JSONMonster {
	return JSONMonster{
		Name:        name,
		ImageName:   imgname,
		Attributes:  attr,
		Armor:       armor,
		MeleeWeapon: meleeWeapon,
	}
}
