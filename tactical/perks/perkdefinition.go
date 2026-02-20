package perks

// PerkLevel determines what entity type a perk can be equipped on.
type PerkLevel int

const (
	PerkLevelSquad     PerkLevel = iota // Equipped on squad entity
	PerkLevelUnit                       // Equipped on unit entity
	PerkLevelCommander                  // Equipped on commander entity
)

// PerkCategory classifies a perk's design intent.
type PerkCategory int

const (
	CategorySpecialization PerkCategory = iota
	CategoryGeneralization
	CategoryAttackPattern
	CategoryAttribute
	CategoryAttackCounter
	CategoryDepth
	CategoryCommander
)

// PerkDefinition is a static blueprint for a perk loaded from JSON.
// Analogous to SpellDefinition in templates/spelldefinitions.go.
type PerkDefinition struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Level         PerkLevel         `json:"level"`
	Category      PerkCategory      `json:"category"`
	RoleGate      string            `json:"roleGate"`      // "" = any, "Tank", "DPS", "Support"
	ExclusiveWith []string          `json:"exclusiveWith"`  // Mutually exclusive perk IDs
	UnlockCost    int               `json:"unlockCost"`

	// Stat modification (Tier 1 perks)
	StatModifiers []PerkStatModifier `json:"statModifiers,omitempty"`

	// Behavioral hook key (Tier 2-4 perks)
	BehaviorID string                 `json:"behaviorId,omitempty"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

// PerkStatModifier defines one stat change a perk applies.
type PerkStatModifier struct {
	Stat     string  `json:"stat"`               // "strength", "dexterity", etc.
	Modifier int     `json:"modifier,omitempty"`  // Flat modifier value
	Percent  float64 `json:"percent,omitempty"`   // Percentage of base stat (0.2 = +20%)
}

// HasStatModifiers returns true if this perk has stat-based effects.
func (pd *PerkDefinition) HasStatModifiers() bool {
	return len(pd.StatModifiers) > 0
}

// HasBehavior returns true if this perk has behavioral hooks.
func (pd *PerkDefinition) HasBehavior() bool {
	return pd.BehaviorID != ""
}
