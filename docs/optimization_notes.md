# Below Are Examples of Changes That Resulted in Improved Performance

**1: In parts throughout the project, we constantly updated GUI or ECS related data. Instead  We want to track if the state changed, and only then perform updates**


```go
//Checks the current faction or squad and only update when state changes
func (cm *CombatMode) Update(deltaTime float64) error {
	// Only update UI displays when state changes (GUI_PERFORMANCE_ANALYSIS.md)
	// This avoids expensive text measurement on every frame (~10-15s CPU savings)

	currentFactionID := cm.combatService.GetCurrentFaction()
	if cm.lastFactionID != currentFactionID {
		cm.turnOrderComponent.Refresh()
		cm.lastFactionID = currentFactionID
		if cm.lastFactionID != 0 {
			cm.factionInfoComponent.ShowFaction(cm.lastFactionID)
		}
	}

	battleState := cm.Context.ModeCoordinator.GetBattleMapState()
	if cm.lastSelectedSquad != battleState.SelectedSquadID {
		cm.lastSelectedSquad = battleState.SelectedSquadID
		if cm.lastSelectedSquad != 0 {
			cm.squadDetailComponent.ShowSquad(cm.lastSelectedSquad)
		}
	}

	return nil
```


```go

// Only creating a new viewport when the screen size changes. 
func (mtr *MovementTileRenderer) Render(screen *ebiten.Image, centerPos coords.LogicalPosition, validTiles []coords.LogicalPosition) {
	screenX, screenY := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Only recreate renderer if screen size changed or not yet created
	if mtr.cachedRenderer == nil || mtr.lastScreenSizeX != screenX || mtr.lastScreenSizeY != screenY {
		mtr.cachedRenderer = NewViewportRenderer(screen, centerPos)
		mtr.lastCenterPos = centerPos
		mtr.lastScreenSizeX = screenX
		mtr.lastScreenSizeY = screenY
	} else if mtr.lastCenterPos != centerPos {
		// Just update center position if only that changed
		mtr.cachedRenderer.UpdateCenter(centerPos)
		mtr.lastCenterPos = centerPos
	}

	vr := mtr.cachedRenderer

}



	// Create overlay image only once or when size changes
	if vr.overlayCache == nil || vr.overlayTileSize != tileSize {
		vr.overlayCache = ebiten.NewImage(tileSize, tileSize)
		vr.overlayTileSize = tileSize
	}

	// Fill with color (still necessary but much faster on existing image)
	vr.overlayCache.Fill(fillColor)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(screenX, screenY)
	screen.DrawImage(vr.overlayCache, op)
}

```


## 2 Removed Repeated Struct Creation.

**Multiple Sections of Code Created New Types when they should have only been initialized once**

```go

func NewCoordinateManager(screenData ScreenData) *CoordinateManager {
	cm := &CoordinateManager{
		dungeonWidth:  screenData.DungeonWidth,
		dungeonHeight: screenData.DungeonHeight,
		tileSize:      screenData.TileSize,
		scaleFactor:   screenData.ScaleFactor,
		screenWidth:   screenData.ScreenWidth,
		screenHeight:  screenData.ScreenHeight,
	}

	// Initialize viewport once with origin (will be updated via SetCenter)
	cm.viewport = NewViewport(cm, LogicalPosition{X: 0, Y: 0})

	return cm
}

```



## 3 Better Usage of the ECS Library ###

**Using ECS Views** 


```go
type GUIQueries struct {
	ECSManager     *common.EntityManager
	factionManager *combat.FactionManager

	// Cached ECS Views (automatically maintained by ECS library)
	squadView       *ecs.View // All SquadTag entities
	squadMemberView *ecs.View // All SquadMemberTag entities
	actionStateView *ecs.View // All ActionStateTag entities
}

//Another example of using views. Since renderable is also a component, we can create
//A view for it and improve map drawing performance
func NewRenderingCache(manager *common.EntityManager) *RenderingCache {
	return &RenderingCache{
		// Create View - one-time O(n) cost
		// View is automatically maintained when RenderableComponent added/removed
		RenderablesView: manager.World.CreateView(RenderablesTag),
	}
}

```

## 4 Better Usage of the Ebiten Library


**Minimizing image creation. Filling colors. NewImage is expensive. Geometric operations are faster**

