package templates

import "fmt"

// JSONInitialSetup is the root container for initial game setup configuration.
// Loaded from gamedata/initialsetup.json at startup; drives commander, squad,
// roster unit, and faction creation in setup/gamesetup.
type JSONInitialSetup struct {
	Commanders  []JSONCommanderSetup `json:"commanders"`
	RosterUnits JSONRosterUnitsSetup `json:"rosterUnits"`
	Factions    JSONFactionsSetup    `json:"factions"`
}

type JSONCommanderSetup struct {
	Name      string         `json:"name"`
	OffsetX   int            `json:"offsetX"`
	OffsetY   int            `json:"offsetY"`
	IsPrimary bool           `json:"isPrimary"`
	Squads    JSONSquadSetup `json:"squads"`
}

type JSONSquadSetup struct {
	Count      int      `json:"count"`
	NamePrefix string   `json:"namePrefix"`
	TypePool   []string `json:"typePool"`
}

type JSONRosterUnitsSetup struct {
	Count int `json:"count"`
}

type JSONFactionsSetup struct {
	StrengthMin       int                `json:"strengthMin"`
	StrengthMax       int                `json:"strengthMax"`
	FallbackPositions []JSONFactionPos   `json:"fallbackPositions"`
	Entries           []JSONFactionEntry `json:"entries"`
}

type JSONFactionPos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type JSONFactionEntry struct {
	Type string `json:"type"`
}

// validSquadTypeIDs are the squad creators available in setup/gamesetup/initial_squads.go.
// Keep in sync with the squadCreators map there.
var validSquadTypeIDs = map[string]bool{
	"balanced": true,
	"ranged":   true,
	"magic":    true,
	"mixed":    true,
	"cavalry":  true,
}

// validFactionTypeIDs are the faction type strings recognized by
// setup/gamesetup/initial_factions.go::factionTypeFromString. Kept here as
// strings (not enum imports) to avoid templates depending on campaign/overworld.
var validFactionTypeIDs = map[string]bool{
	"necromancers": true,
	"bandits":      true,
	"orcs":         true,
	"cultists":     true,
}

func validateInitialSetup(cfg *JSONInitialSetup) {
	if len(cfg.Commanders) == 0 {
		panic("initialsetup: at least one commander required")
	}

	primaryCount := 0
	for i, c := range cfg.Commanders {
		if c.Name == "" {
			panic(fmt.Sprintf("initialsetup: commander[%d] missing name", i))
		}
		if c.Squads.Count < 0 {
			panic(fmt.Sprintf("initialsetup: commander[%d] squads.count must be >= 0", i))
		}
		if c.Squads.Count > 0 && len(c.Squads.TypePool) == 0 {
			panic(fmt.Sprintf("initialsetup: commander[%d] squads.typePool empty but count > 0", i))
		}
		for _, t := range c.Squads.TypePool {
			if !validSquadTypeIDs[t] {
				panic(fmt.Sprintf("initialsetup: commander[%d] unknown squad type %q", i, t))
			}
		}
		if c.IsPrimary {
			primaryCount++
			if c.OffsetX != 0 || c.OffsetY != 0 {
				panic(fmt.Sprintf("initialsetup: primary commander %q must have offset (0,0)", c.Name))
			}
		}
	}
	if primaryCount != 1 {
		panic(fmt.Sprintf("initialsetup: exactly one commander must be marked isPrimary (found %d)", primaryCount))
	}

	if cfg.RosterUnits.Count < 0 {
		panic("initialsetup: rosterUnits.count must be >= 0")
	}

	f := cfg.Factions
	if f.StrengthMin < 1 {
		panic("initialsetup: factions.strengthMin must be >= 1")
	}
	if f.StrengthMax < f.StrengthMin {
		panic("initialsetup: factions.strengthMax must be >= strengthMin")
	}
	if len(f.FallbackPositions) < len(f.Entries) {
		panic(fmt.Sprintf("initialsetup: fallbackPositions (%d) must have at least one entry per faction (%d)",
			len(f.FallbackPositions), len(f.Entries)))
	}
	for i, e := range f.Entries {
		if !validFactionTypeIDs[e.Type] {
			panic(fmt.Sprintf("initialsetup: factions.entries[%d] unknown faction type %q", i, e.Type))
		}
	}
}
