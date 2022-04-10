package airbrake

import "github.com/gdamore/tcell"

func (widget *Widget) initializeKeyboardControls() {
	widget.InitializeHelpTextKeyboardControl(widget.ShowHelp)
	widget.InitializeRefreshKeyboardControl(widget.Refresh)

	widget.SetKeyboardChar("o", widget.openGroup, "Open group in browser")
	widget.SetKeyboardChar("s", widget.resolveGroup, "Resolve group")
	widget.SetKeyboardChar("m", widget.muteGroup, "Mute group")
	widget.SetKeyboardChar("u", widget.unmuteGroup, "Unmute group")
	widget.SetKeyboardChar("t", widget.toggleDisplayText, "Toggle between title and compare views")

	widget.SetKeyboardChar("j", widget.Next, "Select next item")
	widget.SetKeyboardChar("k", widget.Prev, "Select previous item")

	widget.SetKeyboardKey(tcell.KeyDown, widget.Next, "Select next item")
	widget.SetKeyboardKey(tcell.KeyUp, widget.Prev, "Select previous item")
	widget.SetKeyboardKey(tcell.KeyEsc, widget.Unselect, "Clear selection")
	widget.SetKeyboardKey(tcell.KeyEnter, widget.viewGroup, "View group")
}
