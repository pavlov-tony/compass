package steering

import (
	"context"
	"net/netip"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin(pluginName)

type Steering struct {
	next        plugin.Handler
	watcher     Watcher
	metadataKey string
	fallbackIP  netip.Addr
}

type Watcher interface {
	StartWatcher() error
	Close() error
	GetRoutes() map[string][]netip.Addr
}

func NewSteering(watcher Watcher, metadataKey string, fallbackIP netip.Addr) *Steering {
	return &Steering{
		watcher:     watcher,
		metadataKey: metadataKey,
		fallbackIP:  fallbackIP,
	}
}

// ServeDNS implements the plugin.Handler interface.
func (s *Steering) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// Get the country code from metadata.
	countryFunc := metadata.ValueFunc(ctx, s.metadataKey)
	var country string
	if countryFunc != nil {
		country = countryFunc()
	}

	ips, ok := s.watcher.GetRoutes()[country]
	if !ok || len(ips) == 0 {
		// set default fallback IP in case of lack of routes
		ips = []netip.Addr{s.fallbackIP}
	}

	qtype := r.Question[0].Qtype
	var answers []dns.RR

	// Collect all matching records
	for _, ip := range ips {
		if qtype == dns.TypeA && ip.Is4() {
			answers = append(answers, &dns.A{
				Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   ip.AsSlice(),
			})
		}
	}

	if len(answers) > 0 {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = answers

		return dns.RcodeSuccess, w.WriteMsg(m)
	}

	return plugin.NextOrFailure(pluginName, s.next, ctx, w, r)
}

// Name implements the Handler interface.
func (s *Steering) Name() string { return pluginName }
