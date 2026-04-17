package powercore

import "github.com/bytearena/ecs"

// PowerLogger receives activation messages from artifacts and perks. A single
// logger replaces the two previous package-global callbacks (artifactLogger,
// perkLogger), giving combat code one seam to wire.
//
// Source is a short tag identifying where the log came from — e.g. the
// artifact behavior key ("deadlock_shackles") or perk ID ("counterpunch").
// Callers that want categorisation should pair it with a prefix convention
// (the default combat logger prints "[GEAR]" / "[PERK]" based on source).
type PowerLogger interface {
	Log(source string, squadID ecs.EntityID, message string)
}

// LoggerFunc adapts a function to the PowerLogger interface.
type LoggerFunc func(source string, squadID ecs.EntityID, message string)

func (f LoggerFunc) Log(source string, squadID ecs.EntityID, message string) {
	f(source, squadID, message)
}

// Log is a nil-safe helper: callers can invoke ctx.Log(...) without guarding
// on Logger being non-nil.
func (ctx *PowerContext) Log(source string, squadID ecs.EntityID, message string) {
	if ctx == nil || ctx.Logger == nil {
		return
	}
	ctx.Logger.Log(source, squadID, message)
}
