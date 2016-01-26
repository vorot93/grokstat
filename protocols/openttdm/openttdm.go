package openttdm

import (
	"bytes"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
)

const (
	_        = iota
	SLT_IPV4 = iota
	SLT_IPV6 = iota
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: helpers.MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "servers4"}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "OpenTTD Master", "DefaultRequestPort": "3978", "ProtocolVer": string(byte(2)), "IPType": string(byte(0)), "RequestPreludeTemplate": "\x05\x00\x06{{.ProtocolVer}}{{.IPType}}"}}
)

func Handler(packet models.Packet, protocolMap map[string]models.ProtocolEntry, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	return helpers.MasterReceiveHandler(parsePacket, packet, protocolMap, messageChan, protocolMappingInChan, serverEntryChan)
}

func parsePacket(responsePacket models.Packet, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
	response := responsePacket.Data
	buf := bytes.NewBuffer(response)

	servers := make([]string, 0)

	for buf.Len() > 0 {
		_ = buf.Next(2)
		responseNum := int(buf.Next(1)[0])
		if responseNum != 7 {
			return nil, grokstaterrors.MalformedPacket
		}
		ipVer := int(buf.Next(1)[0])
		if ipVer == SLT_IPV6 {
			return nil, grokstaterrors.IPv6NotSupported
		}
		hostnumLE := buf.Next(2)
		hostnum := int(hostnumLE[1])<<8 | int(hostnumLE[0])

		if buf.Len() < hostnum*6 {
			return nil, grokstaterrors.MalformedPacket
		}

		for i := 0; i < hostnum; i += 1 {
			entry, entryErr := helpers.ParseBinaryIPv4Entry(buf.Next(6), true)
			if entryErr == nil {
				servers = append(servers, entry)
			}
		}
	}

	return servers, nil
}
