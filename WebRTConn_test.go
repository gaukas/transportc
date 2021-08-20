package transportc

import (
	"testing"

	"github.com/pion/webrtc/v3"
)

func testInit(conn *WebRTConn, sdpType string) error {
	newDCConfig := DataChannelConfig{
		Label:          "Test-DataChannel",
		SelfSDPType:    sdpType,
		SendBufferSize: DataChannelBufferSizeDefault,
	}
	newSettingEngine := webrtc.SettingEngine{}
	newConfiguration := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	return conn.Init(&newDCConfig, newSettingEngine, newConfiguration)
}

func TestDial(t *testing.T) {
	conn, err := Dial("udp", "0.0.0.0")
	if err != nil {
		t.Fatalf("Dial(): %s\n", err)
	}
	if conn.status != WebRTConnNew || conn.Status() != conn.status {
		t.Fatalf("conn.Status() and/or conn.status is not WebRTConnNew\n")
	}
}

func TestInit(t *testing.T) {
	conn, _ := Dial("udp", "0.0.0.0")
	err := testInit(&conn, "offer")
	if err != nil {
		t.Fatalf("Init(): %s\n", err)
	}
}

func TestCommunication(t *testing.T) {
	pingConn, _ := Dial("udp", "0.0.0.0")
	pongConn, _ := Dial("udp", "0.0.0.0")

	testInit(&pingConn, "offer")
	testInit(&pongConn, "answer")

	// Set Offer on Answerer.
	sdpOffer, pingErr := pingConn.LocalSDP()
	if pingErr != nil {
		t.Fatalf("LocalSDP(): %s\n", pingErr)
	}
	pongErr := pongConn.SetRemoteSDP(sdpOffer)
	if pongErr != nil {
		t.Fatalf("SetRemoteSDP(): %s\n", pongErr)
	}

	// Set Answer on Offerer
	sdpAnswer, pongErr := pongConn.LocalSDP()
	if pongErr != nil {
		t.Fatalf("LocalSDP(): %s\n", pongErr)
	}
	pingErr = pingConn.SetRemoteSDP(sdpAnswer)
	if pingErr != nil {
		t.Fatalf("SetRemoteSDP(): %s\n", pingErr)
	}

	// Wait for datachannel establishment
	for (pingConn.Status() & WebRTConnReady) == 0 {
	}
	for (pongConn.Status() & WebRTConnReady) == 0 {
	}

	// Prepare for real communications
	pingMsg := []byte("Ping")
	pongMsg := []byte("Pong")

	// Send from ping to pong
	_, pingErr = pingConn.Write(pingMsg)
	if pingErr != nil {
		t.Fatalf("Write(): %s\n", pingErr)
	}

	pongRecv := make([]byte, 10)
	for {
		n, _ := pongConn.Read(pongRecv)
		if n > 0 {
			// Copy recv into a "clean" array includes no \x00
			clean := []byte{}
			for _, b := range pongRecv {
				if b != 0 {
					clean = append(clean, b)
				}
			}
			pongRecv = clean
			// logger.Printf("Ping: %s of size %d", string(recv), len(recv))
			break
		}
	}

	if string(pongRecv) != "Ping" {
		t.Fatal("Read() failed the message integrity checking.\n")
	}

	// Send from pong to ping
	_, pongErr = pongConn.Write(pongMsg)
	if pongErr != nil {
		t.Fatalf("Write(): %s\n", pongErr)
	}

	pingRecv := make([]byte, 10)
	for {
		n, _ := pingConn.Read(pingRecv)
		if n > 0 {
			// Copy recv into a "clean" array includes no \x00
			clean := []byte{}
			for _, b := range pingRecv {
				if b != 0 {
					clean = append(clean, b)
				}
			}
			pingRecv = clean
			// logger.Printf("Ping: %s of size %d", string(recv), len(recv))
			break
		}
	}

	if string(pingRecv) != "Pong" {
		t.Fatal("Read() failed the message integrity checking.\n")
	}

	pingErr = pingConn.Close()
	pongErr = pongConn.Close()
	if pingErr != nil && pongErr != nil {
		t.Fatalf("Close(): %s, %s\n", pingErr, pongErr)
	}
}

// T.B.C.
