package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	MINING_DIFFICULTY = 3
	MINING_SENDER     = "THE BLOCKCHAIN"
	MINING_REWARD     = 1.0
	MINING_TIMER      = 20
)

type BlockChain struct {
	transactionPool   []*transaction.Transaction
	chain             []*block.Block
	blockChainAddress string
	port              uint16
	mux               sync.Mutex
}

func NewBlockChain(blockChainAddress string, port uint16) *BlockChain {
	b := block.NewGenesisBlock([]*transaction.Transaction{})
	bc := &BlockChain{
		transactionPool:   []*transaction.Transaction{},
		chain:             []*block.Block{},
		blockChainAddress: blockChainAddress,
		port:              port,
	}

	bc.CreateBlock(0, b.Hash())
	return bc
}

func (bc *BlockChain) GetTransactionsPool() []*transaction.Transaction {
	return bc.transactionPool
}

func (bc *BlockChain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*block.Block `json:"chains"`
	}{
		Blocks: bc.chain,
	})
}

func (bc *BlockChain) CreateBlock(nonce uint64, previousHash [32]byte) *block.Block {
	b := block.NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*transaction.Transaction{}
	return b
}

func (bc *BlockChain) LastBlock() *block.Block {
	return bc.chain[len(bc.chain)-1]
}

func (bc *BlockChain) Print() {
	for i, b := range bc.chain {
		fmt.Printf("%s Block: %d %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		b.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("*", 25))
}

func (bc *BlockChain) CreateTransaction(senderAddress, recipientAddress string, value float32, senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	isTransactionAdded := bc.AddTransaction(senderAddress, recipientAddress, value, senderPublicKey, s)
	// TODO: add mutex
	return isTransactionAdded
}

func (bc *BlockChain) AddTransaction(senderAddress, recipientAddress string, value float32, senderPublicKey *ecdsa.PublicKey, s *utils.Signature) bool {
	t := transaction.NewTransaction(senderAddress, recipientAddress, value)

	if senderAddress == MINING_SENDER {
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {
		//if bc.Balance(senderAddress) < value {
		//	log.Printf("ERROR: Not enough balance in wallet")
		//	return false
		//}
		bc.transactionPool = append(bc.transactionPool, t)
		return true
	}
	log.Printf("ERROR: Invalid transaction from %s\n", senderAddress)
	return false
}

func (bc *BlockChain) VerifyTransactionSignature(senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *transaction.Transaction) bool {
	m, err := t.MarshalJSON()
	if err != nil {
		panic(err)
	}
	h := sha256.Sum256(m)
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
}

func (bc *BlockChain) CopyTransactionPool() []*transaction.Transaction {
	var transactions []*transaction.Transaction
	for _, t := range bc.transactionPool {
		transactions = append(transactions, transaction.NewTransaction(t.SenderAddress, t.RecipientAddress, t.Amount))
	}
	return transactions
}

func (bc *BlockChain) IsValid(nonce uint64, previousHash [32]byte, transactions []*transaction.Transaction, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := block.Block{
		Header: block.Header{
			Nonce:        nonce,
			PreviousHash: previousHash,
			Timestamp:    0,
		},
		Transactions: transactions,
	}
	guessHashStr := fmt.Sprintf("%x", guessBlock.Hash())
	return guessHashStr[:difficulty] == zeros
}

func (bc *BlockChain) ProofOfWork() uint64 {
	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := uint64(0)
	for !bc.IsValid(nonce, previousHash, transactions, MINING_DIFFICULTY) {
		nonce++
	}
	return nonce
}

func (bc *BlockChain) Mine() bool {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	if len(bc.transactionPool) == 0 {
		return false
	}

	bc.AddTransaction(MINING_SENDER, bc.blockChainAddress, MINING_REWARD, nil, nil)
	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)

	return true
}

func (bc *BlockChain) StartMining() {
	bc.Mine()
	time.AfterFunc(time.Second*MINING_TIMER, bc.StartMining)
}

func (bc *BlockChain) Balance(blockChainAddress string) float32 {
	var balance float32
	for _, b := range bc.chain {
		for _, t := range b.Transactions {
			if t.SenderAddress == blockChainAddress {
				balance -= t.Amount
			}
			if t.RecipientAddress == blockChainAddress {
				balance += t.Amount
			}
		}
	}
	return balance
}

type BalanceResponse struct {
	Balance float32 `json:"balance"`
}

func (b *BalanceResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Balance float32 `json:"balance"`
	}{
		Balance: b.Balance,
	})
}
