package nextdns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/view"
)

// Widget is the container for your module's data
type Widget struct {
	view.TextWidget

	settings    *Settings
	nextDnsInfo *NextDnsInfo
	err         error
}

type NextDnsInfo struct {
	Status     string `json:"status,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Profile    string `json:"profile,omitempty"`
	Client     string `json:"client,omitempty"`
	SrcIp      string `json:"srcIP,omitempty"`
	DestIp     string `json:"destIP,omitempty"`
	Anycast    bool   `json:"anycast"`
	Server     string `json:"server,omitempty"`
	ClientName string `json:"clientName,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	DeviceId   string `json:"deviceID,omitempty"`
}

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, pages, settings.common),

		settings: settings,
	}

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	widget.loadNextDnsInfo()

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) loadNextDnsInfo() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	result, err := client.Get("https://test.nextdns.io")
	if err != nil {
		widget.err = err
		return
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		widget.err = err
		return
	}

	// URL regex matching
	urlRegex := regexp.MustCompile(`http(s)?://([\w-]+\.)+[\w-]+(/[\w- ./?%&=]*)?`)
	extractedUrl := urlRegex.FindString(string(body))
	if extractedUrl == "" {
		widget.err = fmt.Errorf("no URL found in response")
		return
	}

	// Second HTTP request
	jsonResp, err := client.Get(extractedUrl)
	if err != nil {
		widget.err = fmt.Errorf("failed to get JSON response: %w", err)
	}
	defer jsonResp.Body.Close()

	// Parse JSON response
	var response NextDnsInfo
	if err := json.NewDecoder(jsonResp.Body).Decode(&response); err != nil {
		widget.err = fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Process protocol

	widget.nextDnsInfo = &response
}

func (widget *Widget) content() string {
	if widget.err != nil {
		return widget.err.Error()
	}

	protocol := ""
	if strings.ToLower(widget.nextDnsInfo.Protocol) == "doh" {
		protocol = "✅[green]DOH[-:-:-]"
	}

	str := ""

	str += widget.LabelValue("Status", widget.nextDnsInfo.Status+" "+protocol)
	str += widget.LabelValue("IP", widget.nextDnsInfo.Client)
	str += widget.LabelValue("Name", widget.nextDnsInfo.ClientName)
	str += widget.LabelValue("Device", widget.nextDnsInfo.DeviceName)
	str += widget.LabelValue("DeviceId", widget.nextDnsInfo.DeviceId)

	return str

}

func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), false
	})
}
