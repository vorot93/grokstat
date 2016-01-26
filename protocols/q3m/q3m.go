package q3m

import (
	"bytes"
	"math"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/util"
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: helpers.MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "servers"}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena Master", "SplitterUsed": "true", "PreludeStarter": "\xFF\xFF\xFF\xFF", "RequestQueryParams": "empty full", "RequestPreludeTemplate": "{{.PreludeStarter}}getservers {{.Version}} {{.RequestQueryParams}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}getserversResponse", "Version": "68", "DefaultRequestPort": "27950"}}
)

func Handler(packet models.Packet, protocolMap map[string]models.ProtocolEntry, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	return helpers.MasterReceiveHandler(parsePacket, packet, protocolMap, messageChan, protocolMappingInChan, serverEntryChan)
}

// Parses the response from Quake III Arena master server.
func parsePacket(responsePacket models.Packet, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	responseBody, rOk := helpers.CheckPrelude(response, responsePrelude)

	if !rOk {
		return nil, grokstaterrors.InvalidResponsePrelude
	}

	servers := make([]string, 0)

	splitterUsed, _ := protocolInfo["SplitterUsed"]
	splitter := []byte{0x5c}
	var responseBodySplit [][]byte
	if splitterUsed == "true" {
		responseBodySplit = bytes.Split(responseBody, splitter)
	} else {
		if math.Mod(float64(len(responseBody)), 6.0) != 0.0 {
			return nil, grokstaterrors.InvalidResponseLength
		}
		for i := 0; i < int(len(responseBody)/6.0); i++ {
			responseBodySplit = append(responseBodySplit, responseBody[i:i+6])
		}
	}
	for _, entryRaw := range responseBodySplit {
		serverEntry, entryErr := helpers.ParseBinaryIPv4Entry(entryRaw, false)

		if entryErr == nil {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
