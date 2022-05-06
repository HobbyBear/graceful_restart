package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var (
	listener net.Listener = nil

	graceful = flag.Bool("graceful", false, "listen on fd open 3 (internal use only)")
	message  = flag.String("message", "Hello World", "message to send")

	connList = make([]uintptr, 0)
)

func main() {
	var err error

	// 解析参数
	flag.Parse()

	fmt.Println(os.Getpid())
	// 设置监听器的监听对象（新建的或已存在的 socket 描述符）
	if *graceful {
		// 子进程监听父进程传递的 socket 描述符
		log.Println("listening on the existing file descriptor 3")
		// 子进程的 0, 1, 2 是预留给标准输入、标准输出、错误输出，故传递的 socket 描述符
		// 应放在子进程的 3
		f := os.NewFile(3, "")
		listener, err = net.FileListener(f)
		for fd := 4; fd < 5; fd++ {
			file := os.NewFile((uintptr)(fd), "/tmp/"+strconv.Itoa(fd))
			conn, _ := net.FileConn(file)
			go func(c net.Conn) {
				for true {
					buf := make([]byte, 100)
					_, err := conn.Read(buf)
					if err != nil {
						log.Fatal(err)
					}
					if len(strings.Trim(string(buf), " ")) == 0 {
						continue
					}
					fmt.Println(strings.Trim(string(buf), " "))
				}
			}(conn)
		}
		fmt.Println("进入子进程")
	} else {
		// 父进程监听新建的 socket 描述符
		log.Println("listening on a new file descriptor")
		listener, err = net.Listen("tcp", ":8080")
	}
	if err != nil {
		log.Fatalf("listener error: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}
			file, err := conn.(*net.TCPConn).File()
			connList = append(connList, file.Fd())
			go func(c net.Conn) {
				for true {
					buf := make([]byte, 100)
					_, err := conn.Read(buf)
					if err != nil {
						log.Fatal(err)
					}
					if len(strings.Trim(string(buf), " ")) == 0 {
						continue
					}
					fmt.Println(strings.Trim(string(buf), " "))
				}
			}(conn)
		}
		log.Printf("server.Serve err: %v\n", err)
	}()
	// 监听信号
	handleSignal()
	log.Println("signal end")
}

func handleSignal() {
	ch := make(chan os.Signal, 1)
	// 监听信号
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Printf("signal receive: %v\n", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM: // 终止进程执行
			log.Println("shutdown")
			signal.Stop(ch)
			log.Println("graceful shutdown")
			listener.Close()
			return
		case syscall.SIGUSR2: // 进程热重启
			log.Println("reload")
			err := reload() // 执行热重启函数
			if err != nil {
				log.Fatalf("graceful reload error: %v", err)
			}
			listener.Close()
			log.Println("graceful reload")
			return
		}
	}
}

func reload() error {
	tl, ok := listener.(*net.TCPListener)
	if !ok {
		return errors.New("listener is not tcp listener")
	}
	// 获取 socket 描述符
	f, err := tl.File()
	if err != nil {
		return err
	}
	// 设置传递给子进程的参数（包含 socket 描述符）
	newpid, err := syscall.ForkExec(os.Args[0], append(os.Args, "-graceful=true"), &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd(), f.Fd()}, connList...),
		Sys:   nil,
	})
	fmt.Println(newpid)
	return err
}
