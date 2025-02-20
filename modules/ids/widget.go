package ids

import (
	"bytes"
	"context"
	"fmt"
	"github.com/rivo/tview"
	"github.com/wtfutil/wtf/view"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Widget is the container for your module's data
type Widget struct {
	view.TextWidget

	settings *Settings
	scanner  *Scanner
	err      error
}

type Scanner struct {
	counter    atomic.Int32
	isScanning atomic.Bool
	Items      []ScanItem
	mu         sync.Mutex
	settings   Settings
}

type ScanItem struct {
	Address       net.IP
	Hostname      string
	RoundtripTime string
}

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, pages, settings.common),
		settings:   settings,
		scanner:    NewScanner(*settings),
	}

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}

	if widget.settings.subnetPrefix == "" {
		widget.err = fmt.Errorf("subnetPrefix is required")
		widget.display()
		return
	}

	widget.scanner.Items = make([]ScanItem, 0)

	ctx := context.Background()

	err := widget.scanner.ScanNetwork(ctx, widget)

	if err != nil {
		widget.err = err
	}

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) content() string {
	if widget.err != nil {
		return fmt.Sprintf("[red]Error: %s[/]", widget.err)
	}
	str := ""
	if widget.scanner.isScanning.Load() {
		str += "Scanning... " + fmt.Sprintf("%d/254\n", widget.scanner.counter.Load())
	}

	// Create a copy of Items to avoid modifying the original slice
	items := make([]ScanItem, len(widget.scanner.Items))
	copy(items, widget.scanner.Items)

	// Sort items by IP address
	sort.Slice(items, func(i, j int) bool {
		return bytes.Compare(items[i].Address, items[j].Address) < 0
	})

	for _, item := range items {
		str += fmt.Sprintf("%-13s  %-25s %s\n", item.Address, item.Hostname, item.RoundtripTime)
	}

	return str
}

func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), false
	})
}

func NewScanner(settings Settings) *Scanner {
	return &Scanner{
		settings: settings,
		Items:    make([]ScanItem, 0),
	}
}

func (s *Scanner) ScanNetwork(ctx context.Context, widget *Widget) error {
	s.counter.Store(0)
	s.isScanning.Store(true)
	widget.display()
	defer s.isScanning.Store(false)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 16) // MaxDegreeOfParallelism

	for i := 1; i <= 254; i++ {

		widget.QuitChan()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			select {
			case <-widget.QuitChan():
				return
			default:
				s.scanAddress(i)
				s.counter.Add(1)
				widget.display()
			}

			//s.scanAddress(i)
			//s.counter.Add(1)
			//widget.display()
		}(i)
	}

	wg.Wait()
	return nil
}

func (s *Scanner) scanAddress(i int) {
	ipAddr := fmt.Sprintf("%s.%d", s.settings.subnetPrefix, i)

	// Using ping command instead of Go's net.Ping as it requires root privileges
	cmd := exec.Command("ping", "-c", "1", "-W", fmt.Sprintf("%d", s.settings.pingTimeout), ipAddr)
	if stdoutb, err := cmd.Output(); err == nil {
		stdout := string(stdoutb)
		rtRegex := regexp.MustCompile(`time=([0-9.]+)`)
		matches := rtRegex.FindStringSubmatch(stdout)

		ip := net.ParseIP(ipAddr)
		hostname, _ := s.getDeviceName(ip)

		rdt := ""
		if matches != nil && len(matches) > 1 {
			rdt = fmt.Sprintf("[grey]%sms[-:-:-]", matches[1])
		}

		s.mu.Lock()
		s.Items = append(s.Items, ScanItem{
			Address:       ip,
			Hostname:      hostname,
			RoundtripTime: rdt,
		})
		s.mu.Unlock()
	}
}

func (s *Scanner) getDeviceName(ip net.IP) (string, error) {
	ipStr := ip.String()

	// Check known hosts by IP
	if hostname, ok := s.settings.knownHosts[ipStr]; ok {
		return hostname, nil
	}

	// Get MAC address
	macAddr := getMacByIp(ipStr)
	if macAddr != "" {
		// Check known hosts by MAC address (various cases)
		macLower := strings.ToLower(macAddr)
		macUpper := strings.ToUpper(macAddr)

		if hostname, ok := s.settings.knownHosts[macLower]; ok {
			return hostname, nil
		}
		if hostname, ok := s.settings.knownHosts[macUpper]; ok {
			return hostname, nil
		}
		if hostname, ok := s.settings.knownHosts[macAddr]; ok {
			return hostname, nil
		}
	}

	// Try DNS lookup
	names, err := net.LookupAddr(ipStr)
	if err == nil && names != nil && len(names) > 0 {
		if hostname, ok := s.settings.knownHosts[names[0]]; ok {
			return hostname, nil
		}
		return names[0], nil
	}

	return fmt.Sprintf("[red]%s[-:-:-]", macAddr), nil
}

func getMacByIp(ipAddress string) string {
	cmd := exec.Command("arp", "-n", ipAddress)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`([0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2})`)
	match := re.FindString(string(output))
	return strings.ToLower(strings.TrimSpace(match))
}
