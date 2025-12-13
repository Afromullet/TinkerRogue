package entitytemplates

import (
	"encoding/json"

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

type CreatureModifiers struct {
	CreatureMods []JSONCreatureModifier
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

		if w.Type == "MeleeWeapon" {
			wep := NewJSONMeleeWeapon(w)
			MeleeWeaponTemplates = append(MeleeWeaponTemplates, wep)

		} else if w.Type == "RangedWeapon" {

			wep := NewJSONRangedWeapon(w)
			RangedWeaponTemplates = append(RangedWeaponTemplates, wep)

		} else {
			// ERROR HANDLING IN FUTURE
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

		ConsumableTemplates = append(ConsumableTemplates, NewJSONAttributeModifier(c))

	}

}

func ReadCreatureModifiers() {
	data, err := os.ReadFile("../assets//gamedata/creaturemodifiers.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var mod CreatureModifiers
	err = json.Unmarshal(data, &mod)

	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, c := range mod.CreatureMods {

		CreatureModifierTemplates = append(CreatureModifierTemplates, CreatureModifierFromJSON(c))

	}

}
