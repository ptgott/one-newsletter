package e2e

import (
	"errors"
	"net/url"

	"github.com/andybalholm/cascadia"
	"github.com/ptgott/one-newsletter/email"
	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/userconfig"
)

// createUserConfig creates a user configuration based on the provided
// starter config. Non-required options are populated automatically using
// defaults intended for e2e testing.
func createUserConfig(opts userconfig.Meta) (userconfig.Meta, error) {
	if opts.Newsletters == nil || len(opts.Newsletters) == 0 || opts.EmailSettings.SMTPServerHost == "" || opts.EmailSettings.SMTPServerPort == "" || opts.Scraping.StorageDirPath == "" {
		return userconfig.Meta{}, errors.New("must supply all required fields in appConfigOptions")
	}

	config := userconfig.Meta{
		EmailSettings: email.UserConfig{
			SMTPServerHost:       opts.EmailSettings.SMTPServerHost,
			SMTPServerPort:       opts.EmailSettings.SMTPServerPort,
			FromAddress:          "mynewsletter@example.com",
			ToAddress:            "recipient@example.com",
			UserName:             "myuser",
			Password:             "password123",
			SkipCertVerification: true,
		},
		Scraping: userconfig.Scraping{
			StorageDirPath: opts.Scraping.StorageDirPath,
			OneOff:         opts.Scraping.OneOff,
			TestMode:       opts.Scraping.TestMode,
			LinkExpiryDays: 180,
		},
	}

	newsletters := make(map[string]userconfig.Newsletter)

	for k, n := range opts.Newsletters {
		sources := make([]linksrc.Config, len(n.LinkSources))
		blankURL := url.URL{}
		for i, ls := range n.LinkSources {
			if ls.URL == blankURL || ls.Name == "" {
				return userconfig.Meta{}, errors.New("each link source must include a URL and Name")
			}
			sources[i] = linksrc.Config{
				Name:            ls.Name,
				URL:             ls.URL,
				MaxItems:        uint(ls.MaxItems),
				ItemSelector:    cascadia.MustCompile("ul li"),
				CaptionSelector: cascadia.MustCompile("p"),
				LinkSelector:    cascadia.MustCompile("a"),
			}
			switch {
			case ls.CaptionSelector != nil:
				sources[i].CaptionSelector = ls.CaptionSelector
			case ls.ItemSelector != nil:
				sources[i].ItemSelector = ls.ItemSelector
			case ls.LinkSelector != nil:
				sources[i].LinkSelector = ls.LinkSelector
			}
		}
		newsletters[k] = userconfig.Newsletter{
			Schedule:    n.Schedule,
			LinkSources: sources,
		}
	}
	config.Newsletters = newsletters

	return config, nil
}
