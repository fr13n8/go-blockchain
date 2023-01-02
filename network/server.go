package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/fr13n8/go-blockchain/blockchain"
	pb "github.com/fr13n8/go-blockchain/gen/peer"
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/network/discovery"
	gr "github.com/fr13n8/go-blockchain/network/grpc"
	"github.com/fr13n8/go-blockchain/network/peer-manager"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"log"
	"net"
)

type Config struct {
	ServerName string
	DNS        multiaddr.Multiaddr
	Addr       *net.TCPAddr
	ProtocolID string
	Rendezvous string

	Bc          *blockchain.BlockChain
	Miner       *miner.Miner
	PeerManager *peer_manager.PeerManager
}

type Server struct {
	GrpcStream *gr.Stream
	Host       host.Host
	Config     *Config
	CancelFunc context.CancelFunc
}

func NewConfig() *Config {
	return &Config{
		ServerName: "go-blockchain",
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 0,
		},
		ProtocolID: "/go-blockchain/0.0.1",
		Rendezvous: "go-blockchain",
	}
}

func (s *Server) Port() int {
	return s.Config.Addr.Port
}

func NewServer(cfg *Config) *Server {
	sourceMultiAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Addr.IP.String(), cfg.Addr.Port))
	if err != nil {
		log.Println("[NETWORK] Error while creating multiaddr: ", err)
		return nil
	}

	cfg.DNS = sourceMultiAddr

	return &Server{
		Config: cfg,
	}
}

func (s *Server) Run(bootstrapPeers []multiaddr.Multiaddr, peerAddress chan<- []string) {
	r := rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		log.Println("[NETWORK] Error while generating key pair: ", err)
		return
	}

	h, err := libp2p.New(libp2p.ListenAddrs(s.Config.DNS), libp2p.Identity(prvKey))
	if err != nil {
		log.Println("[NETWORK] Error while creating host: ", err)
		return
	}
	s.Host = h
	s.GrpcStream = gr.NewStream()
	ctx, cancel := context.WithCancel(context.Background())
	s.CancelFunc = cancel
	handlers := NewPeerHandler(s.Config.PeerManager)
	pb.RegisterPeerServiceServer(s.GrpcStream, handlers)

	s.Host.SetStreamHandler(protocol.ID(s.Config.ProtocolID), s.GrpcStream.Handler())

	s.GrpcStream.Serve()
	log.Println("[NETWORK] Server started")
	log.Printf("[NETWORK] Peer ID: %s\n", s.Host.ID().String())
	log.Println("[NETWORK] Connect to me on:")
	for _, addr := range s.Host.Addrs() {
		log.Printf("[NETWORK] %s/p2p/%s\n", addr, s.Host.ID().String())
	}

	discoveryService := discovery.NewDiscoveryService(s.Config.PeerManager)
	kademliaDHT, err := discoveryService.NewDHT(ctx, s.Host, bootstrapPeers)
	if err != nil {
		log.Println("[NETWORK] Error while creating DHT: ", err)
		return
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Println("[NETWORK] Error while bootstrapping DHT: ", err)
		return
	}

	go discoveryService.Discover(ctx, s.Host, kademliaDHT, s.Config.Rendezvous, s.GrpcStream, s.Config.ProtocolID, peerAddress)
}

func (s *Server) ShutdownGracefully() {
	err := s.Host.Close()
	if err != nil {
		log.Println("[NETWORK] Error while closing host: ", err)
		return
	}
	s.CancelFunc()
	log.Println("[NETWORK] Server successfully stopped")
}
