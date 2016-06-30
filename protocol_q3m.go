package main

import (
	"bytes"
	"math"
)

func Q3MMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "servers"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return MasterReceiveHandler(Q3MParsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "Quake III Arena Master", "SplitterUsed": "true", "PreludeStarter": "\xFF\xFF\xFF\xFF", "RequestQueryParams": "empty full", "RequestPreludeTemplate": "{{.PreludeStarter}}getservers {{.Version}} {{.RequestQueryParams}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}getserversResponse", "Version": "68", "DefaultRequestPort": "27950"}}
}

// Parses the response from Quake III Arena master server.
func Q3MParsePacket(p Packet, protocolInfo ProtocolEntryInfo) ([]string, error) {
	data := p.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	splitterUsed, _ := protocolInfo["SplitterUsed"]

	var header = []byte(ParseTemplate(responsePreludeTemplate, protocolInfo))

	var payload, rOk = CheckPrelude(data, header)

	if !rOk {
		return nil, InvalidResponseHeader
	}

	var servers = []string{}

	var splitter = []byte{0x5c}
	var payloadSplit [][]byte
	if splitterUsed == "true" {
		payloadSplit = bytes.Split(payload, splitter)
	} else {
		if math.Mod(float64(len(payload)), 6.0) != 0.0 {
			return nil, InvalidResponseLength
		}
		for i := 0; i < int(len(payload)/6.0); i++ {
			payloadSplit = append(payloadSplit, payload[i:i+6])
		}
	}
	for _, entryRaw := range payloadSplit {
		var serverEntry, entryErr = ParseBinaryIPv4Entry(entryRaw, false)

		if entryErr == nil {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
