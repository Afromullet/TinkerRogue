// Round-trip save/load tests for the individual save chunks.
//
// These exercise each SaveChunk at the chunk level (Save -> Load -> RemapIDs into
// a fresh EntityManager) rather than through the full SaveGame/LoadGame file path.
// Chunk-level round-trips avoid the on-disk file, the checksum, and — crucially —
// the map/commander chunks' Load-time asset loading (ebiten tile/commander images),
// which is not available in headless `go test`. For those two image-dependent chunks
// only the deterministic Save serialization is asserted.
//
// Ordering note: testing.NewTestEntityManager() rebinds the global ECS component
// pointers (common.PositionComponent, etc.) to the manager it creates. Each test
// therefore builds and Saves the source world (em1) BEFORE constructing the target
// world (em2), so Save queries the correct manager.
package chunks_test

import (
	"encoding/json"
	"testing"

	"game_main/campaign/raid"
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/setup/savesystem"
	"game_main/setup/savesystem/chunks"
	"game_main/tactical/commander"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/progression"
	rstr "game_main/tactical/squads/roster"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	testfx "game_main/testing"
	"game_main/world/worldmapcore"

	"github.com/bytearena/ecs"
)

// --- Player chunk ---

func TestRoundTrip_PlayerChunk(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	player := em1.World.NewEntity()
	attrs := common.NewAttributes(12, 8, 4, 6, 3, 5)
	player.
		AddComponent(common.PlayerComponent, &common.Player{}).
		AddComponent(common.AttributeComponent, &attrs).
		AddComponent(common.ResourceStockpileComponent, &common.ResourceStockpile{Gold: 100, Iron: 50, Wood: 30, Stone: 20}).
		AddComponent(rstr.UnitRosterComponent, rstr.NewUnitRoster(25))
	em1.RegisterEntityPosition(player, coords.LogicalPosition{X: 7, Y: 9})
	em1.WorldTags["players"] = ecs.BuildTag(common.PlayerComponent, common.PositionComponent)

	chunk := &chunks.PlayerChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil data for a populated player")
	}

	em2 := testfx.NewTestEntityManager()
	idMap := savesystem.NewEntityIDMap()
	if err := chunk.Load(em2, data, idMap); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := chunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("RemapIDs: %v", err)
	}

	results := em2.World.Query(em2.WorldTags["players"])
	if len(results) != 1 {
		t.Fatalf("want exactly 1 player after load, got %d", len(results))
	}
	e := results[0].Entity

	res := common.GetComponentType[*common.ResourceStockpile](e, common.ResourceStockpileComponent)
	if res == nil || res.Gold != 100 || res.Iron != 50 || res.Wood != 30 || res.Stone != 20 {
		t.Errorf("resources = %+v, want {Gold:100 Iron:50 Wood:30 Stone:20}", res)
	}
	pos := common.GetComponentType[*coords.LogicalPosition](e, common.PositionComponent)
	if pos == nil || pos.X != 7 || pos.Y != 9 {
		t.Errorf("position = %+v, want (7,9)", pos)
	}
	ros := common.GetComponentType[*rstr.UnitRoster](e, rstr.UnitRosterComponent)
	if ros == nil || ros.MaxUnits != 25 {
		t.Errorf("roster MaxUnits = %v, want 25", ros)
	}
	at := common.GetComponentType[*common.Attributes](e, common.AttributeComponent)
	if at == nil || at.Strength != 12 {
		t.Errorf("attributes Strength = %v, want 12", at)
	}
}

// --- Squad chunk ---

