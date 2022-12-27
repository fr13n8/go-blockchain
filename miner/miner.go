package miner

import (
	"github.com/fr13n8/go-blockchain/pkg/services/blockchain"
	"log"
	"time"

	"github.com/fr13n8/go-blockchain/block"
)

const (
	MINING_REWARD = 1.0
	MINING_TIMER  = 20
)

type Miner struct {
	minerAddress string
	solver       block.Solver
	bc           *blockchain.BlockChain
}

func NewMiner(solver block.Solver, bc *blockchain.BlockChain) *Miner {
	return &Miner{bc: bc, solver: solver}
}

func (m *Miner) SetMinerAddress(address string) {
	m.minerAddress = address
}

func (m *Miner) GetBlockForMine() *block.Block {
	if m.bc.TransactionPool.Size() == 0 {
		return nil
	}
	m.bc.AddTransaction(blockchain.MINING_SENDER, m.minerAddress, MINING_REWARD, nil, nil)
	transactions := m.bc.GetTransactionPool()
	previousHash := m.bc.LastBlock().Hash()

	return block.New(0, previousHash, transactions)
}

func (m *Miner) Mine() bool {
	b := m.GetBlockForMine()
	if b == nil {
		return false
	}

	log.Println("[NODE] Mining new block")
	if m.ProofOfWork(b) {
		m.bc.CreateBlock(b)
		log.Printf("[NODE] Mining block %s success", b.HexHash())
		return true
	}

	log.Printf("[NODE] Mining block %s failed", b.HexHash())
	return false
}

func (m *Miner) ProofOfWork(guessBlock *block.Block) bool {
	return m.solver.Solve(guessBlock)
}

var t *time.Timer

func (m *Miner) StartMining() {
	m.Mine()
	t = time.AfterFunc(time.Second*MINING_TIMER, m.StartMining)
}

func (m *Miner) StopMining() {
	if stoped := t.Stop(); !stoped {
		log.Println("[NODE] Mining already stoped")
	}

	log.Println("[NODE] Mining stoped")
}
