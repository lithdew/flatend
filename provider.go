package flatend

import (
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"sync"
)

type Provider struct {
	id       kademlia.ID
	conn     *monte.Conn
	services map[string]struct{}
}

func (p *Provider) Addr() string { return Addr(p.id.Host, p.id.Port) }

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

	provider, exists := p.providers[conn]
	if !exists {
		return nil
	}
	return provider
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

func (p *Providers) register(conn *monte.Conn, id kademlia.ID, services []string) *Provider {
	p.Lock()
	defer p.Unlock()

	provider, exists := p.providers[conn]
	if !exists {
		provider = &Provider{id: id, conn: conn, services: make(map[string]struct{})}
		p.providers[conn] = provider
	}

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

	return provider
}
