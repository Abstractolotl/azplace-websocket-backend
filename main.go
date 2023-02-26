package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"time"
)

var database *Database
var websocketHandler *WebsocketHandler

func main() {
	var err error

	websocketHandler = NewWebsocketHandler()

	database, err = NewDatabase()

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Println("Connected to database " + dbAddress)

	ip, err := getLocalIP()

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	err = database.insertWebsocket(*NewWebsocket(ip, time.Now().Unix(), websocketHandler.backendKey))

	if err != nil {
		log.Fatal(err.Error())
		return
	}

	log.Println("Inserted websocket ip in database")

	checkSelfInDatabase()

	r := gin.Default()

	registerHandler(r)

	err = r.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
}

func registerHandler(r *gin.Engine) {
	r.GET("/ws", websocketHandler.handler)

	r.GET("/health", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("OK"))
	})
}

func checkSelfInDatabase() {
	go func() {
		for true {
			_, err := database.getWebsocket(websocketHandler.backendKey)

			if err != nil {
				log.Println(err.Error())
				log.Println("database returned error while trying to find self, killing self.")

				os.Exit(1)
			}

			time.Sleep(10 * time.Second)
		}
	}()
}
