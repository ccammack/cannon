package webcom

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebCom struct {
	conn *websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func New(w http.ResponseWriter, r *http.Request) (*WebCom, error) {
	if r.Header.Get("Upgrade") != "websocket" {
		return nil, errors.New("error upgrading websocket")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading websocket: %v", err)
		return nil, err
	}

	wc := WebCom{conn: conn}
	go wc.receive()
	return &wc, nil
}

func (wc *WebCom) receive() {
	defer wc.conn.Close()

	for {
		_, msg, err := wc.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		fmt.Printf("Received: %s\n", msg)
	}
}

func (wc *WebCom) Send(message interface{}) {
	// Send message back to the client
	err := wc.conn.WriteJSON(message)
	if err != nil {
		log.Printf("error sending message: %v", err)
	}
}
