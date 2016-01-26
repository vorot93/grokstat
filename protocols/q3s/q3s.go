package q3s

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/util"
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: helpers.MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "status", ResponsePacketNum: 1}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Quake III Arena", "PreludeStarter": "\xFF\xFF\xFF\xFF", "Challenge": "GrokStat_" + string(time.Now().Unix()), "RequestPreludeTemplate": "{{.PreludeStarter}}getstatus {{.Challenge}}\n", "ResponsePreludeTemplate": "{{.PreludeStarter}}statusResponse", "ServerNameRule": "sv_hostname", "NeedPassRule": "g_needpass", "TerrainRule": "mapname", "ModNameRule": "game", "GameTypeRule": "g_gametype", "MaxClientsRule": "sv_maxclients", "SecureRule": "sv_punkbuster", "Version": "68", "DefaultRequestPort": "27950"}}
)

func Handler(packet models.Packet, protocolMap map[string]models.ProtocolEntry, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	return helpers.SimpleReceiveHandler(parsePacket, packet, protocolMap, messageChan, protocolMappingInChan, serverEntryChan)
}

func parsePlayerstring(playerByteArray [][]byte) ([]models.PlayerEntry, error) {
	playerArray := make([]models.PlayerEntry, 0)
	for _, playerByteEntry := range playerByteArray {
		byteEntrySplit := bytes.Split(playerByteEntry, []byte(" "))
		if len(byteEntrySplit) < 3 {
			continue
		}
		ping, _ := strconv.ParseInt(string(byteEntrySplit[1]), 10, 64)
		playerEntry := models.MakePlayerEntry()
		playerEntry.Name = strings.Trim(string(bytes.Join(byteEntrySplit[2:], []byte(" "))), `"`)
		playerEntry.Ping = ping
		playerEntry.Info["Score"] = string(byteEntrySplit[0])
		playerArray = append(playerArray, playerEntry)
	}
	return playerArray, nil
}

func parseRulestring(rulestring [][]byte) (map[string]string, error) {
	ruleArray := make(map[int][]string)
	ruleMap := make(map[string]string)
	for i, v := range rulestring {
		vstr := string(v)
		if len(vstr) == 0 {
			continue
		}
		if math.Mod(float64(i), 2) != 0 {
			ruleArray[i] = []string{vstr} // Key
		} else {
			ruleArray[i-1] = append(ruleArray[i-1], vstr) // Value
		}
	}
	for _, v := range ruleArray {
		if len(v) < 2 {
			continue
		}
		key := v[0]
		value := v[1]
		ruleMap[key] = value
	}

	return ruleMap, nil
}

// Parses the response from Quake III Arena server
func parsePacket(responsePacket models.Packet, protocolInfo models.ProtocolEntryInfo) (models.ServerEntry, error) {
	packetPing := responsePacket.Ping
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterBody := []byte{0xa}
	splitterRules := []byte{0x5c}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return models.MakeServerEntry(), grokstaterrors.InvalidResponsePrelude
	}

	responseBody := bytes.Trim(response[len(responsePrelude):], string(splitterBody))
	responseBodySplit := bytes.Split(responseBody, splitterBody)

	rulePlayerBoundary := len(responseBodySplit)
	for i, line := range responseBodySplit {
		if line[0] != splitterRules[0] {
			rulePlayerBoundary = i
			break
		}
	}

	ruleByteArray := bytes.Join(responseBodySplit[:rulePlayerBoundary], splitterRules)
	playerByteArray := responseBodySplit[rulePlayerBoundary:]

	ruleByteArraySplit := bytes.Split(ruleByteArray, splitterRules)

	players, playerErr := parsePlayerstring(playerByteArray)
	if playerErr != nil {
		return models.MakeServerEntry(), playerErr
	}

	rules, ruleErr := parseRulestring(ruleByteArraySplit)
	if ruleErr != nil {
		return models.MakeServerEntry(), ruleErr
	}

	entry := models.MakeServerEntry()
	entry.Ping = packetPing
	entry.Players = players
	entry.NumClients = int64(len(players))
	entry.Rules = rules

	serverNameRule, serverNameRuleOk := protocolInfo["ServerNameRule"]
	if serverNameRuleOk {
		serverName, _ := rules[serverNameRule]
		entry.Name = strings.TrimSpace(serverName)
	}

	needPassRule, needPassRuleOk := protocolInfo["NeedPassRule"]
	if needPassRuleOk {
		needPass, _ := rules[needPassRule]
		entry.NeedPass, _ = strconv.ParseBool(needPass)
	}

	terrainRule, terrainRuleOk := protocolInfo["TerrainRule"]
	if terrainRuleOk {
		terrain, _ := rules[terrainRule]
		entry.Terrain = strings.TrimSpace(terrain)
	}

	modNameRule, modNameRuleOk := protocolInfo["ModNameRule"]
	if modNameRuleOk {
		modName, _ := rules[modNameRule]
		entry.ModName = strings.TrimSpace(modName)
	}

	gameTypeRule, gameTypeRuleOk := protocolInfo["GameTypeRule"]
	if gameTypeRuleOk {
		gameType, _ := rules[gameTypeRule]
		entry.GameType = strings.TrimSpace(gameType)
	}

	secureRule, secureRuleOk := protocolInfo["SecureRule"]
	if secureRuleOk {
		secure, _ := rules[secureRule]
		entry.Secure, _ = strconv.ParseBool(secure)
	}

	maxClientsRule, maxClientsRuleOk := protocolInfo["MaxClientsRule"]
	if maxClientsRuleOk {
		maxClients, maxClientsOk := rules[maxClientsRule]
		if maxClientsOk {
			entry.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)
		}
	}

	return entry, nil
}
