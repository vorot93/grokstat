package main

import (
	"bytes"
	"math"
	"strconv"
	"strings"
)

func TEEWORLDSSMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "info"}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return SimpleReceiveHandler(TEEWORLDSSparsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server info"}, Information: ProtocolEntryInfo{"Name": "Teeworlds Server", "PreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "PreludeFinisher": "\x00", "RequestPreludeTemplate": "{{.PreludeStarter}}gie3{{.PreludeFinisher}}", "ResponsePreludeTemplate": "{{.PreludeStarter}}inf3", "DefaultRequestPort": "8305"}}
}

func parsePlayerstring(playerByteArray [][]byte) ([]PlayerEntry, error) {
	if math.Mod(float64(len(playerByteArray)), float64(5)) != 0.0 {
		return nil, InvalidPlayerStringLength
	}

	var playerArray = []PlayerEntry{}

	playerNum := int(len(playerByteArray) / 5)
	for i := 0; i < playerNum; i++ {
		entryRaw := playerByteArray[i*5 : i*5+5]
		playerEntry := MakePlayerEntry()
		playerEntry.Name = string(entryRaw[0])
		playerEntry.Info["clan"] = string(entryRaw[1])
		playerEntry.Info["country"] = string(entryRaw[2])
		playerEntry.Info["score"] = string(entryRaw[3])
		playerEntry.Info["is_player"] = string(entryRaw[4])

		playerArray = append(playerArray, playerEntry)
	}
	return playerArray, nil
}

func parseRulestring(rulestring [][]byte) (map[string]string, error) {
	if len(rulestring) < 10 {
		return map[string]string{}, InvalidRuleStringLength
	}

	ruleMap := make(map[string]string)

	ruleMap["token"] = string(rulestring[0])
	ruleMap["version"] = string(rulestring[1])
	ruleMap["name"] = string(rulestring[2])
	ruleMap["map"] = string(rulestring[3])
	ruleMap["gametype"] = string(rulestring[4])
	ruleMap["flags"] = string(rulestring[5])
	ruleMap["num_players"] = string(rulestring[6])
	ruleMap["max_players"] = string(rulestring[7])
	ruleMap["num_clients"] = string(rulestring[8])
	ruleMap["max_clients"] = string(rulestring[9])

	return ruleMap, nil
}

func TEEWORLDSSParseData(data [][]byte) (ServerEntry, error) {
	var v = MakeServerEntry()

	rulePlayerBoundary := 10

	var ruleByteArray [][]byte
	var playerByteArray [][]byte
	if len(data) >= rulePlayerBoundary {
		ruleByteArray = data[:rulePlayerBoundary]
	}

	if len(data) > rulePlayerBoundary {
		playerByteArray = data[rulePlayerBoundary:]
	}

	if len(data) < rulePlayerBoundary {
		return MakeServerEntry(), InvalidResponseLength
	}

	players, playerErr := parsePlayerstring(playerByteArray)
	if playerErr != nil {
		return MakeServerEntry(), playerErr
	}

	rules, ruleErr := parseRulestring(ruleByteArray)
	if ruleErr != nil {
		return MakeServerEntry(), ruleErr
	}

	v.Players = players

	hostName, _ := rules["name"]
	v.Name = strings.TrimSpace(hostName)

	needPass, _ := rules["flags"]
	v.NeedPass, _ = strconv.ParseBool(needPass)

	terrain, _ := rules["map"]
	v.Terrain = strings.TrimSpace(terrain)

	v.ModName = "Teeworlds"

	gameType, _ := rules["gametype"]
	v.GameType = strings.TrimSpace(gameType)

	v.NumClients = int64(len(players))

	maxClients, nc_ok := rules["max_clients"]
	if nc_ok {
		v.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)
	}

	v.Secure = false

	v.Rules = rules

	return v, nil
}

// Parses the response from Quake III Arena server
func TEEWORLDSSparsePacket(p Packet, protocolInfo ProtocolEntryInfo) (ServerEntry, error) {
	packetPing := p.Ping
	response := p.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(ParseTemplate(responsePreludeTemplate, protocolInfo))

	var sep = []byte{0x0}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return MakeServerEntry(), InvalidResponseHeader
	}

	b := bytes.Trim(response[len(responsePrelude):], string(sep))

	var entry, err = TEEWORLDSSParseData(bytes.Split(b, sep))
	if err != nil {
		return entry, err
	}

	entry.Ping = packetPing

	return entry, nil
}
