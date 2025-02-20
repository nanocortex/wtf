package uptimekuma

import (
	"encoding/base64"
	"fmt"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/view"
	"io"
	"net/http"
	"sort"
)

// Widget is the container for your module's data
type Widget struct {
	view.ScrollableWidget

	settings *Settings

	items []MonitorItem
	err   error
}

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		ScrollableWidget: view.NewScrollableWidget(tviewApp, redrawChan, pages, settings.common),

		settings: settings,
		items:    make([]MonitorItem, 0),
	}

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	widget.items, widget.err = widget.getMonitorItems()

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) content() string {
	if widget.err != nil {
		return widget.err.Error()
	}

	if len(widget.items) == 0 {
		return "No items"
	}

	str := ""

	for _, item := range widget.items {

		//str +=  getCheckmark(item.Status) + " " + item.Name + "[grey]" + strconv.Itoa(item.ResponseTime) + "[-:-:-]\n"
		str += fmt.Sprintf("%s %-20s [grey]%dms[-:-:-]\n", getCheckmark(item.Status), item.Name, item.ResponseTime)

	}

	return str
}

func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), false
	})
}

func (widget *Widget) getMonitorItems() ([]MonitorItem, error) {
	// Create HTTP client with basic auth
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/metrics", widget.settings.url), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth header
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", widget.settings.login, widget.settings.password)))
	req.Header.Add("Authorization", "Basic "+auth)

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	dataRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Prometheus data
	data, err := Parse(string(dataRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus data: %w", err)
	}

	var responseTimeMetric, statusMetric *PrometheusMetric
	for _, metric := range data {
		if metric.Name == "monitor_response_time" {
			responseTimeMetric = metric
		}
		if metric.Name == "monitor_status" {
			statusMetric = metric
		}
	}

	var kumaItems []MonitorItem
	if responseTimeMetric != nil && statusMetric != nil {
		for _, item := range responseTimeMetric.Items {
			kumaItem := MonitorItem{
				Name:         item.Labels["monitor_name"],
				URL:          item.Labels["monitor_url"],
				ResponseTime: int(item.Value),
				Status:       MonitorStatusUnknown,
			}

			// Find matching status item
			for _, statusItem := range statusMetric.Items {
				if statusItem.Labels["monitor_name"] == kumaItem.Name {
					switch int(statusItem.Value) {
					case 0:
						kumaItem.Status = MonitorStatusDown
					case 1:
						kumaItem.Status = MonitorStatusUp
					case 2:
						kumaItem.Status = MonitorStatusPending
					case 3:
						kumaItem.Status = MonitorStatusMaintenance
					default:
						kumaItem.Status = MonitorStatusUnknown
					}
					break
				}
			}

			kumaItems = append(kumaItems, kumaItem)
		}
	}

	// Sort items by status
	sort.Slice(kumaItems, func(i, j int) bool {
		return kumaItems[i].Status < kumaItems[j].Status
	})

	return kumaItems, nil
}

// MonitorItem struct represents an item to be monitored
type MonitorItem struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	ResponseTime int           `json:"responseTime"`
	Status       MonitorStatus `json:"status"`
}

// NewMonitorItem creates a new MonitorItem with default values
func NewMonitorItem(name, url string) *MonitorItem {
	return &MonitorItem{
		Name:   name,
		URL:    url,
		Status: MonitorStatusUnknown,
	}
}

// MonitorStatus represents the status of a monitored item
type MonitorStatus int

// Monitor status constants
const (
	MonitorStatusDown        MonitorStatus = iota // 0
	MonitorStatusUp                               // 1
	MonitorStatusPending                          // 2
	MonitorStatusMaintenance                      // 3
	MonitorStatusUnknown                          // 4
)

// Create returns a string representation of the monitor status
func getCheckmark(status MonitorStatus) string {
	switch status {
	case MonitorStatusUp:
		return "[green]✅[-:-:-]"
	case MonitorStatusDown:
		return "[red]❌[-:-:-]"
	case MonitorStatusPending:
		return "[yellow]⏳[-:-:-]"
	case MonitorStatusMaintenance:
		return "[grey]⚠️[-:-:-]"
	default:
		return "[grey]❓[-:-:-]"
	}
}
