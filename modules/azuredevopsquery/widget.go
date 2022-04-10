package azuredevopsquery

import (
	"context"
	"fmt"
	azr "github.com/microsoft/azure-devops-go-api/azuredevops"
	azrWorkItemTracking "github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
	"github.com/pkg/errors"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/view"
	"strings"
)

type Widget struct {
	view.ScrollableWidget
	settings      *Settings
	displayBuffer string
	ctx           context.Context
	api           azrWorkItemTracking.Client
	workItems     []azrWorkItemTracking.WorkItem
	err           error
}

func NewWidget(tviewApp *tview.Application, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		ScrollableWidget: view.NewScrollableWidget(tviewApp, pages, settings.Common),
		settings:         settings,
	}

	widget.SetRenderFunction(widget.Render)
	widget.initializeKeyboardControls()

	connection := azr.NewPatConnection(settings.organizationUrl, settings.apiToken)
	ctx := context.Background()

	api, err := azrWorkItemTracking.NewClient(ctx, connection)
	if err != nil {
		widget.displayBuffer = errors.Wrap(err, "could not create client 2").Error()
	} else {
		widget.api = api
		widget.ctx = ctx
	}

	return &widget
}

func (widget *Widget) Refresh() {
	if widget.api == nil {
		return
	}

	widget.workItems = nil

	workItems, err := widget.getWorkItems()
	if err != nil {
		widget.err = err
		widget.workItems = nil
		widget.SetItemCount(0)
	} else {
		widget.workItems = append(widget.workItems, workItems...)
		widget.SetItemCount(len(widget.workItems))
	}

	widget.Render()
}

// Render sets up the widget data for redrawing to the screen
func (widget *Widget) Render() {
	widget.Redraw(widget.content)
}

func (widget *Widget) content() (string, string, bool) {
	title := widget.CommonSettings().Title

	if widget.err != nil {
		return title, widget.err.Error(), true
	}

	if len(widget.workItems) == 0 {
		return title, "No items to display", false
	}

	var str string
	//var totalRemainingWork float64
	//var totalOriginalEstimate float64
	//var totalCompletedWork float64
	var totalStoryPoints float64
	for idx, workItem := range widget.workItems {

		id := *workItem.Id
		title := (*workItem.Fields)["System.Title"]
		status := (*workItem.Fields)["System.State"]
		statusColor := "white"
		if status == "Active" {
			statusColor = "#007acc"
		} else if status == "New" {
			statusColor = "#b2b2b2"
		} else if status == "Resolved" {
			statusColor = "#ff9d00"
		}

		itemType := (*workItem.Fields)["System.WorkItemType"]
		itemTypeColor := "white"
		if itemType == "Task" {
			itemTypeColor = "#f2cb1d"
		} else if itemType == "Bug" {
			itemTypeColor = "#cc293d"
		}

		//originalEstimate := (*workItem.Fields)["Microsoft.VSTS.Scheduling.OriginalEstimate"]
		//originalEstimateDisplay := ""
		//if originalEstimate != nil {
		//	totalOriginalEstimate += originalEstimate.(float64)
		//	originalEstimateDisplay = fmt.Sprintf("%vh", originalEstimate)
		//	if originalEstimateDisplay == "0h" {
		//		originalEstimateDisplay = ""
		//	}
		//}

		//remainingWork := (*workItem.Fields)["Microsoft.VSTS.Scheduling.RemainingWork"]
		//remainingWorkDisplay := ""
		//if remainingWork != nil {
		//	totalRemainingWork += remainingWork.(float64)
		//	remainingWorkDisplay = fmt.Sprintf("%vh", remainingWork)
		//	if remainingWorkDisplay == "0h" {
		//		remainingWorkDisplay = ""
		//	}
		//}

		//completedWork := (*workItem.Fields)["Microsoft.VSTS.Scheduling.CompletedWork"]
		//completedWorkDisplay := ""
		//if completedWork != nil {
		//	totalCompletedWork += completedWork.(float64)
		//	completedWorkDisplay = fmt.Sprintf("%vh", completedWork)
		//} else {
		//	completedWorkDisplay = "0h"
		//}

		storyPoints := (*workItem.Fields)["Microsoft.VSTS.Scheduling.StoryPoints"]
		storyPointsDisplay := ""
		if storyPoints != nil {
			totalStoryPoints += storyPoints.(float64)
			storyPointsDisplay = fmt.Sprintf("%v", storyPoints)
		}

		row := fmt.Sprintf(`[%s][#%v] [%s]%4s[white] [%s]%v[white] [green]%v [white] %v`,
			widget.RowColor(idx), id, itemTypeColor, itemType, statusColor, "•", storyPointsDisplay, title)

		str += utils.HighlightableHelper(widget.View, row, idx, 20)
	}

	str += fmt.Sprintf("Total left: [green]%.0f[white]", totalStoryPoints)

	return title, str, false
}

func (widget *Widget) open() {
	workItem := widget.selectedWorkItem()
	if workItem != nil {
		iterationPath := (*workItem.Fields)["System.IterationPath"]
		project := ""
		if strings.Contains(iterationPath.(string), "\\") {
			project = strings.Split(iterationPath.(string), "\\")[0]
		} else {
			project = iterationPath.(string)
		}
		url := fmt.Sprintf("%s/%s/_workitems/edit/%d", widget.settings.organizationUrl,
			project, *workItem.Id)

		utils.OpenFile(url)
	}
}

func (widget *Widget) selectedWorkItem() *azrWorkItemTracking.WorkItem {
	var workItem *azrWorkItemTracking.WorkItem

	sel := widget.GetSelected()
	if sel >= 0 && widget.workItems != nil && sel < len(widget.workItems) {
		workItem = &widget.workItems[sel]
	}

	return workItem
}
