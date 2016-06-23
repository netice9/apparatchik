package bootreactor

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type ClientConnectedListener func(chan *DisplayUpdate, chan *UserEvent, *http.Request) http.Header

func NewReactorHandler(listener ClientConnectedListener) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {

		displayChan := make(chan *DisplayUpdate)
		eventChan := make(chan *UserEvent)

		header := listener(displayChan, eventChan, r)

		conn, err := upgrader.Upgrade(w, r, header)
		if err != nil {
			panic(err)
		}

		go func() {
			for displayUpdate := range displayChan {
				err := conn.WriteJSON(displayUpdate)
				if err != nil {
					break
				}
			}
		}()

		defer func() {
			close(eventChan)
		}()

		for {
			evt := &UserEvent{}
			err := conn.ReadJSON(evt)
			if err != nil {
				fmt.Println("should close! ", err.Error())
				return
			}

			eventChan <- evt

		}

	}

}

type UserEvent struct {
	ElementID string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value,omitempty"`
	Data      string `json:"data,omitempty"`
}
