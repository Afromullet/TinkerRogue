package squads

import (
	"fmt"
	"game_main/common"
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

	// Create entity configuration for the unit
	entityConfig := entitytemplates.EntityConfig{
		Type:      entitytemplates.EntityCreature,
		Name:      monsterData.Name,
		ImagePath: monsterData.ImageName,
		AssetDir:  "../assets/creatures/",
		Visible:   true,
		Position:  nil, // Position will be set when squad is created
		GameMap:   nil, // GameMap will be set when squad is placed
	}

	unit := UnitTemplate{
		Name:           monsterData.Name,
		Attributes:     monsterData.Attributes.NewAttributesFromJson(),
		EntityType:     entitytemplates.EntityCreature,
		EntityConfig:   entityConfig,
		EntityData:     monsterData,
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

// GetTemplateByName finds a unit template by its name.
// Returns nil if no template with the given name is found.
func GetTemplateByName(name string) *UnitTemplate {
	for i := range Units {
		if Units[i].Name == name {
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
		return nil, fmt.Errorf("invalid grid width %d for unit %s: must be 1-3", unit.GridWidth, unit.Name)
	}

	if unit.GridHeight < 1 || unit.GridHeight > 3 {
		return nil, fmt.Errorf("invalid grid height %d for unit %s: must be 1-3", unit.GridHeight, unit.Name)
	}

	// Create base unit entity via entitytemplates (delegates base entity creation)
	// Units are data-only entities (no renderables/images)
	unitEntity := entitytemplates.CreateUnit(
		*squadmanager,
		unit.Name,
		unit.Attributes,
		nil, // Position defaults to 0,0
	)

	if unitEntity == nil {
		return nil, fmt.Errorf("failed to create entity for unit %s", unit.Name)
	}

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
