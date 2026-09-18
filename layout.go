package termforge

//
// Layout
//

// Layout is tab content that owns its own window arrangement.
//
// A Tab is a container for layouts of any kind and knows nothing beyond this
// interface: it calls BuildLayout to hand down the assigned canvas, then Draw,
// and routes events through HandleEvent. *WidgetTree is the Vim-style tiling
// implementation and *WidgetsList the flat one the App uses for its chrome; a
// future layout (a different window arrangement, a single full-tab form, a
// whole embedded app) implements the same interface.
//
// A Layout draws no status line of its own: in the tiling tree every pane
// paints its own status row. A layout nested inside a pane must therefore also
// implement StatusLineDrawer to satisfy NodeWidget.
//
// Do not add Tab methods that forward into a Layout. Code needing
// arrangement-specific operations holds the concrete layout instead.
type Layout interface {
	Widget // HandleEvent, Draw

	// BuildLayout computes geometry for the assigned canvas. Called before
	// Draw on every frame.
	BuildLayout(c Canvas)
}
