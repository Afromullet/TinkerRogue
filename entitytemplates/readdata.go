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
	Weps []Weapon `json:"weapons"` // List of weapons
}

type MeleeWeapons struct {
	Weapons []JSONMeleeWeapon
}

type RangedWeapons struct {
	Weapons []JSONRangedWeapon
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
		MonsterTemplates = append(MonsterTemplates, NewJSONMonster(monster.Name, monster.ImageName, monster.Attributes, monster.Armor, monster.MeleeWeapon, monster.RangedWeapon))

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
			wep := NewJSONMeleeWeapon(w.MinDamage, w.MaxDamage, w.AttackSpeed, w.Name, w.ImgName)
			MeleeWeaponTemplates = append(MeleeWeaponTemplates, wep)

		} else if w.Type == "RangedWeapon" {

			wep := NewJSONRangedWeapon(w.Name, w.ShootingVX, w.ImgName, w.MinDamage, w.MaxDamage, w.ShootingRange, w.AttackSpeed, w.TargetArea)
			RangedWeaponTemplates = append(RangedWeaponTemplates, wep)

		} else {
			fmt.Println("Error in JSON weapon file")
		}
	}

}
