package uptimekuma

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = true
	defaultTitle     = "uptimekuma"
)

// Settings defines the configuration properties for this module
type Settings struct {
	common *cfg.Common

	url      string
	login    string
	password string
	// Define your settings attributes here
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		url:      ymlConfig.UString("url"),
		login:    ymlConfig.UString("login"),
		password: ymlConfig.UString("password"),

		// Configure your settings attributes here. See http://github.com/olebedev/config for type details
	}

	return &settings
}
