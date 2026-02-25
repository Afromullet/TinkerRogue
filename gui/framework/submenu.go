package framework

import "github.com/ebitenui/ebitenui/widget"

// SubMenuController manages sub-menu visibility. Only one sub-menu can be open at a time.
type SubMenuController struct {
	menus  map[string]*widget.Container
	active string
}

func NewSubMenuController() *SubMenuController {
	return &SubMenuController{
		menus: make(map[string]*widget.Container),
	}
}

func (sc *SubMenuController) Register(name string, container *widget.Container) {
	sc.menus[name] = container
}

// Toggle returns a callback that toggles the named sub-menu.
// Opening one menu closes any other open menu.
func (sc *SubMenuController) Toggle(name string) func() {
	return func() {
		if sc.active == name {
			sc.menus[name].GetWidget().Visibility = widget.Visibility_Hide
			sc.active = ""
			return
		}
		sc.CloseAll()
		if c, ok := sc.menus[name]; ok {
			c.GetWidget().Visibility = widget.Visibility_Show
			sc.active = name
		}
	}
}

// Show opens the named sub-menu, closing any other open menu first.
func (sc *SubMenuController) Show(name string) {
	sc.CloseAll()
	if c, ok := sc.menus[name]; ok {
		c.GetWidget().Visibility = widget.Visibility_Show
		sc.active = name
	}
}

// IsActive returns true if the named sub-menu is currently open.
func (sc *SubMenuController) IsActive(name string) bool {
	return sc.active == name
}

// AnyActive returns true if any sub-menu is currently open.
func (sc *SubMenuController) AnyActive() bool {
	return sc.active != ""
}

func (sc *SubMenuController) CloseAll() {
	for _, c := range sc.menus {
		c.GetWidget().Visibility = widget.Visibility_Hide
	}
	sc.active = ""
}
