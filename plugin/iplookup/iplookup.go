package iplookup

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/oschwald/maxminddb-golang/v2"
)

var log = clog.NewWithPlugin(pluginName)

type IPLookup struct {
	next   plugin.Handler
	db     atomic.Pointer[maxminddb.Reader]
	dbPath string
	quit   chan struct{}
}

// OpenDB opens the MaxMind database and verifies it has the correct type.
func OpenDB(dbPath string) (*maxminddb.Reader, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file: %v", err)
	}

	switch dbType := db.Metadata.DatabaseType; dbType {
	case "City", "Country":
	default:
		db.Close()
		return nil, fmt.Errorf("database type %s does not provide country geographic data", dbType)
	}
	return db, nil
}

func NewIPLookup(dbPath string, db *maxminddb.Reader) *IPLookup {
	ipLookup := &IPLookup{dbPath: dbPath, quit: make(chan struct{})}
	ipLookup.db.Store(db)

	return ipLookup
}

// ServeDNS implements the plugin.Handler interface.
func (l *IPLookup) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(pluginName, l.next, ctx, w, r)
}

// Metadata implements the metadata.Provider Interface in the metadata plugin, and is used to store
// the data associated with the source IP of every request.
func (l *IPLookup) Metadata(ctx context.Context, state request.Request) context.Context {
	srcIP := net.ParseIP(state.IP())

	if o := state.Req.IsEdns0(); o != nil {
		for _, s := range o.Option {
			if e, ok := s.(*dns.EDNS0_SUBNET); ok {
				// TODO: apply mask if possible
				srcIP = e.Address
				break
			}
		}
	}

	record := struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}{}

	ip, ok := netip.AddrFromSlice(srcIP)
	if !ok {
		log.Debugf("Invalid ip format: %v", srcIP)
		return ctx
	}

	err := l.db.Load().Lookup(ip).Decode(&record)
	if err != nil {
		log.Debugf("Setting up metadata failed due to database lookup error: %v", err)
		return ctx
	}

	if record.Country.ISOCode != "" {
		metadata.SetValueFunc(ctx, pluginName+"/country/code", func() string {
			return record.Country.ISOCode
		})
	}

	return ctx
}

// Name implements the Handler interface.
func (l *IPLookup) Name() string { return pluginName }
