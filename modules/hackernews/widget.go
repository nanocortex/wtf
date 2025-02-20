package hackernews

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/view"
)

type Widget struct {
	view.ScrollableWidget

	stories  []Story
	settings *Settings
	err      error

	tviewApp *tview.Application
}

func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := &Widget{
		ScrollableWidget: view.NewScrollableWidget(tviewApp, redrawChan, pages, settings.Common),

		settings: settings,
		tviewApp: tviewApp,
	}

	widget.SetRenderFunction(widget.Render)
	widget.initializeKeyboardControls()

	return widget
}

/* -------------------- Exported Functions -------------------- */

func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	widget.fetchStoriesAsync()

	widget.Render()
}

// Render sets up the widget data for redrawing to the screen
func (widget *Widget) Render() {
	widget.Redraw(widget.content)
}

/* -------------------- Unexported Functions -------------------- */

// Create a function to fetch stories concurrently
func (widget *Widget) fetchStoriesAsync() {
	// Start fetching stories in a goroutine
	go func() {
		defer widget.Render()
		storyIds, err := GetStories(widget.settings.storyType)
		if err != nil {
			// Handle error in the main thread
			widget.tviewApp.QueueUpdateDraw(func() {
				widget.err = err
				widget.stories = nil
				widget.SetItemCount(0)
				widget.Render()
			})
			return
		}

		// Create a channel to collect stories
		storyChan := make(chan Story)
		var wg sync.WaitGroup

		// Launch goroutines for each story fetch
		for idx := 0; idx < widget.settings.numberOfStories && idx < len(storyIds); idx++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				story, err := GetStory(id)
				if err == nil {
					storyChan <- story
				}
			}(storyIds[idx])
		}

		// Wait for all goroutines to complete in a separate goroutine
		go func() {
			wg.Wait()
			close(storyChan)
		}()

		// Initialize stories slice
		widget.stories = make([]Story, 0, widget.settings.numberOfStories)

		// Read from channel and update widget for each story
		for story := range storyChan {
			widget.tviewApp.QueueUpdateDraw(func() {
				widget.stories = append(widget.stories, story)
				widget.SetItemCount(len(widget.stories))
				widget.Render()
			})
		}

	}()
}

func (widget *Widget) content() (string, string, bool) {
	title := fmt.Sprintf("%s - %s stories", widget.CommonSettings().Title, widget.settings.storyType)

	if widget.err != nil {
		return title, widget.err.Error(), true
	}

	if len(widget.stories) == 0 {
		return title, "No stories to display", false
	}

	var str string
	for idx, story := range widget.stories {
		u, _ := url.Parse(story.URL)

		row := fmt.Sprintf(
			`[%s]%2d. %s [lightblue](%s)[white]`,
			widget.RowColor(idx),
			idx+1,
			story.Title,
			strings.TrimPrefix(u.Host, "www."),
		)

		str += utils.HighlightableHelper(widget.View, row, idx, len(story.Title))
	}

	return title, str, false
}

func (widget *Widget) openComments() {
	story := widget.selectedStory()
	if story != nil {
		utils.OpenFile(story.CommentLink())
	}
}

func (widget *Widget) openStory() {
	story := widget.selectedStory()
	if story != nil {
		utils.OpenFile(story.Link())
	}
}

func (widget *Widget) selectedStory() *Story {
	var story *Story

	sel := widget.GetSelected()
	if sel >= 0 && widget.stories != nil && sel < len(widget.stories) {
		story = &widget.stories[sel]
	}

	return story
}
