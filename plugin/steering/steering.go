package steering

import (
	"context"
	"net"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin(pluginName)

type Steering struct {
	next        plugin.Handler
	routes      map[string]net.IP
	metadataKey string
	fallbackIP  net.IP
}

func NewSteering(routes map[string]net.IP, metadataKey string, fallbackIP net.IP) *Steering {
	return &Steering{
		routes:      routes,
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

	ip, ok := s.routes[country]
	if !ok {
		// set default fallback IP in case of lack of routes
		ip = s.fallbackIP
	}

	qtype := r.Question[0].Qtype
	// Check if the query type matches the IPv4 version
	if qtype == dns.TypeA && ip.To4() != nil {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true

		rr := &dns.A{
			Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   ip,
		}
		m.Answer = []dns.RR{rr}

		if err := w.WriteMsg(m); err != nil {
			return dns.RcodeServerFailure, err
		}
		return dns.RcodeSuccess, nil
	}

	return plugin.NextOrFailure(pluginName, s.next, ctx, w, r)
}

// Name implements the Handler interface.
func (s *Steering) Name() string { return pluginName }
