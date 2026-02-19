package worldmap

import "game_main/common"

// injectGarrisonTerrain dispatches to per-room-type terrain injection.
// Called after rooms are carved to add tactical features (pillars, barricades, alcoves).
func injectGarrisonTerrain(roomType string, room Rect, width int, result *GenerationResult, images TileImageSet) {
	switch roomType {
	case GarrisonRoomGuardPost:
		injectGuardPostTerrain(room, width, result, images)
	case GarrisonRoomBarracks:
		injectBarracksTerrain(room, width, result, images)
	case GarrisonRoomArmory:
		injectArmoryTerrain(room, width, result, images)
	case GarrisonRoomCommandPost:
		injectCommandPostTerrain(room, width, result, images)
	case GarrisonRoomPatrolRoute:
		injectPatrolRouteTerrain(room, width, result, images)
	case GarrisonRoomMageTower:
		injectMageTowerTerrain(room, width, result, images)
		// Rest Room and Stairs: no terrain injection
	}
}

// ========================================
// TERRAIN PRIMITIVES
// ========================================

// setTileWall sets a floor tile back to wall for terrain features.
// Bounds-checked; no-op if out of range.
func setTileWall(result *GenerationResult, x, y, width int, images TileImageSet) {
	numTiles := len(result.Tiles)
	height := numTiles / width
	if x < 0 || y < 0 || x >= width || y >= height {
		return
	}
	idx := positionToIndex(x, y, width)
	if idx >= 0 && idx < numTiles {
		result.Tiles[idx].Blocked = true
		result.Tiles[idx].TileType = WALL
		result.Tiles[idx].Image = selectRandomImage(images.WallImages)
	}
}

// place2x2Pillar places a 2x2 wall block at (px, py).
func place2x2Pillar(result *GenerationResult, px, py, width int, images TileImageSet) {
	for dx := 0; dx < 2; dx++ {
		for dy := 0; dy < 2; dy++ {
			setTileWall(result, px+dx, py+dy, width, images)
		}
	}
}

// placeHorizontalWall places a horizontal wall segment of given length starting at (x, y).
func placeHorizontalWall(result *GenerationResult, x, y, length, width int, images TileImageSet) {
	for dx := 0; dx < length; dx++ {
		setTileWall(result, x+dx, y, width, images)
	}
}

// placeVerticalWall places a vertical wall segment of given length starting at (x, y).
func placeVerticalWall(result *GenerationResult, x, y, length, width int, images TileImageSet) {
	for dy := 0; dy < length; dy++ {
		setTileWall(result, x, y+dy, width, images)
	}
}

// placeUAlcove places a U-shaped wall alcove opening in the specified direction.
// direction: 0=north(open top), 1=south(open bottom), 2=east(open right), 3=west(open left)
func placeUAlcove(result *GenerationResult, x, y, alcoveW, alcoveH, openDir, width int, images TileImageSet) {
	switch openDir {
	case 0: // Open north: bottom wall + left wall + right wall
		placeHorizontalWall(result, x, y+alcoveH-1, alcoveW, width, images)
		placeVerticalWall(result, x, y, alcoveH, width, images)
		placeVerticalWall(result, x+alcoveW-1, y, alcoveH, width, images)
	case 1: // Open south: top wall + left wall + right wall
		placeHorizontalWall(result, x, y, alcoveW, width, images)
		placeVerticalWall(result, x, y, alcoveH, width, images)
		placeVerticalWall(result, x+alcoveW-1, y, alcoveH, width, images)
	case 2: // Open east: top wall + bottom wall + left wall
		placeHorizontalWall(result, x, y, alcoveW, width, images)
		placeHorizontalWall(result, x, y+alcoveH-1, alcoveW, width, images)
		placeVerticalWall(result, x, y, alcoveH, width, images)
	case 3: // Open west: top wall + bottom wall + right wall
		placeHorizontalWall(result, x, y, alcoveW, width, images)
		placeHorizontalWall(result, x, y+alcoveH-1, alcoveW, width, images)
		placeVerticalWall(result, x+alcoveW-1, y, alcoveH, width, images)
	}
}

