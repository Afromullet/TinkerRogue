package core

import (
	"fmt"
	"image/color"

	"game_main/campaign/overworld/ids"
	"game_main/templates"
)

// ThreatTypeParams defines behavior per threat type.
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	CanSpawnChildren bool
}

// NodeDefinition is the runtime representation of an overworld node.
// This is the new format that separates node properties from encounter properties.
type NodeDefinition struct {
	ID          ids.NodeTypeID // The node type ID (e.g., "necromancer")
	Category    NodeCategory
	DisplayName string

	// Rendering
	Color color.RGBA

	// Overworld behavior
	BaseGrowthRate   float64
	BaseRadius       int
	CanSpawnChildren bool

	// Faction (for threat nodes)
	FactionID ids.FactionID

	// Resource cost to place this node
	Cost ResourceCost
}

// EncounterDefinition is the runtime representation of combat mechanics.
// This is separate from node properties.
type EncounterDefinition struct {
	ID                ids.EncounterID
	EncounterTypeID   ids.EncounterTypeID
	EncounterTypeName string
	SquadPreferences  []string
	DefaultDifficulty int
	Tags              []string

	// Faction
	FactionID ids.FactionID
}

// NodeRegistry provides lookups for node and encounter definitions.
// This is the new unified registry that supports multiple node types.
type NodeRegistry struct {
	nodesByID        map[ids.NodeTypeID]*NodeDefinition  // "necromancer" -> def
	encountersByID   map[ids.EncounterID]*EncounterDefinition
	defaultNode      *NodeDefinition
	defaultEncounter *EncounterDefinition
	initialized      bool
}

// Global node registry instance
var globalNodeRegistry *NodeRegistry

// GetNodeRegistry returns the global node registry.
// The registry is initialized lazily on first access.
func GetNodeRegistry() *NodeRegistry {
	if globalNodeRegistry == nil || !globalNodeRegistry.initialized {
		globalNodeRegistry = newNodeRegistry()
	}
	return globalNodeRegistry
}

// newNodeRegistry creates and initializes a new node registry from JSON templates.
func newNodeRegistry() *NodeRegistry {
	registry := &NodeRegistry{
		nodesByID:      make(map[ids.NodeTypeID]*NodeDefinition),
		encountersByID: make(map[ids.EncounterID]*EncounterDefinition),
	}

	// Load node definitions from JSON templates
	for _, jsonDef := range templates.NodeDefinitionTemplates {
		def := &NodeDefinition{
			ID:          jsonDef.ID, // typed NodeTypeID from JSON DTO
			Category:    NodeCategory(jsonDef.Category),
			DisplayName: jsonDef.DisplayName,
			Color: color.RGBA{
				R: jsonDef.Color.R,
				G: jsonDef.Color.G,
				B: jsonDef.Color.B,
				A: jsonDef.Color.A,
			},
			BaseGrowthRate:   jsonDef.Overworld.BaseGrowthRate,
			BaseRadius:       jsonDef.Overworld.BaseRadius,
			CanSpawnChildren: jsonDef.Overworld.CanSpawnChildren,
			FactionID:        jsonDef.FactionID,
		}

		// Copy resource cost from JSON if present
		if jsonDef.Cost != nil {
			def.Cost = ResourceCost{
				Iron:  jsonDef.Cost.Iron,
				Wood:  jsonDef.Cost.Wood,
				Stone: jsonDef.Cost.Stone,
			}
		}

		// Register by ID (ID is the ThreatType)
		registry.nodesByID[def.ID] = def
	}

	// Load default node from JSON
	if templates.DefaultNodeTemplate != nil {
		registry.defaultNode = &NodeDefinition{
			ID:          "default",
			Category:    "threat",
			DisplayName: templates.DefaultNodeTemplate.DisplayName,
			Color: color.RGBA{
				R: templates.DefaultNodeTemplate.Color.R,
				G: templates.DefaultNodeTemplate.Color.G,
				B: templates.DefaultNodeTemplate.Color.B,
				A: templates.DefaultNodeTemplate.Color.A,
			},
			BaseGrowthRate:   templates.DefaultNodeTemplate.Overworld.BaseGrowthRate,
			BaseRadius:       templates.DefaultNodeTemplate.Overworld.BaseRadius,
			CanSpawnChildren: false,
		}
	}

	// Load encounter definitions from JSON templates
	for _, jsonEnc := range templates.EncounterDefinitionTemplates {
		enc := &EncounterDefinition{
			ID:                jsonEnc.ID,
			EncounterTypeID:   jsonEnc.EncounterTypeID,
			EncounterTypeName: jsonEnc.EncounterTypeName,
			SquadPreferences:  jsonEnc.SquadPreferences,
			DefaultDifficulty: jsonEnc.DefaultDifficulty,
			Tags:              jsonEnc.Tags,
			FactionID:         jsonEnc.FactionID,
		}

		registry.encountersByID[enc.ID] = enc
	}

	// Set bare default encounter as fallback
	registry.defaultEncounter = &EncounterDefinition{
		ID: "default",
	}

	registry.initialized = true
	return registry
}

// --- Node Lookup Methods ---

// GetNodeByID returns a node definition by its NodeTypeID.
// Returns default if not found.
func (r *NodeRegistry) GetNodeByID(id ids.NodeTypeID) *NodeDefinition {
	if def, ok := r.nodesByID[id]; ok {
		return def
	}
	fmt.Printf("WARNING: NodeRegistry: no node definition for %q, using default\n", id)
	return r.defaultNode
}