func TestRoundTrip_SquadChunk(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	// Build the squad and its members directly (CreateSquadFromTemplate skips units
	// whose creature image can't be loaded, which is the case in headless tests).
	squadID := buildSquadWithMembers(em1, "Vanguard", squadcore.FormationRanged, coords.LogicalPosition{X: 2, Y: 3}, 2)
	sd := common.GetComponentTypeByID[*squadcore.SquadData](em1, squadID, squadcore.SquadComponent)
	if sd == nil {
		t.Fatal("squad was not created")
	}
	sd.Morale = 77
	sd.SquadLevel = 4
	wantMembers := len(squadcore.GetUnitIDsInSquad(squadID, em1))
	if wantMembers != 2 {
		t.Fatalf("expected 2 squad members, got %d", wantMembers)
	}

	chunk := &chunks.SquadChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil data for a populated squad")
	}

	em2 := testfx.NewTestEntityManager()
	idMap := savesystem.NewEntityIDMap()
	if err := chunk.Load(em2, data, idMap); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := chunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("RemapIDs: %v", err)
	}

	results := em2.World.Query(squadcore.SquadTag)
	if len(results) != 1 {
		t.Fatalf("want exactly 1 squad after load, got %d", len(results))
	}
	e := results[0].Entity
	lsd := common.GetComponentType[*squadcore.SquadData](e, squadcore.SquadComponent)
	if lsd == nil {
		t.Fatal("loaded squad missing SquadData")
	}
	if lsd.Name != "Vanguard" {
		t.Errorf("squad Name = %q, want Vanguard", lsd.Name)
	}
	if lsd.Formation != squadcore.FormationRanged {
		t.Errorf("squad Formation = %v, want FormationRanged", lsd.Formation)
	}
	if lsd.Morale != 77 {
		t.Errorf("squad Morale = %d, want 77", lsd.Morale)
	}
	if lsd.SquadLevel != 4 {
		t.Errorf("squad SquadLevel = %d, want 4", lsd.SquadLevel)
	}
	if gotMembers := len(squadcore.GetUnitIDsInSquad(e.GetID(), em2)); gotMembers != wantMembers {
		t.Errorf("member count = %d, want %d", gotMembers, wantMembers)
	}
}

// --- Gear chunk (depends on the player chunk creating the owner entity) ---

func TestRoundTrip_GearChunk(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	player := em1.World.NewEntity()
	player.AddComponent(common.PlayerComponent, &common.Player{})
	inv := artifacts.NewArtifactInventory(4)
	inv.OwnedArtifacts["flame_ring"] = []*artifacts.ArtifactInstance{{EquippedOn: 0}, {EquippedOn: 0}}
	inv.OwnedArtifacts["iron_charm"] = []*artifacts.ArtifactInstance{{EquippedOn: 0}}
	player.AddComponent(artifacts.ArtifactInventoryComponent, inv)
	em1.RegisterEntityPosition(player, coords.LogicalPosition{X: 1, Y: 1})
	em1.WorldTags["players"] = ecs.BuildTag(common.PlayerComponent, common.PositionComponent)

	playerChunk := &chunks.PlayerChunk{}
	gearChunk := &chunks.GearChunk{}
	playerData, err := playerChunk.Save(em1)
	if err != nil {
		t.Fatalf("player Save: %v", err)
	}
	gearData, err := gearChunk.Save(em1)
	if err != nil {
		t.Fatalf("gear Save: %v", err)
	}
	if gearData == nil {
		t.Fatal("gear Save returned nil data")
	}

	em2 := testfx.NewTestEntityManager()
	idMap := savesystem.NewEntityIDMap()
	// Player must load first so the owner entity exists and is registered in idMap.
	if err := playerChunk.Load(em2, playerData, idMap); err != nil {
		t.Fatalf("player Load: %v", err)
	}
	if err := gearChunk.Load(em2, gearData, idMap); err != nil {
		t.Fatalf("gear Load: %v", err)
	}
	// RemapIDs attaches the inventory to the loaded player.
	if err := playerChunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("player RemapIDs: %v", err)
	}
	if err := gearChunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("gear RemapIDs: %v", err)
	}

	results := em2.World.Query(em2.WorldTags["players"])
	if len(results) != 1 {
		t.Fatalf("want exactly 1 player after load, got %d", len(results))
	}
	loadedInv := common.GetComponentType[*artifacts.ArtifactInventoryData](results[0].Entity, artifacts.ArtifactInventoryComponent)
	if loadedInv == nil {
		t.Fatal("loaded player missing artifact inventory")
	}
	if loadedInv.MaxArtifacts != 4 {
		t.Errorf("MaxArtifacts = %d, want 4", loadedInv.MaxArtifacts)
	}
	if got := len(loadedInv.OwnedArtifacts["flame_ring"]); got != 2 {
		t.Errorf("flame_ring instances = %d, want 2", got)
	}
	if got := len(loadedInv.OwnedArtifacts["iron_charm"]); got != 1 {
		t.Errorf("iron_charm instances = %d, want 1", got)
	}
}