// placeLCorner places an L-shaped wall corner.
// orientation: 0=NE, 1=NW, 2=SE, 3=SW
func placeLCorner(result *GenerationResult, x, y, armLen, orientation, width int, images TileImageSet) {
	switch orientation {
	case 0: // NE: horizontal going right + vertical going up
		placeHorizontalWall(result, x, y, armLen, width, images)
		placeVerticalWall(result, x+armLen-1, y-armLen+1, armLen, width, images)
	case 1: // NW: horizontal going left + vertical going up
		placeHorizontalWall(result, x-armLen+1, y, armLen, width, images)
		placeVerticalWall(result, x-armLen+1, y-armLen+1, armLen, width, images)
	case 2: // SE: horizontal going right + vertical going down
		placeHorizontalWall(result, x, y, armLen, width, images)
		placeVerticalWall(result, x+armLen-1, y, armLen, width, images)
	case 3: // SW: horizontal going left + vertical going down
		placeHorizontalWall(result, x-armLen+1, y, armLen, width, images)
		placeVerticalWall(result, x-armLen+1, y, armLen, width, images)
	}
}

// placeBarricadeLine places a horizontal or vertical barricade with gaps.
// isHorizontal: true=horizontal line, false=vertical line.
// gapInterval: place a 2-tile gap every N tiles.
func placeBarricadeLine(result *GenerationResult, x, y int, isHorizontal bool, totalLen, gapInterval, width int, images TileImageSet) {
	for i := 0; i < totalLen; i++ {
		// Skip gap positions
		if gapInterval > 0 && i > 0 && i%gapInterval >= gapInterval-2 {
			continue
		}
		if isHorizontal {
			setTileWall(result, x+i, y, width, images)
		} else {
			setTileWall(result, x, y+i, width, images)
		}
	}
}

// placePillarRow places a row of 1x1 pillars with specified spacing.
func placePillarRow(result *GenerationResult, startX, startY int, isHorizontal bool, count, spacing, width int, images TileImageSet) {
	for i := 0; i < count; i++ {
		if isHorizontal {
			setTileWall(result, startX+i*spacing, startY, width, images)
		} else {
			setTileWall(result, startX, startY+i*spacing, width, images)
		}
	}
}

// placeSpacedPillars places 2x2 pillars with minimum spacing, returns positions placed.
func placeSpacedPillars(result *GenerationResult, room Rect, count, minSpacing, width int, images TileImageSet, use1x1 bool) [][2]int {
	positions := make([][2]int, 0, count)
	for attempt := 0; attempt < count*20 && len(positions) < count; attempt++ {
		px := common.GetRandomBetween(room.X1+3, room.X2-4)
		py := common.GetRandomBetween(room.Y1+3, room.Y2-4)

		if isTooCloseToAny(px, py, positions, minSpacing) {
			continue
		}

		positions = append(positions, [2]int{px, py})
		if use1x1 {
			setTileWall(result, px, py, width, images)
		} else {
			place2x2Pillar(result, px, py, width, images)
		}
	}
	return positions
}

// ========================================
// GUARD POST: Chokepoint defense
// ========================================

func injectGuardPostTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 2)
	switch variant {
	case 0:
		injectGuardPostDoublePillar(room, width, result, images)
	case 1:
		injectGuardPostStaggeredWalls(room, width, result, images)
	case 2:
		injectGuardPostKillBox(room, width, result, images)
	}
}

// Layout A: Two 2x2 pillars flanking center chokepoint + rear wall segment.
func injectGuardPostDoublePillar(room Rect, width int, result *GenerationResult, images TileImageSet) {
	cx := (room.X1 + room.X2) / 2
	cy := (room.Y1 + room.Y2) / 2

	gapHalf := 1
	leftPillarX := max(room.X1+2, cx-gapHalf-2)
	rightPillarX := min(room.X2-4, cx+gapHalf)

	place2x2Pillar(result, leftPillarX, cy-1, width, images)
	place2x2Pillar(result, rightPillarX, cy-1, width, images)

	// Rear wall segment offset to one side for secondary defender cover
	rearX := cx + common.GetRandomBetween(1, 3)
	rearY := cy + 2
	rearX = max(room.X1+2, min(rearX, room.X2-5))
	rearY = min(rearY, room.Y2-3)
	placeHorizontalWall(result, rearX, rearY, 3, width, images)
}

