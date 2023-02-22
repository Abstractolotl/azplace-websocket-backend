package main

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
