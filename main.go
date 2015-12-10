/*
grokstat is a tool for querying game servers for various information: server list, player count, active map etc

Usage of grokstat utility:
	-ip string
		IP address of server to query.
	-protocol string
		Server protocol to use.
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
	"text/template"
	"time"
)

// Server query protocol entry defining grokstat's behavior
type ProtocolEntry struct {
	Id                string
	Name              string
	MasterRequest     string
	MasterParseFunc   func([]byte, []byte) ([]string, error)
	MasterResponse    string
	ProtocolVer       string
	DefaultMasterPort string
}

// Construct a new protocol entry and return it to user
func NewProtocolEntry(protocolId string, name string, masterRequestTemplate string, masterParseFunc func([]byte, []byte) ([]string, error), masterResponse string, protocolVer string, defaultMasterPort string) ProtocolEntry {
	entry := ProtocolEntry{Id: protocolId, Name: name, MasterParseFunc: masterParseFunc, MasterResponse: masterResponse, ProtocolVer: protocolVer, DefaultMasterPort: defaultMasterPort}

	var buf = new(bytes.Buffer)

	t, _ := template.New("Request template").Parse(masterRequestTemplate)
	t.Execute(buf, entry)

	masterRequestString := buf.String()

	entry.MasterRequest = masterRequestString

	return entry
}

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

func connect_send_receive(protocol string, addr string, request []byte) ([]byte, error) {
	var status []byte
	var err error
	emptyResponse := errors.New("No response from server")

	if protocol == "tcp" {
		conn, connection_err := newTCPConnection(addr, protocol)
		if connection_err != nil {
			return []byte{}, connection_err
		}
		defer conn.Close()
		var buf string
		buf, err = bufio.NewReader(conn).ReadString('\n')
		status = []byte(buf)
	} else if protocol == "udp" {
		conn, connection_err := newUDPConnection(addr, protocol)
		if connection_err != nil {
			return []byte{}, connection_err
		}
		defer conn.Close()
		conn.Write(request)
		buf_len := 65535
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

// Parses the response from Quake III Arena master server.
func parseQuake3MasterResponse(response []byte, request []byte) ([]string, error) {
	var servers []string

	parseErr := errors.New("Error parsing the response")

	splitter := []byte{0x5c}

	if bytes.Equal(response[:len(request)], request) != true {
		return []string{}, parseErr
	}

	response_body := response[len(request):]
	response_split := bytes.Split(response_body, splitter)
	for _, entry_raw := range response_split {
		if len(entry_raw) == 6 {
			entry := make([]int, 6)
			for i, v := range entry_raw {
				entry[i] = int(v)
			}
			a := entry[0]
			b := entry[1]
			c := entry[2]
			d := entry[3]
			port := entry[4]*(16*16) + entry[5]
			server_entry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)
			servers = append(servers, server_entry)
		}
	}
	return servers, nil
}

func ParseScheme(protocol_string string) string {
	var protocol string

	if protocol_string == "udp" {
		protocol = "udp"
	} else {
		protocol = "tcp"
	}

	return protocol
}

func ParseIPAddr(ipString string, defaultPort string) map[string]string {
	urlInfo, _ := url.Parse(ipString)

	result := make(map[string]string)
	result["http_protocol"] = ParseScheme(urlInfo.Scheme)
	result["host"] = urlInfo.Host

	return result
}

// Forms a JSON string out of server list.
func FormJsonString(servers []string, err error) (string, error) {
	result := make(map[string]interface{})
	if err != nil {
		result["servers"] = []string{}
		result["status"] = "500"
		result["message"] = err.Error()
	} else {
		result["servers"] = servers
		result["status"] = "200"
		result["message"] = "OK"
	}

	jsonOut, jsonErr := json.Marshal(result)

	if jsonErr != nil {
		jsonOut = []byte(`{}`)
	}

	return string(jsonOut), jsonErr
}

func main() {
	var protocolFlag string
	var master_ip string
	flag.StringVar(&master_ip, "ip", "", "IP address of server to query.")
	flag.StringVar(&protocolFlag, "protocol", "", "Server protocol to use.")
	flag.Parse()

	var resultErr error

	if master_ip == "" {
		resultErr = errors.New("Please specify a valid IP.")
	}
	if protocolFlag == "" {
		resultErr = errors.New("Please specify the protocol.")
	}

	protocolCmdMap := make(map[string]ProtocolEntry)
	protocolCmdMap["q3m"] = NewProtocolEntry("quake3master", "Quake III Arena Master", "\xFF\xFF\xFF\xFFgetservers {{.ProtocolVer}} empty full\n", parseQuake3MasterResponse, "\xFF\xFF\xFF\xFFgetserversResponse", "68", "27950")

	var protocol ProtocolEntry
	if resultErr == nil {
		var g_ok bool
		protocol, g_ok = protocolCmdMap[protocolFlag]
		if g_ok == false {
			resultErr = errors.New("Invalid protocol specified.")
		}
	}

	var response []byte
	if resultErr == nil {
		var responseErr error
		ipMap := ParseIPAddr(master_ip, protocol.DefaultMasterPort)
		response, responseErr = connect_send_receive(ipMap["http_protocol"], ipMap["host"], []byte(protocol.MasterRequest))
		resultErr = responseErr
	}

	var servers []string
	if resultErr == nil {
		var masterParseErr error
		servers, masterParseErr = protocol.MasterParseFunc([]byte(response), []byte(protocol.MasterResponse))
		resultErr = masterParseErr
	}

	jsonOut, _ := FormJsonString(servers, resultErr)

	fmt.Println(jsonOut)
}
