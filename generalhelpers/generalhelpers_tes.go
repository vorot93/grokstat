package generalhelpers

import (
	"fmt"
	"net"
	"time"

	"github.com/grokstat/grokstat/models"
)

var Try123 = func() (models.Packet, error) {
	packetData := []byte("\x05\x00\x06\x02\x02")
	packet := models.Packet{Data: packetData}
	addr := "master.openttd.org:3978"
	conn, connErr := net.Dial("udp", addr)
	if connErr == nil {
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
	} else {
		fmt.Println(connErr)
	}
	return GetServerResponse(conn, packet, -1)
}
