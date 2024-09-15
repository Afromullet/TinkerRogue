package equipment

import (
	"game_main/common"
	"game_main/graphics"
	"math/big"

	cryptorand "crypto/rand"

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

// This gets called so often that it might as well be a function
func GetArmor(e *ecs.Entity) *Armor {
	return common.GetComponentType[*Armor](e, ArmorComponent)
}

type MeleeWeapon struct {
	MinDamage   int
	MaxDamage   int
	AttackSpeed int
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

// todo add ammo to this
func (r RangedWeapon) CalculateDamage() int {

	return GetRandomBetween(r.MinDamage, r.MaxDamage)

}

// Gets all of the targets in the weapons AOE by accessing the TileBasedShape
// of the ranged weapon
func (r RangedWeapon) GetTargets(ecsmanger *common.EntityManager) []*ecs.Entity {

	pos := common.GetTilePositions(r.TargetArea.GetIndices())
	targets := make([]*ecs.Entity, 0)

	//TODO, this will be slow in case there are a lot of creatures
	for _, c := range ecsmanger.World.Query(ecsmanger.WorldTags["monsters"]) {

		curPos := c.Components[common.PositionComponent].(*common.Position)

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

	gd := graphics.NewScreenData()

	attX, attY := common.PixelsFromPosition(attackerPos, gd.TileWidth, gd.TileHeight)
	defX, defY := common.PixelsFromPosition(defenderPos, gd.TileWidth, gd.TileHeight)

	arr := graphics.NewProjectile(attX, attY, defX, defY)

	graphics.AddVX(arr)
}

func GetItem(e *ecs.Entity) *Item {
	return common.GetComponentType[*Item](e, ItemComponent)
}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
// Here to avoid circular inclusions of randgen
func GetRandomBetween(low int, high int) int {
	var randy int = -1
	for {
		randy = GetDiceRoll(high)
		if randy >= low {
			break
		}
	}
	return randy
}

// Todo remove later once you change teh random number generation. The same function is in another aprt of the code
// Here to avoid circular inclusions of randgen
func GetDiceRoll(num int) int {
	x, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(num)))
	return int(x.Int64()) + 1

}
