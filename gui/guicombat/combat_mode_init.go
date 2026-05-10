package guicombat

import (
	"fmt"

	"game_main/gui/framework"
	"game_main/gui/guiartifacts"
	"game_main/gui/guicombat/combatbase"
	"game_main/gui/guicombat/combatinput"
	"game_main/gui/guicombat/combatvisualization"
	"game_main/gui/guiinspect"
	"game_main/gui/guispells"
	"game_main/gui/guisquads"
	"game_main/gui/widgets"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/spells"

	"github.com/bytearena/ecs"
	"github.com/ebitenui/ebitenui/widget"
)

func (cm *CombatMode) Initialize(ctx *framework.UIContext) error {
	// Create combat service via injected factory (wires AI + threat stack)
	cm.combatService = cm.serviceFactory(ctx.ECSManager)

	// Build UI using ModeBuilder (minimal config - panels handled by registry)
	err := framework.NewModeBuilder(&cm.BaseMode, framework.ModeConfig{
		ModeName:   "combat",
		ReturnMode: "exploration",
	}).Build(ctx)
	if err != nil {
		return err
	}

	// Initialize action map for semantic keybindings
	cm.actionMap = framework.DefaultCombatBindings()

	// Initialize sub-menu controller before building panels (panels register with it).
	// Pass RootContainer so hidden panels are fully removed from the widget tree,
	// preventing their ScrollContainers from blocking input on overlapping panels.
	cm.subMenus = framework.NewSubMenuController(cm.RootContainer)

	// Build panels using registry
	if err := cm.buildPanelsFromRegistry(); err != nil {
		return err
	}

	// Remove initially-hidden sub-menu panels from widget tree.
	// BuildPanels adds all panels to RootContainer; CloseAll removes the
	// sub-menu panels so they don't block input until explicitly shown.
	cm.subMenus.CloseAll()

	// Build action button clusters (needs callbacks, so done separately)
	cm.RootContainer.AddChild(cm.buildContextActions())
	cm.RootContainer.AddChild(cm.buildNavigationActions())

	// Create consolidated dependencies for handlers
	cm.deps = combatbase.NewCombatModeDeps(
		ctx.ModeCoordinator.GetTacticalState(),
		cm.combatService,
		cm.Queries,
		cm.ModeManager,
		cm.encounterController,
	)

	// Create handlers with deps
	cm.actionHandler = combatbase.NewCombatActionHandler(cm.deps)
	cm.inputHandler = combatinput.NewCombatInputHandler(cm.actionHandler, cm.deps)

	// Create spell handler and panel controller
	spellDeps := &guispells.SpellCastingDeps{
		BattleState: cm.deps.BattleState,
		ECSManager:  cm.deps.Queries.ECSManager,
		GameMap:     ctx.GameMap,
		PlayerPos:   ctx.PlayerData.Pos,
		Queries:     cm.deps.Queries,
		Encounter:   cm.deps.Encounter,
	}
	spellHandler := guispells.NewSpellCastingHandler(spellDeps)
	cm.spellPanel = guispells.NewSpellPanelController(&guispells.SpellPanelDeps{
		Handler:      spellHandler,
		BattleState:  cm.deps.BattleState,
		ShowSubmenu:  func() { cm.subMenus.Show("spell") },
		CloseSubmenu: func() { cm.subMenus.CloseAll() },
	})

	// Extract spell panel widget references
	cm.spellPanel.SetWidgets(
		framework.GetPanelWidget[*widgets.CachedListWrapper](cm.Panels, combatbase.CombatPanelSpellSelection, "spellList"),
		framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](cm.Panels, combatbase.CombatPanelSpellSelection, "detailArea"),
		framework.GetPanelWidget[*widget.Text](cm.Panels, combatbase.CombatPanelSpellSelection, "manaLabel"),
		framework.GetPanelWidget[*widget.Button](cm.Panels, combatbase.CombatPanelSpellSelection, "castButton"),
	)

	// Wire spell panel into input handler
	cm.inputHandler.SetSpellPanel(cm.spellPanel)

	// Create artifact handler and panel controller
	artifactDeps := &guiartifacts.ArtifactActivationDeps{
		BattleState:   cm.deps.BattleState,
		CombatService: cm.deps.CombatService,
		Queries:       cm.deps.Queries,
		Encounter:     cm.deps.Encounter,
	}
	artifactHandler := guiartifacts.NewArtifactActivationHandler(artifactDeps)
	artifactHandler.SetPlayerPosition(ctx.PlayerData.Pos)
	cm.artifactPanel = guiartifacts.NewArtifactPanelController(&guiartifacts.ArtifactPanelDeps{
		Handler:      artifactHandler,
		BattleState:  cm.deps.BattleState,
		ShowSubmenu:  func() { cm.subMenus.Show("artifact") },
		CloseSubmenu: func() { cm.subMenus.CloseAll() },
	})

	// Extract artifact panel widget references
	cm.artifactPanel.SetWidgets(
		framework.GetPanelWidget[*widgets.CachedListWrapper](cm.Panels, combatbase.CombatPanelArtifactSelection, "artifactList"),
		framework.GetPanelWidget[*widgets.CachedTextAreaWrapper](cm.Panels, combatbase.CombatPanelArtifactSelection, "detailArea"),
		framework.GetPanelWidget[*widget.Button](cm.Panels, combatbase.CombatPanelArtifactSelection, "activateButton"),
	)

	// Wire artifact panel into input handler
	cm.inputHandler.SetArtifactPanel(cm.artifactPanel)

	// Create inspect panel controller and wire into input handler
	inspectController := guiinspect.NewInspectPanelController(cm.Queries)
	inspectResult := cm.Panels.Get(guiinspect.InspectPanelType)
	if inspectResult != nil {
		inspectController.SetWidgets(
			framework.GetPanelWidget[*widget.Text](cm.Panels, guiinspect.InspectPanelType, "squadNameLabel"),
			framework.GetPanelWidget[[3][3]*widget.Button](cm.Panels, guiinspect.InspectPanelType, "gridCells"),
			framework.GetPanelWidget[[3][3]*widget.Button](cm.Panels, guiinspect.InspectPanelType, "attackGridCells"),
			inspectResult.Container,
		)
	}
	cm.inputHandler.SetInspectPanel(inspectController)

	// Register cache invalidation callbacks (automatic, fires for both GUI and AI actions)
	cm.registerCombatCallbacks()

	// Initialize visualization systems
	cm.visualization = combatvisualization.NewCombatVisualizationManager(ctx, cm.Queries, ctx.GameMap, cm.combatService)

	// Wire visualization input support into input handler
	cm.inputHandler.SetVisualization(cm.visualization, cm.Panels)

	// Initialize turn flow manager (before initializeUpdateComponents which sets UI refs on it)
	cm.turnFlow = NewCombatTurnFlow(
		cm.combatService,
		cm.visualization,
		cm.actionHandler,
		cm.Queries,
		cm.ModeManager,
		cm.Panels,
		ctx,
	)

	cm.initializeUpdateComponents()

	return nil
}