// --- Progression chunk ---
//
// The progression chunk re-attaches ProgressionData to commanders created by the
// commander chunk. Because the commander chunk's Load reconstructs an image asset
// (unavailable headlessly), this test stands in a commander via CreateCommander
// (which takes a nil image) in both managers and wires the old->new mapping into
// idMap directly, isolating the progression chunk's own Save/Load/RemapIDs logic.
func TestRoundTrip_ProgressionChunk(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	cmdID := commander.CreateCommander(em1, "Aldric", coords.LogicalPosition{X: 3, Y: 4}, 5, 8, nil)
	prog := common.GetComponentTypeByID[*progression.ProgressionData](em1, cmdID, progression.ProgressionComponent)
	if prog == nil {
		t.Fatal("commander missing ProgressionData")
	}
	prog.ArcanaPoints = 30
	prog.SkillPoints = 12
	prog.UnlockedSpellIDs = []string{"fireball", "heal"}
	prog.UnlockedPerkIDs = []string{"tough"}

	chunk := &chunks.ProgressionChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil data for a commander with progression")
	}

	em2 := testfx.NewTestEntityManager()
	newCmdID := commander.CreateCommander(em2, "Aldric", coords.LogicalPosition{X: 3, Y: 4}, 5, 8, nil)
	idMap := savesystem.NewEntityIDMap()
	idMap.Register(cmdID, newCmdID) // stand in for the commander chunk's mapping

	if err := chunk.Load(em2, data, idMap); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := chunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("RemapIDs: %v", err)
	}

	loaded := common.GetComponentTypeByID[*progression.ProgressionData](em2, newCmdID, progression.ProgressionComponent)
	if loaded == nil {
		t.Fatal("loaded commander missing ProgressionData")
	}
	if loaded.ArcanaPoints != 30 || loaded.SkillPoints != 12 {
		t.Errorf("points = {Arcana:%d Skill:%d}, want {30 12}", loaded.ArcanaPoints, loaded.SkillPoints)
	}
	if len(loaded.UnlockedSpellIDs) != 2 || len(loaded.UnlockedPerkIDs) != 1 {
		t.Errorf("unlocked spells=%v perks=%v, want 2 spells / 1 perk", loaded.UnlockedSpellIDs, loaded.UnlockedPerkIDs)
	}
}

// --- Raid chunk ---

func TestRoundTrip_RaidChunk(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	raidEntity := em1.World.NewEntity()
	raidEntity.AddComponent(raid.RaidStateComponent, &raid.RaidStateData{
		CurrentFloor: 2,
		TotalFloors:  5,
		Status:       raid.RaidStatus(1),
	})

	chunk := &chunks.RaidChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil data for an active raid")
	}

	em2 := testfx.NewTestEntityManager()
	idMap := savesystem.NewEntityIDMap()
	if err := chunk.Load(em2, data, idMap); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := chunk.RemapIDs(em2, idMap); err != nil {
		t.Fatalf("RemapIDs: %v", err)
	}

	results := em2.World.Query(raid.RaidStateTag)
	if len(results) != 1 {
		t.Fatalf("want exactly 1 raid state after load, got %d", len(results))
	}
	rs := common.GetComponentType[*raid.RaidStateData](results[0].Entity, raid.RaidStateComponent)
	if rs == nil || rs.CurrentFloor != 2 || rs.TotalFloors != 5 || rs.Status != raid.RaidStatus(1) {
		t.Errorf("raid state = %+v, want {CurrentFloor:2 TotalFloors:5 Status:1}", rs)
	}
}

func TestRoundTrip_RaidChunk_NoActiveRaid(t *testing.T) {
	em1 := testfx.NewTestEntityManager()

	chunk := &chunks.RaidChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	em2 := testfx.NewTestEntityManager()
	idMap := savesystem.NewEntityIDMap()
	if data != nil {
		if err := chunk.Load(em2, data, idMap); err != nil {
			t.Fatalf("Load: %v", err)
		}
	}
	if got := len(em2.World.Query(raid.RaidStateTag)); got != 0 {
		t.Errorf("want 0 raid states with no active raid, got %d", got)
	}
}

// --- Commander chunk (Save-side only; Load needs an image asset) ---

