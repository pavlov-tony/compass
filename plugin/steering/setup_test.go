package steering

import (
	"path/filepath"
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	configFile := filepath.Join("testdata", "steering_config.yaml")
	invalidConfigFile := filepath.Join("testdata", "invalid_config.yaml")

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "Missing config file",
			input:     `steering`,
			shouldErr: true,
		},
		{
			name:      "Too many arguments",
			input:     `steering ` + configFile + ` extra`,
			shouldErr: true,
		},
		{
			name:      "Multiple config paths",
			input:     `steering ` + configFile + ` ` + configFile,
			shouldErr: true,
		},
		{
			name:      "Non-existent config file",
			input:     `steering /tmp/nonexistent.yaml`,
			shouldErr: true,
		},
		{
			name: "Invalid IP in config file",
			input: `steering ` + invalidConfigFile + ` {
				metadata_key iplookup/country/code
				fallback_ip 1.1.1.1
			}`,
			shouldErr: true,
		},
		{
			name: "Missing metadata key",
			input: `steering ` + configFile + ` {
				fallback_ip 1.1.1.1
			}`,
			shouldErr: true,
		},
		{
			name: "Missing fallback IP",
			input: `steering ` + configFile + ` {
				metadata_key iplookup/country/code
			}`,
			shouldErr: true,
		},
		{
			name: "Invalid fallback IP",
			input: `steering ` + configFile + ` {
				metadata_key iplookup/country/code
				fallback_ip invalid-ip
			}`,
			shouldErr: true,
		},
		{
			name: "Valid configuration",
			input: `steering ` + configFile + ` {
				metadata_key iplookup/country/code
				fallback_ip 1.1.1.1
			}`,
			shouldErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := caddy.NewTestController("dns", test.input)
			err := setup(c)
			if test.shouldErr && err == nil {
				t.Errorf("Expected error but got nil for input: %s", test.input)
			}
			if !test.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v for input: %s", err, test.input)
			}
		})
	}
}
