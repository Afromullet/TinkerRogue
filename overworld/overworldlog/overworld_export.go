package overworldlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ExportOverworldJSON writes the overworld record to a JSON file.
// Creates the output directory if it doesn't exist.
func ExportOverworldJSON(record *OverworldRecord, outputDir string) error {
	if record == nil {
		return fmt.Errorf("cannot export nil overworld record")
	}

	// Ensure output directory exists
	if err := ensureOutputDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := generateOverworldFilename(record)
	filePath := filepath.Join(outputDir, filename)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal overworld record: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write overworld log file: %w", err)
	}

	fmt.Printf("Overworld log exported to: %s\n", filePath)
	return nil
}

// generateOverworldFilename creates a filename from the overworld record.
// Format: journey_YYYYMMDD_HHMMSS.mmm.json
func generateOverworldFilename(record *OverworldRecord) string {
	if record.SessionID != "" {
		return record.SessionID + ".json"
	}
	// Fallback using end time
	return fmt.Sprintf("journey_%s.json", record.EndTime.Format("20060102_150405.000"))
}

// ensureOutputDir creates the output directory if it doesn't exist.
func ensureOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	return os.MkdirAll(dir, 0755)
}
