package steering

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "Missing server address",
			input:     `steering`,
			shouldErr: true,
		},
		{
			name:      "Too many arguments",
			input:     `steering localhost:50051 extra`,
			shouldErr: true,
		},
		{
			name: "Missing metadata key",
			input: `steering localhost:50051 {
				fallback_ip 1.1.1.1
				node_id 12345
			}`,
			shouldErr: true,
		},
		{
			name: "Missing fallback IP",
			input: `steering localhost:50051 {
				metadata_key iplookup/country/code
				node_id 12345
			}`,
			shouldErr: true,
		},
		{
			name: "Missing node id",
			input: `steering localhost:50051 {
				metadata_key iplookup/country/code
				fallback_ip 1.1.1.1
			}`,
			shouldErr: true,
		},
		{
			name: "Invalid fallback IP",
			input: `steering localhost:50051 {
				metadata_key iplookup/country/code
				fallback_ip invalid-ip
				node_id 12345
			}`,
			shouldErr: true,
		},
		{
			name: "Valid configuration",
			input: `steering localhost:50051 {
				metadata_key iplookup/country/code
				fallback_ip 1.1.1.1
				node_id 12345
			}`,
			shouldErr: false,
		},
		{
			name: "Valid configuration with grpc scheme",
			input: `steering grpc://localhost:50051 {
				metadata_key iplookup/country/code
				fallback_ip 1.1.1.1
				node_id 12345
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
