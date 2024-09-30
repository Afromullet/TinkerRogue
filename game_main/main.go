package main

/*
When setting up the project, run go mod tidy to install dependencies

*/
//Original import statmenets. Started adding ebiten UI stuff in the other import statements. This is to fall back on

/*
import (
	_ "image/png"
	"log"

	"github.com/bytearena/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)*/

import (
	"game_main/avatar"
	"game_main/common"
	"game_main/entitytemplates"
	"game_main/gear"
	"game_main/graphics"
	"game_main/rendering"

	"game_main/gui"
	"game_main/input"
	"game_main/monsters"
	"game_main/spawning"
	"game_main/testing"
	"game_main/timesystem"
	"game_main/worldmap"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// Using https://www.fatoldyeti.com/categories/roguelike-tutorial/ as a starting point.
// Copying some of the code with modification. Whenever I change a name, it's to help me build a better mental model
// Of what the code is doing as I'm learning GoLang
var DEBUG_MODE = true

type Game struct {
	em         common.EntityManager
	gameUI     gui.PlayerUI
	playerData avatar.PlayerData
	gameMap    worldmap.GameMap
	camera     graphics.Camera

	ts timesystem.GameTurn
}

// NewGame creates a new Game Object and initializes the data
// This is a pretty solid refactor candidate for later
func NewGame() *Game {
	g := &Game{}
	SetupCamera(g)
	g.gameMap = worldmap.NewGameMap()
	g.playerData = avatar.PlayerData{}
	entitytemplates.ReadGameData()
	InitializeECS(&g.em)

	InitializePlayerData(&g.em, &g.playerData, &g.gameMap)

	g.ts.Turn = timesystem.PlayerTurn
	g.ts.TurnCounter = 0

	testing.CreateTestItems(g.em.World, g.em.WorldTags, &g.gameMap)

	testing.UpdateContentsForTest(&g.em, &g.gameMap)
	spawning.SpawnStartingCreatures(0, &g.em, &g.gameMap, &g.playerData)

	testing.CreateTestConsumables(&g.em, &g.gameMap)
	testing.InitTestActionManager(&g.em, &g.playerData, &g.ts)

	g.ts.ActionDispatcher.ResetActionManager()

	//spawning.SpawnStartingLoot(g.em, &g.gameMap)
	spawning.SpawnStartingEquipment(&g.em, &g.gameMap, &g.playerData)

	AddCreaturesToTracker(&g.em)

	return g

}

// Once the player performs an action, the Action Manager adds Monster actions to the queue.
// Performs all of the actions. Then it reorders them.
// When the Turn Counter hits 0, we reset all action points. That's our "unit of time"
func ManageTurn(g *Game) {

	g.playerData.UpdatePlayerAttributes()
	g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.GetPlayerAttributes().AttributeText())
	if g.ts.Turn == timesystem.PlayerTurn && !g.playerData.InputStates.HasKeyInput {

		//Apply Consumabl Effects at beginning of player turn
		//gear.ConsumableEffectApplier(g.playerData.PlayerEntity)

		input.PlayerActions(&g.em, &g.playerData, &g.gameMap, &g.gameUI, &g.ts)
		if g.playerData.InputStates.HasKeyInput {

			gear.RunEffectTracker(g.playerData.PlayerEntity)

			g.gameUI.StatsUI.StatsTextArea.SetText(g.playerData.GetPlayerAttributes().AttributeText())
			g.ts.Turn = timesystem.MonsterTurn

		}

		// The drawing and throwing still work after changing the way the input and actions work
		// Uncommented now because we need to figure out how to implement this in the Action Energy based ystem
		if g.gameUI.IsThrowableItemSelected() {
			g.playerData.InputStates.IsThrowing = true

		} else {
			g.playerData.InputStates.IsThrowing = false
		}
		input.HandlePlayerThrowable(&g.em, &g.playerData, &g.gameMap, &g.gameUI)
		input.HandlePlayerRangedAttack(&g.em, &g.playerData, &g.gameMap)

	}
	if g.ts.Turn == timesystem.MonsterTurn && g.playerData.InputStates.HasKeyInput {
		monsters.MonsterSystems(&g.em, &g.playerData, &g.gameMap, &g.ts)

		// Returns true if the next action is the player.

		//ExecuteActionsUntilPlayer2 places the queue back in priority order. The old function executes each action only once
		// untilk the player

		RemoveDeadEntities(&g.em, g.ts.ActionDispatcher, &g.gameMap)
		g.ts.ActionDispatcher.CleanController()
		if g.ts.ActionDispatcher.ExecuteActionsUntilPlayer2(&g.playerData) {

			//Perform the players action
			g.ts.ActionDispatcher.ExecuteFirst()

		}

		//g.ts.ActionDispatcher.ReorderActions() // If executefirst inserts in priority order I won't need this
		g.ts.UpdateTurnCounter()

		g.playerData.InputStates.HasKeyInput = false
		g.ts.Turn = timesystem.PlayerTurn

		if g.ts.TotalNumTurns%10 == 0 {

			//addspawning.SpawnMonster(g.em, &g.gameMap)
		}

		RemoveDeadEntities(&g.em, g.ts.ActionDispatcher, &g.gameMap)

	}

}

func RemoveDeadEntities(ecsmanager *common.EntityManager, am timesystem.ActionManager, gm *worldmap.GameMap) {
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
		attr := common.GetComponentType[*common.Attributes](c.Entity, common.AttributeComponent)

		if attr.CurrentHealth <= 0 {

			if attr.CurrentHealth <= 0 {

				pos := common.GetPosition(c.Entity)
				ind := graphics.IndexFromXY(pos.X, pos.Y)
				gm.Tiles[ind].Blocked = false

				am.RemoveActionQueueForEntity(c.Entity)

				ecsmanager.World.DisposeEntity(c.Entity)
				monsters.NumMonstersOnMap--

				if monsters.NumMonstersOnMap == -1 {
					monsters.NumMonstersOnMap = 0
				}
			}

		}
	}
}

// Update is called each tic.
// todo still need to remove game
func (g *Game) Update() error {

	g.gameUI.MainPlayerInterface.Update()
	UpdateCameraPosition(g)
	graphics.VXHandler.UpdateVisualEffects()

	input.PlayerDebugActions(&g.playerData)

	ManageTurn(g)

	return nil

}

// Draw is called each draw cycle and is where we will blit.
func (g *Game) Draw(screen *ebiten.Image) {

	g.gameMap.DrawLevel(world, DEBUG_MODE)
	//g.gameMap.DrawLevelSection(world, DEBUG_MODE, g.playerData.Pos, 10)

	rendering.ProcessRenderables(&g.em, g.gameMap, world, DEBUG_MODE)
	g.gameUI.MainPlayerInterface.Draw(world)

	gui.ProcessUserLog(g.em, world, &g.gameUI.MsgUI)
	graphics.VXHandler.DrawVisualEffects(world)
	g.gameUI.MainPlayerInterface.Draw(world)
	g.camera.Render(world, screen)

}

// Layout will return the screen dimensions.

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {

	//zoomScale := math.Pow(1.01, float64(10))
	//scaledOffset := graphics.StatsUIOffset * int(zoomScale)

	return int(g.camera.ViewPort[0]) + graphics.StatsUIOffset, int(g.camera.ViewPort[1]) // Set the layout based on the camera's viewport
}

func main() {

	g := NewGame()

	g.gameUI.CreateMainInterface(&g.playerData, &g.em)

	ebiten.SetWindowResizable(true)

	ebiten.SetWindowTitle("Tower")

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
