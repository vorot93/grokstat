package main

import (
	"bytes"
	"net"
	"time"
)

func GetServerResponse(conn net.Conn, requestPacket Packet, responseN int) (Packet, error) {
	var responsePacket Packet
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

	responsePacket = Packet{Data: bytes.TrimRight(buf, "\x00"), Id: packetId, Ping: ping}
	if len(responsePacket.Data) == 0 {
		err = ServerDown
	}

	return responsePacket, err
}
