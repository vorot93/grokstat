package teeworldss

import (
	"bytes"
	"math"
	"strconv"
	"strings"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/util"
)

var (
	ProtocolTemplate = models.ProtocolEntry{Base: models.ProtocolEntryBase{MakePayloadFunc: helpers.MakePayload, RequestPackets: []models.RequestPacket{models.RequestPacket{Id: "info"}}, HandlerFunc: Handler, HttpProtocol: "udp", ResponseType: "Server info"}, Information: models.ProtocolEntryInfo{"Name": "Teeworlds Server", "PreludeStarter": "\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF", "PreludeFinisher": "\x00", "RequestPreludeTemplate": "{{.PreludeStarter}}gie3{{.PreludeFinisher}}", "ResponsePreludeTemplate": "{{.PreludeStarter}}inf3", "DefaultRequestPort": "8305"}}
)

func Handler(packet models.Packet, protocolMap map[string]models.ProtocolEntry, messageChan chan<- models.ConsoleMsg, protocolMappingInChan chan<- models.HostProtocolIdPair, serverEntryChan chan<- models.ServerEntry) (sendPackets []models.Packet) {
	return helpers.SimpleReceiveHandler(parsePacket, packet, protocolMap, messageChan, protocolMappingInChan, serverEntryChan)
}

func parsePlayerstring(playerByteArray [][]byte) ([]models.PlayerEntry, error) {
	if math.Mod(float64(len(playerByteArray)), float64(5)) != 0.0 {
		return nil, grokstaterrors.InvalidPlayerStringLength
	}

	playerArray := make([]models.PlayerEntry, 0)

	playerNum := int(len(playerByteArray) / 5)
	for i := 0; i < playerNum; i++ {
		entryRaw := playerByteArray[i*5 : i*5+5]
		playerEntry := models.MakePlayerEntry()
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
		return map[string]string{}, grokstaterrors.InvalidRuleStringLength
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

// Parses the response from Quake III Arena server
func parsePacket(responsePacket models.Packet, protocolInfo models.ProtocolEntryInfo) (models.ServerEntry, error) {
	packetPing := responsePacket.Ping
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterBody := []byte{0x0}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return models.MakeServerEntry(), grokstaterrors.InvalidResponsePrelude
	}

	responseBody := bytes.Trim(response[len(responsePrelude):], string(splitterBody))
	responseBodySplit := bytes.Split(responseBody, splitterBody)

	rulePlayerBoundary := 10

	var ruleByteArray [][]byte
	var playerByteArray [][]byte
	if len(responseBodySplit) >= rulePlayerBoundary {
		ruleByteArray = responseBodySplit[:rulePlayerBoundary]
	}

	if len(responseBodySplit) > rulePlayerBoundary {
		playerByteArray = responseBodySplit[rulePlayerBoundary:]
	}

	if len(responseBodySplit) < rulePlayerBoundary {
		return models.MakeServerEntry(), grokstaterrors.InvalidResponseLength
	}

	players, playerErr := parsePlayerstring(playerByteArray)
	if playerErr != nil {
		return models.MakeServerEntry(), playerErr
	}

	rules, ruleErr := parseRulestring(ruleByteArray)
	if ruleErr != nil {
		return models.MakeServerEntry(), ruleErr
	}

	entry := models.MakeServerEntry()
	entry.Ping = packetPing
	entry.Players = players

	hostName, _ := rules["name"]
	entry.Name = strings.TrimSpace(hostName)

	needPass, _ := rules["flags"]
	entry.NeedPass, _ = strconv.ParseBool(needPass)

	terrain, _ := rules["map"]
	entry.Terrain = strings.TrimSpace(terrain)

	entry.ModName = "Teeworlds"

	gameType, _ := rules["gametype"]
	entry.GameType = strings.TrimSpace(gameType)

	entry.NumClients = int64(len(players))

	maxClients, nc_ok := rules["max_clients"]
	if nc_ok {
		entry.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)
	}

	entry.Secure = false

	entry.Rules = rules

	return entry, nil
}
