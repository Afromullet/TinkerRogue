package ai

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/tactical/squadcommands"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ScoredAction represents a possible action with its utility score
type ScoredAction struct {
	Action      SquadAction
	Score       float64
	Description string
}

// SquadAction interface for executable actions
type SquadAction interface {
	Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool
	GetDescription() string
}

// MoveAction represents a movement decision
type MoveAction struct {
	squadID ecs.EntityID
	target  coords.LogicalPosition
}

func (ma *MoveAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	cmd := squadcommands.NewMoveSquadCommand(
		manager,
		movementSystem,
		ma.squadID,
		ma.target,
	)

	err := cmd.Execute()
	return err == nil
}

func (ma *MoveAction) GetDescription() string {
	return fmt.Sprintf("Move to %v", ma.target)
}

// AttackAction represents an attack decision
type AttackAction struct {
	attackerID   ecs.EntityID
	targetID     ecs.EntityID
	aiController *AIController // Reference to queue the attack for animation
}

func (aa *AttackAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	// Execute the attack
	result := combatActSystem.ExecuteAttackAction(aa.attackerID, aa.targetID)

	// Log result for debugging
	if result.Success {
		fmt.Printf("[AI] %s attacked %s\n", result.AttackerName, result.TargetName)
		if result.TargetDestroyed {
			fmt.Printf("[AI] %s was destroyed!\n", result.TargetName)
		}

		// Queue this attack for animation playback after AI turn completes
		if aa.aiController != nil {
			aa.aiController.QueueAttack(aa.attackerID, aa.targetID)
		}
	} else {
		fmt.Printf("[AI] Attack failed: %s\n", result.ErrorReason)
	}

	return result.Success
}

func (aa *AttackAction) GetDescription() string {
	return fmt.Sprintf("Attack squad %d", aa.targetID)
}

// WaitAction represents doing nothing (skip turn)
type WaitAction struct {
	squadID ecs.EntityID
}

func (wa *WaitAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	// CRITICAL: Mark the squad's turn as complete to prevent infinite loops
	// Find the action state entity and set both flags
	for _, result := range manager.World.Query(combat.ActionStateTag) {
		actionEntity := result.Entity
		actionState := common.GetComponentType[*combat.ActionStateData](actionEntity, combat.ActionStateComponent)

		if actionState != nil && actionState.SquadID == wa.squadID {
			// Mark both actions as used so squad finishes turn
			actionState.HasMoved = true
			actionState.HasActed = true
			return true
		}
	}

	return false // Action state not found
}

func (wa *WaitAction) GetDescription() string {
	return "Wait"
}

// ActionEvaluator generates and scores possible actions
type ActionEvaluator struct {
	ctx ActionContext
}

// NewActionEvaluator creates a new action evaluator
func NewActionEvaluator(ctx ActionContext) *ActionEvaluator {
	return &ActionEvaluator{ctx: ctx}
}

// EvaluateAllActions generates all possible actions and scores them
func (ae *ActionEvaluator) EvaluateAllActions() []ScoredAction {
	var actions []ScoredAction

	// Generate movement actions if haven't moved
	if !ae.ctx.ActionState.HasMoved {
		actions = append(actions, ae.evaluateMovement()...)
	}

	// Generate attack actions if haven't acted
	if !ae.ctx.ActionState.HasActed {
		actions = append(actions, ae.evaluateAttacks()...)
	}

	// Always have wait as fallback
	actions = append(actions, ScoredAction{
		Action:      &WaitAction{squadID: ae.ctx.SquadID},
		Score:       0.0, // Lowest priority
		Description: "Wait (fallback)",
	})

	return actions
}

// evaluateMovement generates and scores movement actions
func (ae *ActionEvaluator) evaluateMovement() []ScoredAction {
	var actions []ScoredAction

	// Get valid movement tiles
	validTiles := ae.getValidMovementTiles()

	if len(validTiles) == 0 {
		return actions
	}

	// Score each position based on role
	for _, pos := range validTiles {
		score := ae.scoreMovementPosition(pos)

		actions = append(actions, ScoredAction{
			Action:      &MoveAction{squadID: ae.ctx.SquadID, target: pos},
			Score:       score,
			Description: fmt.Sprintf("Move to %v (role-aware)", pos),
		})
	}

	return actions
}

