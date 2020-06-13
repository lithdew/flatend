package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"net"
	"strconv"
	"sync"
)

var _ monte.Handler = (*Node)(nil)
var _ monte.ConnStateHandler = (*Node)(nil)

type Node struct {
	id  kademlia.ID
	sec kademlia.PrivateKey

	addr string

	srv *monte.Server

	mu sync.Mutex

	providers *Providers
	peers     map[string]*Peer
	handlers  map[string]Handler
}

func NewNode(sec kademlia.PrivateKey, addr string) (*Node, error) {
	node := &Node{
		sec: sec,
		srv: &monte.Server{},

		handlers:  make(map[string]Handler),
		providers: NewProviders(),
		peers:     make(map[string]*Peer),
	}

	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid addr: %w", err)
	}

	node.addr = addr

	port, err := strconv.ParseUint(p, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	node.id = kademlia.ID{
		Pub:  sec.Public(),
		Host: net.ParseIP(h),
		Port: uint16(port),
	}

	node.srv.Handler = node
	node.srv.ConnState = node

	return node, nil
}

func (n *Node) Register(service string, handler Handler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[service] = handler
}

func (n *Node) Deregister(service string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.handlers, service)
}

func (n *Node) Services() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	services := make([]string, 0, len(n.handlers))
	for service := range n.handlers {
		services = append(services, service)
	}
	return services
}

func (n *Node) getHandler(service string) Handler {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.handlers[service]
}

func (n *Node) Dial(addr string) error {
	n.mu.Lock()
	peer, exists := n.peers[addr]
	if !exists {
		peer = newPeer(n, addr)
		n.peers[addr] = peer
		go peer.Dial()
	}
	n.mu.Unlock()

	<-peer.ready
	return peer.err
}

func (n *Node) Process(services []string, data []byte) ([]byte, error) {
	providers := n.providers.getProviders(services...)

	pkt := RequestPacket{
		Services: services,
		Data:     data,
	}

	req := pkt.AppendTo([]byte{OpcodeRequest})

	var (
		dst []byte
		err error
	)

	for _, provider := range providers {
		dst, err = provider.conn.Request(dst[:0], req)
		if err == nil {
			dst, err = UnmarshalResponsePacket(dst)
		}
		if err == nil && dst != nil {
			return dst, nil
		}
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil, fmt.Errorf("no nodes were able to process your request for service(s): %s", services)
}

func (n *Node) HandleMessage(ctx *monte.Context) error {
	body := ctx.Body()
	if len(body) < 1 {
		return errors.New("no opcode recorded in packet")
	}
	opcode := body[0]
	body = body[1:]

	switch opcode {
	case OpcodeHandshake:
		return n.handleHandshakePacket(ctx, body)
	default:
		return fmt.Errorf("unknown opcode %d", opcode)
	}
}

func (n *Node) HandleConnState(conn *monte.Conn, state monte.ConnState) {
	if state != monte.StateClosed {
		return
	}
	provider := n.providers.deregister(conn)
	if provider == nil {
		return
	}
	fmt.Printf("%s has disconnected from you. Services: %s\n", provider.Addr(), provider.Services())
}

func (n *Node) deletePeer(addr string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.peers, addr)
}

func (n *Node) handleHandshakePacket(ctx *monte.Context, body []byte) error {
	pkt, err := UnmarshalHandshakePacket(body)
	if err != nil {
		return err
	}
	err = pkt.Validate(nil)
	if err != nil {
		return err
	}

	provider := n.providers.register(ctx.Conn(), pkt.ID, pkt.Services)
	fmt.Printf("%s has connected to you. Services: %s\n", provider.Addr(), provider.Services())

	pkt = HandshakePacket{
		ID:       &n.id,
		Services: n.Services(),
	}
	pkt.Signature = n.sec.Sign(pkt.AppendPayloadTo(body[:0]))

	return ctx.Reply(pkt.AppendTo(body[:0]))
}

func (n *Node) Serve(ln net.Listener) error { return n.srv.Serve(ln) }

func (n *Node) Shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.srv.Shutdown()

	for addr, peer := range n.peers {
		peer.client.Shutdown()
		delete(n.peers, addr)
	}
}
