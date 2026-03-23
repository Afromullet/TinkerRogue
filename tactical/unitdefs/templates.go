package unitdefs

import (
	"fmt"
	"game_main/common"
	"game_main/config"
	"game_main/tactical/unitprogression"
	"game_main/templates"
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

	IsLeader       bool                       // Squad leader flag
	CoverValue     float64                    // Damage reduction provided (0.0-1.0, 0 = no cover)
	CoverRange     int                        // Rows behind that receive cover (1-3)
	RequiresActive bool                       // If true, dead/stunned units don't provide cover
	AttackRange    int                        // World-based attack range (Melee=1, Ranged=3, Magic=4)
	MovementSpeed  int                        // Movement speed on world map (1 tile per speed point)
	StatGrowths    unitprogression.StatGrowthData // Per-stat growth rates for leveling
}

// Units holds all loaded unit templates from JSON
var Units = make([]UnitTemplate, 0, len(templates.MonsterTemplates))

// CreateUnitTemplates creates a UnitTemplate from JSON monster data
func CreateUnitTemplates(monsterData templates.JSONMonster) (UnitTemplate, error) {
	if monsterData.UnitType == "" {
		return UnitTemplate{}, fmt.Errorf("unit type cannot be empty")
	}

	if monsterData.Width < 1 || monsterData.Width > 3 {
		return UnitTemplate{}, fmt.Errorf("unit width must be 1-3, got %d for %s", monsterData.Width, monsterData.UnitType)
	}

	if monsterData.Height < 1 || monsterData.Height > 3 {
		return UnitTemplate{}, fmt.Errorf("unit height must be 1-3, got %d for %s", monsterData.Height, monsterData.UnitType)
	}

	role, err := GetRole(monsterData.Role)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid role for %s: %w", monsterData.UnitType, err)
	}

	attackType, err := GetAttackType(monsterData.AttackType, monsterData.AttackRange)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid attack type for %s: %w", monsterData.UnitType, err)
	}

	entityConfig := templates.EntityConfig{
		Type:      templates.EntityCreature,
		Name:      monsterData.UnitType,
		ImagePath: monsterData.ImageName,
		AssetDir:  config.AssetPath("creatures"),
		Visible:   true,
		Position:  nil,
		GameMap:   nil,
	}

	growths := unitprogression.StatGrowthData{
		Strength:   unitprogression.GrowthGrade(monsterData.StatGrowths.Strength),
		Dexterity:  unitprogression.GrowthGrade(monsterData.StatGrowths.Dexterity),
		Magic:      unitprogression.GrowthGrade(monsterData.StatGrowths.Magic),
		Leadership: unitprogression.GrowthGrade(monsterData.StatGrowths.Leadership),
		Armor:      unitprogression.GrowthGrade(monsterData.StatGrowths.Armor),
		Weapon:     unitprogression.GrowthGrade(monsterData.StatGrowths.Weapon),
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

// InitUnitTemplatesFromJSON reads the JSON file to create UnitTemplates
func InitUnitTemplatesFromJSON() error {
	Units = Units[:0] // Reset to prevent duplicates on reload
	for _, monster := range templates.MonsterTemplates {
		unit, err := CreateUnitTemplates(monster)
		if err != nil {
			return fmt.Errorf("failed to create unit from %s: %w", monster.UnitType, err)
		}
		Units = append(Units, unit)
	}
	return nil
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
