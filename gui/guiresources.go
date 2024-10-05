package gui

import (
	"image/color"
	"strconv"

	"github.com/ebitenui/ebitenui/image"
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

var smallFace, _ = loadFont(30)
var largeFace, _ = loadFont(50)
var buttonImage, _ = loadButtonImage()
var defaultWidgetColor = e_image.NewNineSliceColor(color.NRGBA{0x13, 0x1a, 0x22, 0xff})

var PanelRes *panelResources = newPanelResources()
var ListRes *listResources = newListResources()
var TextAreaRes *textAreaResources = newTextAreaResources()

const (
	textIdleColor                  = "dff4ff"
	textDisabledColor              = "5a7a91"
	listSelectedBackground         = "4b687a"
	listDisabledSelectedBackground = "2a3944"
	listFocusedBackground          = "2a3944"
)

type panelResources struct {
	image    *e_image.NineSlice
	titleBar *e_image.NineSlice
	padding  widget.Insets
}

func newPanelResources() *panelResources {
	i, err := loadImageNineSlice("../assets/guiassets/panel-idle.png", 10, 10)
	if err != nil {
		return nil
	}
	t, err := loadImageNineSlice("../assets/guiassets/titlebar-idle.png", 10, 10)
	if err != nil {
		return nil
	}
	return &panelResources{
		image:    i,
		titleBar: t,
		padding: widget.Insets{
			Left:   30,
			Right:  30,
			Top:    20,
			Bottom: 20,
		},
	}
}

type listResources struct {
	image        *widget.ScrollContainerImage
	track        *widget.SliderTrackImage
	trackPadding widget.Insets
	handle       *widget.ButtonImage
	handleSize   int
	face         font.Face
	entry        *widget.ListEntryColor
	entryPadding widget.Insets
}

func newListResources() *listResources {
	idle, err := newImageFromFile("../assets/guiassets/list-idle.png")
	if err != nil {
		return nil
	}

	disabled, err := newImageFromFile("../assets/guiassets/list-disabled.png")
	if err != nil {
		return nil
	}

	mask, err := newImageFromFile("../assets/guiassets/list-mask.png")
	if err != nil {
		return nil
	}

	trackIdle, err := newImageFromFile("../assets/guiassets/list-track-idle.png")
	if err != nil {
		return nil
	}

	trackDisabled, err := newImageFromFile("../assets/guiassets/list-track-disabled.png")
	if err != nil {
		return nil
	}

	handleIdle, err := newImageFromFile("../assets/guiassets/slider-handle-idle.png")
	if err != nil {
		return nil
	}

	handleHover, err := newImageFromFile("../assets/guiassets/slider-handle-hover.png")
	if err != nil {
		return nil
	}

	return &listResources{
		image: &widget.ScrollContainerImage{
			Idle:     image.NewNineSlice(idle, [3]int{25, 12, 22}, [3]int{25, 12, 25}),
			Disabled: image.NewNineSlice(disabled, [3]int{25, 12, 22}, [3]int{25, 12, 25}),
			Mask:     image.NewNineSlice(mask, [3]int{26, 10, 23}, [3]int{26, 10, 26}),
		},

		track: &widget.SliderTrackImage{
			Idle:     image.NewNineSlice(trackIdle, [3]int{5, 0, 0}, [3]int{25, 12, 25}),
			Hover:    image.NewNineSlice(trackIdle, [3]int{5, 0, 0}, [3]int{25, 12, 25}),
			Disabled: image.NewNineSlice(trackDisabled, [3]int{0, 5, 0}, [3]int{25, 12, 25}),
		},

		trackPadding: widget.Insets{
			Top:    5,
			Bottom: 24,
		},

		handle: &widget.ButtonImage{
			Idle:     image.NewNineSliceSimple(handleIdle, 0, 5),
			Hover:    image.NewNineSliceSimple(handleHover, 0, 5),
			Pressed:  image.NewNineSliceSimple(handleHover, 0, 5),
			Disabled: image.NewNineSliceSimple(handleIdle, 0, 5),
		},

		handleSize: 5,
		face:       smallFace,

		entry: &widget.ListEntryColor{
			Unselected:         hexToColor(textIdleColor),
			DisabledUnselected: hexToColor(textDisabledColor),

			Selected:         hexToColor(textIdleColor),
			DisabledSelected: hexToColor(textDisabledColor),

			SelectedBackground:         hexToColor(listSelectedBackground),
			DisabledSelectedBackground: hexToColor(listDisabledSelectedBackground),

			FocusedBackground:         hexToColor(listFocusedBackground),
			SelectedFocusedBackground: hexToColor(listSelectedBackground),
		},

		entryPadding: widget.Insets{
			Left:   30,
			Right:  30,
			Top:    2,
			Bottom: 2,
		},
	}
}

type textAreaResources struct {
	image        *widget.ScrollContainerImage
	track        *widget.SliderTrackImage
	trackPadding widget.Insets
	handle       *widget.ButtonImage
	handleSize   int
	face         font.Face
	entryPadding widget.Insets
}

func newTextAreaResources() *textAreaResources {
	idle, err := newImageFromFile("../assets/guiassets/list-idle.png")
	if err != nil {
		return nil
	}

	disabled, err := newImageFromFile("../assets/guiassets/list-disabled.png")
	if err != nil {
		return nil
	}

	mask, err := newImageFromFile("../assets/guiassets/list-mask.png")
	if err != nil {
		return nil
	}

	trackIdle, err := newImageFromFile("../assets/guiassets/list-track-idle.png")
	if err != nil {
		return nil
	}

	trackDisabled, err := newImageFromFile("../assets/guiassets/list-track-disabled.png")
	if err != nil {
		return nil
	}

	handleIdle, err := newImageFromFile("../assets/guiassets/slider-handle-idle.png")
	if err != nil {
		return nil
	}

	handleHover, err := newImageFromFile("../assets/guiassets/slider-handle-hover.png")
	if err != nil {
		return nil
	}

	return &textAreaResources{
		image: &widget.ScrollContainerImage{
			Idle:     image.NewNineSlice(idle, [3]int{25, 12, 22}, [3]int{25, 12, 25}),
			Disabled: image.NewNineSlice(disabled, [3]int{25, 12, 22}, [3]int{25, 12, 25}),
			Mask:     image.NewNineSlice(mask, [3]int{26, 10, 23}, [3]int{26, 10, 26}),
		},

		track: &widget.SliderTrackImage{
			Idle:     image.NewNineSlice(trackIdle, [3]int{5, 0, 0}, [3]int{25, 12, 25}),
			Hover:    image.NewNineSlice(trackIdle, [3]int{5, 0, 0}, [3]int{25, 12, 25}),
			Disabled: image.NewNineSlice(trackDisabled, [3]int{0, 5, 0}, [3]int{25, 12, 25}),
		},

		trackPadding: widget.Insets{
			Top:    5,
			Bottom: 24,
		},

		handle: &widget.ButtonImage{
			Idle:     image.NewNineSliceSimple(handleIdle, 0, 5),
			Hover:    image.NewNineSliceSimple(handleHover, 0, 5),
			Pressed:  image.NewNineSliceSimple(handleHover, 0, 5),
			Disabled: image.NewNineSliceSimple(handleIdle, 0, 5),
		},

		handleSize: 5,
		face:       smallFace,

		entryPadding: widget.Insets{
			Left:   30,
			Right:  30,
			Top:    2,
			Bottom: 2,
		},
	}
}

func loadButtonImage() (*widget.ButtonImage, error) {

	idle, _ := loadImageNineSlice("../assets/guiassets/button-idle.png", 10, 10)

	hover := e_image.NewNineSliceColor(color.NRGBA{R: 130, G: 130, B: 150, A: 255})

	pressed := e_image.NewNineSliceColor(color.NRGBA{R: 100, G: 100, B: 120, A: 255})

	return &widget.ButtonImage{
		Idle:    idle,
		Hover:   hover,
		Pressed: pressed,
	}, nil

}

func loadFont(size float64) (font.Face, error) {
	ttfFont, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, err
	}

	return truetype.NewFace(ttfFont, &truetype.Options{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	}), nil
}

func loadImageNineSlice(path string, centerWidth int, centerHeight int) (*image.NineSlice, error) {
	i, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}
	w := i.Bounds().Dx()
	h := i.Bounds().Dy()
	return e_image.NewNineSlice(i,
			[3]int{(w - centerWidth) / 2, centerWidth, w - (w-centerWidth)/2 - centerWidth},
			[3]int{(h - centerHeight) / 2, centerHeight, h - (h-centerHeight)/2 - centerHeight}),
		nil
}

func newImageFromFile(path string) (*ebiten.Image, error) {
	f, _, err := ebitenutil.NewImageFromFile(path)

	return f, err
}

func hexToColor(h string) color.Color {
	u, err := strconv.ParseUint(h, 16, 0)
	if err != nil {
		panic(err)
	}

	return color.NRGBA{
		R: uint8(u & 0xff0000 >> 16),
		G: uint8(u & 0xff00 >> 8),
		B: uint8(u & 0xff),
		A: 255,
	}
}
