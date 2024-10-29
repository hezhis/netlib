package netlib

import (
	"errors"
	"math"
	"time"
)

var (
	errLenMsgLenInvalid  = errors.New("invalid msgLen")
	errMaxConnNumInvalid = errors.New("invalid MaxConnNum")
	errPendingWriteNum   = errors.New("invalid PendingWriteNum")
)

type Options struct {
	ListenAddr string
	MaxConnNum int

	RemoteAddr      string
	ConnectInterval time.Duration
	AutoReconnect   bool

	PendingWriteNum int

	LenMsgLen    int
	MaxMsgLen    uint32
	LittleEndian bool
}

func (opt *Options) init() error {
	if opt.LenMsgLen != 1 && opt.LenMsgLen != 2 && opt.LenMsgLen != 4 {
		return errLenMsgLenInvalid
	}

	var max uint32
	switch opt.LenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if opt.MaxMsgLen > max {
		opt.MaxMsgLen = max
	}
	return nil
}
