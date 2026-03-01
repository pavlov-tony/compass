package iplookup

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

func TestNewIPLookup(t *testing.T) {
	// Test case: non-existent file
	_, err := NewIPLookup("non-existent-file.mmdb")
	if err == nil {
		t.Fatal("Expected an error for a non-existent database file, but got none")
	}
	// The success case is implicitly tested in TestIPLookupMetadata and TestServeDNS.
	// Testing for an invalid DB type would require a specific test file not in the standard MaxMind test set.
}

func TestServeDNS(t *testing.T) {
	dbPath := filepath.Join("testdata", "GeoIP2-Country-Test.mmdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("Skipping ServeDNS test: test database not found at %s", dbPath)
	}

	ipLookup, err := NewIPLookup(dbPath)
	if err != nil {
		t.Fatalf("Failed to create IPLookup instance: %v", err)
	}

	// Set up a next plugin that records it was called.
	called := false
	ipLookup.next = plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		called = true
		return dns.RcodeSuccess, nil
	})

	ctx := context.TODO()
	req := new(dns.Msg)
	req.SetQuestion("example.org.", dns.TypeA)
	w := &test.ResponseWriter{}

	// Call ServeDNS.
	rcode, err := ipLookup.ServeDNS(ctx, w, req)
	if err != nil {
		t.Fatalf("Expected no error, but got %v", err)
	}
	if rcode != dns.RcodeSuccess {
		t.Fatalf("Expected rcode %d, but got %d", dns.RcodeSuccess, rcode)
	}
	if !called {
		t.Fatal("Expected next plugin in the chain to be called, but it was not")
	}
}

func TestIPLookupMetadata(t *testing.T) {
	dbPath := filepath.Join("testdata", "GeoIP2-Country-Test.mmdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("Skipping metadata test: test database not found at %s. Please download it from MaxMind's GitHub.", dbPath)
	}

	ipLookup, err := NewIPLookup(dbPath)
	if err != nil {
		t.Fatalf("Failed to create IPLookup instance: %v", err)
	}
	defer ipLookup.Close()

	tests := []struct {
		name          string
		remoteIP      string
		edns0subnet   *dns.EDNS0_SUBNET
		expectedCode  string
		shouldHaveVal bool
	}{
		{
			name:          "German IPv4 in DB",
			remoteIP:      "141.1.1.1",
			expectedCode:  "DE",
			shouldHaveVal: true,
		},
		{
			name:          "Netherlands IPv6 in DB",
			remoteIP:      "2a02:7b00::",
			expectedCode:  "NL",
			shouldHaveVal: true,
		},
		{
			name:          "Spain IPv4 in DB",
			remoteIP:      "217.65.48.0",
			expectedCode:  "ES",
			shouldHaveVal: true,
		},
		{
			name:          "IP not in DB",
			remoteIP:      "127.0.0.1",
			expectedCode:  "",
			shouldHaveVal: false, // The test DB doesn't contain localhost.
		},
		{
			name:     "EDNS0 Subnet Italian IPv4",
			remoteIP: "1.1.1.1", // This IP will be ignored
			edns0subnet: &dns.EDNS0_SUBNET{
				Code:          dns.EDNS0SUBNET,
				Family:        1, // IPv4
				SourceNetmask: 24,
				Address:       net.ParseIP("151.5.0.0"),
			},
			expectedCode:  "IT",
			shouldHaveVal: true,
		},
		{
			name:     "EDNS0 Subnet French IPv6",
			remoteIP: "1.1.1.1", // This IP will be ignored
			edns0subnet: &dns.EDNS0_SUBNET{
				Code:          dns.EDNS0SUBNET,
				Family:        2, // IPv6
				SourceNetmask: 64,
				Address:       net.ParseIP("2001:41d0::"),
			},
			expectedCode:  "FR",
			shouldHaveVal: true,
		},
		{
			name:          "Invalid IP format in state",
			remoteIP:      "999.999.999.999", // net.ParseIP returns nil
			expectedCode:  "",
			shouldHaveVal: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := metadata.ContextWithMetadata(t.Context())
			req := new(dns.Msg)
			req.SetQuestion("example.org.", dns.TypeA)

			if tc.edns0subnet != nil {
				o := req.IsEdns0()
				if o == nil {
					req.SetEdns0(4096, false)
					o = req.IsEdns0()
				}
				o.Option = append(o.Option, tc.edns0subnet)
			}

			w := &test.ResponseWriter{}
			w.RemoteIP = net.ParseIP(tc.remoteIP).String()
			state := request.Request{W: w, Req: req}

			ctx = ipLookup.Metadata(ctx, state)

			valFunc := metadata.ValueFunc(ctx, pluginName+"/country/code")

			if !tc.shouldHaveVal {
				if valFunc != nil {
					t.Errorf("Expected no metadata value, but got one: %s", valFunc())
				}
				return
			}

			if valFunc == nil {
				t.Fatal("Expected metadata value to be set, but it was not")
			}

			val := valFunc()
			if val != tc.expectedCode {
				t.Errorf("Expected country code '%s', but got '%s'", tc.expectedCode, val)
			}
		})
	}
}

func TestName(t *testing.T) {
	ipLookup := &IPLookup{}
	if ipLookup.Name() != pluginName {
		t.Errorf("Expected Name() to be %s, but got %s", pluginName, ipLookup.Name())
	}
}
