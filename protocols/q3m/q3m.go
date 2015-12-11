package q3m

import (
	"bytes"
	"errors"
	"fmt"
)

func parseMasterServerEntry(entry_raw []byte) string {
	if len(entry_raw) != 6 {return ""}

	entry := make([]int, 6)
	for i, v := range entry_raw {
		entry[i] = int(v)
	}
	a := entry[0]
	b := entry[1]
	c := entry[2]
	d := entry[3]
	port := entry[4]*(16*16) + entry[5]

	if a == 0 {return ""}

	server_entry := fmt.Sprintf("%d.%d.%d.%d:%d", a, b, c, d, port)

	return server_entry
}

// Parses the response from Quake III Arena master server.
func ParseMasterResponse(response []byte, requestPrelude []byte) ([]string, error) {
	servers := []string{}

	splitter := []byte{0x5c}

	if bytes.Equal(response[:len(requestPrelude)], requestPrelude) != true {
		return []string{}, errors.New("Invalid response prelude.")
	}

	response_body := response[len(requestPrelude):]
	response_split := bytes.Split(response_body, splitter)
	for _, entry_raw := range response_split {
		server_entry := parseMasterServerEntry(entry_raw)

		if len(server_entry) > 0 {
			servers = append(servers, server_entry)
		}
	}
	return servers, nil
}
