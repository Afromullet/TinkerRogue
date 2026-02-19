package raid

// ArchetypeUnit defines a single unit within a squad archetype.
type ArchetypeUnit struct {
	MonsterName string // Key into monsterdata.json (looked up via squads.GetTemplateByName)
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
			{MonsterName: "Knight", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Knight", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Crossbowman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Crossbowman", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Priest", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"guard_post"},

	},
	{
		Name:        "shield_wall",
		DisplayName: "Shield Wall",
		Units: []ArchetypeUnit{
			{MonsterName: "Ogre", GridRow: 0, GridCol: 0, GridWidth: 2, GridHeight: 2, IsLeader: true},
			{MonsterName: "Archer", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Archer", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Cleric", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"barracks", "armory"},

	},
	{
		Name:        "ranged_battery",
		DisplayName: "Ranged Battery",
		Units: []ArchetypeUnit{
			{MonsterName: "Spearman", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Marksman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Marksman", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Archer", GridRow: 1, GridCol: 2, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Mage", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"mage_tower"},

	},
	{
		Name:        "fast_response",
		DisplayName: "Fast Response",
		Units: []ArchetypeUnit{
			{MonsterName: "Swordsman", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Swordsman", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Goblin Raider", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Goblin Raider", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Scout", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"patrol_route"},

	},
	{
		Name:        "mage_tower",
		DisplayName: "Mage Tower",
		Units: []ArchetypeUnit{
			{MonsterName: "Battle Mage", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Wizard", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Wizard", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Warlock", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Sorcerer", GridRow: 2, GridCol: 1, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"mage_tower"},

	},
	{
		Name:        "ambush_pack",
		DisplayName: "Ambush Pack",
		Units: []ArchetypeUnit{
			{MonsterName: "Assassin", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Assassin", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Rogue", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Rogue", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Ranger", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"patrol_route"},

	},
	{
		Name:        "command_post",
		DisplayName: "Command Post Guard",
		Units: []ArchetypeUnit{
			{MonsterName: "Knight", GridRow: 0, GridCol: 0, GridWidth: 1, GridHeight: 1, IsLeader: true},
			{MonsterName: "Paladin", GridRow: 0, GridCol: 1, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Crossbowman", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Cleric", GridRow: 2, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Priest", GridRow: 2, GridCol: 1, GridWidth: 1, GridHeight: 1},
		},
		PreferredRooms: []string{"command_post"},

	},
	{
		Name:        "orc_vanguard",
		DisplayName: "Orc Vanguard",
		Units: []ArchetypeUnit{
			{MonsterName: "Orc Warrior", GridRow: 0, GridCol: 0, GridWidth: 2, GridHeight: 1, IsLeader: true},
			{MonsterName: "Ogre", GridRow: 0, GridCol: 1, GridWidth: 2, GridHeight: 2},
			{MonsterName: "Warrior", GridRow: 1, GridCol: 0, GridWidth: 1, GridHeight: 1},
			{MonsterName: "Warrior", GridRow: 1, GridCol: 1, GridWidth: 1, GridHeight: 1},
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
