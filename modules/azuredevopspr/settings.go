package azuredevopspr

import (
	"os"

	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = true
	defaultTitle     = "azuredevopspr"
)

// Settings defines the configuration options for this module
type Settings struct {
	*cfg.Common

	apiToken        string `help:"Your Azure DevOps Access Token."`
	labelColor      string
	maxRows         int
	organizationUrl string `help:"Your Azure DevOps organization URL."`
	projects        []interface{}
	userUuid        string
}

// NewSettingsFromYAML creates and returns an instance of Settings with configuration options populated
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		apiToken:        ymlConfig.UString("apiToken", os.Getenv("WTF_AZURE_DEVOPS_API_TOKEN")),
		labelColor:      ymlConfig.UString("labelColor", "white"),
		maxRows:         ymlConfig.UInt("maxRows", 3),
		organizationUrl: ymlConfig.UString("organizationUrl", os.Getenv("WTF_AZURE_DEVOPS_ORG_URL")),
		projects:        ymlConfig.UList("projects"),
		userUuid:        ymlConfig.UString("userUuid"),
	}

	cfg.ModuleSecret(name, globalConfig, &settings.apiToken).
		Service(settings.organizationUrl).Load()

	return &settings
}
