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

func MakeRequestPacket(protocolInfo models.ProtocolEntryInfo) []byte {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	return []byte(util.ParseTemplate(templ, protocolInfo))
}

func parsePlayerstring(playerByteArray [][]byte) (playerArray []models.PlayerEntry, err error) {
    playerArray = make([]models.PlayerEntry, 0, 0)
	for _, playerByteEntry := range playerByteArray {
		byteEntrySplit := bytes.Split(playerByteEntry, []byte(" "))
		if len(byteEntrySplit) < 3 {
			continue
		}
		ping, _ := strconv.ParseInt(string(byteEntrySplit[1]), 10, 64)
		playerEntry := models.PlayerEntry{Name: string(bytes.Join(byteEntrySplit[2:], []byte(" "))), Ping: ping, Info: map[string]string{"Score": string(byteEntrySplit[0])}}
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
func ParseResponse(response []byte, protocolInfo models.ProtocolEntryInfo) (models.ServerEntry, error) {
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterBody := []byte{0xa}
	splitterRules := []byte{0x5c}

	entry := models.ServerEntry{}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return models.ServerEntry{}, errors.New("Invalid response prelude.")
	}

	responseBody := response[len(responsePrelude):]

	ruleString := bytes.Split(responseBody, splitterBody)[1]
	playerByteArray := bytes.Split(responseBody, splitterBody)[2:]

	ruleStringSplit := bytes.Split(ruleString, splitterRules)

	players, playerErr := parsePlayerstring(playerByteArray)
	if playerErr != nil {
		return models.ServerEntry{}, errors.New("Invalid player string.")
	}
	entry.Players = players

	rules, ruleErr := parseRulestring(ruleStringSplit)
	if ruleErr != nil {
		return models.ServerEntry{}, errors.New("Invalid rule string.")
	}
	hostName, _ := rules["sv_hostname"]
	entry.Name = strings.TrimSpace(hostName)

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
