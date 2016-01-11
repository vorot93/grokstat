package openttds

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func ParseResponseMap(responsePacketMap map[string]models.Packet, protocolInfo models.ProtocolEntryInfo) (serverEntry models.ServerEntry, err error) {
	defer func() {
		if r := recover(); r != nil {
			serverEntry = models.MakeServerEntry()
			err = grokstaterrors.MalformedPacket
		}
	}()

	infoPacket, infoPacketOk := responsePacketMap["info"]
	if !infoPacketOk {
		return models.MakeServerEntry(), grokstaterrors.NoInfoResponse
	}

	infoData := bytes.NewBuffer(infoPacket.Data[3:])

	protocolVer := int(infoData.Next(1)[0])

	var activeNewGRFsNum int
	var activeNewGRFsInfo string
	if protocolVer >= 4 {
		activeNewGRFsNum = int(infoData.Next(1)[0])
		for n := 0; n < activeNewGRFsNum; n += 1 {
			NewGRFID := util.GetByteString(infoData.Next(4))
			NewGRFMD5 := util.GetByteString(infoData.Next(16))
			activeNewGRFsInfo = activeNewGRFsInfo + fmt.Sprintf("ID:%s/MD5:%s; ", NewGRFID, NewGRFMD5)
		}
		activeNewGRFsInfo = strings.Trim(activeNewGRFsInfo, " ;")
	}

	var timeCurrent uint32
	var timeStart uint32
	if protocolVer >= 3 {
		_ = binary.Read(bytes.NewReader(infoData.Next(4)), binary.BigEndian, &timeCurrent)
		_ = binary.Read(bytes.NewReader(infoData.Next(4)), binary.BigEndian, &timeStart)
	}

	var maxCompanies int
	var currentCompanies int
	var maxSpectators int
	if protocolVer >= 2 {
		maxCompanies = int(infoData.Next(1)[0])
		currentCompanies = int(infoData.Next(1)[0])
		maxSpectators = int(infoData.Next(1)[0])
	}
	serverNameBytes, _ := infoData.ReadBytes(byte(0))
	serverName := string(bytes.Trim(serverNameBytes, "\x00"))

	serverVersionBytes, _ := infoData.ReadBytes(byte(0))
	serverVersion := string(bytes.Trim(serverVersionBytes, "\x00"))

	languageId := int(infoData.Next(1)[0])
	needPass := bool(infoData.Next(1)[0] != 0)
	maxClients := int(infoData.Next(1)[0])
	currentClients := int(infoData.Next(1)[0])
	currentSpectators := int(infoData.Next(1)[0])

	if protocolVer < 3 {
		_ = infoData.Next(2)
		_ = infoData.Next(2)
	}

	mapNameBytes, _ := infoData.ReadBytes(byte(0))
	mapName := string(bytes.Trim(mapNameBytes, "\x00"))

	var mapWidth uint16
	_ = binary.Read(bytes.NewReader(infoData.Next(2)), binary.BigEndian, &mapWidth)

	var mapHeight uint16
	_ = binary.Read(bytes.NewReader(infoData.Next(2)), binary.BigEndian, &mapHeight)

	mapSet := int(infoData.Next(1)[0])
	dedicatedServer := int(infoData.Next(1)[0])

	rules := make(map[string]string)
	rules["protocol-version"] = fmt.Sprint(protocolVer)
	rules["active-newgrfs-num"] = fmt.Sprint(activeNewGRFsNum)
	rules["active-newgrfs"] = fmt.Sprint(activeNewGRFsInfo)
	rules["time-current"] = fmt.Sprint(timeCurrent)
	rules["time-start"] = fmt.Sprint(timeStart)
	rules["max-companies"] = fmt.Sprint(maxCompanies)
	rules["current-companies"] = fmt.Sprint(currentCompanies)
	rules["max-spectators"] = fmt.Sprint(maxSpectators)
	rules["server-name"] = fmt.Sprint(serverName)
	rules["server-version"] = fmt.Sprint(serverVersion)
	rules["language-id"] = fmt.Sprint(languageId)
	rules["need-pass"] = fmt.Sprint(needPass)
	rules["max-clients"] = fmt.Sprint(maxClients)
	rules["current-clients"] = fmt.Sprint(currentClients)
	rules["current-spectators"] = fmt.Sprint(currentSpectators)
	rules["map-name"] = fmt.Sprint(mapName)
	rules["map-set"] = fmt.Sprint(mapSet)
	rules["dedicated"] = fmt.Sprint(dedicatedServer)

	serverEntry = models.ServerEntry{Name: string(serverName), MaxClients: int64(maxClients), NumClients: int64(currentClients), NeedPass: bool(needPass), Terrain: string(mapName), Rules: rules, Players: []models.PlayerEntry{}}
	return serverEntry, nil
}
