package main

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"time"
)

func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8021", 3*time.Second)
	if err != nil {
		fmt.Println("dial error: ", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReaderSize(conn, 1024<<16)
	header, err := textproto.NewReader(reader).ReadMIMEHeader()
	if err != nil {
		fmt.Printf("textproto reader: %#v\n", err)
	}
	fmt.Printf("header: %#v\n", header)
}
