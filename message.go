package netlib

import "encoding/binary"

type CompressType byte

const (
	// None does not compress.
	None CompressType = iota
)

type SerializeType byte

const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota
	// JSON for payload.
	JSON
	// ProtoBuffer for payload.
	ProtoBuffer
)

type Message struct {
	*Header
	Data []byte
}

type Header [8]byte

// CompressType returns compression type of messages.
func (h *Header) CompressType() CompressType {
	return CompressType(h[0])
}

// SetCompressType sets the compression type.
func (h *Header) SetCompressType(ct CompressType) {
	h[0] = byte(ct)
}

// SerializeType returns serialization type of payload.
func (h *Header) SerializeType() SerializeType {
	return SerializeType(h[1])
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
	h[1] = byte(st)
}

// DataLen returns len of data.
func (h *Header) DataLen() uint32 {
	return binary.BigEndian.Uint32(h[2:])
}

// SetDataLen sets  data len.
func (h *Header) SetDataLen(len uint32) {
	binary.BigEndian.PutUint32(h[2:], len)
}

// CmdId returns command id of message.
func (h *Header) CmdId() uint16 {
	return binary.BigEndian.Uint16(h[6:])
}

// SetCmdId sets command id of message.
func (h *Header) SetCmdId(cmdId uint16) {
	binary.BigEndian.PutUint16(h[6:], cmdId)
}
