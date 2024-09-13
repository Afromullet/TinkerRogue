package entitytemplates

import (
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"
	"game_main/monsters"
	"game_main/timesystem"
	"game_main/worldmap"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func CreateCreatureFromTemplate(manager *ecs.Manager, m JSONMonster, gm *worldmap.GameMap, xPos, yPos int) *ecs.Entity {

	fpath := filepath.Join("../assets/creatures/", m.ImageName)

	creatureImg, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	ent := manager.NewEntity()

	ent.AddComponent(common.NameComponent, &common.Name{NameStr: m.Name})

	ent.AddComponent(common.RenderableComponent, &common.Renderable{
		Image:   creatureImg,
		Visible: true,
	})

	ent.AddComponent(monsters.CreatureComponent, &monsters.Creature{Path: make([]common.Position, 0)})

	ent.AddComponent(common.PositionComponent, &common.Position{X: xPos, Y: yPos})

	ind := graphics.IndexFromXY(xPos, yPos)
	gm.Tiles[ind].Blocked = true

	ent.AddComponent(common.AttributeComponent,
		&common.Attributes{
			MaxHealth:          m.Attributes.MaxHealth,
			CurrentHealth:      m.Attributes.MaxHealth,
			AttackBonus:        m.Attributes.AttackBonus,
			BaseArmorClass:     m.Attributes.BaseArmorClass,
			BaseProteciton:     m.Attributes.BaseProtection,
			BaseDodgeChange:    m.Attributes.BaseDodgeChance,
			BaseMovementSpeed:  m.Attributes.BaseMovementSpeed,
			TotalMovementSpeed: m.Attributes.BaseMovementSpeed,
			TotalAttackSpeed:   3})

	if m.Armor != nil {

		armor := equipment.Armor{
			ArmorClass:  m.Armor.ArmorClass,
			Protection:  m.Armor.Protection,
			DodgeChance: m.Armor.DodgeChance,
		}
		ent.AddComponent(equipment.ArmorComponent, &armor)
	}

	if m.MeleeWeapon != nil {

		weapon := equipment.MeleeWeapon{
			MinDamage:   m.MeleeWeapon.MinDamage,
			MaxDamage:   m.MeleeWeapon.MaxDamage,
			AttackSpeed: m.MeleeWeapon.AttackSpeed,
		}

		ent.AddComponent(equipment.WeaponComponent, &weapon)

	}

	ent.AddComponent(timesystem.ActionQueueComponent, &timesystem.ActionQueue{TotalActionPoints: 100})

	return ent

}
