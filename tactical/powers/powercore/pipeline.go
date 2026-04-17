package powercore

import (
	"game_main/tactical/combat/combattypes"

	"github.com/bytearena/ecs"
)

// Handler types for each combat lifecycle event. Subscribers are typed per
// event so the compiler catches signature mistakes at wiring time.
type (
	PostResetHandler      func(factionID ecs.EntityID, squadIDs []ecs.EntityID)
	AttackCompleteHandler func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)
	TurnEndHandler        func(round int)
	MoveCompleteHandler   func(squadID ecs.EntityID)
)

// PowerPipeline owns the ordered subscriber lists for each event. Order is
// established at registration time — CombatService registers artifacts before
// perks before GUI callbacks in NewCombatService, replacing the previous
// hard-coded ordering inside four near-duplicate Fire* method bodies.
type PowerPipeline struct {
	postReset      []PostResetHandler
	attackComplete []AttackCompleteHandler
	turnEnd        []TurnEndHandler
	moveComplete   []MoveCompleteHandler
}

// OnPostReset appends a subscriber to the post-reset event.
func (p *PowerPipeline) OnPostReset(h PostResetHandler) {
	p.postReset = append(p.postReset, h)
}

// OnAttackComplete appends a subscriber to the attack-complete event.
func (p *PowerPipeline) OnAttackComplete(h AttackCompleteHandler) {
	p.attackComplete = append(p.attackComplete, h)
}

// OnTurnEnd appends a subscriber to the turn-end event.
func (p *PowerPipeline) OnTurnEnd(h TurnEndHandler) {
	p.turnEnd = append(p.turnEnd, h)
}

// OnMoveComplete appends a subscriber to the move-complete event.
func (p *PowerPipeline) OnMoveComplete(h MoveCompleteHandler) {
	p.moveComplete = append(p.moveComplete, h)
}

// FirePostReset invokes every post-reset subscriber in registration order.
func (p *PowerPipeline) FirePostReset(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	for _, h := range p.postReset {
		h(factionID, squadIDs)
	}
}

// FireAttackComplete invokes every attack-complete subscriber in registration order.
func (p *PowerPipeline) FireAttackComplete(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
	for _, h := range p.attackComplete {
		h(attackerID, defenderID, result)
	}
}

// FireTurnEnd invokes every turn-end subscriber in registration order.
func (p *PowerPipeline) FireTurnEnd(round int) {
	for _, h := range p.turnEnd {
		h(round)
	}
}

// FireMoveComplete invokes every move-complete subscriber in registration order.
func (p *PowerPipeline) FireMoveComplete(squadID ecs.EntityID) {
	for _, h := range p.moveComplete {
		h(squadID)
	}
}
