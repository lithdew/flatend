package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"io"
	"sync"
)

type Stream struct {
	ID     uint32
	Header *ServiceResponsePacket
	Reader *io.PipeReader
	Writer *io.PipeWriter

	received uint64
	wg       sync.WaitGroup
	once     sync.Once
}

type Provider struct {
	id       *kademlia.ID
	conn     *monte.Conn
	services map[string]struct{}

	lock    sync.Mutex         // protects all stream-related structures
	counter uint32             // total number of outgoing streams
	streams map[uint32]*Stream // maps stream ids to stream instances
}

func (p *Provider) NextStream() *Stream {
	reader, writer := io.Pipe()

	p.lock.Lock()
	defer p.lock.Unlock()

	id := p.counter
	p.counter += 2

	stream := &Stream{
		ID:     id,
		Reader: reader,
		Writer: writer,
	}
	stream.wg.Add(1)
	p.streams[id] = stream

	return stream
}

func (p *Provider) GetStream(id uint32) (*Stream, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	stream, exists := p.streams[id]
	return stream, exists
}

func (p *Provider) RegisterStream(header ServiceRequestPacket) (*Stream, bool) {
	reader, writer := io.Pipe()

	p.lock.Lock()
	defer p.lock.Unlock()

	stream, exists := p.streams[header.ID]
	if exists {
		return stream, false
	}

	stream = &Stream{
		ID:     header.ID,
		Reader: reader,
		Writer: writer,
	}
	stream.wg.Add(1)
	p.streams[header.ID] = stream

	return stream, true
}

func (p *Provider) CloseStreamWithError(stream *Stream, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	_ = stream.Reader.CloseWithError(err)
	_ = stream.Writer.CloseWithError(err)

	stream.once.Do(stream.wg.Done)

	delete(p.streams, stream.ID)
}

func (p *Provider) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for id, stream := range p.streams {
		err := fmt.Errorf("provider connection closed: %w", io.EOF)

		_ = stream.Reader.CloseWithError(err)
		_ = stream.Writer.CloseWithError(err)

		stream.once.Do(stream.wg.Done)

		delete(p.streams, id)
	}
}

var ErrProviderNotAvailable = errors.New("provider unable to provide service")

func (p *Provider) Push(services []string, headers map[string]string, body io.ReadCloser) (*Stream, error) {
	buf := make([]byte, ChunkSize)

	stream := p.NextStream()

	header := ServiceRequestPacket{
		ID:       stream.ID,
		Headers:  headers,
		Services: services,
	}

	err := p.conn.Send(header.AppendTo([]byte{OpcodeServiceRequest}))
	if err != nil {
		err = fmt.Errorf("failed to send stream header: %s: %w", err, ErrProviderNotAvailable)
		p.CloseStreamWithError(stream, err)
		return nil, err
	}

	for {
		nn, err := body.Read(buf[:ChunkSize])
		if err != nil && err != io.EOF {
			err = fmt.Errorf("failed reading body: %w", err)
			p.CloseStreamWithError(stream, err)
			return nil, err
		}

		payload := DataPacket{
			ID:   stream.ID,
			Data: buf[:nn],
		}

		if err := p.conn.Send(payload.AppendTo([]byte{OpcodeData})); err != nil {
			err = fmt.Errorf("failed writing body chunk as a data packet to peer: %w", err)
			p.CloseStreamWithError(stream, err)
			return nil, err
		}

		if err == io.EOF && nn == 0 {
			break
		}
	}

	stream.wg.Wait()

	if stream.Header == nil {
		return nil, fmt.Errorf("no response headers were returned: %w", io.EOF)
	}

	if !stream.Header.Handled {
		return nil, fmt.Errorf("provider unable to service: %s", services)
	}

	return stream, nil
}

func (p *Provider) Addr() string {
	if p.id != nil {
		return Addr(p.id.Host, p.id.Port)
	} else {
		return "???"
	}
}

func (p *Provider) Services() []string {
	services := make([]string, 0, len(p.services))
	for service := range p.services {
		services = append(services, service)
	}
	return services
}

type Providers struct {
	sync.Mutex

	services  map[string]map[*monte.Conn]struct{}
	providers map[*monte.Conn]*Provider
}

func NewProviders() *Providers {
	return &Providers{
		services:  make(map[string]map[*monte.Conn]struct{}),
		providers: make(map[*monte.Conn]*Provider),
	}
}

func (p *Providers) findProvider(conn *monte.Conn) *Provider {
	p.Lock()
	defer p.Unlock()
	return p.providers[conn]
}

func (p *Providers) getProviders(services ...string) []*Provider {
	p.Lock()
	defer p.Unlock()

	var conns []*monte.Conn

	for _, service := range services {
		for conn := range p.services[service] {
			conns = append(conns, conn)
		}
	}

	if conns == nil {
		return nil
	}

	providers := make([]*Provider, 0, len(conns))
	for _, conn := range conns {
		providers = append(providers, p.providers[conn])
	}

	return providers
}

func (p *Providers) register(conn *monte.Conn, id *kademlia.ID, services []string, outgoing bool) *Provider {
	p.Lock()
	defer p.Unlock()

	provider, exists := p.providers[conn]
	if !exists {
		provider = &Provider{
			services: make(map[string]struct{}),
			streams:  make(map[uint32]*Stream),
		}
		if outgoing {
			provider.counter = 1
		} else {
			provider.counter = 0
		}
		p.providers[conn] = provider
	}

	provider.id = id
	provider.conn = conn

	for _, service := range services {
		provider.services[service] = struct{}{}
		if _, exists := p.services[service]; !exists {
			p.services[service] = make(map[*monte.Conn]struct{})
		}
		p.services[service][conn] = struct{}{}
	}

	return provider
}

func (p *Providers) deregister(conn *monte.Conn) *Provider {
	p.Lock()
	defer p.Unlock()

	provider, exists := p.providers[conn]
	if !exists {
		return nil
	}

	delete(p.providers, conn)

	for service := range provider.services {
		delete(p.services[service], conn)
		if len(p.services[service]) == 0 {
			delete(p.services, service)
		}
	}

	provider.Close()

	return provider
}