// GetNodeByType returns a node definition by its ThreatType.
// Since ThreatType is now string-based, this is the same as GetNodeByID.
// Kept for API compatibility.
func (r *NodeRegistry) GetNodeByType(threatType ThreatType) *NodeDefinition {
	return r.GetNodeByID(ids.NodeTypeID(threatType))
}

// GetPlaceableNodeTypes returns all settlement and fortress nodes available for player placement.
func (r *NodeRegistry) GetPlaceableNodeTypes() []*NodeDefinition {
	var nodes []*NodeDefinition
	for _, node := range r.nodesByID {
		if node.Category == NodeCategorySettlement || node.Category == NodeCategoryFortress {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// --- Encounter Lookup Methods ---

// GetEncounterByID returns an encounter definition by its EncounterID.
// Returns default if not found.
func (r *NodeRegistry) GetEncounterByID(id ids.EncounterID) *EncounterDefinition {
	if enc, ok := r.encountersByID[id]; ok {
		return enc
	}
	fmt.Printf("WARNING: NodeRegistry: no encounter definition for %q, using default\n", id)
	return r.defaultEncounter
}

// GetEncounterForThreatType returns the encounter definition for a threat type.
func (r *NodeRegistry) GetEncounterForThreatType(threatType ThreatType) *EncounterDefinition {
	node := r.GetNodeByType(threatType)
	if node == nil || node.FactionID == "" {
		return r.defaultEncounter
	}
	encounters := r.GetEncountersByFaction(node.FactionID)
	if len(encounters) == 0 {
		return r.defaultEncounter
	}
	return encounters[0]
}

// GetEncountersByFaction returns all encounters for a specific faction.
// Supports multiple encounter types per faction (basic, elite, boss variants).
// Returns empty slice if no encounters found for the faction.
func (r *NodeRegistry) GetEncountersByFaction(factionID ids.FactionID) []*EncounterDefinition {
	var encounters []*EncounterDefinition
	for _, enc := range r.encountersByID {
		if enc.FactionID == factionID {
			encounters = append(encounters, enc)
		}
	}
	return encounters
}

// GetEncounterByTypeID returns an encounter definition by its EncounterTypeID field.
// Linear scan — returns first match or default.
func (r *NodeRegistry) GetEncounterByTypeID(encounterTypeID ids.EncounterTypeID) *EncounterDefinition {
	for _, enc := range r.encountersByID {
		if enc.EncounterTypeID == encounterTypeID {
			return enc
		}
	}
	fmt.Printf("WARNING: NodeRegistry: no encounter with type ID %q, using default\n", encounterTypeID)
	return r.defaultEncounter
}

// GetThreatTypeForFaction returns the ThreatType for a faction.
// Finds the first threat-category node whose FactionID matches.
func (r *NodeRegistry) GetThreatTypeForFaction(factionType FactionType) ThreatType {
	factionID := FactionIDFor(factionType)
	for _, node := range r.nodesByID {
		if node.Category == NodeCategoryThreat && node.FactionID == factionID {
			return ThreatType(node.ID)
		}
	}
	return ThreatBanditCamp // fallback
}

// --- Convenience Accessors (for backward compatibility) ---

// GetDisplayName returns the display name for a threat type.
func (r *NodeRegistry) GetDisplayName(threatType ThreatType) string {
	return r.GetNodeByType(threatType).DisplayName
}

// GetOverworldParams returns the overworld parameters for a threat type.
func (r *NodeRegistry) GetOverworldParams(threatType ThreatType) ThreatTypeParams {
	node := r.GetNodeByType(threatType)
	return ThreatTypeParams{
		BaseGrowthRate:   node.BaseGrowthRate,
		BaseRadius:       node.BaseRadius,
		CanSpawnChildren: node.CanSpawnChildren,
	}
}

// GetEncounterTypeID returns the encounter type ID for a threat type.
func (r *NodeRegistry) GetEncounterTypeID(threatType ThreatType) ids.EncounterTypeID {
	enc := r.GetEncounterForThreatType(threatType)
	if enc == nil {
		return ""
	}
	return enc.EncounterTypeID
}

// ValidateNodeRegistry checks that the node registry is internally consistent.
// Warns if threat nodes lack a matching encounter, or if encounters reference
// unknown factions. Should be called after the registry is initialized.
func ValidateNodeRegistry() {
	registry := GetNodeRegistry()

	if registry.defaultNode == nil {
		fmt.Println("WARNING: NodeRegistry: no default node definition loaded")
	}

	// Check that every threat node has at least one encounter via its faction
	for id, node := range registry.nodesByID {
		if node.Category != NodeCategoryThreat {
			continue
		}
		if node.FactionID == "" {
			fmt.Printf("WARNING: NodeRegistry: threat node %q has no FactionID\n", id)
			continue
		}
		encounters := registry.GetEncountersByFaction(node.FactionID)
		if len(encounters) == 0 {
			fmt.Printf("WARNING: NodeRegistry: threat node %q (faction %q) has no matching encounters\n", id, node.FactionID)
		}
	}

	// Check that every encounter references a valid faction with a node
	for id, enc := range registry.encountersByID {
		if enc.FactionID == "" {
			continue // Some encounters may be faction-agnostic
		}
		foundNode := false
		for _, node := range registry.nodesByID {
			if node.FactionID == enc.FactionID {
				foundNode = true
				break
			}
		}
		if !foundNode {
			fmt.Printf("WARNING: NodeRegistry: encounter %q references faction %q with no matching node\n", id, enc.FactionID)
		}
	}

	fmt.Printf("NodeRegistry validated: %d nodes, %d encounters\n",
		len(registry.nodesByID), len(registry.encountersByID))
}
