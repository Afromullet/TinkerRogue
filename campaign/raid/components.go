package raid

import "github.com/bytearena/ecs"

// ECS components for the raid system
var (
	RaidStateComponent    *ecs.Component
	FloorStateComponent   *ecs.Component
	RoomDataComponent     *ecs.Component
	AlertDataComponent    *ecs.Component
	GarrisonSquadComponent *ecs.Component
	DeploymentComponent   *ecs.Component

	RaidStateTag    ecs.Tag
	FloorStateTag   ecs.Tag
	RoomDataTag     ecs.Tag
	AlertDataTag    ecs.Tag
	GarrisonSquadTag ecs.Tag
	DeploymentTag   ecs.Tag
)

// RaidStatus represents the current state of a raid
type RaidStatus int

const (
	RaidNone      RaidStatus = iota // No raid active (zero value)
	RaidActive
	RaidVictory
	RaidDefeat
	RaidRetreated
)

func (s RaidStatus) String() string {
	switch s {
	case RaidNone:
		return "None"
	case RaidActive:
		return "Active"
	case RaidVictory:
		return "Victory"
	case RaidDefeat:
		return "Defeat"
	case RaidRetreated:
		return "Retreated"
	default:
		return "Unknown"
	}
}

// RaidStateData is a singleton tracking the overall raid progress.
type RaidStateData struct {
	CurrentFloor     int
	TotalFloors      int
	Status           RaidStatus
	CommanderID      ecs.EntityID
	PlayerEntityID   ecs.EntityID
	PlayerSquadIDs   []ecs.EntityID
}

// FloorStateData tracks the state of a single garrison floor.
type FloorStateData struct {
	FloorNumber      int
	RoomsCleared     int
	RoomsTotal       int
	GarrisonSquadIDs []ecs.EntityID
	ReserveSquadIDs  []ecs.EntityID
	IsComplete       bool
}

// RoomData tracks an individual room within a floor's DAG.
type RoomData struct {
	NodeID              int
	RoomType            string
	FloorNumber         int
	IsCleared           bool
	IsAccessible        bool
	GarrisonSquadIDs    []ecs.EntityID
	ChildNodeIDs        []int
	ParentNodeIDs       []int
	OnCriticalPath      bool
}

// AlertData tracks the alert level for a floor.
type AlertData struct {
	FloorNumber    int
	CurrentLevel   int // 0-3
	EncounterCount int
}

// GarrisonSquadData marks a squad as part of the garrison defense.
type GarrisonSquadData struct {
	ArchetypeName string
	FloorNumber   int
	RoomNodeID    int
	IsReserve   bool
	IsDestroyed bool
}

// DeploymentData tracks which player squads are deployed vs. reserved for the current encounter.
type DeploymentData struct {
	DeployedSquadIDs []ecs.EntityID
	ReserveSquadIDs  []ecs.EntityID
}
