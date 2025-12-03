package squads

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CreateDummySquadsForTesting creates several test squads with various unit configurations
// This is for GUI testing and development, NOT for unit tests
func CreateDummySquadsForTesting(squadmanager *common.EntityManager) error {
	// Components should already be initialized via InitializeSquadData() before calling this
	// Validate components are initialized
	if SquadComponent == nil {
		return fmt.Errorf("squad components not initialized - call InitializeSquadData() first")
	}

	// Squad 1: "Alpha Squad" - Balanced formation with variety of roles
	CreateEmptySquad(squadmanager, "Alpha Squad")
	alphaSquadID := findSquadByName("Alpha Squad", squadmanager)

	// Add a tank leader in front row
	tankLeader := UnitTemplate{
		Name: "Captain Steel",
		Attributes: common.Attributes{
			Strength:  15,
			Dexterity: 10,
			Magic:     5,
			Leadership: 20, // High leadership for capacity
			Armor:     12,
			Weapon:    8,
		},
		GridRow:        0,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleTank,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		IsLeader:       true,
		CoverValue:     0.3,
		CoverRange:     2,
		RequiresActive: true,
	}
	AddUnitToSquad(alphaSquadID, squadmanager, tankLeader, 0, 1)

	// Add DPS units in front row
	dps1 := UnitTemplate{
		Name: "Rogue Shadow",
		Attributes: common.Attributes{
			Strength:   12,
			Dexterity:  18,
			Magic:      3,
			Leadership: 5,
			Armor:      6,
			Weapon:     14,
		},
		GridRow:        0,
		GridCol:        0,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleDPS,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(alphaSquadID, squadmanager, dps1, 0, 0)

	dps2 := UnitTemplate{
		Name: "Blade Dancer",
		Attributes: common.Attributes{
			Strength:   14,
			Dexterity:  16,
			Magic:      4,
			Leadership: 6,
			Armor:      7,
			Weapon:     13,
		},
		GridRow:        0,
		GridCol:        2,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleDPS,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(alphaSquadID, squadmanager, dps2, 0, 2)

	// Add support units in back row
	support1 := UnitTemplate{
		Name: "Cleric Luna",
		Attributes: common.Attributes{
			Strength:   6,
			Dexterity:  8,
			Magic:      20,
			Leadership: 12,
			Armor:      5,
			Weapon:     4,
		},
		GridRow:        2,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleSupport,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}},
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(alphaSquadID, squadmanager, support1, 2, 1)

	// Add ranged DPS in middle row
	rangedDPS := UnitTemplate{
		Name: "Archer Swift",
		Attributes: common.Attributes{
			Strength:   10,
			Dexterity:  20,
			Magic:      4,
			Leadership: 5,
			Armor:      6,
			Weapon:     15,
		},
		GridRow:        1,
		GridCol:        2,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleDPS,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}},
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(alphaSquadID, squadmanager, rangedDPS, 1, 2)

	// Squad 2: "Bravo Squad" - Tank-heavy defensive formation
	CreateEmptySquad(squadmanager, "Bravo Squad")
	bravoSquadID := findSquadByName("Bravo Squad", squadmanager)

	// Front row tanks
	tankBravo1 := UnitTemplate{
		Name: "Knight Boris",
		Attributes: common.Attributes{
			Strength:   18,
			Dexterity:  8,
			Magic:      2,
			Leadership: 15,
			Armor:      15,
			Weapon:     10,
		},
		GridRow:        0,
		GridCol:        0,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleTank,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		IsLeader:       true,
		CoverValue:     0.4,
		CoverRange:     2,
		RequiresActive: true,
	}
	AddUnitToSquad(bravoSquadID, squadmanager, tankBravo1, 0, 0)

	tankBravo2 := UnitTemplate{
		Name: "Guardian Greta",
		Attributes: common.Attributes{
			Strength:   16,
			Dexterity:  9,
			Magic:      3,
			Leadership: 10,
			Armor:      14,
			Weapon:     9,
		},
		GridRow:        0,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleTank,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		CoverValue:     0.35,
		CoverRange:     1,
		RequiresActive: true,
	}
	AddUnitToSquad(bravoSquadID, squadmanager, tankBravo2, 0, 1)

	tankBravo3 := UnitTemplate{
		Name: "Defender Drake",
		Attributes: common.Attributes{
			Strength:   17,
			Dexterity:  7,
			Magic:      2,
			Leadership: 8,
			Armor:      16,
			Weapon:     8,
		},
		GridRow:        0,
		GridCol:        2,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleTank,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		CoverValue:     0.3,
		CoverRange:     1,
		RequiresActive: true,
	}
	AddUnitToSquad(bravoSquadID, squadmanager, tankBravo3, 0, 2)

	// Back row support
	supportBravo := UnitTemplate{
		Name: "Priest Pavel",
		Attributes: common.Attributes{
			Strength:   5,
			Dexterity:  7,
			Magic:      18,
			Leadership: 10,
			Armor:      4,
			Weapon:     3,
		},
		GridRow:        2,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleSupport,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {2, 0}, {2, 1}, {2, 2}},
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(bravoSquadID, squadmanager, supportBravo, 2, 1)

	// Squad 3: "Charlie Squad" - Offensive DPS-focused formation
	CreateEmptySquad(squadmanager, "Charlie Squad")
	charlieSquadID := findSquadByName("Charlie Squad", squadmanager)

	// Front row DPS leader
	dpsLeader := UnitTemplate{
		Name: "Warlord Zara",
		Attributes: common.Attributes{
			Strength:   20,
			Dexterity:  15,
			Magic:      5,
			Leadership: 18,
			Armor:      10,
			Weapon:     18,
		},
		GridRow:        0,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleDPS,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}},
		IsLeader:       true,
		CoverValue:     0.15,
		CoverRange:     1,
		RequiresActive: true,
	}
	AddUnitToSquad(charlieSquadID, squadmanager, dpsLeader, 0, 1)

	// Multiple DPS units across rows
	for i := 0; i < 3; i++ {
		row := i
		col := i % 3

		dpsUnit := UnitTemplate{
			Name: "Berserker " + string(rune('A'+i)),
			Attributes: common.Attributes{
				Strength:   18 - i,
				Dexterity:  14 + i,
				Magic:      3,
				Leadership: 5,
				Armor:      8,
				Weapon:     16 - i,
			},
			GridRow:        row,
			GridCol:        col,
			GridWidth:      1,
			GridHeight:     1,
			Role:           RoleDPS,
			TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
			CoverValue:     0.0,
			CoverRange:     0,
			RequiresActive: false,
		}

		// Skip if position conflicts with leader
		if !(row == 0 && col == 1) {
			AddUnitToSquad(charlieSquadID, squadmanager, dpsUnit, row, col)
		}
	}

	// Squad 4: "Delta Squad" - Small specialized squad (only 2 units for variety)
	CreateEmptySquad(squadmanager, "Delta Squad")
	deltaSquadID := findSquadByName("Delta Squad", squadmanager)

	mage := UnitTemplate{
		Name: "Archmage Merlin",
		Attributes: common.Attributes{
			Strength:   5,
			Dexterity:  10,
			Magic:      25,
			Leadership: 20,
			Armor:      5,
			Weapon:     5,
		},
		GridRow:        2,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleSupport,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}}, // Front row AOE
		IsLeader:       true,
		CoverValue:     0.0,
		CoverRange:     0,
		RequiresActive: false,
	}
	AddUnitToSquad(deltaSquadID, squadmanager, mage, 2, 1)

	bodyguard := UnitTemplate{
		Name: "Bodyguard Brick",
		Attributes: common.Attributes{
			Strength:   20,
			Dexterity:  8,
			Magic:      2,
			Leadership: 5,
			Armor:      18,
			Weapon:     8,
		},
		GridRow:        1,
		GridCol:        1,
		GridWidth:      1,
		GridHeight:     1,
		Role:           RoleTank,
		TargetCells:    [][2]int{{0, 0}, {0, 1}, {0, 2}},
		CoverValue:     0.5,
		CoverRange:     1,
		RequiresActive: true,
	}
	AddUnitToSquad(deltaSquadID, squadmanager, bodyguard, 1, 1)

	return nil
}

// findSquadByName is a helper to get the last created squad's ID by name
func findSquadByName(name string, squadmanager *common.EntityManager) ecs.EntityID {
	// Get all entities with SquadComponent
	entityIDs := squadmanager.GetAllEntities()
	for _, entityID := range entityIDs {
		if squadmanager.HasComponent(entityID, SquadComponent) {
			if squadDataRaw, ok := squadmanager.GetComponent(entityID, SquadComponent); ok {
				squadData := squadDataRaw.(*SquadData)
				if squadData.Name == name {
					return entityID
				}
			}
		}
	}
	return 0 // Not found
}