// Layout B: Two staggered horizontal walls with gaps at opposite ends.
func injectGuardPostStaggeredWalls(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	// First wall: upper third, gap on the right
	wall1X := room.X1 + 2
	wall1Y := room.Y1 + roomH/3
	wall1Len := roomW - 6 // Leave gap of 4 on right
	if wall1Len > 2 {
		placeHorizontalWall(result, wall1X, wall1Y, wall1Len, width, images)
	}

	// Second wall: lower third, gap on the left
	wall2X := room.X1 + 4 // Gap of 4 on left
	wall2Y := room.Y1 + roomH*2/3
	wall2Len := roomW - 6
	if wall2Len > 2 {
		placeHorizontalWall(result, wall2X, wall2Y, wall2Len, width, images)
	}
}

// Layout C: U-shaped kill box opening toward entry, with attacker cover pillar.
func injectGuardPostKillBox(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	// U-alcove centered in room, opening west (toward entry)
	alcoveW := max(4, roomW/3)
	alcoveH := max(4, roomH/2)
	alcoveX := room.X1 + roomW/3
	alcoveY := room.Y1 + (roomH-alcoveH)/2

	placeUAlcove(result, alcoveX, alcoveY, alcoveW, alcoveH, 3, width, images)

	// Cover pillar for attacker near entry
	pillarX := room.X1 + 2
	pillarY := room.Y2 - 3
	setTileWall(result, pillarX, pillarY, width, images)
}

// ========================================
// BARRACKS: Large combat room with scattered obstacles
// ========================================

func injectBarracksTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 2)
	switch variant {
	case 0:
		injectBarracksScatteredPillars(room, width, result, images)
	case 1:
		injectBarracksBunkRows(room, width, result, images)
	case 2:
		injectBarracksTrainingYard(room, width, result, images)
	}
}

// Layout A: Mixed 1x1 and 2x2 pillars + short lane-dividing wall segments.
func injectBarracksScatteredPillars(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomH := room.Y2 - room.Y1

	// 3-4 pillars (mix of sizes)
	for i := 0; i < 3; i++ {
		use1x1 := common.GetDiceRoll(2) == 1
		placeSpacedPillars(result, room, 1, 4, width, images, use1x1)
	}

	// Two short horizontal wall segments on opposite sides at midpoint
	midY := room.Y1 + roomH/2
	placeHorizontalWall(result, room.X1+2, midY, 2, width, images)
	placeHorizontalWall(result, room.X2-4, midY, 2, width, images)
}

// Layout B: Bunk rows - vertical wall segments creating lanes.
func injectBarracksBunkRows(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	// Two columns of vertical segments
	col1X := room.X1 + roomW/3
	col2X := room.X1 + roomW*2/3

	bunkLen := min(3, roomH/3)

	// Upper bunks
	if bunkLen >= 2 {
		placeVerticalWall(result, col1X, room.Y1+2, bunkLen, width, images)
		placeVerticalWall(result, col2X, room.Y1+2, bunkLen, width, images)
	}

	// Lower bunks
	lowerY := room.Y2 - 2 - bunkLen
	if lowerY > room.Y1+2+bunkLen+1 && bunkLen >= 2 {
		placeVerticalWall(result, col1X, lowerY, bunkLen, width, images)
		placeVerticalWall(result, col2X, lowerY, bunkLen, width, images)
	}
}

// Layout C: Training yard - open center with pillar ring at edges.
func injectBarracksTrainingYard(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	// Pillar positions forming a ring around the room
	pillarSpacingX := max(3, roomW/4)
	pillarSpacingY := max(3, roomH/3)

	// Top row
	for x := room.X1 + 3; x < room.X2-2; x += pillarSpacingX {
		setTileWall(result, x, room.Y1+2, width, images)
	}
	// Bottom row
	for x := room.X1 + 3; x < room.X2-2; x += pillarSpacingX {
		setTileWall(result, x, room.Y2-2, width, images)
	}
	// Left column (skip corners)
	for y := room.Y1 + 2 + pillarSpacingY; y < room.Y2-2; y += pillarSpacingY {
		setTileWall(result, room.X1+2, y, width, images)
	}
	// Right column (skip corners)
	for y := room.Y1 + 2 + pillarSpacingY; y < room.Y2-2; y += pillarSpacingY {
		setTileWall(result, room.X2-2, y, width, images)
	}
}

