package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebsocketHandler struct {
	backendKey         string
	connections        []*websocket.Conn
	backendConnections []*websocket.Conn
}

func NewWebsocketHandler() *WebsocketHandler {
	handler := new(WebsocketHandler)

	handler.backendKey = uuid.New().String()

	return handler
}

func (w *WebsocketHandler) handler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)

	if err != nil {
		log.Printf("failed to set websocket upgrade: %+v\n", err)
		return
	}

	w.connections = append(w.connections, conn)
	w.log("new connection", 200, time.Now(), c.ClientIP())

	for {
		t, bytes, err := conn.ReadMessage()

		since := time.Now()

		if t != websocket.TextMessage {
			err = w.sendResponse(conn, NewResponse(true, "message must be textmessage", "", nil))
			w.log("", 400, since, c.ClientIP())
			return
		}

		var body map[string]interface{}
		err = json.Unmarshal(bytes, &body)

		if err != nil {
			err = w.sendResponse(conn, NewResponse(true, "could not parse json body", "", nil))
			w.log("", 400, since, c.ClientIP())
			return
		}

		method, err := w.methodHandler(conn, body)

		status := 0

		if err == nil && len(method) > 0 {
			status = 200
		} else {
			status = 400
		}

		w.log(method, status, since, c.ClientIP())

		if err != nil {
			w.connections = remove(w.connections, conn)
			w.backendConnections = remove(w.backendConnections, conn)

			err = conn.Close()

			if err != nil {
				log.Println(err.Error())
			}

			return
		}
	}
}

func (w *WebsocketHandler) sendResponse(conn *websocket.Conn, response *Response) error {
	dataBytes, err := json.Marshal(response)

	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, dataBytes)
}

func (w *WebsocketHandler) methodHandler(conn *websocket.Conn, body map[string]interface{}) (string, error) {
	if body["method"] == nil {
		return "", w.sendResponse(conn, NewResponse(true, "no method in json body", "", nil))
	}

	method, s := body["method"].(string)

	if !s {
		return "", w.sendResponse(conn, NewResponse(true, "method is not a string", "", nil))
	}

	switch method {
	case "login":
		return method, w.loginMethod(conn, body)
	case "broadcast":
		return method, w.broadcastMethod(conn, body)
	case "count":
		return method, w.countMethod(conn)
	}

	return method, w.sendResponse(conn, NewResponse(true, "could not find method", method, nil))
}

func (w *WebsocketHandler) loginMethod(conn *websocket.Conn, body map[string]interface{}) error {
	if body["key"] == nil {
		return w.sendResponse(conn, NewResponse(true, "no key in json body", "login", nil))
	}

	key, s := body["key"].(string)

	if !s {
		return w.sendResponse(conn, NewResponse(true, "key is not a string", "login", nil))
	}

	if key != w.backendKey {
		return w.sendResponse(conn, NewResponse(true, "key is not correct", "login", nil))
	}

	w.connections = remove(w.connections, conn)
	w.backendConnections = append(w.backendConnections, conn)

	return w.sendResponse(conn, NewResponse(false, "backend logged in", "login", nil))
}

func (w *WebsocketHandler) broadcastMethod(conn *websocket.Conn, body map[string]interface{}) error {
	connectionIsBackend := w.connectionIsBackend(conn)

	if !connectionIsBackend {
		return w.sendResponse(conn, NewResponse(true, "only backend is allowed to broadcast", "broadcast", nil))
	}

	if body["data"] == nil {
		return w.sendResponse(conn, NewResponse(true, "no data in json body", "broadcast", nil))
	}

	data, s := body["data"].(map[string]interface{})

	if !s {
		return w.sendResponse(conn, NewResponse(true, "data is not json object", "broadcast", nil))
	}

	dataBytes, err := json.Marshal(data)

	if err != nil {
		return w.sendResponse(conn, NewResponse(true, "failed to stringify data json", "broadcast", nil))
	}

	for _, c := range w.connections {
		err = c.WriteMessage(websocket.TextMessage, dataBytes)

		if err != nil {
			w.connections = remove(w.connections, c)
		}

		err = nil
	}

	return w.sendResponse(conn, NewResponse(false, "broadcasted message", "broadcast", nil))
}

func (w *WebsocketHandler) countMethod(conn *websocket.Conn) error {
	connectionIsBackend := w.connectionIsBackend(conn)

	if !connectionIsBackend {
		return w.sendResponse(conn, NewResponse(true, "only backend is allowed to count", "count", nil))
	}

	response := make(map[string]interface{})

	response["connectionCount"] = len(w.connections)
	response["backendConnectionCount"] = len(w.backendConnections)

	return w.sendResponse(conn, NewResponse(false, "connections count", "count", &response))
}

func (w *WebsocketHandler) connectionIsBackend(conn *websocket.Conn) bool {
	for _, c := range w.backendConnections {
		if c == conn {
			return true
		}
	}

	return false
}

func (w *WebsocketHandler) log(method string, statusCode int, since time.Time, clientIp string) {
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

	log.Println(fmt.Sprintf("[GIN-WS] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
	))
}
