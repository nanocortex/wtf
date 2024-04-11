package victorops

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/view"
)

// Widget contains text info
type Widget struct {
	view.TextWidget

	teams    []OnCallTeam
	settings *Settings
	err      error
}

// NewWidget creates a new widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, nil, settings.Common),
		settings:   settings,
	}

	widget.View.SetScrollable(true)
	widget.View.SetRegions(true)

	return &widget
}

// Refresh gets latest content for the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	teams, err := Fetch(widget.settings.apiID, widget.settings.apiKey)

	widget.err = err
	widget.teams = teams

	widget.Redraw(widget.content)
}

func (widget *Widget) content() (string, string, bool) {
	title := widget.CommonSettings().Title
	if widget.err != nil {
		return title, widget.err.Error(), true
	}
	teams := widget.teams
	var str string

	if len(teams) == 0 {
		return title, "No teams specified", false
	}

	for _, team := range teams {
		if len(widget.settings.team) > 0 && widget.settings.team != team.Slug {
			continue
		}

		str = fmt.Sprintf("%s[green]%s\n", str, team.Name)
		if len(team.OnCall) == 0 {
			str = fmt.Sprintf("%s[grey]no one\n", str)
		}
		for _, onCall := range team.OnCall {
			str = fmt.Sprintf("%s[white]%s - %s\n", str, onCall.Policy, onCall.Userlist)
		}

		str = fmt.Sprintf("%s\n", str)
	}

	if str == "" {
		str = "Could not find any teams to display"
	}
	return title, str, false
}
