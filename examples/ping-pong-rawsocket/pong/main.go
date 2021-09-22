package main

import (
	"log"
	"net"
	"os"

	"github.com/Gaukas/transportc"
	"github.com/Gaukas/transportc/examples/ping-pong-rawsocket/internal/strtool"
	"github.com/pion/webrtc/v3"
)

// ping, a client actively establish the server and

func main() {
	logger := log.Logger{}
	logger.SetPrefix("[Logger] ")
	logger.SetOutput(os.Stdout)

	logger.Println("Dialing to get a WebRTConn struct...")
	conn, _ := transportc.Dial("udp", "0.0.0.0")

	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: 8563,
	})

	newDCConfig := transportc.DataChannelConfig{
		Label:          "Ping-DataChannel",
		SelfSDPType:    "answer",
		SendBufferSize: transportc.DataChannelBufferSizeDefault,
		RawSocket:      udpListener,
	}

	newSettingEngine := webrtc.SettingEngine{}
	newConfiguration := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	logger.Println("Initializing WebRTConn instance...")
	err = conn.Init(&newDCConfig, newSettingEngine, newConfiguration)
	if err != nil {
		logger.Panic(err) // Init must not fail. There's no recoverability in a WebRTConn failed to call Init()
	}

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
}
