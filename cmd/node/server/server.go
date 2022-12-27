package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/blockchain"
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"github.com/fr13n8/go-blockchain/wallet"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed public
var frontend embed.FS

var cache = make(map[string]*blockchain.BlockChain)

type Config struct {
	Port       uint16
	Host       string
	ServerName string
}

type Server struct {
	app  *fiber.App
	host string
	port uint16
}

func (s *Server) Port() uint16 {
	return s.port
}

func NewServer(cfg *Config) *Server {
	return &Server{
		app: fiber.New(
			fiber.Config{
				AppName: cfg.ServerName,
			}),
		port: cfg.Port,
		host: cfg.Host,
	}
}

func (s *Server) Run() {
	stripped, err := fs.Sub(frontend, "public")
	if err != nil {
		log.Fatalln(err)
	}

	s.app.Use(cors.New())

	api := s.app.Group("/api")
	api.Get("/chain", s.GetChain)
	api.Get("/blocks/all", s.GetBLocks)
	api.Post("/transactions", s.CreateTransaction)
	api.Get("/transactions", s.GetTransactions)
	api.Get("/mine", s.Mine)
	api.Get("/mine/start", s.StartMining)
	api.Get("/mine/stop", s.StopMining)
	api.Get("/balance/:address", s.GetBalance)
	api.Get("/blocks/:hash", s.GetBlockByHash)
	api.Get("/transactions/:hash", s.GetTransactionByHash)

	s.app.Use("/", filesystem.New(filesystem.Config{
		Root:   http.FS(stripped),
		Index:  "index.html",
		Browse: true,
	}))

	go func() {
		if err := s.app.Listen(fmt.Sprintf("%s:%s", s.host, fmt.Sprintf("%d", s.port))); err != nil {
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

func (s *Server) getBlockChain() *blockchain.BlockChain {
	bc, ok := cache["blockchain"]
	if !ok {
		minerWallet := wallet.NewWallet()
		bc = blockchain.NewBlockChain(minerWallet.BlockChainAddress(), s.Port())
		cache["blockchain"] = bc
	}

	return bc
}

func (s *Server) GetTransactionByHash(c *fiber.Ctx) error {
	bc := s.getBlockChain()
	hash := c.Params("hash")
	tx, err := bc.GetTransactionByHash(hash)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(tx)
}

func (s *Server) GetBlockByHash(c *fiber.Ctx) error {
	bc := s.getBlockChain()
	hash := c.Params("hash")
	block, err := bc.GetBlockByHash(hash)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(block)
}

func (s *Server) GetBLocks(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()
	blocks := make([]string, 0, len(bc.GetBlocks()))
	for _, b := range bc.GetBlocks() {
		blocks = append(blocks, b.HexHash())
	}

	m, err := json.Marshal(struct {
		Blocks []string `json:"blocks"`
	}{
		Blocks: blocks,
	})
	if err != nil {
		return err
	}

	return ctx.SendString(string(m[:]))
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
		return ctx.Status(http.StatusInternalServerError).SendString("Error while getting transactions")
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

	return ctx.Status(http.StatusOK).SendString("")
}

func (s *Server) StopMining(ctx *fiber.Ctx) error {
	bc := s.getBlockChain()

	solver := block.NewSHA256Solver()
	m := miner.NewMiner(bc.BlockChainAddress, solver, bc)
	m.StopMining()

	return ctx.Status(http.StatusOK).SendString("Mining stopped")
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
