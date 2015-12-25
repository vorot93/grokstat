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

func parseRulestring(rulestring [][]byte) (map[string]string, error) {
    ruleArray := make(map[int][]string)
    ruleMap := make(map[string]string)
    for i, v := range rulestring {
        vstr := string(v)
        if len(vstr) == 0 {continue}
        if math.Mod(float64(i), 2) != 0 {
            ruleArray[i] = []string{vstr}  // Key
        } else {
            ruleArray[i-1] = append(ruleArray[i-1], vstr)  // Value
        }
    }
    for _, v := range ruleArray {
        if len(v) < 2 {continue}
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

    splitter := []byte{0x5c}

    entry := models.ServerEntry{}

    if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
        return models.ServerEntry{}, errors.New("Invalid response prelude.")
    }

    responseBody := response[len(responsePrelude):]
	responseBodySplit := bytes.Split(responseBody, splitter)

    rules, ruleErr := parseRulestring(responseBodySplit)
    if ruleErr != nil {
        return models.ServerEntry{}, errors.New("Invalid rule string.")
    }
    hostName, _ := rules["hostname"]
    entry.Name = strings.TrimSpace(hostName)

    numClients, nc_ok := rules["clients"]
    if nc_ok {entry.NumClients, _ = strconv.ParseInt(strings.TrimSpace(numClients), 10, 64)}

    maxClients, nc_ok := rules["sv_maxclients"]
    if nc_ok {entry.MaxClients, _ = strconv.ParseInt(strings.TrimSpace(maxClients), 10, 64)}

    secure, nc_ok := rules["sv_punkbuster"]
    if strings.TrimSpace(secure) == "1" {entry.Secure = true}

    entry.Rules = rules

    return entry, nil
}
