package main

import (
	"fmt"
	"net"
	"os"
)

func startServer(port string) {
	// 监听端口
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Printf("监听 %s 端口失败: %s\n", port, err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("服务启动，监听端口 %s\n", port)

	// 接受连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("接受连接失败:", err)
			continue
		}

		// 处理连接
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("读取数据失败:", err)
			return
		}
		// 打印接收到的数据
		fmt.Printf("从端口接收到数据: %s\n", string(buf[:n]))
		// 发送响应
		_, err = conn.Write([]byte("数据已收到"))
		if err != nil {
			fmt.Println("发送响应失败:", err)
			return
		}
	}
}

func main() {
	// 同时启动两个服务监听不同的端口
	go startServer("5000") // 原有的5000端口
	select {}              // 阻止主goroutine退出
}
