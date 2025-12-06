package gui

import (
	"game_main/gui/widgets"

	"github.com/ebitenui/ebitenui/widget"
)

// NewResponsiveRowPadding creates responsive padding insets for RowLayout.
// Replaces the verbose 6-line padding calculation that appears throughout the codebase.
//
// Example usage:
//
//	padding := gui.NewResponsiveRowPadding(layout, widgets.PaddingExtraSmall)
//
// Replaces:
//
//	widget.Insets{
//	    Left:   int(float64(layout.ScreenWidth) * widgets.PaddingExtraSmall),
//	    Right:  int(float64(layout.ScreenWidth) * widgets.PaddingExtraSmall),
//	    Top:    int(float64(layout.ScreenHeight) * widgets.PaddingExtraSmall),
//	    Bottom: int(float64(layout.ScreenHeight) * widgets.PaddingExtraSmall),
//	}
func NewResponsiveRowPadding(layout *widgets.LayoutConfig, paddingConstant float64) widget.Insets {
	hPadding := int(float64(layout.ScreenWidth) * paddingConstant)
	vPadding := int(float64(layout.ScreenHeight) * paddingConstant)

	return widget.Insets{
		Left:   hPadding,
		Right:  hPadding,
		Top:    vPadding,
		Bottom: vPadding,
	}
}

// NewResponsiveHorizontalPadding creates horizontal-only responsive padding.
// Useful for button containers that only need left/right padding.
//
// Example usage:
//
//	padding := gui.NewResponsiveHorizontalPadding(layout, widgets.PaddingExtraSmall)
func NewResponsiveHorizontalPadding(layout *widgets.LayoutConfig, paddingConstant float64) widget.Insets {
	hPadding := int(float64(layout.ScreenWidth) * paddingConstant)

	return widget.Insets{
		Left:  hPadding,
		Right: hPadding,
	}
}

// NewResponsiveVerticalPadding creates vertical-only responsive padding.
// Useful for top/bottom spacing.
//
// Example usage:
//
//	padding := gui.NewResponsiveVerticalPadding(layout, widgets.PaddingStandard)
func NewResponsiveVerticalPadding(layout *widgets.LayoutConfig, paddingConstant float64) widget.Insets {
	vPadding := int(float64(layout.ScreenHeight) * paddingConstant)

	return widget.Insets{
		Top:    vPadding,
		Bottom: vPadding,
	}
}

// PaddingSide specifies which side(s) to apply padding to for NewResponsivePaddingSingle
type PaddingSide int

const (
	PaddingTop PaddingSide = iota
	PaddingBottom
	PaddingLeft
	PaddingRight
	PaddingTopLeft
	PaddingTopRight
	PaddingBottomLeft
	PaddingBottomRight
)

// NewResponsivePaddingSingle creates single-side responsive padding.
// Used for anchor layout positioning (e.g., Top-only, Left-only).
//
// Example usage:
//
//	padding := gui.NewResponsivePaddingSingle(layout, widgets.PaddingTight, gui.PaddingTop)
func NewResponsivePaddingSingle(layout *widgets.LayoutConfig, paddingConstant float64, side PaddingSide) widget.Insets {
	hPadding := int(float64(layout.ScreenWidth) * paddingConstant)
	vPadding := int(float64(layout.ScreenHeight) * paddingConstant)

	insets := widget.Insets{}

	switch side {
	case PaddingTop:
		insets.Top = vPadding
	case PaddingBottom:
		insets.Bottom = vPadding
	case PaddingLeft:
		insets.Left = hPadding
	case PaddingRight:
		insets.Right = hPadding
	case PaddingTopLeft:
		insets.Top = vPadding
		insets.Left = hPadding
	case PaddingTopRight:
		insets.Top = vPadding
		insets.Right = hPadding
	case PaddingBottomLeft:
		insets.Bottom = vPadding
		insets.Left = hPadding
	case PaddingBottomRight:
		insets.Bottom = vPadding
		insets.Right = hPadding
	}

	return insets
}
