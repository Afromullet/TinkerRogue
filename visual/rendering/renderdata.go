package rendering

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// SquadRenderInfo holds minimal data a renderer needs about a squad
type SquadRenderInfo struct {
	ID          ecs.EntityID
	Position    *coords.LogicalPosition
	FactionID   ecs.EntityID
	IsDestroyed bool
	CurrentHP   int
	MaxHP       int
}

// UnitRenderInfo holds minimal data a renderer needs about a single unit
type UnitRenderInfo struct {
	AnchorRow int
	AnchorCol int
	Width     int
	Height    int
	Image     *ebiten.Image
	IsAlive   bool
}

// SquadInfoProvider supplies squad data for rendering.
// Satisfied by gui/framework.GUIQueries with adapter methods.
type SquadInfoProvider interface {
	GetAllSquadIDs() []ecs.EntityID
	GetSquadRenderInfo(squadID ecs.EntityID) *SquadRenderInfo
}

// UnitInfoProvider supplies unit data for combat animation rendering.
type UnitInfoProvider interface {
	GetUnitIDsInSquad(squadID ecs.EntityID) []ecs.EntityID
	GetUnitRenderInfo(unitID ecs.EntityID) *UnitRenderInfo
}
