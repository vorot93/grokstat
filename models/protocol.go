package models

type ProtocolEntryInfo map[string]string

type ProtocolEntryBase struct {
	IsMaster                bool                                                 `json:"is_master"`
	MakeRequestPacketFunc   func(ProtocolEntryInfo) []byte                       `json:"-"`
	MasterResponseParseFunc func([]byte, ProtocolEntryInfo) ([]string, error)    `json:"-"`
	ServerResponseParseFunc func([]byte, ProtocolEntryInfo) (ServerEntry, error) `json:"-"`
	HttpProtocol            string                                               `json:"http_protocol"`
	ResponseType            string                                               `json:"response_type"`
}

// Server query protocol entry defining grokstat's behavior
type ProtocolEntry struct {
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
