package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/bytesutil"
	"github.com/lithdew/kademlia"
	"io"
	"math"
	"net"
	"strconv"
	"unicode/utf8"
	"unsafe"
)

type Opcode = uint8

const (
	OpcodeHandshake Opcode = iota
	OpcodeServiceRequest
	OpcodeServiceResponse
	OpcodeData
)

type ServiceRequestPacket struct {
	ID       uint32            // stream id
	Services []string          // services this packet may be processed through
	Headers  map[string]string // headers for this packet
}

func (p ServiceRequestPacket) AppendTo(dst []byte) []byte {
	dst = bytesutil.AppendUint32BE(dst, p.ID)

	dst = append(dst, uint8(len(p.Services)))
	for _, service := range p.Services {
		dst = append(dst, uint8(len(service)))
		dst = append(dst, service...)
	}

	if p.Headers != nil {
		dst = bytesutil.AppendUint16BE(dst, uint16(len(p.Headers)))
		for name, value := range p.Headers {
			dst = append(dst, byte(len(name)))
			dst = append(dst, name...)
			dst = bytesutil.AppendUint16BE(dst, uint16(len(value)))
			dst = append(dst, value...)
		}
	} else {
		dst = bytesutil.AppendUint16BE(dst, 0)
	}

	return dst
}

func UnmarshalServiceRequestPacket(buf []byte) (ServiceRequestPacket, error) {
	var packet ServiceRequestPacket

	{
		if len(buf) < 4 {
			return packet, io.ErrUnexpectedEOF
		}

		packet.ID, buf = bytesutil.Uint32BE(buf[:4]), buf[4:]
	}

	{
		var size uint8
		size, buf = buf[0], buf[1:]

		packet.Services = make([]string, size)

		for i := 0; i < len(packet.Services); i++ {
			if len(buf) < 1 {
				return packet, io.ErrUnexpectedEOF
			}
			size, buf = buf[0], buf[1:]
			if len(buf) < int(size) {
				return packet, io.ErrUnexpectedEOF
			}
			packet.Services[i] = string(buf[:size])
			buf = buf[size:]
		}
	}

	{
		if len(buf) < 2 {
			return packet, io.ErrUnexpectedEOF
		}

		var size uint16
		size, buf = bytesutil.Uint16BE(buf[:2]), buf[2:]

		packet.Headers = make(map[string]string, size)
		for i := uint16(0); i < size; i++ {
			{
				if len(buf) < 1 {
					return packet, io.ErrUnexpectedEOF
				}
				var nameSize uint8
				nameSize, buf = buf[0], buf[1:]
				if len(buf) < int(nameSize) {
					return packet, io.ErrUnexpectedEOF
				}
				var name string
				name, buf = string(buf[:nameSize]), buf[nameSize:]

				if len(buf) < 2 {
					return packet, io.ErrUnexpectedEOF
				}
				var valueSize uint16
				valueSize, buf = bytesutil.Uint16BE(buf[:2]), buf[2:]
				if len(buf) < int(valueSize) {
					return packet, io.ErrUnexpectedEOF
				}
				var value string
				value, buf = string(buf[:valueSize]), buf[valueSize:]
				packet.Headers[name] = value
			}
		}
	}

	return packet, nil
}

type ServiceResponsePacket struct {
	ID      uint32            // stream id
	Handled bool              // whether or not the service was handled
	Headers map[string]string // headers for this packet
}

func (p ServiceResponsePacket) AppendTo(dst []byte) []byte {
	dst = bytesutil.AppendUint32BE(dst, p.ID)
	if p.Handled {
		dst = append(dst, 1)
	} else {
		dst = append(dst, 0)
	}
	if p.Headers != nil {
		dst = bytesutil.AppendUint16BE(dst, uint16(len(p.Headers)))
		for name, value := range p.Headers {
			dst = append(dst, byte(len(name)))
			dst = append(dst, name...)
			dst = bytesutil.AppendUint16BE(dst, uint16(len(value)))
			dst = append(dst, value...)
		}
	} else {
		dst = bytesutil.AppendUint16BE(dst, 0)
	}
	return dst
}

