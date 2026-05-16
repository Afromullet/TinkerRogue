// Package roomtypes defines the string identifiers for garrison-raid room
// types. It's a leaf package so both world/garrisongen (the generator) and
// templates (the JSON validator) can reference the same authoritative list
// without templates pulling in worldmap/worldgen dependencies.
package roomtypes

const (
	Barracks    = "barracks"
	GuardPost   = "guard_post"
	Armory      = "armory"
	CommandPost = "command_post"
	PatrolRoute = "patrol_route"
	MageTower   = "mage_tower"
	RestRoom    = "rest_room"
	Stairs      = "stairs"
)

// All lists every valid room type in declaration order. Used by validators
// and by callers that need to iterate the full set.
var All = []string{
	Barracks,
	GuardPost,
	Armory,
	CommandPost,
	PatrolRoute,
	MageTower,
	RestRoom,
	Stairs,
}

// Valid is a set form of All for O(1) membership tests.
var Valid = func() map[string]bool {
	m := make(map[string]bool, len(All))
	for _, t := range All {
		m[t] = true
	}
	return m
}()
