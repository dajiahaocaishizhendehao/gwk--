package transport

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

type TcpTransport struct {
	conn       net.Conn
	localAddr  string
	remoteAddr string
}

func SendStreamSocket(socket net.Conn, data []byte) (err error) {
	length := len(data)
	data2 := append([]byte{uint8(length >> 8), uint8(length % 256)}, data...)
	_, err = socket.Write(data2)
	if err != nil {
		log.Printf("发送数据包失败：目标地址=%s，错误=%v", socket.RemoteAddr().String(), err)
	} else {
		log.Printf("recently,发送数据包成功：目标地址=%s，长度=%d，内容=%s", socket.RemoteAddr().String(), length, hex.EncodeToString(data2))
	}
	return err
}

func (ts *TcpTransport) SendPacket(data []byte) (err error) {
	err = SendStreamSocket(ts.conn, data)
	if err != nil {
		log.Printf("SendPacket 发送失败：本地地址=%s，目标地址=%s，错误=%v", ts.localAddr, ts.remoteAddr, err)
	}
	return err
}

func (wst *TcpTransport) Close() (err error) {
	log.Printf("关闭连接：本地地址=%s，远程地址=%s", wst.localAddr, wst.remoteAddr)
	return wst.conn.Close()
}

func (ts *TcpTransport) ReadPacket() ([]byte, error) {
	// 接收数据
	lenbuf := make([]byte, 2)
	_, err := ts.conn.Read(lenbuf)
	if err != nil {
		log.Printf("读取数据包长度失败：源地址=%s，本地地址=%s，错误=%v", ts.conn.RemoteAddr().String(), ts.localAddr, err)
		return nil, err
	}
	leng := int(lenbuf[0])*256 + int(lenbuf[1])
	databuf := make([]byte, leng)

	totalRead := 0
	for totalRead < leng {
		n, err := ts.conn.Read(databuf[totalRead:])
		if err != nil {
			log.Printf("读取数据包失败：源地址=%s，本地地址=%s，错误=%v", ts.conn.RemoteAddr().String(), ts.localAddr, err)
			return nil, err
		}
		totalRead += n
	}

	if totalRead != leng {
		err = fmt.Errorf("读取数据包长度不匹配：预期=%d，实际=%d", leng, totalRead)
		log.Printf("错误：源地址=%s，本地地址=%s，%v", ts.conn.RemoteAddr().String(), ts.localAddr, err)
		return nil, err
	}

	log.Printf(",now,接收到数据包：源地址=%s，本地地址=%s，长度=%d，内容=%s", ts.conn.RemoteAddr().String(), ts.localAddr, leng, hex.EncodeToString(databuf))
	return databuf, nil
}

func NewTcpTransport(host, port string) (transport *TcpTransport, err error) {
	remoteAddr := fmt.Sprintf("%s:%s", host, port)

	tSocket, err := net.DialTimeout("tcp", remoteAddr, time.Second*18)
	if err != nil {
		log.Printf("连接远程地址失败：%v", err)
		return nil, err
	}

	ts := &TcpTransport{
		conn:       tSocket,
		localAddr:  tSocket.LocalAddr().String(),
		remoteAddr: tSocket.RemoteAddr().String(),
	}
	log.Printf("创建 TcpTransport 成功：本地地址=%s，远程地址=%s", ts.localAddr, ts.remoteAddr)
	return ts, nil
}

func WrapConn(conn net.Conn) *TcpTransport {
	ts := &TcpTransport{
		conn:       conn,
		localAddr:  conn.LocalAddr().String(),
		remoteAddr: conn.RemoteAddr().String(),
	}
	log.Printf("包装现有连接：本地地址=%s，远程地址=%s", ts.localAddr, ts.remoteAddr)
	return ts
}
