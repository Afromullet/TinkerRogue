package gamesetup

import (
	"fmt"
	"log"
	"path/filepath"

	"game_main/common"
	"game_main/setup/config"
	"game_main/setup/savesystem"
	"game_main/setup/savesystem/chunks"
	"game_main/tactical/commander"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/world/coords"
	"game_main/world/worldmapcore"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ConfigureMapChunk sets the GameMap pointer on the MapChunk so it can
// read/write map data during save/load.
func ConfigureMapChunk(gm *worldmapcore.GameMap) {
	if chunk := savesystem.GetChunk("map"); chunk != nil {
		if mc, ok := chunk.(*chunks.MapChunk); ok {
			mc.GameMap = gm
		}
	}
}

// RestorePlayerData reconstructs the PlayerData struct from the loaded player entity.
func RestorePlayerData(em *common.EntityManager, pd *common.PlayerData) {
	playerTag, ok := em.WorldTags["players"]
	if !ok {
		return
	}
	results := em.World.Query(playerTag)
	if len(results) == 0 {
		return
	}
	entity := results[0].Entity
	pd.PlayerEntityID = entity.GetID()
	if pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); pos != nil {
		pd.Pos = pos
	}
}

// RestoreRenderables reconstructs all renderable components after loading a save.
// Images can't be serialized to JSON, so we rebuild them from saved metadata
// (player image path, unit type templates, squad leader sprites).
func RestoreRenderables(em *common.EntityManager) error {
	// Phase 1: Player entity
	if playerTag, ok := em.WorldTags["players"]; ok {
		for _, result := range em.World.Query(playerTag) {
			img, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
			if err != nil {
				return fmt.Errorf("failed to load player image: %w", err)
			}
			result.Entity.AddComponent(common.RenderableComponent, &common.Renderable{
				Image:   img,
				Visible: true,
			})
		}
	}

	// Phase 2: Commander entities
	for _, result := range em.World.Query(commander.CommanderTag) {
		img, _, err := ebitenutil.NewImageFromFile(config.PlayerImagePath)
		if err != nil {
			return fmt.Errorf("failed to load commander image: %w", err)
		}
		result.Entity.AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   img,
			Visible: true,
		})
	}

	// Phase 3: Unit members — look up template by UnitType for image path
	for _, result := range em.World.Query(squadcore.SquadMemberTag) {
		utData := common.GetComponentType[*squadcore.UnitTypeData](result.Entity, squadcore.UnitTypeComponent)
		if utData == nil {
			continue
		}
		template := unitdefs.GetTemplateByUnitType(utData.UnitType)
		if template == nil || template.EntityConfig.ImagePath == "" {
			log.Printf("Warning: no template or image for unit type %q, skipping renderable", utData.UnitType)
			continue
		}
		imagePath := filepath.Join(template.EntityConfig.AssetDir, template.EntityConfig.ImagePath)
		img, _, err := ebitenutil.NewImageFromFile(imagePath)
		if err != nil {
			log.Printf("Warning: could not load image for unit %s at %s: %v", utData.UnitType, imagePath, err)
			continue
		}
		result.Entity.AddComponent(common.RenderableComponent, &common.Renderable{
			Image:   img,
			Visible: false, // Units hidden on world map; squad entity renders instead
		})
	}

	// Phase 4: Squad entities — copy leader's sprite to squad
	for _, result := range em.World.Query(squadcore.SquadTag) {
		squadID := result.Entity.GetID()
		squadcore.SetSquadRenderableFromLeader(squadID, result.Entity, em)
	}

	return nil
}
