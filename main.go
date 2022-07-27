package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 8,
	WriteBufferSize: 1024 * 8,
}

var connections []*websocket.Conn
var backendConnections []*websocket.Conn

var backendKey = "THISNEEDSTOBECHANGEDLATER"

func main() {
	r := gin.Default()

	registerHandler(r)

	err := r.Run()

	if err != nil {
		fmt.Println(err)
	}
}

func registerHandler(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		websocketHandler(c.Writer, c.Request)
	})
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Printf("failed to set websocket upgrade: %+v\n", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte("HELO from server"))

	if err != nil {
		return
	}

	connections = append(connections, conn)

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

		err = bodyHandler(conn, body)

		if err != nil {
			return
		}
	}
}

func bodyHandler(conn *websocket.Conn, body map[string]interface{}) error {
	if body["method"] == nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("no method in json body"))
	}

	method, s := body["method"].(string)

	if !s {
		return conn.WriteMessage(websocket.TextMessage, []byte("method is not a string"))
	}

	return methodHandler(conn, body, method)
}

func methodHandler(conn *websocket.Conn, body map[string]interface{}, method string) error {
	switch method {
	case "login":
		return loginMethod(conn, body)
	case "broadcast":
		return broadcastMethod(conn, body)
	}

	return conn.WriteMessage(websocket.TextMessage, []byte("could not find method"))
}

func loginMethod(conn *websocket.Conn, body map[string]interface{}) error {
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

	connections = removeConnection(connections, conn)

	backendConnections = append(backendConnections, conn)

	return conn.WriteMessage(websocket.TextMessage, []byte("backend logged in"))
}

func broadcastMethod(conn *websocket.Conn, body map[string]interface{}) error {
	connectionIsBackend := false

	for _, c := range backendConnections {
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

	for _, c := range connections {
		err = c.WriteMessage(websocket.TextMessage, dataBytes)

		if err != nil {
			connections = removeConnection(connections, c)
		}
	}

	return conn.WriteMessage(websocket.TextMessage, []byte("broadcasted message"))
}

func removeConnection(s []*websocket.Conn, conn *websocket.Conn) []*websocket.Conn {
	for i, c := range s {
		if c == conn {
			return append(s[:i], s[i+1:]...)
		}
	}

	return s
}