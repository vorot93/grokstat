package main

type ConsoleMsg struct {
	Type    int
	Message string
}

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

type ProtocolEntryInfo map[string]string

type ProtocolEntryBase struct {
	MakePayloadFunc func(Packet, ProtocolEntryInfo) Packet                                                                       `json:"-"`
	RequestPackets  []RequestPacket                                                                                              `json:"-"`
	HandlerFunc     func(Packet, *ProtocolCollection, chan<- ConsoleMsg, chan<- HostProtocolIdPair, chan<- ServerEntry) []Packet `json:"-"`
	HttpProtocol    string                                                                                                       `json:"http_protocol"`
	ResponseType    string                                                                                                       `json:"response_type"`
}

type RequestPacket struct {
	Id                string `json:"id"`
	ResponsePacketNum int    `json:"response_packet_num"`
}

// Server query protocol entry defining grokstat's behavior
type ProtocolEntry struct {
	Id          string
	Base        ProtocolEntryBase
	Information ProtocolEntryInfo
}

type HostProtocolIdPair struct {
	RemoteAddr string
	ProtocolId string
}

func MakeProtocolEntry(entryTemplate ProtocolEntry) ProtocolEntry {
	entryInformation := make(ProtocolEntryInfo, len(entryTemplate.Information))
	for k, v := range entryTemplate.Information {
		entryInformation[k] = v
	}

	entry := ProtocolEntry{Base: entryTemplate.Base, Information: entryInformation}

	return entry
}

type PlayerEntry struct {
	Name string            `json:"name"`
	Ping int64             `json:"ping"`
	Info map[string]string `json:"info"`
}

var MakePlayerEntry = func() PlayerEntry {
	return PlayerEntry{Info: map[string]string{}}
}

type ServerEntry struct {
	Protocol   string            `json:"protocol"`
	Status     int               `json:"status"`
	Error      error             `json:"-"`
	Message    string            `json:"message"`
	Host       string            `json:"host"`
	Name       string            `json:"name"`
	NeedPass   bool              `json:"need-pass"`
	ModName    string            `json:"modname"`
	GameType   string            `json:"gametype"`
	Terrain    string            `json:"terrain"`
	NumClients int64             `json:"numclients"`
	MaxClients int64             `json:"maxclients"`
	NumBots    int64             `json:"numbots"`
	Secure     bool              `json:"secure"`
	Ping       int64             `json:"ping"`
	Players    []PlayerEntry     `json:"players"`
	Rules      map[string]string `json:"rules"`
}

var MakeServerEntry = func() ServerEntry {
	return ServerEntry{Players: []PlayerEntry{}, Rules: map[string]string{}}
}
