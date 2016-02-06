package mumbles

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: helpers.MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "ping"}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server ping"}, Information: models.ProtocolEntryInfo{"Name": "Mumble Server", "PreludeStarter": "\x00\x00\x00\x00", "PreludeFinisher": "", "Challenge": "grokstat", "RequestPreludeTemplate": "{{.PreludeStarter}}{{.Challenge}}{{.PreludeFinisher}}", "DefaultRequestPort": "64738"}}
)

func Handler(packet models.Packet, protocolCollection models.ProtocolCollection, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	return helpers.SimpleReceiveHandler(parsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
}

func parsePacket(infoPacket models.Packet, protocolInfo models.ProtocolEntryInfo) (serverEntry models.ServerEntry, err error) {
	defer func() {
		if r := recover(); r != nil {
			serverEntry = models.MakeServerEntry()
			err = grokstaterrors.MalformedPacket
		}
	}()

	infoData := bytes.NewBuffer(infoPacket.Data)

	protocolVerBytes := infoData.Next(4)
	protocolVer := fmt.Sprintf("%d.%d.%d", protocolVerBytes[1], protocolVerBytes[2], protocolVerBytes[3])

	challenge := string(infoData.Next(8))
	challengeDef, challengeRequired := protocolInfo["Challenge"]
	if challengeRequired {
		if challenge != challengeDef {
			return models.MakeServerEntry(), grokstaterrors.InvalidResponseChallenge
		}
	}

	var currentClients uint32
	currentClientsBytes := infoData.Next(4)
	_ = binary.Read(bytes.NewReader(currentClientsBytes), binary.BigEndian, &currentClients)

	var maxClients uint32
	maxClientsBytes := infoData.Next(4)
	_ = binary.Read(bytes.NewReader(maxClientsBytes), binary.BigEndian, &maxClients)

	var maxBandwidth uint32
	maxBandwidthBytes := infoData.Next(4)
	_ = binary.Read(bytes.NewReader(maxBandwidthBytes), binary.BigEndian, &maxBandwidth)

	rules := make(map[string]string)
	rules["protocol-version"] = protocolVer
	rules["current-clients"] = fmt.Sprint(currentClients)
	rules["max-clients"] = fmt.Sprint(maxClients)
	rules["max-bandwidth"] = fmt.Sprint(maxBandwidth)
	rules["challenge"] = challenge

	serverEntry = models.MakeServerEntry()
	serverEntry.MaxClients = int64(maxClients)
	serverEntry.NumClients = int64(currentClients)
	serverEntry.Rules = rules

	return serverEntry, nil
}
