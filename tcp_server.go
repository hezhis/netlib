package netlib

import (
	"errors"
	"net"
	"sync"
	"time"
)

type TCPServer struct {
	newAgent   func(*TCPConn) Agent
	ln         net.Listener
	conns      ConnSet
	mutexConns sync.Mutex
	wgLn       sync.WaitGroup
	wgConns    sync.WaitGroup

	opt       *Options
	msgParser *MsgParser
	logger    ILogger
}

func NewTCPServer(opt *Options, logger ILogger, newAgent func(*TCPConn) Agent) (s *TCPServer, err error) {
	if newAgent == nil {
		return nil, errors.New("NewAgent must not be nil")
	}
	if err = opt.init(); err != nil {
		return nil, err
	}
	s = new(TCPServer)
	s.logger = logger
	s.opt = opt
	s.newAgent = newAgent

	if err = s.init(); err != nil {
		return nil, err
	}

	return s, err
}

func (server *TCPServer) Start() {
	go server.run()
}

func (server *TCPServer) init() error {
	if server.opt.MaxConnNum <= 0 {
		return errMaxConnNumInvalid
	}
	if server.opt.PendingWriteNum <= 0 {
		return errPendingWriteNum
	}

	ln, err := net.Listen("tcp", server.opt.ListenAddr)
	if err != nil {
		server.logger.LogFatal("%v", err)
	}

	server.ln = ln
	server.conns = make(ConnSet)

	server.msgParser = NewMsgParser(server.opt)
	return nil
}

func (server *TCPServer) run() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	var tempDelay time.Duration
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				server.logger.LogError("accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		server.mutexConns.Lock()
		if len(server.conns) >= server.opt.MaxConnNum {
			server.mutexConns.Unlock()
			conn.Close()
			server.logger.LogDebug("too many connections")
			continue
		}
		server.conns[conn] = struct{}{}
		server.mutexConns.Unlock()

		server.wgConns.Add(1)

		tcpConn := newTCPConn(conn, server.opt.PendingWriteNum, server.msgParser, server.logger)
		agent := server.newAgent(tcpConn)
		go func() {
			agent.Run()

			// cleanup
			tcpConn.Close()
			server.mutexConns.Lock()
			delete(server.conns, conn)
			server.mutexConns.Unlock()
			agent.OnClose()

			server.wgConns.Done()
		}()
	}
}

func (server *TCPServer) Close() {
	server.ln.Close()
	server.wgLn.Wait()

	server.mutexConns.Lock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.mutexConns.Unlock()
	server.wgConns.Wait()
}
