package generalhelpers

import (
	"bytes"
	"net"
	"time"

	"github.com/grokstat/grokstat/grokstaterrors"
	"github.com/grokstat/grokstat/models"
)

var GetServerResponse = func(conn net.Conn, requestPacket models.Packet, responseN int) (models.Packet, error) {
	var responsePacket models.Packet
	var err error

	packetId := requestPacket.Id

	buf_len := 16777215
	var buf []byte
	data := requestPacket.Data
	conn.Write(data)
	sendtime := time.Now()
	var recvtime time.Time
	var incr int
	var reads int
	if responseN > 0 {
		incr = 1
		reads = responseN
	} else {
		incr = 0
		reads = 1
	}
	for i := 0; i < reads; i += incr {
		var n int
		var connErr error
		var readBuf = make([]byte, buf_len)
		n, connErr = conn.Read(readBuf)
		if connErr != nil {
			break
		} else if n != 0 {
			if recvtime == (time.Time{}) {
				recvtime = time.Now()
			}
			buf = bytes.Join([][]byte{buf, bytes.TrimRight(readBuf, "\x00")}, []byte{})
		}
	}
	ping := int64(recvtime.Sub(sendtime) / time.Millisecond)

	responsePacket = models.Packet{Data: bytes.TrimRight(buf, "\x00"), Id: packetId, Ping: ping}
	if len(responsePacket.Data) == 0 {
		err = grokstaterrors.ServerDown
	}

	return responsePacket, err
}
