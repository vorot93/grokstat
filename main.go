/*
grokstat is a tool for querying game servers for various information: server list, player count, active map etc

The program takes protocol name and remote ip address as arguments, fetches information from the remote server, parses it and outputs back as JSON. As convenience the status and message are also provided.

grokstat uses JSON input instead of command line flags. The JSON input is structured as follows:
	hosts - array of strings - hosts to query
	protocol - string - protocol to use
	show-protocols - boolean - if true, show protocols and exit
	output-lvl - int - tune the output from bare JSON to full-fledged debug
	custom-config-path - path of custom config file to be used
*/
package main

//go:generate go-bindata -o "bindata/bindata.go" -pkg "bindata" "data/..."

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"

	"github.com/grokstat/grokstat/bindata"
	"github.com/grokstat/grokstat/grokstatconstants"
	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/network"
	"github.com/grokstat/grokstat/protocols"
	"github.com/grokstat/grokstat/protocols/helpers"
	"github.com/grokstat/grokstat/util"
)

type ServerResponseStruct struct {
	Hostname    string
	ResponseMap map[string]models.Packet
	ResponseErr error
}

type InputData struct {
	Hosts            []string `json:"hosts"`
	Protocol         string   `json:"protocol"`
	ShowProtocols    bool     `json:"show-protocols"`
	OutputLvl        int      `json:"output-lvl"`
	CustomConfigPath string   `json:"custom-config-path"`
}

type ConfigFile struct {
	Protocols []protocols.ProtocolConfig `toml:"Protocols"`
}

type JsonResponse struct {
	Version string      `json:"version"`
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Flags   InputData   `json:"input-flags"`
	Output  interface{} `json:"output"`
}

type PacketErrorPair struct {
	Packet models.Packet
	Error  error
}

func MakePacketErrorPair(hosts []models.HostProtocolIdPair, protColl models.ProtocolCollection) (packErrPairs []PacketErrorPair) {
	packErrPairs = []PacketErrorPair{}

	for _, hostpair := range hosts {
		var hostpackets = []models.Packet{}
		var err error = nil

		hostport := strings.Split(hostpair.RemoteAddr, ":")
		protocolId := hostpair.ProtocolId
		protocol, protocolExists := protColl.FindById(protocolId)
		if protocolExists {
			host := hostport[0]
			var port string
			if len(hostport) < 2 {
				port, _ = protocol.Information["DefaultRequestPort"]
			} else {
				port = hostport[1]
			}
			ipAddr, rErr := net.ResolveIPAddr("ip4", host)
			if rErr == nil {
				addrFinal := strings.Join([]string{ipAddr.String(), port}, ":")

				reqPackets := helpers.MakeSendPackets(models.HostProtocolIdPair{RemoteAddr: addrFinal, ProtocolId: protocolId}, protColl)

				for _, reqPacket := range reqPackets {
					hostpackets = append(hostpackets, reqPacket)
				}
			} else {
				err = rErr
			}
		} else {
			err = grokstaterrors.InvalidProtocol
		}

		for _, packetFinal := range hostpackets {
			packErrPairs = append(packErrPairs, PacketErrorPair{Packet: packetFinal, Error: err})
		}
	}

	return packErrPairs
}

// Forms a JSON string out of Grokstat output.
var FormJsonResponse = func(output interface{}, err error, flags InputData) (string, error) {
	result := JsonResponse{Version: grokstatconstants.VERSION, Flags: flags}

	if err != nil {
		result.Output = make(map[string]interface{})
		result.Status = 500
		result.Message = err.Error()
	} else {
		result.Output = output
		result.Status = 200
		result.Message = grokstaterrors.OK.Error()
	}

	jsonOut, jsonErr := json.Marshal(result)

	if jsonErr != nil {
		jsonOut = []byte(`{"status": 500, "message": "JSON marshaller error."}`)
	}

	return string(jsonOut), jsonErr
}