// getValidMovementTiles returns all tiles the squad can actually move to
// Uses ActionContext.MovementSystem to validate tiles (occupied, blocked, etc)
func (ae *ActionEvaluator) getValidMovementTiles() []coords.LogicalPosition {
	var tiles []coords.LogicalPosition

	moveSpeed := squads.GetSquadMovementSpeed(ae.ctx.SquadID, ae.ctx.Manager)

	for dx := -moveSpeed; dx <= moveSpeed; dx++ {
		for dy := -moveSpeed; dy <= moveSpeed; dy++ {
			pos := coords.LogicalPosition{
				X: ae.ctx.CurrentPos.X + dx,
				Y: ae.ctx.CurrentPos.Y + dy,
			}

			distance := ae.ctx.CurrentPos.ChebyshevDistance(&pos)
			if distance > 0 && distance <= moveSpeed {
				// CRITICAL: Validate tile is actually movable (not occupied/blocked)
				// Without this check, AI generates invalid movement actions that fail,
				// causing it to break out of action loop and stop moving
				if ae.ctx.MovementSystem.CanMoveTo(ae.ctx.SquadID, pos) {
					tiles = append(tiles, pos)
				}
			}
		}
	}

	return tiles
}

// scoreMovementPosition scores a movement position based on role and threat
func (ae *ActionEvaluator) scoreMovementPosition(pos coords.LogicalPosition) float64 {
	// Use threat evaluator to get role-weighted threat
	threat := ae.ctx.ThreatEval.GetRoleWeightedThreat(ae.ctx.SquadID, pos)

	// Base score
	baseScore := 50.0
	score := baseScore - threat

	// Bonus for staying near allies (avoid isolation)
	supportLayer := ae.ctx.ThreatEval.GetSupportLayer()
	allyProximity := supportLayer.GetAllyProximityAt(pos)
	score += float64(allyProximity) * 3.0

	// CRITICAL: Add approach enemy bonus for offensive roles
	// Without this, units only consider threat avoidance and flee
	approachBonus := ae.scoreApproachEnemy(pos)
	score += approachBonus

	return score
}

// scoreApproachEnemy rewards moving closer to enemies for offensive roles
// Tanks get highest bonus (intercept), DPS moderate (engage), Support negative (stay back)
func (ae *ActionEvaluator) scoreApproachEnemy(pos coords.LogicalPosition) float64 {
	// Find nearest enemy
	nearestEnemy, nearestDistance := ae.findNearestEnemy()
	if nearestEnemy == 0 {
		return 0.0 // No enemies found
	}

	// Calculate distance from candidate position to nearest enemy
	enemyPos, err := combat.GetSquadMapPosition(nearestEnemy, ae.ctx.Manager)
	if err != nil {
		return 0.0
	}

	newDistance := pos.ChebyshevDistance(&enemyPos)

	// Calculate distance improvement (positive = getting closer)
	distanceImprovement := nearestDistance - newDistance

	// Role-based approach multipliers
	var approachMultiplier float64
	switch ae.ctx.SquadRole {
	case squads.RoleTank:
		approachMultiplier = 15.0 // Tanks strongly want to close distance
	case squads.RoleDPS:
		approachMultiplier = 8.0 // DPS moderately want to engage
	case squads.RoleSupport:
		approachMultiplier = -5.0 // Support wants to maintain distance
	default:
		approachMultiplier = 5.0 // Default: slight approach preference
	}

	// Base approach score from distance improvement
	approachScore := float64(distanceImprovement) * approachMultiplier

	// Bonus for being in attack range (can attack next turn)
	maxRange := ae.getMaxAttackRange()
	if newDistance <= maxRange {
		approachScore += 20.0 // Strong bonus for being in attack range
	} else if newDistance <= maxRange+2 {
		approachScore += 10.0 // Moderate bonus for being close to attack range
	}

	return approachScore
}

