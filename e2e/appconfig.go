package e2e

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
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
	KeyTTL            string
}

// mockLinksrcInfo contains metadata about test HTTP servers so we can use it
// to configure scraping targets for the application within a test environment.
//
// Fields are exported so we can use them in templates.
type mockLinksrcInfo struct {
	URL      string
	Name     string
	MaxItems int
}

// createAppConfig writes a configuration YAML doc to the given path.
// Use this configuration to start the e2e test environment
func createAppConfig(path string, opts appConfigOptions) error {
	configTemplate := `---
email:
    smtpServerAddress: {{ .SMTPServerAddress }}
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    type: basic
link_sources:
{{ range .LinkSources }}
    - name: {{ .Name }}
      url: {{ .URL }}
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
      maxItems: {{ .MaxItems }}
{{ end }}
polling:
    interval: {{ .PollInterval }}
storage:
    storageDir: {{ .StorageDir }}
    keyTTL: {{ .KeyTTL }}
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
