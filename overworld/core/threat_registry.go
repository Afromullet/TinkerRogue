package core

import (
	"image/color"
)

// ThreatTypeParams defines behavior per threat type.
type ThreatTypeParams struct {
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool
}

// ThreatDefinition is the runtime representation of a threat type.
// It wraps the JSON definition and provides convenient accessors.
// This structure is maintained for backward compatibility.
type ThreatDefinition struct {
	ID          string // The threat type (e.g., "necromancer")
	DisplayName string

	// Encounter config
	EncounterTypeID   string
	EncounterTypeName string
	SquadPreferences  []string
	DefaultDifficulty int
	Tags              []string

	// Rendering
	Color color.RGBA

	// Overworld behavior
	BaseGrowthRate   float64
	BaseRadius       int
	PrimaryEffect    InfluenceEffect
	CanSpawnChildren bool

	// Item drops
	BasicItems    []string
	HighTierItems []string

	// Faction
	FactionID string
}

// ThreatRegistry provides lookups for threat definitions.
// This registry delegates to NodeRegistry for the new split data format,
// while maintaining backward compatibility with existing code.
type ThreatRegistry struct {
	// Internal caches built from NodeRegistry (ThreatType is now string-based)
	byID          map[string]*ThreatDefinition
	defaultThreat *ThreatDefinition
	initialized   bool

	// Reference to NodeRegistry for delegation
	nodeRegistry *NodeRegistry
}

// Global registry instance
var globalThreatRegistry *ThreatRegistry

// GetThreatRegistry returns the global threat registry.
// The registry is initialized lazily on first access.
func GetThreatRegistry() *ThreatRegistry {
	if globalThreatRegistry == nil || !globalThreatRegistry.initialized {
		globalThreatRegistry = newThreatRegistry()
	}
	return globalThreatRegistry
}

// newThreatRegistry creates and initializes a new threat registry from the NodeRegistry.
func newThreatRegistry() *ThreatRegistry {
	registry := &ThreatRegistry{
		byID: make(map[string]*ThreatDefinition),
	}

	registry.nodeRegistry = GetNodeRegistry()
	registry.initFromNodeRegistry()

	registry.initialized = true
	return registry
}

// initFromNodeRegistry populates the ThreatRegistry from the new NodeRegistry.
// This creates ThreatDefinition objects by combining Node and Encounter data.
func (r *ThreatRegistry) initFromNodeRegistry() {
	nodeReg := r.nodeRegistry

	// Build ThreatDefinition for each threat-category node
	for _, node := range nodeReg.GetNodesByCategory(NodeCategoryThreat) {
		enc := nodeReg.GetEncounterByID(node.EncounterID)

		def := &ThreatDefinition{
			ID:               node.ID, // ID is the ThreatType (string)
			DisplayName:      node.DisplayName,
			Color:            node.Color,
			BaseGrowthRate:   node.BaseGrowthRate,
			BaseRadius:       node.BaseRadius,
			PrimaryEffect:    node.PrimaryEffect,
			CanSpawnChildren: node.CanSpawnChildren,
		}

		// Fill in encounter data if available
		if enc != nil {
			def.EncounterTypeID = enc.EncounterTypeID
			def.EncounterTypeName = enc.EncounterTypeName
			def.SquadPreferences = enc.SquadPreferences
			def.DefaultDifficulty = enc.DefaultDifficulty
			def.Tags = enc.Tags
			def.BasicItems = enc.BasicItems
			def.HighTierItems = enc.HighTierItems
			def.FactionID = enc.FactionID
		}

		// Register by ID (ID is the ThreatType)
		r.byID[def.ID] = def
	}

	// Build default threat from node registry
	defaultNode := nodeReg.defaultNode
	defaultEnc := nodeReg.defaultEncounter

	if defaultNode != nil {
		r.defaultThreat = &ThreatDefinition{
			ID:               "default",
			DisplayName:      defaultNode.DisplayName,
			Color:            defaultNode.Color,
			BaseGrowthRate:   defaultNode.BaseGrowthRate,
			BaseRadius:       defaultNode.BaseRadius,
			PrimaryEffect:    defaultNode.PrimaryEffect,
			CanSpawnChildren: defaultNode.CanSpawnChildren,
		}

		if defaultEnc != nil {
			r.defaultThreat.BasicItems = defaultEnc.BasicItems
			r.defaultThreat.HighTierItems = defaultEnc.HighTierItems
		}
	}
}

// --- Lookup Methods ---

