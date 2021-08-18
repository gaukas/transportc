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

// T.B.C.
