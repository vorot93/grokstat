package protocols

import (
	"bytes"
	"text/template"

	"github.com/grokstat/grokstat/protocols/q3m"
)

type ProtocolEntryInfo struct {
	Id                      string                                 `json:"id"`
	Name                    string                                 `json:"name"`
	RequestPreludeTemplate  string                                 `json:"request_prelude_template"`
	ResponseType            string                                 `json:"response_type"`
	ResponseParseFunc       func([]byte, []byte) ([]string, error) `json:"-"`
	ResponsePreludeTemplate string                                 `json:"response_prelude_template"`
	Version                 string                                 `json:"version"`
	DefaultRequestPort      string                                 `json:"default_request_port"`
}

// Server query protocol entry defining grokstat's behavior
type ProtocolEntry struct {
	Information     ProtocolEntryInfo
	RequestPrelude  string
	ResponsePrelude string
}

// Construct a new protocol entry and return it to user
func NewProtocolEntry(entry_info ProtocolEntryInfo) ProtocolEntry {
	entry := ProtocolEntry{Information: entry_info}

	buf1 := new(bytes.Buffer)
	t1, _ := template.New("Request template").Parse(entry.Information.RequestPreludeTemplate)
	t1.Execute(buf1, entry.Information)
	entry.RequestPrelude = buf1.String()

	buf2 := new(bytes.Buffer)
	t2, _ := template.New("Response template").Parse(entry.Information.ResponsePreludeTemplate)
	t2.Execute(buf2, entry.Information)
	entry.ResponsePrelude = buf2.String()

	return entry
}

// Returns a map with protocols initialized
func MakeProtocolMap() map[string]ProtocolEntry {
	q3m_template := ProtocolEntryInfo{Id: "q3m", Name: "Quake III Arena Master", RequestPreludeTemplate: "\xFF\xFF\xFF\xFFgetservers {{.Version}} empty full\n", ResponseType: "Server list", ResponseParseFunc: q3m.ParseMasterResponse, ResponsePreludeTemplate: "\xFF\xFF\xFF\xFFgetserversResponse", Version: "68", DefaultRequestPort: "27950"}

	protocolMap := make(map[string]ProtocolEntry)

	q3m_protocol := q3m_template
	protocolMap["q3m"] = NewProtocolEntry(ProtocolEntryInfo(q3m_protocol))

	xonoticm_protocol := q3m_template
	xonoticm_protocol.Id = "xonoticm"
	xonoticm_protocol.Name = "Xonotic Master"
	xonoticm_protocol.RequestPreludeTemplate = "\377\377\377\377getservers Xonotic {{.Version}} empty full"
	xonoticm_protocol.Version = "3"
	protocolMap["xonoticm"] = NewProtocolEntry(ProtocolEntryInfo(xonoticm_protocol))

	return protocolMap
}
