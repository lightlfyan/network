package network

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"log"
)

var wg sync.WaitGroup

var HandlerBox chan NetContext

func (nw *SocketClient) Start() {
	nw.check()
	HandlerBox = make(chan NetContext, nw.HandlerBoxQueueSize)

CONNECT:
	tcpAddr, err := net.ResolveTCPAddr(nw.Net, nw.Addr)
	if err != nil {
		panic(err)
	}
	conn, err := net.DialTCP(nw.Net, nil, tcpAddr)
	if err != nil {
		log.Println("connect fail  try again")
		time.Sleep(5 * time.Second)
		goto CONNECT
	}
	conn.SetLinger(-1)
	nw.conn = conn

	wg.Add(1)
	go nw.HandleClientReceiver()
	go nw.HandleClientContext()

	nw.WhenConnDone()
	wg.Wait()

	nw.WhenConnExit()
	conn.Close()
	log.Println("reconnect")
	goto CONNECT

}

func (nw *SocketClient) HandleClientReceiver() {
	defer func() { wg.Done() }()

	header := make([]byte, 4)
	method := make([]byte, 4)

	for {
		if nw.ReadDeadLine != 0 {
			nw.conn.SetReadDeadline(time.Now().Add(nw.ReadDeadLine))
		}
		// header
		n, err := io.ReadFull(nw.conn, header)
		if err != nil {
			log.Println("error receiving header:", n, err)
			return
		}

		n, err = io.ReadFull(nw.conn, method)
		if err != nil {
			log.Println("error receiving method:", n, err)
			return
		}

		size := binary.BigEndian.Uint32(header)
		data := make([]byte, size)
		n, err = io.ReadFull(nw.conn, data)

		if err != nil {
			log.Println("error receiving msg:", err)
			return
		}

		HandlerBox <- NetContext{
			Cmd:    "message",
			Method: binary.BigEndian.Uint32(method),
			Msg:    data,
			Data:   make(map[string]interface{}),
		}
	}

}

func (nw *SocketClient) Send(context NetContext) int {

	data := context.Msg.([]byte)
	byteArray := CreateByteArray()
	byteArray.WriteU32(uint32(len(data))) // data + method
	byteArray.WriteU32(context.Method)
	byteArray.WriteBytes(data)

	if nw.WriteDeadLine != 0 {
		nw.conn.SetReadDeadline(time.Now().Add(nw.WriteDeadLine))
	}

	n, err := nw.conn.Write(byteArray.Data())
	if err != nil {
		log.Println("err ", n)
		return -1
	}
	return n

}

func (nw *SocketClient) HandleClientContext() {
	for {
		select {
		case context := <-HandlerBox:
			switch context.Cmd {
			case "message":
				nw.Handler(context)
			}
		}
	}
}

func (nw *SocketClient) check() {

	if nw.Net == "" {
		panic("Net empty")
	}
	if nw.Addr == "" {
		panic("Addr empty")
	}
	if nw.HandlerBoxQueueSize <= 0 {
		panic("HandlerBoxQueueSize error")
	}
	if nw.ReadDeadLine <= 0 {
		panic("ReadDeadLine error")
	}
	if nw.WriteDeadLine <= 0 {
		panic("WriteDeadLine error")
	}
	if nw.Handler == nil {
		panic("Can not find Handler Func")
	}
}
