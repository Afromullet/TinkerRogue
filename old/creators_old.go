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
func createBaseEntity(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *coords.LogicalPosition) *ecs.Entity {
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
		pos = &coords.LogicalPosition{X: 0, Y: 0}
	}
	entity.AddComponent(common.PositionComponent, pos)

	return entity
}

// ComponentAdder is a function type that adds specific components to an entity.
// Used in the template system to compose entities with different component combinations.
type ComponentAdder func(entity *ecs.Entity)

// createFromTemplate creates an entity using a base template and applies additional components.
// It uses the ComponentAdder pattern to compose entities with varying component sets.
func createFromTemplate(manager common.EntityManager, name, imagePath, assetDir string, visible bool, pos *coords.LogicalPosition, adders ...ComponentAdder) *ecs.Entity {
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
		entity.AddComponent(monsters.CreatureComponent, &monsters.Creature{Path: make([]coords.LogicalPosition, 0)})
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

// EntityType represents the category of entity being created.
// Used in the unified factory system to differentiate entity construction logic.
type EntityType int

const (
	EntityMeleeWeapon EntityType = iota
	EntityRangedWeapon
	EntityConsumable
	EntityCreature
)

// EntityConfig holds all configuration data needed to create an entity.
// It provides a unified interface for entity construction across different types.
type EntityConfig struct {
	Type      EntityType
	Name      string
	ImagePath string
	AssetDir  string
	Visible   bool
	Position  *coords.LogicalPosition

	// Optional creature-specific fields
	GameMap *worldmap.GameMap // Only used for creatures to block map tiles
}

// CreateEntityFromTemplate is a unified factory function that creates entities of any type.
// It uses the EntityType enum and type assertions to handle entity-specific construction logic.
//
// Parameters:
//   - manager: ECS entity manager
//   - config: Entity configuration (type, paths, position, etc.)
//   - data: Entity-specific data (JSONMeleeWeapon, JSONRangedWeapon, JSONAttributeModifier, or JSONMonster)
//
// Returns the created entity with all appropriate components attached.
func CreateEntityFromTemplate(manager common.EntityManager, config EntityConfig, data any) *ecs.Entity {
	var adders []ComponentAdder

	switch config.Type {
	case EntityMeleeWeapon:
		w, ok := data.(JSONMeleeWeapon)
		if !ok {
			log.Fatalf("Expected JSONMeleeWeapon for EntityMeleeWeapon, got %T", data)
		}
		adders = []ComponentAdder{addMeleeWeaponComponents(w)}

	case EntityRangedWeapon:
		w, ok := data.(JSONRangedWeapon)
		if !ok {
			log.Fatalf("Expected JSONRangedWeapon for EntityRangedWeapon, got %T", data)
		}
		adders = []ComponentAdder{addRangedWeaponComponents(w)}

	case EntityConsumable:
		c, ok := data.(JSONAttributeModifier)
		if !ok {
			log.Fatalf("Expected JSONAttributeModifier for EntityConsumable, got %T", data)
		}
		adders = []ComponentAdder{addConsumableComponents(c)}

	case EntityCreature:
		m, ok := data.(JSONMonster)
		if !ok {
			log.Fatalf("Expected JSONMonster for EntityCreature, got %T", data)
		}
		adders = []ComponentAdder{addCreatureComponents(m)}

		// Handle creature-specific map blocking logic
		if config.GameMap != nil && config.Position != nil {
			ind := coords.CoordManager.LogicalToIndex(*config.Position)
			config.GameMap.Tiles[ind].Blocked = true
		}

	default:
		log.Fatalf("Unknown entity type: %d", config.Type)
	}

	return createFromTemplate(manager, config.Name, config.ImagePath, config.AssetDir,
		config.Visible, config.Position, adders...)
}

// Deprecated: Use CreateEntityFromTemplate instead.
// Legacy wrapper maintained for backward compatibility with existing spawning code.
func CreateMeleeWepFromTemplate(manager common.EntityManager, w JSONMeleeWeapon) *ecs.Entity {
	return CreateEntityFromTemplate(manager, EntityConfig{
		Type:      EntityMeleeWeapon,
		Name:      w.Name,
		ImagePath: w.ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, w)
}

// Deprecated: Use CreateEntityFromTemplate instead.
// Legacy wrapper maintained for backward compatibility with existing spawning code.
func CreateRangedWepFromTemplate(manager common.EntityManager, w JSONRangedWeapon) *ecs.Entity {
	return CreateEntityFromTemplate(manager, EntityConfig{
		Type:      EntityRangedWeapon,
		Name:      w.Name,
		ImagePath: w.ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, w)
}

// Deprecated: Use CreateEntityFromTemplate instead.
// Legacy wrapper maintained for backward compatibility with existing spawning code.
func CreateConsumableFromTemplate(manager common.EntityManager, c JSONAttributeModifier) *ecs.Entity {
	return CreateEntityFromTemplate(manager, EntityConfig{
		Type:      EntityConsumable,
		Name:      c.Name,
		ImagePath: c.ImgName,
		AssetDir:  "../assets/items/",
		Visible:   false,
		Position:  nil,
	}, c)
}

// Deprecated: Use CreateEntityFromTemplate instead.
// Legacy wrapper maintained for backward compatibility with existing spawning code.
func CreateCreatureFromTemplate(manager common.EntityManager, m JSONMonster, gm *worldmap.GameMap, xPos, yPos int) *ecs.Entity {
	return CreateEntityFromTemplate(manager, EntityConfig{
		Type:      EntityCreature,
		Name:      m.Name,
		ImagePath: m.ImageName,
		AssetDir:  "../assets/creatures/",
		Visible:   true,
		Position:  &coords.LogicalPosition{X: xPos, Y: yPos},
		GameMap:   gm,
	}, m)
}
