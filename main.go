package main

import (
        "bufio"
        "bytes"
        "encoding/json"
        "errors"
        "flag"
        "fmt"
        "net"
        "time"
        )

type Game struct {
    Id string
    Name string
    MasterRequest string
    MasterParseFunc func([]byte, []byte)([]string, error)
    MasterResponse string
    ProtocolVer string
}

func newUDPConnection(addr string, protocol string) (*net.UDPConn, error) {
    raddr, _ := net.ResolveUDPAddr("udp", addr)
    caddr, _ := net.ResolveUDPAddr("udp", ":0")
    conn, err := net.DialUDP(protocol, caddr, raddr)
    if err != nil {
        return nil, err
    }
    return conn, nil
}

func connect_send_receive(protocol string, addr string, request []byte) ([]byte, error) {
    var status []byte
    var err error

    conn, connection_err := newUDPConnection(addr, protocol)
    defer conn.Close()
    if connection_err != nil {
        return []byte{}, connection_err
    }
    conn.Write(request)
    if protocol == "tcp" {
        var buf string
        buf, err = bufio.NewReader(conn).ReadString('\n')
        status = []byte(buf)
    } else if protocol == "udp" {
        buf := make([]byte, 1024)
        conn.SetDeadline(time.Now().Add(time.Duration(5)*time.Second))
        conn.ReadFromUDP(buf)
        if err != nil {
            return []byte{}, err
        } else {
            status = buf
        }
    }
    return status, err
}

// \xFF\xFF\xFF\xFFgetserversResponse\\[...]\\EOT\0\0\0
func parseQuake3MasterResponse(response []byte, request []byte) ([]string, error) {
    var servers []string

    parseErr := errors.New("Error parsing the response")

    splitter := []byte{0x5c}

    if bytes.Equal(response[:len(request)], request) != true {return []string{}, parseErr}

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
            port := entry[4] * (16 * 16) + entry[5]
            server_entry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)
            servers = append(servers, server_entry)
        }
    }
    return servers, nil
}

func main() {
    var master_ip string
    flag.StringVar(&master_ip, "ip", "", "IP address of server to query.")
    flag.Parse()

    if master_ip == "" {fmt.Println("Please specify valid IP");return}

    game := Game{Id: "quake3", Name: "Quake III Arena", MasterRequest: "\xFF\xFF\xFF\xFFgetservers 68 empty full\n", MasterParseFunc: parseQuake3MasterResponse, MasterResponse: "\xFF\xFF\xFF\xFFgetserversResponse", ProtocolVer: "68"}

    response, _ := connect_send_receive("udp", master_ip, []byte(game.MasterRequest))

    servers, masterParseErr := game.MasterParseFunc([]byte(response), []byte(game.MasterResponse))

    result := make(map[string]interface{})
    if masterParseErr != nil {
        result["servers"] = []string{}
        result["status"] = "500"
        result["message"] = masterParseErr.Error()
    } else {
        result["servers"] = servers
        result["status"] = "200"
        result["message"] = "OK"
    }

    json_out, _ := json.Marshal(result)
    fmt.Println(string(json_out))
}
