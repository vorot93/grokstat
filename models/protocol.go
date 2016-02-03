package models

type ProtocolEntryInfo map[string]string

type ProtocolEntryBase struct {
	MakePayloadFunc func(Packet, ProtocolEntryInfo) Packet                                                                      `json:"-"`
	RequestPackets  []RequestPacket                                                                                             `json:"-"`
	HandlerFunc     func(Packet, ProtocolCollection, chan<- ConsoleMsg, chan<- HostProtocolIdPair, chan<- ServerEntry) []Packet `json:"-"`
	HttpProtocol    string                                                                                                      `json:"http_protocol"`
	ResponseType    string                                                                                                      `json:"response_type"`
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

type ProtocolCollection interface {
	FindById(string) (ProtocolEntry, bool)
	All() []ProtocolEntry
	AddEntry(ProtocolEntry)
	DeleteEntry(string) (ProtocolEntry, bool)
}

type sharedProtocolCollection struct {
	datasem chan struct{}
	data    []ProtocolEntry
}

func (c sharedProtocolCollection) getIndex(id string) (ProtocolEntry, int) {
	data := c.data

	searchfunc := func(i int) bool { return data[i].Id == id }

	foundIndex := len(data)
	for i := 0; i < len(data); i++ {
		if searchfunc(i) == true {
			foundIndex = i
		}
	}

	if foundIndex == len(data) {
		return ProtocolEntry{}, -1
	} else {
		return data[foundIndex], foundIndex
	}
}

func (c sharedProtocolCollection) FindById(id string) (ProtocolEntry, bool) {
	var exists bool
	entry, i := c.getIndex(id)
	if i == -1 {
		exists = false
	} else {
		exists = true
	}

	return entry, exists
}

func (c sharedProtocolCollection) All() []ProtocolEntry {
	return c.data
}

func (c *sharedProtocolCollection) AddEntry(entry ProtocolEntry) {
	<-c.datasem
	c.data = append(c.data, entry)
	c.datasem <- struct{}{}
}

func (c *sharedProtocolCollection) DeleteEntry(id string) (ProtocolEntry, bool) {
	_, i := c.getIndex(id)

	if i != -1 {
		<-c.datasem
		deletedEntry := c.data[i]
		c.data[i], c.data = c.data[len(c.data)-1], c.data[:len(c.data)-1]
		c.datasem <- struct{}{}
		return deletedEntry, true
	} else {
		return ProtocolEntry{}, false
	}
}

func MakeSharedProtocolCollection() *sharedProtocolCollection {
	c := new(sharedProtocolCollection)
	c.data = []ProtocolEntry{}
	c.datasem = make(chan struct{}, 1)
	c.datasem <- struct{}{}
	return c
}

func MakeServerProtocolMapping() map[string]string {
	return make(map[string]string)
}

type HostProtocolIdPair struct {
	RemoteAddr string
	ProtocolId string
}
