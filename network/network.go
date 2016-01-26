package network

import (
	"fmt"
	"net"
	"time"

	"github.com/grokstat/grokstat/grokstatconstants"
	"github.com/grokstat/grokstat/models"
	"github.com/grokstat/grokstat/util"
)

func readUDP(conn *net.UDPConn) (models.Packet, error) {
	bufsize := 2048
	buf := make([]byte, bufsize)

	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return models.Packet{}, err
	}

	return models.Packet{Data: buf[:n], Timestamp: time.Now().Unix(), RemoteAddr: addr.String()}, nil
}

func writeUDP(conn *net.UDPConn, packet models.Packet) {
	remoteIpUdp, rErr := net.ResolveUDPAddr("udp4", packet.RemoteAddr)
	util.CheckError(rErr)
	conn.WriteToUDP(packet.Data, remoteIpUdp)
}

func receiveHandler(packet models.Packet, sendRequestChan chan<- models.Packet, parseHandler func(models.Packet) []models.Packet) {
	sendPackets := parseHandler(packet)

	for _, sendPacket := range sendPackets {
		sendRequestChan <- sendPacket
	}
}

func AsyncUDPServer(messageChan chan<- models.ConsoleMsg, sendChan, sendRequestChan, receiveChan chan models.Packet, endChan chan struct{}, parseHandler func(models.Packet) []models.Packet, timeOut time.Duration) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 0,
		IP:   net.IPv4zero,
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("Started UDP server at %s", conn.LocalAddr().String())}

	active := make(chan struct{}, 9999)

	endServer := make(chan struct{})
	endReceive := make(chan struct{})
	endRead := make(chan struct{})
	endWrite := make(chan struct{})

	go func() {
		for {
			select {
			case <-endReceive:
				return
			default:
				packet, err := readUDP(conn)
				if err == nil {
					messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: fmt.Sprintf("Read %d bytes from %s", len(packet.Data), packet.RemoteAddr)}
					receiveChan <- packet
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case dataAvailable := <-receiveChan:
				active <- struct{}{}
				go receiveHandler(dataAvailable, sendRequestChan, parseHandler)
			case <-endRead:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case dataSendPayload := <-sendChan:
				messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: fmt.Sprintf("Writing %d bytes to %s", len(dataSendPayload.Data), dataSendPayload.RemoteAddr)}
				active <- struct{}{}
				go writeUDP(conn, dataSendPayload)
			case <-endWrite:
				return
			}
		}
	}()

	go func() {
		<-active
		for {
			if timeOut > 0 {
				select {
				case <-active:
				case <-time.After(timeOut):
					endRead <- struct{}{}
					endWrite <- struct{}{}
					endServer <- struct{}{}
					return
				}
			} else {
				<-active
			}
		}
	}()
	<-endServer
	messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("Stopping server.")}
	endChan <- struct{}{}
}
