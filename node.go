package flatend

import (
	"errors"
	"fmt"
	"github.com/jpillora/backoff"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"io"
	"log"
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

	tableLock sync.Mutex
	table     *kademlia.Table

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
	var (
		bindHost net.IP
		bindPort uint16
	)

	//if n.SecretKey == kademlia.ZeroPrivateKey {
	//	n.Secre
	//}

	if n.SecretKey != kademlia.ZeroPrivateKey {
		if n.PublicAddr != "" { // resolve the address
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
		} else { // get a random public address
			ln, err := net.Listen("tcp", ":0")
			if err != nil {
				return fmt.Errorf("unable to listen on any port: %w", err)
			}
			bindAddr := ln.Addr().(*net.TCPAddr)
			bindHost = bindAddr.IP
			bindPort = uint16(bindAddr.Port)
			if err := ln.Close(); err != nil {
				return fmt.Errorf("failed to close listener for getting avaialble port: %w", err)
			}
		}
	}

	start := false
	n.start.Do(func() { start = true })
	if !start {
		return errors.New("listener already started")
	}

	if n.SecretKey != kademlia.ZeroPrivateKey {
		n.id = &kademlia.ID{
			Pub:  n.SecretKey.Public(),
			Host: bindHost,
			Port: bindPort,
		}

		n.table = kademlia.NewTable(n.id.Pub)
	} else {
		n.table = kademlia.NewTable(kademlia.ZeroPublicKey)
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

		log.Printf("Listening for Flatend nodes on '%s'.", ln.Addr().String())

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

		log.Printf("Listening for Flatend nodes on '%s'.", ln.Addr().String())

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

	n.Bootstrap()

	return nil
}

func (n *Node) Bootstrap() {
	var mu sync.Mutex

	var pub kademlia.PublicKey
	if n.id != nil {
		pub = n.id.Pub
	}

	busy := make(chan struct{}, kademlia.DefaultBucketSize)
	queue := make(chan kademlia.ID, kademlia.DefaultBucketSize)

	visited := make(map[kademlia.PublicKey]struct{})

	n.tableLock.Lock()
	ids := n.table.ClosestTo(pub, kademlia.DefaultBucketSize)
	n.tableLock.Unlock()

	if len(ids) == 0 {
		return
	}

	visited[pub] = struct{}{}
	for _, id := range ids {
		queue <- id
		visited[id.Pub] = struct{}{}
	}

	var results []kademlia.ID

	for len(queue) > 0 || len(busy) > 0 {
		select {
		case id := <-queue:
			busy <- struct{}{}
			go func() {
				defer func() { <-busy }()

				client := n.getClient(Addr(id.Host, id.Port))

				conn, err := client.Get()
				if err != nil {
					return
				}

				err = n.probe(conn)
				if err != nil {
					return
				}

				req := n.createFindNodeRequest().AppendTo([]byte{OpcodeFindNodeRequest})

				buf, err := conn.Request(req[:0], req)
				if err != nil {
					return
				}

				res, _, err := kademlia.UnmarshalFindNodeResponse(buf)
				if err != nil {
					return
				}

				mu.Lock()
				for _, id := range res.Closest {
					if _, seen := visited[id.Pub]; !seen {
						visited[id.Pub] = struct{}{}
						results = append(results, id)
						queue <- id
					}
				}
				mu.Unlock()
			}()
		default:
			time.Sleep(16 * time.Millisecond)
		}
	}

	log.Printf("Discovered %d peer(s).", len(results))
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

	n.tableLock.Lock()
	if provider.id != nil {
		n.table.Delete(provider.id.Pub)
	}
	n.tableLock.Unlock()

	addr := provider.Addr()

	log.Printf("%s has disconnected from you. Services: %s", addr, provider.Services())

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

		for i := 0; i < 8; i++ { // 8 attempts max
			err := n.Probe(addr)
			if err == nil {
				break
			}

			duration := b.Duration()

			log.Printf("Trying to reconnect to %s. Sleeping for %s.", addr, duration)
			time.Sleep(duration)
		}

		log.Printf("Tried 8 times reconnecting to %s. Giving up.", addr)
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

		provider, exists := n.providers.register(ctx.Conn(), packet.ID, packet.Services, false)
		if !exists {
			log.Printf("%s has connected. Services: %s", provider.Addr(), provider.Services())
		}

		// register the peer to the routing table

		if packet.ID != nil {
			n.tableLock.Lock()
			n.table.Update(*packet.ID)
			n.tableLock.Unlock()
		}

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
	case OpcodeFindNodeRequest:
		packet, _, err := kademlia.UnmarshalFindNodeRequest(body)
		if err != nil {
			return err
		}

		n.tableLock.Lock()
		res := kademlia.FindNodeResponse{Closest: n.table.ClosestTo(packet.Target, kademlia.DefaultBucketSize)}
		n.tableLock.Unlock()

		return ctx.Reply(res.AppendTo(nil))
	}

	return fmt.Errorf("unknown opcode %d", opcode)
}

func (n *Node) getClient(addr string) *monte.Client {
	n.clientsLock.Lock()
	defer n.clientsLock.Unlock()

	client, exists := n.clients[addr]
	if !exists {
		client = &monte.Client{
			Addr:      addr,
			Handler:   n,
			ConnState: n,
		}
		n.clients[addr] = client
	}

	return client
}

func (n *Node) probe(conn *monte.Conn) error {
	provider, exists := n.providers.register(conn, nil, nil, true)

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

	if packet.ID != nil {
		//addr = Addr(packet.ID.Host, packet.ID.Port)
		//if !packet.ID.Host.Equal(resolved.IP) || packet.ID.Port != uint16(resolved.Port) {
		//	return provider, fmt.Errorf("dialed '%s' which advertised '%s'", resolved, addr)
		//}

		// update the provider with id and services info
		provider, _ = n.providers.register(conn, packet.ID, packet.Services, true)
		if !exists {
			log.Printf("You are now connected to %s. Services: %s", provider.Addr(), provider.Services())
		} else {
			log.Printf("Re-probed %s. Services: %s", provider.Addr(), provider.Services())
		}

		// update the routing table
		n.tableLock.Lock()
		n.table.Update(*packet.ID)
		n.tableLock.Unlock()
	}

	return nil
}

func (n *Node) Probe(addr string) error {
	resolved, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	conn, err := n.getClient(resolved.String()).Get()
	if err != nil {
		return err
	}

	err = n.probe(conn)
	if err != nil {
		return err
	}

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

func (n *Node) createFindNodeRequest() kademlia.FindNodeRequest {
	var req kademlia.FindNodeRequest
	if n.id != nil {
		req.Target = n.id.Pub
	}
	return req
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
