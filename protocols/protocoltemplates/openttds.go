package protocoltemplates

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/openttds"
)

var (
	OPENTTDStemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: false, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "info", ResponsePacketNum: 1}}, ServerResponseParseFunc: openttds.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "OpenTTD Server", "PreludeStarter": "", "PreludeFinisher": "\x00\x00", "RequestPreludeTemplate": "{{.PreludeStarter}}\x03{{.PreludeFinisher}}", "DefaultRequestPort": "3979"}}
)
