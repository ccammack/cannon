package connections

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var lock sync.RWMutex
var connections = make(map[*websocket.Conn]bool)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func New(w http.ResponseWriter, r *http.Request) error {
	if r.Header.Get("Upgrade") != "websocket" {
		return errors.New("error upgrading websocket")
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.New("error upgrading websocket")
	}

	lock.Lock()
	defer lock.Unlock()
	connections[c] = true
	go receive(c)
	return nil
}

func receive(c *websocket.Conn) {
	defer func() {
		lock.Lock()
		defer lock.Unlock()
		c.Close()
		delete(connections, c)
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("error reading socket message: %v", err)
			break
		}

		var data map[string]interface{}
		err = json.Unmarshal([]byte(msg), &data)
		if err != nil {
			log.Printf("error parsing socket message: %v", err)
		}

		// action:close
		value, ok := data["action"]
		if ok && value == "close" {
			break
		}
	}
}

func Broadcast(message interface{}) {
	// send message to the clients
	lock.Lock()
	defer lock.Unlock()
	for c := range connections {
		err := c.WriteJSON(message)
		if err != nil {
			log.Printf("error sending message: %v", err)
		}
	}
}
