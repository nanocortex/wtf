package azuredevopspr

import (
	"context"
	"fmt"
	azr "github.com/microsoft/azure-devops-go-api/azuredevops"
	azrGit "github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/pkg/errors"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/view"
	"time"
)

type Widget struct {
	view.ScrollableWidget
	settings             *Settings
	displayBuffer        string
	ctx                  context.Context
	git                  azrGit.Client
	myPullRequests       []azrGit.GitPullRequest
	myReviewPullRequests []azrGit.GitPullRequest
	err                  error
}

func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		ScrollableWidget: view.NewScrollableWidget(tviewApp, redrawChan, pages, settings.Common),
		settings:         settings,
	}

	widget.SetRenderFunction(widget.Render)
	widget.initializeKeyboardControls()

	connection := azr.NewPatConnection(settings.organizationUrl, settings.apiToken)
	ctx := context.Background()

	git, err := azrGit.NewClient(ctx, connection)
	if err != nil {
		widget.displayBuffer = errors.Wrap(err, "could not create client 2").Error()
	} else {
		widget.git = git
		widget.ctx = ctx
	}

	return &widget
}

func (widget *Widget) Refresh() {
	projects := utils.ToStrs(widget.settings.projects)
	if widget.git == nil {
		return
	}

	widget.myPullRequests = nil
	widget.myReviewPullRequests = nil

	itemCount := 0

	for _, project := range projects {
		pullRequests, err := widget.getMyPullRequests(project)
		if err != nil {
			widget.err = err
			widget.myPullRequests = nil
		} else {
			widget.myPullRequests = append(widget.myPullRequests, pullRequests...)
			itemCount += len(widget.myPullRequests)
		}
	}

	for _, project := range projects {
		pullRequests, err := widget.getMyReviewPullRequests(project)
		if err != nil {
			widget.err = err
			widget.myReviewPullRequests = nil
		} else {
			widget.myReviewPullRequests = append(widget.myReviewPullRequests, pullRequests...)
			itemCount += len(widget.myReviewPullRequests)
		}
	}

	widget.SetItemCount(itemCount - 1)

	widget.Render()
}

// Render sets up the widget data for redrawing to the screen
func (widget *Widget) Render() {
	widget.Redraw(widget.content)
}

func (widget *Widget) content() (string, string, bool) {
	//title := fmt.Sprintf("%s - %s stories", widget.CommonSettings().Title, widget.settings.storyType)
	title := widget.CommonSettings().Title

	if widget.err != nil {
		return title, widget.err.Error(), true
	}

	if len(widget.myPullRequests) == 0 {
		return title, "No pull requests to display", false
	}

	var str string
	str += widget.displayPullRequests("Created by me", widget.myPullRequests, 0)
	str += "\n"
	str += widget.displayPullRequests("To review by me", widget.myReviewPullRequests, len(widget.myPullRequests))

	return title, str, false
}

func (widget *Widget) displayPullRequests(title string, pullRequests []azrGit.GitPullRequest, selStart int) string {
	var str = "[red]" + title + "[white]\n"
	for idx, pullRequest := range pullRequests {

		mergeStatusDisplay := ""
		mergeStatus := *pullRequest.MergeStatus

		if mergeStatus == "succeeded" {
			mergeStatusDisplay = "[green]M[white]"
		} else {
			mergeStatusDisplay = "[red]E[white]"
		}

		acDisplay := ""
		acActivated := pullRequest.AutoCompleteSetBy != nil

		if acActivated {
			acDisplay = "[green]AC[white]"
		} else {
			acDisplay = "[red]NAC[white]"
		}

		hours := time.Now().Sub(pullRequest.CreationDate.Time).Hours()
		timeSinceCreation := ""
		if hours > 24 {
			timeSinceCreation = fmt.Sprintf("%dd", int(hours/24))
		} else {
			timeSinceCreation = fmt.Sprintf("%dh", int(hours))
		}

		row := fmt.Sprintf(
			`[%s][%s] [%s] [grey]%4s[white] %s [blue]%s[white]`,
			widget.RowColor(idx+selStart),
			mergeStatusDisplay,
			acDisplay,
			timeSinceCreation,
			*pullRequest.Title,
			*pullRequest.Repository.Name,
		)
		str += utils.HighlightableHelper(widget.View, row, idx+selStart, len(*pullRequest.Title))
	}
	return str
}

func (widget *Widget) open() {
	pullRequest := widget.selectedPullRequest()
	if pullRequest != nil {
		url := fmt.Sprintf("%s/%s/_git/%s/pullrequest/%d", widget.settings.organizationUrl,
			*pullRequest.Repository.Project.Name, *pullRequest.Repository.Name, *pullRequest.PullRequestId)
		utils.OpenFile(url)
	}
}

func (widget *Widget) selectedPullRequest() *azrGit.GitPullRequest {
	var pullRequest *azrGit.GitPullRequest

	sel := widget.GetSelected()

	if sel >= 0 && sel < len(widget.myPullRequests)+len(widget.myReviewPullRequests) {
		if sel < len(widget.myPullRequests) {
			pullRequest = &widget.myPullRequests[sel]
		} else {
			pullRequest = &widget.myReviewPullRequests[sel-len(widget.myPullRequests)]
		}

	}

	return pullRequest
}
