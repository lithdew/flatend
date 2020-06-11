package flatend

import (
	"errors"
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"time"
)

var _ monte.Handler = (*Peer)(nil)
var _ monte.ConnStateHandler = (*Peer)(nil)

type Peer struct {
	node  *Node
	ready chan struct{}
	err   error

	id     kademlia.ID
	addr   string
	svcs   []string
	client monte.Client
}

func newPeer(node *Node, addr string) *Peer {
	peer := &Peer{node: node, ready: make(chan struct{}), addr: addr, client: monte.Client{Addr: addr}}

	peer.client.Handler = peer
	peer.client.ConnState = peer

	return peer
}

func (p *Peer) HandleMessage(ctx *monte.Context) error {
	body := ctx.Body()
	if len(body) < 1 {
		return errors.New("no opcode recorded in packet")
	}
	opcode := body[0]
	body = body[1:]

	if opcode != OpcodeRequest {
		return fmt.Errorf("got unexpected opcode %d", opcode)
	}

	pkt, err := UnmarshalRequestPacket(body)
	if err != nil {
		return fmt.Errorf("failed to unmarshal request: %w", err)
	}

	ftx := acquireContext(ctx.Conn(), pkt.Data)
	defer releaseContext(ftx)

	for _, service := range pkt.Services {
		handler := p.node.getHandler(service)
		if handler == nil {
			continue
		}
		res := handler(ftx)
		if res != nil {
			return ctx.Reply(res)
		}
		return nil
	}

	return nil
}

func (p *Peer) HandleConnState(_ *monte.Conn, state monte.ConnState) {
	if state != monte.StateClosed || p.err != nil {
		return
	}
	p.node.deletePeer(p.addr)

	fmt.Printf("You have been disconnected from %s. Services: %s\n", p.addr, p.svcs)

	go func() {
		b := &backoff.Backoff{
			Factor: 1.25,
			Jitter: true,
			Min:    500 * time.Millisecond,
			Max:    1 * time.Second,
		}

		for {
			err := p.node.Dial(p.addr)
			if err == nil {
				break
			}

			duration := b.Duration()

			fmt.Printf("Trying to reconnect to %s. Sleeping for %s.\n", p.addr, duration)
			time.Sleep(duration)
		}
	}()
}

func (p *Peer) Dial() {
	defer close(p.ready)

	pkt := HandshakePacket{
		ID:       p.node.id,
		Services: p.node.Services(),
	}
	pkt.Signature = p.node.sec.Sign(pkt.AppendPayloadTo(nil))

	buf, err := p.client.Request(nil, pkt.AppendTo([]byte{OpcodeHandshake}))
	if err != nil {
		p.dispose(err)
		return
	}

	pkt, err = UnmarshalHandshakePacket(buf)
	if err != nil {
		p.dispose(err)
		return
	}

	pktAddr := Addr(pkt.ID.Host, pkt.ID.Port)
	if p.addr != pktAddr {
		p.disposef("peer advertises address '%s', but dialed via address '%s'", pktAddr, p.addr)
		return
	}

	err = pkt.Validate(nil)
	if err != nil {
		p.dispose(err)
		return
	}

	p.id = pkt.ID
	p.svcs = pkt.Services

	fmt.Printf("You are now connected to %s. Services: %s\n", p.addr, p.svcs)
}

func (p *Peer) dispose(err error) {
	p.node.deletePeer(p.addr)
	p.client.Shutdown()
	p.err = err
}

func (p *Peer) disposef(format string, args ...interface{}) {
	p.dispose(fmt.Errorf(format, args...))
}