```go

// Border images are filled with white and should be tinted using colorscale.
//Geometric Manipulations (I.E, scale, translate, etc) are usually faster.
func (cache *BorderImageCache) GetOrCreate(tileSize, thickness int) (*ebiten.Image, *ebiten.Image, *ebiten.Image, *ebiten.Image) {
	if cache.top == nil || cache.tileSize != tileSize || cache.thickness != thickness {
		// Recreate images only on size change
		cache.top = ebiten.NewImage(tileSize, thickness)
		cache.bottom = ebiten.NewImage(tileSize, thickness)
		cache.left = ebiten.NewImage(thickness, tileSize)
		cache.right = ebiten.NewImage(thickness, tileSize)

		// Fill with white once - will be tinted with ColorScale in DrawTileBorder
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		cache.top.Fill(white)
		cache.bottom.Fill(white)
		cache.left.Fill(white)
		cache.right.Fill(white)

		cache.tileSize = tileSize
		cache.thickness = thickness
	}

func (vr *ViewportRenderer) DrawTileBorder(screen *ebiten.Image, pos coords.LogicalPosition, borderColor color.Color, thickness int) {
	screenX, screenY := vr.LogicalToScreen(pos)
	tileSize := vr.TileSize()

	// Get cached white border images (created only on first use or size change)
	topBorder, bottomBorder, leftBorder, rightBorder := vr.borderImages.GetOrCreate(tileSize, thickness)

	// Convert border color to ColorScale for GPU-based tinting
	r, g, b, a := borderColor.RGBA()
	colorScale := ebiten.ColorScale{}
	colorScale.SetR(float32(r) / 0xffff)
	colorScale.SetG(float32(g) / 0xffff)
	colorScale.SetB(float32(b) / 0xffff)
	colorScale.SetA(float32(a) / 0xffff)

	// Top border - reuse draw options
	vr.borderDrawOpts[0].GeoM.Reset()
	vr.borderDrawOpts[0].GeoM.Translate(screenX, screenY)
	vr.borderDrawOpts[0].ColorScale = colorScale
	screen.DrawImage(topBorder, &vr.borderDrawOpts[0])

	// Bottom border - reuse draw options
	vr.borderDrawOpts[1].GeoM.Reset()
	vr.borderDrawOpts[1].GeoM.Translate(screenX, screenY+float64(tileSize-thickness))
	vr.borderDrawOpts[1].ColorScale = colorScale
	screen.DrawImage(bottomBorder, &vr.borderDrawOpts[1])

	// Left border - reuse draw options
	vr.borderDrawOpts[2].GeoM.Reset()
	vr.borderDrawOpts[2].GeoM.Translate(screenX, screenY)
	vr.borderDrawOpts[2].ColorScale = colorScale
	screen.DrawImage(leftBorder, &vr.borderDrawOpts[2])

	// Right border - reuse draw options
	vr.borderDrawOpts[3].GeoM.Reset()
	vr.borderDrawOpts[3].GeoM.Translate(screenX+float64(tileSize-thickness), screenY)
	vr.borderDrawOpts[3].ColorScale = colorScale
	screen.DrawImage(rightBorder, &vr.borderDrawOpts[3])
}
```


## 4 Better Usage of EbitenUI

**Creating GUI Widgets Only Once/Recreating Widgets Only When We need to**

```go

// Refresh updates the container with current squad buttons
// OPTIMIZED: Uses widget caching to avoid recreating buttons every frame
func (slc *SquadListComponent) Refresh() {
	if slc.container == nil {
		return
	}

	// Get all squads and apply filter
	allSquads := slc.queries.SquadCache.FindAllSquads()
	newFilteredSquads := make([]ecs.EntityID, 0, len(allSquads))

	for _, squadID := range allSquads {
		squadInfo := slc.queries.GetSquadInfo(squadID)
		if squadInfo == nil || !slc.filter(squadInfo) {
			continue
		}
		newFilteredSquads = append(newFilteredSquads, squadID)
	}

	// OPTIMIZATION: Check if squad list changed
	if !slc.squadListChanged(newFilteredSquads) {
		// FAST PATH: No change - just update button labels if needed
		slc.updateButtonLabels(newFilteredSquads)
		return
	}

	// SLOW PATH: Squad list changed - update widgets
	slc.updateButtonWidgets(newFilteredSquads)
	slc.filteredSquads = newFilteredSquads
}


// updateButtonLabels updates button text without recreating widgets (FAST)
func (slc *SquadListComponent) updateButtonLabels(squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		button, exists := slc.buttons[squadID]
		if !exists {
			continue
		}

		squadInfo := slc.queries.GetSquadInfo(squadID)
		if squadInfo == nil {
			continue
		}

		// Update button text if it changed (Text widget will remeasure on next render, not now)
		textWidget := button.Text()
		if textWidget != nil && textWidget.Label != squadInfo.Name {
			textWidget.Label = squadInfo.Name
		}
	}
}
```

## 5 Batching Drawing Operations

**Tile Drawing Can be Batched due tiles of the same biome using the same image**

See tilebatch.go and tilerenderer.go