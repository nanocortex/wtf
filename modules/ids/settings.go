package ids

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = false
	defaultTitle     = "ids"
)

// Settings defines the configuration properties for this module
type Settings struct {
	common *cfg.Common

	subnetPrefix string
	pingTimeout  int // ms
	knownHosts   map[string]string

	// Define your settings attributes here
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	knownHosts, err := ymlConfig.Map("knownHosts")

	kh := make(map[string]string)
	if err != nil {
		kh = map[string]string{}
	} else {
		for k, v := range knownHosts {
			kh[k] = v.(string)
		}
	}

	settings := Settings{
		common:       cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),
		subnetPrefix: ymlConfig.UString("subnetPrefix"),
		pingTimeout:  ymlConfig.UInt("pingTimeoutMs", 200),
		knownHosts:   kh,

		// Configure your settings attributes here. See http://github.com/olebedev/config for type details
	}

	return &settings
}
