package steering

import (
	"fmt"
	"net"
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"gopkg.in/yaml.v2"
)

const pluginName = "steering"

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	steering, err := steeringParse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		steering.next = next
		return steering
	})

	return nil
}

func steeringParse(c *caddy.Controller) (*Steering, error) {
	var configPath string
	var metadataKey string
	var fallbackIP net.IP

	for c.Next() {
		if !c.NextArg() {
			return nil, c.ArgErr()
		}
		if configPath != "" {
			return nil, c.Errf("configuring multiple paths is not supported")
		}
		configPath = c.Val()
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
				if fallbackIP = net.ParseIP(c.Val()); fallbackIP == nil {
					return nil, c.Errf("invalid fallback IP: %s", c.Val())
				}
			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	if configPath == "" {
		return nil, c.Err("no configuration file specified")
	}

	if metadataKey == "" {
		return nil, c.Err("no metadata key specified")
	}

	if fallbackIP == nil {
		return nil, c.Err("no fallback IP specified")
	}

	routes, err := loadConfig(configPath)
	if err != nil {
		return nil, c.Errf("load config: %v", err)
	}

	return NewSteering(routes, metadataKey, fallbackIP), nil
}

func loadConfig(path string) (map[string]net.IP, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stringRoutes map[string]string
	if err := yaml.Unmarshal(yamlFile, &stringRoutes); err != nil {
		return nil, err
	}

	routes := make(map[string]net.IP, len(stringRoutes))
	for country, ipStr := range stringRoutes {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip format: %s", ipStr)
		}
		routes[country] = ip
	}

	return routes, nil
}
