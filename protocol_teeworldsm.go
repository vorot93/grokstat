package main

import (
	"bytes"
	"fmt"
)

func TEEWORLDSMMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "servers"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return MasterReceiveHandler(TEEWORLDSMparsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "Teeworlds Master", "RequestPreludeStarter": "\x20\x00\x00\x00\x00\x00\xFF\xFF\xFF\xFF", "RequestPreludeTemplate": "{{.RequestPreludeStarter}}req2", "ResponsePreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "ResponsePreludeTemplate": "{{.ResponsePreludeStarter}}lis2", "DefaultRequestPort": "8300"}}
}

func parseMasterServerEntry(entryRaw []byte) (string, error) {
	if len(entryRaw) != 8 {
		return "", InvalidServerEntryInMasterResponse
	}

	if entryRaw[0] != byte(0xff) || entryRaw[1] != byte(0xff) {
		return "", InvalidServerEntryInMasterResponse
	}

	entry := make([]int, 6)
	for i, v := range entryRaw[2:] {
		entry[i] = int(v)
	}
	a := entry[0]
	b := entry[1]
	c := entry[2]
	d := entry[3]
	port := entry[4]*(16*16) + entry[5]

	if a == 0 {
		return "", InvalidServerEntryInMasterResponse
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry, nil
}

// Parses the response from Teeworlds master server.
func TEEWORLDSMparsePacket(responsePacket Packet, protocolInfo ProtocolEntryInfo) ([]string, error) {
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitter := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	servers := []string{}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return nil, InvalidResponseHeader
	}

	responseBody := response[len(responsePrelude):]
	responseBodySplit := bytes.Split(responseBody, splitter)
	for _, entryRaw := range responseBodySplit {
		serverEntry, entryErr := parseMasterServerEntry(entryRaw)

		if entryErr == nil {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
