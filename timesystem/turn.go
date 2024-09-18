package timesystem

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
	TotalNumTurns    int
}

// A "Unit of Time" is 10 turns. After that reset all action points
// TODO, there is something janky when the action points are reset - everything is "offset" by oen turn
// I.E, if a creature has the same movement speed as the player, it will be one tile behind after the reset
func (t *GameTurn) UpdateTurnCounter() {

	t.TurnCounter++
	if t.TurnCounter == 10 {
		t.TurnCounter = 0
		t.ActionDispatcher.ResetActionPoints()

	}

	t.TotalNumTurns++
}
