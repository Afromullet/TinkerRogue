package rendering

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// SpriteBatch batches multiple sprites that share the same source image into a single draw call.
type SpriteBatch struct {
	vertices []ebiten.Vertex
	indices  []uint16
	image    *ebiten.Image
}

// NewSpriteBatch creates a batch for sprites sharing the same image
func NewSpriteBatch(image *ebiten.Image) *SpriteBatch {
	return &SpriteBatch{
		vertices: make([]ebiten.Vertex, 0, 256), // Pre-allocate for ~64 sprites (4 vertices each)
		indices:  make([]uint16, 0, 384),        // Pre-allocate for ~64 sprites (6 indices each)
		image:    image,
	}
}

// AddSprite adds a sprite to this batch with position, source rect, destination size, and color
// dstX, dstY: Screen position where sprite should be drawn
// srcX, srcY, srcW, srcH: Source rectangle in the texture
// dstW, dstH: Destination size (scaled dimensions)
// r, g, b, a: Color modulation (typically 1.0, 1.0, 1.0, 1.0 for normal rendering)
func (sb *SpriteBatch) AddSprite(dstX, dstY float32, srcX, srcY, srcW, srcH float32, dstW, dstH float32, r, g, b, a float32) {
	baseIdx := uint16(len(sb.vertices))

	// Create 4 vertices for this sprite (forming a quad)
	sb.vertices = append(sb.vertices,
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
	sb.indices = append(sb.indices,
		baseIdx+0, baseIdx+1, baseIdx+2, // First triangle
		baseIdx+1, baseIdx+3, baseIdx+2, // Second triangle
	)
}

// Draw renders all batched sprites in a single DrawTriangles call
func (sb *SpriteBatch) Draw(screen *ebiten.Image) {
	if len(sb.vertices) == 0 {
		return // Nothing to draw
	}

	screen.DrawTriangles(sb.vertices, sb.indices, sb.image, nil)
}

// Reset clears the batch for reuse (avoids allocations)
func (sb *SpriteBatch) Reset() {
	sb.vertices = sb.vertices[:0]
	sb.indices = sb.indices[:0]
}

// SpriteCount returns the number of sprites in this batch
func (sb *SpriteBatch) SpriteCount() int {
	return len(sb.vertices) / 4
}
