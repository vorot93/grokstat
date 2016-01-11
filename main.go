/*
grokstat is a tool for querying game servers for various information: server list, player count, active map etc

The program takes protocol name and remote ip address as arguments, fetches information from the remote server, parses it and outputs back as JSON. As convenience the status and message are also provided.

grokstat uses JSON input instead of command line flags. The JSON input is structured as follows:
	hosts - array of strings - hosts to query
	protocol - string - protocol to use
	show-protocols - boolean - if true, show protocols and exit
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
	"sync"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/grokstat/grokstat/bindata"
	"github.com/grokstat/grokstat/generalhelpers"
	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols"
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
	QueryPoolSize    int      `json:"query-pool-size"`
	FullMasterQuery  bool     `json:"full-master-query"`
	EnableStdout     bool     `json:"enable-stdout"`
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

// Forms a JSON string out of Grokstat output.
var FormJsonResponse = func(output interface{}, err error, flags InputData) (string, error) {
	result := JsonResponse{Version: VERSION, Flags: flags}

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

var PrintProtocols = func(protocolCmdMap map[string]models.ProtocolEntry, flags InputData) {
	var outputMapProtocols []models.ProtocolEntryInfo
	for _, v := range protocolCmdMap {
		outputMapProtocols = append(outputMapProtocols, v.Information)
	}

	output := make(map[string]interface{})
	output["protocols"] = outputMapProtocols

	PrintJsonResponse(output, nil, flags)
}

var PrintError = func(err error, flags InputData) {
	PrintJsonResponse(nil, err, flags)
}

var PrintJsonResponse = func(output interface{}, err error, flags InputData) {
	jsonOut, _ := FormJsonResponse(output, err, flags)
	fmt.Println(jsonOut)
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

var RequestWorkerCore = func(queryProtocol models.ProtocolEntry, remoteIp string) ServerResponseStruct {
	responseMap := make(map[string]models.Packet)
	ipMap := ParseIPAddr(remoteIp, queryProtocol.Information["DefaultRequestPort"])
	hostname := ipMap["host"]
	var err error

	conn, connectionErr := net.Dial(queryProtocol.Base.HttpProtocol, hostname)
	if connectionErr != nil {
		responseMap = make(map[string]models.Packet)
		err = connectionErr
	} else {
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
		for _, requestPacketDesc := range queryProtocol.Base.RequestPackets {
			packetId := requestPacketDesc.Id
			responsePacketNum := requestPacketDesc.ResponsePacketNum

			requestPacket := queryProtocol.Base.MakeRequestPacketFunc(packetId, queryProtocol.Information)
			var responseErr error
			responseMap[packetId], responseErr = generalhelpers.GetServerResponse(conn, requestPacket, responsePacketNum)
			if responseErr != nil {
				responseMap = make(map[string]models.Packet)
				err = responseErr
				break
			}
		}
	}
	return ServerResponseStruct{Hostname: hostname, ResponseMap: responseMap, ResponseErr: err}
}

var QueryWorker = func(workerId int, queryProtocol models.ProtocolEntry, remoteIpChan chan string, dataChan chan ServerResponseStruct, wg *sync.WaitGroup) {
	defer func() { wg.Done() }()
	for {
		remoteIp, workAvailable := <-remoteIpChan

		if workAvailable != true {
			return
		}

		if remoteIp == "" {
			continue
		}

		dataChan <- RequestWorkerCore(queryProtocol, remoteIp)
	}
}

var ParserWorkerCore = func(queryProtocol models.ProtocolEntry, responseMap map[string]models.Packet) models.ServerEntry {
	serverEntry, responseParseErr := queryProtocol.Base.ServerResponseParseFunc(responseMap, queryProtocol.Information)

	if responseParseErr != nil {
		emptyEntry := models.MakeServerEntry()
		emptyEntry.Status = 500
		emptyEntry.Error = responseParseErr
		return emptyEntry
	} else {
		serverEntry.Status = 200
		serverEntry.Error = nil
		return serverEntry
	}
}

var Query = func(workerNum int, selectedProtocol string, protocolCmdMap map[string]models.ProtocolEntry, hosts []string, fullMasterQuery bool, StdoutEnable bool) (serverHosts []string, output []models.ServerEntry, err error) {
	output = make([]models.ServerEntry, 0)
	var selectedQueryProtocol string
	var queryProtocol models.ProtocolEntry

	protocol, p_ok := protocolCmdMap[selectedProtocol]
	if p_ok == false {
		return nil, nil, grokstaterrors.InvalidProtocol
	}

	if protocol.Base.IsMaster {
		masterCallActive := make(chan struct{})
		util.Print(StdoutEnable, "Fetching server list")
		go util.PrintWait(StdoutEnable, 250, masterCallActive)

		var serverList []string
		for _, masterIp := range hosts {
			if masterIp == "" {
				continue
			}
			response := RequestWorkerCore(protocol, masterIp)
			responseMap := response.ResponseMap
			responseErr := response.ResponseErr
			if responseErr != nil {
				continue
			}

			data, responseParseErr := protocol.Base.MasterResponseParseFunc(responseMap, protocol.Information)
			if responseParseErr != nil {
				continue
			}

			for _, dataEntry := range data {
				serverList = append(serverList, dataEntry)
			}
		}
		serverHosts = util.RemoveDuplicates(serverList)
		close(masterCallActive)
		util.PrintEmptyLine(StdoutEnable)
	} else {
		serverHosts = hosts
	}

	if protocol.Base.IsMaster {
		if fullMasterQuery == false {
			return serverHosts, output, err
		}
		var q_ok bool
		selectedQueryProtocol, q_ok = protocol.Information["MasterOf"]
		queryProtocol, q_ok = protocolCmdMap[selectedQueryProtocol]
		if q_ok == false {
			return nil, nil, grokstaterrors.InvalidMasterOf
		}
	} else {
		queryProtocol = protocol
	}

	var queryWg sync.WaitGroup

	dataChan := make(chan ServerResponseStruct, len(serverHosts))
	remoteIpChan := make(chan string)

	workerMsgLen := 0
	for i := 0; i < workerNum; i++ {
		workerMsgLen = util.PrintReplace(StdoutEnable, fmt.Sprintf("Launching workers: %d / %d", i+1, workerNum), workerMsgLen)
		queryWg.Add(1)
		go QueryWorker(i, queryProtocol, remoteIpChan, dataChan, &queryWg)
		time.Sleep(10 * time.Millisecond)
	}
	util.PrintEmptyLine(StdoutEnable)

	queryMsgLen := 0
	hostLen := int64(len(serverHosts))
	for i, host := range serverHosts {
		queryMsgLen = util.PrintReplace(StdoutEnable, fmt.Sprintf("Querying: %d / %d", i+1, hostLen), queryMsgLen)
		remoteIpChan <- host
	}
	util.PrintEmptyLine(StdoutEnable)

	close(remoteIpChan)

	queryWg.Wait()

	close(dataChan)

	responseMsgLen := 0
	responseLen := len(dataChan)
	responseN := 0
	for response := range dataChan {
		responseMsgLen = util.PrintReplace(StdoutEnable, fmt.Sprintf("Parsing: %d / %d", responseN+1, responseLen), responseMsgLen)
		responseN += 1
		var serverEntry models.ServerEntry
		responseErr := response.ResponseErr
		if responseErr == nil {
			serverEntry = ParserWorkerCore(queryProtocol, response.ResponseMap)
		} else {
			serverEntry = models.MakeServerEntry()
			serverEntry.Status = 503
			serverEntry.Error = responseErr
		}
		serverEntry.Host = response.Hostname
		serverEntry.Protocol = queryProtocol.Id
		output = append(output, serverEntry)
	}
	for i, _ := range output {
		err := output[i].Error
		if err != nil {
			output[i].Message = err.Error()
		} else {
			output[i].Message = grokstaterrors.OK.Error()
		}
	}
	util.PrintEmptyLine(StdoutEnable)

	return serverHosts, output, err
}

func main() {
	var configInstance ConfigFile

	// Resets flags to default state, reads JSON from stdin
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	jsonFlags := InputData{Hosts: []string{}, Protocol: "", CustomConfigPath: "", ShowProtocols: false, QueryPoolSize: -1, FullMasterQuery: true, EnableStdout: false}
	jsonErr := json.Unmarshal([]byte(text), &jsonFlags)

	if jsonErr != nil {
		PrintError(jsonErr, jsonFlags)
		return
	}

	hosts := util.RemoveDuplicates(jsonFlags.Hosts)
	showProtocols := jsonFlags.ShowProtocols
	customConfigPath := jsonFlags.CustomConfigPath
	selectedProtocol := jsonFlags.Protocol
	fullMasterQuery := jsonFlags.FullMasterQuery
	stdoutEnabled := jsonFlags.EnableStdout
	var queryPoolSize int
	if jsonFlags.QueryPoolSize == -1 {
		queryPoolSize = DEFAULT_QUERY_POOL_SIZE
	} else {
		queryPoolSize = util.Clamp(jsonFlags.QueryPoolSize, MIN_QUERY_POOL_SIZE, MAX_QUERY_POOL_SIZE)
	}

	if customConfigPath == "" {
		configBinData, err := bindata.Asset(DefaultConfigBinPath)
		if err != nil {
			PrintError(grokstaterrors.NoDefaultConfig, jsonFlags)
			return
		}
		toml.Decode(string(configBinData), &configInstance)
	} else {
		_, err := toml.DecodeFile(customConfigPath, &configInstance)
		if err != nil {
			PrintError(grokstaterrors.ErrorLoadingCustomConfig, jsonFlags)
			return
		}
	}

	protocolCmdMap := protocols.MakeProtocolMap(configInstance.Protocols)

	if showProtocols {
		PrintProtocols(protocolCmdMap, jsonFlags)
		return
	}

	if selectedProtocol == "" {
		PrintError(grokstaterrors.NoProtocol, jsonFlags)
		return
	}
	if len(hosts) == 0 {
		PrintError(grokstaterrors.NoHosts, jsonFlags)
		return
	}

	serverList, serverData, err := Query(queryPoolSize, selectedProtocol, protocolCmdMap, hosts, fullMasterQuery, stdoutEnabled)

	if err != nil {
		PrintError(err, jsonFlags)
		return
	} else {
		PrintJsonResponse(map[string]interface{}{"server-list": serverList, "servers": serverData}, err, jsonFlags)
	}
}
