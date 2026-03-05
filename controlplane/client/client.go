package client

import (
	"context"
	"net/netip"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	discoveryv1 "compass/controlplane/gen/discovery/v1"
)

type Client struct {
	cache  atomic.Pointer[map[string][]netip.Addr]
	addr   string
	nodeID string
	quit   chan struct{}
}

func NewClient(addr, nodeID string) *Client {
	cln := &Client{
		addr:   addr,
		nodeID: nodeID,
		quit:   make(chan struct{}),
	}
	cln.cache.Store(&map[string][]netip.Addr{})
	return cln
}

func (c *Client) StartWatcher() error {
	// TODO: make configurable
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 0
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 60 * time.Second
	b.RandomizationFactor = 0.5

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c.quit
		cancel()
	}()

	operation := func() error {
		return c.connectAndWatch(ctx)
	}

	go backoff.Retry(operation, backoff.WithContext(b, ctx))

	return nil
}

func (c *Client) Close() error {
	close(c.quit)
	return nil
}

func (c *Client) connectAndWatch(ctx context.Context) error {
	conn, err := grpc.NewClient(c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	cln := discoveryv1.NewConfigServiceClient(conn)

	stream, err := cln.UpdateConfig(ctx, &discoveryv1.UpdateConfigRequest{NodeId: c.nodeID})
	if err != nil {
		return err
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			return err
		}

		routes := make(map[string][]netip.Addr, len(update.Routes))
		for country, ipList := range update.Routes {
			if !isValidCountryCode(country) {
				continue
			}

			var addrs []netip.Addr
			for _, ip := range ipList.Ips {
				if addr, ok := fromProto(ip); ok {
					addrs = append(addrs, addr)
				}
			}

			if len(addrs) > 0 {
				routes[country] = addrs
			}
		}

		c.cache.Store(&routes)
	}
}

func (c *Client) GetRoutes() map[string][]netip.Addr {
	return *c.cache.Load()
}

func fromProto(p *discoveryv1.IPAddress) (netip.Addr, bool) {
	if p == nil {
		return netip.Addr{}, false
	}
	switch addr := p.Address.(type) {
	case *discoveryv1.IPAddress_V4:
		return netip.AddrFromSlice(addr.V4)
	case *discoveryv1.IPAddress_V6:
		return netip.AddrFromSlice(addr.V6)
	default:
		return netip.Addr{}, false
	}
}

func isValidCountryCode(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
