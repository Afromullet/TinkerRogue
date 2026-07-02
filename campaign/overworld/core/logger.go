package core

import (
	"fmt"

	"game_main/core/config"
)

// OverworldLogger receives overworld debug and warning messages. Mirrors the
// nil-safe seam pattern of powercore.PowerLogger so a GUI event sink can be
// injected later via SetLogger.
type OverworldLogger interface {
	Debugf(format string, args ...any) // trace output, only shown when config.DEBUG_MODE
	Warnf(format string, args ...any)  // data problems, always shown
}

// consoleLogger is the default sink: warnings always print with a WARNING:
// prefix; debug traces are gated on config.DEBUG_MODE.
type consoleLogger struct{}

func (consoleLogger) Debugf(format string, args ...any) {
	if config.DEBUG_MODE {
		fmt.Printf(format+"\n", args...)
	}
}

func (consoleLogger) Warnf(format string, args ...any) {
	fmt.Printf("WARNING: "+format+"\n", args...)
}

var logger OverworldLogger = consoleLogger{}

// SetLogger replaces the overworld log sink (e.g. to route into a GUI log).
func SetLogger(l OverworldLogger) { logger = l }

// Debugf routes a trace message through the installed overworld logger. Nil-safe.
func Debugf(format string, args ...any) {
	if logger != nil {
		logger.Debugf(format, args...)
	}
}

// Warnf routes a warning through the installed overworld logger. Nil-safe.
func Warnf(format string, args ...any) {
	if logger != nil {
		logger.Warnf(format, args...)
	}
}
