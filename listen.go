package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"math"
	"net"
	"sync"
)

type BindFunc func() (net.Listener, error)

func BindAny() BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp", ":0") }
}

func BindTCP(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp", addr) }
}

func BindTCPv4(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp4", addr) }
}

func BindTCPv6(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp6", addr) }
}

var _ monte.Handler = (*Listener)(nil)
var _ monte.ConnStateHandler = (*Listener)(nil)

type Listener struct {
	PublicAddr string
	BindAddrs  []BindFunc

	SecretKey kademlia.PrivateKey
	Services  map[string]Handler

	start sync.Once
	stop  sync.Once

	wg sync.WaitGroup
	id *kademlia.ID

	providers *Providers

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

func (l *Listener) Start() error {
	if l.PublicAddr != "" && l.SecretKey == kademlia.ZeroPrivateKey {
		return errors.New("secret key must be provided if microservice has a public address to advertise")
	}

	var (
		bindHost net.IP
		bindPort uint16
	)

	if l.PublicAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", l.PublicAddr)
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

	once := false
	l.start.Do(func() { once = true })
	if !once {
		return errors.New("listener already started")
	}

	if l.PublicAddr != "" {
		l.id = &kademlia.ID{
			Pub:  l.SecretKey.Public(),
			Host: bindHost,
			Port: bindPort,
		}
	}

	l.providers = NewProviders()

	l.srv = &monte.Server{
		Handler:   l,
		ConnState: l,
	}

	for _, fn := range l.BindAddrs {
		ln, err := fn()
		if err != nil {
			for _, ln := range l.lns {
				ln.Close()
			}
			return err
		}

		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.srv.Serve(ln)
		}()

		l.lns = append(l.lns, ln)
	}

	return nil
}

func (l *Listener) Shutdown() {
	once := false
	l.start.Do(func() { once = true })
	if once {
		return
	}

	stop := false
	l.stop.Do(func() { stop = true })
	if !stop {
		return
	}

	l.srv.Shutdown()
	for _, ln := range l.lns {
		ln.Close()
	}
	l.wg.Wait()
}

func (l *Listener) HandleConnState(conn *monte.Conn, state monte.ConnState) {
	if state != monte.StateClosed {
		return
	}

	provider := l.providers.deregister(conn)
	if provider == nil {
		return
	}

	fmt.Printf("%s has disconnected from you. Services: %s\n", provider.Addr(), provider.Services())
}

func (l *Listener) HandleMessage(ctx *monte.Context) error {
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

		provider := l.providers.register(ctx.Conn(), packet.ID, packet.Services)
		fmt.Printf("%s has connected to you. Services: %s\n", provider.Addr(), provider.Services())

		// always reply back with what services we provide, and our
		// id if we want to publicly advertise our microservice

		packet = HandshakePacket{Services: l.getServiceNames()}
		if l.id != nil {
			packet.ID = l.id
			packet.Signature = l.SecretKey.Sign(packet.AppendPayloadTo(body[:0]))
		}

		return ctx.Reply(packet.AppendTo(body[:0]))
	case OpcodeRequest:
		packet, err := UnmarshalRequestPacket(body)
		if err != nil {
			return err
		}

		ftx := acquireContext(ctx.Conn(), packet.Data)
		defer releaseContext(ftx)

		for _, service := range packet.Services {
			handler, exists := l.Services[service]
			if !exists {
				continue
			}

			return ctx.Reply(ResponsePacket(handler(ftx)).AppendTo(body[:0]))
		}

		return ctx.Reply(ResponsePacket(nil).AppendTo(body[:0]))
	}

	return fmt.Errorf("unknown opcode %d", opcode)
}

func (l *Listener) getServiceNames() []string {
	if l.Services == nil {
		return nil
	}
	services := make([]string, 0, len(l.Services))
	for service := range l.Services {
		services = append(services, service)
	}
	return services
}
