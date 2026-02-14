package core

import (
	"image/color"

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
	ID          string // The node type ID (e.g., "necromancer")
	Category    NodeCategory
	DisplayName string

	// Rendering
	Color color.RGBA

	// Overworld behavior
	BaseGrowthRate   float64
	BaseRadius       int
	CanSpawnChildren bool

	// Faction (for threat nodes)
	FactionID string

	// Resource cost to place this node
	Cost ResourceCost
}

// EncounterDefinition is the runtime representation of combat mechanics.
// This is separate from node properties.
type EncounterDefinition struct {
	ID                string
	EncounterTypeID   string
	EncounterTypeName string
	SquadPreferences  []string
	DefaultDifficulty int
	Tags              []string

	// Item drops
	BasicItems    []string
	HighTierItems []string

	// Faction
	FactionID string
}

// NodeRegistry provides lookups for node and encounter definitions.
// This is the new unified registry that supports multiple node types.
type NodeRegistry struct {
	nodesByID        map[string]*NodeDefinition      // "necromancer" -> def (ThreatType is now string-based)
	encountersByID   map[string]*EncounterDefinition // "necromancer" -> def
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
		nodesByID:      make(map[string]*NodeDefinition),
		encountersByID: make(map[string]*EncounterDefinition),
	}

	// Load node definitions from JSON templates
	for _, jsonDef := range templates.NodeDefinitionTemplates {
		def := &NodeDefinition{
			ID:          jsonDef.ID, // This is the ThreatType (string)
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
			BasicItems:        jsonEnc.BasicDrops,
			HighTierItems:     jsonEnc.HighTierDrops,
			FactionID:         jsonEnc.FactionID,
		}

		registry.encountersByID[enc.ID] = enc
	}

	// Load default encounter from JSON
	if templates.DefaultEncounterTemplate != nil {
		registry.defaultEncounter = &EncounterDefinition{
			ID:            "default",
			BasicItems:    templates.DefaultEncounterTemplate.BasicDrops,
			HighTierItems: templates.DefaultEncounterTemplate.HighTierDrops,
		}
	}

	registry.initialized = true
	return registry
}

// --- Node Lookup Methods ---

// GetNodeByID returns a node definition by its string ID.
// Returns default if not found.
func (r *NodeRegistry) GetNodeByID(id string) *NodeDefinition {
	if def, ok := r.nodesByID[id]; ok {
		return def
	}
	return r.defaultNode
}

// GetNodeByType returns a node definition by its ThreatType.
// Since ThreatType is now string-based, this is the same as GetNodeByID.
// Kept for API compatibility.
func (r *NodeRegistry) GetNodeByType(threatType ThreatType) *NodeDefinition {
	return r.GetNodeByID(string(threatType))
}

// GetAllNodes returns all registered node definitions.
func (r *NodeRegistry) GetAllNodes() []*NodeDefinition {
	nodes := make([]*NodeDefinition, 0, len(r.nodesByID))
	for _, node := range r.nodesByID {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNodesByCategory returns all nodes of a specific category.
func (r *NodeRegistry) GetNodesByCategory(category NodeCategory) []*NodeDefinition {
	var nodes []*NodeDefinition
	for _, node := range r.nodesByID {
		if node.Category == category {
			nodes = append(nodes, node)
		}
	}
	return nodes
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

// HasNode returns true if a node with the given ID exists.
func (r *NodeRegistry) HasNode(id string) bool {
	_, ok := r.nodesByID[id]
	return ok
}

// --- Encounter Lookup Methods ---

// GetEncounterByID returns an encounter definition by its string ID.
// Returns default if not found.
func (r *NodeRegistry) GetEncounterByID(id string) *EncounterDefinition {
	if enc, ok := r.encountersByID[id]; ok {
		return enc
	}
	return r.defaultEncounter
}

// GetEncounterForNode returns an encounter definition linked to a node via faction.
// Returns nil if the node has no faction (non-combat node).
func (r *NodeRegistry) GetEncounterForNode(nodeID string) *EncounterDefinition {
	node := r.GetNodeByID(nodeID)
	if node == nil || node.FactionID == "" {
		return nil
	}
	encounters := r.GetEncountersByFaction(node.FactionID)
	if len(encounters) == 0 {
		return r.defaultEncounter
	}
	return encounters[0]
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
func (r *NodeRegistry) GetEncountersByFaction(factionID string) []*EncounterDefinition {
	var encounters []*EncounterDefinition
	for _, enc := range r.encountersByID {
		if enc.FactionID == factionID {
			encounters = append(encounters, enc)
		}
	}
	return encounters
}

// GetNodesByFaction returns all nodes belonging to a specific faction.
// Returns empty slice if no nodes found for the faction.
func (r *NodeRegistry) GetNodesByFaction(factionID string) []*NodeDefinition {
	var nodes []*NodeDefinition
	for _, node := range r.nodesByID {
		if node.FactionID == factionID {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetAllEncounters returns all registered encounter definitions.
func (r *NodeRegistry) GetAllEncounters() []*EncounterDefinition {
	encounters := make([]*EncounterDefinition, 0, len(r.encountersByID))
	for _, enc := range r.encountersByID {
		encounters = append(encounters, enc)
	}
	return encounters
}

// HasEncounter returns true if an encounter with the given ID exists.
func (r *NodeRegistry) HasEncounter(id string) bool {
	_, ok := r.encountersByID[id]
	return ok
}

// GetEncounterByTypeID returns an encounter definition by its EncounterTypeID field.
// Linear scan — returns first match or default.
func (r *NodeRegistry) GetEncounterByTypeID(encounterTypeID string) *EncounterDefinition {
	for _, enc := range r.encountersByID {
		if enc.EncounterTypeID == encounterTypeID {
			return enc
		}
	}
	return r.defaultEncounter
}

// GetThreatTypeForFaction returns the ThreatType for a faction.
// Finds the first threat-category node whose FactionID matches.
func (r *NodeRegistry) GetThreatTypeForFaction(factionType FactionType) ThreatType {
	factionID := factionType.String()
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

// GetColor returns the display color for a threat type.
func (r *NodeRegistry) GetColor(threatType ThreatType) color.RGBA {
	return r.GetNodeByType(threatType).Color
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
func (r *NodeRegistry) GetEncounterTypeID(threatType ThreatType) string {
	enc := r.GetEncounterForThreatType(threatType)
	if enc == nil {
		return ""
	}
	return enc.EncounterTypeID
}

// GetSquadPreferences returns the squad preferences for a threat type.
func (r *NodeRegistry) GetSquadPreferences(threatType ThreatType) []string {
	enc := r.GetEncounterForThreatType(threatType)
	if enc == nil {
		return nil
	}
	return enc.SquadPreferences
}

// GetItemDropTable returns the item drop tables for a threat type.
func (r *NodeRegistry) GetItemDropTable(threatType ThreatType) (basic, highTier []string) {
	enc := r.GetEncounterForThreatType(threatType)
	if enc == nil {
		return nil, nil
	}
	return enc.BasicItems, enc.HighTierItems
}
