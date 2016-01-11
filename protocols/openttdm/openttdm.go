package openttdm

import (
	"bytes"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols/helpers"
)

const (
	_        = iota
	SLT_IPV4 = iota
	SLT_IPV6 = iota
)

func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) ([]string, error) {
	responsePacket, rOk := responsePacketMap["servers4"]
	if !rOk {
		return nil, grokstaterrors.NoServersResponse
	}
	response := responsePacket.Data
	buf := bytes.NewBuffer(response)

	servers := make([]string, 0)

	for buf.Len() > 0 {
		_ = buf.Next(2)
		responseNum := int(buf.Next(1)[0])
		if responseNum != 7 {
			return nil, grokstaterrors.MalformedPacket
		}
		ipVer := int(buf.Next(1)[0])
		if ipVer == SLT_IPV6 {
			return nil, grokstaterrors.IPv6NotSupported
		}
		hostnumLE := buf.Next(2)
		hostnum := int(hostnumLE[1])<<8 | int(hostnumLE[0])

		if buf.Len() < hostnum*6 {
			return nil, grokstaterrors.MalformedPacket
		}

		for i := 0; i < hostnum; i += 1 {
			entry, entryErr := helpers.ParseBinaryIPv4Entry(buf.Next(6), true)
			if entryErr == nil {
				servers = append(servers, entry)
			}
		}
	}

	return servers, nil
}
