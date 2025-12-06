package gui

import (
	"game_main/gui/widgets"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestNewResponsiveRowPadding(t *testing.T) {
	layout := &widgets.LayoutConfig{
		ScreenWidth:  800,
		ScreenHeight: 600,
	}

	// Test with PaddingExtraSmall (0.0125)
	insets := NewResponsiveRowPadding(layout, widgets.PaddingExtraSmall)

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
	layout := &widgets.LayoutConfig{
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
	layout := &widgets.LayoutConfig{
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
	layout := &widgets.LayoutConfig{
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
