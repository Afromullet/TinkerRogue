package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadBattleRecord reads and parses a battle JSON file.
func LoadBattleRecord(path string) (*BattleRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var record BattleRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &record, nil
}

// FindLatestBattle finds the most recent JSON file in the specified directory.
// Returns the full path to the file.
func FindLatestBattle(dir string) (string, error) {
	files, err := findBattleFiles(dir)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no battle files found in %s", dir)
	}

	// Files are already sorted by name (which includes timestamp)
	// Return the last one (most recent)
	return files[len(files)-1], nil
}

// FindAllBattles finds all JSON battle files in the specified directory.
// Returns a sorted list of full paths.
func FindAllBattles(dir string) ([]string, error) {
	return findBattleFiles(dir)
}

// findBattleFiles finds all JSON files in the directory and returns sorted paths.
func findBattleFiles(dir string) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Read directory entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Collect JSON files
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			fullPath := filepath.Join(dir, name)
			files = append(files, fullPath)
		}
	}

	// Sort by filename (battle_YYYYMMDD_HHMMSS.json sorts chronologically)
	sort.Strings(files)

	return files, nil
}
