package squads

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/templates"
	"game_main/visual/rendering"
	"log"
	"path/filepath"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// UnitTemplate defines a unit to be created in a squad
type UnitTemplate struct {
	UnitType     string
	Attributes   common.Attributes
	EntityType   templates.EntityType
	EntityConfig templates.EntityConfig
	EntityData   any      // JSONMonster, etc.
	GridRow      int      // Anchor row (0-2)
	GridCol      int      // Anchor col (0-2)
	GridWidth    int      // Width in cells (1-3), defaults to 1
	GridHeight   int      // Height in cells (1-3), defaults to 1
	Role         UnitRole // Tank, DPS, Support

	// Targeting fields
	AttackType  AttackType // MeleeRow, MeleeColumn, Ranged, or Magic
	TargetCells [][2]int   // For magic: pattern cells

	IsLeader       bool           // Squad leader flag
	CoverValue     float64        // Damage reduction provided (0.0-1.0, 0 = no cover)
	CoverRange     int            // Rows behind that receive cover (1-3)
	RequiresActive bool           // If true, dead/stunned units don't provide cover
	AttackRange    int            // World-based attack range (Melee=1, Ranged=3, Magic=4)
	MovementSpeed  int            // Movement speed on world map (1 tile per speed point)
	StatGrowths    StatGrowthData // Per-stat growth rates for leveling
}

// Creates the Unit entities used in the Squad
func CreateUnitTemplates(monsterData templates.JSONMonster) (UnitTemplate, error) {
	// Validate name
	if monsterData.UnitType == "" {
		return UnitTemplate{}, fmt.Errorf("unit type cannot be empty")
	}

	// Validate grid dimensions
	if monsterData.Width < 1 || monsterData.Width > 3 {
		return UnitTemplate{}, fmt.Errorf("unit width must be 1-3, got %d for %s", monsterData.Width, monsterData.UnitType)
	}

	if monsterData.Height < 1 || monsterData.Height > 3 {
		return UnitTemplate{}, fmt.Errorf("unit height must be 1-3, got %d for %s", monsterData.Height, monsterData.UnitType)
	}

	// Validate role
	role, err := GetRole(monsterData.Role)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid role for %s: %w", monsterData.UnitType, err)
	}

	// Convert attack type string to enum (with fallback to attackRange)
	attackType, err := GetAttackType(monsterData.AttackType, monsterData.AttackRange)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid attack type for %s: %w", monsterData.UnitType, err)
	}

	// Create entity configuration for the unit
	entityConfig := templates.EntityConfig{
		Type:      templates.EntityCreature,
		Name:      monsterData.UnitType,
		ImagePath: monsterData.ImageName,
		AssetDir:  config.AssetPath("creatures"),
		Visible:   true,
		Position:  nil, // Position will be set when squad is created
		GameMap:   nil, // GameMap will be set when squad is placed
	}

	// Parse stat growth grades from JSON
	growths := StatGrowthData{
		Strength:   GrowthGrade(monsterData.StatGrowths.Strength),
		Dexterity:  GrowthGrade(monsterData.StatGrowths.Dexterity),
		Magic:      GrowthGrade(monsterData.StatGrowths.Magic),
		Leadership: GrowthGrade(monsterData.StatGrowths.Leadership),
		Armor:      GrowthGrade(monsterData.StatGrowths.Armor),
		Weapon:     GrowthGrade(monsterData.StatGrowths.Weapon),
	}

	unit := UnitTemplate{
		UnitType:       monsterData.UnitType,
		Attributes:     monsterData.Attributes.NewAttributesFromJson(),
		EntityType:     templates.EntityCreature,
		EntityConfig:   entityConfig,
		EntityData:     monsterData,
		GridRow:        0,
		GridCol:        0,
		GridWidth:      monsterData.Width,
		GridHeight:     monsterData.Height,
		Role:           role,
		AttackType:     attackType,
		TargetCells:    monsterData.TargetCells,
		IsLeader:       false,
		CoverValue:     monsterData.CoverValue,
		CoverRange:     monsterData.CoverRange,
		RequiresActive: monsterData.RequiresActive,
		AttackRange:    monsterData.AttackRange,
		MovementSpeed:  monsterData.MovementSpeed,
		StatGrowths:    growths,
	}

	return unit, nil
}

// Reads the JSON file to create the UnitTemplates from which Unit entities can be created
func InitUnitTemplatesFromJSON() error {
	for _, monster := range templates.MonsterTemplates {
		unit, err := CreateUnitTemplates(monster)
		if err != nil {
			return fmt.Errorf("failed to create unit from %s: %w", monster.UnitType, err)
		}
		Units = append(Units, unit)
	}
	return nil
}

