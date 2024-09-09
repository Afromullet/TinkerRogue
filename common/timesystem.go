package common

type TurnState int

const (
	BeforePlayerAction = iota
	PlayerTurn
	MonsterTurn
)

type TimeSystem struct {
	Turn        TurnState
	TurnCounter int
}

func GetNextState(state TurnState) TurnState {
	switch state {
	case BeforePlayerAction:
		return PlayerTurn
	case PlayerTurn:
		return MonsterTurn
	case MonsterTurn:
		return BeforePlayerAction
	default:
		return PlayerTurn
	}
}
