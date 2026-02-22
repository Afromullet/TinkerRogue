package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/mind/raid"
	"game_main/savesystem"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&RaidChunk{})
}

// RaidChunk saves/loads raid progression state: RaidState, FloorState,
// RoomData, AlertData, GarrisonSquad, and Deployment entities.
type RaidChunk struct{}

func (c *RaidChunk) ChunkID() string  { return "raid" }
func (c *RaidChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedRaidChunkData struct {
	RaidState      *savedRaidState      `json:"raidState,omitempty"`
	Floors         []savedFloorState    `json:"floors,omitempty"`
	Rooms          []savedRoomData      `json:"rooms,omitempty"`
	Alerts         []savedAlertData     `json:"alerts,omitempty"`
	GarrisonSquads []savedGarrisonSquad `json:"garrisonSquads,omitempty"`
	Deployment     *savedDeployment     `json:"deployment,omitempty"`
}

type savedRaidState struct {
	EntityID       ecs.EntityID   `json:"entityID"`
	CurrentFloor   int            `json:"currentFloor"`
	TotalFloors    int            `json:"totalFloors"`
	Status         int            `json:"status"`
	CommanderID    ecs.EntityID   `json:"commanderID"`
	PlayerSquadIDs []ecs.EntityID `json:"playerSquadIDs"`
}

type savedFloorState struct {
	EntityID         ecs.EntityID   `json:"entityID"`
	FloorNumber      int            `json:"floorNumber"`
	RoomsCleared     int            `json:"roomsCleared"`
	RoomsTotal       int            `json:"roomsTotal"`
	GarrisonSquadIDs []ecs.EntityID `json:"garrisonSquadIDs"`
	ReserveSquadIDs  []ecs.EntityID `json:"reserveSquadIDs"`
	IsComplete       bool           `json:"isComplete"`
}

type savedRoomData struct {
	EntityID         ecs.EntityID   `json:"entityID"`
	NodeID           int            `json:"nodeID"`
	RoomType         string         `json:"roomType"`
	FloorNumber      int            `json:"floorNumber"`
	IsCleared        bool           `json:"isCleared"`
	IsAccessible     bool           `json:"isAccessible"`
	GarrisonSquadIDs []ecs.EntityID `json:"garrisonSquadIDs"`
	ChildNodeIDs     []int          `json:"childNodeIDs"`
	ParentNodeIDs    []int          `json:"parentNodeIDs"`
	OnCriticalPath   bool           `json:"onCriticalPath"`
}

type savedAlertData struct {
	EntityID       ecs.EntityID `json:"entityID"`
	FloorNumber    int          `json:"floorNumber"`
	CurrentLevel   int          `json:"currentLevel"`
	EncounterCount int          `json:"encounterCount"`
}

type savedGarrisonSquad struct {
	EntityID      ecs.EntityID `json:"entityID"`
	ArchetypeName string       `json:"archetypeName"`
	FloorNumber   int          `json:"floorNumber"`
	RoomNodeID    int          `json:"roomNodeID"`
	IsReserve     bool         `json:"isReserve"`
	IsDestroyed   bool         `json:"isDestroyed"`
}

type savedDeployment struct {
	EntityID         ecs.EntityID   `json:"entityID"`
	DeployedSquadIDs []ecs.EntityID `json:"deployedSquadIDs"`
	ReserveSquadIDs  []ecs.EntityID `json:"reserveSquadIDs"`
}

// --- Save ---

func (c *RaidChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	chunkData := savedRaidChunkData{}

	// RaidState (singleton)
	for _, result := range em.World.Query(raid.RaidStateTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.RaidStateData](entity, raid.RaidStateComponent); data != nil {
			chunkData.RaidState = &savedRaidState{
				EntityID:       entity.GetID(),
				CurrentFloor:   data.CurrentFloor,
				TotalFloors:    data.TotalFloors,
				Status:         int(data.Status),
				CommanderID:    data.CommanderID,
				PlayerSquadIDs: copyEntityIDs(data.PlayerSquadIDs),
			}
		}
	}

	// No active raid? Skip the rest.
	if chunkData.RaidState == nil {
		return json.Marshal(chunkData)
	}

	// FloorStates
	for _, result := range em.World.Query(raid.FloorStateTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.FloorStateData](entity, raid.FloorStateComponent); data != nil {
			chunkData.Floors = append(chunkData.Floors, savedFloorState{
				EntityID:         entity.GetID(),
				FloorNumber:      data.FloorNumber,
				RoomsCleared:     data.RoomsCleared,
				RoomsTotal:       data.RoomsTotal,
				GarrisonSquadIDs: copyEntityIDs(data.GarrisonSquadIDs),
				ReserveSquadIDs:  copyEntityIDs(data.ReserveSquadIDs),
				IsComplete:       data.IsComplete,
			})
		}
	}

	// RoomData
	for _, result := range em.World.Query(raid.RoomDataTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.RoomData](entity, raid.RoomDataComponent); data != nil {
			chunkData.Rooms = append(chunkData.Rooms, savedRoomData{
				EntityID:         entity.GetID(),
				NodeID:           data.NodeID,
				RoomType:         data.RoomType,
				FloorNumber:      data.FloorNumber,
				IsCleared:        data.IsCleared,
				IsAccessible:     data.IsAccessible,
				GarrisonSquadIDs: copyEntityIDs(data.GarrisonSquadIDs),
				ChildNodeIDs:     copyInts(data.ChildNodeIDs),
				ParentNodeIDs:    copyInts(data.ParentNodeIDs),
				OnCriticalPath:   data.OnCriticalPath,
			})
		}
	}

	// AlertData
	for _, result := range em.World.Query(raid.AlertDataTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.AlertData](entity, raid.AlertDataComponent); data != nil {
			chunkData.Alerts = append(chunkData.Alerts, savedAlertData{
				EntityID:       entity.GetID(),
				FloorNumber:    data.FloorNumber,
				CurrentLevel:   data.CurrentLevel,
				EncounterCount: data.EncounterCount,
			})
		}
	}

	// GarrisonSquads
	for _, result := range em.World.Query(raid.GarrisonSquadTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.GarrisonSquadData](entity, raid.GarrisonSquadComponent); data != nil {
			chunkData.GarrisonSquads = append(chunkData.GarrisonSquads, savedGarrisonSquad{
				EntityID:      entity.GetID(),
				ArchetypeName: data.ArchetypeName,
				FloorNumber:   data.FloorNumber,
				RoomNodeID:    data.RoomNodeID,
				IsReserve:     data.IsReserve,
				IsDestroyed:   data.IsDestroyed,
			})
		}
	}

	// Deployment
	for _, result := range em.World.Query(raid.DeploymentTag) {
		entity := result.Entity
		if data := common.GetComponentType[*raid.DeploymentData](entity, raid.DeploymentComponent); data != nil {
			chunkData.Deployment = &savedDeployment{
				EntityID:         entity.GetID(),
				DeployedSquadIDs: copyEntityIDs(data.DeployedSquadIDs),
				ReserveSquadIDs:  copyEntityIDs(data.ReserveSquadIDs),
			}
		}
	}

	return json.Marshal(chunkData)
}

