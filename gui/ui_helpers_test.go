package gui

import (
	"game_main/gui/specs"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestNewResponsiveRowPadding(t *testing.T) {
	layout := &specs.LayoutConfig{
		ScreenWidth:  800,
		ScreenHeight: 600,
	}

	// Test with PaddingExtraSmall (0.0125)
	insets := NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)

	expectedH := 10 // int(float64(800) * 0.0125)
	expectedV := 7  // int(float64(600) * 0.0125)

	if insets.Left != expectedH || insets.Right != expectedH {
		t.Errorf("Expected horizontal padding %d, got Left=%d Right=%d",
			expectedH, insets.Left, insets.Right)
	}

	if insets.Top != expectedV || insets.Bottom != expectedV {
		t.Errorf("Expected vertical padding %d, got Top=%d Bottom=%d",
			expectedV, insets.Top, insets.Bottom)
	}
}

func TestNewResponsiveHorizontalPadding(t *testing.T) {
	layout := &specs.LayoutConfig{
		ScreenWidth:  1000,
		ScreenHeight: 800,
	}

	insets := NewResponsiveHorizontalPadding(layout, 0.02)

	if insets.Left != 20 || insets.Right != 20 {
		t.Errorf("Expected L/R=20, got L=%d R=%d", insets.Left, insets.Right)
	}

	if insets.Top != 0 || insets.Bottom != 0 {
		t.Errorf("Expected T/B=0, got T=%d B=%d", insets.Top, insets.Bottom)
	}
}

func TestNewResponsiveVerticalPadding(t *testing.T) {
	layout := &specs.LayoutConfig{
		ScreenWidth:  1000,
		ScreenHeight: 800,
	}

	insets := NewResponsiveVerticalPadding(layout, 0.02)

	if insets.Top != 16 || insets.Bottom != 16 {
		t.Errorf("Expected T/B=16, got T=%d B=%d", insets.Top, insets.Bottom)
	}

	if insets.Left != 0 || insets.Right != 0 {
		t.Errorf("Expected L/R=0, got L=%d R=%d", insets.Left, insets.Right)
	}
}

func TestNewResponsivePaddingSingle(t *testing.T) {
	layout := &specs.LayoutConfig{
		ScreenWidth:  800,
		ScreenHeight: 600,
	}

	tests := []struct {
		name     string
		side     PaddingSide
		expected widget.Insets
	}{
		{
			name:     "Top only",
			side:     PaddingTop,
			expected: widget.Insets{Top: 6}, // int(600 * 0.01)
		},
		{
			name:     "Bottom only",
			side:     PaddingBottom,
			expected: widget.Insets{Bottom: 6}, // int(600 * 0.01)
		},
		{
			name:     "Left only",
			side:     PaddingLeft,
			expected: widget.Insets{Left: 8}, // int(800 * 0.01)
		},
		{
			name:     "Right only",
			side:     PaddingRight,
			expected: widget.Insets{Right: 8}, // int(800 * 0.01)
		},
		{
			name: "TopLeft",
			side: PaddingTopLeft,
			expected: widget.Insets{
				Top:  6, // int(600 * 0.01)
				Left: 8, // int(800 * 0.01)
			},
		},
		{
			name: "TopRight",
			side: PaddingTopRight,
			expected: widget.Insets{
				Top:   6, // int(600 * 0.01)
				Right: 8, // int(800 * 0.01)
			},
		},
		{
			name: "BottomLeft",
			side: PaddingBottomLeft,
			expected: widget.Insets{
				Bottom: 6, // int(600 * 0.01)
				Left:   8, // int(800 * 0.01)
			},
		},
		{
			name: "BottomRight",
			side: PaddingBottomRight,
			expected: widget.Insets{
				Bottom: 6, // int(600 * 0.01)
				Right:  8, // int(800 * 0.01)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insets := NewResponsivePaddingSingle(layout, 0.01, tt.side)
			if insets != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, insets)
			}
		})
	}
}

// ========================================
// ANCHOR LAYOUT HELPER TESTS
// ========================================

