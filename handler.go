package main

import (
	"fmt"
	"os"
	"strings"
)

func PrintHandler(tcp *Conn) {
	go func() {
		for true {
			buf := make([]byte, 2)
			_, err := tcp.Read(buf)
			if err != nil {
				log.Errorf("tcp Read err=%s, pid=%d", err, os.Getpid())
				return
			}
			if len(strings.Trim(string(buf), " ")) == 0 {
				continue
			}
			tcp.wg.Add(1)
			// 逻辑处理
			fmt.Println(strings.Trim(string(buf), " "), os.Getpid())
			tcp.wg.Done()
		}
	}()
}
