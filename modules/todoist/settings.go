package todoist

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
	"os"
)

const (
	defaultFocusable = true
	defaultTitle     = "todoist"
)

// Settings defines the configuration properties for this module
type Settings struct {
	common *cfg.Common

	apiKey string
	filter string

	// Define your settings attributes here
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),
		apiKey: ymlConfig.UString("apiKey", ymlConfig.UString("apikey", os.Getenv("TODOIST_API_KEY"))),
		filter: ymlConfig.UString("filter", "today"),
		// Configure your settings attributes here. See http://github.com/olebedev/config for type details
	}

	return &settings
}
