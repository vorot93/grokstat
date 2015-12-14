/*
grokstat is a tool for querying game servers for various information: server list, player count, active map etc

The program takes protocol name and remote ip address as arguments, fetches information from the remote server, parses it and outputs back as JSON. As convenience the status and message are also provided.

Usage of grokstat utility:
	-ip string
		IP address of server to query.
	-json-input
		Read JSON from standard input instead of flags
	-protocol string
		Server protocol to use.
	-show-protocols
		Output available server protocols.
*/
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
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

type jsonInputFlags struct {
	Ip string `json:"ip"`
	Protocol string `json:"protocol"`
}

type ConfigFile struct {
	Protocols []protocols.ProtocolConfig `toml:"Protocols"`
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

// Forms a JSON string out of Grokstat output.
func FormJsonString(output interface{}, err error) (string, error) {
	result := make(map[string]interface{})
	if err != nil {
		result["output"] = make(map[string]string)
		result["status"] = 500
		result["message"] = err.Error()
	} else {
		result["output"] = output
		result["status"] = 200
		result["message"] = "OK"
	}

	jsonOut, jsonErr := json.Marshal(result)

	if jsonErr != nil {
		jsonOut = []byte(`{}`)
	}

	return string(jsonOut), jsonErr
}

func PrintProtocols(protocolCmdMap map[string]models.ProtocolEntry) {
	var outputMapProtocols []models.ProtocolEntryInfo
	for _, v := range protocolCmdMap {
		outputMapProtocols = append(outputMapProtocols, v.Information)
	}

	output := make(map[string]interface{})
	output["protocols"] = outputMapProtocols

	jsonOut, _ := FormJsonString(output, nil)

	fmt.Println(string(jsonOut))
}

func PrintError(err error) {
	output := ""
	jsonOut, _ := FormJsonString(output, err)
	fmt.Println(jsonOut)
	return
}

func main() {
	var jsonInput bool
	var jsonInputDefault = false

	var customConfigPath string
	var customConfigPathDefault = ""

	var remoteIp string
	var remoteIpDefault = ""

	var selectedProtocol string
	var selectedProtocolDefault = ""

	var showProtocols bool
	var showProtocolsDefault = false

	flag.BoolVar(&jsonInput, "json-input", jsonInputDefault, "Read flags as JSON from standard input.")
	flag.StringVar(&customConfigPath, "custom-config", customConfigPathDefault, "Load a custom config file.")
	flag.StringVar(&remoteIp, "ip", remoteIpDefault, "IP address of server to query.")
	flag.StringVar(&selectedProtocol, "protocol", selectedProtocolDefault, "Server protocol to use.")
	flag.BoolVar(&showProtocols, "show-protocols", showProtocolsDefault, "Output available server protocols.")
	flag.Parse()

	output := make(map[string]interface{})

	var configInstance ConfigFile

	var defaultConfigData ConfigFile
	defaultConfigBinData, err := bindata.Asset(DefaultConfigBinPath)
	if err != nil {PrintError(errors.New("Default config file not found.")); return}
	toml.Decode(string(defaultConfigBinData), &defaultConfigData)

	if flag.NFlag() == 0 {flag.PrintDefaults();return}

	// Resets flags to default state, reads JSON from stdin instead
	if jsonInput {
		remoteIp = remoteIpDefault
		selectedProtocol = selectedProtocolDefault
		showProtocols = showProtocolsDefault

		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		jsonFlags := jsonInputFlags{Ip: remoteIpDefault, Protocol: selectedProtocolDefault}
		_ = json.Unmarshal([]byte(text), &jsonFlags)

		remoteIp = jsonFlags.Ip
		selectedProtocol = jsonFlags.Protocol
	}

	if customConfigPath == "" {
		configInstance = defaultConfigData
	} else {
		var customConfigData ConfigFile
		_, err := toml.DecodeFile(customConfigPath, &customConfigData)
		if err != nil {PrintError(errors.New("Error loading custom config file.")); return}
		configInstance = customConfigData
	}

	protocolCmdMap := protocols.MakeProtocolMap(configInstance.Protocols)

	if showProtocols {PrintProtocols(protocolCmdMap);return}

	if remoteIp == "" {PrintError(errors.New("Please specify a valid IP.")); return}
	if selectedProtocol == "" {PrintError(errors.New("Please specify the protocol.")); return}

	var protocol models.ProtocolEntry
	var g_ok bool
	protocol, g_ok = protocolCmdMap[selectedProtocol]
	if g_ok == false {PrintError(errors.New("Invalid protocol specified.")); return}

	var response []byte
	var responseErr error
	ipMap := ParseIPAddr(remoteIp, protocol.Information["DefaultRequestPort"])
	hostname := ipMap["host"]
	response, responseErr = QueryServer(protocol.Base.HttpProtocol, hostname, protocol.Base.MakeRequestPacketFunc(protocol.Information))
	if responseErr != nil {PrintError(responseErr); return}

	serverData, responseParseErr := protocol.Base.ResponseParseFunc(response, protocol.Information)
	if protocol.Base.IsMaster == true {
		_, assertOk := serverData.([]string)
		if responseParseErr != nil || !assertOk {PrintError(responseParseErr); return}

		output["servers"] = serverData
	} else {
		_, assertOk := serverData.(models.ServerEntry)
		if responseParseErr != nil || !assertOk {PrintError(responseParseErr); return}

		output["server_info"] = serverData
	}

	jsonOut, _ := FormJsonString(output, nil)

	fmt.Println(jsonOut)
}
