package services

import (
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/pkg/services/blockchain"
	"github.com/fr13n8/go-blockchain/pkg/services/wallet"
)

type NodeService struct {
	Bc     *blockchain.BlockChain
	Miner  *miner.Miner
	Wallet *wallet.Wallet
}

func NewNodeService(bc *blockchain.BlockChain, miner *miner.Miner, wallet *wallet.Wallet) *NodeService {
	return &NodeService{
		Bc:     bc,
		Miner:  miner,
		Wallet: wallet,
	}
}
