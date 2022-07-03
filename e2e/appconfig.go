package e2e

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/ptgott/one-newsletter/userconfig"
)

// appConfigOptions is used to fill in a config template with details unique to
// a specific test environment. Keep this as small as possible so the input
// remains as close to a "real" YAML document as we can make it. Also using
// YAML/JSON-compatible types only here.
//
// Fields are exported so we can use them in templates.
type appConfigOptions struct {
	SMTPServerAddress string
	LinkSources       []mockLinksrcInfo
	StorageDir        string
	PollInterval      string
}

// mockLinksrcInfo contains metadata about test HTTP servers so we can use it
// to configure scraping targets for the application within a test environment.
//
// Fields are exported so we can use them in templates.
type mockLinksrcInfo struct {
	URL      string
	Name     string
	MaxItems int
	// The linkSelector, captionSelector, and itemSelector in a link source
	// config. Leave blank if you would like to use valid defaults.
	SelectorsOverride string
}

// createAppConfig creates a user configuration based on the provided
// appConfigOptions. Only options within appConfigOptions are required. The
// are populated automatically using defaults intended for e2e testing.
func createAppConfig(path string, opts appConfigOptions) userconfig.Meta {
	config := userconfig.Meta{
	    EmailSettings: email.UserConfig{
		SMTPServerHost: opts.SMTPServerAddress,
		FromAddress: "mynewsletter@example.com",
		ToAddress: "recipient@example.com",
		UserName: "myuser",
		Password: "password123",
		SkipCertVerification: true,
	    },
	Scraping: userconfig.Scraping{
	    Interval: opts.PollInterval,
	    StorageDirPath: opts.StorageDir,
	},
    }

    config.LinkSources = make([]linksrc.Config, len(opts.LinkSources))
    for i, ls := range opts.LinkSources{
config.LinkSources[i] = linksrc.Config{
    Name: ls.Name,
    URL: ls.URL
    MaxItems: ls.MaxItems,
}

// TODO: Assign SelectorsOverride/default selectors.
if !opts.SelectorsOverride{
    config.LinkSources[i].itemSelector = "ul li"
    config.LinkSources[i].linkSelector = "p" 
config.LinkSources[i].captionSelector = "a"
}
    }
	    

	// This means the config template string was written incorrectly. Not
	// an issue with the application itself.
	if err != nil {
		return fmt.Errorf("couldn't parse the application config template: %v", err)
	}

	var config bytes.Buffer

	err = tmpl.Execute(&config, opts)

	// This is an issue with the test environment, not the application
	if err != nil {
		return fmt.Errorf("couldn't populate the application config template: %v", err)
	}

	cf, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("couldn't create the config file: %v", err)
	}

	_, err = cf.Write(config.Bytes())
	if err != nil {
		return fmt.Errorf("couldn't write to the config file: %v", err)
	}

	return nil

}
