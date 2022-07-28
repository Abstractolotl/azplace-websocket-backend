package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketHandler struct {
	connections        []*websocket.Conn
	backendConnections []*websocket.Conn
	backendKey         string
}

func NewWebsocketHandler() *WebsocketHandler {
	handler := new(WebsocketHandler)

	handler.backendKey = uuid.New().String()

	return handler
}

func (websocketHandler WebsocketHandler) handler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		fmt.Printf("failed to set websocket upgrade: %+v\n", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte("HELO from server"))

	if err != nil {
		return
	}

	websocketHandler.connections = append(websocketHandler.connections, conn)
	websocketHandler.log("new connection", 200, time.Now(), c.ClientIP())

	for {
		t, bytes, err := conn.ReadMessage()

		since := time.Now()

		if t != websocket.TextMessage {
			err = conn.WriteMessage(websocket.TextMessage, []byte("message must be textmessage"))
			websocketHandler.log("", 400, since, c.ClientIP())
			return
		}

		var body map[string]interface{}
		err = json.Unmarshal(bytes, &body)

		if err != nil {
			err = conn.WriteMessage(websocket.TextMessage, []byte("could not parse json body"))
			websocketHandler.log("", 400, since, c.ClientIP())
			return
		}

		method, err := websocketHandler.methodHandler(conn, body)

		status := 0

		if err == nil && len(method) > 0 {
			status = 200
		} else {
			status = 400
		}

		websocketHandler.log(method, status, since, c.ClientIP())

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

func (websocketHandler WebsocketHandler) methodHandler(conn *websocket.Conn, body map[string]interface{}) (string, error) {
	if body["method"] == nil {
		return "", conn.WriteMessage(websocket.TextMessage, []byte("no method in json body"))
	}

	method, s := body["method"].(string)

	if !s {
		return "", conn.WriteMessage(websocket.TextMessage, []byte("method is not a string"))
	}

	switch method {
	case "login":
		return method, websocketHandler.loginMethod(conn, body)
	case "broadcast":
		return method, websocketHandler.broadcastMethod(conn, body)
	}

	return method, conn.WriteMessage(websocket.TextMessage, []byte("could not find method"))
}

func (websocketHandler WebsocketHandler) loginMethod(conn *websocket.Conn, body map[string]interface{}) error {
	if body["key"] == nil {
		return conn.WriteMessage(websocket.TextMessage, []byte("no key in json body"))
	}

	key, s := body["key"].(string)

	if !s {
		return conn.WriteMessage(websocket.TextMessage, []byte("key is not a string"))
	}

	if key != websocketHandler.backendKey {
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

func (websocketHandler WebsocketHandler) log(method string, statusCode int, since time.Time, clientIp string) {
	param := new(gin.LogFormatterParams)
	param.Path = method
	param.Method = http.MethodGet
	param.ClientIP = clientIp
	param.Latency = time.Since(since)
	param.StatusCode = statusCode
	param.TimeStamp = time.Now()

	statusColor := param.StatusCodeColor()
	methodColor := param.MethodColor()
	resetColor := param.ResetColor()

	param.Method = "Websocket"

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}

	fmt.Print(fmt.Sprintf("[GIN-WS] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	))
}
