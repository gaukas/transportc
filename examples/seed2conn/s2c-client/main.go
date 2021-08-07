package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	s2s "github.com/Gaukas/seed2sdp"
	"github.com/Gaukas/transportc"
	"github.com/Gaukas/transportc/examples/seed2conn/internal/s2sHelper"
	"github.com/pion/webrtc/v3"
)

/**************************************************
 * s2c-client
 *
 * The client should generate an offer ACCORDING
 * TO THE SEED, print the deflated form of it and
 * then "guess" the SDP answer the server will generate.
 *
 **************************************************/

// ping, a client actively establish the server and

func usage(v interface{}) {
	fmt.Println("Usage: ./s2c-client remoteIP remoteport longseed")
	panic(v)
}

func main() {
	if len(os.Args) != 4 { // Min seed length: 6
		usage("BAD_NUM_ARGV")
	}

	if len(os.Args[3]) < 6 {
		usage("SEED_TOO_SHORT")
	}

	seed := os.Args[3]

	serverIP := net.ParseIP(os.Args[1])
	if serverIP == nil {
		usage("INVALID_IP_ADDRESS")
	}
	serverPort64, err := strconv.ParseUint(os.Args[2], 10, 16)
	if err != nil {
		usage("INVALID_PORT_NUM")
	}
	serverPort := uint16(serverPort64)

	clientHkdfParams := s2sHelper.HkdfOffer.SetSalt(os.Args[3])
	// serverHkdfParams := s2sHelper.HkdfAnswer.SetSalt(os.Args[3])

	cert, err := s2s.GetCertificate(clientHkdfParams)
	if err != nil {
		panic(err)
	}
	iceParams, err := s2s.PredictIceParameters(clientHkdfParams)
	if err != nil {
		panic(err)
	}

	logger := log.Logger{}
	logger.SetPrefix("[Logger] ")
	logger.SetOutput(os.Stdout)

	logger.Println("Dialing to get a WebRTConn struct...")
	conn, _ := transportc.Dial("udp", "0.0.0.0")

	newDCConfig := transportc.DataChannelConfig{
		Label:          "Seed2WebRTConn Client",
		SelfSDPType:    "offer",
		SendBufferSize: transportc.DataChannelBufferSizeDefault,
	}
	newSettingEngine := webrtc.SettingEngine{}
	iceParams.UpdateSettingEngine(&newSettingEngine)

	newConfiguration := webrtc.Configuration{
		Certificates: []webrtc.Certificate{cert},
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

	// Offerer: Show local SDP
	logger.Println("Acquiring local SDP (offer)...")
	localSdp, err := conn.LocalSDPJsonString()
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}
	// logger.Printf("Offer generated as:\n%s\n", localSdp)
	parsedSdp := s2s.ParseSDP(localSdp)
	// ip := net.ParseIP("10.0.0.11")
	deflatedParsedSdp := parsedSdp.Deflate(nil)
	logger.Printf("Offer generated and deflated as below:\n%s\n", deflatedParsedSdp.String())

	// Offerer: estimate remote SDP
	remoteSdp, err := s2sHelper.CreateSdpWithSeed(seed, serverIP, serverPort)
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}
	s2sHelper.InjectAppSpecs(remoteSdp)
	remoteSdp.AddAttrs(s2s.SDPAttribute{
		Key:   "setup",
		Value: "active", // Uncomment this line, if server calls SetDTLSActive() or by default
		// Value: "passive", // Uncomment this line, if server calls SetDTLSPassive()
	})

	logger.Println("Predicted Answer will be set in 10 seconds...")
	// logger.Printf("Predicted as:\n%s\n", remoteSdp.String())

	for i := 10; i > 0; i-- {
		// logger.Printf("%d...\n", i)
		time.Sleep(1 * time.Second)
	}

	conn.SetRemoteSDPJsonString(remoteSdp.String())

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
