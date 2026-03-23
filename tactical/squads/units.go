package squads

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/unitdefs"
	"game_main/tactical/unitprogression"
	"game_main/templates"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// CreateUnitEntity creates a unit entity from a UnitTemplate and adds it to the manager.
// This does not add the SquadMemberData component.
func CreateUnitEntity(squadmanager *common.EntityManager, unit unitdefs.UnitTemplate) (*ecs.Entity, error) {

	// Validate grid dimensions
	if unit.GridWidth < 1 || unit.GridWidth > 3 {
		return nil, fmt.Errorf("invalid grid width %d for unit %s: must be 1-3", unit.GridWidth, unit.UnitType)
	}

	if unit.GridHeight < 1 || unit.GridHeight > 3 {
		return nil, fmt.Errorf("invalid grid height %d for unit %s: must be 1-3", unit.GridHeight, unit.UnitType)
	}

	// Generate a unique display name for this unit
	displayName := templates.GenerateName("default", unit.UnitType)

	// Create base unit entity via entitytemplates (delegates base entity creation)
	unitEntity := templates.CreateUnit(
		*squadmanager,
		displayName,
		unit.Attributes,
		nil, // Position defaults to 0,0
	)

	if unitEntity == nil {
		return nil, fmt.Errorf("failed to create entity for unit %s", unit.UnitType)
	}

	// Add RenderableComponent with unit's sprite image for display on map
	if unit.EntityConfig.ImagePath != "" {
		imagePath := filepath.Join(unit.EntityConfig.AssetDir, unit.EntityConfig.ImagePath)
		img, _, err := ebitenutil.NewImageFromFile(imagePath)
		if err != nil {
			log.Printf("Warning: Could not load image for unit %s at %s: %v\n", unit.UnitType, imagePath, err)
		} else {
			unitEntity.AddComponent(common.RenderableComponent, &common.Renderable{
				Image:   img,
				Visible: true,
			})
		}
	}

	// Add all squad-specific components from template
	ApplyUnitComponents(unitEntity, unit, 0, 0)

	return unitEntity, nil
}

// ApplyUnitComponents adds all squad-specific components to an entity from a UnitTemplate.
// anchorRow/anchorCol set the initial grid position (0,0 for roster units, actual position for bulk creation).
// This is the single source of truth for unit component composition.
func ApplyUnitComponents(entity *ecs.Entity, template unitdefs.UnitTemplate, anchorRow, anchorCol int) {
	width := template.GridWidth
	if width == 0 {
		width = 1
	}
	height := template.GridHeight
	if height == 0 {
		height = 1
	}

	entity.AddComponent(UnitTypeComponent, &UnitTypeData{
		UnitType: template.UnitType,
	})

	entity.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: anchorRow,
		AnchorCol: anchorCol,
		Width:     width,
		Height:    height,
	})

	entity.AddComponent(UnitRoleComponent, &UnitRoleData{
		Role: template.Role,
	})

	entity.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  template.AttackType,
		TargetCells: template.TargetCells,
	})

	if template.CoverValue > 0 {
		entity.AddComponent(CoverComponent, &CoverData{
			CoverValue:     template.CoverValue,
			CoverRange:     template.CoverRange,
			RequiresActive: template.RequiresActive,
		})
	}

	entity.AddComponent(AttackRangeComponent, &AttackRangeData{
		Range: template.AttackRange,
	})

	entity.AddComponent(MovementSpeedComponent, &MovementSpeedData{
		Speed: template.MovementSpeed,
	})

	entity.AddComponent(unitprogression.ExperienceComponent, &unitprogression.ExperienceData{
		Level:         1,
		CurrentXP:     0,
		XPToNextLevel: 100,
	})

	entity.AddComponent(unitprogression.StatGrowthComponent, &unitprogression.StatGrowthData{
		Strength:   template.StatGrowths.Strength,
		Dexterity:  template.StatGrowths.Dexterity,
		Magic:      template.StatGrowths.Magic,
		Leadership: template.StatGrowths.Leadership,
		Armor:      template.StatGrowths.Armor,
		Weapon:     template.StatGrowths.Weapon,
	})
}
