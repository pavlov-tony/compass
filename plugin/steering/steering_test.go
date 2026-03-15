package steering

import (
	"context"
	"net/netip"
	"testing"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
)

func TestServeDNS(t *testing.T) {
	mockWatcher := &mockWatcher{
		startWatcherFn: func() error {
			return nil
		},
		closeFn: func() error {
			return nil
		},
		getRoutesFn: func() map[string][]netip.Addr {
			return map[string][]netip.Addr{
				"DE": {netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("1.2.3.5")},
				"ES": {netip.MustParseAddr("4.3.2.1")},
			}
		},
	}

	fallbackIP := netip.MustParseAddr("1.1.1.1")
	metadataKey := "iplookup/country/code"

	s := NewSteering(mockWatcher, metadataKey, fallbackIP)

	// Mock next plugin to verify if it's called
	nextCalled := false
	s.next = plugin.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		nextCalled = true
		return dns.RcodeSuccess, nil
	})

	tests := []struct {
		name           string
		countryCode    string
		qtype          uint16
		expectedIP     string
		shouldAnswer   bool
		shouldCallNext bool
	}{
		{
			name:           "Match DE IPv4",
			countryCode:    "DE",
			qtype:          dns.TypeA,
			expectedIP:     "1.2.3.4",
			shouldAnswer:   true,
			shouldCallNext: false,
		},
		{
			name:           "Mismatch DE IPv4 with AAAA query",
			countryCode:    "DE",
			qtype:          dns.TypeAAAA,
			expectedIP:     "",
			shouldAnswer:   false,
			shouldCallNext: true,
		},
		{
			name:           "Unknown country, use fallback",
			countryCode:    "IT",
			qtype:          dns.TypeA,
			expectedIP:     "1.1.1.1",
			shouldAnswer:   true,
			shouldCallNext: false,
		},
		{
			name:           "No metadata, use fallback",
			countryCode:    "",
			qtype:          dns.TypeA,
			expectedIP:     "1.1.1.1",
			shouldAnswer:   true,
			shouldCallNext: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nextCalled = false
			ctx := context.TODO()
			if tc.countryCode != "" {
				ctx = metadata.ContextWithMetadata(ctx)
				metadata.SetValueFunc(ctx, metadataKey, func() string {
					return tc.countryCode
				})
			}

			req := new(dns.Msg)
			req.SetQuestion("example.org.", tc.qtype)
			rec := dnstest.NewRecorder(&test.ResponseWriter{})

			rcode, err := s.ServeDNS(ctx, rec, req)

			if err != nil {
				t.Fatalf("Expected no error, but got %v", err)
			}

			if tc.shouldAnswer {
				if rcode != dns.RcodeSuccess {
					t.Errorf("Expected rcode %d, but got %d", dns.RcodeSuccess, rcode)
				}
				if rec.Msg == nil || len(rec.Msg.Answer) == 0 {
					t.Fatal("Expected answer, but got none")
				}
				if rec.Msg.Answer[0].(*dns.A).A.String() != tc.expectedIP {
					t.Errorf("Expected IP %s, but got %s", tc.expectedIP, rec.Msg.Answer[0].(*dns.A).A.String())
				}
			}

			if tc.shouldCallNext != nextCalled {
				t.Errorf("Expected next plugin called: %v, but got: %v", tc.shouldCallNext, nextCalled)
			}
		})
	}
}

type mockWatcher struct {
	startWatcherFn func() error
	closeFn        func() error
	getRoutesFn    func() map[string][]netip.Addr
}

func (m *mockWatcher) StartWatcher() error {
	return m.startWatcherFn()
}

func (m *mockWatcher) Close() error {
	return m.closeFn()
}

func (m *mockWatcher) GetRoutes() map[string][]netip.Addr {
	return m.getRoutesFn()
}