// GetRole converts a role string from a JSON file to a UnitRole enum value.
// It returns an error if the role string is not recognized.
func GetRole(roleString string) (UnitRole, error) {
	switch roleString {
	case "Tank":
		return RoleTank, nil
	case "DPS":
		return RoleDPS, nil
	case "Support":
		return RoleSupport, nil
	default:
		return 0, fmt.Errorf("invalid role: %q, expected Tank, DPS, or Support", roleString)
	}
}

// GetAttackType converts an attack type string from JSON to an AttackType enum value.
// If attackTypeString is empty, falls back to attackRange for backward compatibility.
// Returns an error if neither can determine a valid attack type.
func GetAttackType(attackTypeString string, attackRange int) (AttackType, error) {
	// Try to parse explicit attackType first
	if attackTypeString != "" {
		switch attackTypeString {
		case "MeleeRow":
			return AttackTypeMeleeRow, nil
		case "MeleeColumn":
			return AttackTypeMeleeColumn, nil
		case "Ranged":
			return AttackTypeRanged, nil
		case "Magic":
			return AttackTypeMagic, nil
		default:
			return 0, fmt.Errorf("invalid attackType: %q, expected MeleeRow, MeleeColumn, Ranged, or Magic", attackTypeString)
		}
	}

	// Fallback to attackRange for backward compatibility
	switch attackRange {
	case 0:
		return AttackTypeMeleeRow, nil // Default for test units / units without specified attack
	case 1:
		return AttackTypeMeleeRow, nil // Default melee to row attack
	case 3:
		return AttackTypeRanged, nil
	case 4:
		return AttackTypeMagic, nil
	default:
		return 0, fmt.Errorf("cannot determine attack type: attackType is empty and attackRange %d is invalid", attackRange)
	}
}

// GetTemplateByUnitType finds a unit template by its unit type.
// Returns nil if no template with the given unit type is found.
func GetTemplateByUnitType(unitType string) *UnitTemplate {
	for i := range Units {
		if Units[i].UnitType == unitType {
			return &Units[i]
		}
	}
	return nil
}

// Uses the UnitTemlate to create the unit entity and add it to the manager.
// This does not add the SquadMemberDat
func CreateUnitEntity(squadmanager *common.EntityManager, unit UnitTemplate) (*ecs.Entity, error) {

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
	// This allows units in squads to be visually rendered alongside squad highlights
	if unit.EntityConfig.ImagePath != "" {
		imagePath := filepath.Join(unit.EntityConfig.AssetDir, unit.EntityConfig.ImagePath)
		img, _, err := ebitenutil.NewImageFromFile(imagePath)
		if err != nil {
			// Log warning but continue - unit will exist but won't render visually
			log.Printf("Warning: Could not load image for unit %s at %s: %v\n", unit.UnitType, imagePath, err)
		} else {
			// Add renderable component with the loaded image
			unitEntity.AddComponent(rendering.RenderableComponent, &rendering.Renderable{
				Image:   img,
				Visible: true,
			})
		}
	}

	// Add unit type component for roster grouping (preserves original type)
	unitEntity.AddComponent(UnitTypeComponent, &UnitTypeData{
		UnitType: unit.UnitType,
	})

	// Add squad-specific components
	unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: 0,
		AnchorCol: 0,
		Width:     unit.GridWidth,
		Height:    unit.GridHeight,
	})

	unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
		Role: unit.Role,
	})

	// Add targeting component
	unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
		AttackType:  unit.AttackType,
		TargetCells: unit.TargetCells,
	})

	// Add cover component if the unit provides cover (CoverValue > 0)
	if unit.CoverValue > 0 {
		unitEntity.AddComponent(CoverComponent, &CoverData{
			CoverValue:     unit.CoverValue,
			CoverRange:     unit.CoverRange,
			RequiresActive: unit.RequiresActive,
		})
	}

	// Add attack range component
	unitEntity.AddComponent(AttackRangeComponent, &AttackRangeData{
		Range: unit.AttackRange,
	})

	// Add movement speed component
	unitEntity.AddComponent(MovementSpeedComponent, &MovementSpeedData{
		Speed: unit.MovementSpeed,
	})

	// Add experience component (all units start at level 1 with 0 XP)
	unitEntity.AddComponent(ExperienceComponent, &ExperienceData{
		Level:         1,
		CurrentXP:     0,
		XPToNextLevel: 100,
	})

	// Add stat growth component
	unitEntity.AddComponent(StatGrowthComponent, &StatGrowthData{
		Strength:   unit.StatGrowths.Strength,
		Dexterity:  unit.StatGrowths.Dexterity,
		Magic:      unit.StatGrowths.Magic,
		Leadership: unit.StatGrowths.Leadership,
		Armor:      unit.StatGrowths.Armor,
		Weapon:     unit.StatGrowths.Weapon,
	})

	return unitEntity, nil

}