// --- Load ---

func (c *RaidChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	var chunkData savedRaidChunkData
	if err := json.Unmarshal(data, &chunkData); err != nil {
		return fmt.Errorf("failed to unmarshal raid data: %w", err)
	}

	// RaidState
	if chunkData.RaidState != nil {
		rs := chunkData.RaidState
		entity := em.World.NewEntity()
		entity.AddComponent(raid.RaidStateComponent, &raid.RaidStateData{
			CurrentFloor:   rs.CurrentFloor,
			TotalFloors:    rs.TotalFloors,
			Status:         raid.RaidStatus(rs.Status),
			CommanderID:    rs.CommanderID,    // remapped later
			PlayerSquadIDs: rs.PlayerSquadIDs, // remapped later
		})
		idMap.Register(rs.EntityID, entity.GetID())
	}

	// FloorStates
	for _, sf := range chunkData.Floors {
		entity := em.World.NewEntity()
		entity.AddComponent(raid.FloorStateComponent, &raid.FloorStateData{
			FloorNumber:      sf.FloorNumber,
			RoomsCleared:     sf.RoomsCleared,
			RoomsTotal:       sf.RoomsTotal,
			GarrisonSquadIDs: sf.GarrisonSquadIDs, // remapped later
			ReserveSquadIDs:  sf.ReserveSquadIDs,   // remapped later
			IsComplete:       sf.IsComplete,
		})
		idMap.Register(sf.EntityID, entity.GetID())
	}

	// RoomData
	for _, sr := range chunkData.Rooms {
		entity := em.World.NewEntity()
		entity.AddComponent(raid.RoomDataComponent, &raid.RoomData{
			NodeID:           sr.NodeID,
			RoomType:         sr.RoomType,
			FloorNumber:      sr.FloorNumber,
			IsCleared:        sr.IsCleared,
			IsAccessible:     sr.IsAccessible,
			GarrisonSquadIDs: sr.GarrisonSquadIDs, // remapped later
			ChildNodeIDs:     copyInts(sr.ChildNodeIDs),
			ParentNodeIDs:    copyInts(sr.ParentNodeIDs),
			OnCriticalPath:   sr.OnCriticalPath,
		})
		idMap.Register(sr.EntityID, entity.GetID())
	}

	// AlertData
	for _, sa := range chunkData.Alerts {
		entity := em.World.NewEntity()
		entity.AddComponent(raid.AlertDataComponent, &raid.AlertData{
			FloorNumber:    sa.FloorNumber,
			CurrentLevel:   sa.CurrentLevel,
			EncounterCount: sa.EncounterCount,
		})
		idMap.Register(sa.EntityID, entity.GetID())
	}

	// GarrisonSquads
	for _, sg := range chunkData.GarrisonSquads {
		entity := em.World.NewEntity()
		entity.AddComponent(raid.GarrisonSquadComponent, &raid.GarrisonSquadData{
			ArchetypeName: sg.ArchetypeName,
			FloorNumber:   sg.FloorNumber,
			RoomNodeID:    sg.RoomNodeID,
			IsReserve:     sg.IsReserve,
			IsDestroyed:   sg.IsDestroyed,
		})
		idMap.Register(sg.EntityID, entity.GetID())
	}

	// Deployment
	if chunkData.Deployment != nil {
		sd := chunkData.Deployment
		entity := em.World.NewEntity()
		entity.AddComponent(raid.DeploymentComponent, &raid.DeploymentData{
			DeployedSquadIDs: sd.DeployedSquadIDs, // remapped later
			ReserveSquadIDs:  sd.ReserveSquadIDs,   // remapped later
		})
		idMap.Register(sd.EntityID, entity.GetID())
	}

	return nil
}

