package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"strings"
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
	// Required. Includes host and port.
	SMTPServerAddress string
	// Required
	LinkSources []mockLinksrcInfo
	// Required
	StorageDir string
	// Required
	PollInterval string
}

// mockLinksrcInfo contains metadata about test HTTP servers so we can use it
// to configure scraping targets for the application within a test environment.
type mockLinksrcInfo struct {
	// Required
	URL string
	// Required
	Name string
	// Not required
	MaxItems int
	// Not required
	LinkSelector string
	// Not required
	CaptionSelector string
	// Not required
	ItemSelector string
	// The linkSelector, captionSelector, and itemSelector in a link source
	// config. Leave blank if you would like to use valid defaults.
	SelectorsOverride string
}

// createUserConfig creates a user configuration based on the provided
// appConfigOptions. Non-required options are populated automatically using
// defaults intended for e2e testing.
func createUserConfig(opts appConfigOptions) (userconfig.Meta, error) {
	if opts.LinkSources == nil || opts.SMTPServerAddress == "" || opts.PollInterval == "" || opts.StorageDir == "" {
		return userconfig.Meta{}, errors.New("must supply all required fields in appConfigOptions")
	}
	v, err := time.ParseDuration(opts.PollInterval)
	if err != nil {
		return userconfig.Meta{}, err
	}

	hp := strings.Split(opts.SMTPServerAddress, ":")

	if len(hp) != 2 {
		return userconfig.Meta{}, errors.New("SMTPServerAddress must be in the form host:port")
	}

	config := userconfig.Meta{
		EmailSettings: email.UserConfig{
			SMTPServerHost:       hp[0],
			SMTPServerPort:       hp[1],
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
		if ls.URL == "" || ls.Name == "" {
			return userconfig.Meta{}, errors.New("each mockLinksrcInfo must include a URL and Name")
		}
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

// createAppConfig writes a configuration YAML doc to the given path.
// Use this configuration to start the e2e test environment
func createAppConfig(path string, opts appConfigOptions) error {
	configTemplate := `---
email:
    smtpServerAddress: {{ .SMTPServerAddress }}
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: myuser
    password: password123
    skipCertVerification: true
link_sources:
{{ range .LinkSources }}
    - name: {{ .Name }}
      url: {{ .URL }}
	  {{- if ne .SelectorsOverride "" }}	  
{{ .SelectorsOverride }}
{{ else }}
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
{{ end }}
      maxItems: {{ .MaxItems }}
{{ end }}
scraping:
    interval: {{ .PollInterval }}
    storageDir: {{ .StorageDir }}
`

	tmpl, err := template.New("conf").Parse(configTemplate)

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
