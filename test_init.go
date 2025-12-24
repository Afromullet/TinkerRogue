package main

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
)

func main() {
	manager := common.NewEntityManager()
	fmt.Printf("Before init: SquadComponent=%v\n", squads.SquadComponent)
	squads.InitSquadComponents(manager)
	fmt.Printf("After init: SquadComponent=%v\n", squads.SquadComponent)
}