// ========================================
// ARMORY: Divided room with barricade line
// ========================================

func injectArmoryTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 1)
	switch variant {
	case 0:
		injectArmoryBarricadeLine(room, width, result, images)
	case 1:
		injectArmoryWeaponRacks(room, width, result, images)
	}
}

// Layout A: Barricade at ~40% depth with gaps + approach cover pillar.
func injectArmoryBarricadeLine(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	barricadeX := room.X1 + (roomW * 4 / 10)

	// Barricade line with gaps
	interiorTop := room.Y1 + 2
	interiorBot := room.Y2 - 2
	barricadeLen := interiorBot - interiorTop
	if barricadeLen > 0 {
		placeBarricadeLine(result, barricadeX, interiorTop, false, barricadeLen, max(4, barricadeLen/3), width, images)
	}

	// Approach cover pillar
	pillarX := max(room.X1+2, room.X1+roomW/5)
	pillarY := room.Y1 + roomH/2 - 1
	place2x2Pillar(result, pillarX, pillarY, width, images)

	// Defender-side L-corner cover near a gap
	defX := min(barricadeX+2, room.X2-4)
	defY := room.Y1 + roomH/3
	placeLCorner(result, defX, defY, 2, 2, width, images)
}

// Layout B: Weapon racks - parallel horizontal walls creating center aisle.
func injectArmoryWeaponRacks(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	rackLen := max(3, roomW-4)

	// Upper rack
	upperY := room.Y1 + roomH/4
	placeHorizontalWall(result, room.X1+2, upperY, rackLen, width, images)

	// Lower rack
	lowerY := room.Y2 - roomH/4
	placeHorizontalWall(result, room.X1+2, lowerY, rackLen, width, images)

	// Perpendicular wall at far end
	endX := room.X2 - 3
	endY := upperY + 1
	endLen := max(2, lowerY-upperY-1)
	if endLen > 1 {
		placeVerticalWall(result, endX, endY, endLen, width, images)
	}
}

// ========================================
// COMMAND POST: Fortified inner chamber
// ========================================

func injectCommandPostTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 1)
	switch variant {
	case 0:
		injectCommandPostUAlcove(room, width, result, images)
	case 1:
		injectCommandPostWarTable(room, width, result, images)
	}
}

// Layout A: U-shaped rear alcove with flanking pillars + staging cover.
func injectCommandPostUAlcove(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	// Alcove in rear 35%
	alcoveW := max(4, roomW/3)
	alcoveH := max(4, roomH/2)
	alcoveX := room.X1 + roomW*65/100
	alcoveY := room.Y1 + (roomH-alcoveH)/2

	// Clamp alcove to room bounds
	if alcoveX+alcoveW > room.X2-1 {
		alcoveX = room.X2 - alcoveW - 1
	}
	if alcoveY+alcoveH > room.Y2-1 {
		alcoveY = room.Y2 - alcoveH - 1
	}

	placeUAlcove(result, alcoveX, alcoveY, alcoveW, alcoveH, 3, width, images)

	// Flanking pillars at alcove entrance
	setTileWall(result, alcoveX-1, alcoveY+1, width, images)
	setTileWall(result, alcoveX-1, alcoveY+alcoveH-2, width, images)

	// Staging cover for attacker
	stagingX := room.X1 + roomW/4
	stagingY := room.Y2 - 3
	placeHorizontalWall(result, stagingX, stagingY, 2, width, images)
}

// Layout B: War table (3x3 central block) + quadrant pillars.
func injectCommandPostWarTable(room Rect, width int, result *GenerationResult, images TileImageSet) {
	cx := (room.X1 + room.X2) / 2
	cy := (room.Y1 + room.Y2) / 2

	// 3x3 central block
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			setTileWall(result, cx+dx, cy+dy, width, images)
		}
	}

	// Four quadrant pillars
	qx := (room.X2 - room.X1) / 4
	qy := (room.Y2 - room.Y1) / 4

	setTileWall(result, room.X1+qx, room.Y1+qy, width, images)
	setTileWall(result, room.X2-qx, room.Y1+qy, width, images)
	setTileWall(result, room.X1+qx, room.Y2-qy, width, images)
	setTileWall(result, room.X2-qx, room.Y2-qy, width, images)
}

