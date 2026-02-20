package raid

// ArchetypeUnit defines a single unit within a squad archetype.
type ArchetypeUnit struct {
	MonsterType string // Key into monsterdata.json (looked up via squads.GetTemplateByUnitType)
	GridRow     int
	GridCol     int
	GridWidth   int // Default 1
	GridHeight  int // Default 1
	IsLeader    bool
}

// SquadArchetype defines a pre-composed garrison squad template.
type SquadArchetype struct {
	Name          string
	DisplayName   string
	Units         []ArchetypeUnit
	PreferredRooms []string
}

// GarrisonArchetypes defines all garrison squad compositions.
var GarrisonArchetypes = []SquadArchetype{
	{
		Name:        "chokepoint_guard",
		DisplayName: "Chokepoint Guard",
		Units: []ArchetypeUnit{
			{MonsterType: "Knight", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Knight", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Crossbowman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Crossbowman", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Priest", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"guard_post"},

	},
	{
		Name:        "shield_wall",
		DisplayName: "Shield Wall",
		Units: []ArchetypeUnit{
			{MonsterType: "Ogre", GridRow: 0, GridCol: 0, GridWidth: 2, GridHeight: 2, IsLeader: true},
			{MonsterType: "Archer", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Archer", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Cleric", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"barracks", "armory"},

	},
	{
		Name:        "ranged_battery",
		DisplayName: "Ranged Battery",
		Units: []ArchetypeUnit{
			{MonsterType: "Spearman", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Marksman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Marksman", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Archer", GridRow: 1, GridCol: 2, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Mage", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"mage_tower"},

	},
	{
		Name:        "fast_response",
		DisplayName: "Fast Response",
		Units: []ArchetypeUnit{
			{MonsterType: "Swordsman", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Swordsman", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Goblin Raider", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Goblin Raider", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Scout", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"patrol_route"},

	},
	{
		Name:        "mage_tower",
		DisplayName: "Mage Tower",
		Units: []ArchetypeUnit{
			{MonsterType: "Battle Mage", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Wizard", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Wizard", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Warlock", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Sorcerer", GridRow: 2, GridCol: 1, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"mage_tower"},

	},
	{
		Name:        "ambush_pack",
		DisplayName: "Ambush Pack",
		Units: []ArchetypeUnit{
			{MonsterType: "Assassin", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Assassin", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Rogue", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Rogue", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Ranger", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"patrol_route"},

	},
	{
		Name:        "command_post",
		DisplayName: "Command Post Guard",
		Units: []ArchetypeUnit{
			{MonsterType: "Knight", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterType: "Paladin", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Crossbowman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Cleric", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Priest", GridRow: 2, GridCol: 1, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"command_post"},

	},
	{
		Name:        "orc_vanguard",
		DisplayName: "Orc Vanguard",
		Units: []ArchetypeUnit{
			{MonsterType: "Orc Warrior", GridRow: 0, GridCol: 0, GridWidth: 2, GridHeight: 1, IsLeader: true},
			{MonsterType: "Ogre", GridRow: 0, GridCol: 1, GridWidth: 2, GridHeight: 2},
			{MonsterType: "Warrior", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterType: "Warrior", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"barracks"},

	},
}

// GetArchetype finds a squad archetype by name. Returns nil if not found.
func GetArchetype(name string) *SquadArchetype {
	for i := range GarrisonArchetypes {
		if GarrisonArchetypes[i].Name == name {
			return &GarrisonArchetypes[i]
		}
	}
	return nil
}
