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

func udpReceiveLoop(endChan chan struct{}, conn *net.UDPConn, messageChan chan<- models.ConsoleMsg, receiveChan chan models.Packet, awakeChan chan struct{}) {
	for {
		select {
		case <-endChan:
			return
		default:
			packet, err := readUDP(conn)
			awakeChan <- struct{}{}
			if err == nil {
				messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: fmt.Sprintf("Read %d bytes from %s", len(packet.Data), packet.RemoteAddr)}
				receiveChan <- packet
			}
		}
	}

}

func receiveHandlerLoop(endChan chan struct{}, receiveChan chan models.Packet, sendRequestChan chan<- models.Packet, receiveHandler func(models.Packet, chan<- models.Packet, func(models.Packet) []models.Packet), parseHandler func(models.Packet) []models.Packet, awakeChan chan<- struct{}) {
	for {
		select {
		case dataAvailable := <-receiveChan:
			awakeChan <- struct{}{}
			go receiveHandler(dataAvailable, sendRequestChan, parseHandler)
		case <-endChan:
			return
		}
	}
}

func udpSendLoop(endChan chan struct{}, conn *net.UDPConn, messageChan chan<- models.ConsoleMsg, sendChan chan models.Packet, awakeChan chan struct{}) {
	for {
		select {
		case dataSendPayload := <-sendChan:
			messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_DEBUG, Message: fmt.Sprintf("Writing %d bytes to %s", len(dataSendPayload.Data), dataSendPayload.RemoteAddr)}
			awakeChan <- struct{}{}
			go writeUDP(conn, dataSendPayload)
		case <-endChan:
			return
		}
	}
}

func keepAliveLoop(awakeChan chan struct{}, timeOut time.Duration, endChans ...chan struct{}) {
	<-awakeChan
	for {
		if timeOut > 0 {
			select {
			case <-awakeChan:
			case <-time.After(timeOut):
				for _, endChannel := range endChans {
					endChannel <- struct{}{}
				}
				return
			}
		} else {
			<-awakeChan
		}
	}
}

func AsyncUDPServer(initChan, endChan chan<- struct{}, messageChan chan<- models.ConsoleMsg, sendChan, receiveChan chan models.Packet, parseHandler func(models.Packet) []models.Packet, timeOut time.Duration) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 0,
		IP:   net.IPv4zero,
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("Starting UDP server at %s", conn.LocalAddr().String())}

	active := make(chan struct{}, 9999)

	endReceive := make(chan struct{}, 1)
	endServer := make(chan struct{}, 1)
	endRead := make(chan struct{}, 1)
	endWrite := make(chan struct{}, 1)

	go udpReceiveLoop(endReceive, conn, messageChan, receiveChan, active)
	go receiveHandlerLoop(endRead, receiveChan, sendChan, receiveHandler, parseHandler, active)
	go udpSendLoop(endWrite, conn, messageChan, sendChan, active)

	go keepAliveLoop(active, timeOut, endReceive, endRead, endWrite, endServer)

	initChan <- struct{}{}
	<-endServer
	messageChan <- models.ConsoleMsg{Type: grokstatconstants.MSG_MINOR, Message: fmt.Sprintf("Stopping server.")}
	endChan <- struct{}{}
}
