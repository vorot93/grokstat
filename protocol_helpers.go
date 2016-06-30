package main

import (
	"bytes"
	"fmt"
)

func MakeRequestPacket(packetId string, protocolInfo ProtocolEntryInfo) (requestPacket Packet) {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	requestPacket = Packet{Data: []byte(ParseTemplate(templ, protocolInfo))}
	return requestPacket
}

func MakePayload(packet Packet, protocolInfo ProtocolEntryInfo) Packet {
	packet.Data = MakeRequestPacket(packet.Id, protocolInfo).Data
	return packet
}

func SimpleReceiveHandler(parseFunc func(Packet, ProtocolEntryInfo) (ServerEntry, error), packet Packet, protColl *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
	sendPackets = []Packet{}

	protocolId := packet.ProtocolId
	protocol, protocolExists := protColl.Get(protocolId)
	if !protocolExists {
		return sendPackets
	}

	protocolInfo := protocol.Information
	remoteIp := packet.RemoteAddr
	serverEntry, sErr := parseFunc(packet, protocolInfo)

	if sErr != nil {
		messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("%s - %s - %s", protocolId, remoteIp, sErr.Error())}
		return sendPackets
	}

	serverEntry.Protocol = protocolId
	serverEntry.Host = remoteIp
	serverEntry.Status = 200

	serverEntryChan <- serverEntry

	return sendPackets
}

func MasterReceiveHandler(parseFunc func(Packet, ProtocolEntryInfo) ([]string, error), packet Packet, protColl *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
	sendPackets = []Packet{}

	protocolId := packet.ProtocolId
	protocol, protocolExists := protColl.Get(protocolId)
	if !protocolExists {
		return sendPackets
	}

	protocolInfo := protocol.Information
	protocolName, _ := protocolInfo["Name"]
	remoteIp := packet.RemoteAddr
	servers, err := parseFunc(packet, protocolInfo)

	if err != nil {
		messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("%s - %s - %s.", protocolId, remoteIp, err.Error())}
		return sendPackets
	}

	masterOf := protocolInfo["MasterOf"]
	for _, ipAddr := range servers {
		pair := HostProtocolIdPair{RemoteAddr: ipAddr, ProtocolId: masterOf}
		protocolMappingInChan <- pair
		sendPackets = append(sendPackets, MakeSendPackets(pair, protColl)...)
	}

	masterServerEntry := MakeServerEntry()
	masterServerEntry.Protocol = packet.ProtocolId
	masterServerEntry.Host = packet.RemoteAddr
	masterServerEntry.Name = fmt.Sprintf("%s Server", protocolName)
	masterServerEntry.Status = 200

	serverEntryChan <- masterServerEntry

	return sendPackets
}

func MakeSendPackets(pair HostProtocolIdPair, protocolCollection *ProtocolCollection) (sendPackets []Packet) {
	sendPackets = []Packet{}

	remoteAddr := pair.RemoteAddr
	protocolId := pair.ProtocolId
	if protocol, exists := protocolCollection.Get(protocolId); exists {
		requestPackets := protocol.Base.RequestPackets
		for _, reqPacketDesc := range requestPackets {
			packetId := reqPacketDesc.Id
			makePayloadFunc := protocol.Base.MakePayloadFunc
			if makePayloadFunc != nil {
				newReqPacket := protocol.Base.MakePayloadFunc(Packet{Id: packetId, RemoteAddr: remoteAddr, ProtocolId: protocolId}, protocol.Information)
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

func ParseHandlerPrint(packet Packet, messageChan chan<- ConsoleMsg) []Packet {
	messageChan <- ConsoleMsg{Type: 3, Message: string(packet.Data)}
	return nil
}

var ParseBinaryIPv4Entry = func(entryRaw []byte, portLittleEndian bool) (string, error) {
	if len(entryRaw) != 6 {
		return "", InvalidServerEntryInMasterResponse
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
		return "", InvalidServerEntryInMasterResponse
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry, nil
}
