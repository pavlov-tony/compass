package steering

import (
	"net/netip"
	"strings"

	"compass/controlplane/client"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

const pluginName = "steering"

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	steering, err := steeringParse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	c.OnStartup(func() error {
		return steering.watcher.StartWatcher()
	})
	c.OnShutdown(func() error {
		return steering.watcher.Close()
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		steering.next = next
		return steering
	})

	return nil
}

func steeringParse(c *caddy.Controller) (*Steering, error) {
	var (
		server      string
		nodeID      string
		metadataKey string
		fallbackIP  netip.Addr
	)

	for c.Next() {
		if !c.NextArg() {
			return nil, c.ArgErr()
		}
		if server != "" {
			return nil, c.Errf("configuring multiple servers is not supported")
		}
		server = c.Val()
		strings.TrimPrefix(server, "grpc://")
		// There shouldn't be any more arguments.
		if len(c.RemainingArgs()) != 0 {
			return nil, c.ArgErr()
		}

		for c.NextBlock() {
			switch c.Val() {
			case "metadata_key":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				metadataKey = c.Val()
			case "fallback_ip":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				var err error
				if fallbackIP, err = netip.ParseAddr(c.Val()); err != nil {
					return nil, c.Errf("parse fallback IP: %s", c.Val())
				}
			case "node_id":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				nodeID = c.Val()
			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	if server == "" {
		return nil, c.Err("no server specified")
	}

	if metadataKey == "" {
		return nil, c.Err("no metadata key specified")
	}

	if !fallbackIP.IsValid() {
		return nil, c.Err("no valid fallback IP specified")
	}

	if nodeID == "" {
		return nil, c.Err("no node ID specified")
	}

	watcher := client.NewClient(server, nodeID)
	return NewSteering(watcher, metadataKey, fallbackIP), nil
}
