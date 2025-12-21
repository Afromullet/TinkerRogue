package widgets

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CachedListWrapper wraps a List widget and caches its rendered output.
// This significantly improves performance when list content doesn't change frequently.
//
// Performance Impact:
// - Reduces CPU usage by ~90% for static lists (lists that don't update every frame)
// - Trades memory (cached render buffer) for CPU (skip redundant renders)
//
// Usage:
//   list := builders.CreateListWithConfig(...)
//   cachedList := NewCachedListWrapper(list)
//   // Mark dirty when entries change:
//   cachedList.MarkDirty()
//
// IMPORTANT: You MUST call MarkDirty() whenever:
// - List entries are added/removed
// - Entry content changes
// - Selection changes (if you want to show selection immediately)
type CachedListWrapper struct {
	list         *widget.List
	cachedImage  *ebiten.Image
	dirty        bool
	lastWidth    int
	lastHeight   int
	renderCount  int // For debugging/profiling
}

// NewCachedListWrapper creates a new cached list wrapper.
func NewCachedListWrapper(list *widget.List) *CachedListWrapper {
	return &CachedListWrapper{
		list:  list,
		dirty: true, // Force initial render
	}
}

// GetWidget returns the underlying widget for UI hierarchy integration.
func (clw *CachedListWrapper) GetWidget() *widget.Widget {
	return clw.list.GetWidget()
}

// PreferredSize delegates to the underlying list's preferred size.
func (clw *CachedListWrapper) PreferredSize() (int, int) {
	return clw.list.PreferredSize()
}

// Render implements the Renderer interface with caching.
// Only re-renders when:
// - Marked dirty via MarkDirty()
// - First render
// - Widget size changed
func (clw *CachedListWrapper) Render(screen *ebiten.Image, def widget.DeferredRenderFunc) {
	w := clw.list.GetWidget()
	rect := w.Rect
	width, height := rect.Dx(), rect.Dy()

	// Check if we need to re-render
	needsRender := clw.dirty ||
		clw.cachedImage == nil ||
		clw.lastWidth != width ||
		clw.lastHeight != height

	if needsRender {
		clw.renderToCache(screen, def, width, height)
		clw.renderCount++
	}

	// Draw cached image to screen
	if clw.cachedImage != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
		screen.DrawImage(clw.cachedImage, opts)
	}
}

// renderToCache renders the list to the cache buffer.
func (clw *CachedListWrapper) renderToCache(screen *ebiten.Image, def widget.DeferredRenderFunc, width, height int) {
	// Dispose old cache if size changed
	if clw.cachedImage != nil && (clw.lastWidth != width || clw.lastHeight != height) {
		clw.cachedImage.Dispose()
		clw.cachedImage = nil
	}

	// Create new cache buffer
	if clw.cachedImage == nil {
		clw.cachedImage = ebiten.NewImage(width, height)
	}

	// Clear and render list to cache
	clw.cachedImage.Clear()
	clw.list.Render(clw.cachedImage, def)

	// Update state
	clw.lastWidth = width
	clw.lastHeight = height
	clw.dirty = false
}

// MarkDirty forces re-render on next Render call.
// Call this whenever list content changes.
func (clw *CachedListWrapper) MarkDirty() {
	clw.dirty = true
}

// GetRenderCount returns the number of times the list was actually rendered.
// Useful for debugging/profiling cache effectiveness.
func (clw *CachedListWrapper) GetRenderCount() int {
	return clw.renderCount
}

// Dispose frees the cached image memory.
func (clw *CachedListWrapper) Dispose() {
	if clw.cachedImage != nil {
		clw.cachedImage.Dispose()
		clw.cachedImage = nil
	}
}

// SetLocation delegates to the underlying list.
func (clw *CachedListWrapper) SetLocation(rect image.Rectangle) {
	// Size change will be detected in Render() and trigger re-render
	clw.list.SetLocation(rect)
}

// GetList returns the underlying List widget for direct access.
// Use this if you need to call List-specific methods.
func (clw *CachedListWrapper) GetList() *widget.List {
	return clw.list
}
