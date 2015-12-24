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

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/grokstat/grokstat/bindata"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/protocols"
)

type InputData struct {
	Hosts            []string `json:"hosts"`
	Protocol         string   `json:"protocol"`
	ShowProtocols    bool     `json:"show-protocols"`
	CustomConfigPath string   `json:"custom-config-path"`
}

type ConfigFile struct {
	Protocols []protocols.ProtocolConfig `toml:"Protocols"`
}

type JsonResponse struct {
	Version string      `json:"version"`
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Output  interface{} `json:"output"`
}

// Forms a JSON string out of Grokstat output.
func FormJsonResponse(output interface{}, err error) (string, error) {
	result := JsonResponse{Version: VERSION}

	if err != nil {
		result.Output = `{}`
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

func QueryServer(httpProtocol string, addr string, request []byte) ([]byte, error) {
	var status []byte
	var err error
	emptyResponse := errors.New("No response from server")

	if httpProtocol == "tcp" {
		conn, connection_err := newTCPConnection(addr, httpProtocol)
		if connection_err != nil {
			return []byte{}, connection_err
		}
		defer conn.Close()
		var buf string
		buf, err = bufio.NewReader(conn).ReadString('\n')
		status = []byte(buf)
	} else if httpProtocol == "udp" {
		conn, connection_err := newUDPConnection(addr, httpProtocol)
		if connection_err != nil {
			return []byte{}, connection_err
		}
		defer conn.Close()
		conn.Write(request)
		buf_len := 16777215
		buf := make([]byte, buf_len)
		conn.SetDeadline(time.Now().Add(time.Duration(5) * time.Second))
		conn.ReadFromUDP(buf)
		if err != nil {
			return []byte{}, err
		} else {
			status = bytes.TrimRight(buf, "\x00")
			if len(status) == 0 {
				err = emptyResponse
			}
		}
	}
	return status, err
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

func PrintProtocols(protocolCmdMap map[string]models.ProtocolEntry) {
	var outputMapProtocols []models.ProtocolEntryInfo
	for _, v := range protocolCmdMap {
		outputMapProtocols = append(outputMapProtocols, v.Information)
	}

	output := make(map[string]interface{})
	output["protocols"] = outputMapProtocols

	jsonOut, _ := FormJsonResponse(output, nil)

	fmt.Println(string(jsonOut))
}

func PrintError(err error) {
	output := ""
	jsonOut, _ := FormJsonResponse(output, err)
	fmt.Println(jsonOut)
	return
}

func PrintJsonResponse(output interface{}, err error) {
	jsonOut, _ := FormJsonResponse(output, err)
	fmt.Println(jsonOut)
}

func Query(selectedProtocol string, protocolCmdMap map[string]models.ProtocolEntry, hosts []string) (output []models.ServerEntry, err error) {
	protocol, g_ok := protocolCmdMap[selectedProtocol]
	if g_ok == false {
		PrintError(errors.New("Invalid protocol specified."))
		return
	}
	for _, remoteIp := range hosts {
		var data []models.ServerEntry
		if remoteIp == "" {
			continue
		}

		var response []byte
		var responseErr error
		ipMap := ParseIPAddr(remoteIp, protocol.Information["DefaultRequestPort"])
		hostname := ipMap["host"]
		response, responseErr = QueryServer(protocol.Base.HttpProtocol, hostname, protocol.Base.MakeRequestPacketFunc(protocol.Information))
		if responseErr != nil {
			continue
		}

		serverData, responseParseErr := protocol.Base.ResponseParseFunc(response, protocol.Information)
		if protocol.Base.IsMaster == true {
			hosts, assertOk := serverData.([]string)
			if responseParseErr != nil || !assertOk {
				continue
			}
			
			data, _ = Query(protocol.Information["MasterOf"], protocolCmdMap, hosts)
		} else {
			serverEntry, assertOk := serverData.(models.ServerEntry)
			if responseParseErr != nil || !assertOk {
				continue
			}
			data = []models.ServerEntry{serverEntry}
			PrintJsonResponse(data, nil)
		}
		for _, v := range data {
			output = append(output, v)
		}
	}

	return output, err
}

func main() {
	var configInstance ConfigFile

	// Resets flags to default state, reads JSON from stdin
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	jsonFlags := InputData{Hosts: []string{}, Protocol: "", CustomConfigPath: "", ShowProtocols: false}
	jsonErr := json.Unmarshal([]byte(text), &jsonFlags)

	if jsonErr != nil {
		PrintError(jsonErr)
		return
	}

	hosts := jsonFlags.Hosts
	showProtocols := jsonFlags.ShowProtocols
	customConfigPath := jsonFlags.CustomConfigPath
	selectedProtocol := jsonFlags.Protocol

	if customConfigPath == "" {
		configBinData, err := bindata.Asset(DefaultConfigBinPath)
		if err != nil {
			PrintError(errors.New("Default config file not found."))
			return
		}
		toml.Decode(string(configBinData), &configInstance)
	} else {
		_, err := toml.DecodeFile(customConfigPath, &configInstance)
		if err != nil {
			PrintError(errors.New("Error loading custom config file."))
			return
		}
	}

	protocolCmdMap := protocols.MakeProtocolMap(configInstance.Protocols)

	if showProtocols {
		PrintProtocols(protocolCmdMap)
		return
	}

	if selectedProtocol == "" {
		PrintError(errors.New("Please specify the protocol."))
		return
	}
	if len(hosts) == 0 {
		PrintError(errors.New("No hosts specified."))
		return
	}

	output, err := Query(selectedProtocol, protocolCmdMap, hosts)

	if err != nil {
		PrintError(err)
		return
	} else {
		PrintJsonResponse(output, err)
	}
}
