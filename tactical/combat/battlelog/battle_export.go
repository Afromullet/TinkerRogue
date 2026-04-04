package battlelog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExportBattleJSON writes the battle record to a JSON file.
// Creates the output directory if it doesn't exist.
func ExportBattleJSON(record *BattleRecord, outputDir string) error {
	if record == nil {
		return fmt.Errorf("cannot export nil battle record")
	}

	// Ensure output directory exists
	if err := ensureOutputDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := generateBattleFilename(record)
	filePath := filepath.Join(outputDir, filename)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal battle record: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write battle log file: %w", err)
	}

	fmt.Printf("Combat log exported to: %s\n", filePath)
	return nil
}

// generateBattleFilename creates a filename from the battle record.
// Format: battle_YYYYMMDD_HHMMSS.json
func generateBattleFilename(record *BattleRecord) string {
	if record.BattleID != "" {
		return record.BattleID + ".json"
	}
	// Fallback using end time
	return fmt.Sprintf("battle_%s.json", record.EndTime.Format("20060102_150405"))
}

// ensureOutputDir creates the output directory if it doesn't exist.
func ensureOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	return os.MkdirAll(dir, 0755)
}
