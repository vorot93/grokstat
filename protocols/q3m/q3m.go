package q3m

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func parseMasterServerEntry(entryRaw []byte) string {
	if len(entryRaw) != 6 {
		return ""
	}

	entry := make([]int, 6)
	for i, v := range entryRaw {
		entry[i] = int(v)
	}
	a := entry[0]
	b := entry[1]
	c := entry[2]
	d := entry[3]
	port := entry[4]*(16*16) + entry[5]

	if a == 0 {
		return ""
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry
}

// Parses the response from Quake III Arena master server.
func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
	responsePacket := responsePacketMap["servers"]
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitterUsed, _ := protocolInfo["SplitterUsed"]
	splitter := []byte{0x5c}

	servers := []string{}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return []string{}, errors.New("Invalid response prelude.")
	}

	responseBody := response[len(responsePrelude):]
	var responseBodySplit [][]byte
	if splitterUsed == "true" {
		responseBodySplit = bytes.Split(responseBody, splitter)
	} else {
		if math.Mod(float64(len(responseBody)), 6.0) != 0.0 {
			return []string{}, errors.New("Invalid response length.")
		}
		for i := 0; i < int(len(responseBody)/6.0); i++ {
			responseBodySplit = append(responseBodySplit, responseBody[i:i+6])
		}
	}
	for _, entryRaw := range responseBodySplit {
		serverEntry := parseMasterServerEntry(entryRaw)

		if len(serverEntry) > 0 {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
