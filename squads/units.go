package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/entitytemplates"

	"github.com/bytearena/ecs"
)

// UnitTemplate defines a unit to be created in a squad
type UnitTemplate struct {
	Name           string
	Attributes     common.Attributes
	EntityType     entitytemplates.EntityType
	EntityConfig   entitytemplates.EntityConfig
	EntityData     any        // JSONMonster, etc.
	GridRow        int        // Anchor row (0-2)
	GridCol        int        // Anchor col (0-2)
	GridWidth      int        // Width in cells (1-3), defaults to 1
	GridHeight     int        // Height in cells (1-3), defaults to 1
	Role           UnitRole   // Tank, DPS, Support
	TargetMode     TargetMode // "row" or "cell"
	TargetRows     []int      // Which rows to attack (row-based)
	IsMultiTarget  bool       // AOE or single-target (row-based)
	MaxTargets     int        // Max targets per row (row-based)
	TargetCells    [][2]int   // Specific cells to target (cell-based)
	IsLeader       bool    // Squad leader flag
	CoverValue     float64 // Damage reduction provided (0.0-1.0, 0 = no cover)
	CoverRange     int     // Rows behind that receive cover (1-3)
	RequiresActive bool    // If true, dead/stunned units don't provide cover
	AttackRange    int     // World-based attack range (Melee=1, Ranged=3, Magic=4)
	MovementSpeed  int     // Movement speed on world map (1 tile per speed point)
}

// Creates the Unit entities used in the Squad
func CreateUnitTemplates(monsterData entitytemplates.JSONMonster) (UnitTemplate, error) {
	// Validate name
	if monsterData.Name == "" {
		return UnitTemplate{}, fmt.Errorf("unit name cannot be empty")
	}

	// Validate grid dimensions
	if monsterData.Width < 1 || monsterData.Width > 3 {
		return UnitTemplate{}, fmt.Errorf("unit width must be 1-3, got %d for %s", monsterData.Width, monsterData.Name)
	}

	if monsterData.Height < 1 || monsterData.Height > 3 {
		return UnitTemplate{}, fmt.Errorf("unit height must be 1-3, got %d for %s", monsterData.Height, monsterData.Name)
	}

	// Validate role
	role, err := GetRole(monsterData.Role)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid role for %s: %w", monsterData.Name, err)
	}

	// Validate role
	targetMode, err := GetTargetMode(monsterData.TargetMode)
	if err != nil {
		return UnitTemplate{}, fmt.Errorf("invalid targetmode for %s: %w", monsterData.Name, err)
	}

	unit := UnitTemplate{
		Name:           monsterData.Name,
		Attributes:     monsterData.Attributes.NewAttributesFromJson(),
		GridRow:        0,
		GridCol:        0,
		GridWidth:      monsterData.Width,
		GridHeight:     monsterData.Height,
		Role:           role,
		TargetMode:     targetMode,
		TargetRows:     monsterData.TargetRows,
		IsMultiTarget:  monsterData.IsMultiTarget,
		MaxTargets:     monsterData.MaxTargets,
		TargetCells:    monsterData.TargetCells,
		IsLeader:       false,
		CoverValue:     monsterData.CoverValue,
		CoverRange:     monsterData.CoverRange,
		RequiresActive: monsterData.RequiresActive,
		AttackRange:    monsterData.AttackRange,
		MovementSpeed:  monsterData.MovementSpeed,
	}

	return unit, nil
}

// Reads the JSON file to create the UnitTemplates from which Unit entities can be created
func InitUnitTemplatesFromJSON() error {
	for _, monster := range entitytemplates.MonsterTemplates {
		unit, err := CreateUnitTemplates(monster)
		if err != nil {
			return fmt.Errorf("failed to create unit from %s: %w", monster.Name, err)
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

// GetRole converts a role string to a UnitRole enum value.
// It returns an error if the role string is not recognized.
func GetTargetMode(targetModeString string) (TargetMode, error) {

	switch targetModeString {
	case "row":
		return TargetModeRowBased, nil
	case "cell":
		return TargetModeCellBased, nil

	default:
		return 0, fmt.Errorf("invalid targetmode: %q, expected row or Support", targetModeString)
	}
}

// Uses the UnitTemlate to create the unit entity and add it to the manager.
// This does not add the SquadMemberDat
func CreateUnitEntity(squadmanager *common.EntityManager, unit UnitTemplate) (*ecs.Entity, error) {

	// Validate grid dimensions
	if unit.GridWidth < 1 || unit.GridWidth > 3 {
		return nil, fmt.Errorf("invalid grid width %d for unit %s: must be 1-3", unit.GridWidth, unit.Name)
	}

	if unit.GridHeight < 1 || unit.GridHeight > 3 {
		return nil, fmt.Errorf("invalid grid height %d for unit %s: must be 1-3", unit.GridHeight, unit.Name)
	}

	unitEntity := squadmanager.World.NewEntity()

	if unitEntity == nil {
		return nil, fmt.Errorf("failed to create entity for unit %s", unit.Name)
	}

	unitEntity.AddComponent(GridPositionComponent, &GridPositionData{
		AnchorRow: 0,
		AnchorCol: 0,
		Width:     unit.GridWidth,
		Height:    unit.GridHeight,
	})

	unitEntity.AddComponent(UnitRoleComponent, &UnitRoleData{
		Role: unit.Role,
	})

	unitEntity.AddComponent(common.NameComponent, &common.Name{
		NameStr: unit.Name,
	})

	unitEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{
		X: 0,
		Y: 0,
	})

	unitEntity.AddComponent(common.AttributeComponent, &common.Attributes{
		Strength:      unit.Attributes.Strength,
		Dexterity:     unit.Attributes.Dexterity,
		Magic:         unit.Attributes.Magic,
		Leadership:    unit.Attributes.Leadership,
		Armor:         unit.Attributes.Armor,
		Weapon:        unit.Attributes.Weapon,
		MaxHealth:     unit.Attributes.GetMaxHealth(),
		CurrentHealth: unit.Attributes.GetMaxHealth(),
		CanAct:        true,
	})

	// Row-based targeting (simple)
	unitEntity.AddComponent(TargetRowComponent, &TargetRowData{
		Mode:          unit.TargetMode,
		TargetRows:    unit.TargetRows,
		IsMultiTarget: unit.IsMultiTarget,
		MaxTargets:    unit.MaxTargets,
		TargetCells:   nil, // Use cell-based mode for precise grid patterns
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

	return unitEntity, nil

}
