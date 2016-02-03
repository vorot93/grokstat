package helpers

import (
	"bytes"
	"fmt"

	"github.com/grokstat/grokstat/grokstatconstants"
	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func MakeRequestPacket(packetId string, protocolInfo models.ProtocolEntryInfo) (requestPacket models.Packet) {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	requestPacket = models.Packet{Data: []byte(util.ParseTemplate(templ, protocolInfo))}
	return requestPacket
}

func MakePayload(packet models.Packet, protocolInfo models.ProtocolEntryInfo) models.Packet {
	packet.Data = MakeRequestPacket(packet.Id, protocolInfo).Data
	return packet
}

func SimpleReceiveHandler(parseFunc func(models.Packet, models.ProtocolEntryInfo) (models.ServerEntry, error), packet models.Packet, protColl models.ProtocolCollection, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	sendPackets = []models.Packet{}

	protocolId := packet.ProtocolId
	protocol, protocolExists := protColl.FindById(protocolId)
	if !protocolExists {
		return sendPackets
	}

	protocolInfo := protocol.Information
	remoteIp := packet.RemoteAddr
	serverEntry, sErr := parseFunc(packet, protocolInfo)

	if sErr != nil {
		messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("%s - %s - %s", protocolId, remoteIp, sErr.Error())}
		return sendPackets
	}

	serverEntry.Protocol = protocolId
	serverEntry.Host = remoteIp
	serverEntry.Status = 200

	serverEntryChan <- serverEntry

	return sendPackets
}

func MasterReceiveHandler(parseFunc func(models.Packet, models.ProtocolEntryInfo) ([]string, error), packet models.Packet, protColl models.ProtocolCollection, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	sendPackets = []models.Packet{}

	protocolId := packet.ProtocolId
	protocol, protocolExists := protColl.FindById(protocolId)
	if !protocolExists {
		return sendPackets
	}

	protocolInfo := protocol.Information
	protocolName, _ := protocolInfo["Name"]
	remoteIp := packet.RemoteAddr
	servers, err := parseFunc(packet, protocolInfo)

	if err != nil {
		messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("%s - %s - %s.", protocolId, remoteIp, err.Error())}
		return sendPackets
	}

	masterOf := protocolInfo["MasterOf"]
	for _, ipAddr := range servers {
		pair := models.HostProtocolIdPair{RemoteAddr: ipAddr, ProtocolId: masterOf}
		protocolMappingInChan <- pair
		sendPackets = append(sendPackets, MakeSendPackets(pair, protColl)...)
	}

	masterServerEntry := models.MakeServerEntry()
	masterServerEntry.Protocol = packet.ProtocolId
	masterServerEntry.Host = packet.RemoteAddr
	masterServerEntry.Name = fmt.Sprintf("%s Server", protocolName)
	masterServerEntry.Status = 200

	serverEntryChan <- masterServerEntry

	return sendPackets
}

func MakeSendPackets(pair models.HostProtocolIdPair, protocolCollection models.ProtocolCollection) (sendPackets []models.Packet) {
	sendPackets = []models.Packet{}

	remoteAddr := pair.RemoteAddr
	protocolId := pair.ProtocolId
	protocol, pOk := protocolCollection.FindById(protocolId)
	if pOk {
		requestPackets := protocol.Base.RequestPackets
		for _, reqPacketDesc := range requestPackets {
			packetId := reqPacketDesc.Id
			makePayloadFunc := protocol.Base.MakePayloadFunc
			if makePayloadFunc != nil {
				newReqPacket := protocol.Base.MakePayloadFunc(models.Packet{Id: packetId, RemoteAddr: remoteAddr, ProtocolId: protocolId}, protocol.Information)
				sendPackets = append(sendPackets, newReqPacket)
			}
		}
	}
	return sendPackets
}

func CheckPrelude(data []byte, prelude []byte) (body []byte, rOk bool) {
	rOk = bytes.Equal(data[:len(prelude)], prelude)
	body = data[len(prelude):]
	return body, rOk
}

func DefaultMasterReceiveHandler() {}

func ParseHandlerPrint(packet models.Packet, messageChan chan<- models.ConsoleMsg) []models.Packet {
	messageChan <- models.ConsoleMsg{Type: 3, Message: string(packet.Data)}
	return nil
}

var ParseBinaryIPv4Entry = func(entryRaw []byte, portLittleEndian bool) (string, error) {
	if len(entryRaw) != 6 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	entry := make([]int, 6)
	for i, v := range entryRaw {
		entry[i] = int(v)
	}
	var a, b, c, d, port int
	a = entry[0]
	b = entry[1]
	c = entry[2]
	d = entry[3]
	if portLittleEndian {
		port = entry[5]<<8 | entry[4]
	} else {
		port = entry[4]<<8 | entry[5]
	}

	if a == 0 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry, nil
}
