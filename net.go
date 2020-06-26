package flatend

import (
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"io"
	"sync"
)

const ChunkSize = 2048

var _ io.Writer = (*Context)(nil)

type Context struct {
	ID      kademlia.ID
	Headers map[string]string
	Body    io.ReadCloser

	nonce   uint32 // stream id
	conn    *monte.Conn
	headers map[string]string // response headers
	written bool              // written before?
}

func (c *Context) WriteHeader(key, val string) {
	c.headers[key] = val
}

func (c *Context) Write(data []byte) (int, error) {
	if len(data) == 0 { // disallow writing zero bytes
		return 0, nil
	}

	if !c.written {
		packet := ServiceResponsePacket{
			ID:      c.nonce,
			Handled: true,
			Headers: c.headers,
		}

		c.written = true

		err := c.conn.Send(packet.AppendTo([]byte{OpcodeServiceResponse}))
		if err != nil {
			return 0, err
		}
	}

	for i := 0; i < len(data); i += ChunkSize {
		start := i
		end := i + ChunkSize
		if end > len(data) {
			end = len(data)
		}

		packet := DataPacket{
			ID:   c.nonce,
			Data: data[start:end],
		}

		err := c.conn.Send(packet.AppendTo([]byte{OpcodeData}))
		if err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

var contextPool sync.Pool

func acquireContext(id kademlia.ID, headers map[string]string, body io.ReadCloser, nonce uint32, conn *monte.Conn) *Context {
	v := contextPool.Get()
	if v == nil {
		v = &Context{headers: make(map[string]string)}
	}
	ctx := v.(*Context)

	ctx.ID = id
	ctx.Headers = headers
	ctx.Body = body

	ctx.nonce = nonce
	ctx.conn = conn

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
