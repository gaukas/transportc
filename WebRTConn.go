package transportc

import (
	"sync"

	"github.com/pion/webrtc/v3"
)

// Will be heavily rely on seed2sdp

// A net.Conn compliance struct
type WebRTConn struct {
	// mutex
	lock sync.RWMutex
	// states
	lasterr error
	role    WebRTCRole
	status  WebRTConnStatus

	// datachannel to net.Conn interface
	dataChannel *DataChannel
	recvBuf     chan byte
	// sendBuf     chan byte // Shouldn't be needed
}

// NewWebRTConn() creates a new WebRTConn instance and returns a pointer to it.
func NewWebRTConn(dcconfig *DataChannelConfig, pionSettingEngine webrtc.SettingEngine, pionConfiguration webrtc.Configuration) *WebRTConn {
	newDataChannel := DeclareDatachannel(dcconfig, pionSettingEngine, pionConfiguration)

	var newRole WebRTCRole
	if dcconfig.SelfSDPType == "answer" {
		newRole = ANSWERER
	} else {
		newRole = OFFERER
	}

	return &WebRTConn{
		lock:    sync.RWMutex{},
		lasterr: nil,
		role:    newRole,
		status:  WebRTConnNew,

		dataChannel: newDataChannel,
		recvBuf:     make(chan byte), // Thread-Safe?
		// sendBuf: make(chan byte),
	}
}