func TestAnchorStartStart(t *testing.T) {
	result := AnchorStartStart(10, 20)

	if result.HorizontalPosition != widget.AnchorLayoutPositionStart {
		t.Errorf("Expected HorizontalPosition Start, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionStart {
		t.Errorf("Expected VerticalPosition Start, got %v", result.VerticalPosition)
	}
	if result.Padding.Left != 10 {
		t.Errorf("Expected Left padding 10, got %d", result.Padding.Left)
	}
	if result.Padding.Top != 20 {
		t.Errorf("Expected Top padding 20, got %d", result.Padding.Top)
	}
	// Verify other sides are zero
	if result.Padding.Right != 0 || result.Padding.Bottom != 0 {
		t.Errorf("Expected Right/Bottom padding 0, got Right=%d Bottom=%d",
			result.Padding.Right, result.Padding.Bottom)
	}
}

func TestAnchorCenterStart(t *testing.T) {
	result := AnchorCenterStart(15)

	if result.HorizontalPosition != widget.AnchorLayoutPositionCenter {
		t.Errorf("Expected HorizontalPosition Center, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionStart {
		t.Errorf("Expected VerticalPosition Start, got %v", result.VerticalPosition)
	}
	if result.Padding.Top != 15 {
		t.Errorf("Expected Top padding 15, got %d", result.Padding.Top)
	}
	// Verify other sides are zero
	if result.Padding.Left != 0 || result.Padding.Right != 0 || result.Padding.Bottom != 0 {
		t.Errorf("Expected L/R/B padding 0, got L=%d R=%d B=%d",
			result.Padding.Left, result.Padding.Right, result.Padding.Bottom)
	}
}

func TestAnchorEndCenter(t *testing.T) {
	result := AnchorEndCenter(25)

	if result.HorizontalPosition != widget.AnchorLayoutPositionEnd {
		t.Errorf("Expected HorizontalPosition End, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionCenter {
		t.Errorf("Expected VerticalPosition Center, got %v", result.VerticalPosition)
	}
	if result.Padding.Right != 25 {
		t.Errorf("Expected Right padding 25, got %d", result.Padding.Right)
	}
	// Verify other sides are zero
	if result.Padding.Left != 0 || result.Padding.Top != 0 || result.Padding.Bottom != 0 {
		t.Errorf("Expected L/T/B padding 0, got L=%d T=%d B=%d",
			result.Padding.Left, result.Padding.Top, result.Padding.Bottom)
	}
}

func TestAnchorEndEnd(t *testing.T) {
	result := AnchorEndEnd(30, 40)

	if result.HorizontalPosition != widget.AnchorLayoutPositionEnd {
		t.Errorf("Expected HorizontalPosition End, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionEnd {
		t.Errorf("Expected VerticalPosition End, got %v", result.VerticalPosition)
	}
	if result.Padding.Right != 30 {
		t.Errorf("Expected Right padding 30, got %d", result.Padding.Right)
	}
	if result.Padding.Bottom != 40 {
		t.Errorf("Expected Bottom padding 40, got %d", result.Padding.Bottom)
	}
	// Verify other sides are zero
	if result.Padding.Left != 0 || result.Padding.Top != 0 {
		t.Errorf("Expected L/T padding 0, got L=%d T=%d",
			result.Padding.Left, result.Padding.Top)
	}
}

func TestAnchorEndStart(t *testing.T) {
	result := AnchorEndStart(35, 45)

	if result.HorizontalPosition != widget.AnchorLayoutPositionEnd {
		t.Errorf("Expected HorizontalPosition End, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionStart {
		t.Errorf("Expected VerticalPosition Start, got %v", result.VerticalPosition)
	}
	if result.Padding.Right != 35 {
		t.Errorf("Expected Right padding 35, got %d", result.Padding.Right)
	}
	if result.Padding.Top != 45 {
		t.Errorf("Expected Top padding 45, got %d", result.Padding.Top)
	}
	// Verify other sides are zero
	if result.Padding.Left != 0 || result.Padding.Bottom != 0 {
		t.Errorf("Expected L/B padding 0, got L=%d B=%d",
			result.Padding.Left, result.Padding.Bottom)
	}
}

func TestAnchorCenterEnd(t *testing.T) {
	result := AnchorCenterEnd(50)

	if result.HorizontalPosition != widget.AnchorLayoutPositionCenter {
		t.Errorf("Expected HorizontalPosition Center, got %v", result.HorizontalPosition)
	}
	if result.VerticalPosition != widget.AnchorLayoutPositionEnd {
		t.Errorf("Expected VerticalPosition End, got %v", result.VerticalPosition)
	}
	if result.Padding.Bottom != 50 {
		t.Errorf("Expected Bottom padding 50, got %d", result.Padding.Bottom)
	}
	// Verify other sides are zero
	if result.Padding.Left != 0 || result.Padding.Right != 0 || result.Padding.Top != 0 {
		t.Errorf("Expected L/R/T padding 0, got L=%d R=%d T=%d",
			result.Padding.Left, result.Padding.Right, result.Padding.Top)
	}
}
