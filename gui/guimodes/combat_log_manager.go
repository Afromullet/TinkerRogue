package guimodes

import (
	"github.com/ebitenui/ebitenui/widget"
)

// CombatLogManager handles combat log entries and text updates
type CombatLogManager struct {
	entries               []string
	messageCountSinceTrim int
	maxMessages           int
	trimThreshold         int
}

// NewCombatLogManager creates a new combat log manager
func NewCombatLogManager() *CombatLogManager {
	return &CombatLogManager{
		entries:               make([]string, 0, 100),
		messageCountSinceTrim: 0,
		maxMessages:           300,
		trimThreshold:         100,
	}
}

// AddEntry appends a new message to the combat log
func (clm *CombatLogManager) AddEntry(message string) {
	clm.entries = append(clm.entries, message)
	clm.messageCountSinceTrim++
}

// UpdateTextArea updates the text area with new message and triggers trim if needed
func (clm *CombatLogManager) UpdateTextArea(logArea *widget.TextArea, message string) {
	clm.AddEntry(message)

	// Use AppendText for O(1) performance - only add the new message
	logArea.AppendText(message + "\n")

	// Every N messages, trim old entries to prevent unbounded growth
	if clm.messageCountSinceTrim >= clm.trimThreshold {
		clm.Trim(logArea)
	}
}

// Trim keeps only the last maxMessages entries and rebuilds the display
func (clm *CombatLogManager) Trim(logArea *widget.TextArea) {
	if len(clm.entries) > clm.maxMessages {
		// Remove oldest messages, keep most recent ones
		removed := len(clm.entries) - clm.maxMessages
		clm.entries = clm.entries[removed:]

		// Rebuild the text area display with trimmed content
		fullText := ""
		for _, msg := range clm.entries {
			fullText += msg + "\n"
		}
		logArea.SetText(fullText)
	}

	clm.messageCountSinceTrim = 0
}

// GetEntries returns a copy of all log entries
func (clm *CombatLogManager) GetEntries() []string {
	entries := make([]string, len(clm.entries))
	copy(entries, clm.entries)
	return entries
}

// GetEntryCount returns the number of entries in the log
func (clm *CombatLogManager) GetEntryCount() int {
	return len(clm.entries)
}

// Clear removes all entries from the log
func (clm *CombatLogManager) Clear() {
	clm.entries = clm.entries[:0]
	clm.messageCountSinceTrim = 0
}
