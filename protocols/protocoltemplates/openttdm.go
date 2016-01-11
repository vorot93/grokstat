package protocoltemplates

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/openttdm"
)

var (
	OPENTTDMtemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: true, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "servers4", ResponsePacketNum: -1}}, MasterResponseParseFunc: openttdm.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "OpenTTD Master", "DefaultRequestPort": "3978", "ProtocolVer": string(byte(2)), "IPType": string(byte(0)), "RequestPreludeTemplate": "\x05\x00\x06{{.ProtocolVer}}{{.IPType}}"}}
)
