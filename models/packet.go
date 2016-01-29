package models

type Packet struct {
	Id         string
	RemoteAddr string
	ProtocolId string
	Data       []byte
	Ping       int64
	Timestamp  int64
}
