package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rivo/tview"
	log "github.com/wtfutil/wtf/logger"
	"github.com/wtfutil/wtf/view"
)

// Widget is the container for your module's data
type Widget struct {
	view.TextWidget

	settings    *Settings
	networkInfo *NetworkInfo
	err         error
}

type NetworkInfo struct {
	If        string
	IpAddr    string
	DnsServer string
	MacAddr   string
	IpInfo    *IpInfo
}

type IpInfo struct {
	Ip           string `json:"ip"`
	Hostname     string `json:"hostname"`
	City         string `json:"city"`
	Region       string `json:"region"`
	Country      string `json:"country"`
	Coordinates  string `json:"loc"`
	PostalCode   string `json:"postal"`
	Organization string `json:"org"`
}

type protocolVersion string

// NewWidget creates and returns an instance of Widget
func NewWidget(tviewApp *tview.Application, redrawChan chan bool, pages *tview.Pages, settings *Settings) *Widget {
	widget := Widget{
		TextWidget: view.NewTextWidget(tviewApp, redrawChan, pages, settings.common),

		settings:    settings,
		networkInfo: &NetworkInfo{},
	}

	return &widget
}

/* -------------------- Exported Functions -------------------- */

// Refresh updates the onscreen contents of the widget
func (widget *Widget) Refresh() {
	if widget.Disabled() {
		return
	}
	widget.networkInfo, widget.err = widget.getNetworkInfo()

	// The last call should always be to the display function
	widget.display()
}

/* -------------------- Unexported Functions -------------------- */

func (widget *Widget) getNetworkInfo() (*NetworkInfo, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	mainInterface := ""
	mainIpAddr := ""
	macAddr := ""

	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) != 0 && (iface.Flags&net.FlagLoopback) == 0 && (iface.Flags&net.FlagPointToPoint) == 0 {

			addrs, err := iface.Addrs()
			if err != nil {
				err = &net.OpError{Op: "route", Net: "ip+net", Source: nil, Addr: nil, Err: err}
				continue
			}

			for _, addr := range addrs {
				if strings.Contains(addr.String(), "192.168") {
					mainInterface = iface.Name
					mainIpAddr = addr.String()
					macAddr = iface.HardwareAddr.String()
				}

			}
		}
	}

	dnsServer, err := readDnsServer()
	if err != nil {
		return nil, err
	}

	ipInfo, err := widget.ipinfo()
	if err != nil {
		return nil, err
	}

	return &NetworkInfo{
		If:        mainInterface,
		IpAddr:    mainIpAddr,
		DnsServer: dnsServer,
		IpInfo:    ipInfo,
		MacAddr:   macAddr,
	}, nil
}

func (widget *Widget) content() string {
	if widget.err != nil {
		return widget.err.Error()
	}

	str := widget.LabelValue("IF", widget.networkInfo.If)
	str += widget.LabelValue("LAN IP", widget.networkInfo.IpAddr)
	str += widget.LabelValue("WAN IP", widget.networkInfo.IpInfo.Ip)
	str += widget.LabelValue("MAC", widget.networkInfo.MacAddr)
	str += widget.LabelValue("DNS", widget.networkInfo.DnsServer)
	// str += widget.formatableText("ORG", widget.networkInfo.IpInfo.Organization)

	return str
}

func (widget *Widget) display() {
	widget.Redraw(func() (string, string, bool) {
		return widget.CommonSettings().Title, widget.content(), false
	})
}

func readDnsServer() (addr string, err error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	d := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				d += parts[1] + ", "
			}
		}
	}

	d = strings.TrimSuffix(d, ", ")

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return d, nil
}

// func (widget *Widget) formatableText(key, value string) string {
// 	return fmt.Sprintf("[%s]%8s[-:-:-] %s\n", widget.settings.common.Colors.Subheading, key, value)
// }

// this method reads the config and calls ipinfo for ip information
func (widget *Widget) ipinfo() (*IpInfo, error) {
	client := &http.Client{}
	var url string
	ip, ipv6, err := getMyIP("v4")
	// err
	if err != nil {
		return nil, err
	}
	if ipv6 {
		url = fmt.Sprintf("https://ipinfo.io/%s", ip.String())
	} else {
		url = "https://ipinfo.io/"
	}

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl")
	// if widget.settings.apiToken != "" {
	// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", widget.settings.apiToken))
	// }

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	var info IpInfo
	err = json.NewDecoder(response.Body).Decode(&info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// getMyIP provides this system's default IPv4 or IPv6 IP address for routing WAN requests.
// It does so by dialing out to a site known to have both an A and AAAA DNS records (IPv6)
// The 'net' package is allowed to decide how to connect, connecting to both IPv4 or IPv6 address
// depending on the availbility of IP protocols.
func getMyIP(version protocolVersion) (ip net.IP, v6 bool, err error) {
	log.Log(fmt.Sprintf("Protocol version: %s", version))
	log.Log(fmt.Sprintf("Network: %s", version.toNetwork()))
	//fmt.Println("Protocol version: ", version)
	conn, err := net.DialTimeout(version.toNetwork(), "fast.com:80", 500*time.Millisecond)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	addr := conn.LocalAddr().(*net.TCPAddr)
	ip = addr.IP
	v6 = ip.To4() == nil

	return
}

func (pv protocolVersion) toNetwork() string {
	switch pv {
	case "v4":
		return "tcp4"
	case "v6":
		return "tcp6"
	default:
		return "tcp"
	}
}
