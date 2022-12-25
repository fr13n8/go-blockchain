package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/trxpool"
	"github.com/fr13n8/go-blockchain/utils"
	"log"
	"strings"
	"sync"
)

const (
	MINING_SENDER = "THE BLOCKCHAIN"
)

type BlockChain struct {
	TransactionPool   *trxpool.TransactionPool
	chain             []*block.Block
	BlockChainAddress string
	port              uint16
	mux               sync.Mutex
}

func NewBlockChain(blockChainAddress string, port uint16) *BlockChain {
	b := block.NewGenesisBlock([]*transaction.Transaction{})
	trxPoll := trxpool.NewTransactionPool()
	bc := &BlockChain{
		TransactionPool:   trxPoll,
		chain:             []*block.Block{},
		BlockChainAddress: blockChainAddress,
		port:              port,
	}

	bc.CreateBlock(b)
	return bc
}

func (bc *BlockChain) ReadTransactionsPool() []*transaction.Transaction {
	return bc.TransactionPool.Read(10)
}

func (bc *BlockChain) GetTransactionPool() []*transaction.Transaction {
	return bc.TransactionPool.GetAndClean(10)
}

func (bc *BlockChain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*block.Block `json:"chains"`
	}{
		Blocks: bc.chain,
	})
}

func (bc *BlockChain) CreateBlock(b *block.Block) *block.Block {
	bc.chain = append(bc.chain, b)
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
		bc.TransactionPool.Add(t)
		return true
	}

	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {
		//if bc.Balance(senderAddress) < value {
		//	log.Printf("ERROR: Not enough balance in wallet")
		//	return false
		//}
		bc.TransactionPool.Add(t)
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

//func (bc *BlockChain) CopyTransactionPool() []*transaction.Transaction {
//	var transactions []*transaction.Transaction
//	for _, t := range bc.TransactionPool {
//		transactions = append(transactions, transaction.NewTransaction(t.SenderAddress, t.RecipientAddress, t.Amount))
//	}
//	return transactions
//}

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
