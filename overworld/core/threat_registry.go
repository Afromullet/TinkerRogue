package core

import (
	"image/color"

	"game_main/templates"
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
type ThreatDefinition struct {
	ID          string
	DisplayName string
	EnumValue   ThreatType // -1 for JSON-only threats

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
// This is the single source of truth for all threat configuration.
type ThreatRegistry struct {
	byID          map[string]*ThreatDefinition     // "necromancer" -> def
	byEnum        map[ThreatType]*ThreatDefinition // ThreatNecromancer -> def
	defaultThreat *ThreatDefinition
	initialized   bool
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

// newThreatRegistry creates and initializes a new threat registry from JSON templates.
func newThreatRegistry() *ThreatRegistry {
	registry := &ThreatRegistry{
		byID:   make(map[string]*ThreatDefinition),
		byEnum: make(map[ThreatType]*ThreatDefinition),
	}

	// Load threat definitions from JSON templates
	for _, jsonDef := range templates.ThreatDefinitionTemplates {
		def := &ThreatDefinition{
			ID:                jsonDef.ID,
			DisplayName:       jsonDef.DisplayName,
			EnumValue:         ThreatType(jsonDef.EnumValue),
			EncounterTypeID:   jsonDef.Encounter.TypeID,
			EncounterTypeName: jsonDef.Encounter.TypeName,
			SquadPreferences:  jsonDef.Encounter.SquadPreferences,
			DefaultDifficulty: jsonDef.Encounter.DefaultDifficulty,
			Tags:              jsonDef.Encounter.Tags,
			Color: color.RGBA{
				R: jsonDef.Color.R,
				G: jsonDef.Color.G,
				B: jsonDef.Color.B,
				A: jsonDef.Color.A,
			},
			BaseGrowthRate:   jsonDef.Overworld.BaseGrowthRate,
			BaseRadius:       jsonDef.Overworld.BaseRadius,
			PrimaryEffect:    stringToInfluenceEffect(jsonDef.Overworld.PrimaryEffect),
			CanSpawnChildren: jsonDef.Overworld.CanSpawnChildren,
			BasicItems:       jsonDef.BasicDrops,
			HighTierItems:    jsonDef.HighTierDrops,
			FactionID:        jsonDef.FactionID,
		}

		// Register by ID
		registry.byID[def.ID] = def

		// Register by enum if valid (non-negative)
		if def.EnumValue >= 0 {
			registry.byEnum[def.EnumValue] = def
		}
	}

	// Load default threat from JSON (required data)
	if templates.DefaultThreatTemplate == nil {
		panic("DefaultThreatTemplate is required in encounterdata.json")
	}
	registry.defaultThreat = &ThreatDefinition{
		ID:          "default",
		DisplayName: templates.DefaultThreatTemplate.DisplayName,
		EnumValue:   -1,
		Color: color.RGBA{
			R: templates.DefaultThreatTemplate.Color.R,
			G: templates.DefaultThreatTemplate.Color.G,
			B: templates.DefaultThreatTemplate.Color.B,
			A: templates.DefaultThreatTemplate.Color.A,
		},
		BaseGrowthRate:   templates.DefaultThreatTemplate.Overworld.BaseGrowthRate,
		BaseRadius:       templates.DefaultThreatTemplate.Overworld.BaseRadius,
		PrimaryEffect:    stringToInfluenceEffect(templates.DefaultThreatTemplate.Overworld.PrimaryEffect),
		CanSpawnChildren: templates.DefaultThreatTemplate.Overworld.CanSpawnChildren,
		BasicItems:       templates.DefaultThreatTemplate.BasicDrops,
		HighTierItems:    templates.DefaultThreatTemplate.HighTierDrops,
	}

	registry.initialized = true
	return registry
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

// GetByEnum returns a threat definition by its ThreatType enum.
// Returns default if not found.
func (r *ThreatRegistry) GetByEnum(threatType ThreatType) *ThreatDefinition {
	if def, ok := r.byEnum[threatType]; ok {
		return def
	}
	return r.defaultThreat
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
// Returns default if not found.
func (r *ThreatRegistry) GetByFactionID(factionID string) *ThreatDefinition {
	for _, def := range r.byID {
		if def.FactionID == factionID {
			return def
		}
	}
	return r.defaultThreat
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

// GetThreatTypeForFaction returns the ThreatType enum for a faction type.
// This replaces MapFactionToThreatType().
func (r *ThreatRegistry) GetThreatTypeForFaction(factionType FactionType) ThreatType {
	factionID := factionType.String()
	def := r.GetByFactionID(factionID)
	if def.EnumValue >= 0 {
		return def.EnumValue
	}
	// Fallback to default mapping if no valid enum
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
