package gui

// LayoutConfig provides responsive positioning based on screen resolution
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

// TopRightPanel returns position and size for top-right panel (stats)
func (lc *LayoutConfig) TopRightPanel() (x, y, width, height int) {
	width = int(float64(lc.ScreenWidth) * 0.15)                    // 15% of screen width
	height = int(float64(lc.ScreenHeight) * 0.2)                   // 20% of screen height
	x = lc.ScreenWidth - width - int(float64(lc.ScreenWidth)*0.01) // 1% margin
	y = int(float64(lc.ScreenHeight) * 0.01)                       // 1% margin from top
	return
}

// BottomRightPanel returns position and size for bottom-right panel (messages)
func (lc *LayoutConfig) BottomRightPanel() (x, y, width, height int) {
	width = int(float64(lc.ScreenWidth) * 0.15)                       // 15% of screen width
	height = int(float64(lc.ScreenHeight) * 0.15)                     // 15% of screen height
	x = lc.ScreenWidth - width - int(float64(lc.ScreenWidth)*0.01)    // 1% margin
	y = lc.ScreenHeight - height - int(float64(lc.ScreenHeight)*0.01) // 1% margin from bottom
	return
}

// BottomCenterButtons returns position for bottom-center button row
func (lc *LayoutConfig) BottomCenterButtons() (x, y int) {
	buttonRowWidth := int(float64(lc.ScreenWidth) * 0.25)    // 25% of screen width
	x = (lc.ScreenWidth - buttonRowWidth) / 2                // Centered horizontally
	y = lc.ScreenHeight - int(float64(lc.ScreenHeight)*0.08) // 8% from bottom
	return
}

// TopCenterPanel returns position and size for top-center panel (turn order)
func (lc *LayoutConfig) TopCenterPanel() (x, y, width, height int) {
	width = int(float64(lc.ScreenWidth) * 0.3)    // 30% of screen width
	height = int(float64(lc.ScreenHeight) * 0.08) // 8% of screen height
	x = (lc.ScreenWidth - width) / 2              // Centered horizontally
	y = int(float64(lc.ScreenHeight) * 0.01)      // 1% margin from top
	return
}

// RightSidePanel returns position and size for right-side panel (combat log)
func (lc *LayoutConfig) RightSidePanel() (x, y, width, height int) {
	width = int(float64(lc.ScreenWidth) * 0.2)                     // 20% of screen width
	height = lc.ScreenHeight - int(float64(lc.ScreenHeight)*0.15)  // Almost full height with margins
	x = lc.ScreenWidth - width - int(float64(lc.ScreenWidth)*0.01) // 1% margin
	y = int(float64(lc.ScreenHeight) * 0.06)                       // Below top panel
	return
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
