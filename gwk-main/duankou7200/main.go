package main

import (
	"fmt"
	"net"
)

func main() {
	port := ":64310"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()
	fmt.Printf("Listening on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}
		fmt.Println("Connection accepted")
		// 发送数据
		message := "Hello, 本地服务!"
		_, err = conn.Write([]byte(message))
		if err != nil {
			fmt.Println("发送数据失败:", err)
			return
		}
		fmt.Printf("发送数据: %s\n", message)
		conn.Close()
	}
}
