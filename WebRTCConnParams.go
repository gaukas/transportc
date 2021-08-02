package transportc

import "errors"

type WebRTCRole uint8

const (
	OFFERER WebRTCRole = iota
	ANSWERER
)

type WebRTConnStatus uint8

const (
	WebRTConnNew               WebRTConnStatus = 0      // No status, newly created.
	WebRTConnInit              WebRTConnStatus = 1 << 0 // Initialized.
	WebRTConnReady             WebRTConnStatus = 1 << 1 // Ready for send/recv
	WebRTConnClosed            WebRTConnStatus = 1 << 2
	WebRTConnErrored           WebRTConnStatus = 1 << 3 // Volatile state: Read error w
	WebRTConnLocalSDPReady     WebRTConnStatus = 1 << 4
	WebRTConnRemoteSDPReceived WebRTConnStatus = 1 << 5
)

var (
	ErrWebRTCUnsupportedNetwork = errors.New("ERR_WEBRTCONN_NETWORK_UNSUPPORTED")
	ErrWebRTConnReinit          = errors.New("ERR_WEBRTCONN_DOUBLE_INIT")
	ErrWebRTConnReadIntegrity   = errors.New("ERR_WEBRTCONN_READ_INTEGRITY_FAILURE")
)
