package network

import (
	"encoding/json"
	pb "github.com/fr13n8/go-blockchain/gen/peer"
	"github.com/fr13n8/go-blockchain/network/peer-manager"
	"github.com/pkg/errors"
	"io"
)

type PeerHandler struct {
	pb.UnimplementedPeerServiceServer
	pm *peer_manager.PeerManager
}

func NewPeerHandler(service *peer_manager.PeerManager) *PeerHandler {
	return &PeerHandler{
		pm: service,
	}
}

func (h *PeerHandler) Message(serverStream pb.PeerService_MessageServer) error {
	if err := h.pm.AddPeer(serverStream); err != nil {
		return errors.Wrap(err, "failed to add server")
	}
	for {
		msg, err := serverStream.Recv()
		if err != nil {
			return err
		}
		if err == io.EOF {
			return nil
		}
		ctx := serverStream.Context()
		switch msg.Type {
		case messageAddTx:
			//var tx transaction.Request
			//if err := json.Unmarshal(msg.Data, &tx); err != nil {
			//	return errors.Wrap(err, "unmarshal transaction")
			//}
			//publicKey := utils.PublicKeyFromString(tx.SenderPublicKey)
			//signature := utils.SignatureFromString(tx.Signature)
			//bc := h.ns.Bc
			//isCreated := bc.CreateTransaction(tx.SenderAddress, tx.RecipientAddress, tx.Amount, publicKey, signature)
			//if !isCreated {
			//	return fmt.Errorf("transaction not created")
			//}
			//h.ns.PeerManager.BroadcastMessage(ctx, msg)
		case messageAddBlock:
			//var b block.Block
			//if err := json.Unmarshal(msg.Data, &b); err != nil {
			//	return errors.Wrap(err, "unmarshal block")
			//}
			//bc := h.ns.Bc
			//bc.CreateBlock(&b)
			//h.ns.PeerManager.BroadcastMessage(ctx, msg)
		case getPeers:
			peers := h.pm.GetPeers()
			peersBytes, err := json.Marshal(peers)
			if err != nil {
				return errors.Wrap(err, "marshal peers")
			}

			h.pm.BroadcastMessage(ctx, NewGetPeersMessage(peersBytes))
		}
	}
}
