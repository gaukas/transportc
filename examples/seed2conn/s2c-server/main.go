package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	s2s "github.com/Gaukas/seed2sdp"
	"github.com/Gaukas/transportc"
	"github.com/Gaukas/transportc/examples/seed2conn/internal/s2sHelper"
	"github.com/Gaukas/transportc/examples/seed2conn/internal/strtool"
	"github.com/pion/webrtc/v3"
)

/**************************************************
 * s2c-server
 *
 * The server should wait for client's deflatedSDP
 * of offer and then generate answer ACCORDING TO
 * THE SEED (non-verbose)
 *
 **************************************************/

func usage(v interface{}) {
	fmt.Println("Usage: ./s2c-server serverIP serviceport longseed")
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

	// clientHkdfParams := s2sHelper.HkdfOffer.SetSalt(seed)
	serverHkdfParams := s2sHelper.HkdfAnswer.SetSalt(seed)

	cert, err := s2s.GetCertificate(serverHkdfParams)
	if err != nil {
		panic(err)
	}
	iceParams, err := s2s.PredictIceParameters(serverHkdfParams)
	if err != nil {
		panic(err)
	}

	logger := log.Logger{}
	logger.SetPrefix("[Logger] ")
	logger.SetOutput(os.Stdout)

	logger.Println("Dialing to get a WebRTConn struct...")
	conn, _ := transportc.Dial("udp", "0.0.0.0")

	newDCConfig := transportc.DataChannelConfig{
		Label:          "Seed2WebRTConn Server",
		SelfSDPType:    "answer",
		SendBufferSize: transportc.DataChannelBufferSizeDefault,

		IPAddr: []string{
			serverIP.String(),
		},
		CandidateType: webrtc.ICECandidateTypeHost,
		Port:          serverPort,
	}
	newSettingEngine := webrtc.SettingEngine{}
	iceParams.UpdateSettingEngine(&newSettingEngine)

	newConfiguration := webrtc.Configuration{
		Certificates: []webrtc.Certificate{
			cert,
		},
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
	logger.Println("Ready for Deflated Offer...")
	remoteDeflatedSdpStr := strtool.MustReadStdin()

	remoteDeflatedSdp, err := s2s.SDPDeflatedFromString(remoteDeflatedSdpStr)
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}

	remoteSdp, err := s2sHelper.InflateSdpWithSeed(seed, remoteDeflatedSdp)
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}

	s2sHelper.InjectAppSpecs(remoteSdp)
	remoteSdp.AddAttrs(s2s.SDPAttribute{
		Key:   "setup",
		Value: "actpass", // client usually generates actpass.
	})

	// logger.Printf("Inflated Offer:\n%s\n", remoteSdp.String())

	conn.SetRemoteSDPJsonString(remoteSdp.String())

	// Offerer: Show local SDP
	logger.Println("Acquiring local SDP (answer)...")
	_, err = conn.LocalSDP()
	if err != nil {
		logger.Panic(err) // If no valid local SDP, you can't establish WebRTC peer connection, and therefore, no datachannel.
	}
	// logger.Printf("Answer generated:\n%s\n", localSdp)
	logger.Printf("Answer generated and set.")

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
