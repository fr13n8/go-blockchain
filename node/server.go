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
}

func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
	}
}

func (s *Server) Addr() net.Addr {
	return s.config.Addr
}

func (s *Server) Run() string {
	s.gRpcServer = grpc.NewServer()
	handlers := NewNodeHandler(s)
	pb.RegisterNodeServiceServer(s.gRpcServer, handlers)

	addr := fmt.Sprintf("%s:%d", s.config.Addr.IP.String(), s.config.Addr.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[NODE] Failed to listen: %v", err)
		return ""
	}

	s.config.Addr = &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: listener.Addr().(*net.TCPAddr).Port,
	}
	log.Println("[NODE] Server started on", listener.Addr().String())
	go func() {
		if err := s.gRpcServer.Serve(listener); err != nil {
			fmt.Printf("failed to serve: %v\n", err)
		}
	}()

	return listener.Addr().String()
}

func (s *Server) ShutdownGracefully() {
	s.gRpcServer.GracefulStop()
	log.Println("[NODE] Server successfully stopped")
}
