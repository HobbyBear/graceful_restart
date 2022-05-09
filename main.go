package main

import (
	"fmt"
	"github.com/apsdehal/go-logger"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	server = &Server{}
	log    *logger.Logger
)

func main() {
	var err error
	log, err = logger.New("test", 1, os.Stdout)

	err = server.ListenAddr(":8080")
	if err != nil {
		log.Fatal(err.Error())
	}

	go func() {
		for {
			conn, err := server.Listener.Accept()
			if err != nil {
				return
			}
			tcp := conn.(*net.TCPConn)
			connection := NewConn(tcp)
			server.AddConn(connection)
			PrintHandler(connection)
		}
	}()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR2)
	<-ch
	server.Shutdown()
	time.Sleep(15 * time.Second)
	log.Info("server shutdown")
}

// 新进程从unix socket中读取 旧进程中 tcp conn 的fd，并转化为tcp conn
func transferRecvType(uc *net.UnixConn) ([]byte, error) {
	buf := make([]byte, 32)
	oob := make([]byte, 32)
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("ReadMsgUnix error: %v", err)
	}

	return oob[:oobn], nil
}

func expressrecvFd(oob []byte) (net.Conn, error) {
	scms, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, fmt.Errorf("ParseSocketControlMessage: %v", err)
	}
	if len(scms) != 1 {
		return nil, fmt.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
	}
	scm := scms[0]
	gotFds, err := unix.ParseUnixRights(&scm)
	if err != nil {
		return nil, fmt.Errorf("unix.ParseUnixRights: %v", err)
	}
	if len(gotFds) != 1 {
		return nil, fmt.Errorf("wanted 1 fd; got %#v", gotFds)
	}
	f := os.NewFile(uintptr(gotFds[0]), "fd-from-old"+strconv.Itoa(gotFds[0]))
	conn, err := net.FileConn(f)
	if err != nil {
		return nil, fmt.Errorf("FileConn error :%v", gotFds)
	}
	return conn, nil
}
