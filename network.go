package main

import (
	"fmt"
	"net"
	"time"
)

func receiveHandler(packet Packet, sendRequestChan chan<- Packet, parseHandler func(Packet) []Packet) {
	sendPackets := parseHandler(packet)

	for _, sendPacket := range sendPackets {
		sendRequestChan <- sendPacket
	}
}

func receiveHandlerLoop(endChan chan struct{}, receiveChan chan Packet, sendRequestChan chan<- Packet, receiveHandler func(Packet, chan<- Packet, func(Packet) []Packet), parseHandler func(Packet) []Packet, awakeChan chan<- struct{}) {
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

func readUDP(conn *net.UDPConn) (Packet, error) {
	bufsize := 2048
	buf := make([]byte, bufsize)

	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return Packet{}, err
	}

	return Packet{Data: buf[:n], Timestamp: time.Now().Unix(), RemoteAddr: addr.String()}, nil
}

func writeUDP(conn *net.UDPConn, packet Packet) {
	remoteIpUdp, rErr := net.ResolveUDPAddr("udp4", packet.RemoteAddr)
	CheckError(rErr)
	conn.WriteToUDP(packet.Data, remoteIpUdp)
}

func udpReceiveLoop(endChan <-chan struct{}, conn *net.UDPConn, messageChan chan<- ConsoleMsg, receiveChan chan Packet, awakeChan chan struct{}) {
	for {
		select {
		case <-endChan:
			return
		default:
			packet, err := readUDP(conn)
			awakeChan <- struct{}{}
			if err == nil {
				messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: fmt.Sprintf("Read %d bytes from %s", len(packet.Data), packet.RemoteAddr)}
				receiveChan <- packet
			}
		}
	}

}

func udpSendLoop(endChan <-chan struct{}, conn *net.UDPConn, messageChan chan<- ConsoleMsg, sendChan chan Packet, awakeChan chan struct{}) {
	for {
		select {
		case dataSendPayload := <-sendChan:
			messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: fmt.Sprintf("Writing %d bytes to %s", len(dataSendPayload.Data), dataSendPayload.RemoteAddr)}
			awakeChan <- struct{}{}
			go writeUDP(conn, dataSendPayload)
		case <-endChan:
			return
		}
	}
}

func AsyncUDPServer(endChan <-chan struct{}, initChan, doneChan chan<- struct{}, messageChan chan<- ConsoleMsg, sendChan, receiveChan chan Packet, parseHandler func(Packet) []Packet, timeOut time.Duration, awakeChan chan struct{}) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: 0,
		IP:   net.IPv4zero,
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("Starting UDP server at %s", conn.LocalAddr().String())}

	endReceive := make(chan struct{}, 1)
	endWrite := make(chan struct{}, 1)

	go udpReceiveLoop(endReceive, conn, messageChan, receiveChan, awakeChan)
	go udpSendLoop(endWrite, conn, messageChan, sendChan, awakeChan)

	initChan <- struct{}{}
	messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("Started UDP server at %s", conn.LocalAddr().String())}
	<-endChan
	endWrite <- struct{}{}
	messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: fmt.Sprintf("Stopped UDP send loop.")}
	endReceive <- struct{}{}
	messageChan <- ConsoleMsg{Type: MSG_DEBUG, Message: fmt.Sprintf("Stopped UDP capture loop.")}
	messageChan <- ConsoleMsg{Type: MSG_MINOR, Message: fmt.Sprintf("UDP server stopped.")}
	doneChan <- struct{}{}
}

func AsyncTCPServer(endChan <-chan struct{}, initChan, doneChan chan<- struct{}, messageChan chan<- ConsoleMsg, sendChan, receiveChan chan Packet, parseHandler func(Packet) []Packet, timeOut time.Duration, awakeChan chan struct{}) {
	initChan <- struct{}{}
	<-endChan
	doneChan <- struct{}{}
}

func splitSendPacketsLoop(genChan <-chan Packet, udpChan, tcpChan chan<- Packet) {
	for {
		packet := <-genChan
		if packet.Type.IsTCP() {
			tcpChan <- packet
		} else {
			udpChan <- packet
		}
	}
}

func AsyncNetworkServer(initChan, doneChan chan<- struct{}, messageChan chan<- ConsoleMsg, sendChan, receiveChan chan Packet, parseHandler func(Packet) []Packet, timeOut time.Duration) {
	awakeChan := make(chan struct{}, 9999)

	udpKillChan := make(chan struct{}, 1)
	tcpKillChan := make(chan struct{}, 1)

	udpStartedChan := make(chan struct{})
	udpStoppedChan := make(chan struct{})
	tcpStartedChan := make(chan struct{})
	tcpStoppedChan := make(chan struct{})

	udpSendChan := make(chan Packet)
	tcpSendChan := make(chan Packet)

	endCallbackChan := make(chan struct{})

	go splitSendPacketsLoop(sendChan, udpSendChan, tcpSendChan)

	go AsyncUDPServer(udpKillChan, udpStartedChan, udpStoppedChan, messageChan, udpSendChan, receiveChan, parseHandler, timeOut, awakeChan)
	go AsyncTCPServer(tcpKillChan, tcpStartedChan, tcpStoppedChan, messageChan, tcpSendChan, receiveChan, parseHandler, timeOut, awakeChan)

	go receiveHandlerLoop(endCallbackChan, receiveChan, sendChan, receiveHandler, parseHandler, awakeChan)

	<-udpStartedChan
	<-tcpStartedChan
	initChan <- struct{}{}

	go keepAliveLoop(awakeChan, timeOut, udpKillChan, tcpKillChan)

	<-udpStoppedChan
	<-tcpStoppedChan
	doneChan <- struct{}{}
}
