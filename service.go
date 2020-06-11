package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/kademlia"
	"net"
	"os"
	"os/signal"
)

type Service struct {
	Addr     string
	BindAddr string

	PrivateKey kademlia.PrivateKey
	Services   map[string]Handler
}

func (s *Service) Register(service string, handler Handler) {
	if s.Services == nil {
		s.Services = make(map[string]Handler, 1)
	}
	s.Services[service] = handler
}

func (s Service) Start() error {
	addr := s.Addr
	if addr == "" {
		return errors.New("address to flatend hub must be provided")
	}

	bindAddr := s.BindAddr
	if bindAddr == "" {
		bindAddr = ":0"
	}

	var err error

	privateKey := s.PrivateKey
	if privateKey == kademlia.ZeroPrivateKey {
		_, privateKey, err = kademlia.GenerateKeys(nil)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}
	}

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on bind addr '%s': %w", bindAddr, err)
	}

	node, err := NewNode(privateKey, ln.Addr().String())
	if err != nil {
		return err
	}

	for service, handler := range s.Services {
		node.Register(service, handler)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- node.Serve(ln)
	}()

	if err := node.Dial(addr); err != nil {
		return fmt.Errorf("failed to reach flatend hub: %w", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
	ln.Close()

	return <-errCh
}
