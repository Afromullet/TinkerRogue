package main

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads/squadcore"
)

func main() {
	manager := common.NewEntityManager()
	fmt.Printf("Before init: SquadComponent=%v\n", squadcore.SquadComponent)
	squadcore.InitSquadComponents(manager)
	fmt.Printf("After init: SquadComponent=%v\n", squadcore.SquadComponent)
}
