package iplookup

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const pluginName = "iplookup"

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	ipLookup, err := iplookupParse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	// StartWatcher will return an error on initial setup failure,
	// but will run the event loop in a goroutine on success.
	c.OnStartup(ipLookup.StartWatcher)
	c.OnShutdown(ipLookup.Close)

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		ipLookup.next = next
		return ipLookup
	})

	return nil
}

func iplookupParse(c *caddy.Controller) (*IPLookup, error) {
	var dbPath string

	for c.Next() {
		if !c.NextArg() {
			return nil, c.ArgErr()
		}
		if dbPath != "" {
			return nil, c.Errf("configuring multiple databases is not supported")
		}
		dbPath = c.Val()
		// There shouldn't be any more arguments.
		if len(c.RemainingArgs()) != 0 {
			return nil, c.ArgErr()
		}
	}

	ipLookup, err := newIPLookup(dbPath)
	if err != nil {
		return nil, c.Errf("initialize iplookup plugin: %v", err)
	}

	return ipLookup, nil
}
