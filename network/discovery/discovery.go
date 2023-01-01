package discovery

import (
	"context"
	"fmt"
	pb "github.com/fr13n8/go-blockchain/gen/peer"
	"github.com/fr13n8/go-blockchain/network/grpc"
	peer_manager "github.com/fr13n8/go-blockchain/network/peer-manager"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"log"
	"strings"
	"sync"
	"time"

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

type addrList []multiaddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := multiaddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
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

func (ds *Service) NewDHT(ctx context.Context, h host.Host, bootstrapPeers []multiaddr.Multiaddr, gr *grpc.Stream, protocolId string, peerAddress chan<- string) (*dht.IpfsDHT, error) {
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
				fmt.Printf("Error while connecting to node %q: %-v\n", peerInfo, err)
			} else {
				fmt.Printf("Connection established with bootstrap node: %s\n", peerInfo.ID.Pretty())
				stream, err := h.NewStream(ctx, peerInfo.ID, protocol.ID(protocolId))
				if err != nil {
					fmt.Println("Error while creating stream: ", err)
				}
				peerAddress <- peerInfo.Addrs[0].String()
				client := gr.Client(stream)
				// Send message to bootstrap node
				cn := pb.NewPeerServiceClient(client)
				pmc, err := cn.Message(ctx)
				if err != nil {
					fmt.Println("Error", err)
					return
				}
				err = pmc.Send(&pb.MessageBody{
					Type: "get_peers",
					Data: []byte("Hello"),
				})
				if err != nil {
					fmt.Println("Error", err)
					return
				}
				go func() {
					for {
						msg, err := pmc.Recv()
						if err != nil {
							fmt.Println("Error", err)
							return
						}
						fmt.Println("Message", msg)
					}
				}()
			}
		}()
	}
	wg.Wait()

	return kdht, nil
}

func (ds *Service) Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string, gr *grpc.Stream, protocolId string, peerAddress chan<- string) {
	var routingDiscovery = drouting.NewRoutingDiscovery(dht)

	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			myPeers := h.Peerstore().PeersWithAddrs()
			log.Println("My peers")
			for _, p := range myPeers {
				log.Printf("Found peer: %s\n", p.Pretty())
			}
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)
			if err != nil {
				fmt.Println("Error finding peers")
				panic(err)
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) != network.Connected {
					stream, err := h.NewStream(ctx, p.ID, protocol.ID(protocolId))
					if err != nil {
						fmt.Println("Connection failed:", err)
						continue
					} else {
						client := gr.Client(stream)
						fmt.Println("Connected to:", p.ID)
						fmt.Println("Stream open success")
						peerAddress <- p.Addrs[0].String()
						cn := pb.NewPeerServiceClient(client)
						pmc, err := cn.Message(ctx)
						if err != nil {
							fmt.Println("Error", err)
							continue
						}
						err = pmc.Send(&pb.MessageBody{
							Type: "get_peers",
							Data: []byte("Hello"),
						})
						if err != nil {
							fmt.Println("Error", err)
							return
						}
						go func() {
							for {
								msg, err := pmc.Recv()
								if err != nil {
									fmt.Println("Error", err)
									return
								}
								fmt.Println("Message", msg)
							}
						}()
					}
				}
			}
		}
	}
}
