package models

type ProtocolEntryInfo map[string]string

type ProtocolEntryBase struct {
	IsMaster                bool                                                            `json:"is_master"`
	MakeRequestPacketFunc   func(string, ProtocolEntryInfo) Packet                          `json:"-"`
	RequestPackets          []string                                                        `json:"-"`
	MasterResponseParseFunc func(map[string]Packet, ProtocolEntryInfo) ([]string, error)    `json:"-"`
	ServerResponseParseFunc func(map[string]Packet, ProtocolEntryInfo) (ServerEntry, error) `json:"-"`
	HttpProtocol            string                                                          `json:"http_protocol"`
	ResponseType            string                                                          `json:"response_type"`
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
