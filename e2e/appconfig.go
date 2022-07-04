package e2e

import (
	"net/url"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/ptgott/one-newsletter/email"
	"github.com/ptgott/one-newsletter/linksrc"
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
type mockLinksrcInfo struct {
	// Required
	URL string
	// Required
	Name string
	// Required
	MaxItems int
	// Not required
	LinkSelector string
	// Not required
	CaptionSelector string
	// Not required
	ItemSelector string
}

// createAppConfig creates a user configuration based on the provided
// appConfigOptions. Only options within appConfigOptions are required. The
// are populated automatically using defaults intended for e2e testing.
func createAppConfig(path string, opts appConfigOptions) (userconfig.Meta, error) {
	v, err := time.ParseDuration(opts.PollInterval)
	if err != nil {
		return userconfig.Meta{}, err
	}
	config := userconfig.Meta{
		EmailSettings: email.UserConfig{
			SMTPServerHost:       opts.SMTPServerAddress,
			FromAddress:          "mynewsletter@example.com",
			ToAddress:            "recipient@example.com",
			UserName:             "myuser",
			Password:             "password123",
			SkipCertVerification: true,
		},
		Scraping: userconfig.Scraping{
			Interval:       v,
			StorageDirPath: opts.StorageDir,
		},
	}

	config.LinkSources = make([]linksrc.Config, len(opts.LinkSources))
	for i, ls := range opts.LinkSources {
		u, err := url.Parse(ls.URL)
		if err != nil {
			return userconfig.Meta{}, err
		}
		config.LinkSources[i] = linksrc.Config{
			Name:            ls.Name,
			URL:             *u,
			MaxItems:        uint(ls.MaxItems),
			ItemSelector:    cascadia.MustCompile("ul li"),
			CaptionSelector: cascadia.MustCompile("p"),
			LinkSelector:    cascadia.MustCompile("a"),
		}
		switch {
		case ls.CaptionSelector != "":
			config.LinkSources[i].CaptionSelector = cascadia.MustCompile(ls.CaptionSelector)
		case ls.ItemSelector != "":
			config.LinkSources[i].ItemSelector = cascadia.MustCompile(ls.ItemSelector)
		case ls.LinkSelector != "":
			config.LinkSources[i].LinkSelector = cascadia.MustCompile(ls.LinkSelector)
		}
	}

	return config, nil

}