// GetByID returns a threat definition by its string ID.
// Returns default if not found.
func (r *ThreatRegistry) GetByID(id string) *ThreatDefinition {
	if def, ok := r.byID[id]; ok {
		return def
	}
	return r.defaultThreat
}

// GetByEnum returns a threat definition by its ThreatType.
// Since ThreatType is now string-based, this is the same as GetByID.
// Kept for API compatibility.
func (r *ThreatRegistry) GetByEnum(threatType ThreatType) *ThreatDefinition {
	return r.GetByID(string(threatType))
}

// GetByEncounterTypeID returns a threat definition by its encounter type ID.
// Returns default if not found.
func (r *ThreatRegistry) GetByEncounterTypeID(encounterTypeID string) *ThreatDefinition {
	for _, def := range r.byID {
		if def.EncounterTypeID == encounterTypeID {
			return def
		}
	}
	return r.defaultThreat
}

// GetByFactionID returns a threat definition by its faction ID.
// NOTE: If multiple threats exist for the same faction, this returns only the FIRST match.
// Use GetThreatsByFaction() to get all threats for a faction.
// Returns default if not found.
func (r *ThreatRegistry) GetByFactionID(factionID string) *ThreatDefinition {
	for _, def := range r.byID {
		if def.FactionID == factionID {
			return def
		}
	}
	return r.defaultThreat
}

// GetThreatsByFaction returns all threat definitions for a specific faction.
// Supports multiple threat types per faction (basic, elite, boss variants).
// Returns empty slice if no threats found for the faction.
func (r *ThreatRegistry) GetThreatsByFaction(factionID string) []*ThreatDefinition {
	var threats []*ThreatDefinition
	for _, def := range r.byID {
		if def.FactionID == factionID {
			threats = append(threats, def)
		}
	}
	return threats
}

// --- Convenience Accessors ---

// GetDisplayName returns the display name for a threat type.
func (r *ThreatRegistry) GetDisplayName(threatType ThreatType) string {
	return r.GetByEnum(threatType).DisplayName
}

// GetColor returns the display color for a threat type.
func (r *ThreatRegistry) GetColor(threatType ThreatType) color.RGBA {
	return r.GetByEnum(threatType).Color
}

// GetEncounterTypeID returns the encounter type ID for a threat type.
func (r *ThreatRegistry) GetEncounterTypeID(threatType ThreatType) string {
	return r.GetByEnum(threatType).EncounterTypeID
}

// GetSquadPreferences returns the squad preferences for a threat type.
func (r *ThreatRegistry) GetSquadPreferences(threatType ThreatType) []string {
	return r.GetByEnum(threatType).SquadPreferences
}

// GetItemDropTable returns the item drop tables for a threat type.
// Returns (basic items, high-tier items).
func (r *ThreatRegistry) GetItemDropTable(threatType ThreatType) (basic, highTier []string) {
	def := r.GetByEnum(threatType)
	return def.BasicItems, def.HighTierItems
}

// GetOverworldParams returns the overworld parameters for a threat type.
func (r *ThreatRegistry) GetOverworldParams(threatType ThreatType) ThreatTypeParams {
	def := r.GetByEnum(threatType)
	return ThreatTypeParams{
		BaseGrowthRate:   def.BaseGrowthRate,
		BaseRadius:       def.BaseRadius,
		PrimaryEffect:    def.PrimaryEffect,
		CanSpawnChildren: def.CanSpawnChildren,
	}
}

// GetThreatTypeForFaction returns the ThreatType for a faction type.
// This replaces MapFactionToThreatType().
func (r *ThreatRegistry) GetThreatTypeForFaction(factionType FactionType) ThreatType {
	factionID := factionType.String()
	def := r.GetByFactionID(factionID)
	if def != nil && def.ID != "" {
		return ThreatType(def.ID)
	}
	// Fallback to default threat type
	return ThreatBanditCamp
}

// GetAllDefinitions returns all registered threat definitions.
func (r *ThreatRegistry) GetAllDefinitions() []*ThreatDefinition {
	defs := make([]*ThreatDefinition, 0, len(r.byID))
	for _, def := range r.byID {
		defs = append(defs, def)
	}
	return defs
}

// HasThreat returns true if a threat with the given ID exists.
func (r *ThreatRegistry) HasThreat(id string) bool {
	_, ok := r.byID[id]
	return ok
}

// GetNodeRegistry returns the underlying NodeRegistry if using the new format.
// Returns nil if using legacy format.
func (r *ThreatRegistry) GetNodeRegistry() *NodeRegistry {
	return r.nodeRegistry
}
