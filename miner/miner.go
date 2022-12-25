package miner

import (
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/blockchain"
	"time"
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

func NewMiner(minerAddress string, solver block.Solver, bc *blockchain.BlockChain) *Miner {
	return &Miner{minerAddress: minerAddress, bc: bc, solver: solver}
}

func (m *Miner) GetBlockForMine() *block.Block {
	if m.bc.TransactionPool.Size() == 0 {
		return nil
	}
	m.bc.AddTransaction(blockchain.MINING_SENDER, m.bc.BlockChainAddress, MINING_REWARD, nil, nil)
	transactions := m.bc.GetTransactionPool()
	previousHash := m.bc.LastBlock().Hash()

	return block.New(0, previousHash, transactions)
}

func (m *Miner) Mine() bool {
	b := m.GetBlockForMine()
	if b == nil {
		return false
	}

	if m.ProofOfWork(b) {
		m.bc.CreateBlock(b)
		return true
	}

	return false
}

func (m *Miner) ProofOfWork(guessBlock *block.Block) bool {
	return m.solver.Solve(guessBlock)
}

func (m *Miner) StartMining() {
	m.Mine()
	time.AfterFunc(time.Second*MINING_TIMER, m.StartMining)
}
