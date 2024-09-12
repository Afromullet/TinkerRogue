package timesystem

import "fmt"

type TurnState int

const (
	BeforePlayerAction = iota
	PlayerTurn
	MonsterTurn
	ExecuteActions
)

type GameTurn struct {
	Turn             TurnState
	TurnCounter      int
	ActionDispatcher ActionManager
}

// A "Unit of Time" is 3 turns. After that reset all action points

func (t *GameTurn) UpdateTurnCounter() {

	t.TurnCounter++
	if t.TurnCounter == 3 {
		t.TurnCounter = 0
		t.ActionDispatcher.ResetActionPoints()
		fmt.Println("Resetting action points")
	}
}
