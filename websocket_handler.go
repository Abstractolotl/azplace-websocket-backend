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

type Response struct {
	Error   bool    `json:"error"`
	Message *string `json:"message"`
	Method  *string `json:"method"`
}

func NewResponse(error bool, message string, method string) *Response {
	response := new(Response)

	response.Error = error

	if len(message) > 0 {
		response.Message = &message
	}

	if len(method) > 0 {
		response.Method = &method
	}

	return response
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

	err = websocketHandler.sendResponse(conn, NewResponse(false, "HELO from server", "HELO"))

	if err != nil {
		return
	}

	websocketHandler.connections = append(websocketHandler.connections, conn)
	websocketHandler.log("new connection", 200, time.Now(), c.ClientIP())

	for {
		t, bytes, err := conn.ReadMessage()

		since := time.Now()

		if t != websocket.TextMessage {
			err = websocketHandler.sendResponse(conn, NewResponse(true, "message must be textmessage", ""))
			websocketHandler.log("", 400, since, c.ClientIP())
			return
		}

		var body map[string]interface{}
		err = json.Unmarshal(bytes, &body)

		if err != nil {
			err = websocketHandler.sendResponse(conn, NewResponse(true, "could not parse json body", ""))
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

func (websocketHandler WebsocketHandler) sendResponse(conn *websocket.Conn, response *Response) error {
	dataBytes, err := json.Marshal(response)

	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, dataBytes)
}

func (websocketHandler WebsocketHandler) methodHandler(conn *websocket.Conn, body map[string]interface{}) (string, error) {
	if body["method"] == nil {
		return "", websocketHandler.sendResponse(conn, NewResponse(true, "no method in json body", ""))
	}

	method, s := body["method"].(string)

	if !s {
		return "", websocketHandler.sendResponse(conn, NewResponse(true, "method is not a string", ""))
	}

	switch method {
	case "login":
		return method, websocketHandler.loginMethod(conn, body)
	case "broadcast":
		return method, websocketHandler.broadcastMethod(conn, body)
	}

	return method, websocketHandler.sendResponse(conn, NewResponse(true, "could not find method", method))
}

func (websocketHandler WebsocketHandler) loginMethod(conn *websocket.Conn, body map[string]interface{}) error {
	if body["key"] == nil {
		return websocketHandler.sendResponse(conn, NewResponse(true, "no key in json body", "login"))
	}

	key, s := body["key"].(string)

	if !s {
		return websocketHandler.sendResponse(conn, NewResponse(true, "key is not a string", "login"))
	}

	if key != websocketHandler.backendKey {
		return websocketHandler.sendResponse(conn, NewResponse(true, "key is not correct", "login"))
	}

	websocketHandler.connections = remove(websocketHandler.connections, conn)

	websocketHandler.backendConnections = append(websocketHandler.backendConnections, conn)

	return websocketHandler.sendResponse(conn, NewResponse(false, "backend logged in", "login"))
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
		return websocketHandler.sendResponse(conn, NewResponse(true, "only backend is allowed to broadcast", "broadcast"))
	}

	if body["data"] == nil {
		return websocketHandler.sendResponse(conn, NewResponse(true, "no data in json body", "broadcast"))
	}

	data, s := body["data"].(map[string]interface{})

	if !s {
		return websocketHandler.sendResponse(conn, NewResponse(true, "data is not json object", "broadcast"))
	}

	dataBytes, err := json.Marshal(data)

	if err != nil {
		return websocketHandler.sendResponse(conn, NewResponse(true, "failed to stringify data json", "broadcast"))
	}

	for _, c := range websocketHandler.connections {
		err = c.WriteMessage(websocket.TextMessage, dataBytes)

		if err != nil {
			websocketHandler.connections = remove(websocketHandler.connections, c)
		}

		err = nil
	}

	return websocketHandler.sendResponse(conn, NewResponse(false, "broadcasted message", "broadcast"))
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
