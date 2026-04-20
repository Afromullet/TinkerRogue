package guisquads

import (
	"fmt"
	"game_main/core/common"
	"game_main/gui/builders"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

// SquadListConfig provides configuration for creating squad selection lists
type SquadListConfig struct {
	SquadIDs      []ecs.EntityID
	Manager       *common.EntityManager
	OnSelect      func(squadID ecs.EntityID)
	ScreenWidth   int
	ScreenHeight  int
	WidthPercent  float64 // Default 0.3
	HeightPercent float64 // Default 0.3
}

// CreateSquadList creates a list widget for selecting squads
func CreateSquadList(config SquadListConfig) *widget.List {
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.3
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.3
	}

	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	entries := make([]interface{}, 0, len(config.SquadIDs))
	for _, squadID := range config.SquadIDs {
		squadName := squadcore.GetSquadName(squadID, config.Manager)
		entries = append(entries, squadName)
	}

	return builders.CreateListWithConfig(builders.ListConfig{
		Entries:   entries,
		MinWidth:  listWidth,
		MinHeight: listHeight,
		EntryLabelFunc: func(e interface{}) string {
			return e.(string)
		},
		OnEntrySelected: func(e interface{}) {
			if config.OnSelect != nil {
				selectedName := e.(string)
				for i, squadID := range config.SquadIDs {
					if i < len(entries) && entries[i].(string) == selectedName {
						config.OnSelect(squadID)
						return
					}
				}
			}
		},
	})
}

// UnitListConfig provides configuration for creating unit lists with health display
type UnitListConfig struct {
	UnitIDs       []ecs.EntityID
	Manager       *common.EntityManager
	ScreenWidth   int
	ScreenHeight  int
	WidthPercent  float64 // Default 0.4
	HeightPercent float64 // Default 0.5
}

// CreateUnitList creates a list widget for displaying units with health info
func CreateUnitList(config UnitListConfig) *widget.List {
	if config.WidthPercent == 0 {
		config.WidthPercent = 0.4
	}
	if config.HeightPercent == 0 {
		config.HeightPercent = 0.5
	}

	listWidth := int(float64(config.ScreenWidth) * config.WidthPercent)
	listHeight := int(float64(config.ScreenHeight) * config.HeightPercent)

	entries := make([]interface{}, 0, len(config.UnitIDs))
	for _, unitID := range config.UnitIDs {
		if attrRaw, ok := config.Manager.GetComponent(unitID, common.AttributeComponent); ok {
			attr := attrRaw.(*common.Attributes)
			nameStr := common.GetEntityName(config.Manager, unitID, "Unknown")
			entries = append(entries, combattypes.UnitIdentity{
				ID:        unitID,
				Name:      nameStr,
				CurrentHP: attr.CurrentHealth,
				MaxHP:     attr.GetMaxHealth(),
				IsLeader:  config.Manager.HasComponent(unitID, squadcore.LeaderComponent),
			})
		}
	}

	return builders.CreateListWithConfig(builders.ListConfig{
		Entries: entries,
		EntryLabelFunc: func(e interface{}) string {
			identity := e.(combattypes.UnitIdentity)
			prefix := ""
			if identity.IsLeader {
				prefix = "(L) "
			}
			return fmt.Sprintf("%s%s - HP: %d/%d", prefix, identity.Name, identity.CurrentHP, identity.MaxHP)
		},
		MinWidth:  listWidth,
		MinHeight: listHeight,
	})
}
