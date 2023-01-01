package node

import (
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/fr13n8/go-blockchain/gen/node"
	"github.com/fr13n8/go-blockchain/network"
	"github.com/fr13n8/go-blockchain/transaction"
	"github.com/fr13n8/go-blockchain/utils"
	"github.com/pkg/errors"
)

type NodeHandler struct {
	pb.UnimplementedNodeServiceServer
	ns *Server
}

func NewNodeHandler(service *Server) *NodeHandler {
	return &NodeHandler{
		ns: service,
	}
}

func (h *NodeHandler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	resp := &pb.PingResponse{
		Message: "pong",
	}
	return resp, nil
}

func (h *NodeHandler) GetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	blocks := make([]string, 0, len(h.ns.config.Bc.GetBlocks()))
	for _, b := range h.ns.config.Bc.GetBlocks() {
		blocks = append(blocks, b.HexHash())
	}

	return &pb.GetBlocksResponse{
		Blocks: blocks,
	}, nil
}

func (h *NodeHandler) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.BlockResponse, error) {
	hash := req.GetHash()
	b, err := h.ns.config.Bc.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	var transactions []*pb.TransactionResponse
	for _, tx := range b.Transactions {
		transactions = append(transactions, &pb.TransactionResponse{
			Id:               tx.HexHash(),
			SenderAddress:    tx.SenderAddress,
			RecipientAddress: tx.RecipientAddress,
			Amount:           tx.Amount,
		})
	}

	return &pb.BlockResponse{
		Header: &pb.Header{
			Timestamp:      b.Timestamp,
			MerkleRootHash: fmt.Sprintf("%x", b.MerkleRootHash),
			Hash:           b.HexHash(),
			PreviousHash:   fmt.Sprintf("%x", b.PreviousHash),
			Nonce:          b.Nonce,
			Target:         fmt.Sprintf("%x", b.Header.Target),
		},
		Transactions: transactions,
	}, nil
}

func (h *NodeHandler) GetTransactions(ctx context.Context, req *pb.GetTransactionsRequest) (*pb.GetTransactionsResponse, error) {
	var transactions []string
	for _, tx := range h.ns.config.Bc.ReadTransactionsPool() {
		transactions = append(transactions, tx.HexHash())
	}

	return &pb.GetTransactionsResponse{
		Transactions: transactions,
	}, nil
}

func (h *NodeHandler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.CreateTransactionResponse, error) {
	tx := transaction.Request{
		SenderAddress:    req.GetSenderAddress(),
		RecipientAddress: req.GetRecipientAddress(),
		Amount:           req.GetAmount(),
		SenderPublicKey:  req.GetSenderPublicKey(),
		Signature:        req.GetSignature(),
	}

	if !tx.Validate() {
		return nil, fmt.Errorf("invalid transaction")
	}

	publicKey := utils.PublicKeyFromString(tx.SenderPublicKey)
	signature := utils.SignatureFromString(tx.Signature)
	bc := h.ns.config.Bc

	isCreated := bc.CreateTransaction(tx.SenderAddress, tx.RecipientAddress, tx.Amount, publicKey, signature)

	if !isCreated {
		return nil, fmt.Errorf("transaction not created")
	}

	txData, err := json.Marshal(tx)
	if err != nil {
		return nil, errors.Wrap(err, "marshal transaction")
	}
	h.ns.config.PeerManager.BroadcastMessage(ctx, network.NewAddTXMessage(txData))

	return &pb.CreateTransactionResponse{
		TransactionId: "1",
	}, nil
}

func (h *NodeHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.TransactionResponse, error) {
	hash := req.GetHash()
	tx, err := h.ns.config.Bc.GetTransactionByHash(hash)
	if err != nil {
		return nil, err
	}

	return &pb.TransactionResponse{
		Id:               tx.HexHash(),
		SenderAddress:    tx.SenderAddress,
		RecipientAddress: tx.RecipientAddress,
		Amount:           tx.Amount,
	}, nil
}

func (h *NodeHandler) StartMining(ctx context.Context, req *pb.StartMiningRequest) (*pb.StartMiningResponse, error) {
	minerAddress := req.GetMinerAddress()
	h.ns.config.Miner.SetMinerAddress(minerAddress)
	h.ns.config.Miner.StartMining()

	return &pb.StartMiningResponse{
		Status: true,
	}, nil
}

func (h *NodeHandler) StopMining(ctx context.Context, req *pb.StopMiningRequest) (*pb.StopMiningResponse, error) {
	h.ns.config.Miner.StopMining()

	return &pb.StopMiningResponse{
		Status: true,
	}, nil
}

func (h *NodeHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	address := req.GetAddress()
	balance := h.ns.config.Bc.Balance(address)

	return &pb.GetBalanceResponse{
		Balance: balance,
	}, nil
}
