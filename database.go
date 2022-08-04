package main

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"os"
)

var (
	dbAddress  = os.Getenv("DB_ADDR")
	dbPort     = os.Getenv("DB_PORT")
	dbName     = os.Getenv("DB_NAME")
	dbUser     = os.Getenv("DB_USER")
	dbPassword = os.Getenv("DB_PW")
)

type Database struct {
	conn *sql.DB
}

type Websocket struct {
	Address   string `json:"address"`
	Timestamp int64  `json:"timestamp_online_since"`
	Secret    string `json:"secret"`
}

func NewWebsocket(address string, timestamp int64, secret string) *Websocket {
	websocket := new(Websocket)

	websocket.Address = address
	websocket.Timestamp = timestamp
	websocket.Secret = secret

	return websocket
}

func NewDatabase() (*Database, error) {
	database := new(Database)

	db, err := sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbAddress+":"+dbPort+")/"+dbName)

	if err != nil {
		return nil, err
	}

	database.conn = db

	return database, err
}

func (database Database) insertWebsocket(websocket Websocket) error {
	statement := `INSERT INTO websockets (address, timestamp_online_since, secret) VALUES (?, ?, ?);`

	_, err := database.conn.Exec(statement, websocket.Address, websocket.Timestamp, websocket.Secret)

	return err
}

func (database Database) getWebsocket(secret string) (*Websocket, error) {
	statement := `SELECT * FROM websockets WHERE secret = ?`

	r, err := database.conn.Query(statement, secret)

	if err != nil {
		return nil, err
	}

	var websockets []Websocket

	for r.Next() {
		var id int64
		var websocket Websocket

		err = r.Scan(&id, &websocket.Address, &websocket.Secret, &websocket.Timestamp)

		if err == nil {
			websockets = append(websockets, websocket)
		}
	}

	if len(websockets) == 1 {
		return &websockets[0], nil
	} else {
		return nil, errors.New("could not find websocket")
	}
}
