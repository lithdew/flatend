package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/bytesutil"
	"github.com/lithdew/kademlia"
	"github.com/lithdew/monte"
	"io"
	"math"
	"net"
	"strconv"
	"sync"
	"unicode/utf8"
	"unsafe"
)

type Context struct {
	conn *monte.Conn
	body []byte
}

func (c *Context) Conn() *monte.Conn { return c.conn }
func (c *Context) Body() []byte      { return c.body }

var contextPool sync.Pool

func acquireContext(conn *monte.Conn, body []byte) *Context {
	v := contextPool.Get()
	if v == nil {
		v = &Context{}
	}
	ctx := v.(*Context)
	ctx.conn = conn
	ctx.body = body
	return ctx
}

func releaseContext(ctx *Context) { contextPool.Put(ctx) }

type Handler func(ctx *Context) []byte

var DefaultHandler Handler = func(ctx *Context) []byte { return nil }

type Opcode = uint8

const (
	OpcodeHandshake Opcode = iota
	OpcodeRequest
)

type HandshakePacket struct {
	ID        kademlia.ID
	Services  []string
	Signature kademlia.Signature
}

func (h HandshakePacket) AppendPayloadTo(dst []byte) []byte {
	dst = h.ID.AppendTo(dst)
	for _, service := range h.Services {
		dst = append(dst, service...)
	}
	return dst
}

func (h HandshakePacket) AppendTo(dst []byte) []byte {
	dst = h.ID.AppendTo(dst)
	dst = append(dst, uint8(len(h.Services)))
	for _, service := range h.Services {
		dst = append(dst, uint8(len(service)))
		dst = append(dst, service...)
	}
	dst = append(dst, h.Signature[:]...)
	return dst
}

func UnmarshalHandshakePacket(buf []byte) (HandshakePacket, error) {
	var pkt HandshakePacket
	id, buf, err := kademlia.UnmarshalID(buf)
	if err != nil {
		return pkt, err
	}
	pkt.ID = id

	if len(buf) < 1 {
		return pkt, io.ErrUnexpectedEOF
	}

	var size uint8
	size, buf = buf[0], buf[1:]

	if len(buf) < int(size) {
		return pkt, io.ErrUnexpectedEOF
	}

	pkt.Services = make([]string, size)
	for i := 0; i < len(pkt.Services); i++ {
		if len(buf) < 1 {
			return pkt, io.ErrUnexpectedEOF
		}
		size, buf = buf[0], buf[1:]
		if len(buf) < int(size) {
			return pkt, io.ErrUnexpectedEOF
		}
		pkt.Services[i] = string(buf[:size])
		buf = buf[size:]
	}

	if len(buf) < kademlia.SizeSignature {
		return pkt, io.ErrUnexpectedEOF
	}

	pkt.Signature, buf = *(*kademlia.Signature)(unsafe.Pointer(&((buf[:kademlia.SizeSignature])[0]))),
		buf[kademlia.SizeSignature:]

	return pkt, nil
}

func (h HandshakePacket) Validate(dst []byte) error {
	if err := h.ID.Validate(); err != nil {
		return err
	}

	for _, service := range h.Services {
		if !utf8.ValidString(service) {
			return fmt.Errorf("service '%s' in hello packet is not valid utf8", service)
		}
		if len(service) > math.MaxUint8 {
			return fmt.Errorf("service '%s' in hello packet is too large - must <= %d bytes",
				service, math.MaxUint8)
		}
	}

	if !h.Signature.Verify(h.ID.Pub, h.AppendPayloadTo(dst)) {
		return errors.New("signature is malformed")
	}

	return nil
}

type RequestPacket struct {
	Services []string
	Data     []byte
}

func (r RequestPacket) AppendTo(dst []byte) []byte {
	dst = append(dst, uint8(len(r.Services)))
	for _, service := range r.Services {
		dst = append(dst, uint8(len(service)))
		dst = append(dst, service...)
	}
	dst = bytesutil.AppendUint32BE(dst, uint32(len(r.Data)))
	dst = append(dst, r.Data...)
	return dst
}

func UnmarshalRequestPacket(buf []byte) (RequestPacket, error) {
	var pkt RequestPacket

	if len(buf) < 1 {
		return pkt, io.ErrUnexpectedEOF
	}

	var size uint8
	size, buf = buf[0], buf[1:]

	if len(buf) < int(size) {
		return pkt, io.ErrUnexpectedEOF
	}

	pkt.Services = make([]string, size)
	for i := 0; i < len(pkt.Services); i++ {
		if len(buf) < 1 {
			return pkt, io.ErrUnexpectedEOF
		}
		size, buf = buf[0], buf[1:]
		if len(buf) < int(size) {
			return pkt, io.ErrUnexpectedEOF
		}
		pkt.Services[i] = string(buf[:size])
		buf = buf[size:]
	}

	if len(buf) < 4 {
		return pkt, io.ErrUnexpectedEOF
	}

	var length uint32
	length, buf = bytesutil.Uint32BE(buf[:4]), buf[4:]

	if uint32(len(buf)) < length {
		return pkt, io.ErrUnexpectedEOF
	}

	pkt.Data = buf[:length]
	return pkt, nil
}

func Addr(host net.IP, port uint16) string {
	h := ""
	if len(host) > 0 {
		h = host.String()
	}
	p := strconv.FormatUint(uint64(port), 10)
	return net.JoinHostPort(h, p)
}
