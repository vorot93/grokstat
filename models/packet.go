package models

type Packet struct {
	Id         string
	Type       PacketType
	RemoteAddr string
	ProtocolId string
	Data       []byte
	Ping       int64
	Timestamp  int64
}

type PacketType int

const (
	TYPE_UNKNOWN PacketType = iota
	TYPE_TCP
	TYPE_TCP4
	TYPE_TCP6
	TYPE_UDP
	TYPE_UDP4
	TYPE_UDP6
)

func (v PacketType) IsTCP() bool {
	if v == TYPE_TCP || v == TYPE_TCP4 || v == TYPE_TCP6 {
		return true
	} else {
		return false
	}
}

func (v PacketType) IsUDP() bool {
	if v == TYPE_UDP || v == TYPE_UDP4 || v == TYPE_UDP6 {
		return true
	} else {
		return false
	}
}

func (v PacketType) IsIP6() bool {
	if v == TYPE_TCP6 || v == TYPE_UDP6 {
		return true
	} else {
		return false
	}
}
