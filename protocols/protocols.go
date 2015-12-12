package protocols

import (
	"bytes"
	"text/template"

	"github.com/grokstat/grokstat/protocols/q3m"
)

type ProtocolEntryBase struct {
	ResponseParseFunc       func([]byte, []byte) ([]string, error) `json:"-"`
	HttpProtocol            string                                 `json:"http_protocol"`
	ResponseType            string                                 `json:"response_type"`
}

type ProtocolEntryInfo map[string]string

// Server query protocol entry defining grokstat's behavior
type ProtocolEntry struct {
	Base            ProtocolEntryBase
	Information     ProtocolEntryInfo
}

func MakeProtocolEntry(entryTemplate ProtocolEntry) ProtocolEntry {
	entryInformation := make(ProtocolEntryInfo, len(entryTemplate.Information))
	for k, v := range entryTemplate.Information {
		entryInformation[k] = v
	}

	entry := ProtocolEntry{Base: entryTemplate.Base, Information: entryInformation}


	return entry
}

// Construct a new protocol entry and return it to user
func MakeRequestPrelude(entry ProtocolEntryInfo) string {
	buf := new(bytes.Buffer)
	t, _ := template.New("Request template").Parse(entry["RequestPreludeTemplate"])
	t.Execute(buf, entry)
	return buf.String()
}

func MakeResponsePrelude(entry ProtocolEntryInfo) string {
	buf := new(bytes.Buffer)
	t, _ := template.New("Response template").Parse(entry["ResponsePreludeTemplate"])
	t.Execute(buf, entry)
	return buf.String()
}

// Returns a map with protocols initialized
func MakeProtocolMap(configData []ProtocolConfig) map[string]ProtocolEntry {
	templates := make(map[string]ProtocolEntry)
	templates["Q3M"] = ProtocolEntry{Base: ProtocolEntryBase{ResponseParseFunc: q3m.ParseMasterResponse, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "Quake III Arena Master", "PreludeStarter": "\xFF\xFF\xFF\xFF", "RequestPreludeTemplate": "{{.PreludeStarter}}getservers {{.Version}} empty full\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}getserversResponse", "Version": "68", "DefaultRequestPort": "27950"}}

	protocolMap := make(map[string]ProtocolEntry)

	for _, configEntry := range configData {
		entryId := configEntry.Id
		templateId := configEntry.Template
		overrides := configEntry.Overrides

		protocolEntry := MakeProtocolEntry(templates[templateId])
		for k, v := range overrides {
			protocolEntry.Information[k] = v
		}

		protocolMap[entryId] = protocolEntry
	}

	return protocolMap
}
