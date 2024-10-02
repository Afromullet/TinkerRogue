package gear

import (
	"game_main/common"
	"game_main/graphics"

	"strconv"

	"github.com/bytearena/ecs"
)

var (
	ArmorComponent        *ecs.Component
	MeleeWeaponComponent  *ecs.Component
	InventoryComponent    *ecs.Component
	RangedWeaponComponent *ecs.Component
)

type Armor struct {
	ArmorClass  int
	Protection  int
	DodgeChance float32
}

func (a *Armor) Stats() string {

	s := ""
	s += "Armor Class: " + strconv.Itoa(a.ArmorClass) + "\n"
	s += "Protection: " + strconv.Itoa(a.Protection) + "\n"
	s += "Dodge: " + strconv.FormatFloat(float64(a.DodgeChance), 'f', 2, 32) + "\n"

	return s

}

type MeleeWeapon struct {
	MinDamage   int
	MaxDamage   int
	AttackSpeed int
}

func (w *MeleeWeapon) Stats() string {

	s := ""
	s += "Min Damage: " + strconv.Itoa(w.MinDamage) + "\n"
	s += "Max Damage: " + strconv.Itoa(w.MaxDamage) + "\n"
	s += "AttackSpeed: " + strconv.Itoa(w.AttackSpeed) + "\n"

	return s

}

func (w MeleeWeapon) CalculateDamage() int {

	return GetRandomBetween(w.MinDamage, w.MaxDamage)

}

// TargetArea is the area the weapon covers, defined by a TileShape
// I.E, a pistol is just a 1 by 1 rectangle, a shotgun uses a cone, and so on
// ShootingVX is the visual effect that is drawn when the weapon shoots
type RangedWeapon struct {
	MinDamage     int
	MaxDamage     int
	ShootingRange int
	TargetArea    graphics.TileBasedShape
	ShootingVX    *graphics.Projectile
	AttackSpeed   int
}

func (w *RangedWeapon) Stats() string {

	s := ""
	s += "Min Damage: " + strconv.Itoa(w.MinDamage) + "\n"
	s += "Max Damage: " + strconv.Itoa(w.MaxDamage) + "\n"
	s += "Attack Speed: " + strconv.Itoa(w.AttackSpeed) + "\n"
	s += "Range: " + strconv.Itoa(w.ShootingRange) + "\n"

	return s

}

// todo add ammo to this
func (r RangedWeapon) CalculateDamage() int {

	return GetRandomBetween(r.MinDamage, r.MaxDamage)

}

// Gets all of the targets in the weapons AOE by accessing the TileBasedShape
// of the ranged weapon
func (r RangedWeapon) GetTargets(ecsmanger *common.EntityManager) []*ecs.Entity {

	pos := common.GetTilePositions(r.TargetArea.GetIndices(), graphics.ScreenInfo.DungeonWidth)
	targets := make([]*ecs.Entity, 0)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		curPos := common.GetPosition(c.Entity)

		for _, p := range pos {
			if curPos.IsEqual(&p) {
				targets = append(targets, c.Entity)

			}
		}

	}

	return targets
}

// Adds the Ranged Weapons VisuaLEffect to the VisualEffectHandler.
// Todo determine whether this can be moved to the graphics package
func (r *RangedWeapon) DisplayShootingVX(attackerPos *common.Position, defenderPos *common.Position) {

	attX, attY := graphics.CoordTransformer.PixelsFromLogicalXY(attackerPos.X, attackerPos.Y, graphics.ScreenInfo.TileSize)
	defX, defY := graphics.CoordTransformer.PixelsFromLogicalXY(defenderPos.X, defenderPos.Y, graphics.ScreenInfo.TileSize)

	arr := graphics.NewProjectile(attX, attY, defX, defY)

	graphics.AddVX(arr)
}

func (r *RangedWeapon) DisplayCenteredShootingVX(attackerPos *common.Position, defenderPos *common.Position) {
	// Convert logical coordinates to screen coordinates for attacker
	attScreenX, attScreenY := graphics.OffsetFromCenter(
		attackerPos.X,
		attackerPos.Y,
		attackerPos.X*graphics.ScreenInfo.TileSize,
		attackerPos.Y*graphics.ScreenInfo.TileSize,
		graphics.ScreenInfo,
	)

	// Convert logical coordinates to screen coordinates for defender
	defScreenX, defScreenY := graphics.OffsetFromCenter(
		attackerPos.X, // Use attacker's position as the reference point
		attackerPos.Y,
		defenderPos.X*graphics.ScreenInfo.TileSize,
		defenderPos.Y*graphics.ScreenInfo.TileSize,
		graphics.ScreenInfo,
	)

	// Create the projectile using the screen coordinates
	arr := graphics.NewProjectile(
		int(attScreenX),
		int(attScreenY),
		int(defScreenX),
		int(defScreenY),
	)

	graphics.AddVX(arr)
}
