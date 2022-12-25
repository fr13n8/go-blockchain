package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/blockchain"
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"github.com/fr13n8/go-blockchain/wallet"
	"github.com/gofiber/fiber/v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var cache = make(map[string]*blockchain.BlockChain)

type ServerConfig struct {
	port       uint16
	host       string
	serverName string
}

type Server struct {
	app  *fiber.App
	host string
	port uint16
}

func (s *Server) Port() uint16 {
	return s.port
}

func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		app: fiber.New(
			fiber.Config{
				AppName: cfg.serverName,
			}),
		port: cfg.port,
		host: cfg.host,
	}
}

func (s *Server) Run() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	s.app.Get("/chain", s.GetChain)
	s.app.Post("/transactions", s.CreateTransaction)
	s.app.Get("/transactions", s.GetTransactions)
	s.app.Get("/mine", s.Mine)
	s.app.Get("/mine/start", s.StartMining)
	s.app.Get("/balance/:address", s.GetBalance)

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

func (s *Server) getBlockChain() *blockchain.BlockChain {
	bc, ok := cache["blockchain"]
	if !ok {
		minerWallet := wallet.NewWallet()
		bc = blockchain.NewBlockChain(minerWallet.BlockChainAddress(), s.Port())
		cache["blockchain"] = bc
	}

	return bc
}

func (s *Server) GetChain(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()
	m, err := bc.MarshalJSON()
	if err != nil {
		panic(err)
	}

	return ctx.SendString(string(m[:]))
}

func (s *Server) CreateTransaction(ctx *fiber.Ctx) error {
	tReq := transaction.Request{}
	if err := ctx.BodyParser(&tReq); err != nil {
		return err
	}

	if !tReq.Validate() {
		return ctx.Status(fiber.StatusBadRequest).SendString("Invalid transaction request")
	}

	publicKey := utils.PublicKeyFromString(tReq.SenderPublicKey)
	signature := utils.SignatureFromString(tReq.Signature)
	bc := s.getBlockChain()

	isCreated := bc.CreateTransaction(tReq.SenderAddress, tReq.RecipientAddress, tReq.Amount, publicKey, signature)

	if !isCreated {
		return ctx.Status(http.StatusBadRequest).SendString("Transaction is not created")
	}

	return ctx.Status(http.StatusCreated).SendString("Transaction is created")
}

func (s *Server) GetTransactions(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()
	transactions := bc.ReadTransactionsPool()
	m, err := json.Marshal(struct {
		Transactions []*transaction.Transaction `json:"transactions"`
		Length       int                        `json:"length"`
	}{
		Transactions: transactions,
		Length:       len(transactions),
	})

	if err != nil {
		panic(err)
		ctx.Status(http.StatusInternalServerError).SendString("Error while getting transactions")
	}

	return ctx.SendString(string(m[:]))
}

func (s *Server) Mine(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()

	solver := block.NewSHA256Solver()
	m := miner.NewMiner(bc.BlockChainAddress, solver, bc)
	isMined := m.Mine()
	if !isMined {
		return ctx.Status(http.StatusInternalServerError).SendString("Error while mining")
	}

	return ctx.Status(http.StatusOK).SendString("Block is mined")
}

func (s *Server) StartMining(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()

	solver := block.NewSHA256Solver()
	m := miner.NewMiner(bc.BlockChainAddress, solver, bc)
	m.StartMining()

	return ctx.Status(http.StatusOK).SendString("Mining started")
}

func (s *Server) GetBalance(ctx *fiber.Ctx) error {
	address := ctx.Params("address")
	bc := s.getBlockChain()
	balance := bc.Balance(address)
	resp := blockchain.BalanceResponse{
		Balance: balance,
	}

	m, err := resp.MarshalJSON()
	if err != nil {
		panic(err)
	}

	return ctx.Status(http.StatusCreated).SendString(string(m[:]))
}
