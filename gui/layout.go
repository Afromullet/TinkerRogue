package gui

// LayoutConfig provides responsive positioning based on screen resolution.
//
// DEPRECATED METHODS REMOVED:
// The following panel positioning methods have been removed as they are replaced by
// BuildPanel functional options in panelconfig.go:
//   - TopRightPanel(), BottomRightPanel(), TopLeftPanel()
//   - TopCenterPanel(), RightSidePanel(), BottomCenterButtons()
//
// Use instead:
//   panel := panelBuilders.BuildPanel(TopRight(), Size(0.15, 0.2), Padding(0.01))
//
// Remaining methods are still used for non-panel calculations.
type LayoutConfig struct {
	ScreenWidth  int
	ScreenHeight int
	TileSize     int
}

// NewLayoutConfig creates a layout configuration from context
func NewLayoutConfig(ctx *UIContext) *LayoutConfig {
	return &LayoutConfig{
		ScreenWidth:  ctx.ScreenWidth,
		ScreenHeight: ctx.ScreenHeight,
		TileSize:     ctx.TileSize,
	}
}

// CenterWindow returns position and size for centered modal window
func (lc *LayoutConfig) CenterWindow(widthPercent, heightPercent float64) (x, y, width, height int) {
	width = int(float64(lc.ScreenWidth) * widthPercent)
	height = int(float64(lc.ScreenHeight) * heightPercent)
	x = (lc.ScreenWidth - width) / 2
	y = (lc.ScreenHeight - height) / 2
	return
}

// GridLayoutArea returns position and size for 2-column grid layout (squad panels)
func (lc *LayoutConfig) GridLayoutArea() (x, y, width, height int) {
	marginPercent := 0.02 // 2% margins
	width = lc.ScreenWidth - int(float64(lc.ScreenWidth)*marginPercent*2)
	height = lc.ScreenHeight - int(float64(lc.ScreenHeight)*0.12) // Leave space for close button
	x = int(float64(lc.ScreenWidth) * marginPercent)
	y = int(float64(lc.ScreenHeight) * marginPercent)
	return
}
