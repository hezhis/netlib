package netlib

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPClient struct {
	sync.Mutex

	NewAgent  func(*TCPConn) Agent
	conn      net.Conn
	wg        sync.WaitGroup
	closeFlag atomic.Bool

	opt       *Options
	msgParser *MsgParser
	logger    ILogger
}

func NewTCPClient(opt *Options, logger ILogger, newAgent func(*TCPConn) Agent) (c *TCPClient, err error) {
	if newAgent == nil {
		return nil, errors.New("NewAgent must not be nil")
	}
	if err = opt.init(); err != nil {
		return nil, err
	}
	c = new(TCPClient)
	c.logger = logger
	c.opt = opt
	c.NewAgent = newAgent

	return c, c.init()
}

func (client *TCPClient) Start() {
	client.init()

	client.wg.Add(1)
	go client.connect()
}

func (client *TCPClient) init() error {
	if client.opt.PendingWriteNum <= 0 {
		return errPendingWriteNum
	}

	if client.opt.ConnectInterval <= 0 {
		client.opt.ConnectInterval = time.Second
		client.logger.LogInfo("invalid ConnectInterval, reset to %v", client.opt.ConnectInterval)
	}

	client.closeFlag.Store(false)

	// msg parser
	client.msgParser = NewMsgParser(client.opt)
	return nil
}

func (client *TCPClient) dial() net.Conn {
	for {
		conn, err := net.Dial("tcp", client.opt.RemoteAddr)
		if err == nil || client.closeFlag.Load() {
			return conn
		}

		client.logger.LogError("connect to %v error: %v", client.opt.RemoteAddr, err)
		time.Sleep(client.opt.ConnectInterval)
		continue
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()

reconnect:
	conn := client.dial()
	if conn == nil {
		return
	}

	if client.closeFlag.Load() {
		conn.Close()
		return
	}

	client.conn = conn

	tcpConn := newTCPConn(conn, client.opt.PendingWriteNum, client.msgParser, client.logger)
	agent := client.NewAgent(tcpConn)
	agent.Run()

	// cleanup
	tcpConn.Close()
	agent.OnClose()

	if client.opt.AutoReconnect {
		time.Sleep(client.opt.ConnectInterval)
		goto reconnect
	}
}

func (client *TCPClient) Close() {
	if client.closeFlag.Load() {
		return
	}
	client.closeFlag.Store(true)
	if client.conn != nil {
		client.conn.Close()
	}

	client.wg.Wait()
}
