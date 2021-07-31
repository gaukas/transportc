package transportc

import "github.com/pion/webrtc/v3"

// Will be heavily rely on seed2sdp

// A net.Conn compliance struct
type WebRTConn struct {
	// states
	errmsg error
	role   WebRTCRole
	status WebRTConnStatus

	// datachannel to net.Conn interface
	dataChannel *DataChannel
	recvBuf     chan byte
	// sendBuf     chan byte // Shouldn't be needed
}

// NewWebRTConn() creates a new WebRTConn instance and returns a pointer to it.
func NewWebRTConn(dcconfig *DataChannelConfig, pionSE webrtc.SettingEngine, pionConf webrtc.Configuration) *WebRTConn {
	newDataChannel := DeclareDatachannel(dcconfig, pionSE, pionConf)

	newRole := OFFERER
	if dcconfig.SelfSDPType == "answer" {
		newRole = ANSWERER
	}

	return &WebRTConn{
		errmsg: nil,
		role:   newRole,
		status: WebRTConnNew,

		dataChannel: newDataChannel,
		recvBuf:     make(chan byte),
		// sendBuf: make(chan byte),
	}
}

// Init() setup the underlying datachannel with everything defined in c.dataChannel.
// once Init() is called upon a WebRTConn, it would be YAY or NAY only then
// i.e. No more configurablility.
func (c *WebRTConn) Init() {

}
