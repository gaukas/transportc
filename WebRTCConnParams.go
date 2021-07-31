package transportc

type WebRTCRole uint8

const (
	OFFERER WebRTCRole = iota
	ANSWERER
)

type WebRTConnStatus uint8

const (
	WebRTConnNew               WebRTConnStatus = 0 // No status, newly created.
	WebRTConnInit              WebRTConnStatus = 1 // Initialized.
	WebRTConnReady             WebRTConnStatus = 2 // Ready for send/recv
	WebRTConnClosed            WebRTConnStatus = 4
	WebRTConnErrored           WebRTConnStatus = 8 // Volatile state: Read error w
	WebRTConnLocalSDPReady     WebRTConnStatus = 16
	WebRTConnRemoteSDPReceived WebRTConnStatus = 32
)
