package network

import (
	"bytes"
	"encoding/binary"
)

const (
	PACKET_MAX_LEN = 1024
)

type ByteArray struct {
	// v1
	data []byte

	// v2
	buf  *bytes.Buffer
}

func (p *ByteArray) Data() []byte {
	return p.buf.Bytes()
}

func CreateByteArray() *ByteArray {
	return &ByteArray{
		buf: bytes.NewBuffer(nil),
		data: make([]byte, 0, PACKET_MAX_LEN)}
}

func (p *ByteArray) WriteByte(v byte) {
	p.data = append(p.data, v)
	p.buf.WriteByte(v)
}

func (p *ByteArray) WriteBytes(v []byte) {
	p.data = append(p.data, v...)
	p.buf.Write(v)
}

func (p *ByteArray) WriteU32(v uint32) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	p.buf.Write(buf)

	p.data = append(p.data, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
