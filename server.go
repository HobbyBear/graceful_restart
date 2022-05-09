package main

import (
	"net"
	"syscall"
	"time"
)

type Server struct {
	Listener     net.Listener
	connListener net.Listener
	ConnList     []*Conn
}

func (s *Server) ListenAddr(addr string) error {
	go s.ListenConnections()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.Listener = listener
	return nil
}

func (s *Server) Shutdown() {
	s.Listener.Close()
	if s.connListener != nil {
		s.connListener.Close()
	}
	for _, c := range s.ConnList {
		c.StopChannel <- true
	}
}

func (s *Server) AddConn(conn *Conn) {
	s.ConnList = append(s.ConnList, conn)
}

func (s *Server) ListenConnections() {
	_ = syscall.Unlink(TransferConnDomainSocket)
	var err error
	s.connListener, err = net.Listen("unix", TransferConnDomainSocket)
	if err != nil {
		log.Infof("getInheritConnections listen %s failed: %s", err)
		return
	}
	time.Sleep(5 * time.Second)
	for {
		ul := s.connListener.(*net.UnixListener)
		uc, err := ul.AcceptUnix()
		if err != nil {
			return
		}
		oob, err := transferRecvType(uc)
		if err != nil {
			return
		}
		co, err := expressrecvFd(oob)
		if err != nil {
			return
		}
		tcp := co.(*net.TCPConn)
		conn := NewConn(tcp)
		s.AddConn(conn)
		PrintHandler(conn)

	}
}
