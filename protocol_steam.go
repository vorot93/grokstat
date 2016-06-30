package main

import (
	"bytes"
	"fmt"
	"math"
)

func STEAMMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakeSteamPayload, RequestPackets: []RequestPacket{RequestPacket{Id: "STEAM_REQUEST"}}, HandlerFunc: SteamHandler, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "Steam Master", "DefaultRequestPort": "27011", "ResponsePreludeTemplate": "\xFF\xFF\xFF\xFF\x66\x0A"}}
}

func makeSteamRequest(lastIp string) []byte {
	return []byte(fmt.Sprintf("\x31\xff%s\x00\x00", lastIp))
}

func MakeSteamRequestPacket(packetId string, protocolInfo ProtocolEntryInfo) Packet {
	return Packet{Data: makeSteamRequest("0.0.0.0:0")}
}

func MakeSteamPayload(packet Packet, protocolEntryInfo ProtocolEntryInfo) Packet {
	if packet.Id == "STEAM_REQUEST" {
		packet.Data = makeSteamRequest("0.0.0.0:0")
	}

	return packet
}

func SteamHandler(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
	sendPackets = make([]Packet, 0)

	protocolId := packet.ProtocolId
	protocol, protocolExists := protocolCollection.Get(protocolId)
	if !protocolExists {
		return sendPackets
	}
	protocolInfo := protocol.Information
	remoteIp := packet.RemoteAddr

	preludeTemplate, pTOk := protocolInfo["ResponsePreludeTemplate"]
	var body []byte
	var preludeOk bool
	if pTOk {
		prelude := []byte(preludeTemplate)
		body, preludeOk = CheckPrelude(packet.Data, prelude)
	} else {
		body = packet.Data
		preludeOk = true
	}

	if preludeOk {
		bodyOk := math.Mod(float64(len(body)), 6.0) == 0
		if !bodyOk {
			messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Invalid body length.", remoteIp)}
			return sendPackets
		}

		pairList := []HostProtocolIdPair{}

		ipBuf := bytes.NewBuffer(body)
		for {
			ipAddrRaw := ipBuf.Next(6)
			ipAddr, ipErr := ParseBinaryIPv4Entry(ipAddrRaw, false)
			if ipErr != nil {
				messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Error parsing IP in response.", remoteIp)}
				return sendPackets
			}
			masterOf, mOk := protocolInfo["MasterOf"]
			if mOk {
				pair := HostProtocolIdPair{RemoteAddr: ipAddr, ProtocolId: masterOf}
				protocolMappingInChan <- pair
				pairList = append(pairList, pair)
			}
			if ipBuf.Len() == 0 {
				break
			}
		}

		lastIp := pairList[len(pairList)-1].RemoteAddr

		messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: fmt.Sprintf("STEAM - %s - Last IP: %s.", remoteIp, lastIp)}
		if lastIp == "0.0.0.0:0" {
			messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: "STEAM: Query complete."}
		} else {
			sendPacket := Packet{Id: "STEAM_REQUEST", Data: []byte(makeSteamRequest(lastIp)), RemoteAddr: remoteIp, ProtocolId: "STEAM"}
			sendPackets = append(sendPackets, sendPacket)
		}

		for _, pair := range pairList {
			sendPackets = append(sendPackets, MakeSendPackets(pair, protocolCollection)...)
		}

		masterServerEntry := ServerEntry{Protocol: packet.ProtocolId, Host: packet.RemoteAddr, Name: "Steam Master Server"}
		serverEntryChan <- masterServerEntry

	} else {
		messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Prelude Error", remoteIp)}
	}
	return sendPackets
}
