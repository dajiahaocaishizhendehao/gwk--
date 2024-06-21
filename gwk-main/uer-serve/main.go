package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// 连接到服务端
	conn, err := net.Dial("tcp", "localhost:7200") // 连接到远程模拟端口 7200
	if err != nil {
		fmt.Println("连接失败:", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 发送数据
	message := "Hello, 本地服务!"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("发送数据失败:", err)
		return
	}
	fmt.Printf("发送数据: %s\n", message)

	// 读取响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return
	}

	fmt.Println("服务端响应:", string(buf[:n]))
}
