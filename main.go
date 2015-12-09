package main

import (
        "bufio"
        "bytes"
        "encoding/json"
        "flag"
        "fmt"
        "net"
        "time"
        )

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
        conn.SetDeadline(time.Now().Add(time.Duration(10)*time.Second))
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
func parse_quake3_master_response(response []byte) []string {
    var servers []string

    prelude := []byte{0xff, 0xff, 0xff, 0xff}
    rs := []byte("getserversResponse")
    splitter := []byte{0x5c}

    response_body := response[len(prelude) + len(rs):]
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
    return servers
}

func main() {
    var master_ip string
    flag.StringVar(&master_ip, "ip", "", "IP address of server to query.")
    flag.Parse()

    if master_ip == "" {fmt.Println("Please specify valid IP");return}

    rs := `getservers`
    protocol := `68`
    filter := `empty full`
    request := []byte(string([]byte{0xff, 0xff, 0xff, 0xff}) + rs + ` ` + protocol + ` ` + filter + "\n")
    response, _ := connect_send_receive("udp", master_ip, request)

    servers := parse_quake3_master_response(response)
    result := make(map[string]interface{})
    result["servers"] = servers

    json_out, _ := json.Marshal(result)
    fmt.Println(string(json_out))
}
