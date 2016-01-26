package models

type Packet struct {
	Id         string
	RemoteAddr string
	ProtocolId string
	Protocol   ProtocolEntry
	Data       []byte
	Ping       int64
	Timestamp  int64
}
