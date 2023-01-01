package block_explorer

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	pb "github.com/fr13n8/go-blockchain/gen/node"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"
)

//go:embed public
var frontend embed.FS

type Config struct {
	ServerName string
	Gateway    string
	Addr       *net.TCPAddr
}

func NewConfig(gateway string) *Config {
	return &Config{
		Gateway:    gateway,
		ServerName: "go-blockchain",
	}
}

type Server struct {
	app    *fiber.App
	nc     pb.NodeServiceClient
	config *Config
}

func NewServer(cfg *Config) *Server {
	conn, err := grpc.Dial(cfg.Gateway, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[BLOCK-EXPLORER] Error while connecting to gateway: %s", err.Error())
	}
	client := pb.NewNodeServiceClient(conn)
	return &Server{
		app: fiber.New(
			fiber.Config{
				AppName:               cfg.ServerName,
				DisableStartupMessage: true,
			}),
		config: cfg,
		nc:     client,
	}
}

func (s *Server) Run(port int) {
	s.config.Addr = &net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: port,
	}

	stripped, err := fs.Sub(frontend, "public")
	if err != nil {
		log.Fatalln(err)
	}

	s.app.Use(cors.New())

	api := s.app.Group("/api")
	api.Get("/blocks/all", s.GetBLocks)
	//api.Get("/transactions", s.GetTransactions)
	api.Get("/blocks/:hash", s.GetBlockByHash)
	api.Get("/transactions/:hash", s.GetTransactionByHash)

	s.app.Use("/", filesystem.New(filesystem.Config{
		Root:   http.FS(stripped),
		Index:  "index.html",
		Browse: true,
	}))

	addr := fmt.Sprintf("%s:%d", s.config.Addr.IP.String(), s.config.Addr.Port)
	go func() {
		if err := s.app.Listen(addr); err != nil {
			log.Fatalf("Error while running server: %s", err.Error())
		}
	}()
}

func (s *Server) ShutdownGracefully() {
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	defer func() {
		// Release resources like Database connections
		cancel()
	}()

	shutdownChan := make(chan error, 1)
	go func() { shutdownChan <- s.app.Shutdown() }()

	select {
	case <-timeout.Done():
		log.Fatal("Server Shutdown Timed out before shutdown.")
	case err := <-shutdownChan:
		if err != nil {
			log.Fatal("Error while shutting down server", err)
		} else {
			log.Printf("[NODE] Server gracefully stopped")
		}
	}
}

func (s *Server) GetBLocks(c *fiber.Ctx) error {
	getBLocksRequest := &pb.GetBlocksRequest{}
	getBLocksResponse, err := s.nc.GetBlocks(context.Background(), getBLocksRequest)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	blocks := make([]string, 0, len(getBLocksResponse.GetBlocks()))
	for _, block := range getBLocksResponse.GetBlocks() {
		blocks = append(blocks, block)
	}

	m, err := json.Marshal(struct {
		Blocks []string `json:"blocks"`
	}{
		Blocks: blocks,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).SendString(string(m[:]))
}

func (s *Server) GetTransactions(ctx *fiber.Ctx) error {
	getTransactionsRequest := &pb.GetTransactionsRequest{}
	getTransactionsResponse, err := s.nc.GetTransactions(context.Background(), getTransactionsRequest)
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	transactions := make([]string, 0, len(getTransactionsResponse.GetTransactions()))
	m, err := json.Marshal(struct {
		Transactions []string `json:"transactions"`
		Length       int      `json:"length"`
	}{
		Transactions: transactions,
		Length:       len(transactions),
	})
	if err != nil {
		return ctx.Status(http.StatusInternalServerError).SendString("Error while getting transactions")
	}

	return ctx.SendString(string(m[:]))
}

func (s *Server) GetBlockByHash(c *fiber.Ctx) error {
	getBlockByHashRequest := &pb.GetBlockRequest{
		Hash: c.Params("hash"),
	}
	getBlockByHashResponse, err := s.nc.GetBlock(context.Background(), getBlockByHashRequest)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	d, err := json.Marshal(getBlockByHashResponse)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendString(string(d[:]))
}

func (s *Server) GetTransactionByHash(c *fiber.Ctx) error {
	getTransactionByHashRequest := &pb.GetTransactionRequest{
		Hash: c.Params("hash"),
	}
	getTransactionByHashResponse, err := s.nc.GetTransaction(context.Background(), getTransactionByHashRequest)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	m, err := json.Marshal(getTransactionByHashResponse)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendString(string(m[:]))
}
