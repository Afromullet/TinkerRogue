package framework

import (
	"fmt"
	"strings"
	"time"

	"github.com/bytearena/ecs"
)

// DebugLogger provides structured debug logging for UI modes.
// Enable/disable with a single flag. All logging includes mode context.
type DebugLogger struct {
	modeName string
	enabled  bool
}

// NewDebugLogger creates a debug logger for a specific mode.
// Disabled by default - call Enable() to turn on.
func NewDebugLogger(modeName string) *DebugLogger {
	return &DebugLogger{
		modeName: modeName,
		enabled:  false,
	}
}

// Enable turns on debug logging
func (dl *DebugLogger) Enable() {
	dl.enabled = true
}

// Disable turns off debug logging
func (dl *DebugLogger) Disable() {
	dl.enabled = false
}

// Toggle switches debug logging on/off
func (dl *DebugLogger) Toggle() bool {
	dl.enabled = !dl.enabled
	return dl.enabled
}

// IsEnabled returns whether debug logging is active
func (dl *DebugLogger) IsEnabled() bool {
	return dl.enabled
}

// Log outputs a simple debug message
func (dl *DebugLogger) Log(message string) {
	if !dl.enabled {
		return
	}
	timestamp := time.Now().Format("15:04:05.000")
	fmt.Printf("[%s][%s] %s\n", timestamp, dl.modeName, message)
}

// Logf outputs a formatted debug message
func (dl *DebugLogger) Logf(format string, args ...interface{}) {
	if !dl.enabled {
		return
	}
	dl.Log(fmt.Sprintf(format, args...))
}

// LogAction logs a user action with context
func (dl *DebugLogger) LogAction(action string, context map[string]interface{}) {
	if !dl.enabled {
		return
	}
	contextStr := formatContext(context)
	dl.Logf("ACTION: %s | %s", action, contextStr)
}

// LogStateChange logs a state transition
func (dl *DebugLogger) LogStateChange(field string, oldValue, newValue interface{}) {
	if !dl.enabled {
		return
	}
	dl.Logf("STATE: %s changed from %v to %v", field, oldValue, newValue)
}

// LogError logs an error with context
func (dl *DebugLogger) LogError(action string, err error, context map[string]interface{}) {
	if !dl.enabled {
		return
	}
	contextStr := formatContext(context)
	dl.Logf("ERROR: %s failed: %v | %s", action, err, contextStr)
}

// LogEntityAction logs an action involving entities
func (dl *DebugLogger) LogEntityAction(action string, entityID ecs.EntityID, details string) {
	if !dl.enabled {
		return
	}
	dl.Logf("ENTITY[%d]: %s - %s", entityID, action, details)
}

// LogModeTransition logs mode enter/exit
func (dl *DebugLogger) LogModeTransition(direction string, otherMode string) {
	if !dl.enabled {
		return
	}
	dl.Logf("MODE: %s %s (other=%s)", direction, dl.modeName, otherMode)
}

// formatContext converts a context map to a readable string
func formatContext(context map[string]interface{}) string {
	if len(context) == 0 {
		return "(no context)"
	}

	parts := make([]string, 0, len(context))
	for k, v := range context {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

// DebugState provides a snapshot of state for debugging
type DebugState struct {
	Fields map[string]interface{}
}

// NewDebugState creates an empty debug state
func NewDebugState() *DebugState {
	return &DebugState{
		Fields: make(map[string]interface{}),
	}
}

// Set adds a field to the debug state
func (ds *DebugState) Set(key string, value interface{}) *DebugState {
	ds.Fields[key] = value
	return ds
}

// String returns a formatted string of all state fields
func (ds *DebugState) String() string {
	if len(ds.Fields) == 0 {
		return "(empty state)"
	}
	return formatContext(ds.Fields)
}

// LogState logs the current debug state
func (dl *DebugLogger) LogState(label string, state *DebugState) {
	if !dl.enabled {
		return
	}
	dl.Logf("STATE[%s]: %s", label, state.String())
}
