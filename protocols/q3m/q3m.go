package q3m

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func MakeRequestPacket(protocolInfo models.ProtocolEntryInfo) []byte {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	return []byte(util.ParseTemplate(templ, protocolInfo))
}

func parseMasterServerEntry(entryRaw []byte) string {
	if len(entryRaw) != 6 {return ""}

	entry := make([]int, 6)
	for i, v := range entryRaw {
		entry[i] = int(v)
	}
	a := entry[0]
	b := entry[1]
	c := entry[2]
	d := entry[3]
	port := entry[4]*(16*16) + entry[5]

	if a == 0 {return ""}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry
}

// Parses the response from Quake III Arena master server.
func ParseResponse(response []byte, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
    responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
    responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitter := []byte{0x5c}

	servers := []string{}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return []string{}, errors.New("Invalid response prelude.")
	}

	responseBody := response[len(responsePrelude):]
	responseBodySplit := bytes.Split(responseBody, splitter)
	for _, entryRaw := range responseBodySplit {
		serverEntry := parseMasterServerEntry(entryRaw)

		if len(serverEntry) > 0 {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
