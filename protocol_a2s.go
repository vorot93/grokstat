package main

import (
	"bytes"
	"fmt"
)

func A2SMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "A2S_INFO"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return SimpleReceiveHandler(A2SparsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server list"}, Information: ProtocolEntryInfo{"Name": "Source Engine Server", "DefaultRequestPort": "27015", "RequestPreludeTemplate": "\xff\xff\xff\xffTSource Engine Query\x00", "ResponsePreludeTemplate": "\xFF\xFF\xFF\xFF"}}
}

func A2SparsePacket(packet Packet, protocolInfo ProtocolEntryInfo) (ServerEntry, error) {
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

	if !preludeOk {
		return ServerEntry{}, InvalidResponseHeader
	}
	entryBuf := bytes.NewBuffer(body)
	serverHeader := entryBuf.Next(1)
	if !bytes.Equal(serverHeader, []byte{0x49}) {
		return ServerEntry{}, InvalidServerHeader
	}
	protocolVer := entryBuf.Next(1)
	serverNameRaw, serverNameErr := entryBuf.ReadBytes(byte(0))
	if serverNameErr != nil {
		return ServerEntry{}, InvalidResponseLength
	}
	serverName := string(bytes.Trim(serverNameRaw, "\x00"))

	mapNameRaw, mapNameErr := entryBuf.ReadBytes(byte(0))
	if mapNameErr != nil {
		return ServerEntry{}, InvalidResponseLength
	}
	mapName := string(bytes.Trim(mapNameRaw, "\x00"))

	folderNameRaw, folderNameErr := entryBuf.ReadBytes(byte(0))
	if folderNameErr != nil {
		return ServerEntry{}, InvalidResponseLength
	}
	folderName := string(bytes.Trim(folderNameRaw, "\x00"))

	modNameRaw, modNameErr := entryBuf.ReadBytes(byte(0))
	if modNameErr != nil {
		return ServerEntry{}, InvalidResponseLength
	}
	modName := string(bytes.Trim(modNameRaw, "\x00"))

	steamAppidLE := entryBuf.Next(2)

	numPlayers := entryBuf.Next(1)
	maxPlayers := entryBuf.Next(1)
	numBots := entryBuf.Next(1)
	serverTypeKW := entryBuf.Next(1)

	var serverType string
	switch string(serverTypeKW) {
	case "d":
		serverType = "dedicated"
	case "l":
		serverType = "non-dedicated"
	case "p":
		serverType = "proxy"
	default:
		return ServerEntry{}, InvalidResponseLength
	}

	serverOSKW := entryBuf.Next(1)
	var serverOS string
	switch string(serverOSKW) {
	case "l":
		serverOS = "linux"
	case "w":
		serverOS = "windows"
	case "m", "o":
		serverOS = "osx"
	default:
		return ServerEntry{}, InvalidResponseLength
	}

	needPassKW := entryBuf.Next(1)
	var needPass bool
	switch int(needPassKW[0]) {
	case 0:
		needPass = false
	case 1:
		needPass = true
	default:
		return ServerEntry{}, InvalidResponseLength
	}

	secureKW := entryBuf.Next(1)
	var secure bool
	switch int(secureKW[0]) {
	case 0:
		secure = false
	case 1:
		secure = true
	default:
		return ServerEntry{}, InvalidResponseLength
	}

	steamAppid := ByteLEToInt64(steamAppidLE)

	var additionalRules map[string]string
	if steamAppid == int64(2400) {
		additionalRules = make(map[string]string)
		additionalRules["theship-mode"] = fmt.Sprint(int(entryBuf.Next(1)[0]))
		additionalRules["theship-witnesses"] = fmt.Sprint(int(entryBuf.Next(1)[0]))
		additionalRules["theship-duration"] = fmt.Sprint(int(entryBuf.Next(1)[0]))
	}

	versionRaw, versionErr := entryBuf.ReadBytes(byte(0))
	if versionErr != nil {
		return ServerEntry{}, InvalidResponseLength
	}
	version := string(bytes.Trim(versionRaw, "\x00"))

	serverEntry := MakeServerEntry()
	serverEntry.Name = serverName
	serverEntry.Terrain = mapName
	serverEntry.ModName = modName
	serverEntry.NumClients = int64(numPlayers[0])
	serverEntry.MaxClients = int64(maxPlayers[0])
	serverEntry.NumBots = int64(numBots[0])
	serverEntry.NeedPass = needPass
	serverEntry.Secure = secure
	serverEntry.Rules["folder-name"] = string(folderName)
	serverEntry.Rules["protocol-version"] = fmt.Sprint(protocolVer)
	serverEntry.Rules["server-type"] = serverType
	serverEntry.Rules["server-os"] = serverOS
	serverEntry.Rules["version"] = version
	if additionalRules != nil {
		for k, v := range additionalRules {
			serverEntry.Rules[k] = v
		}
	}

	return serverEntry, nil
}
