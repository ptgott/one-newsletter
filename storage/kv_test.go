package storage

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestKVConfig_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid/canonical case",
			config: `storageDir: ./tempTestDir3012705204
keyTTL: "168h"
cleanupInterval: "10m"`,
			wantErr: false,
		},
		{
			name: "cleanup interval not a duration",
			config: `storageDir: ./tempTestDir3012705204
keyTTL: "168h"
cleanupInterval: "10"`,
			wantErr: true,
		},
		{
			name: "no cleanup interval",
			config: `storageDir: ./tempTestDir3012705204
keyTTL: "168h"`,
			wantErr: true,
		},
		{
			name: "key TTL not a duration",
			config: `storageDir: ./tempTestDir3012705204
keyTTL: "168"
cleanupInterval: "10m"`,
			wantErr: true,
		},
		{
			name: "no key TTL",
			config: `storageDir: ./tempTestDir3012705204
cleanupInterval: "10m"`,
			wantErr: true,
		},
		{
			name: "no storage path",
			config: `keyTTL: "168h"
cleanupInterval: "10m"`,
			wantErr: true,
		},
		{
			name:    "not a JSON object",
			config:  `[]`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer([]byte(tt.config))
			dec := yaml.NewDecoder(buf)
			var c KVConfig
			if err := dec.Decode(&c); (err != nil) != tt.wantErr {
				t.Errorf("wantErr = %v but got %v with err %v", tt.wantErr, err != nil, err)
			}

		})
	}
}
