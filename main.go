/*
grokstat is a tool for querying game servers for various information: server list, player count, active map etc

The program takes protocol name and remote ip address as arguments, fetches information from the remote server, parses it and outputs back as JSON. As convenience the status and message are also provided.

grokstat uses JSON input instead of command line flags. The JSON input is structured as follows:
	hosts - map of string keys and string array values - hosts to query
	show-protocols - boolean - if true, show protocols and exit
	output-lvl - int - tune the output from bare JSON to full-fledged debug
	custom-config-path - path of custom config file to be used
*/
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/imdario/mergo"
)

type ServerResponseStruct struct {
	Hostname    string
	ResponseMap map[string]Packet
	ResponseErr error
}

type InputData struct {
	Hosts         map[string][]string `json:"hosts"`
	ShowProtocols bool                `json:"show-protocols"`
	OutputLvl     int                 `json:"output-lvl"`
	ConfigPath    string              `json:"config-path"`
}

func MakeInputData() InputData {
	return InputData{Hosts: make(map[string][]string)}
}

type ConfigFile struct {
	Protocols []ProtocolConfig `toml:"Protocols"`
}

type JsonResponse struct {
	Version string      `json:"version"`
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Flags   InputData   `json:"input-flags"`
	Output  interface{} `json:"output"`
}

type PacketErrorPair struct {
	Packet Packet
	Error  error
}

func MakePacketErrorPair(hosts []HostProtocolIdPair, protColl *ProtocolCollection) (packErrPairs []PacketErrorPair) {
	packErrPairs = []PacketErrorPair{}

	for _, hostpair := range hosts {
		var hostpackets = []Packet{}
		var err error

		hostport := strings.Split(hostpair.RemoteAddr, ":")
		protocolId := hostpair.ProtocolId
		protocol, protocolExists := protColl.Get(protocolId)
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

				reqPackets := MakeSendPackets(HostProtocolIdPair{RemoteAddr: addrFinal, ProtocolId: protocolId}, protColl)

				for _, reqPacket := range reqPackets {
					hostpackets = append(hostpackets, reqPacket)
				}
			} else {
				err = rErr
			}
		} else {
			err = InvalidProtocol
		}

		for _, packetFinal := range hostpackets {
			packErrPairs = append(packErrPairs, PacketErrorPair{Packet: packetFinal, Error: err})
		}
	}

	return packErrPairs
}

// FormJSONResponse creates a JSON string out of Grokstat output.
var FormJSONResponse = func(output interface{}, err error, flags InputData) (string, error) {
	result := JsonResponse{Version: VERSION, Flags: flags}

	if err != nil {
		result.Output = make(map[string]interface{})
		result.Status = 500
		result.Message = err.Error()
	} else {
		result.Output = output
		result.Status = 200
		result.Message = OK.Error()
	}

	jsonOut, jsonErr := json.Marshal(result)

	if jsonErr != nil {
		jsonOut = []byte(`{"status": 500, "message": "JSON marshaller error."}`)
	}

	return string(jsonOut), jsonErr
}

func CleanupMessageChan(messageChan chan ConsoleMsg, endChan <-chan struct{}) {
	close(messageChan)
	<-endChan
}

var PrintProtocols = func(messageChan chan ConsoleMsg, protColl *ProtocolCollection, flags InputData) {
	output := make(map[string]interface{})
	output["protocols"] = protColl.Map()

	PrintJsonResponse(messageChan, output, nil, flags)
}

var PrintError = func(messageChan chan ConsoleMsg, err error, flags InputData) {
	PrintJsonResponse(messageChan, nil, err, flags)
}

