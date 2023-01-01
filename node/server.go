package node

import (
	"fmt"
	"github.com/fr13n8/go-blockchain/blockchain"
	pb "github.com/fr13n8/go-blockchain/gen/node"
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/network/peer-manager"
	"google.golang.org/grpc"
	"log"
	"net"
)

type Config struct {
	Addr       *net.TCPAddr
	ServerName string

	Bc    *blockchain.BlockChain
	Miner *miner.Miner

	PeerManager *peer_manager.PeerManager
}

func NewConfig() *Config {
	return &Config{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("0.0.0.0"),
			Port: 0,
		},
		ServerName: "go-blockchain",
	}
}

type Server struct {
	gRpcServer *grpc.Server
	config     *Config
	listener   *net.Listener
}

func NewServer(cfg *Config) *Server {
	addr := fmt.Sprintf("%s:%d", cfg.Addr.IP.String(), cfg.Addr.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	return &Server{
		config:   cfg,
		listener: &listener,
	}
}

func (s *Server) Addr() net.Addr {
	return (*s.listener).Addr()
}

func (s *Server) Run() {
	s.gRpcServer = grpc.NewServer()
	handlers := NewNodeHandler(s)
	pb.RegisterNodeServiceServer(s.gRpcServer, handlers)

	log.Println("[NODE] Server started on", (*s.listener).Addr().String())
	go func() {
		if err := s.gRpcServer.Serve(*s.listener); err != nil {
			fmt.Printf("failed to serve: %v\n", err)
		}
	}()
}

func (s *Server) ShutdownGracefully() {
	s.gRpcServer.GracefulStop()
	log.Println("[NODE] Server successfully stopped")
}
