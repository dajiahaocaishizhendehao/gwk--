package stub

import (
	"errors"
	"fmt"
	"github/xuxihai123/go-gwk/v1/src/protocol"
	"github/xuxihai123/go-gwk/v1/src/transport"
	"github/xuxihai123/go-gwk/v1/src/utils"
	"time"
)

const OK = 0x1
const FAILED = 0x2

type TunnelStub struct {
	tsport   *transport.TcpTransport
	streams  map[string]*GwkStream
	streamch chan *GwkStream
	sendch   chan *protocol.Frame
	closech  chan uint8
	errmsg   string // close msg
	seq      uint32
	//wlock    sync.Mutex
	pongFunc func(up, down int64)
}

func NewTunnelStub(tsport *transport.TcpTransport) *TunnelStub {
	stub := TunnelStub{tsport: tsport}
	stub.streamch = make(chan *GwkStream, 1024)
	stub.sendch = make(chan *protocol.Frame, 1024)
	stub.streams = make(map[string]*GwkStream)
	stub.closech = make(chan uint8)
	go stub.readWorker()
	go stub.writeWorker()
	return &stub
}

func (ts *TunnelStub) NotifyPong(handler func(up, down int64)) {
	ts.pongFunc = handler
}

func (ts *TunnelStub) AwaitClose() {
	<-ts.closech
}

func (ts *TunnelStub) sendTinyFrame(frame *protocol.Frame) error {
	// 序列化数据
	binaryData := protocol.Encode(frame)

	//ts.wlock.Lock()
	//defer ts.wlock.Unlock()
	// 发送数据
	//log.Printf("write stub cid:%s, data[%d]bytes, frame type:%d\n", frame.StreamID, len(binaryData), frame.Type)
	return ts.tsport.SendPacket(binaryData)
}

func (ts *TunnelStub) sendDataFrame(streamId string, data []byte) {
	frame := &protocol.Frame{Type: protocol.STREAM_DATA, StreamID: streamId, Data: data}
	ts.sendch <- frame
}

func (ts *TunnelStub) sendFrame(frame *protocol.Frame) error {
	frames := protocol.SplitFrame(frame)
	for _, smallframe := range frames {
		err := ts.sendTinyFrame(smallframe)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ts *TunnelStub) closeStream(streamId string) {
	ts.destroyStream(streamId)

	frame := &protocol.Frame{Type: protocol.STREAM_FIN, StreamID: streamId, Data: []byte{0x1, 0x1}}
	ts.sendch <- frame
}

func (ts *TunnelStub) resetStream(streamId string) {
	ts.destroyStream(streamId)
	frame := &protocol.Frame{Type: protocol.STREAM_RST, StreamID: streamId, Data: []byte{0x1, 0x2}}
	ts.sendch <- frame
}

func (ts *TunnelStub) writeWorker() {
	//fmt.Println("writeWorker====")
	for {
		select {
		case ref := <-ts.sendch:
			ts.sendFrame(ref)
		case <-ts.closech:
			return
		}
	}
}
func (ts *TunnelStub) readWorker() {
	defer func() {
		close(ts.closech)
	}()
	for {
		packet, err := ts.tsport.ReadPacket()
		if err != nil {
			ts.errmsg = "read packet err:" + err.Error()
			fmt.Println(ts.errmsg)
			return
		}
		respFrame, err := protocol.Decode(packet)
		if err != nil {
			ts.errmsg = "protocol error"
			fmt.Println(ts.errmsg)
			return
		}

		// Handle different frame types
		switch respFrame.Type {
		case protocol.PING_FRAME:
			pongFrame := &protocol.Frame{StreamID: respFrame.StreamID, Type: protocol.PONG_FRAME, Stime: respFrame.Stime, Atime: protocol.GetNowTimestrapInt()}
			ts.sendch <- pongFrame
		case protocol.PONG_FRAME:
			now := protocol.GetNowTimestrapInt()
			up := int64(respFrame.Atime) - int64(respFrame.Stime)
			down := int64(now) - int64(respFrame.Atime)
			if ts.pongFunc != nil {
				ts.pongFunc(up, down)
			}
		case protocol.STREAM_INIT:
			st := NewGwkStream(respFrame.StreamID, ts)
			ts.streams[st.Cid] = st
			fmt.Printf("收到 STREAM_INIT 帧: %s\n", st.Cid)
			ts.streamch <- st
		case protocol.STREAM_EST:
			streamId := respFrame.StreamID
			stream := ts.streams[streamId]
			if stream == nil {
				ts.resetStream(streamId)
				continue
			}
			fmt.Printf("收到 STREAM_EST 帧: %s\n", streamId)
			stream.Ready <- 1
			ts.streamch <- stream
		case protocol.STREAM_DATA:
			streamId := respFrame.StreamID
			stream := ts.streams[streamId]
			if stream == nil {
				ts.resetStream(streamId)
				continue
			}
			err := stream.produce(respFrame.Data)
			if err != nil {
				ts.closeStream(streamId)
			}
		case protocol.STREAM_FIN:
			ts.destroyStream(respFrame.StreamID)
		case protocol.STREAM_RST:
			ts.resetStream(respFrame.StreamID)
		default:
			fmt.Println("未知帧类型:", respFrame.Type)
		}
	}
}

func (ts *TunnelStub) CreateStream() (*GwkStream, error) {
	if ts == nil {
		return nil, errors.New("TunnelStub 未初始化")
	}

	streamId := utils.GetUUID()
	stream := NewGwkStream(streamId, ts)
	ts.streams[streamId] = stream
	frame := &protocol.Frame{Type: protocol.STREAM_INIT, StreamID: streamId}
	ts.sendch <- frame

	fmt.Printf("发送 STREAM_INIT 帧: %s\n", streamId)

	for i := 0; i < 3; i++ { //暂时循环三次创建连接尝试

		select {
		case <-stream.Ready:
			// send to proxy
			fmt.Printf("流已准备好: %s\n", streamId)
			return stream, nil
		case <-time.After(25 * time.Second):
			if i < 2 {
				fmt.Printf("创建流超时，重试 (%d): %s\n", i+1, streamId)
				ts.sendch <- frame
			}
			ts.destroyStream(streamId) // 确保超时后清理资源
			fmt.Printf("创建流超时: %s\n", streamId)
			return nil, errors.New("server connect timeout!(504)")
		}
	}
	return nil, errors.New("连接错误")
}

func (ts *TunnelStub) SetReady(stream *GwkStream) {
	frame := &protocol.Frame{Type: protocol.STREAM_EST, StreamID: stream.Cid}
	ts.sendch <- frame
}

func (ts *TunnelStub) destroyStream(streamId string) {
	stream := ts.streams[streamId]
	if stream != nil {
		stream.Close()
		delete(ts.streams, streamId)
	}
}

func (ts *TunnelStub) Ping() {
	frame := &protocol.Frame{Type: protocol.PING_FRAME, Stime: protocol.GetNowTimestrapInt()}
	ts.sendch <- frame
}

func (ts *TunnelStub) Accept() (*GwkStream, error) {

	select {
	case st := <-ts.streamch: // 收到stream
		return st, nil
	case <-ts.closech:
		// close transport
		return nil, errors.New(ts.errmsg)
	}
}
