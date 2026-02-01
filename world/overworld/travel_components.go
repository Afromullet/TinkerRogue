package overworld

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

var (
	TravelStateComponent *ecs.Component
	TravelStateTag       ecs.Tag
)

// TravelStateData - Singleton component tracking player travel state
type TravelStateData struct {
	IsTraveling       bool                   // Currently traveling
	Origin            coords.LogicalPosition // Starting position
	Destination       coords.LogicalPosition // Target position (threat node)
	TotalDistance     float64                // Euclidean distance (calculated once at start)
	RemainingDistance float64                // Distance left to travel
	TargetThreatID    ecs.EntityID           // Threat being traveled to
	TargetEncounterID ecs.EntityID           // Encounter entity created for this travel
}