// findNearestEnemy returns the nearest enemy squad and distance
func (ae *ActionEvaluator) findNearestEnemy() (ecs.EntityID, int) {
	var nearestEnemy ecs.EntityID
	nearestDistance := 9999

	allFactions := combat.GetAllFactions(ae.ctx.Manager)

	for _, factionID := range allFactions {
		if factionID == ae.ctx.FactionID {
			continue // Skip own faction
		}

		squadIDs := combat.GetSquadsForFaction(factionID, ae.ctx.Manager)

		for _, squadID := range squadIDs {
			if squads.IsSquadDestroyed(squadID, ae.ctx.Manager) {
				continue
			}

			enemyPos, err := combat.GetSquadMapPosition(squadID, ae.ctx.Manager)
			if err != nil {
				continue
			}

			distance := ae.ctx.CurrentPos.ChebyshevDistance(&enemyPos)
			if distance < nearestDistance {
				nearestDistance = distance
				nearestEnemy = squadID
			}
		}
	}

	return nearestEnemy, nearestDistance
}

// evaluateAttacks generates and scores attack actions
func (ae *ActionEvaluator) evaluateAttacks() []ScoredAction {
	var actions []ScoredAction

	// Get all enemy squads in range
	targets := ae.getAttackableTargets()

	for _, targetID := range targets {
		score := ae.scoreAttackTarget(targetID)

		actions = append(actions, ScoredAction{
			Action: &AttackAction{
				attackerID:   ae.ctx.SquadID,
				targetID:     targetID,
				aiController: ae.ctx.AIController, // Pass reference for queueing
			},
			Score:       score,
			Description: fmt.Sprintf("Attack squad %d", targetID),
		})
	}

	return actions
}

// getAttackableTargets returns all enemy squads we can attack
func (ae *ActionEvaluator) getAttackableTargets() []ecs.EntityID {
	var targets []ecs.EntityID

	// Get all enemy factions
	allFactions := combat.GetAllFactions(ae.ctx.Manager)

	for _, factionID := range allFactions {
		if factionID == ae.ctx.FactionID {
			continue // Skip own faction
		}

		squadIDs := combat.GetSquadsForFaction(factionID, ae.ctx.Manager)

		for _, squadID := range squadIDs {
			if squads.IsSquadDestroyed(squadID, ae.ctx.Manager) {
				continue
			}

			// Check if squad is in attack range
			if ae.isInAttackRange(squadID) {
				targets = append(targets, squadID)
			}
		}
	}

	return targets
}

// isInAttackRange checks if target is in attack range
func (ae *ActionEvaluator) isInAttackRange(targetID ecs.EntityID) bool {
	targetPos, err := combat.GetSquadMapPosition(targetID, ae.ctx.Manager)
	if err != nil {
		return false
	}

	distance := ae.ctx.CurrentPos.ChebyshevDistance(&targetPos)

	// Get max attack range from squad's units
	maxRange := ae.getMaxAttackRange()

	return distance <= maxRange
}

// getMaxAttackRange returns maximum attack range of squad's units
func (ae *ActionEvaluator) getMaxAttackRange() int {
	unitIDs := squads.GetUnitIDsInSquad(ae.ctx.SquadID, ae.ctx.Manager)

	maxRange := 1 // Default melee range

	for _, unitID := range unitIDs {
		entity := ae.ctx.Manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		if rangeData != nil {
			if rangeData.Range > maxRange {
				maxRange = rangeData.Range
			}
		}
	}

	return maxRange
}

// scoreAttackTarget scores an attack target
func (ae *ActionEvaluator) scoreAttackTarget(targetID ecs.EntityID) float64 {
	// CRITICAL: Attack base score must be higher than movement base score
	// to ensure AI prefers attacking when targets are in range.
	// Movement base is 50, so attacks start at 100 to always beat movement.
	baseScore := 100.0

	// Prioritize wounded targets (focus fire)
	targetHealth := squads.GetSquadHealthPercent(targetID, ae.ctx.Manager)
	baseScore += (1.0 - targetHealth) * 20.0 // Up to +20 for very wounded targets

	// Prioritize high-threat targets
	targetRole := squads.GetSquadPrimaryRole(targetID, ae.ctx.Manager)
	if targetRole == squads.RoleDPS {
		baseScore += 15.0 // DPS are high priority
	} else if targetRole == squads.RoleSupport {
		baseScore += 10.0 // Support are medium priority
	}

	// Bonus for role counters (DPS good vs Support, Tank good vs DPS)
	if ae.ctx.SquadRole == squads.RoleDPS && targetRole == squads.RoleSupport {
		baseScore += 10.0
	} else if ae.ctx.SquadRole == squads.RoleTank && targetRole == squads.RoleDPS {
		baseScore += 10.0
	}

	return baseScore
}
