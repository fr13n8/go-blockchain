package wallet

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/a-h/templ"
	pb "github.com/fr13n8/go-blockchain/gen/node"
	"github.com/fr13n8/go-blockchain/wallet/components"
	"github.com/fr13n8/go-blockchain/wallet/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/fr13n8/go-blockchain/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// //go:embed public
// var frontend embed.FS

//go:embed assets
var assets embed.FS

type Config struct {
	Port       uint16
	Gateway    string
	Host       string
	ServerName string
}

type Server struct {
	app     *fiber.App
	host    string
	gateway string
	nc      pb.NodeServiceClient
	port    uint16
}

func NewServer(cfg *Config) *Server {
	conn, err := grpc.Dial(cfg.Gateway, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[WALLET] Error while connecting to gateway: %s", err.Error())
	}

	client := pb.NewNodeServiceClient(conn)
	pingResponse, err := client.Ping(context.Background(), &pb.PingRequest{})
	if err != nil {
		log.Fatalf("[WALLET] Error while pinging gateway: %s", err.Error())
		return nil
	}
	log.Printf("[WALLET] Connected to gateway: %s", pingResponse.Message)

	return &Server{
		app: fiber.New(
			fiber.Config{
				AppName:               cfg.ServerName,
				DisableStartupMessage: true,
			}),
		gateway: cfg.Gateway,
		nc:      client,
		port:    cfg.Port,
		host:    cfg.Host,
	}
}

func (s *Server) Run() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	// stripped, err := fs.Sub(frontend, "public")
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	s.app.Use(cors.New())

	s.app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	api := s.app.Group("/api")
	api.Post("/wallet/create", s.WalletCreate)
	api.Post("/transaction/create", s.CreateTransaction)
	api.Get("/wallet/balance/:address", s.GetBalance)

	// s.app.Use("/", filesystem.New(filesystem.Config{
	// 	Root:   http.FS(stripped),
	// 	Index:  "index.html",
	// 	Browse: true,
	// }))
	s.app.Use("/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(assets),
		PathPrefix: "assets",
		Browse:     true,
	}))
	s.app.Get("/", s.MainView)
	s.app.Get("/wallet/balance/:address", s.GetBalanceView)
	go func() {
		if err := s.app.Listen(fmt.Sprintf("%s:%s", s.host, fmt.Sprintf("%d", s.port))); err != nil {
			log.Fatalf("Error while running server: %s", err.Error())
		}
	}()

	return quit
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
			log.Printf("[WALLET] Server gracefully stopped")
		}
	}
}

func (s *Server) Port() uint16 {
	return s.port
}

func (s *Server) Gateway() string {
	return s.gateway
}

func (s *Server) Host() string {
	return s.host
}

func (s *Server) WalletCreate(ctx *fiber.Ctx) error {
	w := NewWallet()
	walletJson, err := w.MarshalJSON()
	if err != nil {
		return err
	}

	return ctx.SendString(string(walletJson[:]))
}

type TransactionRequest struct {
	Amount                     string `json:"amount"`
	SenderPrivateKey           string `json:"sender_private_key"`
	SenderPublicKey            string `json:"sender_public_key"`
	SenderBlockChainAddress    string `json:"sender_blockchain_address"`
	RecipientBlockChainAddress string `json:"recipient_blockchain_address"`
}

func (tr *TransactionRequest) Validate() error {
	amount, err := strconv.ParseFloat(tr.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", tr.Amount)
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	if tr.SenderPrivateKey == "" {
		return fmt.Errorf("sender private key is required")
	}

	if tr.SenderPublicKey == "" {
		return fmt.Errorf("sender public key is required")
	}

	if tr.SenderBlockChainAddress == "" {
		return fmt.Errorf("sender blockchain address is required")
	}

	if tr.RecipientBlockChainAddress == "" {
		return fmt.Errorf("recipient blockchain address is required")
	}

	return nil
}

func (s *Server) CreateTransaction(ctx *fiber.Ctx) error {
	tr := TransactionRequest{}
	if err := ctx.BodyParser(&tr); err != nil {
		return err
	}

	if err := tr.Validate(); err != nil {
		return err
	}

	publicKey := utils.PublicKeyFromString(tr.SenderPublicKey)
	privateKey := utils.PrivateKeyFromString(tr.SenderPrivateKey, publicKey)
	value, err := strconv.ParseFloat(tr.Amount, 64)
	if err != nil {
		return err
	}
	amount32 := float32(value)

	t := NewTransaction(privateKey, publicKey, tr.SenderBlockChainAddress, tr.RecipientBlockChainAddress, amount32)
	signature := t.GenerateSignature()

	signatureStr := signature.String()

	tx := pb.CreateTransactionRequest{
		SenderPublicKey:  tr.SenderPublicKey,
		RecipientAddress: tr.RecipientBlockChainAddress,
		SenderAddress:    tr.SenderBlockChainAddress,
		Amount:           amount32,
		Signature:        signatureStr,
	}

	_, err = s.nc.CreateTransaction(context.Background(), &tx)
	if err != nil {
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
			"success": true,
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Transaction created successfully",
		"success": true,
	})

}

func (s *Server) GetBalance(ctx *fiber.Ctx) error {
	address := ctx.Params("address")
	addressRequest := pb.GetBalanceRequest{
		Address: address,
	}

	balance, err := s.nc.GetBalance(context.Background(), &addressRequest)
	if err != nil {
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
			"success": false,
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Balance retrieved successfully",
		"success": true,
		"balance": balance.Balance,
	})
}

func (s *Server) MainView(ctx *fiber.Ctx) error {
	w := NewWallet()
	address := w.BlockChainAddress()
	addressRequest := pb.GetBalanceRequest{
		Address: address,
	}

	balance, err := s.nc.GetBalance(context.Background(), &addressRequest)
	if err != nil {
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
			"success": false,
		})
	}

	details := []*domain.WalletDetail{
		{
			Field: "Blockchain Address",
			Value: address,
			Id:    "blockchain_address",
		},
		{
			Field: "Private key",
			Value: w.PrivateKeyStr(),
			Id:    "private_key",
		},
		{
			Field: "Public key",
			Value: w.PublicKeyStr(),
			Id:    "publick_key",
		},
		{
			Field: "Balance",
			Value: fmt.Sprintf("%.2f", balance.Balance),
			Id:    "balance",
		},
	}

	return adaptor.HTTPHandler(templ.Handler(components.Page(details)))(ctx)
}

func (s *Server) GetBalanceView(ctx *fiber.Ctx) error {
	address := ctx.Params("address")
	addressRequest := pb.GetBalanceRequest{
		Address: address,
	}

	balance, err := s.nc.GetBalance(context.Background(), &addressRequest)
	if err != nil {
		return ctx.JSON(fiber.Map{
			"message": err.Error(),
			"success": false,
		})
	}

	return adaptor.HTTPHandler(templ.Handler(components.WalletDetailsItem("Balance", fmt.Sprintf("%.2f", balance.Balance), "balance")))(ctx)
}
