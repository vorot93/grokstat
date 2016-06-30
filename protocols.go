package main

import "sync"

type ProtocolConfig struct {
	Id        string            `toml:"Id"`
	Template  string            `toml:"Template"`
	Overrides map[string]string `toml:"Overrides"`
}

type ProtocolCollection struct {
	sync.Mutex
	data map[string]ProtocolEntry
}

func (c *ProtocolCollection) Get(k string) (ProtocolEntry, bool) {
	c.Lock()
	defer c.Unlock()
	var v, exists = c.data[k]
	return v, exists
}

func (c *ProtocolCollection) Set(k string, v ProtocolEntry) {
	c.Lock()
	defer c.Unlock()
	c.data[k] = v
}

func (c *ProtocolCollection) Delete(k string) {
	c.Lock()
	defer c.Unlock()
	delete(c.data, k)
}

func (c *ProtocolCollection) Map() map[string]ProtocolEntry {
	c.Lock()
	defer c.Unlock()

	var m = make(map[string]ProtocolEntry, len(c.data))
	for k, v := range c.data {
		m[k] = v
	}
	return m
}

func MakeProtocolCollection() *ProtocolCollection {
	return &ProtocolCollection{data: map[string]ProtocolEntry{}}
}

// Returns a map with protocols initialized
func LoadProtocols(configData []ProtocolConfig) *ProtocolCollection {
	infoBase := ProtocolEntryInfo{`x20`: "\x20", `xFF`: "\xFF"}

	templates := make(map[string]func() ProtocolEntry)

	templates["Q3M"] = Q3MMakeProtocolTemplate
	templates["Q3S"] = Q3SMakeProtocolTemplate
	templates["TEEWORLDSM"] = TEEWORLDSMMakeProtocolTemplate
	templates["TEEWORLDSS"] = TEEWORLDSSMakeProtocolTemplate
	templates["OPENTTDM"] = OPENTTDMMakeProtocolTemplate
	templates["OPENTTDS"] = OPENTTDSMakeProtocolTemplate
	templates["STEAM"] = STEAMMakeProtocolTemplate
	templates["A2S"] = A2SMakeProtocolTemplate
	templates["MUMBLES"] = MUMBLESMakeProtocolTemplate

	var protMap = make(map[string]ProtocolEntry, len(templates))
	for k, v := range templates {
		entry := v()
		for k1, v1 := range infoBase {
			entry.Information[k1] = v1
		}
		protMap[k] = entry
	}

	var m = MakeProtocolCollection()

	for _, configEntry := range configData {
		entryId := configEntry.Id
		templateId := configEntry.Template
		overrides := configEntry.Overrides

		entryTemplate, eOk := protMap[templateId]
		if eOk == false {
			continue
		}
		protocolEntry := MakeProtocolEntry(entryTemplate)
		for k, v := range overrides {
			protocolEntry.Information[k] = v
		}
		protocolEntry.Id = entryId
		protocolEntry.Information["Id"] = entryId

		m.Set(entryId, protocolEntry)
	}

	return m
}
