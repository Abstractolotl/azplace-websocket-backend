package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var database *Database
var websocketHandler *WebsocketHandler

func main() {
	var err error

	websocketHandler = NewWebsocketHandler()

	database, err = NewDatabase()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Connected to database " + dbAddress)

	ip, err := getLocalIP()

	if err != nil {
		fmt.Println(err)
		return
	}

	err = database.insertWebsocket(ip, time.Now().Unix(), websocketHandler.backendKey)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Inserted websocket ip in database")

	r := gin.Default()

	registerHandler(r)

	err = r.Run()

	if err != nil {
		fmt.Println(err)
	}
}

func registerHandler(r *gin.Engine) {
	r.GET("/ws", func(c *gin.Context) {
		websocketHandler.handler(c.Writer, c.Request)
	})

	r.GET("/health", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("OK"))
	})
}
