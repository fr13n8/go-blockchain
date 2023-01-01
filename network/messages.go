package network

import (
	pb "github.com/fr13n8/go-blockchain/gen/peer"
)

const (
	messageAddTx    = "add_tx"
	messageAddBlock = "add_block"
	getPeers        = "get_peers"
)

func NewAddTXMessage(txData []byte) *pb.MessageBody {
	return &pb.MessageBody{
		Type: messageAddTx,
		Data: txData,
	}
}

func NewAddBlockMessage(blockData []byte) *pb.MessageBody {
	return &pb.MessageBody{
		Type: messageAddBlock,
		Data: blockData,
	}
}

func NewGetPeersMessage(peersBytes []byte) *pb.MessageBody {
	return &pb.MessageBody{
		Type: getPeers,
		Data: peersBytes,
	}
}
