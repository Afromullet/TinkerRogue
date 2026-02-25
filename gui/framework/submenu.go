package framework

import "github.com/ebitenui/ebitenui/widget"

// SubMenuController manages sub-menu visibility. Only one sub-menu can be open at a time.
//
// When parent is non-nil, panels are fully removed from the widget tree when hidden
// and re-added when shown. This prevents ebitenui's SetupInputLayer from running on
// hidden panels (their ScrollContainers create BlockLower input layers at stale positions,
// blocking clicks on overlapping visible panels).
//
// When parent is nil, falls back to visibility toggling only.
type SubMenuController struct {
	menus  map[string]*widget.Container
	parent *widget.Container // root container for add/remove; nil = visibility-only
	active string
}

func NewSubMenuController(parent ...*widget.Container) *SubMenuController {
	sc := &SubMenuController{
		menus: make(map[string]*widget.Container),
	}
	if len(parent) > 0 {
		sc.parent = parent[0]
	}
	return sc
}

func (sc *SubMenuController) Register(name string, container *widget.Container) {
	sc.menus[name] = container
}

// Toggle returns a callback that toggles the named sub-menu.
// Opening one menu closes any other open menu.
func (sc *SubMenuController) Toggle(name string) func() {
	return func() {
		if sc.active == name {
			sc.hidePanel(sc.menus[name])
			sc.active = ""
			return
		}
		sc.CloseAll()
		if c, ok := sc.menus[name]; ok {
			sc.showPanel(c)
			sc.active = name
		}
	}
}

// Show opens the named sub-menu, closing any other open menu first.
func (sc *SubMenuController) Show(name string) {
	sc.CloseAll()
	if c, ok := sc.menus[name]; ok {
		sc.showPanel(c)
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
		sc.hidePanel(c)
	}
	sc.active = ""
}

// showPanel makes a panel visible. If parent is set, adds the panel to the widget tree.
func (sc *SubMenuController) showPanel(c *widget.Container) {
	if sc.parent != nil {
		sc.parent.AddChild(c)
	}
	c.GetWidget().Visibility = widget.Visibility_Show
}

// hidePanel hides a panel. If parent is set, removes the panel from the widget tree
// so its ScrollContainers don't create blocking input layers at stale positions.
func (sc *SubMenuController) hidePanel(c *widget.Container) {
	c.GetWidget().Visibility = widget.Visibility_Hide
	if sc.parent != nil {
		sc.parent.RemoveChild(c)
	}
}
