package flatend

import (
	"errors"
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"io"
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
		n.SecretKey = GenerateSecretKey()
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

	if n.id != nil && len(n.BindAddrs) == 0 {
		ln, err := BindTCP(Addr(n.id.Host, n.id.Port))()
		if err != nil {
			return err
		}

		n.wg.Add(1)
		go func() {
			defer n.wg.Done()
			n.srv.Serve(ln)
		}()

		n.lns = append(n.lns, ln)
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

		provider := n.providers.register(ctx.Conn(), packet.ID, packet.Services, false)
		fmt.Printf("%s has connected to you. Services: %s\n", provider.Addr(), provider.Services())

		// always reply back with what services we provide, and our
		// id if we want to publicly advertise our microservice

		return ctx.Reply(n.createHandshakePacket(body[:0]).AppendTo(body[:0]))

	case OpcodeServiceRequest:
		provider := n.providers.findProvider(ctx.Conn())
		if provider == nil {
			return errors.New("conn is not a provider")
		}

		packet, err := UnmarshalServiceRequestPacket(body)
		if err != nil {
			return fmt.Errorf("failed to decode service request packet: %w", err)
		}

		// if stream does not exist, the stream is a request to process some payload! register
		// a new stream, and handle it.

		// if the stream exists, it's a stream we made! proceed to start receiving its payload,
		// b/c the payload will basically be our peer giving us a response.

		// i guess we need to separate streams from the server-side, and streams from the client-side.
		// client-side starts with stream ids that are odd (1, 3, 5, 7, 9, ...) and server-side
		// starts with stream ids that are even (0, 2, 4, 6, 8, 10).

		stream, created := provider.RegisterStream(packet)
		if !created {
			return fmt.Errorf("got service request with stream id %d, but node is making service request"+
				"with the given id already", packet.ID)
		}

		for _, service := range packet.Services {
			handler, exists := n.Services[service]
			if !exists {
				continue
			}

			go func() {
				ctx := acquireContext(packet.Headers, stream.Reader, stream.ID, ctx.Conn())
				defer releaseContext(ctx)

				handler(ctx)

				if !ctx.written {
					packet := ServiceResponsePacket{
						ID:      ctx.ID,
						Handled: true,
						Headers: ctx.headers,
					}

					ctx.written = true

					err := ctx.Conn.Send(packet.AppendTo([]byte{OpcodeServiceResponse}))
					if err != nil {
						provider.CloseStreamWithError(stream, err)
						return
					}

					return
				}

				err := ctx.Conn.Send(DataPacket{ID: stream.ID}.AppendTo([]byte{OpcodeData}))
				if err != nil {
					provider.CloseStreamWithError(stream, err)
					return
				}
			}()

			return nil
		}

		response := ServiceResponsePacket{
			ID:      packet.ID,
			Handled: false,
		}

		return ctx.Conn().SendNoWait(response.AppendTo([]byte{OpcodeServiceResponse}))
	case OpcodeServiceResponse:
		provider := n.providers.findProvider(ctx.Conn())
		if provider == nil {
			return errors.New("conn is not a provider")
		}

		packet, err := UnmarshalServiceResponsePacket(body)
		if err != nil {
			return fmt.Errorf("failed to decode services response packet: %w", err)
		}

		stream, exists := provider.GetStream(packet.ID)
		if !exists {
			return fmt.Errorf("stream with id %d got a service response although no service request was sent",
				packet.ID)
		}

		stream.Header = &packet
		stream.once.Do(stream.wg.Done)

		return nil
	case OpcodeData:
		provider := n.providers.findProvider(ctx.Conn())
		if provider == nil {
			return errors.New("conn is not a provider")
		}

		packet, err := UnmarshalDataPacket(body)
		if err != nil {
			return fmt.Errorf("failed to decode stream payload packet: %w", err)
		}

		// the stream must always exist

		stream, exists := provider.GetStream(packet.ID)
		if !exists {
			return fmt.Errorf("stream with id %d does not exist", packet.ID)
		}

		// there should never be any empty payload packets

		if len(packet.Data) > ChunkSize {
			err = fmt.Errorf("stream with id %d got packet over max limit of ChunkSize bytes: got %d bytes",
				packet.ID, len(packet.Data))
			provider.CloseStreamWithError(stream, err)
			return err
		}

		if stream.ID%2 == 1 && stream.Header == nil {
			err = fmt.Errorf("outgoing stream with id %d received a payload packet but has not received a header yet",
				packet.ID)
			provider.CloseStreamWithError(stream, err)
			return err
		}

		// if the chunk is zero-length, the stream has been closed

		if len(packet.Data) == 0 {
			provider.CloseStreamWithError(stream, io.EOF)
		} else {
			_, err = stream.Writer.Write(packet.Data)
			if err != nil && !errors.Is(err, io.ErrClosedPipe) {
				err = fmt.Errorf("failed to write payload: %w", err)
				provider.CloseStreamWithError(stream, err)
				return err
			}
		}

		return nil
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

	// pre-register the provider
	n.providers.register(conn, nil, nil, true)

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

		// update the provider with id and services info
		n.providers.register(conn, packet.ID, packet.Services, true)
	}

	fmt.Printf("You are now connected to %s. Services: %s\n", addr, packet.Services)

	return nil
}

func (n *Node) Push(services []string, headers map[string]string, body io.ReadCloser) (*Stream, error) {
	providers := n.providers.getProviders(services...)

	for _, provider := range providers {
		stream, err := provider.Push(services, headers, body)
		if err != nil {
			if errors.Is(err, ErrProviderNotAvailable) {
				continue
			}
			return nil, err
		}
		return stream, nil
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
