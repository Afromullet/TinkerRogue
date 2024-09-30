package entitytemplates

import (
	"game_main/common"
	"game_main/gear"
	"game_main/graphics"
	"game_main/monsters"
	"game_main/rendering"
	"game_main/timesystem"
	"game_main/worldmap"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Creates a creature entity from data read from the JSON files
// All of the creatures read from the JSON file are stored in MonsterTemplates
func CreateCreatureFromTemplate(manager common.EntityManager, m JSONMonster, gm *worldmap.GameMap, xPos, yPos int) *ecs.Entity {

	fpath := filepath.Join("../assets/creatures/", m.ImageName)

	creatureImg, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	ent := manager.World.NewEntity()

	ent.AddComponent(common.NameComponent, &common.Name{NameStr: m.Name})

	ent.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image:   creatureImg,
		Visible: true,
	})

	ent.AddComponent(monsters.CreatureComponent, &monsters.Creature{Path: make([]common.Position, 0)})

	ent.AddComponent(common.PositionComponent, &common.Position{X: xPos, Y: yPos})

	ind := graphics.IndexFromXY(xPos, yPos)
	gm.Tiles[ind].Blocked = true

	attr := common.Attributes{
		MaxHealth:          m.Attributes.MaxHealth,
		CurrentHealth:      m.Attributes.MaxHealth,
		AttackBonus:        m.Attributes.AttackBonus,
		BaseArmorClass:     m.Attributes.BaseArmorClass,
		BaseProtection:     m.Attributes.BaseProtection,
		BaseDodgeChance:    m.Attributes.BaseDodgeChance,
		BaseMovementSpeed:  m.Attributes.BaseMovementSpeed,
		TotalMovementSpeed: m.Attributes.BaseMovementSpeed,
		TotalAttackSpeed:   1}

	if m.Armor != nil {

		armor := gear.Armor{
			ArmorClass:  m.Armor.ArmorClass,
			Protection:  m.Armor.Protection,
			DodgeChance: m.Armor.DodgeChance,
		}
		ent.AddComponent(gear.ArmorComponent, &armor)
	}

	if m.MeleeWeapon != nil {

		weapon := gear.MeleeWeapon{
			MinDamage:   m.MeleeWeapon.MinDamage,
			MaxDamage:   m.MeleeWeapon.MaxDamage,
			AttackSpeed: m.MeleeWeapon.AttackSpeed,
		}

		attr.TotalAttackSpeed = weapon.AttackSpeed

		ent.AddComponent(gear.MeleeWeaponComponent, &weapon)

	}

	if m.RangedWeapon != nil {

		weapon := gear.RangedWeapon{
			MinDamage:     m.RangedWeapon.MinDamage,
			MaxDamage:     m.RangedWeapon.MaxDamage,
			ShootingRange: m.RangedWeapon.ShootingRange,
		}

		attr.TotalAttackSpeed = weapon.AttackSpeed

		ent.AddComponent(gear.RangedWeaponComponent, &weapon)

	}

	if attr.TotalAttackSpeed <= 0 {
		attr.TotalAttackSpeed = 1
	}

	ent.AddComponent(common.UserMsgComponent, &common.UserMessage{})

	ent.AddComponent(common.AttributeComponent, &attr)
	ent.AddComponent(timesystem.ActionQueueComponent, &timesystem.ActionQueue{TotalActionPoints: 100})

	return ent

}

// Creates a melee weapon entity from data read from the JSON files
// All of the melee weapons read from the JSON file are stored in MeleeWeaponTemplates
func CreateMeleeWepFromTemplate(manager common.EntityManager, w JSONMeleeWeapon) *ecs.Entity {

	fpath := filepath.Join("../assets/items/", w.ImgName)

	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	it := manager.World.NewEntity()

	it.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image:   img,
		Visible: false,
	})

	it.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
	it.AddComponent(common.NameComponent, &common.Name{
		NameStr: w.Name,
	})

	it.AddComponent(common.PositionComponent, &common.Position{
		X: 0,
		Y: 0,
	})

	it.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{
		MinDamage:   w.MinDamage,
		MaxDamage:   w.MaxDamage,
		AttackSpeed: w.AttackSpeed,
	})

	return it

}

// Todo add shooting VX
func CreateRangedWepFromTemplate(manager common.EntityManager, w JSONRangedWeapon) *ecs.Entity {

	fpath := filepath.Join("../assets/items/", w.ImgName)

	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	it := manager.World.NewEntity()

	it.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image:   img,
		Visible: false,
	})

	it.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
	it.AddComponent(common.NameComponent, &common.Name{
		NameStr: w.Name,
	})

	it.AddComponent(common.PositionComponent, &common.Position{
		X: 0,
		Y: 0,
	})

	ranged := gear.RangedWeapon{
		MinDamage:     w.MinDamage,
		MaxDamage:     w.MaxDamage,
		ShootingRange: w.ShootingRange,
		AttackSpeed:   w.AttackSpeed}

	ranged.TargetArea = CreateTargetArea(w.TargetArea)

	it.AddComponent(gear.RangedWeaponComponent, &ranged)

	return it

}

// Todo add shooting VX
func CreateConsumableFromTemplate(manager common.EntityManager, c JSONAttributeModifier) *ecs.Entity {

	fpath := filepath.Join("../assets/items/", c.ImgName)

	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	it := manager.World.NewEntity()

	it.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image:   img,
		Visible: false,
	})

	it.AddComponent(common.PositionComponent, &common.Position{
		X: 0,
		Y: 0,
	})

	it.AddComponent(common.NameComponent, &common.Name{
		NameStr: c.Name,
	})

	it.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})

	it.AddComponent(gear.ConsumableComponent, &gear.Consumable{
		Name:         c.Name,
		AttrModifier: CreateAttributesFromJSON(c),
		Duration:     c.Duration,
	})

	/*

		ranged := gear.Consumable{
			Name: c.Name
			AttrModifier: common.NewBaseAttributes(c.),


		ranged.TargetArea = CreateTargetArea(w.TargetArea)

		it.AddComponent(gear.RangedWeaponComponent, &ranged)
	*/
	return it

}
