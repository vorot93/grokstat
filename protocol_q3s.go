package main

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"
)

func Q3SMakeProtocolTemplate() ProtocolEntry {
	return ProtocolEntry{Base: ProtocolEntryBase{MakePayloadFunc: MakePayload, RequestPackets: []RequestPacket{RequestPacket{Id: "status", ResponsePacketNum: 1}}, HandlerFunc: func(packet Packet, protocolCollection *ProtocolCollection, messageChan chan<- ConsoleMsg, protocolMappingInChan chan<- HostProtocolIdPair, serverEntryChan chan<- ServerEntry) (sendPackets []Packet) {
		return SimpleReceiveHandler(Q3SParsePacket, packet, protocolCollection, messageChan, protocolMappingInChan, serverEntryChan)
	}, HttpProtocol: "udp", ResponseType: "Server info"}, Information: ProtocolEntryInfo{"Name": "Quake III Arena", "PreludeStarter": "\xFF\xFF\xFF\xFF", "Challenge": "GrokStat_" + string(time.Now().Unix()), "RequestPreludeTemplate": "{{.PreludeStarter}}getstatus {{.Challenge}}\n", "headerTemplate": "{{.PreludeStarter}}statusResponse", "ServerNameRule": "sv_hostname", "NeedPassRule": "g_needpass", "TerrainRule": "mapname", "ModNameRule": "game", "GameTypeRule": "g_gametype", "MaxClientsRule": "sv_maxclients", "SecureRule": "sv_punkbuster", "Version": "68", "DefaultRequestPort": "27950"}}
}

func Q3SParsePlayerstring(arr [][]byte) []PlayerEntry {
	var v = []PlayerEntry{}
	for _, b := range arr {
		s := bytes.Split(b, []byte(" "))
		if len(s) < 3 {
			continue
		}
		ping, _ := strconv.ParseInt(string(s[1]), 10, 64)
		e := MakePlayerEntry()
		e.Name = strings.Trim(string(bytes.Join(s[2:], []byte(" "))), `"`)
		e.Ping = ping
		e.Info["Score"] = string(s[0])
		v = append(v, e)
	}
	return v
}

func Q3SParseRulestring(str [][]byte) map[string]string {
	var ruleArray = map[int][]string{}
	var m = map[string]string{}
	for i, e := range str {
		vstr := string(e)
		if len(vstr) == 0 {
			continue
		}
		if math.Mod(float64(i), 2) != 0 {
			ruleArray[i] = []string{vstr} // Key
		} else {
			ruleArray[i-1] = append(ruleArray[i-1], vstr) // Value
		}
	}
	for _, e := range ruleArray {
		if len(e) < 2 {
			continue
		}
		var k = e[0]
		var v = e[1]
		m[k] = v
	}

	return m
}

// Parses the response from Quake III Arena server
func Q3SParsePacket(p Packet, info ProtocolEntryInfo) (entry ServerEntry, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = MalformedPacket
		}
	}()
	packetPing := p.Ping
	response := p.Data
	headerTemplate, _ := info["headerTemplate"]
	header := []byte(ParseTemplate(headerTemplate, info))

	sepBody := []byte{0xa}
	sepRules := []byte{0x5c}

	if bytes.Equal(response[:len(header)], header) != true {
		return entry, InvalidResponseHeader
	}

	payload := bytes.Trim(response[len(header):], string(sepBody))
	payloadSplit := bytes.Split(payload, sepBody)

	rulePlayerBoundary := len(payloadSplit)
	for i, line := range payloadSplit {
		if line[0] != sepRules[0] {
			rulePlayerBoundary = i
			break
		}
	}

	ruleByteArray := bytes.Join(payloadSplit[:rulePlayerBoundary], sepRules)
	playerByteArray := payloadSplit[rulePlayerBoundary:]

	ruleByteArraySplit := bytes.Split(ruleByteArray, sepRules)

	var players = Q3SParsePlayerstring(playerByteArray)
	var rules = Q3SParseRulestring(ruleByteArraySplit)

	entry = MakeServerEntry()
	entry.Ping = packetPing
	entry.Players = players
	entry.NumClients = int64(len(players))
	entry.Rules = rules

	serverNameRule, serverNameRuleOk := info["ServerNameRule"]
	if serverNameRuleOk {
		serverName, _ := rules[serverNameRule]
		entry.Name = strings.TrimSpace(serverName)
	}

	needPassRule, needPassRuleOk := info["NeedPassRule"]
	if needPassRuleOk {
		needPass, _ := rules[needPassRule]
		entry.NeedPass, _ = strconv.ParseBool(needPass)
	}

	terrainRule, terrainRuleOk := info["TerrainRule"]
	if terrainRuleOk {
		terrain, _ := rules[terrainRule]
		entry.Terrain = strings.TrimSpace(terrain)
	}

	modNameRule, modNameRuleOk := info["ModNameRule"]
	if modNameRuleOk {
		modName, _ := rules[modNameRule]
		entry.ModName = strings.TrimSpace(modName)
	}

	gameTypeRule, gameTypeRuleOk := info["GameTypeRule"]
	if gameTypeRuleOk {
		gameType, _ := rules[gameTypeRule]
		entry.GameType = strings.TrimSpace(gameType)
	}

	secureRule, secureRuleOk := info["SecureRule"]
	if secureRuleOk {
		secure, _ := rules[secureRule]
		entry.Secure, _ = strconv.ParseBool(secure)
	}

	maxClientsRule, maxClientsRuleOk := info["MaxClientsRule"]
	if maxClientsRuleOk {
		maxClients, maxClientsOk := rules[maxClientsRule]
		if maxClientsOk {
			entry.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)
		}
	}

	numBotsRule, numBotsRuleOk := info["NumBotsRule"]
	if numBotsRuleOk {
		numBots, numBotsOk := rules[numBotsRule]
		if numBotsOk {
			entry.NumBots, _ = strconv.ParseInt(strings.TrimSpace(numBots), 10, 64)
		}
	}

	return entry, nil
}
