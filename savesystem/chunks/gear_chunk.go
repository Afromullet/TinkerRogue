package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/savesystem"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&GearChunk{})
}

// GearChunk saves/loads artifact inventory and equipment data.
// Artifact inventory lives on the player entity; equipment lives on squad entities.
// Since these components attach to entities created by other chunks, the actual
// component attachment happens in RemapIDs (after all entities exist).
// Intermediate data between Load and RemapIDs is stored in EntityIDMap.LoadContext
// so the chunk struct remains stateless.
type GearChunk struct{}

func (c *GearChunk) ChunkID() string  { return "gear" }
func (c *GearChunk) ChunkVersion() int { return 1 }

// --- Serialization structs ---

type savedGearChunkData struct {
	Inventory *savedArtifactInventory `json:"inventory,omitempty"`
	Equipment []savedEquipment        `json:"equipment,omitempty"`
}

type savedArtifactInventory struct {
	OwnerEntityID  ecs.EntityID                      `json:"ownerEntityID"`
	MaxArtifacts   int                               `json:"maxArtifacts"`
	OwnedArtifacts map[string][]savedArtifactInstance `json:"ownedArtifacts"`
}

type savedArtifactInstance struct {
	EquippedOn ecs.EntityID `json:"equippedOn"`
}

type savedEquipment struct {
	SquadEntityID     ecs.EntityID `json:"squadEntityID"`
	EquippedArtifacts []string     `json:"equippedArtifacts"`
}

// --- Save ---

func (c *GearChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	chunkData := savedGearChunkData{}

	// Find artifact inventory on the player entity
	playerTag, ok := em.WorldTags["players"]
	if ok {
		results := em.World.Query(playerTag)
		if len(results) > 0 {
			entity := results[0].Entity
			if inv := common.GetComponentType[*gear.ArtifactInventoryData](entity, gear.ArtifactInventoryComponent); inv != nil {
				si := &savedArtifactInventory{
					OwnerEntityID:  entity.GetID(),
					MaxArtifacts:   inv.MaxArtifacts,
					OwnedArtifacts: make(map[string][]savedArtifactInstance),
				}
				for defID, instances := range inv.OwnedArtifacts {
					savedInstances := make([]savedArtifactInstance, len(instances))
					for i, inst := range instances {
						savedInstances[i] = savedArtifactInstance{EquippedOn: inst.EquippedOn}
					}
					si.OwnedArtifacts[defID] = savedInstances
				}
				chunkData.Inventory = si
			}
		}
	}

	// Save equipment data from all squads
	for _, result := range em.World.Query(squads.SquadTag) {
		entity := result.Entity
		if equipData := common.GetComponentType[*gear.EquipmentData](entity, gear.EquipmentComponent); equipData != nil {
			se := savedEquipment{
				SquadEntityID:     entity.GetID(),
				EquippedArtifacts: make([]string, len(equipData.EquippedArtifacts)),
			}
			copy(se.EquippedArtifacts, equipData.EquippedArtifacts)
			chunkData.Equipment = append(chunkData.Equipment, se)
		}
	}

	return json.Marshal(chunkData)
}

// --- Load ---

const gearLoadContextKey = "gear_pending_data"

func (c *GearChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	var chunkData savedGearChunkData
	if err := json.Unmarshal(data, &chunkData); err != nil {
		return fmt.Errorf("failed to unmarshal gear data: %w", err)
	}

	// Store in LoadContext for RemapIDs phase (player/squad entities may not exist yet)
	idMap.LoadContext[gearLoadContextKey] = &chunkData
	return nil
}

// --- RemapIDs ---

func (c *GearChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	raw, ok := idMap.LoadContext[gearLoadContextKey]
	if !ok {
		return nil
	}
	pendingData := raw.(*savedGearChunkData)
	delete(idMap.LoadContext, gearLoadContextKey)

	// Attach artifact inventory to the player entity
	if pendingData.Inventory != nil {
		si := pendingData.Inventory
		newOwnerID := idMap.Remap(si.OwnerEntityID)
		if newOwnerID != 0 {
			ownerEntity := em.FindEntityByID(newOwnerID)
			if ownerEntity != nil {
				inv := gear.NewArtifactInventory(si.MaxArtifacts)
				for defID, savedInstances := range si.OwnedArtifacts {
					instances := make([]*gear.ArtifactInstance, len(savedInstances))
					for i, si := range savedInstances {
						instances[i] = &gear.ArtifactInstance{
							EquippedOn: idMap.Remap(si.EquippedOn),
						}
					}
					inv.OwnedArtifacts[defID] = instances
				}
				ownerEntity.AddComponent(gear.ArtifactInventoryComponent, inv)
			}
		}
	}

	// Attach equipment to squad entities
	for _, se := range pendingData.Equipment {
		newSquadID := idMap.Remap(se.SquadEntityID)
		if newSquadID == 0 {
			continue
		}
		squadEntity := em.FindEntityByID(newSquadID)
		if squadEntity == nil {
			continue
		}
		artifacts := make([]string, len(se.EquippedArtifacts))
		copy(artifacts, se.EquippedArtifacts)
		squadEntity.AddComponent(gear.EquipmentComponent, &gear.EquipmentData{
			EquippedArtifacts: artifacts,
		})
	}

	return nil
}
