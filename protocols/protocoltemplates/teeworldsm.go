package protocoltemplates

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/teeworldsm"
)

var (
	TEEWORLDSMtemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: true, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "servers", ResponsePacketNum: 1}}, MasterResponseParseFunc: teeworldsm.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Teeworlds Master", "RequestPreludeStarter": "\x20\x00\x00\x00\x00\x00\xFF\xFF\xFF\xFF", "RequestPreludeTemplate": "{{.RequestPreludeStarter}}req2", "ResponsePreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "ResponsePreludeTemplate": "{{.ResponsePreludeStarter}}lis2", "DefaultRequestPort": "8300"}}
)
