package flatend

import (
	"errors"
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"math"
	"net"
	"sync"
	"time"
)

var _ monte.Handler = (*Node)(nil)
var _ monte.ConnStateHandler = (*Node)(nil)

type Node struct {
	PublicAddr string
	SecretKey  kademlia.PrivateKey

	BindAddrs []BindFunc
	Services  map[string]Handler

	start sync.Once
	stop  sync.Once

	wg sync.WaitGroup
	id *kademlia.ID

	providers *Providers

	clientsLock sync.Mutex
	clients     map[string]*monte.Client

	lns []net.Listener
	srv *monte.Server
}

func GenerateSecretKey() kademlia.PrivateKey {
	_, secret, err := kademlia.GenerateKeys(nil)
	if err != nil {
		panic(err)
	}
	return secret
}

func (n *Node) Start(addrs ...string) error {
	if n.PublicAddr != "" && n.SecretKey == kademlia.ZeroPrivateKey {
		return errors.New("secret key must be provided if microservice has a public address to advertise")
	}

	var (
		bindHost net.IP
		bindPort uint16
	)

	if n.PublicAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", n.PublicAddr)
		if err != nil {
			return err
		}

		bindHost = addr.IP
		if bindHost == nil {
			return fmt.Errorf("'%s' is an invalid host: it must be an IPv4 or IPv6 address", addr.IP)
		}

		if addr.Port <= 0 || addr.Port >= math.MaxUint16 {
			return fmt.Errorf("'%d' is an invalid port", addr.Port)
		}

		bindPort = uint16(addr.Port)
	}

	start := false
	n.start.Do(func() { start = true })
	if !start {
		return errors.New("listener already started")
	}

	if n.PublicAddr != "" {
		n.id = &kademlia.ID{
			Pub:  n.SecretKey.Public(),
			Host: bindHost,
			Port: bindPort,
		}
	}

	n.providers = NewProviders()
	n.clients = make(map[string]*monte.Client)

	n.srv = &monte.Server{
		Handler:   n,
		ConnState: n,
	}

	for _, fn := range n.BindAddrs {
		ln, err := fn()
		if err != nil {
			for _, ln := range n.lns {
				ln.Close()
			}
			return err
		}

		n.wg.Add(1)
		go func() {
			defer n.wg.Done()
			n.srv.Serve(ln)
		}()

		n.lns = append(n.lns, ln)
	}

	for _, addr := range addrs {
		err := n.Probe(addr)
		if err != nil {
			return fmt.Errorf("failed to probe '%s': %w", addr, err)
		}
	}

	return nil
}

func (n *Node) Shutdown() {
	once := false
	n.start.Do(func() { once = true })
	if once {
		return
	}

	stop := false
	n.stop.Do(func() { stop = true })
	if !stop {
		return
	}

	n.srv.Shutdown()
	for _, ln := range n.lns {
		ln.Close()
	}
	n.wg.Wait()
}

func (n *Node) HandleConnState(conn *monte.Conn, state monte.ConnState) {
	if state != monte.StateClosed {
		return
	}

	provider := n.providers.deregister(conn)
	if provider == nil {
		return
	}

	addr := provider.Addr()

	fmt.Printf("%s has disconnected from you. Services: %s\n", addr, provider.Services())

	n.clientsLock.Lock()
	_, exists := n.clients[addr]
	n.clientsLock.Unlock()

	if !exists {
		return
	}

	go func() {
		b := &backoff.Backoff{
			Factor: 1.25,
			Jitter: true,
			Min:    500 * time.Millisecond,
			Max:    1 * time.Second,
		}

		for {
			err := n.Probe(addr)
			if err == nil {
				break
			}

			duration := b.Duration()

			fmt.Printf("Trying to reconnect to %s. Sleeping for %s.\n", addr, duration)
			time.Sleep(duration)
		}
	}()
}

func (n *Node) HandleMessage(ctx *monte.Context) error {
	var opcode uint8

	body := ctx.Body()
	if len(body) < 1 {
		return errors.New("no opcode recorded in packet")
	}
	opcode, body = body[0], body[1:]

	switch opcode {
	case OpcodeHandshake:
		packet, err := UnmarshalHandshakePacket(body)
		if err != nil {
			return err
		}

		err = packet.Validate(body[:0])
		if err != nil {
			return err
		}

		// register the microservice if it hasn't been registered before

		provider := n.providers.register(ctx.Conn(), packet.ID, packet.Services)
		fmt.Printf("%s has connected to you. Services: %s\n", provider.Addr(), provider.Services())

		// always reply back with what services we provide, and our
		// id if we want to publicly advertise our microservice

		return ctx.Reply(n.createHandshakePacket(body[:0]).AppendTo(body[:0]))
	case OpcodeRequest:
		packet, err := UnmarshalRequestPacket(body)
		if err != nil {
			return err
		}

		ftx := acquireContext(ctx.Conn(), packet.Data)
		defer releaseContext(ftx)

		for _, service := range packet.Services {
			handler, exists := n.Services[service]
			if !exists {
				continue
			}

			return ctx.Reply(ResponsePacket(handler(ftx)).AppendTo(body[:0]))
		}

		return ctx.Reply(ResponsePacket(nil).AppendTo(body[:0]))
	}

	return fmt.Errorf("unknown opcode %d", opcode)
}

func (n *Node) Probe(addr string) error {
	resolved, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	n.clientsLock.Lock()
	client, exists := n.clients[resolved.String()]
	if !exists {
		client = &monte.Client{
			Addr:      resolved.String(),
			Handler:   n,
			ConnState: n,
		}
		n.clients[resolved.String()] = client
	}
	n.clientsLock.Unlock()

	conn, err := client.Get()
	if err != nil {
		return err
	}
	req := n.createHandshakePacket(nil).AppendTo([]byte{OpcodeHandshake})
	res, err := conn.Request(req[:0], req)
	if err != nil {
		return err
	}
	packet, err := UnmarshalHandshakePacket(res)
	if err != nil {
		return err
	}
	err = packet.Validate(req[:0])
	if err != nil {
		return err
	}

	addr = "???"
	if packet.ID != nil {
		addr = Addr(packet.ID.Host, packet.ID.Port)
		if !packet.ID.Host.Equal(resolved.IP) || packet.ID.Port != uint16(resolved.Port) {
			return fmt.Errorf("dialed '%s' which advertised '%s'", resolved, addr)
		}
		n.providers.register(conn, packet.ID, packet.Services)
	}

	fmt.Printf("You are now connected to %s. Services: %s\n", addr, packet.Services)

	return nil
}

func (n *Node) Request(services []string, data []byte) ([]byte, error) {
	providers := n.providers.getProviders(services...)

	packet := RequestPacket{
		Services: services,
		Data:     data,
	}

	req := packet.AppendTo([]byte{OpcodeRequest})

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
	}

	return nil, fmt.Errorf("no nodes were able to process your request for service(s): %s", services)
}

func (n *Node) createHandshakePacket(buf []byte) HandshakePacket {
	packet := HandshakePacket{Services: n.getServiceNames()}
	if n.id != nil {
		packet.ID = n.id
		packet.Signature = n.SecretKey.Sign(packet.AppendPayloadTo(buf))
	}
	return packet
}

func (n *Node) getServiceNames() []string {
	if n.Services == nil {
		return nil
	}
	services := make([]string, 0, len(n.Services))
	for service := range n.Services {
		services = append(services, service)
	}
	return services
}