// ========================================
// PATROL ROUTE: Wide open, minimal cover
// ========================================

func injectPatrolRouteTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 1)
	switch variant {
	case 0:
		injectPatrolRouteOpenSparse(room, width, result, images)
	case 1:
		injectPatrolRouteBorderPillars(room, width, result, images)
	}
}

// Layout A: 1-2 pillars near entry + short wall at far end.
func injectPatrolRouteOpenSparse(room Rect, width int, result *GenerationResult, images TileImageSet) {
	// 1-2 small pillars in entry third
	numPillars := common.GetRandomBetween(1, 2)
	entryThird := room.X1 + (room.X2-room.X1)/3

	for i := 0; i < numPillars; i++ {
		px := common.GetRandomBetween(room.X1+3, max(room.X1+4, entryThird-1))
		py := common.GetRandomBetween(room.Y1+3, room.Y2-3)
		setTileWall(result, px, py, width, images)
	}

	// Short wall segment at far end
	placeHorizontalWall(result, room.X2-4, room.Y2-3, 2, width, images)
}

// Layout B: Four 1x1 pillars along top and bottom edges only.
func injectPatrolRouteBorderPillars(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	spacing := max(3, roomW/4)

	// Top edge pillars
	for x := room.X1 + 3; x < room.X2-2; x += spacing {
		setTileWall(result, x, room.Y1+2, width, images)
	}
	// Bottom edge pillars
	for x := room.X1 + 3; x < room.X2-2; x += spacing {
		setTileWall(result, x, room.Y2-2, width, images)
	}
}

// ========================================
// MAGE TOWER: Vertical pillar gauntlet
// ========================================

func injectMageTowerTerrain(room Rect, width int, result *GenerationResult, images TileImageSet) {
	variant := common.GetRandomBetween(0, 1)
	switch variant {
	case 0:
		injectMageTowerStaggeredGrid(room, width, result, images)
	case 1:
		injectMageTowerArcaneCorridor(room, width, result, images)
	}
}

// Layout A: 4-6 staggered pillars (mix 1x1 and 2x2), no two share X or Y.
func injectMageTowerStaggeredGrid(room Rect, width int, result *GenerationResult, images TileImageSet) {
	numPillars := common.GetRandomBetween(4, 6)
	positions := make([][2]int, 0, numPillars)

	for attempt := 0; attempt < numPillars*15 && len(positions) < numPillars; attempt++ {
		px := common.GetRandomBetween(room.X1+3, room.X2-4)
		py := common.GetRandomBetween(room.Y1+3, room.Y2-4)

		// Enforce minimum 3-tile spacing AND staggering
		if isTooCloseToAny(px, py, positions, 3) {
			continue
		}

		positions = append(positions, [2]int{px, py})
		if common.GetDiceRoll(2) == 1 {
			setTileWall(result, px, py, width, images)
		} else {
			place2x2Pillar(result, px, py, width, images)
		}
	}
}

// Layout B: Winding approach with staggered vertical walls + rear strongpoint.
func injectMageTowerArcaneCorridor(room Rect, width int, result *GenerationResult, images TileImageSet) {
	roomW := room.X2 - room.X1
	roomH := room.Y2 - room.Y1

	wallLen := min(3, roomH/3)

	// First vertical wall: left side, upper area
	wall1X := room.X1 + roomW/3
	wall1Y := room.Y1 + 2
	if wallLen >= 2 {
		placeVerticalWall(result, wall1X, wall1Y, wallLen, width, images)
	}

	// Second vertical wall: right side, lower area
	wall2X := room.X1 + roomW*2/3
	wall2Y := room.Y2 - 2 - wallLen
	if wallLen >= 2 && wall2Y > wall1Y+wallLen+1 {
		placeVerticalWall(result, wall2X, wall2Y, wallLen, width, images)
	}

	// Rear 2x2 strongpoint
	rearX := room.X2 - 4
	rearY := room.Y1 + roomH/2 - 1
	place2x2Pillar(result, rearX, rearY, width, images)
}
