package helpers

import (
	"bytes"
	"fmt"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

var MakeRequestPacket = func(packetId string, protocolInfo models.ProtocolEntryInfo) (requestPacket models.Packet) {
	templ, _ := protocolInfo["RequestPreludeTemplate"]
	requestPacket = models.Packet{Data: []byte(util.ParseTemplate(templ, protocolInfo))}
	return requestPacket
}

var CheckPrelude = func(data []byte, prelude []byte) (body []byte, rOk bool) {
	rOk = bytes.Equal(data[:len(prelude)], prelude)
	body = data[len(prelude):]
	return body, rOk
}

var ParseBinaryIPv4Entry = func(entryRaw []byte, portLittleEndian bool) (string, error) {
	if len(entryRaw) != 6 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	entry := make([]int, 6)
	for i, v := range entryRaw {
		entry[i] = int(v)
	}
	var a, b, c, d, port int
	a = entry[0]
	b = entry[1]
	c = entry[2]
	d = entry[3]
	if portLittleEndian {
		port = entry[5]<<8 | entry[4]
	} else {
		port = entry[4]<<8 | entry[5]
	}

	if a == 0 {
		return "", grokstaterrors.InvalidServerEntryInMasterResponse
	}

	serverEntry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return serverEntry, nil
}
