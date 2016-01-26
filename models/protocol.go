package models

type ProtocolEntryInfo map[string]string

type ProtocolEntryBase struct {
	MakePayloadFunc func(Packet, ProtocolEntryInfo) Packet                                                                            `json:"-"`
	RequestPackets  []RequestPacket                                                                                                   `json:"-"`
	HandlerFunc     func(Packet, map[string]ProtocolEntry, chan<- ConsoleMsg, chan<- HostProtocolIdPair, chan<- ServerEntry) []Packet `json:"-"`
	HttpProtocol    string                                                                                                            `json:"http_protocol"`
	ResponseType    string                                                                                                            `json:"response_type"`
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

func MakeProtocolEntry(entryTemplate ProtocolEntry) ProtocolEntry {
	entryInformation := make(ProtocolEntryInfo, len(entryTemplate.Information))
	for k, v := range entryTemplate.Information {
		entryInformation[k] = v
	}

	entry := ProtocolEntry{Base: entryTemplate.Base, Information: entryInformation}

	return entry
}

func MakeServerProtocolMapping() map[string]string {
	return make(map[string]string)
}

type HostProtocolIdPair struct {
	RemoteAddr string
	ProtocolId string
}
