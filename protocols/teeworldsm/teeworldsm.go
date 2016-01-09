package teeworldsm

import (
	"bytes"
	"fmt"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func parseMasterServerEntry(entryRaw []byte) (string, error) {
	if len(entryRaw) != 8 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	if entryRaw[0] != byte(0xff) || entryRaw[1] != byte(0xff) {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	entry := make([]int, 6)
	for i, v := range entryRaw[2:] {
		entry[i] = int(v)
	}
	a := entry[0]
	b := entry[1]
	c := entry[2]
	d := entry[3]
	port := entry[4]*(16*16) + entry[5]

	if a == 0 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry, nil
}

// Parses the response from Teeworlds master server.
func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
	responsePacket := responsePacketMap["servers"]
	response := responsePacket.Data
	responsePreludeTemplate, _ := protocolInfo["ResponsePreludeTemplate"]
	responsePrelude := []byte(util.ParseTemplate(responsePreludeTemplate, protocolInfo))

	splitter := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	servers := []string{}

	if bytes.Equal(response[:len(responsePrelude)], responsePrelude) != true {
		return []string{}, grokstaterrors.InvalidResponsePrelude
	}

	responseBody := response[len(responsePrelude):]
	responseBodySplit := bytes.Split(responseBody, splitter)
	for _, entryRaw := range responseBodySplit {
		serverEntry, entryErr := parseMasterServerEntry(entryRaw)

		if entryErr == nil {
			servers = append(servers, serverEntry)
		}
	}
	return servers, nil
}
