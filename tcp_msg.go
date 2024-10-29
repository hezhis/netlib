package netlib

import (
	"encoding/binary"
	"errors"
	"io"
)

// --------------
// | len | data |
// --------------

// -------------------------------------
// | len | trace id len |trace id|data |
// -------------------------------------
type MsgParser struct {
	opt *Options
}

func NewMsgParser(opt *Options) *MsgParser {
	opt.init()

	p := new(MsgParser)
	p.opt = opt

	return p
}

// Read goroutine safe
func (p *MsgParser) Read(conn *TCPConn) ([]byte, error) {
	lenMsgLen := p.opt.LenMsgLen
	var b [4]byte
	bufMsgLen := b[:lenMsgLen]

	// read len
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}

	// parse len
	var msgLen uint32
	switch lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0])
	case 2:
		if p.opt.LittleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case 4:
		if p.opt.LittleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	// check len
	if msgLen > p.opt.MaxMsgLen {
		return nil, errors.New("message too long")
	}

	// data
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}

	return msgData, nil
}

func (p *MsgParser) PackMsg(args ...[]byte) ([]byte, error) {
	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > p.opt.MaxMsgLen {
		return nil, errors.New("message too long")
	}

	msg := make([]byte, uint32(p.opt.LenMsgLen)+msgLen)

	// write len
	switch p.opt.LenMsgLen {
	case 1:
		msg[0] = byte(msgLen)
	case 2:
		if p.opt.LittleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen))
		}
	case 4:
		if p.opt.LittleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen)
		} else {
			binary.BigEndian.PutUint32(msg, msgLen)
		}
	}

	// write data
	l := p.opt.LenMsgLen
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	return msg, nil
}

// Write goroutine safe
func (p *MsgParser) Write(conn *TCPConn, args ...[]byte) error {
	buf, err := p.PackMsg(args...)
	if err != nil {
		return err
	}
	conn.Write(buf)
	return nil
}
