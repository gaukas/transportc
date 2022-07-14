package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Gaukas/transportc"
	"github.com/Gaukas/transportc/examples/rawsocket-multi-client/internal/strtool"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
)

// ping, a client actively establish the server and

var (
	logger         log.Logger = log.Logger{}
	udpSocket      *net.UDPConn
	udpMux         ice.UDPMux
	arrayConnMutex *sync.Mutex      = &sync.Mutex{}
	arrayAllConn   []*SyncWebRTConn = make([]*SyncWebRTConn, 0)
)

func preparePeerConnections() {
	var err error

	udpSocket, err = net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("10.0.0.11"),
		Port: 18981,
	})
	udpMux = webrtc.NewICEUDPMux(nil, udpSocket)

	if err != nil {
		logger.Fatalf("Error listening on UDP: %s", err)
	}

	for i := 0; i < 10; i++ {
		conn, _ := transportc.Dial("udp", "0.0.0.0")

		newDCConfig := transportc.DataChannelConfig{
			Label:          "DC#" + fmt.Sprintf("%d", i),
			SelfSDPType:    "answer",
			SendBufferSize: transportc.DataChannelBufferSizeDefault,
			UDPMux:         udpMux,
		}

		newSettingEngine := webrtc.SettingEngine{}
		newConfiguration := webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		}

		err := conn.Init(&newDCConfig, newSettingEngine, newConfiguration)

		if err != nil {
			logger.Panicf("Problem in initializing WebRTConn Instance #%d due to error %s\n", i, err.Error()) // Init must not fail. There's no recoverability in a WebRTConn failed to call Init()
		} else {
			logger.Printf("Successfully initialized WebRTConn Instance #%d...\n", i)
		}

		syncConn := NewSyncWebRTConn(conn)

		arrayAllConn = append(arrayAllConn, syncConn)
	}
}

func readLocalSDP()

func main() {
	logger.SetPrefix("[Log] ")
	logger.SetOutput(os.Stdout)

	arrayConnMutex.Lock()
	preparePeerConnections()
	arrayConnMutex.Unlock()

	// Checkpoint: Are all local SDPs valid?

	// Answerer: Wait for remote SDP
	logger.Println("Ready for Offer...")
	remoteSdp := strtool.MustReadStdin()
	conn.SetRemoteSDPJsonString(remoteSdp)

	// Offerer: Show local SDP
	logger.Println("Acquiring local SDP (answer)...")
	localSdp, err := conn.LocalSDPJsonString()
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}
	logger.Printf("Answer generated as below:\n%s\n", localSdp)

	// Block until conn is good to go
	if (conn.Status() & transportc.WebRTConnReady) == 0 {
		logger.Println("Waiting for peer...")
		for (conn.Status() & transportc.WebRTConnReady) == 0 {
		}
	}
	logger.Println("DataChannel established.")
	for {
		recv := make([]byte, 10)
		// Block until receive
		for {
			n, _ := conn.Read(recv)
			if n > 0 {
				// Copy recv into a "clean" array includes no \x00
				clean := []byte{}
				for _, b := range recv {
					if b != 0 {
						clean = append(clean, b)
					}
				}
				recv = clean
				// logger.Printf("Ping: %s of size %d", string(recv), len(recv))
				break
			}
		}

		if string(recv) == "BYE" {
			logger.Println("Ping: BYE")
			break
		} else {
			// logger.Printf("Pong: %s of size %d", string(recv), len(recv))
			conn.Write(recv)
		}
	}
	conn.Close()

	select {
	case <-time.After(time.Second * 20):
		logger.Fatalln("Timedout before any WebRTC Datachannel got established.")
	}
}
