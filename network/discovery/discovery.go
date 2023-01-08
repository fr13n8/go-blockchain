package discovery

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/fr13n8/go-blockchain/network/grpc"
	peer_manager "github.com/fr13n8/go-blockchain/network/peer-manager"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type Service struct {
	pm *peer_manager.PeerManager
}

func NewDiscoveryService(pm *peer_manager.PeerManager) *Service {
	return &Service{
		pm: pm,
	}
}

func StringsToAddrs(addrStrings []string) (maddrs []multiaddr.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := multiaddr.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

func (ds *Service) NewDHT(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	kdht, err := dht.New(ctx, h, options...)
	if err != nil {
		return nil, err
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerInfo); err != nil {
				log.Printf("Error while connecting to node %q: %-v\n", peerInfo, err)
				return
			}
			log.Printf("Connection established with bootstrap node: %s\n", peerInfo.ID.String())
		}()
	}
	wg.Wait()

	return kdht, nil
}

func (ds *Service) Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string, gr *grpc.Stream, protocolId string, peerAddress chan<- []string) {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)

	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				log.Printf("Error finding peers: %v\n", err)
				continue
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}

				myPeers := make([]string, 0, len(h.Network().Peers()))
				for _, p := range h.Network().Peers() {
					addrs := h.Network().Peerstore().PeerInfo(p).Addrs
					myPeers = append(myPeers, addrs[0].String()+" <=> "+addrs[1].String())
				}
				peerAddress <- myPeers

				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err := h.NewStream(ctx, p.ID, protocol.ID(protocolId))
					if err != nil {
						log.Println("Connection failed:", err)
						continue
					}
				}
			}
		}
	}
}
