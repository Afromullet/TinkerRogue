package combatservices

import (
	"fmt"

	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// setupPowerDispatch builds the canonical combat PowerLogger and hands it to
// the orchestrator, which fans it out to artifacts, perks, ability messages,
// and gear messages. After this returns, cs.logger and cs.Powers.Logger() are
// the same instance.
//
// Source-tag → prefix routing:
//
//	known artifact behavior → "[GEAR]"
//	"service"               → "[COMBAT]"
//	otherwise               → "[PERK]"
func setupPowerDispatch(cs *CombatService) {
	logger := powercore.LoggerFunc(func(source string, squadID ecs.EntityID, message string) {
		prefix := classifyLogPrefix(source)
		if squadID == 0 {
			fmt.Printf("%s %s\n", prefix, message)
			return
		}
		fmt.Printf("%s %s: %s (squad %d)\n", prefix, source, message, squadID)
	})

	cs.logger = logger
	cs.Powers.InstallLogger(logger)
}

// classifyLogPrefix maps a log source tag to its display prefix. The mapping
// is centralized here so the combat logger format stays consistent across
// subsystems.
func classifyLogPrefix(source string) string {
	if artifacts.IsRegisteredBehavior(source) {
		return "[GEAR]"
	}
	if source == "service" {
		return "[COMBAT]"
	}
	return "[PERK]"
}
