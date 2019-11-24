package main

import (
	"github.com/gorilla/websocket"
	"github.com/withlzc/godemo/websocket/impl"
	"log"
	"net/http"
	"time"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		data     []byte
		wsConn   *websocket.Conn
		err      error
		isClosed bool
	)
	if wsConn, err = upgrader.Upgrade(w, r, nil); err != nil {
		return
	}
	conn := impl.NewWsConnection(wsConn)

	conn.SetPingPeriod(time.Second * 600)

	conn.SetCloseHandler(func() {
		isClosed = true
		log.Println("Close Handler do something1")
	})

	//var allData []interface{}

	//defer func() {
	//	log.Println("finished")
	//}()
	log.Println("Open connection")
	for {
		if data, err = conn.ReadMessage(); err != nil {
			log.Println(err)
			return
		}
		if data != nil {
			log.Printf("%s\n", data)
			//allData = append(allData, data)
			if err = conn.WriteMessage(data); err != nil {
				return
			}
		}
		//log.Println(len(allData))
		//time.Sleep(time.Second * 1)
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	_ = http.ListenAndServe("0.0.0.0:7777", nil)
}
