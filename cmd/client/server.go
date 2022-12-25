package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/blockchain"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"github.com/fr13n8/go-blockchain/wallet"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"net/http"
	"strconv"
)

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServerConfig struct {
	port       uint16
	gateway    string
	host       string
	serverName string
}

type Server struct {
	app     *fiber.App
	host    string
	gateway string
	port    uint16
}

func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		app: fiber.New(
			fiber.Config{
				AppName: cfg.serverName,
			}),
		gateway: cfg.gateway,
		port:    cfg.port,
		host:    cfg.host,
	}
}

func (s *Server) Run() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	s.app.Use(cors.New())

	cfg := fiber.Static{
		Compress:      true,
		ByteRange:     true,
		Browse:        true,
		Index:         "index.html",
		CacheDuration: 10 * time.Second,
		MaxAge:        3600,
	}

	s.app.Static("/assets", "cmd/client/public", fiber.Static{
		Compress:      true,
		ByteRange:     true,
		Browse:        true,
		CacheDuration: 10 * time.Second,
		MaxAge:        3600,
	})
	s.app.Post("/wallet/create", s.WalletCreate)
	s.app.Post("/transaction/create", s.CreateTransaction)
	s.app.Get("/wallet/balance/:address", s.GetBalance)

	s.app.Static("/", "cmd/client/public", cfg)
	s.app.Static("/*", "cmd/client/public", cfg)

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
			fmt.Println("Server Shutdown Successful")
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
	w := wallet.NewWallet()
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

	t := wallet.NewTransaction(privateKey, publicKey, tr.SenderBlockChainAddress, tr.RecipientBlockChainAddress, amount32)
	signature := t.GenerateSignature()

	signatureStr := signature.String()

	bt := transaction.Request{
		SenderPublicKey:  tr.SenderPublicKey,
		RecipientAddress: tr.RecipientBlockChainAddress,
		SenderAddress:    tr.SenderBlockChainAddress,
		Amount:           amount32,
		Signature:        signatureStr,
	}
	m, err := json.Marshal(bt)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(m)

	resp, err := http.Post(fmt.Sprintf("%s/transactions", s.gateway), "application/json", buf)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusCreated {
		return ctx.JSON(fiber.Map{
			"message": "tr created",
			"success": true,
		})
	}
	defer resp.Body.Close()

	return ctx.JSON(fiber.Map{
		"message": "tr failed",
		"success": false,
	})
}

func (s *Server) GetBalance(ctx *fiber.Ctx) error {
	address := ctx.Params("address")
	resp, err := http.Get(fmt.Sprintf("%s/balance/%s", s.gateway, address))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusCreated {
		balance := blockchain.BalanceResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
			return err
		}
		return ctx.JSON(fiber.Map{
			"message": "balance retrieved",
			"success": true,
			"balance": balance.Balance,
		})
	}
	defer resp.Body.Close()

	return ctx.JSON(fiber.Map{
		"message": "get balance failed",
		"success": false,
	})
}
