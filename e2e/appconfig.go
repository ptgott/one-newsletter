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
	RelayAddress   string
	SSLKey         string
	SSLCert        string
	LinkSourceURLs []string
	StorageDir     string
	PollInterval   string
}

// createAppConfig writes a configuration YAML doc to the given path.
// Use this configuration to start the e2e test environment
func createAppConfig(path string, opts appConfigOptions) error {
	configTemplate := `---
email:
	relayAddress: {{ .RelayAddress }}
	key: {{ .SSLKey }}
	cert: {{ .SSLCert }}
	username: myuser123
	password: myuser123
	fromAddress: mynewsletter@example.com
	toAddress: recipient@example.com
link_sources:
{{ range .LinkSourceURLs }}
	- name: publication-at-{{- . -}}
	url: .
	itemSelector: "ul li"
	captionSelector: "p"
	linkSelector: "a"
{{ end }}
polling:
	interval: {{ .PollInterval }}
storage:
	storageDir: {{ .StorageDir }}
	keyTTL: "1y"
	cleanupInterval: "10m"
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
