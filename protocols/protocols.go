package protocols

import (
	"time"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/q3m"
	"github.com/grokstat/grokstat/protocols/q3s"
	"github.com/grokstat/grokstat/protocols/teeworldsm"
	"github.com/grokstat/grokstat/protocols/teeworldss"
)

// Returns a map with protocols initialized
func MakeProtocolMap(configData []ProtocolConfig) map[string]models.ProtocolEntry {
	templates := make(map[string]models.ProtocolEntry)
	templates["Q3M"] = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: true, MakeRequestPacketFunc: q3m.MakeRequestPacket, RequestPackets: []string{"servers"}, MasterResponseParseFunc: q3m.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena Master", "PreludeStarter": "\xFF\xFF\xFF\xFF", "RequestQueryParams": "empty full", "RequestPreludeTemplate": "{{.PreludeStarter}}getservers {{.Version}} {{.RequestQueryParams}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}getserversResponse", "Version": "68", "DefaultRequestPort": "27950"}}
	templates["Q3S"] = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: false, MakeRequestPacketFunc: q3s.MakeRequestPacket, RequestPackets: []string{"status"}, ServerResponseParseFunc: q3s.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena", "PreludeStarter": "\xFF\xFF\xFF\xFF", "Challenge": "GrokStat_" + string(time.Now().Unix()), "RequestPreludeTemplate": "{{.PreludeStarter}}getstatus {{.Challenge}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}statusResponse", "Version": "68", "DefaultRequestPort": "27950"}}
	templates["TEEWORLDSM"] = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: true, MakeRequestPacketFunc: teeworldsm.MakeRequestPacket, RequestPackets: []string{"servers"}, MasterResponseParseFunc: teeworldsm.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Teeworlds Master", "RequestPreludeStarter": "\x20\x00\x00\x00\x00\x00\xFF\xFF\xFF\xFF", "RequestPreludeTemplate": "{{.RequestPreludeStarter}}req2", "ResponsePreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "ResponsePreludeTemplate": "{{.ResponsePreludeStarter}}lis2", "DefaultRequestPort": "8300"}}
	templates["TEEWORLDSS"] = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: false, MakeRequestPacketFunc: teeworldss.MakeRequestPacket, RequestPackets: []string{"info"}, ServerResponseParseFunc: teeworldss.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Teeworlds Server", "PreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "PreludeFinisher": "\x00", "RequestPreludeTemplate": "{{.PreludeStarter}}gie3{{.PreludeFinisher}}", "ResponsePreludeTemplate": "{{.PreludeStarter}}inf3", "DefaultRequestPort": "8305"}}

	protocolMap := make(map[string]models.ProtocolEntry)

	for _, configEntry := range configData {
		entryId := configEntry.Id
		templateId := configEntry.Template
		overrides := configEntry.Overrides

		entryTemplate, eOk := templates[templateId]
		if eOk == false {
			continue
		}
		protocolEntry := models.MakeProtocolEntry(entryTemplate)
		for k, v := range overrides {
			protocolEntry.Information[k] = v
		}
		protocolEntry.Id = entryId

		protocolMap[entryId] = protocolEntry
	}

	return protocolMap
}
