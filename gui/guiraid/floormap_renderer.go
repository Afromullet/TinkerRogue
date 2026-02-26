package guiraid

import (
	"fmt"
	"image/color"
	"math"
	"sort"

	"game_main/gui/widgetresources"
	"game_main/mind/raid"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// RoomCardState represents the interactive state of a room card.
type RoomCardState int

const (
	CardLocked     RoomCardState = iota
	CardAccessible
	CardCleared
)

// CardLayout holds the computed position and metadata for one room card.
type CardLayout struct {
	X, Y, W, H int
	Room        *raid.RoomData
	State       RoomCardState
	Depth       int
}

// roomTypeVisual defines the display properties for a room type.
type roomTypeVisual struct {
	DisplayName string
	Icon        string
	HeaderColor color.NRGBA
}

var roomVisuals = map[string]roomTypeVisual{
	"guard_post":   {DisplayName: "Guard Post", Icon: "\u2694", HeaderColor: color.NRGBA{200, 80, 80, 255}},
	"barracks":     {DisplayName: "Barracks", Icon: "\u26E8", HeaderColor: color.NRGBA{180, 120, 60, 255}},
	"armory":       {DisplayName: "Armory", Icon: "\u2692", HeaderColor: color.NRGBA{160, 160, 180, 255}},
	"command_post": {DisplayName: "Command", Icon: "\u265A", HeaderColor: color.NRGBA{200, 170, 50, 255}},
	"patrol_route": {DisplayName: "Patrol", Icon: "\u21C4", HeaderColor: color.NRGBA{100, 160, 100, 255}},
	"mage_tower":   {DisplayName: "Mage Tower", Icon: "\u2605", HeaderColor: color.NRGBA{120, 80, 200, 255}},
	"rest_room":    {DisplayName: "Rest Room", Icon: "\u2665", HeaderColor: color.NRGBA{80, 180, 80, 255}},
	"stairs":       {DisplayName: "Stairs", Icon: "\u2191", HeaderColor: color.NRGBA{180, 180, 220, 255}},
}

// FloorMapRenderer handles custom drawing of the room card grid.
type FloorMapRenderer struct {
	cards      []CardLayout
	rooms      []*raid.RoomData
	roomByNode map[int]*raid.RoomData

	// Drawing area (set by ComputeLayout)
	areaX, areaY, areaW, areaH int

	// Interaction state
	hoveredNodeID  int
	selectedNodeID int

	// Animation
	pulsePhase float64 // 0..2*pi, drives border pulse on accessible cards
}

// NewFloorMapRenderer creates a renderer with default state.
func NewFloorMapRenderer() *FloorMapRenderer {
	return &FloorMapRenderer{
		hoveredNodeID:  -1,
		selectedNodeID: -1,
	}
}

// Update advances the pulse animation.
func (r *FloorMapRenderer) Update(deltaTime float64) {
	r.pulsePhase += deltaTime * 3.0 // ~3 rad/s → ~0.5 Hz cycle
	if r.pulsePhase > 2*math.Pi {
		r.pulsePhase -= 2 * math.Pi
	}
}

// ComputeLayout arranges rooms into depth columns and computes card positions.
func (r *FloorMapRenderer) ComputeLayout(rooms []*raid.RoomData, areaX, areaY, areaW, areaH int) {
	r.rooms = rooms
	r.areaX = areaX
	r.areaY = areaY
	r.areaW = areaW
	r.areaH = areaH
	r.cards = nil

	if len(rooms) == 0 {
		return
	}

	// Build lookup
	r.roomByNode = make(map[int]*raid.RoomData, len(rooms))
	for _, room := range rooms {
		r.roomByNode[room.NodeID] = room
	}

	// Compute depth per room using Kahn's algorithm (BFS from roots)
	depth := r.computeDepths(rooms)

	// Group rooms by depth column
	columns := make(map[int][]*raid.RoomData)
	maxDepth := 0
	for _, room := range rooms {
		d := depth[room.NodeID]
		columns[d] = append(columns[d], room)
		if d > maxDepth {
			maxDepth = d
		}
	}

	// Sort each column by NodeID for stable order
	for d := range columns {
		sort.Slice(columns[d], func(i, j int) bool {
			return columns[d][i].NodeID < columns[d][j].NodeID
		})
	}

	numCols := maxDepth + 1
	if numCols == 0 {
		numCols = 1
	}

	// Compute card size — fit within area with padding
	colSpacing := 20
	rowSpacing := 16
	totalColSpacing := colSpacing * (numCols + 1)
	availW := areaW - totalColSpacing
	cardW := availW / numCols
	if cardW > 160 {
		cardW = 160
	}
	if cardW < 80 {
		cardW = 80
	}

	// Find max cards in any column to size height
	maxPerCol := 1
	for d := 0; d <= maxDepth; d++ {
		if len(columns[d]) > maxPerCol {
			maxPerCol = len(columns[d])
		}
	}
	totalRowSpacing := rowSpacing * (maxPerCol + 1)
	availH := areaH - totalRowSpacing
	cardH := availH / maxPerCol
	if cardH > 200 {
		cardH = 200
	}
	if cardH < 100 {
		cardH = 100
	}

	// Layout cards centered in each column
	totalGridW := numCols*cardW + (numCols-1)*colSpacing
	gridOffsetX := areaX + (areaW-totalGridW)/2

	for d := 0; d <= maxDepth; d++ {
		col := columns[d]
		colX := gridOffsetX + d*(cardW+colSpacing)

		totalColH := len(col)*cardH + (len(col)-1)*rowSpacing
		colOffsetY := areaY + (areaH-totalColH)/2

		for i, room := range col {
			cardY := colOffsetY + i*(cardH+rowSpacing)

			state := CardLocked
			if room.IsCleared {
				state = CardCleared
			} else if room.IsAccessible {
				state = CardAccessible
			}

			r.cards = append(r.cards, CardLayout{
				X:     colX,
				Y:     cardY,
				W:     cardW,
				H:     cardH,
				Room:  room,
				State: state,
				Depth: d,
			})
		}
	}
}

// Render draws edges, cards, and the hover tooltip.
func (r *FloorMapRenderer) Render(screen *ebiten.Image) {
	if len(r.cards) == 0 {
		return
	}
	r.renderEdges(screen)
	for i := range r.cards {
		r.renderCard(screen, &r.cards[i])
	}
	r.renderHoverDetail(screen)
}

// renderEdges draws connection lines between parent and child cards.
func (r *FloorMapRenderer) renderEdges(screen *ebiten.Image) {
	// Build card lookup by nodeID
	cardByNode := make(map[int]*CardLayout, len(r.cards))
	for i := range r.cards {
		cardByNode[r.cards[i].Room.NodeID] = &r.cards[i]
	}

	for i := range r.cards {
		card := &r.cards[i]
		for _, childID := range card.Room.ChildNodeIDs {
			childCard, ok := cardByNode[childID]
			if !ok {
				continue
			}

			// Line from right edge of parent to left edge of child
			x1 := float32(card.X + card.W)
			y1 := float32(card.Y + card.H/2)
			x2 := float32(childCard.X)
			y2 := float32(childCard.Y + childCard.H/2)

			// Critical path: thicker golden line
			if card.Room.OnCriticalPath && childCard.Room.OnCriticalPath {
				vector.StrokeLine(screen, x1, y1, x2, y2, 2.5,
					color.NRGBA{200, 180, 100, 255}, true)
			} else {
				vector.StrokeLine(screen, x1, y1, x2, y2, 1.5,
					color.NRGBA{80, 80, 100, 200}, true)
			}
		}
	}
}

// renderCard draws a single room card.
func (r *FloorMapRenderer) renderCard(screen *ebiten.Image, card *CardLayout) {
	vis := roomVisuals[card.Room.RoomType]
	if vis.DisplayName == "" {
		vis = roomTypeVisual{
			DisplayName: card.Room.RoomType,
			Icon:        "?",
			HeaderColor: color.NRGBA{120, 120, 120, 255},
		}
	}

	x := float32(card.X)
	y := float32(card.Y)
	w := float32(card.W)
	h := float32(card.H)
	headerH := float32(28)

	// State-dependent colors
	var bodyColor color.NRGBA
	var borderColor color.NRGBA
	var borderWidth float32 = 1.5
	headerColor := vis.HeaderColor
	textColor := color.NRGBA{220, 220, 230, 255}

	switch card.State {
	case CardLocked:
		// Desaturated grayscale
		avg := (uint16(headerColor.R) + uint16(headerColor.G) + uint16(headerColor.B)) / 3
		gray := uint8(avg)
		headerColor = color.NRGBA{gray, gray, gray, 180}
		bodyColor = color.NRGBA{30, 30, 35, 200}
		borderColor = color.NRGBA{50, 50, 55, 200}
		textColor = color.NRGBA{100, 100, 110, 200}

	case CardAccessible:
		bodyColor = color.NRGBA{25, 30, 40, 230}
		// Pulsing border
		pulse := float32(0.5 + 0.5*math.Sin(r.pulsePhase))
		bright := uint8(150 + pulse*105)
		borderColor = color.NRGBA{bright, bright, uint8(float32(bright) * 0.8), 255}
		borderWidth = 2.0

	case CardCleared:
		// Half-brightness
		headerColor.R = headerColor.R / 2
		headerColor.G = headerColor.G / 2
		headerColor.B = headerColor.B / 2
		bodyColor = color.NRGBA{20, 25, 20, 200}
		borderColor = color.NRGBA{60, 160, 60, 255}
		textColor = color.NRGBA{140, 160, 140, 200}
	}

	// Hovered card override
	isHovered := card.Room.NodeID == r.hoveredNodeID
	if isHovered && card.State != CardLocked {
		borderColor = color.NRGBA{255, 255, 255, 255}
		borderWidth = 3.0
	}

	// Selected card override
	isSelected := card.Room.NodeID == r.selectedNodeID
	if isSelected {
		borderColor = color.NRGBA{220, 190, 50, 255}
		borderWidth = 3.0
	}

	// Draw body background
	vector.DrawFilledRect(screen, x, y, w, h, bodyColor, true)

	// Draw header band
	vector.DrawFilledRect(screen, x, y, w, headerH, headerColor, true)

	// Draw border
	vector.StrokeRect(screen, x, y, w, h, borderWidth, borderColor, true)

	face := widgetresources.SmallFace

	// Draw room type name in header
	text.Draw(screen, vis.DisplayName, face, card.X+6, card.Y+20, textColor)

	// Draw icon centered in body
	iconY := card.Y + int(headerH) + (card.H-int(headerH))/2
	bounds := text.BoundString(widgetresources.LargeFace, vis.Icon)
	iconX := card.X + (card.W-bounds.Dx())/2
	text.Draw(screen, vis.Icon, widgetresources.LargeFace, iconX, iconY, textColor)

	// Draw status text at bottom
	statusText := r.getStatusText(card)
	text.Draw(screen, statusText, face, card.X+6, card.Y+card.H-8, textColor)

	// Critical path marker (gold asterisk in top-right)
	if card.Room.OnCriticalPath {
		text.Draw(screen, "*", widgetresources.LargeFace, card.X+card.W-22, card.Y+22,
			color.NRGBA{220, 190, 50, 255})
	}

	// Cleared checkmark overlay
	if card.State == CardCleared {
		checkBounds := text.BoundString(widgetresources.LargeFace, "\u2713")
		checkX := card.X + (card.W-checkBounds.Dx())/2
		checkY := card.Y + int(headerH) + (card.H-int(headerH))/2
		text.Draw(screen, "\u2713", widgetresources.LargeFace, checkX, checkY,
			color.NRGBA{80, 220, 80, 200})
	}
}

func (r *FloorMapRenderer) getStatusText(card *CardLayout) string {
	switch card.State {
	case CardLocked:
		return "Locked"
	case CardCleared:
		return "Cleared"
	case CardAccessible:
		n := len(card.Room.GarrisonSquadIDs)
		if n > 0 {
			return fmt.Sprintf("%d squads", n)
		}
		return "Accessible"
	}
	return ""
}

// renderHoverDetail draws a tooltip panel next to the hovered card.
func (r *FloorMapRenderer) renderHoverDetail(screen *ebiten.Image) {
	if r.hoveredNodeID < 0 {
		return
	}

	var hovered *CardLayout
	for i := range r.cards {
		if r.cards[i].Room.NodeID == r.hoveredNodeID {
			hovered = &r.cards[i]
			break
		}
	}
	if hovered == nil {
		return
	}

	vis := roomVisuals[hovered.Room.RoomType]
	if vis.DisplayName == "" {
		vis.DisplayName = hovered.Room.RoomType
	}

	// Build tooltip lines
	lines := []string{
		fmt.Sprintf("Room %d: %s", hovered.Room.NodeID, vis.DisplayName),
		fmt.Sprintf("Status: %s", r.getStatusText(hovered)),
	}
	if len(hovered.Room.GarrisonSquadIDs) > 0 && !hovered.Room.IsCleared {
		lines = append(lines, fmt.Sprintf("Garrison: %d squads", len(hovered.Room.GarrisonSquadIDs)))
	}
	if hovered.Room.OnCriticalPath {
		lines = append(lines, "Critical Path")
	}

	face := widgetresources.SmallFace
	lineH := face.Metrics().Height.Round() + 4

	// Tooltip size
	tooltipW := 200
	tooltipH := len(lines)*lineH + 16

	// Position: right of card, or left if near right edge
	tooltipX := hovered.X + hovered.W + 8
	tooltipY := hovered.Y

	screenW := screen.Bounds().Dx()
	if tooltipX+tooltipW > screenW {
		tooltipX = hovered.X - tooltipW - 8
	}

	// Draw tooltip background
	vector.DrawFilledRect(screen, float32(tooltipX), float32(tooltipY),
		float32(tooltipW), float32(tooltipH),
		color.NRGBA{15, 18, 25, 230}, true)
	vector.StrokeRect(screen, float32(tooltipX), float32(tooltipY),
		float32(tooltipW), float32(tooltipH),
		1.5, color.NRGBA{140, 140, 160, 200}, true)

	// Draw text lines
	for i, line := range lines {
		textY := tooltipY + 8 + (i+1)*lineH - 4
		text.Draw(screen, line, face, tooltipX+8, textY, color.NRGBA{220, 220, 230, 255})
	}
}

// HitTest returns the RoomData at the given screen coordinates, or nil.
func (r *FloorMapRenderer) HitTest(mx, my int) *raid.RoomData {
	for i := range r.cards {
		card := &r.cards[i]
		if mx >= card.X && mx < card.X+card.W &&
			my >= card.Y && my < card.Y+card.H {
			return card.Room
		}
	}
	return nil
}

// UpdateHover updates the hovered room ID. Returns the node ID or -1.
func (r *FloorMapRenderer) UpdateHover(mx, my int) int {
	room := r.HitTest(mx, my)
	if room != nil {
		r.hoveredNodeID = room.NodeID
		return room.NodeID
	}
	r.hoveredNodeID = -1
	return -1
}

// SetSelected sets the selected room ID for highlight rendering.
func (r *FloorMapRenderer) SetSelected(nodeID int) {
	r.selectedNodeID = nodeID
}

// computeDepths uses Kahn's algorithm to assign depth levels from entry rooms.
func (r *FloorMapRenderer) computeDepths(rooms []*raid.RoomData) map[int]int {
	depth := make(map[int]int, len(rooms))

	// Build in-degree map
	inDegree := make(map[int]int, len(rooms))
	nodeIDs := make([]int, 0, len(rooms))
	for _, room := range rooms {
		nodeIDs = append(nodeIDs, room.NodeID)
		if _, ok := inDegree[room.NodeID]; !ok {
			inDegree[room.NodeID] = 0
		}
		for _, childID := range room.ChildNodeIDs {
			inDegree[childID]++
		}
	}

	// Start BFS from nodes with no parents (in-degree 0)
	queue := make([]int, 0)
	for _, id := range nodeIDs {
		if inDegree[id] == 0 {
			queue = append(queue, id)
			depth[id] = 0
		}
	}

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		room := r.roomByNode[nodeID]
		if room == nil {
			continue
		}

		for _, childID := range room.ChildNodeIDs {
			childDepth := depth[nodeID] + 1
			if existing, ok := depth[childID]; !ok || childDepth > existing {
				depth[childID] = childDepth
			}
			inDegree[childID]--
			if inDegree[childID] == 0 {
				queue = append(queue, childID)
			}
		}
	}

	// Assign depth 0 to any orphaned nodes not reached
	for _, room := range rooms {
		if _, ok := depth[room.NodeID]; !ok {
			depth[room.NodeID] = 0
		}
	}

	return depth
}
