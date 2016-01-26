package steam

import (
	"bytes"
	"fmt"
	"math"

	"github.com/grokstat/grokstat/grokstatconstants"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "STEAM_REQUEST"}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server list"}, Information: models.ProtocolEntryInfo{"Name": "Steam Master", "DefaultRequestPort": "27011", "ResponsePreludeTemplate": "\xFF\xFF\xFF\xFF\x66\x0A"}}
)

func makeSteamRequest(lastIp string) []byte {
	return []byte(fmt.Sprintf("\x31\xff%s\x00\x00", lastIp))
}

func MakeSteamRequestPacket(packetId string, protocolInfo models.ProtocolEntryInfo) models.Packet {
	return models.Packet{Data: makeSteamRequest("0.0.0.0:0")}
}

func MakePayload(packet models.Packet, protocolEntryInfo models.ProtocolEntryInfo) models.Packet {
	if packet.Id == "STEAM_REQUEST" {
		packet.Data = makeSteamRequest("0.0.0.0:0")
	}

	return packet
}

func Handler(packet models.Packet, protocolMap map[string]models.ProtocolEntry, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	sendPackets = make([]models.Packet, 0)

	protocol := packet.Protocol
	protocolInfo := protocol.Information
	remoteIp := packet.RemoteAddr

	preludeTemplate, pTOk := protocolInfo["ResponsePreludeTemplate"]
	var body []byte
	var preludeOk bool
	if pTOk {
		prelude := []byte(preludeTemplate)
		body, preludeOk = helpers.CheckPrelude(packet.Data, prelude)
	} else {
		body = packet.Data
		preludeOk = true
	}

	if preludeOk {
		bodyOk := math.Mod(float64(len(body)), 6.0) == 0
		if !bodyOk {
			messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Invalid body length.", remoteIp)}
			return sendPackets
		}

		pairList := []models.HostProtocolIdPair{}

		ipBuf := bytes.NewBuffer(body)
		for {
			ipAddrRaw := ipBuf.Next(6)
			ipAddr, ipErr := helpers.ParseBinaryIPv4Entry(ipAddrRaw, false)
			if ipErr != nil {
				messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Error parsing IP in response.", remoteIp)}
				return sendPackets
			}
			masterOf, mOk := protocolInfo["MasterOf"]
			if mOk {
				pair := models.HostProtocolIdPair{RemoteAddr: ipAddr, ProtocolId: masterOf}
				protocolMappingInChan <- pair
				pairList = append(pairList, pair)
			}
			if ipBuf.Len() == 0 {
				break
			}
		}

		lastIp := pairList[len(pairList)-1].RemoteAddr

		messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: fmt.Sprintf("STEAM - %s - Last IP: %s.", remoteIp, lastIp)}
		if lastIp == "0.0.0.0:0" {
			messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: "STEAM: Query complete."}
		} else {
			sendPacket := models.Packet{Id: "STEAM_REQUEST", Data: []byte(makeSteamRequest(lastIp)), RemoteAddr: remoteIp, ProtocolId: "STEAM"}
			sendPackets = append(sendPackets, sendPacket)
		}

		for _, pair := range pairList {
			sendPackets = append(sendPackets, helpers.MakeSendPackets(pair, protocolMap)...)
		}

		masterServerEntry := models.ServerEntry{Protocol: packet.ProtocolId, Host: packet.RemoteAddr, Name: "Steam Master Server"}
		serverEntryChan <- masterServerEntry

	} else {
		messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("STEAM - %s - Prelude Error", remoteIp)}
	}
	return sendPackets
}
