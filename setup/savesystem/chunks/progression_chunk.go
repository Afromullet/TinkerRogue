package chunks

import (
	"encoding/json"
	"fmt"
	"game_main/core/common"
	"game_main/setup/savesystem"
	"game_main/tactical/commander"
	"game_main/tactical/powers/progression"

	"github.com/bytearena/ecs"
)

func init() {
	savesystem.RegisterChunk(&ProgressionChunk{})
}

// ProgressionChunk saves/loads every Commander's ProgressionData component:
// ArcanaPoints, SkillPoints, and the unlocked spell / perk libraries. Each
// commander entity exists independently via CommanderChunk, so component
// re-attachment happens in RemapIDs after commanders are rebuilt.
//
// Format bumped to v2 when progression moved from Player-scoped to
// Commander-scoped. Old v1 saves (single player-owned entry) are no longer
// readable.
type ProgressionChunk struct{}

func (c *ProgressionChunk) ChunkID() string   { return "progression" }
func (c *ProgressionChunk) ChunkVersion() int { return 2 }

type savedCommanderProgression struct {
	OwnerEntityID    ecs.EntityID `json:"ownerEntityID"`
	ArcanaPoints     int          `json:"arcanaPoints"`
	SkillPoints      int          `json:"skillPoints"`
	UnlockedSpellIDs []string     `json:"unlockedSpellIDs"`
	UnlockedPerkIDs  []string     `json:"unlockedPerkIDs"`
}

type savedProgressionChunk struct {
	Commanders []savedCommanderProgression `json:"commanders"`
}

// --- Save ---

func (c *ProgressionChunk) Save(em *common.EntityManager) (json.RawMessage, error) {
	entries := make([]savedCommanderProgression, 0)
	for _, result := range em.World.Query(commander.CommanderTag) {
		entity := result.Entity
		data := common.GetComponentType[*progression.ProgressionData](entity, progression.ProgressionComponent)
		if data == nil {
			continue
		}
		entries = append(entries, savedCommanderProgression{
			OwnerEntityID:    entity.GetID(),
			ArcanaPoints:     data.ArcanaPoints,
			SkillPoints:      data.SkillPoints,
			UnlockedSpellIDs: append([]string(nil), data.UnlockedSpellIDs...),
			UnlockedPerkIDs:  append([]string(nil), data.UnlockedPerkIDs...),
		})
	}
	if len(entries) == 0 {
		return nil, nil
	}
	return json.Marshal(savedProgressionChunk{Commanders: entries})
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

	for _, entry := range saved.Commanders {
		newOwnerID := idMap.Remap(entry.OwnerEntityID)
		if newOwnerID == 0 {
			continue
		}
		entity := em.FindEntityByID(newOwnerID)
		if entity == nil {
			continue
		}
		data := &progression.ProgressionData{
			ArcanaPoints:     entry.ArcanaPoints,
			SkillPoints:      entry.SkillPoints,
			UnlockedSpellIDs: append([]string(nil), entry.UnlockedSpellIDs...),
			UnlockedPerkIDs:  append([]string(nil), entry.UnlockedPerkIDs...),
		}
		entity.AddComponent(progression.ProgressionComponent, data)
	}
	return nil
}
