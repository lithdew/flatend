package flathttp

import (
	"fmt"
	"net/url"
	"time"
)

type Addr struct {
	Addr   string
	Scheme string
	Host   string
}

type Config struct {
	Addrs []string

	MaxHeaderBytes    int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration

	addrs []Addr
}

func (c *Config) Parse() error {
	for _, addr := range c.Addrs {
		u, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("flathttp: invalid addr provided in config %q", u)
		}
		switch u.Scheme {
		case "tcp", "tcp4", "tcp6", "unix", "unixpacket":
		case "http", "https":
			u.Scheme = "tcp"
		default:
			return fmt.Errorf("flathttp: invalid scheme %q in addr %q (must be http, https, tcp, tcp4, tcp6, unix, unixpacket)", u.Scheme, addr)
		}
		c.addrs = append(c.addrs, Addr{Addr: addr, Scheme: u.Scheme, Host: u.Host})
	}
	return nil
}

func (c *Config) reset() {
	c.addrs = c.addrs[:0]
}
