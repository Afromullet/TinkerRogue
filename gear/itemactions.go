package gear

import (
	"game_main/visual/graphics"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

const (
	THROWABLE_ACTION_NAME = "Throwable"
)

// ItemAction represents actions that can be performed with items like throwing, consuming, activating
// This is separate from StatusEffects which are ongoing conditions that affect creatures
type ItemAction interface {
	ActionName() string
	ActionComponent() *ecs.Component

	// Execute performs the action and returns any status effects that should be applied to affected entities
	Execute(targetPos *coords.LogicalPosition, sourcePos *coords.LogicalPosition, world *ecs.Manager, worldTags map[string]ecs.Tag) []StatusEffects

	// CanExecute checks if the action can be performed given the target and source positions
	CanExecute(targetPos *coords.LogicalPosition, sourcePos *coords.LogicalPosition) bool

	// GetVisualEffect returns the visual effect associated with this action (if any)
	GetVisualEffect() graphics.VisualEffect

	// GetAOEShape returns the area of effect shape for this action
	GetAOEShape() graphics.TileBasedShape

	// GetRange returns the maximum range this action can be performed at
	GetRange() int

	// Copy creates a copy of this action
	Copy() ItemAction
}

// ThrowableAction represents the action of throwing an item
type ThrowableAction struct {
	MainProps      CommonItemProperties
	ThrowingRange  int
	Damage         int
	Shape          graphics.TileBasedShape
	VX             graphics.VisualEffect
	EffectsToApply []StatusEffects // Effects that get applied when thrown
}

func (t *ThrowableAction) ActionName() string {
	return t.MainProps.Name
}

func (t *ThrowableAction) ActionComponent() *ecs.Component {
	return ThrowableComponent
}

func (t *ThrowableAction) Execute(targetPos *coords.LogicalPosition, sourcePos *coords.LogicalPosition, world *ecs.Manager, worldTags map[string]ecs.Tag) []StatusEffects {
	// Start visual effect if present
	if t.VX != nil {
		t.VX.ResetVX()
		graphics.AddVXArea(graphics.NewVisualEffectArea(sourcePos.X, sourcePos.Y, t.Shape, t.VX))
	}

	//TODO, maybe consider replacing this with targetable spells
	// Get positions affected by the shape
	//affectedPositions := coords.CoordManager.GetTilePositionsAsCommon(t.Shape.GetIndices())
	appliedEffects := make([]StatusEffects, 0)

	// Apply effects to monsters in affected area

	/*
		for _, c := range world.Query(worldTags["monsters"]) {
			monsterPos := common.GetComponentType[*coords.LogicalPosition](c.Entity, common.PositionComponent)

			for _, pos := range affectedPositions {
				if monsterPos.IsEqual(&pos) && monsterPos.InRange(sourcePos, t.ThrowingRange) {
					// Collect all effects that were applied
					for _, effect := range t.EffectsToApply {
						appliedEffects = append(appliedEffects, effect.Copy())
					}
				}
			}
		}
	*/

	return appliedEffects
}

func (t *ThrowableAction) CanExecute(targetPos *coords.LogicalPosition, sourcePos *coords.LogicalPosition) bool {
	return targetPos.InRange(sourcePos, t.ThrowingRange)
}

func (t *ThrowableAction) GetVisualEffect() graphics.VisualEffect {
	return t.VX
}

func (t *ThrowableAction) GetAOEShape() graphics.TileBasedShape {
	return t.Shape
}

func (t *ThrowableAction) GetRange() int {
	return t.ThrowingRange
}

func (t *ThrowableAction) Copy() ItemAction {
	effectsCopy := make([]StatusEffects, len(t.EffectsToApply))
	for i, effect := range t.EffectsToApply {
		effectsCopy[i] = effect.Copy()
	}

	return &ThrowableAction{
		MainProps:      t.MainProps,
		ThrowingRange:  t.ThrowingRange,
		Damage:         t.Damage,
		Shape:          t.Shape,
		VX:             t.VX,
		EffectsToApply: effectsCopy,
	}
}

// InRange checks if the action can reach the target position (legacy method for compatibility)
func (t *ThrowableAction) InRange(endPos *coords.LogicalPosition) bool {
	pixelX, pixelY := t.Shape.StartPositionPixels()
	pixelPos := coords.PixelPosition{X: pixelX, Y: pixelY}
	logicalPos := coords.CoordManager.PixelToLogical(pixelPos)
	startPos := coords.LogicalPosition{X: logicalPos.X, Y: logicalPos.Y}

	return endPos.InRange(&startPos, t.ThrowingRange)
}

// NewThrowableAction creates a new throwable action with the specified effects
func NewThrowableAction(dur, throwRange, dam int, shape graphics.TileBasedShape, effects ...StatusEffects) *ThrowableAction {
	return &ThrowableAction{
		MainProps: CommonItemProperties{
			Name:     THROWABLE_ACTION_NAME,
			Duration: dur,
		},
		ThrowingRange:  throwRange,
		Damage:         dam,
		Shape:          shape,
		EffectsToApply: effects,
	}
}

// NewShapeThrowableAction creates a throwable action with a basic shape and effects
func NewShapeThrowableAction(dur, throwRange, dam int, shapeType graphics.BasicShapeType, size graphics.ShapeSize, direction *graphics.ShapeDirection, effects ...StatusEffects) *ThrowableAction {
	var shape graphics.TileBasedShape

	switch shapeType {
	case graphics.Circular:
		shape = graphics.NewCircle(0, 0, size)
	case graphics.Rectangular:
		shape = graphics.NewSquare(0, 0, size)
	case graphics.Linear:
		if direction != nil {
			shape = graphics.NewLine(0, 0, *direction, size)
		} else {
			shape = graphics.NewLine(0, 0, graphics.LineRight, size)
		}
	}

	return NewThrowableAction(dur, throwRange, dam, shape, effects...)
}
