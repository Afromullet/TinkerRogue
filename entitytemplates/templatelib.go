package entitytemplates

//We are not creating the entities yet, so we use the JSON struct to store the template data.
var MonsterTemplates []JSONMonster
var MeleeWeaponTemplates []JSONMeleeWeapon
var RangedWeaponTemplates []JSONRangedWeapon
var ConsumableTemplates []JSONAttributeModifier

func ReadGameData() {

	ReadMonsterData()
	ReadWeaponData()
	ReadConsumableData()

}
