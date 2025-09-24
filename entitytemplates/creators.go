// Package entitytemplates provides template-based entity creation and data loading systems.
// It handles JSON-based entity definitions, component composition patterns,
// and factory functions for creating entities from reusable templates.
package entitytemplates

import (
	"game_main/common"
	"game_main/coords"
	"game_main/gear"
	"game_main/monsters"
	"game_main/rendering"
	"game_main/worldmap"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// createBaseEntity creates a basic entity with common components (name, renderable, position).
// It loads the image from the specified path and sets up the fundamental entity structure.
func createBaseEntity(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *common.Position) *ecs.Entity {
	fpath := filepath.Join(assetDir, imagePath)
	img, _, err := ebitenutil.NewImageFromFile(fpath)
	if err != nil {
		log.Fatal(err)
	}

	entity := manager.World.NewEntity()
	entity.AddComponent(common.NameComponent, &common.Name{NameStr: name})
	entity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
		Image: img, Visible: visible,
	})

	if pos == nil {
		pos = &common.Position{X: 0, Y: 0}
	}
	entity.AddComponent(common.PositionComponent, pos)

	return entity
}

// ComponentAdder is a function type that adds specific components to an entity.
// Used in the template system to compose entities with different component combinations.
type ComponentAdder func(entity *ecs.Entity)

// createFromTemplate creates an entity using a base template and applies additional components.
// It uses the ComponentAdder pattern to compose entities with varying component sets.
func createFromTemplate(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *common.Position, adders ...ComponentAdder) *ecs.Entity {
	entity := createBaseEntity(manager, name, imagePath, assetDir, visible, pos)

	for _, adder := range adders {
		adder(entity)
	}

	return entity
}

func addMeleeWeaponComponents(w JSONMeleeWeapon) ComponentAdder {
	return func(entity *ecs.Entity) {
		entity.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
		entity.AddComponent(gear.MeleeWeaponComponent, &gear.MeleeWeapon{
			MinDamage:   w.MinDamage,
			MaxDamage:   w.MaxDamage,
			AttackSpeed: w.AttackSpeed,
		})
	}
}

func addRangedWeaponComponents(w JSONRangedWeapon) ComponentAdder {
	return func(entity *ecs.Entity) {
		entity.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})

		ranged := gear.RangedWeapon{
			MinDamage:     w.MinDamage,
			MaxDamage:     w.MaxDamage,
			ShootingRange: w.ShootingRange,
			AttackSpeed:   w.AttackSpeed,
		}
		ranged.TargetArea = CreateTargetArea(w.TargetArea)

		entity.AddComponent(gear.RangedWeaponComponent, &ranged)
	}
}

func addConsumableComponents(c JSONAttributeModifier) ComponentAdder {
	return func(entity *ecs.Entity) {
		entity.AddComponent(gear.ItemComponent, &gear.Item{Count: 1})
		entity.AddComponent(gear.ConsumableComponent, &gear.Consumable{
			Name:         c.Name,
			AttrModifier: CreateAttributesFromJSON(c),
			Duration:     c.Duration,
		})
	}
}

func addCreatureComponents(m JSONMonster) ComponentAdder {
	return func(entity *ecs.Entity) {
		entity.AddComponent(monsters.CreatureComponent, &monsters.Creature{Path: make([]common.Position, 0)})
		entity.AddComponent(common.UserMsgComponent, &common.UserMessage{})

		attr := common.Attributes{
			MaxHealth:          m.Attributes.MaxHealth,
			CurrentHealth:      m.Attributes.MaxHealth,
			AttackBonus:        m.Attributes.AttackBonus,
			BaseArmorClass:     m.Attributes.BaseArmorClass,
			BaseProtection:     m.Attributes.BaseProtection,
			BaseDodgeChance:    m.Attributes.BaseDodgeChance,
			BaseMovementSpeed:  m.Attributes.BaseMovementSpeed,
			TotalMovementSpeed: m.Attributes.BaseMovementSpeed,
			TotalAttackSpeed:   1,
			CanAct:             true,
		}

		if m.Armor != nil {
			armor := gear.Armor{
				ArmorClass:  m.Armor.ArmorClass,
				Protection:  m.Armor.Protection,
				DodgeChance: m.Armor.DodgeChance,
			}
			entity.AddComponent(gear.ArmorComponent, &armor)
		}

		if m.MeleeWeapon != nil {
			weapon := gear.MeleeWeapon{
				MinDamage:   m.MeleeWeapon.MinDamage,
				MaxDamage:   m.MeleeWeapon.MaxDamage,
				AttackSpeed: m.MeleeWeapon.AttackSpeed,
			}
			attr.TotalAttackSpeed = weapon.AttackSpeed
			entity.AddComponent(gear.MeleeWeaponComponent, &weapon)
		}

		if m.RangedWeapon != nil {
			weapon := gear.RangedWeapon{
				MinDamage:     m.RangedWeapon.MinDamage,
				MaxDamage:     m.RangedWeapon.MaxDamage,
				ShootingRange: m.RangedWeapon.ShootingRange,
			}
			attr.TotalAttackSpeed = weapon.AttackSpeed
			entity.AddComponent(gear.RangedWeaponComponent, &weapon)
		}

		if attr.TotalAttackSpeed <= 0 {
			attr.TotalAttackSpeed = 1
		}

		entity.AddComponent(common.AttributeComponent, &attr)
	}
}

func CreateMeleeWepFromTemplate(manager common.EntityManager, w JSONMeleeWeapon) *ecs.Entity {
	return createFromTemplate(manager, w.Name, w.ImgName, "../assets/items/", false, nil,
		addMeleeWeaponComponents(w))
}

func CreateRangedWepFromTemplate(manager common.EntityManager, w JSONRangedWeapon) *ecs.Entity {
	return createFromTemplate(manager, w.Name, w.ImgName, "../assets/items/", false, nil,
		addRangedWeaponComponents(w))
}

func CreateConsumableFromTemplate(manager common.EntityManager, c JSONAttributeModifier) *ecs.Entity {
	return createFromTemplate(manager, c.Name, c.ImgName, "../assets/items/", false, nil,
		addConsumableComponents(c))
}

func CreateCreatureFromTemplate(manager common.EntityManager, m JSONMonster, gm *worldmap.GameMap, xPos, yPos int) *ecs.Entity {
	entity := createFromTemplate(manager, m.Name, m.ImageName, "../assets/creatures/", true,
		&common.Position{X: xPos, Y: yPos}, addCreatureComponents(m))

	logicalPos := coords.LogicalPosition{X: xPos, Y: yPos}
	ind := coords.CoordManager.LogicalToIndex(logicalPos)
	gm.Tiles[ind].Blocked = true

	return entity
}
