package peer_manager

import (
	"context"
	"errors"
	pb "github.com/fr13n8/go-blockchain/gen/peer"
	"google.golang.org/grpc/peer"
	"sync"
)

type PeerManager struct {
	peers map[string]pb.PeerService_MessageServer
	sync.Mutex
}

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[string]pb.PeerService_MessageServer),
	}
}

//func (nm *PeerManager) GetClientFromPeerId(peerID peer.ID) (pb.PeerService_MessageClient, error) {
//	stream, err := nm.mainServer.Host.NewStream(context.Background(), peerID, protocol.ID(nm.mainServer.Config.ProtocolID))
//	if err != nil {
//		return nil, err
//	}
//	client := nm.mainServer.GrpcStream.Client(stream)
//	cn := pb.PeerService_MessageClient(client)
//	return cn, nil
//}

func (nm *PeerManager) AddPeer(s pb.PeerService_MessageServer) error {
	nm.Lock()
	defer nm.Unlock()
	p, ok := peer.FromContext(s.Context())
	if !ok {
		return errors.New("failed to get peer from context")
	}
	nm.peers[p.Addr.String()] = s
	return nil
}

func (nm *PeerManager) RemovePeer(address string) {
	nm.Lock()
	defer nm.Unlock()
	delete(nm.peers, address)
}

func (nm *PeerManager) GetPeers() map[string]pb.PeerService_MessageServer {
	nm.Lock()
	defer nm.Unlock()
	return nm.peers
}

func (nm *PeerManager) GetAllPeersAddresses() []string {
	nm.Lock()
	defer nm.Unlock()
	var addresses []string
	for address := range nm.peers {
		addresses = append(addresses, address)
	}
	return addresses
}

func (nm *PeerManager) BroadcastMessage(ctx context.Context, msg *pb.MessageBody) {
	for addr, s := range nm.GetPeers() {
		go func(server pb.PeerService_MessageServer, addr string) {
			if err := server.Send(msg); err != nil {
				nm.RemovePeer(addr)
			}
		}(s, addr)
	}
}
