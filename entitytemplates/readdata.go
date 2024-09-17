package entitytemplates

import (
	"encoding/json"
	"fmt"

	"os"
)

type MonstersData struct {
	Monsters []JSONMonster `json:"monsters"`
}

// WeaponList struct to hold all weapons
type WeaponData struct {
	Weps []JSONWeapon `json:"weapons"` // List of weapons
}

type MeleeWeapons struct {
	Weapons []JSONMeleeWeapon
}

type RangedWeapons struct {
	Weapons []JSONRangedWeapon
}

type ConsumableData struct {
	Consumables []JSONAttributeModifier
}

func ReadMonsterData() {
	data, err := os.ReadFile("../assets//gamedata/monsterdata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var monsters MonstersData
	err = json.Unmarshal(data, &monsters)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, monster := range monsters.Monsters {

		if monster.RangedWeapon != nil {
			CreateTargetArea(monster.RangedWeapon.TargetArea)
		}
		MonsterTemplates = append(MonsterTemplates, NewJSONMonster(monster))

	}

}

func ReadWeaponData() {
	data, err := os.ReadFile("../assets//gamedata/weapondata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var weaponData WeaponData
	err = json.Unmarshal(data, &weaponData)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, w := range weaponData.Weps {

		fmt.Println(w)

		if w.Type == "MeleeWeapon" {
			wep := NewJSONMeleeWeapon(w)
			MeleeWeaponTemplates = append(MeleeWeaponTemplates, wep)

		} else if w.Type == "RangedWeapon" {

			wep := NewJSONRangedWeapon(w)
			RangedWeaponTemplates = append(RangedWeaponTemplates, wep)

		} else {
			fmt.Println("Error in JSON weapon file")
		}
	}

}

func ReadConsumableData() {
	data, err := os.ReadFile("../assets//gamedata/consumabledata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var consumables ConsumableData
	err = json.Unmarshal(data, &consumables)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, c := range consumables.Consumables {
		fmt.Println(c)

		ConsumableTemplates = append(ConsumableTemplates, NewJSONAttributeModifier(c))

	}

}
