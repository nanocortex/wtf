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
func NewDisplay(screens []WtfScreen, widgets []wtf.Wtfable, config *config.Config) *Display {
	display := Display{
		TabBar:    tview.NewFlex(),
		Grid:      tview.NewGrid(),
		Container: tview.NewFlex(),
		config:    config,
	}

	firstWidget := widgets[0]
	display.Grid.SetBackgroundColor(
		wtf.ColorFor(
			firstWidget.CommonSettings().Colors.WidgetTheme.Background,
		),
	)

	display.build(screens, widgets)

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

func (display *Display) build(screens []WtfScreen, widgets []wtf.Wtfable) {
	cols := utils.ToInts(display.config.UList("wtf.grid.columns"))
	rows := utils.ToInts(display.config.UList("wtf.grid.rows"))

	display.Grid.SetColumns(cols...)
	display.Grid.SetRows(rows...)
	display.Grid.SetBorder(false)

	for _, widget := range widgets {
		display.add(widget)
	}

	for i, screen := range screens {
		tv := tview.NewTextView()
		tv.SetText(fmt.Sprintf("%d: %v", i, screen.title))
		display.TabBar.AddItem(tv, 0, 1, false)
	}

	//display.TabBar.SetDirection(tview.FlexColumn)
	//display.TabBar.SetBorder(true)

}