func UnmarshalServiceResponsePacket(buf []byte) (ServiceResponsePacket, error) {
	var packet ServiceResponsePacket

	{
		if len(buf) < 5 {
			return packet, io.ErrUnexpectedEOF
		}

		packet.ID, buf = bytesutil.Uint32BE(buf[:4]), buf[4:]
		packet.Handled, buf = buf[0] == 1, buf[1:]
	}

	{
		if len(buf) < 2 {
			return packet, io.ErrUnexpectedEOF
		}

		var size uint16
		size, buf = bytesutil.Uint16BE(buf[:2]), buf[2:]

		packet.Headers = make(map[string]string, size)
		for i := uint16(0); i < size; i++ {
			{
				if len(buf) < 1 {
					return packet, io.ErrUnexpectedEOF
				}
				var nameSize uint8
				nameSize, buf = buf[0], buf[1:]
				if len(buf) < int(nameSize) {
					return packet, io.ErrUnexpectedEOF
				}
				var name string
				name, buf = string(buf[:nameSize]), buf[nameSize:]

				if len(buf) < 2 {
					return packet, io.ErrUnexpectedEOF
				}
				var valueSize uint16
				valueSize, buf = bytesutil.Uint16BE(buf[:2]), buf[2:]
				if len(buf) < int(valueSize) {
					return packet, io.ErrUnexpectedEOF
				}
				var value string
				value, buf = string(buf[:valueSize]), buf[valueSize:]
				packet.Headers[name] = value
			}
		}
	}

	return packet, nil
}

type DataPacket struct {
	ID   uint32
	Data []byte
}

func (p DataPacket) AppendTo(dst []byte) []byte {
	dst = bytesutil.AppendUint32BE(dst, p.ID)
	dst = bytesutil.AppendUint16BE(dst, uint16(len(p.Data)))
	dst = append(dst, p.Data...)
	return dst
}

func UnmarshalDataPacket(buf []byte) (DataPacket, error) {
	var packet DataPacket
	if len(buf) < 4+2 {
		return packet, io.ErrUnexpectedEOF
	}
	packet.ID, buf = bytesutil.Uint32BE(buf[:4]), buf[4:]
	var size uint16
	size, buf = bytesutil.Uint16BE(buf[:2]), buf[2:]
	if uint16(len(buf)) < size {
		return packet, io.ErrUnexpectedEOF
	}
	packet.Data, buf = buf[:size], buf[size:]
	return packet, nil
}

type HandshakePacket struct {
	ID        *kademlia.ID
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
	if h.ID != nil {
		dst = append(dst, 1)
		dst = h.ID.AppendTo(dst)
	} else {
		dst = append(dst, 0)
	}
	dst = append(dst, uint8(len(h.Services)))
	for _, service := range h.Services {
		dst = append(dst, uint8(len(service)))
		dst = append(dst, service...)
	}
	if h.ID != nil {
		dst = append(dst, h.Signature[:]...)
	}
	return dst
}

func UnmarshalHandshakePacket(buf []byte) (HandshakePacket, error) {
	var pkt HandshakePacket

	if len(buf) < 1 {
		return pkt, io.ErrUnexpectedEOF
	}

	hasID := buf[0] == 1
	buf = buf[1:]

	if hasID {
		id, leftover, err := kademlia.UnmarshalID(buf)
		if err != nil {
			return pkt, err
		}
		pkt.ID = &id

		buf = leftover
	}

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

	if hasID {
		if len(buf) < kademlia.SizeSignature {
			return pkt, io.ErrUnexpectedEOF
		}

		pkt.Signature, buf = *(*kademlia.Signature)(unsafe.Pointer(&((buf[:kademlia.SizeSignature])[0]))),
			buf[kademlia.SizeSignature:]
	}

	return pkt, nil
}

func (h HandshakePacket) Validate(dst []byte) error {
	if h.ID != nil {
		err := h.ID.Validate()
		if err != nil {
			return err
		}
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

	if h.ID != nil && !h.Signature.Verify(h.ID.Pub, h.AppendPayloadTo(dst)) {
		return errors.New("signature is malformed")
	}

	return nil
}

func Addr(host net.IP, port uint16) string {
	h := ""
	if len(host) > 0 {
		h = host.String()
	}
	p := strconv.FormatUint(uint64(port), 10)
	return net.JoinHostPort(h, p)
}
