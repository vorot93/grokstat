package protocoltemplates

import (
	"time"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/protocols/q3s"
)

var (
	Q3Stemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{IsMaster: false, MakeRequestPacketFunc: helpers.MakeRequestPacket, RequestPackets: []string{"status"}, ServerResponseParseFunc: q3s.ParseResponseMap, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena", "PreludeStarter": "\xFF\xFF\xFF\xFF", "Challenge": "GrokStat_" + string(time.Now().Unix()), "RequestPreludeTemplate": "{{.PreludeStarter}}getstatus {{.Challenge}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}statusResponse", "ServerNameRule": "sv_hostname", "NeedPassRule": "g_needpass", "TerrainRule": "mapname", "ModNameRule": "game", "GameTypeRule": "g_gametype", "MaxClientsRule": "sv_maxclients", "SecureRule": "sv_punkbuster", "Version": "68", "DefaultRequestPort": "27950"}}
)
