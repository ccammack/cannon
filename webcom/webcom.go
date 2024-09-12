package webcom

import (
	"encoding/json"
	"errors"
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

		var data map[string]interface{}
		err = json.Unmarshal([]byte(msg), &data)
		if err != nil {
			log.Printf("error reading socket message: %v", err)
		}

		value, ok := data["action"]
		if ok && value == "close" {
			wc.Close()
			break
		}
	}
}

func (wc *WebCom) Send(message interface{}) {
	// send message to the client
	err := wc.conn.WriteJSON(message)
	if err != nil {
		log.Printf("error sending message: %v", err)
	}
}

func (wc *WebCom) Close() {
	wc.conn.Close()
}
