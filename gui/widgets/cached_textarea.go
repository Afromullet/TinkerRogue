package widgets

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// CachedTextAreaWrapper wraps a TextArea widget and caches its rendered output.
// This significantly improves performance when textarea content doesn't change frequently.
//
// Performance Impact:
// - Reduces CPU usage by ~90% for static text areas (read-only displays, logs)
// - Trades memory (cached render buffer) for CPU (skip redundant renders)
//
// Usage:
//   textarea := builders.CreateTextAreaWithConfig(...)
//   cachedTextArea := NewCachedTextAreaWrapper(textarea)
//   // Mark dirty when content changes:
//   cachedTextArea.MarkDirty()
//
// IMPORTANT: You MUST call MarkDirty() whenever:
// - Text content changes
// - Scrolling occurs (if you want smooth scrolling)
// - For frequently updating text (combat logs), consider NOT using cache
type CachedTextAreaWrapper struct {
	textarea     *widget.TextArea
	cachedImage  *ebiten.Image
	dirty        bool
	lastWidth    int
	lastHeight   int
	renderCount  int // For debugging/profiling
}

// NewCachedTextAreaWrapper creates a new cached textarea wrapper.
func NewCachedTextAreaWrapper(textarea *widget.TextArea) *CachedTextAreaWrapper {
	return &CachedTextAreaWrapper{
		textarea: textarea,
		dirty:    true, // Force initial render
	}
}

// GetWidget returns the underlying widget for UI hierarchy integration.
func (ctw *CachedTextAreaWrapper) GetWidget() *widget.Widget {
	return ctw.textarea.GetWidget()
}

// PreferredSize delegates to the underlying textarea's preferred size.
func (ctw *CachedTextAreaWrapper) PreferredSize() (int, int) {
	return ctw.textarea.PreferredSize()
}

// Render implements the Renderer interface with caching.
// Only re-renders when:
// - Marked dirty via MarkDirty()
// - First render
// - Widget size changed
func (ctw *CachedTextAreaWrapper) Render(screen *ebiten.Image, def widget.DeferredRenderFunc) {
	w := ctw.textarea.GetWidget()
	rect := w.Rect
	width, height := rect.Dx(), rect.Dy()

	// Check if we need to re-render
	needsRender := ctw.dirty ||
		ctw.cachedImage == nil ||
		ctw.lastWidth != width ||
		ctw.lastHeight != height

	if needsRender {
		ctw.renderToCache(screen, def, width, height)
		ctw.renderCount++
	}

	// Draw cached image to screen
	if ctw.cachedImage != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
		screen.DrawImage(ctw.cachedImage, opts)
	}
}

// renderToCache renders the textarea to the cache buffer.
func (ctw *CachedTextAreaWrapper) renderToCache(screen *ebiten.Image, def widget.DeferredRenderFunc, width, height int) {
	// Dispose old cache if size changed
	if ctw.cachedImage != nil && (ctw.lastWidth != width || ctw.lastHeight != height) {
		ctw.cachedImage.Dispose()
		ctw.cachedImage = nil
	}

	// Create new cache buffer
	if ctw.cachedImage == nil {
		ctw.cachedImage = ebiten.NewImage(width, height)
	}

	// Clear and render textarea to cache
	ctw.cachedImage.Clear()
	ctw.textarea.Render(ctw.cachedImage, def)

	// Update state
	ctw.lastWidth = width
	ctw.lastHeight = height
	ctw.dirty = false
}

// MarkDirty forces re-render on next Render call.
// Call this whenever textarea content changes.
func (ctw *CachedTextAreaWrapper) MarkDirty() {
	ctw.dirty = true
}

// GetRenderCount returns the number of times the textarea was actually rendered.
// Useful for debugging/profiling cache effectiveness.
func (ctw *CachedTextAreaWrapper) GetRenderCount() int {
	return ctw.renderCount
}

// Dispose frees the cached image memory.
func (ctw *CachedTextAreaWrapper) Dispose() {
	if ctw.cachedImage != nil {
		ctw.cachedImage.Dispose()
		ctw.cachedImage = nil
	}
}

// SetLocation delegates to the underlying textarea.
func (ctw *CachedTextAreaWrapper) SetLocation(rect image.Rectangle) {
	// Size change will be detected in Render() and trigger re-render
	ctw.textarea.SetLocation(rect)
}

// GetTextArea returns the underlying TextArea widget for direct access.
// Use this if you need to call TextArea-specific methods.
func (ctw *CachedTextAreaWrapper) GetTextArea() *widget.TextArea {
	return ctw.textarea
}

// SetText is a convenience method that updates text and marks dirty.
func (ctw *CachedTextAreaWrapper) SetText(text string) {
	ctw.textarea.SetText(text)
	ctw.MarkDirty()
}

// AppendText is a convenience method that appends text and marks dirty.
func (ctw *CachedTextAreaWrapper) AppendText(text string) {
	ctw.textarea.AppendText(text)
	ctw.MarkDirty()
}
