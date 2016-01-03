package q3s

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func parsePlayerstring(playerByteArray [][]byte) (playerArray []models.PlayerEntry, err error) {
	playerArray = make([]models.PlayerEntry, 0, 0)
	for _, playerByteEntry := range playerByteArray {
		byteEntrySplit := bytes.Split(playerByteEntry, []byte(" "))
		if len(byteEntrySplit) < 3 {
			continue
		}
		ping, _ := strconv.ParseInt(string(byteEntrySplit[1]), 10, 64)
		playerEntry := models.PlayerEntry{Name: strings.Trim(string(bytes.Join(byteEntrySplit[2:], []byte(" "))), `"`), Ping: ping, Info: map[string]string{"Score": string(byteEntrySplit[0])}}
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
func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) (models.ServerEntry, error) {
	responsePacket, rpm_ok := responsePacketMap["status"]
	if !rpm_ok {
		return models.ServerEntry{}, errors.New("No status response.")
	}
	packetPing := responsePacket.Ping
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterBody := []byte{0xa}
	splitterRules := []byte{0x5c}

	entry := models.ServerEntry{Ping: packetPing}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return models.ServerEntry{}, errors.New("Invalid response prelude.")
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
		return models.ServerEntry{}, errors.New("Invalid player string.")
	}
	entry.Players = players

	rules, ruleErr := parseRulestring(ruleByteArraySplit)
	if ruleErr != nil {
		return models.ServerEntry{}, errors.New("Invalid rule string.")
	}
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

	terrain, _ := rules["mapname"]
	entry.Terrain = strings.TrimSpace(terrain)

	game, gameOk := rules["game"]
	if gameOk {
		entry.ModName = strings.TrimSpace(game)
	} else {
		gamename, _ := rules["gamename"]
		entry.ModName = strings.TrimSpace(gamename)
	}

	gameType, _ := rules["g_gametype"]
	entry.GameType = strings.TrimSpace(gameType)

	entry.NumClients = int64(len(players))

	maxClients, nc_ok := rules["sv_maxclients"]
	if nc_ok {
		entry.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)
	}

	secure, nc_ok := rules["sv_punkbuster"]
	if strings.TrimSpace(secure) == "1" {
		entry.Secure = true
	}

	entry.Rules = rules

	return entry, nil
}
