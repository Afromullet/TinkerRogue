package gui

import (
	"game_main/avatar"

	"github.com/ebitenui/ebitenui"
)

type PlayerUI struct {
	ItemsUI             PlayerItemsUI
	StatsUI             PlayerStatsUI
	MainPlayerInterface *ebitenui.UI
}

// Throwing an item will show a square to represent the AOE of the throwable.
// Right now it's a function of Game until I separate the UI more.
// Not going to try to generalize/abstract this until I figure out how I want to handle this
// The impression I get now is that this will take a "state machine" since the throwable window closes
// Once I click out of it
func (p *PlayerUI) IsThrowableItemSelected() bool {

	return p.ItemsUI.ThrowableItemDisplay.ThrowableItemSelected

}

func (p *PlayerUI) SetThrowableItemSelected(selected bool) {

	p.ItemsUI.ThrowableItemDisplay.ThrowableItemSelected = selected

}

// func CreatePlayerItemsUI(playerUI *PlayerUI, inv *gear.Inventory, pl *avatar.PlayerData)
func (playerUI *PlayerUI) CreatePlayerUI(playerData *avatar.PlayerData) {

	//playerUI.MainPlayerInterface = &ebitenui.UI{}

	playerUI.MainPlayerInterface = CreatePlayerItemsUI(playerUI, playerData.Inv, playerData)
	//playerUI.MainPlayerInterface.Container.AddChild(playerUI.StatsUI.rootContainer)

	//playerUI.SecondInterface.Container.AddChild(playerUI.StatsUI.rootContainer)

	//playerUI.StatsUI.CreateStatsUI()
	//playerUI.MainPlayerInterface.Container.AddChild(playerUI.StatsUI.rootContainer)

}
