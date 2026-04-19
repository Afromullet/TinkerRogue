package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/common"
	"game_main/setup/savesystem"
	"game_main/tactical/powers/progression"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&ProgressionChunk{})
}

// ProgressionChunk saves/loads the per-player ProgressionData component:
// ArcanaPoints, SkillPoints, and the unlocked spell / perk libraries.
// The component is attached to a Player entity that exists independently via
// PlayerChunk, so attachment happens in RemapIDs after the player entity is rebuilt.
type ProgressionChunk struct{}

func (c *ProgressionChunk) ChunkID() string   { return "progression" }
func (c *ProgressionChunk) ChunkVersion() int { return 1 }

type savedProgressionChunk struct {
	OwnerEntityID    ecs.EntityID `json:"ownerEntityID"`
	ArcanaPoints     int          `json:"arcanaPoints"`
	SkillPoints      int          `json:"skillPoints"`
	UnlockedSpellIDs []string     `json:"unlockedSpellIDs"`
	UnlockedPerkIDs  []string     `json:"unlockedPerkIDs"`
}

// --- Save ---

func (c *ProgressionChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	playerTag, ok := em.WorldTags["players"]
	if !ok {
		return nil, nil
	}
	results := em.World.Query(playerTag)
	if len(results) == 0 {
		return nil, nil
	}
	entity := results[0].Entity
	data := common.GetComponentType[*progression.ProgressionData](entity, progression.ProgressionComponent)
	if data == nil {
		return nil, nil
	}
	saved := savedProgressionChunk{
		OwnerEntityID:    entity.GetID(),
		ArcanaPoints:     data.ArcanaPoints,
		SkillPoints:      data.SkillPoints,
		UnlockedSpellIDs: append([]string(nil), data.UnlockedSpellIDs...),
		UnlockedPerkIDs:  append([]string(nil), data.UnlockedPerkIDs...),
	}
	return json.Marshal(saved)
}

// --- Load ---

const progressionLoadContextKey = "progression_pending_data"

func (c *ProgressionChunk) Load(em *common.EntityManager, data json.RawMessage, idMap *savesystem.EntityIDMap) error {
	if len(data) == 0 {
		return nil
	}
	var saved savedProgressionChunk
	if err := json.Unmarshal(data, &saved); err != nil {
		return fmt.Errorf("failed to unmarshal progression data: %w", err)
	}
	idMap.LoadContext[progressionLoadContextKey] = &saved
	return nil
}

// --- RemapIDs ---

func (c *ProgressionChunk) RemapIDs(em *common.EntityManager, idMap *savesystem.EntityIDMap) error {
	raw, ok := idMap.LoadContext[progressionLoadContextKey]
	if !ok {
		return nil
	}
	saved := raw.(*savedProgressionChunk)
	delete(idMap.LoadContext, progressionLoadContextKey)

	newOwnerID := idMap.Remap(saved.OwnerEntityID)
	if newOwnerID == 0 {
		return nil
	}
	entity := em.FindEntityByID(newOwnerID)
	if entity == nil {
		return nil
	}
	data := &progression.ProgressionData{
		ArcanaPoints:     saved.ArcanaPoints,
		SkillPoints:      saved.SkillPoints,
		UnlockedSpellIDs: append([]string(nil), saved.UnlockedSpellIDs...),
		UnlockedPerkIDs:  append([]string(nil), saved.UnlockedPerkIDs...),
	}
	entity.AddComponent(progression.ProgressionComponent, data)
	return nil
}