// --- RemapIDs ---

func (c *RaidChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	// RaidState
	for _, result := range em.World.Query(raid.RaidStateTag) {
		data := common.GetComponentType[*raid.RaidStateData](result.Entity, raid.RaidStateComponent)
		if data != nil {
			data.CommanderID = idMap.Remap(data.CommanderID)
			data.PlayerSquadIDs = idMap.RemapSlice(data.PlayerSquadIDs)
		}
	}

	// FloorStates
	for _, result := range em.World.Query(raid.FloorStateTag) {
		data := common.GetComponentType[*raid.FloorStateData](result.Entity, raid.FloorStateComponent)
		if data != nil {
			data.GarrisonSquadIDs = idMap.RemapSlice(data.GarrisonSquadIDs)
			data.ReserveSquadIDs = idMap.RemapSlice(data.ReserveSquadIDs)
		}
	}

	// RoomData
	for _, result := range em.World.Query(raid.RoomDataTag) {
		data := common.GetComponentType[*raid.RoomData](result.Entity, raid.RoomDataComponent)
		if data != nil {
			data.GarrisonSquadIDs = idMap.RemapSlice(data.GarrisonSquadIDs)
		}
	}

	// Deployment
	for _, result := range em.World.Query(raid.DeploymentTag) {
		data := common.GetComponentType[*raid.DeploymentData](result.Entity, raid.DeploymentComponent)
		if data != nil {
			data.DeployedSquadIDs = idMap.RemapSlice(data.DeployedSquadIDs)
			data.ReserveSquadIDs = idMap.RemapSlice(data.ReserveSquadIDs)
		}
	}

	return nil
}

// Helpers (copyEntityIDs, copyInts) are in shared_types.go