var PrintJsonResponse = func(messageChan chan ConsoleMsg, output interface{}, err error, flags InputData) {
	jsonOut, _ := FormJSONResponse(output, err, flags)
	messageChan <- ConsoleMsg{Type: MSG_MAJOR, Message: jsonOut}
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

func identifyPacketProtocol(packet Packet) (string, bool) {
	return "STEAM", true
}

func Query(hosts []HostProtocolIdPair, protColl *ProtocolCollection, messageChan chan<- ConsoleMsg, debugLvl int) (serverHosts []string, output []ServerEntry, err error) {
	serverHosts = []string{}
	output = []ServerEntry{}

	// This is for easier server identification.
	var serverProtocolMapping = map[string]string{}
	var protocolMappingInChan = make(chan HostProtocolIdPair)

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

	serverEntryChan := make(chan ServerEntry, 9999)
	sendPacketChan := make(chan Packet, 9999)
	receivePacketChan := make(chan Packet, 9999)

	serverInitChan := make(chan struct{})
	serverStopChan := make(chan struct{})

	serverDataMap := make(map[string]ServerEntry)

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

	parseHandlerWrapper := func(packet Packet) (sendPackets []Packet) {
		sendPackets = make([]Packet, 0)
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
			protocolEntry, protocolExists := protColl.Get(protocolName)
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
			protocolMappingInChan <- HostProtocolIdPair{RemoteAddr: packet.RemoteAddr, ProtocolId: packet.ProtocolId}
			sendPacketChan <- packet
		} else {
			messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: err.Error()}
		}
	}

	go AsyncNetworkServer(serverInitChan, serverStopChan, messageChan, sendPacketChan, receivePacketChan, parseHandlerWrapper, 5*time.Second)
	<-serverInitChan
	<-serverStopChan

	for _, entry := range serverDataMap {
		serverHosts = append(serverHosts, entry.Host)
		output = append(output, entry)
	}

	return serverHosts, output, err
}

func conditionalPrint(message ConsoleMsg, outputLvl int, useLogging bool) {
	if message.Type <= outputLvl {
		if useLogging {
			log.Println(message.Message)
		} else {
			fmt.Println(message.Message)
		}
	}
}

func outputLoop(messageChan <-chan ConsoleMsg, messageEndChan chan<- struct{}, outputLvl int) {
	for {
		message, mOk := <-messageChan
		if mOk {
			conditionalPrint(message, outputLvl, outputLvl >= MSG_DEBUG)
		} else {
			messageEndChan <- struct{}{}
			return
		}
	}
}

func main() {
	var configInstance ConfigFile
	args := os.Args
	var argJsonText string
	var jsonText string

	if len(args) > 1 {
		argJsonText = args[1]
	}

	if argJsonText != "" {
		jsonText = argJsonText
	} else {
		reader := bufio.NewReader(os.Stdin)
		jsonText, _ = reader.ReadString('\n')
	}

	jsonFlags := MakeInputData()
	jsonFlags.OutputLvl = DEFAULT_OUTPUT_LVL
	jsonErr := json.Unmarshal([]byte(jsonText), &jsonFlags)

	messageChan := make(chan ConsoleMsg)
	messageEndChan := make(chan struct{})

	outputLvl := jsonFlags.OutputLvl

	go outputLoop(messageChan, messageEndChan, outputLvl)

	if jsonErr != nil {
		PrintError(messageChan, jsonErr, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	hostMap := jsonFlags.Hosts
	showProtocols := jsonFlags.ShowProtocols
	configPath := jsonFlags.ConfigPath
	debugLvl := outputLvl

	if configPath == "" {
		PrintError(messageChan, NoConfig, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	_, err := toml.DecodeFile(configPath, &configInstance)
	if err != nil {
		PrintError(messageChan, ErrorLoadingConfig, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	protColl := LoadProtocols(configInstance.Protocols)

	if showProtocols {
		PrintProtocols(messageChan, protColl, jsonFlags)
		CleanupMessageChan(messageChan, messageEndChan)
		return
	}

	hosts := []HostProtocolIdPair{}
	for protocolId, hostList := range hostMap {
		for _, host := range RemoveDuplicates(hostList) {
			hosts = append(hosts, HostProtocolIdPair{RemoteAddr: host, ProtocolId: protocolId})
		}
	}

	if len(hosts) == 0 {
		PrintError(messageChan, NoHosts, jsonFlags)
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
