package gear

import (
	"game_main/common"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// GetEquipmentData returns the EquipmentData for a squad, or nil if none.
func GetEquipmentData(squadID ecs.EntityID, manager *common.EntityManager) *EquipmentData {
	return common.GetComponentTypeByID[*EquipmentData](manager, squadID, EquipmentComponent)
}

// HasMinorArtifact returns true if the squad has any minor-tier artifact equipped.
func HasMinorArtifact(squadID ecs.EntityID, manager *common.EntityManager) bool {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return false
	}
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil && def.Tier == TierMinor {
			return true
		}
	}
	return false
}

// HasMajorArtifact returns true if the squad has any major-tier artifact equipped.
func HasMajorArtifact(squadID ecs.EntityID, manager *common.EntityManager) bool {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return false
	}
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil && def.Tier == TierMajor {
			return true
		}
	}
	return false
}

// HasSpecificArtifactInFaction checks if any squad in the given list has a specific artifact equipped.
func HasSpecificArtifactInFaction(squadIDs []ecs.EntityID, artifactID string, manager *common.EntityManager) bool {
	for _, squadID := range squadIDs {
		data := GetEquipmentData(squadID, manager)
		if data == nil {
			continue
		}
		for _, equipped := range data.EquippedArtifacts {
			if equipped == artifactID {
				return true
			}
		}
	}
	return false
}

// GetArtifactDefinitions returns artifact definitions for all equipped artifacts on a squad.
func GetArtifactDefinitions(squadID ecs.EntityID, manager *common.EntityManager) []*templates.ArtifactDefinition {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return nil
	}
	var defs []*templates.ArtifactDefinition
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil {
			defs = append(defs, def)
		}
	}
	return defs
}

// HasArtifactBehavior returns true if any equipped artifact on the squad has the given behavior.
func HasArtifactBehavior(squadID ecs.EntityID, behavior string, manager *common.EntityManager) bool {
	data := GetEquipmentData(squadID, manager)
	if data == nil {
		return false
	}
	for _, id := range data.EquippedArtifacts {
		def := templates.GetArtifactDefinition(id)
		if def != nil && def.Behavior == behavior {
			return true
		}
	}
	return false
}

// HasBehaviorInFaction returns true if any squad in the list has an artifact with the given behavior.
func HasBehaviorInFaction(squadIDs []ecs.EntityID, behavior string, manager *common.EntityManager) bool {
	return GetFactionSquadWithBehavior(squadIDs, behavior, manager) != 0
}

// GetFactionSquadWithBehavior returns the first squad with the given artifact behavior, or 0.
func GetFactionSquadWithBehavior(squadIDs []ecs.EntityID, behavior string, manager *common.EntityManager) ecs.EntityID {
	for _, sid := range squadIDs {
		if HasArtifactBehavior(sid, behavior, manager) {
			return sid
		}
	}
	return 0
}
