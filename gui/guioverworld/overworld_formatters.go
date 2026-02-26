package guioverworld

import (
	"fmt"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// FormatThreatInfo returns formatted string for threat/node details.
// Uses GUIQueries to access node data and position.
func FormatThreatInfo(nodeID ecs.EntityID, queries *framework.GUIQueries, manager *common.EntityManager) string {
	if nodeID == 0 {
		return "Select a threat to view details"
	}

	data := queries.GetNodeData(nodeID)
	pos := queries.GetEntityPosition(nodeID)

	if data == nil {
		return "Invalid node"
	}

	nodeDef := core.GetNodeRegistry().GetNodeByID(data.NodeTypeID)
	displayName := data.NodeTypeID
	if nodeDef != nil {
		displayName = nodeDef.DisplayName
	}

	if data.Category == core.NodeCategoryThreat {
		containedStatus := ""
		if data.IsContained {
			containedStatus = " (CONTAINED)"
		}

		return fmt.Sprintf(
			"=== Threat Details ===\n"+
				"Type: %s%s\n"+
				"Owner: %s\n"+
				"Position: (%d, %d)\n"+
				"Intensity: %d / %d\n"+
				"Growth: %.1f%%\n"+
				"Age: %d ticks",
			displayName,
			containedStatus,
			data.OwnerID,
			pos.X, pos.Y,
			data.Intensity,
			core.GetMaxThreatIntensity(),
			data.GrowthProgress*100,
			data.CreatedTick,
		)
	}

	// Show garrison info for non-threat nodes
	garrisonInfo := ""
	garrisonData := garrison.GetGarrisonAtNode(manager, nodeID)
	if garrisonData != nil && len(garrisonData.SquadIDs) > 0 {
		garrisonInfo = fmt.Sprintf("\nGarrison: %d squad(s)", len(garrisonData.SquadIDs))
		for _, squadID := range garrisonData.SquadIDs {
			squadName := squads.GetSquadName(squadID, manager)
			garrisonInfo += fmt.Sprintf("\n  - %s", squadName)
		}
	} else {
		garrisonInfo = "\nGarrison: None"
	}

	return fmt.Sprintf(
		"=== Node Details ===\n"+
			"Type: %s (%s)\n"+
			"Owner: %s\n"+
			"Position: (%d, %d)%s",
		displayName,
		data.Category,
		data.OwnerID,
		pos.X, pos.Y,
		garrisonInfo,
	)
}
