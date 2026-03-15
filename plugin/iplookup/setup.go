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
	var (
		dbPath             string
		defaultCountryCode string
	)

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

		for c.NextBlock() {
			switch c.Val() {
			case "default_country_code":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				defaultCountryCode = c.Val()
			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	if dbPath == "" {
		return nil, c.Err("no db file specified")
	}

	if defaultCountryCode == "" {
		return nil, c.Err("no default country code specified")
	}
	if len(defaultCountryCode) != 2 {
		return nil, c.Errf("invalid default country code: %s", defaultCountryCode)
	}
	for _, r := range defaultCountryCode {
		if r < 'A' || r > 'Z' {
			return nil, c.Errf("invalid default country code: %s", defaultCountryCode)
		}
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, c.Errf("open database: %v", err)
	}

	return NewIPLookup(dbPath, db, defaultCountryCode), nil
}
