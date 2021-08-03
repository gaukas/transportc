package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Gaukas/transportc"
	"github.com/Gaukas/transportc/examples/ping-pong/internal/strtool"
	"github.com/pion/webrtc/v3"
)

// ping, a client actively establish the server and

func main() {
	logger := log.Logger{}
	logger.SetPrefix("[Logger] ")
	logger.SetOutput(os.Stdout)

	logger.Println("Dialing to get a WebRTConn struct...")
	conn, _ := transportc.Dial("udp", "0.0.0.0")

	newDCConfig := transportc.DataChannelConfig{
		Label:          "Ping-DataChannel",
		SelfSDPType:    "offer",
		SendBufferSize: transportc.DataChannelBufferSizeDefault,
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
	err := conn.Init(&newDCConfig, newSettingEngine, newConfiguration)
	if err != nil {
		logger.Panic(err) // Init must not fail. There's no recoverability in a WebRTConn failed to call Init()
	}

	// Offerer: Show local SDP
	logger.Println("Acquiring local SDP (offer)...")
	localSdp, err := conn.LocalSDPJsonString()
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}
	logger.Printf("Offer generated as below:\n%s\n", localSdp)

	// Offerer: Wait for remote SDP
	logger.Println("Ready for Answer...")
	remoteSdp := strtool.MustReadStdin()

	conn.SetRemoteSDPJsonString(remoteSdp)

	// Block until conn is good to go
	if (conn.Status() & transportc.WebRTConnReady) == 0 {
		logger.Println("Waiting for peer...")
		for (conn.Status() & transportc.WebRTConnReady) == 0 {
		}
	}
	logger.Println("DataChannel established.")

	lastidx := 0
	for lastidx < 100 {
		b := []byte(fmt.Sprintf("%d", lastidx))
		conn.Write(b)
		// logger.Printf("Ping: %s of size %d\n", string(b), len(b))
		recv := make([]byte, 10)

		// Block until receive
		for {
			n, _ := conn.Read(recv)
			if n > 0 {
				clean := []byte{}
				for _, b := range recv {
					if b != 0 {
						clean = append(clean, b)
					}
				}
				recv = clean
				break
			}
		}
		logger.Printf("Ping: %s of size %d, Pong: %s of size %d\n", string(b), len(b), string(recv), len(recv))
		idx64, err := strconv.ParseInt(fmt.Sprintf("%s", string(recv)), 10, 64)
		if err != nil {
			panic(err)
		}
		// logger.Printf("Pong cast to: %d\n", idx64)
		lastidx = int(idx64) + 1
	}
	logger.Println("Closing WebRTConn...")
	conn.Write([]byte("BYE"))
	conn.Close()
}
