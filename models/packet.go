package models

type Packet struct {
	Id   string
	Data []byte
	Ping int64
}
