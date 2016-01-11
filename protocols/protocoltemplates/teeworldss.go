package protocoltemplates

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/teeworldss"
)

var (
	TEEWORLDSStemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: false, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "info", ResponsePacketNum: 1}}, ServerResponseParseFunc: teeworldss.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Teeworlds Server", "PreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "PreludeFinisher": "\x00", "RequestPreludeTemplate": "{{.PreludeStarter}}gie3{{.PreludeFinisher}}", "ResponsePreludeTemplate": "{{.PreludeStarter}}inf3", "DefaultRequestPort": "8305"}}
)
