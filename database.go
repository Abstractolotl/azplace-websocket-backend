package main

import (
	"database/sql"
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

func NewDatabase() (*Database, error) {
	database := new(Database)

	db, err := sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbAddress+":"+dbPort+")/"+dbName)

	if err != nil {
		return nil, err
	}

	database.conn = db

	return database, err
}

func (database Database) insertWebsocket(address string, timestamp int64, secret string) error {
	statement := `INSERT INTO websockets (address, timestamp_online_since, secret) VALUES (?, ?, ?);`

	_, err := database.conn.Exec(statement, address, timestamp, secret)

	return err
}
