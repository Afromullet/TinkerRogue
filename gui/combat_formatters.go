package gui

import (
	"fmt"
	"game_main/combat"
)

// CombatFormatters provides display formatting functions for combat UI
type CombatFormatters struct {
	queries *GUIQueries
	turnMgr *combat.TurnManager
}

// NewCombatFormatters creates a new formatters instance
func NewCombatFormatters(queries *GUIQueries, turnMgr *combat.TurnManager) *CombatFormatters {
	return &CombatFormatters{
		queries: queries,
		turnMgr: turnMgr,
	}
}

// FormatTurnOrder formats the current turn order display
// Shows round number, faction name, and player indicator if applicable
func (cf *CombatFormatters) FormatTurnOrder() string {
	currentFactionID := cf.turnMgr.GetCurrentFaction()
	if currentFactionID == 0 {
		return "No active combat"
	}

	round := cf.turnMgr.GetCurrentRound()
	factionName := cf.queries.GetFactionName(currentFactionID)

	// Add indicator if player's turn
	playerIndicator := ""
	if cf.queries.IsPlayerFaction(currentFactionID) {
		playerIndicator = " >>> YOUR TURN <<<"
	}

	return fmt.Sprintf("Round %d | %s%s", round, factionName, playerIndicator)
}

// FormatFactionInfo formats faction information for display
// Shows faction name, squad count, and current mana
func (cf *CombatFormatters) FormatFactionInfo(factionInfo *FactionInfo) string {
	if factionInfo == nil {
		return "No faction selected"
	}

	infoText := fmt.Sprintf("%s\n", factionInfo.Name)
	infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
	infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)
	return infoText
}
