package main

import (
	"game_main/common"
	"game_main/equipment"
	"game_main/graphics"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

/*
 */

var (
	RenderableComponent *ecs.Component

	CreatureComponent *ecs.Component

	userMessage *ecs.Component
)

// The ECS library returns pointers to the struct when querying it for components, so the Position methods take a pointer as input
// Other than that, there's no reason for using pointers for the functions below.

type Renderable struct {
	Image   *ebiten.Image
	Visible bool
}

type UserMessage struct {
	AttackMessage    string
	GameStateMessage string
}

// TargetArea is the area the weapon covers
// I.E, a pistol is just a 1 by 1 rectangle, a shotgun uses a cone, and so on
// ShootingVX is the visual effect that is drawn when the weapon shoots
type RangedWeapon struct {
	MinDamage     int
	MaxDamage     int
	ShootingRange int
	TargetArea    graphics.TileBasedShape
	ShootingVX    *graphics.Projectile
}

// todo add ammo to this
func (r RangedWeapon) CalculateDamage() int {

	return GetRandomBetween(r.MinDamage, r.MaxDamage)

}

// Gets all of the targets in the weapons AOE
func (r RangedWeapon) GetTargets(g *Game) []*ecs.Entity {

	pos := GetTilePositions(r.TargetArea)
	targets := make([]*ecs.Entity, 0)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range g.World.Query(g.WorldTags["monsters"]) {

		curPos := c.Components[common.PositionComponent].(*common.Position)

		for _, p := range pos {
			if curPos.IsEqual(&p) {
				targets = append(targets, c.Entity)

			}
		}

	}

	return targets
}

// Adds the Ranged Weapons VisuaLEffect to the VisualEffectHandler. It will be drawn.
func (r *RangedWeapon) DisplayShootingVX(attackerPos *common.Position, defenderPos *common.Position) {

	gd := graphics.NewScreenData()

	attX, attY := common.PixelsFromPosition(attackerPos, gd.TileWidth, gd.TileHeight)
	defX, defY := common.PixelsFromPosition(defenderPos, gd.TileWidth, gd.TileHeight)

	arr := graphics.NewProjectile(attX, attY, defX, defY)

	graphics.AddVX(arr)
}

// This gets called so often that it might as well be a function
func GetCreature(e *ecs.Entity) *Creature {
	return common.GetComponentType[*Creature](e, CreatureComponent)
}

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(g *Game) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	common.PositionComponent = manager.NewComponent()
	RenderableComponent = manager.NewComponent()

	common.NameComponent = manager.NewComponent()

	equipment.InventoryComponent = manager.NewComponent()

	common.AttributeComponent = manager.NewComponent()
	userMessage = manager.NewComponent()

	equipment.WeaponComponent = manager.NewComponent()
	equipment.RangedWeaponComponent = manager.NewComponent()
	equipment.ArmorComponent = manager.NewComponent()

	renderables := ecs.BuildTag(RenderableComponent, common.PositionComponent)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(userMessage)
	tags["messengers"] = messengers

	InitializeMovementComponents(manager, tags)
	equipment.InitializeItemComponents(manager, tags)
	InitializeCreatureComponents(manager, tags)

	g.WorldTags = tags
	g.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	CreatureComponent = manager.NewComponent()

	approachAndAttack = manager.NewComponent()
	distanceRangeAttack = manager.NewComponent()

	creatures := ecs.BuildTag(CreatureComponent, common.PositionComponent, common.AttributeComponent)
	tags["monsters"] = creatures

}

// Creates a slice of Positions from p to other. Uses AStar to build the path
func BuildPath(g *Game, start *common.Position, other *common.Position) []common.Position {

	astar := AStar{}
	return astar.GetPath(g.gameMap, start, other, false)

}
