package perks

// PerkTier classifies perk implementation complexity.
type PerkTier int

const (
	PerkTierConditioning   PerkTier = iota // Tier 1: Simple conditionals
	PerkTierSpecialization                 // Tier 2: Event reactions, targeting, state tracking
)

func (t PerkTier) String() string {
	switch t {
	case PerkTierConditioning:
		return "Combat Conditioning"
	case PerkTierSpecialization:
		return "Combat Specialization"
	default:
		return "Unknown"
	}
}

// PerkCategory classifies the tactical purpose of a perk.
type PerkCategory int

const (
	CategoryOffense  PerkCategory = iota // Damage-oriented perks
	CategoryDefense                      // Damage reduction, cover perks
	CategoryTactical                     // Targeting, positioning perks
	CategoryReactive                     // Event-triggered perks
	CategoryDoctrine                     // Squad-wide behavioral changes
)

func (c PerkCategory) String() string {
	switch c {
	case CategoryOffense:
		return "Offense"
	case CategoryDefense:
		return "Defense"
	case CategoryTactical:
		return "Tactical"
	case CategoryReactive:
		return "Reactive"
	case CategoryDoctrine:
		return "Doctrine"
	default:
		return "Unknown"
	}
}

// PerkDefinition is a static blueprint loaded from JSON.
// Analogous to SpellDefinition in templates/spelldefinitions.go.
type PerkDefinition struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Tier          PerkTier       `json:"tier"`
	Category      PerkCategory   `json:"category"`
	Roles         []string       `json:"roles"`          // ["Tank"], ["DPS", "Support"], etc.
	ExclusiveWith []string       `json:"exclusiveWith"`  // Mutually exclusive perk IDs
	UnlockCost    int            `json:"unlockCost"`     // Perk points to unlock
	BehaviorID    string         `json:"behaviorId"`     // Key into hook registry
	Params        map[string]any `json:"params,omitempty"` // Per-behavior tuning parameters
}
