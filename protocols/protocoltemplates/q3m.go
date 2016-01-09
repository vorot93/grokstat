package protocoltemplates

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/q3m"
)

var (
	Q3Mtemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: true, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []string{"servers"}, MasterResponseParseFunc: q3m.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena Master", "SplitterUsed": "true", "PreludeStarter": "\xFF\xFF\xFF\xFF", "RequestQueryParams": "empty full", "RequestPreludeTemplate": "{{.PreludeStarter}}getservers {{.Version}} {{.RequestQueryParams}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}getserversResponse", "Version": "68", "DefaultRequestPort": "27950"}}
)
