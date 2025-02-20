package app

import (
	"fmt"
	"github.com/olebedev/config"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/wtf"
)

// Display is the container for the onscreen representation of a WtfApp
type Display struct {
	TabBar    *tview.Flex
	Grid      *tview.Grid
	Container *tview.Flex
	config    *config.Config
}

// NewDisplay creates and returns a Display
func NewDisplay(wtfApp *WtfApp) *Display {
	display := Display{
		TabBar:    tview.NewFlex(),
		Grid:      tview.NewGrid(),
		Container: tview.NewFlex(),
		config:    wtfApp.config,
	}

	firstWidget := wtfApp.currentScreen.widgets[0]
	display.Grid.SetBackgroundColor(
		wtf.ColorFor(
			firstWidget.CommonSettings().Colors.WidgetTheme.Background,
		),
	)

	display.build(wtfApp)

	return &display
}

/* -------------------- Unexported Functions -------------------- */

func (display *Display) add(widget wtf.Wtfable) {
	if widget.Disabled() {
		return
	}

	display.Grid.AddItem(
		widget.TextView(),
		widget.CommonSettings().Top,
		widget.CommonSettings().Left,
		widget.CommonSettings().Height,
		widget.CommonSettings().Width,
		0,
		0,
		false,
	)
}

func (display *Display) build(wtfApp *WtfApp) {
	cols := utils.ToInts(display.config.UList("wtf.grid.columns"))
	rows := utils.ToInts(display.config.UList("wtf.grid.rows"))

	display.Grid.SetColumns(cols...)
	display.Grid.SetRows(rows...)
	display.Grid.SetBorder(false)

	for _, widget := range wtfApp.currentScreen.widgets {
		display.add(widget)
	}

	firstWidget := wtfApp.currentScreen.widgets[0]
	for _, screen := range wtfApp.screens {
		tv := tview.NewTextView()
		tv.SetText(fmt.Sprintf("%d: %v", screen.index, screen.title))
		if screen.index == wtfApp.currentScreen.index {
			tv.SetBackgroundColor(tview.Styles.InverseTextColor)
		} else {
			tv.SetBackgroundColor(
				wtf.ColorFor(
					firstWidget.CommonSettings().Colors.WidgetTheme.Background,
				),
			)
		}

		display.TabBar.AddItem(tv, 0, 1, false)
	}
}
