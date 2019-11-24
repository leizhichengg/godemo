package impl

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"time"
)

var (
	// Send ping to peer with this period. Must be less than pongWait.
	pingPeriod = 4 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = (pingPeriod * 10) / 8
)

type wsConnection struct {
	// Package WebSocket connection.
	wsConn *websocket.Conn
	// Store received message.
	inChan chan []byte
	// Store sent message.
	outChan chan []byte
	// Store close message.
	closeChan chan byte
	// State determines if the connection is closed.
	isClosed bool
	mutex    sync.Mutex
}

func NewWsConnection(wsConn *websocket.Conn) (conn *wsConnection) {
	conn = &wsConnection{
		wsConn:    wsConn,
		inChan:    make(chan []byte, 100),
		outChan:   make(chan []byte, 100),
		closeChan: make(chan byte, 1),
	}
	go conn.readLoop()
	go conn.writeLoop()
	go conn.keepAlive()
	return
}

// Read message from websocket
func (conn *wsConnection) ReadMessage() (data []byte, err error) {
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		// If the connection is closed, but inChan is not empty,
		// get all data then return error
		select {
		case data = <-conn.inChan:
		default:
			return nil, errors.New("connection is closed")
		}
		return
	}
	return
}

// Send message to websocket
func (conn *wsConnection) WriteMessage(data []byte) (err error) {
	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

// Do something when websocket closed
func (conn *wsConnection) SetCloseHandler(handlerFunc ...func()) {
	conn.wsConn.SetCloseHandler(func(code int, text string) error {
		for _, f := range handlerFunc {
			f()
		}
		return nil
	})
}

// Set keepalive ping period
func (conn *wsConnection) SetPingPeriod(ping time.Duration) {
	pingPeriod = ping
	pongWait = (pingPeriod * 10) / 8
}

// Close websocket
func (conn *wsConnection) Close() {
	conn.mutex.Lock()
	if !conn.isClosed {
		if err := conn.wsConn.Close(); err == nil {
			close(conn.closeChan)
			conn.isClosed = true
		}
	}
	conn.mutex.Unlock()
}

// Use ping pong to keep websocket alive
func (conn *wsConnection) keepAlive() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	conn.wsConn.SetPongHandler(func(string) error {
		log.Println("get pong...")
		err := conn.wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return err
	})
	conn.wsConn.SetPingHandler(func(string) error {
		log.Println("get ping...")
		err := conn.wsConn.WriteMessage(websocket.PongMessage, nil)
		return err
	})
	for {
		select {
		case <-ticker.C:
			log.Println("send ping...")
			if err := conn.wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("ping: ", err)
				conn.Close()
				return
			}
		case <-conn.closeChan:
			return
		}
	}
}

// Read message to in channel
func (conn *wsConnection) readLoop() {
	var (
		data []byte
		err  error
	)
	for {
		if _, data, err = conn.wsConn.ReadMessage(); err != nil {
			log.Printf("error: %v\n", err)
			conn.Close()
			return
		}
		select {
		case conn.inChan <- data:
			log.Printf("read %s\n", data)
			log.Printf("inchan size: %d", len(conn.inChan))
		case <-conn.closeChan:
			return
		}
	}
}

// Write message from out channel
func (conn *wsConnection) writeLoop() {
	var (
		data []byte
		err  error
	)
	for {
		select {
		case data = <-conn.outChan:
			log.Printf("write %s\n", data)
		case <-conn.closeChan:
			// If the connection is closed, but outChan is not empty,
			// throw all data then return
			select {
			case <-conn.outChan:
			default:
				return
			}
		}
		if err = conn.wsConn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("error: %v\n", err)
			conn.Close()
			return
		}
	}
}
