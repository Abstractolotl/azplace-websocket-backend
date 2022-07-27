package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
)

var (
	dbAddress  = "noucake.ddns.net"
	dbPort     = 3306
	dbName     = "aztube"
	dbUser     = "root"
	dbPassword = "rootpw"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase() (*Database, error) {
	database := new(Database)

	db, err := sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbAddress+":"+strconv.Itoa(dbPort)+")/"+dbName)

	if err != nil {
		return nil, err
	}

	database.conn = db

	return database, err
}

func (database Database) insertWebsocket(address string, timestamp int64) error {
	statement := `INSERT INTO websockets (address, timestamp_online_since) VALUES (?, ?);`

	_, err := database.conn.Exec(statement, address, timestamp)

	return err
}
