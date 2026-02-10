package rendering

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// QuadBatch batches multiple quads (sprites or tiles) that share the same source image
// into a single draw call. This dramatically reduces draw calls per frame.
type QuadBatch struct {
	vertices []ebiten.Vertex
	indices  []uint16
	image    *ebiten.Image
}

// NewQuadBatch creates a batch for quads sharing the same image.
// verticesCap and indicesCap control pre-allocation sizes.
func NewQuadBatch(image *ebiten.Image, verticesCap, indicesCap int) *QuadBatch {
	return &QuadBatch{
		vertices: make([]ebiten.Vertex, 0, verticesCap),
		indices:  make([]uint16, 0, indicesCap),
		image:    image,
	}
}

// Add adds a quad to this batch with position, source rect, destination size, and color.
// dstX, dstY: Screen position where quad should be drawn
// srcX, srcY, srcW, srcH: Source rectangle in the texture
// dstW, dstH: Destination size (scaled dimensions)
// r, g, b, a: Color modulation (typically 1.0, 1.0, 1.0, 1.0 for normal rendering)
func (qb *QuadBatch) Add(dstX, dstY float32, srcX, srcY, srcW, srcH float32, dstW, dstH float32, r, g, b, a float32) {
	baseIdx := uint16(len(qb.vertices))

	qb.vertices = append(qb.vertices,
		ebiten.Vertex{
			DstX: dstX, DstY: dstY,
			SrcX: srcX, SrcY: srcY,
			ColorR: r, ColorG: g, ColorB: b, ColorA: a,
		},
		ebiten.Vertex{
			DstX: dstX + dstW, DstY: dstY,
			SrcX: srcX + srcW, SrcY: srcY,
			ColorR: r, ColorG: g, ColorB: b, ColorA: a,
		},
		ebiten.Vertex{
			DstX: dstX, DstY: dstY + dstH,
			SrcX: srcX, SrcY: srcY + srcH,
			ColorR: r, ColorG: g, ColorB: b, ColorA: a,
		},
		ebiten.Vertex{
			DstX: dstX + dstW, DstY: dstY + dstH,
			SrcX: srcX + srcW, SrcY: srcY + srcH,
			ColorR: r, ColorG: g, ColorB: b, ColorA: a,
		},
	)

	qb.indices = append(qb.indices,
		baseIdx+0, baseIdx+1, baseIdx+2,
		baseIdx+1, baseIdx+3, baseIdx+2,
	)
}

// Draw renders all batched quads in a single DrawTriangles call.
func (qb *QuadBatch) Draw(screen *ebiten.Image) {
	if len(qb.vertices) == 0 {
		return
	}
	screen.DrawTriangles(qb.vertices, qb.indices, qb.image, nil)
}

// Reset clears the batch for reuse (avoids allocations).
func (qb *QuadBatch) Reset() {
	qb.vertices = qb.vertices[:0]
	qb.indices = qb.indices[:0]
}

// Count returns the number of quads in this batch.
func (qb *QuadBatch) Count() int {
	return len(qb.vertices) / 4
}
