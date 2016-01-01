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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/grokstat/grokstat/bindata"
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
func FormJsonResponse(output interface{}, err error, flags InputData) (string, error) {
	result := JsonResponse{Version: VERSION, Flags: flags}

	if err != nil {
		result.Output = make(map[string]interface{})
		result.Status = 500
		result.Message = err.Error()
	} else {
		result.Output = output
		result.Status = 200
		result.Message = "OK"
	}

	jsonOut, jsonErr := json.Marshal(result)

	if jsonErr != nil {
		jsonOut = []byte(`{}`)
	}

	return string(jsonOut), jsonErr
}

var DefaultConfigBinPath string = "data/grokstat.toml"

// A convenience function for creating UDP connections
func newUDPConnection(addr string, protocol string) (*net.UDPConn, error) {
	raddr, _ := net.ResolveUDPAddr("udp", addr)
	caddr, _ := net.ResolveUDPAddr("udp", ":0")
	conn, err := net.DialUDP(protocol, caddr, raddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// A convenience function for creating TCP connections
func newTCPConnection(addr string, protocol string) (*net.TCPConn, error) {
	raddr, _ := net.ResolveTCPAddr("tcp", addr)
	caddr, _ := net.ResolveTCPAddr("tcp", ":0")
	conn, err := net.DialTCP(protocol, caddr, raddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func GetServerResponse(httpProtocol string, addr string, requestPacket models.Packet) (responsePacket models.Packet, err error) {
	emptyResponse := errors.New("No response from server")
	packetId := requestPacket.Id

	if httpProtocol == "tcp" {
		conn, connection_err := newTCPConnection(addr, httpProtocol)
		if connection_err != nil {
			return models.Packet{Id: packetId}, connection_err
		}
		defer conn.Close()
		var buf string
		buf, err = bufio.NewReader(conn).ReadString('\n')
		responsePacket = models.Packet{Data: []byte(buf), Id: packetId}
	} else if httpProtocol == "udp" {
		conn, connection_err := newUDPConnection(addr, httpProtocol)
		defer conn.Close()
		if connection_err != nil {
			return models.Packet{Id: packetId}, connection_err
		}
		buf_len := 16777215
		buf := make([]byte, buf_len)
		sendtime := time.Now()
		conn.Write(requestPacket.Data)
		conn.SetReadDeadline(time.Now().Add(time.Duration(5) * time.Second))
		_, _, err = conn.ReadFromUDP(buf)
		recvtime := time.Now()
		ping := int64(recvtime.Sub(sendtime) / time.Millisecond)
		if err != nil {
			return models.Packet{Id: packetId}, err
		} else {
			responsePacket = models.Packet{Data: bytes.TrimRight(buf, "\x00"), Id: packetId, Ping: ping}
			if len(responsePacket.Data) == 0 {
				err = emptyResponse
			}
		}
	}
	return responsePacket, err
}

func ParseIPAddr(ipString string, defaultPort string) map[string]string {
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

func PrintProtocols(protocolCmdMap map[string]models.ProtocolEntry, flags InputData) {
	var outputMapProtocols []models.ProtocolEntryInfo
	for _, v := range protocolCmdMap {
		outputMapProtocols = append(outputMapProtocols, v.Information)
	}

	output := make(map[string]interface{})
	output["protocols"] = outputMapProtocols

	PrintJsonResponse(output, nil, flags)
}

func PrintError(err error, flags InputData) {
	PrintJsonResponse(nil, err, flags)
}

func PrintJsonResponse(output interface{}, err error, flags InputData) {
	jsonOut, _ := FormJsonResponse(output, err, flags)
	fmt.Println(jsonOut)
}

func RequestWorkerCore(queryProtocol models.ProtocolEntry, remoteIp string) ServerResponseStruct {
	responseMap := make(map[string]models.Packet)
	ipMap := ParseIPAddr(remoteIp, queryProtocol.Information["DefaultRequestPort"])
	hostname := ipMap["host"]
	var err error
	for _, packetId := range queryProtocol.Base.RequestPackets {
		requestPacket := queryProtocol.Base.MakeRequestPacketFunc(packetId, queryProtocol.Information)
		var responseErr error
		responseMap[packetId], responseErr = GetServerResponse(queryProtocol.Base.HttpProtocol, hostname, requestPacket)
		if responseErr != nil {
			responseMap = make(map[string]models.Packet)
			err = responseErr
			break
		}
	}
	return ServerResponseStruct{Hostname: hostname, ResponseMap: responseMap, ResponseErr: err}
}

func QueryWorker(workerId int, queryProtocol models.ProtocolEntry, remoteIpChan chan string, dataChan chan ServerResponseStruct, wg *sync.WaitGroup) {
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

func ParserWorkerCore(queryProtocol models.ProtocolEntry, responseMap map[string]models.Packet) models.ServerEntry {
	serverEntry, responseParseErr := queryProtocol.Base.ServerResponseParseFunc(responseMap, queryProtocol.Information)

	if responseParseErr != nil {
		return models.ServerEntry{Status: 500, Message: responseParseErr.Error()}
	}

	serverEntry.Status = 200
	serverEntry.Message = "OK"

	return serverEntry
}

func Query(workerNum int, selectedProtocol string, protocolCmdMap map[string]models.ProtocolEntry, hosts []string, fullMasterQuery bool, StdoutEnable bool) (serverHosts []string, output []models.ServerEntry, err error) {
	output = []models.ServerEntry{}
	var selectedQueryProtocol string
	var queryProtocol models.ProtocolEntry

	protocol, p_ok := protocolCmdMap[selectedProtocol]
	if p_ok == false {
		return []string{}, []models.ServerEntry{}, errors.New("Invalid protocol specified.")
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
			ipMap := ParseIPAddr(masterIp, protocol.Information["DefaultRequestPort"])
			hostname := ipMap["host"]
			responseMap := make(map[string]models.Packet)
			var responseErr error
			for _, packetId := range protocol.Base.RequestPackets {
				if responseErr != nil {
					break
				}
				requestPacket := protocol.Base.MakeRequestPacketFunc(packetId, protocol.Information)
				responseMap[packetId], responseErr = GetServerResponse(protocol.Base.HttpProtocol, hostname, requestPacket)
			}
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
			return []string{}, []models.ServerEntry{}, errors.New("Invalid query part attached to master protocol.")
		}
	} else {
		queryProtocol = protocol
	}

	var queryWg sync.WaitGroup

	dataChan := make(chan ServerResponseStruct, len(serverHosts))
	remoteIpChan := make(chan string)

	workerMsgLen := 0
	for i := 0; i < workerNum; i++ {
		workerMsgLen = util.PrintReplace(StdoutEnable, "Launching workers: "+strconv.FormatInt(int64(i+1), 10)+" / "+strconv.FormatInt(int64(workerNum), 10), workerMsgLen)
		queryWg.Add(1)
		go QueryWorker(i, queryProtocol, remoteIpChan, dataChan, &queryWg)
		time.Sleep(10 * time.Millisecond)
	}
	util.PrintEmptyLine(StdoutEnable)

	queryMsgLen := 0
	hostLen := int64(len(serverHosts))
	for i, host := range serverHosts {
		queryMsgLen = util.PrintReplace(StdoutEnable, "Querying: "+strconv.FormatInt(int64(i+1), 10)+" / "+strconv.FormatInt(hostLen, 10), queryMsgLen)
		remoteIpChan <- host
	}
	util.PrintEmptyLine(StdoutEnable)

	close(remoteIpChan)

	queryWg.Wait()

	close(dataChan)

	responseMsgLen := 0
	responseLen := int64(len(dataChan))
	responseN := 0
	for response := range dataChan {
		responseMsgLen = util.PrintReplace(StdoutEnable, "Parsing: "+strconv.FormatInt(int64(responseN+1), 10)+" / "+strconv.FormatInt(responseLen, 10), responseMsgLen)
		responseN += 1
		var serverEntry models.ServerEntry
		responseErr := response.ResponseErr
		if responseErr == nil {
			serverEntry = ParserWorkerCore(queryProtocol, response.ResponseMap)
		} else {
			serverEntry = models.ServerEntry{Status: 503, Message: responseErr.Error()}
		}
		serverEntry.Host = response.Hostname
		serverEntry.Protocol = queryProtocol.Id
		output = append(output, serverEntry)
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
		queryClamp := sort.IntSlice([]int{MIN_QUERY_POOL_SIZE, jsonFlags.QueryPoolSize, MAX_QUERY_POOL_SIZE})
		queryClamp.Sort()
		queryPoolSize = int(queryClamp[1])
	}

	if customConfigPath == "" {
		configBinData, err := bindata.Asset(DefaultConfigBinPath)
		if err != nil {
			PrintError(errors.New("Default config file not found."), jsonFlags)
			return
		}
		toml.Decode(string(configBinData), &configInstance)
	} else {
		_, err := toml.DecodeFile(customConfigPath, &configInstance)
		if err != nil {
			PrintError(errors.New("Error loading custom config file."), jsonFlags)
			return
		}
	}

	protocolCmdMap := protocols.MakeProtocolMap(configInstance.Protocols)

	if showProtocols {
		PrintProtocols(protocolCmdMap, jsonFlags)
		return
	}

	if selectedProtocol == "" {
		PrintError(errors.New("Please specify the protocol."), jsonFlags)
		return
	}
	if len(hosts) == 0 {
		PrintError(errors.New("No hosts specified."), jsonFlags)
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
