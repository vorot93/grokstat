package teeworldss

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func parsePlayerstring(playerByteArray [][]byte) ([]models.PlayerEntry, error) {
	if math.Mod(float64(len(playerByteArray)), float64(5)) != 0.0 {
		return []models.PlayerEntry{}, errors.New("Invalid array length.")
	}

	playerArray := make([]models.PlayerEntry, 0, 0)

	playerNum := int(len(playerByteArray) / 5)
	for i := 0; i < playerNum; i++ {
		entryRaw := playerByteArray[i*5 : i*5+5]
		playerEntry := models.PlayerEntry{Info: make(map[string]string)}
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
		return map[string]string{}, errors.New("Invalid string length")
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
func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) (models.ServerEntry, error) {
	responsePacket, rpm_ok := responsePacketMap["info"]
	if !rpm_ok {
		return models.ServerEntry{}, errors.New("No info response.")
	}
	packetPing := responsePacket.Ping
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterBody := []byte{0x0}

	entry := models.ServerEntry{Ping: packetPing}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return models.ServerEntry{}, errors.New("Invalid response prelude.")
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
		return models.ServerEntry{}, errors.New("Invalid response packet length.")
	}

	players, playerErr := parsePlayerstring(playerByteArray)
	if playerErr != nil {
		return models.ServerEntry{}, errors.New("Invalid player string.")
	}
	entry.Players = players

	rules, ruleErr := parseRulestring(ruleByteArray)
	if ruleErr != nil {
		return models.ServerEntry{}, errors.New("Invalid rule string.")
	}
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
