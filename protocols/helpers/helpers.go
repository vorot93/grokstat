package helpers

import (
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func MakeRequestPacket(packetId string, protocolInfo models.ProtocolEntryInfo) (requestPacket models.Packet) {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	requestPacket = models.Packet{Data: []byte(util.ParseTemplate(templ, protocolInfo))}
	return requestPacket
}
