package network

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

func (nw *SocketServer) Start() {
	nw.check()
	tcpAddr, err := net.ResolveTCPAddr(nw.Net, nw.Addr)

	if err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP(nw.Net, tcpAddr)
	if err != nil {
		panic(err)
	}

	log.Println("Server Start Success", nw.Net, nw.Addr)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("accept err: ", err)
			continue
		}

		nw.handleClient(conn)
	}
}

func (nw *SocketServer) handleClient(connect *net.TCPConn) {
	client := &Client{
		Uid:         "None",
		Conn:        connect,
		SenderBox:   make(chan NetContext, nw.SenderBoxQueueSize),
		ReceiverBox: make(chan NetContext, nw.ReceiverBoxQueueSize),
		MainBox:     make(chan NetContext, nw.MainBoxQueueSize),
		IsClose:     false,
	}

	if nw.TickInterval > 0 {
		client.TickChan = time.After(nw.TickInterval)
	} else {
		client.TickChan = make(<-chan time.Time) // 没有空间的chan 用于阻塞
	}

	Accept(client, nw)

	ctx, cancel := context.WithCancel(context.Background())
	//client.ctx = ctx
	//client.cancel = cancel

	go HandlerSender(ctx, client, nw.WriteDeadLine)
	go HandlerReceiver(ctx, client, nw.ReadDeadLine, 10240)
	go MainLoop(ctx, cancel, client, nw)
}

func HandlerSender(cctx context.Context, client *Client, writeDeadLine time.Duration) {
	defer func() {
		client.MainBox <- NetContext{Cmd: SENDQUIT}
	}()

	conn := client.Conn
	var ctx NetContext

	for {
		select {
		case <-cctx.Done():
			return

		case ctx = <-client.SenderBox:
			switch ctx.Cmd {
			case "send":
				if writeDeadLine > 0 {
					conn.SetWriteDeadline(time.Now().Add(writeDeadLine))
				}
				data := ctx.Msg.([]byte)
				byteArray := CreateByteArray()
				byteArray.WriteU32(uint32(len(data)))
				byteArray.WriteU32(ctx.Method)
				byteArray.WriteBytes(data)

				n, err := conn.Write(byteArray.Data())
				if err != nil {
					log.Println("send data err: ", err, n)
					return
				}
			}
		}
	}
}

func HandlerReceiver(ctx context.Context, client *Client, readDeadLine time.Duration, maxLength uint32) {
	defer func() {
		client.MainBox <- NetContext{Cmd: RECVQUIT}
	}()

	conn := client.Conn

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if readDeadLine != 0 {
			conn.SetReadDeadline(time.Now().Add(readDeadLine))
		}

		header := make([]byte, 4)
		method := make([]byte, 4)
		_, err1 := io.ReadFull(conn, header)
		_, err2 := io.ReadFull(conn, method)
		if err1 != nil || err2 != nil {
			return
		}

		methodCode := binary.BigEndian.Uint32(method)

		size := binary.BigEndian.Uint32(header) // 减去method
		if size > maxLength {
			return
		}
		body := make([]byte, size)
		_, err := io.ReadFull(conn, body)
		if err != nil {
			return
		}

		client.MainBox <- NetContext{
			Cmd:    MESSAGE,
			Method: methodCode,
			Data:   make(map[string]interface{}),
			Msg:    body,
		}
	}
}

func MainLoop(ctx context.Context, cancel context.CancelFunc, client *Client, nw *SocketServer) {
	defer func() {
		client.Conn.Close()
		client.IsClose = true
	}()

	defer cancel()
	defer Kick(client, nw)

	for {
		select {
		case <-ctx.Done():
			return

		case ctx := <-client.MainBox:
			switch ctx.Cmd {
			case MESSAGE:
				nw.Handler(client, ctx)
			case KICK:
				return
			case RECVQUIT:
				return
			case SENDQUIT:
				return
			}

		case <-client.TickChan:
			Tick(client, nw)

		}
	}
}

func Accept(client *Client, nw *SocketServer) {
	defer CatchException()
	RegesiterOnline(client.Conn.RemoteAddr().String(), client)
	nw.Accept(client)
}

func Tick(client *Client, nw *SocketServer) {
	defer CatchException()
	defer func() { client.TickChan = time.After(nw.TickInterval) }()
	nw.Tick(client)

}

func Kick(client *Client, nw *SocketServer) {
	defer CatchException()
	UnRegesiterOnline(client.Conn.RemoteAddr().String())
	nw.Kick(client)
}

func (nw *SocketServer) check() {
	if nw.Net == "" {
		panic("Net empty")
	}
	if nw.Addr == "" {
		panic("Addr empty")
	}
	if nw.SenderBoxQueueSize <= 0 {
		panic("SenderBoxQueueSize error")
	}
	if nw.ReceiverBoxQueueSize <= 0 {
		panic("ReceiverBoxQueueSize error")
	}
	if nw.MainBoxQueueSize <= 0 {
		panic("MainBoxQueueSize error")
	}
	if nw.ReadDeadLine <= 0 {
		panic("ReadDeadLine error")
	}
	if nw.WriteDeadLine <= 0 {
		panic("WriteDeadLine error")
	}
	if nw.TickInterval < 0 {
		panic("TickInterval error")
	}
	if nw.Tick == nil {
		nw.TickInterval = 0
		nw.Tick = func(client *Client) {}
	}
	if nw.Kick == nil {
		nw.Kick = func(client *Client) {}
	}
	if nw.Handler == nil {
		panic("Can not find Handler Func")
	}
	if nw.Accept == nil {
		nw.Accept = func(client *Client) {}
	}
}
