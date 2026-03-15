//go:generate go run testdata/generate_mmdb.go -o testdata/GeoIP2-Country-Test.mmdb

package iplookup

import (
	"path/filepath"
	"testing"

	"github.com/coredns/caddy"
	_ "github.com/maxmind/mmdbwriter" // dependencies required for MaxMind DB generating script
	_ "github.com/maxmind/mmdbwriter/mmdbtype"
)

func TestSetup(t *testing.T) {
	dbFile := filepath.Join("testdata", "GeoIP2-Country-Test.mmdb")

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "Missing database path",
			input:     `iplookup`,
			shouldErr: true,
		},
		{
			name:      "Too many arguments",
			input:     `iplookup ` + dbFile + ` extra`,
			shouldErr: true,
		},
		{
			name: "Multiple directives",
			input: `iplookup ` + dbFile + `
					iplookup ` + dbFile,
			shouldErr: true,
		},
		{
			name: "Non-existent file",
			input: `iplookup /tmp/nonexistent.mmdb` + ` {
				default_country_code DE
			}`,
			shouldErr: true,
		},
		{
			name: "Empty default country code",
			input: `iplookup ` + dbFile + ` {
				default_country_code 
			}`,
			shouldErr: true,
		},
		{
			name: "Invalid default country code",
			input: `iplookup ` + dbFile + ` {
				default_country_code Germany
			}`,
			shouldErr: true,
		},
		{
			name: "Valid configuration",
			input: `iplookup ` + dbFile + ` {
				default_country_code DE
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
