package main

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	TransferConnDomainSocket = filepath.Join("./", "conn.sock")
)

type Conn struct {
	*net.TCPConn
	StopChannel chan bool
	wg          sync.WaitGroup
	Server      Server
}

func NewConn(tcp *net.TCPConn) *Conn {
	co := &Conn{
		TCPConn:     tcp,
		StopChannel: make(chan bool, 1),
		wg:          sync.WaitGroup{},
	}
	co.ListenStop()
	return co
}

func (c *Conn) ListenStop() {
	go func() {
		<-c.StopChannel
		var (
			unixConn *net.UnixConn
			err      error
		)

		// 保证接收缓冲区还有数据
		err = c.SetReadDeadline(time.Now())
		if err != nil {
			log.Error(err.Error() + "========SetReadDeadline")
		}
		raddr, err := net.ResolveUnixAddr("unix", TransferConnDomainSocket)
		if err != nil {
			panic(err)
		}
		for i := 1; i <= 10; i++ {
			unixConn, err = net.DialUnix("unix", nil, raddr)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Errorf("sendInheritListeners failed %s ", err)
			return
		}
		file, _ := c.File()
		rights := syscall.UnixRights(int(file.Fd()))
		_, _, err = unixConn.WriteMsgUnix(nil, rights, nil)
		if err != nil {
			log.Errorf("writeMsgUnix failed %s p:%d", err, os.Getpid())
			return
		}
		// 然后等待写结束，然后执行关闭操作
		c.wg.Wait()
		c.Close()
	}()
}