func CleanupMessageChan(messageChan chan models.ConsoleMsg, endChan <-chan struct{}) {
	close(messageChan)
	<-endChan
}

var PrintProtocols = func(messageChan chan models.ConsoleMsg, protColl models.ProtocolCollection, flags InputData) {
	output := make(map[string]interface{})
	output["protocols"] = protColl.All()

	PrintJsonResponse(messageChan, output, nil, flags)
}

var PrintError = func(messageChan chan models.ConsoleMsg, err error, flags InputData) {
	PrintJsonResponse(messageChan, nil, err, flags)
}

var PrintJsonResponse = func(messageChan chan models.ConsoleMsg, output interface{}, err error, flags InputData) {
	jsonOut, _ := FormJsonResponse(output, err, flags)
	messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MAJOR, Message: jsonOut}
}

var DefaultConfigBinPath = "data/grokstat.toml"

var ParseIPAddr = func(ipString string, defaultPort string) map[string]string {
	var ipStringMod string

	if len(strings.Split(ipString, "://")) == 1 {
		ipStringMod = "placeholder://" + ipString
	} else {
		ipStringMod = ipString
	}

	urlInfo, _ := url.Parse(ipStringMod)

	result := make(map[string]string)
	result["http_protocol"] = urlInfo.Scheme
	result["host"] = urlInfo.Host

	if len(strings.Split(result["host"], ":")) == 1 {
		result["host"] = result["host"] + ":" + defaultPort
	}

	return result
}

func identifyPacketProtocol(packet models.Packet) (string, bool) {
	return "STEAM", true
}

func Query(hosts []models.HostProtocolIdPair, protColl models.ProtocolCollection, messageChan chan<- models.ConsoleMsg, debugLvl int) (serverHosts []string, output []models.ServerEntry, err error) {
	serverHosts = []string{}
	output = []models.ServerEntry{}

	// This is for easier server identification.
	serverProtocolMapping := models.MakeServerProtocolMapping()
	protocolMappingInChan := make(chan models.HostProtocolIdPair)

	go func() {
		for {
			mappingEntry := <-protocolMappingInChan
			serverProtocolMapping[mappingEntry.RemoteAddr] = mappingEntry.ProtocolId
		}
	}()
	//

	getProtocolOfServer := func(remoteAddr string) (string, bool) {
		protocolName, pOk := serverProtocolMapping[remoteAddr]
		return protocolName, pOk
	}

	serverEntryChan := make(chan models.ServerEntry, 9999)
	sendPacketChan := make(chan models.Packet, 9999)
	receivePacketChan := make(chan models.Packet, 9999)

	serverInitChan := make(chan struct{})
	serverStopChan := make(chan struct{})

	serverDataMap := make(map[string]models.ServerEntry)

	go func() {
		for {
			serverEntry := <-serverEntryChan
			hostname := serverEntry.Host

			oldEntry, exists := serverDataMap[hostname]
			if !exists {
				serverDataMap[hostname] = serverEntry
			} else {
				mergedEntry := oldEntry
				mergedRules := map[string]string{}

				for k, v := range mergedEntry.Rules {
					mergedRules[k] = v
				}

				mergo.Merge(&mergedEntry, serverEntry)
				mergo.Merge(&mergedRules, serverEntry.Rules)

				mergedEntry.Rules = mergedRules

				serverDataMap[hostname] = mergedEntry
			}
		}
	}()

	parseHandlerWrapper := func(packet models.Packet) (sendPackets []models.Packet) {
		sendPackets = make([]models.Packet, 0)
		var protocolName string
		protocolMappingName, pOk := getProtocolOfServer(packet.RemoteAddr)
		if pOk {
			protocolName = protocolMappingName
		} else {
			protocolIdentifiedName, iOk := identifyPacketProtocol(packet)
			if iOk {
				protocolName = protocolIdentifiedName
			}
		}
		if protocolName != "" {
			protocolEntry, protocolExists := protColl.FindById(protocolName)
			if protocolExists {
				packet.ProtocolId = protocolName
				handlerFunc := protocolEntry.Base.HandlerFunc

				if handlerFunc != nil {
					sendPackets = handlerFunc(packet, protColl, messageChan, protocolMappingInChan, serverEntryChan)
				}
			}
		}

		return sendPackets

	}

	for _, packPair := range MakePacketErrorPair(hosts, protColl) {
		packet := packPair.Packet
		err := packPair.Error

		if err == nil {
			protocolMappingInChan <- models.HostProtocolIdPair{RemoteAddr: packet.RemoteAddr, ProtocolId: packet.ProtocolId}
			sendPacketChan <- packet
		} else {
			messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: err.Error()}
		}
	}

	go network.AsyncUDPServer(serverInitChan, serverStopChan, messageChan, sendPacketChan, receivePacketChan, parseHandlerWrapper, 5*time.Second)
	<-serverInitChan
	<-serverStopChan

	for _, entry := range serverDataMap {
		serverHosts = append(serverHosts, entry.Host)
		output = append(output, entry)
	}

	return serverHosts, output, err
}

