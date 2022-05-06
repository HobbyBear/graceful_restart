package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("客户端 dial err：", err)
		return
	}

	for i := 0; i < 100; i++ {
		_, err := conn.Write([]byte("haha"))
		if err != nil {
			log.Println(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

}
