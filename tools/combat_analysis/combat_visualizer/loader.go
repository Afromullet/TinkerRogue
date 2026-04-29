package visualizer

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadBattleRecord reads and parses a battle JSON file into the visualizer's
// local BattleRecord, which carries the per-engagement Summary the renderer
// needs. Path-discovery helpers (FindAllBattles, FindLatestBattle) live in
// the shared package — only this loader stays local because of the
// Summary-augmented EngagementRecord.
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
