package entitytemplates

import (
	"encoding/json"

	"os"
)

type Monsters struct {
	Monsters []JSONMonster `json:"monsters"`
}

func ReadMonsterData() {
	data, err := os.ReadFile("../assets//gamedata/monsterdata.json")
	if err != nil {
		panic(err)
	}

	// Parse JSON
	var monsters Monsters
	err = json.Unmarshal(data, &monsters)
	if err != nil {
		panic(err)
	}

	// Iterate over monsters
	for _, monster := range monsters.Monsters {

		MonsterTemplates = append(MonsterTemplates, NewJSONMonster(monster.Name, monster.ImageName, monster.Attributes, monster.Armor, monster.MeleeWeapon))

	}

}
