package builders

import (
	"game_main/gui/specs"

	"github.com/ebitenui/ebitenui/widget"
)

// NewResponsiveRowPadding creates responsive padding insets for RowLayout.
// Replaces the verbose 6-line padding calculation that appears throughout the codebase.
//
// Example usage:
//
//	padding := builders.NewResponsiveRowPadding(layout, specs.PaddingExtraSmall)
//
// Replaces:
//
//	widget.Insets{
//	    Left:   int(float64(layout.ScreenWidth) * specs.PaddingExtraSmall),
//	    Right:  int(float64(layout.ScreenWidth) * specs.PaddingExtraSmall),
//	    Top:    int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall),
//	    Bottom: int(float64(layout.ScreenHeight) * specs.PaddingExtraSmall),
//	}
func NewResponsiveRowPadding(layout *specs.LayoutConfig, paddingConstant float64) widget.Insets {
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
//	padding := builders.NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall)
func NewResponsiveHorizontalPadding(layout *specs.LayoutConfig, paddingConstant float64) widget.Insets {
	hPadding := int(float64(layout.ScreenWidth) * paddingConstant)

	return widget.Insets{
		Left:  hPadding,
		Right: hPadding,
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

// ========================================
// ANCHOR LAYOUT HELPERS
// ========================================

// AnchorStartStart creates a Start-Start anchor layout (left-top aligned).
// Common for: Squad selectors, unit lists positioned on left side.
//
// Example usage:
//
//	leftPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
//	panel.GetWidget().LayoutData = builders.AnchorStartStart(leftPad, topOffset)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionStart,
//	    VerticalPosition:   widget.AnchorLayoutPositionStart,
//	    Padding: widget.Insets{
//	        Left: int(float64(layout.ScreenWidth) * specs.PaddingStandard),
//	        Top:  topOffset,
//	    },
//	}
func AnchorStartStart(leftPadding, topPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            widget.Insets{Left: leftPadding, Top: topPadding},
	}
}

// AnchorCenterStart creates a Center-Start anchor layout (center-top aligned).
// Common for: Navigation bars, top-center panels.
//
// Example usage:
//
//	topPad := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
//	panel.GetWidget().LayoutData = builders.AnchorCenterStart(topPad)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionCenter,
//	    VerticalPosition:   widget.AnchorLayoutPositionStart,
//	    Padding: widget.Insets{Top: topPad},
//	}
func AnchorCenterStart(topPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            widget.Insets{Top: topPadding},
	}
}

// AnchorEndCenter creates an End-Center anchor layout (right-middle aligned).
// Common for: Detail panels on right side.
//
// Example usage:
//
//	rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
//	panel.GetWidget().LayoutData = builders.AnchorEndCenter(rightPad)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionEnd,
//	    VerticalPosition:   widget.AnchorLayoutPositionCenter,
//	    Padding: widget.Insets{Right: rightPad},
//	}
func AnchorEndCenter(rightPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding:            widget.Insets{Right: rightPadding},
	}
}

// AnchorEndEnd creates an End-End anchor layout (right-bottom aligned).
// Common for: Combat log, panels positioned above bottom buttons.
//
// Example usage:
//
//	rightPad := int(float64(layout.ScreenWidth) * specs.PaddingTight)
//	bottomOffset := int(float64(layout.ScreenHeight) * (buttonHeight + spacing))
//	panel.GetWidget().LayoutData = builders.AnchorEndEnd(rightPad, bottomOffset)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionEnd,
//	    VerticalPosition:   widget.AnchorLayoutPositionEnd,
//	    Padding: widget.Insets{
//	        Right:  rightPad,
//	        Bottom: bottomOffset,
//	    },
//	}
func AnchorEndEnd(rightPadding, bottomPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding:            widget.Insets{Right: rightPadding, Bottom: bottomPadding},
	}
}

// AnchorEndStart creates an End-Start anchor layout (right-top aligned).
// Common for: Detail panels on right side at top.
//
// Example usage:
//
//	rightPad := int(float64(layout.ScreenWidth) * specs.PaddingStandard)
//	topPad := int(float64(layout.ScreenHeight) * specs.PaddingStandard)
//	panel.GetWidget().LayoutData = builders.AnchorEndStart(rightPad, topPad)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionEnd,
//	    VerticalPosition:   widget.AnchorLayoutPositionStart,
//	    Padding: widget.Insets{
//	        Right: rightPad,
//	        Top:   topPad,
//	    },
//	}
func AnchorEndStart(rightPadding, topPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            widget.Insets{Right: rightPadding, Top: topPadding},
	}
}

// AnchorCenterEnd creates a Center-End anchor layout (center-bottom aligned).
// Common for: Action button groups at bottom center.
//
// Example usage:
//
//	bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
//	panel.GetWidget().LayoutData = builders.AnchorCenterEnd(bottomPad)
//
// Replaces:
//
//	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
//	    HorizontalPosition: widget.AnchorLayoutPositionCenter,
//	    VerticalPosition:   widget.AnchorLayoutPositionEnd,
//	    Padding: widget.Insets{Bottom: bottomPad},
//	}
func AnchorCenterEnd(bottomPadding int) widget.AnchorLayoutData {
	return widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding:            widget.Insets{Bottom: bottomPadding},
	}
}

// CreateBottomActionBar creates a horizontally-laid-out button group anchored at bottom-center.
// Replaces the ~7 lines of boilerplate (spacing calc, bottomPad calc, anchor, CreateButtonGroup)
// that is repeated across multiple panel registries.
func CreateBottomActionBar(layout *specs.LayoutConfig, buttons []ButtonSpec) *widget.Container {
	spacing := int(float64(layout.ScreenWidth) * specs.PaddingTight)
	bottomPad := int(float64(layout.ScreenHeight) * specs.BottomButtonOffset)
	anchorLayout := AnchorCenterEnd(bottomPad)

	return CreateButtonGroup(ButtonGroupConfig{
		Buttons:    buttons,
		Direction:  widget.DirectionHorizontal,
		Spacing:    spacing,
		Padding:    NewResponsiveHorizontalPadding(layout, specs.PaddingExtraSmall),
		LayoutData: &anchorLayout,
	})
}
