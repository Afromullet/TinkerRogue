package rendering

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// TileBatch batches multiple tiles that share the same source image into a single draw call.
// This dramatically reduces draw calls from ~900 (one per tile) to ~15 (one per unique image).
// Performance: Expected 80-90% reduction in tile rendering time.
type TileBatch struct {
	vertices []ebiten.Vertex
	indices  []uint16
	image    *ebiten.Image
}

// NewTileBatch creates a batch for tiles sharing the same image
func NewTileBatch(image *ebiten.Image) *TileBatch {
	return &TileBatch{
		vertices: make([]ebiten.Vertex, 0, TileVerticeBatchSize), // Pre-allocate for ~200 tiles (4 vertices each)
		indices:  make([]uint16, 0, TileIndicesBatchSize),        // Pre-allocate for ~200 tiles (6 indices each)
		image:    image,
	}
}

// AddTile adds a tile to this batch with position, source rect, destination size, and color
func (tb *TileBatch) AddTile(dstX, dstY float32, srcX, srcY, srcW, srcH float32, dstW, dstH float32, r, g, b, a float32) {
	baseIdx := uint16(len(tb.vertices))

	// Create 4 vertices for this tile (forming a quad)
	// DstX/DstY are screen coordinates, SrcX/SrcY are texture coordinates
	// dstW/dstH are the rendered size (may be scaled), srcW/srcH are the texture size
	tb.vertices = append(tb.vertices,
		// Top-left
		ebiten.Vertex{
			DstX:   dstX,
			DstY:   dstY,
			SrcX:   srcX,
			SrcY:   srcY,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		// Top-right
		ebiten.Vertex{
			DstX:   dstX + dstW,
			DstY:   dstY,
			SrcX:   srcX + srcW,
			SrcY:   srcY,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		// Bottom-left
		ebiten.Vertex{
			DstX:   dstX,
			DstY:   dstY + dstH,
			SrcX:   srcX,
			SrcY:   srcY + srcH,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
		// Bottom-right
		ebiten.Vertex{
			DstX:   dstX + dstW,
			DstY:   dstY + dstH,
			SrcX:   srcX + srcW,
			SrcY:   srcY + srcH,
			ColorR: r,
			ColorG: g,
			ColorB: b,
			ColorA: a,
		},
	)

	// Add indices for 2 triangles (forming a quad)
	// Triangle 1: top-left, top-right, bottom-left
	// Triangle 2: top-right, bottom-right, bottom-left
	tb.indices = append(tb.indices,
		baseIdx+0, baseIdx+1, baseIdx+2, // First triangle
		baseIdx+1, baseIdx+3, baseIdx+2, // Second triangle
	)
}

// Draw renders all batched tiles in a single DrawTriangles call
func (tb *TileBatch) Draw(screen *ebiten.Image) {
	if len(tb.vertices) == 0 {
		return // Nothing to draw
	}

	screen.DrawTriangles(tb.vertices, tb.indices, tb.image, nil)
}

// Reset clears the batch for reuse (avoids allocations)
func (tb *TileBatch) Reset() {
	tb.vertices = tb.vertices[:0]
	tb.indices = tb.indices[:0]
}

// TileCount returns the number of tiles in this batch
func (tb *TileBatch) TileCount() int {
	return len(tb.vertices) / 4
}
