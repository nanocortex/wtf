package todoist

import (
	"encoding/json"
	"fmt"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/view"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Widget is the container for your module's data
type Widget struct {
	view.ScrollableWidget

	settings *Settings
	items    []TodoistTask
	err      error
}

// TodoistTask represents a task in Todoist
type TodoistTask struct {
	Content  *string     `json:"content,omitempty"`
	Due      *TodoistDue `json:"due,omitempty"`
	Labels   []string    `json:"labels,omitempty"`
	Url      *string     `json:"url,omitempty"`
	Priority int         `json:"priority"`
}

// TodoistDue represents the due date structure
type TodoistDue struct {
	DateStr     *string `json:"date,omitempty"`
	DateTimeStr *string `json:"datetime,omitempty"`
	String      *string `json:"string,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
	IsRecurring *bool   `json:"is_recurring,omitempty"`
}

func (due *TodoistDue) Date() *time.Time {
	if due.DateStr != nil {
		date, err := time.Parse("2006-01-02", *due.DateStr)
		if err != nil {
			return nil
		}
		return &date
	}
	return nil
}

func (due *TodoistDue) DateTime() *time.Time {
	if due.DateTimeStr != nil {
		date, err := time.Parse("2006-01-02T15:04:05", *due.DateTimeStr)
		if err != nil {
			return nil
		}
		return &date
	}
	return nil
}

func (item *TodoistTask) IsOverdue() bool {
	if item.Due != nil && item.Due.Date() != nil {
		// Get the start of today (00:00)
		startOfToday := time.Now().Truncate(24 * time.Hour)

		// Check if the due date is before the start of today (midnight)
		return item.Due.Date().Before(startOfToday)
	}
	return false
}

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		ScrollableWidget: view.NewScrollableWidget(tviewApp, redrawChan, pages, settings.common),

		settings: settings,
		items:    make([]TodoistTask, 0),
	}

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	widget.loadTasks()
	widget.SetItemCount(len(widget.items))

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) loadTasks() {
	// Create an HTTP client
	httpClient := &http.Client{}

	// Encode the query parameters
	params := url.Values{}
	params.Add("filter", widget.settings.filter)

	// Create the request
	req, err := http.NewRequest("GET", "https://api.todoist.com/rest/v2/tasks?"+params.Encode(), nil)
	if err != nil {
		widget.err = err
		return
	}

	// Set the authorization header
	req.Header.Add("Authorization", "Bearer "+widget.settings.apiKey)

	// Perform the request
	resp, err := httpClient.Do(req)
	if err != nil {
		widget.err = err
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	//// Read the response body as a string
	//bodyBytes, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	fmt.Println("Error reading response body:", err)
	//	return
	//}
	//
	//// Convert body bytes to string for display
	//bodyString := string(bodyBytes)
	//
	//widget.err = fmt.Errorf("Error: %s", bodyString)
	//return

	// Decode the JSON response
	var items []TodoistTask
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		widget.err = err
		return
	}

	// Sort the items by due date
	sort.Slice(items, func(i, j int) bool {
		if items[i].Due != nil && items[j].Due != nil {
			return items[i].Due.Date().Before(*items[j].Due.Date())
		}
		return false
	})

	widget.items = items
}

func (widget *Widget) content() string {
	if widget.err != nil {
		return widget.err.Error()
	}
	if len(widget.items) == 0 {
		return "No items"
	}
	str := ""
	// Check for tasks due today
	anyToday := false
	for _, item := range widget.items {
		if item.Due != nil && item.Due.Date().Truncate(24*time.Hour).Equal(time.Now().Truncate(24*time.Hour)) {
			anyToday = true
			break
		}
	}

	if anyToday {
		str += "[green]Today[-:-:-]\n"
		for _, item := range widget.items {
			if item.Due != nil && item.Due.Date().Truncate(24*time.Hour).Equal(time.Now().Truncate(24*time.Hour)) {
				tags := formatTags(item.Labels)
				str += fmt.Sprintf("  [green]%s[-:-:-] %s %s\n", item.Due.Date().Format("02.01"), *item.Content, tags)
			}
		}

	}

	// Check for overdue tasks
	anyOverdue := false
	for _, item := range widget.items {
		if item.IsOverdue() {
			anyOverdue = true
			break
		}
	}

	if anyOverdue {
		str += "[red]Overdue[-:-:-]\n"
		for _, item := range widget.items {
			if item.IsOverdue() {
				tags := formatTags(item.Labels)
				str += fmt.Sprintf("  [red]%s[-:-:-] %s %s\n", item.Due.Date().Format("02.01"), *item.Content, tags)
			}
		}
	}

	// Check for upcoming tasks
	anyUpcoming := false
	for _, item := range widget.items {
		if !item.IsOverdue() && item.Due != nil && item.Due.Date().After(time.Now()) {
			anyUpcoming = true
			break
		}
	}

	if anyUpcoming {
		str += "[blue]Upcoming[-:-:-]\n"
		for _, item := range widget.items {
			if !item.IsOverdue() && item.Due != nil && item.Due.Date().After(time.Now()) {
				tags := formatTags(item.Labels)
				color := "blue"
				if item.Due.Date().Before(time.Now()) {
					color = "red"
				}
				//row := fmt.Sprintf("  [%s]%s[/] %s %s", color, item.Due.Format("02.01"), item.Content, tags)
				//rows = append(rows, tview.NewTextView().SetText(row))

				str += fmt.Sprintf("  [%s]%s[-:-:-] %s %s\n", color, item.Due.Date().Format("02.01"), *item.Content, tags)
			}
		}
	}

	return str
}

func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), true
	})
}

// formatTags formats the labels into a string
func formatTags(labels []string) string {
	tags := ""
	for _, label := range labels {
		tags += fmt.Sprintf("[grey]%s[-:-:-]", label)
	}
	return tags
}

// For parsing simple YYYY-MM-DD dates
type CustomDate struct {
	Date time.Time
}

func (cd *CustomDate) UnmarshalJSON(data []byte) error {
	// Remove quotes from string
	dateStr := strings.Trim(string(data), "\"")

	// Parse the date string
	parsedTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return err
	}

	cd.Date = parsedTime
	return nil
}
