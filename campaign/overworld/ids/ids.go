// Package ids defines typed string identifiers used across the overworld
// subsystems and the JSON template layer.
//
// Each kind of ID gets its own named string type so the compiler can catch
// mistakes like passing an EncounterTypeID where a FactionID is expected.
// All types are string-based, so they round-trip through encoding/json
// natively and impose no runtime cost.
//
// This package intentionally has no dependency on campaign/overworld/core or
// templates so both can import it without creating a cycle.
package ids

// NodeTypeID identifies a placeable node template (e.g. "necromancer",
// "town", "watchtower"). It matches the "id" field in nodeDefinitions.json.
type NodeTypeID string

const (
	NodeTypeTown       NodeTypeID = "town"
	NodeTypeGuildHall  NodeTypeID = "guild_hall"
	NodeTypeTemple     NodeTypeID = "temple"
	NodeTypeWatchtower NodeTypeID = "watchtower"
)

// FactionID identifies a faction by its display name string.
// It matches FactionType.String() at runtime ("Necromancers", "Bandits", ...).
type FactionID string

// EncounterID is the registry key for a runtime EncounterDefinition.
type EncounterID string

// EncounterTypeID identifies an encounter type for combat spawn logic
// (e.g. "undead_basic"). Consumed by the mind/encounter package.
type EncounterTypeID string

// OwnerID identifies the controller of an overworld node. It is a
// discriminated string holding either a sentinel (player/neutral) or a
// faction display name (matching FactionID).
type OwnerID string

const (
	OwnerPlayer  OwnerID = "player"
	OwnerNeutral OwnerID = "Neutral"
)

// IsHostileOwner returns true if the owner is neither player nor neutral.
func IsHostileOwner(o OwnerID) bool {
	return o != OwnerPlayer && o != OwnerNeutral
}

// IsFriendlyOwner returns true if the owner is the player.
func IsFriendlyOwner(o OwnerID) bool {
	return o == OwnerPlayer
}
