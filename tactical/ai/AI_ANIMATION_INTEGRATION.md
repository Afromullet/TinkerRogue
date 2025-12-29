# AI Attack Animation Integration Guide

This guide shows how to integrate AI attack animations into your combat mode.

## Overview

The AI system now queues attacks for animation playback. After an AI turn completes, you can retrieve the queued attacks and play them back using the `CombatAnimationMode` with auto-play enabled.

## System Components

### 1. AIController (tactical/ai/ai_controller.go)
- **`QueuedAttack`** - Struct containing attacker and defender IDs
- **`attackQueue`** - Stores all attacks that occurred during AI turn
- **`QueueAttack()`** - Adds an attack to the queue (called automatically by AttackAction)
- **`GetQueuedAttacks()`** - Returns all queued attacks
- **`HasQueuedAttacks()`** - Checks if animations are pending
- **`ClearAttackQueue()`** - Clears the queue after playback

### 2. CombatAnimationMode (gui/guicombat/combat_animation_mode.go)
- **`autoPlay`** - When true, skips waiting for user input
- **`SetAutoPlay(bool)`** - Enables/disables auto-play mode

## Integration Steps

### In your Combat Mode (gui/guicombat/combatmode.go or similar):

```go
// After calling AIController.DecideFactionTurn()
func (cm *CombatMode) handleAITurn() {
    // 1. Execute AI turn
    aiController := cm.combatService.GetAIController(currentFactionID)
    aiController.DecideFactionTurn(currentFactionID)

    // 2. Check if there are attacks to animate
    if aiController.HasQueuedAttacks() {
        cm.playAIAttackAnimations(aiController)
    } else {
        // No attacks, proceed to next turn
        cm.advanceToNextTurn()
    }
}

// 3. Play queued attack animations
func (cm *CombatMode) playAIAttackAnimations(aiController *AIController) {
    attacks := aiController.GetQueuedAttacks()

    if len(attacks) == 0 {
        cm.advanceToNextTurn()
        return
    }

    // Play first attack (will chain to next via callbacks)
    cm.playNextAIAttack(attacks, 0, aiController)
}

// 4. Recursive callback to play attacks sequentially
func (cm *CombatMode) playNextAIAttack(attacks []QueuedAttack, index int, aiController *AIController) {
    // All attacks played - clear queue and advance turn
    if index >= len(attacks) {
        aiController.ClearAttackQueue()
        cm.advanceToNextTurn()
        return
    }

    attack := attacks[index]

    // Get animation mode
    if animMode, exists := cm.ModeManager.GetMode("combat_animation"); exists {
        if caMode, ok := animMode.(*CombatAnimationMode); ok {
            // Configure for AI attack (auto-play)
            caMode.SetCombatants(attack.AttackerID, attack.DefenderID)
            caMode.SetAutoPlay(true) // CRITICAL: Enable auto-play for AI

            // Set callback to play next attack
            caMode.SetOnComplete(func() {
                // Animation complete - play next attack
                cm.playNextAIAttack(attacks, index+1, aiController)
            })

            // Trigger animation
            cm.ModeManager.RequestTransition(animMode, "AI Attack Animation")
        }
    }
}

// 5. Advance to next turn after animations complete
func (cm *CombatMode) advanceToNextTurn() {
    cm.combatService.TurnManager.NextTurn()
    // ... other turn management logic
}
```

## Example: Simplified Integration

If you want a simpler approach (skip animations for AI):

```go
func (cm *CombatMode) handleAITurn() {
    aiController := cm.combatService.GetAIController(currentFactionID)
    aiController.DecideFactionTurn(currentFactionID)

    // AI attacks already executed (no animations)
    // Just clear the queue and advance
    aiController.ClearAttackQueue()
    cm.advanceToNextTurn()
}
```

## Key Points

1. **Auto-play is mandatory for AI** - Without `SetAutoPlay(true)`, the animation will freeze waiting for user input

2. **Attacks are queued automatically** - `AttackAction.Execute()` calls `QueueAttack()` when successful

3. **Sequential playback** - Use callbacks to chain animations (see `playNextAIAttack` example)

4. **Clear queue when done** - Always call `ClearAttackQueue()` after playback to prevent stale attacks

5. **Non-blocking execution** - Animations happen via mode transitions, not during AI turn execution

## Animation Timing

- **Idle Phase**: 1.0 seconds (both squads visible)
- **Attack Phase**: 2.0 seconds (colored attack animation)
- **Auto-complete**: Immediate (skips waiting phase)
- **Total per attack**: ~3 seconds

## Debugging

Enable console logging to see AI attack flow:
```
[AI] Tank Squad attacked Archer Squad
[AI] Archer Squad was destroyed!
[DEBUG] SetCombatants: attacker=5, defender=3, attacking_units=2
[DEBUG] Render: attackerID=5, defenderID=3
```

## Future Enhancements

- Add configurable animation speed for AI attacks
- Support simultaneous multi-attack animations
- Add sound effects for AI attacks
- Display attack log overlay during AI animations