func TestRoundTrip_CommanderChunk_Save(t *testing.T) {
	em1 := testfx.NewTestEntityManager()
	commander.CreateCommander(em1, "Aldric", coords.LogicalPosition{X: 3, Y: 4}, 5, 8, nil)

	chunk := &chunks.CommanderChunk{}
	data, err := chunk.Save(em1)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil data for a populated commander")
	}

	var parsed struct {
		Commanders []struct {
			Name        string `json:"name"`
			IsActive    bool   `json:"isActive"`
			ActionState *struct {
				MovementRemaining int `json:"movementRemaining"`
			} `json:"actionState"`
			SquadRoster *struct {
				MaxSquads int `json:"maxSquads"`
			} `json:"squadRoster"`
		} `json:"commanders"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal saved commander data: %v", err)
	}
	if len(parsed.Commanders) != 1 {
		t.Fatalf("want 1 saved commander, got %d", len(parsed.Commanders))
	}
	c := parsed.Commanders[0]
	if c.Name != "Aldric" || !c.IsActive {
		t.Errorf("commander = {Name:%q IsActive:%v}, want {Aldric true}", c.Name, c.IsActive)
	}
	if c.ActionState == nil || c.ActionState.MovementRemaining != 5 {
		t.Errorf("actionState = %+v, want MovementRemaining 5", c.ActionState)
	}
	if c.SquadRoster == nil || c.SquadRoster.MaxSquads != 8 {
		t.Errorf("squadRoster = %+v, want MaxSquads 8", c.SquadRoster)
	}
}

// --- Map chunk (Save-side only; Load reconstructs tile images from disk) ---

func TestRoundTrip_MapChunk_Save(t *testing.T) {
	floor := worldmapcore.NewTile(0, 0, coords.LogicalPosition{X: 0, Y: 0}, false, nil, worldmapcore.FLOOR, true)
	wall := worldmapcore.NewTile(coords.ScreenInfo.TileSize, 0, coords.LogicalPosition{X: 1, Y: 0}, true, nil, worldmapcore.WALL, false)
	gm := worldmapcore.NewGameMapFromParts(2, 1, worldmapcore.GenerationResult{
		Tiles: []*worldmapcore.Tile{&floor, &wall},
	})

	// The map chunk's Save path does not touch ECS entities, but Save takes an
	// *EntityManager, so provide a fresh one.
	chunk := &chunks.MapChunk{GameMap: &gm}
	data, err := chunk.Save(testfx.NewTestEntityManager())
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if data == nil {
		t.Fatal("Save returned nil map data")
	}

	var parsed struct {
		Width  int `json:"width"`
		Height int `json:"height"`
		Tiles  []struct {
			X          int  `json:"x"`
			Y          int  `json:"y"`
			TileType   int  `json:"type"`
			Blocked    bool `json:"blocked"`
			IsRevealed bool `json:"revealed"`
		} `json:"tiles"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal saved map data: %v", err)
	}
	if parsed.Width != coords.ScreenInfo.DungeonWidth || parsed.Height != coords.ScreenInfo.DungeonHeight {
		t.Errorf("dims = %dx%d, want %dx%d", parsed.Width, parsed.Height, coords.ScreenInfo.DungeonWidth, coords.ScreenInfo.DungeonHeight)
	}
	if len(parsed.Tiles) != 2 {
		t.Fatalf("saved tiles = %d, want 2", len(parsed.Tiles))
	}
	var sawFloor, sawWall bool
	for _, tile := range parsed.Tiles {
		if tile.TileType == int(worldmapcore.FLOOR) && !tile.Blocked && tile.IsRevealed {
			sawFloor = true
		}
		if tile.TileType == int(worldmapcore.WALL) && tile.Blocked && !tile.IsRevealed {
			sawWall = true
		}
	}
	if !sawFloor || !sawWall {
		t.Errorf("tiles missing expected floor/wall (sawFloor=%v sawWall=%v)", sawFloor, sawWall)
	}
}

// --- helpers ---

// buildSquadWithMembers creates a squad entity plus memberCount member units with
// the components the squad chunk serializes, without going through the asset-loading
// CreateSquadFromTemplate path. Returns the new squad ID.
func buildSquadWithMembers(em *common.EntityManager, name string, formation squadcore.FormationType, pos coords.LogicalPosition, memberCount int) ecs.EntityID {
	squadEntity := em.World.NewEntity()
	squadID := squadEntity.GetID()
	squadEntity.AddComponent(squadcore.SquadComponent, &squadcore.SquadData{
		SquadID:   squadID,
		Name:      name,
		Formation: formation,
		Morale:    100,
		MaxUnits:  9,
	})
	squadEntity.AddComponent(common.PositionComponent, &pos)

	for i := 0; i < memberCount; i++ {
		member := em.World.NewEntity()
		attrs := common.NewAttributes(6, 6, 0, 0, 1, 1)
		member.
			AddComponent(squadcore.SquadMemberComponent, &squadcore.SquadMemberData{SquadID: squadID}).
			AddComponent(common.NameComponent, &common.Name{NameStr: "Unit"}).
			AddComponent(common.AttributeComponent, &attrs).
			AddComponent(squadcore.GridPositionComponent, &squadcore.GridPositionData{
				AnchorRow: i, AnchorCol: 0, CellWidth: 1, CellHeight: 1,
			}).
			AddComponent(squadcore.UnitRoleComponent, &squadcore.UnitRoleData{Role: unitdefs.RoleTank})
	}

	return squadID
}
