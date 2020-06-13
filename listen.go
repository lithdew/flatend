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

type BindFunc func() (net.Listener, error)

func BindTCP(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp", addr) }
}

func BindTCPv4(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp4", addr) }
}

func BindTCPv6(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("tcp6", addr) }
}

func BindUnix(addr string) BindFunc {
	return func() (net.Listener, error) { return net.Listen("unix", addr) }
}

var _ monte.Handler = (*Listener)(nil)
var _ monte.ConnStateHandler = (*Listener)(nil)

type Listener struct {
	PublicAddr string
	BindAddrs  []BindFunc

	SecretKey kademlia.PrivateKey
	Services  map[string]Handler

	once sync.Once
	id   *kademlia.ID
	srv  *monte.Server
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
		host, port, err := net.SplitHostPort(l.PublicAddr)
		if err != nil {
			return err
		}

		bindHost = net.ParseIP(host)
		if bindHost == nil {
			return fmt.Errorf("'%s' is an invalid host: it must be an IPv4 or IPv6 address", host)
		}

		decodedPort, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return fmt.Errorf("'%s' is an invalid port: %w", port, err)
		}

		bindPort = uint16(decodedPort)
	}

	once := false

	l.once.Do(func() { once = true })

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

	l.srv = &monte.Server{
		Handler:            l,
		ConnState:          nil,
		Handshaker:         nil,
		HandshakeTimeout:   0,
		MaxConns:           0,
		MaxConnWaitTimeout: 0,
		ReadBufferSize:     0,
		WriteBufferSize:    0,
		ReadTimeout:        0,
		WriteTimeout:       0,
		SeqOffset:          0,
		SeqDelta:           0,
	}

	return nil
}

func (l *Listener) HandleConnState(conn *monte.Conn, state monte.ConnState) {
	panic("implement me")
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
	case OpcodeRequest:
	}

	return fmt.Errorf("unknown opcode %d", opcode)
}
