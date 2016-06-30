package main

import "bytes"

const (
	_        = iota
	SLT_IPV4 = iota
	SLT_IPV6 = iota
)

func OPENTTDMMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "servers4"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return MasterReceiveHandler(OPENTTDMparsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "OpenTTD Master", "DefaultRequestPort": "3978", "ProtocolVer": string(byte(2)), "IPType": string(byte(0)), "RequestPreludeTemplate": "\x05\x00\x06{{.ProtocolVer}}{{.IPType}}"}}
}

func OPENTTDMparsePacket(p Packet, i ProtocolEntryInfo) ([]string, error) {
	return OPENTTDMparseData(p.Data)
}

func OPENTTDMparseData(data []byte) ([]string, error) {
	buf := bytes.NewBuffer(data)

	var servers = []string{}

	for buf.Len() > 0 {
		_ = buf.Next(2)
		var responseNum = int(buf.Next(1)[0])
		if responseNum != 7 {
			return nil, MalformedPacket
		}
		var ipVer = int(buf.Next(1)[0])
		if ipVer == SLT_IPV6 {
			return nil, IPv6NotSupported
		}
		var hostnumLE = buf.Next(2)
		var hostnum = int(hostnumLE[1])<<8 | int(hostnumLE[0])

		if buf.Len() < hostnum*6 {
			return nil, MalformedPacket
		}

		for i := 0; i < hostnum; i++ {
			entry, entryErr := ParseBinaryIPv4Entry(buf.Next(6), true)
			if entryErr == nil {
				servers = append(servers, entry)
			}
		}
	}

	return servers, nil
}
