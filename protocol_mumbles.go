package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func MUMBLESMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "ping"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) []Packet {
		return SimpleReceiveHandler(MUMBLESparsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server ping"}, Information: ProtocolEntryInfo{"Name": "Mumble Server", "PreludeStarter": "\x00\x00\x00\x00", "PreludeFinisher": "", "Challenge": "grokstat", "RequestPreludeTemplate": "{{.PreludeStarter}}{{.Challenge}}{{.PreludeFinisher}}", "DefaultRequestPort": "64738"}}
}

func MUMBLESparsePacket(p Packet, info ProtocolEntryInfo) (v ServerEntry, err error) {
	defer func() {
		if r := recover(); r != nil {
			v = MakeServerEntry()
			err = MalformedPacket
		}
	}()
	var challenge *string
	var c, req = info["Challenge"]
	if req {
		challenge = &c
	}
	return MUMBLESparseData(p.Data, challenge)
}

func MUMBLESparseData(b []byte, challenge *string) (v ServerEntry, err error) {
	var data = bytes.NewBuffer(b)

	var protocolVerBytes = data.Next(4)
	var protocolVer = fmt.Sprintf("%d.%d.%d", protocolVerBytes[1], protocolVerBytes[2], protocolVerBytes[3])

	var respChallenge = string(data.Next(8))
	if challenge != nil {
		if *challenge != respChallenge {
			return MakeServerEntry(), InvalidResponseChallenge
		}
	}

	var currentClients uint32
	var currentClientsBytes = data.Next(4)
	_ = binary.Read(bytes.NewReader(currentClientsBytes), binary.BigEndian, &currentClients)

	var maxClients uint32
	var maxClientsBytes = data.Next(4)
	_ = binary.Read(bytes.NewReader(maxClientsBytes), binary.BigEndian, &maxClients)

	var maxBandwidth uint32
	var maxBandwidthBytes = data.Next(4)
	_ = binary.Read(bytes.NewReader(maxBandwidthBytes), binary.BigEndian, &maxBandwidth)

	var rules = map[string]string{}
	rules["protocol-version"] = protocolVer
	rules["current-clients"] = fmt.Sprint(currentClients)
	rules["max-clients"] = fmt.Sprint(maxClients)
	rules["max-bandwidth"] = fmt.Sprint(maxBandwidth)
	if challenge != nil {
		rules["challenge"] = *challenge
	}

	v = MakeServerEntry()
	v.MaxClients = int64(maxClients)
	v.NumClients = int64(currentClients)
	v.Rules = rules

	return v, nil
}
