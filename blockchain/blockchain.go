package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/fr13n8/go-blockchain/transaction"

	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/trxpool"
	"github.com/fr13n8/go-blockchain/utils"
)

const (
	MINING_SENDER = "THE BLOCKCHAIN"
)

type BlockChain struct {
	TransactionPool *trxpool.TransactionPool
	chain           []*block.Block
	mux             sync.Mutex
}

func NewBlockChain() *BlockChain {
	b := block.NewGenesisBlock([]*transaction.Transaction{})
	trxPoll := trxpool.NewTransactionPool()
	bc := &BlockChain{
		TransactionPool: trxPoll,
		chain:           []*block.Block{},
	}

	bc.CreateBlock(b)
	return bc
}

func (bc *BlockChain) GetBlocks() []*block.Block {
	return bc.chain
}

func (bc *BlockChain) ReadTransactionsPool() []*transaction.Transaction {
	return bc.TransactionPool.Read(10)
}

func (bc *BlockChain) GetTransactionPool() []*transaction.Transaction {
	return bc.TransactionPool.GetAndClean(10)
}

func (bc *BlockChain) GetBlockByHash(hash string) (*block.Block, error) {
	blockHashBytes, err := hex.DecodeString(hash)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return nil, err
	}

	for _, b := range bc.chain {
		blockHash := b.Hash()
		if bytes.Equal(blockHash[:], blockHashBytes) {
			return b, nil
		}
	}
	return nil, fmt.Errorf("block with hash %s not found", hash)
}

func (bc *BlockChain) GetTransactionByHash(hash string) (*transaction.Transaction, error) {
	var tx *transaction.Transaction
	for _, b := range bc.chain {
		for _, t := range b.Transactions {
			if t.HexHash() == hash {
				tx = t
			}
		}
	}
	if tx == nil {
		return nil, fmt.Errorf("transaction with hash %s not found", hash)
	}
	return tx, nil
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
	bc.mux.Lock()
	defer bc.mux.Unlock()
	isTransactionAdded := bc.AddTransaction(senderAddress, recipientAddress, value, senderPublicKey, s)
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
	first := sha256.Sum256(m)
	h := sha256.Sum256(first[:])
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S)
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
