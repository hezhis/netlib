package netlib

import (
	"net"
	"sync/atomic"
)

type ConnSet map[net.Conn]struct{}

type TCPConn struct {
	conn      net.Conn
	writeChan chan []byte
	closeFlag atomic.Bool
	msgParser *MsgParser
	logger    ILogger
}

func newTCPConn(conn net.Conn, pendingWriteNum int, msgParser *MsgParser, logger ILogger) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.conn = conn
	tcpConn.writeChan = make(chan []byte, pendingWriteNum)
	tcpConn.msgParser = msgParser
	tcpConn.logger = logger

	go func() {
		for b := range tcpConn.writeChan {
			if b == nil {
				break
			}

			_, err := conn.Write(b)
			if err != nil {
				break
			}
		}

		conn.Close()
		tcpConn.closeFlag.Store(true)
	}()

	return tcpConn
}

func (tcpConn *TCPConn) doDestroy() {
	tcpConn.conn.(*net.TCPConn).SetLinger(0)
	tcpConn.conn.Close()

	if !tcpConn.closeFlag.Load() {
		tcpConn.closeFlag.Store(true)
		close(tcpConn.writeChan)
	}
}

func (tcpConn *TCPConn) Close() {
	if tcpConn.closeFlag.Load() {
		return
	}

	tcpConn.doWrite(nil)
}

func (tcpConn *TCPConn) doWrite(b []byte) {
	if len(tcpConn.writeChan) == cap(tcpConn.writeChan) {
		tcpConn.logger.LogDebug("close conn: channel full")
		tcpConn.doDestroy()
		return
	}

	tcpConn.writeChan <- b
}

// b must not be modified by the others goroutines
func (tcpConn *TCPConn) Write(b []byte) {
	if b == nil {
		return
	}
	if tcpConn.closeFlag.Load() {
		return
	}

	tcpConn.doWrite(b)
}

func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b)
}

func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

func (tcpConn *TCPConn) ReadMsg() ([]byte, error) {
	return tcpConn.msgParser.Read(tcpConn)
}

func (tcpConn *TCPConn) WriteMsg(args ...[]byte) error {
	return tcpConn.msgParser.Write(tcpConn, args...)
}
