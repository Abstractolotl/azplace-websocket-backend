package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

var backendKey = "THISNEEDSTOBECHANGEDLATER"

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketHandler struct {
	connections        []*websocket.Conn
	backendConnections []*websocket.Conn
}

func NewWebsocketHandler() *WebsocketHandler {
	handler := new(WebsocketHandler)

	return handler
}

func (websocketHandler WebsocketHandler) handler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Printf("failed to set websocket upgrade: %+v\n", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte("HELO from server"))

	if err != nil {
		return
	}

	websocketHandler.connections = append(websocketHandler.connections, conn)

	for {
		t, bytes, err := conn.ReadMessage()

		if t != websocket.TextMessage {
			err = conn.WriteMessage(websocket.TextMessage, []byte("message must be textmessage"))
			return
		}

		var body map[string]interface{}
		err = json.Unmarshal(bytes, &body)

		if err != nil {
			err = conn.WriteMessage(websocket.TextMessage, []byte("could not parse json body"))
			return
		}

		err = websocketHandler.methodHandler(conn, body)

		if err != nil {
			websocketHandler.connections = remove(websocketHandler.connections, conn)
			websocketHandler.backendConnections = remove(websocketHandler.backendConnections, conn)

			err = conn.Close()

			if err != nil {
				fmt.Println(err)
			}

			return
		}
	}
}

func (websocketHandler WebsocketHandler) methodHandler(conn *websocket.Conn, body map[string]interface{}) error {
	if body["method"] == nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("no method in json body"))
	}

	method, s := body["method"].(string)

	if !s {
		return conn.WriteMessage(websocket.TextMessage, []byte("method is not a string"))
	}

	switch method {
	case "login":
		return websocketHandler.loginMethod(conn, body)
	case "broadcast":
		return websocketHandler.broadcastMethod(conn, body)
	}

	return conn.WriteMessage(websocket.TextMessage, []byte("could not find method"))
}

func (websocketHandler WebsocketHandler) loginMethod(conn *websocket.Conn, body map[string]interface{}) error {
	if body["key"] == nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("no key in json body"))
	}

	key, s := body["key"].(string)

	if !s {
		return conn.WriteMessage(websocket.TextMessage, []byte("key is not a string"))
	}

	if key != backendKey {
		return conn.WriteMessage(websocket.TextMessage, []byte("key is not correct"))
	}

	websocketHandler.connections = remove(websocketHandler.connections, conn)

	websocketHandler.backendConnections = append(websocketHandler.backendConnections, conn)

	return conn.WriteMessage(websocket.TextMessage, []byte("backend logged in"))
}

func (websocketHandler WebsocketHandler) broadcastMethod(conn *websocket.Conn, body map[string]interface{}) error {
	connectionIsBackend := false

	for _, c := range websocketHandler.backendConnections {
		if c == conn {
			connectionIsBackend = true
			break
		}
	}

	if !connectionIsBackend {
		return conn.WriteMessage(websocket.TextMessage, []byte("only backend is allowed to broadcast"))
	}

	if body["data"] == nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("no data in json body"))
	}

	data, s := body["data"].(map[string]interface{})

	if !s {
		return conn.WriteMessage(websocket.TextMessage, []byte("data is not json object"))
	}

	dataBytes, err := json.Marshal(data)

	if err != nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("failed to stringify data json"))
	}

	for _, c := range websocketHandler.connections {
		err = c.WriteMessage(websocket.TextMessage, dataBytes)

		if err != nil {
			websocketHandler.connections = remove(websocketHandler.connections, c)
		}

		err = nil
	}

	return conn.WriteMessage(websocket.TextMessage, []byte("broadcasted message"))
}