func (cm *CombatMode) initializeUpdateComponents() {
	// Get widgets from registry
	turnOrderLabel := cm.GetTextLabel(combatbase.CombatPanelTurnOrder)
	factionInfoText := cm.GetTextLabel(combatbase.CombatPanelFactionInfo)
	squadDetailText := cm.GetTextLabel(combatbase.CombatPanelSquadDetail)

	// Turn order component - displays current faction and round
	cm.turnOrderComponent = widgets.NewTextDisplayComponent(
		turnOrderLabel,
		func() string {
			currentFactionID := cm.combatService.TurnManager.GetCurrentFaction()
			if currentFactionID == 0 {
				return "No active combat"
			}

			round := cm.combatService.TurnManager.GetCurrentRound()
			factionData := cm.Queries.CombatCache.FindFactionDataByID(currentFactionID)
			factionName := "Unknown"
			turnIndicator := ""

			if factionData != nil {
				factionName = factionData.Name

				if factionData.PlayerID > 0 {
					turnIndicator = fmt.Sprintf(" >>> %s's TURN <<<", factionData.PlayerName)
				} else {
					turnIndicator = " [AI TURN]"
				}
			}

			return fmt.Sprintf("Round %d | %s%s", round, factionName, turnIndicator)
		},
	)

	// Faction info component - displays squad count and commander mana
	cm.factionInfoComponent = guisquads.NewDetailPanelComponent(
		factionInfoText,
		cm.Queries,
		func(data interface{}) string {
			factionInfo := data.(*framework.FactionInfo)
			factionData := cm.Queries.CombatCache.FindFactionDataByID(factionInfo.ID)

			infoText := fmt.Sprintf("%s\n", factionInfo.Name)

			if factionData != nil && factionData.PlayerID > 0 {
				infoText += fmt.Sprintf("[%s]\n", factionData.PlayerName)
			}

			infoText += fmt.Sprintf("Squads: %d/%d\n", factionInfo.AliveSquadCount, len(factionInfo.SquadIDs))
			infoText += fmt.Sprintf("Mana: %d/%d", factionInfo.CurrentMana, factionInfo.MaxMana)

			// Show selected squad's mana if it has a spell pool
			selectedSquadID := cm.deps.BattleState.SelectedSquadID
			if selectedSquadID != 0 {
				manaData := spells.GetManaData(selectedSquadID, cm.Queries.ECSManager)
				if manaData != nil {
					infoText += fmt.Sprintf("\nSquad Mana: %d/%d", manaData.CurrentMana, manaData.MaxMana)
				}
			}
			return infoText
		},
	)

	// Squad detail component - displays selected squad details
	cm.squadDetailComponent = guisquads.NewDetailPanelComponent(
		squadDetailText,
		cm.Queries,
		nil, // Use default formatter
	)

	// Pass UI component refs to turn flow manager
	cm.turnFlow.SetUIComponents(
		cm.turnOrderComponent,
		cm.factionInfoComponent,
		cm.squadDetailComponent,
	)
}

// registerCombatCallbacks sets GUI callbacks on the combat service for cache invalidation.
func (cm *CombatMode) registerCombatCallbacks() {
	cm.combatService.SetOnAttackCompleteGUI(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
		cm.Queries.MarkSquadDirty(attackerID)
		cm.Queries.MarkSquadDirty(defenderID)
		if result.AttackerDestroyed {
			cm.Queries.InvalidateSquad(attackerID)
		}
		if result.TargetDestroyed {
			cm.Queries.InvalidateSquad(defenderID)
		}
	})

	cm.combatService.SetOnMoveCompleteGUI(func(squadID ecs.EntityID) {
		cm.Queries.MarkSquadDirty(squadID)
	})

	cm.combatService.SetOnTurnEndGUI(func(round int) {
		cm.Queries.MarkAllSquadsDirty()
		cm.visualization.UpdateThreatManagers()
		cm.visualization.UpdateThreatEvaluator()

		// Close any open sub-menus (inspect, spell, artifact panels)
		cm.subMenus.CloseAll()
	})
}
