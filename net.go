package flatend

import (
	"github.com/lithdew/monte"
	"io"
	"net"
	"sync"
)

const ChunkSize = 2048

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

var _ io.Writer = (*Context)(nil)

type Context struct {
	Headers map[string]string
	Body    io.ReadCloser

	ID      uint32 // stream id
	Conn    *monte.Conn
	headers map[string]string // response headers
	written bool              // written before?
}

func (c *Context) WriteHeader(key, val string) {
	c.headers[key] = val
}

func (c *Context) Write(data []byte) (int, error) {
	if !c.written {
		packet := ServiceResponsePacket{
			ID:      c.ID,
			Handled: true,
			Headers: c.headers,
		}

		c.written = true

		err := c.Conn.Send(packet.AppendTo([]byte{OpcodeServiceResponse}))
		if err != nil {
			return 0, err
		}
	}

	if len(data) == 0 { // disallow writing zero bytes
		return 0, nil
	}

	for i := 0; i < len(data); i += ChunkSize {
		start := i
		end := i + ChunkSize
		if end > len(data) {
			end = len(data)
		}

		packet := DataPacket{
			ID:   c.ID,
			Data: data[start:end],
		}

		err := c.Conn.Send(packet.AppendTo([]byte{OpcodeData}))
		if err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

var contextPool sync.Pool

func acquireContext(headers map[string]string, body io.ReadCloser, id uint32, conn *monte.Conn) *Context {
	v := contextPool.Get()
	if v == nil {
		v = &Context{headers: make(map[string]string)}
	}
	ctx := v.(*Context)
	ctx.Headers = headers
	ctx.Body = body
	ctx.ID = id
	ctx.Conn = conn
	return ctx
}

func releaseContext(ctx *Context) {
	ctx.written = false
	for key := range ctx.headers {
		delete(ctx.headers, key)
	}
	contextPool.Put(ctx)
}

type Handler func(ctx *Context)
