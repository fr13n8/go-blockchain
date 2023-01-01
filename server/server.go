package server

import (
	"github.com/fr13n8/go-blockchain/block"
	"github.com/fr13n8/go-blockchain/block-explorer"
	"github.com/fr13n8/go-blockchain/blockchain"
	"github.com/fr13n8/go-blockchain/miner"
	"github.com/fr13n8/go-blockchain/network"
	peer_manager "github.com/fr13n8/go-blockchain/network/peer-manager"
	"github.com/fr13n8/go-blockchain/node"
)

type Server struct {
	BlockExplorer *block_explorer.Server
	PeerDiscovery *network.Server
	NodeServer    *node.Server

	Bc          *blockchain.BlockChain
	Miner       *miner.Miner
	PeerManager *peer_manager.PeerManager
}

func NewServer() *Server {
	bc := blockchain.NewBlockChain()
	solver := block.NewSHA256Solver()
	m := miner.NewMiner(solver, bc)
	pm := peer_manager.NewPeerManager()

	pdCfg := network.NewConfig()
	pdCfg.PeerManager = pm
	pdCfg.Bc = bc
	pdCfg.Miner = m
	pd := network.NewServer(pdCfg)

	nCfg := node.NewConfig()
	nCfg.Bc = bc
	nCfg.Miner = m
	nCfg.PeerManager = pm
	ns := node.NewServer(nCfg)

	beCfg := block_explorer.NewConfig(ns.Addr().String())
	be := block_explorer.NewServer(beCfg)

	return &Server{
		BlockExplorer: be,
		PeerDiscovery: pd,
		NodeServer:    ns,
	}
}
