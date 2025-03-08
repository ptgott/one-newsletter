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
	if opts.LinkSources == nil || opts.EmailSettings.SMTPServerHost == "" || opts.EmailSettings.SMTPServerPort == "" || opts.Scraping.StorageDirPath == "" {
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

	config.LinkSources = make([]linksrc.Config, len(opts.LinkSources))
	blankURL := url.URL{}
	for i, ls := range opts.LinkSources {
		if ls.URL == blankURL || ls.Name == "" {
			return userconfig.Meta{}, errors.New("each link source must include a URL and Name")
		}
		config.LinkSources[i] = linksrc.Config{
			Name:            ls.Name,
			URL:             ls.URL,
			MaxItems:        uint(ls.MaxItems),
			ItemSelector:    cascadia.MustCompile("ul li"),
			CaptionSelector: cascadia.MustCompile("p"),
			LinkSelector:    cascadia.MustCompile("a"),
		}
		switch {
		case ls.CaptionSelector != nil:
			config.LinkSources[i].CaptionSelector = ls.CaptionSelector
		case ls.ItemSelector != nil:
			config.LinkSources[i].ItemSelector = ls.ItemSelector
		case ls.LinkSelector != nil:
			config.LinkSources[i].LinkSelector = ls.LinkSelector
		}
	}

	return config, nil

}
