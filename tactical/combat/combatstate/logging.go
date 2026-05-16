package combatstate

import (
	"fmt"

	"github.com/bytearena/ecs"
)

// LogFunc is the signature combatstate uses to emit gear/state log messages.
// It matches powercore.PowerLogger.Log; the dependency direction prevents a
// direct import (powercore imports combatstate, not vice versa).
type LogFunc func(source string, squadID ecs.EntityID, message string)

// gearLogger is invoked for gear/bonus-attack messages emitted from this
// package. Defaults to stdout to preserve historical [GEAR] output.
var gearLogger LogFunc = func(source string, squadID ecs.EntityID, message string) {
	fmt.Printf("[GEAR] %s: %s (squad %d)\n", source, message, squadID)
}

// SetGearLogger injects the logger combatstate emits gear messages through.
// Passing nil restores the default stdout logger.
func SetGearLogger(fn LogFunc) {
	if fn == nil {
		gearLogger = func(source string, squadID ecs.EntityID, message string) {
			fmt.Printf("[GEAR] %s: %s (squad %d)\n", source, message, squadID)
		}
		return
	}
	gearLogger = fn
}
