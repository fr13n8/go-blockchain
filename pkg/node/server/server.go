package server

import (
	"fmt"
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/miner"
	pb "github.com/fr13n8/go-blockchain/pkg/network/node"
	"github.com/fr13n8/go-blockchain/pkg/services/blockchain"
	services "github.com/fr13n8/go-blockchain/pkg/services/node"
	"github.com/fr13n8/go-blockchain/pkg/services/wallet"
	"google.golang.org/grpc"
	"log"
	"net"
)

type Config struct {
	Port       uint16
	Host       string
	ServerName string
}

type Server struct {
	gRpcServer *grpc.Server
	host       string
	port       uint16
}

func (s *Server) Port() uint16 {
	return s.port
}

func NewServer(cfg *Config) *Server {
	return &Server{
		gRpcServer: grpc.NewServer(),
		host:       cfg.Host,
		port:       cfg.Port,
	}
}

func (s *Server) Run() {
	addr := fmt.Sprintf("%s:%s", s.host, fmt.Sprintf("%d", s.port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	newWallet := wallet.NewWallet()
	bc := blockchain.NewBlockChain(s.Port())
	solver := block.NewSHA256Solver()
	m := miner.NewMiner(solver, bc)
	nodeService := services.NewNodeService(bc, m, newWallet)
	handlers := NewNodeHandler(nodeService)
	pb.RegisterNodeServiceServer(s.gRpcServer, handlers)

	go func() {
		if err := s.gRpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
}

func (s *Server) ShutdownGracefully() {
	s.gRpcServer.GracefulStop()
	log.Println("[NODE] Server successfully stopped")
}
