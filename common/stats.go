package common

import "fmt"

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
	CanMove            bool
	Strength           int
	Dexterity          int
	Constitution       int
}

func NewBaseAttributes(maxHealth, attackBonus, baseAC, baseProt, baseMovSpeed int, dodge float32, strength, dexterity, constitution int) Attributes {
	return Attributes{
		MaxHealth:         maxHealth,
		CurrentHealth:     maxHealth,
		AttackBonus:       attackBonus,
		BaseArmorClass:    baseAC,
		BaseProtection:    baseProt,
		BaseDodgeChance:   dodge,
		BaseMovementSpeed: baseMovSpeed,
		Strength:          strength,
		Dexterity:         dexterity,
		Constitution:      constitution,
	}
}

// For Displaying to the player
func (a Attributes) DisplayString() string {

	res := ""
	res += fmt.Sprintln("HP ", a.CurrentHealth, "/", a.MaxHealth)
	res += fmt.Sprintln("AC", a.TotalArmorClass)
	res += fmt.Sprintln("Prot", a.TotalProtection)
	res += fmt.Sprintln("Dodge", a.TotalDodgeChance)
	res += fmt.Sprintln("Move Speed", a.TotalMovementSpeed)
	res += fmt.Sprintln("Attack Speed", a.TotalAttackSpeed)

	return res

}