func conditionalPrint(message models.ConsoleMsg, outputLvl int) {
	if message.Type <= outputLvl {
		fmt.Println(message.Message)
	}
}

func outputLoop(messageChan <-chan models.ConsoleMsg, messageEndChan chan<- struct{}, outputLvl int) {
	for {
		message, mOk := <-messageChan
		if mOk {
			conditionalPrint(message, outputLvl)
		} else {
			messageEndChan <- struct{}{}
			return
		}
	}
}

func main() {
	var configInstance ConfigFile

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	jsonFlags := InputData{Hosts: []string{}, Protocol: "", CustomConfigPath: "", ShowProtocols: false, OutputLvl: grokstatconstants.MSG_MAJOR}
	jsonErr := json.Unmarshal([]byte(text), &jsonFlags)

	messageChan := make(chan models.ConsoleMsg)
	messageEndChan := make(chan struct{})

	outputLvl := jsonFlags.OutputLvl

	go outputLoop(messageChan, messageEndChan, outputLvl)

	if jsonErr != nil {
		PrintError(messageChan, jsonErr, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	hostList := util.RemoveDuplicates(jsonFlags.Hosts)
	showProtocols := jsonFlags.ShowProtocols
	customConfigPath := jsonFlags.CustomConfigPath
	selectedProtocol := jsonFlags.Protocol
	debugLvl := outputLvl

	if customConfigPath == "" {
		configBinData, err := bindata.Asset(DefaultConfigBinPath)
		if err != nil {
			PrintError(messageChan, grokstaterrors.NoDefaultConfig, jsonFlags)
			CleanupMessageChan(messageChan, messageEndChan)
			return
		}
		toml.Decode(string(configBinData), &configInstance)
	} else {
		_, err := toml.DecodeFile(customConfigPath, &configInstance)
		if err != nil {
			PrintError(messageChan, grokstaterrors.ErrorLoadingCustomConfig, jsonFlags)
			CleanupMessageChan(messageChan, messageEndChan)
			return
		}
	}

	protColl := protocols.LoadProtocolCollection(configInstance.Protocols)

	if showProtocols {
		PrintProtocols(messageChan, protColl, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	if selectedProtocol == "" {
		PrintError(messageChan, grokstaterrors.NoProtocol, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	hosts := make([]models.HostProtocolIdPair, len(hostList))
	for i, host := range hostList {
		hosts[i] = models.HostProtocolIdPair{RemoteAddr: host, ProtocolId: selectedProtocol}
	}

	if len(hosts) == 0 {
		PrintError(messageChan, grokstaterrors.NoHosts, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	serverList, serverData, err := Query(hosts, protColl, messageChan, debugLvl)

	if err == nil {
		PrintJsonResponse(messageChan, map[string]interface{}{"server-list": serverList, "servers": serverData}, err, jsonFlags)
	} else {
		PrintError(messageChan, err, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	CleanupMessageChan(messageChan, messageEndChan)
}
