package grpc

import (
	"context"
	"errors"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net"

	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/grpc"
	grpcPeer "google.golang.org/grpc/peer"
)

type Stream struct {
	ctx      context.Context
	streamCh chan network.Stream

	grpcServer *grpc.Server
}

func NewStream() *Stream {
	return &Stream{
		ctx:        context.Background(),
		streamCh:   make(chan network.Stream),
		grpcServer: grpc.NewServer(grpc.UnaryInterceptor(interceptor)),
	}
}

type Context struct {
	context.Context
	PeerID peer.ID
}

func interceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	contextPeer, ok := grpcPeer.FromContext(ctx)
	if !ok {
		return nil, errors.New("invalid type assertion for peer context")
	}

	addr, ok := contextPeer.Addr.(*wrapLibp2pAddr)
	if !ok {
		return nil, errors.New("invalid type assertion")
	}

	return handler(
		&Context{
			Context: ctx,
			PeerID:  addr.id,
		},
		req,
	)
}

func (g *Stream) Client(stream network.Stream) *grpc.ClientConn {
	return WrapClient(stream)
}

func (g *Stream) Serve() {
	go func() {
		err := g.grpcServer.Serve(g)
		if err != nil {
			panic(err)
		}
	}()
}

func (g *Stream) Handler() func(network.Stream) {
	return func(stream network.Stream) {
		select {
		case <-g.ctx.Done():
			return
		case g.streamCh <- stream:
		}
	}
}

func (g *Stream) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	g.grpcServer.RegisterService(sd, ss)
}

func (g *Stream) GrpcServer() *grpc.Server {
	return g.grpcServer
}

func (g *Stream) Accept() (net.Conn, error) {
	select {
	case <-g.ctx.Done():
		return nil, io.EOF
	case stream := <-g.streamCh:
		return &streamConn{Stream: stream}, nil
	}
}

func (g *Stream) Addr() net.Addr {
	return fakeLocalAddr()
}

func (g *Stream) Close() error {
	return nil
}

func WrapClient(s network.Stream) *grpc.ClientConn {
	opts := grpc.WithContextDialer(func(ctx context.Context, peerIdStr string) (net.Conn, error) {
		return &streamConn{s}, nil
	})
	conn, err := grpc.Dial("", grpc.WithTransportCredentials(insecure.NewCredentials()), opts)

	if err != nil {
		// TODO: this should not fail at all
		panic(err)
	}

	return conn
}

type streamConn struct {
	network.Stream
}

type wrapLibp2pAddr struct {
	id peer.ID
	net.Addr
}

func (c *streamConn) LocalAddr() net.Addr {
	addr, err := manet.ToNetAddr(c.Stream.Conn().LocalMultiaddr())
	if err != nil {
		return fakeRemoteAddr()
	}

	return &wrapLibp2pAddr{Addr: addr, id: c.Stream.Conn().LocalPeer()}
}

func (c *streamConn) RemoteAddr() net.Addr {
	addr, err := manet.ToNetAddr(c.Stream.Conn().RemoteMultiaddr())
	if err != nil {
		return fakeRemoteAddr()
	}

	return &wrapLibp2pAddr{Addr: addr, id: c.Stream.Conn().RemotePeer()}
}

var _ net.Conn = &streamConn{}

func fakeLocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}
}

func fakeRemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.1.0.1"),
		Port: 0,
	}
}
